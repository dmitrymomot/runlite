package spec_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/dmitrymomot/runlite/internal/spec"
)

// TestArtifact_UnmarshalYAML tests the custom unmarshaling logic for artifacts.
func TestArtifact_UnmarshalYAML(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		yaml      string
		expected  spec.Artifact
		wantError bool
		errMsg    string
	}{
		{
			name: "string format - same source and dest",
			yaml: `artifacts:
  - "./bin/server"`,
			expected: spec.Artifact{
				Source: "./bin/server",
				Dest:   "./bin/server",
			},
		},
		{
			name: "object format - different source and dest",
			yaml: `artifacts:
  - source: "./bin/app"
    dest: "./app"`,
			expected: spec.Artifact{
				Source: "./bin/app",
				Dest:   "./app",
			},
		},
		{
			name: "object format - same source and dest",
			yaml: `artifacts:
  - source: "./bin/server"
    dest: "./bin/server"`,
			expected: spec.Artifact{
				Source: "./bin/server",
				Dest:   "./bin/server",
			},
		},
		{
			name: "invalid node type - array",
			yaml: `artifacts:
  - ["./bin/server"]`,
			wantError: true,
			errMsg:    "must be string or object",
		},
		{
			name: "object missing source field",
			yaml: `artifacts:
  - dest: "./app"`,
			expected: spec.Artifact{
				Source: "",
				Dest:   "./app",
			},
		},
		{
			name: "object missing dest field",
			yaml: `artifacts:
  - source: "./bin/app"`,
			expected: spec.Artifact{
				Source: "./bin/app",
				Dest:   "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var build struct {
				Artifacts []spec.Artifact `yaml:"artifacts"`
			}

			err := yaml.Unmarshal([]byte(tt.yaml), &build)

			if tt.wantError {
				require.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
				return
			}

			require.NoError(t, err)
			require.Len(t, build.Artifacts, 1)
			assert.Equal(t, tt.expected, build.Artifacts[0])
		})
	}
}

// TestArtifact_UnmarshalYAML_MixedFormats tests mixed string and object formats.
func TestArtifact_UnmarshalYAML_MixedFormats(t *testing.T) {
	t.Parallel()

	yamlContent := `artifacts:
  - "./bin/server"
  - source: "./bin/worker"
    dest: "./worker"
  - "./static/index.html"`

	var build struct {
		Artifacts []spec.Artifact `yaml:"artifacts"`
	}

	err := yaml.Unmarshal([]byte(yamlContent), &build)
	require.NoError(t, err)
	require.Len(t, build.Artifacts, 3)

	assert.Equal(t, spec.Artifact{Source: "./bin/server", Dest: "./bin/server"}, build.Artifacts[0])
	assert.Equal(t, spec.Artifact{Source: "./bin/worker", Dest: "./worker"}, build.Artifacts[1])
	assert.Equal(t, spec.Artifact{Source: "./static/index.html", Dest: "./static/index.html"}, build.Artifacts[2])
}

// TestHealthConfig_UnmarshalYAML tests duration parsing in HealthConfig.
func TestHealthConfig_UnmarshalYAML(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		yaml      string
		checkFunc func(*testing.T, *spec.HealthConfig)
		wantError bool
		errMsg    string
	}{
		{
			name: "valid durations - seconds",
			yaml: `health:
  path: "/health"
  timeout: "30s"
  interval: "1s"`,
			checkFunc: func(t *testing.T, h *spec.HealthConfig) {
				assert.Equal(t, "/health", h.Path)
				assert.Equal(t, 30*time.Second, h.Timeout)
				assert.Equal(t, 1*time.Second, h.Interval)
			},
		},
		{
			name: "valid durations - minutes",
			yaml: `health:
  path: "/api/health"
  timeout: "1m"
  interval: "5s"`,
			checkFunc: func(t *testing.T, h *spec.HealthConfig) {
				assert.Equal(t, "/api/health", h.Path)
				assert.Equal(t, 1*time.Minute, h.Timeout)
				assert.Equal(t, 5*time.Second, h.Interval)
			},
		},
		{
			name: "valid durations - combined",
			yaml: `health:
  timeout: "1m30s"
  interval: "500ms"`,
			checkFunc: func(t *testing.T, h *spec.HealthConfig) {
				assert.Equal(t, 90*time.Second, h.Timeout)
				assert.Equal(t, 500*time.Millisecond, h.Interval)
			},
		},
		{
			name: "empty durations - should be zero",
			yaml: `health:
  path: "/health"`,
			checkFunc: func(t *testing.T, h *spec.HealthConfig) {
				assert.Equal(t, "/health", h.Path)
				assert.Equal(t, time.Duration(0), h.Timeout)
				assert.Equal(t, time.Duration(0), h.Interval)
			},
		},
		{
			name: "invalid timeout format - spaces",
			yaml: `health:
  timeout: "30 seconds"`,
			wantError: true,
			errMsg:    "health.timeout",
		},
		{
			name: "invalid timeout format - text",
			yaml: `health:
  timeout: "invalid"`,
			wantError: true,
			errMsg:    "health.timeout",
		},
		{
			name: "invalid interval format",
			yaml: `health:
  interval: "one second"`,
			wantError: true,
			errMsg:    "health.interval",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var config struct {
				Health *spec.HealthConfig `yaml:"health"`
			}

			err := yaml.Unmarshal([]byte(tt.yaml), &config)

			if tt.wantError {
				require.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
				return
			}

			require.NoError(t, err)
			require.NotNil(t, config.Health)
			tt.checkFunc(t, config.Health)
		})
	}
}

// TestDeployConfig_UnmarshalYAML tests duration parsing in DeployConfig.
func TestDeployConfig_UnmarshalYAML(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		yaml      string
		checkFunc func(*testing.T, *spec.DeployConfig)
		wantError bool
		errMsg    string
	}{
		{
			name: "valid drain period - seconds",
			yaml: `deploy:
  drain_period: "45s"`,
			checkFunc: func(t *testing.T, d *spec.DeployConfig) {
				assert.Equal(t, 45*time.Second, d.DrainPeriod)
			},
		},
		{
			name: "valid drain period - minutes",
			yaml: `deploy:
  drain_period: "2m"`,
			checkFunc: func(t *testing.T, d *spec.DeployConfig) {
				assert.Equal(t, 2*time.Minute, d.DrainPeriod)
			},
		},
		{
			name: "valid drain period - combined",
			yaml: `deploy:
  drain_period: "1m30s"`,
			checkFunc: func(t *testing.T, d *spec.DeployConfig) {
				assert.Equal(t, 90*time.Second, d.DrainPeriod)
			},
		},
		{
			name: "empty drain period - should be zero",
			yaml: `deploy: {}`,
			checkFunc: func(t *testing.T, d *spec.DeployConfig) {
				assert.Equal(t, time.Duration(0), d.DrainPeriod)
			},
		},
		{
			name: "invalid drain period format - spaces",
			yaml: `deploy:
  drain_period: "30 seconds"`,
			wantError: true,
			errMsg:    "deploy.drain_period",
		},
		{
			name: "invalid drain period format - text",
			yaml: `deploy:
  drain_period: "invalid"`,
			wantError: true,
			errMsg:    "deploy.drain_period",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var config struct {
				Deploy *spec.DeployConfig `yaml:"deploy"`
			}

			err := yaml.Unmarshal([]byte(tt.yaml), &config)

			if tt.wantError {
				require.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
				return
			}

			require.NoError(t, err)
			require.NotNil(t, config.Deploy)
			tt.checkFunc(t, config.Deploy)
		})
	}
}
