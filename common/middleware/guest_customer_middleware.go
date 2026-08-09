package middleware

import (
	"net/http"
	"strconv"
	"strings"

	"ecommerce-be/common/auth"
	"ecommerce-be/common/config"
	"ecommerce-be/common/constants"
	"ecommerce-be/common/db"
	commonModel "ecommerce-be/common/model"

	"github.com/gin-gonic/gin"
)

// GuestOrCustomerAuth middleware supports two authentication modes on the same route group:
//
//  1. JWT present (Authorization header) → Full CustomerAuth flow:
//     Validates JWT, extracts userID, email, role, sellerID from claims.
//     Sets all auth context values (user_id, seller_id, role, etc.).
//
//  2. No JWT → Guest mode:
//     Requires X-Device-ID header for guest identification.
//     Requires X-Seller-ID header for multi-tenant data isolation.
//     Validates seller (active, subscription, etc.) using cached validation.
//
// Usage: Apply to guest cart routes (/api/order/cart/guest/*).
func GuestOrCustomerAuth() gin.HandlerFunc {
	secret := config.Get().Auth.JWTSecret

	return func(c *gin.Context) {
		// Check if Authorization header exists (JWT token)
		authHeader := c.GetHeader("Authorization")

		if authHeader != "" {
			// ─── JWT present: use CustomerAuth flow ─────────────────────────────
			authMiddleware := auth.AuthMiddleware(secret)
			authMiddleware(c)

			// Check if auth middleware aborted the request
			if c.IsAborted() {
				return
			}

			_, role, exists := auth.GetUserRoleFromContext(c)
			if !exists {
				commonModel.ErrorWithCode(
					c,
					http.StatusForbidden,
					constants.ROLE_NOT_FOUND_MSG,
					constants.ROLE_NOT_FOUND_CODE,
				)
				c.Abort()
				return
			}

			// Only customers can access cart routes
			if role != constants.CUSTOMER_ROLE_NAME {
				commonModel.ErrorWithCode(
					c,
					http.StatusForbidden,
					"Access denied: customer role required",
					constants.ACCESS_DENIED_CODE,
				)
				c.Abort()
				return
			}

			c.Next()
			return
		}

		// ─── No JWT: guest mode ──────────────────────────────────────────────
		// Device ID is MANDATORY for guest identification
		deviceID := c.GetHeader(constants.DEVICE_ID_HEADER)
		if deviceID == "" || len(strings.TrimSpace(deviceID)) == 0 {
			commonModel.ErrorWithCode(
				c,
				http.StatusBadRequest,
				constants.DEVICE_ID_REQUIRED_MSG,
				constants.DEVICE_ID_REQUIRED_CODE,
			)
			c.Abort()
			return
		}

		deviceID = strings.TrimSpace(deviceID)

		// Validate device ID length (UUID v4 is 36 chars, allow some flexibility)
		if len(deviceID) < 8 || len(deviceID) > 64 {
			commonModel.ErrorWithCode(
				c,
				http.StatusBadRequest,
				constants.DEVICE_ID_INVALID_MSG,
				constants.DEVICE_ID_INVALID_CODE,
			)
			c.Abort()
			return
		}

		// Seller ID is MANDATORY for multi-tenant isolation
		database := db.GetDB()
		sellerIDHeader := c.GetHeader(constants.SELLER_ID_HEADER)
		if sellerIDHeader == "" || len(strings.TrimSpace(sellerIDHeader)) == 0 {
			commonModel.ErrorWithCode(
				c,
				http.StatusBadRequest,
				constants.SELLER_ID_REQUIRED_MSG,
				constants.SELLER_ID_REQUIRED_CODE,
			)
			c.Abort()
			return
		}

		sellerIDHeader = strings.TrimSpace(sellerIDHeader)

		// Parse seller ID to uint
		sellerID64, err := strconv.ParseUint(sellerIDHeader, 10, 32)
		if err != nil || sellerID64 == 0 {
			commonModel.ErrorWithCode(
				c,
				http.StatusBadRequest,
				constants.SELLER_ID_INVALID_MSG,
				constants.SELLER_ID_INVALID_CODE,
			)
			c.Abort()
			return
		}

		sellerID := uint(sellerID64)

		// Validate seller using the cached validation method
		sellerData, validationErr := auth.ValidateSellerCompleteCached(database, sellerID)
		if validationErr != nil {
			commonModel.ErrorWithCode(
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

			commonModel.ErrorWithCode(
				c,
				http.StatusForbidden,
				accessErr.Error(),
				errorCode,
			)
			c.Abort()
			return
		}

		// Store device ID and seller ID in context for downstream handlers
		c.Set(constants.DEVICE_ID_KEY, deviceID)
		c.Set(constants.SELLER_ID_KEY, sellerID)

		c.Next()
	}
}
