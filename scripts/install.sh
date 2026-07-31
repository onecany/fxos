#!/bin/bash
#
# FXOS One-Click Installation Script
# https://github.com/onecany/fxos
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/onecany/fxos/dev/scripts/install.sh | bash
#
# Or with custom directory:
#   curl -fsSL https://raw.githubusercontent.com/onecany/fxos/dev/scripts/install.sh | bash -s -- /opt/fxos
#
# Overrides (environment variables):
#   FXOS_GITHUB_RAW     Base URL to fetch deployment files from.
#                       Default: https://raw.githubusercontent.com/onecany/fxos/dev
#   FXOS_COMPOSE_FILE   Compose file path (relative to GITHUB_RAW).
#                       Default: scripts/docker/docker-compose.prod.yml
#

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Default installation directory
INSTALL_DIR="${1:-$HOME/fxos}"
COMPOSE_FILE="${FXOS_COMPOSE_FILE:-scripts/docker/docker-compose.prod.yml}"
GITHUB_RAW="${FXOS_GITHUB_RAW:-https://raw.githubusercontent.com/onecany/fxos/dev}"

echo -e "${BLUE}"
echo "╔════════════════════════════════════════════════════════════╗"
echo "║                    FXOS AI Trading OS                      ║"
echo "║                   One-Click Installation                   ║"
echo "╚════════════════════════════════════════════════════════════╝"
echo -e "${NC}"

# Check prerequisites (curl) and Docker
check_prereqs() {
    echo -e "${YELLOW}Checking prerequisites...${NC}"
    if ! command -v curl &> /dev/null; then
        echo -e "${RED}Error: curl is not installed.${NC}"
        echo "Please install curl first: https://curl.se/download.html"
        exit 1
    fi

    echo -e "${YELLOW}Checking Docker...${NC}"
    if ! command -v docker &> /dev/null; then
        echo -e "${RED}Error: Docker is not installed.${NC}"
        echo "Please install Docker first: https://docs.docker.com/get-docker/"
        exit 1
    fi

    if ! docker info &> /dev/null; then
        echo -e "${RED}Error: Docker daemon is not running.${NC}"
        echo "Please start Docker and try again."
        exit 1
    fi

    # Check Docker Compose
    if docker compose version &> /dev/null; then
        COMPOSE_CMD="docker compose"
    elif command -v docker-compose &> /dev/null; then
        COMPOSE_CMD="docker-compose"
    else
        echo -e "${RED}Error: Docker Compose is not available.${NC}"
        echo "Please install Docker Compose: https://docs.docker.com/compose/install/"
        exit 1
    fi

    echo -e "${GREEN}✓ Docker is ready${NC}"
}

# Create installation directory
setup_directory() {
    echo -e "${YELLOW}Setting up installation directory: ${INSTALL_DIR}${NC}"
    mkdir -p "$INSTALL_DIR"
    cd "$INSTALL_DIR"
    echo -e "${GREEN}✓ Directory ready${NC}"
}

# Download compose file
download_files() {
    echo -e "${YELLOW}Downloading configuration files...${NC}"
    local url="$GITHUB_RAW/$COMPOSE_FILE"
    if ! curl -fsSL --connect-timeout 10 --max-time 60 "$url" -o docker-compose.yml; then
        echo -e "${RED}✗ Failed to download ${COMPOSE_FILE}${NC}"
        echo -e "${RED}  URL: ${url}${NC}"
        echo -e "${YELLOW}  A 404 usually means the deployment file (or the branch it lives on)"
        echo -e "${YELLOW}  is not published in the GitHub repo yet.${NC}"
        echo -e "${YELLOW}  Override the source with environment variables, e.g.:${NC}"
        echo -e "${BLUE}    FXOS_GITHUB_RAW=https://raw.githubusercontent.com/onecany/fxos/dev \\"
        echo -e "${BLUE}    FXOS_COMPOSE_FILE=scripts/docker/docker-compose.prod.yml \\"
        echo -e "${BLUE}    bash install.sh${NC}"
        exit 1
    fi
    if [ ! -s docker-compose.yml ]; then
        echo -e "${RED}✗ Downloaded file is empty: docker-compose.yml${NC}"
        exit 1
    fi
    echo -e "${GREEN}✓ Files downloaded${NC}"
}

# Generate encryption keys and create .env file
generate_env() {
    echo -e "${YELLOW}Generating encryption keys...${NC}"

    # Skip if .env already exists
    if [ -f ".env" ]; then
        echo -e "${GREEN}✓ .env file already exists, skipping key generation${NC}"
        return
    fi

    # Generate JWT secret (32 bytes, base64)
    JWT_SECRET=$(openssl rand -base64 32)

    # Generate AES data encryption key (32 bytes, base64)
    DATA_ENCRYPTION_KEY=$(openssl rand -base64 32)

    # Generate RSA private key (2048 bits)
    RSA_PRIVATE_KEY=$(openssl genrsa 2048 2>/dev/null | tr '\n' '\\' | sed 's/\\/\\n/g' | sed 's/\\n$//')

    # Create .env file
    cat > .env << EOF
# FXOS Configuration (Auto-generated)
# Generated at: $(date -u +"%Y-%m-%dT%H:%M:%SZ")

# Server ports
# Frontend is HTTPS-only — host ${FXOS_FRONTEND_PORT:-3000} maps to container 443 (TLS)
# Even if you type http://IP:3000, it auto-redirects to https:// on the same port.
FXOS_BACKEND_PORT=8080
FXOS_FRONTEND_PORT=3000

# Timezone
TZ=Asia/Shanghai

# JWT signing secret
JWT_SECRET=${JWT_SECRET}

# AES-256 data encryption key (for encrypting API keys in database)
DATA_ENCRYPTION_KEY=${DATA_ENCRYPTION_KEY}

# RSA private key (for client-server encryption)
RSA_PRIVATE_KEY=${RSA_PRIVATE_KEY}

# ── SSL / HTTPS ──────────────────────────────────────────────────────
# HTTPS is MANDATORY on the frontend port. A self-signed certificate
# with the correct server IP SANs is auto-generated below on first run.
# The self-signed cert triggers a one-time browser warning — click
# "Advanced → Continue" to proceed (see docs/guides/SSL-HTTPS.zh-CN.md).
FXOS_ENABLE_SSL=true

# Optional: use your own trusted CA certificate (e.g. Let's Encrypt)
# instead of the auto-generated self-signed cert. BOTH must be set.
# FXOS_SSL_CERT=/etc/letsencrypt/live/your-domain.com/fullchain.pem
# FXOS_SSL_KEY=/etc/letsencrypt/live/your-domain.com/privkey.pem

# Optional: manually override the server IP written into the
# self-signed cert Subject Alternative Names. Useful on multi-homed
# servers or when auto-detection picks the wrong interface.
# FXOS_SERVER_IP=1.2.3.4
EOF

    # Protect the generated secrets (private keys live in this file)
    chmod 600 .env

    echo -e "${GREEN}✓ Encryption keys generated${NC}"
}

# Read a single var from the just-generated .env file (install.sh-specific — no full read_env_vars)
read_install_env() {
    local var_name="$1" default_val="${2:-}"
    if [ -f ".env" ]; then
        local val
        val=$(grep "^${var_name}=" .env 2>/dev/null | cut -d'=' -f2- || true)
        # Strip only the surrounding whitespace / wrapping quotes; keep the interior intact.
        # (The old tr-based version deleted *every* quote and space character in the value.)
        val=${val#"${val%%[![:space:]]*}"}
        val=${val%"${val##*[![:space:]]}"}
        val=${val#\"}; val=${val%\"}
        val=${val#\'}; val=${val%\'}
        if [ -n "$val" ]; then
            echo "$val"
            return
        fi
    fi
    echo "$default_val"
}

# Verify cert/key modulus match before use (prevent nginx boot crash)
verify_cert_key_match_install() {
    local cert="$1" key="$2"
    if ! command -v openssl &> /dev/null; then return 0; fi
    local cm km
    cm=$(openssl x509 -noout -modulus -in "$cert" 2>/dev/null | openssl md5 2>/dev/null || echo "c")
    km=$(openssl rsa  -noout -modulus -in "$key"  2>/dev/null | openssl md5 2>/dev/null || echo "k")
    [ "$cm" = "$km" ]
}

# Get server IP — FXOS_SERVER_IP override in .env wins, then public IP
# (ifconfig.me / icanhazip.com), then local IP. Used for SSL cert SANs
# and for the final "open this URL" message.
get_server_ip() {
    local override_ip=""
    override_ip=$(read_install_env "FXOS_SERVER_IP" "")
    if [ -n "$override_ip" ]; then
        echo "$override_ip"
        return
    fi
    local public_ip=""
    public_ip=$(curl -s --max-time 3 ifconfig.me 2>/dev/null || curl -s --max-time 3 icanhazip.com 2>/dev/null || echo "")
    if [ -z "$public_ip" ]; then
        if command -v ip &> /dev/null; then
            # The "src" field position varies across iproute2 versions — match it by name
            public_ip=$(ip route get 1 2>/dev/null | awk '{for (i=1;i<=NF;i++) if ($i=="src") {print $(i+1); exit}}')
        elif command -v hostname &> /dev/null; then
            public_ip=$(hostname -I 2>/dev/null | awk '{print $1}')
        fi
    fi
    echo "${public_ip:-127.0.0.1}"
}

# Generate self-signed SSL certificate for HTTPS access
# NOTE: HTTPS is mandatory on ${FXOS_FRONTEND_PORT:-3000}; we always ensure
# certs exist, even if the user accidentally set FXOS_ENABLE_SSL=false.
generate_ssl_cert() {
    local custom_cert="" custom_key=""
    custom_cert=$(read_install_env "FXOS_SSL_CERT" "")
    custom_key=$(read_install_env  "FXOS_SSL_KEY"  "")

    echo -e "${YELLOW}Setting up SSL certificate for mandatory HTTPS (port 3000)...${NC}"

    local SSL_DIR="./ssl"
    local CERT="${SSL_DIR}/fxos.crt"
    local KEY="${SSL_DIR}/fxos.key"
    mkdir -p "${SSL_DIR}" 2>/dev/null || true
    chmod 700 "${SSL_DIR}" 2>/dev/null || true

    # ── Handle custom cert/key from .env if provided ────────────────
    if [ -n "$custom_cert" ] || [ -n "$custom_key" ]; then
        if [ -z "$custom_cert" ] || [ -z "$custom_key" ]; then
            echo -e "${YELLOW}⚠️  Only one of FXOS_SSL_CERT/FXOS_SSL_KEY is set in .env — both required. Skipping custom cert.${NC}"
        elif [ ! -f "$custom_cert" ] || [ ! -f "$custom_key" ]; then
            echo -e "${YELLOW}⚠️  Custom SSL cert/key file(s) not found on disk. Skipping.${NC}"
            [ ! -f "$custom_cert" ] && echo "   Missing CERT: $custom_cert"
            [ ! -f "$custom_key"  ] && echo "   Missing KEY:  $custom_key"
        elif ! verify_cert_key_match_install "$custom_cert" "$custom_key"; then
            echo -e "${RED}✗ Custom SSL certificate and private key do NOT match. Skipping (would crash nginx).${NC}"
            echo "   CERT: $custom_cert"
            echo "   KEY:  $custom_key"
        else
            # Copy to ./ssl/ — set -e safe with explicit error branch
            if cp -f "$custom_cert" "$CERT" 2>/dev/null && \
               cp -f "$custom_key"  "$KEY"  2>/dev/null; then
                chmod 600 "${KEY}"  2>/dev/null || true
                chmod 644 "${CERT}" 2>/dev/null || true
                echo -e "${GREEN}✓ Using custom SSL certificate: ${custom_cert}${NC}"
                return
            else
                echo -e "${YELLOW}⚠️  Failed to copy custom SSL cert/key into $SSL_DIR (permission?). Falling back.${NC}"
                rm -f "$CERT" "$KEY" 2>/dev/null || true
            fi
        fi
    fi

    # ── Skip if we already have usable files in ./ssl/ ──────────────
    if [ -f "${CERT}" ] && [ -f "${KEY}" ]; then
        echo -e "${GREEN}✓ SSL certificate already exists at ${CERT}${NC}"
        return
    fi

    if ! command -v openssl &> /dev/null; then
        echo -e "${YELLOW}⚠️  OpenSSL not found on host. Frontend container entrypoint will generate a cert at runtime.${NC}"
        return
    fi

    # Clean up any half-finished files from previous runs
    rm -f "$CERT" "$KEY" 2>/dev/null || true

    local SERVER_IP
    SERVER_IP=$(get_server_ip)

    local SANS="IP:${SERVER_IP},IP:127.0.0.1,DNS:localhost"
    local EXTERNAL_IP=""
    EXTERNAL_IP=$(curl -s --max-time 3 ifconfig.me 2>/dev/null || curl -s --max-time 3 icanhazip.com 2>/dev/null || true)
    if [ -n "${EXTERNAL_IP}" ] && [ "${EXTERNAL_IP}" != "${SERVER_IP}" ]; then
        SANS="${SANS},IP:${EXTERNAL_IP}"
    fi

    local gen_ok=false
    # Try with SANs first (openssl ≥ 1.1.1) — all wrapped so set -e won't abort
    if openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
        -keyout "${KEY}" -out "${CERT}" \
        -subj "/CN=${SERVER_IP}" \
        -addext "subjectAltName=${SANS}" >/dev/null 2>&1; then
        gen_ok=true
    else
        rm -f "$CERT" "$KEY" 2>/dev/null || true
        # Fallback without SANs — wrapped in if-block so set -e cannot kill us
        echo -e "${YELLOW}ℹ️  OpenSSL lacks -addext, retrying without SANs...${NC}"
        if openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
            -keyout "${KEY}" -out "${CERT}" \
            -subj "/CN=${SERVER_IP}" >/dev/null 2>&1; then
            gen_ok=true
        fi
    fi

    if [ "$gen_ok" != "true" ]; then
        echo -e "${YELLOW}⚠️  Failed to generate host SSL cert. Container entrypoint will create one at runtime.${NC}"
        rm -f "$CERT" "$KEY" 2>/dev/null || true
        return
    fi

    chmod 600 "${KEY}"  2>/dev/null || true
    chmod 644 "${CERT}" 2>/dev/null || true

    echo -e "${GREEN}✓ Self-signed SSL certificate generated:${NC}"
    echo -e "    Cert: ${BLUE}${CERT}${NC}"
    echo -e "    Key:  ${BLUE}${KEY}${NC}"
    echo -e "    ${YELLOW}Browser warning is expected for self-signed certs — proceed to continue.${NC}"
}

# Pull images
pull_images() {
    echo -e "${YELLOW}Pulling Docker images (this may take a few minutes)...${NC}"
    $COMPOSE_CMD pull
    echo -e "${GREEN}✓ Images pulled${NC}"
}

# Ask user if they want to clear trading data
ask_clear_trading_data() {
    local db_file="data/data.db"

    # Only ask if database file exists
    if [ ! -f "$db_file" ]; then
        CLEAR_TRADING_DATA="no"
        return 0
    fi

    echo ""
    echo -e "${YELLOW}═══════════════════════════════════════════════════════════════${NC}"
    echo -e "${YELLOW}Do you want to clear trading data? (orders, fills, positions)${NC}"
    echo -e "${BLUE}  • trader_orders    (Order records)${NC}"
    echo -e "${BLUE}  • trader_fills     (Fill/execution records)${NC}"
    echo -e "${BLUE}  • trader_positions (Position records)${NC}"
    echo -e "${YELLOW}═══════════════════════════════════════════════════════════════${NC}"
    echo ""
    echo -e "${BLUE}Type 'yes' to clear tables, press Enter or any other input to skip${NC}"
    echo -n "Input: "
    if ! read -r confirm < /dev/tty; then
        # No TTY (piped install in CI etc.) — never block the install, default to safe "no"
        echo ""
        echo -e "${YELLOW}No interactive terminal detected, skipping data clear.${NC}"
        CLEAR_TRADING_DATA="no"
        return 0
    fi

    if [ "$confirm" == "yes" ]; then
        CLEAR_TRADING_DATA="yes"
        echo -e "${YELLOW}Trading data will be cleared before services start...${NC}"
    else
        CLEAR_TRADING_DATA="no"
        echo -e "${BLUE}Skipping data clear${NC}"
    fi
    echo ""
}

# Start services
start_services() {
    echo -e "${YELLOW}Starting FXOS services...${NC}"
    $COMPOSE_CMD up -d
    echo -e "${GREEN}✓ Services started${NC}"
}

# Clear trading data (called before services start)
clear_trading_data() {
    if [ "$CLEAR_TRADING_DATA" != "yes" ]; then
        return 0
    fi

    local db_file="data/data.db"

    if [ ! -f "$db_file" ]; then
        echo -e "${YELLOW}Database file not found, skipping...${NC}"
        return 0
    fi

    echo -e "${YELLOW}Clearing trading data tables...${NC}"

    if command -v sqlite3 &> /dev/null; then
        # Run inside an if-condition so a sqlite3 failure does not trip set -e
        # before the error branch below can run.
        if sqlite3 "$db_file" 'DELETE FROM trader_fills; DELETE FROM trader_orders; DELETE FROM trader_positions;'; then
            echo -e "${GREEN}✓ Trading data tables cleared${NC}"
        else
            echo -e "${RED}Failed to clear trading data${NC}"
        fi
    else
        echo -e "${RED}sqlite3 not found. Please install sqlite3 and run manually:${NC}"
        echo -e "${BLUE}  sqlite3 data/data.db 'DELETE FROM trader_fills; DELETE FROM trader_orders; DELETE FROM trader_positions;'${NC}"
    fi
}

# Wait for services
wait_for_services() {
    echo -e "${YELLOW}Waiting for services to be ready...${NC}"

    local max_attempts=30
    local attempt=1

    # Use 127.0.0.1 instead of localhost — on IPv6-first hosts "localhost" can
    # resolve to ::1 while the backend listens on IPv4 only.
    while [ $attempt -le $max_attempts ]; do
        if curl -s http://127.0.0.1:8080/api/health > /dev/null 2>&1; then
            echo -e "${GREEN}✓ Backend is ready${NC}"
            break
        fi
        echo "  Waiting for backend... ($attempt/$max_attempts)"
        sleep 2
        ((attempt++))
    done

    if [ $attempt -gt $max_attempts ]; then
        echo -e "${YELLOW}Backend is still starting, please wait a moment...${NC}"
    fi
}

# Print success message
print_success() {
    local SERVER_IP
    SERVER_IP=$(get_server_ip)

    echo ""
    echo -e "${GREEN}╔════════════════════════════════════════════════════════════╗"
    echo -e "║              🎉 Installation Complete! 🎉                   ║"
    echo -e "╚════════════════════════════════════════════════════════════╝${NC}"
    echo ""
    echo -e "  ${BLUE}Web Dashboard (HTTPS only, port 3000):${NC}"
    echo -e "    Local:    https://localhost:3000"
    echo -e "    Remote:   https://${SERVER_IP}:3000"
    echo ""
    echo -e "  ${CYAN}ℹ️  If you accidentally type http://${SERVER_IP}:3000${NC}"
    echo -e "  ${CYAN}   the server will auto-redirect to https:// on the same port.${NC}"
    echo ""
    echo -e "  ${YELLOW}🔒 First visit: browser will show a security warning.${NC}"
    echo -e "     Click → Advanced → Continue (or Proceed/Accept the risk).${NC}"
    echo -e "     This is expected for self-signed certificates.${NC}"
    echo ""
    echo -e "  ${BLUE}API Endpoint:${NC}   http://127.0.0.1:8080  (local API; the dashboard proxies /api via nginx)"
    echo -e "  ${BLUE}Install Dir:${NC}    $INSTALL_DIR"
    echo ""
    echo -e "${CYAN}╔════════════════════════════════════════════════════════════╗"
    echo -e "║  💡 Keep Updated: Run this command daily to stay current   ║"
    echo -e "╚════════════════════════════════════════════════════════════╝${NC}"
    echo ""
    echo -e "  ${GREEN}curl -fsSL https://raw.githubusercontent.com/onecany/fxos/dev/scripts/install.sh | bash${NC}"
    echo ""
    echo -e "  Updates are frequent. This one-liner pulls the latest"
    echo -e "  official images and restarts services automatically."
    echo ""
    echo -e "${YELLOW}Quick Commands (in $INSTALL_DIR):${NC}"
    echo "  $COMPOSE_CMD logs -f       # View logs"
    echo "  $COMPOSE_CMD restart       # Restart services"
    echo "  $COMPOSE_CMD down          # Stop services"
    echo "  $COMPOSE_CMD pull && $COMPOSE_CMD up -d  # Update to latest"
    echo ""
    echo -e "${YELLOW}Next Steps:${NC}"
    echo "  1. Open https://${SERVER_IP}:3000 in your browser"
    echo "  2. Accept the self-signed certificate warning once"
    echo "  3. Register your admin account"
    echo "  4. Configure AI Models (DeepSeek, OpenAI, etc.)"
    echo "  5. Configure Exchanges (Binance, Hyperliquid, etc.)"
    echo "  6. Create a Strategy in Strategy Studio"
    echo "  7. Create a Trader and start trading!"
    echo ""
    echo -e "${RED}⚠️  Risk Warning: AI trading carries significant risks.${NC}"
    echo -e "${RED}   Only use funds you can afford to lose!${NC}"
    echo ""
}

# Main
main() {
    check_prereqs
    setup_directory
    download_files
    generate_env
    generate_ssl_cert
    ask_clear_trading_data
    pull_images
    clear_trading_data
    start_services
    wait_for_services
    print_success
}

main
