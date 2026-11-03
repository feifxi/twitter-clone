variable "project" {
  description = "Prefix of every resource name; must match bootstrap's"
  type        = string
  default     = "chmtwt"
}

variable "aws_region" {
  type    = string
  default = "ap-southeast-1"
}

variable "domain" {
  description = "The site's hostname (grafana.<domain> is created under it)"
  type        = string
  default     = "twitter.chanombude.me"
  validation {
    condition     = can(regex("^[a-z0-9.-]+\\.[a-z]{2,}$", var.domain))
    error_message = "A bare hostname such as twitter.example.com (no scheme, no path)."
  }
}

variable "delegation_set_id" {
  description = "From infra/terraform/bootstrap (terraform output delegation_set_id)"
  type        = string
}

variable "alert_email" {
  description = "Gets AWS Budget and CloudWatch alarm emails (confirm the SNS email after each up)"
  type        = string
  validation {
    condition     = can(regex("^[^@\\s]+@[^@\\s]+\\.[^@\\s]+$", var.alert_email))
    error_message = "A valid email address."
  }
}

variable "discord_webhook_url" {
  description = "Alertmanager posts app alerts here; empty = alerts only show in Alertmanager/Grafana"
  type        = string
  default     = ""
  sensitive   = true
}

variable "google_client_id" {
  description = "OAuth client ID for Google sign-in (public: it's compiled into the web app)"
  type        = string
}

variable "gemini_api_key" {
  description = "Enables the AI assistant; empty = off"
  type        = string
  default     = ""
  sensitive   = true
}

variable "instance_type" {
  description = "Runs the app, Postgres and the monitoring stack: 4 GB of memory is comfortable"
  type        = string
  default     = "t4g.medium"
  validation {
    condition     = can(regex("^(t4g|m7g|c7g|m8g|c8g)\\.", var.instance_type))
    error_message = "Use an arm64 (Graviton) instance: the images are built for arm64."
  }
}

variable "monthly_budget_usd" {
  description = "AWS Budget: emails at 50% actual and 100% forecast spend"
  type        = number
  default     = 20
}

variable "dev_origins" {
  description = "Extra origins allowed to upload media to the bucket, e.g. http://localhost:3000"
  type        = list(string)
  default     = []
}
