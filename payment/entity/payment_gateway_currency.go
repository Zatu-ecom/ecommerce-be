package entity

// PaymentGatewayCurrency links a gateway to a currency it settles.
// Membership is checked with EXISTS queries against this join table.
type PaymentGatewayCurrency struct {
	GatewayID  uint `json:"gatewayId"  gorm:"column:gateway_id;not null;index"`
	CurrencyID uint `json:"currencyId" gorm:"column:currency_id;not null;index"`

	// Relationships
	Gateway *PaymentGateway `json:"gateway,omitempty" gorm:"foreignKey:GatewayID"`
}

func (PaymentGatewayCurrency) TableName() string {
	return "payment_gateway_currency"
}
