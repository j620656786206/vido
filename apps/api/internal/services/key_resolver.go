package services

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/vido/api/internal/secrets"
)

// KeyName identifies a provider API key. Closed set — an unknown name is a
// programming error, not an unconfigured key (see Get).
type KeyName string

const (
	// KeyClaude is the translation provider key (FR21/FR25).
	KeyClaude KeyName = "claude"
	// KeyTMDb is the metadata provider key.
	KeyTMDb KeyName = "tmdb"
	// KeyOpenAI is the optional ASR (Whisper) key.
	KeyOpenAI KeyName = "openai"
)

// KeySource says WHERE a resolved value came from, so the settings page can be
// honest about it ("目前由環境變數提供") instead of implying the user set it.
type KeySource string

const (
	// KeySourceSecret — a runtime, user-set value from the encrypted store.
	KeySourceSecret KeySource = "secret"
	// KeySourceEnv — a deploy-time environment variable.
	KeySourceEnv KeySource = "env"
	// KeySourceBundled — a build-time key compiled into the release binary
	// (sub-7-7a; today only TMDb). Lowest priority: anything the user set, in
	// the UI or the environment, wins over it. [@contract-v1→v2] additive enum
	// value — the settings page shows it as「內建」.
	KeySourceBundled KeySource = "bundled"
	// KeySourceNone — not configured anywhere. A state, never an error.
	KeySourceNone KeySource = "none"
)

// Secret names follow the established convention (the qBittorrent / DVR
// precedent): `{provider}.{field}`.
const (
	SecretNameClaude = "claude.api_key"
	SecretNameTMDb   = "tmdb.api_key"
	SecretNameOpenAI = "openai.api_key"
)

// EnvKeys carries the deploy-time values. A plain struct rather than a
// *config.Config so the resolver stays trivially testable and `services` does
// not take a dependency on the config package for three strings.
type EnvKeys struct {
	Claude string
	TMDb   string
	OpenAI string
}

// BundledKeys carries the build-time values (config.BundledTMDbKey). Kept apart
// from EnvKeys on purpose: the settings page must be able to say「內建」rather
// than「目前由環境變數提供」, and a bundled key must never look like something
// the operator configured.
type BundledKeys struct {
	TMDb string
}

// ResolverOption tunes NewKeyResolver without breaking its two-year-old
// signature (every existing caller and test passes no options).
type ResolverOption func(*keyResolver)

// WithBundledKeys registers the build-time fallback tier (sub-7-7a).
func WithBundledKeys(b BundledKeys) ResolverOption {
	return func(r *keyResolver) { r.bundled = b }
}

// KeyResolver is the single reader every runtime consumer uses for a provider
// key.
//
// [@contract-v1] — consumed by sub-2-1b (via the settings API), the subtitle
// pipeline's capability gate (sub-1-6 AC #5 re-point), and the provider holder.
// The resolution order is FIXED: an encrypted secret (runtime, user-set) wins
// over an environment variable (deploy-time), which wins over a key bundled
// into the binary (build-time, sub-7-7a). Changing that order is a Rule 20
// bump — the whole point of this contract is that what the user typed in the UI
// is what the pipeline actually uses.
type KeyResolver interface {
	// Get returns the resolved value and where it came from. A key that is
	// configured nowhere returns ("", KeySourceNone, nil) — not an error.
	Get(ctx context.Context, name KeyName) (value string, source KeySource, err error)
	// Has reports whether the key resolves to a non-blank value.
	Has(ctx context.Context, name KeyName) bool
}

// keyResolver resolves secrets-first with an env fallback.
type keyResolver struct {
	secrets secrets.SecretsServiceInterface
	env     EnvKeys
	bundled BundledKeys
	logger  *slog.Logger
}

// NewKeyResolver builds the resolver. `secretsService` may be nil (no
// ENCRYPTION_KEY) — the resolver then degrades to env-only rather than
// panicking, which keeps an env-configured deployment working. logger may be nil.
func NewKeyResolver(secretsService secrets.SecretsServiceInterface, env EnvKeys, logger *slog.Logger, opts ...ResolverOption) KeyResolver {
	if logger == nil {
		logger = slog.Default()
	}
	r := &keyResolver{
		secrets: secretsService,
		env:     env,
		logger:  logger.With("component", "key_resolver"),
	}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

// secretNameFor maps a KeyName onto its storage name, and doubles as the
// validity check for the closed set.
func secretNameFor(name KeyName) (string, bool) {
	switch name {
	case KeyClaude:
		return SecretNameClaude, true
	case KeyTMDb:
		return SecretNameTMDb, true
	case KeyOpenAI:
		return SecretNameOpenAI, true
	default:
		return "", false
	}
}

func (r *keyResolver) envValueFor(name KeyName) string {
	switch name {
	case KeyClaude:
		return r.env.Claude
	case KeyTMDb:
		return r.env.TMDb
	case KeyOpenAI:
		return r.env.OpenAI
	default:
		return ""
	}
}

// bundledValueFor returns the build-time fallback. Only TMDb has one; the
// other providers are the user's own paid keys and are never bundled.
func (r *keyResolver) bundledValueFor(name KeyName) string {
	if name == KeyTMDb {
		return r.bundled.TMDb
	}
	return ""
}

func (r *keyResolver) Get(ctx context.Context, name KeyName) (string, KeySource, error) {
	secretName, known := secretNameFor(name)
	if !known {
		// Deliberately an error, not KeySourceNone: a typo'd key name is a bug
		// and must not be indistinguishable from "the user hasn't set it yet".
		return "", KeySourceNone, fmt.Errorf("unknown key name %q", name)
	}

	if r.secrets != nil {
		value, err := r.secrets.Retrieve(ctx, secretName)
		switch {
		case err != nil:
			// Rule 13 case 3 — deliberately discarded after logging. A secrets
			// failure (a missing row, or the decryption error seen live on the
			// NAS) must degrade to env, never take down a working deployment.
			// Retrieve returns an error for "not found" too, so this is the
			// common path, hence Debug rather than Error.
			r.logger.Debug("secret unavailable, falling back to env",
				"key", name, "secret", secretName, "error", err)
		case strings.TrimSpace(value) != "":
			return value, KeySourceSecret, nil
		default:
			// A stored blank must not shadow a working env-var, or AC #3's
			// explicit-"" delete would break the deployment instead of
			// reverting it to the env value.
			r.logger.Debug("stored secret is blank, falling back to env", "key", name)
		}
	}

	if envValue := r.envValueFor(name); strings.TrimSpace(envValue) != "" {
		return envValue, KeySourceEnv, nil
	}

	if bundled := strings.TrimSpace(r.bundledValueFor(name)); bundled != "" {
		return bundled, KeySourceBundled, nil
	}

	return "", KeySourceNone, nil
}

func (r *keyResolver) Has(ctx context.Context, name KeyName) bool {
	value, _, err := r.Get(ctx, name)
	return err == nil && strings.TrimSpace(value) != ""
}
