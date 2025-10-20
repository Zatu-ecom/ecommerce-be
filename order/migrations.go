package order

import "ecommerce-be/order/entity"

type OrderAutoMigrate struct{}

func (o *OrderAutoMigrate) AutoMigrate() []interface{} {
	return []interface{}{
		&entity.Order{},
		&entity.OrderItem{},
		&entity.OrderShipment{},
		&entity.Cart{},
		&entity.CartItem{},
		&entity.Wishlist{},
		&entity.WishlistItem{},
	}
}

func NewOrderAutoMigrate() *OrderAutoMigrate {
	return &OrderAutoMigrate{}
}
