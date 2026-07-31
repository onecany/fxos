#!/usr/bin/env bash
# FXOS all-in-one installer (non-Docker): backend systemd service + web frontend via nginx
# Builds the Go backend from source, installs it as a systemd service, then
# builds the React frontend and configures nginx (mandatory HTTPS).
#
# Usage:
#   sudo ./scripts/install-nodocker.sh                  # full install (backend + web)
#   sudo ./scripts/install-nodocker.sh status           # status of fxos + nginx
#   sudo ./scripts/install-nodocker.sh restart          # restart fxos + nginx
#   sudo ./scripts/install-nodocker.sh stop             # stop fxos + nginx
#   sudo ./scripts/install-nodocker.sh logs             # tail backend logs
#   sudo ./scripts/install-nodocker.sh web-install      # (re)build + redeploy frontend only
#   sudo ./scripts/install-nodocker.sh uninstall        # remove services, configs, webroot (keeps data/ and .env)
#
# Environment overrides:
#   INSTALL_DIR         runtime dir (default /opt/fxos) — keep it DIFFERENT from the
#                       source checkout; the installer chowns INSTALL_DIR to fxos user
#   FXOS_BACKEND_PORT   backend API port (default 8080)
#   FXOS_FRONTEND_PORT  public HTTPS port (default 3000)
#   FXOS_SSL_CERT/KEY   custom TLS cert/key (BOTH required together)
#   FXOS_SERVER_IP      fixed server IP for self-signed cert SANs
#   NODE_OPTIONS        respected if already set (low-memory default 1536)

set -euo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log()  { echo -e "${GREEN}[install]${NC} $*"; }
warn() { echo -e "${YELLOW}[install]${NC} $*"; }
err()  { echo -e "${RED}[install]${NC} $*" >&2; }

pkg_install() {
  if command -v apt-get &>/dev/null; then
    apt-get install -y "$@"
  elif command -v dnf &>/dev/null; then
    dnf install -y "$@"
  elif command -v yum &>/dev/null; then
    yum install -y "$@"
  else
    err "No supported package manager found (apt-get/dnf/yum)."
    exit 1
  fi
}

# Map package names across distros (CentOS uses gcc-c++, Debian uses g++)
pkg_cxx() {
  if command -v apt-get &>/dev/null; then
    echo "g++"
  else
    echo "gcc-c++"
  fi
}

if [[ $EUID -ne 0 ]]; then
  err "Must run as root: sudo $0"
  exit 1
fi

INSTALL_DIR="${INSTALL_DIR:-/opt/fxos}"
SCRIPT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
WEB_DIR="$SCRIPT_DIR/web"

# Ports / SSL (respect .env if present, then env, then defaults)
read_env_var() { # $1 var name, $2 default
  local v=""
  if [[ -f "$INSTALL_DIR/.env" ]]; then
    v=$(grep -E "^$1=" "$INSTALL_DIR/.env" 2>/dev/null | tail -1 | cut -d= -f2- | tr -d '"' || true)
  fi
  echo "${v:-$2}"
}
BACKEND_PORT="$(read_env_var FXOS_BACKEND_PORT "${FXOS_BACKEND_PORT:-8080}")"
FRONTEND_PORT="$(read_env_var FXOS_FRONTEND_PORT "${FXOS_FRONTEND_PORT:-3000}")"
SSL_DIR="/etc/fxos/ssl"
SSL_CERT="${FXOS_SSL_CERT:-${SSL_DIR}/fxos.crt}"
SSL_KEY="${FXOS_SSL_KEY:-${SSL_DIR}/fxos.key}"

# ── Sanity: must run from the repository checkout ───────────
if [[ ! -f "$SCRIPT_DIR/go.mod" || ! -d "$WEB_DIR" ]]; then
  err "go.mod / web dir not found next to this script — run it from the repo checkout:"
  err "  sudo ./scripts/install-nodocker.sh"
  exit 1
fi

# ── Conflict warning: install dir == source checkout ────────
if [[ "$INSTALL_DIR" == "$SCRIPT_DIR" ]]; then
  warn "INSTALL_DIR ($INSTALL_DIR) is the same as the source checkout."
  warn "The installer will chown the ENTIRE source tree (including .git) to"
  warn "the fxos user, and the service process will be able to write it."
  warn "Recommended: keep source and runtime separate, e.g."
  warn "  sudo mv $INSTALL_DIR /srv/fxos-src"
  warn "  sudo INSTALL_DIR=$INSTALL_DIR /srv/fxos-src/scripts/install-nodocker.sh"
  warn "Continuing anyway in 5s (Ctrl-C to abort)..."
  sleep 5
fi

# ── Distro-aware nginx config paths ─────────────────────────
if [[ -d /etc/nginx/sites-available ]]; then
  NGINX_CONF="/etc/nginx/sites-available/fxos"
  NGINX_LINK="/etc/nginx/sites-enabled/fxos"
else
  NGINX_CONF="/etc/nginx/conf.d/fxos.conf"
  NGINX_LINK=""
fi

# ════════════════════════════════════════════════════════════
# Web frontend (build + nginx)
# ════════════════════════════════════════════════════════════

WEB_ROOT="/var/www/fxos"

get_server_ip() {
  # Public IP first (so remote access matches the cert SAN), then local.
  local ip=""
  ip=$(curl -s --max-time 3 ifconfig.me 2>/dev/null || curl -s --max-time 3 icanhazip.com 2>/dev/null || echo "")
  if [[ -z "$ip" ]]; then
    if command -v ip &>/dev/null; then
      # "src" field position varies across iproute2 versions — match by name
      ip=$(ip route get 1 2>/dev/null | awk '{for (i=1;i<=NF;i++) if ($i=="src") {print $(i+1); exit}}')
    elif command -v hostname &>/dev/null; then
      ip=$(hostname -I 2>/dev/null | awk '{print $1}')
    fi
  fi
  echo "${ip:-127.0.0.1}"
}

build_cert_sans() {
  local primary_ip="$1"
  local sans="IP:${primary_ip},IP:127.0.0.1,DNS:localhost"
  local extra_ip=""
  extra_ip=$(curl -s --max-time 3 ifconfig.me 2>/dev/null || curl -s --max-time 3 icanhazip.com 2>/dev/null || true)
  if [[ -n "$extra_ip" && "$extra_ip" != "$primary_ip" ]]; then
    sans="${sans},IP:${extra_ip}"
  fi
  echo "$sans"
}

ensure_web_deps() {
  log "Checking web dependencies..."
  local missing=()
  command -v node &>/dev/null || missing+=("nodejs")
  command -v npm &>/dev/null || missing+=("npm")
  command -v nginx &>/dev/null || missing+=("nginx")
  if [[ ${#missing[@]} -gt 0 ]]; then
    log "Installing: ${missing[*]}"
    pkg_install "${missing[@]}"
  fi
  if ! command -v nginx &>/dev/null; then
    err "nginx install failed."
    exit 1
  fi
  if [[ -d /etc/nginx/sites-available ]] && [[ ! -d /etc/nginx/sites-enabled ]]; then
    mkdir -p /etc/nginx/sites-enabled
    if ! grep -q "sites-enabled" /etc/nginx/nginx.conf 2>/dev/null; then
      sed -i '/http {/a\    include /etc/nginx/sites-enabled/*;' /etc/nginx/nginx.conf 2>/dev/null || true
    fi
  fi
  log "  nginx $(nginx -v 2>&1 | cut -d/ -f2) ✓"
}

ensure_node22() {
  if ! command -v node &>/dev/null; then
    err "node not installed."
    exit 1
  fi
  local major
  major=$(node -v | tr -d 'v' | cut -d. -f1)
  if (( major < 22 )); then
    warn "Node $(node -v) found, need 22+ — installing via NodeSource."
    if command -v apt-get &>/dev/null; then
      curl -fsSL https://deb.nodesource.com/setup_22.x | bash -
      apt-get install -y nodejs
    elif command -v dnf &>/dev/null || command -v yum &>/dev/null; then
      curl -fsSL https://rpm.nodesource.com/setup_22.x | bash -
      pkg_install nodejs
    fi
  fi
  log "  Node $(node -v) ✓"
}

build_frontend() {
  log "Building frontend..."
  cd "$WEB_DIR"
  # Respect an existing NODE_OPTIONS; default to a low-memory heap so small
  # VPSes don't OOM. Type-checking (tsc) is skipped here — CI enforces it.
  export NODE_OPTIONS="${NODE_OPTIONS:---max-old-space-size=1536}"
  if [[ ! -d node_modules ]] || [[ package-lock.json -nt node_modules/.package-lock.json ]]; then
    npm ci
  fi
  npx vite build
  if [[ ! -f dist/index.html ]]; then
    err "Build failed (dist/index.html missing)."
    exit 1
  fi
  log "  Build succeeded ✓"
}

deploy_frontend() {
  log "Deploying to ${WEB_ROOT}..."
  mkdir -p "$WEB_ROOT"
  rm -rf "${WEB_ROOT:?}"/*
  cp -r dist/* "$WEB_ROOT/"
  chown -R www-data:www-data "$WEB_ROOT" 2>/dev/null || true
  log "  Deployed ✓"
}

ensure_ssl_certs() {
  log "Setting up mandatory HTTPS certificate for port ${FRONTEND_PORT}..."

  if [[ -n "${FXOS_SSL_CERT:-}" || -n "${FXOS_SSL_KEY:-}" ]]; then
    if [[ -z "${FXOS_SSL_CERT:-}" || -z "${FXOS_SSL_KEY:-}" ]]; then
      err "Set BOTH FXOS_SSL_CERT and FXOS_SSL_KEY when using custom certificates."
      exit 1
    fi
    if [[ ! -f "$SSL_CERT" || ! -f "$SSL_KEY" ]]; then
      err "Custom SSL file(s) not found: cert=${SSL_CERT} key=${SSL_KEY}"
      exit 1
    fi
    if command -v openssl &>/dev/null; then
      local cm km
      cm=$(openssl x509 -noout -modulus -in "$SSL_CERT" 2>/dev/null | openssl md5 2>/dev/null || echo "_cert_")
      km=$(openssl rsa  -noout -modulus -in "$SSL_KEY"  2>/dev/null | openssl md5 2>/dev/null || echo "_key_")
      if [[ "$cm" != "$km" ]]; then
        err "Custom SSL certificate and private key DO NOT match (modulus mismatch)."
        exit 1
      fi
    fi
    log "  Using custom SSL cert: ${SSL_CERT} ✓"
    return
  fi

  if ! command -v openssl &>/dev/null; then
    log "Installing openssl..."
    pkg_install openssl
  fi

  if [[ -f "$SSL_CERT" && -f "$SSL_KEY" ]]; then
    log "  Using existing SSL cert: ${SSL_CERT} ✓"
    return
  fi

  if [[ -f "$SSL_CERT" || -f "$SSL_KEY" ]]; then
    warn "Incomplete SSL files in ${SSL_DIR}, regenerating..."
    rm -f "$SSL_CERT" "$SSL_KEY"
  fi

  # FXOS_SERVER_IP: .env (backend) override, then env, then auto-detect
  local server_ip=""
  server_ip="$(read_env_var FXOS_SERVER_IP "${FXOS_SERVER_IP:-}")"
  if [[ -z "$server_ip" ]]; then
    server_ip="$(get_server_ip)"
  else
    log "  FXOS_SERVER_IP set: ${server_ip}"
  fi
  local cert_sans
  cert_sans="$(build_cert_sans "$server_ip")"

  mkdir -p "$SSL_DIR" 2>/dev/null || true
  chmod 700 "$SSL_DIR" 2>/dev/null || true

  local gen_ok=false
  log "Generating self-signed certificate for IP access (${server_ip})..."
  if openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
    -keyout "$SSL_KEY" -out "$SSL_CERT" \
    -subj "/CN=${server_ip}" \
    -addext "subjectAltName=${cert_sans}" >/dev/null 2>&1; then
    gen_ok=true
  else
    warn "OpenSSL lacks -addext, retrying without SANs..."
    rm -f "$SSL_CERT" "$SSL_KEY" 2>/dev/null || true
    if openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
      -keyout "$SSL_KEY" -out "$SSL_CERT" \
      -subj "/CN=${server_ip}" >/dev/null 2>&1; then
      gen_ok=true
    fi
  fi

  if [[ "$gen_ok" != "true" ]]; then
    err "Failed to generate self-signed SSL certificate."
    exit 1
  fi

  chmod 600 "$SSL_KEY"  2>/dev/null || true
  chmod 644 "$SSL_CERT" 2>/dev/null || true
  warn "Self-signed cert created. Browser will warn on first visit — click Advanced → Continue."
}

write_nginx_conf() {
  cat > "$NGINX_CONF" <<NGINX
# ─────────────────────────────────────────────────────────────
# FXOS Frontend (bare-metal systemd / nginx install)
# HTTPS is MANDATORY on port ${FRONTEND_PORT}.
# Even plaintext http://IP:${FRONTEND_PORT} is auto-upgraded to https://.
# ─────────────────────────────────────────────────────────────

# (A) Internal 127.0.0.1:80 loopback-only fallback — never exposed publicly.
server {
    listen 127.0.0.1:80;
    listen [::1]:80;
    server_name _;
    return 301 https://\$host:${FRONTEND_PORT}\$request_uri;
    access_log off;
}

# (B) ONLY public-facing listener: TLS on ${FRONTEND_PORT}
server {
    listen ${FRONTEND_PORT} ssl;
    listen [::]:${FRONTEND_PORT} ssl;
    server_name _;

    ssl_certificate ${SSL_CERT};
    ssl_certificate_key ${SSL_KEY};
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_prefer_server_ciphers on;
    ssl_ciphers ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA256:ECDHE-ECDSA-AES256-GCM-SHA384:ECDHE-RSA-AES256-GCM-SHA384;

    # Same-port HTTP → HTTPS upgrade (nginx error 497 on plaintext→SSL listener)
    error_page 497 = @upgrade_to_https;
    location @upgrade_to_https {
        return 301 https://\$http_host\$request_uri;
    }

    add_header Strict-Transport-Security "max-age=63072000; includeSubDomains" always;

    root ${WEB_ROOT};
    index index.html;

    gzip on;
    gzip_vary on;
    gzip_min_length 1024;
    gzip_types text/plain text/css text/xml text/javascript application/javascript application/json;

    location = /index.html {
        add_header Cache-Control "no-cache, no-store, must-revalidate";
        add_header Pragma "no-cache";
        add_header Expires 0;
    }

    location / {
        try_files \$uri \$uri/ /index.html;

        location ~* \\.(js|css|png|jpg|jpeg|gif|ico|svg|woff|woff2|ttf|eot)$ {
            expires 1y;
            add_header Cache-Control "public, immutable";
        }
    }

    # API reverse proxy to backend on 127.0.0.1
    location /api/ {
        proxy_pass http://127.0.0.1:${BACKEND_PORT}/api/;
        proxy_http_version 1.1;
        proxy_set_header Upgrade \$http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host \$host;
        proxy_cache_bypass \$http_upgrade;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto https;

        proxy_connect_timeout 300s;
        proxy_send_timeout 300s;
        proxy_read_timeout 300s;
    }

    location /health {
        return 200 "OK\n";
        add_header Content-Type text/plain;
        access_log off;
    }
}
NGINX

  if [[ -n "$NGINX_LINK" ]]; then
    ln -sf "$NGINX_CONF" "$NGINX_LINK"
    if [[ -f /etc/nginx/sites-enabled/default ]]; then
      rm -f /etc/nginx/sites-enabled/default
    fi
  fi

  if ! nginx -t 2>&1; then
    err "Nginx config test failed."
    exit 1
  fi
  log "  Nginx configured ✓"
}

start_nginx() {
  log "Starting nginx..."
  systemctl enable nginx 2>/dev/null || true
  systemctl restart nginx

  sleep 1
  if ! systemctl is-active --quiet nginx; then
    err "Nginx failed to start. Check: journalctl -u nginx -n 20"
    exit 1
  fi
  if curl -sfk --max-time 3 "https://127.0.0.1:${FRONTEND_PORT}/health" >/dev/null; then
    log "  HTTPS health probe OK ✓"
  else
    warn "  Could not probe https://127.0.0.1:${FRONTEND_PORT}/health — check firewall."
  fi
}

install_web() {
  ensure_web_deps
  ensure_node22
  build_frontend
  deploy_frontend
  ensure_ssl_certs
  write_nginx_conf
  start_nginx
}

# ════════════════════════════════════════════════════════════
# Sub-commands
# ════════════════════════════════════════════════════════════

cmd="${1:-install}"
case "$cmd" in
  status)
    systemctl status fxos --no-pager || true
    echo ""
    systemctl status nginx --no-pager || true
    exit 0
    ;;
  restart)
    systemctl restart fxos
    systemctl restart nginx
    log "fxos + nginx restarted."
    exit 0
    ;;
  stop)
    systemctl stop fxos
    systemctl stop nginx
    log "fxos + nginx stopped."
    exit 0
    ;;
  logs)
    journalctl -u fxos -f --tail=50
    exit 0
    ;;
  web-install)
    install_web
    log "Web frontend (re)deployed."
    exit 0
    ;;
  uninstall)
    log "Removing FXOS services and configs (keeping $INSTALL_DIR/data and .env)..."
    systemctl stop fxos 2>/dev/null || true
    systemctl disable fxos 2>/dev/null || true
    rm -f /etc/systemd/system/fxos.service
    systemctl daemon-reload
    rm -f "$NGINX_CONF" "$NGINX_LINK"
    rm -rf "$WEB_ROOT" "$SSL_DIR"
    systemctl restart nginx 2>/dev/null || true
    log "Uninstalled. To remove data too: rm -rf $INSTALL_DIR"
    exit 0
    ;;
  install|"") ;;
  *)
    err "Unknown command: $cmd"
    err "Usage: sudo $0 [install|status|restart|stop|logs|web-install|uninstall]"
    exit 1
    ;;
esac

# ════════════════════════════════════════════════════════════
# Backend install
# ════════════════════════════════════════════════════════════

log "Checking dependencies..."

missing=()
command -v go &>/dev/null || missing+=("go")
command -v git &>/dev/null || missing+=("git")
command -v make &>/dev/null || missing+=("make")
command -v gcc &>/dev/null || missing+=("gcc")
command -v g++ &>/dev/null || missing+=("$(pkg_cxx)")
command -v curl &>/dev/null || missing+=("curl")
command -v openssl &>/dev/null || missing+=("openssl")

if [[ ${#missing[@]} -gt 0 ]]; then
  log "Installing missing dependencies: ${missing[*]}"
  pkg_install "${missing[@]}"
fi

# Check Go version >= 1.26 (go.mod requires 1.26.5)
GO_VER=$(go version | awk '{print $3}' | sed 's/^go//' | cut -d. -f1-2)
GO_MAJOR="${GO_VER%%.*}"
GO_MINOR="${GO_VER##*.}"
if (( GO_MAJOR < 1 || (GO_MAJOR == 1 && GO_MINOR < 26) )); then
  err "Go 1.26+ required, found $GO_VER"
  exit 1
fi
log "  Go $GO_VER ✓"

# ── Build binary ────────────────────────────────────────────
log "Building fxos binary..."
cd "$SCRIPT_DIR"
rm -f fxos
CGO_ENABLED=1 GOOS=linux go build -trimpath -ldflags="-s -w" -o fxos .

if [[ ! -x fxos ]]; then
  err "Build failed."
  exit 1
fi
log "  Build succeeded ✓"

# ── Create service user ─────────────────────────────────────
if ! id -u fxos &>/dev/null; then
  log "Creating fxos user..."
  useradd --system --no-create-home --shell /usr/sbin/nologin fxos
fi

# ── Install binary ──────────────────────────────────────────
log "Installing to $INSTALL_DIR..."
mkdir -p "$INSTALL_DIR/data"
cp "$SCRIPT_DIR/fxos" "$INSTALL_DIR/fxos"
chmod 755 "$INSTALL_DIR/fxos"

# ── Ensure .env exists (JWT/DATA_ENCRYPTION/RSA required) ──
if [[ ! -f "$INSTALL_DIR/.env" ]]; then
  if [[ -f "$SCRIPT_DIR/.env" ]]; then
    cp "$SCRIPT_DIR/.env" "$INSTALL_DIR/.env"
    chmod 600 "$INSTALL_DIR/.env"
    log "Copied .env to $INSTALL_DIR/"
  else
    warn "No .env found in the repo — generating one with fresh keys."
    JWT_SECRET=$(openssl rand -base64 32)
    DATA_ENCRYPTION_KEY=$(openssl rand -base64 32)
    # Single-line PEM with literal \n escapes — matches the backend contract
    RSA_PRIVATE_KEY=$(openssl genrsa 2048 2>/dev/null | tr '\n' '\\' | sed 's/\\/\\n/g' | sed 's/\\n$//')
    cat > "$INSTALL_DIR/.env" <<EOF
# FXOS Configuration (Auto-generated by install-nodocker.sh)
JWT_SECRET=${JWT_SECRET}
DATA_ENCRYPTION_KEY=${DATA_ENCRYPTION_KEY}
RSA_PRIVATE_KEY=${RSA_PRIVATE_KEY}
TZ=Asia/Shanghai
FXOS_BACKEND_PORT=${BACKEND_PORT}
FXOS_FRONTEND_PORT=${FRONTEND_PORT}
EOF
    chmod 600 "$INSTALL_DIR/.env"
    log "  Generated $INSTALL_DIR/.env (JWT_SECRET, DATA_ENCRYPTION_KEY, RSA_PRIVATE_KEY)"
    warn "  Review it and rotate keys before exposing this server publicly."
  fi
else
  log "  .env already exists at $INSTALL_DIR/.env, keeping it"
fi

# ── Set ownership ───────────────────────────────────────────
chown -R fxos:fxos "$INSTALL_DIR"
chmod 770 "$INSTALL_DIR/data"

# ── Detect stale database: single-user system locks registration ──
# If an existing data.db already has users, the new install can never
# register/initialize. Surface this loudly instead of silently shipping
# a "registration closed" deployment.
check_stale_db() {
  local db_file="$INSTALL_DIR/data/data.db"
  [[ -f "$db_file" ]] || return 0
  if ! command -v sqlite3 &>/dev/null; then
    warn "data.db already exists at $db_file — if it contains users,"
    warn "  registration is CLOSED (single-user system)."
    warn "  To re-initialize: sudo -u fxos $INSTALL_DIR/fxos reset-account --yes"
    return 0
  fi
  local user_count
  user_count=$(sqlite3 "$db_file" "SELECT COUNT(*) FROM users;" 2>/dev/null || echo "")
  if [[ "$user_count" =~ ^[0-9]+$ ]] && (( user_count > 0 )); then
    err "data.db at $db_file already has $user_count user(s)."
    err "FXOS is a single-user system: registration stays CLOSED until reset."
    err "Re-initialize with: sudo -u fxos $INSTALL_DIR/fxos reset-account --yes"
    err "  (wipes ALL users/traders/strategies/exchange credentials — back up first)"
    exit 1
  fi
}
check_stale_db

# ── Install systemd unit (paths substituted for custom INSTALL_DIR) ──
log "Installing systemd service..."
sed "s|/opt/fxos|${INSTALL_DIR}|g" "$SCRIPT_DIR/scripts/fxos.service" > /etc/systemd/system/fxos.service
systemctl daemon-reload

# ── Enable and start ────────────────────────────────────────
systemctl enable fxos.service
systemctl start fxos.service

ok=0
for _ in $(seq 1 15); do
  if curl -fsS --max-time 2 "http://127.0.0.1:${BACKEND_PORT}/api/health" >/dev/null 2>&1; then
    ok=1
    break
  fi
  sleep 1
done

if [[ "$ok" -eq 1 ]]; then
  log "FXOS backend is running (health check OK)."
else
  err "Backend not healthy. Check: journalctl -u fxos -n 50"
  systemctl status fxos.service --no-pager || true
  exit 1
fi

# ── Web frontend ────────────────────────────────────────────
install_web

# ── Firewall hint ───────────────────────────────────────────
log ""
log "Security note: the backend listens on :${BACKEND_PORT} (all interfaces)."
log "If a firewall is active, only these inbound ports should be open:"
log "  ${FRONTEND_PORT}/tcp (web UI, HTTPS) and 22/tcp (SSH)."
warn "  Example (ufw): sudo ufw allow 22/tcp && sudo ufw allow ${FRONTEND_PORT}/tcp && sudo ufw --force enable"

# ── Done ────────────────────────────────────────────────────
SERVER_IP="$(get_server_ip)"
log ""
log "════════════════════════════════════════════════════════════"
log "  🎉 FXOS installed (backend service + web frontend)"
log "════════════════════════════════════════════════════════════"
log ""
log "  Access URLs:"
log "    Remote: https://${SERVER_IP}:${FRONTEND_PORT}"
log "    Local:  https://127.0.0.1:${FRONTEND_PORT}"
log ""
log "  ℹ️  http://IP:${FRONTEND_PORT} auto-redirects to https:// (same port)."
log "  API proxy: nginx forwards /api/ → http://127.0.0.1:${BACKEND_PORT}/api/"
log ""
if [[ -n "${FXOS_SSL_CERT:-}" ]]; then
  log "  🔒 TLS: custom certificate (${FXOS_SSL_CERT})"
else
  log "  🔒 TLS: self-signed certificate — FIRST VISIT browser warning expected."
fi
log ""
log "  Useful commands:"
log "    sudo systemctl status fxos      # backend status"
log "    sudo $0 restart                 # restart backend + nginx"
log "    sudo $0 logs                    # tail backend logs"
log "    sudo $0 web-install             # rebuild + redeploy frontend"
log "    sudo $0 uninstall               # remove services + configs (keeps data)"
log ""
