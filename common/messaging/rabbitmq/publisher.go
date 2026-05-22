package rabbitmq

import (
	"context"
	"encoding/json"
	"time"

	"ecommerce-be/common/constants"
	"ecommerce-be/common/messaging"

	amqp "github.com/rabbitmq/amqp091-go"
)

// Publisher is RabbitMQ-backed implementation of messaging.Publisher.
type Publisher struct {
	client *Client
}

// NewPublisher creates RabbitMQ publisher.
func NewPublisher(client *Client) *Publisher {
	return &Publisher{client: client}
}

// Publish publishes an envelope to exchange/routing key.
func (p *Publisher) Publish(
	ctx context.Context,
	exchange, routingKey string,
	msg messaging.Envelope,
) error {
	ch, err := p.client.Channel()
	if err != nil {
		return err
	}
	defer ch.Close()

	body, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	headers := amqp.Table{
		constants.MSG_HEADER_RETRY_COUNT: int32(msg.RetryCount),
	}
	if msg.TenantID != "" {
		headers[constants.MSG_HEADER_TENANT_ID] = msg.TenantID
	}
	if msg.ActorID != "" {
		headers[constants.MSG_HEADER_ACTOR_ID] = msg.ActorID
	}
	if msg.TraceID != "" {
		headers[constants.MSG_HEADER_TRACE_ID] = msg.TraceID
	}

	return ch.PublishWithContext(
		ctx,
		exchange,
		routingKey,
		false,
		false,
		amqp.Publishing{
			ContentType:   "application/json",
			DeliveryMode:  amqp.Persistent,
			Body:          body,
			Timestamp:     time.Now().UTC(),
			MessageId:     msg.MessageID,
			CorrelationId: msg.CorrelationID,
			Type:          msg.EventType,
			Headers:       headers,
		},
	)
}

var _ messaging.Publisher = (*Publisher)(nil)
