package shiprocket

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	fulfillmentmodel "ecommerce-be/fulfillment/model"
)

// Shiprocket endpoint paths (adapter-confined; orchestrators never see these).
const (
	pathCreateAdhoc  = "/v1/external/orders/create/adhoc"
	pathAssignAWB    = "/v1/external/courier/assign/awb"
	pathGeneratePick = "/v1/external/courier/generate/pickup"
	pathCancelOrder  = "/v1/external/orders/cancel"
	pathGenerateLbl  = "/v1/external/courier/generate/label"
)

// derivePickupAlias computes the courier pickup nickname from our ids:
// S{seller}L{location}. Sellers register exactly these nicknames (verified
// at onboarding); nothing is stored per shipment.
func derivePickupAlias(sellerID, locationID uint) string {
	return fmt.Sprintf("S%dL%d", sellerID, locationID)
}

// BookShipment creates the Shiprocket order (our shipment id as their
// order id), assigns the AWB, and schedules pickup when requested — one
// contract call, three provider calls. POSTs are never retried: the
// provider dedupes on our order id, and a retried POST risks double AWBs.
func (a *Adapter) BookShipment(
	ctx context.Context,
	in fulfillmentmodel.BookShipmentInput,
	creds map[string]any,
) (*fulfillmentmodel.BookShipmentOutput, error) {
	parsed, err := parseCredentials(creds)
	if err != nil {
		return nil, fmt.Errorf("[shiprocket] %w", err)
	}
	if in.WeightGrams == nil || *in.WeightGrams <= 0 {
		return nil, fmt.Errorf("shiprocket: weight is required for booking")
	}

	out, err := withAuthResult(ctx, a, parsed, func(ctx context.Context, token string) (*fulfillmentmodel.BookShipmentOutput, error) {
		return a.bookAll(ctx, token, in)
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// bookAll runs create → AWB → optional pickup under one token.
func (a *Adapter) bookAll(
	ctx context.Context,
	token string,
	in fulfillmentmodel.BookShipmentInput,
) (*fulfillmentmodel.BookShipmentOutput, error) {
	providerOrderID := strconv.FormatUint(uint64(in.ShipmentID), 10)

	created, err := a.createAdhocOrder(ctx, token, in, providerOrderID)
	if err != nil {
		return nil, err
	}
	providerShipmentID := stringValue(created["shipment_id"])
	if strings.TrimSpace(providerShipmentID) == "" {
		return nil, fmt.Errorf("shiprocket: shipment id missing in create-order response")
	}

	awbResp, err := a.doJSON(ctx, token, http.MethodPost, pathAssignAWB, map[string]any{
		"shipment_id": providerShipmentID,
	})
	if err != nil {
		return nil, err
	}
	awb, courierName := parseAWBResponse(awbResp)
	if strings.TrimSpace(awb) == "" {
		return nil, fmt.Errorf("shiprocket: awb missing in assign-awb response")
	}

	if in.PickupAt != nil {
		if _, err := a.doJSON(ctx, token, http.MethodPost, pathGeneratePick, map[string]any{
			"shipment_id": []string{providerShipmentID},
		}); err != nil {
			return nil, err
		}
	}

	return &fulfillmentmodel.BookShipmentOutput{
		ProviderOrderID:    providerOrderID,
		ProviderShipmentID: providerShipmentID,
		AWB:                awb,
		CourierName:        courierName,
		RawResponse:        awbResp,
	}, nil
}

// createAdhocOrder maps our box to Shiprocket's adhoc order fields.
func (a *Adapter) createAdhocOrder(
	ctx context.Context,
	token string,
	in fulfillmentmodel.BookShipmentInput,
	providerOrderID string,
) (map[string]any, error) {
	paymentMethod := "Prepaid"
	if in.CodCents > 0 {
		paymentMethod = "COD"
	}
	items := make([]map[string]any, 0, len(in.Lines))
	for _, line := range in.Lines {
		items = append(items, map[string]any{
			"name":          line.Name,
			"sku":           line.SKU,
			"units":         line.Quantity,
			"selling_price": line.UnitPriceCents,
		})
	}
	body := map[string]any{
		"order_id":              providerOrderID,
		"order_date":            time.Now().UTC().Format("2006-01-02 15:04"),
		"pickup_location":       derivePickupAlias(in.SellerID, in.PickupLocationID),
		"billing_customer_name": in.RecipientName,
		"billing_phone":         in.RecipientPhone,
		"billing_address":       in.Street,
		"billing_city":          in.City,
		"billing_state":         in.State,
		"billing_pincode":       in.Pincode,
		"shipping_is_billing":   true,
		"order_items":           items,
		"payment_method":        paymentMethod,
		"sub_total":             in.SubTotalCents,
		"length":                floatValue(in.LengthCm),
		"breadth":               floatValue(in.BreadthCm),
		"height":                floatValue(in.HeightCm),
		"weight":                float64(*in.WeightGrams) / 1000.0,
	}
	return a.doJSON(ctx, token, http.MethodPost, pathCreateAdhoc, body)
}

// SchedulePickup schedules pickup separately from booking (optional
// PickupScheduler capability). The pick list is provider shipment ids.
func (a *Adapter) SchedulePickup(
	ctx context.Context,
	in fulfillmentmodel.PickupInput,
	creds map[string]any,
) error {
	parsed, err := parseCredentials(creds)
	if err != nil {
		return fmt.Errorf("[shiprocket] %w", err)
	}
	shipmentID := strings.TrimSpace(in.ProviderShipmentID)
	if shipmentID == "" {
		return fmt.Errorf("shiprocket: provider shipment id is required for pickup")
	}
	_, err = a.withAuth(ctx, parsed, func(ctx context.Context, token string) (map[string]any, error) {
		return a.doJSON(ctx, token, http.MethodPost, pathGeneratePick, map[string]any{
			"shipment_id": []string{shipmentID},
		})
	})
	return err
}

// Cancel cancels a pre-pickup provider order. Post-pickup cancellations are
// refused by the provider (4xx) and surface as errors for the service to map.
func (a *Adapter) Cancel(
	ctx context.Context,
	in fulfillmentmodel.CancelInput,
	creds map[string]any,
) error {
	parsed, err := parseCredentials(creds)
	if err != nil {
		return fmt.Errorf("[shiprocket] %w", err)
	}
	_, err = a.withAuth(ctx, parsed, func(ctx context.Context, token string) (map[string]any, error) {
		return a.doJSON(ctx, token, http.MethodPost, pathCancelOrder, map[string]any{
			"ids": []string{in.ProviderOrderID},
		})
	})
	return err
}

// GetLabel generates the shipping label and streams its bytes. Labels are
// never stored — callers stream them straight to the client.
func (a *Adapter) GetLabel(
	ctx context.Context,
	in fulfillmentmodel.LabelInput,
	creds map[string]any,
) ([]byte, error) {
	parsed, err := parseCredentials(creds)
	if err != nil {
		return nil, fmt.Errorf("[shiprocket] %w", err)
	}
	resp, err := a.withAuth(ctx, parsed, func(ctx context.Context, token string) (map[string]any, error) {
		return a.doJSON(ctx, token, http.MethodPost, pathGenerateLbl, map[string]any{
			"shipment_id": []string{in.ProviderShipmentID},
		})
	})
	if err != nil {
		return nil, err
	}
	labelURL := stringValue(resp["label_url"])
	if strings.TrimSpace(labelURL) == "" {
		if nested, _ := resp["response"].(map[string]any); nested != nil {
			labelURL = stringValue(nested["label_url"])
		}
	}
	if strings.TrimSpace(labelURL) == "" {
		return nil, fmt.Errorf("shiprocket: label_url missing in label response")
	}
	return fetchBytes(ctx, labelURL)
}

// fetchBytes downloads a provider file (label PDF) with a hard size cap.
// The URL comes from the provider response, never from caller input.
func fetchBytes(ctx context.Context, url string) ([]byte, error) {
	if !strings.HasPrefix(url, "https://") && !strings.HasPrefix(url, "http://") {
		return nil, fmt.Errorf("shiprocket: refusing non-http label url")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("shiprocket: build label request: %w", err)
	}
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("shiprocket: download label: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("shiprocket: label download status=%d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<22)) // 4MB cap
	if err != nil {
		return nil, fmt.Errorf("shiprocket: read label: %w", err)
	}
	return body, nil
}

// parseAWBResponse extracts the AWB + courier from assign-awb responses,
// tolerating both nested (response.data) and flat shapes.
func parseAWBResponse(resp map[string]any) (awb, courier string) {
	if nested, _ := resp["response"].(map[string]any); nested != nil {
		if data, _ := nested["data"].(map[string]any); data != nil {
			return stringValue(data["awb_code"]), stringValue(data["courier_name"])
		}
		return stringValue(nested["awb_code"]), stringValue(nested["courier_name"])
	}
	return stringValue(resp["awb_code"]), stringValue(resp["courier_name"])
}

// stringValue coerces provider JSON scalars (string/number) to string.
func stringValue(v any) string {
	switch value := v.(type) {
	case string:
		return value
	case float64:
		if value == float64(int64(value)) {
			return strconv.FormatInt(int64(value), 10)
		}
		return strconv.FormatFloat(value, 'f', -1, 64)
	case json.Number:
		return string(value)
	case int:
		return strconv.Itoa(value)
	case int64:
		return strconv.FormatInt(value, 10)
	default:
		return ""
	}
}

// floatValue dereferences optional dimensions (nil → 0 for the provider).
func floatValue(v *float64) float64 {
	if v == nil {
		return 0
	}
	return *v
}
