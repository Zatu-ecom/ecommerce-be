package blob_adapter

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"ecommerce-be/common/config"
	"ecommerce-be/common/helper"
	"ecommerce-be/file/entity"
	fileError "ecommerce-be/file/error"
)

// NewAdapterFromConfig is the public factory entry point for the blob adapter layer.
//
// Preconditions:
//   - cfg.Provider must be pre-loaded (GORM Preload) so cfg.Provider.AdapterType is set.
//   - cfg.CredentialsEncrypted must contain an AES-GCM encrypted, base64-encoded JSON
//     credential blob produced by the same key that is currently configured.
//
// Key resolution order:
//  1. config.Get().App.EncryptionKey (when the full application config is loaded)
//  2. ENCRYPTION_KEY environment variable (test environments / config not yet loaded)
//
// Returns ErrBlobValidation for missing/unknown provider type or invalid credentials.
// Returns ErrBlobFactoryInit for decryption, parsing, or initialisation failures.
func NewAdapterFromConfig(ctx context.Context, cfg entity.StorageConfig) (BlobAdapter, error) {
	adapterType := strings.TrimSpace(cfg.Provider.AdapterType)
	if adapterType == "" {
		return nil, fileError.ErrBlobValidation.WithMessagef(
			"factory: missing provider adapter type (cfg.Provider.AdapterType); ensure Provider is preloaded",
		)
	}

	if len(cfg.CredentialsEncrypted) == 0 {
		return nil, fileError.ErrBlobValidation.WithMessagef(
			"factory: missing credentials payload (cfg.CredentialsEncrypted)",
		)
	}

	key := resolveEncryptionKey()
	if key == "" {
		return nil, fileError.ErrBlobFactoryInit.WithMessagef(
			"factory: encryption key not configured (set ENCRYPTION_KEY env or load app config)",
		)
	}

	plaintext, err := helper.Decrypt(string(cfg.CredentialsEncrypted), key)
	if err != nil {
		return nil, fileError.ErrBlobFactoryInit.WithMessagef(
			"factory: failed to decrypt credentials (wrong key or corrupt payload)",
		)
	}

	var raw map[string]any
	if err := json.Unmarshal([]byte(plaintext), &raw); err != nil {
		return nil, fileError.ErrBlobValidation.WithMessagef(
			"factory: decrypted payload is not valid JSON",
		)
	}

	switch adapterType {
	case "s3_compatible":
		creds, err := parseS3Creds(raw)
		if err != nil {
			return nil, err
		}
		return NewS3CompatibleAdapter(ctx, S3CompatibleOptions{
			Endpoint:        cfg.Endpoint,
			Region:          cfg.Region,
			ForcePathStyle:  cfg.ForcePathStyle,
			AccessKeyID:     creds.accessKeyID,
			SecretAccessKey: creds.secretAccessKey,
			SessionToken:    creds.sessionToken,
		})
	case "gcs":
		creds, err := parseGCSCreds(raw)
		if err != nil {
			return nil, err
		}
		return NewGCSAdapter(ctx, GCSOptions{
			ServiceAccountJSON: creds.serviceAccountJSON,
			ProjectID:          creds.projectID,
			Endpoint:           cfg.Endpoint,
		})
	case "azure":
		creds, err := parseAzureCreds(raw)
		if err != nil {
			return nil, err
		}
		return NewAzureBlobAdapter(AzureOptions{
			AccountName: creds.accountName,
			AccountKey:  creds.accountKey,
			Endpoint:    cfg.Endpoint,
		})
	default:
		return nil, fileError.ErrBlobValidation.WithMessagef(
			"factory: unsupported adapter type %q", fmt.Sprintf("%s", adapterType),
		)
	}
}

// resolveEncryptionKey returns the AES-256 key to use for credential decryption.
// Uses config singleton when loaded; falls back to ENCRYPTION_KEY env var otherwise.
func resolveEncryptionKey() string {
	if cfg := config.Get(); cfg != nil && cfg.App.EncryptionKey != "" {
		return cfg.App.EncryptionKey
	}
	return os.Getenv("ENCRYPTION_KEY")
}

// ─── Credential structs (unexported — never escape to error messages) ─────────

type s3CompatibleCredentials struct {
	accessKeyID     string
	secretAccessKey string
	sessionToken    string
}

type gcsCredentials struct {
	serviceAccountJSON string
	projectID          string
}

type azureCredentials struct {
	accountName string
	accountKey  string
	sas         string
}

// ─── Credential parsers ───────────────────────────────────────────────────────

func parseS3Creds(raw map[string]any) (s3CompatibleCredentials, error) {
	ak, _ := raw["access_key_id"].(string)
	sk, _ := raw["secret_access_key"].(string)
	st, _ := raw["session_token"].(string)
	ak = strings.TrimSpace(ak)
	sk = strings.TrimSpace(sk)
	if ak == "" || sk == "" {
		return s3CompatibleCredentials{}, fileError.ErrBlobValidation.WithMessagef(
			"[s3_compatible] credentials: missing access_key_id or secret_access_key",
		)
	}
	return s3CompatibleCredentials{
		accessKeyID:     ak,
		secretAccessKey: sk,
		sessionToken:    strings.TrimSpace(st),
	}, nil
}

func parseGCSCreds(raw map[string]any) (gcsCredentials, error) {
	saj, _ := raw["service_account_json"].(string)
	if strings.TrimSpace(saj) == "" {
		return gcsCredentials{}, fileError.ErrBlobValidation.WithMessagef(
			"[gcs] credentials: missing service_account_json",
		)
	}
	pid, _ := raw["project_id"].(string)
	return gcsCredentials{serviceAccountJSON: saj, projectID: strings.TrimSpace(pid)}, nil
}

func parseAzureCreds(raw map[string]any) (azureCredentials, error) {
	an, _ := raw["account_name"].(string)
	ak, _ := raw["account_key"].(string)
	sas, _ := raw["sas"].(string)
	an = strings.TrimSpace(an)
	ak = strings.TrimSpace(ak)
	sas = strings.TrimSpace(sas)
	if an == "" {
		return azureCredentials{}, fileError.ErrBlobValidation.WithMessagef(
			"[azure] credentials: missing account_name",
		)
	}
	if ak == "" && sas == "" {
		return azureCredentials{}, fileError.ErrBlobValidation.WithMessagef(
			"[azure] credentials: missing account_key or sas token",
		)
	}
	return azureCredentials{accountName: an, accountKey: ak, sas: sas}, nil
}
