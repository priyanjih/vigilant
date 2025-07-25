package logs

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
	"vigilant/pkg/config"
)

// SymptomMatch represents a detected issue from logs
type SymptomMatch struct {
	Service  string
	Pattern  string
	Count    int
	LastSeen time.Time
}

// PatternDef defines a symptom label and regex
type PatternDef struct {
	Label string
	Regex *regexp.Regexp
}

// ElasticsearchClient wraps the ES client with our methods
type ElasticsearchClient struct {
	client *elasticsearch.Client
}

// NewElasticsearchClient creates a new ES client
func NewElasticsearchClient(addresses []string) (*ElasticsearchClient, error) {
	cfg := elasticsearch.Config{
		Addresses: addresses,
	}
	
	client, err := elasticsearch.NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create elasticsearch client: %w", err)
	}
	
	return &ElasticsearchClient{client: client}, nil
}

// ESLogEntry represents a log entry from Elasticsearch
type ESLogEntry struct {
	Timestamp time.Time `json:"@timestamp"`
	Message   string    `json:"message"`
	Service   string    `json:"service,omitempty"`
	Container string    `json:"container,omitempty"`
}

// ESSearchResponse represents the Elasticsearch search response
type ESSearchResponse struct {
	Hits struct {
		Total struct {
			Value int `json:"value"`
		} `json:"total"`
		Hits []struct {
			Source ESLogEntry `json:"_source"`
		} `json:"hits"`
	} `json:"hits"`
}

// ScanLogsAndMatchSymptoms queries Elasticsearch for logs and matches patterns
func (es *ElasticsearchClient) ScanLogsAndMatchSymptoms(
	indexPattern string,
	limit int,
	patterns []config.LogPattern,
	timeRange time.Duration,
	serviceMapping *ServiceMapping,
) ([]SymptomMatch, error) {
	
	// Compile regex patterns
	compiled := []PatternDef{}
	for _, p := range patterns {
		re, err := regexp.Compile(p.Regex)
		if err != nil {
			continue
		}
		compiled = append(compiled, PatternDef{
			Label: p.Label,
			Regex: re,
		})
	}

	// Build Elasticsearch query
	query := buildQuery(timeRange, limit)
	
	// Execute search
	logs, err := es.searchLogs(indexPattern, query)
	if err != nil {
		return nil, fmt.Errorf("failed to search logs: %w", err)
	}

	// Process logs and match patterns
	matches := map[string]*SymptomMatch{}
	
	for _, log := range logs {
		service := serviceMapping.extractServiceFromLog(log)
		
		for _, p := range compiled {
			if p.Regex.MatchString(log.Message) {
				key := service + "::" + p.Label
				if _, exists := matches[key]; !exists {
					matches[key] = &SymptomMatch{
						Service:  service,
						Pattern:  p.Label,
						Count:    1,
						LastSeen: log.Timestamp,
					}
				} else {
					matches[key].Count++
					if log.Timestamp.After(matches[key].LastSeen) {
						matches[key].LastSeen = log.Timestamp
					}
				}
			}
		}
	}

	// Convert map to slice
	var result []SymptomMatch
	for _, v := range matches {
		result = append(result, *v)
	}

	return result, nil
}

// searchLogs executes the Elasticsearch query
func (es *ElasticsearchClient) searchLogs(indexPattern string, query map[string]interface{}) ([]ESLogEntry, error) {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(query); err != nil {
		return nil, fmt.Errorf("failed to encode query: %w", err)
	}

	req := esapi.SearchRequest{
		Index: []string{indexPattern},
		Body:  &buf,
	}

	res, err := req.Do(context.Background(), es.client)
	if err != nil {
		return nil, fmt.Errorf("failed to execute search: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("elasticsearch error: %s", res.String())
	}

	var response ESSearchResponse
	if err := json.NewDecoder(res.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	var logs []ESLogEntry
	for _, hit := range response.Hits.Hits {
		logs = append(logs, hit.Source)
	}

	return logs, nil
}

// buildQuery creates the Elasticsearch query
func buildQuery(timeRange time.Duration, limit int) map[string]interface{} {
	query := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"must": []map[string]interface{}{
					{
						"range": map[string]interface{}{
							"@timestamp": map[string]interface{}{
								"gte": time.Now().Add(-timeRange).Format(time.RFC3339),
								"lte": time.Now().Format(time.RFC3339),
							},
						},
					},
				},
			},
		},
		"sort": []map[string]interface{}{
			{
				"@timestamp": map[string]interface{}{
					"order": "desc",
				},
			},
		},
	}

	if limit > 0 {
		query["size"] = limit
	}

	return query
}

// Original file-based scanning function (kept for backward compatibility and fallback)
func ScanLogsAndMatchSymptoms(logFilePath string, limit int, patterns []config.LogPattern) ([]SymptomMatch, error) {
	file, err := os.Open(logFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}
	defer file.Close()

	matches := map[string]*SymptomMatch{}
	scanner := bufio.NewScanner(file)
	linesScanned := 0
	compiled := []PatternDef{}

	for _, p := range patterns {
		re, err := regexp.Compile(p.Regex)
		if err != nil {
			continue
		}
		compiled = append(compiled, PatternDef{
			Label: p.Label,
			Regex: re,
		})
	}

	for scanner.Scan() {
		line := scanner.Text()
		linesScanned++
		if limit > 0 && linesScanned > limit {
			break
		}

		service := extractService(line)
		for _, p := range compiled {
			if p.Regex.MatchString(line) {
				key := service + "::" + p.Label
				if _, exists := matches[key]; !exists {
					matches[key] = &SymptomMatch{
						Service:  service,
						Pattern:  p.Label,
						Count:    1,
						LastSeen: time.Now(),
					}
				} else {
					matches[key].Count++
					matches[key].LastSeen = time.Now()
				}
			}
		}
	}

	var result []SymptomMatch
	for _, v := range matches {
		result = append(result, *v)
	}
	return result, nil
}

func extractService(line string) string {
	if parts := strings.SplitN(line, "|", 2); len(parts) == 2 {
		container := strings.TrimSpace(parts[0])
		if strings.Contains(container, "hotrod") {
			return "hotrod"
		}
		return container
	}
	return "unknown"
}

// ServiceMapping holds service name mappings from config
type ServiceMapping struct {
	ConfiguredServices map[string]bool
}


func NewServiceMapping(profiles map[string]config.ServiceProfile) *ServiceMapping {
	services := make(map[string]bool)
	for serviceName := range profiles {
		services[serviceName] = true
	}
	return &ServiceMapping{ConfiguredServices: services}
}


func (sm *ServiceMapping) extractServiceFromLog(log ESLogEntry) string {

	if log.Service != "" {
		return sm.normalizeServiceName(log.Service)
	}
	

	if log.Container != "" {
		return sm.normalizeServiceName(log.Container)
	}

	if parts := strings.SplitN(log.Message, "|", 2); len(parts) == 2 {
		container := strings.TrimSpace(parts[0])
		return sm.normalizeServiceName(container)
	}
	
	return "unknown"
}

// normalizeServiceName tries to match container/service names to configured services
func (sm *ServiceMapping) normalizeServiceName(rawName string) string {

	if sm.ConfiguredServices[rawName] {
		return rawName
	}
	

	for configuredService := range sm.ConfiguredServices {

		if strings.Contains(strings.ToLower(rawName), strings.ToLower(configuredService)) {
			return configuredService
		}
		
		
		if strings.Contains(strings.ToLower(configuredService), strings.ToLower(rawName)) {
			return configuredService
		}
	}
	
	
	cleanName := cleanContainerName(rawName)
	
	
	if sm.ConfiguredServices[cleanName] {
		return cleanName
	}
	
	
	for configuredService := range sm.ConfiguredServices {
		if strings.Contains(strings.ToLower(cleanName), strings.ToLower(configuredService)) {
			return configuredService
		}
		if strings.Contains(strings.ToLower(configuredService), strings.ToLower(cleanName)) {
			return configuredService
		}
	}
	
	// Return the cleaned name if no match found
	return cleanName
}

// cleanContainerName removes common container name patterns
func cleanContainerName(name string) string {
	
	prefixes := []string{
		"k8s_",
		"docker_",
		"/",
	}
	
	for _, prefix := range prefixes {
		name = strings.TrimPrefix(name, prefix)
	}
	
	
	parts := strings.Split(name, "_")
	if len(parts) > 1 {
		
		return parts[0]
	}
	
	
	if idx := strings.LastIndex(name, "-"); idx != -1 {
		suffix := name[idx+1:]

		if len(suffix) >= 8 && isAlphanumeric(suffix) {
			return name[:idx]
		}
	}
	
	return name
}

// isAlphanumeric checks if string contains only letters and numbers
func isAlphanumeric(s string) bool {
	for _, r := range s {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')) {
			return false
		}
	}
	return true
}

// Legacy function for backward compatibility (file-based scanning)
func extractServiceFromLog(log ESLogEntry) string {

	sm := &ServiceMapping{ConfiguredServices: make(map[string]bool)}
	return sm.extractServiceFromLog(log)
}



func (es *ElasticsearchClient) ScanLogsWithFilters(
	indexPattern string,
	limit int,
	patterns []config.LogPattern,
	timeRange time.Duration,
	serviceFilter string,
	logLevel string,
	serviceMapping *ServiceMapping,
) ([]SymptomMatch, error) {
	
	compiled := []PatternDef{}
	for _, p := range patterns {
		re, err := regexp.Compile(p.Regex)
		if err != nil {
			continue
		}
		compiled = append(compiled, PatternDef{
			Label: p.Label,
			Regex: re,
		})
	}

	query := buildAdvancedQuery(timeRange, limit, serviceFilter, logLevel)
	
	logs, err := es.searchLogs(indexPattern, query)
	if err != nil {
		return nil, fmt.Errorf("failed to search logs: %w", err)
	}

	matches := map[string]*SymptomMatch{}
	
	for _, log := range logs {
		service := serviceMapping.extractServiceFromLog(log)
		
		for _, p := range compiled {
			if p.Regex.MatchString(log.Message) {
				key := service + "::" + p.Label
				if _, exists := matches[key]; !exists {
					matches[key] = &SymptomMatch{
						Service:  service,
						Pattern:  p.Label,
						Count:    1,
						LastSeen: log.Timestamp,
					}
				} else {
					matches[key].Count++
					if log.Timestamp.After(matches[key].LastSeen) {
						matches[key].LastSeen = log.Timestamp
					}
				}
			}
		}
	}

	var result []SymptomMatch
	for _, v := range matches {
		result = append(result, *v)
	}

	return result, nil
}

// buildAdvancedQuery creates a more complex Elasticsearch query with filters
func buildAdvancedQuery(timeRange time.Duration, limit int, serviceFilter, logLevel string) map[string]interface{} {
	mustClauses := []map[string]interface{}{
		{
			"range": map[string]interface{}{
				"@timestamp": map[string]interface{}{
					"gte": time.Now().Add(-timeRange).Format(time.RFC3339),
					"lte": time.Now().Format(time.RFC3339),
				},
			},
		},
	}

	if serviceFilter != "" {
		mustClauses = append(mustClauses, map[string]interface{}{
			"term": map[string]interface{}{
				"service.keyword": serviceFilter,
			},
		})
	}

	if logLevel != "" {
		mustClauses = append(mustClauses, map[string]interface{}{
			"term": map[string]interface{}{
				"level.keyword": logLevel,
			},
		})
	}

	query := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"must": mustClauses,
			},
		},
		"sort": []map[string]interface{}{
			{
				"@timestamp": map[string]interface{}{
					"order": "desc",
				},
			},
		},
	}

	if limit > 0 {
		query["size"] = limit
	}

	return query
}