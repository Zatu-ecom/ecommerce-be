package cachekit

import (
	"encoding/json"
	"errors"
	"time"
)

// MaxValueBytes caps a single cached value. Larger payloads are never stored
// (callers serve the database result and count cache_value_too_large).
// Rationale: big SETs head-of-line-block single-threaded backends and turn
// a slowdown into an outage (pre-spec §0.4).
const MaxValueBytes = 256 * 1024

// ErrValueTooLarge reports a payload over MaxValueBytes.
var ErrValueTooLarge = errors.New("cachekit: value exceeds size guard")

// Marshal encodes a DTO for storage, enforcing the size guard.
func Marshal(v any) ([]byte, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	if len(b) > MaxValueBytes {
		return nil, ErrValueTooLarge
	}
	return b, nil
}

// Unmarshal decodes stored bytes into out. Tombstones must be checked with
// IsTombstone first — decoding a tombstone as a DTO is a caller bug.
func Unmarshal(b []byte, out any) error {
	return json.Unmarshal(b, out)
}

// MarshalWithTTL is a convenience for strategies that compute size-guarded
// payloads together with their jittered wire TTL.
func MarshalWithTTL(v any, baseTTL time.Duration) ([]byte, time.Duration, error) {
	b, err := Marshal(v)
	if err != nil {
		return nil, 0, err
	}
	return b, JitteredTTL(baseTTL), nil
}
