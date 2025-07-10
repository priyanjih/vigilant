package main

import (
	"fmt"
	"time"

	"vigilant/pkg/prometheus"
	"vigilant/pkg/risk"
	"vigilant/pkg/logs"
	"vigilant/pkg/config"
)

const (
	promURL = "http://localhost:9090"
)

func main() {
	fmt.Println("Starting Vigilant...")
	tracker := risk.NewRiskTracker(2 * time.Minute)

	// Load service profiles
	profiles, err := config.LoadServiceProfiles("../../config/services")
	if err != nil {
		fmt.Println("Failed to load service configs:", err)
		return
	}
	

	for {
		fmt.Println("Fetching alerts...")
		alerts, err := prometheus.FetchAlerts(promURL)
		if err != nil {
			fmt.Println("Error fetching alerts:", err)
			continue
		}

		// Update Risk Tracker
		tracker.UpdateFromAlerts(alerts)
		tracker.CleanupExpired()

		// Prevent duplicates
		seen := map[string]bool{}

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

			// 🪵 LOGS
			symptoms, err := logs.ScanLogsAndMatchSymptoms(profile.LogFile, 500, profile.LogPatterns)
			if err == nil {
				for _, sym := range symptoms {
					if sym.Service == item.Service {
						fmt.Printf("[SYMPTOM] %s matched on %s (%d times)\n", sym.Pattern, sym.Service, sym.Count)
					}
				}
			} else {
				fmt.Println("Error scanning logs for", service, ":", err)
			}

			// 📊 METRICS
			var serviceChecks []prometheus.MetricCheck
			for _, check := range profile.Metrics {
				cloned := check
				cloned.QueryTpl = prometheus.RenderQuery(cloned.QueryTpl, map[string]string{
					"Service": service,
				})
				serviceChecks = append(serviceChecks, cloned)
			}

			results, err := prometheus.EvaluateMetricChecks(promURL, []prometheus.ServiceMetricConfig{
				{
					Service: service,
					Checks:  serviceChecks,
				},
			})
			if err == nil {
				fmt.Println("=== Metric Triggers ===")
				for _, r := range results {
					fmt.Printf("[METRIC] %s triggered for %s: %.2f %s %.2f\n",
						r.Check.Name, r.Service, r.Value, r.Check.Operator, r.Check.Threshold)
				}
			} else {
				fmt.Println("Error evaluating metrics for", service, ":", err)
			}
		}

		tracker.Print()
		time.Sleep(15 * time.Second)
	}
}