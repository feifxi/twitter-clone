# Plans against a mocked Grafana provider: no stack or token needed.
#   terraform init -backend=false && terraform test

mock_provider "grafana" {
  mock_data "grafana_data_source" {
    defaults = { uid = "prom-uid" }
  }
  mock_resource "grafana_folder" {
    defaults = { uid = "folder-uid" }
  }
}

variables {
  grafana_url                = "https://example.grafana.net"
  grafana_auth               = "glsa_mock"
  prometheus_datasource_name = "grafanacloud-example-prom"
  alert_email                = "ops@example.com"
}

run "plans_alerts_and_dashboard" {
  command = plan

  assert {
    condition     = length(grafana_rule_group.twitter.rule) == 7
    error_message = "Every alert rule must be planned."
  }
  assert {
    condition     = alltrue([for r in grafana_rule_group.twitter.rule : r.data[0].datasource_uid == "prom-uid" && r.condition == "C"])
    error_message = "Rules must query the Prometheus data source and fire on the threshold."
  }
  assert {
    condition     = one([for r in grafana_rule_group.twitter.rule : r.no_data_state if r.name == "API down"]) == "Alerting"
    error_message = "Missing API metrics (instance or agent down) must alert."
  }
  assert {
    condition     = jsondecode(grafana_dashboard.overview.config_json).uid == "twitter-overview"
    error_message = "The dashboard must keep a stable uid (alerts link to it)."
  }
}
