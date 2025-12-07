package interfaces

import (
	"context"
	"time"

	"github.com/YelzhanWeb/pizzas/internal/domain"
)

// Сообщения RabbitMQ
type OrderMessage struct {
	OrderNumber     string             `json:"order_number"`
	CustomerName    string             `json:"customer_name"`
	OrderType       domain.OrderType   `json:"order_type"`
	TableNumber     *int               `json:"table_number"`
	DeliveryAddress *string            `json:"delivery_address"`
	Items           []domain.OrderItem `json:"items"`
	TotalAmount     float64            `json:"total_amount"`
	Priority        domain.Priority    `json:"priority"`
}

type StatusUpdateMessage struct {
	OrderNumber         string        `json:"order_number"`
	OldStatus           domain.Status `json:"old_status"`
	NewStatus           domain.Status `json:"new_status"`
	ChangedBy           string        `json:"changed_by"`
	Timestamp           time.Time     `json:"timestamp"`
	EstimatedCompletion time.Time     `json:"estimated_completion"`
}

// Команды для сервисов
type CreateOrderCommand struct {
	CustomerName    string
	OrderType       string
	TableNumber     *int
	DeliveryAddress *string
	Items           []CreateOrderItemCommand
}

type CreateOrderItemCommand struct {
	Name     string
	Quantity int
	Price    float64
}

// Интерфейсы Messaging (Adapter/RabbitMQ)
type MessagePublisher interface {
	PublishOrder(ctx context.Context, msg OrderMessage) error
	PublishStatusUpdate(ctx context.Context, msg StatusUpdateMessage) error
}

type MessageConsumer interface {
	ConsumeOrders(ctx context.Context, handler OrderMessageHandler) error
	ConsumeNotifications(ctx context.Context, handler NotificationHandler) error
}

type (
	OrderMessageHandler func(ctx context.Context, body []byte) error
	NotificationHandler func(ctx context.Context, body []byte) error
)
