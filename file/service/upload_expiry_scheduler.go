package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"ecommerce-be/common/cachekit"
	"ecommerce-be/common/scheduler"
	"ecommerce-be/file/utils/constant"
)

// UploadExpiryScheduler wraps common/scheduler to schedule and cancel
// upload-expiry jobs for file_object rows that are still in UPLOADING status.
//
// The scheduler job payload carries correlationID so the handler can emit
// structured log lines with the original request's correlation ID (CA3).
//
// Cache keys are scoped by identity:
//   - Seller: "seller:{sellerId}:file.upload.expiry:{fileObjectId}"
//   - Platform: "platform:file.upload.expiry:{fileObjectId}"
type UploadExpiryScheduler interface {
	// Schedule enqueues an expiry job to run at runAt.
	// Returns the scheduler-generated jobID (UUID string).
	Schedule(
		ctx context.Context,
		fileObjectID uint64,
		fileID string,
		sellerID *uint64,
		runAt time.Time,
		correlationID string,
	) (jobID string, err error)

	// Cancel removes the previously scheduled expiry job for the given file_object.
	// Returns nil if the job doesn't exist or was already executed (idempotent).
	Cancel(
		ctx context.Context,
		fileObjectID uint64,
		sellerID *uint64,
	) error
}

// UploadExpiryPayload is the job payload stored by common/scheduler.
// CA3: correlationID is embedded so the handler can extract it for log context.
type UploadExpiryPayload struct {
	FileObjectID  uint64 `json:"fileObjectId"`
	FileID        string `json:"fileId"`
	CorrelationID string `json:"correlationId"`
}

type uploadExpiryScheduler struct {
	scheduler *scheduler.Scheduler
}

// NewUploadExpiryScheduler creates a new UploadExpiryScheduler.
func NewUploadExpiryScheduler(s *scheduler.Scheduler) UploadExpiryScheduler {
	return &uploadExpiryScheduler{scheduler: s}
}

// Schedule stores an expiry job with the common scheduler and caches the jobID
// under the identity-scoped key for later cancellation.
func (s *uploadExpiryScheduler) Schedule(
	ctx context.Context,
	fileObjectID uint64,
	fileID string,
	sellerID *uint64,
	runAt time.Time,
	correlationID string,
) (string, error) {
	payload := UploadExpiryPayload{
		FileObjectID:  fileObjectID,
		FileID:        fileID,
		CorrelationID: correlationID,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("upload expiry scheduler: marshal payload: %w", err)
	}

	job := scheduler.NewJob(
		constant.SchedulerCommandUploadExpiry,
		json.RawMessage(payloadBytes),
	)

	delay := time.Until(runAt)
	jobID, err := s.scheduler.Schedule(ctx, job, delay)
	if err != nil {
		return "", fmt.Errorf("upload expiry scheduler: schedule: %w", err)
	}

	// Persist the queue-side jobID for Cancel on DURABLE KV (FR-016):
	// scheduler pointers are correctness state and must survive eviction
	// and restart. Durable is mandatory here — without it uploads cannot be
	// cancelled later, so fail the schedule instead of hiding the outage.
	d := cachekit.DefaultDurable()
	if d == nil {
		return "", fmt.Errorf("upload expiry scheduler: durable KV unwired")
	}
	cacheTTL := delay + constant.CacheBufferDuration
	cacheKey := s.cacheKey(fileObjectID, sellerID)
	_ = d.Set(ctx, cacheKey, []byte(jobID), cacheTTL)

	return jobID, nil
}

// Cancel retrieves the cached jobID and invokes the scheduler's Cancel.
// Best-effort: if the cache key is missing (TTL elapsed or Redis unavailable),
// we attempt to cancel with an empty jobID — which is a no-op in the scheduler.
func (s *uploadExpiryScheduler) Cancel(
	ctx context.Context,
	fileObjectID uint64,
	sellerID *uint64,
) error {
	cacheKey := s.cacheKey(fileObjectID, sellerID)

	jobID := s.queuedJobID(ctx, cacheKey)
	if jobID == "" {
		// Pointer miss (TTL expired or store unavailable). Return nil — the
		// scheduler handler is idempotent against ACTIVE rows (FR-029).
		return nil
	}

	if err := s.scheduler.Cancel(ctx, jobID); err != nil {
		return fmt.Errorf("upload expiry scheduler: cancel: %w", err)
	}

	if d := cachekit.DefaultDurable(); d != nil {
		_ = d.Del(ctx, cacheKey)
	}
	return nil
}

// queuedJobID resolves the queue-side job ID from durable KV. Misses (TTL
// expiry, already-run jobs) return "" and Cancel short-circuits to nil.
func (s *uploadExpiryScheduler) queuedJobID(ctx context.Context, cacheKey string) string {
	d := cachekit.DefaultDurable()
	if d == nil {
		return ""
	}
	b, err := d.Get(ctx, cacheKey)
	if err != nil {
		return ""
	}
	return string(b)
}

// cacheKey returns the Redis key for the expiry job ID.
// Seller:   "seller:{sellerId}:file.upload.expiry:{fileObjectId}"
// Platform: "platform:file.upload.expiry:{fileObjectId}"
func (s *uploadExpiryScheduler) cacheKey(fileObjectID uint64, sellerID *uint64) string {
	if sellerID != nil {
		return fmt.Sprintf(constant.SellerSchedulerKeyFmt, *sellerID, fileObjectID)
	}
	return fmt.Sprintf(constant.PlatformSchedulerKeyFmt, fileObjectID)
}
