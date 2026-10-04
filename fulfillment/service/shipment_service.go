package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	commonError "ecommerce-be/common/error"
	"ecommerce-be/common/log"
	"ecommerce-be/fulfillment/entity"
	fulfillmenterrors "ecommerce-be/fulfillment/error"
	fulfillmentfactory "ecommerce-be/fulfillment/factory"
	"ecommerce-be/fulfillment/model"
	"ecommerce-be/fulfillment/repository"
)

// ShipmentService owns draft boxes and their booking lifecycle: manual
// creation, listing, detail, draft edits, booking, pickup, cancel, labels,
// and address confirmation. Tracking lives in US4, NDR/returns in US5.
type ShipmentService interface {
	CreateDraft(ctx context.Context, sellerID uint, req model.CreateShipmentRequest, idempotencyKey string) (*entity.FulfillmentShipment, bool, error)
	ListShipments(ctx context.Context, sellerID uint, filter repository.ShipmentFilter) ([]entity.FulfillmentShipment, int64, error)
	GetShipment(ctx context.Context, sellerID uint, id uint) (*entity.FulfillmentShipment, error)
	UpdateDraft(ctx context.Context, sellerID uint, id uint, req model.UpdateShipmentRequest) (*entity.FulfillmentShipment, error)
	ShipmentEvents(ctx context.Context, sellerID uint, id uint) ([]entity.FulfillmentShipmentEvent, error)
	ShipmentNDR(ctx context.Context, sellerID uint, id uint) (*entity.FulfillmentNDR, error)
	MapShipments(ctx context.Context, shipments []entity.FulfillmentShipment) ([]model.ShipmentResponse, error)
	MapShipment(ctx context.Context, shipment *entity.FulfillmentShipment) (model.ShipmentResponse, error)
	BookDraft(ctx context.Context, sellerID uint, shipmentID uint, opts BookDraftOptions) (*entity.FulfillmentShipment, error)
	BookAllShipments(ctx context.Context, sellerID uint, orderID uint, opts BookDraftOptions) ([]*entity.FulfillmentShipment, []model.BookFailure, error)
	ConfirmAddress(ctx context.Context, sellerID uint, id uint) (*entity.FulfillmentShipment, error)
	CancelShipment(ctx context.Context, sellerID uint, id uint) (*entity.FulfillmentShipment, error)
	SchedulePickup(ctx context.Context, sellerID uint, id uint, pickupAt *time.Time) (*entity.FulfillmentShipment, error)
	GetLabelBytes(ctx context.Context, sellerID uint, id uint) ([]byte, error)
	ActNDR(ctx context.Context, sellerID uint, shipmentID uint, action string, addressNote string) (*entity.FulfillmentNDR, error)
	RequestRTO(ctx context.Context, sellerID uint, shipmentID uint) (*entity.FulfillmentShipment, error)
	RequestReturn(ctx context.Context, sellerID uint, origID uint, reason string, items []model.ShipmentItemRequest) (*entity.FulfillmentShipment, error)
}

// ShipmentServiceImpl implements ShipmentService.
type ShipmentServiceImpl struct {
	shipmentRepo repository.ShipmentRepository
	itemRepo     repository.ShipmentItemRepository
	eventRepo    repository.ShipmentEventRepository
	ndrRepo      repository.NDRRepository
	orderHooks   FulfillmentOrderHooks
	couriers     *fulfillmentfactory.CourierPartnerFactory
	inventory    FulfillmentInventoryHooks
	planner      ShipmentPlanner
	rates        RateService
}

// NewShipmentService builds the draft + booking service. Planner and rates
// attach later via setters to avoid build cycles.
func NewShipmentService(
	shipmentRepo repository.ShipmentRepository,
	itemRepo repository.ShipmentItemRepository,
	eventRepo repository.ShipmentEventRepository,
	ndrRepo repository.NDRRepository,
	orderHooks FulfillmentOrderHooks,
	couriers *fulfillmentfactory.CourierPartnerFactory,
	inventory FulfillmentInventoryHooks,
) ShipmentService {
	return &ShipmentServiceImpl{
		shipmentRepo: shipmentRepo,
		itemRepo:     itemRepo,
		eventRepo:    eventRepo,
		ndrRepo:      ndrRepo,
		orderHooks:   orderHooks,
		couriers:     couriers,
		inventory:    inventory,
	}
}

// BookDraftOptions selects the courier for one booking. ProviderCode is
// always set by callers (explicit body, tenant auto-book, or recover,
// which replays the code stamped at mark time).
type BookDraftOptions struct {
	ProviderCode string
	ServiceCode  string
	PickupAt     *time.Time
}

// DraftBooker is the booking capability the planner and recover paths use.
// Implemented by ShipmentServiceImpl; setter-injected to avoid build cycles.
type DraftBooker interface {
	BookDraft(ctx context.Context, sellerID uint, shipmentID uint, opts BookDraftOptions) (*entity.FulfillmentShipment, error)
}

// SetPlanner wires the planner for the stock-mismatch replan path.
func (s *ShipmentServiceImpl) SetPlanner(planner ShipmentPlanner) {
	s.planner = planner
}

// SetRateService wires rates for the confirm-address serviceability check.
func (s *ShipmentServiceImpl) SetRateService(rates RateService) {
	s.rates = rates
}

// CreateDraft inserts a manual draft for lines the planner did not cover.
// Guards: confirmed directship order owned by the seller; lines belong to
// the order; requested qty fits the uncovered remainder. Repeat
// idempotency keys return the original shipment (replayed=true) without a
// second row.
func (s *ShipmentServiceImpl) CreateDraft(
	ctx context.Context,
	sellerID uint,
	req model.CreateShipmentRequest,
	idempotencyKey string,
) (*entity.FulfillmentShipment, bool, error) {
	if strings.TrimSpace(idempotencyKey) != "" {
		if existing, err := s.shipmentRepo.FindByIdempotencyKey(ctx, idempotencyKey); err == nil {
			owned, err := s.ownedShipment(ctx, sellerID, existing)
			if err != nil {
				return nil, false, err
			}
			return owned, true, nil
		} else if err != fulfillmenterrors.ErrorFulfillmentNotFound {
			return nil, false, err
		}
	}

	view, err := s.plannableView(ctx, sellerID, req.OrderID)
	if err != nil {
		return nil, false, err
	}
	byItem, err := s.indexOrderLines(view, req.Items)
	if err != nil {
		return nil, false, err
	}

	var created *entity.FulfillmentShipment
	err = s.orderHooks.WithOrderLock(ctx, req.OrderID, func(lockCtx context.Context) error {
		for _, line := range req.Items {
			covered, err := s.itemRepo.SumQuantityByOrderItem(lockCtx, line.OrderItemID)
			if err != nil {
				return err
			}
			if line.Quantity > byItem[line.OrderItemID]-covered {
				return fulfillmenterrors.ErrorFulfillmentInvalidState.WithMessagef(
					"only %d of order line %d remains uncovered",
					byItem[line.OrderItemID]-covered, line.OrderItemID)
			}
		}
		draft := &entity.FulfillmentShipment{
			OrderID:                  req.OrderID,
			SellerID:                 sellerID,
			Status:                   entity.SHIPMENT_STATUS_DRAFT,
			PickupLocationID:         &req.PickupLocationID,
			DeliveryAddressID:        uintPtr(view.DeliveryAddressID),
			DeliveryAddressRevisedAt: &view.DeliveryAddressUpdatedAt,
			WeightGrams:              req.WeightGrams,
			LengthCm:                 req.LengthCm,
			BreadthCm:                req.BreadthCm,
			HeightCm:                 req.HeightCm,
			RawRef:                   map[string]any{},
		}
		if strings.TrimSpace(idempotencyKey) != "" {
			draft.IdempotencyKey = &idempotencyKey
		}
		if err := s.shipmentRepo.Create(lockCtx, draft); err != nil {
			return fmt.Errorf("create draft: %w", err)
		}
		for _, line := range req.Items {
			if err := s.itemRepo.Create(lockCtx, &entity.FulfillmentShipmentItem{
				ShipmentID:  draft.ID,
				OrderItemID: line.OrderItemID,
				Quantity:    line.Quantity,
			}); err != nil {
				return fmt.Errorf("create draft item: %w", err)
			}
		}
		if err := s.eventRepo.Create(lockCtx, &entity.FulfillmentShipmentEvent{
			ShipmentID: draft.ID,
			EventType:  string(entity.SHIPMENT_STATUS_DRAFT),
			ToStatus:   string(entity.SHIPMENT_STATUS_DRAFT),
			Source:     "api",
		}); err != nil {
			return fmt.Errorf("create draft ledger: %w", err)
		}
		created = draft
		return nil
	})
	if err != nil {
		return nil, false, err
	}
	reloaded, err := s.shipmentRepo.FindByID(ctx, created.ID)
	if err != nil {
		return nil, false, err
	}
	return reloaded, false, nil
}

// plannableView loads the order view and enforces ownership, confirmation
// (pending orders → 409), and the directship gate (other types → 409).
func (s *ShipmentServiceImpl) plannableView(
	ctx context.Context,
	sellerID uint,
	orderID uint,
) (*model.FulfillmentOrderView, error) {
	view, err := s.orderHooks.GetOrderForFulfillment(ctx, orderID)
	if err != nil {
		return nil, err
	}
	if view.SellerID != sellerID {
		return nil, fulfillmenterrors.ErrorFulfillmentNotFound
	}
	if err := checkPlannable(view); err != nil {
		return nil, err
	}
	return view, nil
}

// indexOrderLines maps order lines by id for membership checks.
func (s *ShipmentServiceImpl) indexOrderLines(
	view *model.FulfillmentOrderView,
	items []model.ShipmentItemRequest,
) (map[uint]int, error) {
	byItem := map[uint]int{}
	for _, line := range view.Items {
		byItem[line.OrderItemID] = line.Quantity
	}
	for _, item := range items {
		if _, ok := byItem[item.OrderItemID]; !ok {
			return nil, fulfillmenterrors.ErrorFulfillmentNotFound
		}
	}
	return byItem, nil
}

// ownedShipment re-checks tenancy on a row located by a non-scoped key
// (idempotency replay, AWB locate). Wrong owner reads as not found.
func (s *ShipmentServiceImpl) ownedShipment(
	ctx context.Context,
	sellerID uint,
	shipment *entity.FulfillmentShipment,
) (*entity.FulfillmentShipment, error) {
	if shipment.SellerID != sellerID {
		return nil, fulfillmenterrors.ErrorFulfillmentNotFound
	}
	return s.shipmentRepo.FindByID(ctx, shipment.ID)
}

// ListShipments returns the seller's boxes with filters and a total count.
// An orderId belonging to another seller reads as not found (no leak).
// Unlike plan/create, listing imposes no confirmation gate: pending orders
// simply have no boxes yet.
func (s *ShipmentServiceImpl) ListShipments(
	ctx context.Context,
	sellerID uint,
	filter repository.ShipmentFilter,
) ([]entity.FulfillmentShipment, int64, error) {
	filter.SellerID = sellerID
	if filter.OrderID != 0 {
		view, err := s.orderHooks.GetOrderForFulfillment(ctx, filter.OrderID)
		if err != nil {
			return nil, 0, err
		}
		if view.SellerID != sellerID {
			return nil, 0, fulfillmenterrors.ErrorFulfillmentNotFound
		}
	}
	shipments, err := s.shipmentRepo.FindShipments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}
	total, err := s.shipmentRepo.CountShipments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}
	return shipments, total, nil
}

// GetShipment returns one box with items preloaded. Tenant-checked.
func (s *ShipmentServiceImpl) GetShipment(
	ctx context.Context,
	sellerID uint,
	id uint,
) (*entity.FulfillmentShipment, error) {
	shipment, err := s.shipmentRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if shipment.SellerID != sellerID {
		return nil, fulfillmenterrors.ErrorFulfillmentNotFound
	}
	return shipment, nil
}

// UpdateDraft edits draft measurements. Non-drafts refuse; empty bodies
// fail validation. Weight/dims arrive as pointers (null = untouched).
func (s *ShipmentServiceImpl) UpdateDraft(
	ctx context.Context,
	sellerID uint,
	id uint,
	req model.UpdateShipmentRequest,
) (*entity.FulfillmentShipment, error) {
	if req.IsEmpty() {
		return nil, commonError.ErrValidation.WithMessagef("at least one field is required")
	}
	shipment, err := s.GetShipment(ctx, sellerID, id)
	if err != nil {
		return nil, err
	}
	if shipment.Status != entity.SHIPMENT_STATUS_DRAFT {
		return nil, fulfillmenterrors.ErrorFulfillmentInvalidState.WithMessagef(
			"only drafts can be edited")
	}
	patch := map[string]any{}
	if req.WeightGrams != nil {
		patch["weight_grams"] = *req.WeightGrams
	}
	if req.LengthCm != nil {
		patch["length_cm"] = *req.LengthCm
	}
	if req.BreadthCm != nil {
		patch["breadth_cm"] = *req.BreadthCm
	}
	if req.HeightCm != nil {
		patch["height_cm"] = *req.HeightCm
	}
	moved, err := s.shipmentRepo.UpdateStatusIfCurrent(ctx, id, entity.SHIPMENT_STATUS_DRAFT, entity.SHIPMENT_STATUS_DRAFT, patch)
	if err != nil {
		return nil, err
	}
	if !moved {
		return nil, fulfillmenterrors.ErrorFulfillmentInvalidState.WithMessagef(
			"draft moved concurrently; reload and retry")
	}
	return s.shipmentRepo.FindByID(ctx, id)
}

// ShipmentEvents returns the box ledger (detail view).
func (s *ShipmentServiceImpl) ShipmentEvents(
	ctx context.Context,
	sellerID uint,
	id uint,
) ([]entity.FulfillmentShipmentEvent, error) {
	if _, err := s.GetShipment(ctx, sellerID, id); err != nil {
		return nil, err
	}
	return s.eventRepo.FindByShipmentID(ctx, id)
}

// ShipmentNDR returns the open NDR round, if any (detail view).
func (s *ShipmentServiceImpl) ShipmentNDR(
	ctx context.Context,
	sellerID uint,
	id uint,
) (*entity.FulfillmentNDR, error) {
	if _, err := s.GetShipment(ctx, sellerID, id); err != nil {
		return nil, err
	}
	return s.ndrRepo.FindOpenByShipment(ctx, id)
}

// MapShipments maps boxes to API responses, resolving display currency per
// order (one order-hook read per distinct order, never per box).
func (s *ShipmentServiceImpl) MapShipments(
	ctx context.Context,
	shipments []entity.FulfillmentShipment,
) ([]model.ShipmentResponse, error) {
	currencyByOrder := map[uint]string{}
	responses := make([]model.ShipmentResponse, 0, len(shipments))
	for i := range shipments {
		shipment := &shipments[i]
		code, ok := currencyByOrder[shipment.OrderID]
		if !ok {
			code = "INR"
			if view, err := s.orderHooks.GetOrderForFulfillment(ctx, shipment.OrderID); err == nil && view.CurrencyCode != "" {
				code = view.CurrencyCode
			}
			currencyByOrder[shipment.OrderID] = code
		}
		responses = append(responses, model.ToShipmentResponse(shipment, shipment.Items, code))
	}
	return responses, nil
}

// MapShipment maps one box (detail paths resolve items via the repository).
func (s *ShipmentServiceImpl) MapShipment(
	ctx context.Context,
	shipment *entity.FulfillmentShipment,
) (model.ShipmentResponse, error) {
	mapped, err := s.MapShipments(ctx, []entity.FulfillmentShipment{*shipment})
	if err != nil {
		return model.ShipmentResponse{}, err
	}
	return mapped[0], nil
}

// resolveBooking resolves (adapter + decrypted creds + frozen config) for
// one booking: production row first, sandbox fallback. Single place for
// book/cancel/label/pickup so env precedence never diverges.
func (s *ShipmentServiceImpl) resolveBooking(
	ctx context.Context,
	sellerID uint,
	providerCode string,
) (*fulfillmentfactory.ResolvedCredentials, error) {
	resolved, err := s.couriers.ResolveForSeller(ctx, sellerID, providerCode, "production")
	if err == nil {
		return resolved, nil
	}
	if err != fulfillmenterrors.ErrorProviderNotConfigured {
		return nil, err
	}
	return s.couriers.ResolveForSeller(ctx, sellerID, providerCode, "sandbox")
}

// BookDraft books one draft: preconditions → pre-HTTP mark → courier → Tx2.
// Already-booked boxes return as-is (idempotent replay); picked+ boxes
// refuse. The mark transaction stamps provider choice + attempt BEFORE any
// courier call, so crashes and 502s are recoverable by re-click or cron.
func (s *ShipmentServiceImpl) BookDraft(
	ctx context.Context,
	sellerID uint,
	shipmentID uint,
	opts BookDraftOptions,
) (*entity.FulfillmentShipment, error) {
	draft, err := s.GetShipment(ctx, sellerID, shipmentID)
	if err != nil {
		return nil, err
	}
	switch draft.Status {
	case entity.SHIPMENT_STATUS_BOOKED, entity.SHIPMENT_STATUS_PICKUP_SCHEDULED:
		// Idempotent replay: heal fulfillment side effects too (covers a
		// crash between commit and the booked hook), then return current.
		if hookErr := s.emitBooked(ctx, draft); hookErr != nil {
			log.ErrorWithContext(ctx, "book replay: booked hook failed", hookErr)
		}
		return s.shipmentRepo.FindByID(ctx, draft.ID)
	case entity.SHIPMENT_STATUS_DRAFT:
	default:
		return nil, fulfillmenterrors.ErrorFulfillmentInvalidState.WithMessagef(
			"only drafts can be booked")
	}

	resolved, err := s.resolveBooking(ctx, sellerID, opts.ProviderCode)
	if err != nil {
		return nil, err
	}
	weight := draft.WeightGrams
	if weight == nil || *weight <= 0 {
		weight = resolved.Config.DefaultWeightGrams
	}
	if weight == nil || *weight <= 0 {
		return nil, fulfillmenterrors.ErrorWeightRequired
	}
	view, lines, err := s.bookInputs(ctx, sellerID, draft)
	if err != nil {
		return nil, err
	}
	if addrChanged(view, draft) {
		return nil, fulfillmenterrors.ErrorAddressChanged
	}
	if err := s.adoptDraftLines(ctx, sellerID, draft, view, lines); err != nil {
		return nil, s.failDraftStock(ctx, draft, sellerID, err)
	}

	if err := s.markBookAttempt(ctx, draft, opts.ProviderCode); err != nil {
		return nil, err
	}
	input := bookAdapterInput(draft, view, weight, opts)
	out, err := resolved.Adapter.BookShipment(ctx, input, resolved.Creds)
	if err != nil {
		return nil, fulfillmenterrors.ErrorBookFailed.WithMessagef("%s", firstLine(err.Error()))
	}
	return s.commitBook(ctx, draft, resolved, opts, out)
}

// bookInputs loads the order view plus variant-linked adopt lines for a draft.
func (s *ShipmentServiceImpl) bookInputs(
	ctx context.Context,
	sellerID uint,
	draft *entity.FulfillmentShipment,
) (*model.FulfillmentOrderView, []model.ReservationLine, error) {
	view, err := s.orderHooks.GetOrderForFulfillment(ctx, draft.OrderID)
	if err != nil {
		return nil, nil, err
	}
	if view.SellerID != sellerID {
		return nil, nil, fulfillmenterrors.ErrorFulfillmentNotFound
	}
	variantByItem := map[uint]*uint{}
	for _, line := range view.Items {
		variantByItem[line.OrderItemID] = line.VariantID
	}
	items, err := s.itemRepo.FindByShipmentID(ctx, draft.ID)
	if err != nil {
		return nil, nil, err
	}
	lines := make([]model.ReservationLine, 0, len(items))
	for _, item := range items {
		lines = append(lines, model.ReservationLine{
			OrderID:    draft.OrderID,
			VariantID:  variantByItem[item.OrderItemID],
			LocationID: derefUint(draft.PickupLocationID),
			Quantity:   item.Quantity,
			ShipmentID: draft.ID,
		})
	}
	return view, lines, nil
}

// addrChanged reports whether the delivery address moved since planning.
func addrChanged(view *model.FulfillmentOrderView, draft *entity.FulfillmentShipment) bool {
	if draft.DeliveryAddressRevisedAt == nil {
		return true
	}
	return view.DeliveryAddressUpdatedAt.After(*draft.DeliveryAddressRevisedAt)
}

// adoptDraftLines verifies the order's holds still cover the draft (adopt,
// never new reservations) through the inventory surface.
func (s *ShipmentServiceImpl) adoptDraftLines(
	ctx context.Context,
	sellerID uint,
	draft *entity.FulfillmentShipment,
	view *model.FulfillmentOrderView,
	lines []model.ReservationLine,
) error {
	_ = view
	_ = draft
	return s.inventory.ReserveForShipment(ctx, sellerID, lines)
}

// inventoryAdopt is deprecated in favor of adoptDraftLines; kept for
// interface symmetry during the US3 rollout.
func (s *ShipmentServiceImpl) inventoryAdopt(
	ctx context.Context,
	sellerID uint,
	lines []model.ReservationLine,
) error {
	return s.inventory.ReserveForShipment(ctx, sellerID, lines)
}

// failDraftStock cancels the draft whose holds died, emits the ledger row,
// and asks the planner to cover the gap again. Returns the stock error.
func (s *ShipmentServiceImpl) failDraftStock(
	ctx context.Context,
	draft *entity.FulfillmentShipment,
	sellerID uint,
	cause error,
) error {
	now := time.Now().UTC()
	if _, err := s.shipmentRepo.UpdateStatusIfCurrent(ctx, draft.ID,
		entity.SHIPMENT_STATUS_DRAFT, entity.SHIPMENT_STATUS_CANCELLED,
		map[string]any{"cancelled_at": now}); err != nil {
		return cause
	}
	_ = s.eventRepo.Create(ctx, &entity.FulfillmentShipmentEvent{
		ShipmentID: draft.ID,
		EventType:  string(entity.SHIPMENT_STATUS_CANCELLED),
		FromStatus: strPtr(string(entity.SHIPMENT_STATUS_DRAFT)),
		ToStatus:   string(entity.SHIPMENT_STATUS_CANCELLED),
		Source:     "system",
	})
	if s.planner != nil {
		if _, err := s.planner.PlanForOrder(ctx, sellerID, draft.OrderID); err != nil {
			log.ErrorWithContext(ctx, "book: replan after stock death failed", err)
		}
	}
	return cause
}

// markBookAttempt stamps provider choice + attempt counter before any
// courier call (survives crashes; drives the recover job + cap).
func (s *ShipmentServiceImpl) markBookAttempt(
	ctx context.Context,
	draft *entity.FulfillmentShipment,
	providerCode string,
) error {
	now := time.Now().UTC()
	updates := map[string]any{
		"provider_code": providerCode,
		"book_attempts": draft.BookAttempts + 1,
	}
	if draft.BookRequestedAt == nil {
		updates["book_requested_at"] = now
	}
	moved, err := s.shipmentRepo.UpdateStatusIfCurrent(ctx, draft.ID,
		entity.SHIPMENT_STATUS_DRAFT, entity.SHIPMENT_STATUS_DRAFT, updates)
	if err != nil {
		return err
	}
	if !moved {
		return fulfillmenterrors.ErrorFulfillmentInvalidState.WithMessagef(
			"draft moved concurrently; reload and retry")
	}
	return nil
}

// bookAdapterInput composes the provider-agnostic booking payload from the
// draft, the order view (recipient, manifest lines with names + prices),
// and the effective weight.
func bookAdapterInput(
	draft *entity.FulfillmentShipment,
	view *model.FulfillmentOrderView,
	weight *int,
	opts BookDraftOptions,
) model.BookShipmentInput {
	priceByItem := map[uint]model.FulfillmentOrderItemView{}
	for _, line := range view.Items {
		priceByItem[line.OrderItemID] = line
	}
	in := model.BookShipmentInput{
		ShipmentID:       draft.ID,
		OrderID:          draft.OrderID,
		SellerID:         draft.SellerID,
		PickupLocationID: derefUint(draft.PickupLocationID),
		WeightGrams:      weight,
		LengthCm:         draft.LengthCm,
		BreadthCm:        draft.BreadthCm,
		HeightCm:         draft.HeightCm,
		RecipientName:    view.RecipientName,
		RecipientPhone:   view.RecipientPhone,
		Street:           view.DeliveryStreet,
		City:             view.DeliveryCity,
		State:            view.DeliveryState,
		Pincode:          view.DeliveryPincode,
		CodCents:         draft.CodCents,
	}
	if opts.PickupAt != nil {
		in.PickupAt = opts.PickupAt
	}
	for _, item := range draft.Items {
		unitPrice := int64(0)
		name, sku := "", ""
		if line, ok := priceByItem[item.OrderItemID]; ok {
			unitPrice, name, sku = line.UnitPriceCents, line.ProductName, line.SKU
		}
		in.Lines = append(in.Lines, model.BookLine{
			Name:           name,
			SKU:            sku,
			Quantity:       item.Quantity,
			UnitPriceCents: unitPrice,
		})
		in.SubTotalCents += int64(item.Quantity) * unitPrice
		in.Items = append(in.Items, model.ShipmentItemInput{
			OrderItemID: item.OrderItemID,
			Quantity:    item.Quantity,
		})
	}
	return in
}

// commitBook persists the provider result: frozen ids, AWB, courier, ETD,
// status move, ledger, and the booked hook. A lost race (already booked by
// a parallel click) reloads and returns the winner.
func (s *ShipmentServiceImpl) commitBook(
	ctx context.Context,
	draft *entity.FulfillmentShipment,
	resolved *fulfillmentfactory.ResolvedCredentials,
	opts BookDraftOptions,
	out *model.BookShipmentOutput,
) (*entity.FulfillmentShipment, error) {
	target := entity.SHIPMENT_STATUS_BOOKED
	if opts.PickupAt != nil {
		target = entity.SHIPMENT_STATUS_PICKUP_SCHEDULED
	}
	now := time.Now().UTC()
	patch := map[string]any{
		"provider_config_id":   resolved.ConfigID,
		"provider_order_id":    out.ProviderOrderID,
		"provider_shipment_id": out.ProviderShipmentID,
		"awb":                  out.AWB,
		"courier_name":         out.CourierName,
		"last_synced_at":       now,
	}
	if out.ServiceCode != "" {
		patch["service_code"] = out.ServiceCode
	}
	moved, err := s.shipmentRepo.UpdateStatusIfCurrent(ctx, draft.ID,
		entity.SHIPMENT_STATUS_DRAFT, target, patch)
	if err != nil {
		return nil, fmt.Errorf("commit book: %w", err)
	}
	if !moved {
		reloaded, err := s.shipmentRepo.FindByID(ctx, draft.ID)
		if err != nil {
			return nil, err
		}
		return reloaded, nil
	}
	if err := s.eventRepo.Create(ctx, &entity.FulfillmentShipmentEvent{
		ShipmentID: draft.ID,
		EventType:  string(target),
		FromStatus: strPtr(string(entity.SHIPMENT_STATUS_DRAFT)),
		ToStatus:   string(target),
		Source:     "api",
	}); err != nil {
		return nil, fmt.Errorf("commit book ledger: %w", err)
	}
	if err := s.emitBooked(ctx, draft); err != nil {
		return nil, err
	}
	return s.shipmentRepo.FindByID(ctx, draft.ID)
}

// emitBooked fires the booked hook with exact box lines. Return boxes skip
// fulfillment entirely: their stock left at forward booking and comes back
// through restock on return delivery — consuming holds again would
// double-count. Failures fail the call: unfulfilled holds would break
// restock later, so louder is safer (re-clicks heal via the replay path).
func (s *ShipmentServiceImpl) emitBooked(
	ctx context.Context,
	draft *entity.FulfillmentShipment,
) error {
	if draft.ReturnOfShipmentID != nil {
		return nil
	}
	items, err := s.itemRepo.FindByShipmentID(ctx, draft.ID)
	if err != nil {
		return fmt.Errorf("booked hook lines: %w", err)
	}
	progress := model.FulfillmentProgress{
		OrderID:          draft.OrderID,
		ShipmentID:       draft.ID,
		PickupLocationID: derefUint(draft.PickupLocationID),
	}
	for _, item := range items {
		progress.Items = append(progress.Items, model.FulfillmentLine{
			OrderItemID: item.OrderItemID,
			Quantity:    item.Quantity,
		})
	}
	return s.orderHooks.OnShipmentBooked(ctx, progress)
}

// BookAllShipments books every remaining draft on the order with one shared
// provider body. Per-draft failures collect and never fail the batch;
// already-booked boxes pass through unchanged. Callers treat empty success
// with failures as a 400 (handler maps the returned error).
func (s *ShipmentServiceImpl) BookAllShipments(
	ctx context.Context,
	sellerID uint,
	orderID uint,
	opts BookDraftOptions,
) ([]*entity.FulfillmentShipment, []model.BookFailure, error) {
	if _, err := s.plannableView(ctx, sellerID, orderID); err != nil {
		return nil, nil, err
	}
	shipments, err := s.shipmentRepo.FindByOrderID(ctx, orderID)
	if err != nil {
		return nil, nil, err
	}
	var done []*entity.FulfillmentShipment
	var failures []model.BookFailure
	for i := range shipments {
		shipment := &shipments[i]
		if shipment.SellerID != sellerID {
			continue
		}
		switch shipment.Status {
		case entity.SHIPMENT_STATUS_DRAFT:
		case entity.SHIPMENT_STATUS_BOOKED, entity.SHIPMENT_STATUS_PICKUP_SCHEDULED:
			done = append(done, shipment)
			continue
		default:
			continue
		}
		booked, err := s.BookDraft(ctx, sellerID, shipment.ID, opts)
		if err != nil {
			failures = append(failures, model.BookFailure{
				ShipmentID: shipment.ID,
				Code:       ErrorCodeOf(err),
				Message:    firstLine(err.Error()),
			})
			continue
		}
		done = append(done, booked)
	}
	if len(done) == 0 && len(failures) > 0 {
		return nil, failures, fulfillmenterrors.ErrorFulfillmentInvalidState.WithMessagef(
			"no draft could be booked")
	}
	return done, failures, nil
}

// ConfirmAddress accepts the current delivery address for a draft: unchanged
// since the stamp → no-op; changed → serviceability re-check, then re-stamp.
// Drafts only — booked boxes already move under the old address.
func (s *ShipmentServiceImpl) ConfirmAddress(
	ctx context.Context,
	sellerID uint,
	id uint,
) (*entity.FulfillmentShipment, error) {
	shipment, err := s.GetShipment(ctx, sellerID, id)
	if err != nil {
		return nil, err
	}
	if shipment.Status != entity.SHIPMENT_STATUS_DRAFT {
		return nil, fulfillmenterrors.ErrorFulfillmentInvalidState.WithMessagef(
			"only drafts confirm addresses")
	}
	view, err := s.orderHooks.GetOrderForFulfillment(ctx, shipment.OrderID)
	if err != nil {
		return nil, err
	}
	if shipment.DeliveryAddressRevisedAt != nil &&
		!view.DeliveryAddressUpdatedAt.After(*shipment.DeliveryAddressRevisedAt) {
		return shipment, nil
	}
	if s.rates != nil {
		if pickup := derefUint(shipment.PickupLocationID); pickup != 0 {
			if err := s.rates.CheckServiceable(ctx, sellerID, pickup, shipment.OrderID); err != nil {
				return nil, err
			}
		}
	}
	now := view.DeliveryAddressUpdatedAt
	moved, err := s.shipmentRepo.UpdateColumns(ctx, id, map[string]any{
		"delivery_address_id":         view.DeliveryAddressID,
		"delivery_address_revised_at": now,
	})
	if err != nil {
		return nil, err
	}
	if !moved {
		return nil, fulfillmenterrors.ErrorFulfillmentInvalidState.WithMessagef(
			"draft moved concurrently; reload and retry")
	}
	return s.shipmentRepo.FindByID(ctx, id)
}

// CancelShipment cancels a box. Drafts cancel locally (no AWB exists).
// Booked boxes cancel through the frozen-config account first; provider
// refusal leaves status and stock untouched (409). Holds are order-scoped,
// so draft cancellation releases nothing — the order cancel/return paths
// settle stock.
func (s *ShipmentServiceImpl) CancelShipment(
	ctx context.Context,
	sellerID uint,
	id uint,
) (*entity.FulfillmentShipment, error) {
	shipment, err := s.GetShipment(ctx, sellerID, id)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	cancel := func(from entity.ShipmentStatus) (*entity.FulfillmentShipment, error) {
		moved, err := s.shipmentRepo.UpdateStatusIfCurrent(ctx, id, from,
			entity.SHIPMENT_STATUS_CANCELLED, map[string]any{"cancelled_at": now})
		if err != nil {
			return nil, err
		}
		if !moved {
			return s.shipmentRepo.FindByID(ctx, id)
		}
		_ = s.eventRepo.Create(ctx, &entity.FulfillmentShipmentEvent{
			ShipmentID: id,
			EventType:  string(entity.SHIPMENT_STATUS_CANCELLED),
			FromStatus: strPtr(string(from)),
			ToStatus:   string(entity.SHIPMENT_STATUS_CANCELLED),
			Source:     "api",
		})
		return s.shipmentRepo.FindByID(ctx, id)
	}

	switch shipment.Status {
	case entity.SHIPMENT_STATUS_DRAFT:
		return cancel(shipment.Status)
	case entity.SHIPMENT_STATUS_BOOKED, entity.SHIPMENT_STATUS_PICKUP_SCHEDULED:
		if shipment.ProviderCode == nil || shipment.ProviderConfigID == nil {
			return nil, fulfillmenterrors.ErrorFulfillmentInvalidState.WithMessagef(
				"booked box without provider reference")
		}
		resolved, err := s.couriers.ResolveConfig(ctx, *shipment.ProviderConfigID)
		if err != nil {
			return nil, err
		}
		if err := resolved.Adapter.Cancel(ctx, model.CancelInput{
			ShipmentID:      id,
			AWB:             derefString(shipment.AWB),
			ProviderOrderID: derefString(shipment.ProviderOrderID),
		}, resolved.Creds); err != nil {
			return nil, fulfillmenterrors.ErrorFulfillmentInvalidState.WithMessagef(
				"courier refused cancel: %s", firstLine(err.Error()))
		}
		if _, err := s.unfulfillBoxHolds(ctx, sellerID, shipment); err != nil {
			return nil, err
		}
		return cancel(shipment.Status)
	default:
		return nil, fulfillmenterrors.ErrorFulfillmentInvalidState.WithMessagef(
			"only pre-pickup boxes can be cancelled")
	}
}

// SchedulePickup schedules pickup separately from booking (optional
// capability). Already-scheduled boxes return as-is.
func (s *ShipmentServiceImpl) SchedulePickup(
	ctx context.Context,
	sellerID uint,
	id uint,
	pickupAt *time.Time,
) (*entity.FulfillmentShipment, error) {
	shipment, err := s.GetShipment(ctx, sellerID, id)
	if err != nil {
		return nil, err
	}
	if shipment.Status == entity.SHIPMENT_STATUS_PICKUP_SCHEDULED {
		return shipment, nil
	}
	if shipment.Status != entity.SHIPMENT_STATUS_BOOKED {
		return nil, fulfillmenterrors.ErrorFulfillmentInvalidState.WithMessagef(
			"only booked boxes schedule pickup")
	}
	if shipment.ProviderCode == nil || shipment.ProviderConfigID == nil {
		return nil, fulfillmenterrors.ErrorFulfillmentInvalidState.WithMessagef(
			"booked box without provider reference")
	}
	resolved, err := s.couriers.ResolveConfig(ctx, *shipment.ProviderConfigID)
	if err != nil {
		return nil, err
	}
	scheduler, ok := resolved.Adapter.(courierPickupScheduler)
	if !ok {
		return nil, fulfillmenterrors.ErrorCapabilityUnsupported
	}
	if err := scheduler.SchedulePickup(ctx, model.PickupInput{
		ShipmentID:         id,
		AWB:                derefString(shipment.AWB),
		ProviderShipmentID: derefString(shipment.ProviderShipmentID),
		PickupAt:           pickupAt,
	}, resolved.Creds); err != nil {
		return nil, fmt.Errorf("schedule pickup: %w", err)
	}
	moved, err := s.shipmentRepo.UpdateStatusIfCurrent(ctx, id,
		entity.SHIPMENT_STATUS_BOOKED, entity.SHIPMENT_STATUS_PICKUP_SCHEDULED,
		map[string]any{"last_synced_at": time.Now().UTC()})
	if err != nil {
		return nil, err
	}
	if !moved {
		return s.shipmentRepo.FindByID(ctx, id)
	}
	_ = s.eventRepo.Create(ctx, &entity.FulfillmentShipmentEvent{
		ShipmentID: id,
		EventType:  string(entity.SHIPMENT_STATUS_PICKUP_SCHEDULED),
		FromStatus: strPtr(string(entity.SHIPMENT_STATUS_BOOKED)),
		ToStatus:   string(entity.SHIPMENT_STATUS_PICKUP_SCHEDULED),
		Source:     "api",
	})
	return s.shipmentRepo.FindByID(ctx, id)
}

// GetLabelBytes streams the printable label. No AWB yet → 400. Provider
// failures surface as plain errors so the handler maps them to 503 under
// the same code (api-contracts §7).
func (s *ShipmentServiceImpl) GetLabelBytes(
	ctx context.Context,
	sellerID uint,
	id uint,
) ([]byte, error) {
	shipment, err := s.GetShipment(ctx, sellerID, id)
	if err != nil {
		return nil, err
	}
	if shipment.AWB == nil || shipment.ProviderConfigID == nil || shipment.ProviderShipmentID == nil {
		return nil, fulfillmenterrors.ErrorLabelFailed
	}
	resolved, err := s.couriers.ResolveConfig(ctx, *shipment.ProviderConfigID)
	if err != nil {
		return nil, err
	}
	label, err := resolved.Adapter.GetLabel(ctx, model.LabelInput{
		ShipmentID:         id,
		AWB:                *shipment.AWB,
		ProviderShipmentID: *shipment.ProviderShipmentID,
	}, resolved.Creds)
	if err != nil {
		return nil, fmt.Errorf("label fetch transient: %w", err)
	}
	return label, nil
}

func derefUint(v *uint) uint {
	if v == nil {
		return 0
	}
	return *v
}

func derefString(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

func strPtr(s string) *string {
	return &s
}

func firstLine(s string) string {
	if i := strings.Index(s, "\n"); i >= 0 {
		return s[:i]
	}
	return s
}

// ActNDR answers the open NDR round: reattempt delivery or convert to RTO.
// Requires an open round (else 400) and an NDR-capable adapter. The box
// status stays ndr_pending until the courier confirms the next leg.
func (s *ShipmentServiceImpl) ActNDR(
	ctx context.Context,
	sellerID uint,
	shipmentID uint,
	action string,
	addressNote string,
) (*entity.FulfillmentNDR, error) {
	shipment, err := s.GetShipment(ctx, sellerID, shipmentID)
	if err != nil {
		return nil, err
	}
	if shipment.Status != entity.SHIPMENT_STATUS_NDR_PENDING {
		return nil, fulfillmenterrors.ErrorCapabilityUnsupported.WithMessagef(
			"no open NDR round on this box")
	}
	open, err := s.ndrRepo.FindOpenByShipment(ctx, shipmentID)
	if err != nil {
		return nil, err
	}
	if open == nil {
		return nil, fulfillmenterrors.ErrorCapabilityUnsupported.WithMessagef(
			"no open NDR round on this box")
	}
	if shipment.ProviderCode == nil || shipment.ProviderConfigID == nil {
		return nil, fulfillmenterrors.ErrorFulfillmentInvalidState.WithMessagef(
			"box without provider reference")
	}
	resolved, err := s.couriers.ResolveConfig(ctx, *shipment.ProviderConfigID)
	if err != nil {
		return nil, err
	}
	handler, ok := resolved.Adapter.(courierNDRHandler)
	if !ok {
		return nil, fulfillmenterrors.ErrorCapabilityUnsupported
	}
	awb := ""
	if shipment.AWB != nil {
		awb = *shipment.AWB
	}
	if err := handler.ActNDR(ctx, model.NDRActionInput{
		AWB:         awb,
		Action:      action,
		AddressNote: addressNote,
	}, resolved.Creds); err != nil {
		return nil, fmt.Errorf("NDR action failed: %w", err)
	}
	if err := s.ndrRepo.MarkActed(ctx, open.ID, action); err != nil {
		return nil, err
	}
	updated, err := s.ndrRepo.FindByShipmentID(ctx, shipmentID)
	if err != nil {
		return nil, err
	}
	for i := range updated {
		if updated[i].ID == open.ID {
			closed := updated[i]
			return &closed, nil
		}
	}
	return open, nil
}

// RequestRTO asks for a mid-transit box back without an open NDR round.
// Statuses picked/in_transit/out_for_delivery/ndr_pending only; local
// status never moves until the courier confirms the RTO leg.
func (s *ShipmentServiceImpl) RequestRTO(
	ctx context.Context,
	sellerID uint,
	shipmentID uint,
) (*entity.FulfillmentShipment, error) {
	shipment, err := s.GetShipment(ctx, sellerID, shipmentID)
	if err != nil {
		return nil, err
	}
	switch shipment.Status {
	case entity.SHIPMENT_STATUS_PICKED,
		entity.SHIPMENT_STATUS_IN_TRANSIT,
		entity.SHIPMENT_STATUS_OUT_FOR_DELIVERY,
		entity.SHIPMENT_STATUS_NDR_PENDING:
	default:
		return nil, fulfillmenterrors.ErrorFulfillmentInvalidState.WithMessagef(
			"only moving boxes can be requested back")
	}
	if shipment.ProviderCode == nil || shipment.ProviderConfigID == nil {
		return nil, fulfillmenterrors.ErrorFulfillmentInvalidState.WithMessagef(
			"box without provider reference")
	}
	resolved, err := s.couriers.ResolveConfig(ctx, *shipment.ProviderConfigID)
	if err != nil {
		return nil, err
	}
	requester, ok := resolved.Adapter.(courierRTORequester)
	if !ok {
		return nil, fulfillmenterrors.ErrorCapabilityUnsupported
	}
	if err := requester.RequestRTO(ctx, model.RTORequestInput{
		ShipmentID: shipmentID,
		AWB:        derefString(shipment.AWB),
	}, resolved.Creds); err != nil {
		return nil, fmt.Errorf("RTO request failed: %w", err)
	}
	if err := s.eventRepo.Create(ctx, &entity.FulfillmentShipmentEvent{
		ShipmentID: shipmentID,
		EventType:  "rto_requested",
		FromStatus: strPtr(string(shipment.Status)),
		ToStatus:   string(shipment.Status),
		Source:     "api",
	}); err != nil {
		return nil, fmt.Errorf("RTO ledger: %w", err)
	}
	return s.shipmentRepo.FindByID(ctx, shipmentID)
}

// RequestReturn creates a return box for a delivered original and books it
// immediately (same book rules as forward: weight required, attempts
// counted). Guards: original delivered, not itself a return, lines within
// the original box quantities. Return drafts inherit the original weight
// (same goods coming back; PATCHable) and collect no COD.
func (s *ShipmentServiceImpl) RequestReturn(
	ctx context.Context,
	sellerID uint,
	origID uint,
	reason string,
	items []model.ShipmentItemRequest,
) (*entity.FulfillmentShipment, error) {
	original, err := s.GetShipment(ctx, sellerID, origID)
	if err != nil {
		return nil, err
	}
	if original.Status != entity.SHIPMENT_STATUS_DELIVERED {
		return nil, fulfillmenterrors.ErrorFulfillmentInvalidState.WithMessagef(
			"only delivered boxes can be returned")
	}
	if original.ReturnOfShipmentID != nil {
		return nil, fulfillmenterrors.ErrorFulfillmentInvalidState.WithMessagef(
			"a return cannot be returned")
	}
	if len(items) == 0 {
		return nil, fulfillmenterrors.ErrorFulfillmentInvalidState.WithMessagef(
			"at least one return line is required")
	}
	origLines, err := s.itemRepo.FindByShipmentID(ctx, origID)
	if err != nil {
		return nil, err
	}
	allowed := map[uint]int{}
	for _, line := range origLines {
		allowed[line.OrderItemID] = line.Quantity
	}
	returned, err := s.returnedQuantities(ctx, origID)
	if err != nil {
		return nil, err
	}
	for _, item := range items {
		remaining, ok := allowed[item.OrderItemID]
		if !ok || item.Quantity <= 0 || item.Quantity > remaining-returned[item.OrderItemID] {
			return nil, fulfillmenterrors.ErrorFulfillmentInvalidState.WithMessagef(
				"return quantity exceeds the original box")
		}
	}

	var created *entity.FulfillmentShipment
	err = s.orderHooks.WithOrderLock(ctx, original.OrderID, func(lockCtx context.Context) error {
		draft := &entity.FulfillmentShipment{
			OrderID:            original.OrderID,
			SellerID:           sellerID,
			Status:             entity.SHIPMENT_STATUS_DRAFT,
			PickupLocationID:   original.PickupLocationID,
			DeliveryAddressID:  original.DeliveryAddressID,
			WeightGrams:        original.WeightGrams,
			LengthCm:           original.LengthCm,
			BreadthCm:          original.BreadthCm,
			HeightCm:           original.HeightCm,
			ReturnOfShipmentID: &origID,
			RawRef:             map[string]any{},
		}
		// Address revision starts current (copied from a just-delivered box).
		if original.DeliveryAddressRevisedAt != nil {
			stamp := *original.DeliveryAddressRevisedAt
			draft.DeliveryAddressRevisedAt = &stamp
		}
		if err := s.shipmentRepo.Create(lockCtx, draft); err != nil {
			return fmt.Errorf("create return draft: %w", err)
		}
		for _, item := range items {
			if err := s.itemRepo.Create(lockCtx, &entity.FulfillmentShipmentItem{
				ShipmentID:  draft.ID,
				OrderItemID: item.OrderItemID,
				Quantity:    item.Quantity,
			}); err != nil {
				return fmt.Errorf("create return draft item: %w", err)
			}
		}
		if err := s.eventRepo.Create(lockCtx, &entity.FulfillmentShipmentEvent{
			ShipmentID: draft.ID,
			EventType:  string(entity.SHIPMENT_STATUS_DRAFT),
			ToStatus:   string(entity.SHIPMENT_STATUS_DRAFT),
			Source:     "api",
		}); err != nil {
			return fmt.Errorf("create return ledger: %w", err)
		}
		created = draft
		return nil
	})
	if err != nil {
		return nil, err
	}
	return s.bookReturnBox(ctx, sellerID, created, reason)
}

// returnedQuantities sums return-box lines per order line for one original
// box, excluding cancelled returns (re-plannable quantity stays available).
func (s *ShipmentServiceImpl) returnedQuantities(
	ctx context.Context,
	origID uint,
) (map[uint]int, error) {
	original, err := s.shipmentRepo.FindByID(ctx, origID)
	if err != nil {
		return nil, err
	}
	siblings, err := s.shipmentRepo.FindByOrderID(ctx, original.OrderID)
	if err != nil {
		return nil, err
	}
	totals := map[uint]int{}
	for i := range siblings {
		box := &siblings[i]
		if box.ReturnOfShipmentID == nil || *box.ReturnOfShipmentID != origID {
			continue
		}
		if box.Status == entity.SHIPMENT_STATUS_CANCELLED {
			continue
		}
		for _, item := range box.Items {
			totals[item.OrderItemID] += item.Quantity
		}
	}
	return totals, nil
}

// ErrorCodeOf extracts the client-facing code for per-draft failure rows.
func ErrorCodeOf(err error) string {
	var appErr *commonError.AppError
	// errors.As unwraps %w chains: many service errors wrap AppErrors with
	// context, which direct assertion would miss and mislabel.
	if errors.As(err, &appErr) {
		return appErr.Code
	}
	return "FULFILLMENT_BOOK_FAILED"
}

// bookReturnBox books a return draft through the provider's return flow
// (ReturnHandler), then commits with the shared Tx2 shape.
func (s *ShipmentServiceImpl) bookReturnBox(
	ctx context.Context,
	sellerID uint,
	draft *entity.FulfillmentShipment,
	reason string,
) (*entity.FulfillmentShipment, error) {
	original, err := s.GetShipment(ctx, sellerID, *draft.ReturnOfShipmentID)
	if err != nil {
		return nil, err
	}
	if original.ProviderCode == nil {
		return nil, fulfillmenterrors.ErrorFulfillmentInvalidState.WithMessagef(
			"original box has no provider")
	}
	resolved, err := s.resolveBooking(ctx, sellerID, *original.ProviderCode)
	if err != nil {
		return nil, err
	}
	returner, ok := resolved.Adapter.(courierReturnHandler)
	if !ok {
		return nil, fulfillmenterrors.ErrorCapabilityUnsupported
	}
	view, _, err := s.bookInputs(ctx, sellerID, draft)
	if err != nil {
		return nil, err
	}
	weight := draft.WeightGrams
	if weight == nil || *weight <= 0 {
		weight = resolved.Config.DefaultWeightGrams
	}
	if weight == nil || *weight <= 0 {
		return nil, fulfillmenterrors.ErrorWeightRequired
	}
	input := bookAdapterInput(draft, view, weight, BookDraftOptions{})
	out, err := returner.BookReturn(ctx, model.BookReturnInput{
		OrigShipmentID: original.ID,
		Reason:         reason,
		Ship:           input,
	}, resolved.Creds)
	if err != nil {
		return nil, fulfillmenterrors.ErrorBookFailed.WithMessagef("%s", firstLine(err.Error()))
	}
	if err := s.markBookAttempt(ctx, draft, *original.ProviderCode); err != nil {
		return nil, err
	}
	return s.commitBook(ctx, draft, resolved, BookDraftOptions{}, out)
}

// unfulfillBoxHolds returns a booked-cancelled box's FULFILLED holds to
// CONFIRMED so re-planning keeps working. Best-effort after a successful
// courier cancel: failures log loudly but don't block the cancel itself
// (stock hygiene must never strand a seller-confirmed cancellation).
func (s *ShipmentServiceImpl) unfulfillBoxHolds(
	ctx context.Context,
	sellerID uint,
	shipment *entity.FulfillmentShipment,
) (*entity.FulfillmentShipment, error) {
	view, err := s.orderHooks.GetOrderForFulfillment(ctx, shipment.OrderID)
	if err != nil {
		log.ErrorWithContext(ctx, "cancel: order view unreadable, holds stay fulfilled", err)
		return shipment, nil
	}
	variantByItem := map[uint]*uint{}
	for _, line := range view.Items {
		variantByItem[line.OrderItemID] = line.VariantID
	}
	items, err := s.itemRepo.FindByShipmentID(ctx, shipment.ID)
	if err != nil {
		log.ErrorWithContext(ctx, "cancel: box lines unreadable, holds stay fulfilled", err)
		return shipment, nil
	}
	lines := make([]model.ReservationLine, 0, len(items))
	for _, item := range items {
		lines = append(lines, model.ReservationLine{
			OrderID:    shipment.OrderID,
			VariantID:  variantByItem[item.OrderItemID],
			LocationID: derefUint(shipment.PickupLocationID),
			Quantity:   item.Quantity,
		})
	}
	if err := s.inventory.UnfulfillHolds(ctx, sellerID, shipment.OrderID, lines); err != nil {
		log.ErrorWithContext(ctx, "cancel: unfulfill failed, holds stay fulfilled", err)
	}
	return shipment, nil
}

// courierPickupScheduler mirrors the optional capability structurally so
// this file never imports the courier contract package.
type courierPickupScheduler interface {
	SchedulePickup(ctx context.Context, in model.PickupInput, creds map[string]any) error
}

// courierNDRHandler mirrors the optional NDR capability structurally.
type courierNDRHandler interface {
	ActNDR(ctx context.Context, in model.NDRActionInput, creds map[string]any) error
}

// courierReturnHandler mirrors the optional return-booking capability.
type courierReturnHandler interface {
	BookReturn(ctx context.Context, in model.BookReturnInput, creds map[string]any) (*model.BookShipmentOutput, error)
}

// courierRTORequester mirrors the optional proactive-RTO capability.
type courierRTORequester interface {
	RequestRTO(ctx context.Context, in model.RTORequestInput, creds map[string]any) error
}
