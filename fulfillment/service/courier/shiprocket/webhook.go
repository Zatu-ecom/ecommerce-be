package shiprocket

import (
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	courier "ecommerce-be/fulfillment/service/courier"
)

// PeekLocators scrapes the AWB and provider order id from a raw webhook
// body WITHOUT verifying authenticity. The hints locate our shipment (and
// therefore the seller secret to verify with); they are discarded when
// verification fails.
func (a *Adapter) PeekLocators(rawBody []byte) courier.TrackLocators {
	var body map[string]any
	if err := json.Unmarshal(rawBody, &body); err != nil {
		return courier.TrackLocators{}
	}
	return courier.TrackLocators{
		AWB:             stringValue(body["awb"]),
		ProviderOrderID: firstNonEmpty(stringValue(body["order_id"]), stringValue(body["sr_order_id"])),
	}
}

// NormalizeWebhook verifies the push signature, then maps the provider
// status to a frozen action. Shiprocket signs pushes with the account's
// webhook secret, delivered as the X-Api-Key header; comparison is
// constant-time on the raw secret. No secret configured (or mismatch) →
// error, and the caller persists nothing.
func (a *Adapter) NormalizeWebhook(
	rawBody []byte,
	headers http.Header,
	creds map[string]any,
) (*courier.NormalizedShipmentEvent, error) {
	parsed, err := parseCredentials(creds)
	if err != nil {
		return nil, fmt.Errorf("[shiprocket] %w", err)
	}
	if strings.TrimSpace(parsed.WebhookSecret) == "" {
		return nil, fmt.Errorf("shiprocket: webhook secret not configured, cannot verify push")
	}
	signature := headers.Get("X-Api-Key")
	if subtle.ConstantTimeCompare([]byte(signature), []byte(parsed.WebhookSecret)) != 1 {
		return nil, fmt.Errorf("shiprocket: webhook signature mismatch")
	}

	var body map[string]any
	if err := json.Unmarshal(rawBody, &body); err != nil {
		return nil, fmt.Errorf("shiprocket: parse webhook body: %w", err)
	}
	awb := strings.TrimSpace(stringValue(body["awb"]))
	if awb == "" {
		return nil, fmt.Errorf("shiprocket: webhook without awb")
	}
	raw := firstNonEmpty(
		stringValue(body["current_status"]),
		stringValue(body["shipment_status"]),
	)
	action := mapShipmentStatus(raw)
	if strings.TrimSpace(raw) == "" {
		action = courier.ShipmentActionIgnore
	}

	return &courier.NormalizedShipmentEvent{
		Action:             action,
		EventID:            webhookEventID(body, awb, raw),
		ProviderEvent:      raw,
		AWB:                awb,
		ProviderOrderID:    firstNonEmpty(stringValue(body["order_id"]), stringValue(body["sr_order_id"])),
		ProviderShipmentID: stringValue(body["sr_shipment_id"]),
		CourierName:        stringValue(body["courier_name"]),
		Payload:            sanitizeTrackPayload(body),
	}, nil
}

// webhookEventID builds the idempotency key: same scan redelivered dedupes,
// a new scan (new timestamp) applies. Falls back deterministically when the
// provider omits timestamps.
func webhookEventID(body map[string]any, awb, status string) string {
	date := firstNonEmpty(
		stringValue(body["date"]),
		stringValue(body["current_timestamp"]),
		stringValue(body["timestamp"]),
	)
	if strings.TrimSpace(date) == "" {
		date = "undated"
	}
	key := awb + "|" + strings.ToUpper(strings.TrimSpace(status)) + "|" + strings.TrimSpace(date)
	return strings.ReplaceAll(key, " ", "_")
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
