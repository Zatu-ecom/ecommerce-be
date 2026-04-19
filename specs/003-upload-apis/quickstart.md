# Quickstart — Running & Writing Tests for the Upload APIs

This is the integration-test playbook. It assumes you have Docker running locally (required by Testcontainers) and
Go 1.25+. No production secrets are ever needed to run tests.

---

## 1. Run the upload suite only

```bash
cd ecommerce-be

# First time only: make sure testcontainers can pull images
docker pull postgres:16-alpine redis:7-alpine \
  rabbitmq:3.13-management-alpine \
  quay.io/minio/minio:latest

# Run all upload integration tests
go test ./test/integration/file/... -run '^TestUploadSuite$' -v -count=1

# Run a single story (example: policy enforcement = US3)
go test ./test/integration/file/... -run '^TestUploadSuite$/TestPolicyRejectsOversizedImage' -v
```

CI command:

```bash
make test-json   # existing make target; the upload suite is picked up automatically
```

---

## 2. What the suite stands up

One `UploadSuite` (`test/integration/file/setup_upload_suite_test.go`) boots **four** containers once per package:

| Container | Image | Purpose |
|---|---|---|
| Postgres | `postgres:16-alpine` | Runs `migrations/*.sql` once |
| Redis | `redis:7-alpine` | Backs `common/scheduler` + idempotency cache |
| MinIO | `quay.io/minio/minio:latest` | Blob provider (S3-compatible) |
| RabbitMQ | `rabbitmq:3.13-management-alpine` | Broker for variant messages |

After boot, the suite:

1. Truncates all file-related tables.
2. Seeds one **admin**, one **seller**, one **customer**, one **platform** `storage_config` pointing at MinIO, and one **seller** `storage_config` + `seller_storage_binding`.
3. Creates the MinIO bucket.
4. Declares the `ecom.commands` exchange and a **test-only** `file.image.process.requested.q` queue bound to it (so the test can assert on messages). This queue does **not** exist in production; only the publisher side is exercised by production code.
5. Starts `common/scheduler` workers against the Redis container and registers the `file.upload.expiry` handler.
6. Stands up the full Gin router via `setup.SetupTestServer(...)`.

---

## 3. Test helper API (new)

File: `test/integration/helpers/upload_helper.go`

```go
type UploadJourney struct {
    Client   *APIClient
    Suite    *UploadSuite
}

// Happy-path helper: init + PUT + complete, returns the final "data" block from complete-upload.
func (u *UploadJourney) RunHappyPath(t *testing.T, req InitUploadRequest) CompleteUploadData

// Just init — useful for negative tests on complete.
func (u *UploadJourney) Init(t *testing.T, req InitUploadRequest, opts ...InitOption) InitUploadData

// Just the PUT — uses the returned upload URL/headers and an arbitrary body.
func (u *UploadJourney) PutBytes(t *testing.T, init InitUploadData, body []byte)

// Assert DB state directly.
// NOTE (CA5 / Constitution §IV API-First exception): This test suite queries the database
// directly to verify state transitions (UPLOADING → ACTIVE → FAILED) because no GET/read
// endpoint exists in this feature slice (download URL API is explicitly out of scope per
// spec §Out of Scope). This is an approved, bounded exception to the API-First Testing Rule.
// When a read endpoint is added in a later slice, assertions should migrate to use that API.
func (u *UploadJourney) AssertFileObject(t *testing.T, fileID string, expected FileObjectAssert)

// Assert a pending/cancelled scheduler job.
func (u *UploadJourney) AssertSchedulerJobExists(t *testing.T, fileObjectID uint64)
func (u *UploadJourney) AssertNoSchedulerJob(t *testing.T, fileObjectID uint64)

// Drain a test-only RabbitMQ consumer registered in SetupSuite.
func (u *UploadJourney) NextVariantMessage(t *testing.T, timeout time.Duration) VariantCommand
func (u *UploadJourney) AssertNoVariantMessage(t *testing.T, within time.Duration)
```

Key design: the helper owns a long-lived RabbitMQ consumer started in `SetupSuite` and buffers messages in a channel so
each test can assert exactly-once without re-subscribing.

---

## 4. Seed data IDs (shared across upload tests)

| Logical role | `user.id` | JWT built by | Notes |
|---|---|---|---|
| Admin | `1` | `helpers.AuthAsAdmin(t)` | role=`admin` |
| Seller A | `10` | `helpers.AuthAsSeller(t, 10)` | role=`seller`, sellerId=10 |
| Seller B | `11` | `helpers.AuthAsSeller(t, 11)` | role=`seller`, sellerId=11 (US4 cross-tenant) |
| Customer | `20` | `helpers.AuthAsCustomer(t)` | role=`customer` — must be 403 |

Platform `storage_config.id` = 1; Seller A binding → `storage_config.id` = 2; Seller B has **no** binding in the default seed
(needed by the `NO_STORAGE_CONFIG` test; individual tests can add one via a helper).

---

## 5. Canonical test scenarios → file mapping

| Story | File | Key assertions |
|---|---|---|
| US1 Happy path (P1) | `upload_init_test.go` + `upload_complete_test.go` | 201 init, object in MinIO after PUT, 200 complete with `variantsQueued=true`, row `ACTIVE`, `file_job{status=PUBLISHED}`, RabbitMQ message received, scheduler job cancelled |
| US2 Non-image doc (P2) | `upload_complete_test.go` | `variantsQueued=false`, no RabbitMQ message, no `file_job` row |
| US3 Policy violation (P2) | `upload_policy_test.go` | 422 before any DB write; no scheduler job; no MinIO call |
| US4 Tenant isolation (P2) | `upload_tenant_isolation_test.go` | Seller B complete → 404; admin cross-seller → 404; DB row untouched |
| US5 Complete-before-object (P3) | `upload_complete_test.go` | 409 `OBJECT_MISSING`, row stays `UPLOADING`, scheduler job still present |
| US5a Idempotency (P2) | `upload_idempotency_test.go` | Same key + same fingerprint → same `fileId`; same key + different fingerprint → 409; no header → new `fileId` |
| US6 Storage outage (P3) | `upload_outage_test.go` | Simulate MinIO pause → 503 on init; no DB row; no scheduler job |
| US6a Abandoned cleanup (P2) | `upload_expiry_test.go` | Init with `uploadExpiryMinutes=5` + jump Redis clock via `scheduler.RunDueNow(t)` test hook → row `FAILED`, stray object deleted best-effort |
| Perf SC-001 | `upload_performance_test.go` (build tag `perf`) | 1 MB round trip < 3 s p95 |

---

## 6. How to assert RabbitMQ without flakiness

- The suite creates a **single** exclusive consumer during `SetupSuite` that pushes into `u.variantMsgs chan VariantCommand` (buffered, size 32).
- Tests call `u.NextVariantMessage(t, 2*time.Second)` to pull.
- To assert "no message", call `u.AssertNoVariantMessage(t, 200*time.Millisecond)`; the helper drains the channel first, runs the upload action, then waits the window.
- `TearDownSuite` closes the channel and cancels the consumer.

---

## 7. How to assert scheduler state

- The helper exposes `AssertSchedulerJobExists(t, fileObjectID)` which does a `ZRANGEBYSCORE delayed_jobs ...` scan and
  filters by payload. This is the same mechanism used by the inventory reservation tests, lifted into a shared helper in
  `test/integration/helpers/scheduler_helper.go` (new).
- To fast-forward an expiry, the test calls `u.FastForwardExpiry(t, fileID)` which:
  1. Reads the job ID.
  2. Rewrites the score in Redis to `now - 1s`.
  3. Waits up to 2 s for the handler loop to pick it up, polling the row's status.

This avoids `time.Sleep(5 * time.Minute)` and keeps tests sub-second.

---

## 8. Writing a new upload test (template)

```go
func (s *UploadSuite) TestMyNewScenario() {
    // Scenario: <one sentence>
    // Validates:
    // 1. <observable behaviour 1>
    // 2. <observable behaviour 2>

    journey := s.NewUploadJourney(helpers.AuthAsSeller(s.T(), 10))

    init := journey.Init(s.T(), helpers.InitReqProductImage(842317), helpers.WithIdempotencyKey("abc"))
    journey.PutBytes(s.T(), init, helpers.SamplePNGBytes(842317))

    data := journey.Complete(s.T(), init.FileID)

    s.Require().Equal("ACTIVE", data.Status)
    s.Require().True(data.VariantsQueued)

    journey.AssertFileObject(s.T(), init.FileID, helpers.FileObjectAssert{
        Status:     "ACTIVE",
        SellerID:   10,
        MimeType:   "image/png",
        HasEtag:    true,
    })
    journey.AssertNoSchedulerJob(s.T(), init.FileObjectID)

    msg := journey.NextVariantMessage(s.T(), 2*time.Second)
    s.Require().Equal(init.FileID, msg.FileID)
}
```

---

## 9. Troubleshooting

| Symptom | Likely cause | Fix |
|---|---|---|
| `failed to start minio container` | Docker daemon not running or image not pulled | `docker pull quay.io/minio/minio:latest` |
| RabbitMQ test consumer never delivers | Test forgot to declare the test-only queue | Ensure `SetupSuite` calls `declareTestVariantQueue(...)` |
| `SCHEDULER_JOB_NOT_FOUND` when asserting | Handler already ran (race) | Use `FastForwardExpiry` after assertion, or assert against final row state instead |
| `NO_STORAGE_CONFIG` for seller A | Seed order issue | Ensure seller binding seed runs after platform config seed |
| Test passes locally, fails in CI | Docker socket perms / slow pulls | Pre-pull images in CI job; set `TESTCONTAINERS_STARTUP_TIMEOUT=5m` |
