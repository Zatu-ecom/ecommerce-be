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
	userService "ecommerce-be/user/service"
)

type CartService interface {
	AddToCart(
		ctx context.Context,
		userID, sellerID uint,
		req model.AddCartItemRequest,
	) (*model.CartResponse, error)
}

type CartServiceImpl struct {
	cartRepo        repository.CartRepository
	promotionSvc    promotionService.PromotionService
	inventorySvc    inventoryService.InventoryQueryService
	variantQuerySvc productVariantService.VariantQueryService
	userSvc         userService.UserService
}

func NewCartService(
	cartRepo repository.CartRepository,
	promotionSvc promotionService.PromotionService,
	inventorySvc inventoryService.InventoryQueryService,
	variantQuerySvc productVariantService.VariantQueryService,
	userSvc userService.UserService,
) CartService {
	return &CartServiceImpl{
		cartRepo:        cartRepo,
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
		// 1. Get or create Cart
		cart, err := s.getOrCreateCart(txCtx, userID)
		if err != nil {
			return nil, err
		}

		// 2. Fetch Preferred Currency
		// TODO [MICROSERVICE]: When moving to microservices, replace this with HTTP/gRPC call to User Service
		currencyMap, err := s.userSvc.GetPreferredCurrency(txCtx, userID, sellerID)
		if err != nil {
			return nil, err
		}

		// 3. Validate inventory locally
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

		// 4.5 Fetch variant details via ListVariants
		variantMap, err := s.fetchVariantMap(txCtx, items, sellerID)
		if err != nil {
			return nil, err
		}

		// 5. Build Promotion Validation Request
		promoReq, err := s.buildPromotionRequest(txCtx, sellerID, userID, items, variantMap)
		if err != nil {
			return nil, err
		}

		// 6. Apply Promotions
		// TODO [MICROSERVICE]: When moving to microservices, replace this with HTTP call to Promotion Service
		log.InfoWithContext(txCtx, "Calling Promotion Service for Cart validation")
		promoSummary, err := s.promotionSvc.ApplyPromotionsToCart(txCtx, promoReq)
		if err != nil {
			log.ErrorWithContext(txCtx, "Failed to apply promotions", err)
			return nil, orderError.ErrPromotionServiceUnavailable(err)
		}

		// 7. Map back to CartResponse
		return factory.BuildCartResponse(cart, items, promoSummary, currencyMap, variantMap), nil
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
		available := 0
		if len(invRes.Items) > 0 {
			available = invRes.Items[0].TotalAvailable
		}
		return orderError.ErrInsufficientStock(available)
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
	ctx context.Context,
	sellerID, userID uint,
	items []entity.CartItem,
	variantMap map[uint]productModel.VariantDetailResponse,
) (*promotionModel.CartValidationRequest, error) {
	promoReq := &promotionModel.CartValidationRequest{
		SellerID:      sellerID,
		CustomerID:    &userID, // Optional but good for segment targeting
		IsFirstOrder:  false,   // TODO [MICROSERVICE]: Check user order history
		Items:         make([]promotionModel.CartItem, len(items)),
		SubtotalCents: 0,
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
			CategoryID: 0, // ListVariants doesn't return CategoryID
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
