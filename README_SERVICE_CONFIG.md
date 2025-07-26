# 📋 Vigilant Service Configuration System

## 🚀 Quick Start

### Generate a New Service Configuration
```bash
# Basic service
./vigilant-config-gen -name="MyAPI" -desc="REST API service"

# Advanced service with custom settings  
./vigilant-config-gen \
  -name="PaymentService" \
  -desc="Payment processing microservice" \
  -tags="payment,api,critical" \
  -namespace="payments" \
  -type="microservice" \
  -criticality="critical"
```

### Run with Environment Variables
```bash
ES_INDEX_PATTERN="fluentbit-*" \
ISTIO_NAMESPACE="production" \
./vigilant -llm=false
```

## 🏗️ Configuration Structure

```yaml
---
# Service Identity
name: "PaymentAPI"                    # 🔑 Primary identifier
description: "Payment processing API"  # 📝 Human description
alert_pattern: "PaymentServiceDown"   # 🚨 Prometheus alert name

# Data Sources
data_sources:
  elasticsearch:
    index_pattern: "${ES_INDEX_PATTERN:-fluentbit-*}"  # 🔄 Environment variables
    namespace_filter: "${PAYMENT_NAMESPACE:-payments}"
    time_range_minutes: 15
  log_file: "/var/log/payment-api.log"  # 📄 Fallback

# Pattern Detection  
log_patterns:
  - name: "payment_failure"           # 🎯 Pattern name
    description: "Payment failures"   # 📖 What it detects
    regex: "(?i)payment.*failed"     # 🔍 Detection regex
    severity: "critical"              # ⚠️ Impact level

# Metrics Monitoring
metrics:
  - name: "PaymentSuccessRate"        # 📊 Metric name
    query_tpl: 'payment_success_rate{service="payments"}'  # 📈 PromQL query
    operator: "<"                     # 🔢 Comparison
    threshold: 95.0                   # 🎯 Threshold value
    weight: 10                        # ⚖️ Risk weight
```

## 🔄 Service Identification Flow

```
1. Prometheus Alert: "PaymentServiceDown" 
   ↓
2. Alert Pattern Lookup: "PaymentServiceDown" → "PaymentAPI"
   ↓  
3. Load Configuration: name: "PaymentAPI"
   ↓
4. Process with enhanced config
```

## 🛠️ Key Features

### ✅ **Smart Service Identification**
- **Primary**: Uses `name` field (not filename)
- **Mapping**: `alert_pattern` → service name
- **Fallback**: Backward compatibility with legacy configs

### ✅ **Environment Variable Support**
- **Syntax**: `${VAR}` or `${VAR:-default}`
- **Dynamic**: Environment-specific configurations
- **Flexible**: Runtime configuration changes

### ✅ **Rich Metadata**
- **Documentation**: Description, version, tags
- **Ownership**: Maintainer and team information
- **Context**: Service type and criticality for AI

### ✅ **Comprehensive Validation**
- **Schema**: Required field validation
- **Regex**: Pattern compilation checking
- **PromQL**: Metric query validation
- **Duplicates**: Service name conflict detection

## 📁 File Organization

```
config/services/
├── PaymentAPI.yml          # Payment service
├── UserService.yml         # User management  
├── DatabaseCluster.yml     # Database monitoring
└── KubernetesInfra.yml     # Infrastructure
```

## 🎯 Best Practices

### ✅ **Naming Conventions**
```yaml
# Clear, descriptive names
name: "PaymentAPI"
alert_pattern: "PaymentServiceDown"

# NOT generic names
name: "Service1"
```

### ✅ **Environment Variables**
```yaml
# Use env vars for environment-specific values
namespace_filter: "${PAYMENT_NAMESPACE:-payments}"

# NOT hardcoded values
namespace_filter: "prod-payments-2024"
```

### ✅ **Meaningful Patterns**
```yaml
# Specific, documented patterns
- name: "payment_timeout"
  description: "Payment processing timeouts with transaction context"
  regex: "(?i)payment.*timeout.*transaction[_\\s]+([a-zA-Z0-9-]+)"

# NOT overly broad patterns  
- name: "error"
  regex: "error"
```

## 🔧 Configuration Tools

### **Config Generator**
```bash
./vigilant-config-gen -name="DatabaseService" -type="database"
```

### **Environment Testing**
```bash
# Test environment variable substitution
ES_INDEX_PATTERN="test-logs-*" ./vigilant -llm=false
```

### **Validation Check**
```bash
# Check configuration loading
./vigilant -llm=false | head -20
```

## 📊 Supported Service Types

| **Type** | **Example** | **Criticality** |
|----------|-------------|-----------------|
| `microservice` | REST API, GraphQL | High |
| `database` | PostgreSQL, MongoDB | Critical |
| `infrastructure` | Load Balancer, Cache | High |
| `kubernetes_infrastructure` | API Server, etcd | Critical |
| `service_mesh_proxy` | Istio, Linkerd | High |
| `monitoring` | Prometheus, Grafana | Medium |

## 🚨 Common Issues & Solutions

### **Service Not Found**
```
No profile found for alert 'MyAlert'
```
**Solution**: Check `alert_pattern` matches Prometheus alert name exactly

### **Invalid Configuration**  
```
Warning: invalid configuration: log pattern missing regex
```
**Solution**: Ensure all required fields are present and valid

### **Environment Variables Not Working**
```
ES scan: index=${ES_INDEX_PATTERN:-fluentbit-*}
```
**Solution**: Set environment variables before running Vigilant

## 📚 Documentation

- **[Complete Guide](docs/SERVICE_CONFIGURATION.md)** - Detailed configuration reference
- **[Flow Diagrams](docs/FLOW_DIAGRAM.md)** - System architecture and data flow
- **[Examples](config/services/)** - Real service configurations

## 🎉 What's New in Enhanced Format

### **Before (Legacy)**
```yaml
log_file: /var/log/app.log
log_patterns:
  - label: timeout
    regex: "(?i)timeout"
metrics:
  - name: Traffic
    query_tpl: sum(requests_total)
    operator: ">"
    threshold: 30
```

### **After (Enhanced)**
```yaml
name: "MyAPI"                         # 🆕 Service identity
description: "REST API service"       # 🆕 Documentation
version: "1.0"                        # 🆕 Versioning
tags: ["api", "microservice"]         # 🆕 Categorization

data_sources:                         # 🆕 Organized data sources
  elasticsearch:
    index_pattern: "${ES_INDEX:-logs-*}"  # 🆕 Environment variables
  log_file: "/var/log/app.log"

log_patterns:
  - name: "timeout_error"             # 🆕 Descriptive naming
    description: "Request timeouts"   # 🆕 Documentation
    regex: "(?i)timeout"
    severity: "warning"               # 🆕 Severity classification

metrics:
  - name: "APITraffic"
    description: "Request volume"     # 🆕 Documentation
    query_tpl: 'sum(requests_total{service="${SERVICE_NAME}"})'  # 🆕 Dynamic queries
    operator: ">"
    threshold: 30
    weight: 5
    unit: "requests"                  # 🆕 Units

analysis_context:                     # 🆕 AI context
  service_type: "rest_api"
  criticality: "high"
  common_causes: ["high_load", "db_issues"]
  escalation_path: "sre-team"
```

---

## 🤝 Contributing

1. **Generate Config**: Use `./vigilant-config-gen` for new services
2. **Follow Structure**: Use the enhanced YAML format
3. **Add Documentation**: Include descriptions and context
4. **Test Configuration**: Validate with `./vigilant -llm=false`
5. **Use Environment Variables**: Make configs environment-agnostic

**Ready to configure your services? Start with the config generator! 🚀**