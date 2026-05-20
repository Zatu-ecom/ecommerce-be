package helpers

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"

	"ecommerce-be/common/constants"
	"ecommerce-be/common/messaging"
	"ecommerce-be/file/entity"
	"ecommerce-be/test/integration/setup"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type UploadJourney interface {
	Init(t *testing.T, req map[string]any) map[string]any
	PutBytes(t *testing.T, initData map[string]any, body []byte)
	Complete(t *testing.T, fileID string, clientEtag *string) map[string]any
	RunHappyPath(
		t *testing.T,
		req map[string]any,
		body []byte,
	) (map[string]any, map[string]any)
	AssertFileObject(
		t *testing.T,
		tc *setup.TestContainer,
		fileID string,
		expectedStatus entity.FileStatus,
	)
	NextVariantMessage(
		t *testing.T,
		messages <-chan any,
		timeout time.Duration,
	) map[string]any
	AssertNoVariantMessage(t *testing.T, messages <-chan any, within time.Duration)
}

// UploadHelper is a small, reusable helper for integration upload journeys.
type UploadHelper struct {
	Server http.Handler
	Token  string
}

func (h *UploadHelper) Init(t *testing.T, req map[string]any) map[string]any {
	t.Helper()
	client := NewAPIClient(h.Server)
	client.SetToken(h.Token)
	client.SetHeader(constants.CORRELATION_ID_HEADER, uuid.NewString())

	w := client.Post(t, "/api/files/init-upload", req)
	resp := AssertSuccessResponse(t, w, http.StatusCreated)
	data, ok := resp["data"].(map[string]any)
	require.True(t, ok, "response should contain object data")
	return data
}

func (h *UploadHelper) PutBytes(t *testing.T, initData map[string]any, body []byte) {
	t.Helper()
	uploadURL, _ := initData["uploadUrl"].(string)
	require.NotEmpty(t, uploadURL)

	req, err := http.NewRequest(http.MethodPut, uploadURL, bytes.NewReader(body))
	require.NoError(t, err)

	if headers, ok := initData["uploadHeaders"].(map[string]any); ok {
		for k, v := range headers {
			if str, ok := v.(string); ok {
				req.Header.Set(k, str)
			}
		}
	}
	if req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "image/jpeg")
	}

	res, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer res.Body.Close()
	require.GreaterOrEqual(t, res.StatusCode, http.StatusOK)
	require.Less(t, res.StatusCode, http.StatusMultipleChoices)
}

func (h *UploadHelper) Complete(
	t *testing.T,
	fileID string,
	clientEtag *string,
) map[string]any {
	t.Helper()
	client := NewAPIClient(h.Server)
	client.SetToken(h.Token)
	client.SetHeader(constants.CORRELATION_ID_HEADER, uuid.NewString())

	body := map[string]any{
		"fileId": fileID,
	}
	if clientEtag != nil {
		body["clientEtag"] = *clientEtag
	}

	w := client.Post(t, "/api/files/complete-upload", body)
	resp := AssertSuccessResponse(t, w, http.StatusOK)
	data, ok := resp["data"].(map[string]any)
	require.True(t, ok, "response should contain object data")
	return data
}

func (h *UploadHelper) RunHappyPath(
	t *testing.T,
	req map[string]any,
	body []byte,
) (map[string]any, map[string]any) {
	t.Helper()
	initData := h.Init(t, req)
	h.PutBytes(t, initData, body)
	fileID, _ := initData["fileId"].(string)
	completeData := h.Complete(t, fileID, nil)
	return initData, completeData
}

func (h *UploadHelper) AssertFileObject(
	t *testing.T,
	tc *setup.TestContainer,
	fileID string,
	expectedStatus entity.FileStatus,
) {
	t.Helper()
	type row struct {
		Status string
	}
	var r row
	err := tc.DB.Raw(
		"SELECT status FROM file_object WHERE file_id = ?",
		fileID,
	).Scan(&r).Error
	require.NoError(t, err)
	require.Equal(t, string(expectedStatus), r.Status)
}

func ParseEnvelopePayload(t *testing.T, env messaging.Envelope) map[string]any {
	t.Helper()
	out := map[string]any{}
	err := json.Unmarshal(env.Payload, &out)
	require.NoError(t, err)
	return out
}

func ReadBodyString(t *testing.T, body io.ReadCloser) string {
	t.Helper()
	defer body.Close()
	raw, err := io.ReadAll(body)
	require.NoError(t, err)
	return string(raw)
}
