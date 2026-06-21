package handler

import (
	"net/http"

	"ecommerce-be/common/auth"
	"ecommerce-be/common/constants"
	"ecommerce-be/common/handler"
	"ecommerce-be/product/model"
	"ecommerce-be/product/service"
	"ecommerce-be/product/utils"

	"github.com/gin-gonic/gin"
)

// PackageOptionHandler handles HTTP requests related to package options
type PackageOptionHandler struct {
	*handler.BaseHandler
	packageOptionService service.PackageOptionService
}

// NewPackageOptionHandler creates a new instance of PackageOptionHandler
func NewPackageOptionHandler(
	packageOptionService service.PackageOptionService,
) *PackageOptionHandler {
	return &PackageOptionHandler{
		BaseHandler:          handler.NewBaseHandler(),
		packageOptionService: packageOptionService,
	}
}

// AddPackageOption handles adding a package option to a product
// POST /api/product/:productId/package-option
func (h *PackageOptionHandler) AddPackageOption(c *gin.Context) {
	productID, err := h.ParseUintParam(c, "productId")
	if err != nil {
		h.HandleError(c, err, utils.INVALID_PRODUCT_ID_MSG)
		return
	}

	_, sellerID, err := auth.ValidateUserHasSellerRoleOrHigherAndReturnAuthData(c)
	if err != nil {
		h.HandleError(c, err, constants.UNAUTHORIZED_ERROR_MSG)
		return
	}

	var req model.PackageOptionCreateRequest
	if err := h.BindJSON(c, &req); err != nil {
		h.HandleValidationError(c, err)
		return
	}

	packageOptionResponse, err := h.packageOptionService.AddPackageOption(
		c,
		productID,
		sellerID,
		req,
	)
	if err != nil {
		h.HandleError(c, err, utils.FAILED_TO_ADD_PACKAGE_OPTION_MSG)
		return
	}

	h.SuccessWithData(
		c,
		http.StatusCreated,
		utils.PACKAGE_OPTION_ADDED_MSG,
		utils.PACKAGE_OPTION_FIELD_NAME,
		packageOptionResponse,
	)
}

// UpdatePackageOption handles updating a package option
// PUT /api/product/:productId/package-option/:packageOptionId
func (h *PackageOptionHandler) UpdatePackageOption(c *gin.Context) {
	productID, err := h.ParseUintParam(c, "productId")
	if err != nil {
		h.HandleError(c, err, utils.INVALID_PRODUCT_ID_MSG)
		return
	}

	packageOptionID, err := h.ParseUintParam(c, "packageOptionId")
	if err != nil {
		h.HandleError(c, err, "Invalid package option ID")
		return
	}

	_, sellerID, err := auth.ValidateUserHasSellerRoleOrHigherAndReturnAuthData(c)
	if err != nil {
		h.HandleError(c, err, constants.UNAUTHORIZED_ERROR_MSG)
		return
	}

	var req model.PackageOptionUpdateRequest
	if err := h.BindJSON(c, &req); err != nil {
		h.HandleValidationError(c, err)
		return
	}

	packageOptionResponse, err := h.packageOptionService.UpdatePackageOption(
		c,
		productID,
		packageOptionID,
		sellerID,
		req,
	)
	if err != nil {
		h.HandleError(c, err, utils.FAILED_TO_UPDATE_PACKAGE_OPTION_MSG)
		return
	}

	h.SuccessWithData(
		c,
		http.StatusOK,
		utils.PACKAGE_OPTION_UPDATED_MSG,
		utils.PACKAGE_OPTION_FIELD_NAME,
		packageOptionResponse,
	)
}

// DeletePackageOption handles deleting a package option
// DELETE /api/product/:productId/package-option/:packageOptionId
func (h *PackageOptionHandler) DeletePackageOption(c *gin.Context) {
	productID, err := h.ParseUintParam(c, "productId")
	if err != nil {
		h.HandleError(c, err, utils.INVALID_PRODUCT_ID_MSG)
		return
	}

	packageOptionID, err := h.ParseUintParam(c, "packageOptionId")
	if err != nil {
		h.HandleError(c, err, "Invalid package option ID")
		return
	}

	_, sellerID, err := auth.ValidateUserHasSellerRoleOrHigherAndReturnAuthData(c)
	if err != nil {
		h.HandleError(c, err, constants.UNAUTHORIZED_ERROR_MSG)
		return
	}

	err = h.packageOptionService.DeletePackageOption(
		c,
		productID,
		packageOptionID,
		sellerID,
	)
	if err != nil {
		h.HandleError(c, err, utils.FAILED_TO_DELETE_PACKAGE_OPTION_MSG)
		return
	}

	h.Success(c, http.StatusOK, utils.PACKAGE_OPTION_DELETED_MSG, nil)
}

// GetPackageOptions handles retrieving all package options for a product
// GET /api/product/:productId/package-option
func (h *PackageOptionHandler) GetPackageOptions(c *gin.Context) {
	productID, err := h.ParseUintParam(c, "productId")
	if err != nil {
		h.HandleError(c, err, utils.INVALID_PRODUCT_ID_MSG)
		return
	}

	packageOptionsResponse, err := h.packageOptionService.GetPackageOptions(c, productID)
	if err != nil {
		h.HandleError(c, err, utils.FAILED_TO_GET_PACKAGE_OPTIONS_MSG)
		return
	}

	h.SuccessWithData(
		c,
		http.StatusOK,
		utils.PACKAGE_OPTIONS_RETRIEVED_MSG,
		utils.PACKAGE_OPTIONS_FIELD_NAME,
		packageOptionsResponse,
	)
}

// BulkUpdatePackageOptions handles bulk updating package options for a product
// PUT /api/product/:productId/package-option/bulk
func (h *PackageOptionHandler) BulkUpdatePackageOptions(c *gin.Context) {
	productID, err := h.ParseUintParam(c, "productId")
	if err != nil {
		h.HandleError(c, err, utils.INVALID_PRODUCT_ID_MSG)
		return
	}

	_, sellerID, err := auth.ValidateUserHasSellerRoleOrHigherAndReturnAuthData(c)
	if err != nil {
		h.HandleError(c, err, constants.UNAUTHORIZED_ERROR_MSG)
		return
	}

	var req model.BulkUpdatePackageOptionsRequest
	if err := h.BindJSON(c, &req); err != nil {
		h.HandleValidationError(c, err)
		return
	}

	updateResponse, err := h.packageOptionService.BulkUpdatePackageOptions(
		c,
		productID,
		sellerID,
		req,
	)
	if err != nil {
		h.HandleError(c, err, utils.FAILED_TO_BULK_UPDATE_PACKAGE_OPTIONS_MSG)
		return
	}

	h.SuccessWithData(
		c,
		http.StatusOK,
		utils.PACKAGE_OPTIONS_BULK_UPDATED_MSG,
		"result",
		updateResponse,
	)
}
