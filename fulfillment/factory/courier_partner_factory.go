package factory

import (
	"context"

	"ecommerce-be/fulfillment/entity"
	"ecommerce-be/fulfillment/repository"
	courier "ecommerce-be/fulfillment/service/courier"
)

// CourierPartnerFactory selects provider adapters by courier code and
// resolves hybrid credentials (seller row else platform default).
// Adding a new courier is purely additive: new adapter + seed rows + one
// registry entry. Orchestrators never branch on provider codes.
type CourierPartnerFactory struct {
	providerRepo repository.CourierProviderRepository
	configRepo   repository.CourierProviderConfigRepository
	registry     map[string]courier.CourierPartner
}

// NewCourierPartnerFactory builds the factory with the given adapters.
func NewCourierPartnerFactory(
	providerRepo repository.CourierProviderRepository,
	configRepo repository.CourierProviderConfigRepository,
	adapters ...courier.CourierPartner,
) *CourierPartnerFactory {
	registry := make(map[string]courier.CourierPartner, len(adapters))
	for _, adapter := range adapters {
		if adapter != nil {
			registry[adapter.Code()] = adapter
		}
	}
	return &CourierPartnerFactory{
		providerRepo: providerRepo,
		configRepo:   configRepo,
		registry:     registry,
	}
}

// GetByCode returns the adapter for a provider code.
func (f *CourierPartnerFactory) GetByCode(code string) (courier.CourierPartner, error) {
	adapter, ok := f.registry[code]
	if !ok {
		return nil, errProviderNotSupported(code)
	}
	return adapter, nil
}

// ResolvedCredentials bundles an adapter with the decrypted credentials of
// a config row. The decrypted map lives only in memory: it travels as a
// function argument, is never logged, and is never persisted. Config
// carries tenant flags (auto_book, rate_preference, default weight).
type ResolvedCredentials struct {
	Adapter  courier.CourierPartner
	ConfigID uint
	Config   *entity.CourierProviderConfig
	Creds    map[string]any
}

// ResolveConfig resolves (adapter + decrypted creds) for a frozen config
// row id — the path webhooks, tracking, cancel, and label take, since they
// must use the account that booked, not whatever is current.
func (f *CourierPartnerFactory) ResolveConfig(
	ctx context.Context,
	configID uint,
) (*ResolvedCredentials, error) {
	config, err := f.configRepo.FindByID(ctx, configID)
	if err != nil {
		return nil, err
	}
	adapter, err := f.GetByCode(config.ProviderCode)
	if err != nil {
		return nil, err
	}
	decrypted, err := adapter.Decrypt(config.Credentials)
	if err != nil {
		return nil, err
	}
	return &ResolvedCredentials{
		Adapter:  adapter,
		ConfigID: config.ID,
		Config:   config,
		Creds:    decrypted,
	}, nil
}

// ResolveForSeller resolves (adapter + decrypted creds + frozen config id)
// for a seller and provider code: the seller's active row for the
// environment wins, else the platform default (seller_id IS NULL).
func (f *CourierPartnerFactory) ResolveForSeller(
	ctx context.Context,
	sellerID uint,
	providerCode, environment string,
) (*ResolvedCredentials, error) {
	adapter, err := f.GetByCode(providerCode)
	if err != nil {
		return nil, err
	}

	provider, err := f.providerRepo.FindByCode(ctx, providerCode)
	if err != nil {
		return nil, err
	}
	if !provider.IsActive {
		return nil, errProviderNotActive(providerCode)
	}

	config, err := f.configRepo.FindSellerConfig(ctx, sellerID, providerCode, environment)
	if err != nil {
		config, err = f.configRepo.FindPlatformDefault(ctx, providerCode, environment)
		if err != nil {
			return nil, err
		}
	}

	decrypted, err := adapter.Decrypt(config.Credentials)
	if err != nil {
		return nil, err
	}

	return &ResolvedCredentials{
		Adapter:  adapter,
		ConfigID: config.ID,
		Config:   config,
		Creds:    decrypted,
	}, nil
}
