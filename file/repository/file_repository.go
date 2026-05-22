package repository

import (
	"context"
	"time"

	"ecommerce-be/file/entity"
	"ecommerce-be/file/model"
)

// FileUploadRepository defines the data-access operations required by the file module.
// All status comparisons use entity constants (CA4 — never inline string literals).
type FileUploadRepository interface {
	// InsertUploading inserts a new file_object row with status=UPLOADING.
	InsertUploading(ctx context.Context, obj *entity.FileObject) error

	// FindByFileIDScoped looks up a file_object by its UUIDv7 fileId
	// enforcing tenant-level scoping: (owner_type, owner_id, seller_id).
	// Returns nil, nil when no row matches (caller should map to ErrFileUploadNotFound).
	FindByFileIDScoped(
		ctx context.Context,
		fileID string,
		ownerType entity.OwnerType,
		ownerID *uint64,
		sellerID *uint64,
	) (*entity.FileObject, error)

	// FindByID looks up a file_object by its BIGSERIAL primary key.
	// Used by the expiry handler where only the job payload's fileObjectId is available.
	// Returns nil, nil when not found.
	FindByID(ctx context.Context, id uint64) (*entity.FileObject, error)

	// MarkActive transitions a file_object row from UPLOADING → ACTIVE.
	// The update is conditional: WHERE id = ? AND status = 'UPLOADING'.
	// Returns ErrFileUploadNotFound (caller wraps) when the condition yields 0 rows.
	MarkActive(
		ctx context.Context,
		id uint64,
		etag string,
		sizeBytes int64,
		completedAt time.Time,
	) error

	// MarkFailed transitions a file_object row from UPLOADING → FAILED with a reason code.
	// The update is conditional: WHERE id = ? AND status = 'UPLOADING'.
	// Idempotent when row is already FAILED or ACTIVE.
	MarkFailed(ctx context.Context, id uint64, reason string) error

	// InsertFileJob inserts a new file_job row (status=PUBLISHED or FAILED_TO_PUBLISH).
	InsertFileJob(ctx context.Context, job *entity.FileJob) error

	// FindFileJobByFileObjectID retrieves the most recent file_job for the given file_object.
	// Returns nil, nil when no row matches.
	FindFileJobByFileObjectID(ctx context.Context, fileObjectID uint64) (*entity.FileJob, error)

	// FindManyScoped returns a paginated, tenant-scoped list of file objects.
	FindManyScoped(
		ctx context.Context,
		ownerType entity.OwnerType,
		ownerID *uint64,
		filter model.GetFilesFilter,
	) ([]entity.FileObject, int64, error)

	// FindVariantsByFileObjectIDs fetches variants for a batch of file object IDs.
	FindVariantsByFileObjectIDs(ctx context.Context, fileObjectIDs []uint64) ([]entity.FileVariant, error)

	// FindVariantByCode returns a file variant for a given parent object and variant code.
	FindVariantByCode(ctx context.Context, fileObjectID uint64, variantCode string) (*entity.FileVariant, error)

	// DeleteFileObject hard-deletes a file_object row by primary key.
	DeleteFileObject(ctx context.Context, id uint64) error
}
