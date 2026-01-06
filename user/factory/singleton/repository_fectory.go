package singleton

import (
	"sync"

	"ecommerce-be/common/db"
	"ecommerce-be/user/repository"
)

// RepositoryFactory manages all repository singleton instances
// Note: DB is fetched dynamically via db.GetDB() to support test scenarios
// where database connections change between test runs
type RepositoryFactory struct {
	userRepo            repository.UserRepository
	addressRepo         repository.AddressRepository
	countryRepo         repository.CountryRepository
	currencyRepo        repository.CurrencyRepository
	countryCurrencyRepo repository.CountryCurrencyRepository
	sellerProfileRepo   repository.SellerProfileRepository
	sellerSettingsRepo  repository.SellerSettingsRepository
	once                sync.Once
}

// NewRepositoryFactory creates a new repository factory
func NewRepositoryFactory() *RepositoryFactory {
	return &RepositoryFactory{}
}

// initialize creates all repository instances (lazy loading)
// Uses db.GetDB() to fetch current database connection dynamically
func (f *RepositoryFactory) initialize() {
	f.once.Do(func() {
		currentDB := db.GetDB()
		f.userRepo = repository.NewUserRepository()
		f.addressRepo = repository.NewAddressRepository(currentDB)
		f.countryRepo = repository.NewCountryRepository()
		f.currencyRepo = repository.NewCurrencyRepository()
		f.countryCurrencyRepo = repository.NewCountryCurrencyRepository()
		f.sellerProfileRepo = repository.NewSellerProfileRepository()
		f.sellerSettingsRepo = repository.NewSellerSettingsRepository()
	})
}

// GetUserRepository returns the singleton user repository
func (f *RepositoryFactory) GetUserRepository() repository.UserRepository {
	f.initialize()
	return f.userRepo
}

// GetAddressRepository returns the singleton address repository
func (f *RepositoryFactory) GetAddressRepository() repository.AddressRepository {
	f.initialize()
	return f.addressRepo
}

// GetCountryRepository returns the singleton country repository
func (f *RepositoryFactory) GetCountryRepository() repository.CountryRepository {
	f.initialize()
	return f.countryRepo
}

// GetCurrencyRepository returns the singleton currency repository
func (f *RepositoryFactory) GetCurrencyRepository() repository.CurrencyRepository {
	f.initialize()
	return f.currencyRepo
}

// GetCountryCurrencyRepository returns the singleton country-currency repository
func (f *RepositoryFactory) GetCountryCurrencyRepository() repository.CountryCurrencyRepository {
	f.initialize()
	return f.countryCurrencyRepo
}

// GetSellerProfileRepository returns the singleton seller profile repository
func (f *RepositoryFactory) GetSellerProfileRepository() repository.SellerProfileRepository {
	f.initialize()
	return f.sellerProfileRepo
}

// GetSellerSettingsRepository returns the singleton seller settings repository
func (f *RepositoryFactory) GetSellerSettingsRepository() repository.SellerSettingsRepository {
	f.initialize()
	return f.sellerSettingsRepo
}
