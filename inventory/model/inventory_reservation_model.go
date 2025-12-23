package model

import "ecommerce-be/inventory/entity"

type ReservationItem struct {
	VariantID        uint `json:"variantId"        binding:"required"`
	ReservedQuantity uint `json:"reservedQuantity" binding:"required,gt=0"`
}

type ReservationRequest struct {
	ReferenceId      uint              `json:"referenceId"      binding:"required"`
	ExpiresInMinutes uint              `json:"expiresInMinutes" binding:"required,gt=0"`
	Items            []ReservationItem `json:"items"            binding:"required,min=1,dive"`
}

type Resevation struct {
	Id                         uint                     `json:"id"`
	InventoryId                uint                     `json:"inventoryId"`
	Quantity                   uint                     `json:"quantity"`
	Status                     entity.ReservationStatus `json:"status"`
	TotalAvailableAfterReserve int                      `json:"totalAvailableAfterReserve"`
}

type ReservationResponse struct {
	ReferenceId uint         `json:"referenceId"`
	ExpiresAt   string       `json:"expiresAt"`
	Resevations []Resevation `json:"resevations"`
}

// UpdateReservationStatusRequest represents the request payload for updating
// the status of reservations based on a reference ID.
// This API updates the status of all reservations with the given reference ID
// that are currently in the pending state.
type UpdateReservationStatusRequest struct {
	ReferenceId uint                     `json:"referenceId" binding:"required"`
	Status      entity.ReservationStatus `json:"status"      binding:"required,oneof='CANCELLED' 'COMPLETED'"`
}
