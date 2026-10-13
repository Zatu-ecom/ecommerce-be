package gateway

import (
	"os"

	"ecommerce-be/common/config"
)

// ResolveEncryptionKey returns the AES-256 key used for credential encryption,
// mirroring the file module's pattern: config singleton first, ENCRYPTION_KEY
// env fallback. Adapters call it internally from Encrypt/Decrypt; base services
// must never encrypt or decrypt provider secrets themselves.
func ResolveEncryptionKey() string {
	if cfg := config.Get(); cfg != nil && cfg.App.EncryptionKey != "" {
		return cfg.App.EncryptionKey
	}
	return os.Getenv("ENCRYPTION_KEY")
}
