package model

import (
	commonModel "ecommerce-be/common/model"
)

// ─── Courier dashboard DTOs (api-contracts.md §§1–3) ─────────────────────────
// Pointer fields distinguish "not provided" from zero values on writes
// (CODING_STANDARDS). Clients never send money amounts; Money appears only
// in rate responses, server-rendered.

// ConfigureCourierRequest saves or updates a seller's courier credentials.
// Environment is required: credentials live per environment, never mixed.
type ConfigureCourierRequest struct {
	Environment    string         `json:"environment" binding:"required,oneof=sandbox production"`
	Credentials    map[string]any `json:"credentials" binding:"required"`
	AutoBook       *bool          `json:"autoBook"`
	RatePreference *string        `json:"ratePreference" binding:"omitempty,oneof=cheapest fastest"`
	IsActive       *bool          `json:"isActive"`
}

// TestCourierRequest probes credentials without persisting anything.
// Credentials are optional: when omitted, the saved row is tested.
type TestCourierRequest struct {
	Environment string         `json:"environment" binding:"required,oneof=sandbox production"`
	Credentials map[string]any `json:"credentials"`
}

// CourierSummaryResponse is one catalog row with the seller's configured state.
type CourierSummaryResponse struct {
	Code             string `json:"code"`
	Name             string `json:"name"`
	Kind             string `json:"kind"`
	SupportsPickup   bool   `json:"supportsPickup"`
	SupportsNDR      bool   `json:"supportsNdr"`
	SupportsReturn   bool   `json:"supportsReturn"`
	SupportsCOD      bool   `json:"supportsCod"`
	WebhookSupported bool   `json:"webhookSupported"`
	Configured       bool   `json:"configured"`
	IsActive         bool   `json:"isActive"`
	WebhookURL       string `json:"webhookUrl"`
}

// CourierFieldResponse is one credential-form field for the dashboard.
type CourierFieldResponse struct {
	FieldName    string `json:"fieldName"`
	DisplayName  string `json:"displayName"`
	FieldType    string `json:"fieldType"`
	IsRequired   bool   `json:"isRequired"`
	IsSensitive  bool   `json:"isSensitive"`
	DisplayOrder int    `json:"displayOrder"`
}

// CourierConfigResponse is the saved-row view: masked hints only, never
// secrets. Config is null when nothing is saved.
type CourierConfigResponse struct {
	Configured     bool           `json:"configured"`
	IsActive       bool           `json:"isActive"`
	AutoBook       bool           `json:"autoBook"`
	RatePreference *string        `json:"ratePreference"`
	Environment    string         `json:"environment"`
	ConfigHints    map[string]any `json:"configHints"`
}

// CourierDetailResponse is the full detail for one provider.
type CourierDetailResponse struct {
	Code   string                 `json:"code"`
	Name   string                 `json:"name"`
	Fields []CourierFieldResponse `json:"fields"`
	Config *CourierConfigResponse `json:"config"`
}

// TestConnectionResponse is the probe result. Invalid keys surface as a
// 400 error, not this payload.
type TestConnectionResponse struct {
	OK          bool   `json:"ok"`
	Environment string `json:"environment"`
}

// RateOptionResponse is one priced courier service for the dashboard.
type RateOptionResponse struct {
	CourierName   string            `json:"courierName"`
	ServiceCode   string            `json:"serviceCode"`
	Rate          commonModel.Money `json:"rate"`
	ETD           *string           `json:"etd"`
	PickupCapable bool              `json:"pickupCapable"`
}

// ToRateOptionResponse maps a provider-agnostic option with display money.
func ToRateOptionResponse(option RateOption, currencyCode string) RateOptionResponse {
	var etd *string
	if option.ETD != nil {
		formatted := option.ETD.UTC().Format("2006-01-02T15:04:05Z")
		etd = &formatted
	}
	return RateOptionResponse{
		CourierName:   option.CourierName,
		ServiceCode:   option.ServiceCode,
		Rate:          Money(option.RateCents, currencyCode),
		ETD:           etd,
		PickupCapable: option.PickupCapable,
	}
}

// BookShipmentRequest books one draft. providerCode selects the courier;
// serviceCode and pickupAt are optional (pickup joins the booking when set).
type BookShipmentRequest struct {
	ProviderCode string  `json:"providerCode" binding:"required"`
	ServiceCode  string  `json:"serviceCode"`
	PickupAt     *string `json:"pickupAt"`
}

// PickupRequest schedules pickup separately from booking.
type PickupRequest struct {
	PickupAt *string `json:"pickupAt"`
}

// RatesRequest shops live rates. providerCode is required only when the
// seller has more than one active courier; weight arrives from the client
// (planner drafts may not carry it yet).
type RatesRequest struct {
	ProviderCode     string   `json:"providerCode"`
	OrderID          uint     `json:"orderId" binding:"required"`
	PickupLocationID uint     `json:"pickupLocationId" binding:"required"`
	WeightGrams      int      `json:"weightGrams" binding:"required,gt=0"`
	LengthCm         *float64 `json:"lengthCm" binding:"omitempty,gt=0"`
	BreadthCm        *float64 `json:"breadthCm" binding:"omitempty,gt=0"`
	HeightCm         *float64 `json:"heightCm" binding:"omitempty,gt=0"`
}
