# Tasks: Backend Cloud Deployment (Oracle Cloud, Phase 1) — 014

**Input**: Design documents from `/specs/014-backend-cloud-deployment/`
**Prerequisites**: `spec.md` (US1–US6), `plan.md` (as-built)
**Status**: **Delivered and applied.** Backend live at `http://130.210.26.64` (app deployed from `014-backend-cloud-deployment-0987949`). Three operator-only items remain — T046 (boot-volume policy), T052 (DNS + certbot), T054 (SSH restrict) — plus optional T055; every other task is done and, where it touches a host, applied and verified.

**Organization**: One phase per concern, in the order the work actually happened. Completed work is kept in the list (marked `[x]`) so nothing is lost or repeated.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Could run in parallel (different hosts/files, no dependency)
- **[Story]**: US1–US6 maps to `spec.md`
- Exact file paths or commands in every task

---

## Phase 1: Oracle Infrastructure (US1)

- [x] T001 Create VCN + public subnet (`10.0.0.0/24`) + internet gateway (OCI console)
- [x] T002 Create VM-1 `VM.Standard.E2.1.Micro` (Oracle Linux 9.8, data role) with public IP `140.238.240.2`
- [x] T003 Create VM-2 `VM.Standard.E2.1.Micro` (Oracle Linux 9.8, app role) with public IP `130.210.26.64`
- [x] T004 Generate SSH keypairs `~/.ssh/oci_vm1`, `~/.ssh/oci_vm2`; add public keys to the instances
- [x] T005 Security list: allow TCP `22` (operator choice: world-open, key-only — see T080)
- [x] T006 Security list: allow TCP `80` from `0.0.0.0/0` (public HTTP)
- [x] T007 Security list: allow TCP `5432` and `6381` from `10.0.0.0/24` only (private DB/KV)
- [x] T008 Reserve both public IPs (VM-2 `130.210.26.64`, VM-1 `140.238.240.2`) so they survive instance lifecycle changes
- [x] T009 External port scan: confirm `5432/6379/6380/6381/8080` unreachable from the internet on both VMs (US1 acceptance)

**Checkpoint**: hosts reachable, private plane gated, public addresses reserved.

---

## Phase 2: Host Bootstrap (US1, FR-006, FR-007)

- [x] T010 [P] [US1] Write `deploy/bootstrap.sh` (static Docker tarball, compose plugin, role-based firewalld, swap, package-scan suppression)
- [x] T011 [P] [US1] Install Docker 29.7.2 + Compose v5.7.1 on VM-1 (`sudo bash bootstrap.sh data`)
- [x] T012 [P] [US1] Install Docker 29.7.2 + Compose v5.7.1 on VM-2 (`sudo bash bootstrap.sh app`)
- [x] T013 [US1] Fix bootstrap bug: `mkswap -q` unsupported on OL9 → use `mkswap -f` with idempotent `swapon --show` guard
- [x] T014 [US1] Fix bootstrap bug: static tarball extracts to `docker/`, not `docker-<version>/` → copy `"$TMPD"/docker/*`
- [x] T015 [US1] Create `docker` group before `usermod -aG docker opc`
- [x] T016 [US1] Disable `dnf-makecache.timer` (and `unbound-anchor.timer`) to stop load spikes that froze sshd on 1 GB hosts
- [x] T017 [US1] Confirm 2.99 GB total swap on both hosts and `docker` usable by `opc`
- [x] T018 [US1] VM-level firewalld: `http`/`https` on VM-2; rich rules `5432`/`6381` from `10.0.0.0/24` on VM-1

**Checkpoint**: both hosts bootstrapped; Docker healthy; no package-manager thrash.

---

## Phase 3: Secrets & Environment (US2)

- [x] T019 [US2] Operator generates `DB_PASSWORD`, `CACHE_PASSWORD`/`KV_PASSWORD`, `JWT_SECRET`, `ENCRYPTION_KEY` (fresh, stored in password manager)
- [x] T020 [US2] Write `deploy/setup-secrets.sh`: hidden-input prompts, auto-detect VM-1 private IP, build both env files, scp, `chmod 600`, wipe temp dir on exit
- [x] T021 [US2] Install `~/ecommerce-data/.env` on VM-1 (mode 600, placeholder-free)
- [x] T022 [US2] Install `~/ecommerce-app/.env` on VM-2 (mode 600; `DB_HOST`/`KV_HOST=10.0.0.208`; `PUBLIC_API_BASE_URL=http://130.210.26.64`)
- [x] T023 [US2] Verify both env files contain the required keys and no `CHANGE_ME` placeholders

**Checkpoint**: secrets only on the servers; both files mode 600.

---

## Phase 4: Database Plane (US3)

- [x] T024 [US3] Write `deploy/vm1-data/docker-compose.yml` (Postgres tuned for 1 GB, durable Redis 6381 with AOF/noeviction, `migrate` tools profile)
- [x] T025 [US3] Copy compose file to `~/ecommerce-data/` on VM-1
- [x] T026 [US3] Copy `migrations/` (including `seeds/`, 37 entries) to `~/ecommerce-data/migrations/`
- [x] T027 [US3] `docker compose up -d`: Postgres + durable Redis healthy
- [x] T028 [US3] Run `docker compose --profile tools run --rm migrate` → `run_migrations.sh --production` (all schema migrations + CORE seeds; `seeds/mock` skipped)
- [x] T029 [US3] Verify 41 rows in `schema_migration` and 82 application tables
- [x] T030 [US3] Verify VM-2 → VM-1 connectivity on `5432` and `6381` over `10.0.0.0/24`

**Checkpoint**: schema + core seeds loaded; private connectivity proven.

---

## Phase 5: Application Plane (US3, US4)

- [x] T031 [US3] Build `ecommerce-backend:latest` for `linux/amd64` on the laptop (78.5 MB) — Docker Desktop context fixed from stale socket
- [x] T032 [US3] Ship image to VM-2 via `docker save | gzip | ssh | docker load`
- [x] T033 [US3] Write `deploy/vm2-app/docker-compose.yml` (app on `127.0.0.1:8080`, volatile Redis 64 MB LRU, nginx `:80`)
- [x] T034 [US3] Copy compose + nginx config to `~/ecommerce-app/`
- [x] T035 [US4] Fix nginx proxy bug: `proxy_pass http://127.0.0.1:8080` (container-local) → `http://app:8080` (compose service)
- [x] T036 [US3] Replace unusable app healthcheck (`/` requires correlation ID / 404s) with TCP probe on `127.0.0.1:8080`
- [x] T037 [US3] `docker compose up -d` on VM-2: app, cache, nginx all healthy
- [x] T038 [US3] Verify app → Postgres connection and middleware behavior (`400 CORRELATION_ID_REQUIRED`, `400 INVALID_SELLER`, `401` on protected routes)

**Checkpoint**: full stack running healthy; nginx → app → DB verified.

---

## Phase 6: Go-Live Verification (US1, US4, US5)

- [x] T039 [US4] Open OCI ingress TCP `80` from `0.0.0.0/0`
- [x] T040 [US4] Verify `http://130.210.26.64/` from the public internet (400 correlation JSON, ~50–60 ms)
- [x] T041 [US4] Re-run external port scan: only `22`+`80` on VM-2, only `22` on VM-1; DB/cache/app ports closed
- [x] T042 [US5] Verify CORS preflight/headers from a browser-equivalent origin (wildcard `*` currently)
- [x] T043 [US3] Confirm container healthchecks on both hosts (`docker ps` all healthy)
- [x] T044 [US2] Confirm no secret material in the repo (`deploy/` untracked templates only; `.env` is gitignored)

**Checkpoint**: backend live and publicly reachable; no data-plane exposure.

---

## Phase 7: Operations & Hardening (US6) — ✅ complete

- [x] T045 [US6] Nightly `pg_dump`: `deploy/vm1-data/backup/pg-backup.sh` (gzip, `gzip -t` integrity check, 7-day retention) + `install-backup.sh`; **installed on VM-1** (first dump 32K, cron `0 3 * * *`)
- [ ] T046 [US6] Boot-volume backup policy (weekly full + daily incremental) — **prepared; operator action pending (OCI console)**; steps in `deploy/ORACLE-SETUP-TODO.md` (Day-2 → Backups)
- [x] T047 [US6] Docker log rotation (`json-file`, `max-size=10m`, `max-file=3`) added to both compose files via an `x-logging` anchor; **applied on VM-1 and VM-2** (verified via `docker inspect .HostConfig.LogConfig`)
- [x] T048 [US6] Monitoring: `deploy/monitor/health-check.sh` (containers + root disk <85% + swap) + `install-monitor.sh` (5-min cron); **installed on VM-1** (`health=ok`); Prometheus deferred (documented)
- [x] T049 [US6] Restore procedure + drill in `deploy/vm1-data/backup/RESTORE.md`; **drill executed on VM-1** (restored 82 tables = live)
- [x] T050 [US4] CORS is now an env allowlist (`CORS_ALLOWED_ORIGINS`; default `*`); code + Config `dev` value shipped. Live value deliberately stays `*` until the frontends deploy (spec 015).
- [x] T051 [US2] `.env` rotation runbook: `deploy/ROTATION.md` (per-secret generate/apply/restart/verify + JWT/ENCRYPTION_KEY cautions)

---

## Phase 8: Deferred / Optional (US4, US1)

- [ ] T052 [US4] TLS rollout **prepared** (`deploy/vm2-app/TLS.md`: DNS `A api → 130.210.26.64`, certbot webroot, auto-renew cron, `PUBLIC_API_BASE_URL` switch). **Operator: add the DNS record**, then run certbot — hard dependency for spec 015.
- [x] T053 [US4] Gin trusted proxies: `TRUSTED_PROXIES=172.16.0.0/12` set in VM-2's `.env` and applied (compose network is `172.18.0.0/16`); code ships in the deployed image
- [ ] T054 [US1] SSH hardening **prepared; operator action pending**: restrict `22` to the operator IP or move to Tailscale (runbook Task 1) — currently key-only, world-open
- [ ] T055 [US1] (Optional) Revisit Ampere A1 capacity in the home region; migrate if available — not required now
- [x] T056 [US3] `ship-image.sh` now tags `:<branch>-<sha>` + `:latest` and preserves `:previous` for one-command rollback; deployed image is `ecommerce-backend:014-backend-cloud-deployment-0987949`

---

## Phase 9: Delivery Automation (GitHub deploy) — ✅ delivered

- [x] T057 Manual-trigger workflow `.github/workflows/deploy-oracle.yml` (`workflow_dispatch`; `ref` / `target_env` / `run_migrations` / `image_tag`; **no push trigger**): builds on a runner, tags `:<branch>-<sha>`, pushes to GHCR, deploys over SSH, writes `~/ecommerce-app/deploy-info.json`
- [x] T058 Deploy SSH key + secrets documented in `deploy/GITHUB-DEPLOY.md` (`DEPLOY_SSH_KEY`, `CONFIG_DEPLOY_KEY`, secret values, `VM1_HOST` / `VM2_HOST` variables) — key creation is an operator step
- [x] T059 Optional "run migrations" workflow input wired to the VM-1 `migrate` profile (`--production`)
- [x] T060 Build visibility: Actions run history + `deploy-info.json` + image tag/digest (`docker inspect`) — documented
- [x] ~~T061 Add a `/version` endpoint~~ — **DROPPED**. GitHub Actions run history + `deploy-info.json` + image tags provide build visibility; no API endpoint is required.

---

## Cross-Cutting Verification (executed 2026-11-17)

- [x] T062 VM-1: `ecommerce-postgres` healthy; `pg_isready -U postgres -d ecommerce` accepts connections
- [x] T063 VM-1: `ecommerce-cache-durable` healthy; `PING` via `redis-cli -a`
- [x] T064 VM-2: `ecommerce-app` healthy; logs show DB connect + cron registration + server on `:8080`
- [x] T065 VM-2: `ecommerce-cache-volatile` healthy; nginx serving
- [x] T066 Public: `curl http://130.210.26.64/` → `400 CORRELATION_ID_REQUIRED` (app alive)
- [x] T067 Security: external port scan confirms zero data-plane exposure
- [x] T068 Docs: `deploy/ORACLE-SETUP-TODO.md` updated with master checklist + current deployment record
- [x] T069 Repo: commit `deploy/`, `specs/014`, `specs/015`, `specs/016`, the app CORS/trusted-proxy config, and `.github/workflows/deploy-oracle.yml` on branch `014-backend-cloud-deployment`
- [x] T070 Docs: spec 015 / 016 cross-references added (spec 014 "Sibling specs"; 015 and 016 link back to 014)
- [x] T071 [US2] Config adoption: the `Zatu-ecom/Config` repo (`scripts/render-backend-env.sh`) renders a server `.env` from Config + secrets (validated, fail-fast); the full deploy-path switch remains future — server-side `.env` stays the source of truth until then
- [x] T072 [US3] Migration-policy guard documented in `deploy/ORACLE-SETUP-TODO.md` (Production migration policy): only `run_migrations.sh --production`; `seeds/mock`, `--all`, `--seed`, `--seed-mock`, `--force-seed`, `--reset` forbidden
- [x] T073 [US3] Post-migration verification SQL documented in the runbook (`schema_migration` all SUCCESS, reference tables populated, zero mock users/sellers)
- [x] T074 [US2] Config populated in `Zatu-ecom/Config` (not the app repo): `local/` + `dev/` for backend/storefront/seller-dashboard, `required/*.keys`, `scripts/validate.sh` + `render-backend-env.sh`, `README.md`, `.gitignore` (local/dev key sets verified identical)

---

## Dependencies & Notes

- **T052 (TLS) blocks spec 015** — an HTTPS frontend cannot call an HTTP API (mixed content). Only the DNS record + certbot remain.
- **Node-nightly `pg_dump` is live on VM-1**; the OCI **boot-volume backup policy (T046)** is the remaining backup gap — attach it in the console.
- **Applied on the VMs (2026-11-22):** log rotation (T047) on both hosts; the CORS + trusted-proxy image (T050/T053) redeployed to VM-2; backup + monitor crons on VM-1.
- **T054 (SSH) and T052 (TLS) are independent**; either can be done first.
- **T055 (Ampere A1) is optional** and not required by the current design.
- Production migrations always use `--production`; never `--all` (mock seeds).
- Configuration lives in `Zatu-ecom/Config` only (the app repo keeps `deploy/.env.example` for reference); adopting it (T071) is a value move, not a redesign.
