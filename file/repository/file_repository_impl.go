package repository

import (
	"context"
	"time"

	"ecommerce-be/common/db"
	"ecommerce-be/file/entity"

	"gorm.io/gorm"
)

type fileUploadRepository struct{}

// NewFileRepository returns a production GORM-backed FileUploadRepository.
func NewFileRepository() FileUploadRepository {
	return &fileUploadRepository{}
}

// InsertUploading inserts a new file_object row with status=UPLOADING.
// The caller is responsible for beginning / committing the enclosing transaction if needed;
// this method uses the context-scoped DB (which may itself be a transaction).
func (r *fileUploadRepository) InsertUploading(
	ctx context.Context,
	obj *entity.FileObject,
) error {
	return db.DB(ctx).Create(obj).Error
}

// FindByFileIDScoped returns a file_object whose fileId matches and whose
// (owner_type, owner_id, seller_id) triple exactly matches the caller's identity.
//
// CA4: status comparisons performed via entity constants inside MarkActive / MarkFailed.
// Cross-tenant isolation: predicate includes ALL three scoping columns.
func (r *fileUploadRepository) FindByFileIDScoped(
	ctx context.Context,
	fileID string,
	ownerType entity.OwnerType,
	ownerID *uint64,
	sellerID *uint64,
) (*entity.FileObject, error) {
	var obj entity.FileObject

	q := db.DB(ctx).
		Where("file_id = ? AND owner_type = ?", fileID, ownerType)

	if ownerID != nil {
		q = q.Where("owner_id = ?", *ownerID)
	} else {
		q = q.Where("owner_id IS NULL")
	}

	if sellerID != nil {
		q = q.Where("seller_id = ?", *sellerID)
	} else {
		q = q.Where("seller_id IS NULL")
	}

	err := q.First(&obj).Error
	if err != nil {
		if isNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return &obj, nil
}

// FindByID looks up a file_object by primary key. Used by the expiry handler.
func (r *fileUploadRepository) FindByID(
	ctx context.Context,
	id uint64,
) (*entity.FileObject, error) {
	var obj entity.FileObject
	err := db.DB(ctx).First(&obj, id).Error
	if err != nil {
		if isNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return &obj, nil
}

// MarkActive transitions status from UPLOADING → ACTIVE with a conditional WHERE clause.
// CA4: entity.FileStatusUploading and entity.FileStatusActive constants used.
func (r *fileUploadRepository) MarkActive(
	ctx context.Context,
	id uint64,
	etag string,
	sizeBytes int64,
	completedAt time.Time,
) error {
	result := db.DB(ctx).
		Model(&entity.FileObject{}).
		Where("id = ? AND status = ?", id, entity.FileStatusUploading).
		Updates(map[string]interface{}{
			"status":       entity.FileStatusActive,
			"etag":         etag,
			"size_bytes":   sizeBytes,
			"completed_at": completedAt,
		})
	return result.Error
}

// MarkFailed transitions status from UPLOADING → FAILED with a reason code.
// The update is conditional — rows already ACTIVE or FAILED are left untouched.
// CA4: entity.FileStatusUploading and entity.FileStatusFailed constants used.
func (r *fileUploadRepository) MarkFailed(
	ctx context.Context,
	id uint64,
	reason string,
) error {
	return db.DB(ctx).
		Model(&entity.FileObject{}).
		Where("id = ? AND status = ?", id, entity.FileStatusUploading).
		Updates(map[string]interface{}{
			"status":         entity.FileStatusFailed,
			"failure_reason": reason,
		}).
		Error
}

// InsertFileJob inserts a new file_job row.
func (r *fileUploadRepository) InsertFileJob(
	ctx context.Context,
	job *entity.FileJob,
) error {
	return db.DB(ctx).Create(job).Error
}

// FindFileJobByFileObjectID returns the most recent file_job for a given file_object.
func (r *fileUploadRepository) FindFileJobByFileObjectID(
	ctx context.Context,
	fileObjectID uint64,
) (*entity.FileJob, error) {
	var job entity.FileJob
	err := db.DB(ctx).
		Where("file_object_id = ?", fileObjectID).
		Order("created_at DESC").
		First(&job).
		Error
	if err != nil {
		if isNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return &job, nil
}

// isNotFound returns true for GORM's record-not-found sentinel.
func isNotFound(err error) bool {
	return err == gorm.ErrRecordNotFound
}
