package variant_media_test

import (
	"testing"

	"ecommerce-be/test/integration/helpers"
)

// variantMediaStorageEnv extends the base env with MinIO and seller storage config
// so variant media file resolution can be exercised end-to-end.
type variantMediaStorageEnv struct {
	*variantMediaTestEnv
	storage *helpers.FileStorageEnv
}

func newVariantMediaStorageTestEnv(t *testing.T) *variantMediaStorageEnv {
	t.Helper()

	storage := helpers.SetupFileStorageEnv(t, helpers.FileStorageEnvConfig{
		Bucket:    "variant-media-test-bucket",
		SellerIDs: []uint64{2},
	})

	return &variantMediaStorageEnv{
		variantMediaTestEnv: &variantMediaTestEnv{
			client:     storage.Client,
			containers: storage.Containers,
		},
		storage: storage,
	}
}

// uploadFileAsSeller uploads a PRODUCT_IMAGE via the file module and returns the fileId.
func uploadFileAsSeller(t *testing.T, env *variantMediaStorageEnv, token string) string {
	t.Helper()
	return helpers.UploadProductImage(t, env.storage.Server, token)
}
