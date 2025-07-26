# Vigilant Service Configuration Flow

## System Architecture Flow

```mermaid
graph TB
    subgraph "Configuration Loading"
        A[config/services/*.yml] --> B[LoadServiceProfiles()]
        B --> C[Environment Variable Substitution]
        C --> D[Configuration Validation]
        D --> E[Apply Defaults]
        E --> F[Service Name Mapping]
    end
    
    subgraph "Alert Processing"
        G[Prometheus Alerts] --> H[Alert Pattern Matching]
        H --> I{Service Found?}
        I -->|Yes| J[Load Service Config]
        I -->|No| K[Mark as Unknown]
        J --> L[Process Logs & Metrics]
    end
    
    subgraph "Data Collection"
        L --> M[Elasticsearch Query]
        L --> N[Prometheus Metrics]
        L --> O[Log File Fallback]
        M --> P[Pattern Matching]
        N --> Q[Threshold Evaluation]
        O --> P
    end
    
    subgraph "Analysis & Output"
        P --> R[Symptom Detection]
        Q --> S[Risk Calculation]
        R --> T[LLM Analysis Context]
        S --> T
        T --> U[Generate Insights]
        U --> V[API Response]
        U --> W[WebSocket Update]
    end
    
    F --> H
```

## Service Identification Flow

```mermaid
sequenceDiagram
    participant P as Prometheus
    participant V as Vigilant
    participant C as Config System
    participant E as Elasticsearch
    participant L as LLM
    
    P->>V: Alert: "IstioProxyHighCPU"
    V->>C: Look up alert pattern
    C->>C: alert_pattern: "IstioProxyHighCPU" → name: "IstioProxyHighCPU"
    C->>V: Return service config
    
    V->>E: Query logs with namespace filter
    E->>V: Return matching logs
    V->>V: Apply log patterns
    
    V->>P: Query metrics
    P->>V: Return metric values
    V->>V: Evaluate thresholds
    
    V->>L: Send context + data
    L->>V: Return analysis
    V->>V: Update risk tracking
```

## Configuration Processing Pipeline

```mermaid
flowchart TD
    subgraph "Stage 1: File Discovery"
        A1[Scan config/services/] --> A2[Find *.yml, *.yaml files]
        A2 --> A3[Read file contents]
    end
    
    subgraph "Stage 2: Content Processing"
        A3 --> B1[Environment Variable Substitution]
        B1 --> B2[YAML Parsing]
        B2 --> B3[Schema Validation]
    end
    
    subgraph "Stage 3: Migration & Validation"
        B3 --> C1[Legacy Format Migration]
        C1 --> C2[Field Validation]
        C2 --> C3[Regex Compilation Check]
        C3 --> C4[Duplicate Name Detection]
    end
    
    subgraph "Stage 4: Finalization"
        C4 --> D1[Apply Default Values]
        D1 --> D2[Create Service Mapping]
        D2 --> D3[Build Alert Pattern Map]
        D3 --> D4[Ready for Runtime]
    end
```

## Runtime Alert Processing

```mermaid
graph LR
    subgraph "Alert Ingestion"
        A[Prometheus Alert] --> B{Alert State}
        B -->|firing| C[Add to Risk Tracker]
        B -->|resolved| D[Remove from Tracker]
    end
    
    subgraph "Service Resolution"
        C --> E[Extract Alert Name]
        E --> F{Alert Pattern Match?}
        F -->|Yes| G[Get Service Name]
        F -->|No| H{Direct Service Match?}
        H -->|Yes| G
        H -->|No| I[Mark as Unknown]
    end
    
    subgraph "Data Processing"
        G --> J[Load Service Config]
        J --> K[Query Elasticsearch]
        J --> L[Query Prometheus]
        K --> M[Apply Log Patterns]
        L --> N[Evaluate Metrics]
    end
    
    subgraph "Analysis"
        M --> O[Symptom Detection]
        N --> P[Risk Calculation]
        O --> Q[Context Building]
        P --> Q
        Q --> R[LLM Analysis]
        R --> S[Generate Response]
    end
```

## Data Flow Through Service Configuration

```mermaid
graph TD
    subgraph "Input Sources"
        I1[Prometheus Alerts]
        I2[Elasticsearch Logs]
        I3[Prometheus Metrics]
        I4[Service Config YAML]
    end
    
    subgraph "Configuration Processing"
        C1[Service Metadata]
        C2[Alert Pattern Matching]
        C3[Data Source Configuration]
        C4[Log Pattern Rules]
        C5[Metric Evaluation Rules]
        C6[Analysis Context]
    end
    
    subgraph "Runtime Processing"
        R1[Alert Correlation]
        R2[Log Pattern Matching]
        R3[Metric Threshold Evaluation]
        R4[Risk Score Calculation]
        R5[LLM Context Building]
    end
    
    subgraph "Output"
        O1[Symptom Matches]
        O2[Metric Violations]
        O3[Risk Assessment]
        O4[Root Cause Analysis]
        O5[Recommended Actions]
    end
    
    I1 --> C2
    I4 --> C1
    I4 --> C3
    I4 --> C4
    I4 --> C5
    I4 --> C6
    
    C2 --> R1
    C3 --> R2
    C4 --> R2
    C5 --> R3
    C6 --> R5
    
    I2 --> R2
    I3 --> R3
    
    R1 --> O3
    R2 --> O1
    R3 --> O2
    R4 --> O3
    R5 --> O4
    R5 --> O5
```

## Configuration Field Relationships

```mermaid
erDiagram
    ServiceProfile {
        string name
        string description
        string version
        array tags
        string maintainer
    }
    
    AlertMatching {
        string alert_pattern
        array severity_levels
    }
    
    DataSources {
        object elasticsearch
        string log_file
    }
    
    ElasticsearchConfig {
        string index_pattern
        int time_range_minutes
        int scan_limit
        string namespace_filter
        array required_fields
    }
    
    LogPattern {
        string name
        string description
        string regex
        string severity
    }
    
    MetricCheck {
        string name
        string description
        string query_tpl
        string operator
        float threshold
        int weight
        string unit
    }
    
    AnalysisContext {
        string service_type
        string criticality
        array common_causes
        string escalation_path
    }
    
    ServiceProfile ||--|| AlertMatching : contains
    ServiceProfile ||--|| DataSources : contains
    ServiceProfile ||--o{ LogPattern : contains
    ServiceProfile ||--o{ MetricCheck : contains
    ServiceProfile ||--|| AnalysisContext : contains
    DataSources ||--|| ElasticsearchConfig : contains
```

## Environment Variable Resolution

```mermaid
graph LR
    subgraph "Configuration File"
        A[${ES_INDEX_PATTERN:-fluentbit-*}]
        B[${NAMESPACE:-production}]
        C[$SERVICE_NAME]
    end
    
    subgraph "Environment Variables"
        D[ES_INDEX_PATTERN=logs-*]
        E[NAMESPACE=staging]
        F[SERVICE_NAME=payment-api]
    end
    
    subgraph "Resolution Process"
        G[Parse ${VAR:-default}]
        H[Check Environment]
        I[Apply Default if Missing]
        J[Substitute Value]
    end
    
    subgraph "Final Configuration"
        K[index_pattern: logs-*]
        L[namespace_filter: staging]
        M[query_tpl: up{job="payment-api"}]
    end
    
    A --> G
    B --> G
    C --> G
    D --> H
    E --> H
    F --> H
    G --> H
    H --> I
    I --> J
    J --> K
    J --> L
    J --> M
```

## Service Configuration Lifecycle

```mermaid
stateDiagram-v2
    [*] --> FileDetected : New .yml file
    FileDetected --> Parsing : Read file content
    Parsing --> EnvironmentSubstitution : Parse YAML
    EnvironmentSubstitution --> Validation : Substitute ${VAR}
    Validation --> Migration : Validate schema
    Migration --> DefaultApplication : Migrate legacy format
    DefaultApplication --> ServiceMapping : Apply defaults
    ServiceMapping --> Ready : Create mappings
    
    Ready --> Processing : Alert received
    Processing --> DataCollection : Service matched
    DataCollection --> Analysis : Logs/metrics collected
    Analysis --> Response : LLM analysis complete
    Response --> Ready : Output generated
    
    Parsing --> Error : Invalid YAML
    Validation --> Error : Schema validation failed
    Migration --> Error : Migration failed
    Error --> [*] : Configuration rejected
    
    Ready --> FileDetected : Configuration updated
```

## Alert Processing State Machine

```mermaid
stateDiagram-v2
    [*] --> AlertReceived : Prometheus fires alert
    AlertReceived --> PatternMatching : Extract alert name
    PatternMatching --> ServiceFound : Pattern matches
    PatternMatching --> DirectLookup : No pattern match
    DirectLookup --> ServiceFound : Direct service match
    DirectLookup --> UnknownService : No service found
    
    ServiceFound --> ConfigLoaded : Load service config
    ConfigLoaded --> DataQuery : Config validation passed
    DataQuery --> LogProcessing : Query Elasticsearch
    DataQuery --> MetricProcessing : Query Prometheus
    
    LogProcessing --> PatternApplication : Logs retrieved
    PatternApplication --> SymptomDetection : Apply regex patterns
    
    MetricProcessing --> ThresholdEvaluation : Metrics retrieved
    ThresholdEvaluation --> RiskCalculation : Evaluate thresholds
    
    SymptomDetection --> ContextBuilding : Symptoms detected
    RiskCalculation --> ContextBuilding : Risk calculated
    ContextBuilding --> LLMAnalysis : Context prepared
    LLMAnalysis --> ResponseGeneration : Analysis complete
    ResponseGeneration --> [*] : Response sent
    
    UnknownService --> [*] : Alert marked unknown
```

## Configuration Validation Flow

```mermaid
flowchart TD
    A[YAML File] --> B{Valid YAML?}
    B -->|No| E1[Reject: Parse Error]
    B -->|Yes| C[Extract Fields]
    
    C --> D{Required Fields Present?}
    D -->|No| E2[Reject: Missing Fields]
    D -->|Yes| F[Validate Log Patterns]
    
    F --> G{Regex Compiles?}
    G -->|No| E3[Reject: Invalid Regex]
    G -->|Yes| H[Validate Metrics]
    
    H --> I{Valid PromQL?}
    I -->|No| E4[Reject: Invalid Query]
    I -->|Yes| J[Check Duplicates]
    
    J --> K{Name Conflicts?}
    K -->|Yes| E5[Reject: Duplicate Name]
    K -->|No| L[Apply Defaults]
    
    L --> M[Configuration Ready]
    
    E1 --> N[Log Warning]
    E2 --> N
    E3 --> N
    E4 --> N
    E5 --> N
    N --> O[Skip Configuration]
```

This comprehensive flow documentation shows how Vigilant processes service configurations from initial loading through runtime alert processing and analysis generation.