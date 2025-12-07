package rabbitmq

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/YelzhanWeb/pizzas/internal/interfaces"
	amqp "github.com/rabbitmq/amqp091-go"
)

type consumer struct {
	conn     Connection
	prefetch int
}

func NewConsumer(conn Connection, prefetch int) interfaces.MessageConsumer {
	return &consumer{conn: conn, prefetch: prefetch}
}

func (c *consumer) ConsumeOrders(ctx context.Context, handler interfaces.OrderMessageHandler) error {
	for {
		err := c.consumeOrdersWithReconnect(ctx, handler)

		// Если контекст отменен или соединение закрыто намеренно - выходим
		if ctx.Err() != nil {
			return ctx.Err()
		}

		if err == nil {
			return nil
		}

		// Логируем ошибку и пытаемся переподключиться
		log.Printf("Orders consumer disconnected: %v. Reconnecting in 5 seconds...", err)

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(5 * time.Second):
			// Продолжаем попытки переподключения
		}
	}
}

func (c *consumer) ConsumeNotifications(ctx context.Context, handler interfaces.NotificationHandler) error {
	for {
		err := c.consumeNotificationsWithReconnect(ctx, handler)

		// Если контекст отменен или соединение закрыто намеренно - выходим
		if ctx.Err() != nil {
			return ctx.Err()
		}

		if err == nil {
			return nil
		}

		// Логируем ошибку и пытаемся переподключиться
		log.Printf("Notifications consumer disconnected: %v. Reconnecting in 5 seconds...", err)

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(5 * time.Second):
			// Продолжаем попытки переподключения
		}
	}
}

func (c *consumer) consumeOrdersWithReconnect(ctx context.Context, handler interfaces.OrderMessageHandler) error {
	ch, err := c.conn.Channel()
	if err != nil {
		return fmt.Errorf("failed to open channel: %w", err)
	}
	defer ch.Close()

	// Отслеживаем закрытие канала
	closeChan := ch.NotifyClose()

	// Set QoS
	if err := ch.Qos(c.prefetch, 0, false); err != nil {
		return fmt.Errorf("failed to set QoS: %w", err)
	}

	// Declare exchanges and queues
	if err := c.setupOrdersInfrastructure(ch); err != nil {
		return err
	}

	// Start consuming
	msgs, err := ch.Consume("kitchen_queue", "", false, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("failed to start consuming: %w", err)
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case err := <-closeChan:
			if err != nil {
				return fmt.Errorf("channel closed: %w", err)
			}
			return fmt.Errorf("channel closed gracefully")

		case msg, ok := <-msgs:
			if !ok {
				return fmt.Errorf("messages channel closed")
			}

			if err := handler(ctx, msg.Body); err != nil {
				// Проверяем, является ли ошибка связанной со специализацией
				if strings.Contains(err.Error(), "cannot handle order type") {
					// Requeue для других воркеров
					msg.Nack(false, true)
				} else {
					// Отправляем в DLQ (requeue=false)
					msg.Nack(false, false)
				}
			} else {
				msg.Ack(false)
			}
		}
	}
}

func (c *consumer) consumeNotificationsWithReconnect(ctx context.Context, handler interfaces.NotificationHandler) error {
	ch, err := c.conn.Channel()
	if err != nil {
		return fmt.Errorf("failed to open channel: %w", err)
	}
	defer ch.Close()

	// Отслеживаем закрытие канала
	closeChan := ch.NotifyClose()

	// Declare exchange
	if err := ch.ExchangeDeclare("notifications_fanout", "fanout", true, false, false, false, nil); err != nil {
		return fmt.Errorf("failed to declare exchange: %w", err)
	}

	// Declare temporary exclusive queue
	q, err := ch.QueueDeclare("", false, true, true, false, nil)
	if err != nil {
		return fmt.Errorf("failed to declare queue: %w", err)
	}

	// Bind queue
	if err := ch.QueueBind(q.Name, "", "notifications_fanout", false, nil); err != nil {
		return fmt.Errorf("failed to bind queue: %w", err)
	}

	// Start consuming
	msgs, err := ch.Consume(q.Name, "", true, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("failed to start consuming: %w", err)
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case err := <-closeChan:
			if err != nil {
				return fmt.Errorf("channel closed: %w", err)
			}
			return fmt.Errorf("channel closed gracefully")

		case msg, ok := <-msgs:
			if !ok {
				return fmt.Errorf("messages channel closed")
			}

			// Игнорируем ошибки обработки уведомлений
			_ = handler(ctx, msg.Body)
		}
	}
}

func (c *consumer) setupOrdersInfrastructure(ch Channel) error {
	// Declare main exchange
	if err := ch.ExchangeDeclare("orders_topic", "topic", true, false, false, false, nil); err != nil {
		return fmt.Errorf("failed to declare orders exchange: %w", err)
	}

	// Declare DLQ exchange
	dlqExchange := "orders_dlq"
	if err := ch.ExchangeDeclare(dlqExchange, "direct", true, false, false, false, nil); err != nil {
		return fmt.Errorf("failed to declare DLQ exchange: %w", err)
	}

	// Declare DLQ queue
	dlqQueue := "kitchen_queue_dlq"
	if _, err := ch.QueueDeclare(dlqQueue, true, false, false, false, nil); err != nil {
		return fmt.Errorf("failed to declare DLQ: %w", err)
	}

	// Bind DLQ
	if err := ch.QueueBind(dlqQueue, "#", dlqExchange, false, nil); err != nil {
		return fmt.Errorf("failed to bind DLQ: %w", err)
	}

	// Declare main queue with DLQ binding
	args := amqp.Table{
		"x-dead-letter-exchange": dlqExchange,
	}

	q, err := ch.QueueDeclare("kitchen_queue", true, false, false, false, args)
	if err != nil {
		return fmt.Errorf("failed to declare kitchen queue: %w", err)
	}

	// Bind main queue
	if err := ch.QueueBind(q.Name, "kitchen.#", "orders_topic", false, nil); err != nil {
		return fmt.Errorf("failed to bind kitchen queue: %w", err)
	}

	return nil
}
