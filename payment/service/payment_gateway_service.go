package service

import (
	"context"
	"errors"
	"strings"

	"ecommerce-be/common/config"
	"ecommerce-be/common/filegateway"
	"ecommerce-be/payment/entity"
	paymenterrors "ecommerce-be/payment/error"
	"ecommerce-be/payment/factory"
	paymentModel "ecommerce-be/payment/model"
	"ecommerce-be/payment/repository"
	userService "ecommerce-be/user/service"
)

// PaymentGatewayService exposes the seller-facing gateway catalog dashboard.
type PaymentGatewayService interface {
	ListForSeller(ctx context.Context, sellerID uint) ([]paymentModel.GatewaySummaryResponse, error)
	GetByCode(ctx context.Context, sellerID uint, code string) (*paymentModel.GatewayDetailResponse, error)
	Configure(
		ctx context.Context,
		sellerID uint,
		code string,
		req paymentModel.ConfigureGatewayRequest,
	) error
	// Deactivate disables one environment row for a gateway.
	Deactivate(
		ctx context.Context,
		sellerID uint,
		code string,
		environment entity.GatewayEnvironment,
	) error
	// TestConnection probes credentials against the provider without
	// persisting anything. Omitted credentials fall back to the saved row.
	TestConnection(
		ctx context.Context,
		sellerID uint,
		code string,
		req paymentModel.TestGatewayRequest,
	) (*paymentModel.GatewayTestResponse, error)
}

// PaymentGatewayServiceImpl implements PaymentGatewayService.
type PaymentGatewayServiceImpl struct {
	gatewayRepo       repository.PaymentGatewayRepository
	gatewayFieldRepo  repository.PaymentGatewayFieldRepository
	gatewayConfigRepo repository.PaymentGatewayConfigRepository
	gatewayFactory    *factory.PaymentGatewayFactory
	userService       userService.UserService
	fileGateway       filegateway.FileDisplayGateway
}

// NewPaymentGatewayService creates the gateway service.
func NewPaymentGatewayService(
	gatewayRepo repository.PaymentGatewayRepository,
	gatewayFieldRepo repository.PaymentGatewayFieldRepository,
	gatewayConfigRepo repository.PaymentGatewayConfigRepository,
	gatewayFactory *factory.PaymentGatewayFactory,
	userService userService.UserService,
	fileGateway filegateway.FileDisplayGateway,
) PaymentGatewayService {
	return &PaymentGatewayServiceImpl{
		gatewayRepo:       gatewayRepo,
		gatewayFieldRepo:  gatewayFieldRepo,
		gatewayConfigRepo: gatewayConfigRepo,
		gatewayFactory:    gatewayFactory,
		userService:       userService,
		fileGateway:       fileGateway,
	}
}

// ListForSeller lists all active gateways with the seller's per-environment
// configured state, geo support, store mode, and composed webhook URL.
func (s *PaymentGatewayServiceImpl) ListForSeller(
	ctx context.Context,
	sellerID uint,
) ([]paymentModel.GatewaySummaryResponse, error) {
	gateways, err := s.gatewayRepo.FindAllActive(ctx)
	if err != nil {
		return nil, err
	}

	configs, err := s.gatewayConfigRepo.FindActiveBySeller(ctx, sellerID)
	if err != nil {
		return nil, err
	}
	activeByGateway := make(map[uint]map[entity.GatewayEnvironment]bool, len(configs))
	for i := range configs {
		c := &configs[i]
		envs, ok := activeByGateway[c.GatewayID]
		if !ok {
			envs = map[entity.GatewayEnvironment]bool{}
			activeByGateway[c.GatewayID] = envs
		}
		if c.IsActive {
			envs[c.Environment] = true
		}
	}

	mode := s.storePaymentsEnvironment(ctx, sellerID)

	responses := make([]paymentModel.GatewaySummaryResponse, 0, len(gateways))
	for i := range gateways {
		g := &gateways[i]
		envs := activeByGateway[g.ID]
		sandbox := envs[entity.EnvironmentSandbox]
		production := envs[entity.EnvironmentProduction]
		countries, err := s.listCountries(ctx, g.ID)
		if err != nil {
			return nil, err
		}
		currencies, err := s.listCurrencies(ctx, g.ID)
		if err != nil {
			return nil, err
		}
		responses = append(responses, paymentModel.GatewaySummaryResponse{
			Code:                    g.Code,
			Name:                    g.Name,
			Logo:                    filegateway.ResolveOptional(ctx, s.fileGateway, g.LogoFileID, &sellerID),
			SupportedCountries:      countries,
			SupportedCurrencies:     currencies,
			SupportedPaymentMethods: []string(g.SupportedPaymentMethods),
			Configured:              sandbox || production,
			ConfiguredSandbox:       sandbox,
			ConfiguredProduction:    production,
			PaymentsEnvironment:     mode,
			WebhookURL:              gatewayWebhookURL(g.Code),
		})
	}

	return responses, nil
}

// GetByCode returns full detail for one gateway: field schema plus both
// environment configs with masked hints (never secrets).
func (s *PaymentGatewayServiceImpl) GetByCode(
	ctx context.Context,
	sellerID uint,
	code string,
) (*paymentModel.GatewayDetailResponse, error) {
	g, err := s.gatewayRepo.FindByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	adapter, err := s.gatewayFactory.GetPaymentGatewayByCode(code)
	if err != nil {
		return nil, err
	}

	fields, err := s.gatewayFieldRepo.FindByGatewayID(ctx, g.ID)
	if err != nil {
		return nil, err
	}

	configs, err := s.gatewayConfigRepo.FindAllBySellerAndGateway(ctx, sellerID, g.ID)
	if err != nil {
		return nil, err
	}
	byEnv := make(map[entity.GatewayEnvironment]*entity.PaymentGatewayConfig, 2)
	for i := range configs {
		byEnv[configs[i].Environment] = &configs[i]
	}

	envConfigs := make(map[string]paymentModel.GatewayEnvConfigResponse, 2)
	for _, env := range []entity.GatewayEnvironment{
		entity.EnvironmentSandbox,
		entity.EnvironmentProduction,
	} {
		row := byEnv[env]
		entry := paymentModel.GatewayEnvConfigResponse{}
		if row != nil {
			entry.Configured = row.IsActive
			entry.IsActive = row.IsActive
			entry.Priority = row.Priority
			if row.IsActive {
				entry.ConfigHints = camelHintKeys(adapter.MaskHints(row.Credentials))
			}
		}
		envConfigs[string(env)] = entry
	}

	fieldResponses := make([]paymentModel.GatewayFieldResponse, 0, len(fields))
	for i := range fields {
		f := &fields[i]
		fieldResponses = append(fieldResponses, paymentModel.GatewayFieldResponse{
			FieldName:       f.FieldName,
			DisplayName:     f.DisplayName,
			FieldType:       string(f.FieldType),
			Description:     f.Description,
			Placeholder:     f.Placeholder,
			IsRequired:      f.IsRequired,
			IsSensitive:     f.IsSensitive,
			DisplayOrder:    f.DisplayOrder,
			ValidationRules: f.ValidationRules,
		})
	}

	sandbox := envConfigs[string(entity.EnvironmentSandbox)].Configured
	production := envConfigs[string(entity.EnvironmentProduction)].Configured
	countries, err := s.listCountries(ctx, g.ID)
	if err != nil {
		return nil, err
	}
	currencies, err := s.listCurrencies(ctx, g.ID)
	if err != nil {
		return nil, err
	}

	return &paymentModel.GatewayDetailResponse{
		GatewaySummaryResponse: paymentModel.GatewaySummaryResponse{
			Code:                    g.Code,
			Name:                    g.Name,
			Logo:                    filegateway.ResolveOptional(ctx, s.fileGateway, g.LogoFileID, &sellerID),
			SupportedCountries:      countries,
			SupportedCurrencies:     currencies,
			SupportedPaymentMethods: []string(g.SupportedPaymentMethods),
			Configured:              sandbox || production,
			ConfiguredSandbox:       sandbox,
			ConfiguredProduction:    production,
			PaymentsEnvironment:     s.storePaymentsEnvironment(ctx, sellerID),
			WebhookURL:              gatewayWebhookURL(g.Code),
		},
		Description: g.Description,
		Fields:      fieldResponses,
		Configs:     envConfigs,
	}, nil
}

// Configure validates, encrypts, and upserts the seller's credentials for one
// environment of a gateway. Partial updates keep stored secrets: empty or
// omitted sensitive keys are merged from the existing row via the adapter
// codec (Validate/MergePartial/Encrypt) — base services never parse provider
// fields. Missing encryption key fails closed: nothing is stored plaintext.
func (s *PaymentGatewayServiceImpl) Configure(
	ctx context.Context,
	sellerID uint,
	code string,
	req paymentModel.ConfigureGatewayRequest,
) error {
	g, err := s.gatewayRepo.FindByCode(ctx, code)
	if err != nil {
		return err
	}
	adapter, err := s.gatewayFactory.GetPaymentGatewayByCode(code)
	if err != nil {
		return err
	}

	env := entity.EnvironmentSandbox
	if req.Environment != "" {
		env = req.Environment
	}
	if env != entity.EnvironmentSandbox && env != entity.EnvironmentProduction {
		return paymenterrors.ErrorGatewayValidation.WithMessagef("[%s] invalid environment %q", code, req.Environment)
	}

	existing, err := s.gatewayConfigRepo.FindBySellerGatewayAndEnvironment(ctx, sellerID, g.ID, env)
	if err != nil {
		return err
	}

	raw := req.Credentials
	if existing != nil {
		raw, err = adapter.MergePartial(existing.Credentials, req.Credentials)
		if err != nil {
			if errors.Is(err, paymenterrors.ErrorEncryptionKeyMissing) {
				return err
			}
			return paymenterrors.ErrorGatewayValidation.WithMessagef("[%s] %v", code, err)
		}
	}
	if err := adapter.Validate(raw, false); err != nil {
		return paymenterrors.ErrorGatewayValidation.WithMessagef("[%s] %v", code, err)
	}

	encrypted, err := adapter.Encrypt(raw)
	if err != nil {
		return err
	}

	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	config := &entity.PaymentGatewayConfig{
		SellerID:    sellerID,
		GatewayID:   g.ID,
		Environment: env,
		Credentials: encrypted,
		IsActive:    isActive,
		Priority:    req.Priority,
	}

	return s.gatewayConfigRepo.UpsertConfig(ctx, config)
}

// Deactivate disables ONE environment row for a gateway; the other
// environment is untouched. Missing rows are a no-op (idempotent).
func (s *PaymentGatewayServiceImpl) Deactivate(
	ctx context.Context,
	sellerID uint,
	code string,
	environment entity.GatewayEnvironment,
) error {
	if environment != entity.EnvironmentSandbox && environment != entity.EnvironmentProduction {
		return paymenterrors.ErrorGatewayValidation.WithMessagef(
			"query environment is required (sandbox|production), got %q", environment)
	}
	g, err := s.gatewayRepo.FindByCode(ctx, code)
	if err != nil {
		return err
	}
	config, err := s.gatewayConfigRepo.FindBySellerGatewayAndEnvironment(ctx, sellerID, g.ID, environment)
	if err != nil {
		return err
	}
	if config == nil || !config.IsActive {
		return nil
	}
	return s.gatewayConfigRepo.DeactivateConfig(ctx, config.ID)
}

// TestConnection probes credentials against the provider without persisting
// anything. Omitted credentials test the saved row for the environment;
// provided values are merged over it for the probe only.
func (s *PaymentGatewayServiceImpl) TestConnection(
	ctx context.Context,
	sellerID uint,
	code string,
	req paymentModel.TestGatewayRequest,
) (*paymentModel.GatewayTestResponse, error) {
	g, err := s.gatewayRepo.FindByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	adapter, err := s.gatewayFactory.GetPaymentGatewayByCode(code)
	if err != nil {
		return nil, err
	}
	if req.Environment != entity.EnvironmentSandbox && req.Environment != entity.EnvironmentProduction {
		return nil, paymenterrors.ErrorGatewayValidation.WithMessagef(
			"environment is required (sandbox|production), got %q", req.Environment)
	}

	stored, err := s.gatewayConfigRepo.FindBySellerGatewayAndEnvironment(
		ctx, sellerID, g.ID, req.Environment)
	if err != nil {
		return nil, err
	}
	if stored == nil {
		return nil, paymenterrors.ErrorGatewayNotConfigured
	}

	raw := req.Credentials
	if len(raw) > 0 {
		raw, err = adapter.MergePartial(stored.Credentials, req.Credentials)
		if err != nil {
			if errors.Is(err, paymenterrors.ErrorEncryptionKeyMissing) {
				return nil, err
			}
			return nil, paymenterrors.ErrorGatewayValidation.WithMessagef("[%s] %v", code, err)
		}
	} else {
		raw, err = adapter.Decrypt(stored.Credentials)
		if err != nil {
			return nil, err
		}
	}

	if err := adapter.TestConnection(ctx, raw); err != nil {
		return nil, err
	}
	return &paymentModel.GatewayTestResponse{OK: true, Environment: string(req.Environment)}, nil
}

// ─── Internal helpers ────────────────────────────────────────────────────────

// storePaymentsEnvironment returns the seller's checkout mode, defaulting to
// sandbox when it cannot be resolved.
func (s *PaymentGatewayServiceImpl) storePaymentsEnvironment(ctx context.Context, sellerID uint) string {
	mode, err := s.userService.GetSellerPaymentsEnvironment(ctx, sellerID)
	if err != nil || (mode != string(entity.EnvironmentSandbox) && mode != string(entity.EnvironmentProduction)) {
		return string(entity.EnvironmentSandbox)
	}
	return mode
}

func (s *PaymentGatewayServiceImpl) listCountries(
	ctx context.Context,
	gatewayID uint,
) ([]paymentModel.GatewayCountryResponse, error) {
	rows, err := s.gatewayRepo.ListSupportedCountries(ctx, gatewayID)
	if err != nil {
		return nil, err
	}
	out := make([]paymentModel.GatewayCountryResponse, 0, len(rows))
	for _, r := range rows {
		out = append(out, paymentModel.GatewayCountryResponse{ID: r.ID, Code: r.Code, Name: r.Name})
	}
	return out, nil
}

func (s *PaymentGatewayServiceImpl) listCurrencies(
	ctx context.Context,
	gatewayID uint,
) ([]paymentModel.GatewayCurrencyResponse, error) {
	rows, err := s.gatewayRepo.ListSupportedCurrencies(ctx, gatewayID)
	if err != nil {
		return nil, err
	}
	out := make([]paymentModel.GatewayCurrencyResponse, 0, len(rows))
	for _, r := range rows {
		out = append(out, paymentModel.GatewayCurrencyResponse{
			ID: r.ID, Code: r.Code, Symbol: r.Symbol, DecimalDigits: r.DecimalDigits,
		})
	}
	return out, nil
}

// gatewayWebhookURL composes the provider callback address sellers paste into
// provider dashboards: {PUBLIC_API_BASE_URL}/api/payment/webhooks/{code}.
func gatewayWebhookURL(code string) string {
	base := "http://localhost:8080"
	if cfg := config.Get(); cfg != nil && strings.TrimSpace(cfg.App.PublicAPIBaseURL) != "" {
		base = strings.TrimSuffix(strings.TrimSpace(cfg.App.PublicAPIBaseURL), "/")
	}
	return base + "/api/payment/webhooks/" + code
}

// camelHintKeys converts adapter hint keys (storage snake_case) to dashboard
// lowerCamelCase generically: key_id → keyId, client_secret → clientSecret.
// No provider-specific key knowledge lives outside adapters.
func camelHintKeys(in map[string]any) map[string]any {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[toLowerCamel(k)] = v
	}
	return out
}

func toLowerCamel(s string) string {
	parts := strings.Split(s, "_")
	for i := 1; i < len(parts); i++ {
		if parts[i] != "" {
			parts[i] = strings.ToUpper(parts[i][:1]) + parts[i][1:]
		}
	}
	return strings.Join(parts, "")
}
