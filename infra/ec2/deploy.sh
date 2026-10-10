#!/usr/bin/env bash
# Deploys the stack on this instance: renders env files from SSM Parameter Store,
# pulls the web and API images from ECR and (re)starts the containers.
#
#   deploy.sh [IMAGE_TAG]    # defaults to the last deployed tag
#
# Run by cloud-init on first boot and by the GitHub Actions deploy job (via SSM).
set -euo pipefail

APP_DIR=/opt/twitter
cd "$APP_DIR"
# REGION, API_SSM_PATH, MONITORING_SSM_PATH, ECR_API_URL, ECR_WEB_URL, SITE_DOMAIN, ACME_EMAIL, MONITORING
source ./deploy.env

tag="${1:-$(cat .image-tag 2>/dev/null || true)}"
if [[ -z "$tag" ]]; then
  echo "No image tag yet: push an image and deploy with deploy.sh <tag>." >&2
  exit 0
fi

# Writes KEY='value' lines for every parameter under a path (the last path segment is the key).
render_env() {
  aws ssm get-parameters-by-path --region "$REGION" --path "$1" --recursive --with-decryption \
    --query 'Parameters[].[Name,Value]' --output text |
    while IFS=$'\t' read -r name value; do
      printf "%s='%s'\n" "${name##*/}" "$value"
    done
}

umask 077 # env files hold secrets
render_env "$API_SSM_PATH" > api.env.tmp && mv api.env.tmp api.env
if [[ "$MONITORING" == "true" ]]; then
  render_env "$MONITORING_SSM_PATH" > monitoring.env.tmp && mv monitoring.env.tmp monitoring.env
  export COMPOSE_PROFILES=monitoring
else
  : > monitoring.env
fi

aws ecr get-login-password --region "$REGION" | docker login --username AWS --password-stdin "${ECR_API_URL%%/*}"

# Both images are tagged with the same commit SHA, so the web app and the API always match.
export API_IMAGE="$ECR_API_URL:$tag" WEB_IMAGE="$ECR_WEB_URL:$tag" SITE_DOMAIN ACME_EMAIL
docker compose pull --quiet
docker compose up -d --remove-orphans
echo "$tag" > .image-tag
docker image prune -f >/dev/null

echo "Deployed $tag ($API_IMAGE, $WEB_IMAGE)"
