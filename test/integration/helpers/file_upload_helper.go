package helpers

import (
	"net/http"
	"testing"

	"ecommerce-be/common/constants"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// UploadFile performs init-upload, PUT, and complete-upload for the given file purpose.
func UploadFile(t *testing.T, server http.Handler, token, purpose, filename string) string {
	t.Helper()

	client := NewAPIClient(server)
	client.SetToken(token)
	client.SetHeader(constants.CORRELATION_ID_HEADER, uuid.NewString())

	initReq := map[string]any{
		"purpose":    purpose,
		"visibility": "PRIVATE",
		"filename":   filename,
		"mimeType":   "image/jpeg",
		"sizeBytes":  1024,
	}
	initW := client.Post(t, "/api/file/init-upload", initReq)
	initResp := AssertSuccessResponse(t, initW, http.StatusCreated)
	initData := initResp["data"].(map[string]any)
	fileID, ok := initData["fileId"].(string)
	require.True(t, ok)
	require.NotEmpty(t, fileID)

	uploadHelper := UploadHelper{Server: server, Token: token}
	uploadHelper.PutBytes(t, initData, make([]byte, 1024))

	completeW := client.Post(t, "/api/file/complete-upload", map[string]any{"fileId": fileID})
	AssertSuccessResponse(t, completeW, http.StatusOK)

	return fileID
}

// UploadProductImage performs init-upload, PUT, and complete-upload for a seller token.
// Returns the fileId of the uploaded PRODUCT_IMAGE.
func UploadProductImage(t *testing.T, server http.Handler, token string) string {
	t.Helper()
	return UploadFile(t, server, token, "PRODUCT_IMAGE", "integration-test.jpg")
}

// UploadSellerLogo uploads a SELLER_LOGO file and returns its fileId.
func UploadSellerLogo(t *testing.T, server http.Handler, token string) string {
	t.Helper()
	return UploadFile(t, server, token, "SELLER_LOGO", "seller-logo.jpg")
}
