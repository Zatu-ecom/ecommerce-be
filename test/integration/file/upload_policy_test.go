package file_test

import (
	"context"
	"net/http"
	"testing"

	"ecommerce-be/common/constants"
	"ecommerce-be/file/entity"
	"ecommerce-be/file/utils"
	uploadConstant "ecommerce-be/file/utils/constant"
	"ecommerce-be/test/integration/helpers"

	"github.com/stretchr/testify/require"
)

func TestUploadPolicyEvaluate(t *testing.T) {
	tests := []struct {
		name      string
		purpose   entity.FilePurpose
		mime      string
		size      int64
		shouldErr bool
	}{
		{"product image valid jpeg", entity.FilePurposeProductImage, "image/jpeg", 1024, false},
		{
			"product image invalid mime",
			entity.FilePurposeProductImage,
			"application/pdf",
			1024,
			true,
		},
		{
			"product image too large",
			entity.FilePurposeProductImage,
			"image/jpeg",
			11 * 1024 * 1024,
			true,
		},
		{"document pdf valid", entity.FilePurposeDocument, "application/pdf", 1024, false},
		{"import csv valid", entity.FilePurposeImportFile, "text/csv", 1024, false},
		{"avatar png valid", entity.FilePurposeUserAvatar, "image/png", 1024, false},
		{"seller logo svg no variants", entity.FilePurposeSellerLogo, "image/svg+xml", 1024, false},
		{"invoice pdf valid", entity.FilePurposeInvoicePDF, "application/pdf", 1024, false},
		{"export file rejected", entity.FilePurposeExportFile, "application/pdf", 1024, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p, err := utils.Evaluate(tc.purpose, tc.mime, tc.size)
			if tc.shouldErr {
				require.NotNil(t, err)
				return
			}
			require.Nil(t, err)
			require.NotNil(t, p)
		})
	}
}

func TestUploadPolicyEvaluate_NegativeBranches(t *testing.T) {
	tests := []struct {
		name    string
		purpose entity.FilePurpose
		mime    string
		size    int64
	}{
		{
			name:    "oversized product image",
			purpose: entity.FilePurposeProductImage,
			mime:    "image/jpeg",
			size:    12 * 1024 * 1024,
		},
		{
			name:    "disallowed mime",
			purpose: entity.FilePurposeProductImage,
			mime:    "application/x-msdownload",
			size:    1024,
		},
		{
			name:    "zero size",
			purpose: entity.FilePurposeProductImage,
			mime:    "image/jpeg",
			size:    0,
		},
		{
			name:    "export purpose",
			purpose: entity.FilePurposeExportFile,
			mime:    "application/pdf",
			size:    1024,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p, err := utils.Evaluate(tc.purpose, tc.mime, tc.size)
			require.Nil(t, p)
			require.NotNil(t, err)
			require.Equal(t, uploadConstant.FILE_UPLOAD_POLICY_VIOLATION_CODE, err.Code)
		})
	}
}

func (s *UploadSuite) TestInitUpload_PolicyRejections_NoSideEffects() {
	type tc struct {
		name         string
		req          map[string]interface{}
		wantHTTPCode int
		wantCode     string
	}

	tests := []tc{
		{
			name: "Reject_OversizedProductImage",
			req: map[string]interface{}{
				"purpose":    "PRODUCT_IMAGE",
				"visibility": "PRIVATE",
				"filename":   "big.jpg",
				"mimeType":   "image/jpeg",
				"sizeBytes":  12 * 1024 * 1024,
			},
			wantHTTPCode: http.StatusUnprocessableEntity,
			wantCode:     uploadConstant.FILE_UPLOAD_POLICY_VIOLATION_CODE,
		},
		{
			name: "Reject_DisallowedMime",
			req: map[string]interface{}{
				"purpose":    "PRODUCT_IMAGE",
				"visibility": "PRIVATE",
				"filename":   "evil.exe",
				"mimeType":   "application/x-msdownload",
				"sizeBytes":  2048,
			},
			wantHTTPCode: http.StatusUnprocessableEntity,
			wantCode:     uploadConstant.FILE_UPLOAD_POLICY_VIOLATION_CODE,
		},
		{
			name: "Reject_EmptyFilename",
			req: map[string]interface{}{
				"purpose":    "PRODUCT_IMAGE",
				"visibility": "PRIVATE",
				"filename":   "",
				"mimeType":   "image/jpeg",
				"sizeBytes":  2048,
			},
			wantHTTPCode: http.StatusBadRequest,
			wantCode:     constants.VALIDATION_ERROR_CODE,
		},
		{
			name: "Reject_ZeroSize",
			req: map[string]interface{}{
				"purpose":    "PRODUCT_IMAGE",
				"visibility": "PRIVATE",
				"filename":   "zero.jpg",
				"mimeType":   "image/jpeg",
				"sizeBytes":  0,
			},
			wantHTTPCode: http.StatusBadRequest,
			wantCode:     constants.VALIDATION_ERROR_CODE,
		},
		{
			name: "Reject_ExportFilePurpose",
			req: map[string]interface{}{
				"purpose":    "EXPORT_FILE",
				"visibility": "PRIVATE",
				"filename":   "export.pdf",
				"mimeType":   "application/pdf",
				"sizeBytes":  2048,
			},
			wantHTTPCode: http.StatusUnprocessableEntity,
			wantCode:     uploadConstant.FILE_UPLOAD_POLICY_VIOLATION_CODE,
		},
	}

	for _, test := range tests {
		s.Run(test.name, func() {
			client := helpers.NewAPIClient(s.server)
			client.SetToken(s.sellerToken)
			client.SetHeader(constants.CORRELATION_ID_HEADER, "upload-policy-reject")

			var fileObjectCountBefore int64
			err := s.container.DB.Raw("SELECT COUNT(1) FROM file_object").
				Scan(&fileObjectCountBefore).
				Error
			s.Require().NoError(err)

			schedulerJobsBefore, err := s.container.RedisClient.ZCard(context.Background(), "delayed_jobs").
				Result()
			s.Require().NoError(err)

			minioObjectsBefore := s.countMinioObjects()

			w := client.Post(s.T(), uploadInitEndpoint, test.req)
			resp := helpers.AssertErrorResponse(s.T(), w, test.wantHTTPCode)
			s.Require().Equal(test.wantCode, resp["code"])

			var fileObjectCountAfter int64
			err = s.container.DB.Raw("SELECT COUNT(1) FROM file_object").
				Scan(&fileObjectCountAfter).
				Error
			s.Require().NoError(err)
			s.Require().Equal(fileObjectCountBefore, fileObjectCountAfter)

			schedulerJobsAfter, err := s.container.RedisClient.ZCard(context.Background(), "delayed_jobs").
				Result()
			s.Require().NoError(err)
			s.Require().Equal(schedulerJobsBefore, schedulerJobsAfter)

			minioObjectsAfter := s.countMinioObjects()
			s.Require().Equal(minioObjectsBefore, minioObjectsAfter)
		})
	}
}

func (s *UploadSuite) TestInitUpload_RejectsUnknownPurposeAndVisibility() {
	client := helpers.NewAPIClient(s.server)
	client.SetToken(s.sellerToken)
	client.SetHeader(constants.CORRELATION_ID_HEADER, "upload-policy-unknown-enum")

	wUnknownPurpose := client.Post(s.T(), uploadInitEndpoint, map[string]interface{}{
		"purpose":    "UNKNOWN_PURPOSE",
		"visibility": "PRIVATE",
		"filename":   "x.jpg",
		"mimeType":   "image/jpeg",
		"sizeBytes":  10,
	})
	respPurpose := helpers.AssertErrorResponse(s.T(), wUnknownPurpose, http.StatusBadRequest)
	s.Require().Equal(constants.VALIDATION_ERROR_CODE, respPurpose["code"])

	wUnknownVisibility := client.Post(s.T(), uploadInitEndpoint, map[string]interface{}{
		"purpose":    "PRODUCT_IMAGE",
		"visibility": "OUTSIDE",
		"filename":   "x.jpg",
		"mimeType":   "image/jpeg",
		"sizeBytes":  10,
	})
	respVisibility := helpers.AssertErrorResponse(s.T(), wUnknownVisibility, http.StatusBadRequest)
	s.Require().Equal(constants.VALIDATION_ERROR_CODE, respVisibility["code"])
}
