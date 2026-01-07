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
	CreatedAt time.Time `db:"created_at" json:"createdAt"`
	UpdatedAt time.Time `db:"updated_at" json:"updatedAt"`
}

// SettingType defines the type of setting value
type SettingType string

const (
	SettingTypeString SettingType = "string"
	SettingTypeInt    SettingType = "int"
	SettingTypeBool   SettingType = "bool"
	SettingTypeJSON   SettingType = "json"
)
