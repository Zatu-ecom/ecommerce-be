package service

import (
	"context"
	"net/url"
	"strings"
	"time"

	"ecommerce-be/common"
	commonError "ecommerce-be/common/error"
	"ecommerce-be/common/log"
	"ecommerce-be/file/entity"
	fileError "ecommerce-be/file/error"
	"ecommerce-be/file/model"
	"ecommerce-be/file/repository"
	"ecommerce-be/file/service/blobAdapter"
	"ecommerce-be/file/utils"
	"ecommerce-be/file/utils/constant"
)

// FileReadService exposes the business operations for file metadata retrieval.
type FileReadService interface {
	GetAllFiles(
		ctx context.Context,
		caller utils.Principal,
		filter model.GetFilesFilter,
	) (*model.GetFilesResponse, error)

	GetFile(
		ctx context.Context,
		caller utils.Principal,
		fileID string,
		query model.GetFileQuery,
	) (*model.GetFileResponse, error)

	GetDownloadURL(
		ctx context.Context,
		caller utils.Principal,
		fileID string,
		query model.DownloadURLQuery,
	) (*model.DownloadURLResponse, error)
}

type fileReadService struct {
	repo       repository.FileUploadRepository
	configRepo repository.ConfigRepository
}

func NewFileReadService(
	repo repository.FileUploadRepository,
	configRepo repository.ConfigRepository,
) FileReadService {
	return &fileReadService{
		repo:       repo,
		configRepo: configRepo,
	}
}

func (s *fileReadService) GetAllFiles(
	ctx context.Context,
	caller utils.Principal,
	filter model.GetFilesFilter,
) (*model.GetFilesResponse, error) {
	filter.SetDefaults()
	if appErr := validateGetFilesFilter(filter); appErr != nil {
		return nil, appErr
	}

	items, total, err := s.repo.FindManyScoped(
		ctx,
		caller.OwnerType,
		ownerIDForPrincipal(caller),
		filter,
	)
	if err != nil {
		return nil, err
	}

	var variantMap map[uint64][]entity.FileVariant
	if filter.IncludeVariants && len(items) > 0 {
		fileObjectIDs := make([]uint64, 0, len(items))
		for _, item := range items {
			fileObjectIDs = append(fileObjectIDs, uint64(item.ID))
		}
		variants, err := s.repo.FindVariantsByFileObjectIDs(ctx, fileObjectIDs)
		if err != nil {
			return nil, err
		}
		variantMap = groupVariantsByFileObjectID(variants)
	}

	storageProviders, err := s.resolveStorageProviders(ctx, items)
	if err != nil {
		return nil, err
	}

	configs := make(map[uint64]*entity.StorageConfig)

	responseItems := make([]model.FileItem, 0, len(items))
	for _, item := range items {
		fileItem := buildFileItem(
			item,
			storageProviders[item.StorageConfigID],
			variantMap[uint64(item.ID)],
			filter.IncludeVariants,
		)

		if filter.IncludeDownloadURL && item.Status == entity.FileStatusActive {
			cfg, ok := configs[item.StorageConfigID]
			if !ok {
				if c, err := s.configRepo.GetConfigByID(ctx, uint(item.StorageConfigID)); err == nil {
					configs[item.StorageConfigID] = c
					cfg = c
				}
			}

			if cfg != nil {
				if item.Visibility == entity.FileVisibilityPublic {
					if urlStr, err := s.buildPublicURL(ctx, cfg, item.BucketOrContainer, item.ObjectKey); err == nil {
						fileItem.DownloadURL = &urlStr
					}
				} else if item.Visibility == entity.FileVisibilityPrivate {
					if presigned, err := s.buildDownloadURL(ctx, cfg, &item, constant.DefaultDownloadURLTTLMinutes); err == nil {
						fileItem.DownloadURL = &presigned.URL
						expiresAt := presigned.ExpiresAt.UTC().Format(time.RFC3339)
						fileItem.DownloadURLExpiresAt = &expiresAt
					}
				}
			}
		}

		responseItems = append(responseItems, fileItem)
	}

	return &model.GetFilesResponse{
		Items:      responseItems,
		Pagination: common.NewPaginationResponse(filter.Page, filter.PageSize, total),
	}, nil
}

func (s *fileReadService) GetFile(
	ctx context.Context,
	caller utils.Principal,
	fileID string,
	query model.GetFileQuery,
) (*model.GetFileResponse, error) {
	if strings.TrimSpace(fileID) == "" {
		return nil, fileError.ErrFileNotFound
	}
	if query.IncludeDownloadURL {
		ttl := query.ResolveURLTTLMinutes()
		if ttl < constant.MinDownloadURLTTLMinutes || ttl > constant.MaxDownloadURLTTLMinutes {
			return nil, commonError.NewAppError(
				commonError.ErrValidation.Code,
				"urlTtlMinutes must be between 5 and 60",
				commonError.ErrValidation.StatusCode,
			)
		}
	}

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

	variants, err := s.repo.FindVariantsByFileObjectIDs(ctx, []uint64{uint64(row.ID)})
	if err != nil {
		return nil, err
	}

	cfg, err := s.configRepo.GetConfigByID(ctx, uint(row.StorageConfigID))
	if err != nil {
		return nil, err
	}

	fileItem := buildFileItem(*row, storageProviderLabel(cfg.Provider.AdapterType), variants, true)
	response := &fileItem

	if query.IncludeDownloadURL && row.Status == entity.FileStatusActive {
		if row.Visibility == entity.FileVisibilityPublic {
			urlStr, err := s.buildPublicURL(ctx, cfg, row.BucketOrContainer, row.ObjectKey)
			if err != nil {
				log.WarnWithContext(
					ctx,
					"file read get_file direct mode"+
						" action=getFile"+
						" fileId="+row.FileID+
						" reason="+err.Error(),
				)
			} else {
				response.DownloadURL = &urlStr
			}
		} else if row.Visibility == entity.FileVisibilityPrivate {
			presigned, err := s.buildDownloadURL(ctx, cfg, row, query.ResolveURLTTLMinutes())
			if err != nil {
				log.WarnWithContext(
					ctx,
					"file read get_file presign degraded mode"+
						" action=getFile"+
						" fileId="+row.FileID+
						" reason="+err.Error(),
				)
			} else {
				response.DownloadURL = &presigned.URL
				expiresAt := presigned.ExpiresAt.UTC().Format(time.RFC3339)
				response.DownloadURLExpiresAt = &expiresAt
			}
		}
	}

	return response, nil
}


func (s *fileReadService) GetDownloadURL(
	ctx context.Context,
	caller utils.Principal,
	fileID string,
	query model.DownloadURLQuery,
) (*model.DownloadURLResponse, error) {
	if strings.TrimSpace(fileID) == "" {
		return nil, fileError.ErrFileNotFound
	}

	ttl := query.ResolveTTLMinutes()
	if ttl < constant.MinDownloadURLTTLMinutes || ttl > constant.MaxDownloadURLTTLMinutes {
		return nil, commonError.NewAppError(
			commonError.ErrValidation.Code,
			"ttlMinutes must be between 5 and 60",
			commonError.ErrValidation.StatusCode,
		)
	}

	disposition := query.ResolveDisposition()
	if disposition != constant.DownloadDispositionInline &&
		disposition != constant.DownloadDispositionAttachment {
		return nil, commonError.NewAppError(
			commonError.ErrValidation.Code,
			"disposition must be one of: inline, attachment",
			commonError.ErrValidation.StatusCode,
		)
	}

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
	if row.Status != entity.FileStatusActive {
		return nil, fileError.ErrFileNotActive
	}

	targetBucket := row.BucketOrContainer
	targetKey := row.ObjectKey
	targetMimeType := row.MimeType
	targetSizeBytes := row.SizeBytes

	var responseVariantCode *string
	if query.VariantCode != nil && strings.TrimSpace(*query.VariantCode) != "" {
		variantCode := strings.TrimSpace(*query.VariantCode)
		variant, err := s.repo.FindVariantByCode(ctx, uint64(row.ID), variantCode)
		if err != nil {
			return nil, err
		}
		if variant == nil {
			return nil, fileError.ErrVariantNotFound
		}
		if strings.ToUpper(variant.Status) != "READY" {
			return nil, fileError.ErrVariantNotReady
		}

		responseVariantCode = &variantCode
		targetBucket = variant.BucketOrContainer
		targetKey = variant.ObjectKey
		targetMimeType = variant.MimeType
		targetSizeBytes = variant.SizeBytes
	}

	cfg, err := s.configRepo.GetConfigByID(ctx, uint(row.StorageConfigID))
	if err != nil {
		return nil, err
	}

	presigned, err := s.buildStandaloneDownloadURL(
		ctx,
		cfg,
		targetBucket,
		targetKey,
		disposition,
		ttl,
	)
	if err != nil {
		return nil, err
	}

	return &model.DownloadURLResponse{
		FileID:      row.FileID,
		VariantCode: responseVariantCode,
		DownloadURL: presigned.URL,
		ExpiresAt:   presigned.ExpiresAt.UTC().Format(time.RFC3339),
		TTLMinutes:  ttl,
		MimeType:    targetMimeType,
		SizeBytes:   targetSizeBytes,
	}, nil
}

func (s *fileReadService) buildDownloadURL(
	ctx context.Context,
	cfg *entity.StorageConfig,
	row *entity.FileObject,
	ttlMinutes int,
) (model.BlobPresignOutput, error) {
	adapter, err := blobAdapter.GetAdapterFromStoredConfig(
		ctx,
		cfg.Provider.AdapterType,
		cfg.ConfigData,
	)
	if err != nil {
		return model.BlobPresignOutput{}, err
	}

	presigned, err := adapter.PresignDownload(ctx, model.BlobPresignDownloadInput{
		Bucket:      row.BucketOrContainer,
		Key:         row.ObjectKey,
		Disposition: constant.DownloadDispositionInline,
		TTL:         time.Duration(ttlMinutes) * time.Minute,
	})
	if err != nil {
		if blobAdapterErrorNeedsReadMapping(err) {
			return model.BlobPresignOutput{}, fileError.ErrStorageUnavailable
		}
		return model.BlobPresignOutput{}, err
	}

	return presigned, nil
}

func (s *fileReadService) buildStandaloneDownloadURL(
	ctx context.Context,
	cfg *entity.StorageConfig,
	bucket string,
	key string,
	disposition string,
	ttlMinutes int,
) (model.BlobPresignOutput, error) {
	adapter, err := blobAdapter.GetAdapterFromStoredConfig(
		ctx,
		cfg.Provider.AdapterType,
		cfg.ConfigData,
	)
	if err != nil {
		return model.BlobPresignOutput{}, err
	}

	presigned, err := adapter.PresignDownload(ctx, model.BlobPresignDownloadInput{
		Bucket:      bucket,
		Key:         key,
		Disposition: disposition,
		TTL:         time.Duration(ttlMinutes) * time.Minute,
	})
	if err != nil {
		if fileError.IsBlobError(err, fileError.ErrBlobPermissionDenied) {
			return model.BlobPresignOutput{}, fileError.ErrStoragePermissionDenied
		}
		if blobAdapterErrorNeedsReadMapping(err) {
			return model.BlobPresignOutput{}, fileError.ErrStorageUnavailable
		}
		return model.BlobPresignOutput{}, err
	}

	return presigned, nil
}

func (s *fileReadService) buildPublicURL(
	ctx context.Context,
	cfg *entity.StorageConfig,
	bucket string,
	key string,
) (string, error) {
	presigned, err := s.buildStandaloneDownloadURL(
		ctx,
		cfg,
		bucket,
		key,
		constant.DownloadDispositionInline,
		1,
	)
	if err != nil {
		return "", err
	}

	u, err := url.Parse(presigned.URL)
	if err != nil {
		return presigned.URL, nil
	}
	u.RawQuery = ""
	return u.String(), nil
}

func (s *fileReadService) resolveStorageProviders(
	ctx context.Context,
	items []entity.FileObject,
) (map[uint64]string, error) {
	providers := make(map[uint64]string, len(items))
	for _, item := range items {
		if _, exists := providers[item.StorageConfigID]; exists {
			continue
		}
		cfg, err := s.configRepo.GetConfigByID(ctx, uint(item.StorageConfigID))
		if err != nil {
			return nil, err
		}
		providers[item.StorageConfigID] = storageProviderLabel(cfg.Provider.AdapterType)
	}
	return providers, nil
}

func buildFileItem(
	row entity.FileObject,
	storageProvider string,
	variants []entity.FileVariant,
	includeVariants bool,
) model.FileItem {
	variantItems := []model.FileVariantItem{}
	if includeVariants {
		variantItems = mapVariantItems(variants)
	}

	return model.FileItem{
		FileID:           row.FileID,
		Status:           string(row.Status),
		Purpose:          string(row.Purpose),
		Visibility:       string(row.Visibility),
		OriginalFilename: row.OriginalFilename,
		MimeType:         row.MimeType,
		SizeBytes:        row.SizeBytes,
		Etag:             row.Etag,
		ObjectKey:        row.ObjectKey,
		StorageProvider:  storageProvider,
		CreatedAt:        model.FormatTimestamp(row.CreatedAt),
		CompletedAt:      model.FormatTimestampPtr(row.CompletedAt),
		Variants:         variantItems,
	}
}


func mapVariantItems(variants []entity.FileVariant) []model.FileVariantItem {
	items := make([]model.FileVariantItem, 0, len(variants))
	for _, variant := range variants {
		items = append(items, model.FileVariantItem{
			VariantCode: variant.VariantCode,
			MimeType:    variant.MimeType,
			SizeBytes:   variant.SizeBytes,
			Width:       variant.Width,
			Height:      variant.Height,
			Status:      variant.Status,
		})
	}
	return items
}

func groupVariantsByFileObjectID(variants []entity.FileVariant) map[uint64][]entity.FileVariant {
	grouped := make(map[uint64][]entity.FileVariant)
	for _, variant := range variants {
		grouped[variant.FileObjectID] = append(grouped[variant.FileObjectID], variant)
	}
	return grouped
}

func storageProviderLabel(adapterType entity.AdapterType) string {
	switch adapterType {
	case entity.AdapterTypeS3Compatible:
		return "S3"
	case entity.AdapterTypeGCS:
		return "GCS"
	case entity.AdapterTypeAzure:
		return "AZURE"
	default:
		return strings.ToUpper(string(adapterType))
	}
}

func validateGetFilesFilter(filter model.GetFilesFilter) *commonError.AppError {
	if len(filter.FileIDs) > 100 {
		return commonError.NewAppError(
			commonError.ErrValidation.Code,
			"fileIds must contain at most 100 entries",
			commonError.ErrValidation.StatusCode,
		)
	}

	for _, purpose := range filter.Purposes {
		if !isValidFilePurpose(purpose) {
			return commonError.NewAppError(
				commonError.ErrValidation.Code,
				"purposes contains an invalid value",
				commonError.ErrValidation.StatusCode,
			)
		}
	}
	for _, status := range filter.Statuses {
		if !isValidFileStatus(status) {
			return commonError.NewAppError(
				commonError.ErrValidation.Code,
				"statuses contains an invalid value",
				commonError.ErrValidation.StatusCode,
			)
		}
	}
	if !isValidFileSortBy(filter.SortBy) {
		return commonError.NewAppError(
			commonError.ErrValidation.Code,
			"sortBy contains an invalid value",
			commonError.ErrValidation.StatusCode,
		)
	}
	if !isValidSortOrder(filter.SortOrder) {
		return commonError.NewAppError(
			commonError.ErrValidation.Code,
			"sortOrder contains an invalid value",
			commonError.ErrValidation.StatusCode,
		)
	}

	return nil
}

func isValidFilePurpose(value string) bool {
	switch entity.FilePurpose(value) {
	case entity.FilePurposeProductImage,
		entity.FilePurposeImportFile,
		entity.FilePurposeExportFile,
		entity.FilePurposeDocument,
		entity.FilePurposeUserAvatar,
		entity.FilePurposeSellerLogo,
		entity.FilePurposeInvoicePDF:
		return true
	default:
		return false
	}
}

func isValidFileStatus(value string) bool {
	switch entity.FileStatus(value) {
	case entity.FileStatusUploading, entity.FileStatusActive, entity.FileStatusFailed:
		return true
	default:
		return false
	}
}

func isValidFileSortBy(value string) bool {
	switch value {
	case "createdAt", "sizeBytes", "originalFilename":
		return true
	default:
		return false
	}
}

func isValidSortOrder(value string) bool {
	switch strings.ToLower(value) {
	case "asc", "desc":
		return true
	default:
		return false
	}
}

func ownerIDForPrincipal(caller utils.Principal) *uint64 {
	if caller.OwnerType == entity.OwnerTypeSeller && caller.SellerID != nil {
		v := *caller.SellerID
		return &v
	}
	return nil
}

func blobAdapterErrorNeedsReadMapping(err error) bool {
	return fileError.IsBlobError(err, fileError.ErrBlobNetwork) ||
		fileError.IsBlobError(err, fileError.ErrBlobPermissionDenied) ||
		fileError.IsBlobError(err, fileError.ErrBlobInternal)
}
