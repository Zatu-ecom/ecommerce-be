package factory

import (
	"time"

	"ecommerce-be/file/entity"
	fileMessaging "ecommerce-be/file/messaging"
	"ecommerce-be/file/model"
	"ecommerce-be/file/utils"
	"ecommerce-be/file/utils/constant"
)

// BuildInitFileObject creates a file_object row for init-upload (status=UPLOADING).
func BuildInitFileObject(
	caller utils.Principal,
	req model.InitUploadRequest,
	cfg *entity.StorageConfig,
	fileID string,
	objectKey string,
	sanitizedFilename string,
	uploadExpiresAt time.Time,
) *entity.FileObject {
	return &entity.FileObject{
		FileID:            fileID,
		SellerID:          caller.SellerID,
		UploaderUserID:    caller.UserID,
		OwnerType:         caller.OwnerType,
		OwnerID:           ownerIDForCaller(caller),
		Purpose:           req.Purpose,
		Visibility:        req.Visibility,
		StorageConfigID:   uint64(cfg.ID),
		BucketOrContainer: cfg.BucketOrContainer,
		ObjectKey:         objectKey,
		OriginalFilename:  req.Filename,
		SanitizedFilename: sanitizedFilename,
		MimeType:          req.MimeType,
		SizeBytes:         req.SizeBytes,
		Status:            entity.FileStatusUploading,
		UploadExpiresAt:   uploadExpiresAt,
	}
}

// BuildInitUploadData creates API response payload for init-upload.
func BuildInitUploadData(
	fileID string,
	mimeType string,
	objectKey string,
	presigned model.BlobPresignOutput,
) *model.InitUploadData {
	return &model.InitUploadData{
		FileID:       fileID,
		Status:       string(entity.FileStatusUploading),
		UploadURL:    presigned.URL,
		UploadMethod: "PUT",
		UploadHeaders: map[string]string{
			"Content-Type": mimeType,
		},
		ObjectKey: objectKey,
		ExpiresAt: presigned.ExpiresAt.UTC().Format(time.RFC3339),
	}
}

// BuildCompleteUploadData creates API response payload for complete-upload.
func BuildCompleteUploadData(
	fileID string,
	mimeType string,
	sizeBytes int64,
	etag string,
	completedAt string,
	variantsQueued bool,
) *model.CompleteUploadData {
	return &model.CompleteUploadData{
		FileID:         fileID,
		Status:         string(entity.FileStatusActive),
		MimeType:       mimeType,
		SizeBytes:      sizeBytes,
		Etag:           etag,
		CompletedAt:    completedAt,
		VariantsQueued: variantsQueued,
	}
}

// BuildImageProcessRequested creates variant-message payload for RabbitMQ.
func BuildImageProcessRequested(
	row *entity.FileObject,
	mimeType string,
	sizeBytes int64,
	variants []string,
) fileMessaging.ImageProcessRequested {
	return fileMessaging.ImageProcessRequested{
		FileID:            row.FileID,
		FileObjectID:      uint64(row.ID),
		StorageConfigID:   row.StorageConfigID,
		BucketOrContainer: row.BucketOrContainer,
		ObjectKey:         row.ObjectKey,
		MimeType:          mimeType,
		SizeBytes:         sizeBytes,
		Purpose:           string(row.Purpose),
		VariantsRequested: variants,
	}
}

// BuildFileJob creates file_job row for publish status tracking.
func BuildFileJob(
	fileObjectID uint64,
	status entity.FileJobStatus,
	lastError *string,
	correlationID string,
) *entity.FileJob {
	return &entity.FileJob{
		FileObjectID:  fileObjectID,
		Command:       constant.RoutingKeyFileImageProcessRequested,
		Status:        status,
		Attempts:      1,
		LastError:     lastError,
		CorrelationID: correlationID,
	}
}

func ownerIDForCaller(caller utils.Principal) *uint64 {
	if caller.OwnerType == entity.OwnerTypeSeller && caller.SellerID != nil {
		v := *caller.SellerID
		return &v
	}
	return nil
}
