package handler

import (
	"net/http"

	"ecommerce-be/common/auth"
	"ecommerce-be/common/constants"
	err "ecommerce-be/common/error"
	"ecommerce-be/common/handler"
	"ecommerce-be/inventory/model"
	"ecommerce-be/inventory/service"
	reservationConst "ecommerce-be/inventory/utils/constant"

	"github.com/gin-gonic/gin"
)

type InventoryReservationHandler struct {
	*handler.BaseHandler
	InventoryReservationHandler service.InventoryReservationService
}

func NewInventoryReservationHandler(
	inventoryReservationService service.InventoryReservationService,
) *InventoryReservationHandler {
	return &InventoryReservationHandler{
		BaseHandler:                 handler.NewBaseHandler(),
		InventoryReservationHandler: inventoryReservationService,
	}
}

func (h *InventoryReservationHandler) CreateReservation(c *gin.Context) {
	var req model.ReservationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.HandleValidationError(c, err)
		return
	}

	sellerID, exist := auth.GetSellerIDFromContext(c)
	if !exist {
		h.HandleError(c, err.ErrSellerDataMissing, constants.SELLER_DATA_MISSING_MSG)
		return
	}

	reservationResp, err := h.InventoryReservationHandler.CreateReservation(c, sellerID, req)
	if err != nil {
		h.HandleError(c, err, reservationConst.FAILED_TO_CREATE_RESERVATION_MSG)
		return
	}

	h.Success(
		c,
		http.StatusOK,
		reservationConst.SUCCESSFUL_RESERVATION_CREATION_MSG,
		reservationResp,
	)
}
