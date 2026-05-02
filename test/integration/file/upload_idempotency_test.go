package file_test

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"ecommerce-be/common/constants"
	"ecommerce-be/file/entity"
	"ecommerce-be/test/integration/helpers"

	"github.com/stretchr/testify/require"
)

func (s *UploadSuite) TestInitUpload_Idempotency() {
	s.Run("SameKey_SameBody_ReturnsSameFileId", func() {
		client := helpers.NewAPIClient(s.server)
		client.SetToken(s.sellerToken)
		client.SetHeader(constants.CORRELATION_ID_HEADER, "us5a-same-key-corr")
		client.SetHeader("Idempotency-Key", "same-key-123")

		req := map[string]any{
			"purpose":             "PRODUCT_IMAGE",
			"visibility":          "PRIVATE",
			"filename":            "idempotent-same.jpg",
			"mimeType":            "image/jpeg",
			"sizeBytes":           2048,
			"uploadExpiryMinutes": 15,
		}

		first := helpers.AssertSuccessResponse(
			s.T(),
			client.Post(s.T(), uploadInitEndpoint, req),
			http.StatusCreated,
		)
		second := helpers.AssertSuccessResponse(
			s.T(),
			client.Post(s.T(), uploadInitEndpoint, req),
			http.StatusOK,
		)

		firstData := first["data"].(map[string]any)
		secondData := second["data"].(map[string]any)
		fileID := firstData["fileId"].(string)
		require.Equal(s.T(), fileID, secondData["fileId"])
		require.Equal(s.T(), firstData["objectKey"], secondData["objectKey"])

		var rows int64
		err := s.container.DB.
			Table("file_object").
			Where("file_id = ?", fileID).
			Count(&rows).Error
		require.NoError(s.T(), err)
		require.EqualValues(s.T(), 1, rows)

		var r struct{ ID uint64 }
		err = s.container.DB.Raw("SELECT id FROM file_object WHERE file_id = ?", fileID).
			Scan(&r).
			Error
		require.NoError(s.T(), err)
		helpers.AssertSchedulerJobExists(s.T(), s.container.RedisClient, r.ID)

		keys, err := s.container.RedisClient.Keys(context.Background(), "file:init:idem:*").Result()
		require.NoError(s.T(), err)
		require.NotEmpty(s.T(), keys)
	})

	s.Run("SameKey_AfterUploadUrlExpired_ReissuesUrl", func() {
		client := helpers.NewAPIClient(s.server)
		client.SetToken(s.sellerToken)
		client.SetHeader(constants.CORRELATION_ID_HEADER, "us5a-reissue-corr")
		client.SetHeader("Idempotency-Key", "reissue-key-123")

		req := map[string]any{
			"purpose":             "PRODUCT_IMAGE",
			"visibility":          "PRIVATE",
			"filename":            "idempotent-reissue.jpg",
			"mimeType":            "image/jpeg",
			"sizeBytes":           2048,
			"uploadExpiryMinutes": 15,
		}

		first := helpers.AssertSuccessResponse(
			s.T(),
			client.Post(s.T(), uploadInitEndpoint, req),
			http.StatusCreated,
		)
		firstData := first["data"].(map[string]any)
		fileID := firstData["fileId"].(string)

		key := s.idempotencyRedisKeyForFileID(fileID)
		raw, err := s.container.RedisClient.Get(context.Background(), key).Bytes()
		require.NoError(s.T(), err)

		record := map[string]any{}
		require.NoError(s.T(), json.Unmarshal(raw, &record))
		record["expiresAt"] = time.Now().Add(-time.Minute).UTC().Format(time.RFC3339)
		updated, err := json.Marshal(record)
		require.NoError(s.T(), err)
		require.NoError(
			s.T(),
			s.container.RedisClient.Set(context.Background(), key, updated, 20*time.Minute).Err(),
		)

		second := helpers.AssertSuccessResponse(
			s.T(),
			client.Post(s.T(), uploadInitEndpoint, req),
			http.StatusOK,
		)
		secondData := second["data"].(map[string]any)
		require.Equal(s.T(), fileID, secondData["fileId"])
		require.NotEqual(s.T(), record["expiresAt"], secondData["expiresAt"])
	})

	s.Run("SameKey_AfterActive_Returns409Conflict", func() {
		client := helpers.NewAPIClient(s.server)
		client.SetToken(s.sellerToken)
		client.SetHeader(constants.CORRELATION_ID_HEADER, "us5a-active-conflict-init")
		client.SetHeader("Idempotency-Key", "active-key-123")

		req := map[string]any{
			"purpose":             "PRODUCT_IMAGE",
			"visibility":          "PRIVATE",
			"filename":            "idempotent-active.jpg",
			"mimeType":            "image/jpeg",
			"sizeBytes":           2048,
			"uploadExpiryMinutes": 15,
		}

		initResp := helpers.AssertSuccessResponse(
			s.T(),
			client.Post(s.T(), uploadInitEndpoint, req),
			http.StatusCreated,
		)
		initData := initResp["data"].(map[string]any)
		fileID := initData["fileId"].(string)

		uploadHelper := helpers.UploadHelper{Server: s.server, Token: s.sellerToken}
		uploadHelper.PutBytes(s.T(), initData, make([]byte, 2048))

		client.SetHeader(constants.CORRELATION_ID_HEADER, "us5a-active-conflict-complete")
		client.SetHeader("Idempotency-Key", "")
		helpers.AssertSuccessResponse(
			s.T(),
			client.Post(s.T(), uploadCompleteEndpoint, map[string]any{"fileId": fileID}),
			http.StatusOK,
		)
		_ = s.nextVariantMessage(3 * time.Second)

		client.SetHeader(constants.CORRELATION_ID_HEADER, "us5a-active-conflict-retry")
		client.SetHeader("Idempotency-Key", "active-key-123")
		resp := helpers.AssertErrorResponse(
			s.T(),
			client.Post(s.T(), uploadInitEndpoint, req),
			http.StatusConflict,
		)
		require.Equal(s.T(), "FILE_UPLOAD_CONFLICT", resp["code"])
		s.assertFileStatus(fileID, entity.FileStatusActive)
	})

	s.Run("MalformedKey_Returns400", func() {
		client := helpers.NewAPIClient(s.server)
		client.SetToken(s.sellerToken)
		client.SetHeader(constants.CORRELATION_ID_HEADER, "us5a-malformed-key")
		client.SetHeader("Idempotency-Key", "bad key")

		w := client.Post(s.T(), uploadInitEndpoint, map[string]any{
			"purpose":             "PRODUCT_IMAGE",
			"visibility":          "PRIVATE",
			"filename":            "bad-key.jpg",
			"mimeType":            "image/jpeg",
			"sizeBytes":           1024,
			"uploadExpiryMinutes": 15,
		})
		resp := helpers.AssertErrorResponse(s.T(), w, http.StatusBadRequest)
		require.Equal(s.T(), "VALIDATION_ERROR", resp["code"])

		errors, ok := resp["errors"].([]any)
		require.True(s.T(), ok)
		require.NotEmpty(s.T(), errors)
		first := errors[0].(map[string]any)
		require.Equal(s.T(), "Idempotency-Key", first["field"])
	})

	s.Run("DifferentSellers_SameKey_Distinct", func() {
		req := map[string]any{
			"purpose":             "DOCUMENT",
			"visibility":          "PRIVATE",
			"filename":            "same-key-different-sellers.pdf",
			"mimeType":            "application/pdf",
			"sizeBytes":           2048,
			"uploadExpiryMinutes": 15,
		}

		seller1 := helpers.NewAPIClient(s.server)
		seller1.SetToken(s.sellerToken)
		seller1.SetHeader(constants.CORRELATION_ID_HEADER, "us5a-seller1")
		seller1.SetHeader("Idempotency-Key", "shared-key-123")

		seller2 := helpers.NewAPIClient(s.server)
		seller2.SetToken(s.seller2Token)
		seller2.SetHeader(constants.CORRELATION_ID_HEADER, "us5a-seller2")
		seller2.SetHeader("Idempotency-Key", "shared-key-123")

		resp1 := helpers.AssertSuccessResponse(
			s.T(),
			seller1.Post(s.T(), uploadInitEndpoint, req),
			http.StatusCreated,
		)
		resp2 := helpers.AssertSuccessResponse(
			s.T(),
			seller2.Post(s.T(), uploadInitEndpoint, req),
			http.StatusCreated,
		)

		fileID1 := resp1["data"].(map[string]any)["fileId"].(string)
		fileID2 := resp2["data"].(map[string]any)["fileId"].(string)
		require.NotEqual(s.T(), fileID1, fileID2)
	})
}

func (s *UploadSuite) idempotencyRedisKeyForFileID(fileID string) string {
	keys, err := s.container.RedisClient.Keys(context.Background(), "file:init:idem:*").Result()
	s.Require().NoError(err)
	for _, key := range keys {
		raw, err := s.container.RedisClient.Get(context.Background(), key).Bytes()
		s.Require().NoError(err)
		record := map[string]any{}
		s.Require().NoError(json.Unmarshal(raw, &record))
		if record["fileId"] == fileID {
			return key
		}
	}
	s.T().Fatalf("idempotency key for fileID %s not found", fileID)
	return ""
}
