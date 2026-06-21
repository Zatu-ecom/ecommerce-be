package filegateway

import "context"

// FileDisplayGateway defines read-side cross-module file operations.
// All methods accept a sellerID pointer so callers can pass nil for
// platform/admin contexts that are not scoped to a single seller.
//
// Implementations live in the file module and must not be imported from
// common beyond this interface boundary.
type FileDisplayGateway interface {
	// GetFileInfo validates that a file exists and is accessible, and returns
	// its display URL and thumbnail where available.
	GetFileInfo(ctx context.Context, fileID string, sellerID *uint) (*FileDisplayInfo, error)

	// GetFilesWithURLs performs a batched lookup for the supplied fileIDs and
	// returns a map keyed by fileID. Missing or inaccessible files are silently
	// omitted so consumer responses degrade gracefully.
	GetFilesWithURLs(
		ctx context.Context,
		fileIDs []string,
		sellerID *uint,
	) (map[string]*FileDisplayInfo, error)
}

// FileLifecycleGateway extends FileDisplayGateway with file deletion for
// modules that own media lifecycle (e.g. product media detach).
type FileLifecycleGateway interface {
	FileDisplayGateway

	// DeleteFile attempts to delete the underlying file asset. Callers MUST
	// treat errors from this method as best-effort degradation.
	DeleteFile(ctx context.Context, fileID string, sellerID *uint) error
}
