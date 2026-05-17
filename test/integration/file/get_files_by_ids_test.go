package file_test

import (
	"fmt"
	"net/http"

	"ecommerce-be/common/constants"
	"ecommerce-be/test/integration/helpers"
	"github.com/stretchr/testify/require"
)

func (s *UploadSuite) TestGetAllFiles_FileIDsQuery_OmitsCrossTenantIDs() {
	sellerFileID := s.createUploadedFile(
		s.sellerToken,
		map[string]any{
			"purpose":    "PRODUCT_IMAGE",
			"visibility": "PRIVATE",
			"filename":   "seller-cross-tenant.jpg",
			"mimeType":   "image/jpeg",
			"sizeBytes":  2048,
		},
		make([]byte, 2048),
	)

	otherSellerFileID := s.createUploadedFile(
		s.seller2Token,
		map[string]any{
			"purpose":    "DOCUMENT",
			"visibility": "PRIVATE",
			"filename":   "seller-two.pdf",
			"mimeType":   "application/pdf",
			"sizeBytes":  1024,
		},
		make([]byte, 1024),
	)

	adminFileID := s.createUploadedFile(
		s.adminToken,
		map[string]any{
			"purpose":    "DOCUMENT",
			"visibility": "PRIVATE",
			"filename":   "platform-admin.pdf",
			"mimeType":   "application/pdf",
			"sizeBytes":  512,
		},
		make([]byte, 512),
	)

	client := helpers.NewAPIClient(s.server)
	client.SetToken(s.sellerToken)
	client.SetHeader(constants.CORRELATION_ID_HEADER, "file-ids-query-cross-tenant")

	w := client.Get(
		s.T(),
		fmt.Sprintf("/api/file?fileIds=%s,%s,%s", otherSellerFileID, sellerFileID, adminFileID),
	)
	resp := helpers.AssertSuccessResponse(s.T(), w, http.StatusOK)
	items := resp["data"].(map[string]any)["items"].([]any)
	require.Len(s.T(), items, 1)
	require.Equal(s.T(), sellerFileID, items[0].(map[string]any)["fileId"])
}

func (s *UploadSuite) TestGetAllFiles_FileIDsQuery_OmitsMissingAndDuplicateIDs() {
	fileID := s.createUploadedFile(
		s.sellerToken,
		map[string]any{
			"purpose":    "DOCUMENT",
			"visibility": "PRIVATE",
			"filename":   "dedupe.pdf",
			"mimeType":   "application/pdf",
			"sizeBytes":  700,
		},
		make([]byte, 700),
	)

	client := helpers.NewAPIClient(s.server)
	client.SetToken(s.sellerToken)
	client.SetHeader(constants.CORRELATION_ID_HEADER, "file-ids-query-dedupe")

	w := client.Get(
		s.T(),
		fmt.Sprintf("/api/file?fileIds=%s,%s,missing-file-id", fileID, fileID),
	)
	resp := helpers.AssertSuccessResponse(s.T(), w, http.StatusOK)
	items := resp["data"].(map[string]any)["items"].([]any)
	require.Len(s.T(), items, 1)
	require.Equal(s.T(), fileID, items[0].(map[string]any)["fileId"])
}
