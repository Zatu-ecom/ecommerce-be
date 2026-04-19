# Contract: RabbitMQ message `file.image.process.requested`

Published by `POST /api/files/complete-upload` for eligible purposes (raster images). Consumed by the future image-variant worker.

## Topology (aligned with `file/RABBITMQ_FILE_MODULE_DESIGN.md`)

- **Exchange**: `ecom.commands` (type `topic`, durable)
- **Routing key**: `file.image.process.requested`
- **Queue (declared by consumer, not by this feature)**: `file.image.process.requested.q` bound to the exchange with that routing key.
- **DLX**: `ecom.dlx` (consumer-managed).

This feature's publisher performs a `PassiveDeclareExchange` on `ecom.commands`, and declares it (durable, topic, no auto-delete) if passive declare fails. It does **not** declare queues or bindings.

## Envelope

Uses `common/messaging/envelope.go` (v1 envelope shape):

```json
{
  "id": "msg-018f2c2d-...",
  "type": "file.image.process.requested",
  "version": 1,
  "source": "ecommerce-be/file",
  "occurredAt": "2026-04-18T10:04:12Z",
  "correlationId": "7f1c...-original-request-cid",
  "sellerId": 42,
  "payload": { ... see below ... }
}
```

## Payload

```json
{
  "fileId": "018f2c1a-7a3e-7b2c-b4e2-c2a9d3e80001",
  "fileObjectId": 12345,
  "storageConfigId": 9,
  "bucketOrContainer": "my-bucket",
  "objectKey": "seller/42/PRODUCT_IMAGE/2026/04/018f2c1a-...-hero-shot.jpg",
  "mimeType": "image/jpeg",
  "sizeBytes": 842317,
  "purpose": "PRODUCT_IMAGE",
  "variantsRequested": ["thumb_200", "thumb_600", "webp_1600"]
}
```

- `variantsRequested` is derived from purpose policy; this feature sends a **fixed list per purpose** (see `upload_policy.go`). The consumer is free to ignore unknown codes.
- `storageConfigId` (not credentials) lets the consumer resolve a fresh adapter without us sharing secrets on the wire.

## Delivery properties

- `delivery_mode = 2` (persistent)
- `content_type = "application/json"`
- `message_id = envelope.id`
- `correlation_id = envelope.correlationId`
- `headers.x-source = "file/upload"`

## Producer guarantees

- Publish is **confirmed** (publisher confirms enabled). Timeout: 2 s.
- On confirm success → insert `file_job{status=PUBLISHED}`.
- On timeout / nack → insert `file_job{status=FAILED_TO_PUBLISH, last_error=...}` and return 200 to the client anyway (upload is finalised; variant re-request is a separate flow).

## Consumer contract expectations (documented here for reference; implemented in a later feature)

- Idempotent on `payload.fileObjectId + variantsRequested`.
- May emit `file.image.process.completed` back on the same exchange.
- On unrecoverable failure, dead-letter to `ecom.dlx` with `x-death` headers preserved.

## Non-goals for this feature

- No `file.image.process.completed` consumer is implemented here.
- No retry loop on the producer side beyond the built-in `amqp091` publish confirm retry (one retry on transient connection error).
