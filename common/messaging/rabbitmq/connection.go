package rabbitmq

import (
	"context"
	"errors"
	"sync"

	"ecommerce-be/common/constants"

	amqp "github.com/rabbitmq/amqp091-go"
)

// Client manages RabbitMQ connection lifecycle.
type Client struct {
	cfg Config

	mu   sync.RWMutex
	conn *amqp.Connection
}

// NewClient creates a new RabbitMQ client.
func NewClient(cfg Config) *Client {
	return &Client{cfg: cfg}
}

// Connect opens a RabbitMQ connection if not already connected.
func (c *Client) Connect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn != nil && !c.conn.IsClosed() {
		return nil
	}
	if c.cfg.URL == "" {
		return errors.New("rabbitmq url is not configured")
	}

	conn, err := amqp.Dial(c.cfg.URL)
	if err != nil {
		return err
	}

	c.conn = conn
	return nil
}

// Close closes open RabbitMQ connection.
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn == nil {
		return nil
	}
	return c.conn.Close()
}

// Channel returns a new channel from active connection.
func (c *Client) Channel() (*amqp.Channel, error) {
	c.mu.RLock()
	conn := c.conn
	c.mu.RUnlock()

	if conn == nil || conn.IsClosed() {
		if err := c.Connect(); err != nil {
			return nil, err
		}
		c.mu.RLock()
		conn = c.conn
		c.mu.RUnlock()
	}

	return conn.Channel()
}

// DeclareBaseExchanges declares shared exchanges used by commands/events.
func (c *Client) DeclareBaseExchanges(
	ctx context.Context,
	commandsExchange, eventsExchange string,
) error {
	ch, err := c.Channel()
	if err != nil {
		return err
	}
	defer ch.Close()

	if err := ch.ExchangeDeclare(
		commandsExchange,
		constants.DEFAULT_EXCHANGE_TYPE,
		true,
		false,
		false,
		false,
		nil,
	); err != nil {
		return err
	}

	if err := ch.ExchangeDeclare(
		eventsExchange,
		constants.DEFAULT_EXCHANGE_TYPE,
		true,
		false,
		false,
		false,
		nil,
	); err != nil {
		return err
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}
