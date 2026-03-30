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
	FindActiveCartByUserID(ctx context.Context, userID uint) (*entity.Cart, error)
	FindCheckoutCartByUserID(ctx context.Context, userID uint) (*entity.Cart, error)
	FindByOrderID(ctx context.Context, orderID uint) (*entity.Cart, error)
	FindByID(ctx context.Context, cartID uint) (*entity.Cart, error)
	CreateCart(ctx context.Context, cart *entity.Cart) error
	CreateNewActiveCart(ctx context.Context, userID uint) (*entity.Cart, error)
	UpdateCartStatus(ctx context.Context, cartID uint, status entity.CartStatus) error
	UpdateCartStatusIfCurrent(
		ctx context.Context,
		cartID uint,
		currentStatus, newStatus entity.CartStatus,
	) (bool, error)
	SetCartOrderID(ctx context.Context, cartID uint, orderID uint) error
	ClearCartOrderID(ctx context.Context, cartID uint) error
	DeleteCart(ctx context.Context, cartID uint) error

	// Cart item operations
	FindItemByVariantID(ctx context.Context, cartID, variantID uint) (*entity.CartItem, error)
	FindItemsByCartID(ctx context.Context, cartID uint) ([]entity.CartItem, error)
	DeleteItemsByCartID(ctx context.Context, cartID uint) error
	AddItem(ctx context.Context, item *entity.CartItem) error
	UpdateItem(ctx context.Context, item *entity.CartItem) error
	DeleteItem(ctx context.Context, itemID uint) error
}

type CartRepositoryImpl struct{}

func NewCartRepository() CartRepository {
	return &CartRepositoryImpl{}
}

// FindByUserID finds the active cart for a user. Returns not found error if none exists.
func (r *CartRepositoryImpl) FindByUserID(ctx context.Context, userID uint) (*entity.Cart, error) {
	return r.FindActiveCartByUserID(ctx, userID)
}

// FindActiveCartByUserID finds the active cart for a user. Returns not found error if none exists.
func (r *CartRepositoryImpl) FindActiveCartByUserID(
	ctx context.Context,
	userID uint,
) (*entity.Cart, error) {
	var cart entity.Cart
	result := db.DB(ctx).
		Where("user_id = ? AND status = ?", userID, entity.CART_STATUS_ACTIVE).
		First(&cart)

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

// FindCheckoutCartByUserID finds a user's currently locked checkout cart.
// Returns nil,nil when no checkout cart exists.
func (r *CartRepositoryImpl) FindCheckoutCartByUserID(
	ctx context.Context,
	userID uint,
) (*entity.Cart, error) {
	var cart entity.Cart
	result := db.DB(ctx).
		Where("user_id = ? AND status = ?", userID, entity.CART_STATUS_CHECKOUT).
		First(&cart)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, nil
		}
		log.ErrorWithContext(ctx, "Failed to find checkout cart by user ID", result.Error)
		return nil, errs.DatabaseError(orderConstants.FAILED_TO_FETCH_CART_MSG)
	}
	return &cart, nil
}

// FindByOrderID finds the converted cart linked to the order.
// Returns nil,nil when no cart is linked (safe for idempotent compensation flows).
func (r *CartRepositoryImpl) FindByOrderID(
	ctx context.Context,
	orderID uint,
) (*entity.Cart, error) {
	var cart entity.Cart
	result := db.DB(ctx).Where("order_id = ?", orderID).First(&cart)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, nil
		}
		log.ErrorWithContext(ctx, "Failed to find cart by order ID", result.Error)
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

// CreateNewActiveCart creates a new empty active cart for a user.
func (r *CartRepositoryImpl) CreateNewActiveCart(
	ctx context.Context,
	userID uint,
) (*entity.Cart, error) {
	cart := &entity.Cart{
		UserID:   userID,
		Status:   entity.CART_STATUS_ACTIVE,
		Metadata: db.JSONMap{},
	}
	if err := db.DB(ctx).Create(cart).Error; err != nil {
		log.ErrorWithContext(ctx, "Failed to create new active cart", err)
		return nil, errs.DatabaseError(orderConstants.FAILED_TO_INSERT_CART_RECORD_MSG)
	}
	return cart, nil
}

// UpdateCartStatus updates cart lifecycle status.
func (r *CartRepositoryImpl) UpdateCartStatus(
	ctx context.Context,
	cartID uint,
	status entity.CartStatus,
) error {
	if err := db.DB(ctx).
		Model(&entity.Cart{}).
		Where("id = ?", cartID).
		Update("status", status).
		Error; err != nil {
		log.ErrorWithContext(ctx, "Failed to update cart status", err)
		return errs.DatabaseError(orderConstants.FAILED_TO_UPDATE_CART_RECORD_MSG)
	}
	return nil
}

// UpdateCartStatusIfCurrent updates status only if current status matches.
// Returns true when exactly one row transitioned.
func (r *CartRepositoryImpl) UpdateCartStatusIfCurrent(
	ctx context.Context,
	cartID uint,
	currentStatus, newStatus entity.CartStatus,
) (bool, error) {
	result := db.DB(ctx).
		Model(&entity.Cart{}).
		Where("id = ? AND status = ?", cartID, currentStatus).
		Update("status", newStatus)
	if result.Error != nil {
		log.ErrorWithContext(ctx, "Failed to transition cart status", result.Error)
		return false, errs.DatabaseError(orderConstants.FAILED_TO_UPDATE_CART_RECORD_MSG)
	}
	return result.RowsAffected == 1, nil
}

// SetCartOrderID links cart to an order ID after successful conversion.
func (r *CartRepositoryImpl) SetCartOrderID(
	ctx context.Context,
	cartID uint,
	orderID uint,
) error {
	if err := db.DB(ctx).
		Model(&entity.Cart{}).
		Where("id = ?", cartID).
		Update("order_id", orderID).
		Error; err != nil {
		log.ErrorWithContext(ctx, "Failed to set cart order ID", err)
		return errs.DatabaseError(orderConstants.FAILED_TO_UPDATE_CART_RECORD_MSG)
	}
	return nil
}

// ClearCartOrderID clears order linkage from a cart.
func (r *CartRepositoryImpl) ClearCartOrderID(ctx context.Context, cartID uint) error {
	if err := db.DB(ctx).
		Model(&entity.Cart{}).
		Where("id = ?", cartID).
		Update("order_id", nil).
		Error; err != nil {
		log.ErrorWithContext(ctx, "Failed to clear cart order ID", err)
		return errs.DatabaseError(orderConstants.FAILED_TO_UPDATE_CART_RECORD_MSG)
	}
	return nil
}

// FindByID finds a cart by ID. Returns not found error if none exists.
func (r *CartRepositoryImpl) FindByID(ctx context.Context, cartID uint) (*entity.Cart, error) {
	var cart entity.Cart
	result := db.DB(ctx).Where("id = ?", cartID).First(&cart)

	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, errs.NewAppError(
				errs.INVALID_ID_CODE,
				orderConstants.CART_NOT_FOUND_MSG,
				404,
			)
		}
		log.ErrorWithContext(ctx, "Failed to find cart by cart ID", result.Error)
		return nil, errs.DatabaseError(orderConstants.FAILED_TO_FETCH_CART_MSG)
	}

	return &cart, nil
}

// DeleteCart deletes a cart by ID
func (r *CartRepositoryImpl) DeleteCart(ctx context.Context, cartID uint) error {
	if err := db.DB(ctx).Delete(&entity.Cart{}, cartID).Error; err != nil {
		log.ErrorWithContext(ctx, "Failed to delete cart", err)
		return errs.DatabaseError(orderConstants.FAILED_TO_DELETE_CART_RECORD_MSG)
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

// DeleteItemsByCartID deletes all cart items for a given cart
func (r *CartRepositoryImpl) DeleteItemsByCartID(ctx context.Context, cartID uint) error {
	if err := db.DB(ctx).Where("cart_id = ?", cartID).Delete(&entity.CartItem{}).Error; err != nil {
		log.ErrorWithContext(ctx, "Failed to delete cart items", err)
		return errs.DatabaseError(orderConstants.FAILED_TO_DELETE_CART_RECORD_MSG)
	}
	return nil
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

// DeleteItem deletes a cart item by ID
func (r *CartRepositoryImpl) DeleteItem(ctx context.Context, itemID uint) error {
	if err := db.DB(ctx).Delete(&entity.CartItem{}, itemID).Error; err != nil {
		log.ErrorWithContext(ctx, "Failed to delete cart item", err)
		return errs.DatabaseError(orderConstants.FAILED_TO_DELETE_CART_RECORD_MSG)
	}
	return nil
}
