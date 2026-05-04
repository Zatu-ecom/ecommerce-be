package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"ecommerce-be/common/log"
	"ecommerce-be/file/entity"
	fileError "ecommerce-be/file/error"
	"ecommerce-be/file/repository"
	"ecommerce-be/file/service/blobAdapter"
	"ecommerce-be/file/utils/constant"
)

// UploadExpiryHandler implements the scheduler.Handler interface for the
// "file.upload.expiry" command (research R4, contract/file.upload.expiry.job.md).
//
// Semantics:
//   - Idempotent: no-op if row is ACTIVE, FAILED, or missing (FR-029).
//   - If status == UPLOADING and upload_expires_at <= now(): best-effort DeleteObject
//     then MarkFailed(reason=UPLOAD_EXPIRED).
//   - Provider errors during DeleteObject are logged but do NOT prevent the status transition.
type UploadExpiryHandler struct {
	fileRepo   repository.FileUploadRepository
	configRepo repository.ConfigRepository
}

// NewUploadExpiryHandler creates a handler wired to the given repositories.
func NewUploadExpiryHandler(
	fileRepo repository.FileUploadRepository,
	configRepo repository.ConfigRepository,
) *UploadExpiryHandler {
	return &UploadExpiryHandler{
		fileRepo:   fileRepo,
		configRepo: configRepo,
	}
}

// Handle is called by the common/scheduler worker pool when a "file.upload.expiry" job fires.
// It accepts a plain context.Context + raw JSON payload (matching scheduler.Handler signature).
func (h *UploadExpiryHandler) Handle(ctx context.Context, rawPayload json.RawMessage) error {
	var payload UploadExpiryPayload
	if err := json.Unmarshal(rawPayload, &payload); err != nil {
		return fmt.Errorf("upload expiry handler: unmarshal payload: %w", err)
	}

	// Add correlationID to log context (CA3).
	log.InfoWithContext(ctx,
		fmt.Sprintf("upload expiry handler: processing fileObjectId=%d fileId=%s correlationId=%s",
			payload.FileObjectID, payload.FileID, payload.CorrelationID),
	)

	row, err := h.fileRepo.FindByID(ctx, payload.FileObjectID)
	if err != nil {
		return fmt.Errorf("upload expiry handler: FindByID: %w", err)
	}

	// No row — already deleted or never committed; idempotent no-op (FR-029).
	if row == nil {
		log.InfoWithContext(
			ctx,
			fmt.Sprintf(
				"upload expiry handler: row not found, skipping fileObjectId=%d",
				payload.FileObjectID,
			),
		)
		return nil
	}

	// Already in a terminal state — idempotent no-op (FR-029).
	if row.Status == entity.FileStatusActive || row.Status == entity.FileStatusFailed {
		log.InfoWithContext(ctx,
			fmt.Sprintf("upload expiry handler: row already %s, skipping fileObjectId=%d",
				row.Status, payload.FileObjectID),
		)
		return nil
	}

	// Race guard: if somehow the row is UPLOADING but not yet past expiry
	// (e.g. scheduler fired slightly early), do nothing. The scheduler will
	// not re-fire, so this is a best-effort safety check.
	if row.UploadExpiresAt.After(time.Now()) {
		log.InfoWithContext(
			ctx,
			fmt.Sprintf(
				"upload expiry handler: row not yet expired (expires=%s), skipping fileObjectId=%d",
				row.UploadExpiresAt,
				payload.FileObjectID,
			),
		)
		return nil
	}

	// Best-effort: delete the stray object from storage.
	deleted := h.tryDeleteObject(ctx, row)

	// Transition to FAILED regardless of delete outcome (contract guarantee).
	if err := h.fileRepo.MarkFailed(ctx, uint64(row.ID), constant.FailureReasonUploadExpired); err != nil {
		return fmt.Errorf("upload expiry handler: MarkFailed: %w", err)
	}

	log.InfoWithContext(ctx,
		fmt.Sprintf(
			"upload expiry handler: marked FAILED fileObjectId=%d fileId=%s deletedObject=%v correlationId=%s",
			payload.FileObjectID,
			payload.FileID,
			deleted,
			payload.CorrelationID,
		),
	)

	return nil
}

// tryDeleteObject attempts to delete the stray provider object.
// Returns true if deletion succeeded or the object was not found (already cleaned up).
// Errors are logged but not propagated — the row transitions to FAILED regardless.
func (h *UploadExpiryHandler) tryDeleteObject(ctx context.Context, row *entity.FileObject) bool {
	cfg, err := h.configRepo.GetConfigByID(ctx, uint(row.StorageConfigID))
	if err != nil {
		log.InfoWithContext(ctx,
			fmt.Sprintf("upload expiry handler: cannot load configId=%d: %v",
				row.StorageConfigID, err),
		)
		return false
	}
	adapter, err := blobAdapter.GetAdapterFromStoredConfig(ctx, cfg.Provider.AdapterType, cfg.ConfigData)
	if err != nil {
		log.InfoWithContext(
			ctx,
			fmt.Sprintf(
				"upload expiry handler: cannot load blob adapter configId=%d: %v (will still mark failed)",
				row.StorageConfigID,
				fileError.ErrFileUploadStorageUnavailable.Message,
			),
		)
		return false
	}
	if adapter == nil {
		return false
	}

	if err := adapter.DeleteObject(ctx, row.BucketOrContainer, row.ObjectKey); err != nil {
		log.InfoWithContext(ctx,
			fmt.Sprintf(
				"upload expiry handler: DeleteObject failed bucket=%s key=%s (will still mark failed)",
				row.BucketOrContainer,
				row.ObjectKey,
			),
		)
		return false
	}
	return true
}
