package spec

import (
	"fmt"
	"time"

	"gopkg.in/yaml.v3"
)

type AppSpec struct {
	App      AppConfig       `yaml:"app"`
	Domains  []string        `yaml:"domains"`
	Build    *BuildConfig    `yaml:"build,omitempty"`
	Run      *RunConfig      `yaml:"run,omitempty"`
	Health   *HealthConfig   `yaml:"health,omitempty"`
	Database *DatabaseConfig `yaml:"database,omitempty"`
	Static   *StaticConfig   `yaml:"static,omitempty"`
	Deploy   *DeployConfig   `yaml:"deploy,omitempty"`
}

type AppConfig struct {
	Name string `yaml:"name"`
}

type BuildConfig struct {
	Script    string        `yaml:"script,omitempty"`
	Artifacts []Artifact    `yaml:"artifacts,omitempty"`
	Timeout   time.Duration `yaml:"timeout,omitempty"`
}

func (b *BuildConfig) UnmarshalYAML(value *yaml.Node) error {
	if b == nil {
		return fmt.Errorf("cannot unmarshal into nil BuildConfig")
	}

	var raw struct {
		Script    string     `yaml:"script"`
		Artifacts []Artifact `yaml:"artifacts"`
		Timeout   string     `yaml:"timeout"`
	}
	if err := value.Decode(&raw); err != nil {
		return err
	}

	b.Script = raw.Script
	b.Artifacts = raw.Artifacts

	if raw.Timeout != "" {
		timeout, err := time.ParseDuration(raw.Timeout)
		if err != nil {
			return fmt.Errorf("build.timeout: %w", err)
		}
		b.Timeout = timeout
	}

	return nil
}

type Artifact struct {
	Source string `yaml:"source"`
	Dest   string `yaml:"dest"`
}

func (a *Artifact) UnmarshalYAML(value *yaml.Node) error {
	if a == nil {
		return fmt.Errorf("cannot unmarshal into nil Artifact")
	}

	if value.Kind == yaml.ScalarNode {
		path := value.Value
		*a = Artifact{Source: path, Dest: path}
		return nil
	}

	if value.Kind == yaml.MappingNode {
		var raw struct {
			Source string `yaml:"source"`
			Dest   string `yaml:"dest"`
		}
		if err := value.Decode(&raw); err != nil {
			return err
		}
		*a = Artifact{Source: raw.Source, Dest: raw.Dest}
		return nil
	}

	return fmt.Errorf("build.artifacts: must be string or object with source/dest, got %v", value.Kind)
}

type RunConfig struct {
	Command string   `yaml:"command,omitempty"`
	Args    []string `yaml:"args,omitempty"`
}

type HealthConfig struct {
	Path     string        `yaml:"path,omitempty"`
	Timeout  time.Duration `yaml:"timeout,omitempty"`
	Interval time.Duration `yaml:"interval,omitempty"`
}

func (h *HealthConfig) UnmarshalYAML(value *yaml.Node) error {
	if h == nil {
		return fmt.Errorf("cannot unmarshal into nil HealthConfig")
	}

	var raw struct {
		Path     string `yaml:"path"`
		Timeout  string `yaml:"timeout"`
		Interval string `yaml:"interval"`
	}
	if err := value.Decode(&raw); err != nil {
		return err
	}

	h.Path = raw.Path

	if raw.Timeout != "" {
		timeout, err := time.ParseDuration(raw.Timeout)
		if err != nil {
			return fmt.Errorf("health.timeout: %w", err)
		}
		h.Timeout = timeout
	}

	if raw.Interval != "" {
		interval, err := time.ParseDuration(raw.Interval)
		if err != nil {
			return fmt.Errorf("health.interval: %w", err)
		}
		h.Interval = interval
	}

	return nil
}

type DatabaseConfig struct {
	Path string `yaml:"path,omitempty"`
	File string `yaml:"file,omitempty"`
}

type StaticConfig struct {
	Path string `yaml:"path,omitempty"`
}

type DeployConfig struct {
	DrainPeriod      time.Duration `yaml:"drain_period,omitempty"`
	KeepReleases     *int          `yaml:"keep_releases,omitempty"`
	CleanupOnFailure bool          `yaml:"cleanup_on_failure,omitempty"`
}

func (d *DeployConfig) UnmarshalYAML(value *yaml.Node) error {
	if d == nil {
		return fmt.Errorf("cannot unmarshal into nil DeployConfig")
	}

	var raw struct {
		DrainPeriod      string `yaml:"drain_period"`
		KeepReleases     *int   `yaml:"keep_releases"`
		CleanupOnFailure bool   `yaml:"cleanup_on_failure"`
	}
	if err := value.Decode(&raw); err != nil {
		return err
	}

	if raw.DrainPeriod != "" {
		drainPeriod, err := time.ParseDuration(raw.DrainPeriod)
		if err != nil {
			return fmt.Errorf("deploy.drain_period: %w", err)
		}
		d.DrainPeriod = drainPeriod
	}

	d.KeepReleases = raw.KeepReleases
	d.CleanupOnFailure = raw.CleanupOnFailure

	return nil
}
