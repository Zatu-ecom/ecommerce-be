package utils

import (
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"ecommerce-be/file/entity"
)

const (
	// sanitizedFilenameMaxBytes is the maximum byte-length of a sanitised filename
	// (excludes extension and leading prefix; see research R9).
	sanitizedFilenameMaxBytes = 120

	// sanitizedFilenameDefault is used when the raw name reduces to empty string.
	sanitizedFilenameDefault = "file"
)

// allowedRunePattern matches characters that are safe in URL path segments
// and object-key components. Anything not matching is stripped.
var allowedRunePattern = regexp.MustCompile(`[^A-Za-z0-9._-]`)

// SanitizeFilename converts a raw, client-supplied filename into a lowercase,
// URL-safe string suitable for embedding in an object key.
//
// Algorithm (research R9):
//
//  1. Lowercase the entire string.
//  2. Strip/replace any character outside [A-Za-z0-9._-]:
//     whitespace → "-";  all other disallowed chars → stripped.
//  3. Collapse consecutive "-" characters to a single "-".
//  4. Truncate to 120 bytes (UTF-8 safe — never splits a multi-byte sequence).
//  5. If the result is empty (e.g. input was "   " or all punctuation), return "file".
func SanitizeFilename(raw string) string {
	// 1. Lowercase.
	s := strings.ToLower(raw)

	// 2. Convert whitespace to "-"; remove other disallowed chars.
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' {
			b.WriteByte('-')
		} else if allowedRunePattern.MatchString(string(r)) {
			// Disallowed character — skip it.
		} else {
			b.WriteRune(r)
		}
	}
	s = b.String()

	// 3. Collapse consecutive dashes.
	for strings.Contains(s, "--") {
		s = strings.ReplaceAll(s, "--", "-")
	}
	// Trim leading / trailing dashes to avoid "--filename" artifacts.
	s = strings.Trim(s, "-")

	// 4. Truncate to maxBytes (UTF-8 safe).
	if utf8.RuneCountInString(s) > 0 && len(s) > sanitizedFilenameMaxBytes {
		s = truncateToBytes(s, sanitizedFilenameMaxBytes)
		// Re-trim in case truncation split a "-".
		s = strings.TrimRight(s, "-")
	}

	// 5. Default if empty.
	if s == "" {
		return sanitizedFilenameDefault
	}
	return s
}

// truncateToBytes truncates a UTF-8 string to at most maxBytes bytes without
// splitting a multi-byte rune at the boundary.
func truncateToBytes(s string, maxBytes int) string {
	if len(s) <= maxBytes {
		return s
	}
	// Walk runes until we exceed the limit.
	byteCount := 0
	for i, r := range s {
		runeSize := utf8.RuneLen(r)
		if byteCount+runeSize > maxBytes {
			return s[:i]
		}
		byteCount += runeSize
	}
	return s
}

// BuildObjectKey constructs the deterministic object-key for a new upload.
//
// Template (research R9):
//   - Seller: seller/{sellerId}/{purpose}/{yyyy}/{mm}/{fileId}-{sanitized}
//   - Admin (platform): platform/{purpose}/{yyyy}/{mm}/{fileId}-{sanitized}
//
// Parameters:
//   - ownerType:  entity.OwnerTypeSeller or entity.OwnerTypePlatform
//   - sellerID:   the seller's numeric ID; ignored when ownerType is PLATFORM
//   - purpose:    e.g. "PRODUCT_IMAGE"
//   - now:        current time used for the yyyy/mm path segments
//   - fileID:     UUIDv7 string (e.g. "018f2c1a-7a3e-7b2c-b4e2-c2a9d3e80001")
//   - sanitized:  output of SanitizeFilename(rawFilename)
func BuildObjectKey(
	ownerType entity.OwnerType,
	sellerID *uint64,
	purpose entity.FilePurpose,
	now time.Time,
	fileID string,
	sanitized string,
) string {
	yyyy := now.UTC().Format("2006")
	mm := now.UTC().Format("01")
	baseName := fileID + "-" + sanitized

	switch ownerType {
	case entity.OwnerTypeSeller:
		sid := uint64(0)
		if sellerID != nil {
			sid = *sellerID
		}
		return fmt.Sprintf("seller/%d/%s/%s/%s/%s", sid, purpose, yyyy, mm, baseName)
	default:
		// PLATFORM / admin
		return fmt.Sprintf("platform/%s/%s/%s/%s", purpose, yyyy, mm, baseName)
	}
}
