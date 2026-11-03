#!/usr/bin/env bash
# Tear the stack down: everything in infra/terraform/app, data included (it's disposable).
# Bootstrap (name servers, GitHub deploy role) stays, so the next up needs no extra steps.
#   infra/scripts/down.sh     Terraform lists what it destroys and asks first
set -euo pipefail
cd "$(dirname "$0")/../.."

terraform -chdir=infra/terraform/app destroy

echo
echo "==> Down. Bring it back with infra/scripts/up.sh"
