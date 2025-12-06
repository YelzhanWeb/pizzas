package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/YelzhanWeb/pizzas/internal/interfaces"
	amqp "github.com/rabbitmq/amqp091-go"
)

type publisher struct {
	conn Connection
}

func NewPublisher(conn Connection) interfaces.MessagePublisher {
	return &publisher{conn: conn}
}

func (p *publisher) PublishOrder(ctx context.Context, msg interfaces.OrderMessage) error {
	ch, err := p.conn.Channel()
	if err != nil {
		return fmt.Errorf("failed to open channel: %w", err)
	}
	defer ch.Close()

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
		return fmt.Errorf("failed to publish message:ssage: %w", err)
	}

	return nil
}

func (p *publisher) PublishStatusUpdate(ctx context.Context, msg interfaces.StatusUpdateMessage) error {
	ch, err := p.conn.Channel()
	if err != nil {
		return fmt.Errorf("failed to open channel: %w", err)
	}
	defer ch.Close()

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
}
