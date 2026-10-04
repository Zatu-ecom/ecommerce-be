package handler

import (
	"net/http"
	"strconv"
	"time"

	"ecommerce-be/common/auth"
	"ecommerce-be/common/constants"
	commonError "ecommerce-be/common/error"
	"ecommerce-be/common/handler"
	commonModel "ecommerce-be/common/model"
	"ecommerce-be/fulfillment/entity"
	fulfillmenterrors "ecommerce-be/fulfillment/error"
	"ecommerce-be/fulfillment/model"
	"ecommerce-be/fulfillment/repository"
	fulfillmentservice "ecommerce-be/fulfillment/service"
	fulfillmentconstant "ecommerce-be/fulfillment/utils/constant"

	"github.com/gin-gonic/gin"
)

// ShipmentHandler exposes draft planning and box reads. Thin by rule:
// parse, extract seller, call services, map responses.
type ShipmentHandler struct {
	*handler.BaseHandler
	planner   fulfillmentservice.ShipmentPlanner
	shipments fulfillmentservice.ShipmentService
	rates     fulfillmentservice.RateService
}

// NewShipmentHandler creates the shipment handler.
func NewShipmentHandler(
	base *handler.BaseHandler,
	planner fulfillmentservice.ShipmentPlanner,
	shipments fulfillmentservice.ShipmentService,
	rates fulfillmentservice.RateService,
) *ShipmentHandler {
	return &ShipmentHandler{BaseHandler: base, planner: planner, shipments: shipments, rates: rates}
}

// sellerID extracts the authenticated seller id, writing an error on failure.
func (h *ShipmentHandler) sellerID(c *gin.Context) (uint, bool) {
	sellerID, exists := auth.GetSellerIDFromContext(c)
	if !exists {
		h.HandleError(c, commonError.ErrSellerDataMissing, constants.SELLER_DATA_MISSING_MSG)
		return 0, false
	}
	return sellerID, true
}

// PlanOrder runs the planner for an order: 201 with new drafts, 200 with
// the live set when already covered.
func (h *ShipmentHandler) PlanOrder(c *gin.Context) {
	sellerID, ok := h.sellerID(c)
	if !ok {
		return
	}
	orderID, err := h.ParseUintParam(c, "orderId")
	if err != nil {
		h.HandleValidationError(c, err)
		return
	}

	result, err := h.planner.PlanForOrder(c, sellerID, orderID)
	if err != nil {
		h.HandleError(c, err, fulfillmentconstant.FAILED_TO_PLAN_MSG)
		return
	}
	mapped, err := h.shipmentsForResult(c, sellerID, result.ShipmentIDs)
	if err != nil {
		h.HandleError(c, err, fulfillmentconstant.FAILED_TO_PLAN_MSG)
		return
	}
	data := map[string]any{"shipments": mapped}
	if result.CreatedNew {
		h.Success(c, http.StatusCreated, fulfillmentconstant.SHIPMENTS_PLANNED_MSG, data)
		return
	}
	h.Success(c, http.StatusOK, fulfillmentconstant.SHIPMENTS_PLANNED_MSG, data)
}

// CreateShipment inserts a manual draft. Idempotency-Key replays return the
// original shipment with 200 instead of a second row.
func (h *ShipmentHandler) CreateShipment(c *gin.Context) {
	sellerID, ok := h.sellerID(c)
	if !ok {
		return
	}

	var req model.CreateShipmentRequest
	if err := h.BindJSON(c, &req); err != nil {
		h.HandleValidationError(c, err)
		return
	}

	created, replayed, err := h.shipments.CreateDraft(c, sellerID, req, c.GetHeader("Idempotency-Key"))
	if err != nil {
		h.HandleError(c, err, fulfillmentconstant.FAILED_TO_CREATE_MSG)
		return
	}
	mapped, err := h.shipments.MapShipment(c, created)
	if err != nil {
		h.HandleError(c, err, fulfillmentconstant.FAILED_TO_CREATE_MSG)
		return
	}
	data := map[string]any{"shipment": mapped}
	if replayed {
		h.Success(c, http.StatusOK, fulfillmentconstant.SHIPMENT_FETCHED_MSG, data)
		return
	}
	h.Success(c, http.StatusCreated, fulfillmentconstant.SHIPMENT_CREATED_MSG, data)
}

// ListShipments returns the seller's boxes with filters and pagination.
func (h *ShipmentHandler) ListShipments(c *gin.Context) {
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

	filter := repository.ShipmentFilter{
		SortBy:    params.SortBy,
		SortOrder: params.SortOrder,
		Limit:     params.PageSize,
		Offset:    (params.Page - 1) * params.PageSize,
	}
	if status := c.Query("status"); status != "" {
		filter.Status = status
	}
	if orderID := c.Query("orderId"); orderID != "" {
		id, err := strconv.ParseUint(orderID, 10, 64)
		if err != nil {
			h.HandleValidationError(c, err)
			return
		}
		filter.OrderID = uint(id)
	}
	if pickup := c.Query("pickupLocationId"); pickup != "" {
		id, err := strconv.ParseUint(pickup, 10, 64)
		if err != nil {
			h.HandleValidationError(c, err)
			return
		}
		filter.PickupLocationID = uint(id)
	}

	shipments, total, err := h.shipments.ListShipments(c, sellerID, filter)
	if err != nil {
		h.HandleError(c, err, fulfillmentconstant.FAILED_TO_LIST_SHIP_MSG)
		return
	}
	mapped, err := h.shipments.MapShipments(c, shipments)
	if err != nil {
		h.HandleError(c, err, fulfillmentconstant.FAILED_TO_LIST_SHIP_MSG)
		return
	}
	h.Success(c, http.StatusOK, fulfillmentconstant.SHIPMENTS_FETCHED_MSG, map[string]any{
		"shipments":  mapped,
		"pagination": commonModel.NewPaginationResponse(params.Page, params.PageSize, total),
	})
}

// GetShipment returns one box with items, events, and the open NDR round.
func (h *ShipmentHandler) GetShipment(c *gin.Context) {
	sellerID, ok := h.sellerID(c)
	if !ok {
		return
	}
	id, err := h.ParseUintParam(c, "id")
	if err != nil {
		h.HandleValidationError(c, err)
		return
	}

	shipment, err := h.shipments.GetShipment(c, sellerID, id)
	if err != nil {
		h.HandleError(c, err, fulfillmentconstant.FAILED_TO_GET_SHIP_MSG)
		return
	}
	mapped, err := h.shipments.MapShipment(c, shipment)
	if err != nil {
		h.HandleError(c, err, fulfillmentconstant.FAILED_TO_GET_SHIP_MSG)
		return
	}
	events, err := h.shipments.ShipmentEvents(c, sellerID, id)
	if err != nil {
		h.HandleError(c, err, fulfillmentconstant.FAILED_TO_GET_SHIP_MSG)
		return
	}
	mapped.Events = make([]model.ShipmentEventResponse, 0, len(events))
	for _, event := range events {
		mapped.Events = append(mapped.Events, model.ToShipmentEventResponse(event))
	}
	ndr, err := h.shipments.ShipmentNDR(c, sellerID, id)
	if err != nil {
		h.HandleError(c, err, fulfillmentconstant.FAILED_TO_GET_SHIP_MSG)
		return
	}
	mapped.NDR = model.ToNDRRoundResponse(ndr)
	h.Success(c, http.StatusOK, fulfillmentconstant.SHIPMENT_FETCHED_MSG, map[string]any{
		"shipment": mapped,
	})
}

// UpdateShipment edits draft measurements. Non-drafts refuse.
func (h *ShipmentHandler) UpdateShipment(c *gin.Context) {
	sellerID, ok := h.sellerID(c)
	if !ok {
		return
	}
	id, err := h.ParseUintParam(c, "id")
	if err != nil {
		h.HandleValidationError(c, err)
		return
	}

	var req model.UpdateShipmentRequest
	if err := h.BindJSON(c, &req); err != nil {
		h.HandleValidationError(c, err)
		return
	}

	updated, err := h.shipments.UpdateDraft(c, sellerID, id, req)
	if err != nil {
		h.HandleError(c, err, fulfillmentconstant.FAILED_TO_UPDATE_SHIP_MSG)
		return
	}
	mapped, err := h.shipments.MapShipment(c, updated)
	if err != nil {
		h.HandleError(c, err, fulfillmentconstant.FAILED_TO_UPDATE_SHIP_MSG)
		return
	}
	h.Success(c, http.StatusOK, fulfillmentconstant.SHIPMENT_UPDATED_MSG, map[string]any{
		"shipment": mapped,
	})
}

// BookShipment books one draft: 200 with the booked box. Provider,
// courier preference, and pickup scheduling come from the body.
func (h *ShipmentHandler) BookShipment(c *gin.Context) {
	sellerID, ok := h.sellerID(c)
	if !ok {
		return
	}
	id, err := h.ParseUintParam(c, "id")
	if err != nil {
		h.HandleValidationError(c, err)
		return
	}

	var req model.BookShipmentRequest
	if err := h.BindJSON(c, &req); err != nil {
		h.HandleValidationError(c, err)
		return
	}
	opts, err := bookOptions(req)
	if err != nil {
		h.HandleValidationError(c, err)
		return
	}

	booked, err := h.shipments.BookDraft(c, sellerID, id, opts)
	if err != nil {
		h.HandleError(c, err, fulfillmentconstant.FAILED_TO_BOOK_MSG)
		return
	}
	mapped, err := h.shipments.MapShipment(c, booked)
	if err != nil {
		h.HandleError(c, err, fulfillmentconstant.FAILED_TO_BOOK_MSG)
		return
	}
	h.Success(c, http.StatusOK, fulfillmentconstant.SHIPMENT_BOOKED_MSG, map[string]any{
		"shipment": mapped,
	})
}

// BookAllShipments books every remaining draft on the order with one shared
// provider body. 200 when at least one box booked (plus per-draft
// failures); 400 when every draft failed.
func (h *ShipmentHandler) BookAllShipments(c *gin.Context) {
	sellerID, ok := h.sellerID(c)
	if !ok {
		return
	}
	orderID, err := h.ParseUintParam(c, "orderId")
	if err != nil {
		h.HandleValidationError(c, err)
		return
	}

	var req model.BookShipmentRequest
	if err := h.BindJSON(c, &req); err != nil {
		h.HandleValidationError(c, err)
		return
	}
	opts, err := bookOptions(req)
	if err != nil {
		h.HandleValidationError(c, err)
		return
	}

	shipments, failures, err := h.shipments.BookAllShipments(c, sellerID, orderID, opts)
	if err != nil {
		h.HandleError(c, err, fulfillmentconstant.FAILED_TO_BOOK_MSG)
		return
	}
	mapped, err := h.shipments.MapShipments(c, derefShipments(shipments))
	if err != nil {
		h.HandleError(c, err, fulfillmentconstant.FAILED_TO_BOOK_MSG)
		return
	}
	h.Success(c, http.StatusOK, fulfillmentconstant.SHIPMENTS_BOOKED_MSG, map[string]any{
		"shipments": mapped,
		"failures":  failures,
	})
}

// SchedulePickup schedules pickup separately from booking.
func (h *ShipmentHandler) SchedulePickup(c *gin.Context) {
	sellerID, ok := h.sellerID(c)
	if !ok {
		return
	}
	id, err := h.ParseUintParam(c, "id")
	if err != nil {
		h.HandleValidationError(c, err)
		return
	}

	var req model.PickupRequest
	if err := h.BindJSON(c, &req); err != nil {
		h.HandleValidationError(c, err)
		return
	}
	var pickupAt *time.Time
	if req.PickupAt != nil && *req.PickupAt != "" {
		parsed, err := time.Parse(time.RFC3339, *req.PickupAt)
		if err != nil {
			h.HandleValidationError(c, err)
			return
		}
		pickupAt = &parsed
	}

	scheduled, err := h.shipments.SchedulePickup(c, sellerID, id, pickupAt)
	if err != nil {
		h.HandleError(c, err, fulfillmentconstant.FAILED_TO_PICKUP_MSG)
		return
	}
	mapped, err := h.shipments.MapShipment(c, scheduled)
	if err != nil {
		h.HandleError(c, err, fulfillmentconstant.FAILED_TO_PICKUP_MSG)
		return
	}
	h.Success(c, http.StatusOK, fulfillmentconstant.PICKUP_SCHEDULED_MSG, map[string]any{
		"shipment": mapped,
	})
}

// CancelShipment cancels a box: local-only for drafts, via the courier for
// booked boxes. Provider refusal leaves everything unchanged (409).
func (h *ShipmentHandler) CancelShipment(c *gin.Context) {
	sellerID, ok := h.sellerID(c)
	if !ok {
		return
	}
	id, err := h.ParseUintParam(c, "id")
	if err != nil {
		h.HandleValidationError(c, err)
		return
	}

	cancelled, err := h.shipments.CancelShipment(c, sellerID, id)
	if err != nil {
		h.HandleError(c, err, fulfillmentconstant.FAILED_TO_CANCEL_MSG)
		return
	}
	mapped, err := h.shipments.MapShipment(c, cancelled)
	if err != nil {
		h.HandleError(c, err, fulfillmentconstant.FAILED_TO_CANCEL_MSG)
		return
	}
	h.Success(c, http.StatusOK, fulfillmentconstant.SHIPMENT_CANCELLED_MSG, map[string]any{
		"shipment": mapped,
	})
}

// GetLabel streams the printable label PDF. Success is raw bytes (never the
// JSON envelope); errors use the envelope. No-AWB boxes are 400; transient
// provider failures are 503 under the same code.
func (h *ShipmentHandler) GetLabel(c *gin.Context) {
	sellerID, ok := h.sellerID(c)
	if !ok {
		return
	}
	id, err := h.ParseUintParam(c, "id")
	if err != nil {
		h.HandleValidationError(c, err)
		return
	}

	label, err := h.shipments.GetLabelBytes(c, sellerID, id)
	if err != nil {
		if err == fulfillmenterrors.ErrorLabelFailed {
			h.HandleError(c, err, fulfillmentconstant.FAILED_TO_LABEL_MSG)
			return
		}
		commonModel.ErrorWithCode(c, http.StatusServiceUnavailable,
			"Label temporarily unavailable", fulfillmentconstant.LABEL_FAILED_CODE)
		return
	}
	c.Header("Content-Type", "application/pdf")
	c.Header("Content-Disposition", "attachment; filename=label.pdf")
	c.Data(http.StatusOK, "application/pdf", label)
}

// ConfirmAddress accepts the current delivery address for a draft after a
// serviceability re-check.

// GetRates shops live rates (cached, throttled). Empty options are 200.
func (h *ShipmentHandler) GetRates(c *gin.Context) {
	sellerID, ok := h.sellerID(c)
	if !ok {
		return
	}

	var req model.RatesRequest
	if err := h.BindJSON(c, &req); err != nil {
		h.HandleValidationError(c, err)
		return
	}

	options, err := h.rates.GetRates(c, sellerID, req)
	if err != nil {
		h.HandleError(c, err, fulfillmentconstant.FAILED_TO_RATE_MSG)
		return
	}
	if options == nil {
		options = []model.RateOptionResponse{}
	}
	h.Success(c, http.StatusOK, fulfillmentconstant.RATES_FETCHED_MSG, map[string]any{
		"options": options,
	})
}

// bookOptions maps the book body to service options (RFC3339 pickup time).
func bookOptions(req model.BookShipmentRequest) (fulfillmentservice.BookDraftOptions, error) {
	opts := fulfillmentservice.BookDraftOptions{
		ProviderCode: req.ProviderCode,
		ServiceCode:  req.ServiceCode,
	}
	if req.PickupAt != nil && *req.PickupAt != "" {
		parsed, err := time.Parse(time.RFC3339, *req.PickupAt)
		if err != nil {
			return opts, err
		}
		opts.PickupAt = &parsed
	}
	return opts, nil
}

func derefShipments(in []*entity.FulfillmentShipment) []entity.FulfillmentShipment {
	out := make([]entity.FulfillmentShipment, 0, len(in))
	for _, s := range in {
		if s != nil {
			out = append(out, *s)
		}
	}
	return out
}

// shipmentsForResult reloads result rows for response mapping.
func (h *ShipmentHandler) shipmentsForResult(
	c *gin.Context,
	sellerID uint,
	ids []uint,
) ([]model.ShipmentResponse, error) {
	mapped := make([]model.ShipmentResponse, 0, len(ids))
	for _, id := range ids {
		shipment, err := h.shipments.GetShipment(c, sellerID, id)
		if err != nil {
			return nil, err
		}
		one, err := h.shipments.MapShipment(c, shipment)
		if err != nil {
			return nil, err
		}
		mapped = append(mapped, one)
	}
	return mapped, nil
}

// ConfirmAddress accepts the current delivery address for a draft after a
// serviceability re-check.
func (h *ShipmentHandler) ConfirmAddress(c *gin.Context) {
	sellerID, ok := h.sellerID(c)
	if !ok {
		return
	}
	id, err := h.ParseUintParam(c, "id")
	if err != nil {
		h.HandleValidationError(c, err)
		return
	}

	confirmed, err := h.shipments.ConfirmAddress(c, sellerID, id)
	if err != nil {
		h.HandleError(c, err, fulfillmentconstant.FAILED_TO_CONFIRM_ADDRESS_MSG)
		return
	}
	mapped, err := h.shipments.MapShipment(c, confirmed)
	if err != nil {
		h.HandleError(c, err, fulfillmentconstant.FAILED_TO_CONFIRM_ADDRESS_MSG)
		return
	}
	h.Success(c, http.StatusOK, fulfillmentconstant.ADDRESS_CONFIRMED_MSG, map[string]any{
		"shipment": mapped,
	})
}

// ActNDR answers the open NDR round: reattempt delivery or convert to RTO.
// The box status stays ndr_pending until the courier confirms the next leg.
func (h *ShipmentHandler) ActNDR(c *gin.Context) {
	sellerID, ok := h.sellerID(c)
	if !ok {
		return
	}
	id, err := h.ParseUintParam(c, "id")
	if err != nil {
		h.HandleValidationError(c, err)
		return
	}

	var req model.NDRActionRequest
	if err := h.BindJSON(c, &req); err != nil {
		h.HandleValidationError(c, err)
		return
	}

	round, err := h.shipments.ActNDR(c, sellerID, id, req.Action, req.AddressNote)
	if err != nil {
		h.HandleError(c, err, fulfillmentconstant.FAILED_TO_NDR_MSG)
		return
	}
	h.Success(c, http.StatusOK, fulfillmentconstant.NDR_ACTED_MSG, map[string]any{
		"ndr": map[string]any{
			"attemptNo":   round.AttemptNo,
			"ndrStatus":   round.NDRStatus,
			"actionTaken": round.ActionTaken,
			"actedAt":     round.ActedAt,
		},
	})
}

// RequestRTO asks for a mid-transit box back without an open NDR round.
func (h *ShipmentHandler) RequestRTO(c *gin.Context) {
	sellerID, ok := h.sellerID(c)
	if !ok {
		return
	}
	id, err := h.ParseUintParam(c, "id")
	if err != nil {
		h.HandleValidationError(c, err)
		return
	}

	box, err := h.shipments.RequestRTO(c, sellerID, id)
	if err != nil {
		h.HandleError(c, err, fulfillmentconstant.FAILED_TO_RTO_MSG)
		return
	}
	mapped, err := h.shipments.MapShipment(c, box)
	if err != nil {
		h.HandleError(c, err, fulfillmentconstant.FAILED_TO_RTO_MSG)
		return
	}
	h.Success(c, http.StatusOK, fulfillmentconstant.RTO_REQUESTED_MSG, map[string]any{
		"shipment": mapped,
	})
}

// RequestReturn creates a return box for a delivered original and books it
// immediately under the same book rules as forward freight.
func (h *ShipmentHandler) RequestReturn(c *gin.Context) {
	sellerID, ok := h.sellerID(c)
	if !ok {
		return
	}
	id, err := h.ParseUintParam(c, "id")
	if err != nil {
		h.HandleValidationError(c, err)
		return
	}

	var req model.ReturnRequest
	if err := h.BindJSON(c, &req); err != nil {
		h.HandleValidationError(c, err)
		return
	}
	items := make([]model.ShipmentItemRequest, 0, len(req.Items))
	for _, item := range req.Items {
		items = append(items, model.ShipmentItemRequest{
			OrderItemID: item.OrderItemID,
			Quantity:    item.Quantity,
		})
	}

	created, err := h.shipments.RequestReturn(c, sellerID, id, req.Reason, items)
	if err != nil {
		// Guard failures (non-delivered original, return-of-return,
		// over-quantity) are 400 under one code; unknown ids stay 404.
		if err == fulfillmenterrors.ErrorFulfillmentNotFound {
			h.HandleError(c, err, fulfillmentconstant.FAILED_TO_RETURN_MSG)
			return
		}
		commonModel.ErrorWithCode(c, http.StatusBadRequest, err.Error(), fulfillmentservice.ErrorCodeOf(err))
		return
	}
	mapped, err := h.shipments.MapShipment(c, created)
	if err != nil {
		h.HandleError(c, err, fulfillmentconstant.FAILED_TO_RETURN_MSG)
		return
	}
	h.Success(c, http.StatusCreated, fulfillmentconstant.RETURN_CREATED_MSG, map[string]any{
		"shipment": mapped,
	})
}
