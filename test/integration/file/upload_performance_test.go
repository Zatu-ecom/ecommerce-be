//go:build perf

package file_test

import (
	"net/http"
	"sort"
	"time"

	"ecommerce-be/common/constants"
	"ecommerce-be/test/integration/helpers"

	"github.com/stretchr/testify/require"
)

func (s *UploadSuite) TestUploadRoundTrip_Performance_1MBJPEG() {
	const iterations = 5
	durations := make([]time.Duration, 0, iterations)

	for i := 0; i < iterations; i++ {
		client := helpers.NewAPIClient(s.server)
		client.SetToken(s.sellerToken)
		client.SetHeader(constants.CORRELATION_ID_HEADER, "perf-upload-round-trip")

		initReq := map[string]interface{}{
			"purpose":             "PRODUCT_IMAGE",
			"visibility":          "PRIVATE",
			"filename":            "perf-1mb.jpg",
			"mimeType":            "image/jpeg",
			"sizeBytes":           1024 * 1024,
			"uploadExpiryMinutes": 15,
		}

		startedAt := time.Now()
		initW := client.Post(s.T(), uploadInitEndpoint, initReq)
		initResp := helpers.AssertSuccessResponse(s.T(), initW, http.StatusCreated)
		initData := initResp["data"].(map[string]interface{})

		uploadHelper := helpers.UploadHelper{Server: s.server, Token: s.sellerToken}
		uploadHelper.PutBytes(s.T(), initData, make([]byte, 1024*1024))

		fileID := initData["fileId"].(string)
		completeW := client.Post(s.T(), uploadCompleteEndpoint, map[string]interface{}{
			"fileId": fileID,
		})
		helpers.AssertSuccessResponse(s.T(), completeW, http.StatusOK)
		durations = append(durations, time.Since(startedAt))

		_ = s.nextVariantMessage(3 * time.Second)
	}

	sort.Slice(durations, func(i, j int) bool {
		return durations[i] < durations[j]
	})
	p95Index := int(float64(len(durations))*0.95 + 0.5)
	if p95Index >= len(durations) {
		p95Index = len(durations) - 1
	}

	require.LessOrEqual(
		s.T(),
		durations[p95Index],
		3*time.Second,
		"1 MB JPEG upload round-trip p95 exceeded SC-001",
	)
}
