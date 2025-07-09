package prometheus

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

// Alert represents a simplified version of a Prometheus alert
type Alert struct {
	Name      string
	Instance  string
	Severity  string
	State     string
	StartsAt  time.Time
}

// FetchAlerts fetches firing alerts from Prometheus
func FetchAlerts(promURL string) ([]Alert, error) {
	resp, err := http.Get(fmt.Sprintf("%s/api/v1/alerts", promURL))
	if err != nil {
		return nil, fmt.Errorf("failed to fetch alerts: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("bad response from Prometheus: %s", resp.Status)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var raw struct {
		Data struct {
			Alerts []struct {
				Labels struct {
					AlertName string `json:"alertname"`
					Instance  string `json:"instance"`
					Severity  string `json:"severity"`
				} `json:"labels"`
				State    string    `json:"state"`
				StartsAt time.Time `json:"activeAt"`
			} `json:"alerts"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse Prometheus JSON: %w", err)
	}

	var alerts []Alert
	for _, a := range raw.Data.Alerts {
		if a.State == "firing" {
			alerts = append(alerts, Alert{
				Name:     a.Labels.AlertName,
				Instance: a.Labels.Instance,
				Severity: a.Labels.Severity,
				State:    a.State,
				StartsAt: a.StartsAt,
			})
		}
	}

	return alerts, nil
}
