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

# ── Domains ─────────────────────────────────────────
variable "root_domain" {
  description = "Registrable domain shared by the web app and API, e.g. chanombude.me"
  type        = string
}

variable "web_domain" {
  description = "Web app hostname (Amplify), e.g. twitter.chanombude.me"
  type        = string
  validation {
    condition     = endswith(var.web_domain, ".${var.root_domain}")
    error_message = "web_domain must be a subdomain of root_domain."
  }
}

variable "api_domain" {
  description = "API hostname served by Caddy on EC2, e.g. api.chanombude.me"
  type        = string
  validation {
    condition     = endswith(var.api_domain, ".${var.root_domain}")
    error_message = "api_domain must be a subdomain of root_domain (the session cookie is scoped to it)."
  }
}

variable "acme_email" {
  description = "Email Let's Encrypt uses for certificate expiry notices"
  type        = string
}

variable "route53_zone_id" {
  description = "Route 53 hosted zone of root_domain. Empty = manage DNS yourself (see the dns_records output)."
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
  type    = string
  default = "t3.micro"
}

variable "docker_compose_version" {
  description = "Docker Compose plugin release installed on the host"
  type        = string
  default     = "v2.39.1"
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
  description = "Branch Amplify builds and GitHub Actions may deploy from"
  type        = string
  default     = "main"
}

variable "gh_token" {
  description = "GitHub token Amplify uses to read the repository"
  type        = string
  sensitive   = true
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
