package rabbitmq

import (
	"context"
	"sync"

	"ecommerce-be/common/config"
	"ecommerce-be/common/messaging"

	amqp "github.com/rabbitmq/amqp091-go"
)

// Consumer is RabbitMQ-backed implementation of messaging.Consumer.
type Consumer struct {
	client *Client
	cfg    *config.MessagingConfig
}

// NewConsumer creates RabbitMQ consumer.
func NewConsumer(client *Client, cfg *config.MessagingConfig) *Consumer {
	return &Consumer{client: client, cfg: cfg}
}

// Consume subscribes to queue and processes messages with controlled concurrency.
func (c *Consumer) Consume(
	ctx context.Context,
	queue string,
	handler messaging.HandlerFunc,
) error {
	ch, err := c.client.Channel()
	if err != nil {
		return err
	}
	defer ch.Close()

	if err := ch.Qos(c.cfg.Prefetch, 0, false); err != nil {
		return err
	}

	deliveries, err := ch.Consume(
		queue,
		"",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	workers := c.cfg.ConsumerConcurrency
	if workers <= 0 {
		workers = 1
	}

	sem := make(chan struct{}, workers)
	var wg sync.WaitGroup

	for {
		select {
		case <-ctx.Done():
			wg.Wait()
			return ctx.Err()
		case d, ok := <-deliveries:
			if !ok {
				wg.Wait()
				return nil
			}

			sem <- struct{}{}
			wg.Add(1)
			go func(delivery amqp.Delivery) {
				defer func() {
					<-sem
					wg.Done()
				}()

				msg := messaging.Message{
					Body:       delivery.Body,
					RoutingKey: delivery.RoutingKey,
					Headers:    mapFromTable(delivery.Headers),
				}

				if err := handler(ctx, msg); err != nil {
					if messaging.IsRetryable(err) {
						_ = delivery.Nack(false, true)
						return
					}
					_ = delivery.Nack(false, false)
					return
				}

				_ = delivery.Ack(false)
			}(d)
		}
	}
}

func mapFromTable(t amqp.Table) map[string]any {
	out := make(map[string]any, len(t))
	for k, v := range t {
		out[k] = v
	}
	return out
}

var _ messaging.Consumer = (*Consumer)(nil)
