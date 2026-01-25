package service

import (
	"context"

	prodErrors "ecommerce-be/product/error"
	"ecommerce-be/product/factory"
	"ecommerce-be/product/model"
	"ecommerce-be/product/repository"
	"ecommerce-be/product/utils"
)

// WishlistService defines the interface for wishlist management business logic
type WishlistService interface {
	GetAllWishlists(ctx context.Context, userID uint) (*model.WishlistsResponse, error)
	GetWishlistByID(ctx context.Context, userID, wishlistID uint) (*model.WishlistDetailResponse, error)
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
	wishlistRepo repository.WishlistRepository
}

// NewWishlistService creates a new instance of WishlistService
func NewWishlistService(wishlistRepo repository.WishlistRepository) WishlistService {
	return &WishlistServiceImpl{
		wishlistRepo: wishlistRepo,
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

// GetWishlistByID retrieves a wishlist with its items
func (s *WishlistServiceImpl) GetWishlistByID(
	ctx context.Context,
	userID, wishlistID uint,
) (*model.WishlistDetailResponse, error) {
	wishlist, err := s.wishlistRepo.FindByIDWithItems(ctx, wishlistID)
	if err != nil {
		return nil, err
	}

	// Verify ownership
	if wishlist.UserID != userID {
		return nil, prodErrors.ErrUnauthorizedWishlist
	}

	return factory.BuildWishlistDetailResponse(wishlist), nil
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
	if count >= utils.MAX_WISHLISTS_PER_USER {
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
