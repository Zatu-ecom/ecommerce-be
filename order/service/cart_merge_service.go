package service

import (
	"context"

	"ecommerce-be/common/db"
	errs "ecommerce-be/common/error"
	"ecommerce-be/order/model"
	"ecommerce-be/order/repository"
)

// CartMergeService defines the operation for merging a device (guest) cart into a user (authenticated) cart.
// Following ISP — single-method interface for the merge operation.
type CartMergeService interface {
	MergeDeviceCartIntoUserCart(
		ctx context.Context,
		userID uint,
		deviceID string,
		sellerID uint,
	) (*model.CartResponse, error)
}

// CartMergeServiceImpl implements CartMergeService.
// Uses GuestCartService to read device cart and CartService.AddToCart to merge into user cart.
type CartMergeServiceImpl struct {
	cartRepo         repository.CartRepository
	guestCartService GuestCartService
	cartService      CartService
}

// NewCartMergeService creates a new CartMergeService.
func NewCartMergeService(
	cartRepo repository.CartRepository,
	guestCartService GuestCartService,
	cartService CartService,
) CartMergeService {
	return &CartMergeServiceImpl{
		cartRepo:         cartRepo,
		guestCartService: guestCartService,
		cartService:      cartService,
	}
}

// MergeDeviceCartIntoUserCart merges items from a device (guest) cart into the user's active cart.
// Process:
//  1. Find device cart by device ID
//  2. If no device cart → return current user cart (no-op)
//  3. Read device cart items
//  4. If empty → delete device cart, return user cart
//  5. Convert device items to AddCartItemRequest
//  6. Call CartService.AddToCart (reuses all validation, inventory, promotion logic)
//  7. Delete device cart
//  8. Return merged cart
func (s *CartMergeServiceImpl) MergeDeviceCartIntoUserCart(
	ctx context.Context,
	userID uint,
	deviceID string,
	sellerID uint,
) (*model.CartResponse, error) {
	return db.WithTransactionResult(ctx, func(txCtx context.Context) (*model.CartResponse, error) {
		// Step 1 & 2: Find device cart — if none, return user cart (no-op)
		deviceCart, err := s.cartRepo.FindActiveCartByDeviceID(txCtx, deviceID)
		if err != nil {
			if appErr, ok := err.(*errs.AppError); ok && appErr.Code == errs.INVALID_ID_CODE {
				// No device cart — return current user cart
				return s.cartService.GetUserCart(txCtx, userID, sellerID)
			}
			return nil, err
		}

		// Step 3: Read device cart items
		deviceItems, err := s.cartRepo.FindItemsByCartID(txCtx, deviceCart.ID)
		if err != nil {
			return nil, err
		}

		// Step 4: Empty device cart — delete it and return user cart
		if len(deviceItems) == 0 {
			if delErr := s.cartRepo.DeleteCart(txCtx, deviceCart.ID); delErr != nil {
				return nil, delErr
			}
			return s.cartService.GetUserCart(txCtx, userID, sellerID)
		}

		// Step 5: Convert device items to AddCartItemRequest
		reqItems := make([]model.AddCartItemDetail, len(deviceItems))
		for i, item := range deviceItems {
			qty := item.Quantity
			reqItems[i] = model.AddCartItemDetail{
				VariantID: item.VariantID,
				Quantity:  &qty,
			}
		}

		// Step 6: Merge into user cart (reuses CartService.AddToCart which handles:
		// - inventory validation
		// - promotion application
		// - quantity summation for same variants
		// - cart creation if none exists)
		mergedCart, err := s.cartService.AddToCart(
			txCtx,
			userID,
			sellerID,
			model.AddCartItemRequest{Items: reqItems},
		)
		if err != nil {
			return nil, err
		}

		// Step 7: Delete device cart (cascade deletes items, coupons, promotions)
		if err := s.cartRepo.DeleteCart(txCtx, deviceCart.ID); err != nil {
			return nil, err
		}

		return mergedCart, nil
	})
}
