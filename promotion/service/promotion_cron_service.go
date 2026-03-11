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
	repo repository.PromotionRepository
}

func NewPromotionCronService(repo repository.PromotionRepository) PromotionCronService {
	return &PromotionCronServiceImpl{
		repo: repo,
	}
}

// SweepStatusTransitions automatically updates promotion statuses based on their start/end dates
func (s *PromotionCronServiceImpl) SweepStatusTransitions() {
	ctx := context.Background()
	now := time.Now()

	// 1. Auto-Start: scheduled -> active
	startedCount, err := s.repo.AutoStartPromotions(ctx, now)
	if err != nil {
		log.ErrorWithContext(ctx, "Cron: Failed to auto-start promotions", err)
	} else if startedCount > 0 {
		log.InfoWithContext(ctx, fmt.Sprintf("Cron: Auto-started %d promotions", startedCount))
	}

	// 2. Auto-End: active -> ended
	endedCount, err := s.repo.AutoEndPromotions(ctx, now)
	if err != nil {
		log.ErrorWithContext(ctx, "Cron: Failed to auto-end promotions", err)
	} else if endedCount > 0 {
		log.InfoWithContext(ctx, fmt.Sprintf("Cron: Auto-ended %d promotions", endedCount))
	}
}
