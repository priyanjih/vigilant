package summarizer

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"

	openai "github.com/sashabaranov/go-openai"
	"vigilant/pkg/logs"
	"vigilant/pkg/prometheus"
	"vigilant/pkg/risk"
)

type SummaryInput struct {
	Correlations []AlertCorrelation
}

type AlertCorrelation struct {
	Alert    risk.RiskItem
	Symptoms []logs.SymptomMatch
	Metrics  []prometheus.MetricResult
}

type RootCauseSummary struct {
	Risk              string   `json:"risk"`
	Confidence        float64  `json:"confidence"`
	RootCause         string   `json:"root_cause"`
	ImmediateActions  []string `json:"immediate_actions"`
	Investigation     []string `json:"investigation_steps"`
	Prevention        string   `json:"prevention"`
	Summary           string   `json:"summary"`  // Keep for backward compatibility
}

func Summarize(input SummaryInput) (RootCauseSummary, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return RootCauseSummary{}, fmt.Errorf("OPENAI_API_KEY not set")
	}

	client := openai.NewClient(apiKey)
	ctx := context.Background()

	systemPrompt := buildSystemPrompt()
	contextPrompt := buildContextPrompt(input)

	resp, err := client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model:       "gpt-4o",  // Use latest model
		Temperature: 0.1,       // Low temperature for consistent technical analysis
		MaxTokens:   1500,      // Adequate for detailed response
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    "system",
				Content: systemPrompt,
			},
			{
				Role:    "user", 
				Content: contextPrompt,
			},
		},
	})
	if err != nil {
		return RootCauseSummary{}, err
	}

	raw := resp.Choices[0].Message.Content
	var result RootCauseSummary
	
	// Clean the response to extract JSON
	cleanedJSON := extractJSON(raw)
	
	// Try to parse JSON response
	if err := json.Unmarshal([]byte(cleanedJSON), &result); err != nil {
		fmt.Printf("LLM JSON parsing failed: %v\nRaw response: %s\n", err, raw)
		// Enhanced fallback parsing for malformed JSON
		result = parseRawResponse(raw)
	}
	
	// Ensure backward compatibility
	if result.Summary == "" {
		result.Summary = result.RootCause
	}
	
	// Validate and set defaults
	if result.Risk == "" {
		result.Risk = "Medium"
	}
	if result.Confidence == 0 {
		result.Confidence = 0.5
	}
	
	return result, nil
}


func buildSystemPrompt() string {
	return `You are a Senior Site Reliability Engineer (SRE) with expertise in Kubernetes, service mesh (Istio), observability, and incident response. You analyze production monitoring data to provide actionable insights.

**ROLE:** Expert SRE performing root cause analysis on real production incidents.

**CONTEXT:** You receive correlated monitoring data from a production system:
- Prometheus alerts indicating service health issues
- Log pattern matches showing symptoms in application logs  
- Metrics showing threshold violations
- All data is from active production workloads

**TASK:** Provide comprehensive root cause analysis with specific remediation steps.

**ANALYSIS FRAMEWORK:**
1. Correlate alert severity with observed symptoms and metrics
2. Identify technical root cause based on service mesh, container, and application patterns
3. Prioritize immediate stabilization actions
4. Recommend investigation steps for confirmation
5. Suggest preventive measures

**RESPONSE REQUIREMENTS:**
- Return ONLY valid JSON in the exact format specified
- Be technically specific, not generic
- Focus on actionable steps, not theory
- Consider Kubernetes/Istio context when relevant
- Prioritize service restoration first, investigation second

**RESPONSE FORMAT (JSON only):**
{
  "risk": "Critical|High|Medium|Low",
  "confidence": 0.8,
  "root_cause": "Technical analysis of the specific problem based on symptoms and metrics",
  "immediate_actions": [
    "Specific action 1 with commands/steps",
    "Specific action 2 with commands/steps",
    "Specific action 3 with commands/steps"
  ],
  "investigation_steps": [
    "Check specific logs: kubectl logs -n namespace pod-name",
    "Verify specific metrics: specific Prometheus queries",
    "Validate specific configurations"
  ],
  "prevention": "Specific measures to prevent this issue in the future"
}

Respond with JSON only. No explanation outside the JSON structure.`
}

func buildContextPrompt(input SummaryInput) string {
	var sb strings.Builder
	
	sb.WriteString("=== PRODUCTION INCIDENT ANALYSIS ===\n\n")
	
	for i, c := range input.Correlations {
		if i > 0 {
			sb.WriteString("\n" + strings.Repeat("=", 50) + "\n\n")
		}
		
		// Service and Alert Context
		sb.WriteString(fmt.Sprintf("SERVICE: %s\n", c.Alert.Service))
		sb.WriteString(fmt.Sprintf("ALERT: %s\n", c.Alert.AlertName))
		sb.WriteString(fmt.Sprintf("SEVERITY: %s\n", c.Alert.Severity))
		sb.WriteString(fmt.Sprintf("ALERT_DURATION: %v\n", c.Alert.LastSeen.Sub(c.Alert.FirstSeen)))
		sb.WriteString(fmt.Sprintf("FIRST_SEEN: %s\n", c.Alert.FirstSeen.Format("2006-01-02 15:04:05 UTC")))
		sb.WriteString("\n")

		// Log Symptoms Analysis
		if len(c.Symptoms) > 0 {
			sb.WriteString("LOG_SYMPTOMS:\n")
			for _, s := range c.Symptoms {
				sb.WriteString(fmt.Sprintf("  - Pattern: %s\n", s.Pattern))
				sb.WriteString(fmt.Sprintf("    Occurrences: %d times\n", s.Count))
				sb.WriteString(fmt.Sprintf("    Last_Seen: %s\n", s.LastSeen.Format("15:04:05")))
			}
			sb.WriteString("\n")
		} else {
			sb.WriteString("LOG_SYMPTOMS: No matching log patterns detected\n\n")
		}

		// Metrics Analysis  
		if len(c.Metrics) > 0 {
			sb.WriteString("METRICS_TRIGGERED:\n")
			for _, m := range c.Metrics {
				status := "CRITICAL"
				if m.Check.Operator == ">" && m.Value > m.Check.Threshold {
					status = "THRESHOLD_EXCEEDED"
				} else if m.Check.Operator == "<" && m.Value < m.Check.Threshold {
					status = "THRESHOLD_UNDERRUN"
				}
				
				sb.WriteString(fmt.Sprintf("  - Metric: %s\n", m.Check.Name))
				sb.WriteString(fmt.Sprintf("    Current_Value: %.3f\n", m.Value))
				sb.WriteString(fmt.Sprintf("    Threshold: %s %.3f\n", m.Check.Operator, m.Check.Threshold))
				sb.WriteString(fmt.Sprintf("    Status: %s\n", status))
				sb.WriteString(fmt.Sprintf("    Weight: %d\n", m.Check.Weight))
			}
			sb.WriteString("\n")
		} else {
			sb.WriteString("METRICS_TRIGGERED: No metric thresholds violated\n\n")
		}

		// Technical Context
		sb.WriteString("TECHNICAL_CONTEXT:\n")
		if strings.Contains(c.Alert.Service, "istio") || strings.Contains(c.Alert.AlertName, "Istio") {
			sb.WriteString("  - Environment: Kubernetes with Istio service mesh\n")
			sb.WriteString("  - Component: Istio proxy sidecar container\n")
		}
		if strings.Contains(c.Alert.AlertName, "CPU") {
			sb.WriteString("  - Issue_Type: Resource utilization (CPU)\n")
		}
		if strings.Contains(c.Alert.AlertName, "Memory") {
			sb.WriteString("  - Issue_Type: Resource utilization (Memory)\n")
		}
		sb.WriteString("  - Monitoring_System: Prometheus + Elasticsearch logs\n")
		sb.WriteString("  - Alert_Correlation: Real-time multi-source analysis\n")
	}
	
	sb.WriteString("\n=== END INCIDENT DATA ===\n")
	sb.WriteString("Provide your technical analysis in the specified JSON format.")
	
	return sb.String()
}

// Legacy function for backward compatibility
func buildPrompt(input SummaryInput) string {
	return buildContextPrompt(input)
}

func extractJSON(raw string) string {
	// Look for JSON block between ```json and ``` 
	jsonBlockRe := regexp.MustCompile("(?s)```json\\s*(\\{.*?\\})\\s*```")
	if matches := jsonBlockRe.FindStringSubmatch(raw); len(matches) >= 2 {
		return strings.TrimSpace(matches[1])
	}
	
	// Look for JSON object that starts with { and ends with }
	jsonRe := regexp.MustCompile("(?s)\\{.*\\}")
	if match := jsonRe.FindString(raw); match != "" {
		return strings.TrimSpace(match)
	}
	
	// Return original if no JSON pattern found
	return raw
}

func parseRawResponse(raw string) RootCauseSummary {
	result := RootCauseSummary{
		Risk:       "Medium",
		Confidence: 0.3,
		Summary:    strings.TrimSpace(raw),
	}
	
	// Extract risk level with regex
	riskRe := regexp.MustCompile(`(?i)"?risk"?\s*[:=]\s*"?(Critical|High|Medium|Low)"?`)
	if matches := riskRe.FindStringSubmatch(raw); len(matches) >= 2 {
		result.Risk = matches[1]
	}
	
	// Extract root cause
	causeRe := regexp.MustCompile(`(?i)"?root_cause"?\s*[:=]\s*"([^"]+)"`)
	if matches := causeRe.FindStringSubmatch(raw); len(matches) >= 2 {
		result.RootCause = matches[1]
	}
	
	return result
}

func formatOutput(result RootCauseSummary) string {
	return fmt.Sprintf("RISK: %s\nSUMMARY: %s", result.Risk, result.Summary)
}

func SummarizeMany(correlations []AlertCorrelation) (map[string]RootCauseSummary, error) {
	results := make(map[string]RootCauseSummary)

	// Group all correlations by service
	grouped := make(map[string][]AlertCorrelation)
	for _, c := range correlations {
		grouped[c.Alert.Service] = append(grouped[c.Alert.Service], c)
	}

	// Summarize each group individually
	for service, group := range grouped {
		input := SummaryInput{Correlations: group}
		summary, err := Summarize(input)
		if err != nil {
			results[service] = RootCauseSummary{
				Risk:    "Unknown",
				Summary: "LLM error or insufficient data",
			}
			continue
		}
		results[service] = summary
	}

	return results, nil
}
