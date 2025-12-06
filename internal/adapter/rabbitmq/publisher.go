package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"wheres-my-pizza/internal/interfaces"

	amqp "github.com/rabbitmq/amqp091-go"
)

type publisher struct {
	conn Connection
}

func NewPublisher(conn Connection) interfaces.MessagePublisher {
	return &publisher{conn: conn}
}

func (p *publisher) PublishOrder(ctx context.Context, msg interfaces.OrderMessage) error {
	return p.publishWithRetry(ctx, func(ch Channel) error {
		// Declare exchange
		if err := ch.ExchangeDeclare("orders_topic", "topic", true, false, false, false, nil); err != nil {
			return fmt.Errorf("failed to declare exchange: %w", err)
		}

		body, err := json.Marshal(msg)
		if err != nil {
			return fmt.Errorf("failed to marshal message: %w", err)
		}

		routingKey := fmt.Sprintf("kitchen.%s.%d", msg.OrderType, msg.Priority)

		err = ch.Publish("orders_topic", routingKey, false, false, amqp.Publishing{
			DeliveryMode: amqp.Persistent,
			ContentType:  "application/json",
			Body:         body,
			Priority:     uint8(msg.Priority),
		})
		if err != nil {
			return fmt.Errorf("failed to publish message: %w", err)
		}

		return nil
	})
}

func (p *publisher) PublishStatusUpdate(ctx context.Context, msg interfaces.StatusUpdateMessage) error {
	return p.publishWithRetry(ctx, func(ch Channel) error {
		// Declare exchange
		if err := ch.ExchangeDeclare("notifications_fanout", "fanout", true, false, false, false, nil); err != nil {
			return fmt.Errorf("failed to declare exchange: %w", err)
		}

		body, err := json.Marshal(msg)
		if err != nil {
			return fmt.Errorf("failed to marshal message: %w", err)
		}

		err = ch.Publish("notifications_fanout", "", false, false, amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		})
		if err != nil {
			return fmt.Errorf("failed to publish message: %w", err)
		}

		return nil
	})
}

// publishWithRetry выполняет публикацию с повторными попытками
func (p *publisher) publishWithRetry(ctx context.Context, publishFn func(Channel) error) error {
	const maxRetries = 3
	const retryDelay = 2 * time.Second

	var lastErr error

	for attempt := 0; attempt < maxRetries; attempt++ {
		// Проверяем контекст перед попыткой
		if ctx.Err() != nil {
			return ctx.Err()
		}

		ch, err := p.conn.Channel()
		if err != nil {
			lastErr = fmt.Errorf("failed to open channel (attempt %d/%d): %w", attempt+1, maxRetries, err)

			if attempt < maxRetries-1 {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(retryDelay):
					continue
				}
			}
			continue
		}

		// Пытаемся опубликовать
		err = publishFn(ch)
		ch.Close()

		if err == nil {
			return nil
		}

		lastErr = fmt.Errorf("publish failed (attempt %d/%d): %w", attempt+1, maxRetries, err)

		// Ждем перед следующей попыткой
		if attempt < maxRetries-1 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(retryDelay):
				continue
			}
		}
	}

	return fmt.Errorf("failed to publish after %d attempts: %w", maxRetries, lastErr)
}
