package model

// ReservationExpiryPayload represents the job payload for reservation expiry
// Used by both the scheduler service and the job handler
type ReservationExpiryPayload struct {
	// For single reservation expiry
	ReservationID uint `json:"reservationID,omitempty"`

	// For bulk reservation expiry
	ReservationIDs []uint `json:"reservationIDs,omitempty"`
	ReferenceID    uint   `json:"referenceID,omitempty"`

	// Flag to indicate if this is a bulk operation
	IsBulk bool `json:"isBulk"`
}
