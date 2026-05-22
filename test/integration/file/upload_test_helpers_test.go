package file_test

import (
	"bytes"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

// putWithMimeOverride performs an HTTP PUT to the presigned upload URL from initData,
// sending the given body with the specified Content-Type header override.
// This is used by T066 to simulate a client that PUTs bytes with a MIME type
// different from the one declared in the init-upload request.
func putWithMimeOverride(
	t *testing.T,
	initData map[string]any,
	body []byte,
	overrideContentType string,
) {
	t.Helper()

	uploadURL, _ := initData["uploadUrl"].(string)
	require.NotEmpty(t, uploadURL, "initData must contain uploadUrl")

	req, err := http.NewRequest(http.MethodPut, uploadURL, bytes.NewReader(body))
	require.NoError(t, err)

	// Apply headers from the init response (e.g. required signed headers).
	if headers, ok := initData["uploadHeaders"].(map[string]any); ok {
		for k, v := range headers {
			if str, ok := v.(string); ok {
				req.Header.Set(k, str)
			}
		}
	}

	// Override Content-Type to simulate a MIME mismatch.
	req.Header.Set("Content-Type", overrideContentType)

	res, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer res.Body.Close()
	// A 2xx is expected; the content-type override is what we care about.
	require.GreaterOrEqual(t, res.StatusCode, http.StatusOK)
	require.Less(t, res.StatusCode, http.StatusMultipleChoices)
}
