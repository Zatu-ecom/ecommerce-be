package service

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"ecommerce-be/common/db"
	commonError "ecommerce-be/common/error"
	"ecommerce-be/common/log"
	"ecommerce-be/promotion/entity"
	promoErrors "ecommerce-be/promotion/error"
	"ecommerce-be/promotion/factory"
	"ecommerce-be/promotion/model"
	"ecommerce-be/promotion/service/promotionStrategy"
)

// CreatePromotion creates a new promotion
func (s *PromotionServiceImpl) CreatePromotion(
	ctx context.Context,
	req model.CreatePromotionRequest,
	sellerID uint,
) (*model.PromotionResponse, error) {
	log.InfoWithContext(ctx, "Creating new promotion")

	// Validate discount config using strategy pattern
	strategy := promotionStrategy.GetPromotionStrategy(req.PromotionType)
	if strategy == nil {
		return nil, promoErrors.ErrInvalidDiscountConfig.WithMessage("Unsupported promotion type")
	}

	if err := strategy.ValidateConfig(req.DiscountConfig); err != nil {
		log.ErrorWithContext(ctx, "Invalid discount config", err)
		return nil, err
	}

	// Validate date ranges
	if err := s.validateDateRanges(req.StartsAt, req.EndsAt); err != nil {
		log.ErrorWithContext(ctx, "Invalid date ranges", err)
		return nil, err
	}

	// Validate eligibility
	if err := s.validateEligibility(req.EligibleFor, req.CustomerSegmentID); err != nil {
		log.ErrorWithContext(ctx, "Invalid eligibility", err)
		return nil, err
	}

	// Check for duplicate slug if provided
	if req.Slug != nil && *req.Slug != "" {
		existingPromotion, err := s.promotionRepo.FindBySlug(ctx, *req.Slug, sellerID)
		if err != nil {
			log.ErrorWithContext(ctx, "Error checking slug uniqueness", err)
			return nil, err
		}
		if existingPromotion != nil {
			return nil, promoErrors.ErrPromotionSlugExists
		}
	}

	// Convert request to entity
	promotion := factory.PromotionRequestToEntity(req, sellerID)

	// Create promotion in database
	if err := s.promotionRepo.Create(ctx, promotion); err != nil {
		log.ErrorWithContext(ctx, "Failed to create promotion", err)
		return nil, commonError.NewAppError(
			"PROMOTION_CREATE_FAILED",
			"Failed to create promotion",
			http.StatusInternalServerError,
		)
	}

	log.InfoWithContext(ctx, "Promotion created successfully")

	// Convert entity to response
	response := factory.PromotionEntityToResponse(promotion)
	return response, nil
}

// UpdatePromotion updates a promotion within a transaction
func (s *PromotionServiceImpl) UpdatePromotion(
	ctx context.Context,
	id uint,
	req model.UpdatePromotionRequest,
	sellerID uint,
) (*model.PromotionResponse, error) {
	log.InfoWithContext(ctx, fmt.Sprintf("Updating promotion %d", id))

	return db.WithTransactionResult(
		ctx,
		func(txCtx context.Context) (*model.PromotionResponse, error) {
			existing, err := s.promotionRepo.FindByID(txCtx, id)
			if err != nil {
				return nil, promoErrors.ErrPromotionNotFound
			}

			if err := s.validateUpdateGuards(existing, req, sellerID); err != nil {
				return nil, err
			}
			if err := s.validateSlugUniqueness(txCtx, req.Slug, existing); err != nil {
				return nil, err
			}
			if err := s.validateUpdatedConfig(existing, req); err != nil {
				return nil, err
			}

			updated := factory.ApplyUpdatePromotionRequest(existing, req)
			if err := s.promotionRepo.Update(txCtx, updated); err != nil {
				log.ErrorWithContext(txCtx, "Failed to update promotion", err)
				return nil, promoErrors.ErrPromotionUpdateFailed
			}

			log.InfoWithContext(txCtx, fmt.Sprintf("Promotion updated successfully: %d", id))
			return factory.PromotionEntityToResponse(updated), nil
		},
	)
}

// validateUpdateGuards checks ownership, terminal status, and active field restrictions
func (s *PromotionServiceImpl) validateUpdateGuards(
	existing *entity.Promotion,
	req model.UpdatePromotionRequest,
	sellerID uint,
) error {
	if existing.SellerID != sellerID {
		return promoErrors.ErrUnauthorizedPromotionAccess
	}
	if existing.Status == entity.StatusEnded || existing.Status == entity.StatusCancelled {
		return promoErrors.ErrCannotEditTerminalPromotion
	}
	if existing.Status == entity.StatusActive {
		if req.PromotionType != nil || req.DiscountConfig != nil ||
			req.AppliesTo != nil || req.StartsAt != nil {
			return promoErrors.ErrCannotEditActivePromotion
		}
	}
	return nil
}

// validateSlugUniqueness checks if the new slug collides with another promotion
func (s *PromotionServiceImpl) validateSlugUniqueness(
	ctx context.Context,
	newSlug *string,
	existing *entity.Promotion,
) error {
	if newSlug == nil || *newSlug == "" {
		return nil
	}
	var existingSlug string
	if existing.Slug != nil {
		existingSlug = *existing.Slug
	}
	if *newSlug == existingSlug {
		return nil
	}
	collision, err := s.promotionRepo.FindBySlug(ctx, *newSlug, existing.SellerID)
	if err != nil {
		return err
	}
	if collision != nil && collision.ID != existing.ID {
		return promoErrors.ErrPromotionSlugExists
	}
	return nil
}

// validateUpdatedConfig re-validates discount config and date ranges when changed
func (s *PromotionServiceImpl) validateUpdatedConfig(
	existing *entity.Promotion,
	req model.UpdatePromotionRequest,
) error {
	if req.PromotionType != nil || req.DiscountConfig != nil {
		newType := existing.PromotionType
		if req.PromotionType != nil {
			newType = *req.PromotionType
		}
		newConfig := existing.DiscountConfig
		if req.DiscountConfig != nil {
			newConfig = entity.DiscountConfig(*req.DiscountConfig)
		}
		strategy := promotionStrategy.GetPromotionStrategy(newType)
		if strategy == nil {
			return promoErrors.ErrInvalidDiscountConfig.WithMessage("Unsupported promotion type")
		}
		if err := strategy.ValidateConfig(newConfig); err != nil {
			return err
		}
	}

	if req.StartsAt != nil || req.EndsAt != nil {
		startsAtStr := existing.StartsAt.Format(time.RFC3339)
		newStarts := &startsAtStr
		if req.StartsAt != nil {
			newStarts = req.StartsAt
		}
		var newEnds *string
		if existing.EndsAt != nil {
			tmp := existing.EndsAt.Format(time.RFC3339)
			newEnds = &tmp
		}
		if req.EndsAt != nil {
			newEnds = req.EndsAt
		}
		if err := s.validateDateRanges(newStarts, newEnds); err != nil {
			return err
		}
	}
	return nil
}

// UpdateStatus updates the status of a promotion
func (s *PromotionServiceImpl) UpdateStatus(
	ctx context.Context,
	id uint,
	req model.UpdateStatusRequest,
	sellerID uint,
) (*model.PromotionResponse, error) {
	log.InfoWithContext(ctx, fmt.Sprintf("Updating promotion status for %d to %s", id, req.Status))

	existing, err := s.promotionRepo.FindByID(ctx, id)
	if err != nil {
		return nil, promoErrors.ErrPromotionNotFound
	}

	if existing.SellerID != sellerID {
		return nil, promoErrors.ErrUnauthorizedPromotionAccess
	}

	if err := validateStatusTransition(existing.Status, req.Status); err != nil {
		return nil, err
	}

	if err := s.promotionRepo.UpdateStatus(ctx, id, req.Status); err != nil {
		log.ErrorWithContext(ctx, "Failed to update promotion status", err)
		return nil, promoErrors.ErrPromotionUpdateFailed
	}

	existing.Status = req.Status
	return factory.PromotionEntityToResponse(existing), nil
}

// DeletePromotion deletes a promotion and its dependent scope rows in a transaction
func (s *PromotionServiceImpl) DeletePromotion(ctx context.Context, id uint, sellerID uint) error {
	log.InfoWithContext(ctx, fmt.Sprintf("Deleting promotion %d", id))

	existing, err := s.promotionRepo.FindByID(ctx, id)
	if err != nil {
		return promoErrors.ErrPromotionNotFound
	}

	if existing.SellerID != sellerID {
		return promoErrors.ErrUnauthorizedPromotionAccess
	}

	if existing.Status == entity.StatusActive {
		return promoErrors.ErrCannotDeleteActivePromotion
	}

	err = db.WithTransaction(ctx, func(txCtx context.Context) error {
		// Delete dependent scope rows first to avoid orphaned records
		scopeTables := []string{
			entity.PromotionProduct{}.TableName(),
			entity.PromotionProductVariant{}.TableName(),
			entity.PromotionCategory{}.TableName(),
			entity.PromotionCollection{}.TableName(),
			entity.PromotionUsage{}.TableName(),
		}

		for _, table := range scopeTables {
			if err := db.DB(txCtx).Exec(
				fmt.Sprintf("DELETE FROM %s WHERE promotion_id = ?", table), id,
			).Error; err != nil {
				log.ErrorWithContext(txCtx, fmt.Sprintf("Failed to delete from %s", table), err)
				return err
			}
		}

		if err := s.promotionRepo.Delete(txCtx, id); err != nil {
			log.ErrorWithContext(txCtx, "Failed to delete promotion", err)
			return err
		}

		return nil
	})
	if err != nil {
		return promoErrors.ErrPromotionDeleteFailed
	}

	log.InfoWithContext(ctx, fmt.Sprintf("Promotion deleted successfully: %d", id))
	return nil
}

// validateDateRanges validates that StartsAt is before EndsAt
func (s *PromotionServiceImpl) validateDateRanges(startsAt *string, endsAt *string) error {
	if startsAt == nil {
		return promoErrors.ErrInvalidDateRange.WithMessage("startsAt is required")
	}

	startsAtTime, err := time.Parse(time.RFC3339, *startsAt)
	if err != nil {
		return promoErrors.ErrInvalidDateRange.WithMessage("startsAt must be in RFC3339 format")
	}

	if endsAt != nil && *endsAt != "" {
		endsAtTime, err := time.Parse(time.RFC3339, *endsAt)
		if err != nil {
			return promoErrors.ErrInvalidDateRange.WithMessage("endsAt must be in RFC3339 format")
		}

		if !endsAtTime.After(startsAtTime) {
			return promoErrors.ErrInvalidDateRange.WithMessage("endsAt must be after startsAt")
		}
	}

	return nil
}

// validateEligibility validates customer eligibility settings
func (s *PromotionServiceImpl) validateEligibility(
	eligibleFor entity.EligibilityType,
	customerSegmentID *uint,
) error {
	// If no eligibleFor is specified, default to everyone
	if eligibleFor == "" {
		return nil
	}

	// If specific_segment is selected, customerSegmentID is required
	if eligibleFor == entity.EligibleSpecificSegment {
		if customerSegmentID == nil || *customerSegmentID == 0 {
			return promoErrors.ErrInvalidEligibility.WithMessage(
				"customerSegmentId is required when eligibleFor is 'specific_segment'",
			)
		}
	}

	return nil
}

// validateStatusTransition checks if the requested status change is allowed
func validateStatusTransition(current, next entity.CampaignStatus) error {
	if current == next {
		return nil
	}

	switch current {
	case entity.StatusDraft:
		if next != entity.StatusScheduled && next != entity.StatusActive &&
			next != entity.StatusCancelled {
			return promoErrors.ErrInvalidStatusTransition
		}
	case entity.StatusScheduled:
		if next != entity.StatusActive && next != entity.StatusPaused &&
			next != entity.StatusCancelled {
			return promoErrors.ErrInvalidStatusTransition
		}
	case entity.StatusActive:
		if next != entity.StatusPaused && next != entity.StatusEnded &&
			next != entity.StatusCancelled {
			return promoErrors.ErrInvalidStatusTransition
		}
	case entity.StatusPaused:
		if next != entity.StatusActive && next != entity.StatusEnded &&
			next != entity.StatusCancelled {
			return promoErrors.ErrInvalidStatusTransition
		}
	case entity.StatusEnded, entity.StatusCancelled:
		return promoErrors.ErrInvalidStatusTransition.WithMessage(
			"Cannot change status of an ended or cancelled promotion",
		)
	default:
		return promoErrors.ErrInvalidStatusTransition
	}
	return nil
}
