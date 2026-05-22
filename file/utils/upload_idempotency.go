package utils

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"

	"ecommerce-be/file/entity"
	"ecommerce-be/file/model"
	"ecommerce-be/file/utils/constant"
)

var idempotencyKeyPattern = regexp.MustCompile(`^[A-Za-z0-9._~-]+$`)

// ValidateIdempotencyKey applies the upload API's Idempotency-Key header rules.
func ValidateIdempotencyKey(key string) bool {
	return len(key) >= 8 && len(key) <= 128 && idempotencyKeyPattern.MatchString(key)
}

// HashIdempotencyKey returns a fixed-length, non-reversible representation for Redis keys.
func HashIdempotencyKey(key string) string {
	sum := sha256.Sum256([]byte(key))
	return hex.EncodeToString(sum[:])
}

// BuildInitIdempotencyRedisKey namespaces init-upload retries by caller and hashed key.
func BuildInitIdempotencyRedisKey(caller Principal, rawKey string) string {
	ownerID := caller.UserID
	if caller.OwnerType == entity.OwnerTypeSeller && caller.SellerID != nil {
		ownerID = *caller.SellerID
	}
	return fmt.Sprintf(
		"%s%s:%d:%s",
		constant.RedisKeyInitIdempotencyPrefix,
		caller.OwnerType,
		ownerID,
		HashIdempotencyKey(rawKey),
	)
}

type initUploadFingerprintPayload struct {
	Purpose             entity.FilePurpose    `json:"purpose"`
	Visibility          entity.FileVisibility `json:"visibility"`
	Filename            string                `json:"filename"`
	MimeType            string                `json:"mimeType"`
	SizeBytes           int64                 `json:"sizeBytes"`
	UploadExpiryMinutes int                   `json:"uploadExpiryMinutes"`
}

// InitUploadFingerprint hashes the canonical request payload after service defaults are applied.
func InitUploadFingerprint(req model.InitUploadRequest, expiryMinutes int) (string, error) {
	payload := initUploadFingerprintPayload{
		Purpose:             req.Purpose,
		Visibility:          req.Visibility,
		Filename:            req.Filename,
		MimeType:            req.MimeType,
		SizeBytes:           req.SizeBytes,
		UploadExpiryMinutes: expiryMinutes,
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}
