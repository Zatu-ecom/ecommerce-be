package file_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"testing"

	"ecommerce-be/common/db"
	"ecommerce-be/file/entity"
	fileError "ecommerce-be/file/error"
	"ecommerce-be/file/model"
	"ecommerce-be/file/service/blobAdapter"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testEncryptionKey is a 32-byte key used across all factory tests.
const testEncryptionKey = "0123456789abcdef0123456789abcdef"

// newAdapterFromStorageConfig mirrors blobAdapter.GetAdapterFromStoredConfig (production path).
func newAdapterFromStorageConfig(
	ctx context.Context,
	sc entity.StorageConfig,
) (blobAdapter.BlobAdapter, error) {
	return blobAdapter.GetAdapterFromStoredConfig(ctx, sc.Provider.AdapterType, sc.ConfigData)
}

// encryptS3Config produces field-level encrypted config_data for an s3_compatible config.
func encryptS3Config(t *testing.T, key string, raw map[string]any) db.JSONMap {
	t.Helper()
	t.Setenv("ENCRYPTION_KEY", key)
	parser, err := blobAdapter.GetBlobConfigParser(entity.AdapterTypeS3Compatible)
	require.NoError(t, err)
	cfg, err := parser.ParseAndValidateConfig(raw)
	require.NoError(t, err)
	require.NoError(t, cfg.Encrypt())
	return db.JSONMap(cfg.ToMap())
}

// encryptGCSConfig produces field-level encrypted config_data for a gcs config.
func encryptGCSConfig(t *testing.T, key string, raw map[string]any) db.JSONMap {
	t.Helper()
	t.Setenv("ENCRYPTION_KEY", key)
	parser, err := blobAdapter.GetBlobConfigParser(entity.AdapterTypeGCS)
	require.NoError(t, err)
	cfg, err := parser.ParseAndValidateConfig(raw)
	require.NoError(t, err)
	require.NoError(t, cfg.Encrypt())
	return db.JSONMap(cfg.ToMap())
}

// encryptAzureConfig produces field-level encrypted config_data for an azure config.
func encryptAzureConfig(t *testing.T, key string, raw map[string]any) db.JSONMap {
	t.Helper()
	t.Setenv("ENCRYPTION_KEY", key)
	parser, err := blobAdapter.GetBlobConfigParser(entity.AdapterTypeAzure)
	require.NoError(t, err)
	cfg, err := parser.ParseAndValidateConfig(raw)
	require.NoError(t, err)
	require.NoError(t, cfg.Encrypt())
	return db.JSONMap(cfg.ToMap())
}

// ─── GetAdapter + Validate tests ──────────────────────────────────────────────

func TestGetAdapter_ReturnsProtoAdapterForKnownTypes(t *testing.T) {
	ctx := context.Background()

	saJSON := generateGCSServiceAccountJSON(t)
	const validB64AzureKey = "Eby8vdM02xNOcqFlqUwJPLlmEtlCDXJ1OUzFT50uSRZ6IFsuFq2UVErCz4I6tq/K1SZFPTOtr/KBHBeksoGMGw=="

	cases := []struct {
		name        string
		adapterType entity.AdapterType
		raw         map[string]any
	}{
		{
			name:        "s3_compatible",
			adapterType: entity.AdapterTypeS3Compatible,
			raw: map[string]any{
				"access_key_id":     "AK",
				"secret_access_key": "SK",
				"bucket":            "test-bucket",
				"region":            "us-east-1",
				"endpoint":          "http://127.0.0.1:9000",
				"force_path_style":  true,
			},
		},
		{
			name:        "gcs",
			adapterType: entity.AdapterTypeGCS,
			raw: map[string]any{
				"service_account_json": saJSON,
				"bucket":               "test-bucket-gcs",
			},
		},
		{
			name:        "azure",
			adapterType: entity.AdapterTypeAzure,
			raw: map[string]any{
				"account_name": "devstoreaccount1",
				"account_key":  validB64AzureKey,
				"container":    "test-container",
				"endpoint":     "http://127.0.0.1:10000/devstoreaccount1",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Plaintext maps match what adapters receive after decrypt in production.
			a, err := blobAdapter.GetAdapter(ctx, tc.adapterType, tc.raw)
			assert.NoError(t, err)
			assert.NotNil(t, a)
		})
	}
}

func TestGetAdapter_UnknownType_ReturnsValidationError(t *testing.T) {
	ctx := context.Background()
	_, err := blobAdapter.GetAdapter(ctx, entity.AdapterType("unknown_provider"), map[string]any{})
	assert.Error(t, err)
	assert.True(t, fileError.IsBlobError(err, fileError.ErrBlobValidation))
}

func TestGetAdapter_EmptyType_ReturnsValidationError(t *testing.T) {
	ctx := context.Background()
	_, err := blobAdapter.GetAdapter(ctx, "", map[string]any{})
	assert.Error(t, err)
	assert.True(t, fileError.IsBlobError(err, fileError.ErrBlobValidation))
}

// ─── ParseXxxConfig + Validate tests ─────────────────────────────────────────

func TestParseS3Config_ValidConfig(t *testing.T) {
	raw := map[string]any{
		"access_key_id":     "AK",
		"secret_access_key": "SK",
		"bucket":            "test-bucket",
		"region":            "us-east-1",
		"endpoint":          "http://localhost:9000",
		"force_path_style":  true,
	}
	parser, err := blobAdapter.GetBlobConfigParser(entity.AdapterTypeS3Compatible)
	require.NoError(t, err)
	cfg, err := parser.ParseAndValidateConfig(raw)
	assert.NoError(t, err)
	assert.NotNil(t, cfg)
}

func TestParseS3Config_MissingBucket_ReturnsValidationError(t *testing.T) {
	raw := map[string]any{
		"access_key_id":     "AK",
		"secret_access_key": "SK",
	}
	parser, err := blobAdapter.GetBlobConfigParser(entity.AdapterTypeS3Compatible)
	require.NoError(t, err)
	_, err = parser.ParseAndValidateConfig(raw)
	assert.Error(t, err)
	assert.True(t, fileError.IsBlobError(err, fileError.ErrBlobValidation))
}

func TestParseGCSConfig_MissingBucket_ReturnsValidationError(t *testing.T) {
	saJSON := generateGCSServiceAccountJSON(t)
	raw := map[string]any{
		"service_account_json": saJSON,
		// missing bucket
	}
	parser, err := blobAdapter.GetBlobConfigParser(entity.AdapterTypeGCS)
	require.NoError(t, err)
	_, err = parser.ParseAndValidateConfig(raw)
	assert.Error(t, err)
	assert.True(t, fileError.IsBlobError(err, fileError.ErrBlobValidation))
}

func TestParseGCSConfig_ServiceAccountJSON_AsObject_Succeeds(t *testing.T) {
	saStr := generateGCSServiceAccountJSON(t)
	var saObj map[string]any
	require.NoError(t, json.Unmarshal([]byte(saStr), &saObj))
	raw := map[string]any{
		"service_account_json": saObj,
		"bucket":               "test-bucket-gcs",
	}
	parser, err := blobAdapter.GetBlobConfigParser(entity.AdapterTypeGCS)
	require.NoError(t, err)
	cfg, err := parser.ParseAndValidateConfig(raw)
	require.NoError(t, err)
	require.NotNil(t, cfg)
}

func TestParseGCSConfig_ServiceAccountJSON_AsString_StillValid(t *testing.T) {
	saJSON := generateGCSServiceAccountJSON(t)
	raw := map[string]any{
		"service_account_json": saJSON,
		"bucket":               "test-bucket-gcs",
	}
	parser, err := blobAdapter.GetBlobConfigParser(entity.AdapterTypeGCS)
	require.NoError(t, err)
	cfg, err := parser.ParseAndValidateConfig(raw)
	require.NoError(t, err)
	require.NotNil(t, cfg)
}

func TestParseGCSConfig_ServiceAccountJSON_InvalidType_ReturnsValidationError(t *testing.T) {
	raw := map[string]any{
		"service_account_json": 12345,
		"bucket":               "test-bucket-gcs",
	}
	parser, err := blobAdapter.GetBlobConfigParser(entity.AdapterTypeGCS)
	require.NoError(t, err)
	_, err = parser.ParseAndValidateConfig(raw)
	require.Error(t, err)
	assert.True(t, fileError.IsBlobError(err, fileError.ErrBlobValidation))
}

func TestParseAzureConfig_MissingContainer_ReturnsValidationError(t *testing.T) {
	const validB64Key = "Eby8vdM02xNOcqFlqUwJPLlmEtlCDXJ1OUzFT50uSRZ6IFsuFq2UVErCz4I6tq/K1SZFPTOtr/KBHBeksoGMGw=="
	raw := map[string]any{
		"account_name": "devstoreaccount1",
		"account_key":  validB64Key,
		// missing container
	}
	parser, err := blobAdapter.GetBlobConfigParser(entity.AdapterTypeAzure)
	require.NoError(t, err)
	_, err = parser.ParseAndValidateConfig(raw)
	assert.Error(t, err)
	assert.True(t, fileError.IsBlobError(err, fileError.ErrBlobValidation))
}

// ─── Dispatch + validation tests ──────────────────────────────────────────────

func TestNewAdapterFromConfig_DispatchAndValidation(t *testing.T) {
	t.Setenv("ENCRYPTION_KEY", testEncryptionKey)
	ctx := context.Background()

	t.Run("missing adapter type -> validation error", func(t *testing.T) {
		_, err := newAdapterFromStorageConfig(ctx, entity.StorageConfig{
			Provider: entity.StorageProvider{AdapterType: ""},
		})
		assert.Error(t, err)
		assert.True(t, fileError.IsBlobError(err, fileError.ErrBlobValidation))
	})

	t.Run("missing credentials payload -> validation error", func(t *testing.T) {
		_, err := newAdapterFromStorageConfig(ctx, entity.StorageConfig{
			Provider:   entity.StorageProvider{AdapterType: "s3_compatible"},
			ConfigData: nil,
		})
		assert.Error(t, err)
		assert.True(t, fileError.IsBlobError(err, fileError.ErrBlobValidation))
	})

	t.Run("unknown adapter type -> validation error", func(t *testing.T) {
		// Build a valid s3 blob but use an unknown adapter type
		b := encryptS3Config(t, testEncryptionKey, map[string]any{
			"access_key_id":     "AK",
			"secret_access_key": "SK",
			"bucket":            "test-bucket",
			"endpoint":          "http://localhost:9000",
		})
		_, err := newAdapterFromStorageConfig(ctx, entity.StorageConfig{
			Provider:   entity.StorageProvider{AdapterType: "unknown_provider"},
			ConfigData: b,
		})
		assert.Error(t, err)
		assert.True(t, fileError.IsBlobError(err, fileError.ErrBlobValidation))
	})

	t.Run("s3_compatible with valid creds returns adapter", func(t *testing.T) {
		b := encryptS3Config(t, testEncryptionKey, map[string]any{
			"access_key_id":     "AK",
			"secret_access_key": "SK",
			"bucket":            "test-bucket",
			"endpoint":          "http://localhost:9000",
			"region":            "us-east-1",
			"force_path_style":  true,
		})
		a, err := newAdapterFromStorageConfig(ctx, entity.StorageConfig{
			Provider:   entity.StorageProvider{AdapterType: "s3_compatible"},
			ConfigData: b,
		})
		assert.NoError(t, err)
		assert.NotNil(t, a)
	})

	t.Run("s3_compatible missing secret_access_key -> validation error on encrypt", func(t *testing.T) {
		var badData db.JSONMap
		require.NoError(t, json.Unmarshal(
			[]byte(`{"access_key_id":"AK","bucket":"b","region":"us-east-1"}`),
			&badData,
		))
		_, err := newAdapterFromStorageConfig(ctx, entity.StorageConfig{
			Provider:   entity.StorageProvider{AdapterType: "s3_compatible"},
			ConfigData: badData,
		})
		assert.Error(t, err)
	})

	t.Run("gcs with valid service_account_json returns adapter", func(t *testing.T) {
		saJSON := generateGCSServiceAccountJSON(t)
		b := encryptGCSConfig(t, testEncryptionKey, map[string]any{
			"service_account_json": saJSON,
			"bucket":               "test-bucket",
		})
		a, err := newAdapterFromStorageConfig(ctx, entity.StorageConfig{
			Provider:   entity.StorageProvider{AdapterType: "gcs"},
			ConfigData: b,
		})
		assert.NoError(t, err)
		assert.NotNil(t, a)
	})

	t.Run("gcs missing service_account_json -> factory init error", func(t *testing.T) {
		var badData db.JSONMap
		require.NoError(t, json.Unmarshal(
			[]byte(`{"bucket":"test-bucket","project_id":"proj"}`),
			&badData,
		))
		_, err := newAdapterFromStorageConfig(ctx, entity.StorageConfig{
			Provider:   entity.StorageProvider{AdapterType: "gcs"},
			ConfigData: badData,
		})
		assert.Error(t, err)
	})

	t.Run("azure with account_name + account_key returns adapter", func(t *testing.T) {
		const validB64Key = "Eby8vdM02xNOcqFlqUwJPLlmEtlCDXJ1OUzFT50uSRZ6IFsuFq2UVErCz4I6tq/K1SZFPTOtr/KBHBeksoGMGw=="
		b := encryptAzureConfig(t, testEncryptionKey, map[string]any{
			"account_name": "devstoreaccount1",
			"account_key":  validB64Key,
			"container":    "my-container",
		})
		a, err := newAdapterFromStorageConfig(ctx, entity.StorageConfig{
			Provider:   entity.StorageProvider{AdapterType: "azure"},
			ConfigData: b,
		})
		assert.NoError(t, err)
		assert.NotNil(t, a)
	})

	t.Run("azure missing account_name -> factory init error", func(t *testing.T) {
		var badData db.JSONMap
		require.NoError(t, json.Unmarshal(
			[]byte(`{"account_key":"key","container":"my-container"}`),
			&badData,
		))
		_, err := newAdapterFromStorageConfig(ctx, entity.StorageConfig{
			Provider:   entity.StorageProvider{AdapterType: "azure"},
			ConfigData: badData,
		})
		assert.Error(t, err)
	})
}

// ─── Decryption success/failure ───────────────────────────────────────────────

func TestNewAdapterFromConfig_Decryption(t *testing.T) {
	ctx := context.Background()

	t.Run("correct key decrypts and returns adapter", func(t *testing.T) {
		t.Setenv("ENCRYPTION_KEY", testEncryptionKey)

		b := encryptS3Config(t, testEncryptionKey, map[string]any{
			"access_key_id":     "REAL_AK",
			"secret_access_key": "REAL_SK",
			"bucket":            "test-bucket",
			"endpoint":          "http://localhost:9000",
			"region":            "us-east-1",
			"force_path_style":  true,
		})
		a, err := newAdapterFromStorageConfig(ctx, entity.StorageConfig{
			Provider:   entity.StorageProvider{AdapterType: "s3_compatible"},
			ConfigData: b,
		})
		assert.NoError(t, err)
		assert.NotNil(t, a)
	})

	t.Run("wrong key -> decrypt leaves ciphertext; adapter still constructs", func(t *testing.T) {
		const wrongKey = "ffffffffffffffffffffffffffffffff"
		b := encryptS3Config(t, testEncryptionKey, map[string]any{
			"access_key_id":     "REAL_AK",
			"secret_access_key": "REAL_SK",
			"bucket":            "test-bucket",
			"endpoint":          "http://localhost:9000",
		})
		t.Setenv("ENCRYPTION_KEY", wrongKey)

		a, err := newAdapterFromStorageConfig(ctx, entity.StorageConfig{
			Provider:   entity.StorageProvider{AdapterType: "s3_compatible"},
			ConfigData: b,
		})
		assert.NoError(t, err)
		assert.NotNil(t, a)
	})

	t.Run("corrupt payload (non-JSON-marshallable map) -> validation error", func(t *testing.T) {
		t.Setenv("ENCRYPTION_KEY", testEncryptionKey)

		// JSONMap column cannot hold invalid JSON; non-JSON-like corruption is modelled as
		// values that cannot round-trip through encoding/json in ParseAndValidateConfig.
		_, err := newAdapterFromStorageConfig(ctx, entity.StorageConfig{
			Provider: entity.StorageProvider{AdapterType: "s3_compatible"},
			ConfigData: db.JSONMap{
				"x": make(chan int),
			},
		})
		assert.Error(t, err)
		assert.True(t, fileError.IsBlobError(err, fileError.ErrBlobValidation))
	})
}

// ─── No secret leak in error strings ─────────────────────────────────────────

func TestNewAdapterFromConfig_NoSecretLeak(t *testing.T) {
	ctx := context.Background()

	t.Run("wrong key does not surface plaintext secrets in adapter construction", func(t *testing.T) {
		const wrongKey = "ffffffffffffffffffffffffffffffff"
		const accessKey = "SUPER_SECRET_AK_12345"
		const secretKey = "SUPER_SECRET_SK_67890"

		b := encryptS3Config(t, testEncryptionKey, map[string]any{
			"access_key_id":     accessKey,
			"secret_access_key": secretKey,
			"bucket":            "test-bucket",
			"endpoint":          "http://localhost:9000",
		})
		t.Setenv("ENCRYPTION_KEY", wrongKey)

		_, err := newAdapterFromStorageConfig(ctx, entity.StorageConfig{
			Provider:   entity.StorageProvider{AdapterType: "s3_compatible"},
			ConfigData: b,
		})
		if err != nil {
			assert.NotContains(t, err.Error(), accessKey)
			assert.NotContains(t, err.Error(), secretKey)
			assert.NotContains(t, err.Error(), wrongKey)
		}
	})

	t.Run("azure validation error on missing field does not leak account_key", func(t *testing.T) {
		t.Setenv("ENCRYPTION_KEY", testEncryptionKey)

		var badData db.JSONMap
		require.NoError(t, json.Unmarshal(
			[]byte(`{"account_key":"TOPSECRET_AK","container":"my-container"}`),
			&badData,
		))
		_, err := newAdapterFromStorageConfig(ctx, entity.StorageConfig{
			Provider:   entity.StorageProvider{AdapterType: "azure"},
			ConfigData: badData,
		})
		require.Error(t, err)
		assert.NotContains(t, err.Error(), "TOPSECRET_AK")
	})
}

// ─── End-to-end factory → adapter → real MinIO ───────────────────────────────

func TestNewAdapterFromConfig_EndToEndWithMinio(t *testing.T) {
	t.Setenv("ENCRYPTION_KEY", testEncryptionKey)
	ctx := context.Background()

	bucket := "factory-e2e-" + RandomHex(6)
	minio := SetupMinio(t, bucket)
	defer minio.Cleanup(t)

	cfgData := encryptS3Config(t, testEncryptionKey, map[string]any{
		"access_key_id":     minio.AccessKey,
		"secret_access_key": minio.SecretKey,
		"bucket":            bucket,
		"endpoint":          minio.Endpoint,
		"region":            minio.Region,
		"force_path_style":  true,
	})

	a, err := newAdapterFromStorageConfig(ctx, entity.StorageConfig{
		Provider:          entity.StorageProvider{AdapterType: "s3_compatible"},
		BucketOrContainer: bucket,
		ConfigData:        cfgData,
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
