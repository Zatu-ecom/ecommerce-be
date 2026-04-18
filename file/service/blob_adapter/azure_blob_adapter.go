package blob_adapter

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

	fileError "ecommerce-be/file/error"
	"ecommerce-be/file/model"
)

// AzureOptions contains parameters needed to initialise an Azure Blob adapter.
// AccountName and AccountKey are required.
// Endpoint is optional; when set, all API calls are routed to that host (used
// with Azurite in integration tests). Example: "http://localhost:10000/devstoreaccount1".
type AzureOptions struct {
	AccountName string
	AccountKey  string
	Endpoint    string
}

// azureBlobAdapter implements BlobAdapter against Azure Blob Storage.
type azureBlobAdapter struct {
	client        *azblob.Client
	sharedKeyCred *azblob.SharedKeyCredential
	serviceURL    string // base URL, no trailing slash; used when building SAS/copy URLs
}

// Compile-time assertion: azureBlobAdapter satisfies BlobAdapter.
var _ BlobAdapter = (*azureBlobAdapter)(nil)

// NewAzureBlobAdapter constructs an Azure BlobAdapter from the supplied options.
// Returns ErrBlobValidation when credentials are missing or structurally invalid.
// Returns ErrBlobFactoryInit when the underlying SDK client cannot be created.
func NewAzureBlobAdapter(opts AzureOptions) (BlobAdapter, error) {
	if strings.TrimSpace(opts.AccountName) == "" {
		return nil, fileError.ErrBlobValidation.WithMessagef("[azure] account_name is required")
	}
	if strings.TrimSpace(opts.AccountKey) == "" {
		return nil, fileError.ErrBlobValidation.WithMessagef("[azure] account_key is required")
	}

	cred, err := azblob.NewSharedKeyCredential(opts.AccountName, opts.AccountKey)
	if err != nil {
		return nil, fileError.ErrBlobValidation.WithMessagef(
			"[azure] invalid shared-key credential: %v", err,
		)
	}

	var (
		client     *azblob.Client
		serviceURL string
	)

	if strings.TrimSpace(opts.Endpoint) != "" {
		serviceURL = strings.TrimRight(opts.Endpoint, "/")
		connStr := fmt.Sprintf(
			"DefaultEndpointsProtocol=http;AccountName=%s;AccountKey=%s;BlobEndpoint=%s;",
			opts.AccountName, opts.AccountKey, serviceURL,
		)
		client, err = azblob.NewClientFromConnectionString(connStr, nil)
	} else {
		serviceURL = fmt.Sprintf("https://%s.blob.core.windows.net", opts.AccountName)
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

// PutObject uploads a blob to Azure Blob Storage.
func (a *azureBlobAdapter) PutObject(ctx context.Context, in model.BlobPutObjectInput) (model.BlobPutObjectOutput, error) {
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
func (a *azureBlobAdapter) HeadObject(ctx context.Context, bucket, key string) (model.BlobObjectMeta, error) {
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
func (a *azureBlobAdapter) GetObjectStream(ctx context.Context, bucket, key string) (io.ReadCloser, model.BlobObjectMeta, error) {
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
func (a *azureBlobAdapter) PresignUpload(ctx context.Context, in model.BlobPresignUploadInput) (model.BlobPresignOutput, error) {
	if err := ctx.Err(); err != nil {
		return model.BlobPresignOutput{}, a.mapErr("presign_upload", err)
	}
	if in.TTL <= 0 {
		return model.BlobPresignOutput{}, fileError.ErrBlobValidation.WithMessagef(
			"[azure] presign_upload: TTL must be positive",
		)
	}
	uploadPerms := sas.BlobPermissions{Write: true, Create: true}
	url, expiresAt, err := a.buildSASURL(in.Bucket, in.Key, in.TTL, uploadPerms.String())
	if err != nil {
		return model.BlobPresignOutput{}, err
	}
	return model.BlobPresignOutput{URL: url, ExpiresAt: expiresAt}, nil
}

// PresignDownload generates a short-lived SAS URL for direct client download.
// TTL must be positive. Requires the adapter to be initialised with an account key.
func (a *azureBlobAdapter) PresignDownload(ctx context.Context, in model.BlobPresignDownloadInput) (model.BlobPresignOutput, error) {
	if err := ctx.Err(); err != nil {
		return model.BlobPresignOutput{}, a.mapErr("presign_download", err)
	}
	if in.TTL <= 0 {
		return model.BlobPresignOutput{}, fileError.ErrBlobValidation.WithMessagef(
			"[azure] presign_download: TTL must be positive",
		)
	}
	downloadPerms := sas.BlobPermissions{Read: true}
	url, expiresAt, err := a.buildSASURL(in.Bucket, in.Key, in.TTL, downloadPerms.String())
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
	_, err = a.client.UploadStream(ctx, in.DestinationBucket, in.DestinationKey, dr.Body, &azblob.UploadStreamOptions{
		HTTPHeaders: &blob.HTTPHeaders{BlobContentType: &ct},
	})
	return a.mapErr("copy_object", err)
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

// buildSASURL generates a shared-access-signature URL for the given blob.
func (a *azureBlobAdapter) buildSASURL(container, key string, ttl time.Duration, permissions string) (string, time.Time, error) {
	expiresAt := time.Now().UTC().Add(ttl)
	sasParams, err := sas.BlobSignatureValues{
		Protocol:      sas.ProtocolHTTPSandHTTP,
		ExpiryTime:    expiresAt,
		ContainerName: container,
		BlobName:      key,
		Permissions:   permissions,
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
// Context errors become ErrBlobNetwork; BlobNotFound/ContainerNotFound become
// ErrBlobNotFound; HTTP 401/403 become ErrBlobPermissionDenied; all other SDK
// errors become ErrBlobInternal. Never exposes raw credentials in the message.
func (a *azureBlobAdapter) mapErr(op string, err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return fileError.ErrBlobNetwork.WithMessagef(
			"[azure] %s: context cancelled or deadline exceeded", op,
		)
	}
	if bloberror.HasCode(err, bloberror.BlobNotFound, bloberror.ContainerNotFound) {
		return fileError.ErrBlobNotFound.WithMessagef("[azure] %s: object not found", op)
	}
	var respErr *azcore.ResponseError
	if errors.As(err, &respErr) {
		switch respErr.StatusCode {
		case http.StatusNotFound:
			return fileError.ErrBlobNotFound.WithMessagef(
				"[azure] %s: not found (HTTP %d)", op, respErr.StatusCode,
			)
		case http.StatusForbidden, http.StatusUnauthorized:
			return fileError.ErrBlobPermissionDenied.WithMessagef(
				"[azure] %s: permission denied (HTTP %d)", op, respErr.StatusCode,
			)
		}
		return fileError.ErrBlobInternal.WithMessagef(
			"[azure] %s: provider error (HTTP %d)", op, respErr.StatusCode,
		)
	}
	return fileError.ErrBlobInternal.WithMessagef("[azure] %s: unexpected error", op)
}
