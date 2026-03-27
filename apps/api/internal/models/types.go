package models

import (
	"database/sql"
	"encoding/json"
	"time"
)

// NullString wraps sql.NullString with proper JSON marshaling.
// Serializes to the string value when valid, null when invalid.
type NullString struct {
	sql.NullString
}

// NewNullString creates a valid NullString with the given value.
func NewNullString(s string) NullString {
	return NullString{sql.NullString{String: s, Valid: true}}
}

func (n NullString) MarshalJSON() ([]byte, error) {
	if !n.Valid {
		return []byte("null"), nil
	}
	return json.Marshal(n.String)
}

func (n *NullString) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		n.Valid = false
		n.String = ""
		return nil
	}
	n.Valid = true
	return json.Unmarshal(data, &n.String)
}

// NullInt64 wraps sql.NullInt64 with proper JSON marshaling.
// Serializes to the int64 value when valid, null when invalid.
type NullInt64 struct {
	sql.NullInt64
}

// NewNullInt64 creates a valid NullInt64 with the given value.
func NewNullInt64(i int64) NullInt64 {
	return NullInt64{sql.NullInt64{Int64: i, Valid: true}}
}

func (n NullInt64) MarshalJSON() ([]byte, error) {
	if !n.Valid {
		return []byte("null"), nil
	}
	return json.Marshal(n.Int64)
}

func (n *NullInt64) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		n.Valid = false
		n.Int64 = 0
		return nil
	}
	n.Valid = true
	return json.Unmarshal(data, &n.Int64)
}

// NullFloat64 wraps sql.NullFloat64 with proper JSON marshaling.
// Serializes to the float64 value when valid, null when invalid.
type NullFloat64 struct {
	sql.NullFloat64
}

// NewNullFloat64 creates a valid NullFloat64 with the given value.
func NewNullFloat64(f float64) NullFloat64 {
	return NullFloat64{sql.NullFloat64{Float64: f, Valid: true}}
}

func (n NullFloat64) MarshalJSON() ([]byte, error) {
	if !n.Valid {
		return []byte("null"), nil
	}
	return json.Marshal(n.Float64)
}

func (n *NullFloat64) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		n.Valid = false
		n.Float64 = 0
		return nil
	}
	n.Valid = true
	return json.Unmarshal(data, &n.Float64)
}

// NullBool wraps sql.NullBool with proper JSON marshaling.
// Serializes to the bool value when valid, null when invalid.
type NullBool struct {
	sql.NullBool
}

// NewNullBool creates a valid NullBool with the given value.
func NewNullBool(b bool) NullBool {
	return NullBool{sql.NullBool{Bool: b, Valid: true}}
}

func (n NullBool) MarshalJSON() ([]byte, error) {
	if !n.Valid {
		return []byte("null"), nil
	}
	return json.Marshal(n.Bool)
}

func (n *NullBool) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		n.Valid = false
		n.Bool = false
		return nil
	}
	n.Valid = true
	return json.Unmarshal(data, &n.Bool)
}

// NullTime wraps sql.NullTime with proper JSON marshaling.
// Serializes to the time value (ISO 8601) when valid, null when invalid.
type NullTime struct {
	sql.NullTime
}

// NewNullTime creates a valid NullTime with the given value.
func NewNullTime(t time.Time) NullTime {
	return NullTime{sql.NullTime{Time: t, Valid: true}}
}

func (n NullTime) MarshalJSON() ([]byte, error) {
	if !n.Valid {
		return []byte("null"), nil
	}
	return json.Marshal(n.Time)
}

func (n *NullTime) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		n.Valid = false
		n.Time = time.Time{}
		return nil
	}
	n.Valid = true
	return json.Unmarshal(data, &n.Time)
}
