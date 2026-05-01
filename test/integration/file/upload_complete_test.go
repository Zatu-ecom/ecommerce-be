package file_test

import (
	"net/http"
	"time"

	"ecommerce-be/common/constants"
	"ecommerce-be/file/entity"
	"ecommerce-be/test/integration/helpers"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func (s *UploadSuite) TestCompleteUpload_ProductImage_HappyPath() {
	client := helpers.NewAPIClient(s.server)
	client.SetToken(s.sellerToken)
	correlationID := "upload-complete-happy-correlation-id"
	client.SetHeader(constants.CORRELATION_ID_HEADER, correlationID)

	initReq := map[string]interface{}{
		"purpose":             "PRODUCT_IMAGE",
		"visibility":          "PRIVATE",
		"filename":            "hero-complete.jpg",
		"mimeType":            "image/jpeg",
		"sizeBytes":           2048,
		"uploadExpiryMinutes": 15,
	}

	initW := client.Post(s.T(), uploadInitEndpoint, initReq)
	initResp := helpers.AssertSuccessResponse(s.T(), initW, http.StatusCreated)
	initData := initResp["data"].(map[string]interface{})
	fileID := initData["fileId"].(string)

	uploadHelper := helpers.UploadHelper{Server: s.server, Token: s.sellerToken}
	uploadHelper.PutBytes(s.T(), initData, make([]byte, 2048))

	client.SetHeader(constants.CORRELATION_ID_HEADER, correlationID)
	completeW := client.Post(s.T(), uploadCompleteEndpoint, map[string]interface{}{
		"fileId": fileID,
	})
	completeResp := helpers.AssertSuccessResponse(s.T(), completeW, http.StatusOK)
	completeData := completeResp["data"].(map[string]interface{})

	require.Equal(s.T(), "ACTIVE", completeData["status"])
	require.Equal(s.T(), true, completeData["variantsQueued"])

	s.assertFileStatus(fileID, entity.FileStatusActive)

	type row struct {
		ID uint64
	}
	var r row
	err := s.container.DB.Raw("SELECT id FROM file_object WHERE file_id = ?", fileID).Scan(&r).Error
	require.NoError(s.T(), err)
	helpers.AssertNoSchedulerJob(s.T(), s.container.RedisClient, r.ID)

	type jobRow struct {
		Status string
	}
	var job jobRow
	err = s.container.DB.Raw(
		"SELECT status FROM file_job WHERE file_object_id = ? ORDER BY created_at DESC LIMIT 1",
		r.ID,
	).Scan(&job).Error
	require.NoError(s.T(), err)
	require.Equal(s.T(), "PUBLISHED", job.Status)

	msg := s.nextVariantMessage(3 * time.Second)
	require.Equal(s.T(), correlationID, msg.Envelope.CorrelationID)
	require.Equal(s.T(), fileID, msg.Payload["fileId"])
}

func (s *UploadSuite) TestUploadHandler_ValidationAndCorrelationBehavior() {
	client := helpers.NewAPIClient(s.server)
	client.SetToken(s.sellerToken)
	client.SetHeader(constants.CORRELATION_ID_HEADER, uuid.NewString())

	// Unknown field -> 400 VALIDATION_ERROR.
	wUnknown := client.PostRaw(s.T(), uploadInitEndpoint, []byte(`{
		"purpose":"PRODUCT_IMAGE",
		"visibility":"PRIVATE",
		"filename":"unknown.jpg",
		"mimeType":"image/jpeg",
		"sizeBytes":100,
		"unknownField":"x"
	}`))
	respUnknown := helpers.AssertErrorResponse(s.T(), wUnknown, http.StatusBadRequest)
	require.Equal(s.T(), constants.VALIDATION_ERROR_CODE, respUnknown["code"])

	// Missing correlation id -> 400.
	client.SetHeader(constants.CORRELATION_ID_HEADER, "")
	wNoCorr := client.Post(s.T(), uploadInitEndpoint, map[string]interface{}{
		"purpose":    "PRODUCT_IMAGE",
		"filename":   "missing-corr.jpg",
		"mimeType":   "image/jpeg",
		"sizeBytes":  100,
		"visibility": "PRIVATE",
	})
	helpers.AssertErrorResponse(s.T(), wNoCorr, http.StatusBadRequest)
}

func (s *UploadSuite) TestCompleteUpload_Document_PDF_NoVariants() {
	client := helpers.NewAPIClient(s.server)
	client.SetToken(s.sellerToken)
	client.SetHeader(constants.CORRELATION_ID_HEADER, "upload-complete-document-correlation-id")

	initReq := map[string]interface{}{
		"purpose":             "DOCUMENT",
		"visibility":          "PRIVATE",
		"filename":            "policy.pdf",
		"mimeType":            "application/pdf",
		"sizeBytes":           1024,
		"uploadExpiryMinutes": 15,
	}

	initW := client.Post(s.T(), uploadInitEndpoint, initReq)
	initResp := helpers.AssertSuccessResponse(s.T(), initW, http.StatusCreated)
	initData := initResp["data"].(map[string]interface{})
	fileID := initData["fileId"].(string)

	uploadHelper := helpers.UploadHelper{Server: s.server, Token: s.sellerToken}
	uploadHelper.PutBytes(s.T(), initData, make([]byte, 1024))

	completeW := client.Post(s.T(), uploadCompleteEndpoint, map[string]interface{}{
		"fileId": fileID,
	})
	completeResp := helpers.AssertSuccessResponse(s.T(), completeW, http.StatusOK)
	completeData := completeResp["data"].(map[string]interface{})

	require.Equal(s.T(), "ACTIVE", completeData["status"])
	require.Equal(s.T(), false, completeData["variantsQueued"])
	s.assertNoVariantMessage(500 * time.Millisecond)

	type row struct {
		ID uint64
	}
	var r row
	err := s.container.DB.Raw("SELECT id FROM file_object WHERE file_id = ?", fileID).Scan(&r).Error
	require.NoError(s.T(), err)

	var fileJobCount int64
	err = s.container.DB.Raw(
		"SELECT COUNT(1) FROM file_job WHERE file_object_id = ?",
		r.ID,
	).Scan(&fileJobCount).Error
	require.NoError(s.T(), err)
	require.Equal(s.T(), int64(0), fileJobCount)
}

func (s *UploadSuite) TestCompleteUpload_ImportFile_CSV_NoVariants() {
	client := helpers.NewAPIClient(s.server)
	client.SetToken(s.sellerToken)
	client.SetHeader(constants.CORRELATION_ID_HEADER, "upload-complete-import-correlation-id")

	initReq := map[string]interface{}{
		"purpose":             "IMPORT_FILE",
		"visibility":          "PRIVATE",
		"filename":            "bulk.csv",
		"mimeType":            "text/csv",
		"sizeBytes":           1536,
		"uploadExpiryMinutes": 15,
	}

	initW := client.Post(s.T(), uploadInitEndpoint, initReq)
	initResp := helpers.AssertSuccessResponse(s.T(), initW, http.StatusCreated)
	initData := initResp["data"].(map[string]interface{})
	fileID := initData["fileId"].(string)

	uploadHelper := helpers.UploadHelper{Server: s.server, Token: s.sellerToken}
	uploadHelper.PutBytes(s.T(), initData, make([]byte, 1536))

	completeW := client.Post(s.T(), uploadCompleteEndpoint, map[string]interface{}{
		"fileId": fileID,
	})
	completeResp := helpers.AssertSuccessResponse(s.T(), completeW, http.StatusOK)
	completeData := completeResp["data"].(map[string]interface{})

	require.Equal(s.T(), "ACTIVE", completeData["status"])
	require.Equal(s.T(), false, completeData["variantsQueued"])
	s.assertNoVariantMessage(500 * time.Millisecond)

	type row struct {
		ID uint64
	}
	var r row
	err := s.container.DB.Raw("SELECT id FROM file_object WHERE file_id = ?", fileID).Scan(&r).Error
	require.NoError(s.T(), err)

	var fileJobCount int64
	err = s.container.DB.Raw(
		"SELECT COUNT(1) FROM file_job WHERE file_object_id = ?",
		r.ID,
	).Scan(&fileJobCount).Error
	require.NoError(s.T(), err)
	require.Equal(s.T(), int64(0), fileJobCount)
}
