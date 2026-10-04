package shiprocket

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	commonError "ecommerce-be/common/error"
	fulfillmenterrors "ecommerce-be/fulfillment/error"
)

// Shiprocket issues a ~10-day JWT via POST /v1/external/auth/login
// (api_email + api_password). Tokens cache in memory per process (one login
// per ~9 days per pod); the reconciler never knows about tokens. No token →
// the outbound call fails CLOSED (credentials error), never ships on a
// stale token.
const (
	authLoginPath = "/v1/external/auth/login"
	// tokenTTL caches the JWT for 9 days, inside its ~10-day lifetime, so
	// refresh always happens proactively. A 401 mid-call purges + retries once.
	tokenTTL = 9 * 24 * time.Hour
)

// tokenEntry is one cached login.
type tokenEntry struct {
	token     string
	expiresAt time.Time
}

// shiprocketTokenKey builds the durable token key for one frozen config row.
// Reserved for a future shared-cache rollout; the live path uses the
// in-memory cache below (same semantics, no cross-pod coordination needed
// for a 9-day credential).
func shiprocketTokenKey(configID uint) string {
	return "shiprocket:token:" + strings.TrimSpace(fmt.Sprintf("%d", configID))
}

// tokenCache holds logins by api_email with mutex-guarded expiry.
func (a *Adapter) cachedToken(email string) (string, bool) {
	a.tokenMu.RLock()
	defer a.tokenMu.RUnlock()
	entry, ok := a.tokens[email]
	if !ok || time.Now().After(entry.expiresAt) {
		return "", false
	}
	return entry.token, true
}

func (a *Adapter) storeToken(email, token string) {
	a.tokenMu.Lock()
	defer a.tokenMu.Unlock()
	if a.tokens == nil {
		a.tokens = map[string]tokenEntry{}
	}
	a.tokens[email] = tokenEntry{token: token, expiresAt: time.Now().Add(tokenTTL)}
}

func (a *Adapter) purgeToken(email string) {
	a.tokenMu.Lock()
	defer a.tokenMu.Unlock()
	delete(a.tokens, email)
}

// token returns a live JWT for the credentials, logging in on miss/expiry.
func (a *Adapter) token(ctx context.Context, creds *shiprocketCredentials) (string, error) {
	if cached, ok := a.cachedToken(creds.APIEmail); ok {
		return cached, nil
	}
	token, err := a.login(ctx, creds)
	if err != nil {
		return "", err
	}
	a.storeToken(creds.APIEmail, token)
	return token, nil
}

// Login exchanges api_email/api_password for a JWT. POST: never retried.
func (a *Adapter) login(ctx context.Context, creds *shiprocketCredentials) (string, error) {
	resp, err := a.doJSON(ctx, "", http.MethodPost, authLoginPath, map[string]any{
		"email":    creds.APIEmail,
		"password": creds.APIPassword,
	})
	if err != nil {
		return "", err
	}
	token, _ := resp["token"].(string)
	if strings.TrimSpace(token) == "" {
		return "", fmt.Errorf("shiprocket: token missing in auth response")
	}
	return token, nil
}

// withAuth runs fn with a live token, purging + retrying once on 401.
// Callers use this for every authenticated call so expiry heals inline.
func (a *Adapter) withAuth(
	ctx context.Context,
	creds *shiprocketCredentials,
	fn func(ctx context.Context, token string) (map[string]any, error),
) (map[string]any, error) {
	token, err := a.token(ctx, creds)
	if err != nil {
		return nil, err
	}
	resp, err := fn(ctx, token)
	if err == nil {
		return resp, nil
	}
	if isUnauthorized(err) {
		a.purgeToken(creds.APIEmail)
		token, err = a.token(ctx, creds)
		if err != nil {
			return nil, err
		}
		return fn(ctx, token)
	}
	return nil, err
}

// withAuthResult is the typed twin of withAuth for calls returning
// provider-specific results (booking): same token + single 401 retry.
// A plain function (not a method): Go methods cannot take type parameters.
func withAuthResult[T any](
	ctx context.Context,
	a *Adapter,
	creds *shiprocketCredentials,
	fn func(ctx context.Context, token string) (T, error),
) (T, error) {
	var zero T
	token, err := a.token(ctx, creds)
	if err != nil {
		return zero, err
	}
	resp, err := fn(ctx, token)
	if err == nil {
		return resp, nil
	}
	if isUnauthorized(err) {
		a.purgeToken(creds.APIEmail)
		token, err = a.token(ctx, creds)
		if err != nil {
			return zero, err
		}
		return fn(ctx, token)
	}
	return zero, err
}

// isUnauthorized reports provider 401s (invalid/expired token or keys) by
// AppError code — never by message text.
func isUnauthorized(err error) bool {
	appErr, ok := commonError.AsAppError(err)
	return ok && appErr.Code == fulfillmenterrors.ErrorCredentialsInvalid.Code
}
