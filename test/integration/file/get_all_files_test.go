package file_test

import (
	"fmt"
	"net/http"
	"strings"

	"ecommerce-be/common/constants"
	"ecommerce-be/test/integration/helpers"

	"github.com/stretchr/testify/require"
)

func (s *UploadSuite) TestGetAllFiles_DefaultActiveOnlyAndIncludeVariants() {
	sellerActiveImage := s.createUploadedFile(
		s.sellerToken,
		map[string]any{
			"purpose":    "PRODUCT_IMAGE",
			"visibility": "PRIVATE",
			"filename":   "hero.jpg",
			"mimeType":   "image/jpeg",
			"sizeBytes":  2048,
		},
		make([]byte, 2048),
	)
	s.insertVariant(sellerActiveImage, "thumb_200", "image/webp", 512, "READY")

	sellerActiveDoc := s.createUploadedFile(
		s.sellerToken,
		map[string]any{
			"purpose":    "DOCUMENT",
			"visibility": "PRIVATE",
			"filename":   "policy.pdf",
			"mimeType":   "application/pdf",
			"sizeBytes":  1024,
		},
		make([]byte, 1024),
	)

	_ = s.createInitOnlyFile(
		s.sellerToken,
		map[string]any{
			"purpose":    "DOCUMENT",
			"visibility": "PRIVATE",
			"filename":   "still-uploading.pdf",
			"mimeType":   "application/pdf",
			"sizeBytes":  700,
		},
	)

	_ = s.createUploadedFile(
		s.seller2Token,
		map[string]any{
			"purpose":    "PRODUCT_IMAGE",
			"visibility": "PRIVATE",
			"filename":   "other-seller.jpg",
			"mimeType":   "image/jpeg",
			"sizeBytes":  900,
		},
		make([]byte, 900),
	)

	client := helpers.NewAPIClient(s.server)
	client.SetToken(s.sellerToken)
	client.SetHeader(constants.CORRELATION_ID_HEADER, "get-all-files-default-active")

	w := client.Get(s.T(), "/api/file?includeVariants=true&page=1&pageSize=10")
	resp := helpers.AssertSuccessResponse(s.T(), w, http.StatusOK)

	data := resp["data"].(map[string]any)
	items := data["items"].([]any)
	require.Len(s.T(), items, 2)

	found := map[string]map[string]any{}
	for _, raw := range items {
		item := raw.(map[string]any)
		found[item["fileId"].(string)] = item
	}

	require.Contains(s.T(), found, sellerActiveImage)
	require.Contains(s.T(), found, sellerActiveDoc)
	require.Equal(s.T(), "PRODUCT_IMAGE", found[sellerActiveImage]["purpose"])
	require.Equal(s.T(), "DOCUMENT", found[sellerActiveDoc]["purpose"])

	imageVariants := found[sellerActiveImage]["variants"].([]any)
	require.Len(s.T(), imageVariants, 1)
	require.Equal(s.T(), "thumb_200", imageVariants[0].(map[string]any)["variantCode"])

	docVariants := found[sellerActiveDoc]["variants"].([]any)
	require.Len(s.T(), docVariants, 0)

	pagination := data["pagination"].(map[string]any)
	require.Equal(s.T(), float64(1), pagination["currentPage"])
	require.Equal(s.T(), float64(2), pagination["totalItems"])
}

func (s *UploadSuite) TestGetAllFiles_FilterByPurposeStatusMimeAndIDs() {
	productImageID := s.createUploadedFile(
		s.sellerToken,
		map[string]any{
			"purpose":    "PRODUCT_IMAGE",
			"visibility": "PRIVATE",
			"filename":   "catalog.jpg",
			"mimeType":   "image/jpeg",
			"sizeBytes":  1500,
		},
		make([]byte, 1500),
	)
	documentID := s.createUploadedFile(
		s.sellerToken,
		map[string]any{
			"purpose":    "DOCUMENT",
			"visibility": "PRIVATE",
			"filename":   "terms.pdf",
			"mimeType":   "application/pdf",
			"sizeBytes":  1200,
		},
		make([]byte, 1200),
	)
	uploadingID := s.createInitOnlyFile(
		s.sellerToken,
		map[string]any{
			"purpose":    "DOCUMENT",
			"visibility": "PRIVATE",
			"filename":   "draft.pdf",
			"mimeType":   "application/pdf",
			"sizeBytes":  800,
		},
	)

	client := helpers.NewAPIClient(s.server)
	client.SetToken(s.sellerToken)
	client.SetHeader(constants.CORRELATION_ID_HEADER, "get-all-files-filters")

	wPurpose := client.Get(s.T(), "/api/file?purposes=DOCUMENT&mimeTypes=application/pdf")
	respPurpose := helpers.AssertSuccessResponse(s.T(), wPurpose, http.StatusOK)
	purposeItems := respPurpose["data"].(map[string]any)["items"].([]any)
	require.Len(s.T(), purposeItems, 1)
	require.Equal(s.T(), documentID, purposeItems[0].(map[string]any)["fileId"])

	wStatus := client.Get(s.T(), "/api/file?statuses=UPLOADING")
	respStatus := helpers.AssertSuccessResponse(s.T(), wStatus, http.StatusOK)
	statusItems := respStatus["data"].(map[string]any)["items"].([]any)
	require.Len(s.T(), statusItems, 1)
	require.Equal(s.T(), uploadingID, statusItems[0].(map[string]any)["fileId"])

	wIDs := client.Get(
		s.T(),
		fmt.Sprintf("/api/file?fileIds=%s,%s", productImageID, documentID),
	)
	respIDs := helpers.AssertSuccessResponse(s.T(), wIDs, http.StatusOK)
	idItems := respIDs["data"].(map[string]any)["items"].([]any)
	require.Len(s.T(), idItems, 2)
}

func (s *UploadSuite) TestGetAllFiles_IncludeDownloadURL() {
	privateDoc := s.createUploadedFile(
		s.sellerToken,
		map[string]any{
			"purpose":    "DOCUMENT",
			"visibility": "PRIVATE",
			"filename":   "private-doc.pdf",
			"mimeType":   "application/pdf",
			"sizeBytes":  1024,
		},
		make([]byte, 1024),
	)

	publicImg := s.createUploadedFile(
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
	client.SetHeader(constants.CORRELATION_ID_HEADER, "get-all-files-download-url")

	w := client.Get(
		s.T(),
		fmt.Sprintf("/api/file?fileIds=%s,%s&includeDownloadUrl=true", privateDoc, publicImg),
	)
	resp := helpers.AssertSuccessResponse(s.T(), w, http.StatusOK)
	items := resp["data"].(map[string]any)["items"].([]any)
	require.Len(s.T(), items, 2)

	found := map[string]map[string]any{}
	for _, raw := range items {
		item := raw.(map[string]any)
		found[item["fileId"].(string)] = item
	}

	require.Contains(s.T(), found, privateDoc)
	privateItem := found[privateDoc]
	require.NotEmpty(s.T(), privateItem["downloadUrl"])
	require.NotEmpty(s.T(), privateItem["downloadUrlExpiresAt"])
	require.Contains(s.T(), privateItem["downloadUrl"].(string), "X-Amz-")

	require.Contains(s.T(), found, publicImg)
	publicItem := found[publicImg]
	require.NotEmpty(s.T(), publicItem["downloadUrl"])
	require.NotContains(s.T(), publicItem, "downloadUrlExpiresAt")
	require.NotContains(s.T(), publicItem["downloadUrl"].(string), "X-Amz-")
}

func (s *UploadSuite) TestGetAllFiles_ValidationAndAuthBehavior() {
	client := helpers.NewAPIClient(s.server)
	client.SetToken(s.sellerToken)
	client.SetHeader(constants.CORRELATION_ID_HEADER, "get-all-files-validation")

	tooManyIDs := strings.Repeat("id,", 100) + "id"
	wTooMany := client.Get(s.T(), "/api/file?fileIds="+tooManyIDs)
	respTooMany := helpers.AssertErrorResponse(s.T(), wTooMany, http.StatusBadRequest)
	require.Equal(s.T(), constants.VALIDATION_ERROR_CODE, respTooMany["code"])

	wBadPurpose := client.Get(s.T(), "/api/file?purposes=NOT_A_REAL_PURPOSE")
	respBadPurpose := helpers.AssertErrorResponse(s.T(), wBadPurpose, http.StatusBadRequest)
	require.Equal(s.T(), constants.VALIDATION_ERROR_CODE, respBadPurpose["code"])

	customerClient := helpers.NewAPIClient(s.server)
	customerClient.SetToken(s.customerToken)
	customerClient.SetHeader(constants.CORRELATION_ID_HEADER, "get-all-files-customer")
	wForbidden := customerClient.Get(s.T(), "/api/file")
	helpers.AssertErrorResponse(s.T(), wForbidden, http.StatusForbidden)
}

func (s *UploadSuite) createInitOnlyFile(token string, req map[string]any) string {
	s.T().Helper()
	client := helpers.NewAPIClient(s.server)
	client.SetToken(token)
	client.SetHeader(constants.CORRELATION_ID_HEADER, "file-read-init-only")

	w := client.Post(s.T(), uploadInitEndpoint, req)
	resp := helpers.AssertSuccessResponse(s.T(), w, http.StatusCreated)
	return resp["data"].(map[string]any)["fileId"].(string)
}

func (s *UploadSuite) createUploadedFile(token string, req map[string]any, body []byte) string {
	s.T().Helper()
	client := helpers.NewAPIClient(s.server)
	client.SetToken(token)
	client.SetHeader(constants.CORRELATION_ID_HEADER, "file-read-active")

	initW := client.Post(s.T(), uploadInitEndpoint, req)
	initResp := helpers.AssertSuccessResponse(s.T(), initW, http.StatusCreated)
	initData := initResp["data"].(map[string]any)
	fileID := initData["fileId"].(string)

	uploadHelper := helpers.UploadHelper{Server: s.server, Token: token}
	uploadHelper.PutBytes(s.T(), initData, body)

	completeW := client.Post(s.T(), uploadCompleteEndpoint, map[string]any{"fileId": fileID})
	helpers.AssertSuccessResponse(s.T(), completeW, http.StatusOK)

	return fileID
}

func (s *UploadSuite) insertVariant(
	fileID string,
	variantCode string,
	mimeType string,
	sizeBytes int64,
	status string,
) {
	s.T().Helper()

	type fileRow struct {
		ID                uint64
		BucketOrContainer string
		ObjectKey         string
	}
	var row fileRow
	err := s.container.DB.Raw(
		"SELECT id, bucket_or_container, object_key FROM file_object WHERE file_id = ?",
		fileID,
	).Scan(&row).Error
	s.Require().NoError(err)

	s.insertVariantWithBucket(fileID, variantCode, mimeType, sizeBytes, status, row.BucketOrContainer)
}

func (s *UploadSuite) insertVariantWithBucket(
	fileID string,
	variantCode string,
	mimeType string,
	sizeBytes int64,
	status string,
	bucket string,
) {
	s.T().Helper()

	type fileRow struct {
		ID                uint64
		ObjectKey         string
	}
	var row fileRow
	err := s.container.DB.Raw(
		"SELECT id, object_key FROM file_object WHERE file_id = ?",
		fileID,
	).Scan(&row).Error
	s.Require().NoError(err)

	width := 200
	height := 200
	err = s.container.DB.Exec(
		`INSERT INTO file_variant (
			file_object_id, variant_code, mime_type, bucket_or_container, object_key,
			size_bytes, width, height, status, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, NOW(), NOW())`,
		row.ID,
		variantCode,
		mimeType,
		bucket,
		row.ObjectKey+"-"+variantCode,
		sizeBytes,
		width,
		height,
		status,
	).Error
	s.Require().NoError(err)
}
