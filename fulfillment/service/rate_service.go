package service

import (
	"context"
	"crypto/sha256"
	"fmt"
	"time"

	"ecommerce-be/common/cachekit"
	"ecommerce-be/common/log"
	"ecommerce-be/fulfillment/cache"
	fulfillmenterrors "ecommerce-be/fulfillment/error"
	fulfillmentfactory "ecommerce-be/fulfillment/factory"
	"ecommerce-be/fulfillment/model"
	"ecommerce-be/fulfillment/repository"
)

// rateLimitPerMinute caps rate-shopping per seller (generous; counting via
// durable Lua, never cached). Reads fail open on KV outage.
const rateLimitPerMinute = 60

// RateService shops live courier rates with volatile caching, singleflight
// collapse, and per-seller throttling. Rates are never stored in PostgreSQL.
type RateService interface {
	GetRates(ctx context.Context, sellerID uint, req model.RatesRequest) ([]model.RateOptionResponse, error)
	// CheckServiceable verifies a pickup→drop lane serves at least one
	// courier (confirm-address gate). Empty options mean unserviceable.
	CheckServiceable(ctx context.Context, sellerID uint, pickupLocationID uint, orderID uint) error
}

// RateServiceImpl implements RateService.
type RateServiceImpl struct {
	factory    *fulfillmentfactory.CourierPartnerFactory
	configRepo repository.CourierProviderConfigRepository
	orderHooks FulfillmentOrderHooks
	inventory  FulfillmentInventoryHooks
	cache      cachekit.Cache
	durable    cachekit.Durable
	flight     *cachekit.Flight
}

// NewRateService builds the rate service. Cache/durable may be nil (tests,
// unwired envs): reads degrade to direct provider calls, limiting to
// fail-open counting.
func NewRateService(
	factory *fulfillmentfactory.CourierPartnerFactory,
	configRepo repository.CourierProviderConfigRepository,
	orderHooks FulfillmentOrderHooks,
	inventory FulfillmentInventoryHooks,
	cache cachekit.Cache,
	durable cachekit.Durable,
) RateService {
	return &RateServiceImpl{
		factory:    factory,
		configRepo: configRepo,
		orderHooks: orderHooks,
		inventory:  inventory,
		cache:      cache,
		durable:    durable,
		flight:     &cachekit.Flight{},
	}
}

// GetRates resolves the provider, throttles, then serves cached or live rates.
func (s *RateServiceImpl) GetRates(
	ctx context.Context,
	sellerID uint,
	req model.RatesRequest,
) ([]model.RateOptionResponse, error) {
	view, err := s.orderHooks.GetOrderForFulfillment(ctx, req.OrderID)
	if err != nil {
		return nil, err
	}
	if view.SellerID != sellerID {
		return nil, fulfillmenterrors.ErrorFulfillmentNotFound
	}
	resolved, err := s.resolveRateProvider(ctx, sellerID, req.ProviderCode)
	if err != nil {
		return nil, err
	}
	pickup, delivery, err := s.lanePincodes(ctx, sellerID, view, req.PickupLocationID)
	if err != nil {
		return nil, err
	}
	if err := s.checkLimit(ctx, sellerID); err != nil {
		return nil, err
	}

	input := model.RateInput{
		PickupPincode:   pickup,
		DeliveryPincode: delivery,
		WeightGrams:     req.WeightGrams,
		CodCents:        view.CodCents,
		CurrencyCode:    view.CurrencyCode,
	}
	if req.LengthCm != nil {
		input.LengthCm = *req.LengthCm
	}
	if req.BreadthCm != nil {
		input.BreadthCm = *req.BreadthCm
	}
	if req.HeightCm != nil {
		input.HeightCm = *req.HeightCm
	}

	options, err := s.cachedOrLive(ctx, sellerID, resolved, view, input)
	if err != nil {
		return nil, err
	}
	currency := view.CurrencyCode
	if currency == "" {
		currency = "INR"
	}
	responses := make([]model.RateOptionResponse, 0, len(options))
	for _, option := range options {
		responses = append(responses, model.ToRateOptionResponse(option, currency))
	}
	return responses, nil
}

// resolveRateProvider picks the provider: explicit code wins; empty code
// needs exactly one active seller courier, else the client must specify.
func (s *RateServiceImpl) resolveRateProvider(
	ctx context.Context,
	sellerID uint,
	code string,
) (*fulfillmentfactory.ResolvedCredentials, error) {
	if code != "" {
		return s.resolveEnv(ctx, sellerID, code)
	}
	configs, err := s.configRepo.FindBySeller(ctx, sellerID)
	if err != nil {
		return nil, err
	}
	codes := map[string]bool{}
	for _, config := range configs {
		if config.IsActive {
			codes[config.ProviderCode] = true
		}
	}
	if len(codes) == 0 {
		return nil, fulfillmenterrors.ErrorProviderNotConfigured
	}
	if len(codes) > 1 {
		return nil, fulfillmenterrors.ErrorFulfillmentInvalidState.WithMessagef(
			"specify providerCode: %d couriers active", len(codes))
	}
	for only := range codes {
		return s.resolveEnv(ctx, sellerID, only)
	}
	return nil, fulfillmenterrors.ErrorProviderNotConfigured
}

// resolveEnv tries production, then sandbox (same rule as booking).
func (s *RateServiceImpl) resolveEnv(
	ctx context.Context,
	sellerID uint,
	code string,
) (*fulfillmentfactory.ResolvedCredentials, error) {
	resolved, err := s.factory.ResolveForSeller(ctx, sellerID, code, "production")
	if err == nil {
		return resolved, nil
	}
	if err != fulfillmenterrors.ErrorProviderNotConfigured {
		return nil, err
	}
	return s.factory.ResolveForSeller(ctx, sellerID, code, "sandbox")
}

// lanePincodes resolves pickup (warehouse) and delivery pincodes from ids
// via hooks — the shipment stores ids only, never pincode columns.
func (s *RateServiceImpl) lanePincodes(
	ctx context.Context,
	sellerID uint,
	view *model.FulfillmentOrderView,
	pickupLocationID uint,
) (string, string, error) {
	variantIDs := make([]uint, 0, len(view.Items))
	for _, line := range view.Items {
		if line.VariantID != nil {
			variantIDs = append(variantIDs, *line.VariantID)
		}
	}
	rows, err := s.inventory.GetAvailability(ctx, sellerID, view.OrderID, variantIDs)
	if err != nil {
		return "", "", err
	}
	for _, row := range rows {
		if row.LocationID == pickupLocationID && row.Pincode != "" {
			return row.Pincode, view.DeliveryPincode, nil
		}
	}
	return "", "", fulfillmenterrors.ErrorFulfillmentInvalidState.WithMessagef(
		"pickup location has no availability")
}

// checkLimit throttles rate shopping per seller (durable Lua counter,
// fail-open on KV outage).
func (s *RateServiceImpl) checkLimit(ctx context.Context, sellerID uint) error {
	if s.durable == nil {
		return nil
	}
	key, err := rateLimitKey(sellerID)
	if err != nil {
		return nil
	}
	count, err := s.durable.IncrWithExpire(ctx, key, time.Minute)
	if err != nil {
		log.ErrorWithContext(ctx, "rates: limiter unavailable, failing open", err)
		return nil
	}
	if count > rateLimitPerMinute {
		return fulfillmenterrors.ErrorRateLimited
	}
	return nil
}

func rateLimitKey(sellerID uint) (string, error) {
	if err := cachekit.ValidateKey(cache.RateLimitKey(sellerID)); err != nil {
		return "", err
	}
	return cache.RateLimitKey(sellerID), nil
}

// cachedOrLive serves volatile-cached rates with singleflight collapse.
// Cache outage degrades to a direct provider call (fail-open reads).
func (s *RateServiceImpl) cachedOrLive(
	ctx context.Context,
	sellerID uint,
	resolved *fulfillmentfactory.ResolvedCredentials,
	view *model.FulfillmentOrderView,
	input model.RateInput,
) ([]model.RateOption, error) {
	key, err := rateCacheKey(sellerID, resolved, input)
	if err != nil || s.cache == nil {
		return s.fetchRates(ctx, resolved, input)
	}
	if raw, err := s.cache.Get(ctx, key); err == nil {
		var options []model.RateOption
		if jsonErr := cachekit.Unmarshal(raw, &options); jsonErr == nil {
			return options, nil
		}
	}
	combined, _, err := s.flight.Do(key, func() (any, error) {
		options, err := s.fetchRates(ctx, resolved, input)
		if err != nil {
			return nil, err
		}
		if raw, err := cachekit.Marshal(options); err == nil {
			// Async bounded SET: the response path never waits (cachekit).
			_ = s.cache.Set(ctx, key, raw, cache.RateTTL)
		}
		return options, nil
	})
	if err != nil {
		return nil, err
	}
	options, _ := combined.([]model.RateOption)
	return options, nil
}

// fetchRates calls the provider directly (miss path).
func (s *RateServiceImpl) fetchRates(
	ctx context.Context,
	resolved *fulfillmentfactory.ResolvedCredentials,
	input model.RateInput,
) ([]model.RateOption, error) {
	return resolved.Adapter.GetRates(ctx, input, resolved.Creds)
}

// rateCacheKey hashes the full rate request into a seller-scoped key.
func rateCacheKey(
	sellerID uint,
	resolved *fulfillmentfactory.ResolvedCredentials,
	input model.RateInput,
) (string, error) {
	sum := sha256.Sum256([]byte(fmt.Sprintf("%d|%s|%s|%s|%d|%f|%f|%f|%d",
		sellerID, resolved.Adapter.Code(), input.PickupPincode, input.DeliveryPincode,
		input.WeightGrams, input.LengthCm, input.BreadthCm, input.HeightCm, input.CodCents)))
	return cache.RateKey(sellerID, fmt.Sprintf("%x", sum))
}

// CheckServiceable verifies a pickup→drop lane serves at least one courier.
// Used by the confirm-address gate; empty options mean unserviceable.
func (s *RateServiceImpl) CheckServiceable(
	ctx context.Context,
	sellerID uint,
	pickupLocationID uint,
	orderID uint,
) error {
	view, err := s.orderHooks.GetOrderForFulfillment(ctx, orderID)
	if err != nil {
		return err
	}
	if view.SellerID != sellerID {
		return fulfillmenterrors.ErrorFulfillmentNotFound
	}
	weight := 500
	options, err := s.GetRates(ctx, sellerID, model.RatesRequest{
		OrderID:          orderID,
		PickupLocationID: pickupLocationID,
		WeightGrams:      weight,
	})
	if err != nil {
		return err
	}
	if len(options) == 0 {
		return fulfillmenterrors.ErrorFulfillmentInvalidState.WithMessagef(
			"new address is not serviceable")
	}
	return nil
}
