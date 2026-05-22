package blobAdapter

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"ecommerce-be/common/config"
	fileError "ecommerce-be/file/error"

	"github.com/go-playground/validator/v10"
)

var validate = validator.New()

// ─── Generic Configuration Parser ─────────────────────────────────────────────

// ParseAndValidateConfig marshals the raw map to JSON, unmarshals it into type T,
// and runs struct validation using the go-playground/validator tags.
func ParseAndValidateConfig[T any](raw map[string]any) (*T, error) {
	b, err := json.Marshal(raw)
	if err != nil {
		return nil, fileError.ErrBlobValidation.WithMessagef(
			"failed to marshal raw config: %v",
			err,
		)
	}

	var cfg T
	if err := json.Unmarshal(b, &cfg); err != nil {
		return nil, fileError.ErrBlobValidation.WithMessagef("failed to unmarshal config: %v", err)
	}

	if err := validate.Struct(&cfg); err != nil {
		// Provide a more readable validation error
		var errMsgs []string
		for _, err := range err.(validator.ValidationErrors) {
			errMsgs = append(errMsgs, fmt.Sprintf("%s is %s", err.Field(), err.Tag()))
		}
		return nil, fileError.ErrBlobValidation.WithMessagef(
			"config validation failed: %s",
			strings.Join(errMsgs, ", "),
		)
	}

	return &cfg, nil
}

// ResolveEncryptionKey returns the AES-256 key for config encryption/decryption.
// Uses config singleton when loaded; falls back to ENCRYPTION_KEY env var otherwise.
// Exported so the service layer can obtain the key for calling cfg.Encrypt().
func ResolveEncryptionKey() string {
	if cfg := config.Get(); cfg != nil && cfg.App.EncryptionKey != "" {
		return cfg.App.EncryptionKey
	}
	return os.Getenv("ENCRYPTION_KEY")
}
