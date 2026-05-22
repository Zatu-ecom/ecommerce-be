package utils

import (
	"strings"

	fileError "ecommerce-be/file/error"
	"ecommerce-be/file/entity"

	commonError "ecommerce-be/common/error"
)

// Policy encapsulates all upload constraints for a given purpose.
// Returned by Evaluate on a successful policy check.
type Policy struct {
	// MaxSize is the maximum allowed file size in bytes.
	MaxSize int64

	// AllowedMimes is the list of allowed MIME types for this purpose (lowercase).
	AllowedMimes []string

	// HasVariants is true when a file.image.process.requested message should be
	// published after a successful complete-upload.
	HasVariants bool

	// VariantCodes is the list of variant codes to include in the messaging payload.
	// Only meaningful when HasVariants=true.
	VariantCodes []string
}

const (
	// MB is a convenience constant.
	mb int64 = 1024 * 1024
)

// purposePolicy is the table-driven policy registry. Indexed by FilePurpose.
//
// Alignment with data-model.md §4.1:
//   - PRODUCT_IMAGE: 10 MB, jpeg/png/webp, variants [thumb_200, thumb_600, webp_1600]
//   - DOCUMENT:      25 MB, pdf/jpeg/png,  no variants
//   - IMPORT_FILE:   50 MB, csv/xls/xlsx,  no variants
//   - EXPORT_FILE:   —      not accepted by init-upload (rejected in Evaluate)
//   - USER_AVATAR:    2 MB, jpeg/png/webp, variants [thumb_200, webp_400]
//   - SELLER_LOGO:    3 MB, jpeg/png/webp/svg, raster only variants [thumb_200, webp_400]; svg→HasVariants=false (applied per-call)
//   - INVOICE_PDF:   10 MB, pdf only, no variants
var purposePolicy = map[entity.FilePurpose]Policy{
	entity.FilePurposeProductImage: {
		MaxSize:      10 * mb,
		AllowedMimes: []string{"image/jpeg", "image/png", "image/webp"},
		HasVariants:  true,
		VariantCodes: []string{"thumb_200", "thumb_600", "webp_1600"},
	},
	entity.FilePurposeDocument: {
		MaxSize:      25 * mb,
		AllowedMimes: []string{"application/pdf", "image/jpeg", "image/png"},
		HasVariants:  false,
		VariantCodes: nil,
	},
	entity.FilePurposeImportFile: {
		MaxSize: 50 * mb,
		AllowedMimes: []string{
			"text/csv",
			"application/vnd.ms-excel",
			"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		},
		HasVariants:  false,
		VariantCodes: nil,
	},
	entity.FilePurposeUserAvatar: {
		MaxSize:      2 * mb,
		AllowedMimes: []string{"image/jpeg", "image/png", "image/webp"},
		HasVariants:  true,
		VariantCodes: []string{"thumb_200", "webp_400"},
	},
	entity.FilePurposeSellerLogo: {
		MaxSize:      3 * mb,
		AllowedMimes: []string{"image/jpeg", "image/png", "image/webp", "image/svg+xml"},
		// HasVariants for seller logo is determined at call time based on mime (SVG → false).
		HasVariants:  true,
		VariantCodes: []string{"thumb_200", "webp_400"},
	},
	entity.FilePurposeInvoicePDF: {
		MaxSize:      10 * mb,
		AllowedMimes: []string{"application/pdf"},
		HasVariants:  false,
		VariantCodes: nil,
	},
	// EXPORT_FILE is intentionally omitted — Evaluate rejects it explicitly.
}

// Evaluate checks the supplied purpose, MIME type, and file size against the policy table.
//
// Returns:
//   - (*Policy, nil) on success. The caller should use Policy.HasVariants and Policy.VariantCodes.
//   - (nil, *AppError) on violation — always ErrFileUploadPolicyViolation (422) unless purpose is
//     unknown/disallowed (also 422).
//
// Special cases:
//   - EXPORT_FILE: always rejected with ErrFileUploadPolicyViolation (system-generated only).
//   - SELLER_LOGO + image/svg+xml: HasVariants is overridden to false (SVG passthrough).
func Evaluate(purpose entity.FilePurpose, mime string, size int64) (*Policy, *commonError.AppError) {
	// EXPORT_FILE is always rejected by init-upload (data-model.md §4.1).
	if purpose == entity.FilePurposeExportFile {
		return nil, fileError.ErrFileUploadPolicyViolation.WithMessage(
			"EXPORT_FILE is system-generated and cannot be uploaded via init-upload",
		)
	}

	policy, ok := purposePolicy[purpose]
	if !ok {
		return nil, fileError.ErrFileUploadPolicyViolation.WithMessagef(
			"unknown or unsupported purpose: %s", purpose,
		)
	}

	// Validate MIME type (case-insensitive comparison per HTTP spec).
	normalizedMime := strings.ToLower(strings.TrimSpace(mime))
	if !isMimeAllowed(normalizedMime, policy.AllowedMimes) {
		return nil, fileError.ErrFileUploadPolicyViolation.WithMessagef(
			"MIME type %q is not allowed for purpose %s", mime, purpose,
		)
	}

	// Validate file size.
	if size <= 0 {
		return nil, fileError.ErrFileUploadPolicyViolation.WithMessage(
			"sizeBytes must be greater than 0",
		)
	}
	if size > policy.MaxSize {
		return nil, fileError.ErrFileUploadPolicyViolation.WithMessagef(
			"file size %d bytes exceeds maximum %d bytes for purpose %s",
			size, policy.MaxSize, purpose,
		)
	}

	// Deep-copy the policy so callers cannot mutate the registry.
	result := policy

	// SVG passthrough for SELLER_LOGO: no raster variants for SVG.
	if purpose == entity.FilePurposeSellerLogo && normalizedMime == "image/svg+xml" {
		result.HasVariants = false
		result.VariantCodes = nil
	}

	return &result, nil
}

// isMimeAllowed checks whether mime appears in the allowed list (case-insensitive).
func isMimeAllowed(mime string, allowed []string) bool {
	for _, a := range allowed {
		if a == mime {
			return true
		}
	}
	return false
}
