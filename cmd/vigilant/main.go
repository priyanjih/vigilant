package main

import (
	"fmt"
	"time"

	"vigilant/pkg/prometheus"
)

func main() {
	promURL := "http://localhost:9090"
	fmt.Println("Starting Vigilant...")

	for {
		fmt.Println("Fetching alerts...")
		alerts, err := prometheus.FetchAlerts(promURL)
		if err != nil {
			fmt.Println("Error:", err)
		} else {
			fmt.Printf("Got %d alert(s)\n", len(alerts))
			for _, a := range alerts {
				fmt.Printf("[ALERT] %s on %s (severity: %s) since %s\n",
					a.Name, a.Instance, a.Severity, a.StartsAt.Format(time.RFC822))
			}
		}

		time.Sleep(15 * time.Second)
	}
}
