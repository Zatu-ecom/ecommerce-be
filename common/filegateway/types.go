package filegateway

// FileDisplayInfo is the cross-module view of an uploaded file.
// Storage-provider internals (bucket, object key, config) are excluded.
type FileDisplayInfo struct {
	FileID       string
	Status       string
	URL          string
	ThumbnailURL *string
}

// FileAssetResponse is the API response shape for a resolved file reference.
type FileAssetResponse struct {
	FileID       string  `json:"fileId"`
	URL          string  `json:"url"`
	ThumbnailURL *string `json:"thumbnailUrl,omitempty"`
}

// ToFileAssetResponse converts internal display info to an API response DTO.
func ToFileAssetResponse(info *FileDisplayInfo) *FileAssetResponse {
	if info == nil {
		return nil
	}
	return &FileAssetResponse{
		FileID:       info.FileID,
		URL:          info.URL,
		ThumbnailURL: info.ThumbnailURL,
	}
}
