package middleware

import (
	"net/http"

	"ecommerce-be/common"
	"ecommerce-be/common/auth"
	"ecommerce-be/common/config"
	"ecommerce-be/common/constants"
	"ecommerce-be/common/db"

	"github.com/gin-gonic/gin"
)

// CustomerAuth middleware for customer-level access (customer, seller, or admin)
// Validates associated seller subscription and verification status for customers
func CustomerAuth() gin.HandlerFunc {
	database := db.GetDB()
	secret := config.Get().Auth.JWTSecret
	return func(c *gin.Context) {
		// First run the basic auth middleware
		authMiddleware := auth.AuthMiddleware(secret)
		authMiddleware(c)

		// If auth failed, the request would have been aborted
		if c.IsAborted() {
			return
		}

		// Check if user has customer-level access or higher
		roleLevel, _, exists := auth.GetUserRoleFromContext(c)
		if !exists || !auth.HasRequiredRoleLevel(roleLevel, constants.CUSTOMER_ROLE_LEVEL) {
			common.ErrorWithCode(
				c,
				http.StatusForbidden,
				constants.INSUFFICIENT_PERMISSIONS_MSG,
				constants.INSUFFICIENT_PERMISSIONS_CODE,
			)
			c.Abort()
			return
		}

		// For customer-level users (not sellers or admins), validate their associated seller
		if roleLevel == constants.CUSTOMER_ROLE_LEVEL {
			sellerID, hasSellerID := auth.GetSellerIDFromContext(c)
			if !hasSellerID {
				common.ErrorWithCode(
					c,
					http.StatusForbidden,
					constants.CUSTOMER_NO_SELLER_MSG,
					constants.CUSTOMER_NO_SELLER_CODE,
				)
				c.Abort()
				return
			}

			// OPTIMIZED: Single query validation with complete seller data
			sellerData, err := auth.GetSellerValidationData(database, sellerID)
			if err != nil {
				common.ErrorWithCode(
					c,
					http.StatusForbidden,
					constants.SELLER_INACTIVE_FOR_CUSTOMER_MSG,
					constants.INVALID_SELLER_CODE,
				)
				c.Abort()
				return
			}

			// Validate seller access using the complete data
			if validationErr := sellerData.ValidateForAccess(); validationErr != nil {
				var errorMsg string
				var errorCode string

				switch validationErr.Error() {
				case constants.SELLER_SUBSCRIPTION_INACTIVE_MSG:
					errorMsg = constants.SELLER_SUBSCRIPTION_INACTIVE_FOR_CUSTOMER_MSG
					errorCode = constants.SELLER_SUBSCRIPTION_INACTIVE_CODE
				default:
					errorMsg = constants.SELLER_INACTIVE_FOR_CUSTOMER_MSG
					errorCode = constants.INVALID_SELLER_CODE
				}

				common.ErrorWithCode(c, http.StatusForbidden, errorMsg, errorCode)
				c.Abort()
				return
			}

			// Set seller context and complete seller data for downstream handlers
			c.Set("validated_seller_id", sellerID)
			c.Set("seller_validation_data", sellerData)
		}

		c.Next()
	}
}

// REMOVED CustomerWithFeatureAuth as plan features are no longer in the model
