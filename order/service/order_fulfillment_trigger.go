package service

import (
	"context"

	"ecommerce-be/common/log"
	"ecommerce-be/order/entity"
)

// afterConfirmTrigger runs the fulfillment planner once an order reaches
// CONFIRMED, post-commit only. Planner failures never fail order operations:
// planning is idempotent and recoverable via explicit replan, while a
// failed confirmation would strand money and stock. Orders without a seller
// (nil SellerID) never plan.
func (s *OrderServiceImpl) afterConfirmTrigger(ctx context.Context, orderID uint, sellerID *uint) {
	if s.shipmentPlanner == nil || sellerID == nil {
		return
	}
	if _, err := s.shipmentPlanner.PlanForOrder(ctx, *sellerID, orderID); err != nil {
		log.ErrorWithContext(ctx, "order confirm: auto-plan failed (recover via replan)", err)
	}
}

// confirmedTarget reports whether a status move lands on CONFIRMED.
func confirmedTarget(target entity.OrderStatus) bool {
	return target == entity.ORDER_STATUS_CONFIRMED
}
