package handler

import (
	"net/http"

	"ecommerce-be/common/auth"
	"ecommerce-be/common/constants"
	commonError "ecommerce-be/common/error"
	"ecommerce-be/common/handler"
	"ecommerce-be/common/log"
	"ecommerce-be/payment/entity"
	paymenterrors "ecommerce-be/payment/error"
	paymentModel "ecommerce-be/payment/model"
	paymentService "ecommerce-be/payment/service"
	paymentConstant "ecommerce-be/payment/utils/constant"

	"github.com/gin-gonic/gin"
)

// GatewayHandler exposes seller-facing gateway catalog endpoints.
type GatewayHandler struct {
	*handler.BaseHandler
	gatewayService paymentService.PaymentGatewayService
}

// NewGatewayHandler creates the gateway handler.
func NewGatewayHandler(
	base *handler.BaseHandler,
	svc paymentService.PaymentGatewayService,
) *GatewayHandler {
	return &GatewayHandler{BaseHandler: base, gatewayService: svc}
}

// sellerID extracts the authenticated seller id, writing an error on failure.
func (h *GatewayHandler) sellerID(c *gin.Context) (uint, bool) {
	sellerID, exists := auth.GetSellerIDFromContext(c)
	if !exists {
		h.HandleError(c, commonError.ErrSellerDataMissing, constants.SELLER_DATA_MISSING_MSG)
		return 0, false
	}
	return sellerID, true
}

// ListGateways lists all active providers with the seller's configured state.
func (h *GatewayHandler) ListGateways(c *gin.Context) {
	sellerID, ok := h.sellerID(c)
	if !ok {
		return
	}

	resp, err := h.gatewayService.ListForSeller(c, sellerID)
	if err != nil {
		log.ErrorWithContext(c, "listGateways: failed", err)
		h.HandleError(c, err, paymentConstant.FAILED_TO_LIST_GATEWAYS_MSG)
		return
	}
	h.Success(c, http.StatusOK, paymentConstant.GATEWAYS_FETCHED_MSG, resp)
}

// GetGateway returns full detail for one provider.
func (h *GatewayHandler) GetGateway(c *gin.Context) {
	sellerID, ok := h.sellerID(c)
	if !ok {
		return
	}

	code := c.Param(paymentConstant.PARAM_CODE)
	resp, err := h.gatewayService.GetByCode(c, sellerID, code)
	if err != nil {
		h.HandleError(c, err, paymentConstant.FAILED_TO_GET_GATEWAY_MSG)
		return
	}
	h.Success(c, http.StatusOK, paymentConstant.GATEWAY_FETCHED_MSG, resp)
}

// ConfigureGateway upserts the seller's credentials for a provider.
func (h *GatewayHandler) ConfigureGateway(c *gin.Context) {
	sellerID, ok := h.sellerID(c)
	if !ok {
		return
	}

	code := c.Param(paymentConstant.PARAM_CODE)
	var req paymentModel.ConfigureGatewayRequest
	if err := h.BindJSON(c, &req); err != nil {
		h.HandleValidationError(c, err)
		return
	}

	if err := h.gatewayService.Configure(c, sellerID, code, req); err != nil {
		h.HandleError(c, err, paymentConstant.FAILED_TO_CONFIGURE_GATEWAY_MSG)
		return
	}
	h.Success(c, http.StatusOK, paymentConstant.GATEWAY_CONFIGURED_MSG, nil)
}

// DeactivateGateway disables one environment of the seller's config.
// Query ?environment= is required: deactivating both environments by accident
// must be impossible.
func (h *GatewayHandler) DeactivateGateway(c *gin.Context) {
	sellerID, ok := h.sellerID(c)
	if !ok {
		return
	}

	code := c.Param(paymentConstant.PARAM_CODE)
	environment := entity.GatewayEnvironment(c.Query(paymentConstant.QUERY_ENVIRONMENT))
	if environment != entity.EnvironmentSandbox && environment != entity.EnvironmentProduction {
		h.HandleError(c, paymenterrors.ErrorGatewayValidation.WithMessagef(
			"query environment is required (sandbox|production)"), paymentConstant.FAILED_TO_DEACTIVATE_GATEWAY_MSG)
		return
	}
	if err := h.gatewayService.Deactivate(c, sellerID, code, environment); err != nil {
		h.HandleError(c, err, paymentConstant.FAILED_TO_DEACTIVATE_GATEWAY_MSG)
		return
	}
	h.Success(c, http.StatusOK, paymentConstant.GATEWAY_DEACTIVATED_MSG, nil)
}

// TestGateway probes credentials against the provider without persisting.
// The handler binds JSON and delegates; all provider logic lives in the
// service/adapter layers.
func (h *GatewayHandler) TestGateway(c *gin.Context) {
	sellerID, ok := h.sellerID(c)
	if !ok {
		return
	}

	code := c.Param(paymentConstant.PARAM_CODE)
	var req paymentModel.TestGatewayRequest
	if err := h.BindJSON(c, &req); err != nil {
		h.HandleValidationError(c, err)
		return
	}

	resp, err := h.gatewayService.TestConnection(c, sellerID, code, req)
	if err != nil {
		h.HandleError(c, err, paymentConstant.FAILED_TO_TEST_GATEWAY_MSG)
		return
	}
	h.Success(c, http.StatusOK, paymentConstant.GATEWAY_TESTED_MSG, resp)
}
