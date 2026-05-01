package service

import (
	"context"
	"strings"
	"time"

	"ecommerce-be/common/auth"
	"ecommerce-be/common/db"
	commonError "ecommerce-be/common/error"
	"ecommerce-be/common/log"
	"ecommerce-be/file/entity"
	fileError "ecommerce-be/file/error"
	"ecommerce-be/file/factory"
	"ecommerce-be/file/model"
	"ecommerce-be/file/repository"
	"ecommerce-be/file/service/blobAdapter"
	"ecommerce-be/file/utils"
	"ecommerce-be/file/utils/constant"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// FileUploadService exposes the business operations for file uploads.
type FileUploadService interface {
	InitUpload(
		ctx context.Context,
		caller utils.Principal,
		req model.InitUploadRequest,
		idempotencyKey *string,
	) (*model.InitUploadData, error)

	CompleteUpload(
		ctx context.Context,
		caller utils.Principal,
		req model.CompleteUploadRequest,
	) (*model.CompleteUploadData, error)
}

type fileUploadService struct {
	repo        repository.FileUploadRepository
	configRepo  repository.ConfigRepository
	scheduler   UploadExpiryScheduler
	publisher   VariantPublisher
	redisClient *redis.Client
}

func NewFileUploadService(
	repo repository.FileUploadRepository,
	configRepo repository.ConfigRepository,
	scheduler UploadExpiryScheduler,
	publisher VariantPublisher,
	redisClient *redis.Client,
) FileUploadService {
	return &fileUploadService{
		repo:        repo,
		configRepo:  configRepo,
		scheduler:   scheduler,
		publisher:   publisher,
		redisClient: redisClient,
	}
}

// initUploadArtifacts holds values computed before the init-upload DB transaction.
type initUploadArtifacts struct {
	fileID            string
	sanitizedFilename string
	objectKey         string
	uploadExpiresAt   time.Time
	expiryMinutes     int
}

func applyInitUploadDefaults(req *model.InitUploadRequest) {
	if req.Visibility == "" {
		req.Visibility = entity.FileVisibilityPrivate
	}
}

func resolveInitUploadExpiryMinutes(req model.InitUploadRequest) (int, *commonError.AppError) {
	expiryMinutes := constant.DefaultUploadExpiryMinutes
	if req.UploadExpiryMinutes != nil {
		expiryMinutes = *req.UploadExpiryMinutes
	}
	if expiryMinutes < constant.MinUploadExpiryMinutes ||
		expiryMinutes > constant.MaxUploadExpiryMinutes {
		return 0, fileError.ErrFileUploadInvalidInput.WithMessage(
			"uploadExpiryMinutes must be between 5 and 60",
		)
	}
	return expiryMinutes, nil
}

func validateInitUploadPolicy(req model.InitUploadRequest) *commonError.AppError {
	_, appErr := utils.Evaluate(req.Purpose, req.MimeType, req.SizeBytes)
	return appErr
}

func prepareInitUploadArtifacts(
	caller utils.Principal,
	req model.InitUploadRequest,
	expiryMinutes int,
) (initUploadArtifacts, error) {
	fileUUID, err := uuid.NewV7()
	if err != nil {
		return initUploadArtifacts{}, fileError.ErrFileUploadInternal.WithMessage(
			"failed to generate fileId",
		)
	}
	fileID := fileUUID.String()
	now := time.Now().UTC()
	sanitizedFilename := utils.SanitizeFilename(req.Filename)
	objectKey := utils.BuildObjectKey(
		caller.OwnerType,
		caller.SellerID,
		req.Purpose,
		now,
		fileID,
		sanitizedFilename,
	)
	uploadExpiresAt := now.Add(time.Duration(expiryMinutes) * time.Minute)
	return initUploadArtifacts{
		fileID:            fileID,
		sanitizedFilename: sanitizedFilename,
		objectKey:         objectKey,
		uploadExpiresAt:   uploadExpiresAt,
		expiryMinutes:     expiryMinutes,
	}, nil
}

func (s *fileUploadService) finalizeInitUploadInTx(
	txCtx context.Context,
	caller utils.Principal,
	req model.InitUploadRequest,
	cfg *entity.StorageConfig,
	adapter blobAdapter.BlobAdapter,
	artifacts initUploadArtifacts,
) (*model.InitUploadData, error) {
	obj := factory.BuildInitFileObject(
		caller,
		req,
		cfg,
		artifacts.fileID,
		artifacts.objectKey,
		artifacts.sanitizedFilename,
		artifacts.uploadExpiresAt,
	)

	if err := s.repo.InsertUploading(txCtx, obj); err != nil {
		return nil, fileError.ErrFileUploadInternal.WithMessage(
			"failed to persist upload object",
		)
	}

	presigned, err := adapter.PresignUpload(txCtx, model.BlobPresignUploadInput{
		Bucket:             cfg.BucketOrContainer,
		Key:                artifacts.objectKey,
		ContentType:        req.MimeType,
		ContentLengthLimit: req.SizeBytes,
		TTL:                time.Duration(artifacts.expiryMinutes) * time.Minute,
	})
	if err != nil {
		return nil, fileError.ErrFileUploadStorageUnavailable
	}

	correlationID, _ := auth.GetCorrelationIDFromContext(txCtx)
	if _, err := s.scheduler.Schedule(
		txCtx,
		uint64(obj.ID),
		artifacts.fileID,
		caller.SellerID,
		artifacts.uploadExpiresAt,
		correlationID,
	); err != nil {
		return nil, fileError.ErrFileUploadStorageUnavailable
	}

	return factory.BuildInitUploadData(
		artifacts.fileID,
		req.MimeType,
		artifacts.objectKey,
		presigned,
	), nil
}

// InitUpload initializes a new file upload, returning a presigned URL.
func (s *fileUploadService) InitUpload(
	ctx context.Context,
	caller utils.Principal,
	req model.InitUploadRequest,
	idempotencyKey *string,
) (*model.InitUploadData, error) {
	_ = idempotencyKey // implemented in US5a

	applyInitUploadDefaults(&req)

	expiryMinutes, appErr := resolveInitUploadExpiryMinutes(req)
	if appErr != nil {
		return nil, appErr
	}

	// Policy check must run before config resolution and DB writes (US3 guard).
	if appErr := validateInitUploadPolicy(req); appErr != nil {
		return nil, appErr
	}

	cfg, appErr := s.resolveStorageConfig(ctx, caller)
	if appErr != nil {
		return nil, appErr
	}

	adapter, err := blobAdapter.NewAdapterFromConfig(ctx, *cfg)
	if err != nil {
		return nil, fileError.ErrFileUploadStorageUnavailable
	}

	artifacts, err := prepareInitUploadArtifacts(caller, req, expiryMinutes)
	if err != nil {
		return nil, err
	}

	return db.WithTransactionResult(
		ctx,
		func(txCtx context.Context) (*model.InitUploadData, error) {
			return s.finalizeInitUploadInTx(txCtx, caller, req, cfg, adapter, artifacts)
		},
	)
}

func (s *fileUploadService) loadUploadForComplete(
	ctx context.Context,
	caller utils.Principal,
	fileID string,
) (*entity.FileObject, error) {
	row, err := s.repo.FindByFileIDScoped(
		ctx,
		fileID,
		caller.OwnerType,
		ownerIDForCaller(caller),
		caller.SellerID,
	)
	if err != nil {
		return nil, fileError.ErrFileUploadInternal.WithMessage("failed to fetch upload state")
	}
	if row == nil {
		return nil, fileError.ErrFileUploadNotFound
	}
	return row, nil
}

func isUploadExpiredFailure(row *entity.FileObject) bool {
	return row.Status == entity.FileStatusFailed &&
		row.FailureReason != nil &&
		*row.FailureReason == constant.FailureReasonUploadExpired
}

func (s *fileUploadService) completeUploadReplayIfActive(
	ctx context.Context,
	row *entity.FileObject,
) (*model.CompleteUploadData, bool) {
	if row.Status != entity.FileStatusActive {
		return nil, false
	}
	variantsQueued, completedAt := s.resolveReplayState(ctx, row)
	return factory.BuildCompleteUploadData(
		row.FileID,
		row.MimeType,
		row.SizeBytes,
		valueOrEmpty(row.Etag),
		completedAt,
		variantsQueued,
	), true
}

func (s *fileUploadService) headObjectForComplete(
	ctx context.Context,
	adapter blobAdapter.BlobAdapter,
	row *entity.FileObject,
) (model.BlobObjectMeta, error) {
	meta, err := adapter.HeadObject(ctx, row.BucketOrContainer, row.ObjectKey)
	if err != nil {
		if fileError.IsBlobError(err, fileError.ErrBlobNotFound) {
			return model.BlobObjectMeta{}, fileError.ErrFileUploadObjectMissing
		}
		return model.BlobObjectMeta{}, fileError.ErrFileUploadStorageUnavailable
	}
	return meta, nil
}

func (s *fileUploadService) validateCompleteObjectAgainstRow(
	ctx context.Context,
	row *entity.FileObject,
	meta model.BlobObjectMeta,
	req model.CompleteUploadRequest,
) error {
	if meta.SizeBytes != row.SizeBytes {
		_ = s.repo.MarkFailed(ctx, uint64(row.ID), constant.FailureReasonObjectMismatch)
		return fileError.ErrFileUploadObjectMismatch
	}
	if !strings.EqualFold(strings.TrimSpace(meta.ContentType), strings.TrimSpace(row.MimeType)) {
		_ = s.repo.MarkFailed(ctx, uint64(row.ID), constant.FailureReasonObjectMismatch)
		return fileError.ErrFileUploadObjectMismatch
	}
	if req.ClientEtag != nil &&
		strings.Trim(*req.ClientEtag, "\"") != strings.Trim(meta.ETag, "\"") {
		_ = s.repo.MarkFailed(ctx, uint64(row.ID), constant.FailureReasonObjectMismatch)
		return fileError.ErrFileUploadObjectMismatch
	}
	return nil
}

func (s *fileUploadService) persistCompletedUpload(
	ctx context.Context,
	row *entity.FileObject,
	meta model.BlobObjectMeta,
) (time.Time, error) {
	now := time.Now().UTC()
	if err := s.repo.MarkActive(
		ctx,
		uint64(row.ID),
		strings.Trim(meta.ETag, "\""),
		meta.SizeBytes,
		now,
	); err != nil {
		return time.Time{}, fileError.ErrFileUploadInternal.WithMessage("failed to finalize upload")
	}
	return now, nil
}

func (s *fileUploadService) cancelExpiryScheduleBestEffort(
	ctx context.Context,
	row *entity.FileObject,
) {
	if err := s.scheduler.Cancel(ctx, uint64(row.ID), row.SellerID); err != nil {
		log.WarnWithContext(ctx, "upload complete: scheduler cancel failed")
	}
}

func (s *fileUploadService) queueVariantProcessingIfApplicable(
	ctx context.Context,
	row *entity.FileObject,
	meta model.BlobObjectMeta,
) bool {
	policy, policyErr := utils.Evaluate(row.Purpose, meta.ContentType, meta.SizeBytes)
	if policyErr != nil || !policy.HasVariants {
		return false
	}

	correlationID, _ := auth.GetCorrelationIDFromContext(ctx)
	jobStatus := entity.FileJobStatusPublished
	var lastError *string
	variantsQueued := false

	if s.publisher != nil {
		publishErr := s.publisher.Publish(
			ctx,
			factory.BuildImageProcessRequested(
				row,
				meta.ContentType,
				meta.SizeBytes,
				policy.VariantCodes,
			),
			correlationID,
		)
		if publishErr != nil {
			jobStatus = entity.FileJobStatusFailedToPublish
			msg := publishErr.Error()
			lastError = &msg
		} else {
			variantsQueued = true
		}
	}

	_ = s.repo.InsertFileJob(
		ctx,
		factory.BuildFileJob(uint64(row.ID), jobStatus, lastError, correlationID),
	)
	return variantsQueued
}

// CompleteUpload finalizes an upload after the client has PUT the bytes.
func (s *fileUploadService) CompleteUpload(
	ctx context.Context,
	caller utils.Principal,
	req model.CompleteUploadRequest,
) (*model.CompleteUploadData, error) {
	row, err := s.loadUploadForComplete(ctx, caller, req.FileID)
	if err != nil {
		return nil, err
	}

	if isUploadExpiredFailure(row) {
		return nil, fileError.ErrFileUploadExpired
	}

	if replay, ok := s.completeUploadReplayIfActive(ctx, row); ok {
		return replay, nil
	}

	cfg, err := s.configRepo.GetConfigByID(ctx, uint(row.StorageConfigID))
	if err != nil {
		return nil, fileError.ErrFileUploadStorageUnavailable
	}
	adapter, err := blobAdapter.NewAdapterFromConfig(ctx, *cfg)
	if err != nil {
		return nil, fileError.ErrFileUploadStorageUnavailable
	}

	meta, err := s.headObjectForComplete(ctx, adapter, row)
	if err != nil {
		return nil, err
	}

	if err := s.validateCompleteObjectAgainstRow(ctx, row, meta, req); err != nil {
		return nil, err
	}

	now, err := s.persistCompletedUpload(ctx, row, meta)
	if err != nil {
		return nil, err
	}

	// Best effort cancellation (FR-026/FR-029).
	s.cancelExpiryScheduleBestEffort(ctx, row)

	variantsQueued := s.queueVariantProcessingIfApplicable(ctx, row, meta)

	return factory.BuildCompleteUploadData(
		row.FileID,
		meta.ContentType,
		meta.SizeBytes,
		strings.Trim(meta.ETag, "\""),
		now.Format(time.RFC3339),
		variantsQueued,
	), nil
}

func (s *fileUploadService) resolveStorageConfig(
	ctx context.Context,
	caller utils.Principal,
) (*entity.StorageConfig, *commonError.AppError) {
	if caller.OwnerType == entity.OwnerTypeSeller && caller.SellerID != nil {
		if cfg, err := s.configRepo.GetActiveSellerStorageConfig(ctx, uint(*caller.SellerID)); err == nil {
			return cfg, nil
		} else if !isNotFound(err) {
			return nil, fileError.ErrFileUploadInternal.WithMessage("failed to resolve seller storage config")
		}
	}

	cfg, err := s.configRepo.GetActivePlatformDefaultConfig(ctx)
	if err != nil {
		if isNotFound(err) {
			return nil, fileError.ErrFileUploadNoStorageConfig
		}
		return nil, fileError.ErrFileUploadInternal.WithMessage(
			"failed to resolve platform storage config",
		)
	}
	return cfg, nil
}

func (s *fileUploadService) resolveReplayState(
	ctx context.Context,
	row *entity.FileObject,
) (variantsQueued bool, completedAt string) {
	completedAt = time.Now().UTC().Format(time.RFC3339)
	if row.CompletedAt != nil {
		completedAt = row.CompletedAt.UTC().Format(time.RFC3339)
	}

	job, err := s.repo.FindFileJobByFileObjectID(ctx, uint64(row.ID))
	if err == nil && job != nil && job.Status == entity.FileJobStatusPublished {
		variantsQueued = true
	}
	return variantsQueued, completedAt
}

func ownerIDForCaller(caller utils.Principal) *uint64 {
	if caller.OwnerType == entity.OwnerTypeSeller && caller.SellerID != nil {
		v := *caller.SellerID
		return &v
	}
	return nil
}

func isNotFound(err error) bool {
	return err == gorm.ErrRecordNotFound
}

func valueOrEmpty(in *string) string {
	if in == nil {
		return ""
	}
	return *in
}
