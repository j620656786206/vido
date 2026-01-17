package models

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSecret_ToInfo(t *testing.T) {
	now := time.Now()

	secret := &Secret{
		ID:             "secret-123",
		Name:           "tmdb-api-key",
		EncryptedValue: "encrypted-value-should-not-be-exposed",
		CreatedAt:      now,
		UpdatedAt:      now.Add(time.Hour),
	}

	info := secret.ToInfo()

	assert.Equal(t, secret.ID, info.ID)
	assert.Equal(t, secret.Name, info.Name)
	assert.Equal(t, secret.CreatedAt, info.CreatedAt)
	assert.Equal(t, secret.UpdatedAt, info.UpdatedAt)
}

func TestSecret_ToInfo_EmptySecret(t *testing.T) {
	secret := &Secret{}

	info := secret.ToInfo()

	assert.NotNil(t, info)
	assert.Empty(t, info.ID)
	assert.Empty(t, info.Name)
}

func TestSecretInfo_Fields(t *testing.T) {
	now := time.Now()

	info := &SecretInfo{
		ID:        "info-123",
		Name:      "api-key",
		CreatedAt: now,
		UpdatedAt: now,
	}

	assert.Equal(t, "info-123", info.ID)
	assert.Equal(t, "api-key", info.Name)
	assert.Equal(t, now, info.CreatedAt)
	assert.Equal(t, now, info.UpdatedAt)
}
