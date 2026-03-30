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
}

// NewWishlistService creates a new instance of WishlistService
func NewWishlistService(
	wishlistRepo repository.WishlistRepository,
	wishlistItemRepo repository.WishlistItemRepository,
	productQueryService ProductQueryService,
) WishlistService {
	return &WishlistServiceImpl{
		wishlistRepo:        wishlistRepo,
		wishlistItemRepo:    wishlistItemRepo,
		productQueryService: productQueryService,
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
// Uses ProductQueryService to get full product details for wishlist items
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

	// Get variant IDs from wishlist items with pagination
	variantIDs, totalItems, err := s.wishlistItemRepo.FindVariantIDsByWishlistID(
		ctx,
		wishlistID,
		page,
		pageSize,
	)
	if err != nil {
		return nil, err
	}

	// Build products response
	var productsResponse model.ProductsResponse

	if len(variantIDs) > 0 {
		// Get products using the variant IDs filter
		filter := model.GetProductsFilter{
			VariantIDs: variantIDs,
		}
		// Use page 1 and high limit since we already paginated variant IDs
		// This fetches products for the paginated variant IDs
		products, err := s.productQueryService.GetAllProducts(
			ctx,
			1,
			len(variantIDs),
			filter,
			&userID,
		)
		if err != nil {
			return nil, err
		}
		productsResponse = model.ProductsResponse{
			Products:   products.Products,
			Pagination: common.NewPaginationResponse(page, pageSize, totalItems),
		}
	} else {
		productsResponse = model.ProductsResponse{
			Products:   []model.ProductResponse{},
			Pagination: common.NewPaginationResponse(page, pageSize, 0),
		}
	}

	return &model.WishlistDetailResponse{
		ID:        wishlist.ID,
		Name:      wishlist.Name,
		IsDefault: wishlist.IsDefault,
		ItemCount: int(totalItems),
		Products:  productsResponse,
		CreatedAt: wishlist.CreatedAt,
		UpdatedAt: wishlist.UpdatedAt,
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
