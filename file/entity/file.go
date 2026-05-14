package entity

import (
	"time"

	"ecommerce-be/common/db"
)

// FileVisibility represents the visibility of a file.
// In v1, PUBLIC is a logical flag only — it does not alter object ACL or presigned PUT headers.
type FileVisibility string

const (
	FileVisibilityPrivate  FileVisibility = "PRIVATE"
	FileVisibilityPublic   FileVisibility = "PUBLIC"
	FileVisibilityInternal FileVisibility = "INTERNAL"
)

// FilePurpose represents the purpose or usage of a file.
// This is a closed enumeration — init-upload rejects any value outside this set with 400 VALIDATION_FAILED.
// EXPORT_FILE is system-generated only and is also rejected by init-upload with 422 FILE_UPLOAD_POLICY_VIOLATION.
type FilePurpose string

const (
	FilePurposeProductImage FilePurpose = "PRODUCT_IMAGE" // seller product / listing images; triggers variant generation
	FilePurposeImportFile   FilePurpose = "IMPORT_FILE"   // bulk-import CSV / spreadsheet uploads
	FilePurposeExportFile   FilePurpose = "EXPORT_FILE"   // system-generated exports; NOT accepted by init-upload
	FilePurposeDocument     FilePurpose = "DOCUMENT"      // general seller documents (PDFs, scans)
	FilePurposeUserAvatar   FilePurpose = "USER_AVATAR"   // profile / avatar images; triggers variant generation
	FilePurposeSellerLogo   FilePurpose = "SELLER_LOGO"   // storefront logo; raster variants generated (SVG passthrough)
	FilePurposeInvoicePDF   FilePurpose = "INVOICE_PDF"   // seller / platform invoice documents
)

// FileStatus represents the lifecycle status of a file_object row.
// Terminal states: ACTIVE and FAILED (no soft-delete in upload feature scope).
type FileStatus string

const (
	FileStatusUploading FileStatus = "UPLOADING"
	FileStatusActive    FileStatus = "ACTIVE"
	FileStatusFailed    FileStatus = "FAILED"
)

// FileJobStatus represents the publishing status of a file_job row.
type FileJobStatus string

const (
	FileJobStatusPublished        FileJobStatus = "PUBLISHED"
	FileJobStatusFailedToPublish  FileJobStatus = "FAILED_TO_PUBLISH"
	FileJobStatusDone             FileJobStatus = "DONE"
)

// FileObject is the canonical registry row for a user-uploaded file.
// Written by init-upload (status=UPLOADING), updated by complete-upload (ACTIVE or FAILED).
// Scoped by (owner_type, owner_id, seller_id); bound to exactly one storage_config.
//
// Column alignment with data-model.md §1.1 — no metadata column.
type FileObject struct {
	db.BaseEntity

	// FileID is a UUIDv7 string exposed to clients as the canonical upload reference.
	FileID string `gorm:"column:file_id;uniqueIndex;not null;size:80"`

	// SellerID is NULL when owner_type = PLATFORM (admin upload).
	SellerID *uint64 `gorm:"column:seller_id;index:idx_file_object_seller"`

	// UploaderUserID is the actor who called init-upload; always populated.
	UploaderUserID uint64 `gorm:"column:uploader_user_id;not null"`

	// OwnerType is SELLER or PLATFORM (CHECK constraint in migration).
	OwnerType OwnerType `gorm:"column:owner_type;not null;size:20;index:idx_file_object_owner"`

	// OwnerID mirrors SellerID for SELLER; NULL for PLATFORM.
	OwnerID *uint64 `gorm:"column:owner_id;index:idx_file_object_owner"`

	// Purpose is one of FilePurpose constants; drives policy evaluation and variant generation.
	Purpose FilePurpose `gorm:"column:purpose;not null;size:40;index:idx_file_object_purpose_status"`

	// Visibility is a closed enum; in v1 PUBLIC is a logical flag only.
	Visibility FileVisibility `gorm:"column:visibility;not null;size:20"`

	// StorageConfigID is resolved at init time and is immutable thereafter.
	StorageConfigID uint64 `gorm:"column:storage_config_id;not null"`

	// BucketOrContainer is snapshotted from storage_config at init time.
	BucketOrContainer string `gorm:"column:bucket_or_container;not null;size:255"`

	// ObjectKey is the deterministic key built by upload_key_builder (research R9).
	ObjectKey string `gorm:"column:object_key;not null;size:1000"`

	// OriginalFilename is the raw client-supplied filename.
	OriginalFilename string `gorm:"column:original_filename;not null;size:255"`

	// SanitizedFilename is used inside ObjectKey.
	SanitizedFilename string `gorm:"column:sanitized_filename;not null;size:255"`

	// MimeType is client-declared at init; re-validated by HeadObject at complete.
	MimeType string `gorm:"column:mime_type;not null;size:150"`

	// SizeBytes is the expected file size declared at init; verified at complete.
	SizeBytes int64 `gorm:"column:size_bytes;not null"`

	// Etag is populated at complete from provider HeadObject response; nil until then.
	Etag *string `gorm:"column:etag;size:200"`

	// Status is UPLOADING → ACTIVE | FAILED.
	Status FileStatus `gorm:"column:status;not null;size:20;index:idx_file_object_purpose_status"`

	// FailureReason is a short machine code set when transitioning to FAILED.
	FailureReason *string `gorm:"column:failure_reason;size:150"`

	// UploadExpiresAt is init time + uploadExpiryMinutes; used as the scheduler deadline.
	UploadExpiresAt time.Time `gorm:"column:upload_expires_at;not null;index:idx_file_object_expiry"`

	// CompletedAt is set on successful complete-upload transition.
	CompletedAt *time.Time `gorm:"column:completed_at"`
}

func (FileObject) TableName() string {
	return "file_object"
}

// FileVariant represents a derived file (thumbnail, webp, optimised export).
// Rows are NOT created by this feature; the table is declared here so the
// variant-worker feature can write to it on day 1.
//
// Column alignment with data-model.md §1.2 — no metadata column.
type FileVariant struct {
	db.BaseEntity

	// FileObjectID is an FK → file_object.id (ON DELETE CASCADE in migration).
	FileObjectID uint64 `gorm:"column:file_object_id;not null;index:idx_file_variant_file_object"`

	// VariantCode e.g. "thumb_200", "thumb_600", "webp_400", "webp_1600".
	VariantCode string `gorm:"column:variant_code;not null;size:40;uniqueIndex:idx_file_variant_unique"`

	MimeType          string  `gorm:"column:mime_type;not null;size:150"`
	BucketOrContainer string  `gorm:"column:bucket_or_container;not null;size:255"`
	ObjectKey         string  `gorm:"column:object_key;not null;size:1000"`
	SizeBytes         int64   `gorm:"column:size_bytes;not null"`
	Width             *int    `gorm:"column:width"`
	Height            *int    `gorm:"column:height"`
	Status            string  `gorm:"column:status;not null;size:20"`
}

func (FileVariant) TableName() string {
	return "file_variant"
}

// FileJob tracks the async publish of a file.image.process.requested command.
// Written at complete-upload time when HasVariants=true for the file's purpose.
//
// Column alignment with data-model.md §1.3.
type FileJob struct {
	db.BaseEntity

	// FileObjectID is an FK → file_object.id (ON DELETE CASCADE in migration).
	FileObjectID uint64 `gorm:"column:file_object_id;not null;index:idx_file_job_file_object"`

	// Command is the routing key / command name (e.g. "file.image.process.requested").
	Command string `gorm:"column:command;not null;size:60;index:idx_file_job_command_status"`

	// Status is PUBLISHED | FAILED_TO_PUBLISH | DONE.
	Status FileJobStatus `gorm:"column:status;not null;size:20;index:idx_file_job_command_status"`

	// Attempts tracks how many times the command was published (usually 1).
	Attempts int `gorm:"column:attempts;not null;default:0"`

	// LastError stores the last publish error message (if any); no PII.
	LastError *string `gorm:"column:last_error;size:300"`

	// CorrelationID is inherited from the HTTP request's X-Correlation-ID header.
	// Required for distributed tracing per constitution §VI.
	CorrelationID string `gorm:"column:correlation_id;not null;size:100"`
}

func (FileJob) TableName() string {
	return "file_job"
}
