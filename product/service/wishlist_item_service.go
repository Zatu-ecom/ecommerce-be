package service

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"ecommerce-be/common/config"
	"ecommerce-be/product/entity"
	prodErrors "ecommerce-be/product/error"
	"ecommerce-be/product/model"
	prodRepo "ecommerce-be/product/repository"
)

// ============================================================================
// Wishlist Item Service Interface
// ============================================================================

// WishlistItemService defines the interface for wishlist item business operations
type WishlistItemService interface {
	// AddItem adds an item to a wishlist
	AddItem(
		ctx context.Context,
		userID, wishlistID uint,
		req model.WishlistItemCreateRequest,
	) (*model.WishlistItemResponse, error)

	// RemoveItem removes an item from a wishlist
	RemoveItem(
		ctx context.Context,
		userID, wishlistID, itemID uint,
	) error

	// MoveItem moves an item to another wishlist
	MoveItem(
		ctx context.Context,
		userID, wishlistID, itemID uint,
		req model.WishlistItemMoveRequest,
	) (*model.WishlistItemResponse, error)

	// IsVariantInUserWishlist checks if a variant is in any of user's wishlists
	// Used for isWishlisted field in product/variant responses
	IsVariantInUserWishlist(ctx context.Context, variantID, userID uint) (bool, error)

	// AreVariantsInUserWishlist checks if multiple variants are in any of user's wishlists
	// Returns a map of variantID -> isWishlisted for efficient batch lookup
	// Used for isWishlisted field in list endpoints (ListVariants, etc.)
	AreVariantsInUserWishlist(
		ctx context.Context,
		variantIDs []uint,
		userID uint,
	) (map[uint]bool, error)

	// GetWishlistItemsByVariantIDs returns wishlist item info for variant IDs.
	// Returns map of variantID -> []WishlistItemInfo for efficient per-variant enrichment.
	GetWishlistItemsByVariantIDs(
		ctx context.Context,
		variantIDs []uint,
		userID uint,
	) (map[uint][]model.WishlistItemInfo, error)
}

// ============================================================================
// Wishlist Item Service Implementation
// ============================================================================

type wishlistItemServiceImpl struct {
	wishlistItemRepo prodRepo.WishlistItemRepository
	wishlistRepo     prodRepo.WishlistRepository
	variantRepo      prodRepo.VariantRepository
	productRepo      prodRepo.ProductRepository
}

// NewWishlistItemService creates a new instance of WishlistItemService
func NewWishlistItemService(
	wishlistItemRepo prodRepo.WishlistItemRepository,
	wishlistRepo prodRepo.WishlistRepository,
	variantRepo prodRepo.VariantRepository,
	productRepo prodRepo.ProductRepository,
) WishlistItemService {
	return &wishlistItemServiceImpl{
		wishlistItemRepo: wishlistItemRepo,
		wishlistRepo:     wishlistRepo,
		variantRepo:      variantRepo,
		productRepo:      productRepo,
	}
}

// AddItem adds an item to a wishlist
// If wishlist has reached MaxWishlistItems (configurable, default 100), removes the oldest item first
// The req.VariantID is resolved defensively: if the ID is a product ID (not found in the
// variant table), the product's default variant is used automatically.
func (s *wishlistItemServiceImpl) AddItem(
	ctx context.Context,
	userID, wishlistID uint,
	req model.WishlistItemCreateRequest,
) (*model.WishlistItemResponse, error) {
	// Verify wishlist exists and belongs to user
	wishlist, err := s.wishlistRepo.FindByID(ctx, wishlistID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, prodErrors.ErrWishlistNotFound
		}
		return nil, err
	}

	// Verify ownership
	if wishlist.UserID != userID {
		return nil, prodErrors.ErrUnauthorizedWishlist
	}

	// Resolve the variant ID (handles product ID → default variant fallback)
	variantID, err := s.resolveVariantID(ctx, req.VariantID)
	if err != nil {
		return nil, err
	}

	// Check if item already exists in wishlist
	exists, err := s.wishlistItemRepo.ExistsByWishlistIDAndVariantID(ctx, wishlistID, variantID)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, prodErrors.ErrWishlistItemExists
	}

	// Check wishlist item count and remove oldest if limit reached
	itemCount, err := s.wishlistItemRepo.CountByWishlistID(ctx, wishlistID)
	if err != nil {
		return nil, err
	}

	if itemCount >= int64(config.Get().App.MaxWishlistItems) {
		// Remove oldest item to make room for new item
		if err := s.wishlistItemRepo.DeleteOldestByWishlistID(ctx, wishlistID); err != nil {
			return nil, err
		}
	}

	// Create wishlist item
	item := &entity.WishlistItem{
		WishlistID: wishlistID,
		VariantID:  variantID,
	}

	if err := s.wishlistItemRepo.Create(ctx, item); err != nil {
		return nil, err
	}

	return &model.WishlistItemResponse{
		ID:        item.ID,
		VariantID: item.VariantID,
		CreatedAt: item.CreatedAt,
	}, nil
}

// resolveVariantID resolves an ID to a valid variant ID.
// If the ID is already a valid variant ID, returns it as-is.
// If the ID is not a variant but is a valid product ID, finds the product's
// default variant (IsDefault=true, fallback to lowest ID) and returns that.
// Returns ErrVariantNotFound if neither a variant nor product is found.
func (s *wishlistItemServiceImpl) resolveVariantID(ctx context.Context, id uint) (uint, error) {
	// Try as variant ID first
	_, err := s.variantRepo.FindVariantByID(ctx, id)
	if err == nil {
		return id, nil
	}

	// If it's not "variant not found", it's a real DB error — propagate it
	if !errors.Is(err, prodErrors.ErrVariantNotFound) {
		return 0, err
	}

	// Try as product ID — find the product's variants and pick the default
	product, err := s.productRepo.FindByID(ctx, id)
	if err != nil {
		// Product not found either — return the original variant-not-found error
		return 0, prodErrors.ErrVariantNotFound
	}

	variants, err := s.variantRepo.FindVariantsByProductID(ctx, product.ID)
	if err != nil {
		return 0, err
	}

	if len(variants) == 0 {
		return 0, prodErrors.ErrVariantNotFound
	}

	defaultVariant := findDefaultVariantEntity(variants)
	if defaultVariant == nil {
		return 0, prodErrors.ErrVariantNotFound
	}

	return defaultVariant.ID, nil
}

// findDefaultVariantEntity returns the default variant from a slice.
// Prefers the variant with IsDefault=true; otherwise returns the one with the lowest ID.

// RemoveItem removes an item from a wishlist
func (s *wishlistItemServiceImpl) RemoveItem(
	ctx context.Context,
	userID, wishlistID, itemID uint,
) error {
	// Verify wishlist exists and belongs to user
	wishlist, err := s.wishlistRepo.FindByID(ctx, wishlistID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return prodErrors.ErrWishlistNotFound
		}
		return err
	}

	// Verify ownership
	if wishlist.UserID != userID {
		return prodErrors.ErrUnauthorizedWishlist
	}

	// Find the wishlist item
	item, err := s.wishlistItemRepo.FindByID(ctx, itemID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return prodErrors.ErrWishlistItemNotFound
		}
		return err
	}

	// Verify item belongs to the wishlist
	if item.WishlistID != wishlistID {
		return prodErrors.ErrWishlistItemNotFound
	}

	// Delete the item
	return s.wishlistItemRepo.Delete(ctx, itemID)
}

// MoveItem moves an item to another wishlist
func (s *wishlistItemServiceImpl) MoveItem(
	ctx context.Context,
	userID, wishlistID, itemID uint,
	req model.WishlistItemMoveRequest,
) (*model.WishlistItemResponse, error) {
	// Verify source wishlist exists and belongs to user
	sourceWishlist, err := s.wishlistRepo.FindByID(ctx, wishlistID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, prodErrors.ErrWishlistNotFound
		}
		return nil, err
	}

	if sourceWishlist.UserID != userID {
		return nil, prodErrors.ErrUnauthorizedWishlist
	}

	// Cannot move to the same wishlist
	if wishlistID == req.TargetWishlistID {
		return nil, prodErrors.ErrSameWishlistMove
	}

	// Verify target wishlist exists and belongs to user
	targetWishlist, err := s.wishlistRepo.FindByID(ctx, req.TargetWishlistID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, prodErrors.ErrWishlistNotFound
		}
		return nil, err
	}

	if targetWishlist.UserID != userID {
		return nil, prodErrors.ErrUnauthorizedWishlist
	}

	// Find the wishlist item
	item, err := s.wishlistItemRepo.FindByID(ctx, itemID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, prodErrors.ErrWishlistItemNotFound
		}
		return nil, err
	}

	// Verify item belongs to the source wishlist
	if item.WishlistID != wishlistID {
		return nil, prodErrors.ErrWishlistItemNotFound
	}

	// Check if item already exists in target wishlist
	exists, err := s.wishlistItemRepo.ExistsByWishlistIDAndVariantID(
		ctx,
		req.TargetWishlistID,
		item.VariantID,
	)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, prodErrors.ErrWishlistItemExists
	}

	// Move item to target wishlist
	item.WishlistID = req.TargetWishlistID
	if err := s.wishlistItemRepo.Update(ctx, item); err != nil {
		return nil, err
	}

	return &model.WishlistItemResponse{
		ID:        item.ID,
		VariantID: item.VariantID,
		CreatedAt: item.CreatedAt,
	}, nil
}

// IsVariantInUserWishlist checks if a variant is in any of user's wishlists
// This method can be extended with additional validation or business logic
func (s *wishlistItemServiceImpl) IsVariantInUserWishlist(
	ctx context.Context,
	variantID, userID uint,
) (bool, error) {
	return s.wishlistItemRepo.IsVariantInUserWishlist(ctx, variantID, userID)
}

// AreVariantsInUserWishlist checks if multiple variants are in any of user's wishlists
// Returns a map of variantID -> isWishlisted for efficient batch lookup
// This method can be extended with additional validation or business logic
func (s *wishlistItemServiceImpl) AreVariantsInUserWishlist(
	ctx context.Context,
	variantIDs []uint,
	userID uint,
) (map[uint]bool, error) {
	return s.wishlistItemRepo.AreVariantsInUserWishlist(ctx, variantIDs, userID)
}

// GetWishlistItemsByVariantIDs returns wishlist item info for variant IDs.
// Returns map of variantID -> []WishlistItemInfo for efficient per-variant enrichment.
// Delegates to repository for the raw SQL query.
func (s *wishlistItemServiceImpl) GetWishlistItemsByVariantIDs(
	ctx context.Context,
	variantIDs []uint,
	userID uint,
) (map[uint][]model.WishlistItemInfo, error) {
	items, err := s.wishlistItemRepo.FindWishlistItemsByVariantIDsAndUserID(ctx, variantIDs, userID)
	if err != nil {
		return nil, err
	}
	result := make(map[uint][]model.WishlistItemInfo)
	for _, item := range items {
		result[item.VariantID] = append(result[item.VariantID], model.WishlistItemInfo{
			WishlistItemID: item.ID,
			WishlistID:     item.WishlistID,
		})
	}
	return result, nil
}
