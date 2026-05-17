package file_test

import (
	"net/http"

	"ecommerce-be/common/constants"
	"ecommerce-be/test/integration/helpers"

	"github.com/stretchr/testify/require"
)

func (s *UploadSuite) TestDeleteFile_ActiveFileRemovesRowAndBlob() {
	fileID := s.createUploadedFile(
		s.sellerToken,
		map[string]any{
			"purpose":    "DOCUMENT",
			"visibility": "PRIVATE",
			"filename":   "delete-active.pdf",
			"mimeType":   "application/pdf",
			"sizeBytes":  1500,
		},
		make([]byte, 1500),
	)

	before := s.countMinioObjects()

	client := helpers.NewAPIClient(s.server)
	client.SetToken(s.sellerToken)
	client.SetHeader(constants.CORRELATION_ID_HEADER, "delete-active")

	w := client.Delete(s.T(), "/api/file/"+fileID)
	resp := helpers.AssertSuccessResponse(s.T(), w, http.StatusOK)
	require.Equal(s.T(), fileID, resp["data"].(map[string]any)["fileId"])

	type row struct{ Count int64 }
	var r row
	err := s.container.DB.Raw("SELECT COUNT(1) AS count FROM file_object WHERE file_id = ?", fileID).Scan(&r).Error
	require.NoError(s.T(), err)
	require.Equal(s.T(), int64(0), r.Count)
	require.Less(s.T(), s.countMinioObjects(), before)
}

func (s *UploadSuite) TestDeleteFile_UploadingCancelsSchedulerAndDeletesRow() {
	fileID := s.createInitOnlyFile(
		s.sellerToken,
		map[string]any{
			"purpose":    "DOCUMENT",
			"visibility": "PRIVATE",
			"filename":   "delete-uploading.pdf",
			"mimeType":   "application/pdf",
			"sizeBytes":  900,
		},
	)

	var row struct{ ID uint64 }
	err := s.container.DB.Raw("SELECT id FROM file_object WHERE file_id = ?", fileID).Scan(&row).Error
	require.NoError(s.T(), err)
	helpers.AssertSchedulerJobExists(s.T(), s.container.RedisClient, row.ID)

	client := helpers.NewAPIClient(s.server)
	client.SetToken(s.sellerToken)
	client.SetHeader(constants.CORRELATION_ID_HEADER, "delete-uploading")
	w := client.Delete(s.T(), "/api/file/"+fileID)
	helpers.AssertSuccessResponse(s.T(), w, http.StatusOK)

	helpers.AssertNoSchedulerJob(s.T(), s.container.RedisClient, row.ID)

	var countRow struct{ Count int64 }
	err = s.container.DB.Raw("SELECT COUNT(1) AS count FROM file_object WHERE file_id = ?", fileID).Scan(&countRow).Error
	require.NoError(s.T(), err)
	require.Equal(s.T(), int64(0), countRow.Count)
}

func (s *UploadSuite) TestDeleteFile_VariantsCascadeAndBestEffortCleanup() {
	fileID := s.createUploadedFile(
		s.sellerToken,
		map[string]any{
			"purpose":    "PRODUCT_IMAGE",
			"visibility": "PRIVATE",
			"filename":   "delete-variants.jpg",
			"mimeType":   "image/jpeg",
			"sizeBytes":  2048,
		},
		make([]byte, 2048),
	)

	s.insertVariant(fileID, "thumb_200", "image/webp", 300, "READY")
	s.insertVariantWithBucket(fileID, "broken_variant", "image/webp", 400, "READY", "missing-bucket")

	var row struct{ ID uint64 }
	err := s.container.DB.Raw("SELECT id FROM file_object WHERE file_id = ?", fileID).Scan(&row).Error
	require.NoError(s.T(), err)

	client := helpers.NewAPIClient(s.server)
	client.SetToken(s.sellerToken)
	client.SetHeader(constants.CORRELATION_ID_HEADER, "delete-variants")
	w := client.Delete(s.T(), "/api/file/"+fileID)
	helpers.AssertSuccessResponse(s.T(), w, http.StatusOK)

	var fileCount struct{ Count int64 }
	err = s.container.DB.Raw("SELECT COUNT(1) AS count FROM file_object WHERE id = ?", row.ID).Scan(&fileCount).Error
	require.NoError(s.T(), err)
	require.Equal(s.T(), int64(0), fileCount.Count)

	var variantCount struct{ Count int64 }
	err = s.container.DB.Raw("SELECT COUNT(1) AS count FROM file_variant WHERE file_object_id = ?", row.ID).Scan(&variantCount).Error
	require.NoError(s.T(), err)
	require.Equal(s.T(), int64(0), variantCount.Count)
}

func (s *UploadSuite) TestDeleteFile_TenantIsolationAndMissingCorrelation() {
	fileID := s.createUploadedFile(
		s.sellerToken,
		map[string]any{
			"purpose":    "DOCUMENT",
			"visibility": "PRIVATE",
			"filename":   "delete-tenant.pdf",
			"mimeType":   "application/pdf",
			"sizeBytes":  1024,
		},
		make([]byte, 1024),
	)

	seller2Client := helpers.NewAPIClient(s.server)
	seller2Client.SetToken(s.seller2Token)
	seller2Client.SetHeader(constants.CORRELATION_ID_HEADER, "delete-cross-tenant")
	wCrossTenant := seller2Client.Delete(s.T(), "/api/file/"+fileID)
	respCrossTenant := helpers.AssertErrorResponse(s.T(), wCrossTenant, http.StatusNotFound)
	require.Equal(s.T(), "FILE_NOT_FOUND", respCrossTenant["code"])

	noCorr := helpers.NewAPIClient(s.server)
	noCorr.SetToken(s.sellerToken)
	noCorr.SetHeader(constants.CORRELATION_ID_HEADER, "")
	wMissingCorr := noCorr.Delete(s.T(), "/api/file/"+fileID)
	helpers.AssertErrorResponse(s.T(), wMissingCorr, http.StatusBadRequest)
}
