package service

import (
	"context"

	"ecommerce-be/common/log"
	"ecommerce-be/product/entity"
	productError "ecommerce-be/product/error"
	"ecommerce-be/product/model"
	"ecommerce-be/product/repository"
)

// ProductMediaService defines the business operations for managing product-media
// associations. Implementation is added incrementally per user story.
type ProductMediaService interface {
	// GetMediaForProducts performs a batched lookup of media for the supplied
	// product IDs and returns a map keyed by productID, ready to embed in
	// product list/detail responses. Missing file data is silently skipped.
	// (Implemented in Phase 3 / User Story 1)
	GetMediaForProducts(
		ctx context.Context,
		productIDs []uint,
		sellerID *uint,
	) (map[uint][]model.ProductMediaResponse, error)

	// AttachMedia links an already-uploaded file to a product with optional
	// primary and display-order attributes. Validates file existence through
	// the File gateway; rejects duplicate links.
	// (Implemented in Phase 4 / User Story 2)
	AttachMedia(
		ctx context.Context,
		productID uint,
		sellerID uint,
		req model.AttachMediaRequest,
	) (*model.ProductMediaResponse, error)

	// UpdateMediaMetadata patches the primary flag and/or display order of an
	// existing product-media link.
	// (Implemented in Phase 4 / User Story 2)
	UpdateMediaMetadata(
		ctx context.Context,
		productID uint,
		fileID string,
		sellerID uint,
		req model.UpdateMediaMetadataRequest,
	) (*model.ProductMediaResponse, error)

	// RemoveMedia unlinks a file from a product, promotes a fallback primary
	// when needed, and attempts best-effort deletion of the underlying file asset.
	// (Implemented in Phase 5 / User Story 3)
	RemoveMedia(
		ctx context.Context,
		productID uint,
		fileID string,
		sellerID uint,
	) error
}

// ─── Constructor ─────────────────────────────────────────────────────────────

type productMediaService struct {
	mediaRepo   repository.ProductMediaRepository
	productRepo repository.ProductRepository
	fileGateway ProductFileGateway
}

// NewProductMediaService creates a ProductMediaService backed by the supplied
// repository and file gateway. productRepo is used for product-ownership checks
// in write operations (AttachMedia, UpdateMediaMetadata, RemoveMedia).
func NewProductMediaService(
	mediaRepo repository.ProductMediaRepository,
	productRepo repository.ProductRepository,
	fileGateway ProductFileGateway,
) ProductMediaService {
	return &productMediaService{
		mediaRepo:   mediaRepo,
		productRepo: productRepo,
		fileGateway: fileGateway,
	}
}

// ─── Shared mapping helpers ───────────────────────────────────────────────────

// MapToProductMediaResponse converts a ProductMedia entity and optional resolved
// file info into a ProductMediaResponse DTO. When fileInfo is nil the media row
// is still mapped using an empty URL so callers can decide how to handle it.
func MapToProductMediaResponse(
	media entity.ProductMedia,
	fileInfo *ProductFileInfo,
) model.ProductMediaResponse {
	resp := model.ProductMediaResponse{
		FileID:       media.FileID,
		IsPrimary:    media.IsPrimary,
		DisplayOrder: media.DisplayOrder,
	}
	if fileInfo != nil {
		resp.URL = fileInfo.DownloadURL
		resp.ThumbnailURL = fileInfo.ThumbnailURL
	}
	return resp
}

// GroupMediaByProductID partitions a flat slice of ProductMedia rows into a map
// keyed by ProductID. The caller can then look up the slice for each product in
// O(1) instead of re-scanning the full list.
func GroupMediaByProductID(rows []entity.ProductMedia) map[uint][]entity.ProductMedia {
	result := make(map[uint][]entity.ProductMedia)
	for _, row := range rows {
		result[row.ProductID] = append(result[row.ProductID], row)
	}
	return result
}

// ─── US1: View Product Media ──────────────────────────────────────────────────

// GetMediaForProducts performs a batched lookup of media for the supplied product
// IDs, resolves file info via the gateway, and returns a map keyed by productID.
// Items whose file data cannot be resolved are silently skipped so product
// responses degrade gracefully instead of failing.
func (s *productMediaService) GetMediaForProducts(
	ctx context.Context,
	productIDs []uint,
	sellerID *uint,
) (map[uint][]model.ProductMediaResponse, error) {
	if len(productIDs) == 0 {
		return map[uint][]model.ProductMediaResponse{}, nil
	}

	rows, err := s.mediaRepo.FindByProductIDs(ctx, productIDs)
	if err != nil {
		log.WarnWithContext(ctx, "product media batch lookup failed: "+err.Error())
		return map[uint][]model.ProductMediaResponse{}, nil
	}
	if len(rows) == 0 {
		return map[uint][]model.ProductMediaResponse{}, nil
	}

	// Collect unique file IDs for a single batch gateway call.
	fileIDSet := make(map[string]struct{}, len(rows))
	for i := range rows {
		fileIDSet[rows[i].FileID] = struct{}{}
	}
	fileIDs := make([]string, 0, len(fileIDSet))
	for id := range fileIDSet {
		fileIDs = append(fileIDs, id)
	}

	fileInfoMap, _ := s.fileGateway.GetFilesWithURLs(ctx, fileIDs, sellerID)

	// Group rows by product and map to DTOs, skipping items with no file data.
	grouped := GroupMediaByProductID(rows)
	result := make(map[uint][]model.ProductMediaResponse, len(grouped))
	for productID, mediaRows := range grouped {
		responses := make([]model.ProductMediaResponse, 0, len(mediaRows))
		for _, row := range mediaRows {
			fileInfo, ok := fileInfoMap[row.FileID]
			if !ok || fileInfo == nil {
				continue
			}
			responses = append(responses, MapToProductMediaResponse(row, fileInfo))
		}
		result[productID] = responses
	}

	return result, nil
}

// ─── US2: Manage Product Media Links ─────────────────────────────────────────

// AttachMedia links an already-uploaded file to a product. It enforces:
//   - Product existence and seller ownership (product.SellerID == sellerID)
//   - Duplicate prevention (same file attached to the same product)
//   - File accessibility via the ProductFileGateway
//   - Primary-flag uniqueness: if isPrimary=true, existing primary is unset first
func (s *productMediaService) AttachMedia(
	ctx context.Context,
	productID uint,
	sellerID uint,
	req model.AttachMediaRequest,
) (*model.ProductMediaResponse, error) {
	// Verify product exists and belongs to the calling seller.
	product, err := s.productRepo.FindByID(ctx, productID)
	if err != nil {
		return nil, err
	}
	if product.SellerID != sellerID {
		// Return the same error as "not found" to avoid leaking product existence.
		return nil, productError.ErrProductNotFound
	}

	// Check for duplicate before the expensive file gateway call.
	_, err = s.mediaRepo.FindByProductAndFile(ctx, productID, req.FileID)
	if err == nil {
		return nil, productError.ErrProductMediaDuplicate
	}
	// Any error other than "not found" is unexpected.
	if err != productError.ErrProductMediaNotFound {
		return nil, err
	}

	// Validate the file exists and is accessible via the file gateway.
	fileInfo, err := s.fileGateway.GetFileInfo(ctx, req.FileID, &sellerID)
	if err != nil {
		return nil, err
	}

	// Unset any existing primary before promoting this item.
	if req.IsPrimary {
		if err := s.mediaRepo.UnsetPrimary(ctx, productID); err != nil {
			return nil, err
		}
	}

	media := &entity.ProductMedia{
		ProductID:    productID,
		FileID:       req.FileID,
		IsPrimary:    req.IsPrimary,
		DisplayOrder: req.DisplayOrder,
	}
	if err := s.mediaRepo.Create(ctx, media); err != nil {
		return nil, err
	}

	resp := MapToProductMediaResponse(*media, fileInfo)
	return &resp, nil
}

// UpdateMediaMetadata patches isPrimary and/or displayOrder for an existing
// product-media link. It enforces seller ownership and primary-flag uniqueness.
func (s *productMediaService) UpdateMediaMetadata(
	ctx context.Context,
	productID uint,
	fileID string,
	sellerID uint,
	req model.UpdateMediaMetadataRequest,
) (*model.ProductMediaResponse, error) {
	// Verify product exists and belongs to the calling seller.
	product, err := s.productRepo.FindByID(ctx, productID)
	if err != nil {
		return nil, err
	}
	if product.SellerID != sellerID {
		return nil, productError.ErrProductNotFound
	}

	// Verify the product-media link exists.
	existing, err := s.mediaRepo.FindByProductAndFile(ctx, productID, fileID)
	if err != nil {
		return nil, err
	}

	// Unset any existing primary before promoting this item.
	if req.IsPrimary != nil && *req.IsPrimary {
		if err := s.mediaRepo.UnsetPrimary(ctx, productID); err != nil {
			return nil, err
		}
	}

	if err := s.mediaRepo.UpdateMetadata(ctx, existing.ID, req.IsPrimary, req.DisplayOrder); err != nil {
		return nil, err
	}

	// Re-fetch to return the latest state.
	updated, err := s.mediaRepo.FindByProductAndFile(ctx, productID, fileID)
	if err != nil {
		return nil, err
	}

	// Best-effort file URL enrichment — degrade gracefully if unavailable.
	fileInfo, _ := s.fileGateway.GetFileInfo(ctx, fileID, &sellerID)

	resp := MapToProductMediaResponse(*updated, fileInfo)
	return &resp, nil
}

// ─── US3: Remove Product Media ────────────────────────────────────────────────

// RemoveMedia unlinks a file from a product and enforces:
//   - Product existence and seller ownership
//   - Product-media link existence
//   - Primary fallback promotion when the removed item was primary
//   - Best-effort file asset cleanup: deletion failure is logged but never
//     propagated — the product-media removal is always reported as successful.
func (s *productMediaService) RemoveMedia(
	ctx context.Context,
	productID uint,
	fileID string,
	sellerID uint,
) error {
	// Verify product exists and belongs to the calling seller.
	product, err := s.productRepo.FindByID(ctx, productID)
	if err != nil {
		return err
	}
	if product.SellerID != sellerID {
		return productError.ErrProductNotFound
	}

	// Find the product-media link.
	existing, err := s.mediaRepo.FindByProductAndFile(ctx, productID, fileID)
	if err != nil {
		return err
	}

	wasPrimary := existing.IsPrimary
	linkID := existing.ID

	// Delete the product-media association row.
	if err := s.mediaRepo.Delete(ctx, linkID); err != nil {
		return err
	}

	// Promote the next-lowest-order item to primary when the removed item was
	// the designated primary. This is best-effort: if it fails the removal has
	// already succeeded so we log the anomaly and continue.
	if wasPrimary {
		if promoteErr := s.mediaRepo.PromoteFallbackPrimary(ctx, productID); promoteErr != nil {
			log.WarnWithContext(ctx,
				"failed to promote fallback primary after removing primary media"+
					" productId="+formatUint(productID)+
					" fileId="+fileID+
					" reason="+promoteErr.Error(),
			)
		}
	}

	// Attempt best-effort file asset cleanup. Any error is intentionally
	// swallowed at this layer so product-media removal always returns success.
	if cleanupErr := s.fileGateway.DeleteFile(ctx, fileID, &sellerID); cleanupErr != nil {
		log.WarnWithContext(ctx,
			"product media removed but file cleanup failed"+
				" productId="+formatUint(productID)+
				" fileId="+fileID+
				" reason="+cleanupErr.Error(),
		)
	}

	return nil
}

// formatUint converts uint to decimal string without importing strconv at call
// sites within this file.
func formatUint(n uint) string {
	if n == 0 {
		return "0"
	}
	buf := [20]byte{}
	pos := len(buf)
	for n > 0 {
		pos--
		buf[pos] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[pos:])
}
