package scheduler

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"ecommerce-be/common/auth"
	"ecommerce-be/common/cachekit"
	"ecommerce-be/common/cachekit/provider"
	"ecommerce-be/common/config"
	commonErr "ecommerce-be/common/error"
)

// Scheduler handles scheduling delayed jobs for future execution on the
// durable KV role. Jobs are claimed atomically so exactly one pod processes
// each job. Transport is a cachekit.DelayQueue — this package never touches
// a backend client (provider blindness, 012).
type Scheduler struct {
	queue cachekit.DelayQueue
}

// New creates a Scheduler over the given durable queue. Callers obtain the
// queue from provider.NewDurable once at wiring time.
func New(queue cachekit.DelayQueue) *Scheduler {
	return &Scheduler{queue: queue}
}

// WiringQueue builds a durable queue from loaded config for factory wiring.
// When config is not loaded (some unit contexts) it uses zero config, whose
// operations fail closed via ErrUnavailable — the correct scheduler posture
// when durable KV is unreachable (pre-spec §8.3). A fresh client is built per
// call on purpose: test suites rebind container ports per run, so queue
// clients must never be cached across singleton resets.
func WiringQueue() cachekit.DelayQueue {
	var rc config.RedisConfig
	if cfg := config.Get(); cfg != nil {
		rc = cfg.Redis
	}
	return provider.NewDurable(rc)
}

// Schedule adds a job to the delayed jobs queue to be executed after the specified duration.
// Returns a jobId that can be used to cancel the job before execution.
// The returned ID is the transport's claim handle: pass it back to Cancel
// verbatim. (The envelope keeps the Job's own UUID for tracing; routing uses
// the transport ID.)
//
// How it works:
//  1. Job is serialized to JSON (ScheduledJob envelope, format unchanged)
//  2. The envelope is handed to the DelayQueue with the requested delay
//  3. Worker pool (StartRedisWorkerPool) picks up jobs when their execution time arrives
//
// Parameters:
//   - ctx: Context for the Redis operation (must contain UserID, CorrelationID; SellerID optional — use 0 when absent for platform-scoped jobs)
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
	scheduledJob, err := s.createScheduledJob(ctx, job)
	if err != nil {
		return "", err
	}

	// Envelope format is unchanged (ScheduledJob JSON) so dispatchers and
	// handlers keep working; only the transport moved to DelayQueue.
	data, err := json.Marshal(scheduledJob)
	if err != nil {
		return "", fmt.Errorf("failed to marshal job: %w", err)
	}

	jobID, err := s.queue.Schedule(ctx, data, after)
	if err != nil {
		return "", fmt.Errorf("failed to schedule job: %w", err)
	}

	return jobID, nil
}

// Cancel removes a scheduled job before it executes.
// Returns nil if job was successfully cancelled or doesn't exist.
//
// Parameters:
//   - ctx: Context for the operation
//   - jobId: The job ID returned from Schedule()
//
// Example:
//
//	err := scheduler.Cancel(ctx, jobId)
func (s *Scheduler) Cancel(ctx context.Context, jobID string) error {
	if err := s.queue.Cancel(ctx, jobID); err != nil {
		return fmt.Errorf("failed to cancel job: %w", err)
	}
	return nil
}

func (s *Scheduler) createScheduledJob(
	ctx context.Context,
	job Job,
) (*ScheduledJob, error) {
	userId, exist := auth.GetUserIDFromContext(ctx)
	if !exist {
		// return "", fmt.Errorf("user ID missing in context")
		return nil, commonErr.ErrUserDataMissing
	}
	sellerId, sellerExists := auth.GetSellerIDFromContext(ctx)
	if !sellerExists {
		// Platform-scoped flows (e.g. admin init-upload) have no seller tenant in context.
		// ScheduledJob.sellerId uses 0 as a sentinel; upload expiry cache keys use the
		// platform branch when the live request had no seller ID.
		sellerId = 0
	}
	correlationId, exist := auth.GetCorrelationIDFromContext(ctx)
	if !exist {
		return nil, commonErr.ErrCorrelationIDMissing
	}

	scheduledJob := ScheduledJob{
		Job:           &job,
		UserID:        userId,
		SellerID:      sellerId,
		CorrelationId: correlationId,
	}
	return &scheduledJob, nil
}
