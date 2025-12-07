package kitchen

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/YelzhanWeb/pizzas/internal/adapter/logger"
	"github.com/YelzhanWeb/pizzas/internal/domain"
	"github.com/YelzhanWeb/pizzas/internal/interfaces"
)

type Service struct {
	orderRepo         interfaces.OrderRepository
	workerRepo        interfaces.WorkerRepository
	publisher         interfaces.MessagePublisher
	logger            logger.Logger
	workerName        string
	orderTypes        []string
	heartbeatInterval time.Duration
}

func NewService(
	orderRepo interfaces.OrderRepository,
	workerRepo interfaces.WorkerRepository,
	publisher interfaces.MessagePublisher,
	logger logger.Logger,
	workerName string,
	orderTypes string,
	heartbeatInterval int,
) *Service {
	var types []string
	if orderTypes != "" {
		types = strings.Split(orderTypes, ",")
	}

	return &Service{
		orderRepo:         orderRepo,
		workerRepo:        workerRepo,
		publisher:         publisher,
		logger:            logger,
		workerName:        workerName,
		orderTypes:        types,
		heartbeatInterval: time.Duration(heartbeatInterval) * time.Second,
	}
}

func (s *Service) Start(ctx context.Context) error {
	// 1. Регистрация воркера
	worker, err := s.workerRepo.FindByName(ctx, s.workerName)
	if err == nil {
		// Воркер существует
		if worker.Status == domain.WorkerStatusOnline {
			return fmt.Errorf("worker with name %s is already online", s.workerName)
		}
		worker.Status = domain.WorkerStatusOnline
		worker.LastSeen = time.Now()
		if err := s.workerRepo.Update(ctx, worker); err != nil {
			return err
		}
	} else {
		// Создаем нового
		typeStr := "general"
		if len(s.orderTypes) > 0 {
			typeStr = strings.Join(s.orderTypes, ",")
		}
		worker, err = domain.NewWorker(s.workerName, typeStr)
		if err != nil {
			return err
		}
		if err := s.workerRepo.Create(ctx, worker); err != nil {
			return err
		}
	}

	s.logger.Info("worker_registered", fmt.Sprintf("Worker %s registered", s.workerName), "", nil)

	// Запуск Heartbeat в фоне
	go s.heartbeatLoop(ctx)

	return nil
}

func (s *Service) heartbeatLoop(ctx context.Context) {
	ticker := time.NewTicker(s.heartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := s.workerRepo.UpdateHeartbeat(ctx, s.workerName); err != nil {
				s.logger.Error("heartbeat_failed", "Failed to update heartbeat", "", nil, err)
			} else {
				s.logger.Debug("heartbeat_sent", "Heartbeat sent", "", nil)
			}
		}
	}
}

func (s *Service) Shutdown(ctx context.Context) error {
	worker, err := s.workerRepo.FindByName(ctx, s.workerName)
	if err != nil {
		return err
	}
	worker.SetOffline()
	return s.workerRepo.Update(ctx, worker)
}

func (s *Service) ProcessOrder(ctx context.Context, msg interfaces.OrderMessage) error {
	// 1. Проверка специализации
	if len(s.orderTypes) > 0 {
		supported := false
		for _, t := range s.orderTypes {
			if t == string(msg.OrderType) {
				supported = true
				break
			}
		}
		if !supported {
			// Возвращаем ошибку, начинающуюся с "worker", чтобы consumer.go сделал Nack с requeue
			return fmt.Errorf("worker %s cannot handle order type %s", s.workerName, msg.OrderType)
		}
	}

	s.logger.Debug("order_processing_started", fmt.Sprintf("Processing order %s", msg.OrderNumber), "", map[string]interface{}{"order": msg.OrderNumber})

	// Находим заказ в БД
	order, err := s.orderRepo.FindByNumber(ctx, msg.OrderNumber)
	if err != nil {
		return err
	}

	// Идемпотентность: если уже готовим или готово, пропускаем
	if order.Status != domain.StatusReceived {
		return nil
	}

	// 2. Начало готовки (Status: Cooking)
	if err := s.updateStatusAndNotify(ctx, order, domain.StatusCooking); err != nil {
		return err
	}

	// 3. Симуляция времени готовки
	cookingTime := order.GetCookingTime()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(cookingTime):
	}

	// 4. Завершение готовки (Status: Ready)
	if err := s.updateStatusAndNotify(ctx, order, domain.StatusReady); err != nil {
		return err
	}

	// Обновляем счетчик обработанных заказов
	if err := s.workerRepo.IncrementOrdersProcessed(ctx, s.workerName); err != nil {
		s.logger.Error("db_error", "Failed to increment worker stats", "", nil, err)
	}

	s.logger.Debug("order_completed", fmt.Sprintf("Order %s completed", msg.OrderNumber), "", nil)
	return nil
}

func (s *Service) updateStatusAndNotify(ctx context.Context, order *domain.Order, newStatus domain.Status) error {
	oldStatus := order.Status

	// Обновляем в памяти
	if err := order.TransitionTo(newStatus, s.workerName); err != nil {
		return err
	}

	if err := s.orderRepo.UpdateStatusWithLog(ctx, order, newStatus, s.workerName); err != nil {
		return fmt.Errorf("failed to update order status: %w", err)
	}

	// Отправляем уведомление
	notification := interfaces.StatusUpdateMessage{
		OrderNumber: order.Number,
		OldStatus:   oldStatus,
		NewStatus:   newStatus,
		ChangedBy:   s.workerName,
		Timestamp:   time.Now(),
	}

	// Если статус Cooking, добавляем примерное время готовности
	if newStatus == domain.StatusCooking {
		estimated := time.Now().Add(order.GetCookingTime())
		notification.EstimatedCompletion = estimated
	}

	if err := s.publisher.PublishStatusUpdate(ctx, notification); err != nil {
		s.logger.Error("rabbitmq_publish_failed", "Failed to publish status update", "", nil, err)
		// Не блокируем процесс из-за ошибки уведомления
	}

	return nil
}
