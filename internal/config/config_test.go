package config

import (
	"os"
	"testing"
)

func TestLoad(t *testing.T) {
	// Clear environment variables before each test
	os.Clearenv()

	t.Run("loads default configuration", func(t *testing.T) {
		cfg, err := Load()
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if cfg.Port != "8080" {
			t.Errorf("expected default port 8080, got %s", cfg.Port)
		}

		if cfg.Env != "development" {
			t.Errorf("expected default env development, got %s", cfg.Env)
		}

		if cfg.LogLevel != "info" {
			t.Errorf("expected default log level info, got %s", cfg.LogLevel)
		}

		if len(cfg.CORSOrigins) != 1 || cfg.CORSOrigins[0] != "http://localhost:3000" {
			t.Errorf("expected default CORS origin, got %v", cfg.CORSOrigins)
		}

		if cfg.APIVersion != "v1" {
			t.Errorf("expected default API version v1, got %s", cfg.APIVersion)
		}

		if cfg.TMDbDefaultLanguage != "zh-TW" {
			t.Errorf("expected default TMDb language zh-TW, got %s", cfg.TMDbDefaultLanguage)
		}

		if cfg.TMDbAPIKey != "" {
			t.Errorf("expected empty TMDb API key by default, got %s", cfg.TMDbAPIKey)
		}
	})

	t.Run("loads configuration from environment variables", func(t *testing.T) {
		os.Setenv("PORT", "9000")
		os.Setenv("ENV", "production")
		os.Setenv("LOG_LEVEL", "debug")
		os.Setenv("CORS_ORIGINS", "https://example.com,https://app.example.com")
		os.Setenv("API_VERSION", "v2")
		defer os.Clearenv()

		cfg, err := Load()
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if cfg.Port != "9000" {
			t.Errorf("expected port 9000, got %s", cfg.Port)
		}

		if cfg.Env != "production" {
			t.Errorf("expected env production, got %s", cfg.Env)
		}

		if cfg.LogLevel != "debug" {
			t.Errorf("expected log level debug, got %s", cfg.LogLevel)
		}

		if len(cfg.CORSOrigins) != 2 {
			t.Errorf("expected 2 CORS origins, got %d", len(cfg.CORSOrigins))
		}

		if cfg.APIVersion != "v2" {
			t.Errorf("expected API version v2, got %s", cfg.APIVersion)
		}
	})

	t.Run("validates log level", func(t *testing.T) {
		os.Setenv("LOG_LEVEL", "invalid")
		defer os.Clearenv()

		_, err := Load()
		if err == nil {
			t.Error("expected error for invalid log level, got nil")
		}
	})

	t.Run("trims whitespace from CORS origins", func(t *testing.T) {
		os.Setenv("CORS_ORIGINS", "https://example.com , https://app.example.com ")
		defer os.Clearenv()

		cfg, err := Load()
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		for _, origin := range cfg.CORSOrigins {
			if origin != "https://example.com" && origin != "https://app.example.com" {
				t.Errorf("unexpected origin with whitespace: %q", origin)
			}
		}
	})
}

func TestIsDevelopment(t *testing.T) {
	tests := []struct {
		env      string
		expected bool
	}{
		{"development", true},
		{"dev", true},
		{"production", false},
		{"staging", false},
	}

	for _, tt := range tests {
		t.Run(tt.env, func(t *testing.T) {
			cfg := &Config{Env: tt.env}
			if got := cfg.IsDevelopment(); got != tt.expected {
				t.Errorf("IsDevelopment() = %v, expected %v", got, tt.expected)
			}
		})
	}
}

func TestIsProduction(t *testing.T) {
	tests := []struct {
		env      string
		expected bool
	}{
		{"production", true},
		{"prod", true},
		{"development", false},
		{"staging", false},
	}

	for _, tt := range tests {
		t.Run(tt.env, func(t *testing.T) {
			cfg := &Config{Env: tt.env}
			if got := cfg.IsProduction(); got != tt.expected {
				t.Errorf("IsProduction() = %v, expected %v", got, tt.expected)
			}
		})
	}
}

func TestGetPort(t *testing.T) {
	t.Run("valid port", func(t *testing.T) {
		cfg := &Config{Port: "8080"}
		port, err := cfg.GetPort()
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if port != 8080 {
			t.Errorf("expected port 8080, got %d", port)
		}
	})

	t.Run("invalid port", func(t *testing.T) {
		cfg := &Config{Port: "invalid"}
		_, err := cfg.GetPort()
		if err == nil {
			t.Error("expected error for invalid port, got nil")
		}
	})
}

func TestGetAddress(t *testing.T) {
	cfg := &Config{Port: "8080"}
	addr := cfg.GetAddress()
	if addr != ":8080" {
		t.Errorf("expected address :8080, got %s", addr)
	}
}

func TestTMDbConfiguration(t *testing.T) {
	t.Run("loads TMDB_API_KEY from environment", func(t *testing.T) {
		os.Clearenv()
		os.Setenv("TMDB_API_KEY", "test-api-key-123")
		defer os.Clearenv()

		cfg, err := Load()
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if cfg.TMDbAPIKey != "test-api-key-123" {
			t.Errorf("expected TMDb API key 'test-api-key-123', got %s", cfg.TMDbAPIKey)
		}
	})

	t.Run("verifies default language is zh-TW", func(t *testing.T) {
		os.Clearenv()
		defer os.Clearenv()

		cfg, err := Load()
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if cfg.TMDbDefaultLanguage != "zh-TW" {
			t.Errorf("expected default TMDb language 'zh-TW', got %s", cfg.TMDbDefaultLanguage)
		}
	})

	t.Run("verifies custom language can be set", func(t *testing.T) {
		os.Clearenv()
		os.Setenv("TMDB_DEFAULT_LANGUAGE", "en-US")
		defer os.Clearenv()

		cfg, err := Load()
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if cfg.TMDbDefaultLanguage != "en-US" {
			t.Errorf("expected TMDb language 'en-US', got %s", cfg.TMDbDefaultLanguage)
		}
	})

	t.Run("loads both TMDB_API_KEY and custom language", func(t *testing.T) {
		os.Clearenv()
		os.Setenv("TMDB_API_KEY", "another-test-key")
		os.Setenv("TMDB_DEFAULT_LANGUAGE", "ja-JP")
		defer os.Clearenv()

		cfg, err := Load()
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if cfg.TMDbAPIKey != "another-test-key" {
			t.Errorf("expected TMDb API key 'another-test-key', got %s", cfg.TMDbAPIKey)
		}

		if cfg.TMDbDefaultLanguage != "ja-JP" {
			t.Errorf("expected TMDb language 'ja-JP', got %s", cfg.TMDbDefaultLanguage)
		}
	})

	t.Run("API key is empty when not set", func(t *testing.T) {
		os.Clearenv()
		defer os.Clearenv()

		cfg, err := Load()
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if cfg.TMDbAPIKey != "" {
			t.Errorf("expected empty TMDb API key, got %s", cfg.TMDbAPIKey)
		}
	})
}
