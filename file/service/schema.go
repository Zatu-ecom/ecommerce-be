package service

import (
	"ecommerce-be/file/entity"
	fileError "ecommerce-be/file/error"
	"ecommerce-be/file/model"
	"ecommerce-be/file/service/blobAdapter"
)

// schemaRegistry maps adapter type to its static form schema (from GCSSchema, S3Schema, AzureSchema).
var schemaRegistry = map[entity.AdapterType]model.AdapterConfigSchema{
	entity.AdapterTypeGCS:          blobAdapter.GCSSchema(),
	entity.AdapterTypeS3Compatible: blobAdapter.S3Schema(),
	entity.AdapterTypeAzure:        blobAdapter.AzureSchema(),
}

// adapterSchemaAllOrder is the stable order when returning every adapter schema (no query filter).
var adapterSchemaAllOrder = []entity.AdapterType{
	entity.AdapterTypeGCS,
	entity.AdapterTypeS3Compatible,
	entity.AdapterTypeAzure,
}

// GetAdapterSchemas returns one or more adapter config schemas.
// If adapterType is empty, all registered schemas are returned in adapterSchemaAllOrder.
// If adapterType is set, only that adapter's schema is returned (single-element slice).
// Returns ErrBlobValidation when adapterType is non-empty but unknown.
func GetAdapterSchemas(adapterType entity.AdapterType) ([]model.AdapterConfigSchema, error) {
	if adapterType == "" {
		out := make([]model.AdapterConfigSchema, 0, len(adapterSchemaAllOrder))
		for _, at := range adapterSchemaAllOrder {
			out = append(out, schemaRegistry[at])
		}
		return out, nil
	}
	s, ok := schemaRegistry[adapterType]
	if !ok {
		return nil, fileError.ErrBlobValidation.WithMessagef(
			"unsupported adapter type %q; supported: gcs, s3_compatible, azure",
			adapterType,
		)
	}
	return []model.AdapterConfigSchema{s}, nil
}
