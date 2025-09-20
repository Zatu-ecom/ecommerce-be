package auth

import (
	"net/http"
	"strings"

	"ecommerce-be/common"
	"ecommerce-be/common/cache"
	"ecommerce-be/common/constants"

	"github.com/gin-gonic/gin"
)

// AuthMiddleware creates a Gin middleware for JWT authentication
func AuthMiddleware(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get the Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			common.ErrorWithCode(
				c,
				http.StatusUnauthorized,
				constants.AUTHENTICATION_REQUIRED_MSG,
				constants.AUTH_REQUIRED_CODE,
			)
			c.Abort()
			return
		}

		// Check if the header has the Bearer prefix
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != constants.BEARER_PREFIX {
			common.ErrorWithCode(
				c,
				http.StatusUnauthorized,
				constants.INVALID_AUTH_FORMAT_MSG,
				constants.INVALID_AUTH_FORMAT_CODE,
			)
			c.Abort()
			return
		}

		// Parse and validate the token
		tokenString := parts[1]

		// Check if token is blacklisted
		if cache.IsTokenBlacklisted(tokenString) {
			common.ErrorWithCode(
				c,
				http.StatusUnauthorized,
				constants.TOKEN_REVOKED_MSG,
				constants.TOKEN_REVOKED_CODE,
			)
			c.Abort()
			return
		}

		claims, err := ParseToken(tokenString, secret)
		if err != nil {
			common.ErrorWithCode(
				c,
				http.StatusUnauthorized,
				constants.TOKEN_INVALID_MSG,
				constants.TOKEN_INVALID_CODE,
			)
			c.Abort()
			return
		}

		// Set user info in context
		c.Set(constants.USER_ID_KEY, claims.UserID)
		c.Set(constants.EMAIL_KEY, claims.Email)
		c.Set(constants.ROLE_ID_KEY, claims.RoleID)
		c.Set(constants.ROLE_NAME_KEY, claims.RoleName)
		c.Set(constants.ROLE_LEVEL_KEY, claims.RoleLevel)
		if claims.SellerID != nil {
			c.Set(constants.SELLER_ID_KEY, *claims.SellerID)
		}
	}
}
