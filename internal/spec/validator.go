package spec

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

const (
	MaxHealthTimeout = 10 * time.Minute
	MaxDrainPeriod   = 5 * time.Minute
)

var appNameRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

func validate(spec *AppSpec) error {
	if spec.App.Name == "" {
		return fmt.Errorf("app.name: required field is empty")
	}

	if !appNameRegex.MatchString(spec.App.Name) {
		return fmt.Errorf("app.name: must match pattern ^[a-zA-Z0-9_-]+$")
	}

	if spec.Build != nil {
		if len(spec.Build.Artifacts) > 0 && spec.Build.Script == "" {
			return fmt.Errorf("build.script: required when artifacts are specified")
		}

		for i, artifact := range spec.Build.Artifacts {
			if !strings.HasPrefix(artifact.Source, "./") {
				return fmt.Errorf("build.artifacts[%d].source: must start with ./", i)
			}
			if !strings.HasPrefix(artifact.Dest, "./") {
				return fmt.Errorf("build.artifacts[%d].dest: must start with ./", i)
			}
		}
	}

	if spec.Run != nil && spec.Run.Command != "" {
		if !strings.HasPrefix(spec.Run.Command, "./") {
			return fmt.Errorf("run.command: must start with ./")
		}
	}

	if spec.Health != nil {
		if spec.Health.Path != "" && !strings.HasPrefix(spec.Health.Path, "/") {
			return fmt.Errorf("health.path: must start with /")
		}
		if spec.Health.Timeout < 0 {
			return fmt.Errorf("health.timeout: cannot be negative")
		}
		if spec.Health.Timeout > MaxHealthTimeout {
			return fmt.Errorf("health.timeout: must be ≤ %v", MaxHealthTimeout)
		}
		if spec.Health.Interval < 0 {
			return fmt.Errorf("health.interval: cannot be negative")
		}
	}

	if spec.Deploy != nil {
		if spec.Deploy.DrainPeriod < 0 {
			return fmt.Errorf("deploy.drain_period: cannot be negative")
		}
		if spec.Deploy.DrainPeriod > MaxDrainPeriod {
			return fmt.Errorf("deploy.drain_period: must be ≤ %v", MaxDrainPeriod)
		}
	}

	if spec.Static != nil && spec.Static.Path != "" {
		if !strings.HasPrefix(spec.Static.Path, "./") {
			return fmt.Errorf("static.path: must start with ./")
		}
	}

	if spec.Database != nil {
		if spec.Database.Path != "" && !strings.HasPrefix(spec.Database.Path, "./") {
			return fmt.Errorf("database.path: must start with ./")
		}
		if spec.Database.File != "" && !strings.HasPrefix(spec.Database.File, "./") {
			return fmt.Errorf("database.file: must start with ./")
		}
	}

	return nil
}
