package shiprocket

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	fulfillmentmodel "ecommerce-be/fulfillment/model"
)

// GetRates shops live courier rates via serviceability. Read-only: safe to
// retry (GET policy) and safe to cache — the service owns both.
func (a *Adapter) GetRates(
	ctx context.Context,
	in fulfillmentmodel.RateInput,
	creds map[string]any,
) ([]fulfillmentmodel.RateOption, error) {
	parsed, err := parseCredentials(creds)
	if err != nil {
		return nil, fmt.Errorf("[shiprocket] %w", err)
	}
	cod := 0
	if in.CodCents > 0 {
		cod = 1
	}
	resp, err := a.withAuth(ctx, parsed, func(ctx context.Context, token string) (map[string]any, error) {
		query := url.Values{}
		query.Set("pickup_postcode", in.PickupPincode)
		query.Set("delivery_postcode", in.DeliveryPincode)
		query.Set("weight", strconv.FormatFloat(gramsToKG(in.WeightGrams), 'f', -1, 64))
		query.Set("cod", strconv.Itoa(cod))
		return a.doJSON(ctx, token, http.MethodGet,
			"/v1/external/courier/serviceability/?"+query.Encode(), nil)
	})
	if err != nil {
		return nil, err
	}
	return parseRateOptions(resp), nil
}

// gramsToKG converts box grams to the kg decimals the API expects.
func gramsToKG(weightGrams int) float64 {
	if weightGrams <= 0 {
		return 0.5
	}
	return float64(weightGrams) / 1000.0
}

// parseRateOptions maps available_courier_companies (nested under data, with
// a flat fallback) to provider-agnostic options. Empty list is valid (route
// unserved) — never an error.
func parseRateOptions(resp map[string]any) []fulfillmentmodel.RateOption {
	companies, _ := resp["available_courier_companies"].([]any)
	if data, _ := resp["data"].(map[string]any); data != nil {
		if list, _ := data["available_courier_companies"].([]any); list != nil {
			companies = list
		}
	}
	options := make([]fulfillmentmodel.RateOption, 0, len(companies))
	for _, item := range companies {
		company, _ := item.(map[string]any)
		if company == nil {
			continue
		}
		name := stringValue(company["courier_name"])
		if strings.TrimSpace(name) == "" {
			continue
		}
		code := stringValue(company["courier_company_id"])
		if strings.TrimSpace(code) == "" {
			code = name
		}
		options = append(options, fulfillmentmodel.RateOption{
			CourierName: name,
			ServiceCode: code,
			// Provider quotes decimal rupees; storage is integer paise.
			RateCents:     int64(numberValue(company["rate"])*100 + 0.5),
			ETD:           parseETD(stringValue(company["etd"])),
			PickupCapable: true,
		})
	}
	return options
}

// parseETD parses provider ETA strings; unparseable stays nil (display-only).
func parseETD(raw string) *time.Time {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	for _, layout := range []string{time.RFC3339, "2006-01-02 15:04:05", "2006-01-02"} {
		if parsed, err := time.Parse(layout, strings.TrimSpace(raw)); err == nil {
			utc := parsed.UTC()
			return &utc
		}
	}
	return nil
}

// numberValue coerces provider JSON numbers to float64.
func numberValue(v any) float64 {
	switch num := v.(type) {
	case float64:
		return num
	case float32:
		return float64(num)
	case int:
		return float64(num)
	case int64:
		return float64(num)
	default:
		return 0
	}
}
