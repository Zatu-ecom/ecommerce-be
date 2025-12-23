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
	inventoryReservationService service.InventoryReservationService
}

func NewInventoryReservationHandler(
	inventoryReservationService service.InventoryReservationService,
) *InventoryReservationHandler {
	return &InventoryReservationHandler{
		BaseHandler:                 handler.NewBaseHandler(),
		inventoryReservationService: inventoryReservationService,
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

	reservationResp, err := h.inventoryReservationService.CreateReservation(c, sellerID, req)
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

func (h *InventoryReservationHandler) UpdateReservationStatus(c *gin.Context) {
	var req model.UpdateReservationStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.HandleValidationError(c, err)
		return
	}

	sellerID, exist := auth.GetSellerIDFromContext(c)
	if !exist {
		h.HandleError(c, err.ErrSellerDataMissing, constants.SELLER_DATA_MISSING_MSG)
		return
	}

	err := h.inventoryReservationService.UpdateReservationStatus(c, sellerID, req)
	if err != nil {
		h.HandleError(c, err, reservationConst.FAILED_TO_UPDATE_RESERVATION_STATUS_MSG)
		return
	}

	h.Success(
		c,
		http.StatusOK,
		reservationConst.SUCCESSFUL_RESERVATION_STATUS_UPDATE_MSG,
		nil,
	)
}
