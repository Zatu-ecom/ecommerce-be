package blobAdapter

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

	"ecommerce-be/common/helper"
	"ecommerce-be/file/entity"
	fileError "ecommerce-be/file/error"
	"ecommerce-be/file/model"
)

// ─── GCSConfig — typed config struct ──────────────────────────────────────────

// GCSConfig is the typed configuration for a GCS blob adapter.
// The decrypted config_data JSON is unmarshalled into this struct.
type GCSConfig struct {
	// ServiceAccountJSON is the raw JSON string of a Google Cloud Service Account key.
	// Required and sensitive — always stored encrypted.
	ServiceAccountJSON string `json:"service_account_json" validate:"required"`

	// ProjectID is the GCP project identifier. Optional — usually embedded in
	// ServiceAccountJSON but can be overridden here.
	ProjectID string `json:"project_id"`

	// Bucket is the GCS bucket name to upload files into.
	// Required — also mirrored in storage_config.bucket_or_container for display.
	Bucket string `json:"bucket" validate:"required"`

	// Endpoint is an optional custom API endpoint (e.g. fake-gcs-server for tests).
	// Leave empty for real GCS.
	Endpoint string `json:"endpoint"`

	// PublicURLPrefix is an optional CDN / public URL base for serving objects.
	// e.g. "https://cdn.example.com/". Not used by the adapter itself.
	PublicURLPrefix string `json:"public_url_prefix"`
}

func (s *GCSConfig) Encrypt() error {
	key := ResolveEncryptionKey()

	encryptField := func(val string) (string, error) {
		if val == "" {
			return "", nil
		}
		if _, err := helper.Decrypt(val, key); err == nil {
			return val, nil
		}
		return helper.Encrypt(val, key)
	}

	var err error
	if s.ServiceAccountJSON, err = encryptField(s.ServiceAccountJSON); err != nil {
		return fileError.ErrEncryptionFailed.WithMessagef(
			"[gcs] encrypt service_account_json: %v",
			err,
		)
	}
	return nil
}

func (s *GCSConfig) ToMap() map[string]any {
	return map[string]any{
		"service_account_json": s.ServiceAccountJSON,
		"project_id":           s.ProjectID,
		"bucket":               s.Bucket,
		"endpoint":             s.Endpoint,
		"public_url_prefix":    s.PublicURLPrefix,
	}
}

func (s *GCSConfig) Decrypt() error {
	key := ResolveEncryptionKey()

	decryptField := func(val string) (string, error) {
		if val == "" {
			return "", nil
		}
		if dec, err := helper.Decrypt(val, key); err == nil {
			return dec, nil
		}
		return val, nil
	}

	s.ServiceAccountJSON, _ = decryptField(s.ServiceAccountJSON)
	return nil
}

// GCSSchema returns the field descriptor schema for the GCS adapter.
// Called once at startup to populate the schema registry.
func GCSSchema() model.AdapterConfigSchema {
	return model.AdapterConfigSchema{
		AdapterType: entity.AdapterTypeGCS,
		Fields: []model.FieldDescriptor{
			{
				Key:         "service_account_json",
				Label:       "Service Account JSON",
				Type:        model.FieldTypeText,
				Required:    true,
				Sensitive:   true,
				Description: "The full JSON key file for a Google Cloud Service Account with Storage Object Admin permissions.",
				Placeholder: `{"type":"service_account","project_id":"..."}`,
			},
			{
				Key:         "project_id",
				Label:       "Project ID",
				Type:        model.FieldTypeString,
				Required:    false,
				Sensitive:   false,
				Description: "GCP project ID. Usually embedded in the service account JSON — only override if needed.",
			},
			{
				Key:         "bucket",
				Label:       "Bucket Name",
				Type:        model.FieldTypeString,
				Required:    true,
				Sensitive:   false,
				Description: "The GCS bucket where files will be stored.",
				Placeholder: "my-ecommerce-bucket",
			},
			{
				Key:         "endpoint",
				Label:       "Custom Endpoint",
				Type:        model.FieldTypeString,
				Required:    false,
				Sensitive:   false,
				Description: "Override the GCS API endpoint (e.g. http://localhost:4443 for fake-gcs-server). Leave empty for production.",
			},
			{
				Key:         "public_url_prefix",
				Label:       "Public URL Prefix",
				Type:        model.FieldTypeString,
				Required:    false,
				Sensitive:   false,
				Description: "Optional CDN or public base URL for serving objects (e.g. https://cdn.example.com/).",
			},
		},
	}
}

// ─── gcsAdapter — internal adapter struct ─────────────────────────────────────
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

// NewGCSAdapterFromMap constructs a GCS BlobAdapter from a raw config map.
func NewGCSAdapterFromMap(ctx context.Context, raw map[string]any) (BlobAdapter, error) {
	cfg, err := ParseAndValidateConfig[GCSConfig](raw)
	if err != nil {
		return nil, err
	}
	return NewGCSAdapter(ctx, cfg)
}

// NewGCSAdapter constructs a GCS BlobAdapter from the supplied config.
// Returns ErrBlobValidation when ServiceAccountJSON is malformed or missing required fields.
// Returns ErrBlobFactoryInit when the underlying GCS client cannot be created.
func NewGCSAdapter(ctx context.Context, cfg *GCSConfig) (BlobAdapter, error) {
	if strings.TrimSpace(cfg.ServiceAccountJSON) == "" {
		return nil, fileError.ErrBlobValidation.WithMessagef("[gcs] missing service_account_json")
	}

	var sa serviceAccountFields
	if err := json.Unmarshal([]byte(cfg.ServiceAccountJSON), &sa); err != nil {
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

	clientOpts := buildGCSClientOptions(cfg)
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
		endpoint:   cfg.Endpoint,
	}, nil
}

func buildGCSClientOptions(cfg *GCSConfig) []option.ClientOption {
	if cfg.Endpoint != "" {
		// Test / custom-endpoint mode: route JSON API calls to fake-gcs-server.
		// Use plain HTTP transport and skip authentication. WithJSONReads keeps
		// Reader traffic on the JSON API because fake-gcs-server doesn't serve
		// the XML read path that the SDK uses by default.
		return []option.ClientOption{
			option.WithHTTPClient(&http.Client{}),
			option.WithEndpoint(cfg.Endpoint + "/storage/v1/"),
			option.WithoutAuthentication(),
			storage.WithJSONReads(),
		}
	}
	return []option.ClientOption{
		option.WithCredentialsJSON([]byte(cfg.ServiceAccountJSON)),
	}
}

const gcsServiceAccountJSONKey = "service_account_json"

// normalizeGCSConfigMap shallow-clones raw and coerces service_account_json from a nested
// JSON object (map) into a JSON string so GCSConfig.ServiceAccountJSON unmarshaling matches SaveConfig API payloads.
func normalizeGCSConfigMap(raw map[string]any) (map[string]any, error) {
	if raw == nil {
		return nil, nil
	}
	out := make(map[string]any, len(raw))
	for k, v := range raw {
		out[k] = v
	}
	v, ok := out[gcsServiceAccountJSONKey]
	if !ok || v == nil {
		return out, nil
	}
	switch t := v.(type) {
	case string:
		return out, nil
	case map[string]any:
		b, err := json.Marshal(t)
		if err != nil {
			return nil, fileError.ErrBlobValidation.WithMessagef(
				"[gcs] service_account_json object could not be serialized: %v",
				err,
			)
		}
		out[gcsServiceAccountJSONKey] = string(b)
		return out, nil
	default:
		return nil, fileError.ErrBlobValidation.WithMessagef(
			"[gcs] service_account_json must be a JSON string or object, got %T",
			v,
		)
	}
}

// ParseAndValidateConfig parses and validates a raw config map into a typed GCSConfig.
// Returns ErrBlobValidation when required fields are missing.
func (a *gcsAdapter) ParseAndValidateConfig(
	raw map[string]any,
) (BlobConfig, error) {
	normalized, err := normalizeGCSConfigMap(raw)
	if err != nil {
		return nil, err
	}
	return ParseAndValidateConfig[GCSConfig](normalized)
}

// PingStorage checks bucket access via the JSON API bucket metadata call.
func (a *gcsAdapter) PingStorage(ctx context.Context, bucketOrContainer string) error {
	name := strings.TrimSpace(bucketOrContainer)
	if name == "" {
		return fileError.ErrBlobValidation.WithMessagef("[gcs] ping_storage: bucket is required")
	}
	if err := ctx.Err(); err != nil {
		return a.mapErr("ping_storage", err, "context cancelled")
	}
	_, err := a.client.Bucket(name).Attrs(ctx)
	if err != nil {
		return a.mapErr("ping_storage", err, "could not access bucket")
	}
	return nil
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
	if disposition := strings.TrimSpace(in.Disposition); disposition != "" {
		opts.QueryParameters = url.Values{
			"response-content-disposition": []string{disposition},
		}
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

// gcsDetailFromGoogleError returns the provider-facing text from a googleapi.Error.
// GCS JSON API errors are surfaced to API clients to aid debugging (no raw credentials in these payloads).
func gcsDetailFromGoogleError(gErr *googleapi.Error) string {
	if gErr == nil {
		return ""
	}
	if s := strings.TrimSpace(gErr.Message); s != "" {
		return s
	}
	if len(gErr.Errors) > 0 {
		return strings.TrimSpace(gErr.Errors[0].Message)
	}
	return ""
}

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
		detail := gcsDetailFromGoogleError(gErr)
		switch gErr.Code {
		case http.StatusNotFound:
			if detail != "" {
				return fileError.ErrBlobNotFound.WithMessagef("[gcs/%s] %s: %s", op, msg, detail)
			}
			return fileError.ErrBlobNotFound.WithMessagef(
				"[gcs/%s] %s: not found (HTTP 404)",
				op,
				msg,
			)
		case http.StatusForbidden, http.StatusUnauthorized:
			if detail != "" {
				return fileError.ErrBlobPermissionDenied.WithMessagef(
					"[gcs/%s] %s: %s",
					op,
					msg,
					detail,
				)
			}
			return fileError.ErrBlobPermissionDenied.WithMessagef(
				"[gcs/%s] %s: access denied",
				op,
				msg,
			)
		case http.StatusBadRequest:
			if detail != "" {
				return fileError.ErrBlobValidation.WithMessagef("[gcs/%s] %s: %s", op, msg, detail)
			}
			return fileError.ErrBlobValidation.WithMessagef("[gcs/%s] %s: bad request", op, msg)
		default:
			if detail != "" {
				return fileError.ErrBlobInternal.WithMessagef(
					"[gcs/%s] %s: %s (HTTP %d)",
					op,
					msg,
					detail,
					gErr.Code,
				)
			}
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
