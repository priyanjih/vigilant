package prometheus

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

// Alert represents a simplified version of a Prometheus alert
type Alert struct {
	Name     string
	Instance string
	Severity string
	Service  string
	StartsAt time.Time
}

// FetchAlerts fetches firing alerts from Prometheus, filtered by configured services
func FetchAlerts(promURL string, validServices map[string]bool) ([]Alert, error) {
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

	// Define a more flexible struct that captures all labels
	var raw struct {
		Data struct {
			Alerts []struct {
				Labels   map[string]string `json:"labels"`
				State    string            `json:"state"`
				StartsAt time.Time         `json:"activeAt"`
			} `json:"alerts"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse Prometheus JSON: %w", err)
	}

	var alerts []Alert
	for _, a := range raw.Data.Alerts {
		if a.State == "firing" {
			alert := Alert{
				Name:     getLabel(a.Labels, "alertname"),
				Instance: getLabel(a.Labels, "instance"),
				Severity: getLabel(a.Labels, "severity"),
				Service:  extractServiceFromLabels(a.Labels, validServices),
				StartsAt: a.StartsAt,
			}
			
			// Only include alerts that match configured service files
			if len(validServices) == 0 || validServices[alert.Name] {
				alerts = append(alerts, alert)
			}
		}
	}

	return alerts, nil
}


func getLabel(labels map[string]string, key string) string {
	if val, ok := labels[key]; ok {
		return val
	}
	return ""
}

func extractServiceFromLabels(labels map[string]string, validServices map[string]bool) string {
	alertname := getLabel(labels, "alertname")
	
	// Simple logic: if alertname matches a service filename, use it
	if validServices[alertname] {
		return alertname
	}
	
	return "unknown"
}


// cleanServiceName cleans up service names
func cleanServiceName(service string) string {

	service = strings.TrimPrefix(service, "prometheus-stack-")
	service = strings.TrimSuffix(service, "-metrics")
	service = strings.TrimSuffix(service, "-exporter")
	
	if strings.Contains(service, "kube-state-metrics") {
		return "kube-state-metrics"
	}
	
	return service
}

// extractServiceFromInstance tries to extract service name from instance field
func extractServiceFromInstance(instance string) string {
	// Remove port numbers
	if colonIndex := strings.LastIndex(instance, ":"); colonIndex != -1 {
		instance = instance[:colonIndex]
	}
	
	// Remove common prefixes
	instance = strings.TrimPrefix(instance, "http://")
	instance = strings.TrimPrefix(instance, "https://")
	
	// Extract hostname/service part
	if dotIndex := strings.Index(instance, "."); dotIndex != -1 {
		return instance[:dotIndex]
	}
	
	return instance
}

// extractServiceFromAlertname tries to extract service from alert name
func extractServiceFromAlertname(alertname string) string {
	
	var result strings.Builder
	for i, r := range alertname {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result.WriteRune('-')
		}
		result.WriteRune(r)
	}
	
	service := strings.ToLower(result.String())
	
	// Clean up common patterns
	service = strings.TrimSuffix(service, "-down")
	service = strings.TrimSuffix(service, "-high")
	service = strings.TrimSuffix(service, "-alert")
	
	return service
}