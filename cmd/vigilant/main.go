package main

import (
	"fmt"
	"time"

	"vigilant/pkg/prometheus"
	"vigilant/pkg/risk"
)

func main() {
	fmt.Println("Starting Vigilant...")
	promURL := "http://localhost:9090"
	tracker := risk.NewRiskTracker(2 * time.Minute)

	for {
		fmt.Println("Fetching alerts...")
		alerts, err := prometheus.FetchAlerts(promURL)
		if err != nil {
			fmt.Println("Error:", err)
		} else {
			tracker.UpdateFromAlerts(alerts)
			tracker.CleanupExpired()
			tracker.Print()
		}

		time.Sleep(15 * time.Second)
	}
}
