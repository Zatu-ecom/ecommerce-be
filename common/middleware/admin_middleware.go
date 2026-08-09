package middleware

import (
	"net/http"

	"ecommerce-be/common/auth"
	"ecommerce-be/common/config"
	"ecommerce-be/common/constants"
	commonModel "ecommerce-be/common/model"

	"github.com/gin-gonic/gin"
)

// AdminAuth middleware for admin-only access
func AdminAuth() gin.HandlerFunc {
	secret := config.Get().Auth.JWTSecret
	return func(c *gin.Context) {
		// First run the basic auth middleware
		authMiddleware := auth.AuthMiddleware(secret)
		authMiddleware(c)

		// If auth failed, the request would have been aborted
		if c.IsAborted() {
			return
		}

		// Check if user has admin role level
		roleLevel, _, exists := auth.GetUserRoleFromContext(c)
		if !exists || !auth.HasRequiredRoleLevel(roleLevel, constants.ADMIN_ROLE_LEVEL) {
			commonModel.ErrorWithCode(
				c,
				http.StatusForbidden,
				constants.INSUFFICIENT_PERMISSIONS_MSG,
				constants.INSUFFICIENT_PERMISSIONS_CODE,
			)
			c.Abort()
			return
		}

		c.Next()
	}
}
