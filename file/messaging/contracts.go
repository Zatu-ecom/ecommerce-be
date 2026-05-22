// Package messaging contains wire-contract structs for messages published or consumed
// by the file module. These structs are serialised into the Payload field of the
// common/messaging.Envelope and must not carry any provider credentials.
package messaging

// ImageProcessRequested is the payload published on exchange "ecom.commands" with
// routing key "file.image.process.requested" when a complete-upload succeeds for an
// eligible purpose (PRODUCT_IMAGE, USER_AVATAR, or raster SELLER_LOGO).
//
// Contract definition: specs/003-upload-apis/contracts/file.image.process.requested.event.md
//
// Serialised into common/messaging.Envelope.Payload (json.RawMessage).
// All field names match the JSON contract exactly.
type ImageProcessRequested struct {
	// FileID is the UUIDv7 string exposed to clients (not the DB primary key).
	FileID string `json:"fileId"`

	// FileObjectID is the file_object.id DB primary key (uint64); used by the consumer
	// to look up the row and write back file_variant rows.
	FileObjectID uint64 `json:"fileObjectId"`

	// StorageConfigID allows the consumer to resolve a fresh blob adapter without
	// secrets being transmitted on the wire.
	StorageConfigID uint64 `json:"storageConfigId"`

	// BucketOrContainer is the bucket/container name snapshotted at init time.
	BucketOrContainer string `json:"bucketOrContainer"`

	// ObjectKey is the full deterministic object key within the bucket.
	ObjectKey string `json:"objectKey"`

	// MimeType is the content-type confirmed by HeadObject at complete time.
	MimeType string `json:"mimeType"`

	// SizeBytes is the provider-confirmed object size in bytes.
	SizeBytes int64 `json:"sizeBytes"`

	// Purpose is the FilePurpose constant (e.g. "PRODUCT_IMAGE").
	Purpose string `json:"purpose"`

	// VariantsRequested lists the variant codes the consumer should produce
	// (e.g. ["thumb_200", "thumb_600", "webp_1600"] for PRODUCT_IMAGE).
	VariantsRequested []string `json:"variantsRequested"`
}
