package helpers

import (
	"time"

	orderEntity "ecommerce-be/order/entity"
)

// OrderEntityBuilder builds order rows for direct DB seeding in integration tests.
type OrderEntityBuilder struct {
	order orderEntity.Order
}

// NewOrderEntity starts a minimal order entity (caller must set user/status/totals as needed).
func NewOrderEntity() *OrderEntityBuilder {
	return &OrderEntityBuilder{
		order: orderEntity.Order{},
	}
}

func (b *OrderEntityBuilder) UserID(id uint) *OrderEntityBuilder {
	b.order.UserID = id
	return b
}

func (b *OrderEntityBuilder) OrderNumber(number string) *OrderEntityBuilder {
	b.order.OrderNumber = number
	return b
}

func (b *OrderEntityBuilder) Status(status orderEntity.OrderStatus) *OrderEntityBuilder {
	b.order.Status = status
	return b
}

func (b *OrderEntityBuilder) Completed() *OrderEntityBuilder {
	return b.Status(orderEntity.ORDER_STATUS_COMPLETED)
}

func (b *OrderEntityBuilder) Confirmed() *OrderEntityBuilder {
	return b.Status(orderEntity.ORDER_STATUS_CONFIRMED)
}

func (b *OrderEntityBuilder) Cancelled() *OrderEntityBuilder {
	return b.Status(orderEntity.ORDER_STATUS_CANCELLED)
}

func (b *OrderEntityBuilder) TotalCents(cents int64) *OrderEntityBuilder {
	b.order.TotalCents = cents
	return b
}

func (b *OrderEntityBuilder) PlacedAt(t time.Time) *OrderEntityBuilder {
	placed := t
	b.order.PlacedAt = &placed
	return b
}

func (b *OrderEntityBuilder) Build() orderEntity.Order {
	return b.order
}
