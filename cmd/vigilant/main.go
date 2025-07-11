package main

import (
	"fmt"
	"time"
	"github.com/joho/godotenv"
	"vigilant/pkg/config"
	"vigilant/pkg/logs"
	"vigilant/pkg/prometheus"
	"vigilant/pkg/risk"
	"vigilant/pkg/summarizer"
)

const promURL = "http://localhost:9090"

func main() {
	fmt.Println("Starting Vigilant...")
	if err := godotenv.Load("../../.env"); err != nil {
		fmt.Println("Warning: .env file not found or failed to load.")
	}

	tracker := risk.NewRiskTracker(2 * time.Minute)

	// Load service profiles
	profiles, err := config.LoadServiceProfiles("../../config/services")
	if err != nil {
		fmt.Println("Failed to load service configs:", err)
		return
	}

	// Simple change tracking
	var (
		lastAlertCount    int
		lastSymptomCount  int
		lastMetricCount   int
		needsLLMUpdate    bool
	)

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
		
		currentAlertCount := len(tracker.Items)
		currentSymptomCount := 0
		currentMetricCount := 0

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
				}
			}

			correlations = append(correlations, summarizer.AlertCorrelation{
				Alert:    *item,
				Symptoms: symptoms,
				Metrics:  metrics,
			})
		}

		// Check if anything changed
		if currentAlertCount != lastAlertCount || 
		   currentSymptomCount != lastSymptomCount || 
		   currentMetricCount != lastMetricCount {
			needsLLMUpdate = true
			
			fmt.Printf("Changes detected - Alerts: %d->%d, Symptoms: %d->%d, Metrics: %d->%d\n",
				lastAlertCount, currentAlertCount,
				lastSymptomCount, currentSymptomCount,
				lastMetricCount, currentMetricCount)
		}

		// Hit LLM only if there are changes
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
			}
			
			// Update last known state
			lastAlertCount = currentAlertCount
			lastSymptomCount = currentSymptomCount
			lastMetricCount = currentMetricCount
			needsLLMUpdate = false
		} else {
			fmt.Println("No changes detected. Skipping LLM summary.")
		}
		
		time.Sleep(30 * time.Second)
	}
}