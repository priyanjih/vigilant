package utils

import (
	"vigilant/pkg/logs"
	"vigilant/pkg/prometheus"
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
