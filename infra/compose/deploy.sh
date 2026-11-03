#!/usr/bin/env bash
# Runs on the host (started by twitter-deploy, which has synced this release's stack files):
# renders the env files from SSM, starts the stack, waits for every health check and reloads
# the configs. If the release doesn't come up healthy, it rolls back to the previous one and fails.
set -euo pipefail

APP_DIR=/opt/twitter
# shellcheck source=/dev/null # PROJECT, REGION, DEPLOY_BUCKET, REGISTRY, DOMAIN, API_SSM_PATH, STACK_SSM_PATH
source "$APP_DIR/deploy.env"
tag=${1:?usage: deploy.sh <commit-sha>}

export STATE_DIR="$APP_DIR/state"
mkdir -p "$STATE_DIR/alertmanager"
cd "$(dirname "$0")"

# Writes KEY='value' lines for every parameter under a path (the last path segment is the key)
render_env() {
  aws ssm get-parameters-by-path --region "$REGION" --path "$1" --recursive --with-decryption \
    --query 'Parameters[].[Name,Value]' --output text |
    while IFS=$'\t' read -r name value; do
      printf "%s='%s'\n" "${name##*/}" "$value"
    done
}

(
  umask 077 # these files hold secrets
  render_env "$API_SSM_PATH" >"$STATE_DIR/api.env.tmp" && mv "$STATE_DIR/api.env.tmp" "$STATE_DIR/api.env"
  # Everything the compose file reads, so `docker compose --env-file stack.env` works by hand too
  {
    render_env "$STACK_SSM_PATH"
    printf "DOMAIN=%s\nREGISTRY=%s\nTAG=%s\nSTATE_DIR=%s\n" "$DOMAIN" "$REGISTRY" "$tag" "$STATE_DIR"
  } >"$STATE_DIR/stack.env.tmp" && mv "$STATE_DIR/stack.env.tmp" "$STATE_DIR/stack.env"
  # Alertmanager reads the webhook at send time; without it, alerts only show in the UI
  webhook="$STATE_DIR/alertmanager/discord_webhook_url"
  if aws ssm get-parameter --region "$REGION" --name "/$PROJECT/discord_webhook_url" --with-decryption \
    --query Parameter.Value --output text >"$webhook.tmp" 2>/dev/null; then
    mv "$webhook.tmp" "$webhook"
  else
    rm -f "$webhook.tmp" "$webhook"
  fi
)
chmod 644 "$STATE_DIR"/alertmanager/* 2>/dev/null || true # the container runs as nobody

dc() { docker compose -f compose.prod.yaml --env-file "$STATE_DIR/stack.env" "$@"; }

aws ecr get-login-password --region "$REGION" | docker login --username AWS --password-stdin "$REGISTRY" >/dev/null
previous=$(cat "$STATE_DIR/release" 2>/dev/null || true)

dc pull --quiet
if dc up -d --no-build --remove-orphans --wait --wait-timeout 180 && ./reload-configs.sh; then
  echo "$tag" >"$STATE_DIR/release"
  docker image prune -af --filter "until=168h" >/dev/null # rollbacks pull from ECR anyway
  echo "Deployed $tag"
  exit 0
fi

echo "Release $tag did not become healthy:" >&2
dc ps -a >&2
dc logs --tail 40 api web caddy >&2 || true
if [[ -n "$previous" && "$previous" != "$tag" && -z "${ROLLING_BACK:-}" ]]; then
  echo "Rolling back to $previous" >&2
  ROLLING_BACK=1 "$TWITTER_DEPLOY" "$previous" || echo "Rollback to $previous failed too" >&2
fi
exit 1
