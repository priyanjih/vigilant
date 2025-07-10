package logs

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"
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

// DefaultPatterns
var DefaultPatterns = []PatternDef{
	{"timeout", regexp.MustCompile(`(?i)timeout`)},
	{"panic", regexp.MustCompile(`(?i)panic`)},
	{"5xx", regexp.MustCompile(`(?i)5\d\d`)},
	{"unavailable", regexp.MustCompile(`(?i)unavailable`)},
	{"connection_refused", regexp.MustCompile(`(?i)connection refused`)}, 
	{"trace_export_fail", regexp.MustCompile(`(?i)traces export.*connect`)}, 
}


// ScanLogsAndMatchSymptoms scans a file for lines that match known patterns
func ScanLogsAndMatchSymptoms(logFilePath string, limit int) ([]SymptomMatch, error) {
	file, err := os.Open(logFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}
	defer file.Close()

	matches := map[string]*SymptomMatch{}
	scanner := bufio.NewScanner(file)

	linesScanned := 0

	for scanner.Scan() {
		line := scanner.Text()
		linesScanned++
		if limit > 0 && linesScanned > limit {
			break
		}

		service := extractService(line)

		for _, p := range DefaultPatterns {
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

