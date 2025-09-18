package middleware

import (
	"net/http"
	"os"

	"ecommerce-be/common"
	"ecommerce-be/common/auth"
	"ecommerce-be/common/constants"

	"github.com/gin-gonic/gin"
)

// AdminAuth middleware for admin-only access
func AdminAuth() gin.HandlerFunc {
	secret := os.Getenv("JWT_SECRET")
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
			common.ErrorWithCode(
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
