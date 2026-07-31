#!/bin/sh
# FXOS Frontend Nginx Entrypoint
#
# Ensures the HTTPS server block in nginx.conf always has a cert to load.
# Priority:
#   1. Host-mounted cert at /etc/nginx/ssl/fxos.{crt,key} (docker-compose ./ssl:ro)
#   2. On-the-fly self-signed cert generated inside the container (fallback)
#
# Design note: this script runs with set -e (fail on error). *Every* command
# that can legitimately fail (mkdir/chmod on a read-only mounted volume,
# openssl without -addext support, etc.) is either guarded by an if-block,
# an explicit `|| true`, or an `|| fallback_command` chain — otherwise the
# container would enter a crash loop on edge cases.

set -eu

SSL_DIR="/etc/nginx/ssl"
CERT="${SSL_DIR}/fxos.crt"
KEY="${SSL_DIR}/fxos.key"
CONF="/etc/nginx/conf.d/default.conf"
TMP_DIR="/tmp/fxos-ssl"

# Best-effort ensure dir/perms; ignore failure on read-only mounted volumes.
mkdir -p "${SSL_DIR}" 2>/dev/null || true
chmod 700 "${SSL_DIR}" 2>/dev/null || true

# ──────────────────────────────────────────────────────────────────
# Helpers
# ──────────────────────────────────────────────────────────────────

# Comment out the entire HTTPS server block (listen 443 ssl) in $CONF
# so nginx can still start on HTTP(80) when certs are impossible.
disable_https_block() {
    if [ ! -w "${CONF}" ]; then
        echo "[nginx-entrypoint] WARNING: cannot patch ${CONF} (not writable)."
        return 1
    fi
    # Use python3 if available for reliable brace-matching; else a simpler
    # awk line-range fallback that works on typical alpine/busybox.
    if command -v python3 >/dev/null 2>&1; then
        python3 - "${CONF}" <<'PY'
import sys, re
path = sys.argv[1]
with open(path) as f:
    src = f.read()
# Match a server block beginning "# HTTPS server" comment through its matching close brace
replacement = "# HTTPS block disabled by entrypoint (no usable certs found)\n"
new = re.sub(
    r"# HTTPS server[^\n]*\n\s*server\s*\{(?:[^{}]|\{(?:[^{}]|\{[^{}]*\})*\})*\}",
    lambda m: "# __SSL_DISABLED_START__\n# " + "\n# ".join(m.group(0).splitlines()) + "\n# __SSL_DISABLED_END__\n",
    src, count=1, flags=re.MULTILINE | re.DOTALL)
with open(path, "w") as f:
    f.write(new)
PY
        return 0
    fi
    # Simpler (less precise) fallback: mark using two sentinel patterns we can
    # find via awk and comment everything in between. The default nginx.conf
    # shipped with FXOS has exactly `listen 443 ssl;` near the top of the
    # block and `}` alone on a line closing it.
    echo "[nginx-entrypoint] python3 missing; using awk fallback to disable HTTPS block."
    awk '
        BEGIN { skip=0; depth=0 }
        /^# HTTPS server \(only active when SSL/ { skip=1; print "# __SSL_DISABLED_START__"; next }
        skip {
            # Count braces to find real end of server block
            n = split($0, chars, "")
            for (i=1; i<=n; i++) {
                if (chars[i] == "{") depth++
                if (chars[i] == "}") depth--
            }
            if (depth <= 0 && /^[[:space:]]*\}[[:space:]]*$/) {
                print "# __SSL_DISABLED_END__ (was HTTPS server block)"
                skip=0
            } else {
                print "# " $0
            }
            next
        }
        { print }
    ' "${CONF}" > "${CONF}.tmp" 2>/dev/null || true
    if [ -s "${CONF}.tmp" ]; then
        mv "${CONF}.tmp" "${CONF}" 2>/dev/null || true
    fi
    return 0
}

# Generate a self-signed cert into $1=crt $2=key $3=SANs $4=CN
# Returns 0 on success. Always safe under set -e because every branch is guarded.
gen_self_signed() {
    out_crt="$1"
    out_key="$2"
    san="$3"
    cn_value="$4"

    # Try with SANs first (openssl >= 1.1.1). In if-block → set -e passes.
    if openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
        -keyout "${out_key}" -out "${out_crt}" \
        -subj "/CN=${cn_value}" \
        -addext "subjectAltName=${san}" >/dev/null 2>&1; then
        return 0
    fi
    rm -f "${out_crt}" "${out_key}" 2>/dev/null || true
    # Fallback: basic cert without SANs (also in if-block)
    if openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
        -keyout "${out_key}" -out "${out_crt}" \
        -subj "/CN=${cn_value}" >/dev/null 2>&1; then
        return 0
    fi
    return 1
}

# Rewrite ssl_certificate(_key) paths in the HTTPS server block to new paths.
# Used when we can't write to the expected SSL_DIR location.
rewrite_nginx_ssl_paths() {
    new_cert="$1"
    new_key="$2"
    if [ ! -w "${CONF}" ]; then
        echo "[nginx-entrypoint] WARNING: cannot rewrite SSL paths in ${CONF} (not writable)."
        return 1
    fi
    if sed -i "s|ssl_certificate     /etc/nginx/ssl/fxos.crt;|ssl_certificate     ${new_cert};|g" "${CONF}" 2>/dev/null && \
       sed -i "s|ssl_certificate_key /etc/nginx/ssl/fxos.key;|ssl_certificate_key ${new_key};|g" "${CONF}" 2>/dev/null; then
        return 0
    fi
    return 1
}

# ──────────────────────────────────────────────────────────────────
# Main logic
# ──────────────────────────────────────────────────────────────────

# Case 1: certs already exist at the expected path → use them
if [ -f "${CERT}" ] && [ -f "${KEY}" ]; then
    echo "[nginx-entrypoint] Using existing SSL certificate: ${CERT}"
    chmod 644 "${CERT}" 2>/dev/null || true
    chmod 600 "${KEY}"  2>/dev/null || true
    echo "[nginx-entrypoint] Starting nginx on HTTP(80) + HTTPS(443)"
    exec nginx -g "daemon off;"
fi

echo "[nginx-entrypoint] No SSL cert at ${CERT}/${KEY} — generating self-signed cert (365 days)."

SANS="IP:127.0.0.1,DNS:localhost"
if [ -n "${FXOS_SERVER_IP:-}" ]; then
    SANS="IP:${FXOS_SERVER_IP},${SANS}"
fi
CN="${FXOS_SERVER_IP:-localhost}"

# ── Case 2: try writing certs to the canonical SSL_DIR location ──
if gen_self_signed "${CERT}" "${KEY}" "${SANS}" "${CN}"; then
    echo "[nginx-entrypoint] Self-signed cert created at ${CERT} for CN=${CN}"
    chmod 644 "${CERT}" 2>/dev/null || true
    chmod 600 "${KEY}"  2>/dev/null || true
    echo "[nginx-entrypoint] Starting nginx on HTTP(80) + HTTPS(443)"
    exec nginx -g "daemon off;"
fi

# ── Case 3: SSL_DIR probably read-only; use /tmp and rewrite nginx ──
echo "[nginx-entrypoint] Canonical SSL dir not writable; using fallback dir ${TMP_DIR}"
rm -rf "${TMP_DIR}" 2>/dev/null || true
mkdir -p "${TMP_DIR}" 2>/dev/null || true

if gen_self_signed "${TMP_DIR}/fxos.crt" "${TMP_DIR}/fxos.key" "${SANS}" "${CN}"; then
    echo "[nginx-entrypoint] Self-signed cert created at ${TMP_DIR} for CN=${CN}"
    chmod 644 "${TMP_DIR}/fxos.crt" 2>/dev/null || true
    chmod 600 "${TMP_DIR}/fxos.key"  2>/dev/null || true

    # First: try symlinking into SSL_DIR (may work if dir is writable but file creation failed)
    rm -f "${CERT}" "${KEY}" 2>/dev/null || true
    if ln -sf "${TMP_DIR}/fxos.crt" "${CERT}" 2>/dev/null && \
       ln -sf "${TMP_DIR}/fxos.key"  "${KEY}"  2>/dev/null; then
        echo "[nginx-entrypoint] Symlinked fallback certs into ${SSL_DIR}"
        echo "[nginx-entrypoint] Starting nginx on HTTP(80) + HTTPS(443)"
        exec nginx -g "daemon off;"
    fi

    # Symlink didn't work; try rewriting the nginx config SSL paths
    if rewrite_nginx_ssl_paths "${TMP_DIR}/fxos.crt" "${TMP_DIR}/fxos.key"; then
        echo "[nginx-entrypoint] Nginx SSL paths rewritten to ${TMP_DIR}"
        echo "[nginx-entrypoint] Starting nginx on HTTP(80) + HTTPS(443)"
        exec nginx -g "daemon off;"
    fi
fi

# ── Case 4: absolute worst case — disable HTTPS, keep only HTTP ──
echo "[nginx-entrypoint] CRITICAL: could not generate usable SSL certs."
echo "[nginx-entrypoint]          Disabling HTTPS server block; nginx will serve HTTP(80) only."
disable_https_block || true

echo "[nginx-entrypoint] Starting nginx on HTTP(80) only (HTTPS disabled)"
exec nginx -g "daemon off;"
