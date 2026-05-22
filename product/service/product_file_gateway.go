package service

import (
	"context"

	commonError "ecommerce-be/common/error"
	"ecommerce-be/common/log"
	fileModel "ecommerce-be/file/model"
	fileService "ecommerce-be/file/service"
	fileUtils "ecommerce-be/file/utils"
	productError "ecommerce-be/product/error"
)

// ProductFileInfo is the product-module view of an uploaded file.
// It contains only the fields that the product domain needs; storage-provider
// internals (bucket, object key, config) are deliberately excluded.
type ProductFileInfo struct {
	FileID       string
	Status       string
	DownloadURL  string
	ThumbnailURL *string // nil when no thumbnail/poster variant is available
}

// ProductFileGateway defines the cross-module file operations used by product-media
// features. All methods accept a sellerID pointer so callers can pass nil for
// platform/admin contexts that are not scoped to a single seller.
//
// Implementations MUST NOT import File repositories or File persistence entities;
// they call File module service interfaces only.
type ProductFileGateway interface {
	// GetFileInfo validates that a file exists and is accessible, and returns
	// its display URL and thumbnail where available.
	GetFileInfo(ctx context.Context, fileID string, sellerID *uint) (*ProductFileInfo, error)

	// GetFilesWithURLs performs a batched lookup for the supplied fileIDs and
	// returns a map keyed by fileID. Missing or inaccessible files are silently
	// omitted so product responses degrade gracefully.
	GetFilesWithURLs(
		ctx context.Context,
		fileIDs []string,
		sellerID *uint,
	) (map[string]*ProductFileInfo, error)

	// DeleteFile attempts to delete the underlying file asset. Callers MUST
	// treat errors from this method as best-effort degradation: the product-media
	// removal has already been committed and must still be reported as successful.
	DeleteFile(ctx context.Context, fileID string, sellerID *uint) error
}

// ─── Variant codes preferred for thumbnail selection ─────────────────────────

var thumbnailVariantCodes = []string{"thumb_200", "poster"}

// ─── GORM-free implementation (wraps File module service interfaces) ──────────

type productFileGateway struct {
	readService   fileService.FileReadService
	deleteService fileService.FileDeleteService
}

// NewProductFileGateway returns a ProductFileGateway that delegates to the
// File module's read and delete services.
func NewProductFileGateway(
	readService fileService.FileReadService,
	deleteService fileService.FileDeleteService,
) ProductFileGateway {
	return &productFileGateway{
		readService:   readService,
		deleteService: deleteService,
	}
}

func (g *productFileGateway) GetFileInfo(
	ctx context.Context,
	fileID string,
	sellerID *uint,
) (*ProductFileInfo, error) {
	caller := buildPrincipal(sellerID)
	resp, err := g.readService.GetFile(ctx, caller, fileID, fileModel.GetFileQuery{
		IncludeDownloadURL: true,
	})
	if err != nil {
		if isFileNotFound(err) {
			return nil, productError.ErrProductMediaInvalidFile
		}
		return nil, err
	}
	if resp.DownloadURL == nil {
		return nil, productError.ErrProductMediaInvalidFile
	}
	info := &ProductFileInfo{
		FileID:       resp.FileID,
		Status:       resp.Status,
		DownloadURL:  *resp.DownloadURL,
		ThumbnailURL: selectThumbnail(resp),
	}
	return info, nil
}

func (g *productFileGateway) GetFilesWithURLs(
	ctx context.Context,
	fileIDs []string,
	sellerID *uint,
) (map[string]*ProductFileInfo, error) {
	if len(fileIDs) == 0 {
		return map[string]*ProductFileInfo{}, nil
	}

	caller := buildPrincipal(sellerID)
	joined := joinFileIDs(fileIDs)
	resp, err := g.readService.GetAllFiles(ctx, caller, fileModel.GetFilesFilter{
		GetFilesBase: fileModel.GetFilesBase{IncludeDownloadURL: true, IncludeVariants: true},
		FileIDs:      fileIDs,
	})
	if err != nil {
		log.WarnWithContext(
			ctx,
			"product file gateway batch lookup failed fileIds="+joined+" reason="+err.Error(),
		)
		return map[string]*ProductFileInfo{}, nil
	}

	result := make(map[string]*ProductFileInfo, len(resp.Items))
	for i := range resp.Items {
		item := &resp.Items[i]
		if item.DownloadURL == nil {
			continue
		}
		result[item.FileID] = &ProductFileInfo{
			FileID:       item.FileID,
			Status:       item.Status,
			DownloadURL:  *item.DownloadURL,
			ThumbnailURL: selectThumbnail(item),
		}
	}
	return result, nil
}

func (g *productFileGateway) DeleteFile(
	ctx context.Context,
	fileID string,
	sellerID *uint,
) error {
	caller := buildPrincipal(sellerID)
	_, err := g.deleteService.DeleteFile(ctx, caller, fileID)
	if err != nil {
		if isFileNotFound(err) {
			return nil
		}
		return err
	}
	return nil
}

// ─── Internal helpers ─────────────────────────────────────────────────────────

func buildPrincipal(sellerID *uint) fileUtils.Principal {
	if sellerID != nil {
		sid := uint64(*sellerID)
		return fileUtils.Principal{
			OwnerType: "SELLER",
			SellerID:  &sid,
		}
	}
	return fileUtils.Principal{OwnerType: "PLATFORM"}
}

func selectThumbnail(item *fileModel.FileItem) *string {
	for _, code := range thumbnailVariantCodes {
		for _, v := range item.Variants {
			if v.VariantCode == code && v.Status == "ACTIVE" {
				// Variant items carry a variant_code; the download URL is resolved
				// separately. For now we return a sentinel string so callers know a
				// variant exists; full URL resolution is done in Phase 3 when the
				// batch URL path is available.
				//
				// TODO(phase3): call GetDownloadURL per variant_code for each fileID
				// when implementing the full mapping in product_media_service.go.
				return nil
			}
		}
	}
	return nil
}

func isFileNotFound(err error) bool {
	if appErr, ok := commonError.AsAppError(err); ok {
		return appErr.Code == "FILE_NOT_FOUND"
	}
	return false
}

func joinFileIDs(ids []string) string {
	result := ""
	for i, id := range ids {
		if i > 0 {
			result += ","
		}
		result += id
	}
	return result
}
