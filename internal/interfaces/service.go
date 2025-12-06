package interfaces

import (
	"context"
	"time"
	"wheres-my-pizza/internal/domain"
)

// Интерфейсы Сервисов (Business Logic)
type OrderService interface {
	CreateOrder(ctx context.Context, cmd CreateOrderCommand) (*domain.Order, error)
}

type KitchenService interface {
	Start(ctx context.Context) error
	Shutdown(ctx context.Context) error
	ProcessOrder(ctx context.Context, msg OrderMessage) error
}

type TrackingService interface {
	GetOrderStatus(ctx context.Context, orderNumber string) (*TrackingOrderResponse, error)
	GetOrderHistory(ctx context.Context, orderNumber string) ([]*domain.StatusLog, error)
	GetWorkersStatus(ctx context.Context) ([]*TrackingWorkerResponse, error)
}

// Ответы Tracking Service
type TrackingOrderResponse struct {
	OrderNumber         string
	CurrentStatus       domain.Status
	UpdatedAt           time.Time
	EstimatedCompletion *time.Time
	ProcessedBy         *string
}

type TrackingWorkerResponse struct {
	WorkerName      string
	Status          domain.WorkerStatus
	OrdersProcessed int
	LastSeen        time.Time
}
