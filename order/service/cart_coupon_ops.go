package service

import (
	"context"
	"strings"

	"ecommerce-be/common/db"
	"ecommerce-be/order/entity"
	"ecommerce-be/order/model"
	promoErrors "ecommerce-be/promotion/error"
	promoFactory "ecommerce-be/promotion/factory"
	promotionModel "ecommerce-be/promotion/model"
)

func (s *CartServiceImpl) ApplyCoupon(
	ctx context.Context,
	userID, sellerID uint,
	code string,
) (*model.CartResponse, error) {
	return db.WithTransactionResult(ctx, func(txCtx context.Context) (*model.CartResponse, error) {
		cart, err := s.cartRepo.FindActiveCartByUserID(txCtx, userID)
		if err != nil {
			return nil, err
		}
		items, err := s.cartRepo.FindItemsByCartID(txCtx, cart.ID)
		if err != nil {
			return nil, err
		}
		if len(items) == 0 {
			return nil, promoErrors.ErrCouponNotApplicable
		}

		currencyMap, err := s.userSvc.GetPreferredCurrency(txCtx, userID, sellerID)
		if err != nil {
			return nil, err
		}

		variantMap, err := s.fetchVariantMap(txCtx, items, sellerID)
		if err != nil {
			return nil, err
		}
		promoReq, err := s.buildPromotionRequest(txCtx, sellerID, userID, items, variantMap)
		if err != nil {
			return nil, err
		}
		promoSummary, err := s.promotionSvc.ApplyPromotionsToCart(txCtx, promoReq)
		if err != nil {
			return nil, err
		}

		existingIDs, err := s.appliedDiscountCodeIDs(txCtx, cart.ID)
		if err != nil {
			return nil, err
		}

		couponReq := buildCouponCartRequestFromPromo(userID, sellerID, promoSummary, promoReq, existingIDs)
		dc, err := s.couponApplySvc.ValidateCouponForCart(txCtx, code, couponReq)
		if err != nil {
			return nil, err
		}

		if err := s.cartRepo.AddAppliedCoupon(txCtx, &entity.CartAppliedCoupon{
			CartID:         cart.ID,
			DiscountCodeID: dc.ID,
		}); err != nil {
			if isUniqueViolation(err) {
				return nil, promoErrors.ErrCouponAlreadyApplied
			}
			return nil, err
		}

		return s.buildCartResponseWithItems(txCtx, sellerID, userID, cart, items, currencyMap)
	})
}

func (s *CartServiceImpl) RemoveCoupon(
	ctx context.Context,
	userID, sellerID uint,
	code string,
) (*model.CartResponse, error) {
	return db.WithTransactionResult(ctx, func(txCtx context.Context) (*model.CartResponse, error) {
		cart, err := s.cartRepo.FindActiveCartByUserID(txCtx, userID)
		if err != nil {
			return nil, err
		}

		normalized := promoFactory.NormalizeDiscountCode(code)
		applied, err := s.cartRepo.FindAppliedCouponsByCartID(txCtx, cart.ID)
		if err != nil {
			return nil, err
		}

		var targetID uint
		found := false
		for _, row := range applied {
			dc, findErr := s.discountCodeRepo.FindByID(txCtx, row.DiscountCodeID)
			if findErr != nil {
				continue
			}
			if promoFactory.NormalizeDiscountCode(dc.Code) == normalized {
				targetID = row.DiscountCodeID
				found = true
				break
			}
		}
		if !found {
			return nil, promoErrors.ErrCouponNotOnCart
		}

		if err := s.cartRepo.RemoveAppliedCoupon(txCtx, cart.ID, targetID); err != nil {
			return nil, err
		}

		items, err := s.cartRepo.FindItemsByCartID(txCtx, cart.ID)
		if err != nil {
			return nil, err
		}
		currencyMap, err := s.userSvc.GetPreferredCurrency(txCtx, userID, sellerID)
		if err != nil {
			return nil, err
		}
		return s.buildCartResponseWithItems(txCtx, sellerID, userID, cart, items, currencyMap)
	})
}

func (s *CartServiceImpl) RemoveAllCoupons(
	ctx context.Context,
	userID, sellerID uint,
) (*model.CartResponse, error) {
	return db.WithTransactionResult(ctx, func(txCtx context.Context) (*model.CartResponse, error) {
		cart, err := s.cartRepo.FindActiveCartByUserID(txCtx, userID)
		if err != nil {
			return nil, err
		}
		if err := s.cartRepo.RemoveAllAppliedCoupons(txCtx, cart.ID); err != nil {
			return nil, err
		}
		items, err := s.cartRepo.FindItemsByCartID(txCtx, cart.ID)
		if err != nil {
			return nil, err
		}
		currencyMap, err := s.userSvc.GetPreferredCurrency(txCtx, userID, sellerID)
		if err != nil {
			return nil, err
		}
		return s.buildCartResponseWithItems(txCtx, sellerID, userID, cart, items, currencyMap)
	})
}

func (s *CartServiceImpl) GetAvailableCoupons(
	ctx context.Context,
	userID, sellerID uint,
) (*promotionModel.AvailableCouponsResponse, error) {
	cart, err := s.cartRepo.FindActiveCartByUserID(ctx, userID)
	if err != nil {
		couponReq := &promotionModel.CouponCartRequest{
			SellerID:               sellerID,
			CustomerID:             &userID,
			ShippingCents:          5000,
			PromotionsAllowCoupons: true,
		}
		return s.couponApplySvc.ListAvailableCouponsForCart(ctx, couponReq)
	}

	items, err := s.cartRepo.FindItemsByCartID(ctx, cart.ID)
	if err != nil {
		return nil, err
	}

	variantMap, err := s.fetchVariantMap(ctx, items, sellerID)
	if err != nil {
		return nil, err
	}
	promoReq, err := s.buildPromotionRequest(ctx, sellerID, userID, items, variantMap)
	if err != nil {
		return nil, err
	}
	promoSummary, err := s.promotionSvc.ApplyPromotionsToCart(ctx, promoReq)
	if err != nil {
		return nil, err
	}
	existingIDs, err := s.appliedDiscountCodeIDs(ctx, cart.ID)
	if err != nil {
		return nil, err
	}
	couponReq := buildCouponCartRequestFromPromo(userID, sellerID, promoSummary, promoReq, existingIDs)
	return s.couponApplySvc.ListAvailableCouponsForCart(ctx, couponReq)
}

// RevalidateCouponsForCheckout rebuilds pricing inputs from the locked checkout cart and
// ensures every applied coupon is still valid before usage is recorded.
func (s *CartServiceImpl) RevalidateCouponsForCheckout(
	ctx context.Context,
	userID, sellerID, cartID uint,
	appliedDiscountCodeIDs []uint,
) error {
	if len(appliedDiscountCodeIDs) == 0 {
		return nil
	}

	items, err := s.cartRepo.FindItemsByCartID(ctx, cartID)
	if err != nil {
		return err
	}
	if len(items) == 0 {
		return promoErrors.ErrCouponNotApplicable
	}

	variantMap, err := s.fetchVariantMap(ctx, items, sellerID)
	if err != nil {
		return err
	}
	promoReq, err := s.buildPromotionRequest(ctx, sellerID, userID, items, variantMap)
	if err != nil {
		return err
	}
	promoSummary, err := s.promotionSvc.ApplyPromotionsToCart(ctx, promoReq)
	if err != nil {
		return err
	}

	couponReq := buildCouponCartRequestFromPromo(
		userID,
		sellerID,
		promoSummary,
		promoReq,
		appliedDiscountCodeIDs,
	)
	return s.couponApplySvc.EnsureCouponsValidAtCheckout(ctx, couponReq)
}

func (s *CartServiceImpl) appliedDiscountCodeIDs(ctx context.Context, cartID uint) ([]uint, error) {
	rows, err := s.cartRepo.FindAppliedCouponsByCartID(ctx, cartID)
	if err != nil {
		return nil, err
	}
	ids := make([]uint, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.DiscountCodeID)
	}
	return ids, nil
}

func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "duplicate key") || strings.Contains(msg, "unique")
}
