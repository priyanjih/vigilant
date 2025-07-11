# 🔍 Vigilant

> **Intelligent observability for the modern engineer**

**Vigilant** is a lightweight, AI-powered observability system that correlates alerts, logs, and metrics to detect anomalies and explain root causes automatically. Built for developers who want smart monitoring without the complexity of enterprise solutions.

## ✨ Features

🚨 **Smart Alert Correlation** - Automatically connects related alerts across services  
📊 **Multi-Source Analysis** - Prometheus metrics + log pattern matching + custom checks  
🤖 **AI-Powered Root Cause** - LLM integration for intelligent incident summaries  
⚡ **Real-time Risk Tracking** - Service health scoring with automatic cleanup  
🎯 **Zero-Config Start** - Works out of the box with minimal setup  
🔌 **Pluggable Architecture** - Extensible for custom data sources and LLM backends  

## 🎯 Perfect For

- **Personal Projects** - Monitor your side projects intelligently
- **Home Labs** - Keep your infrastructure healthy
- **Prototyping** - Build smart monitoring proof-of-concepts
- **Learning** - Explore observability and AI integration
- **Internal Tools** - Lightweight monitoring for small teams

## 🚀 Quick Start

### Prerequisites

- Go 1.20+
- Prometheus instance (local or remote)
- Linux/macOS system

### Installation

```bash
# Clone the repository
git clone https://github.com/priyanjih/vigilant.git
cd vigilant

# Copy environment template
cp .env.example .env

# Install dependencies
go mod download

# Run Vigilant
go run main.go
```

### Basic Configuration

1. **Configure your services** in `config/services/`:

```yaml
# config/services/web-api.yml
log_file: /var/log/web-api.log
metrics:
  - name: ErrorRate
    query_tpl: 'rate(http_requests_total{job="{{.Service}}",code=~"5.."}[5m])'
    operator: ">"
    threshold: 0.05
    weight: 2
  - name: ResponseTime
    query_tpl: 'histogram_quantile(0.95, http_request_duration_seconds_bucket{job="{{.Service}}"})'
    operator: ">"
    threshold: 0.5
    weight: 1
log_patterns:
  - label: database_error
    regex: '(?i)(database|sql|connection).*(error|failed|timeout)'
  - label: memory_issue
    regex: '(?i)(out of memory|oom|memory leak)'
```

2. **Set up environment variables**:

```bash
# .env
PROM_URL=http://localhost:9090
OPENAI_API_KEY=your_openai_key_here  # Optional, for LLM summaries

```

3. **Start monitoring**:

```bash
go run main.go
```

## 📊 Example Output

```
Starting Vigilant...
Fetching alerts...
[ALERT] High CPU usage detected on web-api (severity: warning)
[SYMPTOM] database_error matched on web-api (15 times)
[METRIC] ErrorRate triggered for web-api: 0.12 > 0.05

=== Root Cause Summary ===
🔍 Analysis: The web-api service is experiencing elevated error rates (12% vs 5% threshold) 
combined with 15 database connection errors in the logs. This suggests a database connectivity 
issue causing cascading failures.

💡 Recommendation: Check database connection pool settings and network connectivity between 
web-api and database services.
```

## 🏗️ Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Prometheus    │    │   Log Files     │    │  Custom Checks  │
│     Alerts      │    │   Scanning      │    │   & Metrics     │
└─────────┬───────┘    └─────────┬───────┘    └─────────┬───────┘
          │                      │                      │
          └──────────────────────┼──────────────────────┘
                                 │
                    ┌─────────────▼─────────────┐
                    │      Risk Tracker        │
                    │   (Correlation Engine)   │
                    └─────────────┬─────────────┘
                                 │
                    ┌─────────────▼─────────────┐
                    │     LLM Summarizer       │
                    │   (Root Cause Analysis)  │
                    └─────────────┬─────────────┘
                                 │
                    ┌─────────────▼─────────────┐
                    │      API Server          │
                    │   (Dashboard & Webhooks) │
                    └───────────────────────────┘
```

## 🔧 Configuration

### Service Profiles

Each service gets its own YAML configuration file:

```yaml
# config/services/my-service.yml
log_file: "/path/to/service.log"
metrics:
  - name: "CPUUsage"
    query_tpl: 'cpu_usage{service="{{.Service}}"}'
    operator: ">"
    threshold: 80
    weight: 1
log_patterns:
  - label: "error"
    regex: '(?i)error|exception|failed'
  - label: "timeout"
    regex: '(?i)timeout|timed out'
```

## 🛠️ Development

Still in very basic stage. 

