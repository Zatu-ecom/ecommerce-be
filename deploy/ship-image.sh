#!/usr/bin/env bash
# Build the backend image for linux/amd64 locally and load it on VM-2 (app).
# The 1GB Micro cannot build Go - never build on the box.
#
# Images are tagged by branch+SHA AND :latest, so a build is traceable and the
# previously running image is kept as :previous for one-command rollback
# (spec 014 / T056).
#
# Usage (from your workstation):
#   ./deploy/ship-image.sh opc@<APP_VM_PUBLIC_IP> [ssh-key-path]
#
# Env overrides:
#   IMAGE_TAG=<tag>   force a tag (default: <branch>-<shortsha>)
set -euo pipefail

TARGET="${1:?usage: ship-image.sh opc@<APP_VM_PUBLIC_IP> [key]}"
KEY="${2:-$HOME/.ssh/oci_vm2}"
ROOT="$(cd "$(dirname "$0")/.." && pwd)"   # ecommerce-be root

BRANCH="$(git -C "$ROOT" rev-parse --abbrev-ref HEAD | tr '/' '-')"
SHA="$(git -C "$ROOT" rev-parse --short HEAD)"
TAG="${IMAGE_TAG:-${BRANCH}-${SHA}}"

echo "==> Building linux/amd64 image from $ROOT (tag: $TAG)"
docker buildx build --platform linux/amd64 --load \
  -t "ecommerce-backend:${TAG}" \
  -t "ecommerce-backend:latest" \
  "$ROOT"

echo "==> Preserving the running image as :previous, then shipping $TAG"
# `docker tag` runs before `docker load`; only load reads stdin, so the pipe is safe.
docker save "ecommerce-backend:${TAG}" "ecommerce-backend:latest" | gzip \
  | ssh -i "$KEY" "$TARGET" \
      'sh -c "docker image inspect ecommerce-backend:latest >/dev/null 2>&1 && docker tag ecommerce-backend:latest ecommerce-backend:previous || true; docker load"'

cat <<EOF

==> Shipped. On VM-2:
    cd ~/ecommerce-app
    IMAGE_TAG=$TAG docker compose up -d app      # run the new build
    # rollback:
    IMAGE_TAG=previous docker compose up -d app
EOF
