package blobAdapter

import (
	"context"

	"ecommerce-be/file/entity"
	fileError "ecommerce-be/file/error"
)

var blobConfigParserMap = map[entity.AdapterType]BlobConfigParser{
	entity.AdapterTypeS3Compatible: &s3CompatibleAdapter{},
	entity.AdapterTypeGCS:          &gcsAdapter{},
	entity.AdapterTypeAzure:        &azureBlobAdapter{},
}

func GetBlobConfigParser(adapterType entity.AdapterType) (BlobConfigParser, error) {
	parser, ok := blobConfigParserMap[adapterType]
	if !ok {
		return nil, fileError.ErrBlobValidation.WithMessagef(
			"factory: unsupported adapter type %q", adapterType,
		)
	}
	return parser, nil
}

// GetAdapter builds a provider adapter from a plaintext config map (decrypted fields).
// For rows loaded from the database, use GetAdapterFromStoredConfig instead.
func GetAdapter(
	ctx context.Context,
	adapterType entity.AdapterType,
	config map[string]any,
) (BlobAdapter, error) {
	switch adapterType {
	case entity.AdapterTypeS3Compatible:
		return NewS3CompatibleAdapterFromMap(ctx, config)
	case entity.AdapterTypeGCS:
		return NewGCSAdapterFromMap(ctx, config)
	case entity.AdapterTypeAzure:
		return NewAzureAdapterFromMap(ctx, config)
	case "":
		return nil, fileError.ErrBlobValidation.WithMessagef(
			"factory: adapter type is required",
		)
	default:
		return nil, fileError.ErrBlobValidation.WithMessagef(
			"factory: unsupported adapter type %q", adapterType,
		)
	}
}

// GetAdapterFromStoredConfig parses persisted config_data (field-level AES ciphertext from SaveConfig),
// runs BlobConfig.Decrypt(), then constructs the adapter with plaintext routing credentials.
func GetAdapterFromStoredConfig(
	ctx context.Context,
	adapterType entity.AdapterType,
	storedConfig map[string]any,
) (BlobAdapter, error) {
	if len(storedConfig) == 0 {
		return nil, fileError.ErrBlobValidation.WithMessagef(
			"factory: config_data is required",
		)
	}
	parser, err := GetBlobConfigParser(adapterType)
	if err != nil {
		return nil, err
	}
	cfg, err := parser.ParseAndValidateConfig(storedConfig)
	if err != nil {
		return nil, err
	}
	if err := cfg.Decrypt(); err != nil {
		return nil, err
	}
	return GetAdapter(ctx, adapterType, cfg.ToMap())
}
