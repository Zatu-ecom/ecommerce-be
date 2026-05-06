package file_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"ecommerce-be/common/constants"
	"ecommerce-be/common/scheduler"
	"ecommerce-be/file/entity"
	fileSingleton "ecommerce-be/file/factory/singleton"
	"ecommerce-be/file/service/blobAdapter"
	"ecommerce-be/test/integration/helpers"
	"ecommerce-be/test/integration/setup"

	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/require"
)

func (s *UploadSuite) TestInitUpload_StorageOutage() {
	s.Run("NonexistentBucket_Returns503_NoDbRow", func() {
		restore := s.bindSellerStorageConfig(
			"Outage Missing Bucket",
			"missing-upload-suite-bucket",
			s.minio.AccessKey,
			s.minio.SecretKey,
			true,
		)
		defer restore()

		client := helpers.NewAPIClient(s.server)
		client.SetToken(s.sellerToken)
		client.SetHeader(constants.CORRELATION_ID_HEADER, "us6-missing-bucket")

		filename := "missing-bucket-outage.jpg"
		w := client.Post(s.T(), uploadInitEndpoint, initOutageRequest(filename))
		resp := helpers.AssertErrorResponse(s.T(), w, http.StatusServiceUnavailable)
		require.Equal(s.T(), "FILE_UPLOAD_STORAGE_UNAVAILABLE", resp["code"])
		requireNoUploadRowForFilename(s.T(), s, filename)
	})

	s.Run("InvalidCredentials_Returns503_NoSecretInMessage", func() {
		secret := "definitely-wrong-secret"
		restore := s.bindSellerStorageConfig(
			"Outage Invalid Credentials",
			uploadTestBucket,
			"wrong-access-key",
			secret,
			true,
		)
		defer restore()

		client := helpers.NewAPIClient(s.server)
		client.SetToken(s.sellerToken)
		client.SetHeader(constants.CORRELATION_ID_HEADER, "us6-invalid-creds")

		filename := "invalid-credentials-outage.jpg"
		w := client.Post(s.T(), uploadInitEndpoint, initOutageRequest(filename))
		body := w.Body.String()
		resp := helpers.AssertErrorResponse(s.T(), w, http.StatusServiceUnavailable)
		require.Equal(s.T(), "FILE_UPLOAD_STORAGE_UNAVAILABLE", resp["code"])
		require.NotContains(s.T(), body, secret)
		require.NotContains(s.T(), body, "wrong-access-key")
		requireNoUploadRowForFilename(s.T(), s, filename)
	})

	s.Run("NoStorageConfig_Returns412_NoDbRow", func() {
		restore := s.disableStorageResolution()
		defer restore()

		client := helpers.NewAPIClient(s.server)
		client.SetToken(s.seller2Token)
		client.SetHeader(constants.CORRELATION_ID_HEADER, "us6-no-storage-config")

		filename := "no-storage-config-outage.jpg"
		w := client.Post(s.T(), uploadInitEndpoint, initOutageRequest(filename))
		resp := helpers.AssertErrorResponse(s.T(), w, http.StatusPreconditionFailed)
		require.Equal(s.T(), "FILE_UPLOAD_NO_STORAGE_CONFIG", resp["code"])
		requireNoUploadRowForFilename(s.T(), s, filename)
	})

	s.Run("RedisUnavailable_InitUpload_Returns503_NoDbRow", func() {
		client := helpers.NewAPIClient(s.server)
		client.SetToken(s.sellerToken)
		client.SetHeader(constants.CORRELATION_ID_HEADER, "us6-redis-down")

		filename := "redis-unavailable-outage.jpg"
		ctx := context.Background()
		stopTimeout := 10 * time.Second
		require.NoError(s.T(), s.container.Redis.Stop(ctx, &stopTimeout))
		defer s.restartRedisAfterOutage(ctx)

		w := client.Post(s.T(), uploadInitEndpoint, initOutageRequest(filename))
		resp := helpers.AssertErrorResponse(s.T(), w, http.StatusServiceUnavailable)
		require.Equal(s.T(), "FILE_UPLOAD_STORAGE_UNAVAILABLE", resp["code"])
		requireNoUploadRowForFilename(s.T(), s, filename)
	})
}

func initOutageRequest(filename string) map[string]any {
	return map[string]any{
		"purpose":             "PRODUCT_IMAGE",
		"visibility":          "PRIVATE",
		"filename":            filename,
		"mimeType":            "image/jpeg",
		"sizeBytes":           1024,
		"uploadExpiryMinutes": 15,
	}
}

func (s *UploadSuite) restartRedisAfterOutage(ctx context.Context) {
	require.NoError(s.T(), s.container.Redis.Start(ctx))

	host, err := s.container.Redis.Host(ctx)
	require.NoError(s.T(), err)
	port, err := s.container.Redis.MappedPort(ctx, "6379")
	require.NoError(s.T(), err)

	_ = s.container.RedisClient.Close()
	s.container.RedisClient = redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%s:%s", host, port.Port()),
	})

	require.Eventually(s.T(), func() bool {
		return s.container.RedisClient.Ping(ctx).Err() == nil
	}, 10*time.Second, 100*time.Millisecond)

	fileSingleton.ResetInstance()
	s.server = setup.SetupTestServer(s.T(), s.container.DB, s.container.RedisClient)

	schedulerOnce = sync.Once{}
	schedulerOnce.Do(func() {
		go scheduler.StartRedisWorkerPool()
	})
}

func requireNoUploadRowForFilename(t require.TestingT, s *UploadSuite, filename string) {
	var count int64
	err := s.container.DB.
		Table("file_object").
		Where("original_filename = ?", filename).
		Count(&count).
		Error
	require.NoError(t, err)
	require.EqualValues(t, 0, count)
}

func (s *UploadSuite) bindSellerStorageConfig(
	displayName string,
	bucket string,
	accessKey string,
	secretKey string,
	forcePathStyle bool,
) func() {
	var previousDefaultIDs []uint
	err := s.container.DB.
		Table("storage_config").
		Where("owner_type = ? AND owner_id = ? AND is_default = ?", entity.OwnerTypeSeller, uploadTestSellerID, true).
		Pluck("id", &previousDefaultIDs).
		Error
	s.Require().NoError(err)

	var provider struct{ ID uint }
	err = s.container.DB.
		Table("storage_provider").
		Select("id").
		Where("adapter_type = ? AND is_active = ?", "s3_compatible", true).
		Order("id ASC").
		First(&provider).Error
	s.Require().NoError(err)

	// Match production: typed S3 map, field-level Encrypt(), JSON for config_data (see seedUploadStorageConfig).
	raw := map[string]any{
		"access_key_id":     accessKey,
		"secret_access_key": secretKey,
		"bucket":            bucket,
		"region":            s.minio.Region,
		"endpoint":          s.minio.Endpoint,
		"force_path_style":  forcePathStyle,
	}
	parser, err := blobAdapter.GetBlobConfigParser(entity.AdapterTypeS3Compatible)
	s.Require().NoError(err)
	blobCfg, err := parser.ParseAndValidateConfig(raw)
	s.Require().NoError(err)
	s.Require().NoError(blobCfg.Encrypt())
	configJSON, err := json.Marshal(blobCfg.ToMap())
	s.Require().NoError(err)

	err = s.container.DB.Exec(
		`UPDATE storage_config SET is_default = false WHERE owner_type = ? AND owner_id = ? AND is_default = true`,
		entity.OwnerTypeSeller,
		uploadTestSellerID,
	).Error
	s.Require().NoError(err)

	var configID uint
	err = s.container.DB.Raw(`
		INSERT INTO storage_config (
			owner_type, owner_id, provider_id, display_name, bucket_or_container,
			config_data, is_default, is_active, created_at, updated_at
		) VALUES (
			'SELLER', ?, ?, ?, ?,
			?, true, true, NOW(), NOW()
		) RETURNING id
	`, uploadTestSellerID, provider.ID, displayName, bucket, configJSON).
		Scan(&configID).Error
	s.Require().NoError(err)

	return func() {
		_ = s.container.DB.Exec(`DELETE FROM storage_config WHERE id = ?`, configID).Error
		if len(previousDefaultIDs) > 0 {
			_ = s.container.DB.Exec(
				`UPDATE storage_config SET is_default = true WHERE id IN ?`,
				previousDefaultIDs,
			).Error
		}
	}
}

func (s *UploadSuite) disableStorageResolution() func() {
	var platformDefaultIDs []uint
	err := s.container.DB.
		Table("storage_config").
		Where("owner_type = ? AND is_default = ? AND is_active = ?", entity.OwnerTypePlatform, true, true).
		Pluck("id", &platformDefaultIDs).
		Error
	s.Require().NoError(err)

	err = s.container.DB.Exec(
		`UPDATE storage_config SET is_active = false WHERE owner_type = ? AND is_default = true AND is_active = true`,
		entity.OwnerTypePlatform,
	).Error
	s.Require().NoError(err)

	return func() {
		if len(platformDefaultIDs) == 0 {
			return
		}
		_ = s.container.DB.Exec(
			`UPDATE storage_config SET is_active = true WHERE id IN ?`,
			platformDefaultIDs,
		).Error
	}
}
