package service

import (
	"context"
	"fmt"
	"time"

	"ecommerce-be/common/log"
	"ecommerce-be/promotion/repository"
)

// PromotionCronService handles scheduled background tasks for promotions
type PromotionCronService interface {
	SweepStatusTransitions()
}

type PromotionCronServiceImpl struct {
	repo             repository.PromotionRepository
	discountCodeRepo repository.DiscountCodeRepository
}

func NewPromotionCronService(
	repo repository.PromotionRepository,
	discountCodeRepo repository.DiscountCodeRepository,
) PromotionCronService {
	return &PromotionCronServiceImpl{
		repo:             repo,
		discountCodeRepo: discountCodeRepo,
	}
}

// SweepStatusTransitions automatically updates promotion and discount-code statuses
// based on their start/end dates.
func (s *PromotionCronServiceImpl) SweepStatusTransitions() {
	ctx := context.Background()
	now := time.Now()

	// 1. Auto-Start promotions: scheduled -> active
	startedCount, err := s.repo.AutoStartPromotions(ctx, now)
	if err != nil {
		log.ErrorWithContext(ctx, "Cron: Failed to auto-start promotions", err)
	} else if startedCount > 0 {
		log.InfoWithContext(ctx, fmt.Sprintf("Cron: Auto-started %d promotions", startedCount))
	}

	// 2. Auto-End promotions: active -> ended
	endedCount, err := s.repo.AutoEndPromotions(ctx, now)
	if err != nil {
		log.ErrorWithContext(ctx, "Cron: Failed to auto-end promotions", err)
	} else if endedCount > 0 {
		log.InfoWithContext(ctx, fmt.Sprintf("Cron: Auto-ended %d promotions", endedCount))
	}

	// 3. Auto-Start discount codes: inactive -> active
	dcStarted, err := s.discountCodeRepo.AutoStartDiscountCodes(ctx, now)
	if err != nil {
		log.ErrorWithContext(ctx, "Cron: Failed to auto-start discount codes", err)
	} else if dcStarted > 0 {
		log.InfoWithContext(ctx, fmt.Sprintf("Cron: Auto-started %d discount codes", dcStarted))
	}

	// 4. Auto-End discount codes: active -> inactive
	dcEnded, err := s.discountCodeRepo.AutoEndDiscountCodes(ctx, now)
	if err != nil {
		log.ErrorWithContext(ctx, "Cron: Failed to auto-end discount codes", err)
	} else if dcEnded > 0 {
		log.InfoWithContext(ctx, fmt.Sprintf("Cron: Auto-ended %d discount codes", dcEnded))
	}
}
