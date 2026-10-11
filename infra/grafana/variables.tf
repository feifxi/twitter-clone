variable "grafana_url" {
  description = "Your Grafana Cloud stack, e.g. https://chanombude.grafana.net"
  type        = string
}

variable "grafana_auth" {
  description = "Service account token (Administration → Service accounts, role Editor)"
  type        = string
  sensitive   = true
}

variable "prometheus_datasource_name" {
  description = "The stack's Prometheus data source (Connections → Data sources), e.g. grafanacloud-chanombude-prom"
  type        = string
}

variable "alert_email" {
  description = "Where alerts are emailed"
  type        = string
}
