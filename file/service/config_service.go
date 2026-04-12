package service

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"ecommerce-be/common"
	"ecommerce-be/common/config"
	"ecommerce-be/common/constants"
	"ecommerce-be/common/helper"
	"ecommerce-be/file/entity"
	fileError "ecommerce-be/file/error"
	"ecommerce-be/file/model"
	"ecommerce-be/file/repository"
	"ecommerce-be/file/utils/constant"

	"gorm.io/gorm"
)

type ConfigService interface {
	GetProviders(ctx context.Context) ([]model.ProviderResponse, error)
	SaveConfig(
		ctx context.Context,
		userID uint,
		role string,
		req model.SaveConfigRequest,
	) (*model.ConfigResponse, error)
	ListConfigs(
		ctx context.Context,
		filter model.ListStorageConfigFilter,
	) (*model.ListStorageConfigsResponse, error)
	ActivateConfig(
		ctx context.Context,
		userID uint,
		role string,
		configID uint,
	) (*model.ActivateStorageConfigResponse, error)
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

	if err := s.ensureProviderIsActive(ctx, req.ProviderID); err != nil {
		return nil, err
	}

	cfg, err := s.resolveTargetConfig(ctx, req.ID, userID, isSeller)
	if err != nil {
		return nil, err
	}

	s.applyRequestFields(cfg, req)
	s.applyOwnership(cfg, userID, isSeller, req.IsDefault)

	encryptedCredentials, err := s.encryptCredentials(req.Credentials)
	if err != nil {
		return nil, err
	}
	cfg.CredentialsEncrypted = encryptedCredentials

	s.applyTimestamps(cfg)

	clearPlatformDefaults := !isSeller && cfg.IsDefault
	if err := s.saveConfig(ctx, cfg, clearPlatformDefaults); err != nil {
		return nil, err
	}

	res := model.MapConfigToResponse(*cfg)
	return &res, nil
}

func (s *configService) resolveRole(role string) (bool, error) {
	isSeller := role == constants.SELLER_ROLE_NAME
	isAdmin := role == constants.ADMIN_ROLE_NAME
	if !isSeller && !isAdmin {
		return false, fileError.ErrInvalidRole
	}
	return isSeller, nil
}

func (s *configService) ensureProviderIsActive(
	ctx context.Context,
	providerID uint,
) error {
	if _, err := s.configRepo.GetActiveProviderByID(ctx, providerID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fileError.ErrProviderNotFound
		}
		return fileError.ErrPersistenceFailed.WithMessagef(
			constant.FILE_PROVIDER_LOOKUP_FAILED_FMT,
			err,
		)
	}
	return nil
}

func (s *configService) applyRequestFields(
	cfg *entity.StorageConfig,
	req model.SaveConfigRequest,
) {
	cfg.ProviderID = req.ProviderID
	cfg.DisplayName = req.DisplayName
	cfg.BucketOrContainer = req.BucketOrContainer
	cfg.Region = req.Region
	cfg.Endpoint = req.Endpoint
	cfg.BasePath = req.BasePath
	cfg.ForcePathStyle = req.ForcePathStyle
	if req.ConfigJSON != nil {
		cfg.ConfigJSON = req.ConfigJSON
	}
}

func (s *configService) applyOwnership(
	cfg *entity.StorageConfig,
	userID uint,
	isSeller bool,
	isDefault bool,
) {
	if isSeller {
		cfg.OwnerType = entity.OwnerTypeSeller
		cfg.OwnerID = &userID
		cfg.IsDefault = false
		return
	}

	cfg.OwnerType = entity.OwnerTypePlatform
	cfg.OwnerID = nil
	cfg.IsDefault = isDefault
}

func (s *configService) encryptCredentials(
	credentials map[string]any,
) ([]byte, error) {
	credBytes, err := json.Marshal(credentials)
	if err != nil {
		return nil, fileError.ErrSerializationFailed.WithMessagef(
			constant.FILE_INVALID_CREDENTIALS_PAYLOAD_FMT,
			err,
		)
	}

	cfgSingleton := config.Get()
	if cfgSingleton == nil {
		return nil, fileError.ErrEncryptionFailed.WithMessage(constant.FILE_CONFIG_NOT_LOADED_MSG)
	}

	encrypted, err := helper.Encrypt(string(credBytes), cfgSingleton.App.EncryptionKey)
	if err != nil {
		return nil, fileError.ErrEncryptionFailed.WithMessage(err.Error())
	}

	return []byte(encrypted), nil
}

func (s *configService) applyTimestamps(cfg *entity.StorageConfig) {
	now := time.Now()
	if cfg.ID == 0 {
		cfg.IsActive = true
		cfg.ValidationStatus = constant.FILE_CONFIG_PENDING_STATUS
		cfg.CreatedAt = now
	}
	cfg.UpdatedAt = now
}

func (s *configService) saveConfig(
	ctx context.Context,
	cfg *entity.StorageConfig,
	clearPlatformDefaults bool,
) error {
	if err := s.configRepo.SaveConfig(ctx, cfg, clearPlatformDefaults); err != nil {
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
		items[i] = model.MapConfigToListItem(c)
	}

	return &model.ListStorageConfigsResponse{
		Configs:    items,
		Pagination: common.NewPaginationResponse(filter.Page, filter.PageSize, total),
	}, nil
}

func (s *configService) ActivateConfig(
	ctx context.Context,
	userID uint,
	role string,
	configID uint,
) (*model.ActivateStorageConfigResponse, error) {
	isSeller, err := s.resolveRole(role)
	if err != nil {
		return nil, err
	}

	// Verify config exists and caller is in-scope
	var cfg *entity.StorageConfig
	if isSeller {
		cfg, err = s.configRepo.GetSellerOwnedConfigByID(ctx, configID, userID)
	} else {
		cfg, err = s.configRepo.GetPlatformConfigByID(ctx, configID)
	}
	if err != nil {
		return nil, s.mapScopedLookupError(ctx, configID, err)
	}

	// Determine scope params for activation transaction
	var ownerID *uint
	if isSeller {
		ownerID = &userID
	}

	if err := s.configRepo.ActivateConfig(ctx, configID, cfg.OwnerType, ownerID); err != nil {
		return nil, fileError.ErrActivationFailed.WithMessagef(
			constant.FILE_ACTIVATE_CONFIG_FAILED_FMT,
			err,
		)
	}

	// Re-fetch to return current state
	updatedCfg, err := s.configRepo.GetConfigByID(ctx, configID)
	if err != nil {
		return nil, fileError.ErrPersistenceFailed.WithMessagef(
			constant.FILE_CONFIG_LOOKUP_FAILED_FMT,
			err,
		)
	}

	res := model.MapConfigToActivateResponse(*updatedCfg)
	return &res, nil
}
