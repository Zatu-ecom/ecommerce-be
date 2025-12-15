package scheduler

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/go-redis/redis/v8"
)

// Scheduler handles scheduling delayed jobs to Redis for future execution.
// Jobs are stored in a Redis Sorted Set with the execution timestamp as the score.
// Each job also has a separate key for cancellation support.
type Scheduler struct {
	rdb *redis.Client
}

// New creates a new Scheduler instance with the provided Redis client.
func New(rdb *redis.Client) *Scheduler {
	return &Scheduler{rdb: rdb}
}

// Schedule adds a job to the delayed jobs queue to be executed after the specified duration.
// Returns a jobId that can be used to cancel the job before execution.
//
// How it works:
//  1. Generate unique jobId (UUID)
//  2. Job is serialized to JSON
//  3. Job JSON is stored in "scheduled_job:{jobId}" for cancellation support
//  4. Job is added to Redis Sorted Set "delayed_jobs" with score = execution timestamp
//  5. Worker pool (StartRedisWorkerPool) picks up jobs when their execution time arrives
//
// Parameters:
//   - ctx: Context for the Redis operation (must contain UserID, SellerID, CorrelationID)
//   - job: The Job struct containing command and payload
//   - after: Duration to wait before executing the job (e.g., 15*time.Minute)
//
// Returns:
//   - jobId: Unique identifier to cancel this job later
//   - error: Any error during scheduling
//
// Example:
//
//	job := scheduler.Job{
//	    Command: "expire_reservation",
//	    Payload: json.RawMessage(`{"reservationId": 123}`),
//	}
//	jobId, err := scheduler.Schedule(ctx, job, 15*time.Minute)
//	// Store jobId to cancel later if needed
func (s *Scheduler) Schedule(ctx context.Context, job Job, after time.Duration) (string, error) {
	jobID := uuid.New().String()

	scheduledJob := ScheduledJob{
		Job:           job,
		UserID:        getStringFromContext(ctx, UserIDKey),
		SellerID:      getStringFromContext(ctx, SellerIDKey),
		CorrelationId: getStringFromContext(ctx, CorrelationIDKey),
	}

	data, err := json.Marshal(scheduledJob)
	if err != nil {
		return "", fmt.Errorf("failed to marshal job: %w", err)
	}

	executeAt := time.Now().Add(after).Unix()
	jobKey := scheduledJobKeyPrefix + jobID

	// Use pipeline for atomic operations
	pipe := s.rdb.Pipeline()

	// Store job JSON for cancellation lookup
	pipe.Set(ctx, jobKey, data, after+time.Hour) // TTL = execution time + 1 hour buffer

	// Add to sorted set for scheduling
	pipe.ZAdd(ctx, delayedJobsKey, &redis.Z{
		Score:  float64(executeAt),
		Member: data,
	})

	_, err = pipe.Exec(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to schedule job: %w", err)
	}

	return jobID, nil
}

// Cancel removes a scheduled job before it executes.
// Returns nil if job was successfully cancelled or doesn't exist.
//
// How it works:
//  1. Get job JSON from "scheduled_job:{jobId}"
//  2. Remove job from sorted set "delayed_jobs" using exact JSON match
//  3. Delete the "scheduled_job:{jobId}" key
//
// Parameters:
//   - ctx: Context for the Redis operation
//   - jobId: The job ID returned from Schedule()
//
// Example:
//
//	err := scheduler.Cancel(ctx, jobId)
func (s *Scheduler) Cancel(ctx context.Context, jobID string) error {
	jobKey := scheduledJobKeyPrefix + jobID

	// Get job JSON
	jobData, err := s.rdb.Get(ctx, jobKey).Result()
	if err == redis.Nil {
		// Job doesn't exist (already executed or cancelled)
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to get job data: %w", err)
	}

	// Use pipeline for atomic operations
	pipe := s.rdb.Pipeline()

	// Remove from sorted set
	pipe.ZRem(ctx, delayedJobsKey, jobData)

	// Delete the job key
	pipe.Del(ctx, jobKey)

	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to cancel job: %w", err)
	}

	return nil
}

// getStringFromContext safely extracts a string value from context
func getStringFromContext(ctx context.Context, key contextKey) string {
	if val := ctx.Value(key); val != nil {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}
