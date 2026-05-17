package blobAdapter

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awshttp "github.com/aws/aws-sdk-go-v2/aws/transport/http"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"

	"ecommerce-be/common/helper"
	"ecommerce-be/file/entity"
	fileError "ecommerce-be/file/error"
	"ecommerce-be/file/model"
)

// ─── S3Config — typed config struct ───────────────────────────────────────────

// S3Config is the typed configuration for an S3-compatible blob adapter.
// Works with AWS S3, MinIO, Cloudflare R2, Backblaze B2, and any S3-protocol store.
type S3Config struct {
	// AccessKeyID is the access key (required, sensitive).
	AccessKeyID string `json:"access_key_id"     validate:"required"`
	// SecretAccessKey is the secret key (required, sensitive).
	SecretAccessKey string `json:"secret_access_key" validate:"required"`
	// SessionToken is an optional temporary STS token (sensitive).
	SessionToken string `json:"session_token"`
	// Bucket is the S3 bucket name. Required.
	Bucket string `json:"bucket"            validate:"required"`
	// Region defaults to "us-east-1" when empty.
	Region string `json:"region"`
	// Endpoint is required for non-AWS S3-compatible stores (e.g. http://minio:9000).
	Endpoint string `json:"endpoint"`
	// ForcePathStyle uses path-style URLs (required for MinIO and most compatible stores).
	ForcePathStyle bool `json:"force_path_style"`
}

func (s *S3Config) Encrypt() error {
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
	if s.AccessKeyID, err = encryptField(s.AccessKeyID); err != nil {
		return fileError.ErrEncryptionFailed.WithMessagef(
			"[s3_compatible] encrypt access_key_id: %v",
			err,
		)
	}
	if s.SecretAccessKey, err = encryptField(s.SecretAccessKey); err != nil {
		return fileError.ErrEncryptionFailed.WithMessagef(
			"[s3_compatible] encrypt secret_access_key: %v",
			err,
		)
	}
	if s.SessionToken, err = encryptField(s.SessionToken); err != nil {
		return fileError.ErrEncryptionFailed.WithMessagef(
			"[s3_compatible] encrypt session_token: %v",
			err,
		)
	}
	return nil
}

func (s *S3Config) ToMap() map[string]any {
	return map[string]any{
		"access_key_id":     s.AccessKeyID,
		"secret_access_key": s.SecretAccessKey,
		"session_token":     s.SessionToken,
		"bucket":            s.Bucket,
		"region":            s.Region,
		"endpoint":          s.Endpoint,
		"force_path_style":  s.ForcePathStyle,
	}
}

func (s *S3Config) Decrypt() error {
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

	s.AccessKeyID, _ = decryptField(s.AccessKeyID)
	s.SecretAccessKey, _ = decryptField(s.SecretAccessKey)
	s.SessionToken, _ = decryptField(s.SessionToken)
	return nil
}

type s3CompatibleAdapter struct {
	client    *s3.Client
	presigner *s3.PresignClient
}

// Compile-time assertion: s3CompatibleAdapter satisfies BlobAdapter.
var _ BlobAdapter = (*s3CompatibleAdapter)(nil)

// NewS3CompatibleAdapterFromMap constructs an S3-compatible BlobAdapter from a raw config map.
func NewS3CompatibleAdapterFromMap(ctx context.Context, raw map[string]any) (BlobAdapter, error) {
	cfg, err := ParseAndValidateConfig[S3Config](raw)
	if err != nil {
		return nil, err
	}
	return NewS3CompatibleAdapter(ctx, cfg)
}

// NewS3CompatibleAdapter constructs an S3-compatible BlobAdapter from the supplied config.
// Returns ErrBlobValidation if required fields are missing or the endpoint is unparseable.
func NewS3CompatibleAdapter(ctx context.Context, cfg *S3Config) (BlobAdapter, error) {
	endpoint := strings.TrimSpace(cfg.Endpoint)
	if endpoint == "" {
		return nil, fileError.ErrBlobValidation.WithMessagef("[s3_compatible] missing endpoint")
	}
	if _, err := url.Parse(endpoint); err != nil {
		return nil, fileError.ErrBlobValidation.WithMessagef(
			"[s3_compatible] invalid endpoint url: %v",
			err,
		)
	}
	region := strings.TrimSpace(cfg.Region)
	if region == "" {
		region = "us-east-1"
	}
	if strings.TrimSpace(cfg.AccessKeyID) == "" || strings.TrimSpace(cfg.SecretAccessKey) == "" {
		return nil, fileError.ErrBlobValidation.WithMessagef(
			"[s3_compatible] missing access_key_id or secret_access_key",
		)
	}

	awsCfg, err := config.LoadDefaultConfig(
		ctx,
		config.WithRegion(region),
		config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(
				cfg.AccessKeyID,
				cfg.SecretAccessKey,
				cfg.SessionToken,
			),
		),
		config.WithEndpointResolverWithOptions(
			aws.EndpointResolverWithOptionsFunc(
				func(service, r string, _ ...any) (aws.Endpoint, error) {
					if service == s3.ServiceID {
						return aws.Endpoint{
							URL:               endpoint,
							SigningRegion:     region,
							HostnameImmutable: true,
						}, nil
					}
					return aws.Endpoint{}, &aws.EndpointNotFoundError{}
				},
			),
		),
	)
	if err != nil {
		return nil, fileError.ErrBlobInternal.WithMessagef(
			"[s3_compatible] failed to load aws config: %v",
			err,
		)
	}

	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.UsePathStyle = cfg.ForcePathStyle
	})

	return &s3CompatibleAdapter{
		client:    client,
		presigner: s3.NewPresignClient(client),
	}, nil
}

// ParseS3Config parses and validates a raw config map into a typed S3Config.
// Returns ErrBlobValidation when required fields are missing.
func (a *s3CompatibleAdapter) ParseAndValidateConfig(
	raw map[string]any,
) (BlobConfig, error) {
	return ParseAndValidateConfig[S3Config](raw)
}

// PingStorage checks bucket access via HeadBucket (S3-compatible API).
func (a *s3CompatibleAdapter) PingStorage(ctx context.Context, bucketOrContainer string) error {
	name := strings.TrimSpace(bucketOrContainer)
	if name == "" {
		return fileError.ErrBlobValidation.WithMessagef(
			"[s3_compatible] ping_storage: bucket is required",
		)
	}
	if err := ctx.Err(); err != nil {
		return a.mapErr("ping_storage", err, "context cancelled")
	}
	_, err := a.client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(name),
	})
	if err != nil {
		return a.mapErr("ping_storage", err, "could not access bucket")
	}
	return nil
}

// PutObject uploads an object to the specified bucket and key.
func (a *s3CompatibleAdapter) PutObject(
	ctx context.Context,
	in model.BlobPutObjectInput,
) (model.BlobPutObjectOutput, error) {
	if err := validatePutInput(in); err != nil {
		return model.BlobPutObjectOutput{}, err
	}

	out, err := a.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(in.Bucket),
		Key:           aws.String(in.Key),
		Body:          in.Body,
		ContentType:   aws.String(in.ContentType),
		ContentLength: aws.Int64(in.ContentLength),
	})
	if err != nil {
		return model.BlobPutObjectOutput{}, a.mapErr("put_object", err, "put object failed")
	}

	var etag string
	if out.ETag != nil {
		etag = strings.Trim(*out.ETag, "\"")
	}
	var versionID *string
	if out.VersionId != nil {
		v := *out.VersionId
		versionID = &v
	}
	return model.BlobPutObjectOutput{
		Key:       in.Key,
		ETag:      etag,
		VersionID: versionID,
	}, nil
}

// DeleteObject removes the object identified by bucket + key.
func (a *s3CompatibleAdapter) DeleteObject(ctx context.Context, bucket, key string) error {
	if err := validateBucketKey(bucket, key); err != nil {
		return err
	}
	_, err := a.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return a.mapErr("delete_object", err, "delete object failed")
	}
	return nil
}

// HeadObject fetches metadata for the object at bucket/key without downloading the body.
func (a *s3CompatibleAdapter) HeadObject(
	ctx context.Context,
	bucket, key string,
) (model.BlobObjectMeta, error) {
	if err := validateBucketKey(bucket, key); err != nil {
		return model.BlobObjectMeta{}, err
	}
	out, err := a.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return model.BlobObjectMeta{}, a.mapErr("head_object", err, "head object failed")
	}

	meta := model.BlobObjectMeta{
		ContentType: aws.ToString(out.ContentType),
		SizeBytes:   aws.ToInt64(out.ContentLength),
		ETag:        strings.Trim(aws.ToString(out.ETag), "\""),
	}
	if out.LastModified != nil {
		meta.LastModified = *out.LastModified
	}
	if out.VersionId != nil {
		v := *out.VersionId
		meta.VersionID = &v
	}
	return meta, nil
}

// GetObjectStream opens a streaming download for bucket/key.
// The caller must close the returned io.ReadCloser.
func (a *s3CompatibleAdapter) GetObjectStream(
	ctx context.Context,
	bucket, key string,
) (io.ReadCloser, model.BlobObjectMeta, error) {
	if err := validateBucketKey(bucket, key); err != nil {
		return nil, model.BlobObjectMeta{}, err
	}
	out, err := a.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, model.BlobObjectMeta{}, a.mapErr("get_object_stream", err, "get object failed")
	}

	meta := model.BlobObjectMeta{
		ContentType: aws.ToString(out.ContentType),
		SizeBytes:   aws.ToInt64(out.ContentLength),
		ETag:        strings.Trim(aws.ToString(out.ETag), "\""),
	}
	if out.LastModified != nil {
		meta.LastModified = *out.LastModified
	}
	if out.VersionId != nil {
		v := *out.VersionId
		meta.VersionID = &v
	}
	return out.Body, meta, nil
}

// PresignUpload generates a time-limited URL that allows a client to PUT an object directly.
func (a *s3CompatibleAdapter) PresignUpload(
	ctx context.Context,
	in model.BlobPresignUploadInput,
) (model.BlobPresignOutput, error) {
	if err := validatePresignUploadInput(in); err != nil {
		return model.BlobPresignOutput{}, err
	}

	if _, err := a.client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(in.Bucket),
	}); err != nil {
		return model.BlobPresignOutput{}, a.mapErr(
			"presign_upload_head_bucket",
			err,
			"bucket check failed",
		)
	}

	p, err := a.presigner.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(in.Bucket),
		Key:         aws.String(in.Key),
		ContentType: aws.String(in.ContentType),
	}, func(po *s3.PresignOptions) {
		po.Expires = in.TTL
	})
	if err != nil {
		return model.BlobPresignOutput{}, a.mapErr("presign_upload", err, "presign upload failed")
	}

	return model.BlobPresignOutput{
		URL:       p.URL,
		ExpiresAt: time.Now().Add(in.TTL),
	}, nil
}

// PresignDownload generates a time-limited URL that allows a client to GET an object directly.
func (a *s3CompatibleAdapter) PresignDownload(
	ctx context.Context,
	in model.BlobPresignDownloadInput,
) (model.BlobPresignOutput, error) {
	if err := validatePresignDownloadInput(in); err != nil {
		return model.BlobPresignOutput{}, err
	}

	req := &s3.GetObjectInput{
		Bucket: aws.String(in.Bucket),
		Key:    aws.String(in.Key),
	}
	if disposition := strings.TrimSpace(in.Disposition); disposition != "" {
		req.ResponseContentDisposition = aws.String(disposition)
	}

	p, err := a.presigner.PresignGetObject(ctx, req, func(po *s3.PresignOptions) {
		po.Expires = in.TTL
	})
	if err != nil {
		return model.BlobPresignOutput{}, a.mapErr(
			"presign_download",
			err,
			"presign download failed",
		)
	}

	return model.BlobPresignOutput{
		URL:       p.URL,
		ExpiresAt: time.Now().Add(in.TTL),
	}, nil
}

// CopyObject copies an object from source bucket/key to destination bucket/key within the same provider.
func (a *s3CompatibleAdapter) CopyObject(ctx context.Context, in model.BlobCopyObjectInput) error {
	if strings.TrimSpace(in.SourceBucket) == "" ||
		strings.TrimSpace(in.SourceKey) == "" ||
		strings.TrimSpace(in.DestinationBucket) == "" ||
		strings.TrimSpace(in.DestinationKey) == "" {
		return fileError.ErrBlobValidation.WithMessagef(
			"[s3_compatible] missing source or destination bucket/key",
		)
	}

	copySource := url.PathEscape(in.SourceBucket + "/" + in.SourceKey)
	_, err := a.client.CopyObject(ctx, &s3.CopyObjectInput{
		Bucket:     aws.String(in.DestinationBucket),
		Key:        aws.String(in.DestinationKey),
		CopySource: aws.String(copySource),
	})
	if err != nil {
		return a.mapErr("copy_object", err, "copy object failed")
	}
	return nil
}

// ─── Validation helpers ───────────────────────────────────────────────────────

func validateBucketKey(bucket, key string) error {
	if strings.TrimSpace(bucket) == "" || strings.TrimSpace(key) == "" {
		return fileError.ErrBlobValidation.WithMessagef("[s3_compatible] missing bucket or key")
	}
	return nil
}

func validatePutInput(in model.BlobPutObjectInput) error {
	if strings.TrimSpace(in.Bucket) == "" ||
		strings.TrimSpace(in.Key) == "" ||
		strings.TrimSpace(in.ContentType) == "" ||
		in.Body == nil ||
		in.ContentLength < 0 {
		return fileError.ErrBlobValidation.WithMessagef(
			"[s3_compatible] invalid put input: missing bucket/key/content-type or negative content-length",
		)
	}
	return nil
}

func validatePresignUploadInput(in model.BlobPresignUploadInput) error {
	if strings.TrimSpace(in.Bucket) == "" || strings.TrimSpace(in.Key) == "" ||
		strings.TrimSpace(in.ContentType) == "" {
		return fileError.ErrBlobValidation.WithMessagef(
			"[s3_compatible] presign upload: missing bucket/key/content-type",
		)
	}
	if in.TTL <= 0 {
		return fileError.ErrBlobValidation.WithMessagef(
			"[s3_compatible] presign upload: ttl must be > 0",
		)
	}
	if in.ContentLengthLimit < 0 {
		return fileError.ErrBlobValidation.WithMessagef(
			"[s3_compatible] presign upload: content-length limit must be >= 0",
		)
	}
	return nil
}

func validatePresignDownloadInput(in model.BlobPresignDownloadInput) error {
	if strings.TrimSpace(in.Bucket) == "" || strings.TrimSpace(in.Key) == "" {
		return fileError.ErrBlobValidation.WithMessagef(
			"[s3_compatible] presign download: missing bucket/key",
		)
	}
	if in.TTL <= 0 {
		return fileError.ErrBlobValidation.WithMessagef(
			"[s3_compatible] presign download: ttl must be > 0",
		)
	}
	return nil
}

// ─── Error mapping ────────────────────────────────────────────────────────────

// s3ProviderErrorDetail extracts the provider-facing message from AWS SDK errors
// (typed API errors, operation wrappers, HTTP response errors). Safe to return to API clients.
func s3ProviderErrorDetail(err error) string {
	if err == nil {
		return ""
	}
	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		if m := strings.TrimSpace(apiErr.ErrorMessage()); m != "" {
			return m
		}
		if c := strings.TrimSpace(apiErr.ErrorCode()); c != "" {
			return c
		}
	}
	var opErr *smithy.OperationError
	if errors.As(err, &opErr) && opErr.Unwrap() != nil {
		if d := s3ProviderErrorDetail(opErr.Unwrap()); d != "" {
			return d
		}
	}
	var re *awshttp.ResponseError
	if errors.As(err, &re) {
		if re.Unwrap() != nil {
			if d := s3ProviderErrorDetail(re.Unwrap()); d != "" {
				return d
			}
		}
		if s := strings.TrimSpace(re.Error()); s != "" {
			return s
		}
	}
	return ""
}

// mapErr translates provider SDK errors into categorised *AppError values.
// op is an operation label for contextual messages; provider details are included for debugging.
func (a *s3CompatibleAdapter) mapErr(op string, err error, msg string) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return fileError.ErrBlobNetwork.WithMessagef(
			"[s3_compatible/%s] %s: context cancelled or deadline exceeded",
			op,
			msg,
		)
	}

	detail := s3ProviderErrorDetail(err)

	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		code := apiErr.ErrorCode()
		switch {
		case isS3NotFoundCode(code):
			if detail == "" {
				detail = code
			}
			return fileError.ErrBlobNotFound.WithMessagef(
				"[s3_compatible/%s] %s: %s",
				op,
				msg,
				detail,
			)
		case isS3PermissionCode(code):
			if detail == "" {
				detail = "access denied"
			}
			return fileError.ErrBlobPermissionDenied.WithMessagef(
				"[s3_compatible/%s] %s: %s",
				op,
				msg,
				detail,
			)
		}
	}

	var respErr *awshttp.ResponseError
	if errors.As(err, &respErr) {
		switch respErr.HTTPStatusCode() {
		case 404:
			if detail == "" {
				detail = "not found"
			}
			return fileError.ErrBlobNotFound.WithMessagef(
				"[s3_compatible/%s] %s: %s",
				op,
				msg,
				detail,
			)
		case 401, 403:
			if detail == "" {
				detail = "access denied"
			}
			return fileError.ErrBlobPermissionDenied.WithMessagef(
				"[s3_compatible/%s] %s: %s",
				op,
				msg,
				detail,
			)
		default:
			if detail == "" {
				detail = fmt.Sprintf("HTTP %d", respErr.HTTPStatusCode())
			}
			return fileError.ErrBlobInternal.WithMessagef(
				"[s3_compatible/%s] %s: %s",
				op,
				msg,
				detail,
			)
		}
	}

	if apiErr != nil {
		if detail == "" {
			detail = apiErr.ErrorCode()
		}
		return fileError.ErrBlobInternal.WithMessagef(
			"[s3_compatible/%s] %s: %s",
			op,
			msg,
			detail,
		)
	}

	var nsk *s3types.NoSuchKey
	if errors.As(err, &nsk) {
		if detail == "" {
			detail = "no such key"
		}
		return fileError.ErrBlobNotFound.WithMessagef(
			"[s3_compatible/%s] %s: %s",
			op,
			msg,
			detail,
		)
	}

	var opErr *smithy.OperationError
	if errors.As(err, &opErr) && opErr.Unwrap() != nil {
		return a.mapErr(op, opErr.Unwrap(), msg)
	}

	var netErr net.Error
	if errors.As(err, &netErr) {
		return fileError.ErrBlobNetwork.WithMessagef(
			"[s3_compatible/%s] %s: network error",
			op,
			msg,
		)
	}

	if detail == "" {
		detail = "unexpected error"
	}
	return fileError.ErrBlobInternal.WithMessagef(
		"[s3_compatible/%s] %s: %s",
		op,
		msg,
		detail,
	)
}

func isS3NotFoundCode(code string) bool {
	switch code {
	case "NoSuchKey", "NotFound", "NoSuchBucket":
		return true
	default:
		return false
	}
}

func isS3PermissionCode(code string) bool {
	switch code {
	case "AccessDenied", "InvalidAccessKeyId", "SignatureDoesNotMatch":
		return true
	default:
		return false
	}
}

// S3Schema returns the field descriptor schema for the s3_compatible adapter.
func S3Schema() model.AdapterConfigSchema {
	return model.AdapterConfigSchema{
		AdapterType: entity.AdapterTypeS3Compatible,
		Fields: []model.FieldDescriptor{
			{
				Key:         "access_key_id",
				Label:       "Access Key ID",
				Type:        model.FieldTypeString,
				Required:    true,
				Sensitive:   true,
				Description: "AWS Access Key ID or equivalent for S3-compatible providers.",
				Placeholder: "AKIAIOSFODNN7EXAMPLE",
			},
			{
				Key:         "secret_access_key",
				Label:       "Secret Access Key",
				Type:        model.FieldTypePassword,
				Required:    true,
				Sensitive:   true,
				Description: "AWS Secret Access Key or equivalent.",
			},
			{
				Key:         "session_token",
				Label:       "Session Token",
				Type:        model.FieldTypePassword,
				Required:    false,
				Sensitive:   true,
				Description: "Temporary STS session token. Only required when using short-lived assumed-role credentials.",
			},
			{
				Key:         "bucket",
				Label:       "Bucket Name",
				Type:        model.FieldTypeString,
				Required:    true,
				Sensitive:   false,
				Description: "The S3 bucket where files will be stored.",
				Placeholder: "my-ecommerce-bucket",
			},
			{
				Key:         "region",
				Label:       "Region",
				Type:        model.FieldTypeString,
				Required:    false,
				Sensitive:   false,
				Description: "AWS region (e.g. us-east-1). Defaults to us-east-1 if not set.",
				Placeholder: "us-east-1",
			},
			{
				Key:         "endpoint",
				Label:       "Custom Endpoint",
				Type:        model.FieldTypeString,
				Required:    false,
				Sensitive:   false,
				Description: "Required for non-AWS stores (e.g. http://minio:9000 or https://s3.us-west-1.example.com).",
				Placeholder: "https://s3.us-east-1.amazonaws.com",
			},
			{
				Key:         "force_path_style",
				Label:       "Force Path Style",
				Type:        model.FieldTypeBoolean,
				Required:    false,
				Sensitive:   false,
				Description: "Use path-style URLs (/bucket/key) instead of virtual-hosted style. Required for MinIO and most non-AWS stores.",
			},
		},
	}
}
