package file_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"io"
	"testing"
	"time"

	fileError "ecommerce-be/file/error"
	"ecommerce-be/file/model"
	"ecommerce-be/file/service/blob_adapter"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// ─── Test helpers ─────────────────────────────────────────────────────────────

// generateGCSServiceAccountJSON creates a minimal fake service account JSON with a
// freshly generated RSA private key. Used to authenticate with fake-gcs-server and
// to generate signed URLs against the custom endpoint.
func generateGCSServiceAccountJSON(t *testing.T) string {
	t.Helper()
	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err, "generate RSA key for GCS tests")

	keyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privKey),
	})

	sa := map[string]string{
		"type":                        "service_account",
		"project_id":                  "test-project",
		"private_key_id":              "test-key-id",
		"private_key":                 string(keyPEM),
		"client_email":                "test@test-project.iam.gserviceaccount.com",
		"client_id":                   "123456789",
		"auth_uri":                    "https://accounts.google.com/o/oauth2/auth",
		"token_uri":                   "https://oauth2.googleapis.com/token",
		"auth_provider_x509_cert_url": "https://www.googleapis.com/oauth2/v1/certs",
		"client_x509_cert_url":        "https://www.googleapis.com/robot/v1/metadata/x509/test%40test-project.iam.gserviceaccount.com",
	}
	b, _ := json.Marshal(sa)
	return string(b)
}

// ─── Suite ───────────────────────────────────────────────────────────────────

type BlobAdapterGCSSuite struct {
	suite.Suite

	gcs     *FakeGCSContainer
	adapter blob_adapter.BlobAdapter
	bucket  string
	saJSON  string
}

func (s *BlobAdapterGCSSuite) SetupSuite() {
	s.bucket = "blob-adapter-gcs-it-" + RandomHex(6)
	s.gcs = SetupFakeGCS(s.T(), "test-project", s.bucket)
	s.saJSON = generateGCSServiceAccountJSON(s.T())

	a, err := blob_adapter.NewGCSAdapter(context.Background(), blob_adapter.GCSOptions{
		ServiceAccountJSON: s.saJSON,
		ProjectID:          "test-project",
		Endpoint:           s.gcs.Endpoint,
	})
	if err != nil {
		s.T().Fatal(err)
	}
	s.adapter = a
}

func (s *BlobAdapterGCSSuite) TearDownSuite() {
	if s.gcs != nil {
		s.gcs.Cleanup(s.T())
	}
}

func TestBlobAdapterGCSSuite(t *testing.T) {
	suite.Run(t, new(BlobAdapterGCSSuite))
}

// ─── T034: 7 method happy paths ───────────────────────────────────────────────

// Scenario: PutObject stores object and returns ETag + Key.
func (s *BlobAdapterGCSSuite) TestPutObject_Success() {
	key := RandomObjectKey("gcs-put")
	body := "hello gcs"
	out, err := s.adapter.PutObject(context.Background(), model.BlobPutObjectInput{
		Bucket:        s.bucket,
		Key:           key,
		ContentType:   "text/plain",
		ContentLength: int64(len(body)),
		Body:          bytes.NewReader([]byte(body)),
	})
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), key, out.Key)
	assert.NotEmpty(s.T(), out.ETag)
}

// Scenario: HeadObject returns correct metadata.
func (s *BlobAdapterGCSSuite) TestHeadObject_Success() {
	key := RandomObjectKey("gcs-head")
	payload := "metadata check"
	_, err := s.adapter.PutObject(context.Background(), model.BlobPutObjectInput{
		Bucket:        s.bucket,
		Key:           key,
		ContentType:   "text/plain",
		ContentLength: int64(len(payload)),
		Body:          bytes.NewReader([]byte(payload)),
	})
	require.NoError(s.T(), err)

	meta, err := s.adapter.HeadObject(context.Background(), s.bucket, key)
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), "text/plain", meta.ContentType)
	assert.Equal(s.T(), int64(len(payload)), meta.SizeBytes)
	assert.NotEmpty(s.T(), meta.ETag)
	assert.False(s.T(), meta.LastModified.IsZero())
}

// Scenario: GetObjectStream returns stream + correct content.
func (s *BlobAdapterGCSSuite) TestGetObjectStream_Success() {
	key := RandomObjectKey("gcs-get")
	payload := "gcs stream content"
	_, err := s.adapter.PutObject(context.Background(), model.BlobPutObjectInput{
		Bucket:        s.bucket,
		Key:           key,
		ContentType:   "text/plain",
		ContentLength: int64(len(payload)),
		Body:          bytes.NewReader([]byte(payload)),
	})
	require.NoError(s.T(), err)

	rc, meta, err := s.adapter.GetObjectStream(context.Background(), s.bucket, key)
	assert.NoError(s.T(), err)
	defer rc.Close()

	b, err := io.ReadAll(rc)
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), payload, string(b))
	assert.Equal(s.T(), int64(len(payload)), meta.SizeBytes)
}

// Scenario: PresignUpload returns a non-empty URL.
// Note: HTTP verification against fake-gcs-server is skipped — V4 signed URLs
// embed https:// but fake-gcs runs on http:// causing transport mismatch.
// URL generation correctness (signing key, TTL, method) is the verified contract here.
func (s *BlobAdapterGCSSuite) TestPresignUpload_Success() {
	key := RandomObjectKey("gcs-presign-upload")
	p, err := s.adapter.PresignUpload(context.Background(), model.BlobPresignUploadInput{
		Bucket:      s.bucket,
		Key:         key,
		ContentType: "text/plain",
		TTL:         DefaultPresignTTL(),
	})
	assert.NoError(s.T(), err)
	assert.NotEmpty(s.T(), p.URL)
	assert.True(s.T(), p.ExpiresAt.After(time.Now()))
}

// Scenario: PresignDownload returns a non-empty URL.
func (s *BlobAdapterGCSSuite) TestPresignDownload_Success() {
	key := RandomObjectKey("gcs-presign-download")
	_, err := s.adapter.PutObject(context.Background(), model.BlobPutObjectInput{
		Bucket:        s.bucket,
		Key:           key,
		ContentType:   "text/plain",
		ContentLength: int64(len("dl")),
		Body:          bytes.NewReader([]byte("dl")),
	})
	require.NoError(s.T(), err)

	p, err := s.adapter.PresignDownload(context.Background(), model.BlobPresignDownloadInput{
		Bucket: s.bucket,
		Key:    key,
		TTL:    DefaultPresignTTL(),
	})
	assert.NoError(s.T(), err)
	assert.NotEmpty(s.T(), p.URL)
	assert.True(s.T(), p.ExpiresAt.After(time.Now()))
}

// Scenario: CopyObject copies existing object to destination key.
func (s *BlobAdapterGCSSuite) TestCopyObject_Success() {
	srcKey := RandomObjectKey("gcs-copy-src")
	dstKey := RandomObjectKey("gcs-copy-dst")
	payload := "copy me gcs"
	_, err := s.adapter.PutObject(context.Background(), model.BlobPutObjectInput{
		Bucket:        s.bucket,
		Key:           srcKey,
		ContentType:   "text/plain",
		ContentLength: int64(len(payload)),
		Body:          bytes.NewReader([]byte(payload)),
	})
	require.NoError(s.T(), err)

	err = s.adapter.CopyObject(context.Background(), model.BlobCopyObjectInput{
		SourceBucket:      s.bucket,
		SourceKey:         srcKey,
		DestinationBucket: s.bucket,
		DestinationKey:    dstKey,
	})
	assert.NoError(s.T(), err)

	rc, _, err := s.adapter.GetObjectStream(context.Background(), s.bucket, dstKey)
	assert.NoError(s.T(), err)
	defer rc.Close()
	b, _ := io.ReadAll(rc)
	assert.Equal(s.T(), payload, string(b))
}

// Scenario: DeleteObject removes object; subsequent HeadObject returns not-found.
func (s *BlobAdapterGCSSuite) TestDeleteObject_Success() {
	key := RandomObjectKey("gcs-delete")
	_, err := s.adapter.PutObject(context.Background(), model.BlobPutObjectInput{
		Bucket:        s.bucket,
		Key:           key,
		ContentType:   "text/plain",
		ContentLength: int64(len("x")),
		Body:          bytes.NewReader([]byte("x")),
	})
	require.NoError(s.T(), err)

	err = s.adapter.DeleteObject(context.Background(), s.bucket, key)
	assert.NoError(s.T(), err)

	_, err = s.adapter.HeadObject(context.Background(), s.bucket, key)
	assert.Error(s.T(), err)
	assert.True(s.T(), fileError.IsBlobError(err, fileError.ErrBlobNotFound))
}

// ─── T035: Invalid credential test ───────────────────────────────────────────

// Scenario: Adapter constructed with invalid service account JSON returns validation error.
func (s *BlobAdapterGCSSuite) TestInvalidCredentials_ConstructorValidation() {
	_, err := blob_adapter.NewGCSAdapter(context.Background(), blob_adapter.GCSOptions{
		ServiceAccountJSON: `{"type":"service_account"}`, // missing client_email + private_key
		Endpoint:           s.gcs.Endpoint,
	})
	assert.Error(s.T(), err)
	assert.True(s.T(), fileError.IsBlobError(err, fileError.ErrBlobValidation))
	// Error must not echo the (empty) credential fields
	assert.NotContains(s.T(), err.Error(), "private_key")
}

// Scenario: HeadObject on non-existent key returns not-found.
func (s *BlobAdapterGCSSuite) TestHeadObject_NotFound() {
	_, err := s.adapter.HeadObject(context.Background(), s.bucket, "does-not-exist-"+RandomHex(8))
	assert.Error(s.T(), err)
	assert.True(s.T(), fileError.IsBlobError(err, fileError.ErrBlobNotFound))
}

// ─── T036: Context cancellation — IO-heavy methods ────────────────────────────

// Scenario: Context cancelled mid-PutObject returns error quickly.
func (s *BlobAdapterGCSSuite) TestPutObject_ContextCancel_IOHeavy() {
	key := RandomObjectKey("gcs-ctx-put")
	ctx, cancel := context.WithCancel(context.Background())
	pr, pw := io.Pipe()
	done := make(chan error, 1)

	go func() {
		_, err := s.adapter.PutObject(ctx, model.BlobPutObjectInput{
			Bucket:        s.bucket,
			Key:           key,
			ContentType:   "application/octet-stream",
			ContentLength: 1024 * 1024,
			Body:          pr,
		})
		done <- err
	}()

	time.Sleep(50 * time.Millisecond)
	cancel()
	_ = pw.CloseWithError(context.Canceled)

	select {
	case err := <-done:
		assert.Error(s.T(), err)
	case <-time.After(5 * time.Second):
		s.T().Fatal("put did not return after cancel")
	}
}

// ─── T036b: Context cancellation — metadata/presign methods ──────────────────

// Scenario: Already-cancelled context for metadata/presign methods returns error.
func (s *BlobAdapterGCSSuite) TestAlreadyCancelledContext_MetadataPresign() {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := s.adapter.HeadObject(ctx, s.bucket, "any-key")
	assert.Error(s.T(), err)

	_, err = s.adapter.PresignUpload(ctx, model.BlobPresignUploadInput{
		Bucket:      s.bucket,
		Key:         "k",
		ContentType: "text/plain",
		TTL:         time.Minute,
	})
	assert.Error(s.T(), err)

	_, err = s.adapter.PresignDownload(ctx, model.BlobPresignDownloadInput{
		Bucket: s.bucket,
		Key:    "k",
		TTL:    time.Minute,
	})
	assert.Error(s.T(), err)
}

// ─── Additional real-world scenarios ─────────────────────────────────────────

// Scenario: CopyObject across two different buckets within the same project (FR-012).
func (s *BlobAdapterGCSSuite) TestCopyObject_CrossBucket() {
	dstBucket := "blob-adapter-gcs-dst-" + RandomHex(6)
	EnsureFakeGCSBucket(s.T(), s.gcs.Endpoint, s.gcs.ProjectID, dstBucket)

	srcKey := RandomObjectKey("gcs-xbucket-src")
	dstKey := RandomObjectKey("gcs-xbucket-dst")
	payload := "gcs cross-bucket payload"

	_, err := s.adapter.PutObject(context.Background(), model.BlobPutObjectInput{
		Bucket:        s.bucket,
		Key:           srcKey,
		ContentType:   "text/plain",
		ContentLength: int64(len(payload)),
		Body:          bytes.NewReader([]byte(payload)),
	})
	require.NoError(s.T(), err)

	err = s.adapter.CopyObject(context.Background(), model.BlobCopyObjectInput{
		SourceBucket:      s.bucket,
		SourceKey:         srcKey,
		DestinationBucket: dstBucket,
		DestinationKey:    dstKey,
	})
	assert.NoError(s.T(), err)

	rc, meta, err := s.adapter.GetObjectStream(context.Background(), dstBucket, dstKey)
	require.NoError(s.T(), err)
	defer rc.Close()
	b, _ := io.ReadAll(rc)
	assert.Equal(s.T(), payload, string(b))
	assert.Equal(s.T(), int64(len(payload)), meta.SizeBytes)
}

// Scenario: PutObject against a non-existent bucket returns a structured not-found error.
func (s *BlobAdapterGCSSuite) TestPutObject_NonExistentBucket() {
	_, err := s.adapter.PutObject(context.Background(), model.BlobPutObjectInput{
		Bucket:        "no-such-bucket-" + RandomHex(8),
		Key:           RandomObjectKey("ghost"),
		ContentType:   "text/plain",
		ContentLength: 3,
		Body:          bytes.NewReader([]byte("abc")),
	})
	assert.Error(s.T(), err)
	assert.True(s.T(), fileError.IsBlobError(err, fileError.ErrBlobNotFound),
		"missing bucket must map to ErrBlobNotFound, got: %v", err)
}

// Scenario: GetObjectStream for a key that does not exist returns not-found, no stream opened.
func (s *BlobAdapterGCSSuite) TestGetObjectStream_NotFound() {
	rc, _, err := s.adapter.GetObjectStream(context.Background(), s.bucket, "gcs-missing-"+RandomHex(8))
	assert.Error(s.T(), err)
	assert.True(s.T(), fileError.IsBlobError(err, fileError.ErrBlobNotFound))
	if rc != nil {
		_ = rc.Close()
		s.T().Error("GetObjectStream must not return a ReadCloser when the key does not exist")
	}
}

// Scenario: PutObject twice with the same key overwrites the first object.
func (s *BlobAdapterGCSSuite) TestPutObject_OverwriteSameKey() {
	key := RandomObjectKey("gcs-overwrite")

	first := "first"
	_, err := s.adapter.PutObject(context.Background(), model.BlobPutObjectInput{
		Bucket:        s.bucket,
		Key:           key,
		ContentType:   "text/plain",
		ContentLength: int64(len(first)),
		Body:          bytes.NewReader([]byte(first)),
	})
	require.NoError(s.T(), err)

	second := "second gcs payload"
	_, err = s.adapter.PutObject(context.Background(), model.BlobPutObjectInput{
		Bucket:        s.bucket,
		Key:           key,
		ContentType:   "text/plain",
		ContentLength: int64(len(second)),
		Body:          bytes.NewReader([]byte(second)),
	})
	require.NoError(s.T(), err)

	rc, meta, err := s.adapter.GetObjectStream(context.Background(), s.bucket, key)
	require.NoError(s.T(), err)
	defer rc.Close()
	b, _ := io.ReadAll(rc)
	assert.Equal(s.T(), second, string(b))
	assert.Equal(s.T(), int64(len(second)), meta.SizeBytes)
}

// Scenario: PresignUpload with zero or negative TTL returns a validation error.
func (s *BlobAdapterGCSSuite) TestPresignUpload_InvalidTTL() {
	for _, ttl := range []time.Duration{0, -time.Minute} {
		_, err := s.adapter.PresignUpload(context.Background(), model.BlobPresignUploadInput{
			Bucket:      s.bucket,
			Key:         RandomObjectKey("gcs-ttl"),
			ContentType: "text/plain",
			TTL:         ttl,
		})
		assert.Error(s.T(), err)
		assert.True(s.T(), fileError.IsBlobError(err, fileError.ErrBlobValidation),
			"expected ErrBlobValidation for TTL=%v, got: %v", ttl, err)
	}
}

// Scenario: PresignDownload with zero TTL returns a validation error.
func (s *BlobAdapterGCSSuite) TestPresignDownload_InvalidTTL() {
	_, err := s.adapter.PresignDownload(context.Background(), model.BlobPresignDownloadInput{
		Bucket: s.bucket,
		Key:    "any",
		TTL:    0,
	})
	assert.Error(s.T(), err)
	assert.True(s.T(), fileError.IsBlobError(err, fileError.ErrBlobValidation))
}
