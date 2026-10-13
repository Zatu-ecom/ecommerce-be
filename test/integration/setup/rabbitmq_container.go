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

// SetupRabbitMQContainer returns a handle on the process-wide shared broker
// with a fresh AMQP connection per caller. The broker boots once per `go
// test` package process; Ryuk reaps it at process exit. Queues remain
// caller-owned: suites declare their queues and drain them in teardown.
func SetupRabbitMQContainer(t *testing.T) *RabbitMQContainer {
	t.Helper()

	ctx := context.Background()
	sharedRabbitMu.Lock()
	defer sharedRabbitMu.Unlock()

	if sharedRabbitContainer != nil {
		conn, err := amqp.Dial(sharedRabbitContainer.AMQPURL)
		if err == nil {
			return &RabbitMQContainer{
				Container:  sharedRabbitContainer.Container,
				AMQPURL:    sharedRabbitContainer.AMQPURL,
				Connection: conn,
				ctx:        ctx,
			}
		}
		t.Logf("shared rabbitmq unreachable; starting a replacement")
		_ = sharedRabbitContainer.Container.Terminate(ctx)
		sharedRabbitContainer = nil
	}

	container, amqpURL := startRabbitMQBroker(t, ctx)
	conn, err := amqp.Dial(amqpURL)
	if err != nil {
		_ = container.Terminate(ctx)
		t.Fatalf("failed to connect rabbitmq amqp: %v", err)
	}
	sharedRabbitContainer = &RabbitMQContainer{
		Container: container,
		AMQPURL:   amqpURL,
		ctx:       ctx,
	}
	return &RabbitMQContainer{
		Container:  container,
		AMQPURL:    amqpURL,
		Connection: conn,
		ctx:        ctx,
	}
}

// startRabbitMQBroker boots one RabbitMQ container and returns it with its URL.
func startRabbitMQBroker(t *testing.T, ctx context.Context) (testcontainers.Container, string) {
	t.Helper()
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

	return container, fmt.Sprintf("amqp://guest:guest@%s:%s/", host, port.Port())
}

// Cleanup closes this handle's AMQP connection. The shared broker stays up
// for the next suite in this process; Ryuk reaps it at process exit.
func (r *RabbitMQContainer) Cleanup(t *testing.T) {
	t.Helper()
	if r == nil {
		return
	}
	if r.Connection != nil && !r.Connection.IsClosed() {
		_ = r.Connection.Close()
	}
}
