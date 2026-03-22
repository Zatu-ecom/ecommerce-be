package service

import (
	"context"
	"strconv"
	"strings"

	"ecommerce-be/common/db"
	errs "ecommerce-be/common/error"
	"ecommerce-be/common/log"
	"ecommerce-be/order/entity"
	orderError "ecommerce-be/order/error"
	"ecommerce-be/order/factory"
	"ecommerce-be/order/model"
	"ecommerce-be/order/repository"

	promotionModel "ecommerce-be/promotion/model"
	promotionService "ecommerce-be/promotion/service"

	inventoryModel "ecommerce-be/inventory/model"
	inventoryService "ecommerce-be/inventory/service"

	productModel "ecommerce-be/product/model"
	productVariantService "ecommerce-be/product/service"
	userModel "ecommerce-be/user/model"
	userService "ecommerce-be/user/service"
)

type CartService interface {
	AddToCart(
		ctx context.Context,
		userID, sellerID uint,
		req model.AddCartItemRequest,
	) (*model.CartResponse, error)
	GetUserCart(
		ctx context.Context,
		userID, sellerID uint,
	) (*model.CartResponse, error)
	DeleteCart(
		ctx context.Context,
		userID, sellerID, cartID uint,
	) (*model.CartResponse, error)
}

type CartServiceImpl struct {
	cartRepo        repository.CartRepository
	orderRepo       repository.OrderRepository
	promotionSvc    promotionService.PromotionService
	inventorySvc    inventoryService.InventoryQueryService
	variantQuerySvc productVariantService.VariantQueryService
	userSvc         userService.UserService
}

func NewCartService(
	cartRepo repository.CartRepository,
	orderRepo repository.OrderRepository,
	promotionSvc promotionService.PromotionService,
	inventorySvc inventoryService.InventoryQueryService,
	variantQuerySvc productVariantService.VariantQueryService,
	userSvc userService.UserService,
) CartService {
	return &CartServiceImpl{
		cartRepo:        cartRepo,
		orderRepo:       orderRepo,
		promotionSvc:    promotionSvc,
		inventorySvc:    inventorySvc,
		variantQuerySvc: variantQuerySvc,
		userSvc:         userSvc,
	}
}

func (s *CartServiceImpl) AddToCart(
	ctx context.Context,
	userID, sellerID uint,
	req model.AddCartItemRequest,
) (*model.CartResponse, error) {
	return db.WithTransactionResult(ctx, func(txCtx context.Context) (*model.CartResponse, error) {
		currencyMap, err := s.userSvc.GetPreferredCurrency(txCtx, userID, sellerID)
		if err != nil {
			return nil, err
		}

		hasPositiveQty := hasPositiveQuantity(req.Items)

		cart, err := s.getExistingOrCreateCart(txCtx, userID, hasPositiveQty)
		if err != nil {
			return nil, err
		}

		if cart == nil && !hasPositiveQty {
			return s.buildEmptyCartResponse(userID, currencyMap), nil
		}

		existingItemByVariant, finalQuantityByVariant, err := s.loadCartMutationState(
			txCtx,
			cart,
		)
		if err != nil {
			return nil, err
		}

		variantsNeedingValidation := applyRequestedQuantities(req.Items, finalQuantityByVariant)

		if err := s.validateInventoryForFinalQuantities(
			txCtx,
			sellerID,
			variantsNeedingValidation,
			finalQuantityByVariant,
		); err != nil {
			return nil, err
		}

		if err := s.applyFinalQuantitiesToCart(
			txCtx,
			cart.ID,
			existingItemByVariant,
			finalQuantityByVariant,
		); err != nil {
			return nil, err
		}

		items, err := s.cartRepo.FindItemsByCartID(txCtx, cart.ID)
		if err != nil {
			return nil, err
		}

		if len(items) == 0 {
			if err := s.cartRepo.DeleteCart(txCtx, cart.ID); err != nil {
				return nil, err
			}
			return s.buildEmptyCartResponse(userID, currencyMap), nil
		}

		return s.buildCartResponseWithItems(
			txCtx,
			sellerID,
			userID,
			cart,
			items,
			currencyMap,
		)
	})
}

func (s *CartServiceImpl) GetUserCart(
	ctx context.Context,
	userID, sellerID uint,
) (*model.CartResponse, error) {
	return db.WithTransactionResult(ctx, func(txCtx context.Context) (*model.CartResponse, error) {
		currencyMap, err := s.userSvc.GetPreferredCurrency(txCtx, userID, sellerID)
		if err != nil {
			return nil, err
		}

		cart, err := s.getExistingOrCreateCart(txCtx, userID, false)
		if err != nil {
			return nil, err
		}
		if cart == nil {
			return s.buildEmptyCartResponse(userID, currencyMap), nil
		}

		items, err := s.cartRepo.FindItemsByCartID(txCtx, cart.ID)
		if err != nil {
			return nil, err
		}

		if len(items) == 0 {
			return s.buildEmptyCartResponse(userID, currencyMap), nil
		}

		return s.buildCartResponseWithItems(txCtx, sellerID, userID, cart, items, currencyMap)
	})
}

func (s *CartServiceImpl) DeleteCart(
	ctx context.Context,
	userID, sellerID, cartID uint,
) (*model.CartResponse, error) {
	return db.WithTransactionResult(ctx, func(txCtx context.Context) (*model.CartResponse, error) {
		currencyMap, err := s.userSvc.GetPreferredCurrency(txCtx, userID, sellerID)
		if err != nil {
			return nil, err
		}

		cart, err := s.cartRepo.FindByID(txCtx, cartID)
		if err != nil {
			return nil, err
		}
		if cart.UserID != userID {
			return nil, errs.NewAppError(errs.INVALID_ID_CODE, "Cart not found", 404)
		}

		if err := s.cartRepo.DeleteItemsByCartID(txCtx, cart.ID); err != nil {
			return nil, err
		}
		if err := s.cartRepo.DeleteCart(txCtx, cart.ID); err != nil {
			return nil, err
		}
		return s.buildEmptyCartResponse(userID, currencyMap), nil
	})
}

func hasPositiveQuantity(items []model.AddCartItemDetail) bool {
	for _, item := range items {
		if item.Quantity != nil && *item.Quantity > 0 {
			return true
		}
	}
	return false
}

func (s *CartServiceImpl) loadCartMutationState(
	ctx context.Context,
	cart *entity.Cart,
) (
	map[uint]*entity.CartItem,
	map[uint]int,
	error,
) {
	existingItems := []entity.CartItem{}
	if cart != nil {
		items, err := s.cartRepo.FindItemsByCartID(ctx, cart.ID)
		if err != nil {
			return nil, nil, err
		}
		existingItems = items
	}

	existingItemByVariant := make(map[uint]*entity.CartItem, len(existingItems))
	finalQuantityByVariant := make(map[uint]int, len(existingItems))
	for i := range existingItems {
		item := &existingItems[i]
		existingItemByVariant[item.VariantID] = item
		finalQuantityByVariant[item.VariantID] = item.Quantity
	}

	return existingItemByVariant, finalQuantityByVariant, nil
}

func applyRequestedQuantities(
	reqItems []model.AddCartItemDetail,
	finalQuantityByVariant map[uint]int,
) map[uint]struct{} {
	variantsNeedingValidation := make(map[uint]struct{})
	for _, item := range reqItems {
		quantity := 0
		if item.Quantity != nil {
			quantity = *item.Quantity
		}

		if quantity == 0 {
			finalQuantityByVariant[item.VariantID] = 0
			continue
		}

		variantsNeedingValidation[item.VariantID] = struct{}{}
		finalQuantityByVariant[item.VariantID] += quantity
	}

	return variantsNeedingValidation
}

func (s *CartServiceImpl) applyFinalQuantitiesToCart(
	ctx context.Context,
	cartID uint,
	existingItemByVariant map[uint]*entity.CartItem,
	finalQuantityByVariant map[uint]int,
) error {
	for variantID, finalQty := range finalQuantityByVariant {
		existingItem := existingItemByVariant[variantID]
		switch {
		case finalQty <= 0:
			if existingItem != nil {
				if err := s.cartRepo.DeleteItem(ctx, existingItem.ID); err != nil {
					return err
				}
			}
		case existingItem != nil:
			existingItem.Quantity = finalQty
			if err := s.cartRepo.UpdateItem(ctx, existingItem); err != nil {
				return err
			}
		default:
			if err := s.cartRepo.AddItem(ctx, &entity.CartItem{
				CartID:    cartID,
				VariantID: variantID,
				Quantity:  finalQty,
			}); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *CartServiceImpl) buildCartResponseWithItems(
	ctx context.Context,
	sellerID, userID uint,
	cart *entity.Cart,
	items []entity.CartItem,
	currencyMap *userModel.CurrencyResponse,
) (*model.CartResponse, error) {
	// Fetch variant details once for all cart items
	variantMap, err := s.fetchVariantMap(ctx, items, sellerID)
	if err != nil {
		return nil, err
	}

	promoReq, err := s.buildPromotionRequest(ctx, sellerID, userID, items, variantMap)
	if err != nil {
		return nil, err
	}

	// TODO [MICROSERVICE]: When moving to microservices, replace this with HTTP/grpc call to User Service
	log.InfoWithContext(ctx, "Calling Promotion Service for Cart validation")
	promoSummary, err := s.promotionSvc.ApplyPromotionsToCart(ctx, promoReq)
	if err != nil {
		log.ErrorWithContext(ctx, "Failed to apply promotions", err)
		return nil, orderError.ErrPromotionServiceUnavailable(err)
	}

	return factory.BuildCartResponse(
		cart,
		items,
		promoSummary,
		currencyMap,
		variantMap,
	), nil
}

func (s *CartServiceImpl) getExistingOrCreateCart(
	ctx context.Context,
	userID uint,
	createIfMissing bool,
) (*entity.Cart, error) {
	cart, err := s.cartRepo.FindByUserID(ctx, userID)
	if err != nil {
		if appErr, ok := err.(*errs.AppError); ok && appErr.Code == errs.INVALID_ID_CODE {
			if !createIfMissing {
				return nil, nil
			}
			// Cart doesn't exist, create it
			cart = &entity.Cart{
				UserID:   userID,
				Metadata: db.JSONMap{},
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

func (s *CartServiceImpl) validateInventoryForFinalQuantities(
	ctx context.Context,
	sellerID uint,
	variantsNeedingValidation map[uint]struct{},
	finalQuantityByVariant map[uint]int,
) error {
	if len(variantsNeedingValidation) == 0 {
		return nil
	}

	variantIDs := make([]uint, 0, len(variantsNeedingValidation))
	for variantID := range variantsNeedingValidation {
		variantIDs = append(variantIDs, variantID)
	}

	invReq := inventoryModel.TotalAvailableQuantityRequest{
		VariantIDs: variantIDs,
	}
	invRes, err := s.inventorySvc.GetTotalAvailableQuantities(ctx, invReq, sellerID)
	if err != nil {
		return err
	}

	availableByVariant := make(map[uint]int, len(invRes.Items))
	for _, item := range invRes.Items {
		availableByVariant[item.VariantID] = item.TotalAvailable
	}

	for variantID := range variantsNeedingValidation {
		available, exists := availableByVariant[variantID]
		if !exists {
			return orderError.ErrVariantNotFound
		}
		if finalQuantityByVariant[variantID] > available {
			return orderError.ErrInsufficientStock(available)
		}
	}
	return nil
}

func (s *CartServiceImpl) buildEmptyCartResponse(
	userID uint,
	currencyMap *userModel.CurrencyResponse,
) *model.CartResponse {
	return &model.CartResponse{
		CartBase: model.CartBase{
			ID:     0,
			UserID: userID,
			Currency: model.CurrencyInfo{
				Code:          currencyMap.Code,
				Symbol:        currencyMap.Symbol,
				DecimalDigits: currencyMap.DecimalDigits,
			},
			Metadata: map[string]any{},
		},
		Items:               []model.CartItemWithPricingResponse{},
		AppliedPromotions:   []model.AppliedPromotionInfo{},
		AppliedCoupons:      []model.AppliedCouponInfo{},
		Summary:             model.CartSummary{},
		AvailablePromotions: []model.AvailablePromotionInfo{},
	}
}

func (s *CartServiceImpl) buildPromotionRequest(
	ctx context.Context,
	sellerID, userID uint,
	items []entity.CartItem,
	variantMap map[uint]productModel.VariantDetailResponse,
) (*promotionModel.CartValidationRequest, error) {
	hasPastOrders, err := s.orderRepo.HasPastOrders(ctx, userID)
	if err != nil {
		log.WarnWithContext(ctx, "Failed to check user order history: "+err.Error())
		// Safest is to assume true so we don't accidentally give a first-order discount
		hasPastOrders = true
	}

	promoReq := &promotionModel.CartValidationRequest{
		SellerID:      sellerID,
		CustomerID:    &userID, // Optional but good for segment targeting
		IsFirstOrder:  !hasPastOrders,
		Items:         make([]promotionModel.CartItem, len(items)),
		SubtotalCents: 0,
		// TODO [FULFILLMENT]: Replace metadata fallback with Fulfillment Service quote
		// (shipping, handling, delivery method constraints) when service is ready.
		ShippingCents: 5000,
	}

	for i, item := range items {
		variant, exists := variantMap[item.VariantID]
		if !exists {
			log.WarnWithContext(
				ctx,
				"Variant information not found for variant ID: "+strconv.Itoa(int(item.VariantID)),
			)
			return nil, orderError.ErrVariantNotFound
		}

		variantPriceCents := int64(variant.Price * 100) // Convert floating price format to cents
		lineTotal := variantPriceCents * int64(item.Quantity)
		promoReq.SubtotalCents += lineTotal

		promoReq.Items[i] = promotionModel.CartItem{
			ItemID:     strconv.Itoa(int(item.ID)),
			VariantID:  &item.VariantID,
			ProductID:  variant.ProductID,
			CategoryID: variant.Product.CategoryID,
			Quantity:   item.Quantity,
			PriceCents: variantPriceCents,
			TotalCents: lineTotal,
		}
	}
	return promoReq, nil
}

func (s *CartServiceImpl) fetchVariantMap(
	ctx context.Context,
	items []entity.CartItem,
	sellerID uint,
) (map[uint]productModel.VariantDetailResponse, error) {
	variantMap := make(map[uint]productModel.VariantDetailResponse)
	if len(items) == 0 {
		return variantMap, nil
	}

	ids := make([]string, len(items))
	for i, item := range items {
		ids[i] = strconv.Itoa(int(item.VariantID))
	}

	listReq := &productModel.ListVariantsRequest{
		IDs:      strings.Join(ids, ","),
		PageSize: len(items),
	}

	listResp, err := s.variantQuerySvc.ListVariants(ctx, listReq, &sellerID, nil, nil)
	if err != nil {
		log.ErrorWithContext(ctx, "Failed to fetch variant information using ListVariants", err)
		return nil, err
	}

	for _, v := range listResp.Variants {
		variantMap[v.ID] = v
	}
	return variantMap, nil
}
