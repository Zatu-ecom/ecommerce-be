package shiprocket

import (
	"context"
	"fmt"
	"strings"
)

// TestConnection verifies api_email/api_password with a real login attempt:
// POST {base}/auth/login. A 401 surfaces as a validation error so the
// handler maps it to 400 (bad keys), not 500. The webhook_secret is checked
// for presence only — it cannot be verified without the provider calling
// back. Nothing is persisted.
func (a *Adapter) TestConnection(ctx context.Context, creds map[string]any) error {
	parsed, err := parseCredentials(creds)
	if err != nil {
		return fmt.Errorf("[shiprocket] %w", err)
	}
	if secret := strings.TrimSpace(parsed.WebhookSecret); secret != "" && len(secret) < 8 {
		return fmt.Errorf("shiprocket: webhook_secret looks truncated (length/format check)")
	}

	// doJSON maps provider 401 to the credentials AppError, so a bad login
	// surfaces as 400 (bad keys) rather than 500. Anything else propagates.
	if _, err := a.login(ctx, parsed); err != nil {
		return err
	}
	return nil
}
