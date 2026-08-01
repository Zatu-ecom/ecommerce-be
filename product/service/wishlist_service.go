package service

import (
	"context"

	"ecommerce-be/common"
	"ecommerce-be/common/config"
	prodErrors "ecommerce-be/product/error"
	"ecommerce-be/product/factory"
	"ecommerce-be/product/model"
	"ecommerce-be/product/repository"
)

// WishlistService defines the interface for wishlist management business logic
type WishlistService interface {
	GetAllWishlists(ctx context.Context, userID uint) (*model.WishlistsResponse, error)
	GetWishlistByID(
		ctx context.Context,
		userID, wishlistID uint,
		params common.BaseListParams,
	) (*model.WishlistDetailResponse, error)
	CreateWishlist(
		ctx context.Context,
		userID uint,
		req model.WishlistCreateRequest,
	) (*model.WishlistResponse, error)
	UpdateWishlist(
		ctx context.Context,
		userID, wishlistID uint,
		req model.WishlistUpdateRequest,
	) (*model.WishlistResponse, error)
	DeleteWishlist(ctx context.Context, userID, wishlistID uint) error
}

// WishlistServiceImpl implements the WishlistService interface
type WishlistServiceImpl struct {
	wishlistRepo        repository.WishlistRepository
	wishlistItemRepo    repository.WishlistItemRepository
	productQueryService ProductQueryService
	variantQueryService VariantQueryService
}

// NewWishlistService creates a new instance of WishlistService
func NewWishlistService(
	wishlistRepo repository.WishlistRepository,
	wishlistItemRepo repository.WishlistItemRepository,
	productQueryService ProductQueryService,
	variantQueryService VariantQueryService,
) WishlistService {
	return &WishlistServiceImpl{
		wishlistRepo:        wishlistRepo,
		wishlistItemRepo:    wishlistItemRepo,
		productQueryService: productQueryService,
		variantQueryService: variantQueryService,
	}
}

// GetAllWishlists retrieves all wishlists for a user
func (s *WishlistServiceImpl) GetAllWishlists(
	ctx context.Context,
	userID uint,
) (*model.WishlistsResponse, error) {
	wishlists, err := s.wishlistRepo.FindByUserIDWithItemCount(ctx, userID)
	if err != nil {
		return nil, err
	}

	return factory.BuildWishlistsResponse(wishlists), nil
}

// GetWishlistByID retrieves a wishlist with paginated products
// Returns items with per-wishlist-item metadata (wishlistItemId, variantId, addedAt)
// alongside full resolved ProductResponse for each item
func (s *WishlistServiceImpl) GetWishlistByID(
	ctx context.Context,
	userID, wishlistID uint,
	params common.BaseListParams,
) (*model.WishlistDetailResponse, error) {
	// Get wishlist to verify ownership and get basic info
	wishlist, err := s.wishlistRepo.FindByID(ctx, wishlistID)
	if err != nil {
		return nil, err
	}

	// Verify ownership
	if wishlist.UserID != userID {
		return nil, prodErrors.ErrUnauthorizedWishlist
	}

	// Set default pagination
	params.SetDefaults()
	page := params.Page
	pageSize := params.PageSize

	// Get wishlist items with full metadata (ID, variant_id, created_at)
	items, totalItems, err := s.wishlistItemRepo.FindWishlistItemsByWishlistID(
		ctx,
		wishlistID,
		page,
		pageSize,
	)
	if err != nil {
		return nil, err
	}

	// Build wishlist product items
	wishlistItems := make([]model.WishlistProductItem, 0, len(items))

	if len(items) > 0 {
		// Collect variant IDs for batch product resolution
		variantIDs := make([]uint, len(items))
		for i, item := range items {
			variantIDs[i] = item.VariantID
		}

		// Step 1: Get basic info (variantID → productID) for all wishlist variant IDs.
		// This bridges the gap since GetAllProducts returns summary products without
		// full Variants populated (Variants is nil in listing responses).
		basicInfo, err := s.variantQueryService.GetProductBasicInfoByVariantIDs(
			ctx,
			variantIDs,
			nil,
		)
		if err != nil {
			return nil, err
		}

		// Build variantID → productID map and collect unique product IDs
		variantToProductID := make(map[uint]uint, len(basicInfo))
		uniqueProductIDs := make([]uint, 0, len(basicInfo))
		seenProductIDs := make(map[uint]struct{}, len(basicInfo))
		for _, row := range basicInfo {
			variantToProductID[row.VariantID] = row.ProductID
			if _, seen := seenProductIDs[row.ProductID]; !seen {
				seenProductIDs[row.ProductID] = struct{}{}
				uniqueProductIDs = append(uniqueProductIDs, row.ProductID)
			}
		}

		// Step 2: Get full product details using product IDs filter
		filter := model.GetProductsFilter{
			IDs: uniqueProductIDs,
		}
		products, err := s.productQueryService.GetAllProducts(
			ctx,
			1,
			len(uniqueProductIDs),
			filter,
			&userID,
		)
		if err != nil {
			return nil, err
		}

		// Build a map of productID -> ProductResponse for O(1) lookup
		productByID := make(map[uint]model.ProductResponse, len(products.Products))
		for _, p := range products.Products {
			productByID[p.ID] = p
		}

		// Step 3: Build items list preserving the order from the query.
		// Use variantToProductID to find the correct product for each item.
		for _, item := range items {
			productID, found := variantToProductID[item.VariantID]
			if !found {
				continue // Skip if no product found for this variant
			}
			product, found := productByID[productID]
			if !found {
				continue // Skip if product details not available
			}
			wishlistItems = append(wishlistItems, model.WishlistProductItem{
				WishlistItemID: item.ID,
				VariantID:      item.VariantID,
				AddedAt:        item.CreatedAt,
				Product:        product,
			})
		}
	}

	return &model.WishlistDetailResponse{
		ID:         wishlist.ID,
		Name:       wishlist.Name,
		IsDefault:  wishlist.IsDefault,
		Items:      wishlistItems,
		Pagination: common.NewPaginationResponse(page, pageSize, totalItems),
		CreatedAt:  wishlist.CreatedAt,
		UpdatedAt:  wishlist.UpdatedAt,
	}, nil
}

// CreateWishlist creates a new wishlist for a user
func (s *WishlistServiceImpl) CreateWishlist(
	ctx context.Context,
	userID uint,
	req model.WishlistCreateRequest,
) (*model.WishlistResponse, error) {
	// Check max wishlists limit
	count, err := s.wishlistRepo.CountByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if count >= int64(config.Get().App.MaxWishlistsPerUser) {
		return nil, prodErrors.ErrMaxWishlistsReached
	}

	// Check if wishlist with same name exists
	existing, err := s.wishlistRepo.FindByUserIDAndName(ctx, userID, req.Name)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, prodErrors.ErrWishlistNameExists
	}

	// First wishlist becomes default
	isDefault := count == 0

	wishlist := factory.BuildWishlistEntity(userID, req.Name, isDefault)

	if err := s.wishlistRepo.Create(ctx, wishlist); err != nil {
		return nil, err
	}

	return factory.BuildWishlistResponse(wishlist, 0), nil
}

// UpdateWishlist updates a wishlist (name and/or default status)
func (s *WishlistServiceImpl) UpdateWishlist(
	ctx context.Context,
	userID, wishlistID uint,
	req model.WishlistUpdateRequest,
) (*model.WishlistResponse, error) {
	wishlist, err := s.wishlistRepo.FindByID(ctx, wishlistID)
	if err != nil {
		return nil, err
	}

	// Verify ownership
	if wishlist.UserID != userID {
		return nil, prodErrors.ErrUnauthorizedWishlist
	}

	// Update name if provided
	if req.Name != nil && *req.Name != wishlist.Name {
		// Check if new name already exists
		existing, err := s.wishlistRepo.FindByUserIDAndName(ctx, userID, *req.Name)
		if err != nil {
			return nil, err
		}
		if existing != nil && existing.ID != wishlistID {
			return nil, prodErrors.ErrWishlistNameExists
		}
		wishlist.Name = *req.Name
	}

	// Update default status if provided
	if req.IsDefault != nil && *req.IsDefault && !wishlist.IsDefault {
		// Clear default for other wishlists
		if err := s.wishlistRepo.ClearDefaultForUser(ctx, userID); err != nil {
			return nil, err
		}
		wishlist.IsDefault = true
	}

	if err := s.wishlistRepo.Update(ctx, wishlist); err != nil {
		return nil, err
	}

	itemCount, _ := s.wishlistRepo.CountItemsByWishlistID(ctx, wishlistID)
	return factory.BuildWishlistResponse(wishlist, int(itemCount)), nil
}

// DeleteWishlist deletes a wishlist
func (s *WishlistServiceImpl) DeleteWishlist(
	ctx context.Context,
	userID, wishlistID uint,
) error {
	wishlist, err := s.wishlistRepo.FindByID(ctx, wishlistID)
	if err != nil {
		return err
	}

	// Verify ownership
	if wishlist.UserID != userID {
		return prodErrors.ErrUnauthorizedWishlist
	}

	// Cannot delete default wishlist
	if wishlist.IsDefault {
		return prodErrors.ErrCannotDeleteDefault
	}

	return s.wishlistRepo.Delete(ctx, wishlistID)
}
