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
  mock_resource "aws_s3_bucket" {
    defaults = { arn = "arn:aws:s3:::mock-bucket" }
  }
  mock_resource "aws_ecr_repository" {
    defaults = {
      arn            = "arn:aws:ecr:ap-southeast-1:123456789012:repository/mock"
      repository_url = "123456789012.dkr.ecr.ap-southeast-1.amazonaws.com/mock"
    }
  }
  mock_resource "aws_cloudfront_distribution" {
    defaults = { arn = "arn:aws:cloudfront::123456789012:distribution/MOCK" }
  }
}

variables {
  root_domain         = "example.com"
  web_domain          = "twitter.example.com"
  api_domain          = "api.example.com"
  acme_email          = "ops@example.com"
  db_username         = "admin_user"
  db_password         = "abcdefghijklmnop1234"
  token_symmetric_key = "0123456789abcdef0123456789abcdef0123"
  google_client_id    = "client.apps.googleusercontent.com"
  gh_token            = "github_pat_mock"
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
    condition     = aws_db_instance.main.storage_encrypted && aws_db_instance.main.backup_retention_period >= 1
    error_message = "RDS must be encrypted and backed up."
  }
  assert {
    condition     = length(aws_route53_record.api) == 0
    error_message = "No Route 53 records without a hosted zone."
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
    condition     = length(aws_route53_record.api) == 1
    error_message = "The API record must be managed when a hosted zone is set."
  }
}

run "rejects_weak_database_password" {
  command = plan
  variables {
    db_password = "short"
  }
  expect_failures = [var.db_password]
}

run "rejects_api_domain_outside_root_domain" {
  command = plan
  variables {
    api_domain = "api.other.org"
  }
  expect_failures = [var.api_domain]
}
