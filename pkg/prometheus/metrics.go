package prometheus

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"text/template"
)

// metric-based rule to check against Prometheus
type MetricCheck struct {
    Name      string  `yaml:"name"`
    QueryTpl  string  `yaml:"query_tpl"`  
    Operator  string  `yaml:"operator"`
    Threshold float64 `yaml:"threshold"`
    Weight    int     `yaml:"weight"`
}

// ties a service to its metric checks
type ServiceMetricConfig struct {
	Service string
	Checks  []MetricCheck
}

//  holds one triggered check result
type MetricResult struct {
	Service string
	Check   MetricCheck
	Value   float64
}

// EvaluateMetricChecks renders and evaluates all checks per service
func EvaluateMetricChecks(promURL string, configs []ServiceMetricConfig) ([]MetricResult, error) {
	var allResults []MetricResult

	for _, cfg := range configs {
		for _, check := range cfg.Checks {
			query := RenderQuery(check.QueryTpl, map[string]string{
				"Service": cfg.Service,
			})

			url := fmt.Sprintf("%s/api/v1/query?query=%s", promURL, query)
			resp, err := http.Get(url)
			if err != nil {
				continue
			}
			defer resp.Body.Close()

			var data struct {
				Data struct {
					Result []struct {
						Value []interface{} `json:"value"`
					} `json:"result"`
				} `json:"data"`
			}

			if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
				continue
			}

			if len(data.Data.Result) > 0 {
				raw := data.Data.Result[0].Value[1].(string)
				val, _ := strconv.ParseFloat(raw, 64)

				triggered := false
				switch check.Operator {
				case ">":
					triggered = val > check.Threshold
				case "<":
					triggered = val < check.Threshold
				}

				if triggered {
					allResults = append(allResults, MetricResult{
						Service: cfg.Service,
						Check:   check,
						Value:   val,
					})
				}
			}
		}
	}

	return allResults, nil
}

// RenderQuery replaces template variables like {{.Service}} with values
func RenderQuery(tpl string, vars map[string]string) string {
	t := template.Must(template.New("query").Parse(tpl))
	var buf bytes.Buffer
	t.Execute(&buf, vars)
	return buf.String()
}
