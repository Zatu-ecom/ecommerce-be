package user

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"testing"
	"time"

	"ecommerce-be/test/integration/helpers"
	"ecommerce-be/test/integration/setup"
	"ecommerce-be/user/utils/constant"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

// PasswordResetTestSuite tests the forgot-password and reset-password API endpoints
type PasswordResetTestSuite struct {
	suite.Suite
	container *setup.TestContainer
	server    http.Handler
	client    *helpers.APIClient
}

func (s *PasswordResetTestSuite) SetupSuite() {
	s.container = setup.SetupTestContainers(s.T())

	// Run migrations and seeds
	s.container.RunAllMigrations(s.T())
	s.container.RunAllSeeds(s.T())

	// Setup test server
	s.server = setup.SetupTestServer(s.T(), s.container.DB, s.container.RedisClient)

	// Create API client
	s.client = helpers.NewAPIClient(s.server)
}

func (s *PasswordResetTestSuite) TearDownSuite() {
	if s.container != nil {
		s.container.Cleanup(s.T())
	}
}

func TestPasswordResetSuite(t *testing.T) {
	suite.Run(t, new(PasswordResetTestSuite))
}

// insertResetToken inserts a password_reset_token row directly into the DB
// with a computed SHA-256 hash of the rawToken. Returns the token_hash stored.
// Use this helper to create test tokens that can be consumed by the reset endpoint.
func (s *PasswordResetTestSuite) insertResetToken(
	userID uint,
	rawToken string,
	ttl time.Duration,
) string {
	hash := sha256.Sum256([]byte(rawToken))
	tokenHash := hex.EncodeToString(hash[:])
	expiresAt := time.Now().UTC().Add(ttl)

	result := s.container.DB.Exec(
		`INSERT INTO password_reset_token (user_id, token_hash, expires_at, is_used, created_at, updated_at)
		 VALUES (?, ?, ?, false, NOW(), NOW())`,
		userID,
		tokenHash,
		expiresAt,
	)
	s.Require().NoError(result.Error, "failed to insert test reset token")
	return tokenHash
}

// ========================================
// Forgot Password Tests
// ========================================

// TestForgotPassword_ExistingEmail returns success for a known customer email
func (s *PasswordResetTestSuite) TestForgotPassword_ExistingEmail() {
	w := s.client.Post(
		s.T(),
		FORGOT_PASSWORD_PATH,
		helpers.BuildForgotPasswordRequest(helpers.CustomerEmail),
	)

	assert.Equal(s.T(), http.StatusOK, w.Code)

	response := helpers.ParseResponse(s.T(), w.Body)
	assert.True(s.T(), response["success"].(bool))
	assert.Contains(s.T(), response["message"], "password reset link")
	assert.Nil(s.T(), response["data"])
}

// TestForgotPassword_NonExistentEmail returns the same success message (no user enumeration)
func (s *PasswordResetTestSuite) TestForgotPassword_NonExistentEmail() {
	w := s.client.Post(
		s.T(),
		FORGOT_PASSWORD_PATH,
		helpers.BuildForgotPasswordRequest("nonexistent@example.com"),
	)

	assert.Equal(s.T(), http.StatusOK, w.Code)

	response := helpers.ParseResponse(s.T(), w.Body)
	assert.True(s.T(), response["success"].(bool))
	assert.Contains(s.T(), response["message"], "password reset link")
	assert.Nil(s.T(), response["data"])
}

// TestForgotPassword_ExistingSellerEmail returns success for an existing seller email
func (s *PasswordResetTestSuite) TestForgotPassword_ExistingSellerEmail() {
	w := s.client.Post(
		s.T(),
		FORGOT_PASSWORD_PATH,
		helpers.BuildForgotPasswordRequest(helpers.SellerEmail),
	)

	assert.Equal(s.T(), http.StatusOK, w.Code)

	response := helpers.ParseResponse(s.T(), w.Body)
	assert.True(s.T(), response["success"].(bool))
}

// TestForgotPassword_MissingEmail returns validation error for empty email
func (s *PasswordResetTestSuite) TestForgotPassword_MissingEmail() {
	w := s.client.Post(s.T(), FORGOT_PASSWORD_PATH, helpers.BuildForgotPasswordRequest(""))

	assert.Equal(s.T(), http.StatusBadRequest, w.Code)

	response := helpers.ParseResponse(s.T(), w.Body)
	assert.False(s.T(), response["success"].(bool))
}

// TestForgotPassword_InvalidEmailFormat returns validation error
func (s *PasswordResetTestSuite) TestForgotPassword_InvalidEmailFormat() {
	w := s.client.Post(
		s.T(),
		FORGOT_PASSWORD_PATH,
		helpers.BuildForgotPasswordRequest("not-an-email"),
	)

	assert.Equal(s.T(), http.StatusBadRequest, w.Code)

	response := helpers.ParseResponse(s.T(), w.Body)
	assert.False(s.T(), response["success"].(bool))
}

// TestForgotPassword_EmptyJSONBody returns validation error for missing email key
func (s *PasswordResetTestSuite) TestForgotPassword_EmptyJSONBody() {
	w := s.client.Post(s.T(), FORGOT_PASSWORD_PATH, map[string]any{})

	assert.Equal(s.T(), http.StatusBadRequest, w.Code)

	response := helpers.ParseResponse(s.T(), w.Body)
	assert.False(s.T(), response["success"].(bool))
}

// TestForgotPassword_MultipleRequests allows multiple forgot-password requests for the same user
func (s *PasswordResetTestSuite) TestForgotPassword_MultipleRequests() {
	// First request
	w := s.client.Post(
		s.T(),
		FORGOT_PASSWORD_PATH,
		helpers.BuildForgotPasswordRequest(helpers.CustomerEmail),
	)
	assert.Equal(s.T(), http.StatusOK, w.Code)

	// Second request — should succeed, creating a new token
	w = s.client.Post(
		s.T(),
		FORGOT_PASSWORD_PATH,
		helpers.BuildForgotPasswordRequest(helpers.CustomerEmail),
	)
	assert.Equal(s.T(), http.StatusOK, w.Code)

	response := helpers.ParseResponse(s.T(), w.Body)
	assert.True(s.T(), response["success"].(bool))
}

// ========================================
// Reset Password Tests
// ========================================

// TestResetPassword_Success validates full happy path:
// 1. Create a reset token directly in the DB with a known raw token
// 2. Use the raw token to reset the password
// 3. Login with the new password
func (s *PasswordResetTestSuite) TestResetPassword_Success() {
	rawToken := "test-raw-token-for-success"
	newPassword := "newcustomer123"
	s.insertResetToken(helpers.CustomerUserID, rawToken, 1*time.Hour)

	// Use the raw token to reset the password
	w := s.client.Post(
		s.T(),
		RESET_PASSWORD_PATH,
		helpers.BuildResetPasswordRequest(rawToken, newPassword, newPassword),
	)
	assert.Equal(s.T(), http.StatusOK, w.Code)

	response := helpers.ParseResponse(s.T(), w.Body)
	assert.True(s.T(), response["success"].(bool))
	assert.Equal(s.T(), constant.RESET_PASSWORD_SUCCESS_MSG, response["message"])
	assert.Nil(s.T(), response["data"])

	// Login with the new password to confirm it works
	loginBody := map[string]any{
		"email":    helpers.CustomerEmail,
		"password": newPassword,
	}
	w = s.client.Post(s.T(), "/api/user/auth/login", loginBody)
	assert.Equal(s.T(), http.StatusOK, w.Code)
}

// TestResetPassword_InvalidToken returns error for a token that does not exist in the database
func (s *PasswordResetTestSuite) TestResetPassword_InvalidToken() {
	w := s.client.Post(
		s.T(),
		RESET_PASSWORD_PATH,
		helpers.BuildResetPasswordRequest(
			"invalid-token-that-does-not-exist-in-database",
			"newpassword123",
			"newpassword123",
		),
	)

	assert.Equal(s.T(), http.StatusBadRequest, w.Code)

	response := helpers.ParseResponse(s.T(), w.Body)
	assert.False(s.T(), response["success"].(bool))
	assert.Contains(s.T(), response["message"], "Invalid")
}

// TestResetPassword_ExpiredToken returns error for expired token
func (s *PasswordResetTestSuite) TestResetPassword_ExpiredToken() {
	s.insertResetToken(helpers.CustomerUserID, "test-expired-raw-token", -1*time.Hour)

	w := s.client.Post(
		s.T(),
		RESET_PASSWORD_PATH,
		helpers.BuildResetPasswordRequest(
			"test-expired-raw-token",
			"newpassword123",
			"newpassword123",
		),
	)

	assert.Equal(s.T(), http.StatusBadRequest, w.Code)

	response := helpers.ParseResponse(s.T(), w.Body)
	assert.False(s.T(), response["success"].(bool))
	assert.Contains(s.T(), response["message"], "expired")
}

// TestResetPassword_UsedToken returns error for token already used
func (s *PasswordResetTestSuite) TestResetPassword_UsedToken() {
	rawToken := "test-used-raw-token"
	tokenHash := s.insertResetToken(helpers.CustomerUserID, rawToken, 1*time.Hour)

	// Mark it as used via the repository
	result := s.container.DB.Exec(
		`UPDATE password_reset_token SET is_used = true, used_at = NOW() WHERE token_hash = ?`,
		tokenHash,
	)
	s.Require().NoError(result.Error)

	// Try to use the same token
	w := s.client.Post(
		s.T(),
		RESET_PASSWORD_PATH,
		helpers.BuildResetPasswordRequest(rawToken, "anotherpassword", "anotherpassword"),
	)

	assert.Equal(s.T(), http.StatusBadRequest, w.Code)

	response := helpers.ParseResponse(s.T(), w.Body)
	assert.False(s.T(), response["success"].(bool))
	assert.Contains(s.T(), response["message"], "already been used")
}

// TestResetPassword_PasswordMismatch returns error when passwords don't match
func (s *PasswordResetTestSuite) TestResetPassword_PasswordMismatch() {
	rawToken := "test-mismatch-raw-token"
	s.insertResetToken(helpers.CustomerUserID, rawToken, 1*time.Hour)

	// Try with mismatched passwords
	w := s.client.Post(
		s.T(),
		RESET_PASSWORD_PATH,
		helpers.BuildResetPasswordRequest(rawToken, "newpassword1", "newpassword2"),
	)

	assert.Equal(s.T(), http.StatusBadRequest, w.Code)

	response := helpers.ParseResponse(s.T(), w.Body)
	assert.False(s.T(), response["success"].(bool))
	assert.Contains(s.T(), response["message"], "do not match")
}

// TestResetPassword_MissingToken returns validation error
func (s *PasswordResetTestSuite) TestResetPassword_MissingToken() {
	w := s.client.Post(
		s.T(),
		RESET_PASSWORD_PATH,
		helpers.BuildResetPasswordRequest("", "newpassword123", "newpassword123"),
	)

	assert.Equal(s.T(), http.StatusBadRequest, w.Code)

	response := helpers.ParseResponse(s.T(), w.Body)
	assert.False(s.T(), response["success"].(bool))
}

// TestResetPassword_MissingPassword returns validation error when newPassword is absent
func (s *PasswordResetTestSuite) TestResetPassword_MissingPassword() {
	requestBody := map[string]any{
		"token":           "sometokenvaluehere",
		"confirmPassword": "newpassword123",
	}

	w := s.client.Post(s.T(), RESET_PASSWORD_PATH, requestBody)

	assert.Equal(s.T(), http.StatusBadRequest, w.Code)

	response := helpers.ParseResponse(s.T(), w.Body)
	assert.False(s.T(), response["success"].(bool))
}

// TestResetPassword_MissingConfirmPassword returns validation error when confirmPassword is absent
func (s *PasswordResetTestSuite) TestResetPassword_MissingConfirmPassword() {
	requestBody := map[string]any{
		"token":       "sometokenvaluehere",
		"newPassword": "newpassword123",
	}

	w := s.client.Post(s.T(), RESET_PASSWORD_PATH, requestBody)

	assert.Equal(s.T(), http.StatusBadRequest, w.Code)

	response := helpers.ParseResponse(s.T(), w.Body)
	assert.False(s.T(), response["success"].(bool))
}

// TestResetPassword_ShortPassword returns validation error for password < 6 chars
func (s *PasswordResetTestSuite) TestResetPassword_ShortPassword() {
	w := s.client.Post(
		s.T(),
		RESET_PASSWORD_PATH,
		helpers.BuildResetPasswordRequest("sometokenvaluehere", "abc", "abc"),
	)

	assert.Equal(s.T(), http.StatusBadRequest, w.Code)

	response := helpers.ParseResponse(s.T(), w.Body)
	assert.False(s.T(), response["success"].(bool))
}

// TestResetPassword_InvalidTokenFormat returns error for a long non-hex token that does not exist
func (s *PasswordResetTestSuite) TestResetPassword_InvalidTokenFormat() {
	// Use a very long invalid token (different from empty token test)
	w := s.client.Post(s.T(), RESET_PASSWORD_PATH, helpers.BuildResetPasswordRequest(
		"!!!invalid!!!token!!!format!!!with!!!special!!!chars!!!that!!!does!!!not!!!exist!!!in!!!database!!!",
		"newpassword123",
		"newpassword123",
	))

	assert.Equal(s.T(), http.StatusBadRequest, w.Code)

	response := helpers.ParseResponse(s.T(), w.Body)
	assert.False(s.T(), response["success"].(bool))
	assert.Contains(s.T(), response["message"], "Invalid")
}

// TestResetPassword_SellerUser validates that seller users can also reset passwords
func (s *PasswordResetTestSuite) TestResetPassword_SellerUser() {
	rawToken := "test-seller-raw-token"
	s.insertResetToken(helpers.SellerUserID, rawToken, 1*time.Hour)

	w := s.client.Post(
		s.T(),
		RESET_PASSWORD_PATH,
		helpers.BuildResetPasswordRequest(rawToken, "newseller456", "newseller456"),
	)
	assert.Equal(s.T(), http.StatusOK, w.Code)

	response := helpers.ParseResponse(s.T(), w.Body)
	assert.True(s.T(), response["success"].(bool))
}

// TestResetPassword_SameAsOldPassword allows resetting to the same password
func (s *PasswordResetTestSuite) TestResetPassword_SameAsOldPassword() {
	rawToken := "test-same-pw-raw-token"
	s.insertResetToken(helpers.CustomerUserID, rawToken, 1*time.Hour)

	// Reset to the same password as the seed data
	w := s.client.Post(
		s.T(),
		RESET_PASSWORD_PATH,
		helpers.BuildResetPasswordRequest(
			rawToken,
			helpers.CustomerPassword,
			helpers.CustomerPassword,
		),
	)
	assert.Equal(s.T(), http.StatusOK, w.Code)

	response := helpers.ParseResponse(s.T(), w.Body)
	assert.True(s.T(), response["success"].(bool))

	// Login with the (same) password still works
	loginBody := map[string]any{
		"email":    helpers.CustomerEmail,
		"password": helpers.CustomerPassword,
	}
	w = s.client.Post(s.T(), "/api/user/auth/login", loginBody)
	assert.Equal(s.T(), http.StatusOK, w.Code)
}
