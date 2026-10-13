package file_test

// Shared blob-backend containers for the file integration package.
//
// This package previously booted a fresh MinIO / Azurite / fake-gcs-server
// container per suite. Now one container per backend is shared across all
// suites in this `go test` process; each Setup* call ensures its bucket and
// best-effort purges objects so same-bucket reuse starts empty. Containers
// stay up until Ryuk reaps them at process exit, so Cleanup is a no-op kept
// for existing call sites.

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

var (
	sharedFileMinioMu sync.Mutex
	sharedFileMinio   *MinioContainer

	sharedAzuriteMu sync.Mutex
	sharedAzurite   *AzuriteContainer

	sharedFakeGCSMu sync.Mutex
	sharedFakeGCS   *FakeGCSContainer
)

func acquireSharedFileMinio(t *testing.T, bucket string) *MinioContainer {
	t.Helper()
	sharedFileMinioMu.Lock()
	defer sharedFileMinioMu.Unlock()

	if sharedFileMinio != nil {
		if err := pingFileMinio(sharedFileMinio); err == nil {
			EnsureS3Bucket(t,
				sharedFileMinio.Endpoint, sharedFileMinio.Region,
				sharedFileMinio.AccessKey, sharedFileMinio.SecretKey, bucket)
			purgeFileMinioBucket(sharedFileMinio, bucket)
			return &MinioContainer{
				Container:  sharedFileMinio.Container,
				Endpoint:   sharedFileMinio.Endpoint,
				AccessKey:  sharedFileMinio.AccessKey,
				SecretKey:  sharedFileMinio.SecretKey,
				BucketName: bucket,
				Region:     sharedFileMinio.Region,
			}
		}
		t.Logf("shared file minio died; starting a replacement")
		_ = sharedFileMinio.Container.Terminate(context.Background())
		sharedFileMinio = nil
	}
	fresh := bootFileMinio(t, bucket)
	sharedFileMinio = &MinioContainer{
		Container: fresh.Container,
		Endpoint:  fresh.Endpoint,
		AccessKey: fresh.AccessKey,
		SecretKey: fresh.SecretKey,
		Region:    fresh.Region,
	}
	return fresh
}

func acquireSharedAzurite(t *testing.T, containerName string) *AzuriteContainer {
	t.Helper()
	sharedAzuriteMu.Lock()
	defer sharedAzuriteMu.Unlock()

	if sharedAzurite != nil {
		if err := pingAzurite(sharedAzurite.ConnectionString); err == nil {
			EnsureAzuriteContainer(t, sharedAzurite.ConnectionString, containerName)
			purgeAzuriteContainer(sharedAzurite.ConnectionString, containerName)
			return &AzuriteContainer{
				Container:        sharedAzurite.Container,
				BlobEndpoint:     sharedAzurite.BlobEndpoint,
				ConnectionString: sharedAzurite.ConnectionString,
				AccountName:      sharedAzurite.AccountName,
				AccountKey:       sharedAzurite.AccountKey,
				ContainerName:    containerName,
			}
		}
		t.Logf("shared azurite died; starting a replacement")
		_ = sharedAzurite.Container.Terminate(context.Background())
		sharedAzurite = nil
	}
	fresh := bootAzurite(t, containerName)
	sharedAzurite = &AzuriteContainer{
		Container:        fresh.Container,
		BlobEndpoint:     fresh.BlobEndpoint,
		ConnectionString: fresh.ConnectionString,
		AccountName:      fresh.AccountName,
		AccountKey:       fresh.AccountKey,
	}
	return fresh
}

func acquireSharedFakeGCS(t *testing.T, projectID, bucket string) *FakeGCSContainer {
	t.Helper()
	sharedFakeGCSMu.Lock()
	defer sharedFakeGCSMu.Unlock()

	if sharedFakeGCS != nil {
		if err := pingFakeGCS(sharedFakeGCS.Endpoint); err == nil {
			updateFakeGCSExternalURL(t, sharedFakeGCS.Endpoint)
			EnsureFakeGCSBucket(t, sharedFakeGCS.Endpoint, projectID, bucket)
			purgeFakeGCSBucket(sharedFakeGCS.Endpoint, bucket)
			return &FakeGCSContainer{
				Container:  sharedFakeGCS.Container,
				Endpoint:   sharedFakeGCS.Endpoint,
				BucketName: bucket,
				ProjectID:  projectID,
			}
		}
		t.Logf("shared fake-gcs died; starting a replacement")
		_ = sharedFakeGCS.Container.Terminate(context.Background())
		sharedFakeGCS = nil
	}
	fresh := bootFakeGCS(t, projectID, bucket)
	sharedFakeGCS = &FakeGCSContainer{
		Container: fresh.Container,
		Endpoint:  fresh.Endpoint,
		ProjectID: projectID,
	}
	return fresh
}

func pingFileMinio(m *MinioContainer) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	client, err := fileMinioS3Client(ctx, m)
	if err != nil {
		return err
	}
	_, err = client.ListBuckets(ctx, &s3.ListBucketsInput{})
	return err
}

func pingAzurite(connStr string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	client, err := azblob.NewClientFromConnectionString(connStr, nil)
	if err != nil {
		return err
	}
	pager := client.NewListContainersPager(nil)
	_, err = pager.NextPage(ctx)
	return err
}

func pingFakeGCS(endpoint string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint+"/storage/v1/b?project=test-project", nil)
	if err != nil {
		return err
	}
	resp, err := (&http.Client{Timeout: 3 * time.Second}).Do(req)
	if err != nil {
		return err
	}
	_ = resp.Body.Close()
	return nil
}

// purgeFileMinioBucket deletes all objects; missing bucket is a no-op.
func purgeFileMinioBucket(m *MinioContainer, bucket string) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	client, err := fileMinioS3Client(ctx, m)
	if err != nil {
		return
	}
	var token *string
	for {
		out, err := client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
			Bucket:            aws.String(bucket),
			ContinuationToken: token,
		})
		if err != nil {
			return
		}
		if len(out.Contents) > 0 {
			ids := make([]types.ObjectIdentifier, 0, len(out.Contents))
			for _, o := range out.Contents {
				ids = append(ids, types.ObjectIdentifier{Key: o.Key})
			}
			if _, err := client.DeleteObjects(ctx, &s3.DeleteObjectsInput{
				Bucket: aws.String(bucket),
				Delete: &types.Delete{Objects: ids, Quiet: aws.Bool(true)},
			}); err != nil {
				return
			}
		}
		if !aws.ToBool(out.IsTruncated) {
			return
		}
		token = out.NextContinuationToken
	}
}

// purgeAzuriteContainer deletes all blobs; best-effort only.
func purgeAzuriteContainer(connStr, containerName string) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	client, err := azblob.NewClientFromConnectionString(connStr, nil)
	if err != nil {
		return
	}
	pager := client.NewListBlobsFlatPager(containerName, nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return
		}
		for _, b := range page.Segment.BlobItems {
			if b.Name == nil {
				continue
			}
			_, _ = client.DeleteBlob(ctx, containerName, *b.Name, nil)
		}
	}
}

// purgeFakeGCSBucket deletes all objects via JSON API; best-effort only.
func purgeFakeGCSBucket(endpoint, bucket string) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	httpClient := &http.Client{Timeout: 10 * time.Second}
	listReq, err := http.NewRequestWithContext(ctx, http.MethodGet,
		fmt.Sprintf("%s/storage/v1/b/%s/o", endpoint, bucket), nil)
	if err != nil {
		return
	}
	resp, err := httpClient.Do(listReq)
	if err != nil {
		return
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return
	}
	var listing struct {
		Items []struct {
			Name string `json:"name"`
		} `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&listing); err != nil {
		return
	}
	for _, item := range listing.Items {
		delReq, err := http.NewRequestWithContext(ctx, http.MethodDelete,
			fmt.Sprintf("%s/storage/v1/b/%s/o/%s", endpoint, bucket, item.Name), nil)
		if err != nil {
			continue
		}
		delResp, err := httpClient.Do(delReq)
		if err != nil {
			continue
		}
		_ = delResp.Body.Close()
	}
}
