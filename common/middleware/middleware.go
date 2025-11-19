package middleware

import (
	"ecommerce-be/common"
	"ecommerce-be/common/constants"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// Logger middleware for logging HTTP requests
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		startTime := time.Now()

		// Process request
		c.Next()

		// Log request details after processing
		duration := time.Since(startTime)
		
		fields := logrus.Fields{
			"method":    c.Request.Method,
			"path":      c.Request.URL.Path,
			"status":    c.Writer.Status(),
			"duration":  duration.Milliseconds(),
			"clientIp":  c.ClientIP(),
			"userAgent": c.Request.UserAgent(),
		}
		
		// Add correlation ID if present
		if correlationID, exists := c.Get(constants.CORRELATION_ID_KEY); exists {
			fields["correlationId"] = correlationID
		}
		
		// Add seller ID if present
		if sellerID, exists := c.Get(constants.SELLER_ID_KEY); exists {
			fields["sellerId"] = sellerID
		}
		
		logrus.WithFields(fields).Info("Request processed")
	}
}

// CorrelationID middleware ensures every request has a correlation ID
// If not provided in header, generates a new UUID
// This is mandatory for all requests
func CorrelationID() gin.HandlerFunc {
	return func(c *gin.Context) {
		correlationID := c.GetHeader(constants.CORRELATION_ID_HEADER)

		// If no correlation ID provided, reject the request
		if correlationID == "" {
			common.ErrorWithCode(
				c,
				http.StatusBadRequest,
				constants.CORRELATION_ID_REQUIRED_MSG,
				constants.CORRELATION_ID_REQUIRED_CODE,
			)
			c.Abort()
			return
		}

		// Validate correlation ID format (basic validation)
		correlationID = strings.TrimSpace(correlationID)
		if len(correlationID) == 0 || len(correlationID) > 100 {
			common.ErrorWithCode(
				c,
				http.StatusBadRequest,
				constants.CORRELATION_ID_INVALID_MSG,
				constants.CORRELATION_ID_INVALID_CODE,
			)
			c.Abort()
			return
		}

		// Set correlation ID in context for use throughout the request
		c.Set(constants.CORRELATION_ID_KEY, correlationID)

		// Add correlation ID to response headers for traceability
		c.Writer.Header().Set(constants.CORRELATION_ID_HEADER, correlationID)

		c.Next()
	}
}

// GenerateCorrelationID middleware generates a correlation ID if not provided
// Use this for backward compatibility or non-critical endpoints
func GenerateCorrelationID() gin.HandlerFunc {
	return func(c *gin.Context) {
		correlationID := c.GetHeader(constants.CORRELATION_ID_HEADER)

		// If no correlation ID provided, generate one
		if correlationID == "" {
			correlationID = uuid.New().String()
		}

		// Set correlation ID in context
		c.Set(constants.CORRELATION_ID_KEY, correlationID)

		// Add correlation ID to response headers
		c.Writer.Header().Set(constants.CORRELATION_ID_HEADER, correlationID)

		c.Next()
	}
}

// CORS middleware for handling Cross-Origin Resource Sharing
func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().
			Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE, PATCH")
		c.Writer.Header().
			Set("Access-Control-Allow-Headers",
				"Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, X-Seller-ID, X-Correlation-ID")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
