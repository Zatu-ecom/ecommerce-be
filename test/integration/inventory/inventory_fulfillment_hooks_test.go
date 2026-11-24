package inventory_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"ecommerce-be/common/db"
	fulfillmentmodel "ecommerce-be/fulfillment/model"
	invEntity "ecommerce-be/inventory/entity"
	invModel "ecommerce-be/inventory/model"
	"ecommerce-be/inventory/repository"
	invService "ecommerce-be/inventory/service"
	"ecommerce-be/test/integration/setup"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// stubPincodeResolver maps warehouse address ids to pincodes (mirrors the
// mock seed addresses: 10 → 95112, 11 → 94102).
type stubPincodeResolver struct {
	pincodes map[uint]string
}

func (s *stubPincodeResolver) GetPincode(_ context.Context, addressID uint, _ uint) (string, error) {
	pincode, ok := s.pincodes[addressID]
	if !ok {
		return "", fmt.Errorf("test stub: unknown address %d", addressID)
	}
	return pincode, nil
}

// fakeManageService stands in for InventoryManageService: it records bulk
// calls so expiry tests prove WHICH rows were released without wiring the
// full manage stack.
type fakeManageService struct {
	calls [][]invModel.ManageInventoryRequest
}

func (f *fakeManageService) ManageInventory(
	_ context.Context,
	req invModel.ManageInventoryRequest,
	_ uint,
	_ uint,
) (*invModel.ManageInventoryResponse, error) {
	return &invModel.ManageInventoryResponse{}, nil
}

func (f *fakeManageService) BulkManageInventory(
	_ context.Context,
	req invModel.BulkManageInventoryRequest,
	_ uint,
	_ uint,
) (*invModel.BulkManageInventoryResponse, error) {
	f.calls = append(f.calls, req.Items)
	return &invModel.BulkManageInventoryResponse{SuccessCount: len(req.Items)}, nil
}

// InventoryFulfillmentHooksSuite covers the fulfillment seam: availability,
// adopt/release, once-only restock, and the T021 expiry guard.
type InventoryFulfillmentHooksSuite struct {
	suite.Suite
	container *setup.TestContainer
	hooks     *invService.InventoryFulfillmentHooksImpl
	invRepo   repository.InventoryRepository
	resRepo   repository.InventoryReservationRepository
	txnRepo   repository.InventoryTransactionRepository
	locRepo   repository.LocationRepository
	manage    *fakeManageService
	resSvc    *invService.InventoryReservationServiceImpl
	seq       uint
	base      uint
}

func (s *InventoryFulfillmentHooksSuite) SetupSuite() {
	s.container = setup.SetupTestContainers(s.T())
	s.container.RunAllMigrations(s.T())
	s.container.RunAllSeeds(s.T())
	db.SetDB(s.container.DB)
	s.base = uint(time.Now().UnixNano() % 400000)

	s.invRepo = repository.NewInventoryRepository()
	s.resRepo = repository.NewInventoryReservationRepository()
	s.txnRepo = repository.NewInventoryTransactionRepository()
	s.locRepo = repository.NewLocationRepository()
	s.manage = &fakeManageService{}
	s.hooks = invService.NewInventoryFulfillmentHooks(
		s.invRepo, s.resRepo, s.txnRepo, s.locRepo,
		&stubPincodeResolver{pincodes: map[uint]string{10: "95112", 11: "94102"}},
	)
	s.resSvc = invService.NewInventoryReservationService(
		s.resRepo, s.invRepo,
		invService.NewInventoryQueryServiceImpl(s.invRepo, s.locRepo),
		nil, nil, nil, s.manage,
	)
}

func (s *InventoryFulfillmentHooksSuite) TearDownSuite() {
	s.container.Cleanup(s.T())
}

func TestInventoryFulfillmentHooksSuite(t *testing.T) {
	suite.Run(t, new(InventoryFulfillmentHooksSuite))
}

// ─── Fixtures ────────────────────────────────────────────────────────────────
// nextVariant mints collision-free variant ids so reruns against a
// persistent database never hit the (variant, location) unique index.
func (s *InventoryFulfillmentHooksSuite) nextVariant() uint {
	s.seq++
	return 500000 + s.base + s.seq
}

// makeHolding creates one inventory row plus one reservation row against it.
func (s *InventoryFulfillmentHooksSuite) makeHolding(
	variant, location uint,
	qty, reserved int,
	reference uint,
	status invEntity.ReservationStatus,
) (invID uint) {
	t := s.T()
	ctx := context.Background()
	inv := &invEntity.Inventory{
		VariantID:        variant,
		LocationID:       location,
		Quantity:         qty,
		ReservedQuantity: reserved,
	}
	require.NoError(t, s.container.DB.WithContext(ctx).Create(inv).Error)
	require.NoError(t, s.container.DB.WithContext(ctx).Create(&invEntity.InventoryReservation{
		InventoryID: inv.ID,
		ReferenceID: reference,
		Quantity:    uint(reserved),
		Status:      status,
		ExpiresAt:   time.Now().UTC().Add(time.Hour),
	}).Error)
	return inv.ID
}

func (s *InventoryFulfillmentHooksSuite) readInventory(invID uint) invEntity.Inventory {
	t := s.T()
	rows, err := s.invRepo.FindByIDs(context.Background(), []uint{invID})
	require.NoError(t, err)
	require.Len(t, rows, 1)
	return rows[0]
}

// ─── GetAvailability: priority-ordered rows with pincodes ────────────────────
// Scenario: stock spread over warehouse (pri 1, 95112) and store (pri 2,
// 94102). Fully reserved rows are excluded; rows sort warehouse-first.
func (s *InventoryFulfillmentHooksSuite) TestGetAvailability_PriorityOrderedRows() {
	t := s.T()
	v1, v2 := s.nextVariant(), s.nextVariant()
	s.makeHolding(v1, 1, 10, 0, 7101, invEntity.ResConfirmed)
	s.makeHolding(v1, 2, 5, 0, 7102, invEntity.ResConfirmed)
	s.makeHolding(v2, 1, 2, 2, 7103, invEntity.ResConfirmed) // available 0 → excluded

	rows, err := s.hooks.GetAvailability(context.Background(), 2, 0, []uint{v1, v2})
	require.NoError(t, err)
	require.Len(t, rows, 2)
	require.Equal(t, uint(1), rows[0].LocationID)
	require.Equal(t, 10, rows[0].AvailableQty)
	require.Equal(t, "95112", rows[0].Pincode)
	require.Equal(t, uint(2), rows[1].LocationID)
	require.Equal(t, 5, rows[1].AvailableQty)
	require.Equal(t, "94102", rows[1].Pincode)
}

// ─── ReserveForShipment adopts covered lines ──────────────────────────────────
// Scenario: CONFIRMED hold of 4 covers a line of 3 → nil, nothing created.
func (s *InventoryFulfillmentHooksSuite) TestReserveForShipment_AdoptsCovered() {
	t := s.T()
	v := s.nextVariant()
	invID := s.makeHolding(v, 1, 10, 4, 7201, invEntity.ResConfirmed)

	err := s.hooks.ReserveForShipment(context.Background(), 2, []fulfillmentmodel.ReservationLine{
		{OrderID: 7201, VariantID: &v, LocationID: 1, Quantity: 3, ShipmentID: 21},
	})
	require.NoError(t, err)
	// Adopt-only: the hold row is untouched.
	require.Equal(t, 4, s.readInventory(invID).ReservedQuantity)
}

// ─── ReserveForShipment shortfall is loud ────────────────────────────────────
// Scenario: line of 5 against a hold of 4 → stock-mismatch error, no partial.
func (s *InventoryFulfillmentHooksSuite) TestReserveForShipment_ShortfallErrors() {
	t := s.T()
	v := s.nextVariant()
	s.makeHolding(v, 1, 10, 4, 7202, invEntity.ResConfirmed)

	err := s.hooks.ReserveForShipment(context.Background(), 2, []fulfillmentmodel.ReservationLine{
		{OrderID: 7202, VariantID: &v, LocationID: 1, Quantity: 5, ShipmentID: 22},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "short by 1")
}

// ─── ReleaseReservation restores and is idempotent ───────────────────────────
// Scenario: release 3 of a 4-hold → reserved drops, remainder stays
// CONFIRMED; release the rest → row CANCELLED; release again → no-op.
func (s *InventoryFulfillmentHooksSuite) TestReleaseReservation_RestoresAndIdempotent() {
	t := s.T()
	ctx := context.Background()
	v := s.nextVariant()
	invID := s.makeHolding(v, 1, 10, 4, 7301, invEntity.ResConfirmed)

	release := func(qty int) {
		t.Helper()
		require.NoError(t, s.hooks.ReleaseReservation(ctx, 2, []fulfillmentmodel.ReservationLine{
			{OrderID: 7301, VariantID: &v, Quantity: qty},
		}))
	}
	release(3)
	require.Equal(t, 1, s.readInventory(invID).ReservedQuantity)

	release(1)
	require.Equal(t, 0, s.readInventory(invID).ReservedQuantity)
	rows, err := s.resRepo.FindByReferenceIDAndStatus(ctx, invEntity.ResCancelled, 7301)
	require.NoError(t, err)
	require.Len(t, rows, 1)

	release(1) // nothing left held → no-op, stays non-negative
	require.Equal(t, 0, s.readInventory(invID).ReservedQuantity)
}

// ─── RestockForReturn restocks exactly once ──────────────────────────────────
// Scenario: a FULFILLED hold of 2 restocks +2 on the first call with a
// transaction marker; the second identical call changes nothing.
func (s *InventoryFulfillmentHooksSuite) TestRestockForReturn_ExactlyOnce() {
	t := s.T()
	ctx := context.Background()
	v := s.nextVariant()
	invID := s.makeHolding(v, 1, 8, 2, 7401, invEntity.ResFulfilled)

	lines := []fulfillmentmodel.RestockLine{{VariantID: &v, Quantity: 2}}
	require.NoError(t, s.hooks.RestockForReturn(ctx, 2, 7401, lines, 31))
	require.Equal(t, 10, s.readInventory(invID).Quantity)

	markers, err := s.txnRepo.FindByReferenceID(ctx, "fulfillment-return-31")
	require.NoError(t, err)
	require.Len(t, markers, 1)

	require.NoError(t, s.hooks.RestockForReturn(ctx, 2, 7401, lines, 31))
	require.Equal(t, 10, s.readInventory(invID).Quantity)
	markers, err = s.txnRepo.FindByReferenceID(ctx, "fulfillment-return-31")
	require.NoError(t, err)
	require.Len(t, markers, 1)
}

// ─── T021: expiry skips non-pending rows ─────────────────────────────────────
// Scenario: one CONFIRMED and one PENDING row share an expiry batch. The
// CONFIRMED row (and its stock) is untouched; the PENDING row expires and
// its release is handed to the manage path exactly once.
func (s *InventoryFulfillmentHooksSuite) TestExpireScheduleReservation_SkipsConfirmed() {
	t := s.T()
	ctx := context.Background()
	v := s.nextVariant()
	confirmedInv := s.makeHolding(v, 1, 10, 4, 7501, invEntity.ResConfirmed)

	pendingInvID := s.makeHolding(v, 2, 10, 3, 7501, invEntity.ResPending)
	var pendingResID uint
	require.NoError(t, s.container.DB.WithContext(ctx).
		Model(&invEntity.InventoryReservation{}).Select("id").
		Where("reference_id = ? AND status = ?", 7501, invEntity.ResPending).
		Scan(&pendingResID).Error)

	var confirmedResID uint
	require.NoError(t, s.container.DB.WithContext(ctx).
		Model(&invEntity.InventoryReservation{}).Select("id").
		Where("reference_id = ? AND status = ?", 7501, invEntity.ResConfirmed).
		Scan(&confirmedResID).Error)

	s.manage.calls = nil
	require.NoError(t, s.resSvc.ExpireScheduleReservation(ctx, 2,
		invModel.ReservationExpiryPayload{ReservationIDs: []uint{confirmedResID, pendingResID}}))

	// CONFIRMED row untouched.
	confirmed, err := s.resRepo.FindByID(ctx, confirmedResID)
	require.NoError(t, err)
	require.Equal(t, invEntity.ResConfirmed, confirmed.Status)
	require.Equal(t, 4, s.readInventory(confirmedInv).ReservedQuantity)

	// PENDING row expired and handed to release exactly once.
	pending, err := s.resRepo.FindByID(ctx, pendingResID)
	require.NoError(t, err)
	require.Equal(t, invEntity.ResExpired, pending.Status)
	require.Len(t, s.manage.calls, 1)
	_ = pendingInvID
}
