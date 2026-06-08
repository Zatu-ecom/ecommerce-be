package variant_media_test

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"testing"

	"ecommerce-be/common/cache"
	"ecommerce-be/common/config"
	"ecommerce-be/common/constants"
	"ecommerce-be/file/entity"
	fileSingleton "ecommerce-be/file/factory/singleton"
	"ecommerce-be/file/service/blobAdapter"
	"ecommerce-be/test/integration/helpers"
	"ecommerce-be/test/integration/setup"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

const (
	variantMediaTestBucket = "variant-media-test-bucket"
	variantMediaSellerID   = uint64(2) // john.seller / product 1 owner
)

// variantMediaStorageEnv extends the base env with MinIO and seller storage config
// so variant media file resolution can be exercised end-to-end.
type variantMediaStorageEnv struct {
	*variantMediaTestEnv
	minio *setup.MinioContainer
}

func newVariantMediaStorageTestEnv(t *testing.T) *variantMediaStorageEnv {
	t.Helper()

	containers := setup.SetupTestContainers(t)
	t.Cleanup(func() { containers.Cleanup(t) })

	containers.RunAllMigrations(t)
	containers.RunAllCoreSeeds(t)
	containers.RunSeeds(t, "migrations/seeds/mock/001_seed_users.sql")
	containers.RunSeeds(t, "migrations/seeds/mock/002_seed_products.sql")

	minio := setup.SetupMinioContainer(t)
	t.Cleanup(func() { minio.Cleanup(t) })
	require.NoError(t, minio.CreateBucket(context.Background(), variantMediaTestBucket))

	_ = os.Setenv("ENCRYPTION_KEY", "0123456789abcdef0123456789abcdef")
	config.Reset()
	cache.SetRedisClient(containers.RedisClient)
	fileSingleton.ResetInstance()

	server := setup.SetupTestServer(t, containers.DB, containers.RedisClient)
	client := helpers.NewAPIClient(server)

	env := &variantMediaStorageEnv{
		variantMediaTestEnv: &variantMediaTestEnv{
			client:     client,
			containers: containers,
		},
		minio: minio,
	}
	env.seedSellerStorageConfig(t)

	return env
}

func (e *variantMediaStorageEnv) seedSellerStorageConfig(t *testing.T) {
	t.Helper()

	raw := map[string]any{
		"access_key_id":     e.minio.AccessKey,
		"secret_access_key": e.minio.SecretKey,
		"bucket":            variantMediaTestBucket,
		"region":            e.minio.Region,
		"endpoint":          e.minio.Endpoint,
		"force_path_style":  true,
	}
	parser, err := blobAdapter.GetBlobConfigParser(entity.AdapterTypeS3Compatible)
	require.NoError(t, err)
	blobCfg, err := parser.ParseAndValidateConfig(raw)
	require.NoError(t, err)
	require.NoError(t, blobCfg.Encrypt())
	encryptedData, err := json.Marshal(blobCfg.ToMap())
	require.NoError(t, err)

	type providerRow struct {
		ID uint
	}
	var provider providerRow
	err = e.containers.DB.
		Table("storage_provider").
		Select("id").
		Where("adapter_type = ? AND is_active = ?", "s3_compatible", true).
		Order("id ASC").
		First(&provider).Error
	require.NoError(t, err)

	require.NoError(t, e.containers.DB.Exec(`DELETE FROM storage_config`).Error)

	// Platform default (fallback).
	require.NoError(t, e.containers.DB.Exec(`
		INSERT INTO storage_config (
			owner_type, owner_id, provider_id, display_name, bucket_or_container,
			config_data, is_default, is_active, created_at, updated_at
		) VALUES (
			'PLATFORM', 1, ?, 'Variant Media Platform Default', ?,
			?, true, true, NOW(), NOW()
		)
	`, provider.ID, variantMediaTestBucket, encryptedData).Error)

	// Seller 2 default — matches product 1 owner and john.seller uploads.
	require.NoError(t, e.containers.DB.Exec(`
		INSERT INTO storage_config (
			owner_type, owner_id, provider_id, display_name, bucket_or_container,
			config_data, is_default, is_active, created_at, updated_at
		) VALUES (
			'SELLER', ?, ?, 'Variant Media Seller Config', ?,
			?, true, true, NOW(), NOW()
		)
	`, variantMediaSellerID, provider.ID, variantMediaTestBucket, encryptedData).Error)
}

// uploadFileAsSeller uploads a PRODUCT_IMAGE via the file module and returns the fileId.
func uploadFileAsSeller(t *testing.T, env *variantMediaStorageEnv, token string) string {
	t.Helper()

	client := helpers.NewAPIClient(env.client.Handler)
	client.SetToken(token)
	client.SetHeader(constants.CORRELATION_ID_HEADER, uuid.NewString())

	initReq := map[string]any{
		"purpose":    "PRODUCT_IMAGE",
		"visibility": "PRIVATE",
		"filename":   "variant-media-test.jpg",
		"mimeType":   "image/jpeg",
		"sizeBytes":  1024,
	}
	initW := client.Post(t, "/api/file/init-upload", initReq)
	initResp := helpers.AssertSuccessResponse(t, initW, http.StatusCreated)
	initData := initResp["data"].(map[string]any)
	fileID := initData["fileId"].(string)

	uploadHelper := helpers.UploadHelper{Server: env.client.Handler, Token: token}
	uploadHelper.PutBytes(t, initData, make([]byte, 1024))

	completeW := client.Post(t, "/api/file/complete-upload", map[string]any{"fileId": fileID})
	helpers.AssertSuccessResponse(t, completeW, http.StatusOK)

	return fileID
}
