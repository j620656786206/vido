# Story 1.4: Secrets Management with AES-256 Encryption

Status: ready-for-dev

## Story

As a **security-conscious user**,
I want **my API keys and credentials encrypted when stored**,
So that **my sensitive data is protected even if the database file is accessed**.

## Acceptance Criteria

1. **Given** an API key is saved through the UI
   **When** it is stored in the database
   **Then** it is encrypted using AES-256-GCM encryption
   **And** the plaintext key never appears in database files

2. **Given** the `ENCRYPTION_KEY` environment variable is set
   **When** the application encrypts/decrypts secrets
   **Then** it uses this key for encryption operations

3. **Given** the `ENCRYPTION_KEY` environment variable is NOT set
   **When** the application needs an encryption key
   **Then** it derives a key from the machine ID as fallback
   **And** logs a warning recommending setting ENCRYPTION_KEY for better security

4. **Given** any application component logs data
   **When** the log contains API keys or credentials
   **Then** the sensitive values are masked (e.g., `TMDB_****1234`)
   **And** the full value never appears in logs, errors, or HTTP responses

## Tasks / Subtasks

### Task 1: Create Crypto Package with AES-256-GCM (AC: #1, #2)
- [ ] 1.1 Create `apps/api/internal/crypto/crypto.go`
- [ ] 1.2 Implement `Encrypt(plaintext []byte, key []byte) ([]byte, error)`
- [ ] 1.3 Implement `Decrypt(ciphertext []byte, key []byte) ([]byte, error)`
- [ ] 1.4 Use AES-256-GCM (Galois/Counter Mode) for authenticated encryption
- [ ] 1.5 Generate random nonce for each encryption operation

### Task 2: Implement Key Derivation (AC: #2, #3)
- [ ] 2.1 Create `apps/api/internal/crypto/key_derivation.go`
- [ ] 2.2 Implement `DeriveKeyFromEnv() ([]byte, error)` - reads ENCRYPTION_KEY
- [ ] 2.3 Implement `DeriveKeyFromMachineID() ([]byte, error)` - fallback
- [ ] 2.4 Use PBKDF2 or Argon2 for key derivation from string
- [ ] 2.5 Log warning when using machine ID fallback

### Task 3: Create Secrets Service (AC: #1, #2, #3)
- [ ] 3.1 Create `apps/api/internal/secrets/secrets_service.go`
- [ ] 3.2 Implement `SecretsService` interface with `Store`, `Retrieve`, `Delete`
- [ ] 3.3 Integrate with SettingsRepository for persistence
- [ ] 3.4 Auto-detect key source (env var vs machine ID)
- [ ] 3.5 Handle key rotation scenarios

### Task 4: Implement Secret Masking for Logs (AC: #4)
- [ ] 4.1 Create `apps/api/internal/secrets/masking.go`
- [ ] 4.2 Implement `MaskSecret(value string) string` - returns `****1234` format
- [ ] 4.3 Implement `MaskSecretFull(value string) string` - returns `****`
- [ ] 4.4 Create custom slog Handler that auto-masks sensitive fields
- [ ] 4.5 Define sensitive field patterns: `*_key`, `*_secret`, `password`, `token`

### Task 5: Create Database Schema for Encrypted Secrets (AC: #1)
- [ ] 5.1 Create migration `005_create_secrets_table.go`
- [ ] 5.2 Schema: `id`, `name`, `encrypted_value`, `created_at`, `updated_at`
- [ ] 5.3 Create `SecretsRepository` with CRUD operations
- [ ] 5.4 Ensure encrypted values are stored as base64 encoded strings

### Task 6: Update Settings Service for API Keys (AC: #1, #2)
- [ ] 6.1 Update `SettingsService` to use `SecretsService` for API keys
- [ ] 6.2 Migrate existing plaintext keys to encrypted storage
- [ ] 6.3 Add methods: `SetAPIKey(name, value)`, `GetAPIKey(name)`

### Task 7: Write Tests (AC: #1, #2, #3, #4)
- [ ] 7.1 Create `apps/api/internal/crypto/crypto_test.go`
- [ ] 7.2 Test encrypt/decrypt roundtrip
- [ ] 7.3 Test key derivation from env var
- [ ] 7.4 Test key derivation from machine ID
- [ ] 7.5 Test secret masking patterns
- [ ] 7.6 Test SecretsService integration

## Dev Notes

### Current Implementation Status

**Does NOT Exist (to be created):**
- `apps/api/internal/crypto/` - Encryption package
- `apps/api/internal/secrets/` - Secrets management service
- Migration for secrets table
- Custom slog handler for masking

**Already Exists (can be extended):**
- `apps/api/internal/config/` - Config loading (Story 1.3 adds ENCRYPTION_KEY)
- `apps/api/internal/services/settings_service.go` - Settings service
- `apps/api/internal/repository/settings_repository.go` - Settings persistence

### Architecture Requirements

From `project-context.md`:

```
Rule 2: Logging with slog ONLY
✅ slog.Info("Storing API key", "key_name", "tmdb", "key_value", MaskSecret(key))
❌ slog.Info("Storing API key", "key_value", plaintextKey)
```

**CRITICAL:** NEVER log plaintext secrets. Always use masking functions.

### AES-256-GCM Implementation Pattern

```go
// crypto/crypto.go
package crypto

import (
    "crypto/aes"
    "crypto/cipher"
    "crypto/rand"
    "errors"
    "io"
)

var (
    ErrInvalidKeySize   = errors.New("invalid key size: must be 32 bytes for AES-256")
    ErrCiphertextShort  = errors.New("ciphertext too short")
    ErrDecryptionFailed = errors.New("decryption failed: authentication error")
)

// Encrypt encrypts plaintext using AES-256-GCM
// Returns: nonce (12 bytes) + ciphertext + auth tag (16 bytes)
func Encrypt(plaintext []byte, key []byte) ([]byte, error) {
    if len(key) != 32 {
        return nil, ErrInvalidKeySize
    }

    block, err := aes.NewCipher(key)
    if err != nil {
        return nil, err
    }

    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return nil, err
    }

    nonce := make([]byte, gcm.NonceSize())
    if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
        return nil, err
    }

    // Seal appends ciphertext + auth tag to nonce
    ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
    return ciphertext, nil
}

// Decrypt decrypts ciphertext using AES-256-GCM
func Decrypt(ciphertext []byte, key []byte) ([]byte, error) {
    if len(key) != 32 {
        return nil, ErrInvalidKeySize
    }

    block, err := aes.NewCipher(key)
    if err != nil {
        return nil, err
    }

    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return nil, err
    }

    nonceSize := gcm.NonceSize()
    if len(ciphertext) < nonceSize {
        return nil, ErrCiphertextShort
    }

    nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
    plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
    if err != nil {
        return nil, ErrDecryptionFailed
    }

    return plaintext, nil
}
```

### Key Derivation Pattern

```go
// crypto/key_derivation.go
package crypto

import (
    "crypto/sha256"
    "encoding/hex"
    "os"
    "os/exec"
    "runtime"
    "strings"

    "golang.org/x/crypto/pbkdf2"
    "log/slog"
)

const (
    keyLength   = 32 // AES-256 requires 32 bytes
    iterations  = 100000
    saltDefault = "vido-secrets-salt-v1"
)

// DeriveKey returns the encryption key, preferring env var over machine ID
func DeriveKey() ([]byte, error) {
    // Try environment variable first
    if envKey := os.Getenv("ENCRYPTION_KEY"); envKey != "" {
        slog.Info("Using encryption key from ENCRYPTION_KEY environment variable")
        return deriveKeyFromString(envKey), nil
    }

    // Fallback to machine ID
    slog.Warn("ENCRYPTION_KEY not set, using machine ID as fallback",
        "recommendation", "Set ENCRYPTION_KEY environment variable for better security")

    machineID, err := getMachineID()
    if err != nil {
        return nil, err
    }
    return deriveKeyFromString(machineID), nil
}

func deriveKeyFromString(input string) []byte {
    return pbkdf2.Key([]byte(input), []byte(saltDefault), iterations, keyLength, sha256.New)
}

func getMachineID() (string, error) {
    switch runtime.GOOS {
    case "linux":
        // Try /etc/machine-id first
        if data, err := os.ReadFile("/etc/machine-id"); err == nil {
            return strings.TrimSpace(string(data)), nil
        }
        // Fallback to /var/lib/dbus/machine-id
        if data, err := os.ReadFile("/var/lib/dbus/machine-id"); err == nil {
            return strings.TrimSpace(string(data)), nil
        }
    case "darwin":
        // macOS: Use IOPlatformUUID
        out, err := exec.Command("ioreg", "-rd1", "-c", "IOPlatformExpertDevice").Output()
        if err == nil {
            // Parse IOPlatformUUID from output
            for _, line := range strings.Split(string(out), "\n") {
                if strings.Contains(line, "IOPlatformUUID") {
                    parts := strings.Split(line, "=")
                    if len(parts) == 2 {
                        uuid := strings.Trim(strings.TrimSpace(parts[1]), "\"")
                        return uuid, nil
                    }
                }
            }
        }
    case "windows":
        // Windows: Use MachineGuid from registry
        out, err := exec.Command("reg", "query",
            "HKEY_LOCAL_MACHINE\\SOFTWARE\\Microsoft\\Cryptography",
            "/v", "MachineGuid").Output()
        if err == nil {
            lines := strings.Split(string(out), "\n")
            for _, line := range lines {
                if strings.Contains(line, "MachineGuid") {
                    fields := strings.Fields(line)
                    if len(fields) >= 3 {
                        return fields[len(fields)-1], nil
                    }
                }
            }
        }
    }

    // Generate a fallback based on hostname
    hostname, _ := os.Hostname()
    hash := sha256.Sum256([]byte(hostname + "vido-fallback"))
    return hex.EncodeToString(hash[:]), nil
}
```

### Secret Masking Pattern

```go
// secrets/masking.go
package secrets

import (
    "context"
    "log/slog"
    "strings"
)

// Sensitive field name patterns
var sensitivePatterns = []string{
    "_key", "_secret", "password", "token", "credential",
    "api_key", "apikey", "auth", "encryption",
}

// MaskSecret masks a secret value, showing first 4 and last 4 chars
func MaskSecret(value string) string {
    if value == "" {
        return "(not set)"
    }
    if len(value) <= 8 {
        return "****"
    }
    return value[:4] + "****" + value[len(value)-4:]
}

// MaskSecretFull completely masks a secret
func MaskSecretFull(value string) string {
    if value == "" {
        return "(not set)"
    }
    return "****"
}

// IsSensitiveField checks if a field name indicates sensitive data
func IsSensitiveField(fieldName string) bool {
    lower := strings.ToLower(fieldName)
    for _, pattern := range sensitivePatterns {
        if strings.Contains(lower, pattern) {
            return true
        }
    }
    return false
}

// MaskingHandler wraps slog.Handler to auto-mask sensitive fields
type MaskingHandler struct {
    inner slog.Handler
}

func NewMaskingHandler(inner slog.Handler) *MaskingHandler {
    return &MaskingHandler{inner: inner}
}

func (h *MaskingHandler) Enabled(ctx context.Context, level slog.Level) bool {
    return h.inner.Enabled(ctx, level)
}

func (h *MaskingHandler) Handle(ctx context.Context, r slog.Record) error {
    // Create new record with masked attributes
    newRecord := slog.NewRecord(r.Time, r.Level, r.Message, r.PC)
    r.Attrs(func(a slog.Attr) bool {
        if IsSensitiveField(a.Key) {
            if str, ok := a.Value.Any().(string); ok {
                newRecord.AddAttrs(slog.String(a.Key, MaskSecret(str)))
            } else {
                newRecord.AddAttrs(slog.String(a.Key, "****"))
            }
        } else {
            newRecord.AddAttrs(a)
        }
        return true
    })
    return h.inner.Handle(ctx, newRecord)
}

func (h *MaskingHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
    return &MaskingHandler{inner: h.inner.WithAttrs(attrs)}
}

func (h *MaskingHandler) WithGroup(name string) slog.Handler {
    return &MaskingHandler{inner: h.inner.WithGroup(name)}
}
```

### Secrets Service Pattern

```go
// secrets/secrets_service.go
package secrets

import (
    "context"
    "encoding/base64"
    "fmt"
    "log/slog"

    "vido/internal/crypto"
    "vido/internal/repository"
)

type SecretsService struct {
    repo repository.SecretsRepositoryInterface
    key  []byte
}

type SecretsServiceInterface interface {
    Store(ctx context.Context, name string, value string) error
    Retrieve(ctx context.Context, name string) (string, error)
    Delete(ctx context.Context, name string) error
    Exists(ctx context.Context, name string) (bool, error)
}

func NewSecretsService(repo repository.SecretsRepositoryInterface) (*SecretsService, error) {
    key, err := crypto.DeriveKey()
    if err != nil {
        return nil, fmt.Errorf("failed to derive encryption key: %w", err)
    }

    return &SecretsService{
        repo: repo,
        key:  key,
    }, nil
}

func (s *SecretsService) Store(ctx context.Context, name string, value string) error {
    slog.Info("Storing secret", "name", name, "value", MaskSecret(value))

    // Encrypt the value
    encrypted, err := crypto.Encrypt([]byte(value), s.key)
    if err != nil {
        return fmt.Errorf("failed to encrypt secret: %w", err)
    }

    // Base64 encode for storage
    encoded := base64.StdEncoding.EncodeToString(encrypted)

    return s.repo.Set(ctx, name, encoded)
}

func (s *SecretsService) Retrieve(ctx context.Context, name string) (string, error) {
    slog.Debug("Retrieving secret", "name", name)

    // Get from repository
    encoded, err := s.repo.Get(ctx, name)
    if err != nil {
        return "", err
    }

    // Base64 decode
    encrypted, err := base64.StdEncoding.DecodeString(encoded)
    if err != nil {
        return "", fmt.Errorf("failed to decode secret: %w", err)
    }

    // Decrypt
    decrypted, err := crypto.Decrypt(encrypted, s.key)
    if err != nil {
        return "", fmt.Errorf("failed to decrypt secret: %w", err)
    }

    return string(decrypted), nil
}

func (s *SecretsService) Delete(ctx context.Context, name string) error {
    slog.Info("Deleting secret", "name", name)
    return s.repo.Delete(ctx, name)
}

func (s *SecretsService) Exists(ctx context.Context, name string) (bool, error) {
    return s.repo.Exists(ctx, name)
}
```

### Database Schema

```go
// migrations/005_create_secrets_table.go
package migrations

const CreateSecretsTable = `
CREATE TABLE IF NOT EXISTS secrets (
    id TEXT PRIMARY KEY,
    name TEXT UNIQUE NOT NULL,
    encrypted_value TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_secrets_name ON secrets(name);
`
```

### File Locations

| Component | Path |
|-----------|------|
| Crypto package | `apps/api/internal/crypto/crypto.go` |
| Key derivation | `apps/api/internal/crypto/key_derivation.go` |
| Crypto tests | `apps/api/internal/crypto/crypto_test.go` |
| Secrets service | `apps/api/internal/secrets/secrets_service.go` |
| Secret masking | `apps/api/internal/secrets/masking.go` |
| Secrets repository | `apps/api/internal/repository/secrets_repository.go` |
| Migration | `apps/api/internal/database/migrations/005_create_secrets_table.go` |

### Naming Conventions

From architecture documentation:

| Element | Pattern | Example |
|---------|---------|---------|
| Packages | lowercase singular | `crypto`, `secrets` |
| Interfaces | PascalCase | `SecretsServiceInterface` |
| Functions | PascalCase | `Encrypt`, `MaskSecret` |
| Files | snake_case.go | `key_derivation.go`, `secrets_service.go` |
| Tests | *_test.go | `crypto_test.go` |

### Project Structure Notes

Target directory structure after this story:

```
apps/api/internal/
├── crypto/                      # NEW DIRECTORY
│   ├── crypto.go                # AES-256-GCM encryption
│   ├── key_derivation.go        # Key from env/machine ID
│   └── crypto_test.go           # Encryption tests
├── secrets/                     # NEW DIRECTORY
│   ├── secrets_service.go       # Secrets management
│   ├── masking.go               # Secret masking for logs
│   └── secrets_service_test.go  # Service tests
├── repository/
│   └── secrets_repository.go    # NEW: Secrets persistence
├── database/
│   └── migrations/
│       └── 005_create_secrets_table.go  # NEW
```

### Security Considerations

1. **Key Length:** AES-256 requires exactly 32-byte keys
2. **Nonce Uniqueness:** Generate new random nonce for each encryption
3. **Authenticated Encryption:** GCM mode provides integrity verification
4. **Key Storage:** Never store encryption key in database
5. **Masking:** Always mask secrets before logging or returning in errors

### Testing Strategy

```go
func TestEncryptDecryptRoundtrip(t *testing.T) {
    key := make([]byte, 32)
    rand.Read(key)

    plaintext := "my-secret-api-key"

    encrypted, err := crypto.Encrypt([]byte(plaintext), key)
    require.NoError(t, err)
    require.NotEqual(t, plaintext, string(encrypted))

    decrypted, err := crypto.Decrypt(encrypted, key)
    require.NoError(t, err)
    assert.Equal(t, plaintext, string(decrypted))
}

func TestDecryptWithWrongKey(t *testing.T) {
    key1 := make([]byte, 32)
    key2 := make([]byte, 32)
    rand.Read(key1)
    rand.Read(key2)

    encrypted, _ := crypto.Encrypt([]byte("secret"), key1)

    _, err := crypto.Decrypt(encrypted, key2)
    assert.Error(t, err)
    assert.Equal(t, crypto.ErrDecryptionFailed, err)
}

func TestMaskSecret(t *testing.T) {
    tests := []struct {
        input    string
        expected string
    }{
        {"", "(not set)"},
        {"short", "****"},
        {"12345678", "****"},
        {"1234567890abcdef", "1234****cdef"},
    }

    for _, tt := range tests {
        result := secrets.MaskSecret(tt.input)
        assert.Equal(t, tt.expected, result)
    }
}
```

### Dependencies

Add to `go.mod`:
```
golang.org/x/crypto v0.18.0  // For PBKDF2
```

### Previous Story Intelligence

From Story 1.3:
- `ENCRYPTION_KEY` environment variable will be added to config
- Config validation and fail-fast behavior established
- Environment variable loading pattern with source tracking

**Dependency:** This story uses `ENCRYPTION_KEY` from Story 1.3's config.

### References

- [Source: project-context.md#Security Requirements]
- [Source: architecture.md#NFR-S2: AES-256 encryption for UI-stored keys]
- [Source: architecture.md#NFR-S3: Encryption key from env var or machine-ID]
- [Source: architecture.md#NFR-S4: Zero-logging policy for secrets]
- [Source: epics.md#Story 1.4: Secrets Management with AES-256 Encryption]

### NFR Traceability

| NFR | Requirement | Implementation |
|-----|-------------|----------------|
| NFR-S2 | AES-256 encryption for UI-stored keys | AES-256-GCM in crypto package |
| NFR-S3 | Encryption key from env var or machine-ID | DeriveKey() with fallback |
| NFR-S4 | Zero-logging policy for secrets | MaskingHandler for slog |
| FR51 | Store sensitive data in encrypted format | SecretsService with encryption |

## Dev Agent Record

### Agent Model Used

{{agent_model_name_version}}

### Debug Log References

### Completion Notes List

### File List

