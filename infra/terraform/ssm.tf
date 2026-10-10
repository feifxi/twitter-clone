# Runtime configuration. Each parameter becomes one line of the container's env
# file (see ../ec2/deploy.sh), so names match the API's environment variables.

locals {
  # Values derived from other resources: never empty, often unknown until apply.
  api_env_from_resources = {
    DATABASE_URL      = "postgresql://${var.db_username}:${var.db_password}@${aws_db_instance.main.endpoint}/${var.db_name}?sslmode=require"
    S3_BUCKET_NAME    = aws_s3_bucket.media.id
    CLOUDFRONT_DOMAIN = aws_cloudfront_distribution.media.domain_name
  }

  # Values from variables: known at plan time, possibly empty.
  api_env_from_variables = {
    ENVIRONMENT         = "production"
    ALLOWED_ORIGINS     = local.site_origin
    LOG_LEVEL           = var.log_level
    HTTP_SERVER_ADDRESS = "0.0.0.0:8080"
    DB_MAX_CONNS        = tostring(var.db_max_conns)
    # No REDIS_URL: one instance keeps rate limits and live updates in memory.
    # No COOKIE_DOMAIN: the session cookies belong to the one host.
    TOKEN_SYMMETRIC_KEY    = var.token_symmetric_key
    ACCESS_TOKEN_TTL       = var.access_token_ttl
    REFRESH_TOKEN_TTL      = var.refresh_token_ttl
    GOOGLE_CLIENT_ID       = var.google_client_id
    S3_REGION              = var.aws_region
    MAX_MEDIA_BYTES        = tostring(var.max_media_bytes)
    MAX_AVATAR_BYTES       = tostring(var.max_avatar_bytes)
    MAX_BANNER_BYTES       = tostring(var.max_banner_bytes)
    GEMINI_API_KEY         = var.gemini_api_key
    GEMINI_CHAT_MODEL      = var.gemini_chat_model
    GEMINI_EMBEDDING_MODEL = var.gemini_embedding_model
    ENABLE_RAG             = tostring(var.enable_rag)
  }

  monitoring_env = {
    GRAFANA_PROMETHEUS_URL  = var.grafana_cloud_prometheus_url
    GRAFANA_PROMETHEUS_USER = var.grafana_cloud_prometheus_user
    GRAFANA_LOKI_URL        = var.grafana_cloud_loki_url
    GRAFANA_LOKI_USER       = var.grafana_cloud_loki_user
    GRAFANA_API_TOKEN       = var.grafana_cloud_api_token
  }
}

# SSM rejects empty values and the API treats a missing variable as empty, so empty
# settings are skipped. The filter only looks at plan-time values: for_each keys
# must be known before apply. nonsensitive() reveals only whether a secret is empty.
resource "aws_ssm_parameter" "api" {
  for_each = merge(
    { for k, v in local.api_env_from_variables : k => v if nonsensitive(v != "") },
    local.api_env_from_resources,
  )
  name  = "${local.api_ssm_path}${each.key}"
  type  = "SecureString"
  value = each.value
}

resource "aws_ssm_parameter" "monitoring" {
  for_each = { for k, v in local.monitoring_env : k => v if local.monitoring_enabled && nonsensitive(v != "") }
  name     = "${local.monitoring_ssm_path}${each.key}"
  type     = "SecureString"
  value    = each.value
}
