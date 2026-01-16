package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Task 1.1: Test new config fields loading from environment variables
// =============================================================================

func TestLoad_NewFields(t *testing.T) {
	// Save original environment
	originalEnv := os.Environ()
	defer func() {
		os.Clearenv()
		for _, e := range originalEnv {
			pair := splitEnvPair(e)
			if len(pair) == 2 {
				os.Setenv(pair[0], pair[1])
			}
		}
	}()

	tests := []struct {
		name     string
		envVars  map[string]string
		validate func(t *testing.T, cfg *Config)
	}{
		{
			name: "loads DataDir from VIDO_DATA_DIR",
			envVars: map[string]string{
				"VIDO_DATA_DIR": "/custom/data",
			},
			validate: func(t *testing.T, cfg *Config) {
				assert.Equal(t, "/custom/data", cfg.DataDir)
			},
		},
		{
			name: "loads MediaDirs from VIDO_MEDIA_DIRS comma-separated",
			envVars: map[string]string{
				"VIDO_MEDIA_DIRS": "/media1,/media2,/media3",
			},
			validate: func(t *testing.T, cfg *Config) {
				assert.Equal(t, []string{"/media1", "/media2", "/media3"}, cfg.MediaDirs)
			},
		},
		{
			name: "loads TMDbAPIKey from TMDB_API_KEY",
			envVars: map[string]string{
				"TMDB_API_KEY": "test-tmdb-key-12345",
			},
			validate: func(t *testing.T, cfg *Config) {
				assert.Equal(t, "test-tmdb-key-12345", cfg.TMDbAPIKey)
			},
		},
		{
			name: "loads GeminiAPIKey from GEMINI_API_KEY",
			envVars: map[string]string{
				"GEMINI_API_KEY": "test-gemini-key-67890",
			},
			validate: func(t *testing.T, cfg *Config) {
				assert.Equal(t, "test-gemini-key-67890", cfg.GeminiAPIKey)
			},
		},
		{
			name: "loads EncryptionKey from ENCRYPTION_KEY",
			envVars: map[string]string{
				"ENCRYPTION_KEY": "secret-encryption-key",
			},
			validate: func(t *testing.T, cfg *Config) {
				assert.Equal(t, "secret-encryption-key", cfg.EncryptionKey)
			},
		},
		{
			name: "loads LogLevel from VIDO_LOG_LEVEL",
			envVars: map[string]string{
				"VIDO_LOG_LEVEL": "debug",
			},
			validate: func(t *testing.T, cfg *Config) {
				assert.Equal(t, "debug", cfg.LogLevel)
			},
		},
		{
			name: "loads CORSOrigins from VIDO_CORS_ORIGINS",
			envVars: map[string]string{
				"VIDO_CORS_ORIGINS": "http://localhost:3000,http://example.com",
			},
			validate: func(t *testing.T, cfg *Config) {
				assert.Equal(t, []string{"http://localhost:3000", "http://example.com"}, cfg.CORSOrigins)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Clearenv()
			for k, v := range tt.envVars {
				os.Setenv(k, v)
			}

			cfg, err := Load()
			require.NoError(t, err)
			tt.validate(t, cfg)
		})
	}
}

// =============================================================================
// Task 1.2: Test VIDO_PORT with backward compatibility for PORT
// =============================================================================

func TestLoad_PortBackwardCompatibility(t *testing.T) {
	tests := []struct {
		name        string
		envVars     map[string]string
		expectedPort string
	}{
		{
			name:        "uses VIDO_PORT when set",
			envVars:     map[string]string{"VIDO_PORT": "9000"},
			expectedPort: "9000",
		},
		{
			name:        "falls back to PORT if VIDO_PORT not set",
			envVars:     map[string]string{"PORT": "9001"},
			expectedPort: "9001",
		},
		{
			name:        "VIDO_PORT takes precedence over PORT",
			envVars:     map[string]string{"VIDO_PORT": "9002", "PORT": "9003"},
			expectedPort: "9002",
		},
		{
			name:        "uses default 8080 when neither is set",
			envVars:     map[string]string{},
			expectedPort: "8080",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Clearenv()
			for k, v := range tt.envVars {
				os.Setenv(k, v)
			}

			cfg, err := Load()
			require.NoError(t, err)
			assert.Equal(t, tt.expectedPort, cfg.Port)
		})
	}
}

// =============================================================================
// Task 1.3: Test comma-separated value parsing helper
// =============================================================================

func TestGetEnvStringSliceOrDefault(t *testing.T) {
	tests := []struct {
		name         string
		envKey       string
		envValue     string
		defaultValue string
		expected     []string
	}{
		{
			name:         "parses comma-separated values",
			envKey:       "TEST_SLICE",
			envValue:     "a,b,c",
			defaultValue: "default",
			expected:     []string{"a", "b", "c"},
		},
		{
			name:         "trims whitespace from values",
			envKey:       "TEST_SLICE",
			envValue:     " a , b , c ",
			defaultValue: "default",
			expected:     []string{"a", "b", "c"},
		},
		{
			name:         "filters empty values",
			envKey:       "TEST_SLICE",
			envValue:     "a,,b,,,c",
			defaultValue: "default",
			expected:     []string{"a", "b", "c"},
		},
		{
			name:         "returns default when env not set",
			envKey:       "TEST_SLICE_UNSET",
			envValue:     "",
			defaultValue: "/media",
			expected:     []string{"/media"},
		},
		{
			name:         "handles single value",
			envKey:       "TEST_SLICE",
			envValue:     "single",
			defaultValue: "default",
			expected:     []string{"single"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Clearenv()
			if tt.envValue != "" {
				os.Setenv(tt.envKey, tt.envValue)
			}

			result := getEnvStringSliceOrDefault(tt.envKey, tt.defaultValue)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// =============================================================================
// Task 2: Test configuration source tracking
// =============================================================================

func TestLoad_SourceTracking(t *testing.T) {
	tests := []struct {
		name           string
		envVars        map[string]string
		checkKey       string
		expectedSource ConfigSource
	}{
		{
			name:           "tracks env var source for VIDO_PORT",
			envVars:        map[string]string{"VIDO_PORT": "9000"},
			checkKey:       "VIDO_PORT",
			expectedSource: SourceEnvVar,
		},
		{
			name:           "tracks default source when VIDO_PORT not set",
			envVars:        map[string]string{},
			checkKey:       "VIDO_PORT",
			expectedSource: SourceDefault,
		},
		{
			name:           "tracks env var source for VIDO_DATA_DIR",
			envVars:        map[string]string{"VIDO_DATA_DIR": "/custom"},
			checkKey:       "VIDO_DATA_DIR",
			expectedSource: SourceEnvVar,
		},
		{
			name:           "tracks default source for VIDO_DATA_DIR",
			envVars:        map[string]string{},
			checkKey:       "VIDO_DATA_DIR",
			expectedSource: SourceDefault,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Clearenv()
			for k, v := range tt.envVars {
				os.Setenv(k, v)
			}

			cfg, err := Load()
			require.NoError(t, err)
			assert.Equal(t, tt.expectedSource, cfg.Sources[tt.checkKey])
		})
	}
}

func TestConfigSource_String(t *testing.T) {
	tests := []struct {
		source   ConfigSource
		expected string
	}{
		{SourceDefault, "default"},
		{SourceEnvVar, "env"},
		{SourceConfigFile, "file"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.source.String())
		})
	}
}

// =============================================================================
// Test defaults
// =============================================================================

func TestLoad_Defaults(t *testing.T) {
	os.Clearenv()

	cfg, err := Load()
	require.NoError(t, err)

	assert.Equal(t, "8080", cfg.Port)
	assert.Equal(t, "development", cfg.Env)
	assert.Equal(t, "/vido-data", cfg.DataDir)
	assert.Equal(t, []string{"/media"}, cfg.MediaDirs)
	assert.Equal(t, "info", cfg.LogLevel)
	assert.Equal(t, []string{"*"}, cfg.CORSOrigins)
	assert.Empty(t, cfg.TMDbAPIKey)
	assert.Empty(t, cfg.GeminiAPIKey)
	assert.Empty(t, cfg.EncryptionKey)
}

// =============================================================================
// Task 2.4: Test LogConfigSources and maskSecret
// =============================================================================

// =============================================================================
// Task 3: Test validation with fail-fast
// =============================================================================

func TestValidate_Port(t *testing.T) {
	tests := []struct {
		name      string
		port      string
		wantError bool
		errorMsg  string
	}{
		{
			name:      "valid port 8080",
			port:      "8080",
			wantError: false,
		},
		{
			name:      "valid port 1",
			port:      "1",
			wantError: false,
		},
		{
			name:      "valid port 65535",
			port:      "65535",
			wantError: false,
		},
		{
			name:      "invalid port 0",
			port:      "0",
			wantError: true,
			errorMsg:  "VIDO_PORT",
		},
		{
			name:      "invalid port 65536",
			port:      "65536",
			wantError: true,
			errorMsg:  "VIDO_PORT",
		},
		{
			name:      "invalid port non-numeric",
			port:      "invalid",
			wantError: true,
			errorMsg:  "VIDO_PORT",
		},
		{
			name:      "invalid port negative",
			port:      "-1",
			wantError: true,
			errorMsg:  "VIDO_PORT",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Port:     tt.port,
				LogLevel: "info",
				DataDir:  t.TempDir(),
				Sources:  make(map[string]ConfigSource),
			}

			err := cfg.Validate()
			if tt.wantError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				// May still have other validation errors, so just check port doesn't fail
				if err != nil {
					assert.NotContains(t, err.Error(), "VIDO_PORT")
				}
			}
		})
	}
}

func TestValidate_LogLevel(t *testing.T) {
	tests := []struct {
		name      string
		logLevel  string
		wantError bool
	}{
		{name: "debug is valid", logLevel: "debug", wantError: false},
		{name: "info is valid", logLevel: "info", wantError: false},
		{name: "warn is valid", logLevel: "warn", wantError: false},
		{name: "error is valid", logLevel: "error", wantError: false},
		{name: "DEBUG uppercase is valid", logLevel: "DEBUG", wantError: false},
		{name: "invalid level", logLevel: "invalid", wantError: true},
		{name: "empty is invalid", logLevel: "", wantError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Port:     "8080",
				LogLevel: tt.logLevel,
				DataDir:  t.TempDir(),
				Sources:  make(map[string]ConfigSource),
			}

			err := cfg.Validate()
			if tt.wantError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "VIDO_LOG_LEVEL")
			} else {
				// May have other errors, just check log level doesn't fail
				if err != nil {
					assert.NotContains(t, err.Error(), "VIDO_LOG_LEVEL")
				}
			}
		})
	}
}

func TestValidate_DataDir(t *testing.T) {
	t.Run("creates directory if not exists", func(t *testing.T) {
		tempDir := t.TempDir()
		newDir := tempDir + "/new-data-dir"

		cfg := &Config{
			Port:     "8080",
			LogLevel: "info",
			DataDir:  newDir,
			Sources:  make(map[string]ConfigSource),
		}

		err := cfg.Validate()
		require.NoError(t, err)

		// Verify directory was created
		info, err := os.Stat(newDir)
		require.NoError(t, err)
		assert.True(t, info.IsDir())
	})

	t.Run("fails on empty directory", func(t *testing.T) {
		cfg := &Config{
			Port:     "8080",
			LogLevel: "info",
			DataDir:  "",
			Sources:  make(map[string]ConfigSource),
		}

		err := cfg.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "VIDO_DATA_DIR")
	})
}

func TestValidate_MultipleErrors(t *testing.T) {
	cfg := &Config{
		Port:     "invalid",
		LogLevel: "invalid",
		DataDir:  "",
		Sources:  make(map[string]ConfigSource),
	}

	err := cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "VIDO_PORT")
	assert.Contains(t, err.Error(), "VIDO_LOG_LEVEL")
	assert.Contains(t, err.Error(), "VIDO_DATA_DIR")
}

func TestValidationError(t *testing.T) {
	err := ValidationError{
		Field:   "TEST_FIELD",
		Message: "test error message",
	}

	assert.Equal(t, "TEST_FIELD: test error message", err.Error())
}

// =============================================================================
// Task 4: Test API key configuration helpers
// =============================================================================

func TestAPIKeyHelpers(t *testing.T) {
	t.Run("HasTMDbKey returns true when set", func(t *testing.T) {
		cfg := &Config{TMDbAPIKey: "test-key"}
		assert.True(t, cfg.HasTMDbKey())
	})

	t.Run("HasTMDbKey returns false when empty", func(t *testing.T) {
		cfg := &Config{TMDbAPIKey: ""}
		assert.False(t, cfg.HasTMDbKey())
	})

	t.Run("HasGeminiKey returns true when set", func(t *testing.T) {
		cfg := &Config{GeminiAPIKey: "test-key"}
		assert.True(t, cfg.HasGeminiKey())
	})

	t.Run("HasGeminiKey returns false when empty", func(t *testing.T) {
		cfg := &Config{GeminiAPIKey: ""}
		assert.False(t, cfg.HasGeminiKey())
	})

	t.Run("HasEncryptionKey returns true when set", func(t *testing.T) {
		cfg := &Config{EncryptionKey: "test-key"}
		assert.True(t, cfg.HasEncryptionKey())
	})

	t.Run("HasEncryptionKey returns false when empty", func(t *testing.T) {
		cfg := &Config{EncryptionKey: ""}
		assert.False(t, cfg.HasEncryptionKey())
	})

	t.Run("HasAIProvider returns true when Gemini key is set", func(t *testing.T) {
		cfg := &Config{GeminiAPIKey: "test-key"}
		assert.True(t, cfg.HasAIProvider())
	})

	t.Run("HasAIProvider returns false when no AI keys set", func(t *testing.T) {
		cfg := &Config{}
		assert.False(t, cfg.HasAIProvider())
	})

	t.Run("GetTMDbAPIKey returns the key", func(t *testing.T) {
		cfg := &Config{TMDbAPIKey: "my-tmdb-key"}
		assert.Equal(t, "my-tmdb-key", cfg.GetTMDbAPIKey())
	})

	t.Run("GetGeminiAPIKey returns the key", func(t *testing.T) {
		cfg := &Config{GeminiAPIKey: "my-gemini-key"}
		assert.Equal(t, "my-gemini-key", cfg.GetGeminiAPIKey())
	})

	t.Run("GetEncryptionKey returns the key", func(t *testing.T) {
		cfg := &Config{EncryptionKey: "my-encryption-key"}
		assert.Equal(t, "my-encryption-key", cfg.GetEncryptionKey())
	})
}

// =============================================================================
// Task 2.4: Test LogConfigSources and maskSecret
// =============================================================================

func TestMaskSecret(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty string returns not set",
			input:    "",
			expected: "(not set)",
		},
		{
			name:     "short string is fully masked",
			input:    "short",
			expected: "****",
		},
		{
			name:     "8 char string is fully masked",
			input:    "12345678",
			expected: "****",
		},
		{
			name:     "longer string shows first and last 4 chars",
			input:    "abcd12345678efgh",
			expected: "abcd****efgh",
		},
		{
			name:     "typical API key is partially masked",
			input:    "sk-1234567890abcdef",
			expected: "sk-1****cdef",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := maskSecret(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Helper function to split environment variable pair
func splitEnvPair(e string) []string {
	for i := 0; i < len(e); i++ {
		if e[i] == '=' {
			return []string{e[:i], e[i+1:]}
		}
	}
	return []string{e}
}
