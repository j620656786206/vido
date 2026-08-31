package crypto

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestSessionSecret_Explicit(t *testing.T) {
	a, err := SessionSecret("my-explicit-secret", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(a) != KeySize {
		t.Fatalf("want %d bytes, got %d", KeySize, len(a))
	}
	b, _ := SessionSecret("my-explicit-secret", "", "")
	if !bytes.Equal(a, b) {
		t.Fatal("explicit secret should be deterministic")
	}
	c, _ := SessionSecret("different", "", "")
	if bytes.Equal(a, c) {
		t.Fatal("different explicit secrets should differ")
	}
}

func TestSessionSecret_EncryptionKeyFallback(t *testing.T) {
	a, err := SessionSecret("", "enc-key-value", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(a) != KeySize {
		t.Fatalf("want %d bytes, got %d", KeySize, len(a))
	}
	// Must be domain-separated from the raw encryption-key derivation.
	if bytes.Equal(a, deriveKeyFromString("enc-key-value")) {
		t.Fatal("session secret must not equal the encryption key derivation")
	}
	// Explicit secret takes priority: a set VIDO_SESSION_SECRET must win over a
	// different ENCRYPTION_KEY.
	viaExplicit, _ := SessionSecret("explicit-secret", "enc-key-value", "")
	viaKey, _ := SessionSecret("", "enc-key-value", "")
	if bytes.Equal(viaExplicit, viaKey) {
		t.Fatal("explicit secret should take priority over the encryption key")
	}
}

func TestSessionSecret_PersistedRandom(t *testing.T) {
	dir := t.TempDir()
	a, err := SessionSecret("", "", dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(a) != KeySize {
		t.Fatalf("want %d bytes, got %d", KeySize, len(a))
	}
	// Persisted: a second call in the same dir returns the SAME secret.
	b, _ := SessionSecret("", "", dir)
	if !bytes.Equal(a, b) {
		t.Fatal("persisted secret should be stable across calls")
	}
	// The secret file exists with owner-only perms.
	fi, err := os.Stat(filepath.Join(dir, sessionSecretFile))
	if err != nil {
		t.Fatalf("secret file missing: %v", err)
	}
	if fi.Mode().Perm() != 0o600 {
		t.Fatalf("want 0600 perms, got %v", fi.Mode().Perm())
	}
	// A separate install (different dir) gets an independent random secret.
	c, _ := SessionSecret("", "", t.TempDir())
	if bytes.Equal(a, c) {
		t.Fatal("independent installs should get independent secrets")
	}
}
