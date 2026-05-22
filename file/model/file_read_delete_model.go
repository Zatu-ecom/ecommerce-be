package model

import (
	"time"

	"ecommerce-be/common"
	"ecommerce-be/common/helper"
	"ecommerce-be/file/entity"
	"ecommerce-be/file/utils/constant"
)

// GetFilesBase contains the shared list controls for file read operations.
type GetFilesBase struct {
	common.BaseListParams
	IncludeVariants    bool `form:"includeVariants" binding:"omitempty"`
	IncludeDownloadURL bool `form:"includeDownloadUrl" binding:"omitempty"`
}

// GetFilesParam is the query-binding model for GET /api/file.
type GetFilesParam struct {
	GetFilesBase
	FileIDs   *string `form:"fileIds" binding:"omitempty"`
	Purposes  *string `form:"purposes" binding:"omitempty"`
	Statuses  *string `form:"statuses" binding:"omitempty"`
	MimeTypes *string `form:"mimeTypes" binding:"omitempty"`
}

// GetFilesFilter is the normalized filter passed into the repository layer.
type GetFilesFilter struct {
	GetFilesBase
	FileIDs   []string
	Purposes  []string
	Statuses  []string
	MimeTypes []string
}

// GetFileQuery contains the optional query params for GET /api/file/:fileId.
type GetFileQuery struct {
	IncludeDownloadURL bool `form:"includeDownloadUrl" binding:"omitempty"`
	URLTTLMinutes      *int `form:"urlTtlMinutes" binding:"omitempty"`
}

// DownloadURLQuery contains the query params for GET /api/file/:fileId/download-url.
type DownloadURLQuery struct {
	TTLMinutes  *int    `form:"ttlMinutes" binding:"omitempty"`
	VariantCode *string `form:"variantCode" binding:"omitempty"`
	Disposition string  `form:"disposition" binding:"omitempty,oneof=inline attachment"`
}

// FileVariantItem is the API response shape for a file variant.
type FileVariantItem struct {
	VariantCode string `json:"variantCode"`
	MimeType    string `json:"mimeType"`
	SizeBytes   int64  `json:"sizeBytes"`
	Width       *int   `json:"width,omitempty"`
	Height      *int   `json:"height,omitempty"`
	Status      string `json:"status"`
}

// FileItem is the list/get response shape for file metadata.
type FileItem struct {
	FileID           string            `json:"fileId"`
	Status           string            `json:"status"`
	Purpose          string            `json:"purpose"`
	Visibility       string            `json:"visibility"`
	OriginalFilename string            `json:"originalFilename"`
	MimeType         string            `json:"mimeType"`
	SizeBytes        int64             `json:"sizeBytes"`
	Etag             *string           `json:"etag,omitempty"`
	ObjectKey        string            `json:"objectKey"`
	StorageProvider  string            `json:"storageProvider"`
	CreatedAt        string            `json:"createdAt"`
	CompletedAt          *string           `json:"completedAt,omitempty"`
	Variants             []FileVariantItem `json:"variants"`
	DownloadURL          *string           `json:"downloadUrl,omitempty"`
	DownloadURLExpiresAt *string           `json:"downloadUrlExpiresAt,omitempty"`
}

// GetFilesResponse is the list payload for GET /api/file.
type GetFilesResponse struct {
	Items      []FileItem                 `json:"items"`
	Pagination common.PaginationResponse  `json:"pagination"`
}

// GetFileResponse is the single-file payload for GET /api/file/:fileId.
type GetFileResponse = FileItem

// DownloadURLResponse is the standalone download-url payload.
type DownloadURLResponse struct {
	FileID      string  `json:"fileId"`
	VariantCode *string `json:"variantCode,omitempty"`
	DownloadURL string  `json:"downloadUrl"`
	ExpiresAt   string  `json:"expiresAt"`
	TTLMinutes  int     `json:"ttlMinutes"`
	MimeType    string  `json:"mimeType"`
	SizeBytes   int64   `json:"sizeBytes"`
}

// DeleteFileResponse is the DELETE /api/file/:fileId response payload.
type DeleteFileResponse struct {
	FileID    string `json:"fileId"`
	DeletedAt string `json:"deletedAt"`
}

// ToFilter normalizes the list query params into a repository filter.
func (p *GetFilesParam) ToFilter() GetFilesFilter {
	filter := GetFilesFilter{
		GetFilesBase: p.GetFilesBase,
	}

	if p.FileIDs != nil {
		filter.FileIDs = helper.ParseCommaSeparatedPtr[string](p.FileIDs)
	}
	if p.Purposes != nil {
		filter.Purposes = helper.ParseCommaSeparatedPtr[string](p.Purposes)
	}
	if p.Statuses != nil {
		filter.Statuses = helper.ParseCommaSeparatedPtr[string](p.Statuses)
	}
	if p.MimeTypes != nil {
		filter.MimeTypes = helper.ParseCommaSeparatedPtr[string](p.MimeTypes)
	}

	return filter
}

// SetDefaults normalizes pagination, sorting, and default ACTIVE-status filtering.
func (f *GetFilesFilter) SetDefaults() {
	f.BaseListParams.SetDefaults()
	if f.SortBy == "" || f.SortBy == "created_at" {
		f.SortBy = "createdAt"
	}
	if len(f.Statuses) == 0 {
		f.Statuses = []string{string(entity.FileStatusActive)}
	}
}

// ResolveURLTTLMinutes returns the effective download URL TTL in minutes.
func (q GetFileQuery) ResolveURLTTLMinutes() int {
	if q.URLTTLMinutes == nil {
		return constant.DefaultDownloadURLTTLMinutes
	}
	return *q.URLTTLMinutes
}

// ResolveTTLMinutes returns the effective standalone download URL TTL.
func (q DownloadURLQuery) ResolveTTLMinutes() int {
	if q.TTLMinutes == nil {
		return constant.DefaultDownloadURLTTLMinutes
	}
	return *q.TTLMinutes
}

// ResolveDisposition returns the effective download disposition.
func (q DownloadURLQuery) ResolveDisposition() string {
	if q.Disposition == "" {
		return constant.DownloadDispositionInline
	}
	return q.Disposition
}

// FormatTimestamp returns an RFC3339 UTC timestamp string.
func FormatTimestamp(t time.Time) string {
	return t.UTC().Format(time.RFC3339)
}

// FormatTimestampPtr returns an RFC3339 UTC timestamp string pointer.
func FormatTimestampPtr(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.UTC().Format(time.RFC3339)
	return &s
}
