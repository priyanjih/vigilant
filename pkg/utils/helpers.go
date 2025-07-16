package utils

import (
	"vigilant/pkg/logs"
	"vigilant/pkg/prometheus"
	"vigilant/pkg/api"
)

func ExtractPatterns(symptoms []logs.SymptomMatch) []string {
	var patterns []string
	for _, s := range symptoms {
		patterns = append(patterns, s.Pattern)
	}
	return patterns
}

func ExtractMetricNames(metrics []prometheus.MetricResult) []string {
	var names []string
	for _, m := range metrics {
		names = append(names, m.Check.Name)
	}
	return names
}

func ConvertSymptoms(symptoms []logs.SymptomMatch) []api.APISymptom {
	var out []api.APISymptom
	for _, s := range symptoms {
		out = append(out, api.APISymptom{
			Pattern: s.Pattern,
			Count:   s.Count,
		})
	}
	return out
}

func ConvertMetrics(metrics []prometheus.MetricResult) []api.APIMetric {
	var out []api.APIMetric
	for _, m := range metrics {
		out = append(out, api.APIMetric{
			Name:      m.Check.Name,
			Value:     m.Value,
			Operator:  m.Check.Operator,
			Threshold: m.Check.Threshold,
		})
	}
	return out
}
