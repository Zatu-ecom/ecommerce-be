package model

import (
	"time"

	"ecommerce-be/common/filegateway"
	"ecommerce-be/payment/entity"
)

// PaymentTransactionResponse is the API view of a payment transaction.
// Environment, RefundableAmountCents and Events are detail-only (GET by id);
// list responses omit them (hot path stays small, no N+1).
type PaymentTransactionResponse struct {
	ID                    uint                              `json:"id"`
	TransactionID         string                            `json:"transactionId"`
	ReferenceType         entity.ReferenceType              `json:"referenceType"`
	ReferenceID           uint                              `json:"referenceId"`
	GatewayID             *uint                             `json:"gatewayId,omitempty"`
	GatewaySessionID      string                            `json:"gatewaySessionId,omitempty"`
	GatewayPaymentID      string                            `json:"gatewayPaymentId,omitempty"`
	Currency              string                            `json:"currency"`
	AmountCents           int64                             `json:"amountCents"`
	GatewayFeeCents       *int64                            `json:"gatewayFeeCents,omitempty"`
	Status                entity.TransactionStatus          `json:"status"`
	FailureCode           string                            `json:"failureCode,omitempty"`
	FailureMessage        string                            `json:"failureMessage,omitempty"`
	PaymentMethodType     entity.PaymentMethodType          `json:"paymentMethodType,omitempty"`
	Environment           string                            `json:"environment,omitempty"`
	RefundableAmountCents *int64                            `json:"refundableAmountCents,omitempty"`
	Events                []PaymentTransactionEventResponse `json:"events,omitempty"`
	CompletedAt           *time.Time                        `json:"completedAt,omitempty"`
	CreatedAt             time.Time                         `json:"createdAt"`
}

// PaymentTransactionEventResponse is one ledger entry on the detail view.
type PaymentTransactionEventResponse struct {
	EventType      string    `json:"eventType"`
	FromStatus     string    `json:"fromStatus,omitempty"`
	ToStatus       string    `json:"toStatus"`
	GatewayEventID string    `json:"gatewayEventId,omitempty"`
	FailureCode    string    `json:"failureCode,omitempty"`
	FailureMessage string    `json:"failureMessage,omitempty"`
	Source         string    `json:"source"`
	CreatedAt      time.Time `json:"createdAt"`
}

// WebhookLogResponse is one seller-scoped webhook delivery on the ledger view.
type WebhookLogResponse struct {
	ID            uint                 `json:"id"`
	EventType     string               `json:"eventType"`
	EventID       string               `json:"eventId"`
	Status        entity.WebhookStatus `json:"status"`
	ErrorMessage  string               `json:"errorMessage,omitempty"`
	TransactionID string               `json:"transactionId,omitempty"`
	RefundID      *uint                `json:"refundId,omitempty"`
	CreatedAt     time.Time            `json:"createdAt"`
}

// InitiatePaymentResponse is the customer-facing checkout payload.
// Checkout is provider-agnostic (gatewayCode + adapter fields); there is
// intentionally no top-level keyId — FE reads checkout.fields.
type InitiatePaymentResponse struct {
	TransactionID    string          `json:"transactionId"`
	GatewaySessionID string          `json:"gatewaySessionId"`
	AmountCents      int64           `json:"amountCents"`
	Currency         string          `json:"currency"`
	Status           string          `json:"status"`
	Checkout         CheckoutPayload `json:"checkout"`
}

// RefundResponse is the API view of a refund.
type RefundResponse struct {
	RefundID        string              `json:"refundId"`
	TransactionID   uint                `json:"transactionId"`
	GatewayRefundID string              `json:"gatewayRefundId,omitempty"`
	AmountCents     int64               `json:"amountCents"`
	Currency        string              `json:"currency"`
	Status          entity.RefundStatus `json:"status"`
	Reason          entity.RefundReason `json:"reason,omitempty"`
	Notes           string              `json:"notes,omitempty"`
	CreatedAt       time.Time           `json:"createdAt"`
}

// GatewayFieldResponse describes a config field a seller must supply for a gateway.
type GatewayFieldResponse struct {
	FieldName       string      `json:"fieldName"`
	DisplayName     string      `json:"displayName"`
	FieldType       string      `json:"fieldType"`
	Description     string      `json:"description,omitempty"`
	Placeholder     string      `json:"placeholder,omitempty"`
	IsRequired      bool        `json:"isRequired"`
	IsSensitive     bool        `json:"isSensitive"`
	DisplayOrder    int         `json:"displayOrder"`
	ValidationRules interface{} `json:"validationRules,omitempty"`
}

// GatewayCountryResponse is one country a gateway serves.
type GatewayCountryResponse struct {
	ID   uint   `json:"id"`
	Code string `json:"code"`
	Name string `json:"name"`
}

// GatewayCurrencyResponse is one currency a gateway settles.
type GatewayCurrencyResponse struct {
	ID            uint   `json:"id"`
	Code          string `json:"code"`
	Symbol        string `json:"symbol"`
	DecimalDigits int    `json:"decimalDigits"`
}

// GatewayEnvConfigResponse is one environment's config state on the dashboard.
// ConfigHints carries adapter-defined masked hints (e.g. keyId, accountId);
// secrets and encrypted blobs are never included.
type GatewayEnvConfigResponse struct {
	Configured  bool           `json:"configured"`
	IsActive    bool           `json:"isActive"`
	Priority    int            `json:"priority"`
	ConfigHints map[string]any `json:"configHints"`
}

// GatewaySummaryResponse is the seller dashboard row for one provider.
type GatewaySummaryResponse struct {
	Code                    string                         `json:"code"`
	Name                    string                         `json:"name"`
	Logo                    *filegateway.FileAssetResponse `json:"logo,omitempty"`
	SupportedCountries      []GatewayCountryResponse       `json:"supportedCountries"`
	SupportedCurrencies     []GatewayCurrencyResponse      `json:"supportedCurrencies"`
	SupportedPaymentMethods []string                       `json:"supportedPaymentMethods"`
	Configured              bool                           `json:"configured"`
	ConfiguredSandbox       bool                           `json:"configuredSandbox"`
	ConfiguredProduction    bool                           `json:"configuredProduction"`
	PaymentsEnvironment     string                         `json:"paymentsEnvironment"`
	WebhookURL              string                         `json:"webhookUrl"`
}

// GatewayDetailResponse is the full detail for one provider, including fields.
type GatewayDetailResponse struct {
	GatewaySummaryResponse
	Description string                              `json:"description,omitempty"`
	Fields      []GatewayFieldResponse              `json:"fields"`
	Configs     map[string]GatewayEnvConfigResponse `json:"configs"`
}

// GatewayTestResponse is the result of probing credentials without persisting.
type GatewayTestResponse struct {
	OK          bool   `json:"ok"`
	Environment string `json:"environment"`
}
