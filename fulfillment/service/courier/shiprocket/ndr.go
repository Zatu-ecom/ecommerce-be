package shiprocket

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	fulfillmentmodel "ecommerce-be/fulfillment/model"
	courier "ecommerce-be/fulfillment/service/courier"
)

// NDR + return endpoints (adapter-confined). Shiprocket models failed
// delivery as NDR state on the AWB: query it, then answer with reattempt or
// RTO. Proactive RTO (no open NDR) goes through the same action surface —
// the courier accepts RTO actions on in-transit AWBs.
const (
	pathNDRAction = "/v1/external/ndr/%s/action"
	pathGetNDR    = "/v1/external/ndr/%s"
	pathCreateRet = "/v1/external/orders/create/return"
)

// compile-time guards: implemented optional capabilities.
var (
	_ courier.NDRHandler    = (*Adapter)(nil)
	_ courier.ReturnHandler = (*Adapter)(nil)
	_ courier.RTORequester  = (*Adapter)(nil)
)

// GetNDR fetches the provider's failed-delivery detail for one AWB.
func (a *Adapter) GetNDR(
	ctx context.Context,
	awb string,
	creds map[string]any,
) (*fulfillmentmodel.NDRDetail, error) {
	parsed, err := parseCredentials(creds)
	if err != nil {
		return nil, fmt.Errorf("[shiprocket] %w", err)
	}
	if strings.TrimSpace(awb) == "" {
		return nil, fmt.Errorf("shiprocket: awb is required for NDR detail")
	}
	resp, err := a.withAuth(ctx, parsed, func(ctx context.Context, token string) (map[string]any, error) {
		return a.doJSON(ctx, token, http.MethodGet, fmt.Sprintf(pathGetNDR, awb), nil)
	})
	if err != nil {
		return nil, err
	}
	detail := &fulfillmentmodel.NDRDetail{AWB: awb, Raw: resp}
	if data, _ := resp["data"].(map[string]any); data != nil {
		detail.NDRStatus = stringValue(data["ndr_status"])
		detail.Reason = stringValue(data["reason"])
	} else {
		detail.NDRStatus = stringValue(resp["ndr_status"])
		detail.Reason = stringValue(resp["reason"])
	}
	if detail.NDRStatus == "" {
		detail.NDRStatus = stringValue(resp["current_status"])
	}
	return detail, nil
}

// ActNDR answers an open NDR round: reattempt delivery or convert to RTO.
func (a *Adapter) ActNDR(
	ctx context.Context,
	in fulfillmentmodel.NDRActionInput,
	creds map[string]any,
) error {
	parsed, err := parseCredentials(creds)
	if err != nil {
		return fmt.Errorf("[shiprocket] %w", err)
	}
	if strings.TrimSpace(in.AWB) == "" {
		return fmt.Errorf("shiprocket: awb is required for NDR action")
	}
	action := strings.ToLower(strings.TrimSpace(in.Action))
	if action != "reattempt" && action != "rto" {
		return fmt.Errorf("shiprocket: unknown NDR action %q", in.Action)
	}
	body := map[string]any{"action": action}
	if strings.TrimSpace(in.AddressNote) != "" {
		body["address_note"] = in.AddressNote
	}
	_, err = a.withAuth(ctx, parsed, func(ctx context.Context, token string) (map[string]any, error) {
		return a.doJSON(ctx, token, http.MethodPost, fmt.Sprintf(pathNDRAction, in.AWB), body)
	})
	return err
}

// RequestRTO asks for a mid-transit box back without an open NDR round.
// Same action surface as ActNDR (the courier accepts RTO on in-transit
// AWBs); local status never moves until the courier confirms the RTO leg.
func (a *Adapter) RequestRTO(
	ctx context.Context,
	in fulfillmentmodel.RTORequestInput,
	creds map[string]any,
) error {
	return a.ActNDR(ctx, fulfillmentmodel.NDRActionInput{
		AWB:    in.AWB,
		Action: "rto",
	}, creds)
}

// BookReturn registers a return (RTO-after-delivery) order, then assigns
// its AWB through the shared path. The service composes the full shipping
// context (input.Ship mirrors a forward book); the original AWB travels as
// the return reference.
func (a *Adapter) BookReturn(
	ctx context.Context,
	in fulfillmentmodel.BookReturnInput,
	creds map[string]any,
) (*fulfillmentmodel.BookShipmentOutput, error) {
	parsed, err := parseCredentials(creds)
	if err != nil {
		return nil, fmt.Errorf("[shiprocket] %w", err)
	}
	if in.Ship.PickupLocationID == 0 {
		return nil, fmt.Errorf("shiprocket: return needs a pickup location")
	}
	out, err := withAuthResult(ctx, a, parsed, func(ctx context.Context, token string) (*fulfillmentmodel.BookShipmentOutput, error) {
		return a.bookReturnAll(ctx, token, in)
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// bookReturnAll runs return-create → AWB assignment under one token.
func (a *Adapter) bookReturnAll(
	ctx context.Context,
	token string,
	in fulfillmentmodel.BookReturnInput,
) (*fulfillmentmodel.BookShipmentOutput, error) {
	ship := in.Ship
	providerOrderID := fmt.Sprintf("%d", ship.ShipmentID)

	items := make([]map[string]any, 0, len(ship.Lines))
	for _, line := range ship.Lines {
		items = append(items, map[string]any{
			"name":          line.Name,
			"sku":           line.SKU,
			"units":         line.Quantity,
			"selling_price": line.UnitPriceCents,
		})
	}
	body := map[string]any{
		"order_id":              providerOrderID,
		"order_date":            nowOrderDate(),
		"pickup_location":       derivePickupAlias(ship.SellerID, ship.PickupLocationID),
		"billing_customer_name": ship.RecipientName,
		"billing_phone":         ship.RecipientPhone,
		"billing_address":       ship.Street,
		"billing_city":          ship.City,
		"billing_state":         ship.State,
		"billing_pincode":       ship.Pincode,
		"shipping_is_billing":   true,
		"order_items":           items,
		"payment_method":        "Prepaid",
		"sub_total":             0,
		"length":                floatValue(ship.LengthCm),
		"breadth":               floatValue(ship.BreadthCm),
		"height":                floatValue(ship.HeightCm),
		"weight":                gramsToKG(derefWeight(ship.WeightGrams)),
		"reason":                in.Reason,
	}
	created, err := a.doJSON(ctx, token, http.MethodPost, pathCreateRet, body)
	if err != nil {
		return nil, err
	}
	providerShipmentID := stringValue(created["shipment_id"])
	if strings.TrimSpace(providerShipmentID) == "" {
		return nil, fmt.Errorf("shiprocket: shipment id missing in create-return response")
	}
	awbResp, err := a.doJSON(ctx, token, http.MethodPost, pathAssignAWB, map[string]any{
		"shipment_id": providerShipmentID,
	})
	if err != nil {
		return nil, err
	}
	awb, courierName := parseAWBResponse(awbResp)
	if strings.TrimSpace(awb) == "" {
		return nil, fmt.Errorf("shiprocket: awb missing in return assign-awb response")
	}
	return &fulfillmentmodel.BookShipmentOutput{
		ProviderOrderID:    providerOrderID,
		ProviderShipmentID: providerShipmentID,
		AWB:                awb,
		CourierName:        courierName,
		RawResponse:        awbResp,
	}, nil
}

// nowOrderDate formats the provider order timestamp.
func nowOrderDate() string {
	return time.Now().UTC().Format("2006-01-02 15:04")
}

func derefWeight(v *int) int {
	if v == nil {
		return 0
	}
	return *v
}
