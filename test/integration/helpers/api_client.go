package helpers

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// APIClient wraps HTTP requests for testing
type APIClient struct {
	Handler http.Handler
	Token   string
}

// NewAPIClient creates a new API client for testing
func NewAPIClient(handler http.Handler) *APIClient {
	return &APIClient{
		Handler: handler,
	}
}

// SetToken sets the authentication token for subsequent requests
func (c *APIClient) SetToken(token string) {
	c.Token = token
}

// Post makes a POST request
func (c *APIClient) Post(t *testing.T, url string, body interface{}) *httptest.ResponseRecorder {
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("failed to marshal request body: %v", err)
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(bodyBytes))
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}

	w := httptest.NewRecorder()
	c.Handler.ServeHTTP(w, req)

	return w
}

// Get makes a GET request
func (c *APIClient) Get(t *testing.T, url string) *httptest.ResponseRecorder {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}

	w := httptest.NewRecorder()
	c.Handler.ServeHTTP(w, req)

	return w
}

// Put makes a PUT request
func (c *APIClient) Put(t *testing.T, url string, body interface{}) *httptest.ResponseRecorder {
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("failed to marshal request body: %v", err)
	}

	req, err := http.NewRequest(http.MethodPut, url, bytes.NewBuffer(bodyBytes))
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}

	w := httptest.NewRecorder()
	c.Handler.ServeHTTP(w, req)

	return w
}

// Delete makes a DELETE request
func (c *APIClient) Delete(t *testing.T, url string) *httptest.ResponseRecorder {
	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}

	w := httptest.NewRecorder()
	c.Handler.ServeHTTP(w, req)

	return w
}

// ParseResponse parses JSON response into a map
func ParseResponse(t *testing.T, body io.Reader) map[string]interface{} {
	var response map[string]interface{}
	if err := json.NewDecoder(body).Decode(&response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	return response
}
