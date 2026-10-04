package handler

import (
	"net/http"

	"ecommerce-be/common/auth"
	"ecommerce-be/common/constants"
	commonError "ecommerce-be/common/error"
	"ecommerce-be/common/handler"
	"ecommerce-be/common/log"
	fulfillmentmodel "ecommerce-be/fulfillment/model"
	fulfillmentservice "ecommerce-be/fulfillment/service"
	fulfillmentconstant "ecommerce-be/fulfillment/utils/constant"

	"github.com/gin-gonic/gin"
)

// ProviderHandler exposes the seller-facing courier dashboard: catalog,
// detail, configure, and credential probing. Thin by rule: parse, extract
// seller, call the service, map the response.
type ProviderHandler struct {
	*handler.BaseHandler
	configService fulfillmentservice.ProviderConfigService
}

// NewProviderHandler creates the dashboard handler.
func NewProviderHandler(
	base *handler.BaseHandler,
	svc fulfillmentservice.ProviderConfigService,
) *ProviderHandler {
	return &ProviderHandler{BaseHandler: base, configService: svc}
}

// sellerID extracts the authenticated seller id, writing an error on failure.
func (h *ProviderHandler) sellerID(c *gin.Context) (uint, bool) {
	sellerID, exists := auth.GetSellerIDFromContext(c)
	if !exists {
		h.HandleError(c, commonError.ErrSellerDataMissing, constants.SELLER_DATA_MISSING_MSG)
		return 0, false
	}
	return sellerID, true
}

// ListCouriers lists active providers with the seller's configured state.
func (h *ProviderHandler) ListCouriers(c *gin.Context) {
	sellerID, ok := h.sellerID(c)
	if !ok {
		return
	}

	resp, err := h.configService.ListForSeller(c, sellerID)
	if err != nil {
		log.ErrorWithContext(c, "listCouriers: failed", err)
		h.HandleError(c, err, fulfillmentconstant.FAILED_TO_LIST_MSG)
		return
	}
	h.Success(c, http.StatusOK, fulfillmentconstant.COURIERS_FETCHED_MSG, map[string]any{
		"couriers": resp,
	})
}

// GetCourier returns the credential form plus the saved (masked) config.
func (h *ProviderHandler) GetCourier(c *gin.Context) {
	sellerID, ok := h.sellerID(c)
	if !ok {
		return
	}

	resp, err := h.configService.GetByCode(c, sellerID, c.Param(fulfillmentconstant.PARAM_CODE))
	if err != nil {
		h.HandleError(c, err, fulfillmentconstant.FAILED_TO_GET_MSG)
		return
	}
	h.Success(c, http.StatusOK, fulfillmentconstant.COURIER_FETCHED_MSG, resp)
}

// ConfigureCourier saves or updates the seller's credentials for one
// provider and environment, including tenant automation flags.
func (h *ProviderHandler) ConfigureCourier(c *gin.Context) {
	sellerID, ok := h.sellerID(c)
	if !ok {
		return
	}

	var req fulfillmentmodel.ConfigureCourierRequest
	if err := h.BindJSON(c, &req); err != nil {
		h.HandleValidationError(c, err)
		return
	}

	resp, err := h.configService.Configure(c, sellerID, c.Param(fulfillmentconstant.PARAM_CODE), req)
	if err != nil {
		h.HandleError(c, err, fulfillmentconstant.FAILED_TO_CONFIGURE_MSG)
		return
	}
	h.Success(c, http.StatusOK, fulfillmentconstant.COURIER_CONFIGURED_MSG, resp)
}

// TestCourier probes credentials against the provider without persisting.
func (h *ProviderHandler) TestCourier(c *gin.Context) {
	sellerID, ok := h.sellerID(c)
	if !ok {
		return
	}

	var req fulfillmentmodel.TestCourierRequest
	if err := h.BindJSON(c, &req); err != nil {
		h.HandleValidationError(c, err)
		return
	}

	resp, err := h.configService.TestConnection(c, sellerID, c.Param(fulfillmentconstant.PARAM_CODE), req)
	if err != nil {
		h.HandleError(c, err, fulfillmentconstant.FAILED_TO_TEST_MSG)
		return
	}
	h.Success(c, http.StatusOK, fulfillmentconstant.COURIER_TESTED_MSG, resp)
}
