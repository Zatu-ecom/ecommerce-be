package handler

import (
	"context"
	"encoding/json"

	"ecommerce-be/common/auth"
	"ecommerce-be/common/log"
	"ecommerce-be/inventory/model"
	"ecommerce-be/inventory/service"
)

type ScheduleInventoryReservationHandler struct {
	inventoryReservationService service.InventoryReservationService
}

func NewScheduleInventoryReservationHandler(
	inventoryReservationService service.InventoryReservationService,
) *ScheduleInventoryReservationHandler {
	return &ScheduleInventoryReservationHandler{
		inventoryReservationService: inventoryReservationService,
	}
}

func (h *ScheduleInventoryReservationHandler) ExpireScheduleReservation(
	ctx context.Context,
	payload json.RawMessage,
) error {
	var expiryPayload model.ReservationExpiryPayload
	if err := json.Unmarshal(payload, &expiryPayload); err != nil {
		log.ErrorWithContext(ctx, "Failed to unmarshal reservation expiry payload", err)
		return err
	}

	sellerID, exist := auth.GetSellerIDFromContext(ctx)
	if !exist {
		log.ErrorWithContext(ctx, "Seller ID missing in context", nil)
	}

	err := h.inventoryReservationService.ExpireScheduleReservation(ctx, sellerID, expiryPayload)
	if err != nil {
		log.ErrorWithContext(ctx, "Failed to expire scheduled reservation", err)
		return err
	}

	return nil
}
