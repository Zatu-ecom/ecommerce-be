package CartTest

import (
	"net/http"
	"testing"

	orderEntity "ecommerce-be/order/entity"
	"ecommerce-be/test/integration/helpers"
	"ecommerce-be/test/integration/setup"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"
)

// Default test device ID and seller ID for guest cart tests.
const (
	testDeviceID  = "test-device-uuid-0000-0000-000000000001"
	testDeviceID2 = "test-device-uuid-0000-0000-000000000002"
	testSellerID  = "2"
)

// GuestCartTestSuite exercises guest cart and merge APIs.
// Pattern: follows CartTestSuite (suite, seller/customer/guest clients, cleanup helpers).
type GuestCartTestSuite struct {
	suite.Suite
	container *setup.TestContainer
	server    http.Handler

	// Guest client: no JWT, uses X-Device-ID + X-Seller-ID headers
	guestClient  *helpers.APIClient
	guestClient2 *helpers.APIClient

	// Authenticated clients (for merge tests)
	sellerClient   *helpers.APIClient
	customerClient *helpers.APIClient
}

func (s *GuestCartTestSuite) SetupSuite() {
	s.container = setup.SetupTestContainers(s.T())
	s.container.RunAllMigrations(s.T())
	s.container.RunAllSeeds(s.T())

	s.server = setup.SetupTestServer(s.T(), s.container.DB, s.container.RedisClient)

	// Guest client 1 (device 1)
	s.guestClient = helpers.NewAPIClient(s.server)
	s.guestClient.SetHeader("X-Device-ID", testDeviceID)
	s.guestClient.SetHeader("X-Seller-ID", testSellerID)

	// Guest client 2 (device 2, different device for isolation tests)
	s.guestClient2 = helpers.NewAPIClient(s.server)
	s.guestClient2.SetHeader("X-Device-ID", testDeviceID2)
	s.guestClient2.SetHeader("X-Seller-ID", testSellerID)

	// Seller client
	s.sellerClient = helpers.NewAPIClient(s.server)
	sellerToken := helpers.Login(
		s.T(),
		s.sellerClient,
		helpers.Seller2Email,
		helpers.Seller2Password,
	)
	s.sellerClient.SetToken(sellerToken)

	// Customer client
	s.customerClient = helpers.NewAPIClient(s.server)
	customerToken := helpers.Login(
		s.T(),
		s.customerClient,
		helpers.CustomerEmail,
		helpers.CustomerPassword,
	)
	s.customerClient.SetToken(customerToken)
}

func (s *GuestCartTestSuite) TearDownSuite() {
	if s.container != nil {
		s.container.Cleanup(s.T())
	}
}

func (s *GuestCartTestSuite) SetupTest() {
	s.cleanupAllGuestCarts()
}

func TestGuestCartIntegration(t *testing.T) {
	suite.Run(t, new(GuestCartTestSuite))
}

// ============================================================================
// Cleanup helpers
// ============================================================================

func (s *GuestCartTestSuite) cleanupAllGuestCarts() {
	deviceIDs := []string{testDeviceID, testDeviceID2}
	userIDs := []uint{helpers.CustomerUserID}

	// Clean device-based carts
	var cartIDs []uint
	s.container.DB.Model(&orderEntity.Cart{}).
		Where("device_id IN ?", deviceIDs).
		Pluck("id", &cartIDs)
	if len(cartIDs) > 0 {
		s.Require().NoError(
			s.container.DB.Where("cart_id IN ?", cartIDs).Delete(&orderEntity.CartItem{}).Error,
		)
		s.Require().NoError(
			s.container.DB.Where("id IN ?", cartIDs).Delete(&orderEntity.Cart{}).Error,
		)
	}

	// Clean user-based carts (for merge tests)
	var userCartIDs []uint
	s.container.DB.Model(&orderEntity.Cart{}).
		Where("user_id IN ?", userIDs).
		Pluck("id", &userCartIDs)
	if len(userCartIDs) > 0 {
		s.Require().NoError(
			s.container.DB.Where("cart_id IN ?", userCartIDs).Delete(&orderEntity.CartItem{}).Error,
		)
		s.Require().NoError(
			s.container.DB.Where("id IN ?", userCartIDs).Delete(&orderEntity.Cart{}).Error,
		)
	}
}

// ============================================================================
// Assertion helpers
// ============================================================================

func (s *GuestCartTestSuite) assertGuestCartEnvelope(data map[string]any) {
	assert.NotNil(s.T(), data["id"])
	// Guest cart has no userId (nil)
	assert.Nil(s.T(), data["userId"], "guest cart should have nil userId")
	curr, ok := data["currency"].(map[string]any)
	require.True(s.T(), ok)
	assert.NotEmpty(s.T(), curr["code"])
	assert.NotEmpty(s.T(), curr["symbol"])
	meta, ok := data["metadata"].(map[string]any)
	require.True(s.T(), ok)
	assert.NotNil(s.T(), meta)
}

func (s *GuestCartTestSuite) assertUserCartEnvelope(data map[string]any, expectedUserID uint) {
	assert.NotNil(s.T(), data["id"])
	assert.Equal(s.T(), float64(expectedUserID), data["userId"])
	curr, ok := data["currency"].(map[string]any)
	require.True(s.T(), ok)
	assert.NotEmpty(s.T(), curr["code"])
	assert.NotEmpty(s.T(), curr["symbol"])
	meta, ok := data["metadata"].(map[string]any)
	require.True(s.T(), ok)
	assert.NotNil(s.T(), meta)
}

func (s *GuestCartTestSuite) assertGuestCartItemCount(deviceID string, expected int64) {
	var cart orderEntity.Cart
	err := s.container.DB.Where("device_id = ?", deviceID).First(&cart).Error
	if expected == 0 {
		assert.ErrorIs(s.T(), err, gorm.ErrRecordNotFound)
		return
	}
	require.NoError(s.T(), err)
	var count int64
	err = s.container.DB.Model(&orderEntity.CartItem{}).
		Where("cart_id = ?", cart.ID).
		Count(&count).Error
	require.NoError(s.T(), err)
	assert.Equal(s.T(), expected, count)
}

func (s *GuestCartTestSuite) assertNoDeviceCart(deviceID string) {
	var cart orderEntity.Cart
	err := s.container.DB.Where("device_id = ?", deviceID).First(&cart).Error
	assert.ErrorIs(s.T(), err, gorm.ErrRecordNotFound)
}

func (s *GuestCartTestSuite) assertHasDeviceCart(deviceID string) {
	var cart orderEntity.Cart
	err := s.container.DB.Where("device_id = ?", deviceID).First(&cart).Error
	require.NoError(s.T(), err)
	assert.Equal(s.T(), deviceID, *cart.DeviceID)
}

// ============================================================================
// Helper payload constructors
// ============================================================================

// guestAddItemPayload creates a payload for adding items to guest cart.
// Uses the same pattern as cartItem() in add_to_cart_test.go.
func guestAddItemPayload(variantID, quantity int) map[string]any {
	return map[string]any{
		"items": []any{
			map[string]any{
				"variantId": variantID,
				"quantity":  quantity,
			},
		},
	}
}

// ============================================================================
// HAPPY PATH — Guest Cart Add Item
// ============================================================================

func (s *GuestCartTestSuite) TestHP001GuestAddFirstItem() {
	w := s.guestClient.Post(s.T(), GuestCartItemAPIEndpoint, guestAddItemPayload(1, 1))
	resp := helpers.AssertSuccessResponse(s.T(), w, http.StatusCreated)
	data := resp["data"].(map[string]any)
	s.assertGuestCartEnvelope(data)
	items := data["items"].([]any)
	assert.Len(s.T(), items, 1)
}

func (s *GuestCartTestSuite) TestHP002GuestSameVariantAccumulates() {
	w := s.guestClient.Post(s.T(), GuestCartItemAPIEndpoint, guestAddItemPayload(1, 1))
	helpers.AssertSuccessResponse(s.T(), w, http.StatusCreated)
	w = s.guestClient.Post(s.T(), GuestCartItemAPIEndpoint, guestAddItemPayload(1, 1))
	resp := helpers.AssertSuccessResponse(s.T(), w, http.StatusCreated)
	data := resp["data"].(map[string]any)
	items := data["items"].([]any)
	assert.Len(s.T(), items, 1)
	item := items[0].(map[string]any)
	assert.Equal(s.T(), float64(1), item["variantId"])
	assert.Equal(s.T(), float64(2), item["quantity"])
}

func (s *GuestCartTestSuite) TestHP003GuestTwoDifferentVariants() {
	w := s.guestClient.Post(s.T(), GuestCartItemAPIEndpoint, guestAddItemPayload(1, 1))
	helpers.AssertSuccessResponse(s.T(), w, http.StatusCreated)
	w = s.guestClient.Post(s.T(), GuestCartItemAPIEndpoint, guestAddItemPayload(5, 1))
	resp := helpers.AssertSuccessResponse(s.T(), w, http.StatusCreated)
	data := resp["data"].(map[string]any)
	items := data["items"].([]any)
	assert.Len(s.T(), items, 2)
}

func (s *GuestCartTestSuite) TestHP004GuestQuantityMinBoundary() {
	w := s.guestClient.Post(s.T(), GuestCartItemAPIEndpoint, guestAddItemPayload(2, 1))
	resp := helpers.AssertSuccessResponse(s.T(), w, http.StatusCreated)
	data := resp["data"].(map[string]any)
	items := data["items"].([]any)
	require.Len(s.T(), items, 1)
	item := items[0].(map[string]any)
	assert.Equal(s.T(), float64(2), item["variantId"])
	assert.Equal(s.T(), float64(1), item["quantity"])
	_ = resp
}

func (s *GuestCartTestSuite) TestHP005GuestQuantityMaxBoundary() {
	w := s.guestClient.Post(s.T(), GuestCartItemAPIEndpoint, guestAddItemPayload(2, 50))
	helpers.AssertSuccessResponse(s.T(), w, http.StatusCreated)
}

func (s *GuestCartTestSuite) TestHP006GuestAddItemQuantityZeroNoOp() {
	// Quantity 0 should not create a cart
	w := s.guestClient.Post(s.T(), GuestCartItemAPIEndpoint, guestAddItemPayload(1, 0))
	resp := helpers.AssertSuccessResponse(s.T(), w, http.StatusCreated)
	data := resp["data"].(map[string]any)
	// Should return empty cart with id=0
	assert.Equal(s.T(), float64(0), data["id"])
	s.assertGuestCartEnvelope(data)
	// No device cart should exist
	s.assertGuestCartItemCount(testDeviceID, 0)
}

// ============================================================================
// HAPPY PATH — Guest Get Cart
// ============================================================================

func (s *GuestCartTestSuite) TestHP007GuestGetCartWithItems() {
	w := s.guestClient.Post(s.T(), GuestCartItemAPIEndpoint, guestAddItemPayload(1, 1))
	helpers.AssertSuccessResponse(s.T(), w, http.StatusCreated)

	w = s.guestClient.Get(s.T(), GuestCartAPIEndpoint)
	resp := helpers.AssertSuccessResponse(s.T(), w, http.StatusOK)
	data := resp["data"].(map[string]any)
	s.assertGuestCartEnvelope(data)
	items := data["items"].([]any)
	assert.Len(s.T(), items, 1)
}

func (s *GuestCartTestSuite) TestHP008GuestGetCartEmpty() {
	// No items added — should return empty cart envelope
	w := s.guestClient.Get(s.T(), GuestCartAPIEndpoint)
	resp := helpers.AssertSuccessResponse(s.T(), w, http.StatusOK)
	data := resp["data"].(map[string]any)
	assert.Equal(s.T(), float64(0), data["id"])
	s.assertGuestCartEnvelope(data)
}

// ============================================================================
// HAPPY PATH — Guest Delete Cart
// ============================================================================

func (s *GuestCartTestSuite) TestHP009GuestDeleteCart() {
	w := s.guestClient.Post(s.T(), GuestCartItemAPIEndpoint, guestAddItemPayload(1, 1))
	resp := helpers.AssertSuccessResponse(s.T(), w, http.StatusCreated)
	data := resp["data"].(map[string]any)
	cartID := uint(data["id"].(float64))

	// Delete the cart
	w = s.guestClient.Delete(s.T(), guestCartDeleteURL(cartID))
	resp = helpers.AssertSuccessResponse(s.T(), w, http.StatusOK)

	// Verify cart is deleted from DB
	s.assertGuestCartItemCount(testDeviceID, 0)
	_ = resp
}

// ============================================================================
// NEGATIVE PATH — Guest Cart Auth/Validation
// ============================================================================

func (s *GuestCartTestSuite) TestNEG001NoDeviceID() {
	cl := helpers.NewAPIClient(s.server)
	// No X-Device-ID header, no JWT
	cl.SetHeader("X-Seller-ID", testSellerID)
	w := cl.Post(s.T(), GuestCartItemAPIEndpoint, guestAddItemPayload(1, 1))
	helpers.AssertErrorResponse(s.T(), w, http.StatusBadRequest)
}

func (s *GuestCartTestSuite) TestNEG002EmptyDeviceID() {
	cl := helpers.NewAPIClient(s.server)
	cl.SetHeader("X-Device-ID", "")
	cl.SetHeader("X-Seller-ID", testSellerID)
	w := cl.Post(s.T(), GuestCartItemAPIEndpoint, guestAddItemPayload(1, 1))
	helpers.AssertErrorResponse(s.T(), w, http.StatusBadRequest)
}

func (s *GuestCartTestSuite) TestNEG003NoSellerID() {
	cl := helpers.NewAPIClient(s.server)
	cl.SetHeader("X-Device-ID", testDeviceID)
	// No X-Seller-ID header
	w := cl.Post(s.T(), GuestCartItemAPIEndpoint, guestAddItemPayload(1, 1))
	helpers.AssertErrorResponse(s.T(), w, http.StatusBadRequest)
}

func (s *GuestCartTestSuite) TestNEG004InvalidRequestBody() {
	w := s.guestClient.PostRaw(s.T(), GuestCartItemAPIEndpoint, []byte(`{invalid json`))
	helpers.AssertErrorResponse(s.T(), w, http.StatusBadRequest)
}

func (s *GuestCartTestSuite) TestNEG005MissingItemsField() {
	w := s.guestClient.Post(s.T(), GuestCartItemAPIEndpoint, map[string]any{})
	helpers.AssertErrorResponse(s.T(), w, http.StatusBadRequest)
}

func (s *GuestCartTestSuite) TestNEG006EmptyItemsArray() {
	w := s.guestClient.Post(s.T(), GuestCartItemAPIEndpoint, map[string]any{"items": []any{}})
	helpers.AssertErrorResponse(s.T(), w, http.StatusBadRequest)
}

func (s *GuestCartTestSuite) TestNEG007InvalidVariantID() {
	w := s.guestClient.Post(s.T(), GuestCartItemAPIEndpoint, guestAddItemPayload(0, 1))
	helpers.AssertErrorResponse(s.T(), w, http.StatusBadRequest)
}

func (s *GuestCartTestSuite) TestNEG008InvalidQuantityNegative() {
	w := s.guestClient.Post(s.T(), GuestCartItemAPIEndpoint, guestAddItemPayload(1, -1))
	helpers.AssertErrorResponse(s.T(), w, http.StatusBadRequest)
}

func (s *GuestCartTestSuite) TestNEG009QuantityExceedsMax() {
	w := s.guestClient.Post(s.T(), GuestCartItemAPIEndpoint, guestAddItemPayload(1, 100))
	helpers.AssertErrorResponse(s.T(), w, http.StatusBadRequest)
}

func (s *GuestCartTestSuite) TestNEG010GetGuestCartNoDeviceID() {
	cl := helpers.NewAPIClient(s.server)
	w := cl.Get(s.T(), GuestCartAPIEndpoint)
	helpers.AssertErrorResponse(s.T(), w, http.StatusBadRequest)
}

func (s *GuestCartTestSuite) TestNEG011DeleteOtherDeviceCart() {
	// Device 1 adds item
	w := s.guestClient.Post(s.T(), GuestCartItemAPIEndpoint, guestAddItemPayload(1, 1))
	resp := helpers.AssertSuccessResponse(s.T(), w, http.StatusCreated)
	data := resp["data"].(map[string]any)
	cartID := uint(data["id"].(float64))

	// Device 2 tries to delete device 1's cart — should fail
	w = s.guestClient2.Delete(s.T(), guestCartDeleteURL(cartID))
	helpers.AssertErrorResponse(s.T(), w, http.StatusNotFound)
}

// ============================================================================
// MERGE PATH — Guest Cart Merge into User Cart
// ============================================================================

func (s *GuestCartTestSuite) TestMERGE001MergeGuestCartIntoUserCart() {
	// Arrange: guest adds item
	w := s.guestClient.Post(s.T(), GuestCartItemAPIEndpoint, guestAddItemPayload(1, 1))
	helpers.AssertSuccessResponse(s.T(), w, http.StatusCreated)

	// Act: user merges guest cart
	w = s.customerClient.Post(s.T(), CartMergeAPIEndpoint, mergeRequest(testDeviceID))
	resp := helpers.AssertSuccessResponse(s.T(), w, http.StatusOK)
	data := resp["data"].(map[string]any)
	s.assertUserCartEnvelope(data, helpers.CustomerUserID)
	items := data["items"].([]any)
	assert.Len(s.T(), items, 1)
	item := items[0].(map[string]any)
	assert.Equal(s.T(), float64(1), item["quantity"])

	// Assert: device cart is deleted after merge
	s.assertNoDeviceCart(testDeviceID)
}

func (s *GuestCartTestSuite) TestMERGE002MergeWithNoGuestCart() {
	// No guest cart exists — merge should be a no-op, return user cart
	w := s.customerClient.Post(s.T(), CartMergeAPIEndpoint, mergeRequest(testDeviceID))
	resp := helpers.AssertSuccessResponse(s.T(), w, http.StatusOK)
	data := resp["data"].(map[string]any)
	s.assertUserCartEnvelope(data, helpers.CustomerUserID)
	items := data["items"].([]any)
	assert.Len(s.T(), items, 0)
}

func (s *GuestCartTestSuite) TestMERGE003MergeWithEmptyGuestCart() {
	// Create guest cart item, delete it, then merge
	w := s.guestClient.Post(s.T(), GuestCartItemAPIEndpoint, guestAddItemPayload(1, 1))
	resp := helpers.AssertSuccessResponse(s.T(), w, http.StatusCreated)
	data := resp["data"].(map[string]any)
	cartID := uint(data["id"].(float64))

	// Delete guest cart items
	s.Require().NoError(
		s.container.DB.Where("cart_id = ?", cartID).Delete(&orderEntity.CartItem{}).Error,
	)

	// Merge should be a no-op since device cart is empty
	w = s.customerClient.Post(s.T(), CartMergeAPIEndpoint, mergeRequest(testDeviceID))
	resp = helpers.AssertSuccessResponse(s.T(), w, http.StatusOK)
	data = resp["data"].(map[string]any)
	s.assertUserCartEnvelope(data, helpers.CustomerUserID)
}

func (s *GuestCartTestSuite) TestMERGE004MergeWithSameVariantQuantitiesSum() {
	// Arrange: guest adds variant 1, quantity 2
	w := s.guestClient.Post(s.T(), GuestCartItemAPIEndpoint, guestAddItemPayload(1, 2))
	helpers.AssertSuccessResponse(s.T(), w, http.StatusCreated)

	// User already has variant 1, quantity 1 in cart
	w = s.customerClient.Post(s.T(), CartItemAPIEndpoint, cartItemsPayload(cartItem(uint(1), 1)))
	helpers.AssertSuccessResponse(s.T(), w, http.StatusCreated)

	// Act: merge — quantities should sum (2 + 1 = 3)
	w = s.customerClient.Post(s.T(), CartMergeAPIEndpoint, mergeRequest(testDeviceID))
	resp := helpers.AssertSuccessResponse(s.T(), w, http.StatusOK)
	data := resp["data"].(map[string]any)
	items := data["items"].([]any)
	assert.Len(s.T(), items, 1)
	item := items[0].(map[string]any)
	assert.Equal(s.T(), float64(3), item["quantity"])

	// Assert: device cart is deleted
	s.assertNoDeviceCart(testDeviceID)
}

func (s *GuestCartTestSuite) TestMERGE005MergeWithDifferentVariants() {
	// Arrange: guest adds variant 5
	w := s.guestClient.Post(s.T(), GuestCartItemAPIEndpoint, guestAddItemPayload(5, 1))
	helpers.AssertSuccessResponse(s.T(), w, http.StatusCreated)

	// User has variant 1 in cart
	w = s.customerClient.Post(s.T(), CartItemAPIEndpoint, cartItemsPayload(cartItem(uint(1), 1)))
	helpers.AssertSuccessResponse(s.T(), w, http.StatusCreated)

	// Act: merge — both variants should be in the user cart
	w = s.customerClient.Post(s.T(), CartMergeAPIEndpoint, mergeRequest(testDeviceID))
	resp := helpers.AssertSuccessResponse(s.T(), w, http.StatusOK)
	data := resp["data"].(map[string]any)
	items := data["items"].([]any)
	assert.Len(s.T(), items, 2)

	// Assert: device cart is deleted
	s.assertNoDeviceCart(testDeviceID)
}

func (s *GuestCartTestSuite) TestMERGE006MergeWithEmptyGuestCartNoItems() {
	// Guest creates a cart but no items (quantity 0), then merge
	w := s.guestClient.Post(s.T(), GuestCartItemAPIEndpoint, guestAddItemPayload(1, 0))
	_ = helpers.AssertSuccessResponse(s.T(), w, http.StatusCreated)

	// Merge should work fine (no-op)
	w2 := s.customerClient.Post(s.T(), CartMergeAPIEndpoint, mergeRequest(testDeviceID))
	resp := helpers.AssertSuccessResponse(s.T(), w2, http.StatusOK)
	data := resp["data"].(map[string]any)
	s.assertUserCartEnvelope(data, helpers.CustomerUserID)
}

// ============================================================================
// MERGE NEGATIVE PATH
// ============================================================================

func (s *GuestCartTestSuite) TestMERGENEG001NoAuth() {
	cl := helpers.NewAPIClient(s.server)
	w := cl.Post(s.T(), CartMergeAPIEndpoint, mergeRequest(testDeviceID))
	helpers.AssertErrorResponse(s.T(), w, http.StatusUnauthorized)
}

func (s *GuestCartTestSuite) TestMERGENEG002MissingDeviceID() {
	// DeviceID is empty string
	w := s.customerClient.Post(s.T(), CartMergeAPIEndpoint, mergeRequest(""))
	helpers.AssertErrorResponse(s.T(), w, http.StatusBadRequest)
}

func (s *GuestCartTestSuite) TestMERGENEG003InvalidToken() {
	cl := helpers.NewAPIClient(s.server)
	cl.SetToken("not-a-valid-jwt")
	w := cl.Post(s.T(), CartMergeAPIEndpoint, mergeRequest(testDeviceID))
	helpers.AssertErrorResponse(s.T(), w, http.StatusUnauthorized)
}

func (s *GuestCartTestSuite) TestMERGENEG004NonExistentDeviceCart() {
	// Device ID that has no cart at all (never used) — should be no-op
	w := s.customerClient.Post(
		s.T(),
		CartMergeAPIEndpoint,
		mergeRequest("non-existent-device-uuid"),
	)
	resp := helpers.AssertSuccessResponse(s.T(), w, http.StatusOK)
	data := resp["data"].(map[string]any)
	s.assertUserCartEnvelope(data, helpers.CustomerUserID)
	items := data["items"].([]any)
	assert.Len(s.T(), items, 0)
}

// ============================================================================
// EDGE CASES
// ============================================================================

func (s *GuestCartTestSuite) TestEDGE001MultipleDevicesIsolated() {
	// Device 1 adds variant 1
	w := s.guestClient.Post(s.T(), GuestCartItemAPIEndpoint, guestAddItemPayload(1, 1))
	helpers.AssertSuccessResponse(s.T(), w, http.StatusCreated)

	// Device 2 adds variant 5
	w = s.guestClient2.Post(s.T(), GuestCartItemAPIEndpoint, guestAddItemPayload(5, 1))
	helpers.AssertSuccessResponse(s.T(), w, http.StatusCreated)

	// Device 1 should only see their own items
	w = s.guestClient.Get(s.T(), GuestCartAPIEndpoint)
	resp := helpers.AssertSuccessResponse(s.T(), w, http.StatusOK)
	data := resp["data"].(map[string]any)
	items := data["items"].([]any)
	assert.Len(s.T(), items, 1)
	item := items[0].(map[string]any)
	assert.Equal(s.T(), float64(1), item["variantId"])

	// Device 2 should only see their own items
	w = s.guestClient2.Get(s.T(), GuestCartAPIEndpoint)
	resp = helpers.AssertSuccessResponse(s.T(), w, http.StatusOK)
	data = resp["data"].(map[string]any)
	items = data["items"].([]any)
	assert.Len(s.T(), items, 1)
	item = items[0].(map[string]any)
	assert.Equal(s.T(), float64(5), item["variantId"])
}

func (s *GuestCartTestSuite) TestEDGE002MergeThenAddMoreGuestItems() {
	// Guest adds item, merges
	w := s.guestClient.Post(s.T(), GuestCartItemAPIEndpoint, guestAddItemPayload(1, 1))
	helpers.AssertSuccessResponse(s.T(), w, http.StatusCreated)

	w = s.customerClient.Post(s.T(), CartMergeAPIEndpoint, mergeRequest(testDeviceID))
	helpers.AssertSuccessResponse(s.T(), w, http.StatusOK)

	// Device cart should be gone
	s.assertNoDeviceCart(testDeviceID)

	// Guest adds more items — should create a new device cart
	w = s.guestClient.Post(s.T(), GuestCartItemAPIEndpoint, guestAddItemPayload(5, 2))
	helpers.AssertSuccessResponse(s.T(), w, http.StatusCreated)

	// New device cart should exist with the new item
	s.assertHasDeviceCart(testDeviceID)
}

// ============================================================================
// URL helpers
// ============================================================================

func guestCartDeleteURL(cartID uint) string {
	return "/api/order/cart/guest/" + formatUint(cartID)
}

func mergeRequest(deviceID string) map[string]any {
	return map[string]any{"deviceId": deviceID}
}

func formatUint(n uint) string {
	if n == 0 {
		return "0"
	}
	result := ""
	for n > 0 {
		digit := n % 10
		result = string(rune('0'+digit)) + result
		n /= 10
	}
	return result
}
