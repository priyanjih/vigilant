package main

import (
	"fmt"
	"time"

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
