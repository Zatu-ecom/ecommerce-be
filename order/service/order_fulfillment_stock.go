package service

import (
	"context"
	"fmt"

	"ecommerce-be/common/log"
	fulfillmentmodel "ecommerce-be/fulfillment/model"
)

// InventoryStockResolver is the narrow inventory surface this file needs:
// release of CONFIRMED holds and once-only return restocks. It is declared
// locally (structural, no import) so the order module never depends on the
// fulfillment module or vice versa; the inventory implementation
// (inventory/service/inventory_fulfillment_hooks.go) satisfies it
// method-for-method, asserted by the T022 conformance test.
type InventoryStockResolver interface {
	ReleaseReservation(ctx context.Context, sellerID uint, lines []fulfillmentmodel.ReservationLine) error
	RestockForReturn(
		ctx context.Context,
		sellerID uint,
		orderID uint,
		lines []fulfillmentmodel.RestockLine,
		returnShipmentID uint,
	) error
	FulfillHolds(ctx context.Context, sellerID uint, orderID uint, lines []fulfillmentmodel.ReservationLine) error
	UnfulfillHolds(ctx context.Context, sellerID uint, orderID uint, lines []fulfillmentmodel.ReservationLine) error
}

// releaseBoxHolds returns the CONFIRMED holds backing one failed box.
// Called after the order moves to CANCELLED. Variants resolve from the
// order's own lines; LocationID 0 releases across warehouses for the order.
func (h *OrderFulfillmentHooksImpl) releaseBoxHolds(
	ctx context.Context,
	sellerID, orderID uint,
	p fulfillmentmodel.FulfillmentProgress,
	variantByItem map[uint]*uint,
) error {
	if len(p.Items) == 0 {
		return nil
	}
	lines := make([]fulfillmentmodel.ReservationLine, 0, len(p.Items))
	for _, item := range p.Items {
		lines = append(lines, fulfillmentmodel.ReservationLine{
			OrderID:   orderID,
			VariantID: variantByItem[item.OrderItemID],
			Quantity:  item.Quantity,
		})
	}
	if err := h.inventoryStock.ReleaseReservation(ctx, sellerID, lines); err != nil {
		return fmt.Errorf("fulfillment hook release holds order %d: %w", orderID, err)
	}
	return nil
}

// restockReturnBoxes returns shipped units to stock exactly once per return
// box. The inventory implementation keys idempotency on the return shipment
// id, so double-observed returns restock nothing the second time.
func (h *OrderFulfillmentHooksImpl) restockReturnBoxes(
	ctx context.Context,
	sellerID, orderID, returnShipmentID uint,
	p fulfillmentmodel.FulfillmentProgress,
	variantByItem map[uint]*uint,
) error {
	if len(p.Items) == 0 {
		return nil
	}
	lines := make([]fulfillmentmodel.RestockLine, 0, len(p.Items))
	for _, item := range p.Items {
		lines = append(lines, fulfillmentmodel.RestockLine{
			VariantID: variantByItem[item.OrderItemID],
			Quantity:  item.Quantity,
		})
	}
	if err := h.inventoryStock.RestockForReturn(ctx, sellerID, orderID, lines, returnShipmentID); err != nil {
		return fmt.Errorf("fulfillment hook restock order %d: %w", orderID, err)
	}
	log.InfoWithContext(ctx, fmt.Sprintf("fulfillment restocked return box %d", returnShipmentID))
	return nil
}
