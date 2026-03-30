package CartTest

import (
	"fmt"
	"net/http"

	"ecommerce-be/test/integration/helpers"

	"github.com/stretchr/testify/assert"
)

func (s *CartTestSuite) TestDEL001DeleteCartRemovesAllItemsAndCart() {
	s.cleanupCartsForTestUsers()

	w := s.customerClient.Post(
		s.T(),
		CartItemAPIEndpoint,
		cartItemsPayload(cartItem(uint(1), 1), cartItem(uint(5), 1)),
	)
	helpers.AssertSuccessResponse(s.T(), w, http.StatusCreated)
	cartID := s.getActiveCartID(helpers.CustomerUserID)

	w = s.customerClient.Delete(s.T(), fmt.Sprintf("%s/%d", CartAPIEndpoint, cartID))
	resp := helpers.AssertSuccessResponse(s.T(), w, http.StatusOK)
	data := resp["data"].(map[string]any)
	items := data["items"].([]any)
	assert.Len(s.T(), items, 0)
	s.assertNoCartItems(cartID)
	s.assertNoActiveCart(helpers.CustomerUserID)
}

func (s *CartTestSuite) TestDEL002DeleteCartWrongOwnerRejected() {
	s.cleanupCartsForTestUsers()

	sellerCl := helpers.NewAPIClient(s.server)
	tok := helpers.Login(s.T(), sellerCl, helpers.Seller2Email, helpers.Seller2Password)
	sellerCl.SetToken(tok)

	w := sellerCl.Post(s.T(), CartItemAPIEndpoint, cartItemsPayload(cartItem(uint(1), 1)))
	helpers.AssertSuccessResponse(s.T(), w, http.StatusCreated)
	cartID := s.getActiveCartID(helpers.Seller2UserID)

	w = s.customerClient.Delete(s.T(), fmt.Sprintf("%s/%d", CartAPIEndpoint, cartID))
	helpers.AssertErrorResponse(s.T(), w, http.StatusNotFound)
	s.assertHasActiveCart(helpers.Seller2UserID)
	s.assertCartItemCount(cartID, 1)
}

func (s *CartTestSuite) TestDEL003InvalidCartIDRejected() {
	s.cleanupCartsForTestUsers()

	w := s.customerClient.Delete(s.T(), CartAPIEndpoint+"/invalid-id")
	helpers.AssertErrorResponse(s.T(), w, http.StatusBadRequest)
}

func (s *CartTestSuite) TestDEL004DeleteNonExistingCartRejected() {
	s.cleanupCartsForTestUsers()

	w := s.customerClient.Delete(s.T(), CartAPIEndpoint+"/999999")
	helpers.AssertErrorResponse(s.T(), w, http.StatusNotFound)
}

func (s *CartTestSuite) TestDEL005DeletingSameCartTwiceSecondCallNotFound() {
	s.cleanupCartsForTestUsers()

	w := s.customerClient.Post(s.T(), CartItemAPIEndpoint, cartItemsPayload(cartItem(uint(1), 1)))
	helpers.AssertSuccessResponse(s.T(), w, http.StatusCreated)
	cartID := s.getActiveCartID(helpers.CustomerUserID)

	w = s.customerClient.Delete(s.T(), fmt.Sprintf("%s/%d", CartAPIEndpoint, cartID))
	helpers.AssertSuccessResponse(s.T(), w, http.StatusOK)

	w = s.customerClient.Delete(s.T(), fmt.Sprintf("%s/%d", CartAPIEndpoint, cartID))
	helpers.AssertErrorResponse(s.T(), w, http.StatusNotFound)
}

func (s *CartTestSuite) TestDEL006NoAuth() {
	s.cleanupCartsForTestUsers()
	cl := helpers.NewAPIClient(s.server)
	cl.SetToken("")

	w := cl.Delete(s.T(), CartAPIEndpoint+"/1")
	helpers.AssertErrorResponse(s.T(), w, http.StatusUnauthorized)
}

func (s *CartTestSuite) TestDEL007InvalidTokenRejected() {
	s.cleanupCartsForTestUsers()
	cl := helpers.NewAPIClient(s.server)
	cl.SetToken("bad-token")

	w := cl.Delete(s.T(), CartAPIEndpoint+"/1")
	helpers.AssertErrorResponse(s.T(), w, http.StatusUnauthorized)
}

func (s *CartTestSuite) TestDEL008ZeroCartIDRejected() {
	s.cleanupCartsForTestUsers()

	w := s.customerClient.Delete(s.T(), CartAPIEndpoint+"/0")
	helpers.AssertErrorResponse(s.T(), w, http.StatusBadRequest)
}

func (s *CartTestSuite) TestDEL009SellerCanDeleteOwnCart() {
	s.cleanupCartsForTestUsers()

	sellerCl := helpers.NewAPIClient(s.server)
	tok := helpers.Login(s.T(), sellerCl, helpers.Seller2Email, helpers.Seller2Password)
	sellerCl.SetToken(tok)

	w := sellerCl.Post(s.T(), CartItemAPIEndpoint, cartItemsPayload(cartItem(uint(1), 1)))
	helpers.AssertSuccessResponse(s.T(), w, http.StatusCreated)
	cartID := s.getActiveCartID(helpers.Seller2UserID)

	w = sellerCl.Delete(s.T(), fmt.Sprintf("%s/%d", CartAPIEndpoint, cartID))
	resp := helpers.AssertSuccessResponse(s.T(), w, http.StatusOK)
	data := resp["data"].(map[string]any)
	items := data["items"].([]any)
	assert.Len(s.T(), items, 0)
	s.assertNoCartItems(cartID)
	s.assertNoActiveCart(helpers.Seller2UserID)
}

func (s *CartTestSuite) TestDEL010DeleteCartWithoutItemsStillDeletesCart() {
	s.cleanupCartsForTestUsers()

	cartID := s.createCartOnly(helpers.CustomerUserID)
	w := s.customerClient.Delete(s.T(), fmt.Sprintf("%s/%d", CartAPIEndpoint, cartID))
	resp := helpers.AssertSuccessResponse(s.T(), w, http.StatusOK)
	data := resp["data"].(map[string]any)
	items := data["items"].([]any)
	assert.Len(s.T(), items, 0)
	s.assertNoCartItems(cartID)
	s.assertNoActiveCart(helpers.CustomerUserID)
}
