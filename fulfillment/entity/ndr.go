package entity

import (
	"time"

	"ecommerce-be/common/db"
)

// ============================================================================
// FulfillmentNDR — one open round per shipment (032, T8).
// A repeat reason is a NEW round (next attempt_no) after the previous one is
// actioned — never blocked by reason text. Partial unique index enforces a
// single open round (action_taken IS NULL) per shipment.
// ============================================================================

type FulfillmentNDR struct {
	db.BaseEntity
	ShipmentID uint   `json:"shipmentId" gorm:"column:shipment_id;not null;index"`
	AWB        string `json:"awb"        gorm:"column:awb;not null"`

	AttemptNo int     `json:"attemptNo" gorm:"column:attempt_no;not null"`
	NDRStatus string  `json:"ndrStatus" gorm:"column:ndr_status;size:50;not null"`
	Reason    *string `json:"reason"   gorm:"column:reason"`

	// 'reattempt', 'rto'. NULL = round still open.
	ActionTaken *string    `json:"actionTaken" gorm:"column:action_taken;size:30"`
	ActedAt     *time.Time `json:"actedAt"     gorm:"column:acted_at"`
}

// TableName returns the singular table name.
func (FulfillmentNDR) TableName() string {
	return "fulfillment_ndr"
}
