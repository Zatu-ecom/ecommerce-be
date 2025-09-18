package middleware

import (
	"net/http"
	"os"

	"ecommerce-be/common"
	"ecommerce-be/common/db"

	"ecommerce-be/common/auth"
	"ecommerce-be/common/constants"

	"github.com/gin-gonic/gin"
)

// SellerAuth middleware for seller-level access (seller or admin)
// Validates seller subscription and verification status
func SellerAuth() gin.HandlerFunc {
	db := db.GetDB()
	secret := os.Getenv("JWT_SECRET")
	return func(c *gin.Context) {
		// First run the basic auth middleware
		authMiddleware := auth.AuthMiddleware(secret)
		authMiddleware(c)

		// If auth failed, the request would have been aborted
		if c.IsAborted() {
			return
		}

		// Check if user has seller-level access or higher
		roleLevel, _, exists := auth.GetUserRoleFromContext(c)
		if !exists || !auth.HasRequiredRoleLevel(roleLevel, constants.SELLER_ROLE_LEVEL) {
			common.ErrorWithCode(
				c,
				http.StatusForbidden,
				constants.INSUFFICIENT_PERMISSIONS_MSG,
				constants.INSUFFICIENT_PERMISSIONS_CODE,
			)
			c.Abort()
			return
		}

		// For seller-level users (not admins), validate seller-specific requirements
		if roleLevel == constants.SELLER_ROLE_LEVEL {
			sellerID, hasSellerID := auth.GetSellerIDFromContext(c)
			if !hasSellerID {
				common.ErrorWithCode(
					c,
					http.StatusForbidden,
					constants.INVALID_SELLER_MSG,
					constants.INVALID_SELLER_CODE,
				)
				c.Abort()
				return
			}

			// OPTIMIZED: Single query validation with complete seller data
			sellerData, err := auth.ValidateSellerCompleteCached(db, sellerID)
			if err != nil {
				common.ErrorWithCode(
					c,
					http.StatusForbidden,
					err.Error(),
					constants.INVALID_SELLER_CODE,
				)
				c.Abort()
				return
			}

			// Validate seller access using the complete data
			if validationErr := sellerData.ValidateForAccess(); validationErr != nil {
				var errorCode string
				switch validationErr.Error() {
				case constants.SELLER_SUBSCRIPTION_INACTIVE_MSG:
					errorCode = constants.SELLER_SUBSCRIPTION_INACTIVE_CODE
				default:
					errorCode = constants.INVALID_SELLER_CODE
				}
				common.ErrorWithCode(c, http.StatusForbidden, validationErr.Error(), errorCode)
				c.Abort()
				return
			}

			// Set complete seller data for downstream handlers
			c.Set("seller_validation_data", sellerData)
		}

		c.Next()
	}
}
