package messaging

import "strings"

// QueueType identifies the underlying message broker implementation.
type QueueType string

const (
	QueueTypeRabbitMQ QueueType = "rabbitmq"
	QueueTypeKafka    QueueType = "kafka"
)

// ParseQueueType parses user/config input and defaults to RabbitMQ.
func ParseQueueType(in string) QueueType {
	switch QueueType(strings.ToLower(strings.TrimSpace(in))) {
	case QueueTypeKafka:
		return QueueTypeKafka
	case QueueTypeRabbitMQ:
		fallthrough
	default:
		return QueueTypeRabbitMQ
	}
}
