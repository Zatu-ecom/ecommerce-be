package service

import (
	"context"

	"ecommerce-be/common/log"
	"ecommerce-be/product/entity"
	productError "ecommerce-be/product/error"
	"ecommerce-be/product/model"
	"ecommerce-be/product/repository"
)

// VariantMediaService manages the association between product variants and file
// assets owned by the File module. All binary assets must be uploaded via the
// File module first; this service only stores the association metadata.
type VariantMediaService interface {
	// GetMediaForVariants returns a map[variantID → []VariantMediaResponse] for
	// all supplied variant IDs. File data is resolved via the File gateway;
	// items whose file cannot be resolved are silently omitted.
	GetMediaForVariants(
		ctx context.Context,
		variantIDs []uint,
		sellerID *uint,
	) (map[uint][]model.VariantMediaResponse, error)

	// AttachMedia links an already-uploaded file to a variant. Validates product
	// ownership, file existence (via File gateway), and rejects duplicate links.
	AttachMedia(
		ctx context.Context,
		variantID uint,
		productID uint,
		sellerID uint,
		req model.AttachVariantMediaRequest,
	) (*model.VariantMediaResponse, error)

	// UpdateMediaMetadata patches the primary flag and/or display order of an
	// existing variant-media link.
	UpdateMediaMetadata(
		ctx context.Context,
		variantID uint,
		productID uint,
		fileID string,
		sellerID uint,
		req model.UpdateVariantMediaMetadataRequest,
	) (*model.VariantMediaResponse, error)

	// RemoveMedia unlinks a file from a variant, promotes a fallback primary
	// when needed, and attempts best-effort deletion of the underlying file
	// asset. Cleanup failures are logged but never propagated.
	RemoveMedia(
		ctx context.Context,
		variantID uint,
		productID uint,
		fileID string,
		sellerID uint,
	) error
}

type variantMediaService struct {
	mediaRepo   repository.VariantMediaRepository
	variantRepo repository.VariantRepository
	productRepo repository.ProductRepository
	fileGateway ProductFileGateway
}

// NewVariantMediaService returns a VariantMediaService backed by GORM repositories
// and the shared ProductFileGateway.
func NewVariantMediaService(
	mediaRepo repository.VariantMediaRepository,
	variantRepo repository.VariantRepository,
	productRepo repository.ProductRepository,
	fileGateway ProductFileGateway,
) VariantMediaService {
	return &variantMediaService{
		mediaRepo:   mediaRepo,
		variantRepo: variantRepo,
		productRepo: productRepo,
		fileGateway: fileGateway,
	}
}

// ─── US1-equivalent: Batch media resolution ───────────────────────────────────

func (s *variantMediaService) GetMediaForVariants(
	ctx context.Context,
	variantIDs []uint,
	sellerID *uint,
) (map[uint][]model.VariantMediaResponse, error) {
	result := make(map[uint][]model.VariantMediaResponse, len(variantIDs))
	for _, id := range variantIDs {
		result[id] = []model.VariantMediaResponse{}
	}

	rows, err := s.mediaRepo.FindByVariantIDs(ctx, variantIDs)
	if err != nil || len(rows) == 0 {
		return result, err
	}

	// Collect unique file IDs for batch resolution.
	fileIDSet := make(map[string]struct{}, len(rows))
	for _, r := range rows {
		fileIDSet[r.FileID] = struct{}{}
	}
	fileIDs := make([]string, 0, len(fileIDSet))
	for fid := range fileIDSet {
		fileIDs = append(fileIDs, fid)
	}

	fileMap, err := s.fileGateway.GetFilesWithURLs(ctx, fileIDs, sellerID)
	if err != nil {
		// Graceful degradation: return empty media slices instead of error.
		return result, nil
	}

	// Group resolved items back by variant ID.
	for _, row := range rows {
		fi, ok := fileMap[row.FileID]
		if !ok {
			continue
		}
		item := model.VariantMediaResponse{
			FileID:       row.FileID,
			URL:          fi.URL,
			IsPrimary:    row.IsPrimary,
			DisplayOrder: row.DisplayOrder,
			ThumbnailURL: fi.ThumbnailURL,
		}
		result[row.VariantID] = append(result[row.VariantID], item)
	}

	return result, nil
}

// ─── Attach media ─────────────────────────────────────────────────────────────

func (s *variantMediaService) AttachMedia(
	ctx context.Context,
	variantID uint,
	productID uint,
	sellerID uint,
	req model.AttachVariantMediaRequest,
) (*model.VariantMediaResponse, error) {
	// Verify product exists and belongs to the calling seller.
	product, err := s.productRepo.FindByID(ctx, productID)
	if err != nil {
		return nil, err
	}
	if product.SellerID != sellerID {
		return nil, productError.ErrProductNotFound
	}

	// Verify variant belongs to this product.
	variant, err := s.variantRepo.FindVariantByProductIDAndVariantID(ctx, productID, variantID)
	if err != nil {
		return nil, err
	}
	_ = variant

	// Reject duplicate links before going to the file gateway.
	_, err = s.mediaRepo.FindByVariantAndFile(ctx, variantID, req.FileID)
	if err == nil {
		return nil, productError.ErrVariantMediaDuplicate
	}
	if err != productError.ErrVariantMediaNotFound {
		return nil, err
	}

	// Validate the file exists and is accessible via the file gateway.
	fileInfo, err := s.fileGateway.GetFileInfo(ctx, req.FileID, &sellerID)
	if err != nil {
		return nil, productError.ErrVariantMediaInvalidFile
	}

	// Unset any existing primary before promoting this item.
	if req.IsPrimary {
		if err := s.mediaRepo.UnsetPrimary(ctx, variantID); err != nil {
			return nil, err
		}
	}

	media := &entity.VariantMedia{
		VariantID:    variantID,
		FileID:       req.FileID,
		IsPrimary:    req.IsPrimary,
		DisplayOrder: req.DisplayOrder,
	}
	if err := s.mediaRepo.Create(ctx, media); err != nil {
		return nil, err
	}

	return &model.VariantMediaResponse{
		FileID:       media.FileID,
		URL:          fileInfo.URL,
		IsPrimary:    media.IsPrimary,
		DisplayOrder: media.DisplayOrder,
		ThumbnailURL: fileInfo.ThumbnailURL,
	}, nil
}

// ─── Update media metadata ────────────────────────────────────────────────────

func (s *variantMediaService) UpdateMediaMetadata(
	ctx context.Context,
	variantID uint,
	productID uint,
	fileID string,
	sellerID uint,
	req model.UpdateVariantMediaMetadataRequest,
) (*model.VariantMediaResponse, error) {
	// Verify product ownership.
	product, err := s.productRepo.FindByID(ctx, productID)
	if err != nil {
		return nil, err
	}
	if product.SellerID != sellerID {
		return nil, productError.ErrProductNotFound
	}

	// Verify the variant belongs to this product.
	_, err = s.variantRepo.FindVariantByProductIDAndVariantID(ctx, productID, variantID)
	if err != nil {
		return nil, err
	}

	// Verify the media link exists.
	existing, err := s.mediaRepo.FindByVariantAndFile(ctx, variantID, fileID)
	if err != nil {
		return nil, err
	}

	// Unset existing primary if we are promoting a new one.
	if req.IsPrimary != nil && *req.IsPrimary {
		if err := s.mediaRepo.UnsetPrimary(ctx, variantID); err != nil {
			return nil, err
		}
	}

	if err := s.mediaRepo.UpdateMetadata(ctx, existing.ID, req.IsPrimary, req.DisplayOrder); err != nil {
		return nil, err
	}

	// Re-fetch the updated row for accurate response data.
	updated, err := s.mediaRepo.FindByVariantAndFile(ctx, variantID, fileID)
	if err != nil {
		return nil, err
	}

	resp := &model.VariantMediaResponse{
		FileID:       updated.FileID,
		URL:          "",
		IsPrimary:    updated.IsPrimary,
		DisplayOrder: updated.DisplayOrder,
	}

	// Best-effort URL enrichment.
	if fi, fErr := s.fileGateway.GetFileInfo(ctx, fileID, &sellerID); fErr == nil {
		resp.URL = fi.URL
		resp.ThumbnailURL = fi.ThumbnailURL
	}

	return resp, nil
}

// ─── Remove media ─────────────────────────────────────────────────────────────

func (s *variantMediaService) RemoveMedia(
	ctx context.Context,
	variantID uint,
	productID uint,
	fileID string,
	sellerID uint,
) error {
	// Verify product ownership.
	product, err := s.productRepo.FindByID(ctx, productID)
	if err != nil {
		return err
	}
	if product.SellerID != sellerID {
		return productError.ErrProductNotFound
	}

	// Verify the variant belongs to this product.
	_, err = s.variantRepo.FindVariantByProductIDAndVariantID(ctx, productID, variantID)
	if err != nil {
		return err
	}

	// Find the media link.
	existing, err := s.mediaRepo.FindByVariantAndFile(ctx, variantID, fileID)
	if err != nil {
		return err
	}

	wasPrimary := existing.IsPrimary
	linkID := existing.ID

	if err := s.mediaRepo.Delete(ctx, linkID); err != nil {
		return err
	}

	// Promote the next-lowest-order item to primary when the removed item was
	// the designated primary.
	if wasPrimary {
		if promoteErr := s.mediaRepo.PromoteFallbackPrimary(ctx, variantID); promoteErr != nil {
			log.WarnWithContext(ctx,
				"failed to promote fallback primary after removing primary variant media"+
					" variantId="+formatUint(variantID)+
					" fileId="+fileID+
					" reason="+promoteErr.Error(),
			)
		}
	}

	// Best-effort file asset cleanup.
	if cleanupErr := s.fileGateway.DeleteFile(ctx, fileID, &sellerID); cleanupErr != nil {
		log.WarnWithContext(ctx,
			"variant media removed but file cleanup failed"+
				" variantId="+formatUint(variantID)+
				" fileId="+fileID+
				" reason="+cleanupErr.Error(),
		)
	}

	return nil
}
