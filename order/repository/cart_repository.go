package repository

import (
	"context"

	"ecommerce-be/common/db"
	errs "ecommerce-be/common/error"
	"ecommerce-be/common/log"
	"ecommerce-be/order/entity"
	orderConstants "ecommerce-be/order/utils/constant"

	"gorm.io/gorm"
)

type CartRepository interface {
	// Cart operations
	FindByUserID(ctx context.Context, userID uint) (*entity.Cart, error)
	CreateCart(ctx context.Context, cart *entity.Cart) error

	// Cart item operations
	FindItemByVariantID(ctx context.Context, cartID, variantID uint) (*entity.CartItem, error)
	FindItemsByCartID(ctx context.Context, cartID uint) ([]entity.CartItem, error)
	AddItem(ctx context.Context, item *entity.CartItem) error
	UpdateItem(ctx context.Context, item *entity.CartItem) error
}

type CartRepositoryImpl struct{}

func NewCartRepository() CartRepository {
	return &CartRepositoryImpl{}
}

// FindByUserID finds the active cart for a user. Returns not found error if none exists.
func (r *CartRepositoryImpl) FindByUserID(ctx context.Context, userID uint) (*entity.Cart, error) {
	var cart entity.Cart
	result := db.DB(ctx).Where("user_id = ?", userID).First(&cart)

	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, errs.NewAppError(
				errs.INVALID_ID_CODE,
				orderConstants.CART_NOT_FOUND_MSG,
				404,
			)
		}
		log.ErrorWithContext(ctx, "Failed to find cart by user ID", result.Error)
		return nil, errs.DatabaseError(orderConstants.FAILED_TO_FETCH_CART_MSG)
	}

	return &cart, nil
}

// CreateCart creates a new cart for a user
func (r *CartRepositoryImpl) CreateCart(ctx context.Context, cart *entity.Cart) error {
	if err := db.DB(ctx).Create(cart).Error; err != nil {
		log.ErrorWithContext(ctx, "Failed to create cart", err)
		return errs.DatabaseError(orderConstants.FAILED_TO_INSERT_CART_RECORD_MSG)
	}
	return nil
}

// FindItemByVariantID finds a specific item in a cart by variant ID
func (r *CartRepositoryImpl) FindItemByVariantID(
	ctx context.Context,
	cartID, variantID uint,
) (*entity.CartItem, error) {
	var item entity.CartItem
	result := db.DB(ctx).
		Where("cart_id = ? AND variant_id = ?", cartID, variantID).
		First(&item)

	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, nil // Return nil, nil when not found to indicate it's a new item addition
		}
		log.ErrorWithContext(ctx, "Failed to find cart item", result.Error)
		return nil, errs.DatabaseError(orderConstants.FAILED_TO_FETCH_CART_ITEM_MSG)
	}

	return &item, nil
}

// FindItemsByCartID gets all items for a given cart
func (r *CartRepositoryImpl) FindItemsByCartID(
	ctx context.Context,
	cartID uint,
) ([]entity.CartItem, error) {
	var items []entity.CartItem
	err := db.DB(ctx).
		Where("cart_id = ?", cartID).
		Find(&items).Error
	if err != nil {
		log.ErrorWithContext(ctx, "Failed to fetch cart items", err)
		return nil, errs.DatabaseError(orderConstants.FAILED_TO_FETCH_CART_ITEMS_MSG)
	}

	return items, nil
}

// AddItem adds a new item to the cart
func (r *CartRepositoryImpl) AddItem(ctx context.Context, item *entity.CartItem) error {
	if err := db.DB(ctx).Create(item).Error; err != nil {
		log.ErrorWithContext(ctx, "Failed to add cart item", err)
		return errs.DatabaseError(orderConstants.FAILED_TO_INSERT_CART_RECORD_MSG)
	}
	return nil
}

// UpdateItem updates an existing cart item (e.g. quantity)
func (r *CartRepositoryImpl) UpdateItem(ctx context.Context, item *entity.CartItem) error {
	if err := db.DB(ctx).Save(item).Error; err != nil {
		log.ErrorWithContext(ctx, "Failed to update cart item", err)
		return errs.DatabaseError(orderConstants.FAILED_TO_UPDATE_CART_RECORD_MSG)
	}
	return nil
}
