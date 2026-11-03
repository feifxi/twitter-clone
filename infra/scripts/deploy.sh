#!/usr/bin/env bash
# Deploy a commit to the running stack: build and push both images, upload the release's stack
# files, run it on the host over SSM (which rolls back by itself if it isn't healthy), then
# check the site. No Terraform changes.
#   infra/scripts/deploy.sh                 this checkout (uncommitted changes get a -dirty tag)
#   infra/scripts/deploy.sh <commit-sha>    redeploy / roll back to an earlier release
#   deploy.sh --skip-if-down                from CI: exit 0 (nothing to do) when the stack is down
# Everything it needs comes from the /chmtwt/deploy/config SSM parameter the app stack writes,
# so it works the same here and in GitHub Actions.
set -euo pipefail
cd "$(dirname "$0")/../.."
export AWS_REGION=${AWS_REGION:-ap-southeast-1}

skip_if_down="" tag=${TAG:-}
for arg in "$@"; do
  case "$arg" in
    --skip-if-down) skip_if_down=1 ;;
    *) tag=$arg ;;
  esac
done

if ! config=$(aws ssm get-parameter --name /chmtwt/deploy/config --query Parameter.Value --output text 2>/dev/null); then
  if [[ -n "$skip_if_down" ]]; then
    echo "The stack is down (no /chmtwt/deploy/config): nothing to deploy."
    exit 0
  fi
  echo "The stack is down. Run infra/scripts/up.sh first."
  exit 1
fi
cfg() { python3 -c 'import json,sys; print(json.loads(sys.argv[1])[sys.argv[2]])' "$config" "$1"; }
registry=$(cfg registry) bucket=$(cfg bucket) instance=$(cfg instance_id) domain=$(cfg domain)

# One immutable tag per release: the commit, plus a timestamp for uncommitted local changes
if [[ -z "$tag" ]]; then
  tag=$(git rev-parse HEAD)
  git diff --quiet HEAD || tag="$tag-dirty-$(date +%Y%m%d%H%M%S)"
fi

ssm_run() { # <command> -> waits, prints the output, fails if the command failed
  local id status
  id=$(aws ssm send-command --instance-ids "$instance" --document-name AWS-RunShellScript \
    --parameters "commands=[\"$1\"]" --timeout-seconds 900 --query Command.CommandId --output text)
  aws ssm wait command-executed --command-id "$id" --instance-id "$instance" 2>/dev/null || true
  status=$(aws ssm get-command-invocation --command-id "$id" --instance-id "$instance" --query Status --output text)
  aws ssm get-command-invocation --command-id "$id" --instance-id "$instance" \
    --query '[StandardOutputContent, StandardErrorContent]' --output text | tail -25
  [[ "$status" == "Success" ]]
}

if aws s3 ls "s3://$bucket/stack/$tag/compose/rollout.sh" >/dev/null 2>&1; then
  echo "==> Release $tag was uploaded before: redeploying it as it was"
else
  [[ "$tag" == "$(git rev-parse HEAD)"* ]] || {
    echo "Release $tag was never uploaded. To deploy another commit, check it out first."
    exit 1
  }

  echo "==> Building and pushing images ($tag)"
  aws ecr get-login-password | docker login --username AWS --password-stdin "$registry" >/dev/null
  # Safe to re-run: an image already in ECR is skipped (tags are immutable)
  build_and_push() { # <name> <context> [docker build args...]
    local name=$1 context=$2 image="$registry/chmtwt-$1:$tag"
    shift 2
    if aws ecr describe-images --repository-name "chmtwt-$name" --image-ids "imageTag=$tag" >/dev/null 2>&1; then
      echo "    chmtwt-$name:$tag is already in ECR"
      return
    fi
    # One manifest per image (no attestations) keeps ECR's image count and lifecycle simple
    docker build --platform linux/arm64 --provenance=false --sbom=false -t "$image" "$@" "$context"
    for attempt in 1 2 3; do
      docker push --quiet "$image" && return
      echo "    push failed (attempt $attempt), retrying"
      sleep 5
    done
    echo "Pushing chmtwt-$name failed 3 times"
    exit 1
  }
  build_and_push api twitter-go-api
  build_and_push web twitter-next-web --build-arg "NEXT_PUBLIC_GOOGLE_CLIENT_ID=$(cfg google_client_id)"

  echo "==> Uploading the stack files"
  aws s3 sync infra/compose "s3://$bucket/stack/$tag/compose" --only-show-errors --exclude "README.md"
  aws s3 sync infra/observability "s3://$bucket/stack/$tag/observability" --only-show-errors \
    --exclude "*.py" --exclude "README.md" --exclude "alertmanager/secrets/*"
fi

echo "==> Waiting for the host to be ready" # a new instance needs a few minutes for first boot
ready=""
for _ in $(seq 1 60); do
  online=$(aws ssm describe-instance-information --filters "Key=InstanceIds,Values=$instance" \
    --query 'InstanceInformationList[0].PingStatus' --output text 2>/dev/null || true)
  if [[ "$online" == "Online" ]] && ssm_run "test -f /opt/twitter/ready" >/dev/null 2>&1; then
    ready=1
    break
  fi
  sleep 10
done
[[ -n "$ready" ]] || {
  echo "The host isn't ready after 10 minutes. Look at it: aws ssm start-session --target $instance"
  exit 1
}

echo "==> Deploying on the host"
ssm_run "twitter-deploy $tag" || { echo "The deploy failed on the host (see above; it rolled back if it could)"; exit 1; }

echo "==> Checking the site"
for _ in $(seq 1 30); do
  if curl -fsS -m 5 -o /dev/null "https://$domain/readyz" && curl -fsS -m 5 -o /dev/null "https://$domain/"; then
    echo
    echo "==> Deployed $tag: https://$domain · Grafana https://grafana.$domain"
    exit 0
  fi
  sleep 5
done
echo "https://$domain isn't healthy 2.5 minutes after the deploy (a brand-new stack may still be waiting for DNS or its certificate)"
exit 1
