package setup

import (
	"context"
	"strings"
	"sync"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// Shared auxiliary containers (test infra only).
//
// MinIO and RabbitMQ boots (~5-15s each) were repeated per suite. Both are
// now shared per `go test` package process: one container each, reused across
// suites. Buckets are (re)ensured per caller and purged on demand; RabbitMQ
// hands out a fresh AMQP connection per caller over the shared broker.

var (
	sharedMinioMu sync.Mutex
	sharedMinio   *MinioContainer

	sharedRabbitMu        sync.Mutex
	sharedRabbitContainer *RabbitMQContainer
)

// acquireSharedMinio returns the process-wide MinIO instance, starting it on
// first use. Callers still create/ensure their own buckets.
func acquireSharedMinio(t *testing.T) *MinioContainer {
	t.Helper()
	sharedMinioMu.Lock()
	defer sharedMinioMu.Unlock()

	if sharedMinio != nil {
		if err := pingMinio(context.Background(), sharedMinio); err == nil {
			return sharedMinio
		}
		t.Logf("shared minio died; starting a replacement")
		_ = sharedMinio.Container.Terminate(context.Background())
		sharedMinio = nil
	}
	sharedMinio = startMinioContainer(t)
	return sharedMinio
}

func pingMinio(ctx context.Context, m *MinioContainer) error {
	if m == nil {
		return context.Canceled
	}
	client, err := m.s3Client(ctx)
	if err != nil {
		return err
	}
	_, err = client.ListBuckets(ctx, &s3.ListBucketsInput{})
	return err
}

// isBucketExistsError reports S3 already-exists conditions so shared-bucket
// reuse stays idempotent.
func isBucketExistsError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	for _, s := range []string{
		"bucketalreadyownedbyyou", "bucketalreadyexists", "already exists",
	} {
		if strings.Contains(msg, strings.ToLower(s)) {
			return true
		}
	}
	return false
}

// PurgeBucket deletes all objects (all versions omitted — tests use
// non-versioned buckets) so a reused bucket starts empty.
func (m *MinioContainer) PurgeBucket(ctx context.Context, bucket string) error {
	client, err := m.s3Client(ctx)
	if err != nil {
		return err
	}
	var contToken *string
	for {
		out, err := client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
			Bucket:            aws.String(bucket),
			ContinuationToken: contToken,
		})
		if err != nil {
			if strings.Contains(strings.ToLower(err.Error()), "nosuchbucket") {
				return nil
			}
			return err
		}
		if len(out.Contents) > 0 {
			ids := make([]types.ObjectIdentifier, 0, len(out.Contents))
			for _, o := range out.Contents {
				ids = append(ids, types.ObjectIdentifier{Key: o.Key})
			}
			if _, err := client.DeleteObjects(ctx, &s3.DeleteObjectsInput{
				Bucket: aws.String(bucket),
				Delete: &types.Delete{Objects: ids, Quiet: aws.Bool(true)},
			}); err != nil {
				return err
			}
		}
		if !aws.ToBool(out.IsTruncated) {
			return nil
		}
		contToken = out.NextContinuationToken
	}
}
