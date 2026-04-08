package entity

import (
	"time"

	"ecommerce-be/common/db"
)

// FileVisibility represents the visibility of a file
type FileVisibility string

const (
	FileVisibilityPrivate  FileVisibility = "PRIVATE"
	FileVisibilityPublic   FileVisibility = "PUBLIC"
	FileVisibilityInternal FileVisibility = "INTERNAL"
)

// FilePurpose represents the purpose or usage of a file
type FilePurpose string

const (
	FilePurposeProductImage FilePurpose = "PRODUCT_IMAGE"
	FilePurposeImportFile   FilePurpose = "IMPORT_FILE"
	FilePurposeExportFile   FilePurpose = "EXPORT_FILE"
	FilePurposeDocument     FilePurpose = "DOCUMENT"
)

// FileStatus represents the status of a file
type FileStatus string

const (
	FileStatusUploading FileStatus = "UPLOADING"
	FileStatusActive    FileStatus = "ACTIVE"
	FileStatusFailed    FileStatus = "FAILED"
	FileStatusDeleted   FileStatus = "DELETED"
)

// FileObject represents the main file registry table
type FileObject struct {
	db.BaseEntity
	FileID            string         `gorm:"column:file_id;unique;not null;size:80"`
	OwnerType         OwnerType      `gorm:"column:owner_type;not null;size:20;index:idx_file_object_owner"`
	OwnerID           uint           `gorm:"column:owner_id;not null;index:idx_file_object_owner"`
	SellerID          *uint          `gorm:"column:seller_id;index:idx_file_object_seller"`
	StorageConfigID   uint           `gorm:"column:storage_config_id;not null"`
	ProviderCode      string         `gorm:"column:provider_code;not null;size:50"`
	BucketOrContainer string         `gorm:"column:bucket_or_container;not null;size:255"`
	ObjectKey         string         `gorm:"column:object_key;not null;size:1000"`
	OriginalFileName  string         `gorm:"column:original_file_name;not null;size:500"`
	Extension         string         `gorm:"column:extension;size:20"`
	MimeType          string         `gorm:"column:mime_type;size:120"`
	SizeBytes         int64          `gorm:"column:size_bytes;not null"`
	ChecksumSha256    string         `gorm:"column:checksum_sha256;size:64"`
	ETag              string         `gorm:"column:e_tag;size:200"`
	Visibility        FileVisibility `gorm:"column:visibility;not null;default:'PRIVATE';size:20"`
	Purpose           FilePurpose    `gorm:"column:purpose;not null;size:50;index:idx_file_object_purpose"`
	Status            FileStatus     `gorm:"column:status;not null;default:'UPLOADING';size:30;index:idx_file_object_purpose"`
	Metadata          db.JSONMap     `gorm:"column:metadata;type:jsonb"`
	Tags              []string       `gorm:"column:tags;type:jsonb;serializer:json"`
	CreatedBy         *uint          `gorm:"column:created_by"`

	StorageConfig StorageConfig `gorm:"foreignKey:StorageConfigID"`
}

func (FileObject) TableName() string {
	return "file_object"
}

// FileVariant represents derived files (thumbnails/webp/optimized exports)
type FileVariant struct {
	db.BaseEntity
	FileObjectID      uint       `gorm:"column:file_object_id;not null;uniqueIndex:idx_file_variant_unique"`
	VariantType       string     `gorm:"column:variant_type;not null;size:50;uniqueIndex:idx_file_variant_unique"`
	BucketOrContainer string     `gorm:"column:bucket_or_container;not null;size:255"`
	ObjectKey         string     `gorm:"column:object_key;not null;size:1000"`
	MimeType          string     `gorm:"column:mime_type;size:120"`
	SizeBytes         *int64     `gorm:"column:size_bytes"`
	Width             *int       `gorm:"column:width"`
	Height            *int       `gorm:"column:height"`
	Status            string     `gorm:"column:status;not null;default:'PROCESSING';size:30"`
	Metadata          db.JSONMap `gorm:"column:metadata;type:jsonb"`

	FileObject FileObject `gorm:"foreignKey:FileObjectID"`
}

func (FileVariant) TableName() string {
	return "file_variant"
}

// FileJob represents async processing/import/export jobs
type FileJob struct {
	db.BaseEntity
	JobID           string     `gorm:"column:job_id;unique;not null;size:80"`
	SellerID        *uint      `gorm:"column:seller_id;index:idx_file_job_seller_status"`
	InitiatedBy     *uint      `gorm:"column:initiated_by"`
	JobType         string     `gorm:"column:job_type;not null;size:50"`
	Status          string     `gorm:"column:status;not null;size:30;index:idx_file_job_seller_status"`
	ProgressPercent int        `gorm:"column:progress_percent;default:0"`
	InputFileID     *uint      `gorm:"column:input_file_id"`
	OutputFileID    *uint      `gorm:"column:output_file_id"`
	ErrorCode       string     `gorm:"column:error_code;size:100"`
	ErrorMessage    string     `gorm:"column:error_message"`
	Payload         db.JSONMap `gorm:"column:payload;type:jsonb"`
	ResultJSON      db.JSONMap `gorm:"column:result_json;type:jsonb"`
	StartedAt       *time.Time `gorm:"column:started_at"`
	CompletedAt     *time.Time `gorm:"column:completed_at"`

	InputFile  *FileObject `gorm:"foreignKey:InputFileID"`
	OutputFile *FileObject `gorm:"foreignKey:OutputFileID"`
}

func (FileJob) TableName() string {
	return "file_job"
}
