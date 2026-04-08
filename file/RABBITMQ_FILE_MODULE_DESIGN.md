# RabbitMQ Design for File Module (Common vs Module-Specific)

> **Purpose**: Production-oriented RabbitMQ architecture for file processing, image/video variants, import/export jobs, and future module reuse.  
> **Last Updated**: April 5, 2026  
> **Decision**: Use **RabbitMQ** as primary async job backbone.

---

## 1) Why RabbitMQ Here

RabbitMQ is the best fit for your current workload:
- Task/job queues (image processing, variant generation, imports, exports)
- Reliable ack/nack semantics
- Retry + dead-letter queues (DLQ) are straightforward
- Lower operational complexity than Kafka for this stage

Kafka can still be added later for analytics/event-stream use-cases.

---

## 2) Architecture Overview

```text
HTTP API (file/product/order)
        |
        v
File Service / Product Service
  publish command messages
        |
        v
RabbitMQ Exchange(s)
        |
        +--> File Worker Queue(s)
        |       |
        |       +--> image/video processor
        |       +--> import worker
        |       +--> export worker
        |
        +--> Retry Queue(s) --> back to worker queues
        |
        +--> Dead Letter Queue(s) (manual inspection/replay)
```

---

## 3) What Goes in `common` vs `file` Module

## A) `common` (shared, reusable by all modules)

These should be generic and not file-domain specific.

1. **Messaging config**
- `common/config/messaging.go`
- env parsing:
  - `RABBITMQ_URL`
  - `RABBITMQ_EXCHANGE_COMMANDS`
  - `RABBITMQ_EXCHANGE_EVENTS`
  - `RABBITMQ_PREFETCH`
  - `RABBITMQ_CONSUMER_CONCURRENCY`
  - `RABBITMQ_RETRY_DELAY_MS`
  - `RABBITMQ_MAX_RETRIES`

2. **Connection/channel manager**
- `common/messaging/rabbitmq/connection.go`
- manages reconnect and lifecycle

3. **Publisher abstraction**
- `common/messaging/publisher.go`
- interface:
  - `Publish(ctx, exchange, routingKey, envelope)`

4. **Consumer abstraction**
- `common/messaging/consumer.go`
- interface:
  - `Consume(queue, handler)`
  - ack/nack handling helpers

5. **Envelope + metadata (shared schema)**
- `common/messaging/envelope.go`
- standard fields for all async messages:
  - `messageId`
  - `correlationId`
  - `traceId`
  - `tenantId`
  - `actorId`
  - `eventType`
  - `occurredAt`
  - `version`
  - `retryCount`
  - `payload`

6. **Retry/DLQ policy utilities**
- `common/messaging/retry.go`
- shared retry strategy + headers

7. **Idempotency helper**
- `common/messaging/idempotency.go`
- dedupe by `messageId` or `jobId`

8. **Shared constants**
- `common/constants/messaging_constants.go`
- exchange names, queue suffix conventions, header keys

9. **Observability integration**
- `common/log` integration for structured message logs
- shared metrics naming conventions

## B) `file` module (domain logic only)

1. **Routing keys and payload contracts**
- `file/messaging/contracts.go`

2. **Producers (publishers)**
- `file/service` publishes processing/import/export commands

3. **Consumers (workers)**
- `file/worker/*`
- handlers for:
  - image variants
  - video poster/transcode
  - import execution
  - export execution
  - virus scan

4. **Domain retry decisions**
- temporary failure vs permanent failure mapping

5. **DB state transitions**
- updates `file_job`, `file_object`, `file_variant`

6. **Business fallback logic**
- if thumbnail failed -> fallback to original
- if video transcode failed -> keep poster + original

---

## 4) Exchange, Queue, and Routing-Key Design

## Exchanges

1. `ecom.commands` (`topic`, durable)
- command-style work requests

2. `ecom.events` (`topic`, durable)
- optional completion/failure events for other modules

## Command Routing Keys (file module)

1. `file.image.process.requested`
2. `file.video.process.requested`
3. `file.virus.scan.requested`
4. `file.import.run.requested`
5. `file.export.run.requested`
6. `file.cleanup.delete.requested`

## Queues

1. `q.file.image.process`
2. `q.file.video.process`
3. `q.file.virus.scan`
4. `q.file.import.run`
5. `q.file.export.run`
6. `q.file.cleanup.delete`

Each queue gets:
- retry queue: `q.<name>.retry`
- dead-letter queue: `q.<name>.dlq`

---

## 5) Retry and DLQ Strategy

## Policy

1. On transient errors (network timeout, provider 5xx):
- nack/requeue via retry queue with delay

2. On permanent errors (invalid mime, corrupted payload):
- send directly to DLQ

3. Max retries:
- default 5 attempts

4. Retry backoff:
- exponential (e.g. 10s, 30s, 2m, 5m, 15m)

## Mechanism

Use queue TTL + DLX pattern:
- worker queue -> fail -> retry queue (x-message-ttl) -> back to worker queue
- after `maxRetries` -> DLQ

---

## 6) Message Envelope and Payloads

## Shared Envelope (common)

```json
{
  "messageId": "msg_01J...",
  "correlationId": "req_01J...",
  "traceId": "trace_abc",
  "tenantId": "seller_42",
  "actorId": "user_110",
  "eventType": "file.image.process.requested",
  "version": 1,
  "occurredAt": "2026-04-05T10:20:00Z",
  "retryCount": 0,
  "payload": {}
}
```

## File payload examples (module-specific)

### Image process command

```json
{
  "jobId": "job_5001",
  "fileObjectId": 9001,
  "sellerId": 42,
  "requestedVariants": ["THUMBNAIL_SM", "THUMBNAIL_MD", "WEBP"],
  "sourceMime": "image/jpeg"
}
```

### Import run command

```json
{
  "jobId": "job_9001",
  "importType": "PRODUCT_BULK_IMPORT",
  "inputFileId": 8011,
  "sellerId": 42,
  "initiatedBy": 110
}
```

---

## 7) File Module Workflows with RabbitMQ

## A) Upload -> image variants

1. upload complete API marks file `ACTIVE`
2. publish `file.image.process.requested`
3. worker generates derived variants and stores in `file_variant`
4. publish `file.image.process.completed` (optional event)
5. listing/detail APIs resolve best variant by profile

## B) Variant media link update

1. product variant update saves `product_variant_media`
2. verify each `file_object` ownership/seller scope
3. for unprocessed media, publish process command

## C) Export

1. create export job row (`QUEUED`)
2. publish `file.export.run.requested`
3. worker streams export and uploads result file
4. update job `SUCCESS` with `output_file_id`

---

## 8) Common Interfaces (recommended)

```go
// common/messaging/publisher.go
type Publisher interface {
    Publish(ctx context.Context, exchange, routingKey string, msg Envelope) error
}

// common/messaging/consumer.go
type Consumer interface {
    Consume(ctx context.Context, queue string, handler MessageHandler) error
}

type MessageHandler interface {
    Handle(ctx context.Context, msg Envelope) error
}
```

```go
// file/messaging/contracts.go
type ImageProcessRequested struct {
    JobID             string   `json:"jobId"`
    FileObjectID      uint     `json:"fileObjectId"`
    SellerID          uint     `json:"sellerId"`
    RequestedVariants []string `json:"requestedVariants"`
    SourceMime        string   `json:"sourceMime"`
}
```

---

## 9) Docker Compose Additions (for local)

Add service:

```yaml
rabbitmq:
  image: rabbitmq:3.13-management-alpine
  container_name: ecommerce-rabbitmq
  restart: unless-stopped
  ports:
    - "5672:5672"
    - "15672:15672"
  environment:
    RABBITMQ_DEFAULT_USER: ${RABBITMQ_USER}
    RABBITMQ_DEFAULT_PASS: ${RABBITMQ_PASSWORD}
  networks:
    - ecommerce-network
  healthcheck:
    test: ["CMD", "rabbitmq-diagnostics", "-q", "ping"]
    interval: 10s
    timeout: 5s
    retries: 5
```

And app env:
- `RABBITMQ_URL=amqp://user:pass@rabbitmq:5672/`

---

## 10) Reliability and Safety Rules

1. Consumers must be idempotent (safe on duplicate delivery).
2. Use manual ack only after DB + storage updates succeed.
3. Keep messages small; store large data in DB/object storage and pass IDs.
4. Add poison-message handling (DLQ alerting).
5. Add replay tooling for DLQ.
6. Enforce tenant/seller checks in worker before processing.

---

## 11) Observability Checklist

Metrics:
- queue depth per queue
- consume rate
- success/failure/retry counts
- processing latency by job type
- DLQ count

Logs:
- include `messageId`, `jobId`, `fileObjectId`, `sellerId`, `routingKey`

Tracing:
- propagate `traceId` from HTTP to async jobs

Alerts:
- DLQ > threshold
- retry storm
- queue lag > threshold time

---

## 12) Implementation Plan (Practical Order)

1. Build `common/messaging` abstractions and rabbit implementation.
2. Add RabbitMQ config and docker-compose service.
3. Add file command contracts and routing keys.
4. Implement first consumer: `image.process`.
5. Integrate publish from upload-complete flow.
6. Add retry/DLQ and metrics.
7. Add import/export consumers.
8. Add optional `ecom.events` publish for cross-module consumers.

---

## 13) Final Recommendation

For this project, implement RabbitMQ now with:
- shared infra in `common`
- domain workflows in `file`

This gives immediate reliability for media/import/export jobs and stays clean for future modules.

