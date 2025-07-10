package main

import (
	"fmt"
	"time"
	"os"

	"vigilant/pkg/prometheus"
	"vigilant/pkg/risk"
	"vigilant/pkg/logs"
)

const (
	promURL     = "http://localhost:9090"
	logFilePath = "/home/priyanjith/copilot-stack/hotrod.log"
)

func main() {
	fmt.Println("Starting Vigilant...")
	tracker := risk.NewRiskTracker(2 * time.Minute)

	for {
		fmt.Println("Fetching alerts...")
		alerts, err := prometheus.FetchAlerts(promURL)
		if err != nil {
			fmt.Println("Error fetching alerts:", err)
			continue
		}
		
path := os.Getenv("VIGILANT_METRICS_CONFIG")
if path == "" {
	path = "../../config/metrics.yml"
}
metricChecks, err := prometheus.LoadMetricChecksFromFile(path)

// Prevent duplicates
seen := map[string]bool{}

for _, item := range tracker.Items {
	service := item.Service
	if seen[service] {
		continue
	}
	seen[service] = true

	var serviceChecks []prometheus.MetricCheck
	for _, check := range metricChecks {
		cloned := check // avoid mutating the global one
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
	}
}
		

		// Update Risk Tracker
		tracker.UpdateFromAlerts(alerts)
		tracker.CleanupExpired()

		// Scan Logs
		symptoms, err := logs.ScanLogsAndMatchSymptoms(logFilePath, 500)
		if err != nil {
			fmt.Println("Error scanning logs:", err)
		}

		// Match Symptoms to Risk Items
		for _, item := range tracker.Items {
			for _, sym := range symptoms {
				if sym.Service == item.Service {
					fmt.Printf("[SYMPTOM] %s matched on %s (%d times)\n",
						sym.Pattern, sym.Service, sym.Count)
				}
			}
		}
		
		

		tracker.Print()
		time.Sleep(15 * time.Second)
	}
}

