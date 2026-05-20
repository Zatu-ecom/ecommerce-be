package file_test

import (
	"context"
	"fmt"
	"net/url"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// MinioContainer holds the runtime details of a running MinIO Testcontainer.
type MinioContainer struct {
	Container  testcontainers.Container
	Endpoint   string
	AccessKey  string
	SecretKey  string
	BucketName string
	Region     string
}

// SetupMinio starts a MinIO container and creates the requested bucket.
// The container is terminated automatically when Cleanup is called.
func SetupMinio(t *testing.T, bucketName string) *MinioContainer {
	t.Helper()

	ctx := context.Background()
	accessKey := "minioadmin"
	secretKey := "minioadmin"
	region := "us-east-1"

	req := testcontainers.ContainerRequest{
		Image:        "minio/minio:latest",
		ExposedPorts: []string{"9000/tcp", "9001/tcp"},
		Env: map[string]string{
			"MINIO_ROOT_USER":     accessKey,
			"MINIO_ROOT_PASSWORD": secretKey,
		},
		Cmd:        []string{"server", "/data", "--console-address", ":9001"},
		WaitingFor: wait.ForLog("API:").WithStartupTimeout(2 * time.Minute),
	}

	c, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Fatalf("failed to start minio container: %v", err)
	}

	host, err := c.Host(ctx)
	if err != nil {
		_ = c.Terminate(ctx)
		t.Fatalf("failed to get minio host: %v", err)
	}
	port, err := c.MappedPort(ctx, "9000/tcp")
	if err != nil {
		_ = c.Terminate(ctx)
		t.Fatalf("failed to get minio port: %v", err)
	}

	endpoint := fmt.Sprintf("http://%s:%s", host, port.Port())

	EnsureS3Bucket(t, endpoint, region, accessKey, secretKey, bucketName)

	return &MinioContainer{
		Container:  c,
		Endpoint:   endpoint,
		AccessKey:  accessKey,
		SecretKey:  secretKey,
		BucketName: bucketName,
		Region:     region,
	}
}

// Cleanup terminates the MinIO container. Safe to call multiple times.
func (m *MinioContainer) Cleanup(t *testing.T) {
	t.Helper()
	if m == nil || m.Container == nil {
		return
	}
	_ = m.Container.Terminate(context.Background())
}

// EnsureS3Bucket creates the given bucket on a MinIO/S3-compatible endpoint.
// Safe to call multiple times; already-exists errors are ignored.
func EnsureS3Bucket(t *testing.T, endpoint, region, accessKey, secretKey, bucket string) {
	t.Helper()

	if _, err := url.Parse(endpoint); err != nil {
		t.Fatalf("invalid minio endpoint: %v", err)
	}

	cfg, err := config.LoadDefaultConfig(
		context.Background(),
		config.WithRegion(region),
		config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(accessKey, secretKey, ""),
		),
		config.WithEndpointResolverWithOptions(
			aws.EndpointResolverWithOptionsFunc(
				func(service, r string, _ ...any) (aws.Endpoint, error) {
					if service == s3.ServiceID {
						return aws.Endpoint{
							URL:               endpoint,
							SigningRegion:     region,
							HostnameImmutable: true,
						}, nil
					}
					return aws.Endpoint{}, &aws.EndpointNotFoundError{}
				},
			),
		),
	)
	if err != nil {
		t.Fatalf("failed to load aws config for minio: %v", err)
	}

	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.UsePathStyle = true
	})

	_, _ = client.CreateBucket(
		context.Background(),
		&s3.CreateBucketInput{Bucket: aws.String(bucket)},
	)
	// Accept both success (nil) and "already exists" errors; fatal on setup errors is done above.
}
