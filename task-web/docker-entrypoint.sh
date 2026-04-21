#!/bin/sh
set -eu

# Render runtime /config.js so the SPA can pick up API_URL without rebuilding.
# API_URL defaults to "" which makes the client use the same-origin /api proxy.
API_URL="${API_URL:-}"

cat > /usr/share/nginx/html/config.js <<EOF
window.__APP_CONFIG__ = {
  API_URL: "${API_URL}"
};
EOF

exec "$@"
