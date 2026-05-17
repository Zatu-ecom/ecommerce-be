package file_test

import (
	"net/http"
	"strings"

	"ecommerce-be/common/constants"
	"ecommerce-be/test/integration/helpers"

	"github.com/stretchr/testify/require"
)

func (s *UploadSuite) TestGetDownloadURL_HappyPath_DefaultFileAndVariant() {
	fileID := s.createUploadedFile(
		s.sellerToken,
		map[string]any{
			"purpose":    "PRODUCT_IMAGE",
			"visibility": "PRIVATE",
			"filename":   "downloadable.jpg",
			"mimeType":   "image/jpeg",
			"sizeBytes":  4096,
		},
		make([]byte, 4096),
	)
	s.insertVariant(fileID, "thumb_200", "image/webp", 640, "READY")

	client := helpers.NewAPIClient(s.server)
	client.SetToken(s.sellerToken)
	client.SetHeader(constants.CORRELATION_ID_HEADER, "download-url-happy")

	w := client.Get(s.T(), "/api/file/"+fileID+"/download-url?ttlMinutes=10&disposition=attachment")
	resp := helpers.AssertSuccessResponse(s.T(), w, http.StatusOK)
	data := resp["data"].(map[string]any)
	require.Equal(s.T(), fileID, data["fileId"])
	require.Equal(s.T(), float64(10), data["ttlMinutes"])
	require.Equal(s.T(), "image/jpeg", data["mimeType"])
	require.NotEmpty(s.T(), data["downloadUrl"])
	require.Contains(s.T(), data["downloadUrl"].(string), "attachment")

	wVariant := client.Get(s.T(), "/api/file/"+fileID+"/download-url?variantCode=thumb_200")
	respVariant := helpers.AssertSuccessResponse(s.T(), wVariant, http.StatusOK)
	dataVariant := respVariant["data"].(map[string]any)
	require.Equal(s.T(), "thumb_200", dataVariant["variantCode"])
	require.Equal(s.T(), "image/webp", dataVariant["mimeType"])
	require.Equal(s.T(), float64(640), dataVariant["sizeBytes"])
}

func (s *UploadSuite) TestGetDownloadURL_FileAndVariantStateValidation() {
	uploadingID := s.createInitOnlyFile(
		s.sellerToken,
		map[string]any{
			"purpose":    "DOCUMENT",
			"visibility": "PRIVATE",
			"filename":   "uploading.pdf",
			"mimeType":   "application/pdf",
			"sizeBytes":  900,
		},
	)

	activeFileID := s.createUploadedFile(
		s.sellerToken,
		map[string]any{
			"purpose":    "PRODUCT_IMAGE",
			"visibility": "PRIVATE",
			"filename":   "variant-parent.jpg",
			"mimeType":   "image/jpeg",
			"sizeBytes":  2048,
		},
		make([]byte, 2048),
	)
	s.insertVariant(activeFileID, "thumb_pending", "image/webp", 320, "PENDING")

	client := helpers.NewAPIClient(s.server)
	client.SetToken(s.sellerToken)
	client.SetHeader(constants.CORRELATION_ID_HEADER, "download-url-state")

	wUploading := client.Get(s.T(), "/api/file/"+uploadingID+"/download-url")
	respUploading := helpers.AssertErrorResponse(s.T(), wUploading, http.StatusConflict)
	require.Equal(s.T(), "FILE_NOT_ACTIVE", respUploading["code"])

	wMissingVariant := client.Get(s.T(), "/api/file/"+activeFileID+"/download-url?variantCode=nope")
	respMissingVariant := helpers.AssertErrorResponse(s.T(), wMissingVariant, http.StatusNotFound)
	require.Equal(s.T(), "VARIANT_NOT_FOUND", respMissingVariant["code"])

	wPendingVariant := client.Get(s.T(), "/api/file/"+activeFileID+"/download-url?variantCode=thumb_pending")
	respPendingVariant := helpers.AssertErrorResponse(s.T(), wPendingVariant, http.StatusConflict)
	require.Equal(s.T(), "VARIANT_NOT_READY", respPendingVariant["code"])
}

func (s *UploadSuite) TestGetDownloadURL_ValidationAndTenantIsolation() {
	fileID := s.createUploadedFile(
		s.sellerToken,
		map[string]any{
			"purpose":    "DOCUMENT",
			"visibility": "PRIVATE",
			"filename":   "tenant-check.pdf",
			"mimeType":   "application/pdf",
			"sizeBytes":  1024,
		},
		make([]byte, 1024),
	)

	client := helpers.NewAPIClient(s.server)
	client.SetToken(s.sellerToken)
	client.SetHeader(constants.CORRELATION_ID_HEADER, "download-url-validation")

	wBadTTL := client.Get(s.T(), "/api/file/"+fileID+"/download-url?ttlMinutes=1")
	respBadTTL := helpers.AssertErrorResponse(s.T(), wBadTTL, http.StatusBadRequest)
	require.Equal(s.T(), constants.VALIDATION_ERROR_CODE, respBadTTL["code"])

	wBadDisposition := client.Get(s.T(), "/api/file/"+fileID+"/download-url?disposition=bad")
	respBadDisposition := helpers.AssertErrorResponse(s.T(), wBadDisposition, http.StatusBadRequest)
	require.Equal(s.T(), constants.VALIDATION_ERROR_CODE, respBadDisposition["code"])

	seller2Client := helpers.NewAPIClient(s.server)
	seller2Client.SetToken(s.seller2Token)
	seller2Client.SetHeader(constants.CORRELATION_ID_HEADER, "download-url-cross-tenant")
	wCrossTenant := seller2Client.Get(s.T(), "/api/file/"+fileID+"/download-url")
	respCrossTenant := helpers.AssertErrorResponse(s.T(), wCrossTenant, http.StatusNotFound)
	require.Equal(s.T(), "FILE_NOT_FOUND", respCrossTenant["code"])

	noCorr := helpers.NewAPIClient(s.server)
	noCorr.SetToken(s.sellerToken)
	noCorr.SetHeader(constants.CORRELATION_ID_HEADER, "")
	wMissingCorr := noCorr.Get(s.T(), "/api/file/"+fileID+"/download-url")
	respMissingCorr := helpers.AssertErrorResponse(s.T(), wMissingCorr, http.StatusBadRequest)
	require.True(s.T(), strings.Contains(respMissingCorr["message"].(string), "Correlation"))
}
