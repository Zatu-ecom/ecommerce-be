package singleton

import (
	"sync"

	fileSingleton "ecommerce-be/file/factory/singleton"
	filegw "ecommerce-be/file/gateway"
	"ecommerce-be/user/service"
)

// ServiceFactory manages all service singleton instances
type ServiceFactory struct {
	repoFactory *RepositoryFactory

	userService            service.UserService
	addressService         service.AddressService
	userQueryService       service.UserQueryService
	countryService         service.CountryService
	currencyService        service.CurrencyService
	countryCurrencyService service.CountryCurrencyService
	sellerSettingsService  service.SellerSettingsService
	sellerService          service.SellerService
	sellerProfileService   service.SellerProfileService

	once sync.Once
}

// NewServiceFactory creates a new service factory
func NewServiceFactory(repoFactory *RepositoryFactory) *ServiceFactory {
	return &ServiceFactory{repoFactory: repoFactory}
}

// initialize creates all service instances (lazy loading)
func (f *ServiceFactory) initialize() {
	f.once.Do(func() {
		userRepo := f.repoFactory.GetUserRepository()
		addressRepo := f.repoFactory.GetAddressRepository()
		countryRepo := f.repoFactory.GetCountryRepository()
		currencyRepo := f.repoFactory.GetCurrencyRepository()
		countryCurrencyRepo := f.repoFactory.GetCountryCurrencyRepository()
		sellerProfileRepo := f.repoFactory.GetSellerProfileRepository()
		sellerSettingsRepo := f.repoFactory.GetSellerSettingsRepository()

		displayFileGateway := filegw.NewDisplayGateway(
			fileSingleton.GetInstance().GetFileReadService(),
		)

		f.addressService = service.NewAddressService(addressRepo)
		f.userQueryService = service.NewUserQueryService(userRepo)
		f.countryService = service.NewCountryService(countryRepo)
		f.currencyService = service.NewCurrencyService(currencyRepo)
		f.countryCurrencyService = service.NewCountryCurrencyService(
			countryCurrencyRepo,
			countryRepo,
			currencyRepo,
		)
		f.sellerSettingsService = service.NewSellerSettingsService(
			sellerSettingsRepo,
			f.countryService,
			f.currencyService,
		)

		f.userService = service.NewUserService(
			userRepo,
			sellerProfileRepo,
			f.addressService,
			f.sellerSettingsService,
			f.currencyService,
			displayFileGateway,
		)
		f.sellerService = service.NewSellerService(
			f.userService,
			f.sellerSettingsService,
			userRepo,
			sellerProfileRepo,
			displayFileGateway,
		)
		f.sellerProfileService = service.NewSellerProfileService(
			userRepo,
			sellerProfileRepo,
			f.sellerSettingsService,
			displayFileGateway,
		)
	})
}

func (f *ServiceFactory) GetUserService() service.UserService {
	f.initialize()
	return f.userService
}

func (f *ServiceFactory) GetAddressService() service.AddressService {
	f.initialize()
	return f.addressService
}

func (f *ServiceFactory) GetUserQueryService() service.UserQueryService {
	f.initialize()
	return f.userQueryService
}

func (f *ServiceFactory) GetCountryService() service.CountryService {
	f.initialize()
	return f.countryService
}

func (f *ServiceFactory) GetCurrencyService() service.CurrencyService {
	f.initialize()
	return f.currencyService
}

func (f *ServiceFactory) GetCountryCurrencyService() service.CountryCurrencyService {
	f.initialize()
	return f.countryCurrencyService
}

func (f *ServiceFactory) GetSellerSettingsService() service.SellerSettingsService {
	f.initialize()
	return f.sellerSettingsService
}

func (f *ServiceFactory) GetSellerService() service.SellerService {
	f.initialize()
	return f.sellerService
}

func (f *ServiceFactory) GetSellerProfileService() service.SellerProfileService {
	f.initialize()
	return f.sellerProfileService
}
