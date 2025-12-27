package middleware

import (
	"net/http"
	"strconv"
	"strings"

	"ecommerce-be/common"
	"ecommerce-be/common/auth"
	"ecommerce-be/common/config"
	"ecommerce-be/common/constants"
	"ecommerce-be/common/db"

	"github.com/gin-gonic/gin"
)

// PublicAPIAuth middleware for public APIs that don't require JWT token
// but REQUIRE seller ID in header for multi-tenant data isolation
//
// This middleware:
// 1. Checks if JWT token exists (Authorization header)
// 2. If token does NOT exist:
//   - Seller ID in X-Seller-ID header is MANDATORY
//   - Validates seller using seller validation (active, subscription, etc.)
//   - Stores seller ID and validation data in context
//
// 3. If token exists:
//   - Skip validation (will be handled by auth middleware)
//
// Usage: Apply to public routes like GET /products, GET /categories, etc.
func PublicAPIAuth() gin.HandlerFunc {
	database := db.GetDB()
	secret := config.Get().Auth.JWTSecret

	return func(c *gin.Context) {
		// Check if Authorization header exists (JWT token)
		authHeader := c.GetHeader("Authorization")

		// If token exists then check respective role middleware
		if authHeader != "" {
			authMiddleware := auth.AuthMiddleware(secret)
			authMiddleware(c)

			// Check if auth middleware aborted the request
			if c.IsAborted() {
				return // ← ADD THIS CHECK
			}

			_, role, exists := auth.GetUserRoleFromContext(c)

			if !exists {
				common.ErrorWithCode(
					c,
					http.StatusForbidden,
					constants.ROLE_NOT_FOUND_MSG,
					constants.ROLE_NOT_FOUND_CODE,
				)
				c.Abort()
				return
			}

			switch role {
			case constants.CUSTOMER_ROLE_NAME:
				CustomerAuth()
			case constants.SELLER_ROLE_NAME:
				SellerAuth()
			case constants.ADMIN_ROLE_NAME:
				AdminAuth()
			}
			c.Next()
			return
		}

		// Token does NOT exist - this is a public API call
		// Seller ID is MANDATORY for multi-tenant isolation

		// Extract seller ID from X-Seller-ID header
		sellerIDHeader := c.GetHeader(constants.SELLER_ID_HEADER)

		// Validate that seller ID is provided and not empty/whitespace
		// Handles: "", "  ", null header
		if sellerIDHeader == "" || len(strings.TrimSpace(sellerIDHeader)) == 0 {
			common.ErrorWithCode(
				c,
				http.StatusBadRequest,
				constants.SELLER_ID_REQUIRED_MSG,
				constants.SELLER_ID_REQUIRED_CODE,
			)
			c.Abort()
			return
		}

		// Trim whitespace before parsing
		sellerIDHeader = strings.TrimSpace(sellerIDHeader)

		// Parse seller ID to uint
		// Handles: "abc", "null", "-1", "1.5", etc. → returns error
		sellerID, err := strconv.ParseUint(sellerIDHeader, 10, 32)
		if err != nil {
			common.ErrorWithCode(
				c,
				http.StatusBadRequest,
				constants.SELLER_ID_INVALID_MSG,
				constants.SELLER_ID_INVALID_CODE,
			)
			c.Abort()
			return
		}

		// Validate seller ID is greater than 0
		// Handles: "0" → returns error
		if sellerID == 0 {
			common.ErrorWithCode(
				c,
				http.StatusBadRequest,
				constants.SELLER_ID_INVALID_MSG,
				constants.SELLER_ID_INVALID_CODE,
			)
			c.Abort()
			return
		}

		// Validate seller using the cached validation method
		// This checks:
		// - Seller exists
		// - Seller is active
		// - Subscription is active
		// - All other seller validations
		sellerData, validationErr := auth.ValidateSellerCompleteCached(database, uint(sellerID))
		if validationErr != nil {
			common.ErrorWithCode(
				c,
				http.StatusForbidden,
				validationErr.Error(),
				constants.INVALID_SELLER_CODE,
			)
			c.Abort()
			return
		}

		// Validate seller access (active, subscription status, etc.)
		if accessErr := sellerData.ValidateForAccess(); accessErr != nil {
			var errorCode string
			switch accessErr.Error() {
			case constants.SELLER_SUBSCRIPTION_INACTIVE_MSG:
				errorCode = constants.SELLER_SUBSCRIPTION_INACTIVE_CODE
			case constants.SELLER_NOT_VERIFIED_MSG:
				errorCode = constants.SELLER_NOT_VERIFIED_CODE
			default:
				errorCode = constants.INVALID_SELLER_CODE
			}

			common.ErrorWithCode(
				c,
				http.StatusForbidden,
				accessErr.Error(),
				errorCode,
			)
			c.Abort()
			return
		}

		// Store seller ID and validation data in context for downstream handlers
		c.Set(constants.SELLER_ID_KEY, uint(sellerID))

		c.Next()
	}
}
