package service

import (
	"context"
	"fmt"

	"ecommerce-be/common/db"
	"ecommerce-be/common/log"
	"ecommerce-be/fulfillment/entity"
	fulfillmenterrors "ecommerce-be/fulfillment/error"
	"ecommerce-be/fulfillment/model"
	"ecommerce-be/fulfillment/repository"
)

// fulfillmentTypeDirectship is the only order type this module plans, and
// orderStatusConfirmed the only plannable state. Compared as strings so
// fulfillment never imports order entities.
const (
	fulfillmentTypeDirectship = "directship"
	orderStatusConfirmed      = "confirmed"
)

// ShipmentPlanner auto-creates one draft box per warehouse when an order is
// confirmed. Plans are idempotent (covered orders return the live set) and
// race-safe (coverage is re-checked inside the order-row lock).
type ShipmentPlanner interface {
	PlanForOrder(ctx context.Context, sellerID uint, orderID uint) (*model.PlanResult, error)
}

// ShipmentPlannerImpl implements ShipmentPlanner.
type ShipmentPlannerImpl struct {
	shipmentRepo   repository.ShipmentRepository
	itemRepo       repository.ShipmentItemRepository
	eventRepo      repository.ShipmentEventRepository
	configRepo     repository.CourierProviderConfigRepository
	orderHooks     FulfillmentOrderHooks
	inventoryHooks FulfillmentInventoryHooks
	// productHooks fills draft weights from catalog specs. Nil-safe: a nil
	// reader (or variants without specs) leaves drafts NULL-weighted for
	// manual PATCH — missing data degrades to human action.
	productHooks FulfillmentProductHooks
	// booker runs tenant auto-book after planning; nil until wired (US3).
	// Setter-injected like order's SetShipmentPlanner: planner and booking
	// reference each other, so neither constructor takes the other.
	booker DraftBooker
}

// NewShipmentPlanner builds the planner. productHooks may be nil (US2).
func NewShipmentPlanner(
	shipmentRepo repository.ShipmentRepository,
	itemRepo repository.ShipmentItemRepository,
	eventRepo repository.ShipmentEventRepository,
	configRepo repository.CourierProviderConfigRepository,
	orderHooks FulfillmentOrderHooks,
	inventoryHooks FulfillmentInventoryHooks,
	productHooks FulfillmentProductHooks,
) ShipmentPlanner {
	return &ShipmentPlannerImpl{
		shipmentRepo:   shipmentRepo,
		itemRepo:       itemRepo,
		eventRepo:      eventRepo,
		configRepo:     configRepo,
		orderHooks:     orderHooks,
		inventoryHooks: inventoryHooks,
		productHooks:   productHooks,
	}
}

// SetBooker wires tenant auto-book after planning (called by the factory;
// nil keeps plan-only behavior).
func (p *ShipmentPlannerImpl) SetBooker(booker DraftBooker) {
	p.booker = booker
}

// PlanForOrder plans uncovered lines into per-warehouse drafts. Covered
// orders return the live set without writing. Auto-book is NOT here: US3
// (T040) runs the book path per draft afterwards for opted-in tenants.
func (p *ShipmentPlannerImpl) PlanForOrder(
	ctx context.Context,
	sellerID uint,
	orderID uint,
) (*model.PlanResult, error) {
	view, err := p.orderHooks.GetOrderForFulfillment(ctx, orderID)
	if err != nil {
		return nil, err
	}
	if view.SellerID != sellerID {
		return nil, fulfillmenterrors.ErrorFulfillmentNotFound
	}
	if err := checkPlannable(view); err != nil {
		return nil, err
	}

	result := &model.PlanResult{}
	err = p.orderHooks.WithOrderLock(ctx, orderID, func(lockCtx context.Context) error {
		uncovered, liveIDs, err := p.uncoveredLines(lockCtx, view)
		if err != nil {
			return err
		}
		if len(uncovered) == 0 {
			result.ShipmentIDs = liveIDs
			return nil
		}
		ids, err := p.allocateAndInsert(lockCtx, view, uncovered)
		if err != nil {
			return err
		}
		result.ShipmentIDs = append(liveIDs, ids...)
		result.CreatedNew = true
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// uncoveredLines returns order lines with quantity not yet on a
// non-cancelled shipment, plus the live shipment ids for the replay path.
func (p *ShipmentPlannerImpl) uncoveredLines(
	ctx context.Context,
	view *model.FulfillmentOrderView,
) ([]model.ShipmentItemRequest, []uint, error) {
	shipments, err := p.shipmentRepo.FindByOrderID(ctx, view.OrderID)
	if err != nil {
		return nil, nil, err
	}
	covered := map[uint]int{}
	var liveIDs []uint
	for _, shipment := range shipments {
		if shipment.Status == entity.SHIPMENT_STATUS_CANCELLED {
			continue
		}
		liveIDs = append(liveIDs, shipment.ID)
		for _, item := range shipment.Items {
			covered[item.OrderItemID] += item.Quantity
		}
	}
	var uncovered []model.ShipmentItemRequest
	for _, line := range view.Items {
		if rest := line.Quantity - covered[line.OrderItemID]; rest > 0 {
			uncovered = append(uncovered, model.ShipmentItemRequest{
				OrderItemID: line.OrderItemID,
				Quantity:    rest,
			})
		}
	}
	return uncovered, liveIDs, nil
}

// allocation is one line slice assigned to a warehouse.
type allocation struct {
	orderItemID uint
	variantID   *uint
	locationID  uint
	quantity    int
}

// allocateAndInsert splits uncovered lines by warehouse priority, verifies
// the holds (adopt), and inserts one draft per warehouse in a single tx.
func (p *ShipmentPlannerImpl) allocateAndInsert(
	ctx context.Context,
	view *model.FulfillmentOrderView,
	uncovered []model.ShipmentItemRequest,
) ([]uint, error) {
	variantByItem := map[uint]*uint{}
	variantIDs := make([]uint, 0, len(view.Items))
	for _, line := range view.Items {
		variantByItem[line.OrderItemID] = line.VariantID
		if line.VariantID != nil {
			variantIDs = append(variantIDs, *line.VariantID)
		}
	}

	rows, err := p.inventoryHooks.GetAvailability(ctx, view.SellerID, view.OrderID, variantIDs)
	if err != nil {
		return nil, err
	}
	allocations, err := allocateByPriority(view, uncovered, variantByItem, rows)
	if err != nil {
		return nil, err
	}
	if err := p.inventoryHooks.ReserveForShipment(ctx, view.SellerID, adoptLines(view.OrderID, allocations)); err != nil {
		return nil, err
	}

	draftIDs, err := p.insertDraftsTx(ctx, view, allocations, variantByItem)
	if err != nil {
		return nil, err
	}

	if err := p.orderHooks.OnShipmentsPlanned(ctx, view.OrderID, draftIDs); err != nil {
		return nil, fmt.Errorf("planner emit planned: %w", err)
	}
	p.autoBookNewDrafts(ctx, view, draftIDs)
	return draftIDs, nil
}

// insertDraftsTx resolves catalog measures, then inserts one draft per
// warehouse in a single tx.
func (p *ShipmentPlannerImpl) insertDraftsTx(
	ctx context.Context,
	view *model.FulfillmentOrderView,
	allocations []allocation,
	variantByItem map[uint]*uint,
) ([]uint, error) {
	measures, err := p.draftMeasures(ctx, allocations)
	if err != nil {
		return nil, err
	}
	var draftIDs []uint
	err = db.WithTransaction(ctx, func(txCtx context.Context) error {
		ids, err := p.insertDrafts(txCtx, view, allocations, variantByItem, measures)
		if err != nil {
			return err
		}
		draftIDs = ids
		return nil
	})
	if err != nil {
		return nil, err
	}
	return draftIDs, nil
}

// autoBookNewDrafts books fresh drafts for opted-in tenants: exactly one
// active auto_book row picks the provider (plus its rate preference);
// zero rows stay manual; two or more raise AMBIGUOUS once for the dashboard.
// Booking failures log and continue — drafts remain for manual booking.
func (p *ShipmentPlannerImpl) autoBookNewDrafts(
	ctx context.Context,
	view *model.FulfillmentOrderView,
	draftIDs []uint,
) {
	if p.booker == nil || len(draftIDs) == 0 {
		return
	}
	configs, err := p.configRepo.FindBySeller(ctx, view.SellerID)
	if err != nil {
		log.ErrorWithContext(ctx, "planner auto-book: config read failed", err)
		return
	}
	var candidates []string
	for i := range configs {
		if configs[i].IsActive && configs[i].AutoBook {
			candidates = append(candidates, configs[i].ProviderCode)
		}
	}
	if len(candidates) == 0 {
		return
	}
	if len(candidates) > 1 {
		log.ErrorWithContext(ctx, "planner auto-book ambiguous: seller has several auto_book rows",
			fulfillmenterrors.ErrorAutoBookAmbiguous)
		return
	}
	for _, id := range draftIDs {
		if _, err := p.booker.BookDraft(ctx, view.SellerID, id, BookDraftOptions{
			ProviderCode: candidates[0],
		}); err != nil {
			log.ErrorWithContext(ctx, "planner auto-book failed, draft left for manual booking", err)
		}
	}
}

// allocateByPriority assigns each uncovered line from its CONFIRMED holds,
// grouped by warehouse in priority order. Held-first is deliberate: checkout
// already chose warehouse placement once, and a second allocator re-deciding
// from free stock can disagree with it (units held at loc2 planned from
// loc1). Free availability never feeds allocation in v1 — a shortfall means
// the holds don't cover, which raises STOCK_MISMATCH instead of improvising.
// (Future routing replaces this whole function; the interface stays.)
func allocateByPriority(
	view *model.FulfillmentOrderView,
	uncovered []model.ShipmentItemRequest,
	variantByItem map[uint]*uint,
	rows []model.AvailabilityRow,
) ([]allocation, error) {
	byVariant := map[uint][]model.AvailabilityRow{}
	for _, row := range rows {
		byVariant[row.VariantID] = append(byVariant[row.VariantID], row)
	}
	var allocations []allocation
	for _, line := range uncovered {
		variant := variantByItem[line.OrderItemID]
		if variant == nil {
			return nil, fmt.Errorf("%w: line %d has no variant linkage",
				fulfillmenterrors.ErrorStockMismatch, line.OrderItemID)
		}
		need := line.Quantity
		for _, row := range byVariant[*variant] {
			if need <= 0 {
				break
			}
			take := row.HeldQty
			if take <= 0 {
				continue
			}
			if take > need {
				take = need
			}
			allocations = append(allocations, allocation{
				orderItemID: line.OrderItemID,
				variantID:   variant,
				locationID:  row.LocationID,
				quantity:    take,
			})
			need -= take
		}
		if need > 0 {
			return nil, fmt.Errorf("%w: order %d variant %d short by %d",
				fulfillmenterrors.ErrorStockMismatch, view.OrderID, *variant, need)
		}
	}
	return allocations, nil
}

// adoptLines shapes allocations for the adopt-only reservation check.
func adoptLines(orderID uint, allocations []allocation) []model.ReservationLine {
	lines := make([]model.ReservationLine, 0, len(allocations))
	for _, a := range allocations {
		lines = append(lines, model.ReservationLine{
			OrderID:    orderID,
			VariantID:  a.variantID,
			LocationID: a.locationID,
			Quantity:   a.quantity,
		})
	}
	return lines
}

// insertDrafts writes one draft per warehouse with items, a draft ledger
// row each, and COD stamped on exactly one new box (the best-priority one)
// when no live box already carries it.
func (p *ShipmentPlannerImpl) insertDrafts(
	ctx context.Context,
	view *model.FulfillmentOrderView,
	allocations []allocation,
	variantByItem map[uint]*uint,
	measures map[uint]*draftMeasure,
) ([]uint, error) {
	byLocation := map[uint][]allocation{}
	var locationOrder []uint
	for _, a := range allocations {
		if _, ok := byLocation[a.locationID]; !ok {
			locationOrder = append(locationOrder, a.locationID)
		}
		byLocation[a.locationID] = append(byLocation[a.locationID], a)
	}

	codRemaining := view.CodCents
	if codRemaining > 0 {
		carried, err := p.liveCODTotal(ctx, view.OrderID)
		if err != nil {
			return nil, err
		}
		if carried > 0 {
			codRemaining = 0
		}
	}

	var draftIDs []uint
	for i, locationID := range locationOrder {
		id, err := p.createDraftBox(ctx, view, locationID, i == 0, codRemaining, byLocation[locationID], measures[locationID])
		if err != nil {
			return nil, err
		}
		draftIDs = append(draftIDs, id)
	}
	return draftIDs, nil
}

// createDraftBox writes one draft box with its items plus the draft ledger
// row. firstBox carries the order's COD remainder; measures stamp weight
// and dims when the catalog provided them.
func (p *ShipmentPlannerImpl) createDraftBox(
	ctx context.Context,
	view *model.FulfillmentOrderView,
	locationID uint,
	firstBox bool,
	codRemaining int64,
	lines []allocation,
	measure *draftMeasure,
) (uint, error) {
	shipment := &entity.FulfillmentShipment{
		OrderID:                  view.OrderID,
		SellerID:                 view.SellerID,
		Status:                   entity.SHIPMENT_STATUS_DRAFT,
		PickupLocationID:         &locationID,
		DeliveryAddressID:        uintPtr(view.DeliveryAddressID),
		DeliveryAddressRevisedAt: &view.DeliveryAddressUpdatedAt,
		RawRef:                   map[string]any{},
	}
	if firstBox {
		shipment.CodCents = codRemaining
	}
	measure.stamp(shipment)
	if err := p.shipmentRepo.Create(ctx, shipment); err != nil {
		return 0, fmt.Errorf("planner insert draft: %w", err)
	}
	for _, a := range lines {
		if err := p.itemRepo.Create(ctx, &entity.FulfillmentShipmentItem{
			ShipmentID:  shipment.ID,
			OrderItemID: a.orderItemID,
			Quantity:    a.quantity,
		}); err != nil {
			return 0, fmt.Errorf("planner insert draft item: %w", err)
		}
	}
	if err := p.eventRepo.Create(ctx, &entity.FulfillmentShipmentEvent{
		ShipmentID: shipment.ID,
		EventType:  string(entity.SHIPMENT_STATUS_DRAFT),
		ToStatus:   string(entity.SHIPMENT_STATUS_DRAFT),
		Source:     "system",
	}); err != nil {
		return 0, fmt.Errorf("planner draft ledger: %w", err)
	}
	return shipment.ID, nil
}

// liveCODTotal sums COD across live (non-cancelled) boxes of the order.
func (p *ShipmentPlannerImpl) liveCODTotal(ctx context.Context, orderID uint) (int64, error) {
	shipments, err := p.shipmentRepo.FindByOrderID(ctx, orderID)
	if err != nil {
		return 0, err
	}
	var total int64
	for _, shipment := range shipments {
		if shipment.Status == entity.SHIPMENT_STATUS_CANCELLED {
			continue
		}
		total += shipment.CodCents
	}
	return total, nil
}

func uintPtr(v uint) *uint {
	return &v
}

func intPtr(v int) *int {
	return &v
}

// checkPlannable enforces the plan gates shared by the planner and manual
// draft creation: confirmed directship orders only. Pending orders and
// bopis/transfer/delivery types refuse with INVALID_STATE.
func checkPlannable(view *model.FulfillmentOrderView) error {
	if view.FulfillmentType != fulfillmentTypeDirectship {
		return fulfillmenterrors.ErrorFulfillmentInvalidState.WithMessagef(
			"planner handles directship only, got %s", view.FulfillmentType)
	}
	if view.Status != orderStatusConfirmed {
		return fulfillmenterrors.ErrorFulfillmentInvalidState.WithMessagef(
			"order must be confirmed to plan shipments")
	}
	return nil
}
