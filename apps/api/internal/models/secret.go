package models

import (
	"time"
)

// Secret represents an encrypted secret stored in the database.
// The encrypted_value field contains base64-encoded AES-256-GCM encrypted data.
type Secret struct {
	ID             string `db:"id" json:"id"`
	Name           string `db:"name" json:"name"`
	EncryptedValue string `db:"encrypted_value" json:"-"` // Never expose in JSON

	// Timestamps
	CreatedAt time.Time `db:"created_at" json:"createdAt"`
	UpdatedAt time.Time `db:"updated_at" json:"updatedAt"`
}

// SecretInfo represents secret metadata without the encrypted value.
// Used for listing secrets without exposing sensitive data.
type SecretInfo struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// ToInfo converts a Secret to SecretInfo (without encrypted value).
func (s *Secret) ToInfo() *SecretInfo {
	return &SecretInfo{
		ID:        s.ID,
		Name:      s.Name,
		CreatedAt: s.CreatedAt,
		UpdatedAt: s.UpdatedAt,
	}
}
