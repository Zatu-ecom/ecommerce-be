package razorpay

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
	paymenterrors "ecommerce-be/payment/error"
)

// HTTP client policy for the Razorpay REST API.
//
//   - One pooled client per adapter, 15s timeout (no hung initiate calls).
//   - Auth: HTTP Basic with key_id:key_secret.
//   - POST (create order / refund): NO retry — receipts are provider-side
//     idempotency keys and a retried POST risks double charges.
//   - GET (TestConnection / FetchRemoteStatus): up to 2 retries with jitter,
//     ONLY on 429 and 5xx. 4xx (bad keys, bad ids) fails fast.
//   - Logs never carry Authorization headers, secrets, or full bodies:
//     status + path + a truncated body prefix for debugging only.
const (
	httpTimeout      = 15 * time.Second
	maxGetRetries    = 2
	retryBaseBackoff = 200 * time.Millisecond
	logBodyTruncate  = 512
)

// doJSON performs an authenticated request and returns the decoded JSON body.
func (a *Adapter) doJSON(
	ctx context.Context,
	creds *razorpayCredentials,
	method, path string,
	body any,
) (map[string]any, error) {
	var payload []byte
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("razorpay: marshal request: %w", err)
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
			backoffWithJitter(ctx, attempt)
		}
		result, retryable, err := a.doOnce(ctx, creds, method, path, payload)
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
	creds *razorpayCredentials,
	method, path string,
	payload []byte,
) (result map[string]any, retryable bool, err error) {
	var reader io.Reader
	if payload != nil {
		reader = bytes.NewReader(payload)
	}

	req, err := http.NewRequestWithContext(ctx, method, a.BaseURL+path, reader)
	if err != nil {
		return nil, false, fmt.Errorf("razorpay: build request: %w", err)
	}
	req.SetBasicAuth(creds.KeyID, creds.KeySecret)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		// Transport error: safe to retry idempotent GETs, never POSTs.
		return nil, method == http.MethodGet, fmt.Errorf("razorpay: request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, method == http.MethodGet, fmt.Errorf("razorpay: read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// Redacted: status + path only in the error; a truncated body prefix
		// goes to logs for debugging. Never headers (Authorization) or secrets.
		log.WarnWithContext(ctx,
			"razorpay api error status="+resp.Status+" path="+path+" body="+truncateForLog(respBody))
		if resp.StatusCode == http.StatusUnauthorized {
			// Typed AppError: invalid provider keys are a client error (400),
			// not a server failure — in test-connection and configure flows.
			return nil, false, paymenterrors.ErrorGatewayValidation.WithMessagef(
				"[razorpay] invalid credentials (status=401 path=%s)", path)
		}
		if method == http.MethodGet && (resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500) {
			return nil, true, fmt.Errorf("razorpay: api error status=%d path=%s", resp.StatusCode, path)
		}
		return nil, false, fmt.Errorf("razorpay: api error status=%d path=%s", resp.StatusCode, path)
	}

	var decoded map[string]any
	if err := json.Unmarshal(respBody, &decoded); err != nil {
		return nil, false, fmt.Errorf("razorpay: parse response: %w", err)
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
