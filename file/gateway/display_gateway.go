package gateway

import (
	"context"

	commonError "ecommerce-be/common/error"
	"ecommerce-be/common/filegateway"
	"ecommerce-be/common/log"
	fileModel "ecommerce-be/file/model"
	fileService "ecommerce-be/file/service"
	"ecommerce-be/file/entity"
	fileUtils "ecommerce-be/file/utils"
	fileConstant "ecommerce-be/file/utils/constant"
)

type displayGateway struct {
	readService fileService.FileReadService
}

type lifecycleGateway struct {
	displayGateway
	deleteService fileService.FileDeleteService
}

// NewDisplayGateway returns a FileDisplayGateway backed by FileReadService.
func NewDisplayGateway(readService fileService.FileReadService) filegateway.FileDisplayGateway {
	return &displayGateway{readService: readService}
}

// NewLifecycleGateway returns a FileLifecycleGateway backed by File read and delete services.
func NewLifecycleGateway(
	readService fileService.FileReadService,
	deleteService fileService.FileDeleteService,
) filegateway.FileLifecycleGateway {
	return &lifecycleGateway{
		displayGateway: displayGateway{readService: readService},
		deleteService:  deleteService,
	}
}

func (g *displayGateway) GetFileInfo(
	ctx context.Context,
	fileID string,
	sellerID *uint,
) (*filegateway.FileDisplayInfo, error) {
	caller := buildPrincipal(sellerID)
	resp, err := g.readService.GetFile(ctx, caller, fileID, fileModel.GetFileQuery{
		IncludeDownloadURL: true,
	})
	if err != nil {
		return nil, err
	}
	if resp.DownloadURL == nil {
		return nil, commonError.ErrFileNotAccessible
	}
	return mapFileItem(resp), nil
}

func (g *displayGateway) GetFilesWithURLs(
	ctx context.Context,
	fileIDs []string,
	sellerID *uint,
) (map[string]*filegateway.FileDisplayInfo, error) {
	if len(fileIDs) == 0 {
		return map[string]*filegateway.FileDisplayInfo{}, nil
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
			"file gateway batch lookup failed fileIds="+joined+" reason="+err.Error(),
		)
		return map[string]*filegateway.FileDisplayInfo{}, nil
	}

	result := make(map[string]*filegateway.FileDisplayInfo, len(resp.Items))
	for i := range resp.Items {
		item := &resp.Items[i]
		if item.DownloadURL == nil {
			continue
		}
		result[item.FileID] = mapFileItem(item)
	}
	return result, nil
}

func (g *lifecycleGateway) DeleteFile(
	ctx context.Context,
	fileID string,
	sellerID *uint,
) error {
	caller := buildPrincipal(sellerID)
	_, err := g.deleteService.DeleteFile(ctx, caller, fileID)
	if err != nil {
		if IsFileNotFound(err) {
			return nil
		}
		return err
	}
	return nil
}

func buildPrincipal(sellerID *uint) fileUtils.Principal {
	if sellerID != nil {
		sid := uint64(*sellerID)
		return fileUtils.Principal{
			OwnerType: entity.OwnerTypeSeller,
			SellerID:  &sid,
		}
	}
	return fileUtils.Principal{OwnerType: entity.OwnerTypePlatform}
}

func mapFileItem(item *fileModel.FileItem) *filegateway.FileDisplayInfo {
	return &filegateway.FileDisplayInfo{
		FileID:       item.FileID,
		Status:       item.Status,
		URL:          *item.DownloadURL,
		ThumbnailURL: selectThumbnail(item),
	}
}

func selectThumbnail(item *fileModel.FileItem) *string {
	for _, code := range fileConstant.ThumbnailVariantCodes {
		for _, v := range item.Variants {
			if v.VariantCode == code && v.Status == "ACTIVE" {
				return nil
			}
		}
	}
	return nil
}

// IsFileNotFound reports whether err is a FILE_NOT_FOUND application error.
func IsFileNotFound(err error) bool {
	if appErr, ok := commonError.AsAppError(err); ok {
		return appErr.Code == fileConstant.FILE_NOT_FOUND_CODE
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
