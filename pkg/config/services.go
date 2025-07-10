package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
	"vigilant/pkg/prometheus"
)

type ServiceProfile struct {
	LogFile string                   `yaml:"log_file"`
	Metrics []prometheus.MetricCheck `yaml:"metrics"`
}

// loads all service profile files from a directory.
func LoadServiceProfiles(dir string) (map[string]ServiceProfile, error) {
	profiles := make(map[string]ServiceProfile)

	files, err := filepath.Glob(filepath.Join(dir, "*.yml"))
	if err != nil {
		return nil, fmt.Errorf("failed to glob YAML files: %w", err)
	}

	for _, file := range files {
		name := filepath.Base(file)
		service := name[:len(name)-len(filepath.Ext(name))] // filename without .yml

		data, err := os.ReadFile(file)
		if err != nil {
			fmt.Printf("Warning: cannot read file %s: %v\n", file, err)
			continue
		}

		var profile ServiceProfile
		if err := yaml.Unmarshal(data, &profile); err != nil {
			fmt.Printf("Warning: invalid YAML in %s: %v\n", file, err)
			continue
		}

		profiles[service] = profile
	}

	return profiles, nil
}
