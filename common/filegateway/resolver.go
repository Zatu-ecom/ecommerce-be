package filegateway

import "context"

// ResolveSingle validates and resolves a single file for write paths.
func ResolveSingle(
	ctx context.Context,
	gw FileDisplayGateway,
	fileID string,
	sellerID *uint,
) (*FileAssetResponse, error) {
	info, err := gw.GetFileInfo(ctx, fileID, sellerID)
	if err != nil {
		return nil, err
	}
	return ToFileAssetResponse(info), nil
}

// ResolveManyOrdered resolves file IDs in order, skipping missing entries.
func ResolveManyOrdered(
	ctx context.Context,
	gw FileDisplayGateway,
	fileIDs []string,
	sellerID *uint,
) []FileAssetResponse {
	if len(fileIDs) == 0 {
		return []FileAssetResponse{}
	}

	fileMap, _ := gw.GetFilesWithURLs(ctx, fileIDs, sellerID)
	result := make([]FileAssetResponse, 0, len(fileIDs))
	for _, id := range fileIDs {
		if info, ok := fileMap[id]; ok {
			if asset := ToFileAssetResponse(info); asset != nil {
				result = append(result, *asset)
			}
		}
	}
	return result
}

// ResolveOptional resolves a single optional file reference for read paths.
func ResolveOptional(
	ctx context.Context,
	gw FileDisplayGateway,
	fileID *string,
	sellerID *uint,
) *FileAssetResponse {
	if fileID == nil || *fileID == "" {
		return nil
	}
	info, err := gw.GetFileInfo(ctx, *fileID, sellerID)
	if err != nil {
		return nil
	}
	return ToFileAssetResponse(info)
}
