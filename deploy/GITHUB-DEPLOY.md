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

The **branch/tag is chosen with GitHub's "Use workflow from" selector** at the
top of the Run-workflow dialog — there is no separate ref input, so the workflow
file and the code always come from the same ref.

| Input | Meaning |
|---|---|
| `target_env` | `dev` → which `Config/<env>/backend.env` to render |
| `run_migrations` | When true, runs DB migrations (schema + CORE seeds; never mock data) |
| `image_tag` | Optional tag override; default `<branch>-<sha>` (T056) |

## What a run does

No container registry is involved: VM-2 is the only consumer, so the image is
copied straight over SSH. That removes the token/visibility failure mode a
private GHCR package adds. Add a registry later only if a second machine needs
to pull the image.

1. Checks out the app at the "Use workflow from" ref, plus the Config repo.
2. Validates + renders `.env` from Config + secrets (fails early on a missing key).
3. Builds a `linux/amd64` image on the runner — **no registry is used**.
4. Optionally runs DB migrations on VM-1 **before** the app is switched.
5. Ships the image to VM-2 over SSH (`docker save | gzip | ssh | docker load`),
   writes the rendered `.env` (mode 600 from the first byte), retags the running
   image as `:previous`, and starts the new tag with `--wait` — so the job fails
   if the app never becomes healthy.
6. Writes `~/ecommerce-app/deploy-info.json` **after** the health check passes.

The job runs under a GitHub Environment named after `target_env` (`dev`), so you
can attach required reviewers / protection rules there if you want a gate.

## Build visibility (T060)

- Actions run history shows branch / commit / actor / inputs.
- `~/ecommerce-app/deploy-info.json` on VM-2 records tag, config SHA, actor, ref, env, timestamp.
- `docker inspect ecommerce-backend:latest` shows the exact image digest running.

## Rollback

On VM-2: `cd ~/ecommerce-app && IMAGE_TAG=previous docker compose up -d app`.
Or re-run the workflow from an earlier branch/tag ("Use workflow from") or with an `image_tag`.

## Still manual (by design, for now)

The first production deploys may continue to use `deploy/ship-image.sh` from a
workstation; this workflow is the tracked path to replace it.

---

# Frontend deploys (Vercel Hobby) — spec 015

Two Next.js apps deploy to **Vercel Hobby** from their own repositories:
`Zatu-ecom/storefront` and `Zatu-ecom/seller-dashboard`. Each repo carries
`.github/workflows/deploy-vercel.yml` — **manual only** (`workflow_dispatch`),
same shape as the backend workflow.

## Why GitHub Actions and not Vercel's Git integration

Vercel does not support connecting a project on a **Hobby team** to a Git
repository owned by a **GitHub organization**. `Zatu-ecom` is an organization, so
the Git integration is unavailable. The CLI path (`vercel deploy`) is the only
option on Hobby.

## Required repository settings (in BOTH frontend repos)

**Secrets**:

| Secret | Purpose |
|---|---|
| `VERCEL_TOKEN` | Account token from Vercel → Account → Tokens |
| `VERCEL_TEAM_ID` | Team/org ID (`orgId` in `.vercel/project.json`) — mapped to `VERCEL_ORG_ID` in the workflow |
| `VERCEL_PROJECT_ID` | `projectId` from `.vercel/project.json` |
| `CONFIG_DEPLOY_KEY` | Read-only deploy key for `Zatu-ecom/Config` (same key as the backend) |

The workflow maps `secrets.VERCEL_TEAM_ID` onto the `VERCEL_ORG_ID` environment
variable, because that is the name the Vercel CLI reads.

**Variables**: none — the API URL and tenant values live in Vercel's project
environment, and Config is the source of truth (see below).

## What a run does

1. Checks out the app at the "Use workflow from" ref, plus the Config repo.
2. Records the Config commit SHA.
3. Validates `config/<env>/<service>.env` against `required/<service>.keys`.
4. **Drift check** — pulls the Vercel project's production variables and fails if
   any `NEXT_PUBLIC_*` value differs from Config. A config change therefore
   cannot silently miss a deploy.
5. `npm ci`, then strips `.git` and runs `vercel deploy --prod` (Vercel builds
   and serves; no artifacts are shipped from the runner).
6. Writes the deployment URL and Config SHA to the Actions summary.

The build runs on Vercel rather than on the runner so it uses the project's own
build cache and system environment variables — `vercel build` + `--prebuilt`
drops the Vercel system variables at build time.

### Why the workflow deletes `.git`

A Vercel **Hobby** team only accepts deployments whose commit author is the team
owner; anything else fails with `TEAM_ACCESS_REQUIRED`. Club that with the fact
that a Hobby project cannot be git-connected to an organization repository in the
first place, and the checkout's git metadata is pure liability — it can only block
the deploy. The Config SHA is recorded *before* the strip, so traceability is
unaffected.

If commits are ever signed by a second identity (e.g. a CI bot), the durable fix
is to add that email under Vercel → Account Settings → Email and verify it.

## Environment variables

`NEXT_PUBLIC_*` values are inlined by Next.js at build time, so they must exist
in the Vercel project. Set them once per project, per environment:

```bash
# storefront
vercel env add NEXT_PUBLIC_API_URL   production --value "https://api.dev.ecom.vitrapay.io/api" --no-sensitive --yes
vercel env add NEXT_PUBLIC_SELLER_ID production --value "<real seller id>" --no-sensitive --yes
vercel env add NEXT_PUBLIC_SITE_NAME production --value "E-Shop" --no-sensitive --yes
# repeat with --environment preview

# seller-dashboard (derives the seller id from the login session)
vercel env add NEXT_PUBLIC_API_URL   production --value "https://api.dev.ecom.vitrapay.io/api" --no-sensitive --yes
# repeat with --environment preview
```

Keep them as **Config** type (not Secret) — the drift check reads the values
back. `NEXT_PUBLIC_*` variables are public by definition; they must never hold a
backend secret.

`NEXT_PUBLIC_API_URL` requires the `/api` path prefix. `PUBLIC_API_BASE_URL` on
the backend stays origin-only — they are not the same value.

## Rollback

Promote the previous deployment — no backend involvement:

- Dashboard → Project → Deployments → pick the good one → **Promote to
  Production**.
- CLI: `vercel promote <deployment-url>`.

A Config rollback needs the Vercel variables updated first; the drift check will
otherwise refuse the deploy and name the mismatched key.

## Still manual

- The two frontends are separate Vercel projects, each needing its own
  `vercel link` and secret set.
- Custom domains (`shop.vitrapay.io`, `seller.vitrapay.io`) are optional and
  attached in the Vercel dashboard (spec 015 US6, P3).
- Vercel Hobby is restricted to non-commercial use; a live commercial storefront
  requires Pro or a different host.
