package constant

import "time"

// ========================================
// UPLOAD ERROR CODES (research R11)
// ========================================

const (
	// FILE_UPLOAD_UNAUTHORIZED is returned when the request has no valid authentication token.
	FILE_UPLOAD_UNAUTHORIZED_CODE = "FILE_UPLOAD_UNAUTHORIZED"

	// FILE_UPLOAD_FORBIDDEN is returned when the caller's role is not permitted (e.g., customer).
	FILE_UPLOAD_FORBIDDEN_CODE = "FILE_UPLOAD_FORBIDDEN"

	// FILE_UPLOAD_INVALID_INPUT is returned on request-body validation failures.
	FILE_UPLOAD_INVALID_INPUT_CODE = "FILE_UPLOAD_INVALID_INPUT"

	// FILE_UPLOAD_POLICY_VIOLATION is returned when purpose/mime/size policy rejects the upload.
	FILE_UPLOAD_POLICY_VIOLATION_CODE = "FILE_UPLOAD_POLICY_VIOLATION"

	// FILE_UPLOAD_STORAGE_UNAVAILABLE is returned when the blob adapter or presign call fails.
	FILE_UPLOAD_STORAGE_UNAVAILABLE_CODE = "FILE_UPLOAD_STORAGE_UNAVAILABLE"

	// FILE_UPLOAD_NO_STORAGE_CONFIG is returned when neither a seller binding nor a platform
	// default storage config resolves to an active, validated config.
	FILE_UPLOAD_NO_STORAGE_CONFIG_CODE = "FILE_UPLOAD_NO_STORAGE_CONFIG"

	// FILE_UPLOAD_NOT_FOUND is returned when the fileId is not found OR cross-tenant access is
	// attempted (no enumeration — always 404, never 403).
	FILE_UPLOAD_NOT_FOUND_CODE = "FILE_UPLOAD_NOT_FOUND"

	// FILE_UPLOAD_CONFLICT is returned when the Idempotency-Key fingerprint mismatches or the
	// previous upload for this key is no longer UPLOADING.
	FILE_UPLOAD_CONFLICT_CODE = "FILE_UPLOAD_CONFLICT"

	// FILE_UPLOAD_OBJECT_MISSING is returned by complete-upload when HeadObject returns not-found.
	FILE_UPLOAD_OBJECT_MISSING_CODE = "FILE_UPLOAD_OBJECT_MISSING"

	// FILE_UPLOAD_OBJECT_MISMATCH is returned when the ETag, size, or content-type verification fails.
	FILE_UPLOAD_OBJECT_MISMATCH_CODE = "FILE_UPLOAD_OBJECT_MISMATCH"

	// FILE_UPLOAD_EXPIRED is returned when complete-upload is called after the row has been
	// transitioned to FAILED with reason UPLOAD_EXPIRED.
	FILE_UPLOAD_EXPIRED_CODE = "FILE_UPLOAD_EXPIRED"

	// FILE_UPLOAD_INTERNAL is returned for unhandled/unexpected errors; always secret-stripped.
	FILE_UPLOAD_INTERNAL_CODE = "FILE_UPLOAD_INTERNAL"
)

// ========================================
// UPLOAD ERROR MESSAGES
// ========================================

const (
	FILE_UPLOAD_UNAUTHORIZED_MSG         = "Authentication required to upload files"
	FILE_UPLOAD_FORBIDDEN_MSG            = "Only seller or admin users may upload files"
	FILE_UPLOAD_INVALID_INPUT_MSG        = "Request validation failed"
	FILE_UPLOAD_POLICY_VIOLATION_MSG     = "Upload rejected by file policy (size, MIME type, or purpose)"
	FILE_UPLOAD_STORAGE_UNAVAILABLE_MSG  = "Storage provider is currently unavailable; please retry"
	FILE_UPLOAD_NO_STORAGE_CONFIG_MSG    = "No active storage configuration found for this account"
	FILE_UPLOAD_NOT_FOUND_MSG            = "File not found"
	FILE_UPLOAD_CONFLICT_MSG             = "Idempotency key conflict: the upload state has changed"
	FILE_UPLOAD_OBJECT_MISSING_MSG       = "Object has not been uploaded to the provider yet"
	FILE_UPLOAD_OBJECT_MISMATCH_MSG      = "Uploaded object does not match the declared size, MIME type, or ETag"
	FILE_UPLOAD_EXPIRED_MSG              = "Upload window has expired; please initiate a new upload"
	FILE_UPLOAD_INTERNAL_MSG             = "An internal error occurred; please try again"
)

// ========================================
// SCHEDULER COMMAND (common/scheduler)
// ========================================

const (
	// SchedulerCommandUploadExpiry is the command key registered with common/scheduler.
	// When this command fires the UploadExpiryHandler transitions UPLOADING → FAILED/UPLOAD_EXPIRED.
	SchedulerCommandUploadExpiry = "file.upload.expiry"
)

// ========================================
// RABBITMQ CONSTANTS
// ========================================

const (
	// ExchangeEcomCommands is the durable topic exchange used for all command messages.
	ExchangeEcomCommands = "ecom.commands"

	// RoutingKeyFileImageProcessRequested is the routing key for the image-variant command.
	RoutingKeyFileImageProcessRequested = "file.image.process.requested"
)

// ========================================
// REDIS KEY PREFIXES
// ========================================

const (
	// RedisKeyInitIdempotencyPrefix is used to cache init-upload idempotency records.
	// Full key: file:init:idem:{ownerType}:{ownerId}:{sha256(idempotencyKey)}
	RedisKeyInitIdempotencyPrefix = "file:init:idem:"

	// RedisKeySchedulerFileUploadExpiryPrefix is the prefix for caching the scheduler job ID
	// so complete-upload can cancel the expiry job.
	// Full key (seller): seller:{sellerId}:file.upload.expiry:{fileObjectId}
	// Full key (platform): platform:file.upload.expiry:{fileObjectId}
	RedisKeySchedulerFileUploadExpiryPrefix = "file.upload.expiry"

	// SellerSchedulerKeyFmt is the sprintf format for seller-scoped scheduler cache keys.
	// Args: sellerID (uint64), fileObjectID (uint64)
	SellerSchedulerKeyFmt = "seller:%d:file.upload.expiry:%d"

	// PlatformSchedulerKeyFmt is the sprintf format for platform-scoped scheduler cache keys.
	// Args: fileObjectID (uint64)
	PlatformSchedulerKeyFmt = "platform:file.upload.expiry:%d"
)

// ========================================
// TIMING / DEFAULTS
// ========================================

const (
	// CacheBufferDuration is added to the scheduler TTL and the idempotency Redis TTL
	// to ensure the cache key outlives the scheduler-managed expiry by a small margin.
	CacheBufferDuration = 5 * time.Minute

	// DefaultUploadExpiryMinutes is used when the caller omits uploadExpiryMinutes.
	DefaultUploadExpiryMinutes = 15

	// MinUploadExpiryMinutes is the minimum allowed value for uploadExpiryMinutes.
	MinUploadExpiryMinutes = 5

	// MaxUploadExpiryMinutes is the maximum allowed value for uploadExpiryMinutes.
	MaxUploadExpiryMinutes = 60
)

// ========================================
// FAILURE REASON CODES (stored in file_object.failure_reason)
// ========================================

const (
	// FailureReasonUploadExpired is set when the scheduler fires against an UPLOADING row.
	FailureReasonUploadExpired = "UPLOAD_EXPIRED"

	// FailureReasonObjectMismatch is set when size/mime/etag verification fails at complete time.
	FailureReasonObjectMismatch = "OBJECT_MISMATCH"
)

// ========================================
// SUCCESS MESSAGES
// ========================================

const (
	FILE_UPLOAD_INIT_SUCCESS_MSG     = "Upload initialised successfully"
	FILE_UPLOAD_COMPLETE_SUCCESS_MSG = "File upload completed successfully"
)
