package cachekit

import (
	"bytes"
	"time"
)

// TombstoneBytes is the negative-entry payload: a JSON sentinel sharing the
// missing entry's key (never a separate neg: key), so write-path Del stays
// uniform. Readers treat it as a documented miss via IsTombstone — decoding
// it as a DTO is a caller bug (asserted per module, e.g. T11).
var TombstoneBytes = []byte(`{"_neg":true}`)

// DefaultNegativeTTL bounds negative entries; jittered at write time like
// every SET so negative expiries cannot synchronize either.
const DefaultNegativeTTL = 60 * time.Second

// IsTombstone reports whether stored bytes are a negative-entry sentinel.
func IsTombstone(b []byte) bool {
	return bytes.Equal(bytes.TrimSpace(b), TombstoneBytes)
}
