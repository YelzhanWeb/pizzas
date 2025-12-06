package rabbitmq

import (
	"context"
	"fmt"
	"wheres-my-pizza/internal/interfaces"
)

type consumer struct {
	conn     Connection
	prefetch int
}

func NewConsumer(conn Connection, prefetch int) interfaces.MessageConsumer {
	return &consumer{conn: conn, prefetch: prefetch}
}

func (c *consumer) ConsumeOrders(ctx context.Context, handler interfaces.OrderMessageHandler) error {
	ch, err := c.conn.Channel()
	if err != nil {
		return fmt.Errorf("failed to open channel: %w", err)
	}
	defer ch.Close()

	// Set QoS
	if err := ch.Qos(c.prefetch, 0, false); err != nil {
		return fmt.Errorf("failed to set QoS: %w", err)
	}

	// Declare exchange
	if err := ch.ExchangeDeclare("orders_topic", "topic", true, false, false, false, nil); err != nil {
		return fmt.Errorf("failed to declare exchange: %w", err)
	}

	// Declare queue
	q, err := ch.QueueDeclare("kitchen_queue", true, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("failed to declare queue: %w", err)
	}

	// Bind queue
	if err := ch.QueueBind(q.Name, "kitchen.#", "orders_topic", false, nil); err != nil {
		return fmt.Errorf("failed to bind queue: %w", err)
	}

	// Start consuming
	msgs, err := ch.Consume(q.Name, "", false, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("failed to start consuming: %w", err)
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case msg, ok := <-msgs:
			if !ok {
				return fmt.Errorf("channel closed")
			}

			if err := handler(ctx, msg.Body); err != nil {
				// Check if it's a specialization mismatch
				if err.Error()[:6] == "worker" {
					msg.Nack(false, true) // Requeue for other workers
				} else {
					msg.Nack(false, false) // Send to DLQ
				}
			} else {
				msg.Ack(false)
			}
		}
	}
}

func (c *consumer) ConsumeNotifications(ctx context.Context, handler interfaces.NotificationHandler) error {
	ch, err := c.conn.Channel()
	if err != nil {
		return fmt.Errorf("failed to open channel: %w", err)
	}
	defer ch.Close()

	// Declare exchange
	if err := ch.ExchangeDeclare("notifications_fanout", "fanout", true, false, false, false, nil); err != nil {
		return fmt.Errorf("failed to declare exchange: %w", err)
	}

	// Declare temporary queue
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
			return nil
		case msg, ok := <-msgs:
			if !ok {
				return fmt.Errorf("channel closed")
			}

			handler(ctx, msg.Body)
		}
	}
}
