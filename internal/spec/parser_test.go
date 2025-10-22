package spec_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmitrymomot/runlite/internal/spec"
)

// TestParseFile_MinimalConfig tests parsing of a minimal valid config.
func TestParseFile_MinimalConfig(t *testing.T) {
	t.Parallel()

	yaml := `app:
  name: my-app`

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "runlite.yml")
	err := os.WriteFile(configPath, []byte(yaml), 0o644)
	require.NoError(t, err)

	appSpec, err := spec.ParseFile(configPath)
	require.NoError(t, err)
	require.NotNil(t, appSpec)

	assert.Equal(t, "my-app", appSpec.App.Name)
	assert.Nil(t, appSpec.Build)
	assert.Nil(t, appSpec.Run)
	assert.Nil(t, appSpec.Health)
	assert.Nil(t, appSpec.Database)
	assert.Nil(t, appSpec.Static)
	assert.Nil(t, appSpec.Deploy)
}

// TestParseFile_FullConfig tests parsing of a complete configuration.
func TestParseFile_FullConfig(t *testing.T) {
	t.Parallel()

	yaml := `app:
  name: full-app

build:
  script: "go build -o ./bin/server ./cmd/server"
  artifacts:
    - source: "./bin/server"
      dest: "./server"
    - "./static/index.html"

run:
  command: "./server"
  args: ["--port", "8080"]

health:
  path: "/api/health"
  timeout: "45s"
  interval: "2s"

database:
  path: "./data"
  file: "./data/app.db"

static:
  path: "./public"

deploy:
  drain_period: "60s"`

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "runlite.yml")
	err := os.WriteFile(configPath, []byte(yaml), 0o644)
	require.NoError(t, err)

	appSpec, err := spec.ParseFile(configPath)
	require.NoError(t, err)
	require.NotNil(t, appSpec)

	// App
	assert.Equal(t, "full-app", appSpec.App.Name)

	// Build
	require.NotNil(t, appSpec.Build)
	assert.Equal(t, "go build -o ./bin/server ./cmd/server", appSpec.Build.Script)
	require.Len(t, appSpec.Build.Artifacts, 2)
	assert.Equal(t, "./bin/server", appSpec.Build.Artifacts[0].Source)
	assert.Equal(t, "./server", appSpec.Build.Artifacts[0].Dest)
	assert.Equal(t, "./static/index.html", appSpec.Build.Artifacts[1].Source)
	assert.Equal(t, "./static/index.html", appSpec.Build.Artifacts[1].Dest)

	// Run
	require.NotNil(t, appSpec.Run)
	assert.Equal(t, "./server", appSpec.Run.Command)
	require.Len(t, appSpec.Run.Args, 2)
	assert.Equal(t, "--port", appSpec.Run.Args[0])
	assert.Equal(t, "8080", appSpec.Run.Args[1])

	// Health
	require.NotNil(t, appSpec.Health)
	assert.Equal(t, "/api/health", appSpec.Health.Path)
	assert.Equal(t, 45*time.Second, appSpec.Health.Timeout)
	assert.Equal(t, 2*time.Second, appSpec.Health.Interval)

	// Database
	require.NotNil(t, appSpec.Database)
	assert.Equal(t, "./data", appSpec.Database.Path)
	assert.Equal(t, "./data/app.db", appSpec.Database.File)

	// Static
	require.NotNil(t, appSpec.Static)
	assert.Equal(t, "./public", appSpec.Static.Path)

	// Deploy
	require.NotNil(t, appSpec.Deploy)
	assert.Equal(t, 60*time.Second, appSpec.Deploy.DrainPeriod)
}

// TestParseFile_Defaults tests that default values are applied correctly.
func TestParseFile_Defaults(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		yaml      string
		checkFunc func(*testing.T, *spec.AppSpec)
	}{
		{
			name: "health defaults when section exists but empty",
			yaml: `app:
  name: test-app
health: {}`,
			checkFunc: func(t *testing.T, s *spec.AppSpec) {
				require.NotNil(t, s.Health)
				assert.Equal(t, "/health", s.Health.Path)
				assert.Equal(t, 30*time.Second, s.Health.Timeout)
				assert.Equal(t, 1*time.Second, s.Health.Interval)
			},
		},
		{
			name: "health path defaults when only timeout specified",
			yaml: `app:
  name: test-app
health:
  timeout: "60s"`,
			checkFunc: func(t *testing.T, s *spec.AppSpec) {
				require.NotNil(t, s.Health)
				assert.Equal(t, "/health", s.Health.Path)
				assert.Equal(t, 60*time.Second, s.Health.Timeout)
				assert.Equal(t, 1*time.Second, s.Health.Interval)
			},
		},
		{
			name: "health timeout defaults when only path specified",
			yaml: `app:
  name: test-app
health:
  path: "/status"`,
			checkFunc: func(t *testing.T, s *spec.AppSpec) {
				require.NotNil(t, s.Health)
				assert.Equal(t, "/status", s.Health.Path)
				assert.Equal(t, 30*time.Second, s.Health.Timeout)
				assert.Equal(t, 1*time.Second, s.Health.Interval)
			},
		},
		{
			name: "health interval defaults when only path specified",
			yaml: `app:
  name: test-app
health:
  path: "/ready"`,
			checkFunc: func(t *testing.T, s *spec.AppSpec) {
				require.NotNil(t, s.Health)
				assert.Equal(t, "/ready", s.Health.Path)
				assert.Equal(t, 30*time.Second, s.Health.Timeout)
				assert.Equal(t, 1*time.Second, s.Health.Interval)
			},
		},
		{
			name: "deploy drain_period defaults when section exists but empty",
			yaml: `app:
  name: test-app
deploy: {}`,
			checkFunc: func(t *testing.T, s *spec.AppSpec) {
				require.NotNil(t, s.Deploy)
				assert.Equal(t, 30*time.Second, s.Deploy.DrainPeriod)
			},
		},
		{
			name: "no defaults when health section missing",
			yaml: `app:
  name: test-app`,
			checkFunc: func(t *testing.T, s *spec.AppSpec) {
				assert.Nil(t, s.Health)
			},
		},
		{
			name: "no defaults when deploy section missing",
			yaml: `app:
  name: test-app`,
			checkFunc: func(t *testing.T, s *spec.AppSpec) {
				assert.Nil(t, s.Deploy)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tempDir := t.TempDir()
			configPath := filepath.Join(tempDir, "runlite.yml")
			err := os.WriteFile(configPath, []byte(tt.yaml), 0o644)
			require.NoError(t, err)

			appSpec, err := spec.ParseFile(configPath)
			require.NoError(t, err)
			require.NotNil(t, appSpec)

			tt.checkFunc(t, appSpec)
		})
	}
}

// TestParseFile_FileErrors tests file-related error cases.
func TestParseFile_FileErrors(t *testing.T) {
	t.Parallel()

	t.Run("file not found", func(t *testing.T) {
		t.Parallel()

		appSpec, err := spec.ParseFile("/nonexistent/path/config.yml")
		require.Error(t, err)
		assert.Nil(t, appSpec)
		assert.Contains(t, err.Error(), "read spec file")
	})

	t.Run("invalid YAML syntax", func(t *testing.T) {
		t.Parallel()

		invalidYAML := `app:
  name: test
    invalid: indentation
  broken: yaml`

		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "runlite.yml")
		err := os.WriteFile(configPath, []byte(invalidYAML), 0o644)
		require.NoError(t, err)

		appSpec, err := spec.ParseFile(configPath)
		require.Error(t, err)
		assert.Nil(t, appSpec)
		assert.Contains(t, err.Error(), "unmarshal spec")
	})
}

// TestParseFile_Validation_AppName tests app name validation rules.
func TestParseFile_Validation_AppName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		appName   string
		wantError bool
		errMsg    string
	}{
		{
			name:      "valid name - lowercase",
			appName:   "myapp",
			wantError: false,
		},
		{
			name:      "valid name - uppercase",
			appName:   "MYAPP",
			wantError: false,
		},
		{
			name:      "valid name - mixed case",
			appName:   "MyApp",
			wantError: false,
		},
		{
			name:      "valid name - with hyphens",
			appName:   "my-app",
			wantError: false,
		},
		{
			name:      "valid name - with underscores",
			appName:   "my_app",
			wantError: false,
		},
		{
			name:      "valid name - with numbers",
			appName:   "app123",
			wantError: false,
		},
		{
			name:      "valid name - complex",
			appName:   "My-App_123",
			wantError: false,
		},
		{
			name:      "empty name",
			appName:   "",
			wantError: true,
			errMsg:    "app.name: required field is empty",
		},
		{
			name:      "invalid characters - at sign",
			appName:   "my@app",
			wantError: true,
			errMsg:    "app.name: must match pattern",
		},
		{
			name:      "invalid characters - dot",
			appName:   "my.app",
			wantError: true,
			errMsg:    "app.name: must match pattern",
		},
		{
			name:      "invalid characters - space",
			appName:   "my app",
			wantError: true,
			errMsg:    "app.name: must match pattern",
		},
		{
			name:      "invalid characters - slash",
			appName:   "my/app",
			wantError: true,
			errMsg:    "app.name: must match pattern",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			yaml := `app:
  name: ` + tt.appName

			tempDir := t.TempDir()
			configPath := filepath.Join(tempDir, "runlite.yml")
			err := os.WriteFile(configPath, []byte(yaml), 0o644)
			require.NoError(t, err)

			appSpec, err := spec.ParseFile(configPath)

			if tt.wantError {
				require.Error(t, err)
				assert.Nil(t, appSpec)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				require.NoError(t, err)
				require.NotNil(t, appSpec)
				assert.Equal(t, tt.appName, appSpec.App.Name)
			}
		})
	}
}

// TestParseFile_Validation_Build tests build section validation.
func TestParseFile_Validation_Build(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		yaml      string
		wantError bool
		errMsg    string
	}{
		{
			name: "artifacts without script",
			yaml: `app:
  name: test-app
build:
  artifacts:
    - "./bin/server"`,
			wantError: true,
			errMsg:    "build.script: required when artifacts are specified",
		},
		{
			name: "script without artifacts is OK",
			yaml: `app:
  name: test-app
build:
  script: "go build -o ./bin/server"`,
			wantError: false,
		},
		{
			name: "artifact source missing ./ prefix",
			yaml: `app:
  name: test-app
build:
  script: "go build"
  artifacts:
    - source: "bin/server"
      dest: "./server"`,
			wantError: true,
			errMsg:    "build.artifacts[0].source: must start with ./",
		},
		{
			name: "artifact dest missing ./ prefix",
			yaml: `app:
  name: test-app
build:
  script: "go build"
  artifacts:
    - source: "./bin/server"
      dest: "server"`,
			wantError: true,
			errMsg:    "build.artifacts[0].dest: must start with ./",
		},
		{
			name: "string artifact missing ./ prefix",
			yaml: `app:
  name: test-app
build:
  script: "go build"
  artifacts:
    - "bin/server"`,
			wantError: true,
			errMsg:    "build.artifacts[0].source: must start with ./",
		},
		{
			name: "valid build with script and artifacts",
			yaml: `app:
  name: test-app
build:
  script: "go build -o ./bin/server"
  artifacts:
    - "./bin/server"`,
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tempDir := t.TempDir()
			configPath := filepath.Join(tempDir, "runlite.yml")
			err := os.WriteFile(configPath, []byte(tt.yaml), 0o644)
			require.NoError(t, err)

			appSpec, err := spec.ParseFile(configPath)

			if tt.wantError {
				require.Error(t, err)
				assert.Nil(t, appSpec)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				require.NoError(t, err)
				require.NotNil(t, appSpec)
			}
		})
	}
}

// TestParseFile_Validation_Run tests run section validation.
func TestParseFile_Validation_Run(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		yaml      string
		wantError bool
		errMsg    string
	}{
		{
			name: "command missing ./ prefix",
			yaml: `app:
  name: test-app
run:
  command: "bin/server"`,
			wantError: true,
			errMsg:    "run.command: must start with ./",
		},
		{
			name: "valid command with ./ prefix",
			yaml: `app:
  name: test-app
run:
  command: "./bin/server"`,
			wantError: false,
		},
		{
			name: "valid command with args",
			yaml: `app:
  name: test-app
run:
  command: "./bin/server"
  args: ["--port", "8080"]`,
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tempDir := t.TempDir()
			configPath := filepath.Join(tempDir, "runlite.yml")
			err := os.WriteFile(configPath, []byte(tt.yaml), 0o644)
			require.NoError(t, err)

			appSpec, err := spec.ParseFile(configPath)

			if tt.wantError {
				require.Error(t, err)
				assert.Nil(t, appSpec)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				require.NoError(t, err)
				require.NotNil(t, appSpec)
			}
		})
	}
}

// TestParseFile_Validation_Health tests health section validation.
func TestParseFile_Validation_Health(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		yaml      string
		wantError bool
		errMsg    string
	}{
		{
			name: "path missing / prefix",
			yaml: `app:
  name: test-app
health:
  path: "health"`,
			wantError: true,
			errMsg:    "health.path: must start with /",
		},
		{
			name: "valid path with / prefix",
			yaml: `app:
  name: test-app
health:
  path: "/health"`,
			wantError: false,
		},
		{
			name: "valid nested path",
			yaml: `app:
  name: test-app
health:
  path: "/api/v1/health"`,
			wantError: false,
		},
		{
			name: "timeout exceeds max (10 minutes)",
			yaml: `app:
  name: test-app
health:
  timeout: "11m"`,
			wantError: true,
			errMsg:    "health.timeout: must be ≤",
		},
		{
			name: "timeout at max boundary (10 minutes)",
			yaml: `app:
  name: test-app
health:
  timeout: "10m"`,
			wantError: false,
		},
		{
			name: "negative timeout",
			yaml: `app:
  name: test-app
health:
  timeout: "-30s"`,
			wantError: true,
			errMsg:    "health.timeout: cannot be negative",
		},
		{
			name: "negative interval",
			yaml: `app:
  name: test-app
health:
  interval: "-1s"`,
			wantError: true,
			errMsg:    "health.interval: cannot be negative",
		},
		{
			name: "valid timeout and interval",
			yaml: `app:
  name: test-app
health:
  timeout: "1m"
  interval: "5s"`,
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tempDir := t.TempDir()
			configPath := filepath.Join(tempDir, "runlite.yml")
			err := os.WriteFile(configPath, []byte(tt.yaml), 0o644)
			require.NoError(t, err)

			appSpec, err := spec.ParseFile(configPath)

			if tt.wantError {
				require.Error(t, err)
				assert.Nil(t, appSpec)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				require.NoError(t, err)
				require.NotNil(t, appSpec)
			}
		})
	}
}

// TestParseFile_Validation_Deploy tests deploy section validation.
func TestParseFile_Validation_Deploy(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		yaml      string
		wantError bool
		errMsg    string
	}{
		{
			name: "drain_period exceeds max (5 minutes)",
			yaml: `app:
  name: test-app
deploy:
  drain_period: "6m"`,
			wantError: true,
			errMsg:    "deploy.drain_period: must be ≤",
		},
		{
			name: "drain_period at max boundary (5 minutes)",
			yaml: `app:
  name: test-app
deploy:
  drain_period: "5m"`,
			wantError: false,
		},
		{
			name: "negative drain_period",
			yaml: `app:
  name: test-app
deploy:
  drain_period: "-30s"`,
			wantError: true,
			errMsg:    "deploy.drain_period: cannot be negative",
		},
		{
			name: "valid drain_period",
			yaml: `app:
  name: test-app
deploy:
  drain_period: "45s"`,
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tempDir := t.TempDir()
			configPath := filepath.Join(tempDir, "runlite.yml")
			err := os.WriteFile(configPath, []byte(tt.yaml), 0o644)
			require.NoError(t, err)

			appSpec, err := spec.ParseFile(configPath)

			if tt.wantError {
				require.Error(t, err)
				assert.Nil(t, appSpec)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				require.NoError(t, err)
				require.NotNil(t, appSpec)
			}
		})
	}
}

// TestParseFile_Validation_StaticDatabase tests static and database validation.
func TestParseFile_Validation_StaticDatabase(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		yaml      string
		wantError bool
		errMsg    string
	}{
		{
			name: "static path missing ./ prefix",
			yaml: `app:
  name: test-app
static:
  path: "public"`,
			wantError: true,
			errMsg:    "static.path: must start with ./",
		},
		{
			name: "valid static path",
			yaml: `app:
  name: test-app
static:
  path: "./public"`,
			wantError: false,
		},
		{
			name: "database path missing ./ prefix",
			yaml: `app:
  name: test-app
database:
  path: "data"`,
			wantError: true,
			errMsg:    "database.path: must start with ./",
		},
		{
			name: "database file missing ./ prefix",
			yaml: `app:
  name: test-app
database:
  file: "data/app.db"`,
			wantError: true,
			errMsg:    "database.file: must start with ./",
		},
		{
			name: "valid database config",
			yaml: `app:
  name: test-app
database:
  path: "./data"
  file: "./data/app.db"`,
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tempDir := t.TempDir()
			configPath := filepath.Join(tempDir, "runlite.yml")
			err := os.WriteFile(configPath, []byte(tt.yaml), 0o644)
			require.NoError(t, err)

			appSpec, err := spec.ParseFile(configPath)

			if tt.wantError {
				require.Error(t, err)
				assert.Nil(t, appSpec)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				require.NoError(t, err)
				require.NotNil(t, appSpec)
			}
		})
	}
}

// TestParseFile_EdgeCases tests edge cases and boundary conditions.
func TestParseFile_EdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("empty health section gets defaults", func(t *testing.T) {
		t.Parallel()

		yaml := `app:
  name: test-app
health: {}`

		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "runlite.yml")
		err := os.WriteFile(configPath, []byte(yaml), 0o644)
		require.NoError(t, err)

		appSpec, err := spec.ParseFile(configPath)
		require.NoError(t, err)
		require.NotNil(t, appSpec)
		require.NotNil(t, appSpec.Health)

		assert.Equal(t, "/health", appSpec.Health.Path)
		assert.Equal(t, 30*time.Second, appSpec.Health.Timeout)
		assert.Equal(t, 1*time.Second, appSpec.Health.Interval)
	})

	t.Run("empty deploy section gets defaults", func(t *testing.T) {
		t.Parallel()

		yaml := `app:
  name: test-app
deploy: {}`

		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "runlite.yml")
		err := os.WriteFile(configPath, []byte(yaml), 0o644)
		require.NoError(t, err)

		appSpec, err := spec.ParseFile(configPath)
		require.NoError(t, err)
		require.NotNil(t, appSpec)
		require.NotNil(t, appSpec.Deploy)

		assert.Equal(t, 30*time.Second, appSpec.Deploy.DrainPeriod)
	})

	t.Run("unknown fields are ignored", func(t *testing.T) {
		t.Parallel()

		yaml := `app:
  name: test-app
  unknown_field: "ignored"
unknown_section:
  foo: "bar"`

		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "runlite.yml")
		err := os.WriteFile(configPath, []byte(yaml), 0o644)
		require.NoError(t, err)

		appSpec, err := spec.ParseFile(configPath)
		require.NoError(t, err)
		require.NotNil(t, appSpec)
		assert.Equal(t, "test-app", appSpec.App.Name)
	})

	t.Run("large file with many artifacts", func(t *testing.T) {
		t.Parallel()

		yaml := `app:
  name: test-app
build:
  script: "make build"
  artifacts:`

		// Add 50 artifacts
		for i := 0; i < 50; i++ {
			yaml += "\n    - \"./bin/artifact" + string(rune('0'+i%10)) + "\""
		}

		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "runlite.yml")
		err := os.WriteFile(configPath, []byte(yaml), 0o644)
		require.NoError(t, err)

		appSpec, err := spec.ParseFile(configPath)
		require.NoError(t, err)
		require.NotNil(t, appSpec)
		require.NotNil(t, appSpec.Build)
		assert.Len(t, appSpec.Build.Artifacts, 50)
	})

	t.Run("empty run command is OK", func(t *testing.T) {
		t.Parallel()

		yaml := `app:
  name: test-app
run:
  args: ["--help"]`

		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "runlite.yml")
		err := os.WriteFile(configPath, []byte(yaml), 0o644)
		require.NoError(t, err)

		appSpec, err := spec.ParseFile(configPath)
		require.NoError(t, err)
		require.NotNil(t, appSpec)
		require.NotNil(t, appSpec.Run)
		assert.Empty(t, appSpec.Run.Command)
		assert.Len(t, appSpec.Run.Args, 1)
	})

	t.Run("empty health path gets default", func(t *testing.T) {
		t.Parallel()

		yaml := `app:
  name: test-app
health:
  path: ""`

		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "runlite.yml")
		err := os.WriteFile(configPath, []byte(yaml), 0o644)
		require.NoError(t, err)

		appSpec, err := spec.ParseFile(configPath)
		require.NoError(t, err)
		require.NotNil(t, appSpec)
		require.NotNil(t, appSpec.Health)
		assert.Equal(t, "/health", appSpec.Health.Path)
	})
}
