package singleton

import (
	"sync"

	"ecommerce-be/common/cachekit"
	fileSingleton "ecommerce-be/file/factory/singleton"
	filegw "ecommerce-be/file/gateway"
	usercache "ecommerce-be/user/cache"
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
	passwordResetService   service.PasswordResetService

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
		passwordResetRepo := f.repoFactory.GetPasswordResetRepository()

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

		f.passwordResetService = service.NewPasswordResetService(
			passwordResetRepo,
			userRepo,
		)

		// 012: attach cache strategies (nil-safe; flags gate at call time).
		f.wireCacheStrategies()
	})
}

// wireCacheStrategies builds user-module cache strategies from the shared
// cachekit defaults and attaches them to services.
func (f *ServiceFactory) wireCacheStrategies() {
	currencyCache := usercache.NewCurrencyCache(cachekit.DefaultCache(), nil)
	settingsCache := usercache.NewSettingsCache(cachekit.DefaultCache(), nil)
	geoCache := usercache.NewGeoCache(
		cachekit.DefaultCache(),
		cachekit.DefaultDurable(),
		nil,
	)
	if us, ok := f.userService.(*service.UserServiceImpl); ok {
		us.SetCurrencyCache(currencyCache)
	}
	if ss, ok := f.sellerSettingsService.(*service.SellerSettingsServiceImpl); ok {
		ss.SetCacheHooks(currencyCache, settingsCache)
	}
	if cs, ok := f.countryService.(*service.CountryServiceImpl); ok {
		cs.SetGeoCache(geoCache)
	}
	if cs, ok := f.currencyService.(*service.CurrencyServiceImpl); ok {
		cs.SetGeoCache(geoCache)
	}
	if ms, ok := f.countryCurrencyService.(*service.CountryCurrencyServiceImpl); ok {
		ms.SetGeoCache(geoCache)
	}
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

// GetPasswordResetService returns the singleton password reset service
func (f *ServiceFactory) GetPasswordResetService() service.PasswordResetService {
	f.initialize()
	return f.passwordResetService
}
