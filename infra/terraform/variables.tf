# ── Project ─────────────────────────────────────────
variable "project_name" {
  description = "Prefix for resource names"
  type        = string
  default     = "chmtwt"
}

variable "environment" {
  description = "Environment name (prod, staging, …)"
  type        = string
  default     = "prod"
}

variable "aws_region" {
  type    = string
  default = "ap-southeast-1"
}

variable "vpc_cidr" {
  type    = string
  default = "10.0.0.0/16"
}

# ── Domain ──────────────────────────────────────────
variable "domain" {
  description = "Public hostname of the app, e.g. twitter.chanombude.me (the web app, with the API under /api)"
  type        = string
  validation {
    condition     = can(regex("^([a-z0-9-]+\\.)+[a-z]{2,}$", var.domain))
    error_message = "domain must be a lowercase hostname like twitter.example.com."
  }
}

variable "acme_email" {
  description = "Email Let's Encrypt uses for certificate expiry notices"
  type        = string
}

variable "alert_email" {
  description = "Where alerts go (site down, database and instance alarms). Confirm the subscription email AWS sends."
  type        = string
  validation {
    condition     = can(regex("^[^@\\s]+@[^@\\s]+$", var.alert_email))
    error_message = "alert_email must be an email address."
  }
}

variable "route53_zone_id" {
  description = "Route 53 hosted zone that contains domain. Empty = manage DNS yourself (see the dns_records output)."
  type        = string
  default     = ""
}

variable "dev_origins" {
  description = "Extra browser origins allowed to upload to the media bucket (e.g. http://localhost:3000)"
  type        = list(string)
  default     = []
}

# ── Compute ─────────────────────────────────────────
variable "ec2_instance_type" {
  description = "Runs nginx, Next.js, the API and the monitoring agents; 2 GB is comfortable"
  type        = string
  default     = "t3.small"
}

variable "docker_compose_version" {
  description = "Docker Compose plugin release installed on the host"
  type        = string
  default     = "v2.39.1"
}

variable "docker_compose_sha256" {
  description = "SHA-256 of docker-compose-linux-x86_64 for docker_compose_version (from the release's .sha256 file)"
  type        = string
  default     = "a5ea28722d5da628b59226626f7d6c33c89a7ed19e39f750645925242044c9d2"
}

variable "patch_schedule" {
  description = "When OS patches are installed (UTC). Default: Sunday 20:00 UTC = Monday 03:00 in Bangkok"
  type        = string
  default     = "cron(0 20 ? * SUN *)"
}

variable "ecr_images_to_keep" {
  type    = number
  default = 10
}

# ── Database ────────────────────────────────────────
variable "rds_instance_class" {
  type    = string
  default = "db.t4g.micro"
}

variable "db_name" {
  type    = string
  default = "twitter_db"
}

variable "db_username" {
  type      = string
  sensitive = true
}

variable "db_password" {
  description = "RDS master password"
  type        = string
  sensitive   = true
  validation {
    # It is embedded in a URL and in a single-quoted env file line.
    condition     = length(var.db_password) >= 16 && can(regex("^[A-Za-z0-9_-]+$", var.db_password))
    error_message = "db_password must be at least 16 characters of letters, digits, '_' or '-'."
  }
}

variable "db_max_conns" {
  type    = number
  default = 25
}

# ── API settings ────────────────────────────────────
variable "log_level" {
  type    = string
  default = "info"
}

variable "token_symmetric_key" {
  description = "JWT signing key (32+ characters). Generate with: openssl rand -base64 48"
  type        = string
  sensitive   = true
  validation {
    condition     = length(var.token_symmetric_key) >= 32 && !strcontains(var.token_symmetric_key, "'")
    error_message = "token_symmetric_key must be at least 32 characters and must not contain a single quote."
  }
}

variable "access_token_ttl" {
  type    = string
  default = "15m"
}

variable "refresh_token_ttl" {
  type    = string
  default = "720h"
}

variable "google_client_id" {
  description = "Google OAuth web client ID"
  type        = string
}

variable "max_media_bytes" {
  type    = number
  default = 104857600 # 100 MiB
}

variable "max_avatar_bytes" {
  type    = number
  default = 5242880 # 5 MiB
}

variable "max_banner_bytes" {
  type    = number
  default = 10485760 # 10 MiB
}

variable "cloudfront_price_class" {
  description = "PriceClass_200 covers Asia, Europe and North America"
  type        = string
  default     = "PriceClass_200"
}

# ── Assistant / RAG ─────────────────────────────────
variable "gemini_api_key" {
  description = "Empty disables the assistant and RAG"
  type        = string
  sensitive   = true
  default     = ""
}

variable "gemini_chat_model" {
  type    = string
  default = "gemini-2.5-flash"
}

variable "gemini_embedding_model" {
  type    = string
  default = "gemini-embedding-2-preview"
}

variable "enable_rag" {
  type    = bool
  default = true
}

# ── GitHub ──────────────────────────────────────────
variable "gh_repo_url" {
  type    = string
  default = "https://github.com/feifxi/twitter-monolith"
}

variable "gh_branch" {
  description = "Branch GitHub Actions may deploy from"
  type        = string
  default     = "main"
}

variable "create_github_oidc_provider" {
  description = "false if the account already has the GitHub OIDC provider (only one is allowed per account)"
  type        = bool
  default     = true
}

# ── Grafana Cloud (optional) ────────────────────────
variable "grafana_cloud_prometheus_url" {
  type    = string
  default = ""
}

variable "grafana_cloud_prometheus_user" {
  type    = string
  default = ""
}

variable "grafana_cloud_loki_url" {
  type    = string
  default = ""
}

variable "grafana_cloud_loki_user" {
  type    = string
  default = ""
}

variable "grafana_cloud_api_token" {
  description = "Empty disables monitoring"
  type        = string
  sensitive   = true
  default     = ""
}
