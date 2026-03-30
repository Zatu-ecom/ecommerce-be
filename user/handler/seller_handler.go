package handler

import (
	"net/http"

	"ecommerce-be/common/auth"
	commonError "ecommerce-be/common/error"
	"ecommerce-be/common/handler"
	"ecommerce-be/user/model"
	"ecommerce-be/user/service"
	"ecommerce-be/user/utils/constant"

	"github.com/gin-gonic/gin"
)

// SellerHandler handles HTTP requests for seller registration and profile
type SellerHandler struct {
	*handler.BaseHandler
	sellerService        service.SellerService
	sellerProfileService service.SellerProfileService
}

// NewSellerHandler creates a new instance of SellerHandler
func NewSellerHandler(
	sellerService service.SellerService,
	sellerProfileService service.SellerProfileService,
) *SellerHandler {
	return &SellerHandler{
		BaseHandler:          handler.NewBaseHandler(),
		sellerService:        sellerService,
		sellerProfileService: sellerProfileService,
	}
}

// RegisterSeller handles seller registration
// @Summary		Register a new seller
// @Description	Creates a new seller account with user, profile, and optional settings
// @Tags			Seller Registration
// @Accept			json
// @Produce		json
// @Param			request	body		model.SellerRegisterRequest	true	"Seller registration request"
// @Success		201		{object}	model.SellerRegisterResponse
// @Failure		400		{object}	common.ErrorResponse	"Validation error"
// @Failure		409		{object}	common.ErrorResponse	"Email or Tax ID already exists"
// @Failure		500		{object}	common.ErrorResponse	"Internal server error"
// @Router			/api/seller/register [post]
func (h *SellerHandler) RegisterSeller(c *gin.Context) {
	var req model.SellerRegisterRequest

	if err := h.BindJSON(c, &req); err != nil {
		h.HandleValidationError(c, err)
		return
	}

	response, err := h.sellerService.RegisterSeller(c, req)
	if err != nil {
		h.HandleError(c, err, constant.FAILED_TO_REGISTER_USER_MSG)
		return
	}

	h.SuccessWithData(
		c,
		http.StatusCreated,
		constant.REGISTER_SUCCESS_MSG,
		constant.SELLER_FIELD_NAME,
		response,
	)
}

// GetProfile handles getting seller's own profile
// @Summary		Get seller profile
// @Description	Retrieves the authenticated seller's full profile (user, business profile, settings)
// @Tags			Seller Profile
// @Produce		json
// @Security		BearerAuth
// @Success		200		{object}	model.SellerFullProfileResponse
// @Failure		401		{object}	common.ErrorResponse	"Unauthorized"
// @Failure		404		{object}	common.ErrorResponse	"Profile not found"
// @Failure		500		{object}	common.ErrorResponse	"Internal server error"
// @Router			/api/seller/profile [get]
func (h *SellerHandler) GetProfile(c *gin.Context) {
	// Get authenticated user ID from context
	userID, exists := auth.GetUserIDFromContext(c)
	if !exists {
		h.HandleError(c, commonError.UnauthorizedError, constant.FAILED_TO_GET_PROFILE_MSG)
		return
	}

	response, err := h.sellerService.GetProfile(c, userID)
	if err != nil {
		h.HandleError(c, err, constant.FAILED_TO_GET_PROFILE_MSG)
		return
	}

	h.SuccessWithData(
		c,
		http.StatusOK,
		constant.SELLER_PROFILE_FETCHED_MSG,
		constant.SELLER_FIELD_NAME,
		response,
	)
}

// UpdateProfile handles updating seller's business profile
// @Summary		Update seller profile
// @Description	Updates the authenticated seller's business profile
// @Tags			Seller Profile
// @Accept			json
// @Produce		json
// @Security		BearerAuth
// @Param			request	body		model.SellerProfileUpdateRequest	true	"Profile update request"
// @Success		200		{object}	model.SellerProfileResponse
// @Failure		400		{object}	common.ErrorResponse	"Validation error"
// @Failure		401		{object}	common.ErrorResponse	"Unauthorized"
// @Failure		404		{object}	common.ErrorResponse	"Profile not found"
// @Failure		409		{object}	common.ErrorResponse	"Tax ID already exists"
// @Failure		500		{object}	common.ErrorResponse	"Internal server error"
// @Router			/api/seller/profile [put]
func (h *SellerHandler) UpdateProfile(c *gin.Context) {
	// Get authenticated user ID from context
	userID, exists := auth.GetUserIDFromContext(c)
	if !exists {
		h.HandleError(c, commonError.UnauthorizedError, constant.FAILED_TO_UPDATE_PROFILE_MSG)
		return
	}

	var req model.SellerProfileUpdateRequest
	if err := h.BindJSON(c, &req); err != nil {
		h.HandleValidationError(c, err)
		return
	}

	response, err := h.sellerProfileService.UpdateProfile(c, userID, req)
	if err != nil {
		h.HandleError(c, err, constant.FAILED_TO_UPDATE_PROFILE_MSG)
		return
	}

	h.SuccessWithData(
		c,
		http.StatusOK,
		constant.SELLER_PROFILE_UPDATED_MSG,
		constant.PROFILE_FIELD_NAME,
		response,
	)
}
