package service

import (
	"context"
	"fmt"
	"strconv"

	"ecommerce-be/common/db"
	errs "ecommerce-be/common/error"
	"ecommerce-be/common/log"
	"ecommerce-be/order/entity"
	"ecommerce-be/order/model"
	"ecommerce-be/order/repository"

	promotionModel "ecommerce-be/promotion/model"
	promotionService "ecommerce-be/promotion/service"

	inventoryModel "ecommerce-be/inventory/model"
	inventoryService "ecommerce-be/inventory/service"
)

type CartService interface {
	AddToCart(
		ctx context.Context,
		userID, sellerID uint,
		req model.AddCartItemRequest,
	) (*model.CartResponse, error)
}

type CartServiceImpl struct {
	cartRepo     repository.CartRepository
	promotionSvc promotionService.PromotionService
	inventorySvc inventoryService.InventoryQueryService
}

func NewCartService(
	cartRepo repository.CartRepository,
	promotionSvc promotionService.PromotionService,
	inventorySvc inventoryService.InventoryQueryService,
) CartService {
	return &CartServiceImpl{
		cartRepo:     cartRepo,
		promotionSvc: promotionSvc,
		inventorySvc: inventorySvc,
	}
}

func (s *CartServiceImpl) AddToCart(
	ctx context.Context,
	userID, sellerID uint,
	req model.AddCartItemRequest,
) (*model.CartResponse, error) {
	return db.WithTransactionResult(ctx, func(txCtx context.Context) (*model.CartResponse, error) {
		// 1. Get or create Cart
		cart, err := s.getOrCreateCart(txCtx, userID)
		if err != nil {
			return nil, err
		}

		// 2. Validate inventory locally
		if err := s.validateInventory(txCtx, req.VariantID, req.Quantity, sellerID); err != nil {
			return nil, err
		}

		// 3. Add or update item
		if err := s.addOrUpdateCartItem(txCtx, cart.ID, req.VariantID, req.Quantity); err != nil {
			return nil, err
		}

		// 4. Fetch all items to calculate current state
		items, err := s.cartRepo.FindItemsByCartID(txCtx, cart.ID)
		if err != nil {
			return nil, err
		}

		// 5. Build Promotion Validation Request
		promoReq := s.buildPromotionRequest(sellerID, userID, items)

		// 6. Apply Promotions
		// TODO [MICROSERVICE]: When moving to microservices, replace this with HTTP call to Promotion Service
		log.InfoWithContext(txCtx, "Calling Promotion Service for Cart validation")
		promoSummary, err := s.promotionSvc.ApplyPromotionsToCart(txCtx, promoReq)
		if err != nil {
			log.ErrorWithContext(txCtx, "Failed to apply promotions", err)
			return nil, errs.NewAppError(
				"SYSTEM_ERROR",
				"Promotion service unavailable: "+err.Error(),
				500,
			)
		}

		// 7. Map back to CartResponse
		return s.mapToCartResponse(cart, items, promoSummary), nil
	})
}

func (s *CartServiceImpl) getOrCreateCart(ctx context.Context, userID uint) (*entity.Cart, error) {
	cart, err := s.cartRepo.FindByUserID(ctx, userID)
	if err != nil {
		if appErr, ok := err.(*errs.AppError); ok && appErr.Code == errs.INVALID_ID_CODE {
			// Cart doesn't exist, create it
			cart = &entity.Cart{
				UserID:   userID,
				Metadata: map[string]interface{}{},
			}
			if err := s.cartRepo.CreateCart(ctx, cart); err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}
	return cart, nil
}

func (s *CartServiceImpl) validateInventory(
	ctx context.Context,
	variantID uint,
	quantity int,
	sellerID uint,
) error {
	invReq := inventoryModel.TotalAvailableQuantityRequest{
		VariantIDs: []uint{variantID},
	}

	invRes, err := s.inventorySvc.GetTotalAvailableQuantities(ctx, invReq, sellerID)
	if err != nil {
		return err
	}

	if len(invRes.Items) == 0 || invRes.Items[0].TotalAvailable < quantity {
		return errs.NewAppError(
			"INSUFFICIENT_STOCK",
			fmt.Sprintf("Only %d items available for this variant", func() int {
				if len(invRes.Items) > 0 {
					return invRes.Items[0].TotalAvailable
				}
				return 0
			}()),
			400,
		)
	}
	return nil
}

func (s *CartServiceImpl) addOrUpdateCartItem(
	ctx context.Context,
	cartID uint,
	variantID uint,
	quantity int,
) error {
	existingItem, err := s.cartRepo.FindItemByVariantID(ctx, cartID, variantID)
	if err != nil {
		return err
	}

	if existingItem != nil {
		// Update quantity
		existingItem.Quantity += quantity
		return s.cartRepo.UpdateItem(ctx, existingItem)
	}

	// Add new item
	newItem := &entity.CartItem{
		CartID:    cartID,
		VariantID: variantID,
		Quantity:  quantity,
	}
	return s.cartRepo.AddItem(ctx, newItem)
}

func (s *CartServiceImpl) buildPromotionRequest(
	sellerID, userID uint,
	items []entity.CartItem,
) *promotionModel.CartValidationRequest {
	promoReq := &promotionModel.CartValidationRequest{
		SellerID:      sellerID,
		CustomerID:    &userID, // Optional but good for segment targeting
		IsFirstOrder:  false,   // TODO [MICROSERVICE]: Check user order history
		Items:         make([]promotionModel.CartItem, len(items)),
		SubtotalCents: 0,
	}

	for i, item := range items {
		// TODO [MICROSERVICE]: When moving to microservices, replace this with HTTP call to Product Service
		// For now, we simulate fetching the variant price (assuming 100000 cents = ₹1000 for demo if we can't fetch it)
		// Usually you'd inject a product repository here, but for module separation we mock the struct we feed to promotion

		variantPriceCents := int64(100000) // Mock price: ₹1000.00

		lineTotal := variantPriceCents * int64(item.Quantity)
		promoReq.SubtotalCents += lineTotal

		promoReq.Items[i] = promotionModel.CartItem{
			ItemID:     strconv.Itoa(int(item.ID)),
			VariantID:  &item.VariantID,
			ProductID:  1, // Mock product ID
			CategoryID: 1, // Mock category ID
			Quantity:   item.Quantity,
			PriceCents: variantPriceCents,
			TotalCents: lineTotal,
		}
	}
	return promoReq
}

// mapToCartResponse converts entities and promotion summary into the final CartResponse
func (s *CartServiceImpl) mapToCartResponse(
	cart *entity.Cart,
	items []entity.CartItem,
	promo *promotionModel.AppliedPromotionSummary,
) *model.CartResponse {
	response := &model.CartResponse{
		CartBase: model.CartBase{
			ID:     cart.ID,
			UserID: cart.UserID,
			Currency: model.CurrencyInfo{
				Code:          "INR",
				Symbol:        "₹",
				DecimalDigits: 2,
			},
			Metadata: cart.Metadata,
		},
		Summary: model.CartSummary{
			ItemCount:                  0,
			UniqueItems:                len(items),
			Subtotal:                   promo.OriginalSubtotal,
			SubtotalFormatted:          formatCurrency(promo.OriginalSubtotal),
			PromotionCount:             len(promo.AppliedPromotions),
			PromotionDiscount:          promo.TotalDiscountCents,
			PromotionDiscountFormatted: formatCurrency(promo.TotalDiscountCents),
			TotalDiscount:              promo.TotalDiscountCents, // No coupons yet
			TotalDiscountFormatted:     formatCurrency(promo.TotalDiscountCents),
			AfterDiscount:              promo.FinalSubtotal,
			AfterDiscountFormatted:     formatCurrency(promo.FinalSubtotal),
			Total:                      promo.FinalSubtotal,
			TotalFormatted:             formatCurrency(promo.FinalSubtotal),
		},
		Items:          make([]model.CartItemWithPricingResponse, len(items)),
		AppliedCoupons: make([]model.AppliedCouponInfo, 0), // Not implemented yet
	}

	// Build map of item promotions from summary
	itemPromoMap := make(map[string]promotionModel.CartItemSummary)
	for _, summaryItem := range promo.Items {
		itemPromoMap[summaryItem.ItemID] = summaryItem
	}

	for i, item := range items {
		response.Summary.ItemCount += item.Quantity
		itemIDStr := strconv.Itoa(int(item.ID))

		summaryItem, exists := itemPromoMap[itemIDStr]

		// Fallback values if not found in summary
		unitPrice := int64(100000)
		lineTotal := unitPrice * int64(item.Quantity)
		discountedLineTotal := lineTotal
		totalItemDiscount := int64(0)
		var appliedPromos []model.ItemAppliedPromotionInfo

		if exists {
			unitPrice = summaryItem.OriginalPriceCents
			lineTotal = unitPrice * int64(item.Quantity)
			discountedLineTotal = summaryItem.FinalPriceCents * int64(item.Quantity)
			totalItemDiscount = summaryItem.TotalDiscountCents

			for _, p := range summaryItem.AppliedPromotions {
				appliedPromos = append(appliedPromos, model.ItemAppliedPromotionInfo{
					PromotionID:       p.PromotionID,
					Name:              p.PromotionName,
					Type:              "applied_promotion", // Generic type for now
					Discount:          p.DiscountCents,
					DiscountFormatted: formatCurrency(p.DiscountCents),
				})
			}
		}

		response.Items[i] = model.CartItemWithPricingResponse{
			CartItemBase: model.CartItemBase{
				ID:        item.ID,
				CartID:    item.CartID,
				VariantID: item.VariantID,
				Quantity:  item.Quantity,
				Variant: model.VariantInfo{
					ID:      item.VariantID,
					SKU:     "MOCK-SKU", // TODO [MICROSERVICE] Get from Product Service
					Product: model.ProductBasicInfo{Name: "Mock Product"},
				},
			},
			UnitPrice:              unitPrice,
			LineTotal:              lineTotal,
			TotalPromotionDiscount: totalItemDiscount,
			DiscountedLineTotal:    discountedLineTotal,
			AppliedPromotions:      appliedPromos,
		}
	}

	// Add savings info if discount > 0
	if response.Summary.TotalDiscount > 0 {
		percentage := float64(
			response.Summary.TotalDiscount,
		) / float64(
			response.Summary.Subtotal,
		) * 100
		response.Summary.Savings = &model.SavingsInfo{
			Amount:     response.Summary.TotalDiscount,
			Percentage: percentage,
			Message: fmt.Sprintf(
				"You're saving %s (%.0f%% off)!",
				response.Summary.TotalDiscountFormatted,
				percentage,
			),
		}
	}

	return response
}

func formatCurrency(cents int64) string {
	return fmt.Sprintf("₹%.2f", float64(cents)/100.0)
}
