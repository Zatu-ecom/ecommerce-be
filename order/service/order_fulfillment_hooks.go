package service

import (
	"context"
	"errors"
	"fmt"

	"ecommerce-be/common/db"
	"ecommerce-be/common/log"
	fulfillmentmodel "ecommerce-be/fulfillment/model"
	"ecommerce-be/order/entity"
	orderError "ecommerce-be/order/error"
	"ecommerce-be/order/repository"
	"ecommerce-be/order/utils"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// OrderFulfillmentHooksImpl implements fulfillment's FulfillmentOrderHooks
// (see fulfillment/service/hooks.go for the contract). Method signatures
// match that interface method-for-method; the compile-time conformance
// assertion lives in the T022 integration test to avoid a production import
// cycle (order must stay importable without fulfillment and vice versa).
//
// Transition rule (fulfillment decides, order applies): the fulfillment
// applier owns all box rows and calls a hook only when an order-level move
// is due. Each method applies exactly one documented transition and no-ops
// (with an alert log) when the order is in an unexpected state.
type OrderFulfillmentHooksImpl struct {
	orderRepo        repository.OrderRepository
	orderHistoryRepo repository.OrderHistoryRepository
	inventoryStock   InventoryStockResolver
	currencyResolver CurrencyResolver
	profileResolver  ProfileResolver
}

// NewOrderFulfillmentHooks builds the fulfillment hook implementation.
// Wired by the fulfillment factory (Phase 3+); order module owns the code.
// Resolvers are optional (nil-safe): currency falls back to INR display,
// recipient fields stay empty.
func NewOrderFulfillmentHooks(
	orderRepo repository.OrderRepository,
	orderHistoryRepo repository.OrderHistoryRepository,
	inventoryStock InventoryStockResolver,
	currencyResolver CurrencyResolver,
	profileResolver ProfileResolver,
) *OrderFulfillmentHooksImpl {
	return &OrderFulfillmentHooksImpl{
		orderRepo:        orderRepo,
		orderHistoryRepo: orderHistoryRepo,
		inventoryStock:   inventoryStock,
		currencyResolver: currencyResolver,
		profileResolver:  profileResolver,
	}
}

// OnShipmentsPlanned observes draft creation. No order change by design.
func (h *OrderFulfillmentHooksImpl) OnShipmentsPlanned(
	ctx context.Context,
	orderID uint,
	draftIDs []uint,
) error {
	log.InfoWithContext(ctx,
		fmt.Sprintf("fulfillment planned %d drafts", len(draftIDs)),
	)
	return nil
}

// OnShipmentBooked consumes the box's holds into FULFILLED (courier custody
// begins). No order status change by design.
func (h *OrderFulfillmentHooksImpl) OnShipmentBooked(
	ctx context.Context,
	p fulfillmentmodel.FulfillmentProgress,
) error {
	order, err := h.orderRepo.FindOrderByID(ctx, p.OrderID)
	if err != nil {
		return err
	}
	if order == nil {
		return orderError.ErrOrderNotFound
	}
	if order.SellerID == nil {
		return fmt.Errorf("fulfillment hook: order %d has no seller", p.OrderID)
	}
	return h.inventoryStock.FulfillHolds(ctx, *order.SellerID, p.OrderID, h.holdLines(order, p))
}

// holdLines maps progress box lines to variant/location hold lines using
// the order's own items (variant linkage) and the box warehouse.
func (h *OrderFulfillmentHooksImpl) holdLines(
	order *entity.Order,
	p fulfillmentmodel.FulfillmentProgress,
) []fulfillmentmodel.ReservationLine {
	variantByItem := variantMap(order.Items)
	lines := make([]fulfillmentmodel.ReservationLine, 0, len(p.Items))
	for _, item := range p.Items {
		lines = append(lines, fulfillmentmodel.ReservationLine{
			OrderID:    p.OrderID,
			VariantID:  variantByItem[item.OrderItemID],
			LocationID: p.PickupLocationID,
			Quantity:   item.Quantity,
		})
	}
	return lines
}

// OnShipmentDelivered completes the order. Called only when every box is
// delivered. Order: CONFIRMED → COMPLETED.
func (h *OrderFulfillmentHooksImpl) OnShipmentDelivered(
	ctx context.Context,
	p fulfillmentmodel.FulfillmentProgress,
) error {
	return h.transitionOrder(ctx, p.OrderID, entity.ORDER_STATUS_COMPLETED, nil)
}

// OnShipmentFailed ends the order. CONFIRMED → CANCELLED with the reason
// noted. Already-COMPLETED orders are left alone (the refund flow owns
// them) with an alert.
func (h *OrderFulfillmentHooksImpl) OnShipmentFailed(
	ctx context.Context,
	p fulfillmentmodel.FulfillmentProgress,
) error {
	order, err := h.orderRepo.FindOrderByID(ctx, p.OrderID)
	if err != nil {
		return err
	}
	if order == nil {
		return orderError.ErrOrderNotFound
	}
	if order.Status == entity.ORDER_STATUS_COMPLETED {
		log.ErrorWithContext(ctx,
			"fulfillment box failed on completed order; leaving status, needs refund flow",
			fmt.Errorf("order %d completed, box failed", p.OrderID))
		return nil
	}
	if err := h.transitionOrder(ctx, p.OrderID, entity.ORDER_STATUS_CANCELLED, &p.Reason); err != nil {
		return err
	}
	if order.SellerID == nil {
		return fmt.Errorf("fulfillment hook: order %d has no seller", p.OrderID)
	}
	return h.releaseBoxHolds(ctx, *order.SellerID, p.OrderID, p, variantMap(order.Items))
}

// OnShipmentReturned returns the order. COMPLETED → RETURNED. An order that
// never completed (CONFIRMED) → CANCELLED: fulfillment ended without a
// completed sale.
func (h *OrderFulfillmentHooksImpl) OnShipmentReturned(
	ctx context.Context,
	p fulfillmentmodel.FulfillmentProgress,
) error {
	order, err := h.orderRepo.FindOrderByID(ctx, p.OrderID)
	if err != nil {
		return err
	}
	if order == nil {
		return orderError.ErrOrderNotFound
	}
	switch order.Status {
	case entity.ORDER_STATUS_COMPLETED:
		if err := h.transitionOrder(ctx, p.OrderID, entity.ORDER_STATUS_RETURNED, &p.Reason); err != nil {
			return err
		}
	case entity.ORDER_STATUS_CONFIRMED:
		if err := h.transitionOrder(ctx, p.OrderID, entity.ORDER_STATUS_CANCELLED, &p.Reason); err != nil {
			return err
		}
	default:
		log.ErrorWithContext(ctx,
			"fulfillment return on unexpected order status; leaving unchanged",
			fmt.Errorf("order %d status %s", p.OrderID, order.Status))
		return nil
	}
	// p.ShipmentID is the return box: restock exactly once per return box.
	if order.SellerID == nil {
		return fmt.Errorf("fulfillment hook: order %d has no seller", p.OrderID)
	}
	return h.restockReturnBoxes(ctx, *order.SellerID, p.OrderID, p.ShipmentID, p, variantMap(order.Items))
}

// WithOrderLock runs fn inside a transaction holding a row lock on the
// order. Callers re-check coverage after acquiring it: the lock serializes
// concurrent planners so only one can pass an empty-coverage guard.
func (h *OrderFulfillmentHooksImpl) WithOrderLock(
	ctx context.Context,
	orderID uint,
	fn func(lockCtx context.Context) error,
) error {
	return db.WithTransaction(ctx, func(txCtx context.Context) error {
		var locked entity.Order
		if err := db.DB(txCtx).
			Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ?", orderID).
			First(&locked).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return orderError.ErrOrderNotFound
			}
			return fmt.Errorf("fulfillment hook lock order: %w", err)
		}
		return fn(txCtx)
	})
}

// transitionOrder applies one validated status move plus a history row.
// Invalid transitions are no-ops with an alert (never forced).
func (h *OrderFulfillmentHooksImpl) transitionOrder(
	ctx context.Context,
	orderID uint,
	target entity.OrderStatus,
	reason *string,
) error {
	order, err := h.orderRepo.FindOrderByID(ctx, orderID)
	if err != nil {
		return err
	}
	if order == nil {
		return orderError.ErrOrderNotFound
	}
	if order.Status == target {
		return nil
	}
	if !utils.IsValidTransition(order.Status, target) {
		log.ErrorWithContext(ctx,
			fmt.Sprintf("fulfillment refusing order transition %s -> %s", order.Status, target),
			fmt.Errorf("invalid transition %s -> %s", order.Status, target),
		)
		return nil
	}

	from := string(order.Status)
	role := "system"
	return db.WithTransaction(ctx, func(txCtx context.Context) error {
		if err := h.orderRepo.UpdateOrderStatus(txCtx, orderID, target); err != nil {
			return fmt.Errorf("fulfillment hook update order status: %w", err)
		}
		return h.orderHistoryRepo.CreateHistoryEntry(txCtx, &entity.OrderHistory{
			OrderID:       orderID,
			FromStatus:    &from,
			ToStatus:      string(target),
			ChangedByRole: &role,
			FailureReason: reason,
			Note:          strPtr("fulfillment: shipment " + string(target)),
			Metadata:      db.JSONMap{},
		})
	})
}

func strPtr(s string) *string {
	return &s
}

// variantMap indexes order items by id for Progress → variant resolution.
func variantMap(items []entity.OrderItem) map[uint]*uint {
	m := make(map[uint]*uint, len(items))
	for i := range items {
		m[items[i].ID] = items[i].VariantID
	}
	return m
}
