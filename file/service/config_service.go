package service

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"ecommerce-be/common/config"
	"ecommerce-be/common/helper"
	"ecommerce-be/file/entity"
	"ecommerce-be/file/model"
	"ecommerce-be/file/repository"
)

var (
	ErrProviderNotFound    = errors.New("storage provider not found or inactive")
	ErrUnauthorized        = errors.New("unauthorized to manage this storage config")
	ErrConfigNotFound      = errors.New("storage config not found")
	ErrSerializationFailed = errors.New("failed to process configuration data")
)

type ConfigService interface {
	GetProviders(ctx context.Context) ([]model.ProviderResponse, error)
	SaveConfig(ctx context.Context, userID uint, role string, req model.SaveConfigRequest) (*model.ConfigResponse, error)
}

type configService struct {
	configRepo repository.ConfigRepository
}

func NewConfigService(configRepo repository.ConfigRepository) ConfigService {
	return &configService{
		configRepo: configRepo,
	}
}

func (s *configService) GetProviders(ctx context.Context) ([]model.ProviderResponse, error) {
	providers, err := s.configRepo.GetProviders(ctx)
	if err != nil {
		return nil, err
	}

	res := make([]model.ProviderResponse, len(providers))
	for i, p := range providers {
		res[i] = model.MapProviderToResponse(p)
	}

	return res, nil
}

func (s *configService) SaveConfig(ctx context.Context, userID uint, role string, req model.SaveConfigRequest) (*model.ConfigResponse, error) {
	isSeller := role == "SELLER"

	var cfg *entity.StorageConfig
	var err error

	// Find existing or initialize new
	if req.ID != nil {
		cfg, err = s.configRepo.GetConfigByID(ctx, *req.ID)
		if err != nil {
			return nil, ErrConfigNotFound
		}

		// Check authorization for updates
		if isSeller && (cfg.OwnerType != entity.OwnerTypeSeller || cfg.OwnerID == nil || *cfg.OwnerID != userID) {
			return nil, ErrUnauthorized
		}
		if !isSeller && cfg.OwnerType != entity.OwnerTypePlatform {
			return nil, ErrUnauthorized
		}
	} else {
		cfg = &entity.StorageConfig{}
	}

	// Update basic fields
	cfg.ProviderID = req.ProviderID
	cfg.DisplayName = req.DisplayName
	cfg.BucketOrContainer = req.BucketOrContainer
	cfg.Region = req.Region
	cfg.Endpoint = req.Endpoint
	cfg.BasePath = req.BasePath
	cfg.ForcePathStyle = req.ForcePathStyle

	// Restrict defaults and owner scope based on role
	if isSeller {
		cfg.OwnerType = entity.OwnerTypeSeller
		var uid = userID
		cfg.OwnerID = &uid
		cfg.IsDefault = false
	} else {
		cfg.OwnerType = entity.OwnerTypePlatform
		cfg.OwnerID = nil
		cfg.IsDefault = req.IsDefault
	}

	// Serialize configs
	if req.ConfigJSON != nil {
		cfg.ConfigJSON = req.ConfigJSON
	}

	// Encrypt credentials
	credBytes, err := json.Marshal(req.Credentials)
	if err != nil {
		return nil, ErrSerializationFailed
	}

	appConfig := config.Get().App
	encrypted, err := helper.Encrypt(string(credBytes), appConfig.EncryptionKey)
	if err != nil {
		return nil, errors.New("failed to encrypt credentials")
	}

	cfg.CredentialsEncrypted = []byte(encrypted)

	// Admin clear defaults logically
	if !isSeller && cfg.IsDefault {
		_ = s.configRepo.ClearDefaultConfigs(ctx)
	}

	now := time.Now()
	// Create or Update
	if req.ID != nil {
		cfg.UpdatedAt = now
		err = s.configRepo.UpdateConfig(ctx, cfg)
	} else {
		cfg.IsActive = true
		cfg.ValidationStatus = "PENDING"
		cfg.CreatedAt = now
		cfg.UpdatedAt = now
		err = s.configRepo.CreateConfig(ctx, cfg)
	}

	if err != nil {
		return nil, err
	}

	res := model.MapConfigToResponse(*cfg)
	return &res, nil
}
