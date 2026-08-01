package service

import (
	"context"

	"ecommerce-be/promotion/entity"
	promoErrors "ecommerce-be/promotion/error"
	"ecommerce-be/promotion/repository"
)

func findOwnedDiscountCode(
	ctx context.Context,
	repo repository.DiscountCodeRepository,
	discountCodeID, sellerID uint,
) (*entity.DiscountCode, error) {
	code, err := repo.FindByID(ctx, discountCodeID)
	if err != nil {
		return nil, err
	}
	if code.SellerID != sellerID {
		return nil, promoErrors.ErrDiscountCodeNotFound
	}
	return code, nil
}

func validateDiscountCodeScopeAppliesTo(
	code *entity.DiscountCode,
	expected entity.ScopeType,
) error {
	if code.AppliesTo == entity.ScopeAllProducts || code.AppliesTo != expected {
		return promoErrors.ErrInvalidDiscountCodeScope
	}
	return nil
}

func ensureDiscountCodeScope(
	ctx context.Context,
	repo repository.DiscountCodeRepository,
	discountCodeID, sellerID uint,
	expected entity.ScopeType,
) (*entity.DiscountCode, error) {
	code, err := findOwnedDiscountCode(ctx, repo, discountCodeID, sellerID)
	if err != nil {
		return nil, err
	}
	if err := validateDiscountCodeScopeAppliesTo(code, expected); err != nil {
		return nil, err
	}
	return code, nil
}
