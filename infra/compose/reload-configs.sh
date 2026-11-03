#!/usr/bin/env bash
# Run by deploy.sh after `docker compose up`. Compose only recreates containers whose definition
# changed, and the configs are bind-mounted folders, so a changed Caddyfile or alert rule alone
# restarts nothing. Each step is a graceful reload: an invalid config is refused, the old one
# keeps running and this script fails, so the deploy fails (and rolls back) loudly.
set -euo pipefail
cd "$(dirname "$0")"
dc() { docker compose -f compose.prod.yaml --env-file "$STATE_DIR/stack.env" "$@"; }

# Open connections (event streams included) stay up
dc exec -T caddy caddy reload --config /etc/caddy/Caddyfile

# Ask from the Prometheus container: it has wget, Alloy's image has no HTTP client
for target in localhost:9090 alertmanager:9093 alloy:12345; do
  dc exec -T prometheus wget -q -O /dev/null --post-data= "http://$target/-/reload"
done
echo "Reloaded Caddy, Prometheus, Alertmanager and Alloy"
