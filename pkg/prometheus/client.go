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
				Service:  extractServiceFromLabels(a.Labels),
				StartsAt: a.StartsAt,
			}
			

			fmt.Printf("DEBUG Alert '%s' labels: %+v -> Service: '%s'\n", 
				alert.Name, a.Labels, alert.Service)
			
			alerts = append(alerts, alert)
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

func extractServiceFromLabels(labels map[string]string) string {

	if job, exists := labels["job"]; exists && job == "kube-state-metrics" {

		if alertname, exists := labels["alertname"]; exists {
			return extractKubernetesServiceName(alertname, labels)
		}
		return "kubernetes-cluster"
	}
	

	serviceLabels := []string{
		"service",           
		"app",            
		"job",              
		"component",        
		"app_kubernetes_io_name", 
		"k8s_app",         
		"application",      
		"workload",        
		"deployment",     
	}

	// Try each label in order
	for _, labelName := range serviceLabels {
		if service, exists := labels[labelName]; exists && service != "" {
			return cleanServiceName(service)
		}
	}


	if instance, exists := labels["instance"]; exists && instance != "" {
		return extractServiceFromInstance(instance)
	}

	if alertname, exists := labels["alertname"]; exists && alertname != "" {
		return extractServiceFromAlertname(alertname)
	}

	return "unknown"
}

// extractKubernetesServiceName creates service names for Kubernetes monitoring alerts
func extractKubernetesServiceName(alertname string, labels map[string]string) string {
	switch {
	case strings.Contains(alertname, "Pod") || strings.Contains(alertname, "pod"):

		if namespace, exists := labels["namespace"]; exists && namespace != "" {
			return fmt.Sprintf("kubernetes-pods-%s", namespace)
		}
		return "kubernetes-pods"
		
	case strings.Contains(alertname, "Node") || strings.Contains(alertname, "node"):
		return "kubernetes-nodes"
		
	case strings.Contains(alertname, "Deployment") || strings.Contains(alertname, "deployment"):
		if namespace, exists := labels["namespace"]; exists && namespace != "" {
			return fmt.Sprintf("kubernetes-deployments-%s", namespace)
		}
		return "kubernetes-deployments"
		
	case strings.Contains(alertname, "Persistent") || strings.Contains(alertname, "Volume"):
		return "kubernetes-storage"
		
	case strings.Contains(alertname, "Namespace") || strings.Contains(alertname, "namespace"):
		return "kubernetes-namespaces"
		
	default:
		return "kubernetes-cluster"
	}
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