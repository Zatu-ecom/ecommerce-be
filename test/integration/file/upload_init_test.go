package file_test

import (
	"context"
	"net/http"

	"ecommerce-be/file/entity"
	"ecommerce-be/test/integration/helpers"

	"github.com/stretchr/testify/require"
)

func (s *UploadSuite) TestInitUpload_ProductImage_HappyPath() {
	client := helpers.NewAPIClient(s.server)
	client.SetToken(s.sellerToken)

	correlationID := "upload-init-happy-correlation-id"
	client.SetHeader("X-Correlation-ID", correlationID)

	req := map[string]interface{}{
		"purpose":             "PRODUCT_IMAGE",
		"visibility":          "PRIVATE",
		"filename":            "hero-shot.jpg",
		"mimeType":            "image/jpeg",
		"sizeBytes":           1024,
		"uploadExpiryMinutes": 15,
	}

	w := client.Post(s.T(), uploadInitEndpoint, req)
	resp := helpers.AssertSuccessResponse(s.T(), w, http.StatusCreated)

	data := resp["data"].(map[string]interface{})
	require.NotEmpty(s.T(), data["fileId"])
	require.NotEmpty(s.T(), data["uploadUrl"])
	require.Equal(s.T(), "UPLOADING", data["status"])
	require.NotNil(s.T(), data["uploadHeaders"])

	fileID := data["fileId"].(string)
	type row struct {
		ID     uint64
		Status string
	}
	var r row
	err := s.container.DB.Raw(
		"SELECT id, status FROM file_object WHERE file_id = ?",
		fileID,
	).Scan(&r).Error
	require.NoError(s.T(), err)
	require.Equal(s.T(), string(entity.FileStatusUploading), r.Status)

	helpers.AssertSchedulerJobExists(s.T(), s.container.RedisClient, r.ID)

	keys, err := s.container.RedisClient.Keys(context.Background(), "file:init:idem:*").Result()
	require.NoError(s.T(), err)
	require.Len(s.T(), keys, 0)
}
