package file_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// FakeGCSContainer holds the runtime details of a running fake-gcs-server Testcontainer.
type FakeGCSContainer struct {
	Container  testcontainers.Container
	Endpoint   string
	BucketName string
	ProjectID  string
}

// SetupFakeGCS starts a fake-gcs-server container and creates the requested bucket.
func SetupFakeGCS(t *testing.T, projectID, bucketName string) *FakeGCSContainer {
	t.Helper()

	ctx := context.Background()
	if projectID == "" {
		projectID = "test-project"
	}

	req := testcontainers.ContainerRequest{
		Image:        "fsouza/fake-gcs-server:latest",
		ExposedPorts: []string{"4443/tcp"},
		Cmd: []string{
			"-scheme", "http",
			"-port", "4443",
		},
		WaitingFor: wait.ForListeningPort("4443/tcp").WithStartupTimeout(2 * time.Minute),
	}

	c, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Fatalf("failed to start fake-gcs-server container: %v", err)
	}

	host, err := c.Host(ctx)
	if err != nil {
		_ = c.Terminate(ctx)
		t.Fatalf("failed to get fake-gcs host: %v", err)
	}
	port, err := c.MappedPort(ctx, "4443/tcp")
	if err != nil {
		_ = c.Terminate(ctx)
		t.Fatalf("failed to get fake-gcs port: %v", err)
	}

	endpoint := fmt.Sprintf("http://%s:%s", host, port.Port())

	// Patch external URL so that mediaLink / download URLs returned by the
	// server point to the testcontainer-mapped host:port (NewReader uses them).
	updateFakeGCSExternalURL(t, endpoint)

	EnsureFakeGCSBucket(t, endpoint, projectID, bucketName)

	return &FakeGCSContainer{
		Container:  c,
		Endpoint:   endpoint,
		BucketName: bucketName,
		ProjectID:  projectID,
	}
}

// Cleanup terminates the fake-gcs-server container. Safe to call multiple times.
func (g *FakeGCSContainer) Cleanup(t *testing.T) {
	t.Helper()
	if g == nil || g.Container == nil {
		return
	}
	_ = g.Container.Terminate(context.Background())
}

// EnsureFakeGCSBucket creates the given bucket on a fake-gcs-server endpoint.
// Safe to call multiple times; already-exists responses are ignored.
func EnsureFakeGCSBucket(t *testing.T, endpoint, projectID, bucket string) {
	t.Helper()

	body := map[string]any{"name": bucket}
	b, _ := json.Marshal(body)
	req, err := http.NewRequest(
		http.MethodPost,
		fmt.Sprintf("%s/storage/v1/b?project=%s", endpoint, projectID),
		bytes.NewReader(b),
	)
	if err != nil {
		t.Fatalf("failed to build fake-gcs create bucket request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("failed to create fake-gcs bucket: %v", err)
	}
	_ = resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusConflict {
		t.Fatalf("fake-gcs create bucket unexpected status: %d", resp.StatusCode)
	}
}

// updateFakeGCSExternalURL PUTs the mapped endpoint to /_internal/config so
// fake-gcs-server rewrites mediaLink / download URLs with the reachable host:port.
// Without this, NewReader receives a 0.0.0.0:4443 URL and downloads 404.
func updateFakeGCSExternalURL(t *testing.T, endpoint string) {
	t.Helper()

	body, _ := json.Marshal(map[string]string{"externalUrl": endpoint})
	req, err := http.NewRequest(
		http.MethodPut,
		endpoint+"/_internal/config",
		bytes.NewReader(body),
	)
	if err != nil {
		t.Fatalf("failed to build fake-gcs config request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("failed to patch fake-gcs externalUrl: %v", err)
	}
	_ = resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("fake-gcs config PUT unexpected status: %d", resp.StatusCode)
	}
}
