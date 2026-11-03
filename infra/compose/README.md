# Production stack

Everything runs on one host with Docker Compose ([`compose.prod.yaml`](compose.prod.yaml)):

| Service | What |
|---|---|
| `caddy` | HTTPS (automatic certificates) and routing: `$DOMAIN` → web, `/api/*` → api, `grafana.$DOMAIN` → Grafana |
| `web`, `api` | The Next.js app and the Go API (images from ECR, tagged with the commit SHA) |
| `postgres` | PostgreSQL 16 + pgvector; data on the host's disk |
| `prometheus`, `alertmanager`, `loki`, `alloy`, `grafana`, `node-exporter` | Monitoring ([`../observability`](../observability/README.md)) |

Only Caddy publishes ports. Every service has a health check: `rollout.sh` waits for all of them
and rolls back if they don't pass.

## Files

- `rollout.sh`: runs on the host for each release (env files from SSM, `compose up --wait`,
  config reload, rollback). Started by `twitter-deploy` (installed at first boot, see
  [`../terraform/app/twitter-deploy`](../terraform/app/twitter-deploy)).
- `reload-configs.sh`: configs are mounted as **folders**, so a deploy's new files are visible
  in the running containers; this reloads Caddy, Prometheus, Alertmanager and Alloy in place
  (no restart, no gap). An invalid config is refused and fails the deploy.

## Try it on this machine

`DOMAIN=localhost` makes Caddy use its own local certificates. The API runs with
`ENVIRONMENT=development` here (production refuses `localhost` origins).

```bash
cd infra/compose
export STATE_DIR=$(mktemp -d) DOMAIN=localhost
cat > $STATE_DIR/api.env <<'EOF'
ENVIRONMENT=development
ALLOWED_ORIGINS=https://localhost
DATABASE_URL=postgres://twitter:localpass@postgres:5432/twitter?sslmode=disable
TOKEN_SYMMETRIC_KEY=0123456789abcdef0123456789abcdef0123456789abcdef
EOF
printf "POSTGRES_PASSWORD=localpass\nGF_ADMIN_PASSWORD=admin\n" > $STATE_DIR/stack.env
mkdir -p $STATE_DIR/alertmanager

dc() { docker compose -f compose.prod.yaml --env-file $STATE_DIR/stack.env "$@"; }
dc up -d --build --wait     # https://localhost (accept the certificate), Grafana: https://grafana.localhost
dc down -v
```
