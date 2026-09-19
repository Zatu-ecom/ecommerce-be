package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"ecommerce-be/common/cache"
	"ecommerce-be/common/cachekit"
	"ecommerce-be/common/scheduler"
	"ecommerce-be/inventory/model"
	"ecommerce-be/inventory/utils/constant"

	"github.com/google/uuid"
)

// ReservationSchedulerService handles scheduling and cancellation of reservation expiry jobs
type ReservationSchedulerService interface {
	// ScheduleReservationExpiry schedules a single reservation expiry
	ScheduleReservationExpiry(
		ctx context.Context,
		sellerID uint,
		reservationID uint,
		expiresAt time.Time,
	) (uuid.UUID, error)

	// CancelReservationExpiry cancels a scheduled single reservation expiry
	CancelReservationExpiry(ctx context.Context, sellerID, reservationID uint) error

	// ScheduleBulkReservationExpiry schedules multiple reservation expiries as a single job
	ScheduleBulkReservationExpiry(
		ctx context.Context,
		sellerID uint,
		referenceID uint,
		reservationIDs []uint,
		expiresAt time.Time,
	) (uuid.UUID, error)

	// CancelBulkReservationExpiry cancels a scheduled bulk reservation expiry
	CancelBulkReservationExpiry(ctx context.Context, sellerID, referenceID uint) error
}

type reservationSchedulerServiceImpl struct {
	scheduler scheduler.Scheduler
}

func NewReservationSchedulerService(
	scheduler scheduler.Scheduler,
) ReservationSchedulerService {
	return &reservationSchedulerServiceImpl{
		scheduler: scheduler,
	}
}

// Cache key prefixes
const (
	singleReservationKeyPrefix = "seller:%d:inventory.reservation:%d"
	bulkReservationKeyPrefix   = "seller:%d:inventory.reservation.bulk:%d"
	cacheBufferDuration        = 3 * time.Minute
)

// ScheduleReservationExpiry schedules a single reservation expiry job
func (s *reservationSchedulerServiceImpl) ScheduleReservationExpiry(
	ctx context.Context,
	sellerID uint,
	reservationID uint,
	expiresAt time.Time,
) (uuid.UUID, error) {
	payload := model.ReservationExpiryPayload{
		ReservationID: reservationID,
		IsBulk:        false,
	}

	cacheKey := fmt.Sprintf(singleReservationKeyPrefix, sellerID, reservationID)
	return s.scheduleJob(ctx, payload, expiresAt, cacheKey)
}

// CancelReservationExpiry cancels a scheduled single reservation expiry
func (s *reservationSchedulerServiceImpl) CancelReservationExpiry(
	ctx context.Context,
	sellerID, reservationID uint,
) error {
	cacheKey := fmt.Sprintf(singleReservationKeyPrefix, sellerID, reservationID)
	return s.cancelJob(ctx, cacheKey)
}

// ScheduleBulkReservationExpiry schedules multiple reservation expiries as a single job
func (s *reservationSchedulerServiceImpl) ScheduleBulkReservationExpiry(
	ctx context.Context,
	sellerID uint,
	referenceID uint,
	reservationIDs []uint,
	expiresAt time.Time,
) (uuid.UUID, error) {
	payload := model.ReservationExpiryPayload{
		ReservationIDs: reservationIDs,
		ReferenceID:    referenceID,
		IsBulk:         true,
	}

	cacheKey := fmt.Sprintf(bulkReservationKeyPrefix, sellerID, referenceID)
	return s.scheduleJob(ctx, payload, expiresAt, cacheKey)
}

// CancelBulkReservationExpiry cancels a scheduled bulk reservation expiry
func (s *reservationSchedulerServiceImpl) CancelBulkReservationExpiry(
	ctx context.Context,
	sellerID, referenceID uint,
) error {
	cacheKey := fmt.Sprintf(bulkReservationKeyPrefix, sellerID, referenceID)
	return s.cancelJob(ctx, cacheKey)
}

// ============================================================================
// Private helper methods
// ============================================================================

// scheduleJob is a generic method to schedule a job with the given payload
func (s *reservationSchedulerServiceImpl) scheduleJob(
	ctx context.Context,
	payload model.ReservationExpiryPayload,
	expiresAt time.Time,
	cacheKey string,
) (uuid.UUID, error) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to marshal reservation payload: %w", err)
	}

	job := scheduler.NewJob(
		constant.INVENTORYY_RESERVATION_EXPRIY_EVENT_COMMAND,
		json.RawMessage(payloadBytes),
	)

	delay := time.Until(expiresAt)
	// The queue owns job identity: Schedule returns the queue-side ID (the
	// ZSET member Cancel must name). The envelope's job.JobID is NOT the
	// queue ID and must never be used for cancellation.
	queueID, err := s.scheduler.Schedule(ctx, job, delay)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to schedule reservation expiry: %w", err)
	}

	// Persist the queue ID for later cancellation on DURABLE KV (FR-016):
	// scheduler pointers are correctness state and must survive eviction
	// and restart. The legacy single-Redis fallback only applies when the
	// durable role is unwired (the scheduler would not run there either).
	// Buffer time keeps the pointer alive past the job's execution.
	cacheTTL := delay + cacheBufferDuration
	if d := cachekit.DefaultDurable(); d != nil {
		_ = d.Set(ctx, cacheKey, []byte(queueID), cacheTTL)
	} else {
		cache.Set(cacheKey, queueID, cacheTTL)
	}

	return job.JobID, nil
}

// cancelJob is a generic method to cancel a scheduled job
func (s *reservationSchedulerServiceImpl) cancelJob(ctx context.Context, cacheKey string) error {
	jobID, err := s.queuedJobID(ctx, cacheKey)
	if err != nil {
		return fmt.Errorf("failed to get job ID from cache: %w", err)
	}

	if err = s.scheduler.Cancel(ctx, jobID); err != nil {
		return fmt.Errorf("failed to cancel scheduled job: %w", err)
	}

	// Clean up the pointer after successful cancellation (both stores:
	// pre-fix pointers live in the legacy client).
	s.clearQueuedJobID(ctx, cacheKey)

	return nil
}

// queuedJobID resolves the queue-side job ID: durable first, legacy client
// as fallback (covers pre-fix pointers and unwired-durable environments).
func (s *reservationSchedulerServiceImpl) queuedJobID(ctx context.Context, cacheKey string) (string, error) {
	if d := cachekit.DefaultDurable(); d != nil {
		if b, err := d.Get(ctx, cacheKey); err == nil {
			return string(b), nil
		}
	}
	return cache.Get(cacheKey)
}

// clearQueuedJobID removes the cancellation pointer from both stores.
func (s *reservationSchedulerServiceImpl) clearQueuedJobID(ctx context.Context, cacheKey string) {
	if d := cachekit.DefaultDurable(); d != nil {
		_ = d.Del(ctx, cacheKey)
	}
	cache.Del(cacheKey)
}
