package model

import (
	"time"

	"ecommerce-be/common/filegateway"
	"ecommerce-be/payment/entity"
)

// PaymentTransactionResponse is the API view of a payment transaction.
type PaymentTransactionResponse struct {
	ID                uint                     `json:"id"`
	TransactionID     string                   `json:"transactionId"`
	ReferenceType     entity.ReferenceType     `json:"referenceType"`
	ReferenceID       uint                     `json:"referenceId"`
	GatewayID         *uint                    `json:"gatewayId,omitempty"`
	GatewaySessionID  string                   `json:"gatewaySessionId,omitempty"`
	GatewayPaymentID  string                   `json:"gatewayPaymentId,omitempty"`
	Currency          string                   `json:"currency"`
	AmountCents       int64                    `json:"amountCents"`
	GatewayFeeCents   *int64                   `json:"gatewayFeeCents,omitempty"`
	Status            entity.TransactionStatus `json:"status"`
	FailureCode       string                   `json:"failureCode,omitempty"`
	FailureMessage    string                   `json:"failureMessage,omitempty"`
	PaymentMethodType entity.PaymentMethodType `json:"paymentMethodType,omitempty"`
	CompletedAt       *time.Time               `json:"completedAt,omitempty"`
	CreatedAt         time.Time                `json:"createdAt"`
}

// InitiatePaymentResponse is the customer-facing checkout payload.
type InitiatePaymentResponse struct {
	TransactionID    string `json:"transactionId"`
	GatewaySessionID string `json:"gatewaySessionId"`
	KeyID            string `json:"keyId"`
	AmountCents      int64  `json:"amountCents"`
	Currency         string `json:"currency"`
	Status           string `json:"status"`
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

// GatewaySummaryResponse is the seller dashboard row for one provider.
type GatewaySummaryResponse struct {
	Code                    string                         `json:"code"`
	Name                    string                         `json:"name"`
	Logo                    *filegateway.FileAssetResponse `json:"logo,omitempty"`
	SupportedCountries      []string                       `json:"supportedCountries"`
	SupportedCurrencies     []string                       `json:"supportedCurrencies"`
	SupportedPaymentMethods []string                       `json:"supportedPaymentMethods"`
	Configured              bool                           `json:"configured"`
	Environment             string                         `json:"environment,omitempty"`
}

// GatewayDetailResponse is the full detail for one provider, including fields.
type GatewayDetailResponse struct {
	GatewaySummaryResponse
	Description string                 `json:"description,omitempty"`
	Fields      []GatewayFieldResponse `json:"fields"`
}
