# Runtime config and secrets. Every secret is generated here (nothing to invent or paste) and
# stored as an SSM SecureString; the host renders each path into an env file at deploy time.
# Each parameter is one line of that file, so the names are the environment variable names.

resource "random_password" "db" {
  length  = 32
  special = false # goes into a URL
}

resource "random_password" "token_key" {
  length  = 64
  special = false
}

resource "random_password" "grafana" {
  length  = 24
  special = false
}

locals {
  # The API container's environment
  api_env = merge(
    {
      ENVIRONMENT         = "production"
      ALLOWED_ORIGINS     = local.site_origin
      DATABASE_URL        = "postgres://twitter:${random_password.db.result}@postgres:5432/twitter?sslmode=disable" # the compose network
      TOKEN_SYMMETRIC_KEY = random_password.token_key.result
      GOOGLE_CLIENT_ID    = var.google_client_id
      S3_REGION           = var.aws_region
      S3_BUCKET_NAME      = aws_s3_bucket.media.id
      CLOUDFRONT_DOMAIN   = aws_cloudfront_distribution.media.domain_name
      # No REDIS_URL: one instance keeps rate limits and live updates in memory.
    },
    # SSM rejects empty values, and an unset variable means "off" to the API
    nonsensitive(var.gemini_api_key != "") ? { GEMINI_API_KEY = var.gemini_api_key } : {},
  )

  # Variables the compose file itself uses
  stack_env = {
    POSTGRES_PASSWORD = random_password.db.result
    GF_ADMIN_PASSWORD = random_password.grafana.result
  }
}

resource "aws_ssm_parameter" "api" {
  for_each = nonsensitive(toset(keys(local.api_env)))
  name     = "${local.api_ssm_path}${each.key}"
  type     = "SecureString"
  value    = local.api_env[each.key]
}

resource "aws_ssm_parameter" "stack" {
  for_each = nonsensitive(toset(keys(local.stack_env)))
  name     = "${local.stack_ssm_path}${each.key}"
  type     = "SecureString"
  value    = local.stack_env[each.key]
}

resource "aws_ssm_parameter" "discord_webhook" {
  count = nonsensitive(var.discord_webhook_url != "") ? 1 : 0
  name  = "/${var.project}/discord_webhook_url"
  type  = "SecureString"
  value = var.discord_webhook_url
}
