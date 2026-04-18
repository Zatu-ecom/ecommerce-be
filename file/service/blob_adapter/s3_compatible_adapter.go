package blob_adapter

import (
	"context"
	"errors"
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

	fileError "ecommerce-be/file/error"
	"ecommerce-be/file/model"
)

// S3CompatibleOptions contains all parameters needed to initialise an S3-compatible adapter.
// Endpoint is mandatory (e.g. http://minio:9000). Region defaults to us-east-1 if empty.
type S3CompatibleOptions struct {
	Endpoint        string
	Region          string
	ForcePathStyle  bool
	AccessKeyID     string
	SecretAccessKey string
	SessionToken    string
}

type s3CompatibleAdapter struct {
	client    *s3.Client
	presigner *s3.PresignClient
}

// Compile-time assertion: s3CompatibleAdapter satisfies BlobAdapter.
var _ BlobAdapter = (*s3CompatibleAdapter)(nil)

// NewS3CompatibleAdapter constructs an S3-compatible BlobAdapter from the supplied options.
// Returns ErrBlobValidation if required fields are missing or the endpoint is unparseable.
func NewS3CompatibleAdapter(ctx context.Context, opts S3CompatibleOptions) (BlobAdapter, error) {
	endpoint := strings.TrimSpace(opts.Endpoint)
	if endpoint == "" {
		return nil, fileError.ErrBlobValidation.WithMessagef("[s3_compatible] missing endpoint")
	}
	if _, err := url.Parse(endpoint); err != nil {
		return nil, fileError.ErrBlobValidation.WithMessagef(
			"[s3_compatible] invalid endpoint url: %v",
			err,
		)
	}
	region := strings.TrimSpace(opts.Region)
	if region == "" {
		region = "us-east-1"
	}
	if strings.TrimSpace(opts.AccessKeyID) == "" || strings.TrimSpace(opts.SecretAccessKey) == "" {
		return nil, fileError.ErrBlobValidation.WithMessagef(
			"[s3_compatible] missing access_key_id or secret_access_key",
		)
	}

	awsCfg, err := config.LoadDefaultConfig(
		ctx,
		config.WithRegion(region),
		config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(
				opts.AccessKeyID,
				opts.SecretAccessKey,
				opts.SessionToken,
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
		o.UsePathStyle = opts.ForcePathStyle
	})

	return &s3CompatibleAdapter{
		client:    client,
		presigner: s3.NewPresignClient(client),
	}, nil
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

	p, err := a.presigner.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(in.Bucket),
		Key:    aws.String(in.Key),
	}, func(po *s3.PresignOptions) {
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

// mapErr translates provider SDK errors into categorised *AppError values.
// op is an operation label for contextual messages; no credentials are included.
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

	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		code := apiErr.ErrorCode()
		switch {
		case isS3NotFoundCode(code):
			return fileError.ErrBlobNotFound.WithMessagef(
				"[s3_compatible/%s] %s: %s",
				op,
				msg,
				code,
			)
		case isS3PermissionCode(code):
			return fileError.ErrBlobPermissionDenied.WithMessagef(
				"[s3_compatible/%s] %s: access denied",
				op,
				msg,
			)
		}
		// Fall through to HTTP-status inspection for APIError values with
		// an empty / unrecognised ErrorCode (e.g. HeadObject 403 with no body).
	}

	// HEAD requests return no response body, so the SDK cannot extract an
	// ErrorCode. Inspect the underlying HTTP status instead.
	var respErr *awshttp.ResponseError
	if errors.As(err, &respErr) {
		switch respErr.HTTPStatusCode() {
		case 404:
			return fileError.ErrBlobNotFound.WithMessagef(
				"[s3_compatible/%s] %s: not found",
				op,
				msg,
			)
		case 401, 403:
			return fileError.ErrBlobPermissionDenied.WithMessagef(
				"[s3_compatible/%s] %s: access denied",
				op,
				msg,
			)
		}
	}

	if apiErr != nil {
		return fileError.ErrBlobInternal.WithMessagef(
			"[s3_compatible/%s] %s: provider error %s",
			op,
			msg,
			apiErr.ErrorCode(),
		)
	}

	var nsk *s3types.NoSuchKey
	if errors.As(err, &nsk) {
		return fileError.ErrBlobNotFound.WithMessagef("[s3_compatible/%s] %s: no such key", op, msg)
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

	return fileError.ErrBlobInternal.WithMessagef(
		"[s3_compatible/%s] %s: unexpected error",
		op,
		msg,
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
