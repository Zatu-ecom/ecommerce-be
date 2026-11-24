package helpers

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"ecommerce-be/common/scheduler"
	fileService "ecommerce-be/file/service"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

// Scheduler keyspace (mirrors the DelayQueue layout): queue members are job
// IDs; payloads live under the index key. Helpers resolve payloads through
// the index so assertions survive member-format changes.
const (
	delayedJobsKey        = "delayed_jobs"
	scheduledJobKeyPrefix = "scheduled_job:"
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
	for _, id := range matchingJobIDs(t, redisClient, fileObjectID) {
		err := redisClient.ZAdd(ctx, delayedJobsKey, redis.Z{
			Score:  float64(time.Now().Add(-1 * time.Second).Unix()),
			Member: id,
		}).Err()
		require.NoError(t, err)
	}
}

func findExpiryJob(t *testing.T, redisClient *redis.Client, fileObjectID uint64) bool {
	t.Helper()
	return len(matchingJobIDs(t, redisClient, fileObjectID)) > 0
}

// matchingJobIDs returns queue member IDs whose indexed payload targets the
// given file object. Payloads (not members) carry the job JSON.
func matchingJobIDs(t *testing.T, redisClient *redis.Client, fileObjectID uint64) []string {
	t.Helper()
	ctx := context.Background()
	members, err := redisClient.ZRange(ctx, delayedJobsKey, 0, -1).Result()
	require.NoError(t, err)
	var matched []string
	for _, id := range members {
		raw, err := redisClient.Get(ctx, scheduledJobKeyPrefix+id).Bytes()
		if err != nil {
			continue
		}
		if matchesFileObjectID(string(raw), fileObjectID) {
			matched = append(matched, id)
		}
	}
	return matched
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
