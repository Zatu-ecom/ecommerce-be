package CartTest

import (
	"net/http"
	"testing"

	"ecommerce-be/test/integration/helpers"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVariantImagesPopulatedFromMedia(t *testing.T) {
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

	w := customerClient.Post(t, CartItemAPIEndpoint, cartItemsPayload(cartItem(uint(1), 1)))
	resp := helpers.AssertSuccessResponse(t, w, http.StatusCreated)
	data := resp["data"].(map[string]any)
	items := data["items"].([]any)
	require.Len(t, items, 1)

	item := items[0].(map[string]any)
	variant, ok := item["variant"].(map[string]any)
	require.True(t, ok)

	images, ok := variant["images"].([]any)
	require.True(t, ok)
	require.NotEmpty(t, images, "variant.images must be populated from variant media")
	assert.NotEmpty(t, images[0])

	imageFileID, ok := variant["imageFileId"].(string)
	require.True(t, ok)
	assert.Equal(t, fileID, imageFileID)
}
