#!/usr/bin/env python3
"""Builds dashboards/overview.json. Edit the panels here, then: python3 infra/observability/grafana/gen_dashboard.py
(CI fails if the JSON is out of date.)"""
import json
from pathlib import Path

PROM = {"type": "prometheus", "uid": "prometheus"}
LOKI = {"type": "loki", "uid": "loki"}
REQUESTS = 'path!~"/metrics|/healthz|/readyz"'
TIMED = 'path!~"/metrics|/healthz|/readyz|/api/v1/events|/api/v1/assistant"'
SITE = 'host!~"grafana\\\\..*|caddy:9180"'  # not Grafana, not Prometheus scraping Caddy
panels, y, pid = [], 0, 0


def nid():
    global pid
    pid += 1
    return pid


def row(title):
    global y
    panels.append({"type": "row", "title": title, "id": nid(), "collapsed": False, "gridPos": {"h": 1, "w": 24, "x": 0, "y": y}})
    y += 1


def stat(title, expr, x, w, unit="percentunit", decimals=2, thresholds=None, desc=""):
    panels.append({
        "type": "stat", "title": title, "id": nid(), "description": desc, "datasource": PROM,
        "gridPos": {"h": 5, "w": w, "x": x, "y": y},
        "targets": [{"refId": "A", "expr": expr, "datasource": PROM}],
        "options": {"reduceOptions": {"calcs": ["lastNotNull"]}, "colorMode": "background", "graphMode": "none", "textMode": "value"},
        "fieldConfig": {"defaults": {"unit": unit, "decimals": decimals, "noValue": "no traffic",
                                     "thresholds": {"mode": "absolute", "steps": thresholds or [{"color": "green", "value": None}]}}, "overrides": []},
    })


def ts(title, targets, x, w, unit="short", desc="", stack=False):
    panels.append({
        "type": "timeseries", "title": title, "id": nid(), "description": desc, "datasource": PROM,
        "gridPos": {"h": 8, "w": w, "x": x, "y": y},
        "targets": [{"refId": chr(65 + i), "expr": e, "legendFormat": l, "datasource": PROM} for i, (e, l) in enumerate(targets)],
        "options": {"legend": {"displayMode": "list", "placement": "bottom"}, "tooltip": {"mode": "multi"}},
        "fieldConfig": {"defaults": {"unit": unit, "custom": {"lineWidth": 2, "fillOpacity": 10, "stacking": {"mode": "normal" if stack else "none"}}}, "overrides": []},
    })


def budget_left(slo, objective):
    # The stack lives for days, not a month: use the longest window kept (3d)
    return f"1 - (slo:{slo}:error_ratio_rate3d / (1 - {objective}))"


def objective_steps(objective):
    return [{"color": "red", "value": None}, {"color": "green", "value": objective}]


BUDGET = [{"color": "red", "value": None}, {"color": "orange", "value": 0.25}, {"color": "green", "value": 0.5}]

row("SLOs: is it good enough for people right now?")
stat("API availability (1h)", "1 - slo:api_availability:error_ratio_rate1h", 0, 5, thresholds=objective_steps(0.995), desc="Target: 99.5% of requests aren't 5xx")
stat("Budget left: availability", budget_left("api_availability", 0.995), 5, 4, decimals=0, thresholds=BUDGET, desc="Error budget left over the last 3 days")
stat("API latency < 500 ms (1h)", "1 - slo:api_latency:error_ratio_rate1h", 9, 5, thresholds=objective_steps(0.99), desc="Target: 99% of requests within 500 ms")
stat("Budget left: latency", budget_left("api_latency", 0.99), 14, 4, decimals=0, thresholds=BUDGET)
stat("Burn rate: availability (1h)", "slo:api_availability:error_ratio_rate1h / (1 - 0.995)", 18, 6, unit="none", decimals=1,
     thresholds=[{"color": "green", "value": None}, {"color": "orange", "value": 1}, {"color": "red", "value": 6}],
     desc="1 = spending the budget exactly on schedule; 14.4 over an hour pages")
y += 5

row("Site (Caddy: every request, pages included)")
ts("Requests / s", [('sum by (host) (rate(caddy_http_request_duration_seconds_count{host!="caddy:9180"}[$__rate_interval]))', "{{host}}")], 0, 8, unit="reqps", stack=True)
ts("Status codes / s", [(f"sum by (code) (rate(caddy_http_request_duration_seconds_count{{{SITE}}}[$__rate_interval]))", "{{code}}")], 8, 8, unit="reqps", stack=True)
ts("Response time", [(f"histogram_quantile({q}, sum by (le) (rate(caddy_http_request_duration_seconds_bucket{{{SITE}}}[$__rate_interval])))", f"p{int(q * 100)}") for q in (0.5, 0.95)],
   16, 8, unit="s", desc="Event streams are included here and stay open for minutes")
y += 8

row("API (by route)")
ts("Requests / s", [(f"sum by (path) (rate(http_requests_total{{{REQUESTS}}}[$__rate_interval]))", "{{path}}")], 0, 8, unit="reqps", stack=True)
ts("5xx / s", [(f'sum by (method, path) (rate(http_requests_total{{{REQUESTS},status=~"5.."}}[$__rate_interval])) > 0', "{{method}} {{path}}")], 8, 8, unit="reqps")
ts("Slowest routes (p95)", [(f"topk(8, histogram_quantile(0.95, sum by (le, method, path) (rate(http_request_duration_seconds_bucket{{{TIMED}}}[$__rate_interval]))))", "{{method}} {{path}}")],
   16, 8, unit="s")
y += 8

row("Background work and runtime")
ts("Embedding queue", [('max(embedding_jobs{state="pending"})', "pending"), ('max(embedding_jobs{state="failed"})', "failed")], 0, 8,
   desc="Tweets waiting for (or given up on) their RAG embedding")
ts("Embedding jobs / min", [("sum by (result) (rate(embedding_jobs_processed_total[$__rate_interval])) * 60", "{{result}}")], 8, 8)
ts("Goroutines and heap", [('go_goroutines{job="api"}', "goroutines"), ('go_memstats_heap_inuse_bytes{job="api"} / 1024 / 1024', "heap MiB")], 16, 8)
y += 8

row("Host")
ts("CPU", [('1 - avg(rate(node_cpu_seconds_total{mode="idle"}[$__rate_interval]))', "used")], 0, 8, unit="percentunit")
ts("Memory", [("1 - node_memory_MemAvailable_bytes / node_memory_MemTotal_bytes", "memory"),
              ("1 - node_memory_SwapFree_bytes / clamp_min(node_memory_SwapTotal_bytes, 1)", "swap")], 8, 8, unit="percentunit")
ts("Disk", [('1 - node_filesystem_avail_bytes{fstype=~"ext4|xfs"} / node_filesystem_size_bytes{fstype=~"ext4|xfs"}', "{{mountpoint}}")], 16, 8, unit="percentunit")
y += 8

row("Logs")
panels.append({
    "type": "logs", "title": "API warnings and errors", "id": nid(), "datasource": LOKI,
    "gridPos": {"h": 10, "w": 24, "x": 0, "y": y},
    "targets": [{"refId": "A", "expr": '{service="api", level=~"warn|error"}', "datasource": LOKI}],
    "options": {"showTime": True, "wrapLogMessage": True, "sortOrder": "Descending", "enableLogDetails": True},
})

dashboard = {
    "uid": "twitter-overview",
    "title": "Twitter overview",
    "tags": ["twitter", "slo"],
    "timezone": "browser",
    "refresh": "30s",
    "time": {"from": "now-3h", "to": "now"},
    "schemaVersion": 39,
    "editable": True,
    "panels": panels,
}
Path(__file__).with_name("dashboards").joinpath("overview.json").write_text(json.dumps(dashboard, indent=2) + "\n")
print(f"wrote dashboards/overview.json ({len(panels)} panels)")
