package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"gopkg.in/yaml.v3"
	"vigilant/pkg/prometheus"
)

// ServiceMetadata holds descriptive information about the service
type ServiceMetadata struct {
	Name        string   `yaml:"name"`
	Description string   `yaml:"description,omitempty"`
	Version     string   `yaml:"version,omitempty"`
	Tags        []string `yaml:"tags,omitempty"`
	Maintainer  string   `yaml:"maintainer,omitempty"`
}

// AlertMatching defines how alerts are matched to this service
type AlertMatching struct {
	AlertPattern    string   `yaml:"alert_pattern"`
	SeverityLevels  []string `yaml:"severity_levels,omitempty"`
}

// LogPattern defines symptom detection patterns with enhanced metadata
type LogPattern struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description,omitempty"`
	Regex       string `yaml:"regex"`
	Severity    string `yaml:"severity,omitempty"`
	
	// Backward compatibility
	Label string `yaml:"label,omitempty"`
}

// DataSources defines where to fetch observability data
type DataSources struct {
	Elasticsearch ElasticsearchConfig `yaml:"elasticsearch,omitempty"`
	LogFile       string             `yaml:"log_file,omitempty"`
}

// ElasticsearchConfig with enhanced configuration
type ElasticsearchConfig struct {
	IndexPattern     string   `yaml:"index_pattern,omitempty"`
	TimeRangeMinutes int      `yaml:"time_range_minutes,omitempty"`
	ScanLimit        int      `yaml:"scan_limit,omitempty"`
	ServiceFields    []string `yaml:"service_fields,omitempty"`
	NamespaceFilter  string   `yaml:"namespace_filter,omitempty"`
	RequiredFields   []string `yaml:"required_fields,omitempty"`
	
	// Backward compatibility
	TimeRangeMin int `yaml:"time_range_min,omitempty"`
}

// EnhancedMetricCheck extends prometheus.MetricCheck with metadata
type EnhancedMetricCheck struct {
	prometheus.MetricCheck `yaml:",inline"`
	Description            string `yaml:"description,omitempty"`
	Unit                   string `yaml:"unit,omitempty"`
}

// AnalysisContext provides hints for LLM analysis
type AnalysisContext struct {
	ServiceType    string   `yaml:"service_type,omitempty"`
	Criticality    string   `yaml:"criticality,omitempty"`
	CommonCauses   []string `yaml:"common_causes,omitempty"`
	EscalationPath string   `yaml:"escalation_path,omitempty"`
}

// ServiceProfile represents the complete service configuration
type ServiceProfile struct {
	// New enhanced structure
	Metadata        ServiceMetadata       `yaml:",inline,omitempty"`
	AlertMatching   AlertMatching         `yaml:",inline,omitempty"`
	DataSources     DataSources           `yaml:"data_sources,omitempty"`
	LogPatterns     []LogPattern          `yaml:"log_patterns,omitempty"`
	Metrics         []EnhancedMetricCheck `yaml:"metrics,omitempty"`
	AnalysisContext AnalysisContext       `yaml:"analysis_context,omitempty"`
	
	// Backward compatibility fields
	LogFile        string                   `yaml:"log_file,omitempty"`
	Elasticsearch  ElasticsearchConfig      `yaml:"elasticsearch,omitempty"`
	LegacyMetrics  []prometheus.MetricCheck `yaml:"-"` // Populated during migration
}


// LoadServiceProfiles loads all service profile files from a directory with enhanced features
func LoadServiceProfiles(dir string) (map[string]ServiceProfile, error) {
	profiles := make(map[string]ServiceProfile)

	// Support both .yml and .yaml extensions
	ymlFiles, err := filepath.Glob(filepath.Join(dir, "*.yml"))
	if err != nil {
		return nil, fmt.Errorf("failed to glob .yml files: %w", err)
	}
	
	yamlFiles, err := filepath.Glob(filepath.Join(dir, "*.yaml"))
	if err != nil {
		return nil, fmt.Errorf("failed to glob .yaml files: %w", err)
	}
	
	files := append(ymlFiles, yamlFiles...)

	for _, file := range files {
		name := filepath.Base(file)
		service := name[:len(name)-len(filepath.Ext(name))] // filename without extension

		data, err := os.ReadFile(file)
		if err != nil {
			fmt.Printf("Warning: cannot read file %s: %v\n", file, err)
			continue
		}

		// Perform environment variable substitution
		content := expandEnvironmentVariables(string(data))

		var profile ServiceProfile
		if err := yaml.Unmarshal([]byte(content), &profile); err != nil {
			fmt.Printf("Warning: invalid YAML in %s: %v\n", file, err)
			continue
		}

		// Migrate legacy format to new format
		profile = migrateLegacyConfig(profile, service)
		
		// Validate configuration
		if err := validateServiceProfile(profile, service); err != nil {
			fmt.Printf("Warning: invalid configuration in %s: %v\n", file, err)
			continue
		}

		// Apply defaults
		profile = applyDefaults(profile)

		// Use the name field as the primary service identifier
		serviceName := profile.Metadata.Name
		if serviceName == "" {
			// Fallback to filename for backward compatibility
			serviceName = service
			fmt.Printf("Warning: Service config %s has no name field, using filename as identifier\n", file)
		}
		
		// Check for duplicate service names
		if _, exists := profiles[serviceName]; exists {
			fmt.Printf("Warning: Duplicate service name '%s' found in %s, skipping\n", serviceName, file)
			continue
		}

		profiles[serviceName] = profile
	}

	return profiles, nil
}

// CreateAlertToServiceMapping creates a mapping from alert patterns to service names
func CreateAlertToServiceMapping(profiles map[string]ServiceProfile) map[string]string {
	mapping := make(map[string]string)
	
	for serviceName, profile := range profiles {
		// Map alert pattern to service name
		alertPattern := profile.AlertMatching.AlertPattern
		if alertPattern == "" {
			// Fallback to service name itself
			alertPattern = serviceName
		}
		mapping[alertPattern] = serviceName
	}
	
	return mapping
}

// expandEnvironmentVariables replaces ${VAR} and $VAR patterns with environment values
func expandEnvironmentVariables(content string) string {
	// Replace ${VAR} patterns
	re := regexp.MustCompile(`\$\{([^}]+)\}`)
	content = re.ReplaceAllStringFunc(content, func(match string) string {
		varName := match[2 : len(match)-1] // Remove ${ and }
		if value := os.Getenv(varName); value != "" {
			return value
		}
		return match // Keep original if env var not found
	})
	
	// Replace $VAR patterns (word boundaries)
	re = regexp.MustCompile(`\$([A-Za-z_][A-Za-z0-9_]*)`)
	content = re.ReplaceAllStringFunc(content, func(match string) string {
		varName := match[1:] // Remove $
		if value := os.Getenv(varName); value != "" {
			return value
		}
		return match // Keep original if env var not found
	})
	
	return content
}

// migrateLegacyConfig converts legacy format to new enhanced format
func migrateLegacyConfig(profile ServiceProfile, serviceName string) ServiceProfile {
	// If using legacy structure, migrate to new format
	if profile.LogFile != "" && profile.DataSources.LogFile == "" {
		profile.DataSources.LogFile = profile.LogFile
	}
	
	// Migrate elasticsearch config
	if (profile.Elasticsearch.IndexPattern != "" || profile.Elasticsearch.NamespaceFilter != "") &&
		(profile.DataSources.Elasticsearch.IndexPattern == "") {
		profile.DataSources.Elasticsearch = profile.Elasticsearch
	}
	
	// Handle time range field variations
	if profile.DataSources.Elasticsearch.TimeRangeMinutes == 0 && profile.DataSources.Elasticsearch.TimeRangeMin > 0 {
		profile.DataSources.Elasticsearch.TimeRangeMinutes = profile.DataSources.Elasticsearch.TimeRangeMin
	}
	if profile.Elasticsearch.TimeRangeMinutes == 0 && profile.Elasticsearch.TimeRangeMin > 0 {
		profile.Elasticsearch.TimeRangeMinutes = profile.Elasticsearch.TimeRangeMin
	}
	
	// Migrate log patterns - handle both label and name fields
	for i := range profile.LogPatterns {
		if profile.LogPatterns[i].Name == "" && profile.LogPatterns[i].Label != "" {
			profile.LogPatterns[i].Name = profile.LogPatterns[i].Label
		}
	}
	
	// Set service name if not provided (accessing embedded fields directly)
	if profile.Metadata.Name == "" {
		profile.Metadata.Name = serviceName
	}
	
	// Set alert pattern if not provided
	if profile.AlertMatching.AlertPattern == "" {
		profile.AlertMatching.AlertPattern = serviceName
	}
	
	return profile
}

// validateServiceProfile validates the service configuration
func validateServiceProfile(profile ServiceProfile, serviceName string) error {
	// Validate required fields
	if profile.Metadata.Name == "" && serviceName == "" {
		return fmt.Errorf("service name is required")
	}
	
	// Validate log patterns
	for i, pattern := range profile.LogPatterns {
		if pattern.Regex == "" {
			return fmt.Errorf("log pattern %d is missing regex", i)
		}
		
		// Test regex compilation
		if _, err := regexp.Compile(pattern.Regex); err != nil {
			return fmt.Errorf("invalid regex in pattern %d (%s): %v", i, pattern.Name, err)
		}
	}
	
	// Validate metrics
	for i, metric := range profile.Metrics {
		if metric.Name == "" {
			return fmt.Errorf("metric %d is missing name", i)
		}
		if metric.QueryTpl == "" {
			return fmt.Errorf("metric %d (%s) is missing query template", i, metric.Name)
		}
	}
	
	return nil
}

// applyDefaults sets reasonable defaults for missing configuration
func applyDefaults(profile ServiceProfile) ServiceProfile {
	// Default Elasticsearch configuration
	if profile.DataSources.Elasticsearch.IndexPattern == "" && profile.Elasticsearch.IndexPattern == "" {
		profile.DataSources.Elasticsearch.IndexPattern = "fluentbit-*"
	}
	
	if profile.DataSources.Elasticsearch.TimeRangeMinutes == 0 && profile.Elasticsearch.TimeRangeMinutes == 0 {
		profile.DataSources.Elasticsearch.TimeRangeMinutes = 15
	}
	
	if profile.DataSources.Elasticsearch.ScanLimit == 0 && profile.Elasticsearch.ScanLimit == 0 {
		profile.DataSources.Elasticsearch.ScanLimit = 500
	}
	
	// Default required fields for Elasticsearch
	if len(profile.DataSources.Elasticsearch.RequiredFields) == 0 {
		profile.DataSources.Elasticsearch.RequiredFields = []string{"@timestamp", "log", "kubernetes.container_name"}
	}
	
	// Default severity levels
	if len(profile.AlertMatching.SeverityLevels) == 0 {
		profile.AlertMatching.SeverityLevels = []string{"warning", "critical"}
	}
	
	// Default version
	if profile.Metadata.Version == "" {
		profile.Metadata.Version = "1.0"
	}
	
	return profile
}

// GetEffectiveElasticsearchConfig returns the active Elasticsearch configuration
func (p *ServiceProfile) GetEffectiveElasticsearchConfig() ElasticsearchConfig {
	// Prefer new format over legacy
	if p.DataSources.Elasticsearch.IndexPattern != "" || p.DataSources.Elasticsearch.NamespaceFilter != "" {
		return p.DataSources.Elasticsearch
	}
	return p.Elasticsearch
}

// GetEffectiveLogFile returns the active log file path
func (p *ServiceProfile) GetEffectiveLogFile() string {
	if p.DataSources.LogFile != "" {
		return p.DataSources.LogFile
	}
	return p.LogFile
}

// GetEffectiveMetrics returns metrics in the standard format
func (p *ServiceProfile) GetEffectiveMetrics() []prometheus.MetricCheck {
	var metrics []prometheus.MetricCheck
	
	// Convert enhanced metrics to standard format
	for _, metric := range p.Metrics {
		metrics = append(metrics, metric.MetricCheck)
	}
	
	return metrics
}
