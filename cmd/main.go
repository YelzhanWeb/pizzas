package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/YelzhanWeb/pizzas/internal/adapter/logger"
	"github.com/YelzhanWeb/pizzas/internal/adapter/postgres"
	"github.com/YelzhanWeb/pizzas/internal/adapter/rabbitmq"
	"github.com/YelzhanWeb/pizzas/internal/app/kitchen"
	"github.com/YelzhanWeb/pizzas/internal/app/order"
	"github.com/YelzhanWeb/pizzas/internal/app/tracking"
	"github.com/YelzhanWeb/pizzas/internal/config"

	amqpAdapter "github.com/YelzhanWeb/pizzas/internal/adapter/amqp"
	httpAdapter "github.com/YelzhanWeb/pizzas/internal/adapter/http"
)

func main() {
	// Parse command-line flags
	mode := flag.String("mode", "", "Service mode: order-service, kitchen-worker, tracking-service, notification-subscriber")
	port := flag.Int("port", 3000, "HTTP port")
	workerName := flag.String("worker-name", "", "Worker name (for kitchen-worker)")
	orderTypes := flag.String("order-types", "", "Comma-separated order types (for kitchen-worker)")
	heartbeatInterval := flag.Int("heartbeat-interval", 30, "Heartbeat interval in seconds")
	prefetch := flag.Int("prefetch", 1, "RabbitMQ prefetch count")
	maxConcurrent := flag.Int("max-concurrent", 50, "Max concurrent orders")
	flag.Parse()

	if *mode == "" {
		log.Fatal("--mode flag is required")
	}

	// Load configuration
	cfg, err := config.Load("config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Setup infrastructure
	ctx := context.Background()

	// Initialize logger
	lgr := logger.New(*mode)

	// Connect to PostgreSQL
	db, err := postgres.Connect(ctx, cfg.Database)
	if err != nil {
		log.Fatalf("Failed to connect to PostgreSQL: %v", err)
	}
	defer db.Close()

	lgr.Info("db_connected", "Connected to PostgreSQL database", "startup", map[string]interface{}{
		"host": cfg.Database.Host,
		"db":   cfg.Database.Database,
	})

	// Connect to RabbitMQ
	mqConn, err := rabbitmq.Connect(cfg.RabbitMQ)
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}
	defer mqConn.Close()

	lgr.Info("rabbitmq_connected", "Connected to RabbitMQ", "startup", map[string]interface{}{
		"host": cfg.RabbitMQ.Host,
	})

	// Route to appropriate service
	switch *mode {
	case "order-service":
		runOrderService(ctx, db, mqConn, lgr, *port, *maxConcurrent)

	case "kitchen-worker":
		if *workerName == "" {
			log.Fatal("--worker-name is required for kitchen-worker mode")
		}
		runKitchenWorker(ctx, db, mqConn, lgr, *workerName, *orderTypes, *heartbeatInterval, *prefetch)

	case "tracking-service":
		runTrackingService(ctx, db, lgr, *port)

	case "notification-subscriber":
		runNotificationSubscriber(ctx, mqConn, lgr)

	default:
		log.Fatalf("Invalid mode: %s", *mode)
	}
}

func runOrderService(ctx context.Context, db postgres.DB, mqConn rabbitmq.Connection, lgr logger.Logger, port, maxConcurrent int) {
	// Initialize repositories
	orderRepo := postgres.NewOrderRepository(db)

	// Initialize messaging
	publisher := rabbitmq.NewPublisher(mqConn)

	// Initialize service
	orderService := order.NewService(orderRepo, publisher, lgr)

	// Initialize HTTP handler
	orderHandler := httpAdapter.NewOrderHandler(orderService, lgr)

	// Setup HTTP server
	mux := http.NewServeMux()
	mux.HandleFunc("/orders", orderHandler.CreateOrder)

	// Apply middleware
	handler := httpAdapter.LoggingMiddleware(lgr)(mux)
	handler = httpAdapter.RecoveryMiddleware(lgr)(handler)

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	lgr.Info("service_started", fmt.Sprintf("Order Service started on port %d", port), "startup", map[string]interface{}{
		"port":           port,
		"max_concurrent": maxConcurrent,
	})

	// Graceful shutdown
	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, os.Interrupt, syscall.SIGTERM)
		<-sigint

		lgr.Info("shutdown_initiated", "Shutting down Order Service", "shutdown", nil)

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			lgr.Error("shutdown_error", "Error during shutdown", "shutdown", nil, err)
		}
	}()

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		lgr.Error("server_error", "Server error", "runtime", nil, err)
	}
}

func runKitchenWorker(ctx context.Context, db postgres.DB, mqConn rabbitmq.Connection, lgr logger.Logger, workerName, orderTypes string, heartbeatInterval, prefetch int) {
	// Initialize repositories
	orderRepo := postgres.NewOrderRepository(db)
	workerRepo := postgres.NewWorkerRepository(db)

	// Initialize messaging
	publisher := rabbitmq.NewPublisher(mqConn)
	consumer := rabbitmq.NewConsumer(mqConn, prefetch)

	// Initialize service
	kitchenService := kitchen.NewService(orderRepo, workerRepo, publisher, lgr, workerName, orderTypes, heartbeatInterval)

	// Initialize AMQP handler
	orderHandlerAMQP := amqpAdapter.NewOrderHandler(kitchenService, lgr)

	// Start worker
	if err := kitchenService.Start(ctx); err != nil {
		log.Fatalf("Failed to start kitchen worker: %v", err)
	}

	lgr.Info("service_started", fmt.Sprintf("Kitchen Worker %s started", workerName), "startup", map[string]interface{}{
		"worker_name": workerName,
		"order_types": orderTypes,
		"prefetch":    prefetch,
	})

	// Start consuming messages
	go func() {
		if err := consumer.ConsumeOrders(ctx, orderHandlerAMQP.HandleOrder); err != nil {
			lgr.Error("consumer_error", "Error consuming orders", "runtime", nil, err)
		}
	}()

	// Wait for shutdown signal
	sigint := make(chan os.Signal, 1)
	signal.Notify(sigint, os.Interrupt, syscall.SIGTERM)
	<-sigint

	lgr.Info("graceful_shutdown", "Shutting down Kitchen Worker", "shutdown", nil)

	if err := kitchenService.Shutdown(ctx); err != nil {
		lgr.Error("shutdown_error", "Error during shutdown", "shutdown", nil, err)
	}
}

func runTrackingService(ctx context.Context, db postgres.DB, lgr logger.Logger, port int) {
	// Initialize repositories
	orderRepo := postgres.NewOrderRepository(db)
	workerRepo := postgres.NewWorkerRepository(db)

	// Initialize service
	trackingService := tracking.NewService(orderRepo, workerRepo, lgr)

	// Initialize HTTP handler
	trackingHandler := httpAdapter.NewTrackingHandler(trackingService, lgr)

	// Setup HTTP server
	mux := http.NewServeMux()
	mux.HandleFunc("/orders/", trackingHandler.HandleOrders)
	mux.HandleFunc("/workers/status", trackingHandler.GetWorkersStatus)

	// Apply middleware
	handler := httpAdapter.LoggingMiddleware(lgr)(mux)
	handler = httpAdapter.RecoveryMiddleware(lgr)(handler)

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	lgr.Info("service_started", fmt.Sprintf("Tracking Service started on port %d", port), "startup", map[string]interface{}{
		"port": port,
	})

	// Graceful shutdown
	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, os.Interrupt, syscall.SIGTERM)
		<-sigint

		lgr.Info("shutdown_initiated", "Shutting down Tracking Service", "shutdown", nil)

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			lgr.Error("shutdown_error", "Error during shutdown", "shutdown", nil, err)
		}
	}()

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		lgr.Error("server_error", "Server error", "runtime", nil, err)
	}
}

func runNotificationSubscriber(ctx context.Context, mqConn rabbitmq.Connection, lgr logger.Logger) {
	// Initialize consumer
	consumer := rabbitmq.NewConsumer(mqConn, 1)

	// Initialize handler
	notificationHandler := amqpAdapter.NewNotificationHandler(lgr)

	lgr.Info("service_started", "Notification Subscriber started", "startup", nil)

	// Start consuming notifications
	go func() {
		if err := consumer.ConsumeNotifications(ctx, notificationHandler.HandleNotification); err != nil {
			lgr.Error("consumer_error", "Error consuming notifications", "runtime", nil, err)
		}
	}()

	// Wait for shutdown signal
	sigint := make(chan os.Signal, 1)
	signal.Notify(sigint, os.Interrupt, syscall.SIGTERM)
	<-sigint

	lgr.Info("shutdown_initiated", "Shutting down Notification Subscriber", "shutdown", nil)
}
