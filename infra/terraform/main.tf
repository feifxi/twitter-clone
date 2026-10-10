locals {
  name = "${var.project_name}-${var.environment}"

  # Runtime config lives in SSM; deploy.sh renders each path into an env file.
  api_ssm_path        = "/${var.project_name}/${var.environment}/api/"
  monitoring_ssm_path = "/${var.project_name}/${var.environment}/monitoring/"
  monitoring_enabled  = nonsensitive(var.grafana_cloud_api_token != "")

  web_origin     = "https://${var.web_domain}"
  amplify_url    = "https://${var.gh_branch}.${aws_amplify_app.web.default_domain}"
  manage_dns     = var.route53_zone_id != ""
  api_base_url   = "https://${var.api_domain}/api/v1"
  upload_origins = concat([local.web_origin, local.amplify_url], var.dev_origins)
}

data "aws_caller_identity" "current" {}

data "aws_availability_zones" "available" {
  state = "available"
}

data "aws_ssm_parameter" "al2023_ami" {
  name = "/aws/service/ami-amazon-linux-latest/al2023-ami-kernel-default-x86_64"
}
