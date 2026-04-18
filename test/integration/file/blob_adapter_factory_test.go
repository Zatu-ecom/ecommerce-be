package file_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"testing"

	"ecommerce-be/common/helper"
	"ecommerce-be/file/entity"
	fileError "ecommerce-be/file/error"
	"ecommerce-be/file/model"
	"ecommerce-be/file/service/blob_adapter"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testEncryptionKey is a 32-byte key used across all factory tests.
const testEncryptionKey = "0123456789abcdef0123456789abcdef"

// encryptCreds marshals creds to JSON then AES-GCM encrypts with testEncryptionKey.
// Returns the ciphertext bytes to pass as cfg.CredentialsEncrypted.
func encryptCreds(t *testing.T, key string, creds map[string]any) []byte {
	t.Helper()
	raw, err := json.Marshal(creds)
	require.NoError(t, err)
	enc, err := helper.Encrypt(string(raw), key)
	require.NoError(t, err)
	return []byte(enc)
}

// ─── Phase 4 dispatch + validation tests (updated to use encrypted payloads) ──

func TestNewAdapterFromConfig_DispatchAndValidation(t *testing.T) {
	t.Setenv("ENCRYPTION_KEY", testEncryptionKey)
	ctx := context.Background()

	t.Run("missing adapter type -> validation error", func(t *testing.T) {
		_, err := blob_adapter.NewAdapterFromConfig(ctx, entity.StorageConfig{
			Provider: entity.StorageProvider{AdapterType: ""},
		})
		assert.Error(t, err)
		assert.True(t, fileError.IsBlobError(err, fileError.ErrBlobValidation))
	})

	t.Run("missing credentials payload -> validation error", func(t *testing.T) {
		_, err := blob_adapter.NewAdapterFromConfig(ctx, entity.StorageConfig{
			Provider:             entity.StorageProvider{AdapterType: "s3_compatible"},
			CredentialsEncrypted: nil,
		})
		assert.Error(t, err)
		assert.True(t, fileError.IsBlobError(err, fileError.ErrBlobValidation))
	})

	t.Run("unknown adapter type -> validation error", func(t *testing.T) {
		b := encryptCreds(t, testEncryptionKey, map[string]any{})
		_, err := blob_adapter.NewAdapterFromConfig(ctx, entity.StorageConfig{
			Provider:             entity.StorageProvider{AdapterType: "unknown_provider"},
			CredentialsEncrypted: b,
		})
		assert.Error(t, err)
		assert.True(t, fileError.IsBlobError(err, fileError.ErrBlobValidation))
	})

	t.Run("s3_compatible with valid creds returns adapter", func(t *testing.T) {
		b := encryptCreds(t, testEncryptionKey, map[string]any{
			"access_key_id":     "AK",
			"secret_access_key": "SK",
		})
		a, err := blob_adapter.NewAdapterFromConfig(ctx, entity.StorageConfig{
			Provider:             entity.StorageProvider{AdapterType: "s3_compatible"},
			Endpoint:             "http://localhost:9000",
			Region:               "us-east-1",
			ForcePathStyle:       true,
			CredentialsEncrypted: b,
		})
		assert.NoError(t, err)
		assert.NotNil(t, a)
	})

	t.Run("s3_compatible missing secret_access_key -> validation error", func(t *testing.T) {
		b := encryptCreds(t, testEncryptionKey, map[string]any{"access_key_id": "AK"})
		_, err := blob_adapter.NewAdapterFromConfig(ctx, entity.StorageConfig{
			Provider:             entity.StorageProvider{AdapterType: "s3_compatible"},
			Endpoint:             "http://localhost:9000",
			CredentialsEncrypted: b,
		})
		assert.Error(t, err)
		assert.True(t, fileError.IsBlobError(err, fileError.ErrBlobValidation))
	})

	t.Run("gcs with valid service_account_json returns adapter", func(t *testing.T) {
		saJSON := generateGCSServiceAccountJSON(t)
		b := encryptCreds(t, testEncryptionKey, map[string]any{"service_account_json": saJSON})
		a, err := blob_adapter.NewAdapterFromConfig(ctx, entity.StorageConfig{
			Provider:             entity.StorageProvider{AdapterType: "gcs"},
			CredentialsEncrypted: b,
		})
		assert.NoError(t, err)
		assert.NotNil(t, a)
	})

	t.Run("gcs missing service_account_json -> validation error", func(t *testing.T) {
		b := encryptCreds(t, testEncryptionKey, map[string]any{})
		_, err := blob_adapter.NewAdapterFromConfig(ctx, entity.StorageConfig{
			Provider:             entity.StorageProvider{AdapterType: "gcs"},
			CredentialsEncrypted: b,
		})
		assert.Error(t, err)
		assert.True(t, fileError.IsBlobError(err, fileError.ErrBlobValidation))
	})

	t.Run("azure with account_name + account_key returns adapter", func(t *testing.T) {
		// Azurite well-known development key — valid base64, safe to commit.
		const validB64Key = "Eby8vdM02xNOcqFlqUwJPLlmEtlCDXJ1OUzFT50uSRZ6IFsuFq2UVErCz4I6tq/K1SZFPTOtr/KBHBeksoGMGw=="
		b := encryptCreds(t, testEncryptionKey, map[string]any{
			"account_name": "devstoreaccount1",
			"account_key":  validB64Key,
		})
		a, err := blob_adapter.NewAdapterFromConfig(ctx, entity.StorageConfig{
			Provider:             entity.StorageProvider{AdapterType: "azure"},
			CredentialsEncrypted: b,
		})
		assert.NoError(t, err)
		assert.NotNil(t, a)
	})

	t.Run("azure missing account_name -> validation error", func(t *testing.T) {
		b := encryptCreds(t, testEncryptionKey, map[string]any{"account_key": "key"})
		_, err := blob_adapter.NewAdapterFromConfig(ctx, entity.StorageConfig{
			Provider:             entity.StorageProvider{AdapterType: "azure"},
			CredentialsEncrypted: b,
		})
		assert.Error(t, err)
		assert.True(t, fileError.IsBlobError(err, fileError.ErrBlobValidation))
	})
}

// ─── Phase 5: T029 — Decryption success/failure ───────────────────────────────

func TestNewAdapterFromConfig_Decryption(t *testing.T) {
	ctx := context.Background()

	t.Run("correct key decrypts and returns adapter", func(t *testing.T) {
		t.Setenv("ENCRYPTION_KEY", testEncryptionKey)

		b := encryptCreds(t, testEncryptionKey, map[string]any{
			"access_key_id":     "REAL_AK",
			"secret_access_key": "REAL_SK",
		})
		a, err := blob_adapter.NewAdapterFromConfig(ctx, entity.StorageConfig{
			Provider:             entity.StorageProvider{AdapterType: "s3_compatible"},
			Endpoint:             "http://localhost:9000",
			Region:               "us-east-1",
			ForcePathStyle:       true,
			CredentialsEncrypted: b,
		})
		assert.NoError(t, err)
		assert.NotNil(t, a)
	})

	t.Run("wrong key -> factory init error", func(t *testing.T) {
		const wrongKey = "ffffffffffffffffffffffffffffffff"
		// Encrypt with testKey but factory will see wrongKey.
		b := encryptCreds(t, testEncryptionKey, map[string]any{
			"access_key_id":     "REAL_AK",
			"secret_access_key": "REAL_SK",
		})
		t.Setenv("ENCRYPTION_KEY", wrongKey)

		_, err := blob_adapter.NewAdapterFromConfig(ctx, entity.StorageConfig{
			Provider:             entity.StorageProvider{AdapterType: "s3_compatible"},
			Endpoint:             "http://localhost:9000",
			CredentialsEncrypted: b,
		})
		assert.Error(t, err)
		assert.True(t, fileError.IsBlobError(err, fileError.ErrBlobFactoryInit))
	})

	t.Run("corrupt payload (not base64) -> factory init error", func(t *testing.T) {
		t.Setenv("ENCRYPTION_KEY", testEncryptionKey)

		_, err := blob_adapter.NewAdapterFromConfig(ctx, entity.StorageConfig{
			Provider:             entity.StorageProvider{AdapterType: "s3_compatible"},
			Endpoint:             "http://localhost:9000",
			CredentialsEncrypted: []byte("this-is-not-valid-base64-ciphertext!!!"),
		})
		assert.Error(t, err)
		assert.True(t, fileError.IsBlobError(err, fileError.ErrBlobFactoryInit))
	})

	t.Run("missing encryption key env -> factory init error", func(t *testing.T) {
		t.Setenv("ENCRYPTION_KEY", "")

		b := encryptCreds(t, testEncryptionKey, map[string]any{
			"access_key_id":     "AK",
			"secret_access_key": "SK",
		})
		_, err := blob_adapter.NewAdapterFromConfig(ctx, entity.StorageConfig{
			Provider:             entity.StorageProvider{AdapterType: "s3_compatible"},
			Endpoint:             "http://localhost:9000",
			CredentialsEncrypted: b,
		})
		assert.Error(t, err)
		assert.True(t, fileError.IsBlobError(err, fileError.ErrBlobFactoryInit))
	})
}

// ─── Phase 5: T030 — No secret leak in error strings ─────────────────────────

func TestNewAdapterFromConfig_NoSecretLeak(t *testing.T) {
	ctx := context.Background()

	t.Run("wrong key error does not contain access key or secret key", func(t *testing.T) {
		const wrongKey = "ffffffffffffffffffffffffffffffff"
		const accessKey = "SUPER_SECRET_AK_12345"
		const secretKey = "SUPER_SECRET_SK_67890"

		b := encryptCreds(t, testEncryptionKey, map[string]any{
			"access_key_id":     accessKey,
			"secret_access_key": secretKey,
		})
		t.Setenv("ENCRYPTION_KEY", wrongKey)

		_, err := blob_adapter.NewAdapterFromConfig(ctx, entity.StorageConfig{
			Provider:             entity.StorageProvider{AdapterType: "s3_compatible"},
			Endpoint:             "http://localhost:9000",
			CredentialsEncrypted: b,
		})
		require.Error(t, err)
		assert.NotContains(t, err.Error(), accessKey)
		assert.NotContains(t, err.Error(), secretKey)
		assert.NotContains(t, err.Error(), wrongKey)
	})

	t.Run("validation error on missing field does not leak other cred fields", func(t *testing.T) {
		t.Setenv("ENCRYPTION_KEY", testEncryptionKey)

		// Encrypt payload with account_key present but account_name missing.
		// Factory decrypts successfully, then validation fires — error must not echo account_key.
		b := encryptCreds(t, testEncryptionKey, map[string]any{
			"account_key": "TOPSECRET_AK",
		})
		_, err := blob_adapter.NewAdapterFromConfig(ctx, entity.StorageConfig{
			Provider:             entity.StorageProvider{AdapterType: "azure"},
			CredentialsEncrypted: b,
		})
		require.Error(t, err)
		assert.NotContains(t, err.Error(), "TOPSECRET_AK")
	})
}

// ─── Phase 6: End-to-end factory → adapter → real MinIO ──────────────────────

// TestNewAdapterFromConfig_EndToEndWithMinio verifies the full production seam:
// encrypted StorageConfig → factory decrypts → adapter constructed → adapter
// actually communicates with a real S3-compatible server. This guards against
// the adapter being a zero-value stub that panics on first use.
func TestNewAdapterFromConfig_EndToEndWithMinio(t *testing.T) {
	t.Setenv("ENCRYPTION_KEY", testEncryptionKey)
	ctx := context.Background()

	bucket := "factory-e2e-" + RandomHex(6)
	minio := SetupMinio(t, bucket)
	defer minio.Cleanup(t)

	creds := encryptCreds(t, testEncryptionKey, map[string]any{
		"access_key_id":     minio.AccessKey,
		"secret_access_key": minio.SecretKey,
	})

	a, err := blob_adapter.NewAdapterFromConfig(ctx, entity.StorageConfig{
		Provider:             entity.StorageProvider{AdapterType: "s3_compatible"},
		BucketOrContainer:    bucket,
		Endpoint:             minio.Endpoint,
		Region:               minio.Region,
		ForcePathStyle:       true,
		CredentialsEncrypted: creds,
	})
	require.NoError(t, err)
	require.NotNil(t, a)

	key := RandomObjectKey("factory-e2e")
	payload := "end-to-end factory roundtrip"

	_, err = a.PutObject(ctx, model.BlobPutObjectInput{
		Bucket:        bucket,
		Key:           key,
		ContentType:   "text/plain",
		ContentLength: int64(len(payload)),
		Body:          bytes.NewReader([]byte(payload)),
	})
	require.NoError(t, err, "adapter returned by factory must be able to PutObject against real server")

	rc, meta, err := a.GetObjectStream(ctx, bucket, key)
	require.NoError(t, err)
	defer rc.Close()
	b, _ := io.ReadAll(rc)
	assert.Equal(t, payload, string(b))
	assert.Equal(t, int64(len(payload)), meta.SizeBytes)
}
