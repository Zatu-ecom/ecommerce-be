package entity

// PaymentGatewayCountry links a gateway to a country it serves.
// Membership is checked with EXISTS queries against this join table.
type PaymentGatewayCountry struct {
	GatewayID uint `json:"gatewayId" gorm:"column:gateway_id;not null;index"`
	CountryID uint `json:"countryId" gorm:"column:country_id;not null;index"`

	// Relationships
	Gateway *PaymentGateway `json:"gateway,omitempty" gorm:"foreignKey:GatewayID"`
}

func (PaymentGatewayCountry) TableName() string {
	return "payment_gateway_country"
}
