package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"ecommerce-be/common/middleware"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMatchesSkipRule_PrefixAndMethod(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rules := []middleware.PathSkipRule{
		{Methods: []string{http.MethodPost}, PathPrefix: "/api/payment/webhooks/"},
	}

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	req := httptest.NewRequest(http.MethodPost, "/api/payment/webhooks/razorpay", nil)
	c.Request = req
	assert.True(t, middleware.MatchesSkipRule(c, rules))

	c.Request = httptest.NewRequest(http.MethodGet, "/api/payment/webhooks/razorpay", nil)
	assert.False(t, middleware.MatchesSkipRule(c, rules), "GET should not match POST-only rule")

	c.Request = httptest.NewRequest(http.MethodPost, "/api/payment/initiate", nil)
	assert.False(t, middleware.MatchesSkipRule(c, rules), "unrelated path should not match")
}

func TestMatchesSkipRule_ExactPath(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rules := []middleware.PathSkipRule{
		{PathExact: "/api/health"},
	}

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodGet, "/api/health", nil)
	assert.True(t, middleware.MatchesSkipRule(c, rules))

	c.Request = httptest.NewRequest(http.MethodGet, "/api/health/live", nil)
	assert.False(t, middleware.MatchesSkipRule(c, rules), "prefix of exact path should not match")
}

func TestMatchesSkipRule_EmptyRules(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/api/payment/webhooks/razorpay", nil)
	assert.False(t, middleware.MatchesSkipRule(c, nil))
}

func TestUnlessSkipped_RunsFallbackOnMatch(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middleware.UnlessSkipped(
		[]middleware.PathSkipRule{{Methods: []string{http.MethodPost}, PathPrefix: "/api/payment/webhooks/"}},
		func(c *gin.Context) {
			c.AbortWithStatus(http.StatusBadRequest)
		},
		func(c *gin.Context) {
			c.Header("X-Fallback", "1")
			c.Next()
		},
	))
	router.POST("/api/payment/webhooks/razorpay", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	router.POST("/api/payment/initiate", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	matched := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/payment/webhooks/razorpay", nil)
	router.ServeHTTP(matched, req)
	require.Equal(t, http.StatusOK, matched.Code)
	assert.Equal(t, "1", matched.Header().Get("X-Fallback"))

	unmatched := httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/payment/initiate", nil)
	router.ServeHTTP(unmatched, req)
	require.Equal(t, http.StatusBadRequest, unmatched.Code)
}

func TestUnlessSkipped_NilFallbackCallsNext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middleware.UnlessSkipped(
		[]middleware.PathSkipRule{{PathPrefix: "/skip/"}},
		func(c *gin.Context) {
			c.AbortWithStatus(http.StatusTeapot)
		},
		nil,
	))
	router.GET("/skip/me", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/skip/me", nil))
	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestCorrelationIDWebhookSkipRules_MatchesRazorpayWebhook(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/api/payment/webhooks/razorpay", nil)
	assert.True(t, middleware.MatchesSkipRule(c, middleware.CorrelationIDWebhookSkipRules))
}
