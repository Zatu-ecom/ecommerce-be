package file_test

import (
	"net/http"
	"strings"

	"ecommerce-be/common/constants"
	"ecommerce-be/file/entity"
	"ecommerce-be/test/integration/helpers"

	"github.com/stretchr/testify/require"
)

func (s *UploadSuite) TestGetFile_HappyPathWithAndWithoutDownloadURL() {
	fileID := s.createUploadedFile(
		s.sellerToken,
		map[string]any{
			"purpose":    "PRODUCT_IMAGE",
			"visibility": "PRIVATE",
			"filename":   "single-file.jpg",
			"mimeType":   "image/jpeg",
			"sizeBytes":  3072,
		},
		make([]byte, 3072),
	)
	s.insertVariant(fileID, "thumb_200", "image/webp", 700, "READY")

	client := helpers.NewAPIClient(s.server)
	client.SetToken(s.sellerToken)
	client.SetHeader(constants.CORRELATION_ID_HEADER, "get-file-happy")

	wPlain := client.Get(s.T(), "/api/file/"+fileID)
	respPlain := helpers.AssertSuccessResponse(s.T(), wPlain, http.StatusOK)
	dataPlain := respPlain["data"].(map[string]any)
	require.Equal(s.T(), fileID, dataPlain["fileId"])
	require.Equal(s.T(), "ACTIVE", dataPlain["status"])
	require.NotContains(s.T(), dataPlain, "downloadUrl")
	require.Len(s.T(), dataPlain["variants"].([]any), 1)

	wDownload := client.Get(s.T(), "/api/file/"+fileID+"?includeDownloadUrl=true&urlTtlMinutes=15")
	respDownload := helpers.AssertSuccessResponse(s.T(), wDownload, http.StatusOK)
	dataDownload := respDownload["data"].(map[string]any)
	require.Equal(s.T(), fileID, dataDownload["fileId"])
	require.NotEmpty(s.T(), dataDownload["downloadUrl"])
	require.NotEmpty(s.T(), dataDownload["downloadUrlExpiresAt"])
	require.Contains(s.T(), dataDownload["downloadUrl"].(string), "X-Amz-")
}

func (s *UploadSuite) TestGetFile_PublicFileDirectURL() {
	fileID := s.createUploadedFile(
		s.sellerToken,
		map[string]any{
			"purpose":    "PRODUCT_IMAGE",
			"visibility": "PUBLIC",
			"filename":   "public-image.jpg",
			"mimeType":   "image/jpeg",
			"sizeBytes":  2048,
		},
		make([]byte, 2048),
	)

	client := helpers.NewAPIClient(s.server)
	client.SetToken(s.sellerToken)
	client.SetHeader(constants.CORRELATION_ID_HEADER, "get-file-public")

	wDownload := client.Get(s.T(), "/api/file/"+fileID+"?includeDownloadUrl=true")
	respDownload := helpers.AssertSuccessResponse(s.T(), wDownload, http.StatusOK)
	dataDownload := respDownload["data"].(map[string]any)
	require.Equal(s.T(), fileID, dataDownload["fileId"])
	require.NotEmpty(s.T(), dataDownload["downloadUrl"])
	require.NotContains(s.T(), dataDownload, "downloadUrlExpiresAt")
	require.NotContains(s.T(), dataDownload["downloadUrl"].(string), "X-Amz-")
}

func (s *UploadSuite) TestGetFile_TenantIsolationAndNotFound() {
	fileID := s.createUploadedFile(
		s.sellerToken,
		map[string]any{
			"purpose":    "DOCUMENT",
			"visibility": "PRIVATE",
			"filename":   "tenant-only.pdf",
			"mimeType":   "application/pdf",
			"sizeBytes":  1100,
		},
		make([]byte, 1100),
	)

	seller2Client := helpers.NewAPIClient(s.server)
	seller2Client.SetToken(s.seller2Token)
	seller2Client.SetHeader(constants.CORRELATION_ID_HEADER, "get-file-other-seller")
	wCrossTenant := seller2Client.Get(s.T(), "/api/file/"+fileID)
	respCrossTenant := helpers.AssertErrorResponse(s.T(), wCrossTenant, http.StatusNotFound)
	require.Equal(s.T(), entity.FileStatusActive, entity.FileStatus("ACTIVE"))
	require.Equal(s.T(), "FILE_NOT_FOUND", respCrossTenant["code"])

	client := helpers.NewAPIClient(s.server)
	client.SetToken(s.sellerToken)
	client.SetHeader(constants.CORRELATION_ID_HEADER, "get-file-missing")
	wMissing := client.Get(s.T(), "/api/file/does-not-exist")
	respMissing := helpers.AssertErrorResponse(s.T(), wMissing, http.StatusNotFound)
	require.Equal(s.T(), "FILE_NOT_FOUND", respMissing["code"])
}

func (s *UploadSuite) TestGetFile_ValidationAndRoleBehavior() {
	fileID := s.createUploadedFile(
		s.sellerToken,
		map[string]any{
			"purpose":    "DOCUMENT",
			"visibility": "PRIVATE",
			"filename":   "validate.pdf",
			"mimeType":   "application/pdf",
			"sizeBytes":  1000,
		},
		make([]byte, 1000),
	)

	client := helpers.NewAPIClient(s.server)
	client.SetToken(s.sellerToken)
	client.SetHeader(constants.CORRELATION_ID_HEADER, "get-file-validation")
	wBadTTL := client.Get(s.T(), "/api/file/"+fileID+"?includeDownloadUrl=true&urlTtlMinutes=1")
	respBadTTL := helpers.AssertErrorResponse(s.T(), wBadTTL, http.StatusBadRequest)
	require.Equal(s.T(), constants.VALIDATION_ERROR_CODE, respBadTTL["code"])

	customerClient := helpers.NewAPIClient(s.server)
	customerClient.SetToken(s.customerToken)
	customerClient.SetHeader(constants.CORRELATION_ID_HEADER, "get-file-customer")
	wForbidden := customerClient.Get(s.T(), "/api/file/"+fileID)
	helpers.AssertErrorResponse(s.T(), wForbidden, http.StatusForbidden)

	noCorr := helpers.NewAPIClient(s.server)
	noCorr.SetToken(s.sellerToken)
	noCorr.SetHeader(constants.CORRELATION_ID_HEADER, "")
	wMissingCorr := noCorr.Get(s.T(), "/api/file/"+fileID)
	respMissingCorr := helpers.AssertErrorResponse(s.T(), wMissingCorr, http.StatusBadRequest)
	require.True(s.T(), strings.Contains(respMissingCorr["message"].(string), "Correlation"))
}
