package interfaces

import (
	"context"
	"wheres-my-pizza/internal/domain"
)

// Интерфейсы Репозиториев (Adapter/Postgres)
type OrderRepository interface {
	Create(ctx context.Context, order *domain.Order) error
	FindByNumber(ctx context.Context, number string) (*domain.Order, error)
	GenerateOrderNumber(ctx context.Context) (string, error)
	Update(ctx context.Context, order *domain.Order) error
	LogStatus(ctx context.Context, orderID int, status domain.Status, changedBy string) error
	GetStatusHistory(ctx context.Context, orderID int) ([]*domain.StatusLog, error)
}

type WorkerRepository interface {
	Create(ctx context.Context, worker *domain.Worker) error
	FindByName(ctx context.Context, name string) (*domain.Worker, error)
	Update(ctx context.Context, worker *domain.Worker) error
	UpdateHeartbeat(ctx context.Context, name string) error
	ListAll(ctx context.Context) ([]*domain.Worker, error)
	IncrementOrdersProcessed(ctx context.Context, name string) error
}
