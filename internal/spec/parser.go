package spec

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

func ParseFile(path string) (*AppSpec, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read spec file: %w", err)
	}

	var spec AppSpec
	if err := yaml.Unmarshal(data, &spec); err != nil {
		return nil, fmt.Errorf("unmarshal spec: %w", err)
	}

	applyDefaults(&spec)

	if err := validate(&spec); err != nil {
		return nil, err
	}

	return &spec, nil
}

// applyDefaults sets default values for optional fields.
// Note: Zero-value durations (0s) are treated as unset and will be replaced with defaults.
func applyDefaults(spec *AppSpec) {
	if spec.Health != nil {
		if spec.Health.Path == "" {
			spec.Health.Path = "/health"
		}
		if spec.Health.Timeout == 0 {
			spec.Health.Timeout = 30 * time.Second
		}
		if spec.Health.Interval == 0 {
			spec.Health.Interval = 1 * time.Second
		}
	}

	if spec.Deploy != nil && spec.Deploy.DrainPeriod == 0 {
		spec.Deploy.DrainPeriod = 30 * time.Second
	}
}
