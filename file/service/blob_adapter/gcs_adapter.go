package blob_adapter

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"

	fileError "ecommerce-be/file/error"
	"ecommerce-be/file/model"
)

// GCSOptions contains all parameters needed to initialise a GCS adapter.
// ServiceAccountJSON is mandatory (JSON-encoded service account credential blob).
// Endpoint is optional; when set, all API calls are routed to that host instead
// of storage.googleapis.com (used with fake-gcs-server in integration tests).
type GCSOptions struct {
	ServiceAccountJSON string
	ProjectID          string
	Endpoint           string
}

type gcsAdapter struct {
	client     *storage.Client
	accessID   string
	privateKey []byte
	endpoint   string
}

// Compile-time assertion: gcsAdapter satisfies BlobAdapter.
var _ BlobAdapter = (*gcsAdapter)(nil)

// serviceAccountFields is used only to extract signing credentials from the JSON blob.
type serviceAccountFields struct {
	ClientEmail string `json:"client_email"`
	PrivateKey  string `json:"private_key"`
}

// NewGCSAdapter constructs a GCS BlobAdapter from the supplied options.
// Returns ErrBlobValidation when ServiceAccountJSON is malformed or missing required fields.
// Returns ErrBlobFactoryInit when the underlying GCS client cannot be created.
func NewGCSAdapter(ctx context.Context, opts GCSOptions) (BlobAdapter, error) {
	if strings.TrimSpace(opts.ServiceAccountJSON) == "" {
		return nil, fileError.ErrBlobValidation.WithMessagef("[gcs] missing service_account_json")
	}

	var sa serviceAccountFields
	if err := json.Unmarshal([]byte(opts.ServiceAccountJSON), &sa); err != nil {
		return nil, fileError.ErrBlobValidation.WithMessagef(
			"[gcs] invalid service_account_json: %v",
			err,
		)
	}
	if sa.ClientEmail == "" {
		return nil, fileError.ErrBlobValidation.WithMessagef(
			"[gcs] service_account_json missing client_email",
		)
	}
	if sa.PrivateKey == "" {
		return nil, fileError.ErrBlobValidation.WithMessagef(
			"[gcs] service_account_json missing private_key",
		)
	}

	clientOpts := buildGCSClientOptions(opts)
	client, err := storage.NewClient(ctx, clientOpts...)
	if err != nil {
		return nil, fileError.ErrBlobFactoryInit.WithMessagef(
			"[gcs] failed to create client: %v",
			err,
		)
	}

	return &gcsAdapter{
		client:     client,
		accessID:   sa.ClientEmail,
		privateKey: []byte(sa.PrivateKey),
		endpoint:   opts.Endpoint,
	}, nil
}

func buildGCSClientOptions(opts GCSOptions) []option.ClientOption {
	if opts.Endpoint != "" {
		// Test / custom-endpoint mode: route JSON API calls to fake-gcs-server.
		// Use plain HTTP transport and skip authentication. WithJSONReads keeps
		// Reader traffic on the JSON API because fake-gcs-server doesn't serve
		// the XML read path that the SDK uses by default.
		return []option.ClientOption{
			option.WithHTTPClient(&http.Client{}),
			option.WithEndpoint(opts.Endpoint + "/storage/v1/"),
			option.WithoutAuthentication(),
			storage.WithJSONReads(),
		}
	}
	return []option.ClientOption{
		option.WithCredentialsJSON([]byte(opts.ServiceAccountJSON)),
	}
}

// PutObject uploads an object to the specified bucket and key.
func (a *gcsAdapter) PutObject(
	ctx context.Context,
	in model.BlobPutObjectInput,
) (model.BlobPutObjectOutput, error) {
	if err := gcsValidatePutInput(in); err != nil {
		return model.BlobPutObjectOutput{}, err
	}

	obj := a.client.Bucket(in.Bucket).Object(in.Key)
	w := obj.NewWriter(ctx)
	w.ContentType = in.ContentType

	if _, err := io.Copy(w, in.Body); err != nil {
		_ = w.Close()
		return model.BlobPutObjectOutput{}, a.mapErr("put_object", err, "upload failed")
	}
	if err := w.Close(); err != nil {
		return model.BlobPutObjectOutput{}, a.mapErr("put_object", err, "finalise upload failed")
	}

	attrs, err := obj.Attrs(ctx)
	if err != nil {
		return model.BlobPutObjectOutput{}, a.mapErr(
			"put_object",
			err,
			"read attrs after upload failed",
		)
	}

	return model.BlobPutObjectOutput{
		Key:  in.Key,
		ETag: strings.Trim(attrs.Etag, "\""),
	}, nil
}

// DeleteObject removes the object identified by bucket + key.
func (a *gcsAdapter) DeleteObject(ctx context.Context, bucket, key string) error {
	if err := gcsValidateBucketKey(bucket, key); err != nil {
		return err
	}
	err := a.client.Bucket(bucket).Object(key).Delete(ctx)
	if err != nil {
		return a.mapErr("delete_object", err, "delete failed")
	}
	return nil
}

// HeadObject fetches metadata for the object at bucket/key without downloading the body.
func (a *gcsAdapter) HeadObject(
	ctx context.Context,
	bucket, key string,
) (model.BlobObjectMeta, error) {
	if err := gcsValidateBucketKey(bucket, key); err != nil {
		return model.BlobObjectMeta{}, err
	}
	attrs, err := a.client.Bucket(bucket).Object(key).Attrs(ctx)
	if err != nil {
		return model.BlobObjectMeta{}, a.mapErr("head_object", err, "attrs fetch failed")
	}
	return model.BlobObjectMeta{
		ContentType:  attrs.ContentType,
		SizeBytes:    attrs.Size,
		LastModified: attrs.Updated,
		ETag:         strings.Trim(attrs.Etag, "\""),
	}, nil
}

// GetObjectStream opens a streaming download for bucket/key.
// The caller must close the returned io.ReadCloser.
func (a *gcsAdapter) GetObjectStream(
	ctx context.Context,
	bucket, key string,
) (io.ReadCloser, model.BlobObjectMeta, error) {
	if err := gcsValidateBucketKey(bucket, key); err != nil {
		return nil, model.BlobObjectMeta{}, err
	}

	attrs, err := a.client.Bucket(bucket).Object(key).Attrs(ctx)
	if err != nil {
		return nil, model.BlobObjectMeta{}, a.mapErr("get_object_stream", err, "attrs fetch failed")
	}

	r, err := a.client.Bucket(bucket).Object(key).NewReader(ctx)
	if err != nil {
		return nil, model.BlobObjectMeta{}, a.mapErr("get_object_stream", err, "open reader failed")
	}

	meta := model.BlobObjectMeta{
		ContentType:  attrs.ContentType,
		SizeBytes:    attrs.Size,
		LastModified: attrs.Updated,
		ETag:         strings.Trim(attrs.Etag, "\""),
	}
	return r, meta, nil
}

// PresignUpload generates a time-limited URL that allows a client to PUT an object directly.
// Note: the URL uses the XMLApi signing scheme (V4). When an endpoint override is configured
// (e.g. fake-gcs-server), the Hostname is derived from that endpoint.
func (a *gcsAdapter) PresignUpload(
	ctx context.Context,
	in model.BlobPresignUploadInput,
) (model.BlobPresignOutput, error) {
	if err := gcsValidatePresignUploadInput(in); err != nil {
		return model.BlobPresignOutput{}, err
	}
	if err := ctx.Err(); err != nil {
		return model.BlobPresignOutput{}, a.mapErr("presign_upload", err, "context cancelled")
	}

	opts := &storage.SignedURLOptions{
		GoogleAccessID: a.accessID,
		PrivateKey:     a.privateKey,
		Method:         http.MethodPut,
		Expires:        time.Now().Add(in.TTL),
		Scheme:         storage.SigningSchemeV4,
		ContentType:    in.ContentType,
	}
	if h := gcsSigningHostname(a.endpoint); h != "" {
		opts.Hostname = h
	}

	signedURL, err := a.client.Bucket(in.Bucket).SignedURL(in.Key, opts)
	if err != nil {
		return model.BlobPresignOutput{}, a.mapErr("presign_upload", err, "sign URL failed")
	}
	return model.BlobPresignOutput{
		URL:       signedURL,
		ExpiresAt: time.Now().Add(in.TTL),
	}, nil
}

// PresignDownload generates a time-limited URL that allows a client to GET an object directly.
func (a *gcsAdapter) PresignDownload(
	ctx context.Context,
	in model.BlobPresignDownloadInput,
) (model.BlobPresignOutput, error) {
	if err := gcsValidatePresignDownloadInput(in); err != nil {
		return model.BlobPresignOutput{}, err
	}
	if err := ctx.Err(); err != nil {
		return model.BlobPresignOutput{}, a.mapErr("presign_download", err, "context cancelled")
	}

	opts := &storage.SignedURLOptions{
		GoogleAccessID: a.accessID,
		PrivateKey:     a.privateKey,
		Method:         http.MethodGet,
		Expires:        time.Now().Add(in.TTL),
		Scheme:         storage.SigningSchemeV4,
	}
	if h := gcsSigningHostname(a.endpoint); h != "" {
		opts.Hostname = h
	}

	signedURL, err := a.client.Bucket(in.Bucket).SignedURL(in.Key, opts)
	if err != nil {
		return model.BlobPresignOutput{}, a.mapErr("presign_download", err, "sign URL failed")
	}
	return model.BlobPresignOutput{
		URL:       signedURL,
		ExpiresAt: time.Now().Add(in.TTL),
	}, nil
}

// CopyObject copies an object from source bucket/key to destination bucket/key.
func (a *gcsAdapter) CopyObject(ctx context.Context, in model.BlobCopyObjectInput) error {
	if strings.TrimSpace(in.SourceBucket) == "" || strings.TrimSpace(in.SourceKey) == "" ||
		strings.TrimSpace(
			in.DestinationBucket,
		) == "" || strings.TrimSpace(in.DestinationKey) == "" {
		return fileError.ErrBlobValidation.WithMessagef(
			"[gcs] copy_object: missing source or destination bucket/key",
		)
	}

	src := a.client.Bucket(in.SourceBucket).Object(in.SourceKey)
	dst := a.client.Bucket(in.DestinationBucket).Object(in.DestinationKey)

	if _, err := dst.CopierFrom(src).Run(ctx); err != nil {
		return a.mapErr("copy_object", err, "copy failed")
	}
	return nil
}

// ─── Signing helpers ──────────────────────────────────────────────────────────

// gcsSigningHostname derives the hostname to use in signed URLs from a custom endpoint.
// Returns empty string when no override is needed (production path).
func gcsSigningHostname(endpoint string) string {
	if endpoint == "" {
		return ""
	}
	u, err := url.Parse(endpoint)
	if err != nil || u.Host == "" {
		return ""
	}
	return u.Host
}

// ─── Validation helpers ───────────────────────────────────────────────────────

func gcsValidateBucketKey(bucket, key string) error {
	if strings.TrimSpace(bucket) == "" || strings.TrimSpace(key) == "" {
		return fileError.ErrBlobValidation.WithMessagef("[gcs] missing bucket or key")
	}
	return nil
}

func gcsValidatePutInput(in model.BlobPutObjectInput) error {
	if strings.TrimSpace(in.Bucket) == "" || strings.TrimSpace(in.Key) == "" ||
		strings.TrimSpace(in.ContentType) == "" || in.Body == nil || in.ContentLength < 0 {
		return fileError.ErrBlobValidation.WithMessagef(
			"[gcs] put_object: missing bucket/key/content-type or negative content-length",
		)
	}
	return nil
}

func gcsValidatePresignUploadInput(in model.BlobPresignUploadInput) error {
	if strings.TrimSpace(in.Bucket) == "" || strings.TrimSpace(in.Key) == "" ||
		strings.TrimSpace(in.ContentType) == "" {
		return fileError.ErrBlobValidation.WithMessagef(
			"[gcs] presign_upload: missing bucket/key/content-type",
		)
	}
	if in.TTL <= 0 {
		return fileError.ErrBlobValidation.WithMessagef("[gcs] presign_upload: ttl must be > 0")
	}
	return nil
}

func gcsValidatePresignDownloadInput(in model.BlobPresignDownloadInput) error {
	if strings.TrimSpace(in.Bucket) == "" || strings.TrimSpace(in.Key) == "" {
		return fileError.ErrBlobValidation.WithMessagef(
			"[gcs] presign_download: missing bucket/key",
		)
	}
	if in.TTL <= 0 {
		return fileError.ErrBlobValidation.WithMessagef("[gcs] presign_download: ttl must be > 0")
	}
	return nil
}

// ─── Error mapping ────────────────────────────────────────────────────────────

func (a *gcsAdapter) mapErr(op string, err error, msg string) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return fileError.ErrBlobNetwork.WithMessagef(
			"[gcs/%s] %s: context cancelled or deadline exceeded",
			op,
			msg,
		)
	}
	if errors.Is(err, storage.ErrObjectNotExist) || errors.Is(err, storage.ErrBucketNotExist) {
		return fileError.ErrBlobNotFound.WithMessagef("[gcs/%s] %s: not found", op, msg)
	}

	var gErr *googleapi.Error
	if errors.As(err, &gErr) {
		switch gErr.Code {
		case http.StatusNotFound:
			return fileError.ErrBlobNotFound.WithMessagef(
				"[gcs/%s] %s: not found (HTTP 404)",
				op,
				msg,
			)
		case http.StatusForbidden, http.StatusUnauthorized:
			return fileError.ErrBlobPermissionDenied.WithMessagef(
				"[gcs/%s] %s: access denied",
				op,
				msg,
			)
		case http.StatusBadRequest:
			return fileError.ErrBlobValidation.WithMessagef("[gcs/%s] %s: bad request", op, msg)
		default:
			return fileError.ErrBlobInternal.WithMessagef(
				"[gcs/%s] %s: provider error HTTP %d",
				op,
				msg,
				gErr.Code,
			)
		}
	}

	var netErr net.Error
	if errors.As(err, &netErr) {
		return fileError.ErrBlobNetwork.WithMessagef("[gcs/%s] %s: network error", op, msg)
	}

	return fileError.ErrBlobInternal.WithMessagef("[gcs/%s] %s: unexpected error", op, msg)
}
