package service

import (
	"context"

	commonError "ecommerce-be/common/error"
	"ecommerce-be/common/filegateway"
	fileGateway "ecommerce-be/file/gateway"
	fileService "ecommerce-be/file/service"
	productError "ecommerce-be/product/error"
)

// ProductFileInfo is the product-module view of an uploaded file.
type ProductFileInfo = filegateway.FileDisplayInfo

// ProductFileGateway defines cross-module file operations used by product-media features.
type ProductFileGateway interface {
	GetFileInfo(ctx context.Context, fileID string, sellerID *uint) (*ProductFileInfo, error)
	GetFilesWithURLs(
		ctx context.Context,
		fileIDs []string,
		sellerID *uint,
	) (map[string]*ProductFileInfo, error)
	DeleteFile(ctx context.Context, fileID string, sellerID *uint) error
}

type productFileGateway struct {
	inner filegateway.FileLifecycleGateway
}

// NewProductFileGateway returns a ProductFileGateway that delegates to the shared file gateway.
func NewProductFileGateway(
	readService fileService.FileReadService,
	deleteService fileService.FileDeleteService,
) ProductFileGateway {
	return &productFileGateway{
		inner: fileGateway.NewLifecycleGateway(readService, deleteService),
	}
}

func (g *productFileGateway) GetFileInfo(
	ctx context.Context,
	fileID string,
	sellerID *uint,
) (*ProductFileInfo, error) {
	info, err := g.inner.GetFileInfo(ctx, fileID, sellerID)
	if err != nil {
		if fileGateway.IsFileNotFound(err) || err == commonError.ErrFileNotAccessible {
			return nil, productError.ErrProductMediaInvalidFile
		}
		return nil, err
	}
	return info, nil
}

func (g *productFileGateway) GetFilesWithURLs(
	ctx context.Context,
	fileIDs []string,
	sellerID *uint,
) (map[string]*ProductFileInfo, error) {
	return g.inner.GetFilesWithURLs(ctx, fileIDs, sellerID)
}

func (g *productFileGateway) DeleteFile(
	ctx context.Context,
	fileID string,
	sellerID *uint,
) error {
	return g.inner.DeleteFile(ctx, fileID, sellerID)
}
