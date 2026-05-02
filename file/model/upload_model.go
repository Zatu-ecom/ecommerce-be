package model

import "ecommerce-be/file/entity"

// ─── Init-Upload Request ─────────────────────────────────────────────────────

// InitUploadRequest is the request body for POST /api/files/init-upload.
//
// All fields are bound from JSON. Unknown JSON keys are rejected by the handler
// via json.Decoder.DisallowUnknownFields() (FR-002).
//
// Validation rules (see data-model.md §5):
//   - Purpose:              required; closed enum FilePurpose; EXPORT_FILE additionally rejected by policy
//   - Visibility:           optional (default PRIVATE); closed enum FileVisibility
//   - Filename:             required; 1..255 chars; server sanitises before storage
//   - MimeType:             required; must match purpose policy (checked in upload_policy.Evaluate)
//   - SizeBytes:            required; >= 1; <= purpose policy max (checked in upload_policy.Evaluate)
//   - UploadExpiryMinutes:  optional (default 15); [5, 60] — validated by struct tag
type InitUploadRequest struct {
	// Purpose determines policy (max size, allowed mimes, variant generation).
	Purpose entity.FilePurpose `json:"purpose" binding:"required,oneof=PRODUCT_IMAGE IMPORT_FILE EXPORT_FILE DOCUMENT USER_AVATAR SELLER_LOGO INVOICE_PDF"`

	// Visibility is PRIVATE by default; in v1 it is a logical flag only (FR-008a).
	Visibility entity.FileVisibility `json:"visibility" binding:"omitempty,oneof=PRIVATE PUBLIC INTERNAL"`

	// Filename is the original client-supplied filename (before sanitisation).
	Filename string `json:"filename" binding:"required,min=1,max=255"`

	// MimeType declared by the client at init time; re-validated against HeadObject at complete time.
	MimeType string `json:"mimeType" binding:"required"`

	// SizeBytes declared by the client; must be >= 1 and <= purpose max size.
	SizeBytes int64 `json:"sizeBytes" binding:"required,min=1"`

	// UploadExpiryMinutes is optional; when omitted the default (15) is applied in the service.
	// Range [5, 60] validated by the service; out-of-range values return 400 VALIDATION_FAILED.
	UploadExpiryMinutes *int `json:"uploadExpiryMinutes"`
}

// ─── Complete-Upload Request ──────────────────────────────────────────────────

// CompleteUploadRequest is the request body for POST /api/files/complete-upload.
//
// Only FileID is required. ClientEtag and ActualSizeBytes are optional trust-but-verify
// hints; the service always calls HeadObject regardless (FR-016).
type CompleteUploadRequest struct {
	// FileID is the UUIDv7 string returned by init-upload.
	FileID string `json:"fileId" binding:"required"`

	// ClientEtag is the ETag the client received from the provider's PUT response.
	// When present, it is compared against HeadObject.ETag; mismatch → 422.
	ClientEtag *string `json:"clientEtag"`

	// ActualSizeBytes is the byte count reported by the provider for the uploaded object.
	// When present, it is compared against HeadObject.SizeBytes; mismatch → 422.
	ActualSizeBytes *int64 `json:"actualSizeBytes"`
}

// ─── Init-Upload Response ─────────────────────────────────────────────────────

// InitUploadData is the data payload inside the 201/200 response for init-upload.
//
// Matches the contract defined in contracts/init-upload.http.md.
type InitUploadData struct {
	// FileID is the canonical client-facing identifier (UUIDv7 string).
	FileID string `json:"fileId"`

	// Status is always "UPLOADING" at init time.
	Status string `json:"status"`

	// UploadURL is the presigned PUT URL the client must use to upload bytes directly to the provider.
	UploadURL string `json:"uploadUrl"`

	// UploadMethod is always "PUT".
	UploadMethod string `json:"uploadMethod"`

	// UploadHeaders contains provider-required headers the client must echo on the PUT request.
	// Typically includes "Content-Type"; may include "Content-Length" for some providers.
	UploadHeaders map[string]string `json:"uploadHeaders"`

	// ObjectKey is the deterministic key within the bucket/container.
	ObjectKey string `json:"objectKey"`

	// ExpiresAt is the UTC timestamp after which the presigned URL and the upload window expire.
	ExpiresAt string `json:"expiresAt"` // RFC3339 UTC

	// Replayed is true when this payload came from an Idempotency-Key replay.
	Replayed bool `json:"-"`
}

// ─── Complete-Upload Response ─────────────────────────────────────────────────

// CompleteUploadData is the data payload inside the 200 response for complete-upload.
//
// Matches the contract defined in contracts/complete-upload.http.md.
type CompleteUploadData struct {
	// FileID is the canonical client-facing identifier.
	FileID string `json:"fileId"`

	// Status is "ACTIVE" on success.
	Status string `json:"status"`

	// MimeType is the content-type confirmed by HeadObject.
	MimeType string `json:"mimeType"`

	// SizeBytes is the provider-confirmed object size in bytes.
	SizeBytes int64 `json:"sizeBytes"`

	// Etag is the ETag returned by HeadObject, stripped of surrounding quotes.
	// May be empty for providers that do not support ETags.
	Etag string `json:"etag,omitempty"`

	// CompletedAt is the UTC timestamp when the row was transitioned to ACTIVE.
	CompletedAt string `json:"completedAt"` // RFC3339 UTC

	// VariantsQueued is true when a file.image.process.requested message was published
	// and a file_job{status=PUBLISHED} row was inserted.
	// false means the purpose has no variants (DOCUMENT, IMPORT_FILE, etc.).
	VariantsQueued bool `json:"variantsQueued"`
}
