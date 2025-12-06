package rabbitmq

import (
	"fmt"
	"wheres-my-pizza/internal/config"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Connection interface {
	Channel() (Channel, error)
	Close() error
}

type Channel interface {
	ExchangeDeclare(name, kind string, durable, autoDelete, internal, noWait bool, args amqp.Table) error
	QueueDeclare(name string, durable, autoDelete, exclusive, noWait bool, args amqp.Table) (Queue, error)
	QueueBind(name, key, exchange string, noWait bool, args amqp.Table) error
	Publish(exchange, key string, mandatory, immediate bool, msg amqp.Publishing) error
	Consume(queue, consumer string, autoAck, exclusive, noLocal, noWait bool, args amqp.Table) (<-chan amqp.Delivery, error)
	Qos(prefetchCount, prefetchSize int, global bool) error
	Close() error
}

type Queue struct {
	Name      string
	Messages  int
	Consumers int
}

type amqpConnection struct {
	conn *amqp.Connection
}

type amqpChannel struct {
	ch *amqp.Channel
}

func Connect(cfg config.RabbitMQConfig) (Connection, error) {
	url := fmt.Sprintf("amqp://%s:%s@%s:%d/",
		cfg.User, cfg.Password, cfg.Host, cfg.Port)

	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	return &amqpConnection{conn: conn}, nil
}

func (c *amqpConnection) Channel() (Channel, error) {
	ch, err := c.conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("failed to open channel: %w", err)
	}
	return &amqpChannel{ch: ch}, nil
}

func (c *amqpConnection) Close() error {
	return c.conn.Close()
}

func (ch *amqpChannel) ExchangeDeclare(name, kind string, durable, autoDelete, internal, noWait bool, args amqp.Table) error {
	return ch.ch.ExchangeDeclare(name, kind, durable, autoDelete, internal, noWait, args)
}

func (ch *amqpChannel) QueueDeclare(name string, durable, autoDelete, exclusive, noWait bool, args amqp.Table) (Queue, error) {
	q, err := ch.ch.QueueDeclare(name, durable, autoDelete, exclusive, noWait, args)
	if err != nil {
		return Queue{}, err
	}
	return Queue{Name: q.Name, Messages: q.Messages, Consumers: q.Consumers}, nil
}

func (ch *amqpChannel) QueueBind(name, key, exchange string, noWait bool, args amqp.Table) error {
	return ch.ch.QueueBind(name, key, exchange, noWait, args)
}

func (ch *amqpChannel) Publish(exchange, key string, mandatory, immediate bool, msg amqp.Publishing) error {
	return ch.ch.Publish(exchange, key, mandatory, immediate, msg)
}

func (ch *amqpChannel) Consume(queue, consumer string, autoAck, exclusive, noLocal, noWait bool, args amqp.Table) (<-chan amqp.Delivery, error) {
	return ch.ch.Consume(queue, consumer, autoAck, exclusive, noLocal, noWait, args)
}

func (ch *amqpChannel) Qos(prefetchCount, prefetchSize int, global bool) error {
	return ch.ch.Qos(prefetchCount, prefetchSize, global)
}

func (ch *amqpChannel) Close() error {
	return ch.ch.Close()
}
