#!/usr/bin/env bash
# Oracle Cloud 1GB-Micro bootstrap — NO dnf transactions (they OOM-storm on
# free-tier Micros). Roles: VM-1=data, VM-2=app.
# Installs: swap iff <2GB, static Docker + compose plugin, firewalld rules.
# Usage: sudo bash bootstrap.sh data|app     (SUBNET_CIDR override optional)
set -uo pipefail
ROLE="${1:?usage: bootstrap.sh data|app}"
SUBNET_CIDR="${SUBNET_CIDR:-10.0.0.0/24}"

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

echo "==> Static Docker (tarball) + compose plugin (no dnf)"
if ! command -v docker >/dev/null 2>&1; then
  TMPD=$(mktemp -d)
  echo "    downloading engine tarball..."
  curl -fL --retry 3 -o "$TMPD/docker.tgz" https://download.docker.com/linux/static/stable/x86_64/docker-29.7.2.tgz
  tar xzf "$TMPD/docker.tgz" -C "$TMPD"
  # static tarball extracts to $TMPD/docker/ (not docker-<version>/)
  cp "$TMPD"/docker/* /usr/local/bin/
  chmod +x /usr/local/bin/docker /usr/local/bin/dockerd /usr/local/bin/containerd \
    /usr/local/bin/containerd-shim-runc-v2 /usr/local/bin/runc /usr/local/bin/ctr \
    /usr/local/bin/docker-init /usr/local/bin/docker-proxy 2>/dev/null || true
  mkdir -p /usr/local/lib/docker/cli-plugins
  echo "    downloading compose plugin..."
  curl -fL --retry 3 -o /usr/local/lib/docker/cli-plugins/docker-compose \
    https://github.com/docker/compose/releases/latest/download/docker-compose-linux-x86_64
  chmod +x /usr/local/lib/docker/cli-plugins/docker-compose
  rm -rf "$TMPD"
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
