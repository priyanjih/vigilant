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
	Risk    string `json:"risk"`
	Summary string `json:"summary"`
}

func Summarize(input SummaryInput) (string, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return "", fmt.Errorf("OPENAI_API_KEY not set")
	}

	client := openai.NewClient(apiKey)
	ctx := context.Background()

	prompt := buildPrompt(input)

	resp, err := client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: openai.GPT4,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    "system",
				Content: `You are a DevOps assistant. Respond only in this JSON format:
{
  "risk": "High|Medium|Low",
  "summary": "short, concise root cause and fix suggestion"
}`,
			},
			{
				Role:    "user",
				Content: prompt,
			},
		},
	})
	if err != nil {
		return "", err
	}

	raw := resp.Choices[0].Message.Content
	var result RootCauseSummary
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		// Try to fallback using regex if not clean JSON
		re := regexp.MustCompile(`(?i)"?risk"?\s*[:=]\s*"?(High|Medium|Low)"?`)
		riskMatch := re.FindStringSubmatch(raw)
		summary := strings.TrimSpace(raw)
		if len(riskMatch) >= 2 {
			result.Risk = riskMatch[1]
			result.Summary = summary
			return formatOutput(result), nil
		}
		return raw, nil // fallback to raw text
	}

	return formatOutput(result), nil
}

func buildPrompt(input SummaryInput) string {
	var sb strings.Builder
	for _, c := range input.Correlations {
		sb.WriteString(fmt.Sprintf("Service: %s\n", c.Alert.Service))
		sb.WriteString(fmt.Sprintf("Alert: %s (Severity: %s)\n", c.Alert.AlertName, c.Alert.Severity))

		sb.WriteString("Symptoms:\n")
		for _, s := range c.Symptoms {
			sb.WriteString(fmt.Sprintf("- [%s] (%d hits)\n", s.Pattern, s.Count))
		}

		sb.WriteString("Metrics:\n")
		for _, m := range c.Metrics {
			sb.WriteString(fmt.Sprintf("- %s: %.2f %s %.2f\n",
				m.Check.Name, m.Value, m.Check.Operator, m.Check.Threshold))
		}

		sb.WriteString("\n")
	}
	return sb.String()
}

func formatOutput(result RootCauseSummary) string {
	return fmt.Sprintf("RISK: %s\nSUMMARY: %s", result.Risk, result.Summary)
}
