package factory

import (
	"context"
	"errors"
	"fmt"

	"ecommerce-be/common/config"
	"ecommerce-be/common/messaging"
	"ecommerce-be/common/messaging/rabbitmq"
)

// Factory builds messaging components (publisher/consumer) using selected broker.
type Factory struct {
	cfg       *config.MessagingConfig
	queueType messaging.QueueType

	rabbitClient *rabbitmq.Client
}

// New creates a messaging factory.
func New(queueType messaging.QueueType) (*Factory, error) {
	appCfg := config.Get()
	if appCfg == nil {
		return nil, errors.New("app config is not loaded")
	}

	msgCfg := &appCfg.Messaging

	qType := queueType
	if qType == "" {
		qType = msgCfg.QueueType
	}
	if qType == "" {
		qType = messaging.QueueTypeRabbitMQ
	}

	return &Factory{
		cfg:       msgCfg,
		queueType: qType,
	}, nil
}

// QueueType returns resolved broker type.
func (f *Factory) QueueType() messaging.QueueType {
	return f.queueType
}

// Publisher returns broker-backed publisher interface.
func (f *Factory) Publisher() (messaging.Publisher, error) {
	if !f.cfg.Enabled {
		return messaging.NoopPublisher{}, nil
	}

	switch f.queueType {
	case messaging.QueueTypeRabbitMQ:
		client, err := f.ensureRabbitClient()
		if err != nil {
			return nil, err
		}
		return rabbitmq.NewPublisher(client), nil
	case messaging.QueueTypeKafka:
		return nil, errors.New("kafka publisher is not implemented yet")
	default:
		return nil, fmt.Errorf("unsupported queue type: %s", f.queueType)
	}
}

// Consumer returns broker-backed consumer interface.
func (f *Factory) Consumer() (messaging.Consumer, error) {
	if !f.cfg.Enabled {
		return messaging.NoopConsumer{}, nil
	}

	switch f.queueType {
	case messaging.QueueTypeRabbitMQ:
		client, err := f.ensureRabbitClient()
		if err != nil {
			return nil, err
		}
		return rabbitmq.NewConsumer(client, f.cfg), nil
	case messaging.QueueTypeKafka:
		return nil, errors.New("kafka consumer is not implemented yet")
	default:
		return nil, fmt.Errorf("unsupported queue type: %s", f.queueType)
	}
}

// DeclareBaseInfrastructure declares common exchanges/topics for selected broker.
func (f *Factory) DeclareBaseInfrastructure(ctx context.Context) error {
	if !f.cfg.Enabled {
		return nil
	}

	switch f.queueType {
	case messaging.QueueTypeRabbitMQ:
		client, err := f.ensureRabbitClient()
		if err != nil {
			return err
		}
		return client.DeclareBaseExchanges(
			ctx,
			f.cfg.CommandsExchange,
			f.cfg.EventsExchange,
		)
	case messaging.QueueTypeKafka:
		return errors.New("kafka infrastructure declaration is not implemented yet")
	default:
		return fmt.Errorf("unsupported queue type: %s", f.queueType)
	}
}

// Close closes any open broker resources.
func (f *Factory) Close() error {
	if f.rabbitClient != nil {
		return f.rabbitClient.Close()
	}
	return nil
}

func (f *Factory) ensureRabbitClient() (*rabbitmq.Client, error) {
	if f.rabbitClient != nil {
		return f.rabbitClient, nil
	}

	rabbitCfg, err := rabbitmq.LoadConfigFromEnv()
	if err != nil {
		return nil, err
	}

	client := rabbitmq.NewClient(rabbitCfg)
	if err := client.Connect(); err != nil {
		return nil, err
	}
	f.rabbitClient = client
	return client, nil
}
