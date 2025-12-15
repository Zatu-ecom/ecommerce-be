package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"ecommerce-be/common/cache"
	"ecommerce-be/common/scheduler"
	"ecommerce-be/inventory/utils/constant"

	"github.com/google/uuid"
)

type ReservationExpiryItem struct {
	ReservationID uint
	ExpiresAt     time.Time
}

type ReservationShedulerService interface {
	ScheduleReservationExpiry(
		ctx context.Context,
		reservationID uint,
		sellerID uint,
		expiresAt time.Time,
	) (uuid.UUID, error)

	CancelReservationExpiryScheduler(
		ctx context.Context,
		sellerID, reservationID uint,
	) error

	ScheduleBulkReservationExpiry(
		ctx context.Context,
		sellerID uint,
		items []ReservationExpiryItem,
	) error

	CancelBulkReservationExpiryScheduler(
		ctx context.Context,
		sellerID uint,
		reservationIDs []uint,
	) error
}

type ReservationShedulerServiceImpl struct {
	scheduler scheduler.Scheduler
}

func NewReservationShedulerService(
	scheduler scheduler.Scheduler,
) *ReservationShedulerServiceImpl {
	return &ReservationShedulerServiceImpl{
		scheduler: scheduler,
	}
}

const (
	RESERVATION_SCHEDULER_CACHE_KEY_PREFIX = "inventory.reservation"
	SELLER_KEY_PREFIX                      = "seller"
)

func (s *ReservationShedulerServiceImpl) ScheduleReservationExpiry(
	ctx context.Context,
	reservationID uint,
	sellerID uint,
	expiresAt time.Time,
) (uuid.UUID, error) {
	payload := json.RawMessage(`{
		"reservationID": reservationID,
	}`)
	job := scheduler.NewJob(
		constant.INVENTORYY_RESERVATION_EXPRIY_EVENT_COMMAND,
		payload,
	)
	_, err := s.scheduler.Schedule(ctx, job, time.Until(expiresAt))
	if err != nil {
		return uuid.Nil, err
	}
	finalKey := s.generateSchedulerCacheKey(sellerID, reservationID)
	cache.Set(finalKey, job.JobID, time.Until(expiresAt)+time.Minute*3)
	return job.JobID, err
}

func (s *ReservationShedulerServiceImpl) CancelReservationExpiryScheduler(
	ctx context.Context,
	sellerID, reservationID uint,
) error {
	schedulerCacheKey := s.generateSchedulerCacheKey(sellerID, reservationID)
	jobIDStr, err := cache.Get(schedulerCacheKey)
	if err != nil {
		return err
	}
	return s.scheduler.Cancel(ctx, jobIDStr)
}

func (s *ReservationShedulerServiceImpl) ScheduleBulkReservationExpiry(
	ctx context.Context,
	sellerID uint,
	items []ReservationExpiryItem,
) error {
	for _, item := range items {
		_, err := s.ScheduleReservationExpiry(
			ctx,
			item.ReservationID,
			sellerID,
			item.ExpiresAt,
		)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *ReservationShedulerServiceImpl) CancelBulkReservationExpiryScheduler(
	ctx context.Context,
	sellerID uint,
	reservationIDs []uint,
) error {
	for _, reservationID := range reservationIDs {
		err := s.CancelReservationExpiryScheduler(
			ctx,
			sellerID,
			reservationID,
		)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *ReservationShedulerServiceImpl) generateSchedulerCacheKey(
	sellerID, reservationID uint,
) string {
	return fmt.Sprintf(
		"%s:%d:%s:%d",
		SELLER_KEY_PREFIX,
		sellerID,
		RESERVATION_SCHEDULER_CACHE_KEY_PREFIX,
		reservationID,
	)
}
