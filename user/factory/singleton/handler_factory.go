package singleton

import (
	"sync"

	"ecommerce-be/user/handler"
)

// HandlerFactory manages all handler singleton instances
type HandlerFactory struct {
	serviceFactory *ServiceFactory

	userHandler            *handler.UserHandler
	addressHandler         *handler.AddressHandler
	userQueryHandler       *handler.UserQueryHandler
	countryHandler         *handler.CountryHandler
	currencyHandler        *handler.CurrencyHandler
	countryCurrencyHandler *handler.CountryCurrencyHandler
	sellerHandler          *handler.SellerHandler

	once sync.Once
}

// NewHandlerFactory creates a new handler factory
func NewHandlerFactory(serviceFactory *ServiceFactory) *HandlerFactory {
	return &HandlerFactory{serviceFactory: serviceFactory}
}

// initialize creates all handler instances (lazy loading)
func (f *HandlerFactory) initialize() {
	f.once.Do(func() {
		f.userHandler = handler.NewUserHandler(f.serviceFactory.GetUserService())
		f.addressHandler = handler.NewAddressHandler(
			f.serviceFactory.GetAddressService(),
		)
		f.userQueryHandler = handler.NewUserQueryHandler(
			f.serviceFactory.GetUserQueryService(),
		)
		f.countryHandler = handler.NewCountryHandler(
			f.serviceFactory.GetCountryService(),
		)
		f.currencyHandler = handler.NewCurrencyHandler(
			f.serviceFactory.GetCurrencyService(),
		)
		f.countryCurrencyHandler = handler.NewCountryCurrencyHandler(
			f.serviceFactory.GetCountryCurrencyService(),
		)
		f.sellerHandler = handler.NewSellerHandler(
			f.serviceFactory.GetSellerService(),
			f.serviceFactory.GetSellerProfileService(),
		)
	})
}

// GetUserHandler returns the singleton user handler
func (f *HandlerFactory) GetUserHandler() *handler.UserHandler {
	f.initialize()
	return f.userHandler
}

// GetAddressHandler returns the singleton address handler
func (f *HandlerFactory) GetAddressHandler() *handler.AddressHandler {
	f.initialize()
	return f.addressHandler
}

// GetUserQueryHandler returns the singleton user query handler
func (f *HandlerFactory) GetUserQueryHandler() *handler.UserQueryHandler {
	f.initialize()
	return f.userQueryHandler
}

// GetCountryHandler returns the singleton country handler
func (f *HandlerFactory) GetCountryHandler() *handler.CountryHandler {
	f.initialize()
	return f.countryHandler
}

// GetCurrencyHandler returns the singleton currency handler
func (f *HandlerFactory) GetCurrencyHandler() *handler.CurrencyHandler {
	f.initialize()
	return f.currencyHandler
}

// GetCountryCurrencyHandler returns the singleton country-currency handler
func (f *HandlerFactory) GetCountryCurrencyHandler() *handler.CountryCurrencyHandler {
	f.initialize()
	return f.countryCurrencyHandler
}

// GetSellerHandler returns the singleton seller handler
func (f *HandlerFactory) GetSellerHandler() *handler.SellerHandler {
	f.initialize()
	return f.sellerHandler
}
