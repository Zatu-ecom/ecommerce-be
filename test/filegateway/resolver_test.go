package filegateway_test

import (
	"context"
	"testing"

	commonError "ecommerce-be/common/error"
	"ecommerce-be/common/filegateway"
)

func TestToFileAssetResponse(t *testing.T) {
	info := &filegateway.FileDisplayInfo{
		FileID: "file-123",
		URL:    "https://example.com/a.jpg",
	}
	asset := filegateway.ToFileAssetResponse(info)
	if asset == nil || asset.FileID != "file-123" || asset.URL != info.URL {
		t.Fatalf("unexpected asset: %+v", asset)
	}
}

func TestResolveManyOrderedPreservesOrderAndSkipsMissing(t *testing.T) {
	gw := stubGateway{
		files: map[string]*filegateway.FileDisplayInfo{
			"b": {FileID: "b", URL: "https://example.com/b.jpg"},
			"c": {FileID: "c", URL: "https://example.com/c.jpg"},
		},
	}

	result := filegateway.ResolveManyOrdered(context.Background(), gw, []string{"a", "b", "c"}, nil)
	if len(result) != 2 {
		t.Fatalf("expected 2 resolved assets, got %d", len(result))
	}
	if result[0].FileID != "b" || result[1].FileID != "c" {
		t.Fatalf("unexpected order: %+v", result)
	}
}

type stubGateway struct {
	files map[string]*filegateway.FileDisplayInfo
}

func (s stubGateway) GetFileInfo(
	_ context.Context,
	fileID string,
	_ *uint,
) (*filegateway.FileDisplayInfo, error) {
	if info, ok := s.files[fileID]; ok {
		return info, nil
	}
	return nil, commonError.ErrFileNotAccessible
}

func (s stubGateway) GetFilesWithURLs(
	_ context.Context,
	fileIDs []string,
	_ *uint,
) (map[string]*filegateway.FileDisplayInfo, error) {
	result := make(map[string]*filegateway.FileDisplayInfo, len(fileIDs))
	for _, id := range fileIDs {
		if info, ok := s.files[id]; ok {
			result[id] = info
		}
	}
	return result, nil
}
