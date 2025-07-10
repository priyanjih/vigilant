# 🛡️ Vigilant

**Vigilant** is a local observability early-warning system that combines Prometheus alerts, log pattern matching, and basic metric thresholding. It detects issues *before* your services catch fire.

Built for resource-constrained environments, burnout-prone teams, and engineers tired of being paged *after* things break.

---

## 🚀 What It Does (So Far)

- Connects to **Prometheus** and fetches active alerts
- Tracks active risk services/nodes with a TTL
- Matches known failure **log patterns** (per service)
- Runs **metric threshold checks** using PromQL
- Outputs clean CLI summaries

---

## 🔧 What's Coming

- ✨ LLM-powered root cause summarizer (local, lightweight models)
- ✨ Rule-based prediction engine (fallout prediction, confidence scoring)
- ✨ JSON & Grafana export support
- ✨ Live dashboard view

---

## ⚙️ Getting Started

You need:
- Go 1.21+
- Prometheus running at `localhost:9090`

Clone this repo and run:

```bash
go run ./cmd/vigilant
```

By default, it will:
- Load alert data from Prometheus
- Track any services flagged by alerts
- Load per-service configs from `config/services.yml`
- Use that to get log file + PromQL metric checks

---

## 🔍 Example Service Config (`config/services.yml`)

```yaml
services:
  hotrod:
    log_file: /home/user/copilot-stack/hotrod.log
    metrics:
      - name: HotrodTraffic
        query_tpl: sum(hotrod_http_requests_total)
        operator: ">"
        threshold: 100
        weight: 1
    log_patterns:
      - label: trace_export_fail
        regex: "traces export.*connect"
      - label: timeout
        regex: "timeout"

  node:
    log_file: /var/log/syslog
    metrics:
      - name: RAMUsage
        query_tpl: node_memory_Active_bytes
        operator: ">"
        threshold: 4.5e+9
        weight: 2
    log_patterns:
      - label: cpu_hog
        regex: "hogged CPU for >[0-9]+us"
```

---

## ✅ Status Summary

| Feature                    | Status     |
|----------------------------|------------|
| Prometheus Alert Fetcher  | ✅ Done     |
| TTL Risk Tracker          | ✅ Done     |
| Log Pattern Scanner       | ✅ Done     |
| Metrics Check Engine      | ✅ Done     |
| Config-Driven Profiles    | ✅ Done     |
| LLM Integration           | ⏳ Planned  |
| Fallout Prediction Engine | ⏳ Planned  |
| Export Options            | ⏳ Planned  |

---



