package CartTest

import (
	"fmt"
	"testing"

	"ecommerce-be/test/integration/helpers"
)

// deleteCartByID deletes the customer's active cart (by ID) so tests start fresh.
func deleteCartByID(t *testing.T, client *helpers.APIClient) {
	gw := client.Get(t, CartAPIEndpoint)
	gr := helpers.ParseResponse(t, gw.Body)
	if cart, ok := gr["data"].(map[string]any); ok {
		if id, ok := cart["id"].(float64); ok && id > 0 {
			_ = client.Delete(t, fmt.Sprintf("%s/%d", CartAPIEndpoint, int(id)))
		}
	}
}
