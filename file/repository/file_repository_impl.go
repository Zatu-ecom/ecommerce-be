package repository

import (
	"context"
	"strings"
	"time"

	"ecommerce-be/common/db"
	"ecommerce-be/file/entity"
	"ecommerce-be/file/model"

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
		Updates(map[string]any{
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
		Updates(map[string]any{
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

// FindManyScoped returns a paginated, tenant-scoped page of file_object rows.
func (r *fileUploadRepository) FindManyScoped(
	ctx context.Context,
	ownerType entity.OwnerType,
	ownerID *uint64,
	filter model.GetFilesFilter,
) ([]entity.FileObject, int64, error) {
	query := db.DB(ctx).
		Model(&entity.FileObject{}).
		Where("owner_type = ?", ownerType)

	if ownerID != nil {
		query = query.Where("owner_id = ?", *ownerID)
	} else {
		query = query.Where("owner_id IS NULL")
	}

	if len(filter.FileIDs) > 0 {
		query = query.Where("file_id IN ?", filter.FileIDs)
	}
	if len(filter.Purposes) > 0 {
		query = query.Where("purpose IN ?", filter.Purposes)
	}
	if len(filter.Statuses) > 0 {
		query = query.Where("status IN ?", filter.Statuses)
	}
	if len(filter.MimeTypes) > 0 {
		query = query.Where("mime_type IN ?", filter.MimeTypes)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	sortBy := resolveFileSortColumn(filter.SortBy)
	sortOrder := "DESC"
	if strings.ToLower(filter.SortOrder) == "asc" {
		sortOrder = "ASC"
	}

	offset := (filter.Page - 1) * filter.PageSize

	var items []entity.FileObject
	err := query.
		Order(sortBy + " " + sortOrder).
		Offset(offset).
		Limit(filter.PageSize).
		Find(&items).
		Error

	return items, total, err
}

// FindVariantsByFileObjectIDs returns all variants for the supplied file object IDs.
func (r *fileUploadRepository) FindVariantsByFileObjectIDs(
	ctx context.Context,
	fileObjectIDs []uint64,
) ([]entity.FileVariant, error) {
	if len(fileObjectIDs) == 0 {
		return []entity.FileVariant{}, nil
	}

	var variants []entity.FileVariant
	err := db.DB(ctx).
		Where("file_object_id IN ?", fileObjectIDs).
		Find(&variants).
		Error

	return variants, err
}

// FindVariantByCode fetches a single variant for a parent file object.
func (r *fileUploadRepository) FindVariantByCode(
	ctx context.Context,
	fileObjectID uint64,
	variantCode string,
) (*entity.FileVariant, error) {
	var variant entity.FileVariant
	err := db.DB(ctx).
		Where("file_object_id = ? AND variant_code = ?", fileObjectID, variantCode).
		First(&variant).
		Error
	if err != nil {
		if isNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return &variant, nil
}

// DeleteFileObject hard-deletes a file_object row by primary key.
func (r *fileUploadRepository) DeleteFileObject(ctx context.Context, id uint64) error {
	return db.DB(ctx).Delete(&entity.FileObject{}, id).Error
}

// resolveFileSortColumn maps client sort keys to safe database column names.
func resolveFileSortColumn(sortBy string) string {
	switch sortBy {
	case "sizeBytes":
		return "size_bytes"
	case "originalFilename":
		return "original_filename"
	case "createdAt":
		return "created_at"
	default:
		return "created_at"
	}
}

// isNotFound returns true for GORM's record-not-found sentinel.
func isNotFound(err error) bool {
	return err == gorm.ErrRecordNotFound
}
