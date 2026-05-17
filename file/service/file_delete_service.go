package service

import (
	"context"
	"time"

	"ecommerce-be/common/log"
	"ecommerce-be/file/entity"
	fileError "ecommerce-be/file/error"
	"ecommerce-be/file/model"
	"ecommerce-be/file/repository"
	"ecommerce-be/file/service/blobAdapter"
	"ecommerce-be/file/utils"
)

// FileDeleteService exposes the business operation for hard-deleting files.
type FileDeleteService interface {
	DeleteFile(
		ctx context.Context,
		caller utils.Principal,
		fileID string,
	) (*model.DeleteFileResponse, error)
}

type fileDeleteService struct {
	repo       repository.FileUploadRepository
	configRepo repository.ConfigRepository
	scheduler  UploadExpiryScheduler
}

func NewFileDeleteService(
	repo repository.FileUploadRepository,
	configRepo repository.ConfigRepository,
	scheduler UploadExpiryScheduler,
) FileDeleteService {
	return &fileDeleteService{
		repo:       repo,
		configRepo: configRepo,
		scheduler:  scheduler,
	}
}

func (s *fileDeleteService) DeleteFile(
	ctx context.Context,
	caller utils.Principal,
	fileID string,
) (*model.DeleteFileResponse, error) {
	row, err := s.repo.FindByFileIDScoped(
		ctx,
		fileID,
		caller.OwnerType,
		ownerIDForPrincipal(caller),
		caller.SellerID,
	)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, fileError.ErrFileNotFound
	}

	if row.Status == entity.FileStatusUploading && s.scheduler != nil {
		if err := s.scheduler.Cancel(ctx, uint64(row.ID), row.SellerID); err != nil {
			log.WarnWithContext(
				ctx,
				"file delete scheduler cancel failed action=deleteFile fileId="+row.FileID,
			)
		}
	}

	cfg, err := s.configRepo.GetConfigByID(ctx, uint(row.StorageConfigID))
	if err != nil {
		return nil, err
	}

	adapter, err := blobAdapter.GetAdapterFromStoredConfig(ctx, cfg.Provider.AdapterType, cfg.ConfigData)
	if err != nil {
		return nil, fileError.ErrStorageUnavailable
	}

	variants, err := s.repo.FindVariantsByFileObjectIDs(ctx, []uint64{uint64(row.ID)})
	if err != nil {
		return nil, err
	}

	if err := s.deletePrimaryBlob(ctx, adapter, row); err != nil {
		return nil, err
	}

	s.deleteVariantBlobsBestEffort(ctx, adapter, variants, row.FileID)

	if err := s.repo.DeleteFileObject(ctx, uint64(row.ID)); err != nil {
		return nil, err
	}

	now := time.Now().UTC().Format(time.RFC3339)
	return &model.DeleteFileResponse{
		FileID:    row.FileID,
		DeletedAt: now,
	}, nil
}

func (s *fileDeleteService) deletePrimaryBlob(
	ctx context.Context,
	adapter blobAdapter.BlobAdapter,
	row *entity.FileObject,
) error {
	err := adapter.DeleteObject(ctx, row.BucketOrContainer, row.ObjectKey)
	if err == nil || fileError.IsBlobError(err, fileError.ErrBlobNotFound) {
		return nil
	}
	return fileError.ErrStorageUnavailable
}

func (s *fileDeleteService) deleteVariantBlobsBestEffort(
	ctx context.Context,
	adapter blobAdapter.BlobAdapter,
	variants []entity.FileVariant,
	fileID string,
) {
	for _, variant := range variants {
		err := adapter.DeleteObject(ctx, variant.BucketOrContainer, variant.ObjectKey)
		if err == nil || fileError.IsBlobError(err, fileError.ErrBlobNotFound) {
			continue
		}

		log.WarnWithContext(
			ctx,
			"file delete variant cleanup failed action=deleteFile fileId="+fileID+
				" variantCode="+variant.VariantCode,
		)
	}
}
