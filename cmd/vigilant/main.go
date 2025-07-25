package main

import (
	"fmt"
	"os"
	"time"

	"github.com/joho/godotenv"

	"vigilant/pkg/api"
	"vigilant/pkg/config"
	"vigilant/pkg/hashutil"
	"vigilant/pkg/logs"
	"vigilant/pkg/prometheus"
	"vigilant/pkg/risk"
	"vigilant/pkg/summarizer"
	"vigilant/pkg/utils"
)

// StateSnapshot represents the current state for change detection
type StateSnapshot struct {
	AlertCount   int
	SymptomCount int
	MetricCount  int
	
	// Hash of actual content for detecting value changes
	AlertsHash    string
	SymptomsHash  string
	MetricsHash   string
	
	// Timestamp for periodic forced updates
	LastLLMUpdate time.Time
}

func (s *StateSnapshot) HasChanged(other StateSnapshot) bool {
	return s.AlertCount != other.AlertCount ||
		s.SymptomCount != other.SymptomCount ||
		s.MetricCount != other.MetricCount ||
		s.AlertsHash != other.AlertsHash ||
		s.SymptomsHash != other.SymptomsHash ||
		s.MetricsHash != other.MetricsHash
}

func (s *StateSnapshot) ShouldForceUpdate(maxAge time.Duration) bool {
	return time.Since(s.LastLLMUpdate) > maxAge
}

func main() {
	fmt.Println("Starting Vigilant...")
	if err := godotenv.Load("../../.env"); err != nil {
		fmt.Println("Warning: .env file not found or failed to load.")
	}

	promURL := os.Getenv("PROM_URL")
	if promURL == "" {
		promURL = "http://localhost:9090"
		fmt.Println("PROM_URL not set in env, using default:", promURL)
	}

	// Initialize Elasticsearch client
	esURLs := []string{os.Getenv("ELASTICSEARCH_URL")}
	if esURLs[0] == "" {
		esURLs = []string{"http://localhost:9200"}
		fmt.Println("ELASTICSEARCH_URL not set in env, using default:", esURLs[0])
	}

	esClient, err := logs.NewElasticsearchClient(esURLs)
	if err != nil {
		fmt.Printf("Failed to initialize Elasticsearch client: %v\n", err)
		fmt.Println("Falling back to file-based log scanning...")
		esClient = nil
	} else {
		fmt.Println("Successfully connected to Elasticsearch")
	}

	// Default ES configuration (can be overridden per service)
	defaultESIndexPattern := os.Getenv("ES_INDEX_PATTERN")
	if defaultESIndexPattern == "" {
		defaultESIndexPattern = "logs-*"
		fmt.Println("ES_INDEX_PATTERN not set in env, using default:", defaultESIndexPattern)
	}

	// Start REST API server
	go api.StartServer()

	tracker := risk.NewRiskTracker(2 * time.Minute)

	profiles, err := config.LoadServiceProfiles("../../config/services")
	if err != nil {
		fmt.Println("Failed to load service configs:", err)
		return
	}

	// Create service mapping from loaded profiles
	serviceMapping := logs.NewServiceMapping(profiles)
	
	// Create map of valid services for alert filtering
	validServices := make(map[string]bool)
	for serviceName := range profiles {
		validServices[serviceName] = true
	}
	fmt.Printf("Loaded %d service configurations: %v\n", len(validServices), getServiceNames(validServices))

	var lastState StateSnapshot
	maxLLMUpdateAge := 5 * time.Minute // Force LLM update every 5 minutes regardless

	for {
		fmt.Println("Fetching alerts...")
		alerts, err := prometheus.FetchAlerts(promURL, validServices)
		if err != nil {
			fmt.Println("Error fetching alerts:", err)
			continue
		}

		tracker.UpdateFromAlerts(alerts)
		tracker.CleanupExpired()
		
		// Log active alerts being processed
		if len(tracker.Items) > 0 {
			fmt.Printf("Processing %d active alerts:\n", len(tracker.Items))
			for _, item := range tracker.Items {
				fmt.Printf("[ALERT] %s on %s (severity: %s)\n", item.AlertName, item.Service, item.Severity)
			}
		} else {
			fmt.Println("No active alerts to process")
		}

		seen := map[string]bool{}
		var correlations []summarizer.AlertCorrelation
		var uiData []api.APIRiskItem

		// Collections for hashing
		var simplifiedAlerts []hashutil.SimplifiedAlert
		var simplifiedSymptoms []hashutil.SimplifiedSymptom
		var simplifiedMetrics []hashutil.SimplifiedMetric

		currentAlertCount := len(tracker.Items)
		currentSymptomCount := 0
		currentMetricCount := 0

		// Process alerts for hash comparison
		for _, item := range tracker.Items {
			simplifiedAlerts = append(simplifiedAlerts, hashutil.SimplifiedAlert{
				Service:   item.Service,
				AlertName: item.AlertName,
				Severity:  item.Severity,
			})
		}

		for _, item := range tracker.Items {
			service := item.Service
			if seen[service] {
				continue
			}
			seen[service] = true

			profile, ok := profiles[service]
			if !ok {
				fmt.Println("No profile found for service:", service)
				continue
			}

			// Logs - Use Elasticsearch if available, otherwise fall back to file-based
			var symptoms []logs.SymptomMatch
			if esClient != nil {
				// Get service-specific ES configuration or use defaults
				indexPattern := profile.Elasticsearch.IndexPattern
				if indexPattern == "" {
					indexPattern = defaultESIndexPattern
				}
				
				scanLimit := profile.Elasticsearch.ScanLimit
				if scanLimit == 0 {
					scanLimit = 500 // default
				}
				
				timeRangeMin := profile.Elasticsearch.TimeRangeMin
				if timeRangeMin == 0 {
					timeRangeMin = 10 // default
				}
				timeRange := time.Duration(timeRangeMin) * time.Minute
				
				namespaceFilter := profile.Elasticsearch.NamespaceFilter
				
				fmt.Printf("ES scan for %s: index=%s, limit=%d, time=%dmin, namespace=%s\n", 
					service, indexPattern, scanLimit, timeRangeMin, namespaceFilter)
				
				// Use Elasticsearch with namespace filtering
				symptoms, err = esClient.ScanLogsAndMatchSymptomsWithFilter(
					indexPattern,
					scanLimit,
					profile.LogPatterns,
					timeRange,
					serviceMapping,
					namespaceFilter,
				)
				if err != nil {
					fmt.Printf("Error scanning Elasticsearch logs for %s: %v\n", service, err)
					fmt.Println("Attempting fallback to file-based scanning...")
					
					// Fallback to file-based if ES fails
					if profile.LogFile != "" {
						symptoms, err = logs.ScanLogsAndMatchSymptoms(profile.LogFile, scanLimit, profile.LogPatterns)
						if err != nil {
							fmt.Printf("File-based fallback also failed for %s: %v\n", service, err)
						}
					}
				}
			} else {
				// Use file-based scanning
				if profile.LogFile != "" {
					scanLimit := profile.Elasticsearch.ScanLimit
					if scanLimit == 0 {
						scanLimit = 500 // default
					}
					symptoms, err = logs.ScanLogsAndMatchSymptoms(profile.LogFile, scanLimit, profile.LogPatterns)
					if err != nil {
						fmt.Printf("Error scanning file logs for %s: %v\n", service, err)
					}
				} else {
					fmt.Printf("No log file configured for service %s and Elasticsearch unavailable\n", service)
				}
			}

			// Filter symptoms for current service (important for ES which might return all services)
			var serviceSymptoms []logs.SymptomMatch
			for _, sym := range symptoms {
				// Map symptoms to the service we're processing (since ES might return generic matches)
				if sym.Service == service || sym.Service == "unknown" {
					// Force map unknown symptoms to the current service we're processing
					if sym.Service == "unknown" {
						sym.Service = service
					}
					serviceSymptoms = append(serviceSymptoms, sym)
					fmt.Printf("[SYMPTOM] %s matched on %s (%d times)\n", sym.Pattern, sym.Service, sym.Count)
					simplifiedSymptoms = append(simplifiedSymptoms, hashutil.SimplifiedSymptom{
						Service: sym.Service,
						Pattern: sym.Pattern,
						Count:   sym.Count,
					})
				}
			}
			currentSymptomCount += len(serviceSymptoms)

			// Metrics
			var checks []prometheus.MetricCheck
			for _, check := range profile.Metrics {
				cloned := check
				cloned.QueryTpl = prometheus.RenderQuery(cloned.QueryTpl, map[string]string{
					"Service": service,
				})
				checks = append(checks, cloned)
			}

			metrics, err := prometheus.EvaluateMetricChecks(promURL, []prometheus.ServiceMetricConfig{
				{Service: service, Checks: checks},
			})
			if err != nil {
				fmt.Println("Error evaluating metrics for", service, ":", err)
			} else {
				currentMetricCount += len(metrics)
				for _, m := range metrics {
					fmt.Printf("[METRIC] %s triggered for %s: %.2f %s %.2f\n",
						m.Check.Name, m.Service, m.Value, m.Check.Operator, m.Check.Threshold)
					simplifiedMetrics = append(simplifiedMetrics, hashutil.SimplifiedMetric{
						Service:   m.Service,
						CheckName: m.Check.Name,
						Value:     m.Value,
						Operator:  m.Check.Operator,
						Threshold: m.Check.Threshold,
					})
				}
			}

			correlations = append(correlations, summarizer.AlertCorrelation{
				Alert:    *item,
				Symptoms: serviceSymptoms, // Use filtered symptoms
				Metrics:  metrics,
			})

			uiData = append(uiData, api.APIRiskItem{
				Service:  service,
				Alert:    item.AlertName,
				Severity: item.Severity,
				Symptoms: utils.ConvertSymptoms(serviceSymptoms), // Use filtered symptoms
				Metrics:  utils.ConvertMetrics(metrics),				
				Summary:  "", // will be updated after LLM
			})
		}

		// Create current state snapshot
		currentState := StateSnapshot{
			AlertCount:    currentAlertCount,
			SymptomCount:  currentSymptomCount,
			MetricCount:   currentMetricCount,
			AlertsHash:    hashutil.HashData(simplifiedAlerts),
			SymptomsHash:  hashutil.HashData(simplifiedSymptoms),
			MetricsHash:   hashutil.HashData(simplifiedMetrics),
			LastLLMUpdate: lastState.LastLLMUpdate,
		}

		// Check if anything changed or if we need a forced update
		needsLLMUpdate := currentState.HasChanged(lastState) || currentState.ShouldForceUpdate(maxLLMUpdateAge)

		if currentState.HasChanged(lastState) {
			fmt.Printf("Changes detected:\n")
			fmt.Printf("  Alerts: %d→%d (hash: %s→%s)\n", 
				lastState.AlertCount, currentState.AlertCount,
				hashutil.SafeHashDisplay(lastState.AlertsHash), hashutil.SafeHashDisplay(currentState.AlertsHash))
			fmt.Printf("  Symptoms: %d→%d (hash: %s→%s)\n", 
				lastState.SymptomCount, currentState.SymptomCount,
				hashutil.SafeHashDisplay(lastState.SymptomsHash), hashutil.SafeHashDisplay(currentState.SymptomsHash))
			fmt.Printf("  Metrics: %d→%d (hash: %s→%s)\n", 
				lastState.MetricCount, currentState.MetricCount,
				hashutil.SafeHashDisplay(lastState.MetricsHash), hashutil.SafeHashDisplay(currentState.MetricsHash))
		} else if currentState.ShouldForceUpdate(maxLLMUpdateAge) {
			fmt.Printf("Forcing LLM update - last update was %v ago\n", time.Since(lastState.LastLLMUpdate))
		}

		if needsLLMUpdate {
			summaryMap, err := summarizer.SummarizeMany(correlations)
			if err != nil {
				fmt.Println("Error generating per-service summaries:", err)
			} else {
				fmt.Println("=== Root Cause Summaries ===")
				for svc, summary := range summaryMap {
					fmt.Printf("[%s]\n%s\n\n", svc, summary)
				}
				for i := range uiData {
					if s, ok := summaryMap[uiData[i].Service]; ok {
						uiData[i].Summary = s.Summary
						uiData[i].Risk = s.Risk
					}
				}
			}
			
			// Update the timestamp
			currentState.LastLLMUpdate = time.Now()
			lastState = currentState
		} else {
			fmt.Println("No changes detected. Skipping LLM summary.")
		}

		// Push to REST API memory
		api.UpdateRisks(uiData)

		time.Sleep(30 * time.Second)
	}
}

// getServiceNames extracts service names from validServices map for logging
func getServiceNames(validServices map[string]bool) []string {
	var names []string
	for name := range validServices {
		names = append(names, name)
	}
	return names
}