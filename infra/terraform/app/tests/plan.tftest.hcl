# Plans the whole stack against a mocked AWS provider: no credentials, no cost. Catches
# evaluation errors `terraform validate` can't (for_each keys, conditions, templates).
#   terraform init && terraform test

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
  mock_data "aws_iam_policy_document" {
    defaults = { json = "{}" }
  }
  mock_resource "aws_iam_role" {
    defaults = { arn = "arn:aws:iam::123456789012:role/mock" }
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
  mock_resource "aws_cloudfront_distribution" {
    defaults = { arn = "arn:aws:cloudfront::123456789012:distribution/MOCK", domain_name = "d123.cloudfront.net" }
  }
  mock_resource "aws_eip" {
    defaults = { public_ip = "203.0.113.10" }
  }
}

variables {
  delegation_set_id = "N0123456789"
  alert_email       = "ops@example.com"
  google_client_id  = "client.apps.googleusercontent.com"
  domain            = "twitter.example.com"
}

run "plans_with_minimal_settings" {
  command = plan

  assert {
    condition     = !contains(keys(aws_ssm_parameter.api), "GEMINI_API_KEY") && length(aws_ssm_parameter.discord_webhook) == 0
    error_message = "Empty settings must not be stored (SSM rejects empty values)."
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
    condition     = toset(keys(aws_ssm_parameter.stack)) == toset(["POSTGRES_PASSWORD", "GF_ADMIN_PASSWORD"])
    error_message = "The compose file's secrets must be stored."
  }
  assert {
    condition     = toset(keys(aws_ecr_repository.app)) == toset(["api", "web"]) && aws_ecr_repository.app["api"].image_tag_mutability == "IMMUTABLE"
    error_message = "Both images need a repository with immutable tags (rollback = redeploy an old SHA)."
  }
  assert {
    condition     = toset([for r in aws_route53_record.host : r.name]) == toset(["twitter.example.com", "grafana.twitter.example.com"])
    error_message = "The site and Grafana both need a DNS record."
  }
  assert {
    condition     = aws_route53_zone.main.delegation_set_id == "N0123456789"
    error_message = "The zone must use bootstrap's delegation set, or the NS records break on every up."
  }
  assert {
    condition     = toset([for r in aws_vpc_security_group_ingress_rule.host : r.from_port]) == toset([80, 443])
    error_message = "Only Caddy's ports may be open (no SSH)."
  }
  assert {
    condition     = aws_instance.host.metadata_options[0].http_tokens == "required" && aws_instance.host.root_block_device[0].encrypted
    error_message = "IMDSv2 only, encrypted disk."
  }
  assert {
    condition     = aws_s3_bucket_public_access_block.deploy.restrict_public_buckets
    error_message = "Releases must be private."
  }
}

run "stores_optional_settings_when_set" {
  command = plan

  variables {
    gemini_api_key      = "gemini-key"
    discord_webhook_url = "https://discord.com/api/webhooks/1/x"
  }

  assert {
    condition     = contains(keys(aws_ssm_parameter.api), "GEMINI_API_KEY") && length(aws_ssm_parameter.discord_webhook) == 1
    error_message = "Set optional settings must be stored."
  }
}

# Apply against the mocks (still no AWS calls): user data and the deploy config are only known after apply.
run "host_and_deploy_config" {
  command = apply

  assert {
    condition     = strcontains(aws_instance.host.user_data, "docker-compose-linux-aarch64") && strcontains(aws_instance.host.user_data, "API_SSM_PATH=/chmtwt/api/")
    error_message = "First boot must install the arm64 Compose plugin and know where the config is."
  }
  assert {
    condition     = jsondecode(aws_ssm_parameter.deploy_config.value).domain == "twitter.example.com" && aws_ssm_parameter.deploy_config.name == "/chmtwt/deploy/config"
    error_message = "The deploy config must be where the deploy script and the GitHub role look for it."
  }
  assert {
    condition     = contains(aws_cloudwatch_metric_alarm.instance["system-check"].alarm_actions, "arn:aws:automate:ap-southeast-1:ec2:recover")
    error_message = "A failed host must be recovered automatically."
  }
}

run "rejects_x86_instances" {
  command = plan
  variables {
    instance_type = "t3.medium"
  }
  expect_failures = [var.instance_type]
}

run "rejects_a_url_as_domain" {
  command = plan
  variables {
    domain = "https://twitter.example.com"
  }
  expect_failures = [var.domain]
}

run "rejects_an_invalid_alert_email" {
  command = plan
  variables {
    alert_email = "not-an-email"
  }
  expect_failures = [var.alert_email]
}
