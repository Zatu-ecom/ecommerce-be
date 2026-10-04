package service

import (
	"context"
	"fmt"
	"sort"

	"ecommerce-be/common/db"
	"ecommerce-be/common/log"
	fulfillmenterrors "ecommerce-be/fulfillment/error"
	fulfillmentmodel "ecommerce-be/fulfillment/model"
	invEntity "ecommerce-be/inventory/entity"
	"ecommerce-be/inventory/repository"
	userService "ecommerce-be/user/service"

	"gorm.io/gorm"
)

// PincodeResolver narrows the user address surface this file needs: pickup
// pincodes for warehouse locations. Production wires the real address
// service (precedent: location_service already depends on it); tests stub it.
// It returns the pincode string (not the address shape) so external test
// packages can implement it.
type PincodeResolver interface {
	GetPincode(ctx context.Context, addressID uint, userID uint) (string, error)
}

// InventoryFulfillmentHooksImpl implements fulfillment's
// FulfillmentInventoryHooks plus the return-restock used by order hooks.
// Same-module repo access only (inventory tables); user addresses resolve
// through the user service, never direct reads.
//
// Adopt-only rule: checkout created the CONFIRMED hold; nothing here ever
// creates a reservation row. ReserveForShipment verifies coverage,
// ReleaseReservation returns holds, RestockForReturn puts shipped units back.
type InventoryFulfillmentHooksImpl struct {
	inventoryRepo   repository.InventoryRepository
	reservationRepo repository.InventoryReservationRepository
	txnRepo         repository.InventoryTransactionRepository
	locationRepo    repository.LocationRepository
	addressSvc      PincodeResolver
}

// NewInventoryFulfillmentHooks builds the fulfillment hook implementation.
func NewInventoryFulfillmentHooks(
	inventoryRepo repository.InventoryRepository,
	reservationRepo repository.InventoryReservationRepository,
	txnRepo repository.InventoryTransactionRepository,
	locationRepo repository.LocationRepository,
	addressSvc PincodeResolver,
) *InventoryFulfillmentHooksImpl {
	return &InventoryFulfillmentHooksImpl{
		inventoryRepo:   inventoryRepo,
		reservationRepo: reservationRepo,
		txnRepo:         txnRepo,
		locationRepo:    locationRepo,
		addressSvc:      addressSvc,
	}
}

// realAddressService adapts userService.AddressService to PincodeResolver.
type realAddressService struct {
	svc userService.AddressService
}

func (r *realAddressService) GetPincode(ctx context.Context, addressID uint, userID uint) (string, error) {
	addr, err := r.svc.GetAddressByID(ctx, addressID, userID)
	if err != nil {
		return "", err
	}
	return addr.ZipCode, nil
}

// NewInventoryFulfillmentHooksWithUserService wires the production address
// service. Prefer this constructor outside tests.
func NewInventoryFulfillmentHooksWithUserService(
	inventoryRepo repository.InventoryRepository,
	reservationRepo repository.InventoryReservationRepository,
	txnRepo repository.InventoryTransactionRepository,
	locationRepo repository.LocationRepository,
	addressSvc userService.AddressService,
) *InventoryFulfillmentHooksImpl {
	return NewInventoryFulfillmentHooks(
		inventoryRepo, reservationRepo, txnRepo, locationRepo,
		&realAddressService{svc: addressSvc},
	)
}

// GetAvailability returns available quantity per (variant, location) with
// warehouse priority and pickup pincode. Locations sort by priority
// ascending (1 first: warehouses before stores, per seed convention).
// Only rows with available stock are returned.
func (h *InventoryFulfillmentHooksImpl) GetAvailability(
	ctx context.Context,
	sellerID uint,
	orderID uint,
	variantIDs []uint,
) ([]fulfillmentmodel.AvailabilityRow, error) {
	if len(variantIDs) == 0 {
		return nil, nil
	}
	locations, err := h.locationRepo.FindActiveByPriority(ctx, sellerID)
	if err != nil {
		return nil, fmt.Errorf("fulfillment hooks list locations: %w", err)
	}
	if len(locations) == 0 {
		return nil, nil
	}
	sort.Slice(locations, func(i, j int) bool {
		if locations[i].Priority == locations[j].Priority {
			return locations[i].ID < locations[j].ID
		}
		return locations[i].Priority < locations[j].Priority
	})

	locationIDs := make([]uint, 0, len(locations))
	for _, loc := range locations {
		locationIDs = append(locationIDs, loc.ID)
	}
	inventories, err := h.inventoryRepo.FindByVariantAndLocationBatch(ctx, variantIDs, locationIDs)
	if err != nil {
		return nil, fmt.Errorf("fulfillment hooks list inventory: %w", err)
	}

	pincodes := map[uint]string{}
	for _, loc := range locations {
		pincode, err := h.addressSvc.GetPincode(ctx, loc.AddressID, sellerID)
		if err != nil {
			// A warehouse without a resolvable address cannot be routed to:
			// skip it loudly rather than planning with an empty pincode.
			log.ErrorWithContext(ctx, "fulfillment hooks: location address unresolvable, skipping",
				fmt.Errorf("location %d: %w", loc.ID, err))
			continue
		}
		pincodes[loc.ID] = pincode
	}

	priorityByLocation := map[uint]int{}
	for _, loc := range locations {
		priorityByLocation[loc.ID] = loc.Priority
	}

	// Own-order holds at each spot (CONFIRMED, keyed by order reference).
	// Availability figures exclude held units, so planners add both.
	heldByInventory := map[uint]int{}
	if orderID != 0 {
		holds, err := h.reservationRepo.FindByReferenceIDAndStatus(ctx, invEntity.ResConfirmed, orderID)
		if err != nil {
			return nil, fmt.Errorf("fulfillment hooks load own holds: %w", err)
		}
		for _, hold := range holds {
			heldByInventory[hold.InventoryID] += int(hold.Quantity)
		}
	}

	var rows []fulfillmentmodel.AvailabilityRow
	for _, inv := range inventories {
		available := inv.Quantity - inv.ReservedQuantity
		held := heldByInventory[inv.ID]
		if available+held <= 0 {
			continue
		}
		if available < 0 {
			available = 0
		}
		pincode, ok := pincodes[inv.LocationID]
		if !ok {
			continue
		}
		rows = append(rows, fulfillmentmodel.AvailabilityRow{
			VariantID:    inv.VariantID,
			LocationID:   inv.LocationID,
			AvailableQty: available,
			HeldQty:      held,
			Priority:     priorityByLocation[inv.LocationID],
			Pincode:      pincode,
			SellerID:     sellerID,
		})
	}
	// Deterministic priority order (warehouses before stores): the planner
	// allocates greedily in this order, so the hook defines it.
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Priority == rows[j].Priority {
			if rows[i].VariantID == rows[j].VariantID {
				return rows[i].LocationID < rows[j].LocationID
			}
			return rows[i].VariantID < rows[j].VariantID
		}
		return rows[i].Priority < rows[j].Priority
	})
	return rows, nil
}

// ReserveForShipment verifies the order's CONFIRMED hold still covers the
// lines (adopt-only; creates nothing). Lines may span one order — grouping
// by order is the caller's job. Missing variant linkage or shortfall is a
// stock-mismatch error, never a partial adoption.
func (h *InventoryFulfillmentHooksImpl) ReserveForShipment(
	ctx context.Context,
	sellerID uint,
	lines []fulfillmentmodel.ReservationLine,
) error {
	byOrder := map[uint][]fulfillmentmodel.ReservationLine{}
	for _, line := range lines {
		if line.VariantID == nil {
			return fmt.Errorf("%w: line without variant linkage",
				fulfillmenterrors.ErrorStockMismatch)
		}
		byOrder[line.OrderID] = append(byOrder[line.OrderID], line)
	}
	for orderID, orderLines := range byOrder {
		if err := h.adoptOrderLines(ctx, orderID, orderLines); err != nil {
			return err
		}
	}
	return nil
}

// adoptOrderLines checks CONFIRMED holds cover one order's lines.
func (h *InventoryFulfillmentHooksImpl) adoptOrderLines(
	ctx context.Context,
	orderID uint,
	lines []fulfillmentmodel.ReservationLine,
) error {
	holds, err := h.reservationRepo.FindByReferenceIDAndStatus(
		ctx, invEntity.ResConfirmed, orderID,
	)
	if err != nil {
		return fmt.Errorf("fulfillment hooks load holds: %w", err)
	}
	invIDs := make([]uint, 0, len(holds))
	for _, hold := range holds {
		invIDs = append(invIDs, hold.InventoryID)
	}
	inventories, err := h.inventoryRepo.FindByIDs(ctx, invIDs)
	if err != nil {
		return fmt.Errorf("fulfillment hooks load hold inventory: %w", err)
	}
	variantByInventory := map[uint]uint{}
	locationByInventory := map[uint]uint{}
	for _, inv := range inventories {
		variantByInventory[inv.ID] = inv.VariantID
		locationByInventory[inv.ID] = inv.LocationID
	}
	held := map[uint]map[uint]int{}
	for _, hold := range holds {
		variant := variantByInventory[hold.InventoryID]
		location := locationByInventory[hold.InventoryID]
		if _, ok := held[variant]; !ok {
			held[variant] = map[uint]int{}
		}
		held[variant][location] += int(hold.Quantity)
	}
	for _, line := range lines {
		need := line.Quantity
		locations := held[*line.VariantID]
		if line.LocationID != 0 {
			need -= locations[line.LocationID]
		} else {
			for _, qty := range locations {
				need -= qty
			}
		}
		if need > 0 {
			return fmt.Errorf("%w: order %d variant %d short by %d",
				fulfillmenterrors.ErrorStockMismatch, orderID, *line.VariantID, need)
		}
	}
	return nil
}

// FulfillHolds consumes CONFIRMED holds into FULFILLED on booking: stock
// leaves available (quantity and reserved both drop) with an OUTBOUND audit
// row. Partial-qty aware; missing coverage errors loudly. Idempotent: lines
// already fully FULFILLED (replay after a crash between commit and hook)
// succeed without moving anything.
func (h *InventoryFulfillmentHooksImpl) FulfillHolds(
	ctx context.Context,
	sellerID uint,
	orderID uint,
	lines []fulfillmentmodel.ReservationLine,
) error {
	return db.WithTransaction(ctx, func(txCtx context.Context) error {
		for _, line := range lines {
			if err := h.consumeHold(txCtx, sellerID, orderID, line, invEntity.ResConfirmed, invEntity.ResFulfilled); err != nil {
				// Replay tolerance: a crash between booking commit and the
				// booked hook leaves FULFILLED rows with no CONFIRMED
				// remainder. If fulfilled coverage already matches, the
				// earlier call did its job — succeed without moving stock.
				if h.fulfilledCovers(txCtx, orderID, line) {
					continue
				}
				return err
			}
		}
		return nil
	})
}

// fulfilledCovers reports whether FULFILLED rows already cover one line.
func (h *InventoryFulfillmentHooksImpl) fulfilledCovers(
	ctx context.Context,
	orderID uint,
	line fulfillmentmodel.ReservationLine,
) bool {
	if line.VariantID == nil {
		return false
	}
	holds, err := h.reservationRepo.FindByReferenceIDAndStatus(ctx, invEntity.ResFulfilled, orderID)
	if err != nil {
		return false
	}
	covered := 0
	for _, hold := range holds {
		inventories, err := h.inventoryRepo.FindByIDs(ctx, []uint{hold.InventoryID})
		if err != nil || len(inventories) == 0 {
			continue
		}
		inv := inventories[0]
		if inv.VariantID != *line.VariantID {
			continue
		}
		if line.LocationID != 0 && inv.LocationID != line.LocationID {
			continue
		}
		covered += int(hold.Quantity)
	}
	return covered >= line.Quantity
}

// UnfulfillHolds returns FULFILLED holds to CONFIRMED when a booked box is
// cancelled pre-pickup (goods never left the warehouse). Restores quantity
// and reserved with an adjustment audit row, so re-planning keeps working.
func (h *InventoryFulfillmentHooksImpl) UnfulfillHolds(
	ctx context.Context,
	sellerID uint,
	orderID uint,
	lines []fulfillmentmodel.ReservationLine,
) error {
	return db.WithTransaction(ctx, func(txCtx context.Context) error {
		for _, line := range lines {
			if err := h.restoreHold(txCtx, sellerID, orderID, line); err != nil {
				return err
			}
		}
		return nil
	})
}

// consumeHold moves take units of one line from one hold state to another,
// adjusting stock down (fulfill path).
func (h *InventoryFulfillmentHooksImpl) consumeHold(
	ctx context.Context,
	sellerID uint,
	orderID uint,
	line fulfillmentmodel.ReservationLine,
	from, to invEntity.ReservationStatus,
) error {
	if line.VariantID == nil {
		return fmt.Errorf("fulfillment hooks: hold line without variant linkage")
	}
	holds, err := h.reservationRepo.FindByReferenceIDAndStatus(ctx, from, orderID)
	if err != nil {
		return fmt.Errorf("fulfillment hooks load holds: %w", err)
	}
	remaining := line.Quantity
	for _, hold := range holds {
		if remaining <= 0 {
			break
		}
		inv, err := h.inventoryByID(ctx, hold.InventoryID)
		if err != nil {
			return err
		}
		if inv.VariantID != *line.VariantID {
			continue
		}
		if line.LocationID != 0 && inv.LocationID != line.LocationID {
			continue
		}
		take := int(hold.Quantity)
		if take > remaining {
			take = remaining
		}
		if err := h.consumeUnits(ctx, sellerID, orderID, inv, take, to); err != nil {
			return err
		}
		remaining -= take
		// Split the hold row when partially consumed: the taken part moves
		// to `to`, the remainder stays behind in `from`. Row quantities
		// always sum to the truth, so restock/unfulfill can consume exactly.
		if take < int(hold.Quantity) {
			remainder := &invEntity.InventoryReservation{
				InventoryID: hold.InventoryID,
				ReferenceID: hold.ReferenceID,
				Quantity:    hold.Quantity - uint(take),
				Status:      from,
				ExpiresAt:   hold.ExpiresAt,
			}
			hold.Quantity = uint(take)
			if err := h.reservationRepo.CreateOrSave(ctx, []*invEntity.InventoryReservation{remainder}); err != nil {
				return fmt.Errorf("fulfillment hooks split hold remainder: %w", err)
			}
		}
		hold.Status = to
		if err := h.reservationRepo.CreateOrSave(ctx, []*invEntity.InventoryReservation{hold}); err != nil {
			return fmt.Errorf("fulfillment hooks persist consumed hold: %w", err)
		}
	}
	if remaining > 0 {
		return fmt.Errorf("%w: order %d holds short",
			fulfillmenterrors.ErrorStockMismatch, orderID)
	}
	return nil
}

// consumeUnits moves take units of stock out (quantity and reserved both
// drop) with an OUTBOUND audit row. Shared by fulfill paths.
func (h *InventoryFulfillmentHooksImpl) consumeUnits(
	ctx context.Context,
	sellerID uint,
	orderID uint,
	inv *invEntity.Inventory,
	take int,
	to invEntity.ReservationStatus,
) error {
	_ = to
	beforeQty, beforeRes := inv.Quantity, inv.ReservedQuantity
	if err := db.DB(ctx).
		Model(&invEntity.Inventory{}).
		Where("id = ?", inv.ID).
		Updates(map[string]any{
			"quantity":          gorm.Expr("quantity - ?", take),
			"reserved_quantity": gorm.Expr("reserved_quantity - ?", take),
		}).Error; err != nil {
		return fmt.Errorf("fulfillment hooks consume stock: %w", err)
	}
	ref := fmt.Sprintf("%d", orderID)
	refType := "ORDER"
	return h.txnRepo.Create(ctx, &invEntity.InventoryTransaction{
		InventoryID:            inv.ID,
		Type:                   invEntity.TXN_OUTBOUND,
		QuantityChange:         -take,
		BeforeQuantity:         beforeQty,
		AfterQuantity:          beforeQty - take,
		ReservedQuantityChange: -take,
		BeforeReservedQuantity: beforeRes,
		AfterReservedQuantity:  beforeRes - take,
		PerformedBy:            sellerID,
		ReferenceID:            &ref,
		ReferenceType:          &refType,
		Reason:                 "fulfillment: box booked, stock outbound",
	})
}

// restoreHold returns FULFILLED units to CONFIRMED (booked-cancel path).
func (h *InventoryFulfillmentHooksImpl) restoreHold(
	ctx context.Context,
	sellerID uint,
	orderID uint,
	line fulfillmentmodel.ReservationLine,
) error {
	if line.VariantID == nil {
		return fmt.Errorf("fulfillment hooks: hold line without variant linkage")
	}
	holds, err := h.reservationRepo.FindByReferenceIDAndStatus(ctx, invEntity.ResFulfilled, orderID)
	if err != nil {
		return fmt.Errorf("fulfillment hooks load fulfilled holds: %w", err)
	}
	remaining := line.Quantity
	for _, hold := range holds {
		if remaining <= 0 {
			break
		}
		inv, err := h.inventoryByID(ctx, hold.InventoryID)
		if err != nil {
			return err
		}
		if inv.VariantID != *line.VariantID {
			continue
		}
		if line.LocationID != 0 && inv.LocationID != line.LocationID {
			continue
		}
		take := int(hold.Quantity)
		if take > remaining {
			take = remaining
		}
		beforeQty, beforeRes := inv.Quantity, inv.ReservedQuantity
		if err := db.DB(ctx).
			Model(&invEntity.Inventory{}).
			Where("id = ?", inv.ID).
			Updates(map[string]any{
				"quantity":          gorm.Expr("quantity + ?", take),
				"reserved_quantity": gorm.Expr("reserved_quantity + ?", take),
			}).Error; err != nil {
			return fmt.Errorf("fulfillment hooks restore stock: %w", err)
		}
		ref := fmt.Sprintf("%d", orderID)
		refType := "ORDER"
		reason := "fulfillment: booked box cancelled pre-pickup, hold restored"
		if err := h.txnRepo.Create(ctx, &invEntity.InventoryTransaction{
			InventoryID:            inv.ID,
			Type:                   invEntity.TXN_ADJUSTMENT,
			QuantityChange:         take,
			BeforeQuantity:         beforeQty,
			AfterQuantity:          beforeQty + take,
			ReservedQuantityChange: take,
			BeforeReservedQuantity: beforeRes,
			AfterReservedQuantity:  beforeRes + take,
			PerformedBy:            sellerID,
			ReferenceID:            &ref,
			ReferenceType:          &refType,
			Reason:                 reason,
		}); err != nil {
			return fmt.Errorf("fulfillment hooks record restore: %w", err)
		}
		hold.Quantity -= uint(take)
		remaining -= take
		if hold.Quantity == 0 {
			hold.Status = invEntity.ResConfirmed
		}
		if err := h.reservationRepo.CreateOrSave(ctx, []*invEntity.InventoryReservation{hold}); err != nil {
			return fmt.Errorf("fulfillment hooks persist restored hold: %w", err)
		}
	}
	if remaining > 0 {
		return fmt.Errorf("%w: order %d fulfilled holds short",
			fulfillmenterrors.ErrorStockMismatch, orderID)
	}
	return nil
}

// ReleaseReservation returns CONFIRMED-held units for the lines (cancel and
// failed-pre-pickup paths). LocationID 0 releases across the order's
// warehouses for the variant. Idempotent: already-released lines find no
// CONFIRMED rows and change nothing.
func (h *InventoryFulfillmentHooksImpl) ReleaseReservation(
	ctx context.Context,
	sellerID uint,
	lines []fulfillmentmodel.ReservationLine,
) error {
	return db.WithTransaction(ctx, func(txCtx context.Context) error {
		for _, line := range lines {
			if err := h.releaseLine(txCtx, sellerID, line); err != nil {
				return err
			}
		}
		return nil
	})
}

func (h *InventoryFulfillmentHooksImpl) releaseLine(
	ctx context.Context,
	sellerID uint,
	line fulfillmentmodel.ReservationLine,
) error {
	if line.VariantID == nil {
		return fmt.Errorf("%w: release line without variant linkage",
			fulfillmenterrors.ErrorStockMismatch)
	}
	holds, err := h.reservationRepo.FindByReferenceIDAndStatus(
		ctx, invEntity.ResConfirmed, line.OrderID,
	)
	if err != nil {
		return fmt.Errorf("fulfillment hooks load holds for release: %w", err)
	}
	remaining := line.Quantity
	for _, hold := range holds {
		if remaining <= 0 {
			break
		}
		inv, err := h.inventoryByID(ctx, hold.InventoryID)
		if err != nil {
			return err
		}
		if inv.VariantID != *line.VariantID {
			continue
		}
		if line.LocationID != 0 && inv.LocationID != line.LocationID {
			continue
		}
		take := int(hold.Quantity)
		if take > remaining {
			take = remaining
		}
		if err := h.restoreReserved(ctx, sellerID, line.OrderID, inv, take); err != nil {
			return err
		}
		hold.Quantity -= uint(take)
		remaining -= take
		if hold.Quantity == 0 {
			hold.Status = invEntity.ResCancelled
		}
		if err := h.reservationRepo.CreateOrSave(ctx, []*invEntity.InventoryReservation{hold}); err != nil {
			return fmt.Errorf("fulfillment hooks persist released hold: %w", err)
		}
	}
	return nil
}

// restoreReserved returns take units to available stock with a RELEASED
// audit row. Atomic decrement; the CHECK (reserved_quantity >= 0) fails
// loudly on underflow instead of silently going negative.
func (h *InventoryFulfillmentHooksImpl) restoreReserved(
	ctx context.Context,
	sellerID, orderID uint,
	inv *invEntity.Inventory,
	take int,
) error {
	before := inv.ReservedQuantity
	err := db.DB(ctx).
		Model(&invEntity.Inventory{}).
		Where("id = ?", inv.ID).
		UpdateColumn("reserved_quantity", gorm.Expr("reserved_quantity - ?", take)).Error
	if err != nil {
		return fmt.Errorf("fulfillment hooks restore reserved: %w", err)
	}
	ref := fmt.Sprintf("%d", orderID)
	refType := "ORDER"
	return h.txnRepo.Create(ctx, &invEntity.InventoryTransaction{
		InventoryID:            inv.ID,
		Type:                   invEntity.TXN_RELEASED,
		QuantityChange:         0,
		BeforeQuantity:         inv.Quantity,
		AfterQuantity:          inv.Quantity,
		ReservedQuantityChange: -take,
		BeforeReservedQuantity: before,
		AfterReservedQuantity:  before - take,
		PerformedBy:            sellerID,
		ReferenceID:            &ref,
		ReferenceType:          &refType,
		Reason:                 "fulfillment: released cancelled/failed box hold",
	})
}

// RestockForReturn puts shipped units back to stock exactly once per return
// box. Idempotency keys on the inventory transaction reference
// (fulfillment-return-{shipmentID}): a double-observed return finds the
// marker and restocks nothing. Consumed FULFILLED rows flip to CANCELLED;
// partially consumed rows keep their remainder as FULFILLED.
func (h *InventoryFulfillmentHooksImpl) RestockForReturn(
	ctx context.Context,
	sellerID uint,
	orderID uint,
	lines []fulfillmentmodel.RestockLine,
	returnShipmentID uint,
) error {
	ref := fmt.Sprintf("fulfillment-return-%d", returnShipmentID)
	return db.WithTransaction(ctx, func(txCtx context.Context) error {
		existing, err := h.txnRepo.FindByReferenceID(txCtx, ref)
		if err != nil {
			return fmt.Errorf("fulfillment hooks check restock marker: %w", err)
		}
		if len(existing) > 0 {
			return nil
		}
		for _, line := range lines {
			if err := h.restockLine(txCtx, sellerID, orderID, ref, line); err != nil {
				return err
			}
		}
		return nil
	})
}

func (h *InventoryFulfillmentHooksImpl) restockLine(
	ctx context.Context,
	sellerID, orderID uint,
	ref string,
	line fulfillmentmodel.RestockLine,
) error {
	if line.VariantID == nil {
		return fmt.Errorf("%w: restock line without variant linkage",
			fulfillmenterrors.ErrorStockMismatch)
	}
	holds, err := h.reservationRepo.FindByReferenceIDAndStatus(
		ctx, invEntity.ResFulfilled, orderID,
	)
	if err != nil {
		return fmt.Errorf("fulfillment hooks load fulfilled holds: %w", err)
	}
	remaining := line.Quantity
	for _, hold := range holds {
		if remaining <= 0 {
			break
		}
		inv, err := h.inventoryByID(ctx, hold.InventoryID)
		if err != nil {
			return err
		}
		if inv.VariantID != *line.VariantID {
			continue
		}
		take := int(hold.Quantity)
		if take > remaining {
			take = remaining
		}
		before := inv.Quantity
		if err := db.DB(ctx).
			Model(&invEntity.Inventory{}).
			Where("id = ?", inv.ID).
			UpdateColumn("quantity", gorm.Expr("quantity + ?", take)).Error; err != nil {
			return fmt.Errorf("fulfillment hooks restock quantity: %w", err)
		}
		refType := "FULFILLMENT_RETURN"
		if err := h.txnRepo.Create(ctx, &invEntity.InventoryTransaction{
			InventoryID:            inv.ID,
			Type:                   invEntity.TXN_RETURN,
			QuantityChange:         take,
			BeforeQuantity:         before,
			AfterQuantity:          before + take,
			ReservedQuantityChange: 0,
			BeforeReservedQuantity: inv.ReservedQuantity,
			AfterReservedQuantity:  inv.ReservedQuantity,
			PerformedBy:            sellerID,
			ReferenceID:            &ref,
			ReferenceType:          &refType,
			Reason:                 "fulfillment: return box restock",
		}); err != nil {
			return fmt.Errorf("fulfillment hooks record restock: %w", err)
		}
		hold.Quantity -= uint(take)
		remaining -= take
		if hold.Quantity == 0 {
			hold.Status = invEntity.ResCancelled
		}
		if err := h.reservationRepo.CreateOrSave(ctx, []*invEntity.InventoryReservation{hold}); err != nil {
			return fmt.Errorf("fulfillment hooks persist restocked hold: %w", err)
		}
	}
	return nil
}

func (h *InventoryFulfillmentHooksImpl) inventoryByID(
	ctx context.Context,
	id uint,
) (*invEntity.Inventory, error) {
	inventories, err := h.inventoryRepo.FindByIDs(ctx, []uint{id})
	if err != nil {
		return nil, fmt.Errorf("fulfillment hooks load inventory: %w", err)
	}
	if len(inventories) == 0 {
		return nil, fmt.Errorf("fulfillment hooks: inventory %d missing", id)
	}
	inv := inventories[0]
	return &inv, nil
}
