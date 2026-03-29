package models

import (
	"time"
)

// Setting represents a key-value configuration setting in the database
type Setting struct {
	Key   string `db:"key" json:"key"`
	Value string `db:"value" json:"value"`
	Type  string `db:"type" json:"type"` // "string", "int", "bool", "json"

	// Timestamps
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

// SettingType defines the type of setting value
type SettingType string

const (
	SettingTypeString SettingType = "string"
	SettingTypeInt    SettingType = "int"
	SettingTypeBool   SettingType = "bool"
	SettingTypeJSON   SettingType = "json"
)

// SetupLibraryEntry represents a single library entry from the setup wizard.
type SetupLibraryEntry struct {
	Path        string `json:"path"`
	ContentType string `json:"content_type"`
}

// SetupConfig holds all wizard settings collected during setup.
type SetupConfig struct {
	Language        string               `json:"language"`
	QBTUrl          string               `json:"qbt_url,omitempty"`
	QBTUsername     string               `json:"qbt_username,omitempty"`
	QBTPassword     string               `json:"qbt_password,omitempty"`
	MediaFolderPath string               `json:"media_folder_path,omitempty"` // Deprecated: use Libraries
	Libraries       []SetupLibraryEntry  `json:"libraries,omitempty"`
	TMDbApiKey      string               `json:"tmdb_api_key,omitempty"`
	AIProvider      string               `json:"ai_provider,omitempty"`
	AIApiKey        string               `json:"ai_api_key,omitempty"`
}
