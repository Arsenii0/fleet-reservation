#!/usr/bin/env bash
# ─────────────────────────────────────────────────────────────────────────────
# OpenClaw bootstrap — Ubuntu 24.04 LTS
set -euo pipefail

# ─────────────────────────────────────────────────────────────────────────────
harden_os() {
  echo "=== [1/3] Security hardening ==="

  # Non-interactive upgrades; upgrade before installing anything new
  export DEBIAN_FRONTEND=noninteractive
  apt-get update -y
  apt-get upgrade -y --no-install-recommends

  apt-get install -y --no-install-recommends \
    ca-certificates curl gnupg unzip jq ufw snapd

  # Install SSM agent via snap (amazon-ssm-agent is not in Ubuntu 24.04 apt repos)
  snap install amazon-ssm-agent --classic
  systemctl enable snap.amazon-ssm-agent.amazon-ssm-agent
  systemctl start  snap.amazon-ssm-agent.amazon-ssm-agent

  # Disable root SSH login (belt-and-suspenders; port 22 is not open anyway)
  sed -i 's/^PermitRootLogin.*/PermitRootLogin no/'  /etc/ssh/sshd_config 2>/dev/null || true
  sed -i 's/^#PermitRootLogin.*/PermitRootLogin no/' /etc/ssh/sshd_config 2>/dev/null || true

  # Firewall: drop all inbound; outbound is unrestricted.
  # The gateway (port 18789) stays on loopback – reachable only via SSM port-forward.
  ufw --force reset
  ufw default deny incoming
  ufw default allow outgoing
  ufw allow from ${vnc_allowed_cidr} to any port 5901 proto tcp comment "VNC"
  ufw --force enable
}

# ─────────────────────────────────────────────────────────────────────────────
install_openclaw() {
  echo "=== [2/3] Installing OpenClaw ==="

  echo "--- Installing Node.js 22 LTS ---"
  curl -fsSL https://deb.nodesource.com/setup_lts.x | bash -
  apt-get install -y nodejs
  node --version
  npm --version

  echo "--- Installing openclaw@${openclaw_version} ---"
  npm install -g "openclaw@${openclaw_version}"
  openclaw --version || true
}

# ─────────────────────────────────────────────────────────────────────────────
# XFCE is used instead of full GNOME — much lighter, fine on t3.medium.
# TigerVNC is the server; connects on port 5901 (display :1).
setup_vnc() {
  echo "=== [3/3] Setting up desktop + VNC ==="

  # Snap-based Firefox does not work inside a VNC session (no systemd user session).
  echo "--- Installing XFCE desktop + TigerVNC + Firefox ---"
  install -d -m 0755 /etc/apt/keyrings
  curl -fsSL https://packages.mozilla.org/apt/repo-signing-key.gpg \
    | tee /etc/apt/keyrings/packages.mozilla.org.asc > /dev/null
  echo "deb [signed-by=/etc/apt/keyrings/packages.mozilla.org.asc] https://packages.mozilla.org/apt mozilla main" \
    > /etc/apt/sources.list.d/mozilla.list
  printf 'Package: *\nPin: origin packages.mozilla.org\nPin-Priority: 1000\n' \
    > /etc/apt/preferences.d/mozilla
  apt-get update -y

  apt-get install -y \
    xubuntu-desktop \
    tigervnc-standalone-server \
    tigervnc-common \
    dbus-x11 \
    xterm \
    firefox

  echo "--- Configuring TigerVNC for ubuntu user ---"
  local vnc_user=ubuntu
  local vnc_home=/home/$vnc_user

  mkdir -p "$vnc_home/.vnc"

  # Set VNC password non-interactively (-f writes the binary passwd file to stdout)
  printf '%s' "${vnc_password}" | vncpasswd -f > "$vnc_home/.vnc/passwd"
  chmod 600 "$vnc_home/.vnc/passwd"

  # xstartup: launch XFCE session
  cat > "$vnc_home/.vnc/xstartup" <<'XSTARTUP'
#!/bin/bash
unset SESSION_MANAGER
unset DBUS_SESSION_BUS_ADDRESS
exec startxfce4
XSTARTUP
  chmod +x "$vnc_home/.vnc/xstartup"

  chown -R "$vnc_user:$vnc_user" "$vnc_home/.vnc"

  # Systemd service — runs VNC on display :1 (port 5901) as the ubuntu user
  cat > /etc/systemd/system/vncserver@.service <<'VNCSERVICE'
[Unit]
Description=TigerVNC server on display %i
After=network.target syslog.target

[Service]
Type=forking
User=ubuntu
Group=ubuntu
WorkingDirectory=/home/ubuntu

ExecStartPre=-/usr/bin/vncserver -kill :%i > /dev/null 2>&1
ExecStart=/usr/bin/vncserver :%i -geometry 1920x1080 -depth 24 -localhost no
ExecStop=/usr/bin/vncserver -kill :%i

Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
VNCSERVICE

  systemctl daemon-reload
  systemctl enable vncserver@1
  systemctl start  vncserver@1
}

# ─────────────────────────────────────────────────────────────────────────────
main() {
  harden_os
  install_openclaw
  setup_vnc

  echo "=== Bootstrap complete ==="
  echo ""
  echo "VNC connect: <elastic-ip>:5901"
  echo ""
  echo "SSM shell:   aws ssm start-session --target INSTANCE_ID --profile personal"
}

main
