package auth

import (
	"ecommerce-be/common/constants"
	commonError "ecommerce-be/common/error"

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

func GetUserRoleLevelFromContext(c *gin.Context) (roleLevel uint, exists bool) {
	roleLevelValue, exists := c.Get(constants.ROLE_LEVEL_KEY)
	if !exists {
		return 0, false
	}
	return roleLevelValue.(uint), true
}

// ValidateUserHasSellerRoleOrHigherAndReturnAuthData validates that:
// 1. Role level exists in context
// 2. If user has seller role level or higher (>=SELLER_ROLE_LEVEL), they must have a seller ID
// Returns roleLevel, sellerID, and error if validation fails
// Note: Lower role level numbers = higher authority (1=ADMIN, 2=SELLER, 3=CUSTOMER)
func ValidateUserHasSellerRoleOrHigherAndReturnAuthData(
	c *gin.Context,
) (roleLevel uint, sellerID uint, err error) {
	roleLevel, exists := GetUserRoleLevelFromContext(c)
	if !exists {
		return 0, 0, commonError.ErrRoleDataMissing
	}

	sellerID, sellerExists := GetSellerIDFromContext(c)
	// If user has seller role level or higher (numerical value >= SELLER_ROLE_LEVEL)
	// they must have a seller ID in context
	// note: lower role level numbers = higher authority
	if !sellerExists && roleLevel >= constants.SELLER_ROLE_LEVEL {
		return 0, 0, commonError.UnauthorizedError
	}

	return roleLevel, sellerID, nil
}
