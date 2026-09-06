// Package razorpay implements the PaymentGateway contract for Razorpay.
//
// All Razorpay specifics — credential shape, HMAC headers, dotted event names,
// API base URL, HTTP auth — live ONLY in this folder. Base services resolve the
// adapter through PaymentGatewayFactory and never import this package (except
// the payment factory singleton that registers it).
package razorpay

import (
	"encoding/json"
	"fmt"
	"strings"

	"ecommerce-be/common/helper"
	paymenterrors "ecommerce-be/payment/error"
	gateway "ecommerce-be/payment/service/payment_gateway"
)

// Credential keys as stored in payment_gateway_config.credentials.
// key_id and account_id are public identifiers (plaintext); the secrets are
// AES-256-GCM encrypted at rest.
const (
	fieldKeyID         = "key_id"
	fieldKeySecret     = "key_secret"
	fieldWebhookSecret = "webhook_secret"
	fieldAccountID     = "account_id"
)

// razorpayCredentials is the typed view of a decrypted credential map.
// It never leaves this package: orchestrators pass opaque maps.
type razorpayCredentials struct {
	KeyID         string `json:"key_id"`
	KeySecret     string `json:"key_secret"`
	WebhookSecret string `json:"webhook_secret"`
	AccountID     string `json:"account_id,omitempty"`
}

// Validate checks credential presence. On full saves (partial=false) every
// required field must be present and non-empty; on partial updates only the
// provided keys are checked. Format rules (key patterns, min lengths) live in
// the payment_gateway_field seed rows for the dashboard UI; the backend stays
// lenient here to avoid false rejections of provider-issued values.
func (a *Adapter) Validate(raw map[string]any, partial bool) error {
	if raw == nil {
		return fmt.Errorf("razorpay credentials missing")
	}
	required := []string{fieldKeyID, fieldKeySecret, fieldWebhookSecret}
	for _, key := range required {
		if partial {
			if _, ok := raw[key]; !ok {
				continue
			}
		}
		if strings.TrimSpace(stringValue(raw[key])) == "" {
			return fmt.Errorf("razorpay credentials must include %s", key)
		}
	}
	return nil
}

// Encrypt returns a storage-ready copy of a validated raw credential map:
// secrets encrypted, public identifiers plaintext. Fails closed when no
// encryption key is configured — secrets are never stored plaintext.
func (a *Adapter) Encrypt(raw map[string]any) (map[string]any, error) {
	if raw == nil {
		return nil, fmt.Errorf("razorpay credentials missing")
	}
	key := gateway.ResolveEncryptionKey()
	if key == "" {
		return nil, paymenterrors.ErrorEncryptionKeyMissing
	}

	keySecret, err := helper.Encrypt(stringValue(raw[fieldKeySecret]), key)
	if err != nil {
		return nil, fmt.Errorf("encrypt key_secret: %w", err)
	}
	webhookSecret, err := helper.Encrypt(stringValue(raw[fieldWebhookSecret]), key)
	if err != nil {
		return nil, fmt.Errorf("encrypt webhook_secret: %w", err)
	}

	return map[string]any{
		fieldKeyID:         stringValue(raw[fieldKeyID]),
		fieldKeySecret:     keySecret,
		fieldWebhookSecret: webhookSecret,
		fieldAccountID:     stringValue(raw[fieldAccountID]),
	}, nil
}

// Decrypt reverses Encrypt and returns the plaintext credential map.
// Unknown keys pass through untouched for forward compatibility.
func (a *Adapter) Decrypt(stored map[string]any) (map[string]any, error) {
	if stored == nil {
		return nil, fmt.Errorf("razorpay credentials missing")
	}
	key := gateway.ResolveEncryptionKey()
	if key == "" {
		return nil, paymenterrors.ErrorEncryptionKeyMissing
	}

	creds, err := parseCredentials(stored)
	if err != nil {
		return nil, err
	}

	decryptedKeySecret, err := helper.Decrypt(creds.KeySecret, key)
	if err != nil {
		return nil, fmt.Errorf("decrypt key_secret: %w", err)
	}
	decryptedWebhookSecret, err := helper.Decrypt(creds.WebhookSecret, key)
	if err != nil {
		return nil, fmt.Errorf("decrypt webhook_secret: %w", err)
	}

	out := map[string]any{
		fieldKeyID:         creds.KeyID,
		fieldKeySecret:     decryptedKeySecret,
		fieldWebhookSecret: decryptedWebhookSecret,
		fieldAccountID:     creds.AccountID,
	}
	for k, v := range stored {
		if _, known := out[k]; !known {
			out[k] = v
		}
	}
	return out, nil
}

// MaskHints returns the dashboard-safe view of stored credentials: the public
// key id partially masked, account id in full, and NEVER any secret material
// (not even encrypted blobs — their presence alone aids oracle attacks).
func (a *Adapter) MaskHints(stored map[string]any) map[string]any {
	hints := map[string]any{}
	if stored == nil {
		return hints
	}
	if keyID := stringValue(stored[fieldKeyID]); keyID != "" {
		hints[fieldKeyID] = maskKeyID(keyID)
	}
	if accountID := stringValue(stored[fieldAccountID]); accountID != "" {
		hints[fieldAccountID] = accountID
	}
	return hints
}

// MergePartial overlays an incoming (raw, possibly partial) credential update
// onto the existing STORED map and returns a RAW merged map ready for
// Validate → Encrypt. Empty or omitted sensitive keys keep the existing stored
// values; provided values replace them. The existing row is decrypted
// internally (fail closed without a key) so the result never double-encrypts.
func (a *Adapter) MergePartial(existingStored, incoming map[string]any) (map[string]any, error) {
	if len(incoming) == 0 {
		return nil, fmt.Errorf("razorpay credentials update is empty")
	}
	decrypted, err := a.Decrypt(existingStored)
	if err != nil {
		return nil, err
	}

	merged := map[string]any{}
	for k, v := range decrypted {
		merged[k] = v
	}
	for _, key := range []string{fieldKeyID, fieldKeySecret, fieldWebhookSecret, fieldAccountID} {
		if v, ok := incoming[key]; ok && strings.TrimSpace(stringValue(v)) != "" {
			merged[key] = stringValue(v)
		}
	}
	return merged, nil
}

// parseCredentials converts a credential map into the typed struct without
// decrypting. Both stored (encrypted secrets) and raw maps share the shape.
func parseCredentials(raw map[string]any) (*razorpayCredentials, error) {
	if raw == nil {
		return nil, fmt.Errorf("razorpay credentials missing")
	}
	b, err := json.Marshal(raw)
	if err != nil {
		return nil, fmt.Errorf("marshal razorpay credentials: %w", err)
	}
	var creds razorpayCredentials
	if err := json.Unmarshal(b, &creds); err != nil {
		return nil, fmt.Errorf("unmarshal razorpay credentials: %w", err)
	}
	if strings.TrimSpace(creds.KeyID) == "" ||
		strings.TrimSpace(creds.KeySecret) == "" ||
		strings.TrimSpace(creds.WebhookSecret) == "" {
		return nil, fmt.Errorf("razorpay credentials must include key_id, key_secret and webhook_secret")
	}
	return &creds, nil
}

// maskKeyID shows the first 8 and last 4 characters (e.g. rzp_test_…aaaa).
// Short values are returned as-is: the key id is public (shipped to checkout)
// and masking is cosmetic only.
func maskKeyID(keyID string) string {
	if len(keyID) > 12 {
		return keyID[:8] + "…" + keyID[len(keyID)-4:]
	}
	return keyID
}

func stringValue(v any) string {
	s, _ := v.(string)
	return s
}
