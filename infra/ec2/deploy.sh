#!/usr/bin/env bash
# Deploys one release (both images share the commit SHA tag): renders the env files
# from SSM Parameter Store, starts the stack and waits for every health check. If the
# release doesn't come up healthy, it rolls back to the previous one and fails.
#
# Started by twitter-deploy, which has synced this commit's stack files next to it.
set -euo pipefail

APP_DIR=${APP_DIR:-/opt/twitter}
# REGION, DEPLOY_BUCKET, RELEASE_PARAM, API_SSM_PATH, MONITORING_SSM_PATH, MONITORING,
# ECR_API_URL, ECR_WEB_URL, SITE_DOMAIN, ACME_EMAIL
# shellcheck source=/dev/null # written by cloud-init
source "$APP_DIR/deploy.env"
tag=${1:?usage: deploy.sh <commit-sha>}

export STATE_DIR="$APP_DIR/state"
mkdir -p "$STATE_DIR"/{tls,letsencrypt,acme}
cd "$(dirname "$0")"

# Writes KEY='value' lines for every parameter under a path (the last path segment is the key).
render_env() {
  aws ssm get-parameters-by-path --region "$REGION" --path "$1" --recursive --with-decryption \
    --query 'Parameters[].[Name,Value]' --output text |
    while IFS=$'\t' read -r name value; do
      printf "%s='%s'\n" "${name##*/}" "$value"
    done
}

(
  umask 077 # env files and keys hold secrets
  render_env "$API_SSM_PATH" > "$STATE_DIR/api.env.tmp" && mv "$STATE_DIR/api.env.tmp" "$STATE_DIR/api.env"
  if [[ "$MONITORING" == "true" ]]; then
    render_env "$MONITORING_SSM_PATH" > "$STATE_DIR/monitoring.env.tmp" && mv "$STATE_DIR/monitoring.env.tmp" "$STATE_DIR/monitoring.env"
  else
    : > "$STATE_DIR/monitoring.env"
  fi

  # nginx can't start without a certificate. Until certbot has the real one, serve a
  # short-lived self-signed placeholder (only on a brand-new instance).
  if [[ ! -s "$STATE_DIR/tls/fullchain.pem" ]]; then
    openssl req -x509 -nodes -newkey rsa:2048 -days 7 -subj "/CN=$SITE_DOMAIN" \
      -keyout "$STATE_DIR/tls/privkey.pem" -out "$STATE_DIR/tls/fullchain.pem" 2>/dev/null
  fi
)
[[ "$MONITORING" == "true" ]] && export COMPOSE_PROFILES=monitoring

aws ecr get-login-password --region "$REGION" | docker login --username AWS --password-stdin "${ECR_API_URL%%/*}"

export API_IMAGE="$ECR_API_URL:$tag" WEB_IMAGE="$ECR_WEB_URL:$tag" SITE_DOMAIN ACME_EMAIL
previous=$(cat "$STATE_DIR/release" 2>/dev/null || true)

docker compose pull --quiet
if docker compose up -d --remove-orphans --wait --wait-timeout 180; then
  echo "$tag" > "$STATE_DIR/release"
  # Remembered outside the instance too: a replacement instance deploys it at first boot.
  aws ssm put-parameter --region "$REGION" --name "$RELEASE_PARAM" --value "$tag" --type String --overwrite >/dev/null
  # Keep the disk from filling up; rollbacks pull from ECR anyway.
  docker image prune -af --filter "until=168h" >/dev/null
  echo "Deployed $tag"
  exit 0
fi

echo "Release $tag did not become healthy:" >&2
docker compose ps -a >&2
docker compose logs --tail 40 api web nginx >&2 || true
if [[ -n "$previous" && "$previous" != "$tag" && -z "${ROLLING_BACK:-}" ]]; then
  echo "Rolling back to $previous" >&2
  ROLLING_BACK=1 "$TWITTER_DEPLOY" "$previous" || echo "Rollback to $previous failed too" >&2
fi
exit 1
