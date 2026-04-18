package blob_adapter

import (
	"context"
	"io"

	"ecommerce-be/file/model"
)

// BlobAdapter defines the provider-agnostic blob storage interface used by file services.
//
// Contract rules:
//   - context.Context is always the first argument and must be honoured for cancellation/deadline.
//   - GetObjectStream returns an io.ReadCloser; the caller is responsible for calling Close().
//   - All returned errors are *commonError.AppError from the file/error package.
//   - No raw credentials appear in any error message or log output.
type BlobAdapter interface {
	// PutObject uploads the object described by in to the provider.
	// Returns the assigned ETag and the canonical key on success.
	PutObject(ctx context.Context, in model.BlobPutObjectInput) (model.BlobPutObjectOutput, error)

	// DeleteObject removes the object at key inside bucket.
	// Returns nil when the object does not exist (idempotent).
	DeleteObject(ctx context.Context, bucket, key string) error

	// HeadObject returns normalised metadata for an existing object
	// without downloading its body. Returns ErrBlobNotFound when the
	// object or bucket does not exist.
	HeadObject(ctx context.Context, bucket, key string) (model.BlobObjectMeta, error)

	// GetObjectStream opens a streaming reader for an existing object.
	// The caller must close the returned io.ReadCloser regardless of error.
	// Returns ErrBlobNotFound when the object or bucket does not exist.
	GetObjectStream(ctx context.Context, bucket, key string) (io.ReadCloser, model.BlobObjectMeta, error)

	// PresignUpload returns a time-limited URL that allows an unauthenticated
	// client to upload directly to the provider. in.TTL must be > 0.
	PresignUpload(ctx context.Context, in model.BlobPresignUploadInput) (model.BlobPresignOutput, error)

	// PresignDownload returns a time-limited URL that allows an unauthenticated
	// client to download an existing object directly from the provider.
	// in.TTL must be > 0.
	PresignDownload(ctx context.Context, in model.BlobPresignDownloadInput) (model.BlobPresignOutput, error)

	// CopyObject copies an existing object from source to destination within
	// the same provider account. Cross-provider copy is out of scope.
	CopyObject(ctx context.Context, in model.BlobCopyObjectInput) error
}
