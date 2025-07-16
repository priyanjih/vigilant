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

	// Start REST API server
	go api.StartServer()

	tracker := risk.NewRiskTracker(2 * time.Minute)

	profiles, err := config.LoadServiceProfiles("../../config/services")
	if err != nil {
		fmt.Println("Failed to load service configs:", err)
		return
	}

	var lastState StateSnapshot
	maxLLMUpdateAge := 5 * time.Minute // Force LLM update every 5 minutes regardless

	for {
		fmt.Println("Fetching alerts...")
		alerts, err := prometheus.FetchAlerts(promURL)
		if err != nil {
			fmt.Println("Error fetching alerts:", err)
			continue
		}

		tracker.UpdateFromAlerts(alerts)
		tracker.CleanupExpired()

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

			// Logs
			symptoms, err := logs.ScanLogsAndMatchSymptoms(profile.LogFile, 500, profile.LogPatterns)
			if err != nil {
				fmt.Println("Error scanning logs for", service, ":", err)
			} else {
				currentSymptomCount += len(symptoms)
				for _, sym := range symptoms {
					if sym.Service == service {
						fmt.Printf("[SYMPTOM] %s matched on %s (%d times)\n", sym.Pattern, sym.Service, sym.Count)
						simplifiedSymptoms = append(simplifiedSymptoms, hashutil.SimplifiedSymptom{
							Service: sym.Service,
							Pattern: sym.Pattern,
							Count:   sym.Count,
						})
					}
				}
			}

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
				Symptoms: symptoms,
				Metrics:  metrics,
			})

			uiData = append(uiData, api.APIRiskItem{
				Service:  service,
				Alert:    item.AlertName,
				Severity: item.Severity,
				Symptoms: utils.ConvertSymptoms(symptoms),
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
			currentInput := summarizer.SummaryInput{
				Correlations: correlations,
			}
			summary, err := summarizer.Summarize(currentInput)
			if err != nil {
				fmt.Println("Error generating summary:", err)
			} else {
				fmt.Println("=== Root Cause Summary ===")
				fmt.Println(summary)
				for i := range uiData {
					uiData[i].Summary = summary
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