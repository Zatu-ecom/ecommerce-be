package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"ecommerce-be/common"
	"ecommerce-be/common/constants"
	"ecommerce-be/file/entity"
	fileError "ecommerce-be/file/error"
	"ecommerce-be/file/model"
	"ecommerce-be/file/repository"
	"ecommerce-be/file/service/blobAdapter"
	"ecommerce-be/file/utils/constant"

	"gorm.io/gorm"
)

type ConfigService interface {
	GetProviders(
		ctx context.Context,
	) ([]model.ProviderResponse, error)
	SaveConfig(
		ctx context.Context,
		userID uint,
		role string,
		req model.SaveConfigRequest,
	) (*model.ConfigResponse, error)
	UpdateConfig(
		ctx context.Context,
		userID uint,
		role string,
		configID uint,
		req model.UpdateStorageConfigRequest,
	) (*model.ConfigResponse, error)
	TestStorageConfig(
		ctx context.Context,
		req model.SaveConfigRequest,
	) (*model.TestStorageConfigResponse, error)
	ListConfigs(
		ctx context.Context,
		filter model.ListStorageConfigFilter,
	) (*model.ListStorageConfigsResponse, error)
}

type configService struct {
	configRepo repository.ConfigRepository
}

func NewConfigService(configRepo repository.ConfigRepository) ConfigService {
	return &configService{
		configRepo: configRepo,
	}
}

func (s *configService) GetProviders(
	ctx context.Context,
) ([]model.ProviderResponse, error) {
	providers, err := s.configRepo.GetProviders(ctx)
	if err != nil {
		return nil, fileError.ErrPersistenceFailed.WithMessagef(
			"Failed to fetch providers: %v",
			err,
		)
	}

	res := make([]model.ProviderResponse, len(providers))
	for i, p := range providers {
		res[i] = model.MapProviderToResponse(p)
	}

	return res, nil
}

func (s *configService) SaveConfig(
	ctx context.Context,
	userID uint,
	role string,
	req model.SaveConfigRequest,
) (*model.ConfigResponse, error) {
	isSeller, err := s.resolveRole(role)
	if err != nil {
		return nil, err
	}

	// Load the provider to get its adapter_type for validation routing.
	provider, err := s.ensureProviderIsActive(ctx, req.ProviderID)
	if err != nil {
		return nil, err
	}

	// Parse and validate the config using the adapter's typed config struct.
	encryptedData, err := s.validateAndEncryptConfig(provider.AdapterType, req.Config)
	if err != nil {
		return nil, err
	}

	cfg := &entity.StorageConfig{}
	isActive := boolPtrOrDefault(req.IsActive, true)
	isDefault := boolPtrOrDefault(req.IsDefault, true)
	cfg.IsActive = isActive

	s.applyCoreConfigFields(cfg, req.ProviderID, req.DisplayName, req.BucketOrContainer)
	s.applyOwnership(cfg, userID, isSeller, isDefault)

	cfg.ConfigData = encryptedData

	s.applyTimestamps(cfg)

	if err := s.saveConfig(ctx, cfg); err != nil {
		return nil, err
	}

	res := model.MapConfigToResponse(*cfg)
	return &res, nil
}

func (s *configService) TestStorageConfig(
	ctx context.Context,
	req model.SaveConfigRequest,
) (*model.TestStorageConfigResponse, error) {
	provider, err := s.ensureProviderIsActive(ctx, req.ProviderID)
	if err != nil {
		return nil, err
	}
	parser, err := blobAdapter.GetBlobConfigParser(provider.AdapterType)
	if err != nil {
		return nil, err
	}
	cfg, err := parser.ParseAndValidateConfig(req.Config)
	if err != nil {
		return nil, err
	}
	if err := ensureConfigBucketMatchesRequest(
		provider.AdapterType,
		cfg,
		req.BucketOrContainer,
	); err != nil {
		return nil, err
	}

	testCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()

	adapter, err := blobAdapter.GetAdapter(testCtx, provider.AdapterType, cfg.ToMap())
	if err != nil {
		return nil, err
	}
	if err := adapter.PingStorage(testCtx, req.BucketOrContainer); err != nil {
		return nil, err
	}

	return &model.TestStorageConfigResponse{OK: true}, nil
}

func (s *configService) UpdateConfig(
	ctx context.Context,
	userID uint,
	role string,
	configID uint,
	req model.UpdateStorageConfigRequest,
) (*model.ConfigResponse, error) {
	isSeller, err := s.resolveRole(role)
	if err != nil {
		return nil, err
	}

	provider, err := s.ensureProviderIsActive(ctx, req.ProviderID)
	if err != nil {
		return nil, err
	}

	encryptedData, err := s.validateAndEncryptConfig(provider.AdapterType, req.Config)
	if err != nil {
		return nil, err
	}

	cfg, err := s.resolveTargetConfig(ctx, &configID, userID, isSeller)
	if err != nil {
		return nil, err
	}

	s.applyCoreConfigFields(cfg, req.ProviderID, req.DisplayName, req.BucketOrContainer)
	cfg.IsActive = req.IsActive
	cfg.IsDefault = req.IsDefault
	cfg.ConfigData = encryptedData

	s.applyTimestamps(cfg)

	if err := s.saveConfig(ctx, cfg); err != nil {
		return nil, err
	}

	res := model.MapConfigToResponse(*cfg)
	return &res, nil
}

func boolPtrOrDefault(p *bool, def bool) bool {
	if p == nil {
		return def
	}
	return *p
}

func ensureConfigBucketMatchesRequest(
	adapterType entity.AdapterType,
	cfg blobAdapter.BlobConfig,
	bucketOrContainer string,
) error {
	want := strings.TrimSpace(bucketOrContainer)
	if want == "" {
		return fileError.ErrBlobValidation.WithMessagef("bucket_or_container is required")
	}

	m := cfg.ToMap()
	switch adapterType {
	case entity.AdapterTypeS3Compatible, entity.AdapterTypeGCS:
		got, _ := m["bucket"].(string)
		if strings.TrimSpace(got) != want {
			return fileError.ErrBlobValidation.WithMessagef(
				"config.bucket must match bucketOrContainer",
			)
		}
	case entity.AdapterTypeAzure:
		got, _ := m["container"].(string)
		if strings.TrimSpace(got) != want {
			return fileError.ErrBlobValidation.WithMessagef(
				"config.container must match bucketOrContainer",
			)
		}
	default:
		return fileError.ErrBlobValidation.WithMessagef(
			"unsupported adapter type %q",
			adapterType,
		)
	}
	return nil
}

func (s *configService) resolveRole(role string) (bool, error) {
	isSeller := role == constants.SELLER_ROLE_NAME
	isAdmin := role == constants.ADMIN_ROLE_NAME
	if !isSeller && !isAdmin {
		return false, fileError.ErrInvalidRole
	}
	return isSeller, nil
}

// validateAndEncryptConfig parses the raw config map for adapterType,
// validates all required fields, then returns a field-level encrypted JSON blob.
func (s *configService) validateAndEncryptConfig(
	adapterType entity.AdapterType,
	raw map[string]any,
) (map[string]any, error) {
	parser, err := blobAdapter.GetBlobConfigParser(adapterType)
	if err != nil {
		return nil, err
	}
	config, err := parser.ParseAndValidateConfig(raw)
	if err != nil {
		return nil, err
	}
	config.Encrypt()

	return config.ToMap(), nil
}

func (s *configService) ensureProviderIsActive(
	ctx context.Context,
	providerID uint,
) (*entity.StorageProvider, error) {
	provider, err := s.configRepo.GetActiveProviderByID(ctx, providerID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fileError.ErrProviderNotFound
		}
		return nil, fileError.ErrPersistenceFailed.WithMessagef(
			constant.FILE_PROVIDER_LOOKUP_FAILED_FMT,
			err,
		)
	}
	return provider, nil
}

func (s *configService) applyCoreConfigFields(
	cfg *entity.StorageConfig,
	providerID uint,
	displayName string,
	bucketOrContainer string,
) {
	cfg.ProviderID = providerID
	cfg.DisplayName = displayName
	cfg.BucketOrContainer = bucketOrContainer
}

func (s *configService) applyOwnership(
	cfg *entity.StorageConfig,
	userID uint,
	isSeller bool,
	isDefault bool,
) {
	cfg.OwnerID = &userID
	cfg.IsDefault = isDefault
	if isSeller {
		cfg.OwnerType = entity.OwnerTypeSeller
		return
	}

	cfg.OwnerType = entity.OwnerTypePlatform
}

func (s *configService) applyTimestamps(cfg *entity.StorageConfig) {
	now := time.Now()
	if cfg.ID == 0 {
		cfg.CreatedAt = now
	}
	cfg.UpdatedAt = now
}

func (s *configService) saveConfig(
	ctx context.Context,
	cfg *entity.StorageConfig,
) error {
	if err := s.configRepo.SaveConfig(ctx, cfg); err != nil {
		return fileError.ErrPersistenceFailed.WithMessagef(
			constant.FILE_SAVE_CONFIG_FAILED_FMT,
			err,
		)
	}
	return nil
}

func (s *configService) resolveTargetConfig(
	ctx context.Context,
	id *uint,
	userID uint,
	isSeller bool,
) (*entity.StorageConfig, error) {
	if id == nil {
		return &entity.StorageConfig{}, nil
	}

	if isSeller {
		return s.resolveSellerScopedConfig(ctx, *id, userID)
	}

	return s.resolvePlatformScopedConfig(ctx, *id)
}

func (s *configService) resolveSellerScopedConfig(
	ctx context.Context,
	configID uint,
	sellerID uint,
) (*entity.StorageConfig, error) {
	cfg, err := s.configRepo.GetSellerOwnedConfigByID(ctx, configID, sellerID)
	if err == nil {
		return cfg, nil
	}
	return nil, s.mapScopedLookupError(ctx, configID, err)
}

func (s *configService) resolvePlatformScopedConfig(
	ctx context.Context,
	configID uint,
) (*entity.StorageConfig, error) {
	cfg, err := s.configRepo.GetPlatformConfigByID(ctx, configID)
	if err == nil {
		return cfg, nil
	}
	return nil, s.mapScopedLookupError(ctx, configID, err)
}

func (s *configService) mapScopedLookupError(
	ctx context.Context,
	configID uint,
	err error,
) error {
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return fileError.ErrPersistenceFailed.WithMessagef(
			constant.FILE_CONFIG_LOOKUP_FAILED_FMT,
			err,
		)
	}

	_, lookupErr := s.configRepo.GetConfigByID(ctx, configID)
	if lookupErr == nil {
		return fileError.ErrUnauthorized
	}
	if errors.Is(lookupErr, gorm.ErrRecordNotFound) {
		return fileError.ErrConfigNotFound
	}
	return fileError.ErrPersistenceFailed.WithMessagef(
		constant.FILE_CONFIG_LOOKUP_FAILED_FMT,
		lookupErr,
	)
}

func (s *configService) ListConfigs(
	ctx context.Context,
	filter model.ListStorageConfigFilter,
) (*model.ListStorageConfigsResponse, error) {
	configs, total, err := s.configRepo.ListConfigs(ctx, filter)
	if err != nil {
		return nil, fileError.ErrListFailed.WithMessagef(
			constant.FILE_LIST_CONFIG_FAILED_FMT,
			err,
		)
	}

	items := make([]model.StorageConfigListItem, len(configs))
	for i, c := range configs {
		parser, err := blobAdapter.GetBlobConfigParser(c.Provider.AdapterType)
		if err != nil {
			return nil, fileError.ErrPersistenceFailed.WithMessagef(
				constant.FILE_LIST_CONFIG_FAILED_FMT,
				err,
			)
		}
		cfg, err := parser.ParseAndValidateConfig(c.ConfigData)
		if err != nil {
			return nil, fileError.ErrPersistenceFailed.WithMessagef(
				constant.FILE_LIST_CONFIG_FAILED_FMT,
				err,
			)
		}
		if err := cfg.Decrypt(); err != nil {
			return nil, err
		}
		items[i] = model.MapConfigToListItem(c, cfg.ToMap())
	}

	return &model.ListStorageConfigsResponse{
		Configs:    items,
		Pagination: common.NewPaginationResponse(filter.Page, filter.PageSize, total),
	}, nil
}
