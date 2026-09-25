package cachekit

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// ErrUnscopedKey rejects keys that carry neither a seller scope nor an
// allowlisted platform form. The builder is the only way to mint keys.
var ErrUnscopedKey = errors.New("cachekit: key lacks seller scope or allowlist form")

// platformAllowlist holds exact or prefix forms that may bypass seller scope.
// Closed set: adding a resource means adding a pre-spec §4.5 row AND an entry
// here in the same change (consistency rule).
var platformAllowlist = []struct {
	prefix string
	// exact matches the whole key; otherwise prefix match.
	exact bool
	// hex64 requires a 64-char lowercase hex suffix (sha256 identities).
	hex64 bool
}{
	{prefix: "bl:", hex64: true},
	{prefix: "coupon:apply:rate:"},
	{prefix: "file:init:idem:"},
	{prefix: "file:providers", exact: true},
	{prefix: "file:schema:"},
	{prefix: "geo:country:"},
	{prefix: "geo:currency:"},
	{prefix: "geo:countries:active:"},
	{prefix: "geo:currencies:active:"},
	{prefix: "gateway:catalog:"},
	{prefix: "attributes:all:"}, // global attribute-list version (definitions are not seller-owned)
	{prefix: "delayed_jobs", exact: true},
	{prefix: "scheduled_job:"},
}

// hex64Suffix matches a 64-char lowercase hex digest.
var hex64Suffix = regexp.MustCompile(`^[0-9a-f]{64}$`)

// BuildSellerKey mints a tenant-scoped key: seller:{id}:{resource...}.
// Resource segments must be non-empty and free of spaces, braces, and glob
// characters so keys stay SCAN-safe and log-safe.
func BuildSellerKey(sellerID uint, segments ...string) (string, error) {
	if sellerID == 0 {
		return "", ErrUnscopedKey
	}
	for _, s := range segments {
		if s == "" || strings.ContainsAny(s, " {}[]*?\\") {
			return "", fmt.Errorf("%w: bad segment %q", ErrUnscopedKey, s)
		}
	}
	return "seller:" + strconv.FormatUint(uint64(sellerID), 10) + ":" + strings.Join(segments, ":"), nil
}

// MustBuildSellerKey is BuildSellerKey for static call sites; it panics on
// invalid input so key-shape bugs fail fast in tests, never in production.
func MustBuildSellerKey(sellerID uint, segments ...string) string {
	k, err := BuildSellerKey(sellerID, segments...)
	if err != nil {
		panic(err)
	}
	return k
}

// ValidateKey accepts seller-scoped keys and allowlisted platform keys,
// rejecting everything else at build time (unit-tested, incl. cross-seller
// probes that must never validate as another seller's key).
func ValidateKey(key string) error {
	if strings.HasPrefix(key, "seller:") {
		rest := strings.TrimPrefix(key, "seller:")
		id, tail, found := strings.Cut(rest, ":")
		if !found || id == "" {
			return fmt.Errorf("%w: %q", ErrUnscopedKey, key)
		}
		if _, err := strconv.ParseUint(id, 10, 64); err != nil {
			return fmt.Errorf("%w: %q", ErrUnscopedKey, key)
		}
		// Same segment hygiene as BuildSellerKey: spaces/globs anywhere in
		// the key fail closed so validators and builders never disagree.
		if tail == "" || strings.ContainsAny(tail, " {}[]*?\\") {
			return fmt.Errorf("%w: %q", ErrUnscopedKey, key)
		}
		return nil
	}
	for _, a := range platformAllowlist {
		if a.exact {
			if key == a.prefix {
				return nil
			}
			continue
		}
		if !strings.HasPrefix(key, a.prefix) {
			continue
		}
		// Non-exact forms require a non-empty remainder (an empty user ID,
		// code, or adapter name is never a valid key). Remainders reject
		// path separators, parent references, whitespace, and glob
		// characters so allowlisted keys cannot smuggle ambiguous shapes.
		rest := strings.TrimPrefix(key, a.prefix)
		if rest == "" || strings.Contains(rest, "/") || strings.Contains(rest, "..") ||
			strings.ContainsAny(rest, " {}[]*?\\") {
			continue
		}
		if a.hex64 && !hex64Suffix.MatchString(strings.TrimPrefix(key, a.prefix)) {
			continue
		}
		return nil
	}
	return fmt.Errorf("%w: %q", ErrUnscopedKey, key)
}
