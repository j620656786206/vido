package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// A source build must carry NO key: AC #1 of sub-7-7a says the key exists only
// in release binaries built with the ldflags injection, never in the repo.
func TestBundledTMDbKey_EmptyInSourceBuild(t *testing.T) {
	assert.Equal(t, "", BundledTMDbKey())
	assert.False(t, HasBundledTMDbKey())
}

// Whitespace-only injection (a mis-set CI secret) must count as "no key",
// otherwise the resolver would hand TMDb a blank key and every call would 401.
func TestBundledTMDbKey_WhitespaceIsAbsent(t *testing.T) {
	orig := bundledTMDbKey
	t.Cleanup(func() { bundledTMDbKey = orig })

	bundledTMDbKey = "  \n"
	assert.Equal(t, "", BundledTMDbKey())
	assert.False(t, HasBundledTMDbKey())

	bundledTMDbKey = " abc123 "
	assert.Equal(t, "abc123", BundledTMDbKey())
	assert.True(t, HasBundledTMDbKey())
}
