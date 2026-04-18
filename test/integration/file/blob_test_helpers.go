package file_test

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"io"
	"strings"
	"testing"
	"time"
)

// RandomHex returns a hex-encoded string of nBytes random bytes.
func RandomHex(nBytes int) string {
	if nBytes <= 0 {
		nBytes = 8
	}
	b := make([]byte, nBytes)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// RandomObjectKey returns a slash-separated key with a random suffix.
// prefix should not include a trailing slash.
func RandomObjectKey(prefix string) string {
	p := strings.Trim(prefix, "/")
	if p == "" {
		return RandomHex(12)
	}
	return p + "/" + RandomHex(12)
}

// SmallTextBody wraps a string as an io.Reader for use in PutObject calls.
func SmallTextBody(s string) io.Reader {
	return strings.NewReader(s)
}

// DefaultPresignTTL returns the standard TTL used for presigned URL tests.
func DefaultPresignTTL() time.Duration { return 2 * time.Minute }

// ShortDeadlineCtx returns a context that expires in 250 ms, used to
// verify that adapter methods honour context cancellation.
func ShortDeadlineCtx() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 250*time.Millisecond)
}

// AssertNoSecretLeak fails the test if any of the provided secret strings appear
// in err.Error(). Pass every sensitive credential value (account keys, passwords,
// private-key fragments) to validate that adapter errors are sanitised.
// Empty secret strings are silently skipped.
func AssertNoSecretLeak(t *testing.T, err error, secrets ...string) {
	t.Helper()
	if err == nil {
		return
	}
	msg := err.Error()
	for _, s := range secrets {
		if s != "" && strings.Contains(msg, s) {
			t.Errorf("secret leaked in error message: error contains %q", s)
		}
	}
}
