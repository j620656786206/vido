package ai

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewProvider(t *testing.T) {
	tests := []struct {
		name    string
		cfg     FactoryConfig
		wantErr bool
		errType error
		wantProvider ProviderName
	}{
		{
			name: "gemini provider success",
			cfg: FactoryConfig{
				ProviderName: "gemini",
				GeminiAPIKey: "test-gemini-key",
			},
			wantErr: false,
			wantProvider: ProviderGemini,
		},
		{
			name: "gemini provider uppercase",
			cfg: FactoryConfig{
				ProviderName: "GEMINI",
				GeminiAPIKey: "test-gemini-key",
			},
			wantErr: false,
			wantProvider: ProviderGemini,
		},
		{
			name: "claude provider success",
			cfg: FactoryConfig{
				ProviderName: "claude",
				ClaudeAPIKey: "test-claude-key",
			},
			wantErr: false,
			wantProvider: ProviderClaude,
		},
		{
			name: "claude provider mixed case",
			cfg: FactoryConfig{
				ProviderName: "Claude",
				ClaudeAPIKey: "test-claude-key",
			},
			wantErr: false,
			wantProvider: ProviderClaude,
		},
		{
			name: "gemini without api key",
			cfg: FactoryConfig{
				ProviderName: "gemini",
				GeminiAPIKey: "",
			},
			wantErr: true,
			errType: ErrAINotConfigured,
		},
		{
			name: "claude without api key",
			cfg: FactoryConfig{
				ProviderName: "claude",
				ClaudeAPIKey: "",
			},
			wantErr: true,
			errType: ErrAINotConfigured,
		},
		{
			name: "invalid provider name",
			cfg: FactoryConfig{
				ProviderName: "openai",
			},
			wantErr: true,
			errType: ErrAINotConfigured,
		},
		{
			name: "empty provider name",
			cfg: FactoryConfig{
				ProviderName: "",
			},
			wantErr: true,
			errType: ErrAINotConfigured,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := NewProvider(tt.cfg)

			if tt.wantErr {
				assert.Error(t, err)
				assert.ErrorIs(t, err, tt.errType)
				assert.Nil(t, provider)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, provider)
				assert.Equal(t, tt.wantProvider, provider.Name())
			}
		})
	}
}

func TestNewProviderWithFallback(t *testing.T) {
	t.Run("primary succeeds", func(t *testing.T) {
		primary := FactoryConfig{
			ProviderName: "gemini",
			GeminiAPIKey: "test-key",
		}
		secondary := FactoryConfig{
			ProviderName: "claude",
			ClaudeAPIKey: "test-key",
		}

		provider, err := NewProviderWithFallback(primary, secondary)

		require.NoError(t, err)
		assert.Equal(t, ProviderGemini, provider.Name())
	})

	t.Run("primary fails fallback to secondary", func(t *testing.T) {
		primary := FactoryConfig{
			ProviderName: "gemini",
			GeminiAPIKey: "", // Missing key
		}
		secondary := FactoryConfig{
			ProviderName: "claude",
			ClaudeAPIKey: "test-key",
		}

		provider, err := NewProviderWithFallback(primary, secondary)

		require.NoError(t, err)
		assert.Equal(t, ProviderClaude, provider.Name())
	})

	t.Run("both fail", func(t *testing.T) {
		primary := FactoryConfig{
			ProviderName: "gemini",
			GeminiAPIKey: "", // Missing key
		}
		secondary := FactoryConfig{
			ProviderName: "claude",
			ClaudeAPIKey: "", // Missing key
		}

		_, err := NewProviderWithFallback(primary, secondary)

		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrAINotConfigured)
	})
}

func TestMustNewProvider_Success(t *testing.T) {
	cfg := FactoryConfig{
		ProviderName: "gemini",
		GeminiAPIKey: "test-key",
	}

	// Should not panic
	provider := MustNewProvider(cfg)
	assert.NotNil(t, provider)
	assert.Equal(t, ProviderGemini, provider.Name())
}

func TestMustNewProvider_Panic(t *testing.T) {
	cfg := FactoryConfig{
		ProviderName: "gemini",
		GeminiAPIKey: "", // Missing key
	}

	assert.Panics(t, func() {
		MustNewProvider(cfg)
	})
}
