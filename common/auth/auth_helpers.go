package auth

import (
	"ecommerce-be/common/constants"

	"github.com/gin-gonic/gin"
)

/********************************************************************
*			Helper functions for role-based authentication			*
*********************************************************************/

// HasRequiredRoleLevel checks if the user's role level meets the minimum requirement
// Lower numbers indicate higher authority (1=ADMIN, 2=SELLER, 3=CUSTOMER)
func HasRequiredRoleLevel(userRoleLevel, requiredRoleLevel uint) bool {
	return userRoleLevel <= requiredRoleLevel
}

// GetUserRoleFromContext extracts user role information from Gin context
func GetUserRoleFromContext(c *gin.Context) (roleLevel uint, roleName string, exists bool) {
	roleLevelValue, exists := c.Get(constants.ROLE_LEVEL_KEY)
	if !exists {
		return 0, "", false
	}
	roleNameValue, exists := c.Get(constants.ROLE_NAME_KEY)
	if !exists {
		return 0, "", false
	}
	return roleLevelValue.(uint), roleNameValue.(string), true
}

// GetSellerIDFromContext extracts seller ID from Gin context
func GetSellerIDFromContext(c *gin.Context) (sellerID uint, exists bool) {
	sellerIDValue, exists := c.Get(constants.SELLER_ID_KEY)
	if !exists {
		return 0, false
	}
	return sellerIDValue.(uint), true
}
