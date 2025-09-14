package middleware

import (
	"net/http"
	"os"
	"time"

	"ecommerce-be/common"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// Logger middleware for logging HTTP requests
func Logger() gin.HandlerFunc {
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})

	return func(c *gin.Context) {
		startTime := time.Now()

		// Process request
		c.Next()

		// Log request details after processing
		duration := time.Since(startTime)
		logger.WithFields(logrus.Fields{
			"method":     c.Request.Method,
			"path":       c.Request.URL.Path,
			"status":     c.Writer.Status(),
			"duration":   duration,
			"client_ip":  c.ClientIP(),
			"user_agent": c.Request.UserAgent(),
		}).Info("Request processed")
	}
}

// CORS middleware for handling Cross-Origin Resource Sharing
func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().
			Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE, PATCH")
		c.Writer.Header().
			Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// ADMIN > SELLER > CUSTOMER
// Auth middleware for authentication
func Auth() gin.HandlerFunc {
	return common.AuthMiddleware(os.Getenv("JWT_SECRET"))
}
