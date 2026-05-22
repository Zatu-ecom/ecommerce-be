package file_test

import (
	"context"
	"encoding/json"
	"net/http"

	"ecommerce-be/common/constants"
	"ecommerce-be/file/entity"
	"ecommerce-be/file/service/blobAdapter"
	"ecommerce-be/test/integration/helpers"

	"github.com/stretchr/testify/require"
)

// ============================================================================
// Storage config resolution (seller vs platform default) — integration tests
//
// Behavior under test (see upload_service.resolveStorageConfig):
// - If caller is seller and has an active default seller config => use that
// - Otherwise => fallback to active platform default
// - If neither exists => 412 FILE_UPLOAD_NO_STORAGE_CONFIG (covered elsewhere)
// ============================================================================

func (s *UploadSuite) TestInitUpload_StorageConfigResolution_FallbackMatrix() {
	// We need distinct, real buckets so the adapter ping passes.
	ctx := context.Background()

	platformBucket := "platform-default-resolution-bucket"
	seller1Bucket := "seller1-default-resolution-bucket"
	seller2NonDefaultBucket := "seller2-nondefault-resolution-bucket"
	seller2DefaultBucket := "seller2-default-resolution-bucket"

	for _, b := range []string{platformBucket, seller1Bucket, seller2NonDefaultBucket, seller2DefaultBucket} {
		require.NoError(s.T(), s.minio.CreateBucket(ctx, b))
	}

	restorePlatform := s.setPlatformDefaultS3Config(platformBucket)
	defer restorePlatform()

	// Case 1: Seller1 has active default seller config => use seller config.
	restoreSeller1 := s.setSellerS3Config(uint64(uploadTestSellerID), seller1Bucket, true, true)
	defer restoreSeller1()

	s.Run("SellerWithActiveDefault_UsesSellerConfig", func() {
		client := helpers.NewAPIClient(s.server)
		client.SetToken(s.sellerToken)
		client.SetHeader(constants.CORRELATION_ID_HEADER, "res-seller-default")

		w := client.Post(s.T(), uploadInitEndpoint, initOutageRequest("seller-default.jpg"))
		resp := helpers.AssertSuccessResponse(s.T(), w, http.StatusCreated)

		fileID := resp["data"].(map[string]any)["fileId"].(string)
		row := s.getFileObjectRowByFileID(fileID)

		require.Equal(s.T(), seller1Bucket, row.BucketOrContainer)
		require.EqualValues(s.T(), uint64(uploadTestSellerID), *row.SellerID)
	})

	// Case 2: Seller2 has NO seller configs => fallback to platform default.
	s.Run("SellerWithNoConfigs_FallsBackToPlatformDefault", func() {
		seller2ID := uint64(s.lookupUserIDByEmail(helpers.Seller2Email))
		// Ensure seller2 has no configs at all (IDs are not fixed across seeds).
		require.NoError(s.T(), s.deleteStorageConfigsForSellerOwner(seller2ID))

		client := helpers.NewAPIClient(s.server)
		client.SetToken(s.seller2Token)
		client.SetHeader(constants.CORRELATION_ID_HEADER, "res-seller2-fallback")

		w := client.Post(s.T(), uploadInitEndpoint, initOutageRequest("seller2-fallback.jpg"))
		resp := helpers.AssertSuccessResponse(s.T(), w, http.StatusCreated)

		fileID := resp["data"].(map[string]any)["fileId"].(string)
		row := s.getFileObjectRowByFileID(fileID)

		require.Equal(s.T(), platformBucket, row.BucketOrContainer)
	})

	// Case 3: Seller2 has an active seller config, but NOT default => still fallback to platform default.
	s.Run("SellerWithActiveNonDefault_FallsBackToPlatformDefault", func() {
		// Resolve seller2 numeric id from jwt-backed principal by reading seeded user id.
		seller2ID := s.lookupUserIDByEmail(helpers.Seller2Email)

		restore := s.setSellerS3Config(uint64(seller2ID), seller2NonDefaultBucket, true, false)
		defer restore()

		client := helpers.NewAPIClient(s.server)
		client.SetToken(s.seller2Token)
		client.SetHeader(constants.CORRELATION_ID_HEADER, "res-seller2-nondefault")

		w := client.Post(s.T(), uploadInitEndpoint, initOutageRequest("seller2-nondefault.jpg"))
		resp := helpers.AssertSuccessResponse(s.T(), w, http.StatusCreated)

		fileID := resp["data"].(map[string]any)["fileId"].(string)
		row := s.getFileObjectRowByFileID(fileID)

		require.Equal(s.T(), platformBucket, row.BucketOrContainer)
	})

	// Case 4: Seller2 has a default config but inactive => fallback to platform default.
	s.Run("SellerWithInactiveDefault_FallsBackToPlatformDefault", func() {
		seller2ID := s.lookupUserIDByEmail(helpers.Seller2Email)
		restore := s.setSellerS3Config(uint64(seller2ID), seller2DefaultBucket, false, true)
		defer restore()

		client := helpers.NewAPIClient(s.server)
		client.SetToken(s.seller2Token)
		client.SetHeader(constants.CORRELATION_ID_HEADER, "res-seller2-inactive-default")

		w := client.Post(s.T(), uploadInitEndpoint, initOutageRequest("seller2-inactive-default.jpg"))
		resp := helpers.AssertSuccessResponse(s.T(), w, http.StatusCreated)

		fileID := resp["data"].(map[string]any)["fileId"].(string)
		row := s.getFileObjectRowByFileID(fileID)

		require.Equal(s.T(), platformBucket, row.BucketOrContainer)
	})

	// Case 5: Seller2 has active default seller config => use seller config.
	s.Run("Seller2WithActiveDefault_UsesSellerConfig", func() {
		seller2ID := s.lookupUserIDByEmail(helpers.Seller2Email)
		restore := s.setSellerS3Config(uint64(seller2ID), seller2DefaultBucket, true, true)
		defer restore()

		client := helpers.NewAPIClient(s.server)
		client.SetToken(s.seller2Token)
		client.SetHeader(constants.CORRELATION_ID_HEADER, "res-seller2-default")

		w := client.Post(s.T(), uploadInitEndpoint, initOutageRequest("seller2-default.jpg"))
		resp := helpers.AssertSuccessResponse(s.T(), w, http.StatusCreated)

		fileID := resp["data"].(map[string]any)["fileId"].(string)
		row := s.getFileObjectRowByFileID(fileID)

		require.Equal(s.T(), seller2DefaultBucket, row.BucketOrContainer)
	})

	// Case 6: Admin upload uses platform default.
	s.Run("AdminUpload_UsesPlatformDefault", func() {
		client := helpers.NewAPIClient(s.server)
		client.SetToken(s.adminToken)
		client.SetHeader(constants.CORRELATION_ID_HEADER, "res-admin-platform")

		w := client.Post(s.T(), uploadInitEndpoint, initOutageRequest("admin-platform.jpg"))
		resp := helpers.AssertSuccessResponse(s.T(), w, http.StatusCreated)

		fileID := resp["data"].(map[string]any)["fileId"].(string)
		row := s.getFileObjectRowByFileID(fileID)

		require.Equal(s.T(), platformBucket, row.BucketOrContainer)
	})
}

type fileObjectRow struct {
	ID                uint64
	SellerID          *uint64
	BucketOrContainer string
	StorageConfigID   uint64
}

func (s *UploadSuite) getFileObjectRowByFileID(fileID string) fileObjectRow {
	var r fileObjectRow
	err := s.container.DB.Raw(
		`SELECT id, seller_id, bucket_or_container, storage_config_id FROM file_object WHERE file_id = ?`,
		fileID,
	).Scan(&r).Error
	require.NoError(s.T(), err)
	require.NotZero(s.T(), r.ID)
	require.NotZero(s.T(), r.StorageConfigID)
	require.NotEmpty(s.T(), r.BucketOrContainer)
	return r
}

func (s *UploadSuite) lookupUserIDByEmail(email string) uint {
	type row struct{ ID uint }
	var r row
	err := s.container.DB.Table(`"user"`).Select("id").Where("email = ?", email).First(&r).Error
	require.NoError(s.T(), err)
	require.NotZero(s.T(), r.ID)
	return r.ID
}

// deleteFileObjectsForStorageConfigID removes rows that FK-reference a storage_config (test cleanup).
func (s *UploadSuite) deleteFileObjectsForStorageConfigID(storageConfigID uint) error {
	if err := s.container.DB.Exec(
		`DELETE FROM file_job WHERE file_object_id IN (SELECT id FROM file_object WHERE storage_config_id = ?)`,
		storageConfigID,
	).Error; err != nil {
		return err
	}
	if err := s.container.DB.Exec(
		`DELETE FROM file_variant WHERE file_object_id IN (SELECT id FROM file_object WHERE storage_config_id = ?)`,
		storageConfigID,
	).Error; err != nil {
		return err
	}
	return s.container.DB.Exec(`DELETE FROM file_object WHERE storage_config_id = ?`, storageConfigID).Error
}

func (s *UploadSuite) deleteStorageConfigsForSellerOwner(sellerOwnerID uint64) error {
	var ids []uint
	if err := s.container.DB.Table("storage_config").
		Where("owner_type = ? AND owner_id = ?", entity.OwnerTypeSeller, sellerOwnerID).
		Pluck("id", &ids).Error; err != nil {
		return err
	}
	for _, id := range ids {
		if err := s.deleteFileObjectsForStorageConfigID(id); err != nil {
			return err
		}
		if err := s.container.DB.Exec(`DELETE FROM storage_config WHERE id = ?`, id).Error; err != nil {
			return err
		}
	}
	return nil
}

func (s *UploadSuite) providerIDForS3Compatible() uint {
	type row struct{ ID uint }
	var r row
	err := s.container.DB.
		Table("storage_provider").
		Select("id").
		Where("adapter_type = ? AND is_active = ?", "s3_compatible", true).
		Order("id ASC").
		First(&r).Error
	require.NoError(s.T(), err)
	require.NotZero(s.T(), r.ID)
	return r.ID
}

func (s *UploadSuite) encryptedS3ConfigJSON(bucket, accessKey, secretKey string, forcePathStyle bool) []byte {
	raw := map[string]any{
		"access_key_id":     accessKey,
		"secret_access_key": secretKey,
		"bucket":            bucket,
		"region":            s.minio.Region,
		"endpoint":          s.minio.Endpoint,
		"force_path_style":  forcePathStyle,
	}
	parser, err := blobAdapter.GetBlobConfigParser(entity.AdapterTypeS3Compatible)
	require.NoError(s.T(), err)
	blobCfg, err := parser.ParseAndValidateConfig(raw)
	require.NoError(s.T(), err)
	require.NoError(s.T(), blobCfg.Encrypt())
	b, err := json.Marshal(blobCfg.ToMap())
	require.NoError(s.T(), err)
	return b
}

func (s *UploadSuite) setPlatformDefaultS3Config(bucket string) func() {
	type row struct {
		ID                uint
		BucketOrContainer string
		ConfigData        []byte
		IsActive          bool
		IsDefault         bool
	}
	var existing row
	err := s.container.DB.Raw(
		`SELECT id, bucket_or_container, config_data, is_active, is_default
		 FROM storage_config
		 WHERE owner_type = 'PLATFORM' AND is_default = true
		 ORDER BY updated_at DESC
		 LIMIT 1`,
	).Scan(&existing).Error
	require.NoError(s.T(), err)
	require.NotZero(s.T(), existing.ID)

	enc := s.encryptedS3ConfigJSON(bucket, s.minio.AccessKey, s.minio.SecretKey, true)
	providerID := s.providerIDForS3Compatible()
	adminID := s.lookupUserIDByEmail(helpers.AdminEmail)

	require.NoError(s.T(), s.container.DB.Exec(
		`UPDATE storage_config
		 SET owner_id = ?, provider_id = ?, bucket_or_container = ?, config_data = ?, is_active = true, is_default = true, updated_at = NOW()
		 WHERE id = ?`,
		adminID, providerID, bucket, enc, existing.ID,
	).Error)

	return func() {
		_ = s.container.DB.Exec(
			`UPDATE storage_config
			 SET bucket_or_container = ?, config_data = ?, is_active = ?, is_default = ?, updated_at = NOW()
			 WHERE id = ?`,
			existing.BucketOrContainer, existing.ConfigData, existing.IsActive, existing.IsDefault, existing.ID,
		).Error
	}
}

func (s *UploadSuite) setSellerS3Config(sellerID uint64, bucket string, isActive bool, isDefault bool) func() {
	providerID := s.providerIDForS3Compatible()
	enc := s.encryptedS3ConfigJSON(bucket, s.minio.AccessKey, s.minio.SecretKey, true)

	// Clear any existing defaults for this seller so "active default" is deterministic.
	_ = s.container.DB.Exec(
		`UPDATE storage_config SET is_default = false WHERE owner_type = 'SELLER' AND owner_id = ?`,
		sellerID,
	).Error

	var id uint
	err := s.container.DB.Raw(
		`INSERT INTO storage_config (
			owner_type, owner_id, provider_id, display_name, bucket_or_container,
			config_data, is_default, is_active, created_at, updated_at
		) VALUES (
			'SELLER', ?, ?, ?, ?,
			?, ?, ?, NOW(), NOW()
		) RETURNING id`,
		sellerID, providerID, "Resolution Test Config", bucket, enc, isDefault, isActive,
	).Scan(&id).Error
	require.NoError(s.T(), err)
	require.NotZero(s.T(), id)

	return func() {
		require.NoError(s.T(), s.deleteFileObjectsForStorageConfigID(id))
		require.NoError(s.T(), s.container.DB.Exec(`DELETE FROM storage_config WHERE id = ?`, id).Error)
	}
}
