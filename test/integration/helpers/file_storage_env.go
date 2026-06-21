package helpers

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"testing"

	"ecommerce-be/common/cache"
	"ecommerce-be/file/entity"
	"ecommerce-be/file/service/blobAdapter"
	"ecommerce-be/test/integration/setup"

	"github.com/stretchr/testify/require"
)

const DefaultIntegrationTestBucket = "integration-test-bucket"

// FileStorageEnvConfig controls MinIO bucket name and seller storage-config rows.
type FileStorageEnvConfig struct {
	Bucket    string
	SellerIDs []uint64
}

// DefaultFileStorageEnvConfig returns config for seller 2 (john.seller / product 1 owner).
func DefaultFileStorageEnvConfig() FileStorageEnvConfig {
	return FileStorageEnvConfig{
		Bucket:    DefaultIntegrationTestBucket,
		SellerIDs: []uint64{2},
	}
}

// FileStorageEnv holds containers, MinIO, server, and API client for file-upload tests.
type FileStorageEnv struct {
	Containers *setup.TestContainer
	Server     http.Handler
	Client     *APIClient
	Minio      *setup.MinioContainer
	Bucket     string
}

// SetupFileStorageEnv boots testcontainers, seeds core data, MinIO, and storage_config rows.
func SetupFileStorageEnv(t *testing.T, cfg FileStorageEnvConfig) *FileStorageEnv {
	t.Helper()

	if cfg.Bucket == "" {
		cfg.Bucket = DefaultIntegrationTestBucket
	}
	if len(cfg.SellerIDs) == 0 {
		cfg.SellerIDs = []uint64{2}
	}

	containers := setup.SetupTestContainers(t)
	prevRedis, _ := cache.GetRedisClient()
	t.Cleanup(func() {
		containers.Cleanup(t)
		if prevRedis != nil {
			cache.SetRedisClient(prevRedis)
		}
	})

	containers.RunAllMigrations(t)
	containers.RunAllSeeds(t)

	minio := AttachMinIOStorage(t, containers, cfg)

	server := setup.SetupTestServer(t, containers.DB, containers.RedisClient)
	client := NewAPIClient(server)

	return &FileStorageEnv{
		Containers: containers,
		Server:     server,
		Client:     client,
		Minio:      minio,
		Bucket:     cfg.Bucket,
	}
}

// AttachMinIOStorage adds MinIO and storage_config rows to an existing test container.
// Call before SetupTestServer so file resolution uses the seeded storage config.
func AttachMinIOStorage(
	t *testing.T,
	containers *setup.TestContainer,
	cfg FileStorageEnvConfig,
) *setup.MinioContainer {
	t.Helper()

	if cfg.Bucket == "" {
		cfg.Bucket = DefaultIntegrationTestBucket
	}
	if len(cfg.SellerIDs) == 0 {
		cfg.SellerIDs = []uint64{2}
	}

	minio := setup.SetupMinioContainer(t)
	t.Cleanup(func() { minio.Cleanup(t) })
	require.NoError(t, minio.CreateBucket(context.Background(), cfg.Bucket))

	_ = os.Setenv("ENCRYPTION_KEY", "0123456789abcdef0123456789abcdef")

	seedStorageConfigRows(t, containers, minio, cfg.Bucket, cfg.SellerIDs)
	return minio
}

func seedStorageConfigRows(
	t *testing.T,
	containers *setup.TestContainer,
	minio *setup.MinioContainer,
	bucket string,
	sellerIDs []uint64,
) {
	t.Helper()

	raw := map[string]any{
		"access_key_id":     minio.AccessKey,
		"secret_access_key": minio.SecretKey,
		"bucket":            bucket,
		"region":            minio.Region,
		"endpoint":          minio.Endpoint,
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
	err = containers.DB.
		Table("storage_provider").
		Select("id").
		Where("adapter_type = ? AND is_active = ?", "s3_compatible", true).
		Order("id ASC").
		First(&provider).Error
	require.NoError(t, err)

	require.NoError(t, containers.DB.Exec(`DELETE FROM storage_config`).Error)

	require.NoError(t, containers.DB.Exec(`
		INSERT INTO storage_config (
			owner_type, owner_id, provider_id, display_name, bucket_or_container,
			config_data, is_default, is_active, created_at, updated_at
		) VALUES (
			'PLATFORM', 1, ?, 'Integration Platform Default', ?,
			?, true, true, NOW(), NOW()
		)
	`, provider.ID, bucket, encryptedData).Error)

	for _, sellerID := range sellerIDs {
		require.NoError(t, containers.DB.Exec(`
			INSERT INTO storage_config (
				owner_type, owner_id, provider_id, display_name, bucket_or_container,
				config_data, is_default, is_active, created_at, updated_at
			) VALUES (
				'SELLER', ?, ?, 'Integration Seller Config', ?,
				?, true, true, NOW(), NOW()
			)
		`, sellerID, provider.ID, bucket, encryptedData).Error)
	}
}

// SeedVariantMediaRow inserts a variant_media association for integration tests.
func SeedVariantMediaRow(
	t *testing.T,
	containers *setup.TestContainer,
	variantID int,
	fileID string,
	isPrimary bool,
	displayOrder int,
) {
	t.Helper()
	sqlDB, err := containers.DB.DB()
	require.NoError(t, err)
	_, err = sqlDB.Exec(
		`INSERT INTO variant_media (variant_id, file_id, is_primary, display_order, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, NOW(), NOW())
		 ON CONFLICT (variant_id, file_id) DO NOTHING`,
		variantID, fileID, isPrimary, displayOrder,
	)
	require.NoError(t, err)
}

// SeedProductMediaRow inserts a product_media association for integration tests.
func SeedProductMediaRow(
	t *testing.T,
	containers *setup.TestContainer,
	productID int,
	fileID string,
	isPrimary bool,
	displayOrder int,
) {
	t.Helper()
	sqlDB, err := containers.DB.DB()
	require.NoError(t, err)
	_, err = sqlDB.Exec(
		`INSERT INTO product_media (product_id, file_id, is_primary, display_order, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, NOW(), NOW())
		 ON CONFLICT (product_id, file_id) DO NOTHING`,
		productID, fileID, isPrimary, displayOrder,
	)
	require.NoError(t, err)
}

// PredictNextUserID returns the next auto-increment user ID for pre-registration file ownership.
func PredictNextUserID(t *testing.T, containers *setup.TestContainer) uint {
	t.Helper()
	var maxID uint
	err := containers.DB.Raw(`SELECT COALESCE(MAX(id), 0) FROM "user"`).Scan(&maxID).Error
	require.NoError(t, err)
	return maxID + 1
}

// ReassignFileOwnerToSeller updates file_object ownership for seller registration tests.
func ReassignFileOwnerToSeller(
	t *testing.T,
	containers *setup.TestContainer,
	fileID string,
	sellerID uint,
) {
	t.Helper()
	err := containers.DB.Exec(`
		UPDATE file_object
		SET owner_type = 'SELLER', owner_id = ?, seller_id = ?
		WHERE file_id = ?
	`, sellerID, sellerID, fileID).Error
	require.NoError(t, err)
}
