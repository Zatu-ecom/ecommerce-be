package singleton

import (
	"sync"

	"ecommerce-be/user/service"
)

// ServiceFactory manages all service singleton instances
type ServiceFactory struct {
	repoFactory *RepositoryFactory

	userService               service.UserService
	addressService            service.AddressService
	userQueryService          service.UserQueryService
	countryService            service.CountryService
	currencyService           service.CurrencyService
	countryCurrencyService    service.CountryCurrencyService
	sellerSettingsService     service.SellerSettingsService
	sellerService             service.SellerService
	sellerProfileService      service.SellerProfileService

	once sync.Once
}

// NewServiceFactory creates a new service factory
func NewServiceFactory(repoFactory *RepositoryFactory) *ServiceFactory {
	return &ServiceFactory{repoFactory: repoFactory}
}

// initialize creates all service instances (lazy loading)
func (f *ServiceFactory) initialize() {
	f.once.Do(func() {
		// Get repositories
		userRepo := f.repoFactory.GetUserRepository()
		addressRepo := f.repoFactory.GetAddressRepository()
		countryRepo := f.repoFactory.GetCountryRepository()
		currencyRepo := f.repoFactory.GetCurrencyRepository()
		countryCurrencyRepo := f.repoFactory.GetCountryCurrencyRepository()
		sellerProfileRepo := f.repoFactory.GetSellerProfileRepository()
		sellerSettingsRepo := f.repoFactory.GetSellerSettingsRepository()

		// Initialize services - order matters due to dependencies
		f.addressService = service.NewAddressService(
			addressRepo,
		)
		f.userQueryService = service.NewUserQueryService(
			userRepo,
		)
		f.countryService = service.NewCountryService(
			countryRepo,
		)
		f.currencyService = service.NewCurrencyService(
			currencyRepo,
		)
		f.countryCurrencyService = service.NewCountryCurrencyService(
			countryCurrencyRepo,
			countryRepo,
			currencyRepo,
		)
		// SellerSettingsService uses CountryService and CurrencyService
		f.sellerSettingsService = service.NewSellerSettingsService(
			sellerSettingsRepo,
			f.countryService,
			f.currencyService,
		)

		// Initialize UserService now that Address, Settings, and Currency are ready
		f.userService = service.NewUserService(
			userRepo,
			f.addressService,
			f.sellerSettingsService,
			f.currencyService,
		)
		// SellerService (registration) uses UserService and SellerSettingsService (SOLID)
		f.sellerService = service.NewSellerService(
			f.userService,
			f.sellerSettingsService,
			userRepo,
			sellerProfileRepo,
		)
		// SellerProfileService handles profile get/update operations
		f.sellerProfileService = service.NewSellerProfileService(
			userRepo,
			sellerProfileRepo,
			f.sellerSettingsService,
		)
	})
}

// GetCategoryService returns the singleton category service
func (f *ServiceFactory) GetUserService() service.UserService {
	f.initialize()
	return f.userService
}

// GetAttributeDefinitionService returns the singleton attribute service
func (f *ServiceFactory) GetAddressService() service.AddressService {
	f.initialize()
	return f.addressService
}

// GetUserQueryService returns the singleton user query service
func (f *ServiceFactory) GetUserQueryService() service.UserQueryService {
	f.initialize()
	return f.userQueryService
}

// GetCountryService returns the singleton country service
func (f *ServiceFactory) GetCountryService() service.CountryService {
	f.initialize()
	return f.countryService
}

// GetCurrencyService returns the singleton currency service
func (f *ServiceFactory) GetCurrencyService() service.CurrencyService {
	f.initialize()
	return f.currencyService
}

// GetCountryCurrencyService returns the singleton country-currency service
func (f *ServiceFactory) GetCountryCurrencyService() service.CountryCurrencyService {
	f.initialize()
	return f.countryCurrencyService
}

// GetSellerSettingsService returns the singleton seller settings service
func (f *ServiceFactory) GetSellerSettingsService() service.SellerSettingsService {
	f.initialize()
	return f.sellerSettingsService
}

// GetSellerService returns the singleton seller service
func (f *ServiceFactory) GetSellerService() service.SellerService {
	f.initialize()
	return f.sellerService
}

// GetSellerProfileService returns the singleton seller profile service
func (f *ServiceFactory) GetSellerProfileService() service.SellerProfileService {
	f.initialize()
	return f.sellerProfileService
}
