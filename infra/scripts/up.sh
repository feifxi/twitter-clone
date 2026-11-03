#!/usr/bin/env bash
# Bring the stack up (or change its infrastructure): terraform apply, then deploy the code.
#   infra/scripts/up.sh       Terraform shows the plan and asks before changing anything
# For code-only changes, infra/scripts/deploy.sh skips Terraform.
set -euo pipefail
cd "$(dirname "$0")/../.."
TF="terraform -chdir=infra/terraform/app"

[[ -f infra/terraform/app/terraform.tfvars ]] || {
  echo "Create infra/terraform/app/terraform.tfvars first (see terraform.tfvars.example)"
  exit 1
}

$TF init -input=false >/dev/null
$TF apply

bash infra/scripts/deploy.sh

echo
echo "    Grafana login: admin / \$($TF output -raw grafana_password)"
echo "    First up? Confirm the SNS email from AWS to get alarm emails."
echo "    Shell on the host: aws ssm start-session --target \$($TF output -raw instance_id)"
