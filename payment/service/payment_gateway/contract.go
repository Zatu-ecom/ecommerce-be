package gateway

import (
	"context"
	"net/http"

	paymentModel "ecommerce-be/payment/model"
)

// PaymentGateway is the single provider contract every payment adapter implements.
// Base services (payment, webhook, gateway-dashboard, reconcile) speak ONLY to
// this interface, resolved by gateway code through PaymentGatewayFactory.
// Adding a new provider is purely additive: a new folder implementing this
// interface + seed rows + one factory registry line. Orchestrators must never
// switch on provider codes or import provider subpackages.
type PaymentGateway interface {
	Code() string

	// Credentials (seller dashboard + decrypt for outbound calls).
	Validate(raw map[string]any, partial bool) error
	Encrypt(raw map[string]any) (map[string]any, error)
	Decrypt(stored map[string]any) (map[string]any, error)
	MaskHints(stored map[string]any) map[string]any
	MergePartial(existingStored, incoming map[string]any) (map[string]any, error)

	InitiatePayment(ctx context.Context, in paymentModel.InitiatePaymentInput) (*paymentModel.InitiatePaymentOutput, error)
	Refund(ctx context.Context, in paymentModel.RefundInput) (*paymentModel.RefundOutput, error)
	TestConnection(ctx context.Context, creds map[string]any) error

	// PeekLocators scrapes locator hints from a raw webhook body WITHOUT
	// verifying authenticity. Hints are used only to find which transaction
	// (and therefore which seller secret) to verify with; they are discarded
	// if HMAC verification fails.
	PeekLocators(rawBody []byte) Locators

	// NormalizeWebhook verifies the request signature from headers + creds,
	// then maps the provider event to a frozen WebhookAction.
	NormalizeWebhook(rawBody []byte, headers http.Header, creds map[string]any) (*NormalizedWebhook, error)

	// FetchRemoteStatus polls the provider for the current state of a session
	// and/or payment (either id may be empty). Returns Action=ignore while the
	// session is still open; payment_completed / payment_failed / refund_*
	// when terminal. Used by the reconciliation cron.
	FetchRemoteStatus(ctx context.Context, sessionID, paymentID string, creds map[string]any) (*NormalizedWebhook, error)
}

// Locators are the untrusted transaction hints scraped from a raw webhook body.
type Locators struct {
	TransactionID string
	SessionID     string
	PaymentID     string
	RefundID      string
}

// WebhookAction is the frozen set of normalized outcomes base services may
// switch on. New providers map their own event names to these actions inside
// their adapter folder — the base switch never grows.
type WebhookAction string

const (
	WebhookActionIgnore           WebhookAction = "ignore"
	WebhookActionAuthorized       WebhookAction = "authorized"        // observation; status unchanged
	WebhookActionPaymentCompleted WebhookAction = "payment_completed" // pending → completed + confirm order
	WebhookActionPaymentFailed    WebhookAction = "payment_failed"    // pending → failed + fail order
	WebhookActionRefundPending    WebhookAction = "refund_pending"
	WebhookActionRefundCompleted  WebhookAction = "refund_completed"
	WebhookActionRefundFailed     WebhookAction = "refund_failed"
)

// NormalizedWebhook is the provider-agnostic result of verifying (webhook) or
// polling (cron) a provider event. Apply paths switch ONLY on Action.
type NormalizedWebhook struct {
	Action         WebhookAction
	EventID        string // idempotency key (provider unique; never empty)
	ProviderEvent  string // raw provider name, stored on webhook_log.event_type for audit only
	TransactionID  string // our TXN_* when present in provider notes
	SessionID      string // gateway_session_id
	PaymentID      string // gateway_payment_id
	RefundID       string // gateway_refund_id
	AmountCents    int64
	Currency       string
	FeeCents       *int64
	FailureCode    string
	FailureMessage string
	Payload        map[string]any // sanitized for storage
}

// RefundStatusFetcher is an OPTIONAL adapter capability for polling a single
// refund's remote state (used by the stuck-refund reconciler). It is separate
// from PaymentGateway so providers without a refund-fetch API still satisfy
// the core contract; the reconciler type-asserts and skips adapters that
// don't implement it.
type RefundStatusFetcher interface {
	FetchRefundStatus(ctx context.Context, refundID string, creds map[string]any) (*NormalizedWebhook, error)
}
