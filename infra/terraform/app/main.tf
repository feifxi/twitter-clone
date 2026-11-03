# The app stack: everything a running site needs, created by `infra/scripts/up.sh` and removed
# by `down.sh`. Long-lived pieces (DNS name servers, the GitHub deploy role) are
# in ../bootstrap.

locals {
  name        = var.project
  site_origin = "https://${var.domain}"

  # Runtime config in SSM Parameter Store; the host renders each path into an env file
  # at deploy time (see infra/compose/rollout.sh).
  api_ssm_path   = "/${var.project}/api/"   # the API container's environment
  stack_ssm_path = "/${var.project}/stack/" # variables the compose file needs (passwords, ...)
}

data "aws_caller_identity" "current" {}

data "aws_availability_zones" "available" {
  state = "available"
}
