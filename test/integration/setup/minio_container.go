package setup

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// MinioContainer captures runtime details of an integration MinIO instance.
type MinioContainer struct {
	Container testcontainers.Container
	Endpoint  string
	AccessKey string
	SecretKey string
	Region    string
	ctx       context.Context
}

// SetupMinioContainer boots a MinIO test container.
func SetupMinioContainer(t *testing.T) *MinioContainer {
	t.Helper()

	ctx := context.Background()
	accessKey := "minioadmin"
	secretKey := "minioadmin"
	region := "us-east-1"

	container, err := testcontainers.GenericContainer(
		ctx,
		testcontainers.GenericContainerRequest{
			ContainerRequest: testcontainers.ContainerRequest{
				Image:        "quay.io/minio/minio:latest",
				ExposedPorts: []string{"9000/tcp", "9001/tcp"},
				Env: map[string]string{
					"MINIO_ROOT_USER":     accessKey,
					"MINIO_ROOT_PASSWORD": secretKey,
				},
				Cmd: []string{"server", "/data", "--console-address", ":9001"},
				WaitingFor: wait.ForLog("API:").
					WithStartupTimeout(3 * time.Minute),
			},
			Started: true,
		},
	)
	if err != nil {
		t.Fatalf("failed to start minio container: %v", err)
	}

	host, err := container.Host(ctx)
	if err != nil {
		_ = container.Terminate(ctx)
		t.Fatalf("failed to resolve minio host: %v", err)
	}
	port, err := container.MappedPort(ctx, "9000")
	if err != nil {
		_ = container.Terminate(ctx)
		t.Fatalf("failed to resolve minio mapped port: %v", err)
	}

	endpoint := fmt.Sprintf("http://%s:%s", host, port.Port())
	return &MinioContainer{
		Container: container,
		Endpoint:  endpoint,
		AccessKey: accessKey,
		SecretKey: secretKey,
		Region:    region,
		ctx:       ctx,
	}
}

// CreateBucket creates a bucket on MinIO; safe to call repeatedly.
func (m *MinioContainer) CreateBucket(ctx context.Context, name string) error {
	cfg, err := config.LoadDefaultConfig(
		ctx,
		config.WithRegion(m.Region),
		config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(m.AccessKey, m.SecretKey, ""),
		),
		config.WithEndpointResolverWithOptions(
			aws.EndpointResolverWithOptionsFunc(
				func(service, region string, options ...any) (aws.Endpoint, error) {
					if service == s3.ServiceID {
						return aws.Endpoint{
							URL:               m.Endpoint,
							SigningRegion:     m.Region,
							HostnameImmutable: true,
						}, nil
					}
					return aws.Endpoint{}, &aws.EndpointNotFoundError{}
				},
			),
		),
	)
	if err != nil {
		return err
	}

	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.UsePathStyle = true
	})
	_, err = client.CreateBucket(ctx, &s3.CreateBucketInput{Bucket: aws.String(name)})
	return err
}

// Cleanup terminates MinIO container.
func (m *MinioContainer) Cleanup(t *testing.T) {
	t.Helper()
	if m == nil || m.Container == nil {
		return
	}
	_ = m.Container.Terminate(m.ctx)
}
