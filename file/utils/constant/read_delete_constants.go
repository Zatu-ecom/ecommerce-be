package constant

// ========================================
// FILE READ/DELETE ERROR CODES
// ========================================

const (
	FILE_NOT_FOUND_CODE                 = "FILE_NOT_FOUND"
	FILE_NOT_ACTIVE_CODE                = "FILE_NOT_ACTIVE"
	VARIANT_NOT_FOUND_CODE              = "VARIANT_NOT_FOUND"
	VARIANT_NOT_READY_CODE              = "VARIANT_NOT_READY"
	FILE_DELETE_CONFLICT_CODE           = "FILE_DELETE_CONFLICT"
	STORAGE_PERMISSION_DENIED_CODE      = "STORAGE_PERMISSION_DENIED"
	STORAGE_UNAVAILABLE_CODE            = "STORAGE_UNAVAILABLE"
)

// ========================================
// FILE READ/DELETE ERROR MESSAGES
// ========================================

const (
	FILE_NOT_FOUND_MSG            = "File not found"
	FILE_NOT_ACTIVE_MSG           = "File must be ACTIVE for this operation"
	VARIANT_NOT_FOUND_MSG         = "Variant not found"
	VARIANT_NOT_READY_MSG         = "Variant is not ready"
	FILE_DELETE_CONFLICT_MSG      = "File cannot be deleted in its current state"
	STORAGE_PERMISSION_DENIED_MSG = "Storage provider denied access for this operation"
	STORAGE_UNAVAILABLE_MSG       = "Storage provider is currently unavailable; please retry"
)

// ========================================
// FILE READ/DELETE DOWNLOAD DEFAULTS
// ========================================

const (
	DefaultDownloadURLTTLMinutes = 15
	MinDownloadURLTTLMinutes     = 5
	MaxDownloadURLTTLMinutes     = 60
)

// ========================================
// FILE READ/DELETE DISPOSITIONS
// ========================================

const (
	DownloadDispositionInline     = "inline"
	DownloadDispositionAttachment = "attachment"
)

// ThumbnailVariantCodes lists file variant codes preferred for thumbnail selection.
var ThumbnailVariantCodes = []string{"thumb_200", "poster"}

// ========================================
// FILE READ/DELETE SUCCESS MESSAGES
// ========================================

const (
	FILE_LIST_SUCCESS_MSG         = "Files retrieved"
	FILE_GET_SUCCESS_MSG          = "File retrieved"
	FILE_DOWNLOAD_URL_SUCCESS_MSG = "Download URL generated"
	FILE_DELETE_SUCCESS_MSG       = "File deleted"
)
