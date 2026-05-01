package file_test

import (
	"net/http"

	"ecommerce-be/common/constants"
	"ecommerce-be/file/entity"
	"ecommerce-be/test/integration/helpers"

	"github.com/stretchr/testify/require"
)

func (s *UploadSuite) TestCompleteUpload_TenantIsolation() {
	initAsSellerA := func() string {
		client := helpers.NewAPIClient(s.server)
		client.SetToken(s.sellerToken)
		client.SetHeader(constants.CORRELATION_ID_HEADER, "tenant-isolation-init")

		initReq := map[string]interface{}{
			"purpose":             "PRODUCT_IMAGE",
			"visibility":          "PRIVATE",
			"filename":            "tenant-a.jpg",
			"mimeType":            "image/jpeg",
			"sizeBytes":           1024,
			"uploadExpiryMinutes": 15,
		}

		initW := client.Post(s.T(), uploadInitEndpoint, initReq)
		initResp := helpers.AssertSuccessResponse(s.T(), initW, http.StatusCreated)
		return initResp["data"].(map[string]interface{})["fileId"].(string)
	}

	s.Run("SellerB_Cannot_Complete_SellerA_File", func() {
		fileID := initAsSellerA()

		client := helpers.NewAPIClient(s.server)
		client.SetToken(s.seller2Token)
		client.SetHeader(constants.CORRELATION_ID_HEADER, "tenant-isolation-seller-b")

		w := client.Post(s.T(), uploadCompleteEndpoint, map[string]interface{}{
			"fileId": fileID,
		})
		resp := helpers.AssertErrorResponse(s.T(), w, http.StatusNotFound)
		require.Equal(s.T(), "FILE_UPLOAD_NOT_FOUND", resp["code"])

		s.assertFileStatus(fileID, entity.FileStatusUploading)
	})

	s.Run("Unauthenticated_Returns_401", func() {
		fileID := initAsSellerA()

		client := helpers.NewAPIClient(s.server)
		client.SetToken("")
		client.SetHeader(constants.CORRELATION_ID_HEADER, "tenant-isolation-unauth")

		w := client.Post(s.T(), uploadCompleteEndpoint, map[string]interface{}{
			"fileId": fileID,
		})
		helpers.AssertErrorResponse(s.T(), w, http.StatusUnauthorized)

		s.assertFileStatus(fileID, entity.FileStatusUploading)
	})

	s.Run("Customer_Returns_403", func() {
		fileID := initAsSellerA()

		client := helpers.NewAPIClient(s.server)
		client.SetToken(s.customerToken)
		client.SetHeader(constants.CORRELATION_ID_HEADER, "tenant-isolation-customer")

		w := client.Post(s.T(), uploadCompleteEndpoint, map[string]interface{}{
			"fileId": fileID,
		})
		helpers.AssertErrorResponse(s.T(), w, http.StatusForbidden)

		s.assertFileStatus(fileID, entity.FileStatusUploading)
	})

	s.Run("Admin_Cannot_Complete_Seller_File", func() {
		fileID := initAsSellerA()

		client := helpers.NewAPIClient(s.server)
		client.SetToken(s.adminToken)
		client.SetHeader(constants.CORRELATION_ID_HEADER, "tenant-isolation-admin")

		w := client.Post(s.T(), "/api/admin/files/complete-upload", map[string]interface{}{
			"fileId": fileID,
		})
		resp := helpers.AssertErrorResponse(s.T(), w, http.StatusNotFound)
		require.Equal(s.T(), "FILE_UPLOAD_NOT_FOUND", resp["code"])

		s.assertFileStatus(fileID, entity.FileStatusUploading)
	})
}
