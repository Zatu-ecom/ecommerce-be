package handler

import (
	"io"
	"net/http"

	"ecommerce-be/common/handler"
	"ecommerce-be/common/log"
	"ecommerce-be/common/model"
	paymentService "ecommerce-be/payment/service"
	paymentConstant "ecommerce-be/payment/utils/constant"

	"github.com/gin-gonic/gin"
)

// WebhookHandler receives inbound gateway callbacks.
type WebhookHandler struct {
	*handler.BaseHandler
	webhookService paymentService.WebhookService
}

// NewWebhookHandler creates the webhook handler.
func NewWebhookHandler(base *handler.BaseHandler, svc paymentService.WebhookService) *WebhookHandler {
	return &WebhookHandler{BaseHandler: base, webhookService: svc}
}

// HandleRazorpay processes a Razorpay webhook.
// The signature header is the only authentication; the route is public.
func (h *WebhookHandler) HandleRazorpay(c *gin.Context) {
	rawBody, err := io.ReadAll(c.Request.Body)
	if err != nil {
		h.HandleError(c, err, paymentConstant.FAILED_TO_READ_WEBHOOK_BODY_MSG)
		return
	}

	signature := c.GetHeader(paymentConstant.WEBHOOK_SIGNATURE_HEADER)
	ipAddress := c.ClientIP()

	err = h.webhookService.HandleWebhook(
		c, paymentConstant.GATEWAY_CODE_RAZORPAY, rawBody, signature, ipAddress)
	if err != nil {
		// A verification failure is a client error (401); everything else is logged.
		if isWebhookAuthError(err) {
			model.ErrorWithCode(c, http.StatusUnauthorized, err.Error(), paymentConstant.INVALID_WEBHOOK_SIGNATURE_CODE)
			return
		}
		log.ErrorWithContext(c, "webhook processing failed", err)
		// Idempotent contract: verified-but-unknown events still return 200 so the
		// provider stops retrying. Processing errors are recorded in the webhook log.
		model.SuccessResponse(c, http.StatusOK, paymentConstant.WEBHOOK_RECEIVED_MSG, nil)
		return
	}

	model.SuccessResponse(c, http.StatusOK, paymentConstant.WEBHOOK_RECEIVED_MSG, nil)
}
