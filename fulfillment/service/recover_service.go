package service

import (
	"context"
	"fmt"
	"time"

	"ecommerce-be/common/log"
	"ecommerce-be/fulfillment/entity"
	"ecommerce-be/fulfillment/repository"
)

// recoverGracePeriod lets just-marked drafts settle before the sweeper
// considers them (avoids racing an in-flight book call).
const recoverGracePeriod = 2 * time.Minute

// recoverBatchSize bounds one sweep (SKIP LOCKED semantics live in the
// repository query contract; the service additionally filters by age and
// attempts so manual-only drafts are never touched).
const recoverBatchSize = 100

// maxBookAttempts caps AUTOMATIC retries. Manual clicks always re-attempt
// (the seller is watching); the cron leaves attempts>=5 alone with an alert.
const maxBookAttempts = 5

// RecoverService finishes interrupted bookings: drafts marked for booking
// (book_requested_at set) but still lacking provider ids. Never-attempted
// drafts wait for a human — the job must not book them.
type RecoverService interface {
	RecoverDrafts(ctx context.Context) (int, error)
}

// RecoverServiceImpl implements RecoverService.
type RecoverServiceImpl struct {
	shipmentRepo repository.ShipmentRepository
	shipments    ShipmentService
}

// NewRecoverService builds the recover job. ShipmentService is the same
// booking path handlers use, so recovery and clicks can never diverge.
func NewRecoverService(
	shipmentRepo repository.ShipmentRepository,
	shipments ShipmentService,
) RecoverService {
	return &RecoverServiceImpl{
		shipmentRepo: shipmentRepo,
		shipments:    shipments,
	}
}

// RecoverDrafts replays interrupted books: attempted drafts without provider
// ids, under the attempt cap, past the grace period. Returns recovered count.
func (s *RecoverServiceImpl) RecoverDrafts(ctx context.Context) (int, error) {
	drafts, err := s.shipmentRepo.FindRecoverableDrafts(ctx, recoverBatchSize)
	if err != nil {
		return 0, err
	}
	recovered := 0
	for i := range drafts {
		draft := &drafts[i]
		if draft.BookRequestedAt == nil {
			continue
		}
		if time.Since(*draft.BookRequestedAt) < recoverGracePeriod {
			continue
		}
		if draft.BookAttempts >= maxBookAttempts {
			log.ErrorWithContext(ctx, "recover: draft past attempt cap, needs a human",
				fmt.Errorf("shipment %d capped at %d attempts", draft.ID, maxBookAttempts))
			continue
		}
		if draft.ProviderCode == nil || *draft.ProviderCode == "" {
			continue
		}
		booked, err := s.shipments.BookDraft(ctx, draft.SellerID, draft.ID, BookDraftOptions{
			ProviderCode: *draft.ProviderCode,
		})
		if err != nil {
			log.ErrorWithContext(ctx, "recover: replay failed, will retry next tick", err)
			continue
		}
		if booked.Status == entity.SHIPMENT_STATUS_BOOKED ||
			booked.Status == entity.SHIPMENT_STATUS_PICKUP_SCHEDULED {
			recovered++
		}
	}
	return recovered, nil
}
