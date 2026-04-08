package messaging

import "context"

// Message is normalized delivery data exposed to handlers.
type Message struct {
	Body       []byte
	RoutingKey string
	Headers    map[string]any
}

// HandlerFunc handles incoming messages.
type HandlerFunc func(ctx context.Context, msg Message) error

// Consumer subscribes to queue and invokes handler for each message.
type Consumer interface {
	Consume(
		ctx context.Context,
		queue string,
		handler HandlerFunc,
	) error
}
