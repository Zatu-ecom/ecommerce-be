# Secret rotation (spec 014 / T051)

Rotating a secret is **generate → update the `.env` → restart the affected
service → verify**. Do one secret at a time so a failure is attributable.

`.env` files live on the VMs only (mode 600). Never commit them. Regenerate the
files with `deploy/setup-secrets.sh`, or edit on the host:

```bash
ssh -i ~/.ssh/oci_vm1 opc@140.238.240.2 'nano ~/ecommerce-data/.env'
ssh -i ~/.ssh/oci_vm2 opc@130.210.26.64 'nano ~/ecommerce-app/.env'
```

## Per-secret runbook

| Secret | Generate | Where | Restart order | Notes |
|---|---|---|---|---|
| `DB_PASSWORD` | `openssl rand -base64 24` | VM-1 `.env` **and** VM-2 `.env` (must match) | VM-1 `postgres` → VM-2 `app` | Postgres password is set at first init; changing it requires `ALTER USER postgres PASSWORD '<new>';` inside the container, then update both `.env` files. Expect a short DB reconnect window. |
| `CACHE_PASSWORD` | `openssl rand -base64 24` | VM-1 `.env` (durable) **and** VM-2 `.env` (volatile + app) | both `cache-*` → `app` | Same value is used by cache and KV. `KV_PASSWORD` falls back to it. |
| `KV_PASSWORD` | reuse `CACHE_PASSWORD` or separate | VM-1 `.env` + VM-2 `.env` | `cache-durable` → `app` | Independent value only if you set it on both sides. |
| `JWT_SECRET` | `openssl rand -hex 32` | VM-2 `.env` | `app` | **All existing sessions are invalidated** — every user must log in again. Rotate during a quiet window. |
| `ENCRYPTION_KEY` | **do not rotate lightly** | VM-2 `.env` | — | Encrypts stored gateway/courier credentials in the DB. Rotating without re-encrypting those rows makes them undecryptable. Only rotate with a migration that re-encrypts. |

## Procedure

```bash
# 1. Generate the new value locally and store it in the password manager.
# 2. Edit the matching .env on the VM(s) (values quoted if they contain $ or #).
# 3. Recreate only the affected services:
ssh -i ~/.ssh/oci_vm2 opc@130.210.26.64 'cd ~/ecommerce-app && docker compose up -d app'
#    (VM-1: ssh -i ~/.ssh/oci_vm1 opc@140.238.240.2 \
#       "cd ~/ecommerce-data && docker compose up -d postgres cache-durable")
# 4. Verify:
curl -s -o /dev/null -w '%{http_code}\n' http://130.210.26.64/   # expect 400
docker compose logs --tail=50 app | grep -iE 'connect|error'
```

Record each rotation (secret name + date) in the deployment record. Rotate
immediately if a value is ever exposed (chat, screenshot, git).
