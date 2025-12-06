package order

import (
	"context"
	"fmt"

	"github.com/YelzhanWeb/pizzas/internal/adapter/logger"
	"github.com/YelzhanWeb/pizzas/internal/domain"
	"github.com/YelzhanWeb/pizzas/internal/interfaces"
)

type Service struct {
	repo      interfaces.OrderRepository
	publisher interfaces.MessagePublisher
	logger    logger.Logger
}

func NewService(repo interfaces.OrderRepository, publisher interfaces.MessagePublisher, logger logger.Logger) *Service {
	return &Service{
		repo:      repo,
		publisher: publisher,
		logger:    logger,
	}
}

func (s *Service) CreateOrder(ctx context.Context, cmd interfaces.CreateOrderCommand) (*domain.Order, error) {
	// 1. Преобразование команд в доменные модели
	items := make([]domain.OrderItem, len(cmd.Items))
	for i, item := range cmd.Items {
		items[i] = domain.OrderItem{
			Name:     item.Name,
			Quantity: item.Quantity,
			Price:    item.Price,
		}
	}

	orderType := domain.OrderType(cmd.OrderType)

	// 2. Создание доменной сущности (здесь происходит валидация и расчет приоритета)
	order, err := domain.NewOrder(cmd.CustomerName, orderType, items, cmd.TableNumber, cmd.DeliveryAddress)
	if err != nil {
		s.logger.Error("validation_failed", "Order validation failed", "", nil, err)
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// 3. Генерация номера заказа
	number, err := s.repo.GenerateOrderNumber(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to generate order number: %w", err)
	}
	order.Number = number

	// 4. Сохранение в БД (Транзакционно вместе с логами)
	if err := s.repo.Create(ctx, order); err != nil {
		s.logger.Error("db_transaction_failed", "Failed to create order", "", nil, err)
		return nil, err
	}
	s.logger.Debug("order_received", "Order created in DB", "", map[string]interface{}{"order_number": order.Number})

	// 5. Публикация сообщения в RabbitMQ
	msg := interfaces.OrderMessage{
		OrderNumber:     order.Number,
		CustomerName:    order.CustomerName,
		OrderType:       order.Type,
		TableNumber:     order.TableNumber,
		DeliveryAddress: order.DeliveryAddress,
		Items:           order.Items,
		TotalAmount:     order.TotalAmount,
		Priority:        order.Priority,
	}

	if err := s.publisher.PublishOrder(ctx, msg); err != nil {
		s.logger.Error("rabbitmq_publish_failed", "Failed to publish order", "", nil, err)
		// В реальной системе здесь могла бы быть логика Outbox Pattern или отката транзакции,
		// но для учебного проекта вернем ошибку.
		return nil, err
	}

	s.logger.Debug("order_published", "Order published to RabbitMQ", "", map[string]interface{}{"order_number": order.Number})

	return order, nil
}
