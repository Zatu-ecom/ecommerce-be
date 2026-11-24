package razorpay

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	paymenterrors "ecommerce-be/payment/error"
	gateway "ecommerce-be/payment/service/payment_gateway"
)

// Razorpay webhook headers and event names. These MUST NOT leave this folder:
// base services switch only on gateway.WebhookAction.
const (
	signatureHeader = "X-Razorpay-Signature"
	eventIDHeader   = "X-Razorpay-Event-Id"

	eventPaymentAuthorized = "payment.authorized"
	eventPaymentCaptured   = "payment.captured"
	eventPaymentFailed     = "payment.failed"
	eventRefundCreated     = "refund.created"
	eventRefundProcessed   = "refund.processed"
	eventRefundFailed      = "refund.failed"
)

// eventToAction maps provider event names to frozen webhook actions.
// Unknown events map to ignore (recorded, no state change).
func eventToAction(event string) gateway.WebhookAction {
	switch event {
	case eventPaymentAuthorized:
		return gateway.WebhookActionAuthorized
	case eventPaymentCaptured:
		return gateway.WebhookActionPaymentCompleted
	case eventPaymentFailed:
		return gateway.WebhookActionPaymentFailed
	case eventRefundCreated:
		return gateway.WebhookActionRefundPending
	case eventRefundProcessed:
		return gateway.WebhookActionRefundCompleted
	case eventRefundFailed:
		return gateway.WebhookActionRefundFailed
	default:
		return gateway.WebhookActionIgnore
	}
}

// PeekLocators scrapes transaction hints from a raw webhook body WITHOUT
// verifying authenticity. The hints only select which transaction (and hence
// which seller secret) to verify with; they are discarded if HMAC fails.
// Garbage bodies yield empty locators, never an error.
func (a *Adapter) PeekLocators(rawBody []byte) gateway.Locators {
	event, entity, _ := splitEnvelope(rawBody)
	if event == "" && len(entity) == 0 {
		return gateway.Locators{}
	}
	loc := gateway.Locators{}
	if id, ok := entity["id"].(string); ok {
		if strings.HasPrefix(event, "refund.") {
			loc.RefundID = id
		} else {
			loc.PaymentID = id
		}
	}
	if orderID, ok := entity["order_id"].(string); ok {
		loc.SessionID = orderID
	}
	if paymentID, ok := entity["payment_id"].(string); ok {
		loc.PaymentID = paymentID
	}
	if notes, ok := entity["notes"].(map[string]any); ok {
		if txID, ok := notes["transaction_id"].(string); ok {
			loc.TransactionID = txID
		}
	}
	return loc
}

// NormalizeWebhook verifies X-Razorpay-Signature = hex(HMAC-SHA256(rawBody,
// webhook_secret)) using constant-time comparison, then maps the provider
// event to a NormalizedWebhook. The raw body is used verbatim: re-serialized
// JSON would break the signature. Verification failure returns an error and
// the caller must change no state.
func (a *Adapter) NormalizeWebhook(
	rawBody []byte,
	headers http.Header,
	creds map[string]any,
) (*gateway.NormalizedWebhook, error) {
	parsed, err := parseCredentials(creds)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(parsed.WebhookSecret) == "" {
		return nil, fmt.Errorf("razorpay: webhook_secret missing, cannot verify")
	}

	signature := ""
	if headers != nil {
		signature = strings.TrimSpace(headers.Get(signatureHeader))
	}
	// Typed AppError: the HTTP layer maps INVALID_WEBHOOK_SIGNATURE to 401
	// while every other apply failure stays 200. No state changes here.
	if signature == "" || !verifySignature(rawBody, signature, parsed.WebhookSecret) {
		return nil, paymenterrors.ErrorInvalidWebhookSignature
	}

	event, entity, payload := splitEnvelope(rawBody)
	if event == "" {
		return nil, fmt.Errorf("razorpay: webhook missing event type")
	}

	n := &gateway.NormalizedWebhook{
		Action:        eventToAction(event),
		ProviderEvent: event,
		Payload:       payload,
	}

	// Idempotency key: canonical provider event id first, composite fallback
	// {event}:{entity.id} so authorized vs captured never collide.
	if headers != nil {
		n.EventID = strings.TrimSpace(headers.Get(eventIDHeader))
	}
	if n.EventID == "" {
		if id, ok := payload["id"].(string); ok && id != "" {
			n.EventID = id
		} else if id, ok := entity["id"].(string); ok && id != "" {
			n.EventID = event + ":" + id
		} else {
			n.EventID = event + ":unknown"
		}
	}

	fillLocators(n, event, entity)
	return n, nil
}

// verifySignature compares hex(HMAC-SHA256(body, secret)) in constant time.
func verifySignature(rawBody []byte, signature, secret string) bool {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(rawBody)
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(signature))
}

// fillLocators extracts transaction/session/payment/refund ids, money fields
// and failure details from the webhook entity into the normalized result.
func fillLocators(n *gateway.NormalizedWebhook, event string, entity map[string]any) {
	if id, ok := entity["id"].(string); ok {
		if strings.HasPrefix(event, "refund.") {
			n.RefundID = id
		} else {
			n.PaymentID = id
		}
	}
	if orderID, ok := entity["order_id"].(string); ok {
		n.SessionID = orderID
	}
	if paymentID, ok := entity["payment_id"].(string); ok {
		n.PaymentID = paymentID
	}
	if amount, ok := entity["amount"].(float64); ok {
		n.AmountCents = int64(amount)
	}
	if currency, ok := entity["currency"].(string); ok {
		n.Currency = strings.ToUpper(currency)
	}
	if fee, ok := entity["fee"].(float64); ok {
		feeCents := int64(fee)
		n.FeeCents = &feeCents
	}
	if failureCode, ok := entity["error_code"].(string); ok {
		n.FailureCode = failureCode
	}
	if failureMessage, ok := entity["error_description"].(string); ok {
		n.FailureMessage = failureMessage
	}
	if notes, ok := entity["notes"].(map[string]any); ok {
		if txID, ok := notes["transaction_id"].(string); ok {
			n.TransactionID = txID
		}
	}
	// Capture snapshot for payment_method_details (filled on capture when present).
	if method, ok := entity["method"].(string); ok && method != "" {
		n.Payload["method"] = method
	}
}

// splitEnvelope parses a Razorpay webhook body into (event, entity, payload).
// payment.* events carry payload.payment.entity; refund.* events carry
// payload.refund.entity (with one nesting level unwrapped). Garbage input
// yields empty results, never an error — callers decide how to react.
func splitEnvelope(rawBody []byte) (event string, entity, payload map[string]any) {
	payload = map[string]any{}
	entity = map[string]any{}
	var envelope struct {
		Event   string         `json:"event"`
		Payload map[string]any `json:"payload"`
	}
	if err := json.Unmarshal(rawBody, &envelope); err != nil {
		return "", entity, payload
	}
	if envelope.Payload != nil {
		payload = envelope.Payload
	}
	entityKey := "payment"
	if strings.HasPrefix(envelope.Event, "refund.") {
		entityKey = "refund"
	}
	if container, ok := payload[entityKey].(map[string]any); ok {
		entity = container
	}
	if container, ok := payload[entityKey+"_entity"].(map[string]any); ok {
		entity = container
	}
	// Razorpay nests the entity under payload.<key>.entity — unwrap one level.
	if nested, ok := entity["entity"].(map[string]any); ok {
		entity = nested
	}
	return envelope.Event, entity, payload
}
