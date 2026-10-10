locals {
  name = "${var.project_name}-${var.environment}"

  # Runtime config lives in SSM; deploy.sh renders each path into an env file.
  api_ssm_path        = "/${var.project_name}/${var.environment}/api/"
  monitoring_ssm_path = "/${var.project_name}/${var.environment}/monitoring/"
  monitoring_enabled  = nonsensitive(var.grafana_cloud_api_token != "")
  release_param       = "/${var.project_name}/${var.environment}/deploy/release"

  # The web app and the API share one origin (nginx routes /api to the API).
  site_origin    = "https://${var.domain}"
  manage_dns     = var.route53_zone_id != ""
  upload_origins = concat([local.site_origin], var.dev_origins)
}

data "aws_caller_identity" "current" {}

data "aws_availability_zones" "available" {
  state = "available"
}

data "aws_ssm_parameter" "al2023_ami" {
  name = "/aws/service/ami-amazon-linux-latest/al2023-ami-kernel-default-x86_64"
}
