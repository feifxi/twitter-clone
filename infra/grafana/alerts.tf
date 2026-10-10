locals {
  prometheus = data.grafana_data_source.prometheus.uid

  # Each rule: a PromQL expression and the threshold that makes it fire.
  # "no_data" says what silence means: OK for ratios that need traffic, Alerting for
  # signals that should always be there.
  rules = {
    "Site error rate" = {
      summary   = "More than 5% of requests answered 5xx (last 5 minutes)"
      expr      = <<-EOT
        sum(rate(nginx_http_requests_total{status=~"5.."}[5m]))
          / sum(rate(nginx_http_requests_total[5m]))
        and sum(rate(nginx_http_requests_total[5m])) > 0.05
      EOT
      op        = "gt"
      threshold = 0.05
      for       = "5m"
      severity  = "critical"
      no_data   = "OK"
    }
    "Slow responses" = {
      summary   = "95th percentile response time above 1 second (event streams excluded)"
      expr      = <<-EOT
        histogram_quantile(0.95, sum by (le) (rate(nginx_http_request_duration_seconds_bucket{upstream=~"api|web"}[5m])))
        and sum(rate(nginx_http_requests_total{upstream=~"api|web"}[5m])) > 0.05
      EOT
      op        = "gt"
      threshold = 1
      for       = "10m"
      severity  = "warning"
      no_data   = "OK"
    }
    "API down" = {
      summary   = "Prometheus can't scrape the API (or no metrics arrive at all: instance or Alloy down)"
      expr      = "min(up{job=\"twitter-api\"})"
      op        = "lt"
      threshold = 1
      for       = "5m"
      severity  = "critical"
      no_data   = "Alerting"
    }
    "Disk almost full" = {
      summary   = "A disk is more than 85% full"
      expr      = "max(1 - node_filesystem_avail_bytes{fstype=~\"ext4|xfs\"} / node_filesystem_size_bytes{fstype=~\"ext4|xfs\"})"
      op        = "gt"
      threshold = 0.85
      for       = "15m"
      severity  = "warning"
      no_data   = "OK"
    }
    "Memory almost full" = {
      summary   = "Less than 10% of memory available for 10 minutes"
      expr      = "max(1 - node_memory_MemAvailable_bytes / node_memory_MemTotal_bytes)"
      op        = "gt"
      threshold = 0.9
      for       = "10m"
      severity  = "warning"
      no_data   = "OK"
    }
    "Embedding jobs failing" = {
      summary   = "Tweets whose embedding gave up after every retry (see embedding_jobs.last_error)"
      expr      = "max(embedding_jobs{state=\"failed\"})"
      op        = "gt"
      threshold = 0
      for       = "0s"
      severity  = "warning"
      no_data   = "OK"
    }
    "Embedding backlog" = {
      summary   = "More than 100 tweets waiting to be embedded for 30 minutes (Gemini down or quota?)"
      expr      = "max(embedding_jobs{state=\"pending\"})"
      op        = "gt"
      threshold = 100
      for       = "30m"
      severity  = "warning"
      no_data   = "OK"
    }
  }
}

resource "grafana_rule_group" "twitter" {
  name             = "twitter"
  folder_uid       = grafana_folder.twitter.uid
  interval_seconds = 60

  dynamic "rule" {
    for_each = local.rules
    content {
      name           = rule.key
      condition      = "C"
      for            = rule.value.for
      no_data_state  = rule.value.no_data
      exec_err_state = "Error"
      labels         = { severity = rule.value.severity }
      annotations = {
        summary       = rule.value.summary
        dashboard_uid = jsondecode(grafana_dashboard.overview.config_json).uid
      }

      data {
        ref_id         = "A"
        datasource_uid = local.prometheus
        relative_time_range {
          from = 600
          to   = 0
        }
        model = jsonencode({ refId = "A", expr = trimspace(rule.value.expr), instant = true })
      }

      data {
        ref_id         = "C"
        datasource_uid = "__expr__"
        relative_time_range {
          from = 0
          to   = 0
        }
        model = jsonencode({
          refId      = "C"
          type       = "threshold"
          expression = "A"
          conditions = [{ evaluator = { type = rule.value.op, params = [rule.value.threshold] } }]
        })
      }
    }
  }
}
