# Quickstart: Payment Gateway Platform (implementer)

Audience: smaller/faster models implementing this feature.  
**Do not invent design.** Follow `spec.md` + `pre-spec.md` + this folder’s plan/research/data-model/contracts.

Payment is **not in production**. Edit existing SQL. Recreate DB. **No** `032`. **No** API aliases.

---

## 0. Before coding

1. Read `contracts/gateway-adapter.md` and `contracts/payment-api.md` end to end.
2. Confirm branch `010-payment-gateway-platform`.
3. Local DB: you will drop/recreate or re-run migrations after rewriting `009`/`031`/`008`.
4. Unit tests live under `test/…`, never next to `payment/service`.

Forbidden after the adapter move (zero matches in orchestrators):

```bash
rg "RAZORPAY_EVENT|ParseRazorpay|DecryptSensitive|GATEWAY_CODE_RAZORPAY|HandleRazorpay" \
  payment/service/*.go payment/handler payment/route
rg "json:\"keyId\"" payment/model
```

Factory singleton may construct `razorpay.New`. Razorpay event constants live only in `payment/service/payment_gateway/razorpay/`.

---

## 1. Schema (do first)

Edit in place:

| File | Change |
|------|--------|
| `migrations/008_create_geo_tables.sql` | `payments_environment VARCHAR(20) NOT NULL DEFAULT 'sandbox'` on `seller_settings` |
| `migrations/009_create_payment_tables.sql` | Final CREATE: join tables; no arrays for country/currency; no `webhook_url`; no `config.country`; no `payment_method`; txn has `environment`, session/payment ids, `payment_method_details`, nullable fee; webhook `event_id NOT NULL`; unique `(gateway_id, event_id)`; index `(status, created_at)` |
| `migrations/031_…sql` | Slim: event table if not in 009; unique index without partial `WHERE event_id IS NOT NULL` |
| `migrations/seeds/core/004_seed_payment_gateways.sql` | No arrays/`webhook_url`; `INSERT` into join tables via `country.code='IN'` and `currency.code='INR'` |

GORM entities must match. Delete `payment/entity/payment_method.go`. Add join entities.

User entity + `SellerSettings*` models: `PaymentsEnvironment`. PUT settings persists it.

---

## 2. Adapter extract (do before more Razorpay switches)

Create:

```
payment/service/payment_gateway/contract.go
payment/service/payment_gateway/crypto.go
payment/service/payment_gateway/razorpay/adapter.go
payment/service/payment_gateway/razorpay/credentials.go
payment/service/payment_gateway/razorpay/http.go
payment/service/payment_gateway/razorpay/webhook.go
```

Delete flat `razorpay_gateway.go`, `razorpay_credentials.go` after move.  
Delete `payment/model/gateway_response.go`.

`service_factory.go`: `razorpay.New(os.Getenv("RAZORPAY_BASE_URL"))` only Razorpay import in payment factories.

---

## 3. Webhook pipeline

Route: `POST /:code` → `HandleWebhook` (not `HandleRazorpay`).  
Skip correlation still prefix `/api/payment/webhooks/`.

Service must **not** `switch` on `payment.captured`. Switch on `WebhookAction` only.  
**Must not** `FindAllForGateway` and take `[0]`.

Tests (integration, httptest fake Razorpay):

- Capture without `X-Correlation-ID` → 200
- Unknown code → 404
- Two sellers: B’s signature cannot complete A’s payment → 401
- Duplicate event id → 200, status stays completed once
- Amount mismatch → not completed
- `authorized` → still pending, event stored

---

## 4. Seller dashboard

Repos: `FindBySellerGatewayAndEnvironment`, `FindAllBySellerAndGateway`.  
List/detail/configure/deactivate/test per `contracts/payment-api.md`.  
Compose webhook URL from `PUBLIC_API_BASE_URL`.

Initiate uses store mode + priority + geo `EXISTS`.

---

## 5. Transactions / refunds / logs

- `?status=` on seller list
- Detail: `refundableAmountCents`, `environment`, `events`
- GET by id: customer **or** seller
- Second partial refund allowed
- `GET /webhook-logs` seller scoped

---

## 6. Reconcile cron

New `ReconcileService`. Register in payment `container.go` like promotion:

```go
cron.RegisterIntervalJob(5*time.Minute, "payment.reconcile_pending", svc.ReconcilePending)
```

Query pending older than TTL `FOR UPDATE SKIP LOCKED LIMIT 100`.  
`FetchRemoteStatus` → `applyNormalized`. Still open after TTL → fail `EXPIRED_UNPAID` + fail order.  
No session after 5 min → fail.  
Refunds stuck 30 min similarly.  
Overlap: if previous run still active, skip.

Do not hit real Razorpay in CI.

---

## 7. Test matrix (minimum)

| Test | Where |
|------|--------|
| HMAC map/mask/merge | `test/payment/payment_gateway/razorpay/` |
| applyNormalized fake adapter | `test/payment/` |
| Initiate checkout.fields, no keyId | integration |
| Dual env configure + partial secret | integration |
| Store production without prod keys | initiate 400 |
| Status filter totals | integration |
| Events on GET id, not list | integration |
| Webhook isolation two sellers | integration |
| Reconcile expires unpaid | integration (time travel / created_at backdate) |
| Test connection 200 vs 401 | integration httptest |
| Correlation still required on initiate | integration |

Run: `make test` or package `go test ./test/integration/payment/...`

---

## 8. Suggested implementation order

Matches pre-spec §17:

1. SQL + entities  
2. Adapter folder + purge Razorpay from base services  
3. Generic webhook + locate-then-verify + amount check  
4. Dual env + geo select + settings toggle  
5. Test connection  
6. Status filter + refundable + second partial  
7. Events on detail + webhook-logs  
8. Cron  
9. Docs (this spec folder is source of truth; update `plans/payment-backend-handoff.md` current surface if still referenced)

Do **not** add more Razorpay `switch` in `webhook_service.go` even if FE only needs `?status=`.

---

## 9. Manual local smoke (after implement)

1. Recreate DB, seed.  
2. Seller PUT configure sandbox.  
3. POST test → ok.  
4. Copy `webhookUrl` (providers cannot call localhost without a tunnel).  
5. Customer create pending order, POST initiate, confirm JSON has `checkout.fields.keyId` not top-level `keyId`.  
6. Signed webhook `payment.captured` without correlation header → 200, order confirmed.  
7. Seller GET txn id → events present; GET list `status=completed` → totals match.
