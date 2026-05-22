//go:build perf

package file_test

import (
	"fmt"
	"net/http"
	"sort"
	"time"

	"ecommerce-be/common/constants"
	"ecommerce-be/test/integration/helpers"

	"github.com/stretchr/testify/require"
)

func (s *UploadSuite) TestFileReadDelete_Performance() {
	s.measureBatchListP95()
	s.measureSingleGetP95()
	s.measureDownloadURLP95()
	s.measureDeleteP95()
}

func (s *UploadSuite) measureBatchListP95() {
	fileIDs := make([]string, 0, 20)
	for i := 0; i < 20; i++ {
		fileIDs = append(
			fileIDs,
			s.createUploadedFile(
				s.sellerToken,
				map[string]any{
					"purpose":    "DOCUMENT",
					"visibility": "PRIVATE",
					"filename":   fmt.Sprintf("perf-list-%d.pdf", i),
					"mimeType":   "application/pdf",
					"sizeBytes":  512,
				},
				make([]byte, 512),
			),
		)
	}

	client := helpers.NewAPIClient(s.server)
	client.SetToken(s.sellerToken)
	client.SetHeader(constants.CORRELATION_ID_HEADER, "perf-list")

	durations := make([]time.Duration, 0, 5)
	for i := 0; i < 5; i++ {
		start := time.Now()
		w := client.Get(s.T(), "/api/file?fileIds="+joinCSV(fileIDs))
		helpers.AssertSuccessResponse(s.T(), w, http.StatusOK)
		durations = append(durations, time.Since(start))
	}

	require.LessOrEqual(s.T(), p95(durations), 100*time.Millisecond, "SC-002 failed")
}

func (s *UploadSuite) measureSingleGetP95() {
	fileID := s.createUploadedFile(
		s.sellerToken,
		map[string]any{
			"purpose":    "DOCUMENT",
			"visibility": "PRIVATE",
			"filename":   "perf-get.pdf",
			"mimeType":   "application/pdf",
			"sizeBytes":  700,
		},
		make([]byte, 700),
	)

	client := helpers.NewAPIClient(s.server)
	client.SetToken(s.sellerToken)
	client.SetHeader(constants.CORRELATION_ID_HEADER, "perf-get")

	durations := make([]time.Duration, 0, 5)
	for i := 0; i < 5; i++ {
		start := time.Now()
		w := client.Get(s.T(), "/api/file/"+fileID)
		helpers.AssertSuccessResponse(s.T(), w, http.StatusOK)
		durations = append(durations, time.Since(start))
	}

	require.LessOrEqual(s.T(), p95(durations), 200*time.Millisecond, "SC-003 failed")
}

func (s *UploadSuite) measureDownloadURLP95() {
	fileID := s.createUploadedFile(
		s.sellerToken,
		map[string]any{
			"purpose":    "DOCUMENT",
			"visibility": "PRIVATE",
			"filename":   "perf-download.pdf",
			"mimeType":   "application/pdf",
			"sizeBytes":  700,
		},
		make([]byte, 700),
	)

	client := helpers.NewAPIClient(s.server)
	client.SetToken(s.sellerToken)
	client.SetHeader(constants.CORRELATION_ID_HEADER, "perf-download-url")

	durations := make([]time.Duration, 0, 5)
	for i := 0; i < 5; i++ {
		start := time.Now()
		w := client.Get(s.T(), "/api/file/"+fileID+"/download-url")
		helpers.AssertSuccessResponse(s.T(), w, http.StatusOK)
		durations = append(durations, time.Since(start))
	}

	require.LessOrEqual(s.T(), p95(durations), 500*time.Millisecond, "SC-004 failed")
}

func (s *UploadSuite) measureDeleteP95() {
	client := helpers.NewAPIClient(s.server)
	client.SetToken(s.sellerToken)
	client.SetHeader(constants.CORRELATION_ID_HEADER, "perf-delete")

	durations := make([]time.Duration, 0, 5)
	for i := 0; i < 5; i++ {
		fileID := s.createUploadedFile(
			s.sellerToken,
			map[string]any{
				"purpose":    "DOCUMENT",
				"visibility": "PRIVATE",
				"filename":   fmt.Sprintf("perf-delete-%d.pdf", i),
				"mimeType":   "application/pdf",
				"sizeBytes":  512,
			},
			make([]byte, 512),
		)

		start := time.Now()
		w := client.Delete(s.T(), "/api/file/"+fileID)
		helpers.AssertSuccessResponse(s.T(), w, http.StatusOK)
		durations = append(durations, time.Since(start))
	}

	require.LessOrEqual(s.T(), p95(durations), 300*time.Millisecond, "SC-005 failed")
}

func p95(durations []time.Duration) time.Duration {
	sorted := append([]time.Duration(nil), durations...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
	index := int(float64(len(sorted))*0.95 + 0.5)
	if index >= len(sorted) {
		index = len(sorted) - 1
	}
	return sorted[index]
}

func joinCSV(values []string) string {
	if len(values) == 0 {
		return ""
	}
	out := values[0]
	for i := 1; i < len(values); i++ {
		out += "," + values[i]
	}
	return out
}
