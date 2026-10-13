package service

import (
	"context"
	"fmt"
	"strings"

	"ecommerce-be/common/config"
	"ecommerce-be/fulfillment/entity"
	fulfillmenterrors "ecommerce-be/fulfillment/error"
	"ecommerce-be/fulfillment/model"
	"ecommerce-be/fulfillment/repository"
	courier "ecommerce-be/fulfillment/service/courier"
)

// ProviderConfigService is the seller-facing courier dashboard: catalog,
// detail, configure, and credential probing. Secrets are encrypted before
// storage and masked on every read; TestConnection never persists.
type ProviderConfigService interface {
	ListForSeller(ctx context.Context, sellerID uint) ([]model.CourierSummaryResponse, error)
	GetByCode(ctx context.Context, sellerID uint, code string) (*model.CourierDetailResponse, error)
	Configure(ctx context.Context, sellerID uint, code string, req model.ConfigureCourierRequest) (*model.CourierConfigResponse, error)
	TestConnection(ctx context.Context, sellerID uint, code string, req model.TestCourierRequest) (*model.TestConnectionResponse, error)
}

// AdapterFactory is the subset of the courier factory the dashboard needs.
type AdapterFactory interface {
	GetByCode(code string) (courier.CourierPartner, error)
}

type ProviderConfigServiceImpl struct {
	providerRepo repository.CourierProviderRepository
	configRepo   repository.CourierProviderConfigRepository
	factory      AdapterFactory
}

// NewProviderConfigService builds the dashboard service.
func NewProviderConfigService(
	providerRepo repository.CourierProviderRepository,
	configRepo repository.CourierProviderConfigRepository,
	factory AdapterFactory,
) ProviderConfigService {
	return &ProviderConfigServiceImpl{
		providerRepo: providerRepo,
		configRepo:   configRepo,
		factory:      factory,
	}
}

// ListForSeller lists active providers with the seller's configured state.
// Config overlays stay live (no caching of seller state).
func (s *ProviderConfigServiceImpl) ListForSeller(
	ctx context.Context,
	sellerID uint,
) ([]model.CourierSummaryResponse, error) {
	providers, err := s.providerRepo.FindAllActive(ctx)
	if err != nil {
		return nil, err
	}
	configs, err := s.configRepo.FindBySeller(ctx, sellerID)
	if err != nil {
		return nil, err
	}
	configured := map[string]bool{}
	for i := range configs {
		if configs[i].IsActive {
			configured[configs[i].ProviderCode] = true
		}
	}

	responses := make([]model.CourierSummaryResponse, 0, len(providers))
	for _, p := range providers {
		responses = append(responses, model.CourierSummaryResponse{
			Code:             p.Code,
			Name:             p.Name,
			Kind:             p.Kind,
			SupportsPickup:   p.SupportsPickup,
			SupportsNDR:      p.SupportsNDR,
			SupportsReturn:   p.SupportsReturn,
			SupportsCOD:      p.SupportsCOD,
			WebhookSupported: p.WebhookSupported,
			Configured:       configured[p.Code],
			IsActive:         p.IsActive,
			WebhookURL:       courierWebhookURL(p.Code),
		})
	}
	return responses, nil
}

// GetByCode returns the credential form plus the seller's saved config for
// one provider. Production row wins the display; else sandbox; else null.
func (s *ProviderConfigServiceImpl) GetByCode(
	ctx context.Context,
	sellerID uint,
	code string,
) (*model.CourierDetailResponse, error) {
	provider, err := s.providerRepo.FindByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	adapter, err := s.factory.GetByCode(code)
	if err != nil {
		return nil, err
	}
	fields, err := s.providerRepo.FindFields(ctx, code)
	if err != nil {
		return nil, err
	}

	detail := &model.CourierDetailResponse{
		Code:   provider.Code,
		Name:   provider.Name,
		Fields: make([]model.CourierFieldResponse, 0, len(fields)),
	}
	for _, f := range fields {
		detail.Fields = append(detail.Fields, model.CourierFieldResponse{
			FieldName:    f.FieldName,
			DisplayName:  f.DisplayName,
			FieldType:    f.FieldType,
			IsRequired:   f.IsRequired,
			IsSensitive:  f.IsSensitive,
			DisplayOrder: f.DisplayOrder,
		})
	}

	for _, env := range []string{"production", "sandbox"} {
		config, err := s.configRepo.FindSellerConfig(ctx, sellerID, code, env)
		if err != nil {
			continue
		}
		detail.Config = configResponse(config, adapter)
		break
	}
	return detail, nil
}

// Configure saves or updates the seller's credentials for one provider and
// environment. Partial updates keep stored secrets for omitted keys
// (MergePartial); provided values replace them. Tenant flags follow the
// same provided-or-kept rule.
func (s *ProviderConfigServiceImpl) Configure(
	ctx context.Context,
	sellerID uint,
	code string,
	req model.ConfigureCourierRequest,
) (*model.CourierConfigResponse, error) {
	adapter, err := s.factory.GetByCode(code)
	if err != nil {
		return nil, err
	}
	if _, err := s.providerRepo.FindByCode(ctx, code); err != nil {
		return nil, err
	}

	existing, err := s.configRepo.FindSellerRow(ctx, sellerID, code, req.Environment)
	if err != nil && err != fulfillmenterrors.ErrorProviderNotConfigured {
		return nil, err
	}

	raw := req.Credentials
	if existing != nil {
		raw, err = adapter.MergePartial(existing.Credentials, req.Credentials)
		if err != nil {
			return nil, err
		}
	}
	if err := adapter.Validate(raw, false); err != nil {
		return nil, fulfillmenterrors.ErrorCredentialsInvalid.WithMessagef("%s", err.Error())
	}
	encrypted, err := adapter.Encrypt(raw)
	if err != nil {
		return nil, err
	}

	config := mergeConfigRow(existing, sellerID, code, req, encrypted)
	if existing != nil {
		if err := s.configRepo.Save(ctx, config); err != nil {
			return nil, fmt.Errorf("fulfillment configure save: %w", err)
		}
	} else {
		if err := s.configRepo.Create(ctx, config); err != nil {
			return nil, fmt.Errorf("fulfillment configure create: %w", err)
		}
	}
	return configResponse(config, adapter), nil
}

// mergeConfigRow overlays the request onto the existing row (or a fresh
// one): provided flags win, omitted flags keep stored values (or defaults).
func mergeConfigRow(
	existing *entity.CourierProviderConfig,
	sellerID uint,
	code string,
	req model.ConfigureCourierRequest,
	encrypted map[string]any,
) *entity.CourierProviderConfig {
	config := &entity.CourierProviderConfig{
		SellerID:     &sellerID,
		ProviderCode: code,
		Environment:  req.Environment,
		Credentials:  encrypted,
		IsActive:     true,
	}
	if existing != nil {
		config.ID = existing.ID
		config.CreatedAt = existing.CreatedAt
		config.IsActive = existing.IsActive
		config.AutoBook = existing.AutoBook
		config.RatePreference = existing.RatePreference
	}
	if req.IsActive != nil {
		config.IsActive = *req.IsActive
	}
	if req.AutoBook != nil {
		config.AutoBook = *req.AutoBook
	}
	if req.RatePreference != nil {
		config.RatePreference = req.RatePreference
	}
	return config
}

// TestConnection probes credentials without persisting anything. Supplied
// credentials are merged over the saved row (empty secrets keep stored
// values) for the probe only; omitted credentials test the saved row.
func (s *ProviderConfigServiceImpl) TestConnection(
	ctx context.Context,
	sellerID uint,
	code string,
	req model.TestCourierRequest,
) (*model.TestConnectionResponse, error) {
	adapter, err := s.factory.GetByCode(code)
	if err != nil {
		return nil, err
	}
	if _, err := s.providerRepo.FindByCode(ctx, code); err != nil {
		return nil, err
	}

	var creds map[string]any
	if len(req.Credentials) > 0 {
		existing, err := s.configRepo.FindSellerRow(ctx, sellerID, code, req.Environment)
		if err != nil && err != fulfillmenterrors.ErrorProviderNotConfigured {
			return nil, err
		}
		creds = req.Credentials
		if existing != nil {
			creds, err = adapter.MergePartial(existing.Credentials, req.Credentials)
			if err != nil {
				return nil, err
			}
		}
		if err := adapter.Validate(creds, false); err != nil {
			return nil, fulfillmenterrors.ErrorCredentialsInvalid.WithMessagef("%s", err.Error())
		}
	} else {
		config, err := s.configRepo.FindSellerConfig(ctx, sellerID, code, req.Environment)
		if err != nil {
			return nil, err
		}
		creds, err = adapter.Decrypt(config.Credentials)
		if err != nil {
			return nil, err
		}
	}

	if err := adapter.TestConnection(ctx, creds); err != nil {
		return nil, err
	}
	return &model.TestConnectionResponse{OK: true, Environment: req.Environment}, nil
}

// configResponse renders the dashboard-safe view of a saved row.
func configResponse(
	config *entity.CourierProviderConfig,
	adapter courier.CourierPartner,
) *model.CourierConfigResponse {
	return &model.CourierConfigResponse{
		Configured:     true,
		IsActive:       config.IsActive,
		AutoBook:       config.AutoBook,
		RatePreference: config.RatePreference,
		Environment:    config.Environment,
		ConfigHints:    adapter.MaskHints(config.Credentials),
	}
}

// courierWebhookURL composes the callback sellers paste into provider
// dashboards: {PUBLIC_API_BASE_URL}/api/fulfillment/webhooks/{code}.
func courierWebhookURL(code string) string {
	base := "http://localhost:8080"
	if cfg := config.Get(); cfg != nil && strings.TrimSpace(cfg.App.PublicAPIBaseURL) != "" {
		base = strings.TrimSuffix(strings.TrimSpace(cfg.App.PublicAPIBaseURL), "/")
	}
	return base + "/api/fulfillment/webhooks/" + code
}
