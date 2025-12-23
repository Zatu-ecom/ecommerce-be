package auth

import (
	"context"
	"strconv"

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

// getValueFromContext is a helper that extracts a value from context
// Works with both *gin.Context and standard context.Context
func getValueFromContext(ctx context.Context, key string) (interface{}, bool) {
	// Try Gin context first (uses internal map storage)
	if ginCtx, ok := ctx.(*gin.Context); ok {
		return ginCtx.Get(key)
	}

	// Fallback to standard context.Context
	val := ctx.Value(key)
	if val != nil {
		return val, true
	}

	return nil, false
}

// getUintFromContext extracts a uint value from context
// Handles both uint and string types (string for scheduler job context)
func getUintFromContext(ctx context.Context, key string) (uint, bool) {
	val, exists := getValueFromContext(ctx, key)
	if !exists {
		return 0, false
	}

	switch v := val.(type) {
	case uint:
		return v, true
	case string:
		// Handle string values (from scheduler jobs)
		if parsed, err := strconv.ParseUint(v, 10, 32); err == nil {
			return uint(parsed), true
		}
	case int:
		return uint(v), true
	case int64:
		return uint(v), true
	case uint64:
		return uint(v), true
	}

	return 0, false
}

// getStringFromContext extracts a string value from context
func getStringFromContext(ctx context.Context, key string) (string, bool) {
	val, exists := getValueFromContext(ctx, key)
	if !exists {
		return "", false
	}

	if str, ok := val.(string); ok {
		return str, true
	}

	return "", false
}

// GetUserRoleFromContext extracts user role information from context
// Works with both *gin.Context and context.Context
func GetUserRoleFromContext(ctx context.Context) (roleLevel uint, roleName string, exists bool) {
	roleLevel, exists = getUintFromContext(ctx, constants.ROLE_LEVEL_KEY)
	if !exists {
		return 0, "", false
	}
	roleName, exists = getStringFromContext(ctx, constants.ROLE_NAME_KEY)
	if !exists {
		return 0, "", false
	}
	return roleLevel, roleName, true
}

// GetSellerIDFromContext extracts seller ID from context
// Works with both *gin.Context and context.Context
func GetSellerIDFromContext(ctx context.Context) (sellerID uint, exists bool) {
	return getUintFromContext(ctx, constants.SELLER_ID_KEY)
}

// GetUserRoleLevelFromContext extracts user role level from context
// Works with both *gin.Context and context.Context
func GetUserRoleLevelFromContext(ctx context.Context) (roleLevel uint, exists bool) {
	return getUintFromContext(ctx, constants.ROLE_LEVEL_KEY)
}

// GetUserIDFromContext extracts user ID from context
// Works with both *gin.Context and context.Context
func GetUserIDFromContext(ctx context.Context) (userID uint, exists bool) {
	return getUintFromContext(ctx, constants.USER_ID_KEY)
}

// GetCorrelationIDFromContext extracts correlation ID from context
// Works with both *gin.Context and context.Context
func GetCorrelationIDFromContext(ctx context.Context) (correlationID string, exists bool) {
	return getStringFromContext(ctx, constants.CORRELATION_ID_KEY)
}

// ValidateUserHasSellerRoleOrHigherAndReturnAuthData validates that:
// 1. Role level exists in context
// 2. If user has seller role level or higher (>=SELLER_ROLE_LEVEL), they must have a seller ID
// Returns roleLevel, sellerID, and error if validation fails
// Note: Lower role level numbers = higher authority (1=ADMIN, 2=SELLER, 3=CUSTOMER)
// Works with both *gin.Context and context.Context
func ValidateUserHasSellerRoleOrHigherAndReturnAuthData(
	ctx context.Context,
) (roleLevel uint, sellerID uint, err error) {
	roleLevel, exists := GetUserRoleLevelFromContext(ctx)
	if !exists {
		return 0, 0, commonError.ErrRoleDataMissing
	}

	sellerID, sellerExists := GetSellerIDFromContext(ctx)
	// If user has seller role level or higher (numerical value >= SELLER_ROLE_LEVEL)
	// they must have a seller ID in context
	// note: lower role level numbers = higher authority
	if !sellerExists && roleLevel >= constants.SELLER_ROLE_LEVEL {
		return 0, 0, commonError.UnauthorizedError
	}

	return roleLevel, sellerID, nil
}
