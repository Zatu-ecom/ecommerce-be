package setup

import (
	"context"
	"fmt"
	"testing"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// RabbitMQContainer captures runtime handles for an integration RabbitMQ instance.
type RabbitMQContainer struct {
	Container  testcontainers.Container
	AMQPURL    string
	Connection *amqp.Connection
	ctx        context.Context
}

// SetupRabbitMQContainer boots RabbitMQ and returns a live AMQP connection.
func SetupRabbitMQContainer(t *testing.T) *RabbitMQContainer {
	t.Helper()

	ctx := context.Background()
	container, err := testcontainers.GenericContainer(
		ctx,
		testcontainers.GenericContainerRequest{
			ContainerRequest: testcontainers.ContainerRequest{
				Image:        "rabbitmq:3.13-management-alpine",
				ExposedPorts: []string{"5672/tcp", "15672/tcp"},
				WaitingFor: wait.ForLog("Server startup complete").
					WithStartupTimeout(3 * time.Minute),
			},
			Started: true,
		},
	)
	if err != nil {
		t.Fatalf("failed to start rabbitmq container: %v", err)
	}

	host, err := container.Host(ctx)
	if err != nil {
		_ = container.Terminate(ctx)
		t.Fatalf("failed to resolve rabbitmq host: %v", err)
	}
	port, err := container.MappedPort(ctx, "5672")
	if err != nil {
		_ = container.Terminate(ctx)
		t.Fatalf("failed to resolve rabbitmq mapped port: %v", err)
	}

	amqpURL := fmt.Sprintf("amqp://guest:guest@%s:%s/", host, port.Port())
	conn, err := amqp.Dial(amqpURL)
	if err != nil {
		_ = container.Terminate(ctx)
		t.Fatalf("failed to connect rabbitmq amqp: %v", err)
	}

	return &RabbitMQContainer{
		Container:  container,
		AMQPURL:    amqpURL,
		Connection: conn,
		ctx:        ctx,
	}
}

// Cleanup closes AMQP connection and container.
func (r *RabbitMQContainer) Cleanup(t *testing.T) {
	t.Helper()
	if r == nil {
		return
	}
	if r.Connection != nil && !r.Connection.IsClosed() {
		_ = r.Connection.Close()
	}
	if r.Container != nil {
		_ = r.Container.Terminate(r.ctx)
	}
}
