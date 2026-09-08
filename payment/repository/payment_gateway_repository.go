package repository

import (
	"context"
	"errors"

	"ecommerce-be/common/db"
	"ecommerce-be/payment/entity"
	paymenterrors "ecommerce-be/payment/error"

	"gorm.io/gorm"
)

type PaymentGatewayRepository interface {
	FindById(ctx context.Context, id uint) (*entity.PaymentGateway, error)
	FindByCode(ctx context.Context, code string) (*entity.PaymentGateway, error)
	FindAllActive(ctx context.Context) ([]entity.PaymentGateway, error)
	// SupportsCountry reports whether gateway serves the seller's business country.
	SupportsCountry(ctx context.Context, gatewayID, countryID uint) (bool, error)
	// SupportsCurrency reports whether gateway settles the seller's base currency.
	SupportsCurrency(ctx context.Context, gatewayID, currencyID uint) (bool, error)
	// ListSupportedCountries returns the gateway's served countries for display.
	ListSupportedCountries(ctx context.Context, gatewayID uint) ([]SupportedCountry, error)
	// ListSupportedCurrencies returns the gateway's settled currencies for display.
	ListSupportedCurrencies(ctx context.Context, gatewayID uint) ([]SupportedCurrency, error)
}

// SupportedCountry is a dashboard read model for gateway geo membership.
type SupportedCountry struct {
	ID   uint
	Code string
	Name string
}

// SupportedCurrency is a dashboard read model for gateway geo membership.
type SupportedCurrency struct {
	ID            uint
	Code          string
	Symbol        string
	DecimalDigits int
}

type PaymentGatewayRepositoryImpl struct{}

func NewPaymentGatewayRepository() PaymentGatewayRepository {
	return &PaymentGatewayRepositoryImpl{}
}

func (r *PaymentGatewayRepositoryImpl) FindById(
	ctx context.Context,
	id uint,
) (*entity.PaymentGateway, error) {
	var paymentGateway entity.PaymentGateway
	err := db.DB(ctx).Where("id = ?", id).First(&paymentGateway).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, paymenterrors.ErrorPaymentGatewayNotFound
		}
		return nil, err
	}

	return &paymentGateway, nil
}

func (r *PaymentGatewayRepositoryImpl) FindByCode(
	ctx context.Context,
	code string,
) (*entity.PaymentGateway, error) {
	var paymentGateway entity.PaymentGateway
	err := db.DB(ctx).Where("code = ?", code).First(&paymentGateway).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, paymenterrors.ErrorPaymentGatewayNotFound
		}
		return nil, err
	}

	return &paymentGateway, nil
}

func (r *PaymentGatewayRepositoryImpl) FindAllActive(
	ctx context.Context,
) ([]entity.PaymentGateway, error) {
	var gateways []entity.PaymentGateway
	err := db.DB(ctx).
		Where("is_active = ?", true).
		Order("name ASC").
		Find(&gateways).Error
	if err != nil {
		return nil, err
	}

	return gateways, nil
}

// SupportsCountry checks geo membership via the payment_gateway_country join
// table (indexed EXISTS — no array scans as gateways/countries grow).
func (r *PaymentGatewayRepositoryImpl) SupportsCountry(
	ctx context.Context,
	gatewayID, countryID uint,
) (bool, error) {
	var exists bool
	err := db.DB(ctx).
		Model(&entity.PaymentGatewayCountry{}).
		Select("COUNT(*) > 0").
		Where("gateway_id = ? AND country_id = ?", gatewayID, countryID).
		Scan(&exists).Error
	if err != nil {
		return false, err
	}
	return exists, nil
}

// SupportsCurrency checks geo membership via the payment_gateway_currency
// join table (indexed EXISTS).
func (r *PaymentGatewayRepositoryImpl) SupportsCurrency(
	ctx context.Context,
	gatewayID, currencyID uint,
) (bool, error) {
	var exists bool
	err := db.DB(ctx).
		Model(&entity.PaymentGatewayCurrency{}).
		Select("COUNT(*) > 0").
		Where("gateway_id = ? AND currency_id = ?", gatewayID, currencyID).
		Scan(&exists).Error
	if err != nil {
		return false, err
	}
	return exists, nil
}

// ListSupportedCountries returns the gateway's served countries for display.
func (r *PaymentGatewayRepositoryImpl) ListSupportedCountries(
	ctx context.Context,
	gatewayID uint,
) ([]SupportedCountry, error) {
	var countries []SupportedCountry
	err := db.DB(ctx).
		Table("payment_gateway_country AS gc").
		Select("c.id, c.code, c.name").
		Joins("JOIN country AS c ON c.id = gc.country_id").
		Where("gc.gateway_id = ?", gatewayID).
		Order("c.code ASC").
		Scan(&countries).Error
	if err != nil {
		return nil, err
	}
	return countries, nil
}

// ListSupportedCurrencies returns the gateway's settled currencies for display.
func (r *PaymentGatewayRepositoryImpl) ListSupportedCurrencies(
	ctx context.Context,
	gatewayID uint,
) ([]SupportedCurrency, error) {
	var currencies []SupportedCurrency
	err := db.DB(ctx).
		Table("payment_gateway_currency AS gc").
		Select("c.id, c.code, c.symbol, c.decimal_digits").
		Joins("JOIN currency AS c ON c.id = gc.currency_id").
		Where("gc.gateway_id = ?", gatewayID).
		Order("c.code ASC").
		Scan(&currencies).Error
	if err != nil {
		return nil, err
	}
	return currencies, nil
}
