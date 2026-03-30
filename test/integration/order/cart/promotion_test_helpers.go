package CartTest

import (
	"net/http"
	"testing"

	"ecommerce-be/promotion/model"
	"ecommerce-be/test/integration/helpers"

	"github.com/stretchr/testify/require"
)

// PromotionAPIEndpoint is POST /api/promotion (create promotion; same as promotion integration tests).
const PromotionAPIEndpoint = "/api/promotion"

// Bundle / promotion payload constants — copied from test/integration/promotion/bundle_test.go
const (
	promoTypeBundle = "bundle"

	bundleDiscountTypePercentage  = "percentage"
	bundleDiscountTypeFixedAmount = "fixed_amount"
	bundleDiscountTypeFixedPrice  = "fixed_price"

	appliesAllProducts = "all_products"
	eligibleEveryone   = "everyone"
	promoStatusActive  = "active"
)

// postPromotion POSTs a promotion payload as the seller and returns the new promotion ID.
func postPromotion(t *testing.T, seller *helpers.APIClient, payload map[string]interface{}) uint {
	res := seller.Post(t, PromotionAPIEndpoint, payload)
	require.Equal(t, http.StatusCreated, res.Code)
	respData := helpers.ParseResponse(t, res.Body)
	promo := respData["data"].(map[string]interface{})["promotion"].(map[string]interface{})
	return uint(promo["id"].(float64))
}

// buildBundlePayload builds a bundle promotion request body (bundle_test.go).
func buildBundlePayload(
	name string,
	discountType string,
	discountValue *float64,
	bundlePrice *int64,
	bundleItems []model.BundleItemConfig,
) map[string]interface{} {
	return map[string]interface{}{
		"name":          name,
		"promotionType": promoTypeBundle,
		"discountConfig": model.BundleConfig{
			BundleItems:         bundleItems,
			BundleDiscountType:  model.DiscountType(discountType),
			BundleDiscountValue: discountValue,
			BundlePriceCents:    bundlePrice,
		},
		"appliesTo":   appliesAllProducts,
		"eligibleFor": eligibleEveryone,
		"startsAt":    "2023-01-01T00:00:00Z",
		"endsAt":      "2029-12-31T23:59:59Z",
		"status":      promoStatusActive,
	}
}

// bundleItem builds one bundle requirement line (bundle_test.go).
func bundleItem(productID, variantID uint, quantity int) model.BundleItemConfig {
	v := variantID
	return model.BundleItemConfig{
		ProductID: productID,
		VariantID: &v,
		Quantity:  quantity,
	}
}

// createBundlePromotion is a convenience: build bundle payload + POST + return promotion id.
func createBundlePromotion(
	t *testing.T,
	seller *helpers.APIClient,
	name string,
	discountType string,
	discountValue *float64,
	bundlePrice *int64,
	bundleItems []model.BundleItemConfig,
) uint {
	payload := buildBundlePayload(name, discountType, discountValue, bundlePrice, bundleItems)
	return postPromotion(t, seller, payload)
}
