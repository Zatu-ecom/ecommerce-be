package file_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// AzuriteContainer holds the runtime details of a running Azurite Testcontainer.
type AzuriteContainer struct {
	Container        testcontainers.Container
	BlobEndpoint     string
	ConnectionString string
	AccountName      string
	AccountKey       string
	ContainerName    string
}

// SetupAzurite returns a handle on the process-wide shared Azurite container,
// ensuring the requested blob container exists and starts empty. The backend
// stays up for the next suite; Ryuk reaps it at process exit.
func SetupAzurite(t *testing.T, containerName string) *AzuriteContainer {
	t.Helper()
	return acquireSharedAzurite(t, containerName)
}

// bootAzurite starts one Azurite container and creates the requested blob container.
func bootAzurite(t *testing.T, containerName string) *AzuriteContainer {
	t.Helper()

	ctx := context.Background()

	accountName := "devstoreaccount1"
	// Azurite well-known development key — safe to commit, never used in production.
	accountKey := "Eby8vdM02xNOcqFlqUwJPLlmEtlCDXJ1OUzFT50uSRZ6IFsuFq2UVErCz4I6tq/K1SZFPTOtr/KBHBeksoGMGw=="

	req := testcontainers.ContainerRequest{
		Image:        "mcr.microsoft.com/azure-storage/azurite:latest",
		ExposedPorts: []string{"10000/tcp"},
		Cmd: []string{
			"azurite-blob",
			"--blobHost", "0.0.0.0",
			"--blobPort", "10000",
			"--disableProductStyleUrl",
			"--skipApiVersionCheck",
		},
		WaitingFor: wait.ForListeningPort("10000/tcp").WithStartupTimeout(2 * time.Minute),
	}

	c, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Fatalf("failed to start azurite container: %v", err)
	}

	host, err := c.Host(ctx)
	if err != nil {
		_ = c.Terminate(ctx)
		t.Fatalf("failed to get azurite host: %v", err)
	}
	port, err := c.MappedPort(ctx, "10000/tcp")
	if err != nil {
		_ = c.Terminate(ctx)
		t.Fatalf("failed to get azurite port: %v", err)
	}

	blobEndpoint := fmt.Sprintf("http://%s:%s/%s", host, port.Port(), accountName)
	connStr := fmt.Sprintf(
		"DefaultEndpointsProtocol=http;AccountName=%s;AccountKey=%s;BlobEndpoint=%s;",
		accountName,
		accountKey,
		blobEndpoint,
	)

	EnsureAzuriteContainer(t, connStr, containerName)

	return &AzuriteContainer{
		Container:        c,
		BlobEndpoint:     blobEndpoint,
		ConnectionString: connStr,
		AccountName:      accountName,
		AccountKey:       accountKey,
		ContainerName:    containerName,
	}
}

// Cleanup is a no-op for the shared container: it stays up for the next
// suite in this process and Ryuk reaps it at process exit. Safe to call
// multiple times.
func (a *AzuriteContainer) Cleanup(t *testing.T) {
	t.Helper()
}

// EnsureAzuriteContainer creates the given blob container on an Azurite endpoint.
// Safe to call multiple times; already-exists errors are ignored.
func EnsureAzuriteContainer(t *testing.T, connStr, containerName string) {
	t.Helper()

	client, err := azblob.NewClientFromConnectionString(connStr, nil)
	if err != nil {
		t.Fatalf("failed to create azblob client: %v", err)
	}

	_, _ = client.CreateContainer(context.Background(), containerName, nil)
	// Accept both nil (created) and "already exists" errors.
}
