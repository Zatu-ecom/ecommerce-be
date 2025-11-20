package helpers

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// LoginRequest represents the login request body
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginResponse represents the login response data
type LoginResponse struct {
	Token string `json:"token"`
}

// Login performs a login and returns the token
func Login(t *testing.T, client *APIClient, email, password string) string {
	requestBody := LoginRequest{
		Email:    email,
		Password: password,
	}

	w := client.Post(t, "/api/auth/login", requestBody)

	assert.Equal(t, 200, w.Code, "Login should succeed")

	response := ParseResponse(t, w.Body)
	assert.True(t, response["success"].(bool), "Response should be successful")

	data, ok := response["data"].(map[string]interface{})
	assert.True(t, ok, "Response should contain data")

	token, tokenOk := data["token"].(string)
	assert.True(t, tokenOk, "Data should contain token")
	assert.NotEmpty(t, token, "Token should not be empty")

	return token
}
