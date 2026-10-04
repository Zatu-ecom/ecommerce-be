package handler

import (
	"net/http"

	"ecommerce-be/common/auth"
	"ecommerce-be/common/constants"
	commonError "ecommerce-be/common/error"
	"ecommerce-be/common/handler"
	commonModel "ecommerce-be/common/model"
	"ecommerce-be/fulfillment/model"
	"ecommerce-be/fulfillment/repository"
	fulfillmentservice "ecommerce-be/fulfillment/service"
	fulfillmentconstant "ecommerce-be/fulfillment/utils/constant"

	"github.com/gin-gonic/gin"
)

// WebhookHandler receives raw courier pushes. No JWT: the signature is the
// auth. No correlation id required (middleware generates one).
type WebhookHandler struct {
	*handler.BaseHandler
	webhooks fulfillmentservice.WebhookService
}

// NewWebhookHandler creates the webhook handler.
func NewWebhookHandler(
	base *handler.BaseHandler,
	webhooks fulfillmentservice.WebhookService,
) *WebhookHandler {
	return &WebhookHandler{BaseHandler: base, webhooks: webhooks}
}

// ReceiveWebhook verifies, dedupes, and applies one push. Unknown provider
// → 404; bad signature → 401 with nothing stored; every verified outcome
// (applied, duplicate, ignored, apply-failed) → 200 with data omitted so
// couriers stop retrying.
func (h *WebhookHandler) ReceiveWebhook(c *gin.Context) {
	code := c.Param(fulfillmentconstant.PARAM_CODE)
	rawBody, err := c.GetRawData()
	if err != nil {
		h.HandleError(c, err, fulfillmentconstant.FAILED_TO_WEBHOOK_MSG)
		return
	}

	ip := c.ClientIP()
	if err := h.webhooks.HandleWebhook(c, code, rawBody, c.Request.Header, ip); err != nil {
		h.HandleError(c, err, fulfillmentconstant.FAILED_TO_WEBHOOK_MSG)
		return
	}
	h.Success(c, http.StatusOK, fulfillmentconstant.WEBHOOK_RECEIVED_MSG, nil)
}

// TrackingHandler serves customer tracking, seller refresh, and the
// seller's webhook audit log.
type TrackingHandler struct {
	*handler.BaseHandler
	tracking fulfillmentservice.TrackingService
}

// NewTrackingHandler creates the tracking handler.
func NewTrackingHandler(
	base *handler.BaseHandler,
	tracking fulfillmentservice.TrackingService,
) *TrackingHandler {
	return &TrackingHandler{BaseHandler: base, tracking: tracking}
}

// CustomerTrack returns a buyer's boxes for one order (PG reads only).
func (h *TrackingHandler) CustomerTrack(c *gin.Context) {
	userID, ok := h.customerID(c)
	if !ok {
		return
	}
	orderID, err := h.ParseUintParam(c, "orderId")
	if err != nil {
		h.HandleValidationError(c, err)
		return
	}

	shipments, err := h.tracking.GetCustomerTrack(c, userID, orderID)
	if err != nil {
		h.HandleError(c, err, fulfillmentconstant.FAILED_TO_TRACK_MSG)
		return
	}
	if shipments == nil {
		shipments = []model.ShipmentResponse{}
	}
	h.Success(c, http.StatusOK, fulfillmentconstant.TRACK_FETCHED_MSG, map[string]any{
		"shipments": shipments,
	})
}

// RefreshTrack pulls live provider state for one box (seller only).
func (h *TrackingHandler) RefreshTrack(c *gin.Context) {
	sellerID, ok := h.sellerID(c)
	if !ok {
		return
	}
	id, err := h.ParseUintParam(c, "id")
	if err != nil {
		h.HandleValidationError(c, err)
		return
	}

	mapped, err := h.tracking.RefreshTrack(c, sellerID, id)
	if err != nil {
		h.HandleError(c, err, fulfillmentconstant.FAILED_TO_REFRESH_MSG)
		return
	}
	h.Success(c, http.StatusOK, fulfillmentconstant.TRACK_FETCHED_MSG, map[string]any{
		"shipment": mapped,
	})
}

// ListWebhookLogs returns verified rows linked to the seller's shipments.
func (h *TrackingHandler) ListWebhookLogs(c *gin.Context) {
	sellerID, ok := h.sellerID(c)
	if !ok {
		return
	}

	var params commonModel.BaseListParams
	if err := c.ShouldBindQuery(&params); err != nil {
		h.HandleValidationError(c, err)
		return
	}
	params.SetDefaults()

	filter := repository.WebhookLogFilter{
		Status: c.Query("status"),
		AWB:    c.Query("awb"),
		Limit:  params.PageSize,
		Offset: (params.Page - 1) * params.PageSize,
	}
	logs, total, err := h.tracking.ListWebhookLogs(c, sellerID, filter)
	if err != nil {
		h.HandleError(c, err, fulfillmentconstant.FAILED_TO_WEBHOOK_LOGS_MSG)
		return
	}
	h.Success(c, http.StatusOK, fulfillmentconstant.WEBHOOK_LOGS_FETCHED_MSG, map[string]any{
		"logs":       logs,
		"pagination": commonModel.NewPaginationResponse(params.Page, params.PageSize, total),
	})
}

// customerID extracts the authenticated customer id.
func (h *TrackingHandler) customerID(c *gin.Context) (uint, bool) {
	userID, exists := auth.GetUserIDFromContext(c)
	if !exists {
		h.HandleError(c, commonError.ErrSellerDataMissing, constants.SELLER_DATA_MISSING_MSG)
		return 0, false
	}
	return userID, true
}

// sellerID extracts the authenticated seller id.
func (h *TrackingHandler) sellerID(c *gin.Context) (uint, bool) {
	sellerID, exists := auth.GetSellerIDFromContext(c)
	if !exists {
		h.HandleError(c, commonError.ErrSellerDataMissing, constants.SELLER_DATA_MISSING_MSG)
		return 0, false
	}
	return sellerID, true
}
