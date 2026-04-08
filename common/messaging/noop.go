package messaging

import "context"

// NoopPublisher is used when messaging is disabled.
type NoopPublisher struct{}

func (NoopPublisher) Publish(
	ctx context.Context,
	exchange, routingKey string,
	msg Envelope,
) error {
	return nil
}

// NoopConsumer is used when messaging is disabled.
type NoopConsumer struct{}

func (NoopConsumer) Consume(
	ctx context.Context,
	queue string,
	handler HandlerFunc,
) error {
	<-ctx.Done()
	return ctx.Err()
}
