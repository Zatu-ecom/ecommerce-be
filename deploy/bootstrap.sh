#!/usr/bin/env bash
# Oracle Cloud 1GB-Micro bootstrap — NO dnf transactions (they OOM-storm on
# free-tier Micros). Roles: VM-1=data, VM-2=app.
# Installs: swap iff <2GB, static Docker + compose plugin, firewalld rules.
# Usage: sudo bash bootstrap.sh data|app     (SUBNET_CIDR override optional)
#
# Pinned versions (override via env): DOCKER_VERSION, COMPOSE_VERSION.
# Verification: the compose plugin checksum is always verified; set
# DOCKER_TGZ_SHA256 to verify the engine tarball too.
set -euo pipefail
ROLE="${1:?usage: bootstrap.sh data|app}"
SUBNET_CIDR="${SUBNET_CIDR:-10.0.0.0/24}"
DOCKER_VERSION="${DOCKER_VERSION:-29.7.2}"
COMPOSE_VERSION="${COMPOSE_VERSION:-v5.7.1}"

if [ "$(awk '/MemTotal/{print int($2/1024)}' /proc/meminfo)" -gt 1600 ]; then
  echo "Box >1.5GB RAM — plain dnf would work, keeping static install anyway."
fi

echo "==> Swap (want >=2GB total; Micro images ship ~1G /swapfile)"
SWAP_MB_TOTAL=$(awk '/SwapTotal/{print int($2/1024)}' /proc/meminfo)
if [ "${SWAP_MB_TOTAL:-0}" -lt 2000 ]; then
  if [ ! -f /swapfile2 ]; then
    echo "    creating 2GB /swapfile2"
    dd if=/dev/zero of=/swapfile2 bs=1M count=2048 status=none
    chmod 600 /swapfile2
  fi
  if ! swapon --show 2>/dev/null | grep -q '/swapfile2'; then
    # -q is not supported by this util-linux; -f allows re-init of stale files
    mkswap -f /swapfile2 >/dev/null
    swapon /swapfile2
  fi
  grep -qE '^/swapfile2[[:space:]]' /etc/fstab || \
    echo '/swapfile2 none swap defaults 0 0' >> /etc/fstab
fi
free -m | awk 'NR==1||/Mem|Swap/'

echo "==> Static Docker ${DOCKER_VERSION} (tarball) + compose ${COMPOSE_VERSION} (no dnf)"
if ! command -v docker >/dev/null 2>&1; then
  TMPD=$(mktemp -d)
  trap 'rm -rf "$TMPD"' EXIT

  echo "    downloading engine tarball..."
  curl -fL --retry 3 -o "$TMPD/docker.tgz" \
    "https://download.docker.com/linux/static/stable/x86_64/docker-${DOCKER_VERSION}.tgz"
  if [ -n "${DOCKER_TGZ_SHA256:-}" ]; then
    actual="$(sha256sum "$TMPD/docker.tgz" | awk '{print $1}')"
    [ "$actual" = "$DOCKER_TGZ_SHA256" ] || { echo "ERROR: engine tarball checksum mismatch" >&2; exit 1; }
    echo "    engine checksum ok"
  else
    echo "    WARNING: DOCKER_TGZ_SHA256 not set — engine tarball not verified"
  fi
  tar xzf "$TMPD/docker.tgz" -C "$TMPD"
  # static tarball extracts to $TMPD/docker/ (not docker-<version>/)
  cp "$TMPD"/docker/* /usr/local/bin/
  chmod +x /usr/local/bin/docker /usr/local/bin/dockerd /usr/local/bin/containerd \
    /usr/local/bin/containerd-shim-runc-v2 /usr/local/bin/runc /usr/local/bin/ctr \
    /usr/local/bin/docker-init /usr/local/bin/docker-proxy 2>/dev/null || true

  echo "    downloading compose plugin ${COMPOSE_VERSION}..."
  mkdir -p /usr/local/lib/docker/cli-plugins
  curl -fL --retry 3 -o "$TMPD/compose" \
    "https://github.com/docker/compose/releases/download/${COMPOSE_VERSION}/docker-compose-linux-x86_64"
  curl -fL --retry 3 -o "$TMPD/compose.sha256" \
    "https://github.com/docker/compose/releases/download/${COMPOSE_VERSION}/docker-compose-linux-x86_64.sha256"
  expected="$(awk '{print $1}' "$TMPD/compose.sha256")"
  actual="$(sha256sum "$TMPD/compose" | awk '{print $1}')"
  [ "$expected" = "$actual" ] || { echo "ERROR: compose checksum mismatch" >&2; exit 1; }
  echo "    compose checksum ok"
  install -m 755 "$TMPD/compose" /usr/local/lib/docker/cli-plugins/docker-compose

  tee /etc/systemd/system/containerd.service >/dev/null <<'EOF'
[Unit]
Description=containerd container runtime
After=network.target
[Service]
ExecStart=/usr/local/bin/containerd
Restart=always
[Install]
WantedBy=multi-user.target
EOF
  tee /etc/systemd/system/docker.service >/dev/null <<'EOF'
[Unit]
Description=Docker Engine
Requires=containerd.service
After=network.target
[Service]
ExecStart=/usr/local/bin/dockerd
Restart=always
[Install]
WantedBy=multi-user.target
EOF
  systemctl daemon-reload
fi
systemctl enable --now containerd docker
groupadd -f docker
usermod -aG docker opc || true
docker --version && docker compose version

echo "==> Quiet OS background package scans (they freeze 1GB Micros)"
systemctl disable --now dnf-makecache.timer 2>/dev/null || true
systemctl disable --now unbound-anchor.timer 2>/dev/null || true

echo "==> firewalld (role: $ROLE)"
if command -v firewall-cmd >/dev/null 2>&1; then
  systemctl enable --now firewalld 2>/dev/null || true
  if [ "$ROLE" = "app" ]; then
    firewall-cmd --permanent --add-service=http
    firewall-cmd --permanent --add-service=https
  else
    # data-plane: freeze to inside-subnet only (10.0.0.0/24)
    firewall-cmd --permanent --add-rich-rule="rule family=\"ipv4\" source address=\"${SUBNET_CIDR}\" port port=\"5432\" protocol=\"tcp\" accept"
    firewall-cmd --permanent --add-rich-rule="rule family=\"ipv4\" source address=\"${SUBNET_CIDR}\" port port=\"6381\" protocol=\"tcp\" accept"
  fi
  firewall-cmd --reload
else
  echo "    firewalld not installed - skipping (OCI security list still gates traffic)"
fi

echo "==> Done (role: $ROLE). docker group applies on next login."
