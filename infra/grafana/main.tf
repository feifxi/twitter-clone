# Grafana Cloud as code: the dashboard, the alert rules and where alerts go. Applied
# separately from ../terraform (needs a Grafana token, not AWS credentials).
#
# The metrics come from Alloy on the instance (../ec2/config.alloy):
#   nginx_http_*           every request to the site, by upstream (api, web, stream) and status
#   http_request_*         the API, by route
#   embedding_jobs*        the embedding queue
#   node_*                 the host

data "grafana_data_source" "prometheus" {
  name = var.prometheus_datasource_name
}

resource "grafana_folder" "twitter" {
  title = "Twitter"
}

resource "grafana_dashboard" "overview" {
  folder      = grafana_folder.twitter.uid
  config_json = file("${path.module}/dashboard.json")
}

resource "grafana_contact_point" "email" {
  name = "email"
  email {
    addresses               = [var.alert_email]
    disable_resolve_message = false
  }
}

# The stack's root policy: every alert goes to email, grouped per rule, repeated every 4h
# while it keeps firing.
resource "grafana_notification_policy" "root" {
  contact_point   = grafana_contact_point.email.name
  group_by        = ["alertname"]
  group_wait      = "30s"
  group_interval  = "5m"
  repeat_interval = "4h"
}
