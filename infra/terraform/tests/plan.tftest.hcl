# Plans the whole stack against a mocked AWS provider: no credentials, no cost.
# Catches evaluation errors `terraform validate` can't (for_each, outputs, conditions).
#   terraform init -backend=false && terraform test

mock_provider "aws" {
  mock_data "aws_availability_zones" {
    defaults = { names = ["ap-southeast-1a", "ap-southeast-1b"] }
  }
  mock_data "aws_ssm_parameter" {
    defaults = { value = "ami-0123456789abcdef0" }
  }
  mock_data "aws_caller_identity" {
    defaults = { account_id = "123456789012" }
  }
  mock_resource "aws_iam_role" {
    defaults = { arn = "arn:aws:iam::123456789012:role/mock" }
  }
  mock_resource "aws_iam_openid_connect_provider" {
    defaults = { arn = "arn:aws:iam::123456789012:oidc-provider/token.actions.githubusercontent.com" }
  }
  mock_resource "aws_instance" {
    defaults = { arn = "arn:aws:ec2:ap-southeast-1:123456789012:instance/i-mock" }
  }
  mock_resource "aws_ecr_repository" {
    defaults = {
      arn            = "arn:aws:ecr:ap-southeast-1:123456789012:repository/mock"
      repository_url = "123456789012.dkr.ecr.ap-southeast-1.amazonaws.com/mock"
    }
  }
  mock_resource "aws_sns_topic" {
    defaults = { arn = "arn:aws:sns:ap-southeast-1:123456789012:mock" }
  }
  mock_resource "aws_s3_bucket" {
    defaults = { arn = "arn:aws:s3:::mock-bucket", id = "mock-bucket" }
  }
  mock_resource "aws_ssm_parameter" {
    defaults = { arn = "arn:aws:ssm:ap-southeast-1:123456789012:parameter/mock" }
  }
  mock_resource "aws_cloudfront_distribution" {
    defaults = { arn = "arn:aws:cloudfront::123456789012:distribution/MOCK" }
  }
}

mock_provider "aws" {
  alias = "us_east_1"
  mock_data "aws_availability_zones" {
    defaults = { names = ["ap-southeast-1a", "ap-southeast-1b"] }
  }
  mock_data "aws_ssm_parameter" {
    defaults = { value = "ami-0123456789abcdef0" }
  }
  mock_data "aws_caller_identity" {
    defaults = { account_id = "123456789012" }
  }
  mock_resource "aws_iam_role" {
    defaults = { arn = "arn:aws:iam::123456789012:role/mock" }
  }
  mock_resource "aws_iam_openid_connect_provider" {
    defaults = { arn = "arn:aws:iam::123456789012:oidc-provider/token.actions.githubusercontent.com" }
  }
  mock_resource "aws_instance" {
    defaults = { arn = "arn:aws:ec2:ap-southeast-1:123456789012:instance/i-mock" }
  }
  mock_resource "aws_ecr_repository" {
    defaults = {
      arn            = "arn:aws:ecr:ap-southeast-1:123456789012:repository/mock"
      repository_url = "123456789012.dkr.ecr.ap-southeast-1.amazonaws.com/mock"
    }
  }
  mock_resource "aws_sns_topic" {
    defaults = { arn = "arn:aws:sns:ap-southeast-1:123456789012:mock" }
  }
  mock_resource "aws_s3_bucket" {
    defaults = { arn = "arn:aws:s3:::mock-bucket", id = "mock-bucket" }
  }
  mock_resource "aws_ssm_parameter" {
    defaults = { arn = "arn:aws:ssm:ap-southeast-1:123456789012:parameter/mock" }
  }
  mock_resource "aws_cloudfront_distribution" {
    defaults = { arn = "arn:aws:cloudfront::123456789012:distribution/MOCK" }
  }
}

variables {
  domain              = "twitter.example.com"
  acme_email          = "ops@example.com"
  alert_email         = "ops@example.com"
  db_username         = "admin_user"
  db_password         = "abcdefghijklmnop1234"
  token_symmetric_key = "0123456789abcdef0123456789abcdef0123"
  google_client_id    = "client.apps.googleusercontent.com"
}

run "plans_with_minimal_settings" {
  command = plan

  assert {
    condition     = length(aws_ssm_parameter.monitoring) == 0
    error_message = "Monitoring parameters must not be created without a Grafana token."
  }
  assert {
    condition     = !contains(keys(aws_ssm_parameter.api), "GEMINI_API_KEY")
    error_message = "Empty settings must not be stored (SSM rejects empty values)."
  }
  assert {
    condition     = aws_db_instance.main.storage_encrypted && aws_db_instance.main.backup_retention_period >= 7 && aws_db_instance.main.deletion_protection
    error_message = "RDS must be encrypted, backed up for a week and protected from deletion."
  }
  assert {
    condition     = nonsensitive(aws_ssm_parameter.api["ALLOWED_ORIGINS"].value) == "https://twitter.example.com"
    error_message = "The API must allow exactly the site's own origin."
  }
  assert {
    condition     = !contains(keys(aws_ssm_parameter.api), "REDIS_URL") && !contains(keys(aws_ssm_parameter.api), "COOKIE_DOMAIN")
    error_message = "A single host needs neither Redis nor a shared cookie domain."
  }
  assert {
    condition     = toset(keys(aws_ecr_repository.app)) == toset(["api", "web"])
    error_message = "Both images need a repository."
  }
  assert {
    condition     = length(aws_route53_record.app) == 0
    error_message = "No Route 53 records without a hosted zone."
  }
  assert {
    condition     = aws_s3_bucket_public_access_block.deploy.restrict_public_buckets && aws_ssm_parameter.release.value == "none"
    error_message = "Releases must be private, and the release parameter must start empty."
  }
  assert {
    condition     = aws_route53_health_check.site.fqdn == "twitter.example.com" && aws_route53_health_check.site.resource_path == "/readyz"
    error_message = "The uptime check must hit the site's readiness endpoint."
  }
  assert {
    condition     = contains(aws_cloudwatch_metric_alarm.instance_system_check.alarm_actions, "arn:aws:automate:ap-southeast-1:ec2:recover")
    error_message = "A failed host must be recovered automatically."
  }
  assert {
    condition     = toset(keys(aws_cloudwatch_metric_alarm.rds)) == toset(["storage", "cpu", "memory"])
    error_message = "The database alarms are missing."
  }
  assert {
    condition     = aws_ssm_association.patch.parameters["Operation"] == "Install"
    error_message = "OS patches must be installed on a schedule."
  }
}

run "plans_with_everything_enabled" {
  command = plan

  variables {
    route53_zone_id         = "Z0123456789"
    gemini_api_key          = "gemini-key"
    grafana_cloud_api_token = "glc_token"
    grafana_cloud_loki_url  = "https://logs.example.com/push"
  }

  assert {
    condition     = contains(keys(aws_ssm_parameter.api), "GEMINI_API_KEY")
    error_message = "A set Gemini key must be stored."
  }
  assert {
    condition     = contains(keys(aws_ssm_parameter.monitoring), "GRAFANA_API_TOKEN")
    error_message = "Monitoring parameters must be created with a Grafana token."
  }
  assert {
    condition     = length(aws_route53_record.app) == 1 && aws_route53_record.app[0].name == "twitter.example.com"
    error_message = "The domain's record must be managed when a hosted zone is set."
  }
}

run "rejects_weak_database_password" {
  command = plan
  variables {
    db_password = "short"
  }
  expect_failures = [var.db_password]
}

# Apply against the mocks (still no AWS calls): user data is only known after apply.
run "first_boot_restores_the_last_release" {
  command = apply

  assert {
    condition     = strcontains(aws_instance.app.user_data, var.docker_compose_sha256) && strcontains(aws_instance.app.user_data, "RELEASE_PARAM=/chmtwt/prod/deploy/release")
    error_message = "First boot must verify Compose and know where the last release is recorded."
  }
}

run "rejects_an_invalid_alert_email" {
  command = plan
  variables {
    alert_email = "not-an-email"
  }
  expect_failures = [var.alert_email]
}

run "rejects_a_url_as_domain" {
  command = plan
  variables {
    domain = "https://twitter.example.com"
  }
  expect_failures = [var.domain]
}
