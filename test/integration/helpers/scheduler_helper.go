package helpers

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"ecommerce-be/common/scheduler"
	fileService "ecommerce-be/file/service"

	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/require"
)

// AssertSchedulerJobExists asserts a delayed file.upload.expiry job exists for the given fileObjectID.
func AssertSchedulerJobExists(t *testing.T, redisClient *redis.Client, fileObjectID uint64) {
	t.Helper()
	found := findExpiryJob(t, redisClient, fileObjectID)
	require.True(t, found, "expected scheduler job for fileObjectID=%d", fileObjectID)
}

// AssertNoSchedulerJob asserts no delayed file.upload.expiry job exists for the given fileObjectID.
func AssertNoSchedulerJob(t *testing.T, redisClient *redis.Client, fileObjectID uint64) {
	t.Helper()
	found := findExpiryJob(t, redisClient, fileObjectID)
	require.False(t, found, "did not expect scheduler job for fileObjectID=%d", fileObjectID)
}

// FastForwardExpiry rewrites matching delayed job score to now-1, forcing near-immediate execution.
func FastForwardExpiry(t *testing.T, redisClient *redis.Client, fileObjectID uint64) {
	t.Helper()
	ctx := context.Background()
	members, err := redisClient.ZRange(ctx, "delayed_jobs", 0, -1).Result()
	require.NoError(t, err)

	for _, m := range members {
		if matchesFileObjectID(m, fileObjectID) {
			err := redisClient.ZAdd(ctx, "delayed_jobs", &redis.Z{
				Score:  float64(time.Now().Add(-1 * time.Second).Unix()),
				Member: m,
			}).Err()
			require.NoError(t, err)
		}
	}
}

func findExpiryJob(t *testing.T, redisClient *redis.Client, fileObjectID uint64) bool {
	t.Helper()
	ctx := context.Background()
	members, err := redisClient.ZRange(ctx, "delayed_jobs", 0, -1).Result()
	require.NoError(t, err)
	for _, m := range members {
		if matchesFileObjectID(m, fileObjectID) {
			return true
		}
	}
	return false
}

func matchesFileObjectID(raw string, fileObjectID uint64) bool {
	var job scheduler.ScheduledJob
	if err := json.Unmarshal([]byte(raw), &job); err != nil {
		return false
	}
	if job.Job == nil || job.Command != "file.upload.expiry" {
		return false
	}

	var payload fileService.UploadExpiryPayload
	if err := json.Unmarshal(job.Payload, &payload); err != nil {
		return false
	}
	return payload.FileObjectID == fileObjectID
}
