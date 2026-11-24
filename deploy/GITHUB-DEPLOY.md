# Manual GitHub deploy (spec 014 / T057–T060)

Workflow: `.github/workflows/deploy-oracle.yml` — **manual only**
(`workflow_dispatch`); there is deliberately no push trigger, so nothing ships
without an explicit Run.

## Required repository settings

**Secrets** (Settings → Secrets and variables → Actions):

| Secret | Purpose |
|---|---|
| `DEPLOY_SSH_KEY` | Private key permitted to SSH to VM-1 and VM-2 as `opc` (T058) |
| `CONFIG_DEPLOY_KEY` | Read-only deploy key for the private `Zatu-ecom/Config` repo |
| `DB_PASSWORD`, `JWT_SECRET`, `ENCRYPTION_KEY`, `CACHE_PASSWORD`, `KV_PASSWORD`, `RABBITMQ_PASSWORD` | Secret values rendered into the server `.env` |

**Variables**:

| Variable | Value |
|---|---|
| `VM1_HOST` | `140.238.240.2` |
| `VM2_HOST` | `130.210.26.64` |

## Inputs

| Input | Meaning |
|---|---|
| `ref` | Branch or tag to build (default `main`) |
| `target_env` | `dev` or `prod` → which `Config/<env>/backend.env` to render |
| `run_migrations` | When true, runs `run_migrations.sh --production` on VM-1 (schema + CORE seeds only) |
| `image_tag` | Optional tag override; default `<branch>-<sha>` (T056) |

## What a run does

No container registry is involved: VM-2 is the only consumer, so the image is
copied straight over SSH. That removes the token/visibility failure mode a
private GHCR package adds. Add a registry later only if a second machine needs
to pull the image.

1. Checks out the app at `ref` and the Config repo.
2. Validates + renders `.env` from Config + secrets (fails early on a missing key).
3. Builds a `linux/amd64` image on the runner — **no registry is used**.
4. Ships the image to VM-2 over SSH (`docker save | gzip | ssh | docker load`),
   then ships the rendered `.env`, retags the running image as `:previous`,
   starts the new tag, and writes `~/ecommerce-app/deploy-info.json`.
5. Optionally runs migrations on VM-1.

## Build visibility (T060)

- Actions run history shows branch / commit / actor / inputs.
- `~/ecommerce-app/deploy-info.json` on VM-2 records tag, config SHA, actor, ref, env, timestamp.
- `docker inspect ecommerce-backend:latest` shows the exact image digest running.

## Rollback

On VM-2: `cd ~/ecommerce-app && IMAGE_TAG=previous docker compose up -d app`.
Or re-run the workflow with an earlier `ref`/`image_tag`.

## Still manual (by design, for now)

The first production deploys may continue to use `deploy/ship-image.sh` from a
workstation; this workflow is the tracked path to replace it.
