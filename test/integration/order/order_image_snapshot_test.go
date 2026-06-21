package order_test

import (
	"fmt"
	"net/http"
	"testing"

	"ecommerce-be/test/integration/helpers"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOrderItemImageSnapshotFromVariantMedia(t *testing.T) {
	env := helpers.SetupFileStorageEnv(t, helpers.DefaultFileStorageEnvConfig())

	sellerToken := helpers.Login(t, env.Client, helpers.Seller2Email, helpers.Seller2Password)
	fileID := helpers.UploadProductImage(t, env.Server, sellerToken)
	helpers.SeedVariantMediaRow(t, env.Containers, 1, fileID, true, 0)

	customerClient := helpers.NewAPIClient(env.Server)
	customerToken := helpers.Login(
		t,
		customerClient,
		helpers.CustomerEmail,
		helpers.CustomerPassword,
	)
	customerClient.SetToken(customerToken)

	addW := customerClient.Post(t, "/api/order/cart/item", map[string]any{
		"items": []map[string]any{
			{"variantId": 1, "quantity": 1},
		},
	})
	helpers.AssertSuccessResponse(t, addW, http.StatusCreated)

	orderW := customerClient.Post(t, OrderAPIEndpoint, map[string]any{
		"shippingAddressId": 1,
		"billingAddressId":  1,
		"fulfillmentType":   "directship",
	})
	orderResp := helpers.AssertSuccessResponse(t, orderW, http.StatusCreated)
	orderData := orderResp["data"].(map[string]any)
	orderID := uint(orderData["id"].(float64))

	getW := customerClient.Get(t, fmt.Sprintf(OrderByIDAPIEndpoint, orderID))
	getResp := helpers.AssertSuccessResponse(t, getW, http.StatusOK)
	fetched := getResp["data"].(map[string]any)
	items, ok := fetched["items"].([]any)
	require.True(t, ok)
	require.NotEmpty(t, items)

	orderItem := items[0].(map[string]any)
	imageURL, ok := orderItem["imageUrl"].(string)
	require.True(t, ok)
	assert.NotEmpty(t, imageURL)

	imageFileID, ok := orderItem["imageFileId"].(string)
	require.True(t, ok)
	assert.Equal(t, fileID, imageFileID)
}
