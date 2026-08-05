package services

import (
	"context"
	"fmt"
	"strings"

	"github.com/vido/api/internal/secrets"
)

// KeyState is one key's row in the settings page (AC #3's GET shape).
//
// The VALUE is never included. `Masked` is populated only for secret-sourced
// keys: an env-sourced key is a deploy secret and re-exposing any part of it
// through an API — even masked — would leak something the operator put in the
// environment precisely to keep it out of the app.
type KeyState struct {
	Name       KeyName   `json:"name"`
	Configured bool      `json:"configured"`
	Source     KeySource `json:"source"`
	Masked     string    `json:"masked,omitempty"`
}

// KeySettings is the full GET payload.
type KeySettings struct {
	Keys []KeyState `json:"keys"`
	// Writable is false when ENCRYPTION_KEY is absent — the secrets store
	// cannot safely hold anything, so the page must render read-only rather
	// than accepting input that will fail at the storage layer (AC #4).
	Writable bool `json:"writable"`
	// Reason explains a false Writable. Empty when writable.
	Reason string `json:"reason,omitempty"`
}

// ReasonEncryptionKeyMissing is the machine-readable Writable=false cause.
const ReasonEncryptionKeyMissing = "encryption_key_missing"

// ErrKeysNotWritable is returned by Save when no encryption key is configured.
var ErrKeysNotWritable = fmt.Errorf("key storage unavailable: %s", ReasonEncryptionKeyMissing)

// KeySettingsService is the settings-page half of key management: read current
// state, store/clear values. Resolution itself belongs to KeyResolver.
type KeySettingsService struct {
	resolver   KeyResolver
	secrets    secrets.SecretsServiceInterface
	writable   bool
	orderedIDs []KeyName
}

// NewKeySettingsService builds the service. `writable` comes from
// cfg.HasEncryptionKey() — the service does not import config for one bool.
func NewKeySettingsService(resolver KeyResolver, secretsService secrets.SecretsServiceInterface, writable bool) *KeySettingsService {
	return &KeySettingsService{
		resolver: resolver,
		secrets:  secretsService,
		// Stable order so the page does not reshuffle between loads.
		orderedIDs: []KeyName{KeyClaude, KeyTMDb, KeyOpenAI},
		writable:   writable && secretsService != nil,
	}
}

// List returns every key's state. Never returns a value.
func (s *KeySettingsService) List(ctx context.Context) KeySettings {
	out := KeySettings{Keys: make([]KeyState, 0, len(s.orderedIDs)), Writable: s.writable}
	if !s.writable {
		out.Reason = ReasonEncryptionKeyMissing
	}

	for _, name := range s.orderedIDs {
		value, source, err := s.resolver.Get(ctx, name)
		state := KeyState{Name: name, Source: source, Configured: err == nil && value != ""}
		if state.Configured && source == KeySourceSecret {
			state.Masked = maskKey(value)
		}
		out.Keys = append(out.Keys, state)
	}
	return out
}

// Save stores or clears keys. A nil entry means "leave untouched"; an explicit
// empty string DELETES the secret, which reverts resolution to the env value —
// the "I mistyped my key" recovery path. Without it a bad key saved from the UI
// would be unfixable from the UI.
func (s *KeySettingsService) Save(ctx context.Context, updates map[KeyName]string) error {
	if !s.writable {
		return ErrKeysNotWritable
	}

	// Validate EVERY name before touching storage, then apply in orderedIDs
	// order: map iteration is random, so failing mid-loop would otherwise
	// commit an unpredictable subset of the request (CR sub-2-1a M3). A write
	// failure can still leave earlier keys committed — the wrapped error names
	// the failing key so the caller knows where it stopped.
	for name := range updates {
		if _, known := secretNameFor(name); !known {
			return fmt.Errorf("unknown key name %q", name)
		}
	}

	for _, name := range s.orderedIDs {
		value, present := updates[name]
		if !present {
			continue
		}
		secretName, _ := secretNameFor(name)

		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			// Delete is idempotent: clearing an already-absent key is a no-op,
			// not an error, so a page that submits its whole form works.
			if err := s.secrets.Delete(ctx, secretName); err != nil {
				if exists, checkErr := s.secrets.Exists(ctx, secretName); checkErr == nil && !exists {
					continue
				}
				return fmt.Errorf("clear %s: %w", name, err)
			}
			continue
		}

		// Store the value as typed, not the trimmed copy: only the emptiness
		// check is whitespace-insensitive. Trimming a real key would silently
		// corrupt one that legitimately ends in whitespace.
		if err := s.secrets.Store(ctx, secretName, value); err != nil {
			return fmt.Errorf("store %s: %w", name, err)
		}
	}
	return nil
}

// maskKey shows enough to recognise a key and not enough to use it: first 6 and
// last 4 characters. Short values are fully masked rather than mostly revealed.
func maskKey(value string) string {
	const head, tail = 6, 4
	if len(value) <= head+tail {
		return strings.Repeat("•", len(value))
	}
	return value[:head] + "…" + value[len(value)-tail:]
}
