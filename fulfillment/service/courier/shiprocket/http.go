package shiprocket

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"time"

	"ecommerce-be/common/log"
	fulfillmenterrors "ecommerce-be/fulfillment/error"
)

// HTTP client policy for the Shiprocket REST API (apiv2.shiprocket.in).
//
//   - One pooled client per adapter; per-call timeouts (GET 5s, POST 10s)
//     so no hung book calls.
//   - Auth: Bearer JWT from auth.go (login with api_email/api_password).
//   - POST (book/cancel/pickup/label/returns): NO retry — our shipment id is
//     the provider-side order id and a retried POST risks double AWBs.
//   - GET (rates/track/test-connection): up to 2 retries with jitter,
//     ONLY on 429 and 5xx. 4xx (bad keys, bad ids) fails fast.
//   - Logs never carry Authorization headers, secrets, or full bodies:
//     status + path + a truncated body prefix for debugging only.
const (
	getTimeout       = 5 * time.Second
	postTimeout      = 10 * time.Second
	maxGetRetries    = 2
	retryBaseBackoff = 200 * time.Millisecond
	logBodyTruncate  = 512
)

// defaultBaseURL is the production Shiprocket API endpoint. Tests override it
// through New (factory wires SHIPROCKET_BASE_URL).
const defaultBaseURL = "https://apiv2.shiprocket.in"

// doJSON performs an authenticated request and returns the decoded JSON body.
// A 401 surfaces as a credentials validation error so handlers map it to 400
// (bad keys), not 500. Callers treat it as a signal to purge the cached
// token and retry once at most.
func (a *Adapter) doJSON(
	ctx context.Context,
	token string,
	method, path string,
	body any,
) (map[string]any, error) {
	timeout := postTimeout
	if method == http.MethodGet {
		timeout = getTimeout
	}
	callCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var payload []byte
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("shiprocket: marshal request: %w", err)
		}
		payload = b
	}

	attempts := 1
	if method == http.MethodGet {
		attempts = 1 + maxGetRetries
	}

	var lastErr error
	for attempt := 0; attempt < attempts; attempt++ {
		if attempt > 0 {
			backoffWithJitter(callCtx, attempt)
		}
		result, retryable, err := a.doOnce(callCtx, token, method, path, payload)
		if err == nil {
			return result, nil
		}
		lastErr = err
		if !retryable {
			return nil, err
		}
	}
	return nil, lastErr
}

// doOnce executes a single attempt. The retryable flag reports whether the
// caller may try again (GET + 429/5xx only).
func (a *Adapter) doOnce(
	ctx context.Context,
	token, method, path string,
	payload []byte,
) (result map[string]any, retryable bool, err error) {
	var reader io.Reader
	if payload != nil {
		reader = bytes.NewReader(payload)
	}

	req, err := http.NewRequestWithContext(ctx, method, a.BaseURL+path, reader)
	if err != nil {
		return nil, false, fmt.Errorf("shiprocket: build request: %w", err)
	}
	// SSRF guard: BaseURL is compiled into the adapter (or the test override);
	// paths are adapter constants, never caller-supplied URLs.
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		// Transport error: safe to retry idempotent GETs, never POSTs.
		return nil, method == http.MethodGet, fmt.Errorf("shiprocket: request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, method == http.MethodGet, fmt.Errorf("shiprocket: read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// Redacted: status + path only in the error; a truncated body prefix
		// goes to logs for debugging. Never headers (Authorization) or secrets.
		log.WarnWithContext(ctx,
			"shiprocket api error status="+resp.Status+" path="+path+" body="+truncateForLog(respBody))
		if resp.StatusCode == http.StatusUnauthorized {
			// Typed AppError: invalid/expired token or keys are a client
			// error (400), not a server failure.
			return nil, false, fulfillmenterrors.ErrorCredentialsInvalid.WithMessagef(
				"[shiprocket] unauthorized (status=401 path=%s)", path)
		}
		if method == http.MethodGet && (resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500) {
			return nil, true, fmt.Errorf("shiprocket: api error status=%d path=%s", resp.StatusCode, path)
		}
		return nil, false, fmt.Errorf("shiprocket: api error status=%d path=%s", resp.StatusCode, path)
	}

	var decoded map[string]any
	if err := json.Unmarshal(respBody, &decoded); err != nil {
		return nil, false, fmt.Errorf("shiprocket: parse response: %w", err)
	}
	return decoded, false, nil
}

// backoffWithJitter sleeps ~200ms * attempt plus up to 100ms jitter.
// It aborts early when the request context is done.
func backoffWithJitter(ctx context.Context, attempt int) {
	jitter := time.Duration(rand.Int63n(int64(100 * time.Millisecond)))
	delay := time.Duration(attempt)*retryBaseBackoff + jitter
	select {
	case <-ctx.Done():
	case <-time.After(delay):
	}
}

func truncateForLog(body []byte) string {
	if len(body) > logBodyTruncate {
		return string(body[:logBodyTruncate]) + "…(truncated)"
	}
	return string(body)
}
