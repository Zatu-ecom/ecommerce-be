package shiprocket

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
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

// tokenKeyFor scopes cached logins by credential identity (email + password
// hash), not email alone: rotation and seller-override rows sharing an email
// must not collide. Password never leaves this hash (key only, never logged).
func tokenKeyFor(creds *shiprocketCredentials) string {
	sum := sha256.Sum256([]byte(creds.APIPassword))
	return strings.ToLower(strings.TrimSpace(creds.APIEmail)) + "|" + fmt.Sprintf("%x", sum)
}

// tokenCache holds logins by credential key with mutex-guarded expiry.
func (a *Adapter) cachedToken(key string) (string, bool) {
	a.tokenMu.RLock()
	defer a.tokenMu.RUnlock()
	entry, ok := a.tokens[key]
	if !ok || time.Now().After(entry.expiresAt) {
		return "", false
	}
	return entry.token, true
}

func (a *Adapter) storeToken(key, token string) {
	a.tokenMu.Lock()
	defer a.tokenMu.Unlock()
	if a.tokens == nil {
		a.tokens = map[string]tokenEntry{}
	}
	a.tokens[key] = tokenEntry{token: token, expiresAt: tokenExpiry(token)}
}

func (a *Adapter) purgeToken(key string) {
	a.tokenMu.Lock()
	defer a.tokenMu.Unlock()
	delete(a.tokens, key)
}

// tokenExpiry prefers the JWT exp claim (minus skew) over the fixed 9d cap,
// so provider lifetime changes heal without waiting for 401s.
func tokenExpiry(token string) time.Time {
	cap := time.Now().Add(tokenTTL)
	if exp, ok := parseJWTExp(token); ok && !exp.IsZero() {
		if skewed := exp.Add(-5 * time.Minute); skewed.Before(cap) {
			return skewed
		}
	}
	return cap
}

// parseJWTExp extracts exp without verification (expiry only, never auth).
func parseJWTExp(token string) (time.Time, bool) {
	parts := strings.Split(token, ".")
	if len(parts) < 2 {
		return time.Time{}, false
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return time.Time{}, false
	}
	var claims struct {
		Exp int64 `json:"exp"`
	}
	if err := json.Unmarshal(payload, &claims); err != nil || claims.Exp == 0 {
		return time.Time{}, false
	}
	return time.Unix(claims.Exp, 0), true
}

// token returns a live JWT for the credentials, logging in on miss/expiry.
// Concurrent misses collapse to one provider login via singleflight.
func (a *Adapter) token(ctx context.Context, creds *shiprocketCredentials) (string, error) {
	key := tokenKeyFor(creds)
	if cached, ok := a.cachedToken(key); ok {
		return cached, nil
	}
	v, err, _ := a.loginFlight.Do(key, func() (any, error) {
		if cached, ok := a.cachedToken(key); ok {
			return cached, nil
		}
		tok, lerr := a.login(ctx, creds)
		if lerr != nil {
			return "", lerr
		}
		a.storeToken(key, tok)
		return tok, nil
	})
	if err != nil {
		return "", err
	}
	tok, _ := v.(string)
	if tok == "" {
		return "", fmt.Errorf("shiprocket: token missing after login")
	}
	return tok, nil
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
		a.purgeToken(tokenKeyFor(creds))
		token, err = a.token(ctx, creds)
		if err != nil {
			return nil, err
		}
		return fn(ctx, token)
	}
	return nil, err
}

// withAuthResult is the typed twin of withAuth for single-call bookings.
// WARNING: never pass a multi-step fn (create→assign→pickup) here — a 401 on
// a later leg would re-run earlier POSTs and double-create provider orders.
// Multi-step flows must use stepWithRefresh per leg (see shipment.go/ndr.go).
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
		a.purgeToken(tokenKeyFor(creds))
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

// stepWithRefresh runs one provider POST with the current token, purging +
// re-logging in and retrying that leg once on 401. It returns the (possibly
// refreshed) token so subsequent legs continue on the live token without
// re-running already-successful steps.
func (a *Adapter) stepWithRefresh(
	ctx context.Context,
	creds *shiprocketCredentials,
	token string,
	op func(ctx context.Context, token string) (map[string]any, error),
) (map[string]any, string, error) {
	out, err := op(ctx, token)
	if err == nil {
		return out, token, nil
	}
	if !isUnauthorized(err) {
		return nil, token, err
	}
	a.purgeToken(tokenKeyFor(creds))
	fresh, lerr := a.token(ctx, creds)
	if lerr != nil {
		return nil, token, lerr
	}
	out, err = op(ctx, fresh)
	if err != nil {
		return nil, fresh, err
	}
	return out, fresh, nil
}
