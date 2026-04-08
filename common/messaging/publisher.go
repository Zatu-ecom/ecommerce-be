package messaging

import "context"

// Publisher publishes envelopes to the configured broker.
type Publisher interface {
	Publish(
		ctx context.Context,
		exchange, routingKey string,
		msg Envelope,
	) error
}
