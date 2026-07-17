-- Migration: 026_create_password_reset_token.sql
-- Description: Create password_reset_token table for forgot/reset password flow
-- Created: 2026-07-17

-- Create password_reset_token table
CREATE TABLE IF NOT EXISTS password_reset_token (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
    token_hash VARCHAR(64) NOT NULL,      -- SHA-256 hash of the raw reset token
    expires_at TIMESTAMPTZ NOT NULL,       -- Token expiration timestamp
    is_used BOOLEAN NOT NULL DEFAULT FALSE, -- Whether the token has been used
    used_at TIMESTAMPTZ,                    -- When the token was used (NULL if not used)
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes on password_reset_token
CREATE INDEX IF NOT EXISTS idx_password_reset_token_user_id ON password_reset_token(user_id);
CREATE INDEX IF NOT EXISTS idx_password_reset_token_token_hash ON password_reset_token(token_hash);
CREATE INDEX IF NOT EXISTS idx_password_reset_token_expires_at ON password_reset_token(expires_at);

-- Create trigger for updated_at
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'update_password_reset_token_updated_at') THEN
        CREATE TRIGGER update_password_reset_token_updated_at BEFORE UPDATE ON password_reset_token
        FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
    END IF;
END
$$ language plpgsql;
