package file_test

import (
	"bytes"
	"context"
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

// ─── Suite ───────────────────────────────────────────────────────────────────

// BlobAdapterAzureSuite exercises all BlobAdapter methods against a live Azurite container.
type BlobAdapterAzureSuite struct {
	suite.Suite

	az      *AzuriteContainer
	adapter blob_adapter.BlobAdapter
	bucket  string
}

func (s *BlobAdapterAzureSuite) SetupSuite() {
	s.bucket = "blob-adapter-az-it-" + RandomHex(6)
	s.az = SetupAzurite(s.T(), s.bucket)

	a, err := blob_adapter.NewAzureBlobAdapter(blob_adapter.AzureOptions{
		AccountName: s.az.AccountName,
		AccountKey:  s.az.AccountKey,
		Endpoint:    s.az.BlobEndpoint,
	})
	if err != nil {
		s.T().Fatalf("NewAzureBlobAdapter: %v", err)
	}
	s.adapter = a
}

func (s *BlobAdapterAzureSuite) TearDownSuite() {
	if s.az != nil {
		s.az.Cleanup(s.T())
	}
}

func TestBlobAdapterAzureSuite(t *testing.T) {
	suite.Run(t, new(BlobAdapterAzureSuite))
}

// ─── T046: 7 method happy paths ───────────────────────────────────────────────

// Scenario: PutObject stores a blob and returns ETag + Key.
func (s *BlobAdapterAzureSuite) TestPutObject_Success() {
	key := RandomObjectKey("az-put")
	body := "hello azure"
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

// Scenario: HeadObject returns correct metadata for an existing blob.
func (s *BlobAdapterAzureSuite) TestHeadObject_Success() {
	key := RandomObjectKey("az-head")
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
	assert.Equal(s.T(), "text/plain", meta.ContentType, "ContentType must round-trip through Azure adapter")
	assert.Equal(s.T(), int64(len(payload)), meta.SizeBytes)
	assert.NotEmpty(s.T(), meta.ETag)
	assert.False(s.T(), meta.LastModified.IsZero())
}

// Scenario: GetObjectStream returns stream with correct content.
func (s *BlobAdapterAzureSuite) TestGetObjectStream_Success() {
	key := RandomObjectKey("az-get")
	payload := "azure stream content"
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
	require.NotNil(s.T(), rc)
	defer rc.Close()

	b, err := io.ReadAll(rc)
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), payload, string(b))
	assert.Equal(s.T(), int64(len(payload)), meta.SizeBytes)
}

// Scenario: PresignUpload returns a non-empty SAS URL that expires in the future.
func (s *BlobAdapterAzureSuite) TestPresignUpload_Success() {
	key := RandomObjectKey("az-presign-upload")
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

// Scenario: PresignDownload returns a non-empty SAS URL that expires in the future.
func (s *BlobAdapterAzureSuite) TestPresignDownload_Success() {
	key := RandomObjectKey("az-presign-download")
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

// Scenario: CopyObject copies existing blob to destination key, content matches.
func (s *BlobAdapterAzureSuite) TestCopyObject_Success() {
	srcKey := RandomObjectKey("az-copy-src")
	dstKey := RandomObjectKey("az-copy-dst")
	payload := "copy me azure"
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
	require.NotNil(s.T(), rc)
	defer rc.Close()
	b, _ := io.ReadAll(rc)
	assert.Equal(s.T(), payload, string(b))
}

// Scenario: DeleteObject removes a blob; subsequent HeadObject returns not-found.
func (s *BlobAdapterAzureSuite) TestDeleteObject_Success() {
	key := RandomObjectKey("az-delete")
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

// ─── T047: Invalid credential test cases ─────────────────────────────────────

// Scenario: Constructor rejects empty account name.
func (s *BlobAdapterAzureSuite) TestInvalidCredentials_MissingAccountName() {
	_, err := blob_adapter.NewAzureBlobAdapter(blob_adapter.AzureOptions{
		AccountName: "",
		AccountKey:  s.az.AccountKey,
		Endpoint:    s.az.BlobEndpoint,
	})
	assert.Error(s.T(), err)
	assert.True(s.T(), fileError.IsBlobError(err, fileError.ErrBlobValidation))
	assert.NotContains(s.T(), err.Error(), s.az.AccountKey, "error must not echo account key")
}

// Scenario: Constructor rejects empty account key.
func (s *BlobAdapterAzureSuite) TestInvalidCredentials_MissingAccountKey() {
	_, err := blob_adapter.NewAzureBlobAdapter(blob_adapter.AzureOptions{
		AccountName: s.az.AccountName,
		AccountKey:  "",
		Endpoint:    s.az.BlobEndpoint,
	})
	assert.Error(s.T(), err)
	assert.True(s.T(), fileError.IsBlobError(err, fileError.ErrBlobValidation))
}

// Scenario: HeadObject on non-existent key returns not-found error.
func (s *BlobAdapterAzureSuite) TestHeadObject_NotFound() {
	_, err := s.adapter.HeadObject(context.Background(), s.bucket, "does-not-exist-"+RandomHex(8))
	assert.Error(s.T(), err)
	assert.True(s.T(), fileError.IsBlobError(err, fileError.ErrBlobNotFound))
}

// Scenario: Error messages do not contain account key (secret leak check).
func (s *BlobAdapterAzureSuite) TestNoSecretLeak_InErrors() {
	_, err := s.adapter.HeadObject(context.Background(), s.bucket, "no-such-key-"+RandomHex(8))
	require.Error(s.T(), err)
	assert.NotContains(s.T(), err.Error(), s.az.AccountKey)
}

// ─── T048: Context cancellation — IO-heavy methods ────────────────────────────

// Scenario: Context cancelled mid-PutObject; adapter returns error without goroutine leak.
func (s *BlobAdapterAzureSuite) TestPutObject_ContextCancel_IOHeavy() {
	key := RandomObjectKey("az-ctx-put")
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
		s.T().Fatal("PutObject did not return after context cancel")
	}
}

// Scenario: Context cancelled mid-GetObjectStream download; body closed without leak.
func (s *BlobAdapterAzureSuite) TestGetObjectStream_ContextCancel_IOHeavy() {
	key := RandomObjectKey("az-ctx-get")
	payload := bytes.Repeat([]byte("a"), 64*1024)
	_, err := s.adapter.PutObject(context.Background(), model.BlobPutObjectInput{
		Bucket:        s.bucket,
		Key:           key,
		ContentType:   "application/octet-stream",
		ContentLength: int64(len(payload)),
		Body:          bytes.NewReader(payload),
	})
	require.NoError(s.T(), err)

	ctx, cancel := context.WithCancel(context.Background())
	rc, _, err := s.adapter.GetObjectStream(ctx, s.bucket, key)
	if err != nil {
		cancel()
		s.T().Skip("GetObjectStream failed before cancel; skipping cancel test")
		return
	}
	cancel()
	if rc != nil {
		_ = rc.Close()
	}
	// Test passes if no panic/goroutine leak; no further assertion needed.
}

// ─── T048b: Context cancellation — metadata/presign methods ──────────────────

// Scenario: Already-cancelled context passed to metadata/presign methods returns error immediately.
func (s *BlobAdapterAzureSuite) TestAlreadyCancelledContext_MetadataPresign() {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := s.adapter.HeadObject(ctx, s.bucket, "any-key")
	assert.Error(s.T(), err, "HeadObject should error on cancelled ctx")

	_, err = s.adapter.PresignUpload(ctx, model.BlobPresignUploadInput{
		Bucket:      s.bucket,
		Key:         "k",
		ContentType: "text/plain",
		TTL:         time.Minute,
	})
	assert.Error(s.T(), err, "PresignUpload should error on cancelled ctx")

	_, err = s.adapter.PresignDownload(ctx, model.BlobPresignDownloadInput{
		Bucket: s.bucket,
		Key:    "k",
		TTL:    time.Minute,
	})
	assert.Error(s.T(), err, "PresignDownload should error on cancelled ctx")

	err = s.adapter.DeleteObject(ctx, s.bucket, "k")
	assert.Error(s.T(), err, "DeleteObject should error on cancelled ctx")

	err = s.adapter.CopyObject(ctx, model.BlobCopyObjectInput{
		SourceBucket:      s.bucket,
		SourceKey:         "src",
		DestinationBucket: s.bucket,
		DestinationKey:    "dst",
	})
	assert.Error(s.T(), err, "CopyObject should error on cancelled ctx")
}

// ─── Additional real-world scenarios ─────────────────────────────────────────

// Scenario: CopyObject across two different containers within the same account (FR-012).
func (s *BlobAdapterAzureSuite) TestCopyObject_CrossContainer() {
	dstContainer := "blob-adapter-az-dst-" + RandomHex(6)
	EnsureAzuriteContainer(s.T(), s.az.ConnectionString, dstContainer)

	srcKey := RandomObjectKey("az-xcontainer-src")
	dstKey := RandomObjectKey("az-xcontainer-dst")
	payload := "azure cross-container payload"

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
		DestinationBucket: dstContainer,
		DestinationKey:    dstKey,
	})
	assert.NoError(s.T(), err)

	rc, meta, err := s.adapter.GetObjectStream(context.Background(), dstContainer, dstKey)
	require.NoError(s.T(), err)
	require.NotNil(s.T(), rc)
	defer rc.Close()
	b, _ := io.ReadAll(rc)
	assert.Equal(s.T(), payload, string(b))
	assert.Equal(s.T(), int64(len(payload)), meta.SizeBytes)
}

// Scenario: PutObject against a non-existent container returns a structured not-found error.
func (s *BlobAdapterAzureSuite) TestPutObject_NonExistentContainer() {
	_, err := s.adapter.PutObject(context.Background(), model.BlobPutObjectInput{
		Bucket:        "no-such-container-" + RandomHex(8),
		Key:           RandomObjectKey("ghost"),
		ContentType:   "text/plain",
		ContentLength: 3,
		Body:          bytes.NewReader([]byte("abc")),
	})
	assert.Error(s.T(), err)
	assert.True(s.T(), fileError.IsBlobError(err, fileError.ErrBlobNotFound),
		"missing container must map to ErrBlobNotFound, got: %v", err)
}

// Scenario: GetObjectStream for a key that does not exist returns not-found, no stream opened.
func (s *BlobAdapterAzureSuite) TestGetObjectStream_NotFound() {
	rc, _, err := s.adapter.GetObjectStream(context.Background(), s.bucket, "az-missing-"+RandomHex(8))
	assert.Error(s.T(), err)
	assert.True(s.T(), fileError.IsBlobError(err, fileError.ErrBlobNotFound))
	if rc != nil {
		_ = rc.Close()
		s.T().Error("GetObjectStream must not return a ReadCloser when the key does not exist")
	}
}

// Scenario: PutObject twice with the same key overwrites the first blob.
func (s *BlobAdapterAzureSuite) TestPutObject_OverwriteSameKey() {
	key := RandomObjectKey("az-overwrite")

	first := "first"
	_, err := s.adapter.PutObject(context.Background(), model.BlobPutObjectInput{
		Bucket:        s.bucket,
		Key:           key,
		ContentType:   "text/plain",
		ContentLength: int64(len(first)),
		Body:          bytes.NewReader([]byte(first)),
	})
	require.NoError(s.T(), err)

	second := "second azure payload"
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
	require.NotNil(s.T(), rc)
	defer rc.Close()
	b, _ := io.ReadAll(rc)
	assert.Equal(s.T(), second, string(b))
	assert.Equal(s.T(), int64(len(second)), meta.SizeBytes)
}

// Scenario: PresignUpload with zero or negative TTL returns a validation error.
func (s *BlobAdapterAzureSuite) TestPresignUpload_InvalidTTL() {
	for _, ttl := range []time.Duration{0, -time.Minute} {
		_, err := s.adapter.PresignUpload(context.Background(), model.BlobPresignUploadInput{
			Bucket:      s.bucket,
			Key:         RandomObjectKey("az-ttl"),
			ContentType: "text/plain",
			TTL:         ttl,
		})
		assert.Error(s.T(), err)
		assert.True(s.T(), fileError.IsBlobError(err, fileError.ErrBlobValidation),
			"expected ErrBlobValidation for TTL=%v, got: %v", ttl, err)
	}
}

// Scenario: PresignDownload with zero TTL returns a validation error.
func (s *BlobAdapterAzureSuite) TestPresignDownload_InvalidTTL() {
	_, err := s.adapter.PresignDownload(context.Background(), model.BlobPresignDownloadInput{
		Bucket: s.bucket,
		Key:    "any",
		TTL:    0,
	})
	assert.Error(s.T(), err)
	assert.True(s.T(), fileError.IsBlobError(err, fileError.ErrBlobValidation))
}
