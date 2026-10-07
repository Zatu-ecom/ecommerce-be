package shiprocket

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	courier "ecommerce-be/fulfillment/service/courier"
)

// FetchTracking polls one AWB. Read-only: safe to retry and safe to cache —
// the service owns both. Unknown statuses map to ignore (observation without
// a state move); the service logs the raw label from ProviderEvent.
func (a *Adapter) FetchTracking(
	ctx context.Context,
	awb string,
	creds map[string]any,
) (*courier.NormalizedShipmentEvent, error) {
	parsed, err := parseCredentials(creds)
	if err != nil {
		return nil, fmt.Errorf("[shiprocket] %w", err)
	}
	if strings.TrimSpace(awb) == "" {
		return nil, fmt.Errorf("shiprocket: awb is required for tracking")
	}
	resp, err := a.withAuth(ctx, parsed, func(ctx context.Context, token string) (map[string]any, error) {
		return a.doJSON(ctx, token, http.MethodGet,
			"/v1/external/courier/track/awb/"+awb, nil)
	})
	if err != nil {
		return nil, err
	}

	raw := trackingStatus(resp)
	event := &courier.NormalizedShipmentEvent{
		Action:        mapShipmentStatus(raw),
		EventID:       "poll:" + awb + ":" + string(mapShipmentStatus(raw)),
		ProviderEvent: raw,
		AWB:           awb,
		CourierName:   trackingCourier(resp),
		Payload:       sanitizeTrackPayload(resp),
	}
	return event, nil
}

// FetchTrackingBulk polls many AWBs in one provider call (the reconciler
// path, batched per account). Each item maps with the same rules as the
// single path; unparseable items are skipped, never fatal.
func (a *Adapter) FetchTrackingBulk(
	ctx context.Context,
	awbs []string,
	creds map[string]any,
) ([]*courier.NormalizedShipmentEvent, error) {
	parsed, err := parseCredentials(creds)
	if err != nil {
		return nil, fmt.Errorf("[shiprocket] %w", err)
	}
	if len(awbs) == 0 {
		return nil, nil
	}
	resp, err := a.withAuth(ctx, parsed, func(ctx context.Context, token string) (map[string]any, error) {
		return a.doJSON(ctx, token, http.MethodPost,
			"/v1/external/courier/track/awbs", map[string]any{"awbs": awbs})
	})
	if err != nil {
		return nil, err
	}
	return parseBulkTracking(resp, awbs), nil
}

// parseBulkTracking maps bulk items, tolerating array-under-data and
// top-level array shapes. Items are matched to requested AWBs by their own
// awb field; unmatched shapes fall back to positional mapping.
func parseBulkTracking(resp map[string]any, awbs []string) []*courier.NormalizedShipmentEvent {
	var items []any
	if data, _ := resp["data"].([]any); data != nil {
		items = data
	} else if list, _ := resp["shipments"].([]any); list != nil {
		items = list
	}
	events := make([]*courier.NormalizedShipmentEvent, 0, len(items))
	for i, item := range items {
		detail, _ := item.(map[string]any)
		if detail == nil {
			continue
		}
		awb := stringValue(detail["awb"])
		if awb == "" && i < len(awbs) {
			awb = awbs[i]
		}
		if awb == "" {
			continue
		}
		raw := firstNonEmpty(
			stringValue(detail["shipment_status"]),
			stringValue(detail["current_status"]),
		)
		action := mapShipmentStatus(raw)
		events = append(events, &courier.NormalizedShipmentEvent{
			Action:        action,
			EventID:       "poll:" + awb + ":" + string(action),
			ProviderEvent: raw,
			AWB:           awb,
			CourierName:   stringValue(detail["courier_name"]),
			Payload:       sanitizeTrackPayload(detail),
		})
	}
	return events
}

// trackingStatus extracts the status label, tolerating nested (tracking_data)
// and flat shapes.
func trackingStatus(resp map[string]any) string {
	if data, _ := resp["tracking_data"].(map[string]any); data != nil {
		if s := stringValue(data["shipment_status"]); s != "" {
			return s
		}
		if s := stringValue(data["current_status"]); s != "" {
			return s
		}
	}
	if s := stringValue(resp["shipment_status"]); s != "" {
		return s
	}
	return stringValue(resp["current_status"])
}

// trackingCourier extracts the courier name with the same tolerance.
func trackingCourier(resp map[string]any) string {
	if data, _ := resp["tracking_data"].(map[string]any); data != nil {
		if s := stringValue(data["courier_name"]); s != "" {
			return s
		}
	}
	return stringValue(resp["courier_name"])
}

// sanitizeTrackPayload keeps the raw payload for audit minus volume: phones
// and street addresses never persist. (Tracking payloads carry no PII by
// contract today; this stays as a guard.)
func sanitizeTrackPayload(resp map[string]any) map[string]any {
	out := map[string]any{}
	for k, v := range resp {
		lower := strings.ToLower(k)
		if strings.Contains(lower, "phone") || strings.Contains(lower, "address") {
			continue
		}
		out[k] = v
	}
	return out
}

// mapShipmentStatus maps Shiprocket status labels to frozen actions.
// Matching is case-insensitive substring: provider labels vary by courier
// ("In Transit - Bag Added To Trip" is still in_transit). Unknown labels
// map to ignore — observation without a state move.
func mapShipmentStatus(raw string) courier.ShipmentAction {
	upper := strings.ToUpper(strings.TrimSpace(raw))
	// Order matters: UNDELIVERED contains DELIVERED, and PICKUP SCHEDULED
	// contains PICKUP — the longer/more-specific shapes match first.
	switch {
	case upper == "":
		return courier.ShipmentActionIgnore
	case containsAny(upper, "RTO DELIVERED", "RTO_DELIVERED", "RETURNED", "RTO COMPLETED"):
		return courier.ShipmentActionReturned
	case containsAny(upper, "RTO INITIATED", "RTO_IN_TRANSIT", "REVERSE"):
		return courier.ShipmentActionRTOInTransit
	case containsAny(upper, "UNDELIVERED", "NDR", "NOT DELIVERED", "FAILED ATTEMPT", "CONSIGNEE", "REFUSED"):
		return courier.ShipmentActionNDRPending
	case containsAny(upper, "DELIVERED"):
		return courier.ShipmentActionDelivered
	case containsAny(upper, "CANCELLED", "CANCELED"):
		return courier.ShipmentActionCancelled
	case containsAny(upper, "LOST", "DAMAGED"):
		return courier.ShipmentActionFailed
	case containsAny(upper, "PICKUP SCHEDULED", "PICKUP_GENERATED", "MANIFEST"):
		return courier.ShipmentActionPickupScheduled
	case containsAny(upper, "PICKED UP", "PICKEDUP", "PICKUP"):
		return courier.ShipmentActionPicked
	case containsAny(upper, "IN TRANSIT", "INTRANSIT", "SHIPPED", "BAGGED", "RECEIVED AT", "TRIP", "WEIGHT"):
		return courier.ShipmentActionInTransit
	default:
		return courier.ShipmentActionIgnore
	}
}

func containsAny(haystack string, needles ...string) bool {
	for _, needle := range needles {
		if strings.Contains(haystack, needle) {
			return true
		}
	}
	return false
}
