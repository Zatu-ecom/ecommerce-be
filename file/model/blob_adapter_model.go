package model

import (
	"io"
	"time"
)

// ─── BlobAdapter Operation Input/Output Models ───────────────────────────────
// These are internal service-layer DTOs used by blob adapter implementations.
// They do not carry JSON tags as they are never serialised to HTTP responses.

// BlobPutObjectInput carries the parameters for an object upload operation.
// All fields are required; Body must not be nil.
type BlobPutObjectInput struct {
	Bucket        string
	Key           string
	ContentType   string
	ContentLength int64
	Body          io.Reader
}

// BlobPutObjectOutput is returned after a successful object upload.
// ETag is stripped of surrounding double-quotes for consistent comparisons.
type BlobPutObjectOutput struct {
	Key       string
	ETag      string
	VersionID *string
}

// BlobObjectMeta holds provider-normalised metadata about an existing object.
// Returned by HeadObject and GetObjectStream.
type BlobObjectMeta struct {
	ContentType  string
	SizeBytes    int64
	LastModified time.Time
	ETag         string
	VersionID    *string
}

// BlobPresignUploadInput carries parameters for generating a time-limited upload URL.
// TTL must be > 0 or the adapter returns a validation error before any SDK call.
type BlobPresignUploadInput struct {
	Bucket             string
	Key                string
	ContentType        string
	ContentLengthLimit int64
	TTL                time.Duration
}

// BlobPresignDownloadInput carries parameters for generating a time-limited download URL.
// TTL must be > 0 or the adapter returns a validation error before any SDK call.
type BlobPresignDownloadInput struct {
	Bucket string
	Key    string
	TTL    time.Duration
}

// BlobPresignOutput is returned by PresignUpload and PresignDownload.
// Azure SAS URLs and S3 presigned URLs are both represented here.
type BlobPresignOutput struct {
	URL       string
	ExpiresAt time.Time
}

// BlobCopyObjectInput carries parameters for an intra-account object copy.
// Cross-provider copy is out of scope for the BlobAdapter interface.
type BlobCopyObjectInput struct {
	SourceBucket      string
	SourceKey         string
	DestinationBucket string
	DestinationKey    string
}
