package utils

import (
	"ecommerce-be/common/auth"
	"ecommerce-be/common/constants"
	commonError "ecommerce-be/common/error"
	"ecommerce-be/file/entity"
	fileError "ecommerce-be/file/error"

	"github.com/gin-gonic/gin"
)

// Principal represents the authenticated caller (Seller or Admin/Platform).
type Principal struct {
	Role      string
	UserID    uint64
	SellerID  *uint64
	OwnerType entity.OwnerType // entity.OwnerTypeSeller or entity.OwnerTypePlatform
}

// ExtractPrincipal builds a Principal from the Gin context using common/auth helpers.
// Returns ErrFileUploadForbidden if the role is missing or unrecognised.
func ExtractPrincipal(c *gin.Context) (Principal, *commonError.AppError) {
	var p Principal

	userID, exists := auth.GetUserIDFromContext(c)
	if !exists {
		return p, fileError.ErrFileUploadUnauthorized
	}
	p.UserID = uint64(userID)

	_, roleName, exists := auth.GetUserRoleFromContext(c)
	if !exists {
		return p, fileError.ErrFileUploadForbidden
	}
	p.Role = roleName

	switch roleName {
	case constants.SELLER_ROLE_NAME:
		p.OwnerType = entity.OwnerTypeSeller
		// Fallback to GetSellerIDFromContext if present
		sellerID, exists := auth.GetSellerIDFromContext(c)
		if !exists {
			return p, fileError.ErrFileUploadForbidden
		}
		sid := uint64(sellerID)
		p.SellerID = &sid
	case constants.ADMIN_ROLE_NAME:
		p.OwnerType = entity.OwnerTypePlatform
	default:
		// Customer or unrecognised roles are forbidden for upload.
		return p, fileError.ErrFileUploadForbidden
	}

	return p, nil
}
