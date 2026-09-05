package service

import (
	"context"

	"ecommerce-be/common/filegateway"
	"ecommerce-be/payment/entity"
	paymenterrors "ecommerce-be/payment/error"
	paymentModel "ecommerce-be/payment/model"
	"ecommerce-be/payment/repository"
	gateway "ecommerce-be/payment/service/payment_gateway"
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
	Deactivate(ctx context.Context, sellerID uint, code string) error
}

// PaymentGatewayServiceImpl implements PaymentGatewayService.
type PaymentGatewayServiceImpl struct {
	gatewayRepo       repository.PaymentGatewayRepository
	gatewayFieldRepo  repository.PaymentGatewayFieldRepository
	gatewayConfigRepo repository.PaymentGatewayConfigRepository
	fileGateway       filegateway.FileDisplayGateway
}

// NewPaymentGatewayService creates the gateway service.
func NewPaymentGatewayService(
	gatewayRepo repository.PaymentGatewayRepository,
	gatewayFieldRepo repository.PaymentGatewayFieldRepository,
	gatewayConfigRepo repository.PaymentGatewayConfigRepository,
	fileGateway filegateway.FileDisplayGateway,
) PaymentGatewayService {
	return &PaymentGatewayServiceImpl{
		gatewayRepo:       gatewayRepo,
		gatewayFieldRepo:  gatewayFieldRepo,
		gatewayConfigRepo: gatewayConfigRepo,
		fileGateway:       fileGateway,
	}
}

// ListForSeller lists all active gateways with the seller's configured state.
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
	configuredByGateway := make(map[uint]string, len(configs))
	for _, c := range configs {
		configuredByGateway[c.GatewayID] = string(c.Environment)
	}

	responses := make([]paymentModel.GatewaySummaryResponse, 0, len(gateways))
	for i := range gateways {
		g := &gateways[i]
		env, configured := configuredByGateway[g.ID]
		responses = append(responses, paymentModel.GatewaySummaryResponse{
			Code:                    g.Code,
			Name:                    g.Name,
			Logo:                    filegateway.ResolveOptional(ctx, s.fileGateway, g.LogoFileID, &sellerID),
			SupportedCountries:      []string(g.SupportedCountries),
			SupportedCurrencies:     []string(g.SupportedCurrencies),
			SupportedPaymentMethods: []string(g.SupportedPaymentMethods),
			Configured:              configured,
			Environment:             env,
		})
	}

	return responses, nil
}

// GetByCode returns full detail for one gateway, including its config fields.
func (s *PaymentGatewayServiceImpl) GetByCode(
	ctx context.Context,
	sellerID uint,
	code string,
) (*paymentModel.GatewayDetailResponse, error) {
	g, err := s.gatewayRepo.FindByCode(ctx, code)
	if err != nil {
		return nil, err
	}

	fields, err := s.gatewayFieldRepo.FindByGatewayID(ctx, g.ID)
	if err != nil {
		return nil, err
	}

	config, err := s.gatewayConfigRepo.FindBySellerAndGateway(ctx, sellerID, g.ID)
	if err != nil {
		return nil, err
	}
	configured := config != nil && config.IsActive

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

	env := ""
	if configured {
		env = string(config.Environment)
	}

	return &paymentModel.GatewayDetailResponse{
		GatewaySummaryResponse: paymentModel.GatewaySummaryResponse{
			Code:                    g.Code,
			Name:                    g.Name,
			Logo:                    filegateway.ResolveOptional(ctx, s.fileGateway, g.LogoFileID, &sellerID),
			SupportedCountries:      []string(g.SupportedCountries),
			SupportedCurrencies:     []string(g.SupportedCurrencies),
			SupportedPaymentMethods: []string(g.SupportedPaymentMethods),
			Configured:              configured,
			Environment:             env,
		},
		Description: g.Description,
		Fields:      fieldResponses,
	}, nil
}

// Configure validates, encrypts, and upserts the seller's credentials for a gateway.
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

	// Validate required credentials for the gateway (Razorpay: key_id/key_secret/webhook_secret).
	creds, err := gateway.ParseRazorpayCredentials(req.Credentials)
	if err != nil {
		return paymenterrors.ErrorGatewayValidation.WithMessagef("[%s] %v", code, err)
	}

	encrypted, err := gateway.EncryptSensitive(creds)
	if err != nil {
		return err
	}

	env := entity.EnvironmentSandbox
	if req.Environment != "" {
		env = req.Environment
	}
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}
	country := "IN" // Razorpay is India-only.
	if c, ok := req.Credentials["country"].(string); ok && c != "" {
		country = c
	}

	config := &entity.PaymentGatewayConfig{
		SellerID:    sellerID,
		GatewayID:   g.ID,
		Environment: env,
		Credentials: encrypted,
		IsActive:    isActive,
		Priority:    req.Priority,
		Country:     country,
	}

	return s.gatewayConfigRepo.UpsertConfig(ctx, config)
}

// Deactivate disables the seller's config for a gateway.
func (s *PaymentGatewayServiceImpl) Deactivate(
	ctx context.Context,
	sellerID uint,
	code string,
) error {
	g, err := s.gatewayRepo.FindByCode(ctx, code)
	if err != nil {
		return err
	}
	config, err := s.gatewayConfigRepo.FindBySellerAndGateway(ctx, sellerID, g.ID)
	if err != nil {
		return err
	}
	if config == nil {
		return nil // idempotent: nothing to deactivate.
	}
	return s.gatewayConfigRepo.DeactivateConfig(ctx, config.ID)
}
