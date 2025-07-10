package prometheus

import (
	"fmt"
	"os"
	"gopkg.in/yaml.v3"
)

func LoadMetricChecksFromFile(path string) ([]MetricCheck, error) {
	var checks []MetricCheck
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	
	err = yaml.Unmarshal(data, &checks)
	if err != nil {
		return nil, err
	}
	
	// Keep this for operational visibility
	fmt.Printf("Loaded %d metric checks from %s\n", len(checks), path)
	
	return checks, err
}
