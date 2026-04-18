package file_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"testing"
	"time"

	fileError "ecommerce-be/file/error"
	"ecommerce-be/file/model"
	"ecommerce-be/file/service/blob_adapter"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type BlobAdapterS3Suite struct {
	suite.Suite

	minio   *MinioContainer
	adapter blob_adapter.BlobAdapter
	bucket  string
}

func (s *BlobAdapterS3Suite) SetupSuite() {
	s.bucket = "blob-adapter-it-" + RandomHex(6)
	s.minio = SetupMinio(s.T(), s.bucket)

	a, err := blob_adapter.NewS3CompatibleAdapter(
		context.Background(),
		blob_adapter.S3CompatibleOptions{
			Endpoint:        s.minio.Endpoint,
			Region:          s.minio.Region,
			ForcePathStyle:  true,
			AccessKeyID:     s.minio.AccessKey,
			SecretAccessKey: s.minio.SecretKey,
		},
	)
	if err != nil {
		s.T().Fatal(err)
	}
	s.adapter = a
}

func (s *BlobAdapterS3Suite) TearDownSuite() {
	if s.minio != nil {
		s.minio.Cleanup(s.T())
	}
}

func TestBlobAdapterS3Suite(t *testing.T) {
	suite.Run(t, new(BlobAdapterS3Suite))
}

// Scenario: PutObject stores object and returns ETag + Key.
func (s *BlobAdapterS3Suite) TestPutObject_Success() {
	key := RandomObjectKey("put")
	body := "hello s3"
	r := bytes.NewReader([]byte(body))

	out, err := s.adapter.PutObject(context.Background(), model.BlobPutObjectInput{
		Bucket:        s.bucket,
		Key:           key,
		ContentType:   "text/plain",
		ContentLength: int64(len(body)),
		Body:          r,
	})
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), key, out.Key)
	assert.NotEmpty(s.T(), out.ETag)
}

// Scenario: HeadObject returns correct metadata.
func (s *BlobAdapterS3Suite) TestHeadObject_Success() {
	key := RandomObjectKey("head")
	payload := "meta"
	_, err := s.adapter.PutObject(context.Background(), model.BlobPutObjectInput{
		Bucket:        s.bucket,
		Key:           key,
		ContentType:   "text/plain",
		ContentLength: int64(len(payload)),
		Body:          bytes.NewReader([]byte(payload)),
	})
	assert.NoError(s.T(), err)

	meta, err := s.adapter.HeadObject(context.Background(), s.bucket, key)
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), "text/plain", meta.ContentType)
	assert.Equal(s.T(), int64(len(payload)), meta.SizeBytes)
	assert.NotEmpty(s.T(), meta.ETag)
	assert.False(s.T(), meta.LastModified.IsZero())
}

// Scenario: GetObjectStream returns stream + metadata; caller can read full content.
func (s *BlobAdapterS3Suite) TestGetObjectStream_Success() {
	key := RandomObjectKey("get")
	payload := "stream content"
	_, err := s.adapter.PutObject(context.Background(), model.BlobPutObjectInput{
		Bucket:        s.bucket,
		Key:           key,
		ContentType:   "text/plain",
		ContentLength: int64(len(payload)),
		Body:          bytes.NewReader([]byte(payload)),
	})
	assert.NoError(s.T(), err)

	rc, meta, err := s.adapter.GetObjectStream(context.Background(), s.bucket, key)
	assert.NoError(s.T(), err)
	defer rc.Close()

	b, err := io.ReadAll(rc)
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), payload, string(b))
	assert.Equal(s.T(), int64(len(payload)), meta.SizeBytes)
}

// Scenario: PresignUpload returns URL; client PUT succeeds.
func (s *BlobAdapterS3Suite) TestPresignUpload_Success() {
	key := RandomObjectKey("presign-upload")
	ttl := 1 * time.Minute

	p, err := s.adapter.PresignUpload(context.Background(), model.BlobPresignUploadInput{
		Bucket:      s.bucket,
		Key:         key,
		ContentType: "text/plain",
		TTL:         ttl,
	})
	assert.NoError(s.T(), err)
	assert.NotEmpty(s.T(), p.URL)

	req, err := http.NewRequest(http.MethodPut, p.URL, bytes.NewReader([]byte("via presign")))
	assert.NoError(s.T(), err)
	req.Header.Set("Content-Type", "text/plain")

	resp, err := http.DefaultClient.Do(req)
	assert.NoError(s.T(), err)
	defer resp.Body.Close()
	assert.True(s.T(), resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusNoContent)

	_, err = s.adapter.HeadObject(context.Background(), s.bucket, key)
	assert.NoError(s.T(), err)
}

// Scenario: PresignDownload returns URL; client GET retrieves correct content.
func (s *BlobAdapterS3Suite) TestPresignDownload_Success() {
	key := RandomObjectKey("presign-download")
	payload := "download me"
	_, err := s.adapter.PutObject(context.Background(), model.BlobPutObjectInput{
		Bucket:        s.bucket,
		Key:           key,
		ContentType:   "text/plain",
		ContentLength: int64(len(payload)),
		Body:          bytes.NewReader([]byte(payload)),
	})
	assert.NoError(s.T(), err)

	p, err := s.adapter.PresignDownload(context.Background(), model.BlobPresignDownloadInput{
		Bucket: s.bucket,
		Key:    key,
		TTL:    1 * time.Minute,
	})
	assert.NoError(s.T(), err)
	assert.NotEmpty(s.T(), p.URL)

	resp, err := http.Get(p.URL)
	assert.NoError(s.T(), err)
	defer resp.Body.Close()
	assert.Equal(s.T(), http.StatusOK, resp.StatusCode)
	b, err := io.ReadAll(resp.Body)
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), payload, string(b))
}

// Scenario: CopyObject copies existing object to destination key.
func (s *BlobAdapterS3Suite) TestCopyObject_Success() {
	srcKey := RandomObjectKey("copy-src")
	dstKey := RandomObjectKey("copy-dst")
	payload := "copy me"
	_, err := s.adapter.PutObject(context.Background(), model.BlobPutObjectInput{
		Bucket:        s.bucket,
		Key:           srcKey,
		ContentType:   "text/plain",
		ContentLength: int64(len(payload)),
		Body:          bytes.NewReader([]byte(payload)),
	})
	assert.NoError(s.T(), err)

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
func (s *BlobAdapterS3Suite) TestDeleteObject_Success() {
	key := RandomObjectKey("delete")
	_, err := s.adapter.PutObject(context.Background(), model.BlobPutObjectInput{
		Bucket:        s.bucket,
		Key:           key,
		ContentType:   "text/plain",
		ContentLength: int64(len("x")),
		Body:          bytes.NewReader([]byte("x")),
	})
	assert.NoError(s.T(), err)

	err = s.adapter.DeleteObject(context.Background(), s.bucket, key)
	assert.NoError(s.T(), err)

	_, err = s.adapter.HeadObject(context.Background(), s.bucket, key)
	assert.Error(s.T(), err)
	assert.True(s.T(), fileError.IsBlobError(err, fileError.ErrBlobNotFound))
}

// Scenario: Wrong credentials return a structured, secret-free permission-denied error.
// Per spec FR-008 / User Story 1 scenario #8, invalid credentials must surface as
// permission-denied, not as "not found" — otherwise operators chase ghost bugs.
func (s *BlobAdapterS3Suite) TestInvalidCredentials_ReturnsPermissionDenied() {
	a, err := blob_adapter.NewS3CompatibleAdapter(
		context.Background(),
		blob_adapter.S3CompatibleOptions{
			Endpoint:        s.minio.Endpoint,
			Region:          s.minio.Region,
			ForcePathStyle:  true,
			AccessKeyID:     s.minio.AccessKey,
			SecretAccessKey: "WRONG_SECRET",
		},
	)
	assert.NoError(s.T(), err)

	_, err = a.HeadObject(context.Background(), s.bucket, "does-not-matter")
	assert.Error(s.T(), err)
	assert.True(s.T(),
		fileError.IsBlobError(err, fileError.ErrBlobPermissionDenied),
		"wrong credentials must map to ErrBlobPermissionDenied, got: %v", err,
	)
	assert.NotContains(s.T(), err.Error(), s.minio.SecretKey)
}

// Scenario: Context cancelled mid-PutObject returns quickly.
func (s *BlobAdapterS3Suite) TestPutObject_ContextCancel_IOHeavy() {
	key := RandomObjectKey("ctx-put")

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
	case <-time.After(3 * time.Second):
		s.T().Fatal("put did not return after cancel")
	}
}

// Scenario: Already-cancelled context for metadata/presign methods returns error quickly.
func (s *BlobAdapterS3Suite) TestAlreadyCancelledContext_MetadataPresign() {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := s.adapter.HeadObject(ctx, s.bucket, "nope")
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

// Scenario: CopyObject across two different buckets within the same account (FR-012).
func (s *BlobAdapterS3Suite) TestCopyObject_CrossBucket() {
	dstBucket := "blob-adapter-it-dst-" + RandomHex(6)
	EnsureS3Bucket(s.T(), s.minio.Endpoint, s.minio.Region, s.minio.AccessKey, s.minio.SecretKey, dstBucket)

	srcKey := RandomObjectKey("xbucket-src")
	dstKey := RandomObjectKey("xbucket-dst")
	payload := "cross-bucket payload"

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
// Per spec Edge Cases.
func (s *BlobAdapterS3Suite) TestPutObject_NonExistentBucket() {
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
func (s *BlobAdapterS3Suite) TestGetObjectStream_NotFound() {
	rc, _, err := s.adapter.GetObjectStream(context.Background(), s.bucket, "missing-"+RandomHex(8))
	assert.Error(s.T(), err)
	assert.True(s.T(), fileError.IsBlobError(err, fileError.ErrBlobNotFound))
	if rc != nil {
		_ = rc.Close()
		s.T().Error("GetObjectStream must not return a ReadCloser when the key does not exist")
	}
}

// Scenario: PutObject twice with the same key overwrites the first object.
// Real-world flow: client retries / re-uploads a file.
func (s *BlobAdapterS3Suite) TestPutObject_OverwriteSameKey() {
	key := RandomObjectKey("overwrite")

	first := "first version"
	_, err := s.adapter.PutObject(context.Background(), model.BlobPutObjectInput{
		Bucket:        s.bucket,
		Key:           key,
		ContentType:   "text/plain",
		ContentLength: int64(len(first)),
		Body:          bytes.NewReader([]byte(first)),
	})
	require.NoError(s.T(), err)

	second := "second version — longer"
	out2, err := s.adapter.PutObject(context.Background(), model.BlobPutObjectInput{
		Bucket:        s.bucket,
		Key:           key,
		ContentType:   "text/plain",
		ContentLength: int64(len(second)),
		Body:          bytes.NewReader([]byte(second)),
	})
	require.NoError(s.T(), err)
	assert.NotEmpty(s.T(), out2.ETag)

	rc, meta, err := s.adapter.GetObjectStream(context.Background(), s.bucket, key)
	require.NoError(s.T(), err)
	defer rc.Close()
	b, _ := io.ReadAll(rc)
	assert.Equal(s.T(), second, string(b))
	assert.Equal(s.T(), int64(len(second)), meta.SizeBytes)
}

// Scenario: HeadObject.LastModified is recent (within 60s of now), not just non-zero.
func (s *BlobAdapterS3Suite) TestHeadObject_LastModifiedRecent() {
	key := RandomObjectKey("lm")
	before := time.Now().Add(-1 * time.Minute)
	_, err := s.adapter.PutObject(context.Background(), model.BlobPutObjectInput{
		Bucket:        s.bucket,
		Key:           key,
		ContentType:   "text/plain",
		ContentLength: 1,
		Body:          bytes.NewReader([]byte("x")),
	})
	require.NoError(s.T(), err)

	meta, err := s.adapter.HeadObject(context.Background(), s.bucket, key)
	require.NoError(s.T(), err)
	after := time.Now().Add(1 * time.Minute)
	assert.True(s.T(), meta.LastModified.After(before) && meta.LastModified.Before(after),
		"LastModified %v must be within [%v, %v]", meta.LastModified, before, after)
}

// Scenario: PresignUpload with a zero or negative TTL returns a validation error
// before any network call. Per spec Edge Cases and FR-009.
func (s *BlobAdapterS3Suite) TestPresignUpload_InvalidTTL() {
	cases := []struct {
		name string
		ttl  time.Duration
	}{
		{"zero TTL", 0},
		{"negative TTL", -5 * time.Minute},
	}
	for _, tc := range cases {
		s.Run(tc.name, func() {
			_, err := s.adapter.PresignUpload(context.Background(), model.BlobPresignUploadInput{
				Bucket:      s.bucket,
				Key:         RandomObjectKey("ttl"),
				ContentType: "text/plain",
				TTL:         tc.ttl,
			})
			assert.Error(s.T(), err)
			assert.True(s.T(), fileError.IsBlobError(err, fileError.ErrBlobValidation),
				"expected ErrBlobValidation for %s, got: %v", tc.name, err)
		})
	}
}

// Scenario: PresignDownload with a zero TTL returns a validation error.
func (s *BlobAdapterS3Suite) TestPresignDownload_InvalidTTL() {
	_, err := s.adapter.PresignDownload(context.Background(), model.BlobPresignDownloadInput{
		Bucket: s.bucket,
		Key:    "any",
		TTL:    0,
	})
	assert.Error(s.T(), err)
	assert.True(s.T(), fileError.IsBlobError(err, fileError.ErrBlobValidation))
}
