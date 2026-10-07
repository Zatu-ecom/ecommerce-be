package order_test

import (
	"context"
	"strconv"
	"testing"
	"time"

	"ecommerce-be/common/db"
	fulfillmentmodel "ecommerce-be/fulfillment/model"
	orderEntity "ecommerce-be/order/entity"
	"ecommerce-be/order/repository"
	orderService "ecommerce-be/order/service"
	"ecommerce-be/test/integration/setup"
	userModel "ecommerce-be/user/model"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// stubInventoryStock records hook-driven inventory calls without touching
// stock. Real stock behavior is covered in the inventory suite.
type stubInventoryStock struct {
	released [][]fulfillmentmodel.ReservationLine
	restocks []restockCall
}

type restockCall struct {
	orderID          uint
	lines            []fulfillmentmodel.RestockLine
	returnShipmentID uint
}

func (s *stubInventoryStock) ReleaseReservation(
	_ context.Context,
	_ uint,
	lines []fulfillmentmodel.ReservationLine,
) error {
	s.released = append(s.released, lines)
	return nil
}

func (s *stubInventoryStock) RestockForReturn(
	_ context.Context,
	_ uint,
	orderID uint,
	lines []fulfillmentmodel.RestockLine,
	returnShipmentID uint,
) error {
	s.restocks = append(s.restocks, restockCall{orderID: orderID, lines: lines, returnShipmentID: returnShipmentID})
	return nil
}

func (s *stubInventoryStock) FulfillHolds(
	_ context.Context,
	_ uint,
	_ uint,
	_ []fulfillmentmodel.ReservationLine,
) error {
	return nil
}

func (s *stubInventoryStock) UnfulfillHolds(
	_ context.Context,
	_ uint,
	_ uint,
	_ []fulfillmentmodel.ReservationLine,
) error {
	return nil
}

// stubCurrencyResolver returns a fixed display currency.
type stubCurrencyResolver struct{ code string }

func (s *stubCurrencyResolver) GetSellerDefaultCurrency(
	_ context.Context,
	_ uint,
) (*userModel.CurrencyResponse, error) {
	return &userModel.CurrencyResponse{
		CurrencyBase: userModel.CurrencyBase{Code: s.code},
	}, nil
}
// against real order rows. Fixtures are inserted directly (hook methods have
// no HTTP surface yet); observable order status is asserted via the repo,
// mirroring what the customer GET would return.
// OrderFulfillmentHooksSuite exercises the fulfillment hook implementation
// against real order rows. Fixtures are inserted directly (hook methods have
// no HTTP surface yet); observable order status is asserted via the repo,
// mirroring what the customer GET would return.
type OrderFulfillmentHooksSuite struct {
	suite.Suite
	container *setup.TestContainer
	hooks     *orderService.OrderFulfillmentHooksImpl
	stock     *stubInventoryStock
	orderRepo repository.OrderRepository
	base      int64
	seq       int64
}

func (s *OrderFulfillmentHooksSuite) SetupSuite() {
	s.container = setup.SetupTestContainers(s.T())
	s.container.RunAllMigrations(s.T())
	s.container.RunAllSeeds(s.T())
	db.SetDB(s.container.DB)

	s.base = time.Now().UnixNano() % 400000
	s.orderRepo = repository.NewOrderRepository()
	s.stock = &stubInventoryStock{}
	s.hooks = orderService.NewOrderFulfillmentHooks(
		s.orderRepo,
		repository.NewOrderHistoryRepository(),
		s.stock,
		&stubCurrencyResolver{code: "INR"},
		nil,
	)
}

func (s *OrderFulfillmentHooksSuite) TearDownSuite() {
	s.container.Cleanup(s.T())
}

func TestOrderFulfillmentHooksSuite(t *testing.T) {
	suite.Run(t, new(OrderFulfillmentHooksSuite))
}

var fulfillmentTestSellerID = uint(2)
var fulfillmentTestUserID = uint(3)

// seedOrder inserts a minimal confirmed order with one item + one shipping
// address. PaidAt nil means unpaid (COD semantics in the view). Order
// numbers and variant ids derive from a run-unique base so reruns against a
// persistent database never collide.
func (s *OrderFulfillmentHooksSuite) seedOrder(paid bool) uint {
	t := s.T()
	ctx := context.Background()
	sellerID := fulfillmentTestSellerID
	now := time.Now().UTC()
	s.seq++
	orderNum := "FULF-HOOK-" + strconv.FormatInt(now.UnixNano(), 10) + "-" + strconv.FormatInt(s.seq, 10)
	order := &orderEntity.Order{
		UserID:          fulfillmentTestUserID,
		SellerID:        &sellerID,
		OrderNumber:     orderNum,
		Status:          orderEntity.ORDER_STATUS_CONFIRMED,
		SubtotalCents:   50000,
		TotalCents:      50000,
		Metadata:        db.JSONMap{},
		FulfillmentType: orderEntity.DIRECTSHIP,
	}
	if paid {
		order.PaidAt = &now
	}
	require.NoError(t, s.container.DB.WithContext(ctx).Create(order).Error)

	variantID := uint(4) // seeded mock variant (order_item.variant_id is a real FK)
	require.NoError(t, s.container.DB.WithContext(ctx).Create(&orderEntity.OrderItem{
		OrderID:        order.ID,
		VariantID:      &variantID,
		ProductName:    "Hook Test Item",
		Quantity:       2,
		UnitPriceCents: 25000,
		LineTotalCents: 50000,
		Attributes:     db.JSONMap{},
	}).Error)
	require.NoError(t, s.container.DB.WithContext(ctx).Create(&orderEntity.OrderAddress{
		OrderID:   order.ID,
		Type:      orderEntity.ORDER_ADDR_SHIPPING,
		Address:   "1 Hook Street",
		City:      "San Jose",
		State:     "CA",
		ZipCode:   "95112",
		CountryID: 1,
	}).Error)
	return order.ID
}

// ─── GetOrderForFulfillment: unpaid order exposes COD ─────────────────────────
// Scenario: planner input for an unpaid (COD) order carries the collect
// amount, item variant linkage, and the delivery snapshot source.
func (s *OrderFulfillmentHooksSuite) TestGetOrderForFulfillment_UnpaidExposesCOD() {
	t := s.T()
	view, err := s.hooks.GetOrderForFulfillment(context.Background(), s.seedOrder(false))
	require.NoError(t, err)
	require.Equal(t, int64(50000), view.CodCents)
	require.Equal(t, "directship", view.FulfillmentType)
	require.Len(t, view.Items, 1)
	require.Equal(t, 2, view.Items[0].Quantity)
	require.NotNil(t, view.Items[0].VariantID)
	require.NotZero(t, view.DeliveryAddressID)
	require.Equal(t, "95112", view.DeliveryPincode)
	require.False(t, view.DeliveryAddressUpdatedAt.IsZero())
}

// ─── GetOrderForFulfillment: paid order collects nothing ──────────────────────
// Scenario: a prepaid order (PaidAt set) yields CodCents 0.
func (s *OrderFulfillmentHooksSuite) TestGetOrderForFulfillment_PaidCollectsNothing() {
	t := s.T()
	view, err := s.hooks.GetOrderForFulfillment(context.Background(), s.seedOrder(true))
	require.NoError(t, err)
	require.Equal(t, int64(0), view.CodCents)
}

// ─── OnShipmentDelivered completes + is idempotent ───────────────────────────
// Scenario: delivered box on a confirmed order moves it to completed with
// one history row; a replay changes nothing.
func (s *OrderFulfillmentHooksSuite) TestOnShipmentDelivered_CompletesIdempotent() {
	t := s.T()
	ctx := context.Background()
	orderID := s.seedOrder(false)
	progress := fulfillmentmodel.FulfillmentProgress{OrderID: orderID, ShipmentID: 11}

	require.NoError(t, s.hooks.OnShipmentDelivered(ctx, progress))
	order, err := s.orderRepo.FindOrderByID(ctx, orderID)
	require.NoError(t, err)
	require.Equal(t, orderEntity.ORDER_STATUS_COMPLETED, order.Status)

	require.NoError(t, s.hooks.OnShipmentDelivered(ctx, progress))
	order, err = s.orderRepo.FindOrderByID(ctx, orderID)
	require.NoError(t, err)
	require.Equal(t, orderEntity.ORDER_STATUS_COMPLETED, order.Status)
}

// ─── OnShipmentFailed cancels and releases ────────────────────────────────────
// Scenario: failed box on a confirmed order cancels it with the reason
// noted and releases the box holds with variant linkage.
func (s *OrderFulfillmentHooksSuite) TestOnShipmentFailed_CancelsAndReleases() {
	t := s.T()
	ctx := context.Background()
	orderID := s.seedOrder(false)

	var itemID uint
	require.NoError(t, s.container.DB.WithContext(ctx).
		Model(&orderEntity.OrderItem{}).Select("id").
		Where("order_id = ?", orderID).Scan(&itemID).Error)

	s.stock.released = nil
	progress := fulfillmentmodel.FulfillmentProgress{
		OrderID: orderID, ShipmentID: 12,
		Items:  []fulfillmentmodel.FulfillmentLine{{OrderItemID: itemID, Quantity: 2}},
		Reason: "lost",
	}
	require.NoError(t, s.hooks.OnShipmentFailed(ctx, progress))

	order, err := s.orderRepo.FindOrderByID(ctx, orderID)
	require.NoError(t, err)
	require.Equal(t, orderEntity.ORDER_STATUS_CANCELLED, order.Status)

	require.Len(t, s.stock.released, 1)
	require.Len(t, s.stock.released[0], 1)
	require.Equal(t, orderID, s.stock.released[0][0].OrderID)
	require.Equal(t, 2, s.stock.released[0][0].Quantity)
	require.NotNil(t, s.stock.released[0][0].VariantID)
}

// ─── OnShipmentFailed on completed order leaves it ───────────────────────────
// Scenario: a late failure on an already-completed order changes nothing
// (the refund flow owns completed money).
func (s *OrderFulfillmentHooksSuite) TestOnShipmentFailed_CompletedLeftAlone() {
	t := s.T()
	ctx := context.Background()
	orderID := s.seedOrder(false)
	require.NoError(t, s.hooks.OnShipmentDelivered(ctx,
		fulfillmentmodel.FulfillmentProgress{OrderID: orderID, ShipmentID: 13}))

	s.stock.released = nil
	require.NoError(t, s.hooks.OnShipmentFailed(ctx,
		fulfillmentmodel.FulfillmentProgress{OrderID: orderID, ShipmentID: 14, Reason: "lost"}))
	order, err := s.orderRepo.FindOrderByID(ctx, orderID)
	require.NoError(t, err)
	require.Equal(t, orderEntity.ORDER_STATUS_COMPLETED, order.Status)
	require.Empty(t, s.stock.released)
}

// ─── OnShipmentReturned restocks via the return box ──────────────────────────
// Scenario: returned box on a completed order marks it returned and restocks
// exactly the box lines, keyed by the return shipment id.
func (s *OrderFulfillmentHooksSuite) TestOnShipmentReturned_RestocksByReturnBox() {
	t := s.T()
	ctx := context.Background()
	orderID := s.seedOrder(false)
	require.NoError(t, s.hooks.OnShipmentDelivered(ctx,
		fulfillmentmodel.FulfillmentProgress{OrderID: orderID, ShipmentID: 15}))

	var itemID uint
	require.NoError(t, s.container.DB.WithContext(ctx).
		Model(&orderEntity.OrderItem{}).Select("id").
		Where("order_id = ?", orderID).Scan(&itemID).Error)

	s.stock.restocks = nil
	progress := fulfillmentmodel.FulfillmentProgress{
		OrderID: orderID, ShipmentID: 16,
		Items:  []fulfillmentmodel.FulfillmentLine{{OrderItemID: itemID, Quantity: 2}},
		Reason: "rto_delivered",
	}
	require.NoError(t, s.hooks.OnShipmentReturned(ctx, progress))

	order, err := s.orderRepo.FindOrderByID(ctx, orderID)
	require.NoError(t, err)
	require.Equal(t, orderEntity.ORDER_STATUS_RETURNED, order.Status)
	require.Len(t, s.stock.restocks, 1)
	require.Equal(t, uint(16), s.stock.restocks[0].returnShipmentID)
	require.Len(t, s.stock.restocks[0].lines, 1)
}
