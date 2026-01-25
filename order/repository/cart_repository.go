package repository

import (
	"context"

	"gorm.io/gorm"

	"ecommerce-be/common/db"
	"ecommerce-be/order/entity"
)

// ============================================================================
// Cart Repository Interface
// ============================================================================

// CartRepository defines the interface for cart data operations
type CartRepository interface {
	// FindByUserID finds a cart by user ID
	FindByUserID(ctx context.Context, userID uint) (*entity.Cart, error)

	// FindOrCreateByUserID finds or creates a cart for a user
	FindOrCreateByUserID(ctx context.Context, userID uint) (*entity.Cart, error)

	// AddItem adds an item to the cart (or updates quantity if exists)
	AddItem(ctx context.Context, cartID, variantID uint, quantity int) (*entity.CartItem, error)

	// FindItemByCartAndVariant finds a cart item by cart and variant
	FindItemByCartAndVariant(ctx context.Context, cartID, variantID uint) (*entity.CartItem, error)

	// UpdateItemQuantity updates the quantity of a cart item
	UpdateItemQuantity(ctx context.Context, itemID uint, quantity int) error
}

// ============================================================================
// Cart Repository Implementation
// ============================================================================

type cartRepositoryImpl struct{}

// NewCartRepository creates a new instance of CartRepository
func NewCartRepository() CartRepository {
	return &cartRepositoryImpl{}
}

// getDB returns the database connection
func (r *cartRepositoryImpl) getDB() *gorm.DB {
	return db.GetDB()
}

// FindByUserID finds a cart by user ID
func (r *cartRepositoryImpl) FindByUserID(ctx context.Context, userID uint) (*entity.Cart, error) {
	var cart entity.Cart
	err := r.getDB().WithContext(ctx).
		Where("user_id = ?", userID).
		First(&cart).Error
	if err != nil {
		return nil, err
	}
	return &cart, nil
}

// FindOrCreateByUserID finds or creates a cart for a user
func (r *cartRepositoryImpl) FindOrCreateByUserID(ctx context.Context, userID uint) (*entity.Cart, error) {
	var cart entity.Cart
	err := r.getDB().WithContext(ctx).
		Where("user_id = ?", userID).
		First(&cart).Error

	if err == gorm.ErrRecordNotFound {
		// Create new cart
		cart = entity.Cart{
			UserID: &userID,
		}
		if err := r.getDB().WithContext(ctx).Create(&cart).Error; err != nil {
			return nil, err
		}
		return &cart, nil
	}

	if err != nil {
		return nil, err
	}
	return &cart, nil
}

// AddItem adds an item to the cart (or updates quantity if exists)
func (r *cartRepositoryImpl) AddItem(ctx context.Context, cartID, variantID uint, quantity int) (*entity.CartItem, error) {
	// Check if item already exists
	var existingItem entity.CartItem
	err := r.getDB().WithContext(ctx).
		Where("cart_id = ? AND variant_id = ?", cartID, variantID).
		First(&existingItem).Error

	if err == nil {
		// Item exists, update quantity
		existingItem.Quantity += quantity
		if err := r.getDB().WithContext(ctx).Save(&existingItem).Error; err != nil {
			return nil, err
		}
		return &existingItem, nil
	}

	if err != gorm.ErrRecordNotFound {
		return nil, err
	}

	// Create new cart item
	newItem := &entity.CartItem{
		CartID:    cartID,
		VariantID: variantID,
		Quantity:  quantity,
	}
	if err := r.getDB().WithContext(ctx).Create(newItem).Error; err != nil {
		return nil, err
	}
	return newItem, nil
}

// FindItemByCartAndVariant finds a cart item by cart and variant
func (r *cartRepositoryImpl) FindItemByCartAndVariant(ctx context.Context, cartID, variantID uint) (*entity.CartItem, error) {
	var item entity.CartItem
	err := r.getDB().WithContext(ctx).
		Where("cart_id = ? AND variant_id = ?", cartID, variantID).
		First(&item).Error
	if err != nil {
		return nil, err
	}
	return &item, nil
}

// UpdateItemQuantity updates the quantity of a cart item
func (r *cartRepositoryImpl) UpdateItemQuantity(ctx context.Context, itemID uint, quantity int) error {
	return r.getDB().WithContext(ctx).
		Model(&entity.CartItem{}).
		Where("id = ?", itemID).
		Update("quantity", quantity).Error
}
