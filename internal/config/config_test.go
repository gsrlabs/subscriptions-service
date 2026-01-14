package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLoad checks basic loading from a file and overriding via ENV
func TestLoad(t *testing.T) {
	// Creating a temporary directory for the test config
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yml")

	// Writing the test YAML
	content := []byte(`
app:
  port: "8090"
database:
  host: "localhost"
  port: 5432
  user: "user"
  name: "db"
`)
	err := os.WriteFile(configPath, content, 0644)
	require.NoError(t, err)

	t.Run("Load from file only", func(t *testing.T) {
		// Clearing environment variables before the test
		_ = os.Unsetenv("APP_PORT")
		_ = os.Unsetenv("DB_HOST")

		cfg, err := Load(configPath)
		require.NoError(t, err)

		assert.Equal(t, "8090", cfg.App.Port)
		assert.Equal(t, "localhost", cfg.Database.Host)
		assert.Equal(t, 5432, cfg.Database.Port)
	})

	t.Run("Environment variables override file", func(t *testing.T) {
		// Install ENV, which should interrupt the file
		_ = os.Setenv("APP_PORT", "9090")
		_ = os.Setenv("DB_HOST", "postgres_container")
		defer func() {
			_ = os.Unsetenv("APP_PORT")
			_ = os.Setenv("DB_HOST", "")
		}()

		cfg, err := Load(configPath)
		require.NoError(t, err)

		assert.Equal(t, "9090", cfg.App.Port) // Must be from ENV
		assert.Equal(t, "postgres_container", cfg.Database.Host)
	})

	t.Run("Config file not found", func(t *testing.T) {
		_, err := Load("non_existent.yml")
		assert.Error(t, err)
	})
}

// TestConfig_Validate checks the business logic of config validation
func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *Config
		wantErr bool
		msg     string
	}{
		{
			name: "Valid config",
			cfg: &Config{
				Database: DatabaseConfig{
					Host:     "localhost",
					Password: "pass",
				},
			},
			wantErr: false,
		},
		{
			name: "Missing password",
			cfg: &Config{
				Database: DatabaseConfig{
					Host: "localhost",
				},
			},
			wantErr: true,
			msg:     "DB_PASSWORD is required",
		},
		{
			name: "Missing host",
			cfg: &Config{
				Database: DatabaseConfig{
					Password: "pass",
				},
			},
			wantErr: true,
			msg:     "DB_HOST is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.msg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
