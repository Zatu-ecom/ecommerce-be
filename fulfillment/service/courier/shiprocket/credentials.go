// Package shiprocket implements the CourierPartner contract for Shiprocket.
// All Shiprocket specifics — credential shape, auth headers, status ids,
// API base URL, webhook shape — live ONLY in this folder. Base services
// resolve the adapter through CourierPartnerFactory and never import this
// package (except the fulfillment factory singleton that registers it).
package shiprocket

import (
	"encoding/json"
	"fmt"
	"strings"

	"ecommerce-be/common/helper"
	fulfillmenterrors "ecommerce-be/fulfillment/error"
	courier "ecommerce-be/fulfillment/service/courier"
)

// Credential keys as stored in courier_provider_config.credentials.
// api_email is a public identifier (plaintext); the secrets are
// AES-256-GCM encrypted at rest.
const (
	fieldAPIEmail      = "api_email"
	fieldAPIPassword   = "api_password"
	fieldWebhookSecret = "webhook_secret"
)

// shiprocketCredentials is the typed view of a decrypted credential map.
// It never leaves this package: orchestrators pass opaque maps.
type shiprocketCredentials struct {
	APIEmail      string `json:"api_email"`
	APIPassword   string `json:"api_password"`
	WebhookSecret string `json:"webhook_secret"`
}

// Validate checks credential presence. On full saves (partial=false) every
// required field must be present and non-empty; on partial updates only the
// provided keys are checked. The webhook secret is optional (unsigned pushes
// are located but never applied without it — see webhook.go).
func (a *Adapter) Validate(raw map[string]any, partial bool) error {
	if raw == nil {
		return fmt.Errorf("shiprocket credentials missing")
	}
	required := []string{fieldAPIEmail, fieldAPIPassword}
	for _, key := range required {
		if partial {
			if _, ok := raw[key]; !ok {
				continue
			}
		}
		if strings.TrimSpace(stringValue(raw[key])) == "" {
			return fmt.Errorf("shiprocket credentials must include %s", key)
		}
	}
	return nil
}

// Encrypt returns a storage-ready copy of a validated raw credential map:
// secrets encrypted, public identifiers plaintext. Fails closed when no
// encryption key is configured — secrets are never stored plaintext.
func (a *Adapter) Encrypt(raw map[string]any) (map[string]any, error) {
	if raw == nil {
		return nil, fmt.Errorf("shiprocket credentials missing")
	}
	key := courier.ResolveEncryptionKey()
	if key == "" {
		return nil, fulfillmenterrors.ErrorEncryptionKeyMissing
	}

	apiPassword, err := helper.Encrypt(stringValue(raw[fieldAPIPassword]), key)
	if err != nil {
		return nil, fmt.Errorf("encrypt api_password: %w", err)
	}
	webhookSecret := ""
	if v := strings.TrimSpace(stringValue(raw[fieldWebhookSecret])); v != "" {
		webhookSecret, err = helper.Encrypt(v, key)
		if err != nil {
			return nil, fmt.Errorf("encrypt webhook_secret: %w", err)
		}
	}

	return map[string]any{
		fieldAPIEmail:      stringValue(raw[fieldAPIEmail]),
		fieldAPIPassword:   apiPassword,
		fieldWebhookSecret: webhookSecret,
	}, nil
}

// Decrypt reverses Encrypt and returns the plaintext credential map.
// Unknown keys pass through untouched for forward compatibility.
func (a *Adapter) Decrypt(stored map[string]any) (map[string]any, error) {
	if stored == nil {
		return nil, fmt.Errorf("shiprocket credentials missing")
	}
	key := courier.ResolveEncryptionKey()
	if key == "" {
		return nil, fulfillmenterrors.ErrorEncryptionKeyMissing
	}

	creds, err := parseCredentials(stored)
	if err != nil {
		return nil, err
	}

	decryptedPassword, err := helper.Decrypt(creds.APIPassword, key)
	if err != nil {
		return nil, fmt.Errorf("decrypt api_password: %w", err)
	}
	decryptedWebhookSecret := ""
	if strings.TrimSpace(creds.WebhookSecret) != "" {
		decryptedWebhookSecret, err = helper.Decrypt(creds.WebhookSecret, key)
		if err != nil {
			return nil, fmt.Errorf("decrypt webhook_secret: %w", err)
		}
	}

	out := map[string]any{
		fieldAPIEmail:      creds.APIEmail,
		fieldAPIPassword:   decryptedPassword,
		fieldWebhookSecret: decryptedWebhookSecret,
	}
	for k, v := range stored {
		if _, known := out[k]; !known {
			out[k] = v
		}
	}
	return out, nil
}

// MaskHints returns the dashboard-safe view of stored credentials: the API
// email partially masked, and NEVER any secret material (not even encrypted
// blobs — their presence alone aids oracle attacks).
func (a *Adapter) MaskHints(stored map[string]any) map[string]any {
	hints := map[string]any{}
	if stored == nil {
		return hints
	}
	if email := stringValue(stored[fieldAPIEmail]); email != "" {
		hints[fieldAPIEmail] = maskEmail(email)
	}
	return hints
}

// MergePartial overlays an incoming (raw, possibly partial) credential update
// onto the existing STORED map and returns a RAW merged map ready for
// Validate → Encrypt. Empty or omitted sensitive keys keep the existing
// stored values; provided values replace them. The existing row is decrypted
// internally (fail closed without a key) so the result never double-encrypts.
func (a *Adapter) MergePartial(existingStored, incoming map[string]any) (map[string]any, error) {
	if len(incoming) == 0 {
		return nil, fmt.Errorf("shiprocket credentials update is empty")
	}
	decrypted, err := a.Decrypt(existingStored)
	if err != nil {
		return nil, err
	}

	merged := map[string]any{}
	for k, v := range decrypted {
		merged[k] = v
	}
	for _, key := range []string{fieldAPIEmail, fieldAPIPassword, fieldWebhookSecret} {
		if v, ok := incoming[key]; ok && strings.TrimSpace(stringValue(v)) != "" {
			merged[key] = stringValue(v)
		}
	}
	return merged, nil
}

// parseCredentials converts a credential map into the typed struct without
// decrypting. Both stored (encrypted secrets) and raw maps share the shape.
func parseCredentials(raw map[string]any) (*shiprocketCredentials, error) {
	if raw == nil {
		return nil, fmt.Errorf("shiprocket credentials missing")
	}
	b, err := json.Marshal(raw)
	if err != nil {
		return nil, fmt.Errorf("marshal shiprocket credentials: %w", err)
	}
	var creds shiprocketCredentials
	if err := json.Unmarshal(b, &creds); err != nil {
		return nil, fmt.Errorf("unmarshal shiprocket credentials: %w", err)
	}
	if strings.TrimSpace(creds.APIEmail) == "" ||
		strings.TrimSpace(creds.APIPassword) == "" {
		return nil, fmt.Errorf("shiprocket credentials must include api_email and api_password")
	}
	return &creds, nil
}

// maskEmail shows first 1 and domain (e.g. s***@shop.com).
func maskEmail(email string) string {
	parts := strings.Split(email, "@")
	if len(parts) != 2 || len(parts[0]) == 0 {
		return "***"
	}
	return parts[0][:1] + "***@" + parts[1]
}

// stringValue lives in shipment.go (shared JSON-scalar coercion).
