package gateway

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"ecommerce-be/common/config"
	"ecommerce-be/common/helper"
	paymentModel "ecommerce-be/payment/model"
)

// ResolveEncryptionKey returns the AES-256 key used for credential encryption,
// mirroring the file module's pattern: config singleton first, ENCRYPTION_KEY env fallback.
func ResolveEncryptionKey() string {
	if cfg := config.Get(); cfg != nil && cfg.App.EncryptionKey != "" {
		return cfg.App.EncryptionKey
	}
	return os.Getenv("ENCRYPTION_KEY")
}

// ParseRazorpayCredentials converts a raw credentials map into the typed struct.
func ParseRazorpayCredentials(raw map[string]any) (*paymentModel.RazorpayCredentials, error) {
	if raw == nil {
		return nil, fmt.Errorf("razorpay credentials missing")
	}

	b, err := json.Marshal(raw)
	if err != nil {
		return nil, fmt.Errorf("marshal razorpay credentials: %w", err)
	}

	var creds paymentModel.RazorpayCredentials
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

// EncryptSensitive returns a copy of the credentials map with secret fields encrypted.
// key_id is a public identifier and stays plaintext.
func EncryptSensitive(creds *paymentModel.RazorpayCredentials) (map[string]any, error) {
	if creds == nil {
		return nil, fmt.Errorf("razorpay credentials missing")
	}
	key := ResolveEncryptionKey()

	keySecret, err := helper.Encrypt(creds.KeySecret, key)
	if err != nil {
		return nil, fmt.Errorf("encrypt key_secret: %w", err)
	}
	webhookSecret, err := helper.Encrypt(creds.WebhookSecret, key)
	if err != nil {
		return nil, fmt.Errorf("encrypt webhook_secret: %w", err)
	}

	return map[string]any{
		"key_id":         creds.KeyID,
		"key_secret":     keySecret,
		"webhook_secret": webhookSecret,
		"account_id":     creds.AccountID,
	}, nil
}

// DecryptSensitive returns the typed credentials with secret fields decrypted.
func DecryptSensitive(raw map[string]any) (*paymentModel.RazorpayCredentials, error) {
	if raw == nil {
		return nil, fmt.Errorf("razorpay credentials missing")
	}

	creds, err := ParseRazorpayCredentials(raw)
	if err != nil {
		return nil, err
	}

	key := ResolveEncryptionKey()
	decryptedKeySecret, err := helper.Decrypt(creds.KeySecret, key)
	if err != nil {
		return nil, fmt.Errorf("decrypt key_secret: %w", err)
	}
	decryptedWebhookSecret, err := helper.Decrypt(creds.WebhookSecret, key)
	if err != nil {
		return nil, fmt.Errorf("decrypt webhook_secret: %w", err)
	}

	return &paymentModel.RazorpayCredentials{
		KeyID:         creds.KeyID,
		KeySecret:     decryptedKeySecret,
		WebhookSecret: decryptedWebhookSecret,
		AccountID:     creds.AccountID,
	}, nil
}
