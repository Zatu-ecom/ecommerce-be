package file_test

import (
	"strings"
	"testing"
	"time"

	"ecommerce-be/file/entity"
	"ecommerce-be/file/utils"

	"github.com/stretchr/testify/require"
)

func TestUploadSanitizeFilename(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"simple", "Hero Shot.JPG", "hero-shot.jpg"},
		{"slashes", "../../etc/passwd", "....etcpasswd"},
		{"nul and rtl", "x\u0000\u202Ejpg", "xjpg"},
		{"empty to default", "   ", "file"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := utils.SanitizeFilename(tc.in)
			require.Equal(t, tc.want, got)
		})
	}

	veryLong := strings.Repeat("a", 500)
	require.LessOrEqual(t, len(utils.SanitizeFilename(veryLong)), 120)
}

func TestUploadBuildObjectKey(t *testing.T) {
	now := time.Date(2026, 4, 18, 10, 0, 0, 0, time.UTC)
	sellerID := uint64(42)

	sellerKey := utils.BuildObjectKey(
		entity.OwnerTypeSeller,
		&sellerID,
		entity.FilePurposeProductImage,
		now,
		"file-123",
		"hero.jpg",
	)
	require.Equal(t, "seller/42/PRODUCT_IMAGE/2026/04/file-123-hero.jpg", sellerKey)

	platformKey := utils.BuildObjectKey(
		entity.OwnerTypePlatform,
		nil,
		entity.FilePurposeDocument,
		now,
		"file-456",
		"invoice.pdf",
	)
	require.Equal(t, "platform/DOCUMENT/2026/04/file-456-invoice.pdf", platformKey)
}
