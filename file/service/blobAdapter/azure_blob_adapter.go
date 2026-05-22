package blobAdapter

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/bloberror"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/sas"

	"ecommerce-be/common/helper"
	"ecommerce-be/file/entity"
	fileError "ecommerce-be/file/error"
	"ecommerce-be/file/model"
)

// ─── AzureConfig — typed config struct ────────────────────────────────────────

// AzureConfig is the typed configuration for an Azure Blob Storage adapter.
type AzureConfig struct {
	// AccountName is the Azure Storage account name. Required and sensitive.
	AccountName string `json:"account_name" validate:"required"`
	// AccountKey is the primary/secondary access key. Either AccountKey or SAS is required.
	AccountKey string `json:"account_key"  validate:"required_without=SAS"`
	// SAS is a Shared Access Signature token. Either AccountKey or SAS is required.
	SAS string `json:"sas"          validate:"required_without=AccountKey"`
	// Container is the Azure Blob container name. Required.
	Container string `json:"container"    validate:"required"`
	// Endpoint is an optional custom base URL (e.g. http://localhost:10000/devstoreaccount1 for Azurite).
	Endpoint string `json:"endpoint"`
}

func (s *AzureConfig) Encrypt() error {
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
	if s.AccountName, err = encryptField(s.AccountName); err != nil {
		return fileError.ErrEncryptionFailed.WithMessagef("[azure] encrypt account_name: %v", err)
	}
	if s.AccountKey, err = encryptField(s.AccountKey); err != nil {
		return fileError.ErrEncryptionFailed.WithMessagef("[azure] encrypt account_key: %v", err)
	}
	if s.SAS, err = encryptField(s.SAS); err != nil {
		return fileError.ErrEncryptionFailed.WithMessagef("[azure] encrypt sas: %v", err)
	}
	return nil
}

func (s *AzureConfig) ToMap() map[string]any {
	return map[string]any{
		"account_name": s.AccountName,
		"account_key":  s.AccountKey,
		"sas":          s.SAS,
		"container":    s.Container,
		"endpoint":     s.Endpoint,
	}
}

func (s *AzureConfig) Decrypt() error {
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

	s.AccountName, _ = decryptField(s.AccountName)
	s.AccountKey, _ = decryptField(s.AccountKey)
	s.SAS, _ = decryptField(s.SAS)
	return nil
}

// AzureSchema returns the field descriptor schema for the Azure adapter.
func AzureSchema() model.AdapterConfigSchema {
	return model.AdapterConfigSchema{
		AdapterType: entity.AdapterTypeAzure,
		Fields: []model.FieldDescriptor{
			{
				Key:         "account_name",
				Label:       "Storage Account Name",
				Type:        model.FieldTypeString,
				Required:    true,
				Sensitive:   true,
				Description: "The name of your Azure Storage account.",
				Placeholder: "mystorageaccount",
			},
			{
				Key:         "account_key",
				Label:       "Account Key",
				Type:        model.FieldTypePassword,
				Required:    false,
				Sensitive:   true,
				Description: "Primary or secondary access key for the storage account. Required if SAS token is not provided.",
			},
			{
				Key:         "sas",
				Label:       "SAS Token",
				Type:        model.FieldTypePassword,
				Required:    false,
				Sensitive:   true,
				Description: "Shared Access Signature token. Required if Account Key is not provided.",
			},
			{
				Key:         "container",
				Label:       "Container Name",
				Type:        model.FieldTypeString,
				Required:    true,
				Sensitive:   false,
				Description: "The Azure Blob container where files will be stored.",
				Placeholder: "my-ecommerce-container",
			},
			{
				Key:         "endpoint",
				Label:       "Custom Endpoint",
				Type:        model.FieldTypeString,
				Required:    false,
				Sensitive:   false,
				Description: "Override the Azure Blob endpoint (e.g. http://localhost:10000/devstoreaccount1 for Azurite). Leave empty for production.",
			},
		},
	}
}

// azureBlobAdapter implements BlobAdapter against Azure Blob Storage.
type azureBlobAdapter struct {
	client        *azblob.Client
	sharedKeyCred *azblob.SharedKeyCredential
	serviceURL    string // base URL, no trailing slash; used when building SAS/copy URLs
}

// Compile-time assertion: azureBlobAdapter satisfies BlobAdapter.
var _ BlobAdapter = (*azureBlobAdapter)(nil)

// NewAzureAdapterFromMap constructs an Azure BlobAdapter from a raw config map.
func NewAzureAdapterFromMap(ctx context.Context, raw map[string]any) (BlobAdapter, error) {
	cfg, err := ParseAndValidateConfig[AzureConfig](raw)
	if err != nil {
		return nil, err
	}
	return NewAzureBlobAdapter(cfg)
}

// NewAzureBlobAdapter constructs an Azure BlobAdapter from the supplied config.
// Returns ErrBlobValidation when credentials are missing or structurally invalid.
// Returns ErrBlobFactoryInit when the underlying SDK client cannot be created.
func NewAzureBlobAdapter(cfg *AzureConfig) (BlobAdapter, error) {
	if strings.TrimSpace(cfg.AccountName) == "" {
		return nil, fileError.ErrBlobValidation.WithMessagef("[azure] account_name is required")
	}
	if strings.TrimSpace(cfg.AccountKey) == "" {
		return nil, fileError.ErrBlobValidation.WithMessagef("[azure] account_key is required")
	}

	cred, err := azblob.NewSharedKeyCredential(cfg.AccountName, cfg.AccountKey)
	if err != nil {
		return nil, fileError.ErrBlobValidation.WithMessagef(
			"[azure] invalid shared-key credential: %v", err,
		)
	}

	var (
		client     *azblob.Client
		serviceURL string
	)

	if strings.TrimSpace(cfg.Endpoint) != "" {
		serviceURL = strings.TrimRight(cfg.Endpoint, "/")
		connStr := fmt.Sprintf(
			"DefaultEndpointsProtocol=http;AccountName=%s;AccountKey=%s;BlobEndpoint=%s;",
			cfg.AccountName, cfg.AccountKey, serviceURL,
		)
		client, err = azblob.NewClientFromConnectionString(connStr, nil)
	} else {
		serviceURL = fmt.Sprintf("https://%s.blob.core.windows.net", cfg.AccountName)
		client, err = azblob.NewClientWithSharedKeyCredential(serviceURL+"/", cred, nil)
	}

	if err != nil {
		return nil, fileError.ErrBlobFactoryInit.WithMessagef(
			"[azure] failed to create client: %v", err,
		)
	}

	return &azureBlobAdapter{
		client:        client,
		sharedKeyCred: cred,
		serviceURL:    serviceURL,
	}, nil
}

// ─── BlobAdapter interface implementation ─────────────────────────────────────

// ParseAzureConfig parses and validates a raw config map into a typed AzureConfig.
// Returns ErrBlobValidation when required fields are missing.
func (a *azureBlobAdapter) ParseAndValidateConfig(
	raw map[string]any,
) (BlobConfig, error) {
	return ParseAndValidateConfig[AzureConfig](raw)
}

// PingStorage checks container access via GetProperties on the container client.
func (a *azureBlobAdapter) PingStorage(ctx context.Context, bucketOrContainer string) error {
	name := strings.TrimSpace(bucketOrContainer)
	if name == "" {
		return fileError.ErrBlobValidation.WithMessagef("[azure] ping_storage: container is required")
	}
	if err := ctx.Err(); err != nil {
		return a.mapErr("ping_storage", err)
	}
	_, err := a.client.ServiceClient().
		NewContainerClient(name).
		GetProperties(ctx, nil)
	if err != nil {
		return a.mapErr("ping_storage", err)
	}
	return nil
}

// PutObject uploads a blob to Azure Blob Storage.
func (a *azureBlobAdapter) PutObject(
	ctx context.Context,
	in model.BlobPutObjectInput,
) (model.BlobPutObjectOutput, error) {
	if err := ctx.Err(); err != nil {
		return model.BlobPutObjectOutput{}, a.mapErr("put_object", err)
	}
	ct := in.ContentType
	opts := &azblob.UploadStreamOptions{
		HTTPHeaders: &blob.HTTPHeaders{BlobContentType: &ct},
	}
	resp, err := a.client.UploadStream(ctx, in.Bucket, in.Key, in.Body, opts)
	if err != nil {
		return model.BlobPutObjectOutput{}, a.mapErr("put_object", err)
	}
	var etag string
	if resp.ETag != nil {
		etag = string(*resp.ETag)
	}
	return model.BlobPutObjectOutput{Key: in.Key, ETag: etag}, nil
}

// DeleteObject removes a blob from Azure Blob Storage.
func (a *azureBlobAdapter) DeleteObject(ctx context.Context, bucket, key string) error {
	if err := ctx.Err(); err != nil {
		return a.mapErr("delete_object", err)
	}
	_, err := a.client.DeleteBlob(ctx, bucket, key, nil)
	return a.mapErr("delete_object", err)
}

// HeadObject returns metadata for a blob without downloading its body.
func (a *azureBlobAdapter) HeadObject(
	ctx context.Context,
	bucket, key string,
) (model.BlobObjectMeta, error) {
	if err := ctx.Err(); err != nil {
		return model.BlobObjectMeta{}, a.mapErr("head_object", err)
	}
	resp, err := a.client.ServiceClient().
		NewContainerClient(bucket).
		NewBlobClient(key).
		GetProperties(ctx, nil)
	if err != nil {
		return model.BlobObjectMeta{}, a.mapErr("head_object", err)
	}

	meta := model.BlobObjectMeta{}
	if resp.ContentLength != nil {
		meta.SizeBytes = *resp.ContentLength
	}
	if resp.ContentType != nil {
		meta.ContentType = *resp.ContentType
	}
	if resp.ETag != nil {
		meta.ETag = string(*resp.ETag)
	}
	if resp.LastModified != nil {
		meta.LastModified = *resp.LastModified
	}
	return meta, nil
}

// GetObjectStream returns a streaming reader plus metadata for a blob.
// Callers must close the returned io.ReadCloser.
func (a *azureBlobAdapter) GetObjectStream(
	ctx context.Context,
	bucket, key string,
) (io.ReadCloser, model.BlobObjectMeta, error) {
	if err := ctx.Err(); err != nil {
		return nil, model.BlobObjectMeta{}, a.mapErr("get_object_stream", err)
	}
	resp, err := a.client.DownloadStream(ctx, bucket, key, nil)
	if err != nil {
		return nil, model.BlobObjectMeta{}, a.mapErr("get_object_stream", err)
	}

	meta := model.BlobObjectMeta{}
	if resp.ContentLength != nil {
		meta.SizeBytes = *resp.ContentLength
	}
	if resp.ContentType != nil {
		meta.ContentType = *resp.ContentType
	}
	if resp.ETag != nil {
		meta.ETag = string(*resp.ETag)
	}
	if resp.LastModified != nil {
		meta.LastModified = *resp.LastModified
	}
	return resp.Body, meta, nil
}

// PresignUpload generates a short-lived SAS URL for direct client upload.
// TTL must be positive. Requires the adapter to be initialised with an account key.
func (a *azureBlobAdapter) PresignUpload(
	ctx context.Context,
	in model.BlobPresignUploadInput,
) (model.BlobPresignOutput, error) {
	if err := ctx.Err(); err != nil {
		return model.BlobPresignOutput{}, a.mapErr("presign_upload", err)
	}
	if in.TTL <= 0 {
		return model.BlobPresignOutput{}, fileError.ErrBlobValidation.WithMessagef(
			"[azure] presign_upload: TTL must be positive",
		)
	}
	uploadPerms := sas.BlobPermissions{Write: true, Create: true}
	url, expiresAt, err := a.buildSASURL(
		in.Bucket,
		in.Key,
		in.TTL,
		uploadPerms.String(),
		"",
	)
	if err != nil {
		return model.BlobPresignOutput{}, err
	}
	return model.BlobPresignOutput{URL: url, ExpiresAt: expiresAt}, nil
}

// PresignDownload generates a short-lived SAS URL for direct client download.
// TTL must be positive. Requires the adapter to be initialised with an account key.
func (a *azureBlobAdapter) PresignDownload(
	ctx context.Context,
	in model.BlobPresignDownloadInput,
) (model.BlobPresignOutput, error) {
	if err := ctx.Err(); err != nil {
		return model.BlobPresignOutput{}, a.mapErr("presign_download", err)
	}
	if in.TTL <= 0 {
		return model.BlobPresignOutput{}, fileError.ErrBlobValidation.WithMessagef(
			"[azure] presign_download: TTL must be positive",
		)
	}
	downloadPerms := sas.BlobPermissions{Read: true}
	url, expiresAt, err := a.buildSASURL(
		in.Bucket,
		in.Key,
		in.TTL,
		downloadPerms.String(),
		strings.TrimSpace(in.Disposition),
	)
	if err != nil {
		return model.BlobPresignOutput{}, err
	}
	return model.BlobPresignOutput{URL: url, ExpiresAt: expiresAt}, nil
}

// CopyObject copies a blob from source to destination via streaming download + re-upload.
// Both buckets must exist prior to the call.
func (a *azureBlobAdapter) CopyObject(ctx context.Context, in model.BlobCopyObjectInput) error {
	if err := ctx.Err(); err != nil {
		return a.mapErr("copy_object", err)
	}
	dr, err := a.client.DownloadStream(ctx, in.SourceBucket, in.SourceKey, nil)
	if err != nil {
		return a.mapErr("copy_object", err)
	}
	defer dr.Body.Close()

	ct := ""
	if dr.ContentType != nil {
		ct = *dr.ContentType
	}
	_, err = a.client.UploadStream(
		ctx,
		in.DestinationBucket,
		in.DestinationKey,
		dr.Body,
		&azblob.UploadStreamOptions{
			HTTPHeaders: &blob.HTTPHeaders{BlobContentType: &ct},
		},
	)
	return a.mapErr("copy_object", err)
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

// buildSASURL generates a shared-access-signature URL for the given blob.
func (a *azureBlobAdapter) buildSASURL(
	container, key string,
	ttl time.Duration,
	permissions string,
	contentDisposition string,
) (string, time.Time, error) {
	expiresAt := time.Now().UTC().Add(ttl)
	sasParams, err := sas.BlobSignatureValues{
		Protocol:      sas.ProtocolHTTPSandHTTP,
		ExpiryTime:    expiresAt,
		ContainerName: container,
		BlobName:      key,
		Permissions:   permissions,
		ContentDisposition: contentDisposition,
	}.SignWithSharedKey(a.sharedKeyCred)
	if err != nil {
		return "", time.Time{}, fileError.ErrBlobInternal.WithMessagef(
			"[azure] failed to sign SAS: %v", err,
		)
	}
	blobURL := fmt.Sprintf("%s/%s/%s?%s", a.serviceURL, container, key, sasParams.Encode())
	return blobURL, expiresAt, nil
}

// mapErr translates Azure SDK errors into categorised fileError sentinels.
// Full *azcore.ResponseError text (status, code, response body) is included for debugging.
func (a *azureBlobAdapter) mapErr(op string, err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return fileError.ErrBlobNetwork.WithMessagef(
			"[azure] %s: context cancelled or deadline exceeded", op,
		)
	}

	var respErr *azcore.ResponseError
	if errors.As(err, &respErr) {
		detail := strings.TrimSpace(respErr.Error())
		switch respErr.StatusCode {
		case http.StatusNotFound:
			if detail != "" {
				return fileError.ErrBlobNotFound.WithMessagef("[azure] %s: %s", op, detail)
			}
			return fileError.ErrBlobNotFound.WithMessagef(
				"[azure] %s: not found (HTTP %d)", op, respErr.StatusCode,
			)
		case http.StatusForbidden, http.StatusUnauthorized:
			if detail != "" {
				return fileError.ErrBlobPermissionDenied.WithMessagef("[azure] %s: %s", op, detail)
			}
			return fileError.ErrBlobPermissionDenied.WithMessagef(
				"[azure] %s: permission denied (HTTP %d)", op, respErr.StatusCode,
			)
		default:
			if detail != "" {
				return fileError.ErrBlobInternal.WithMessagef("[azure] %s: %s", op, detail)
			}
			return fileError.ErrBlobInternal.WithMessagef(
				"[azure] %s: provider error (HTTP %d)", op, respErr.StatusCode,
			)
		}
	}

	if bloberror.HasCode(err, bloberror.BlobNotFound, bloberror.ContainerNotFound) {
		detail := strings.TrimSpace(err.Error())
		if detail != "" {
			return fileError.ErrBlobNotFound.WithMessagef("[azure] %s: %s", op, detail)
		}
		return fileError.ErrBlobNotFound.WithMessagef("[azure] %s: object not found", op)
	}

	detail := strings.TrimSpace(err.Error())
	if detail != "" {
		return fileError.ErrBlobInternal.WithMessagef("[azure] %s: %s", op, detail)
	}
	return fileError.ErrBlobInternal.WithMessagef("[azure] %s: unexpected error", op)
}
