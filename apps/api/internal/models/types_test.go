package models

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNullString_MarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		value    NullString
		expected string
	}{
		{
			name:     "valid string",
			value:    NewNullString("hello"),
			expected: `"hello"`,
		},
		{
			name:     "empty valid string",
			value:    NewNullString(""),
			expected: `""`,
		},
		{
			name:     "null string",
			value:    NullString{},
			expected: `null`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.value)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, string(data))
		})
	}
}

func TestNullString_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantValid bool
		wantValue string
	}{
		{
			name:      "valid string",
			input:     `"hello"`,
			wantValid: true,
			wantValue: "hello",
		},
		{
			name:      "null",
			input:     `null`,
			wantValid: false,
			wantValue: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var ns NullString
			err := json.Unmarshal([]byte(tt.input), &ns)
			require.NoError(t, err)
			assert.Equal(t, tt.wantValid, ns.Valid)
			assert.Equal(t, tt.wantValue, ns.String)
		})
	}
}

func TestNullString_Roundtrip(t *testing.T) {
	original := NewNullString("roundtrip test")
	data, err := json.Marshal(original)
	require.NoError(t, err)

	var restored NullString
	err = json.Unmarshal(data, &restored)
	require.NoError(t, err)
	assert.Equal(t, original.String, restored.String)
	assert.Equal(t, original.Valid, restored.Valid)
}

func TestNullInt64_MarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		value    NullInt64
		expected string
	}{
		{
			name:     "valid int",
			value:    NewNullInt64(42),
			expected: `42`,
		},
		{
			name:     "zero valid int",
			value:    NewNullInt64(0),
			expected: `0`,
		},
		{
			name:     "null int",
			value:    NullInt64{},
			expected: `null`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.value)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, string(data))
		})
	}
}

func TestNullInt64_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantValid bool
		wantValue int64
	}{
		{
			name:      "valid int",
			input:     `42`,
			wantValid: true,
			wantValue: 42,
		},
		{
			name:      "null",
			input:     `null`,
			wantValid: false,
			wantValue: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var ni NullInt64
			err := json.Unmarshal([]byte(tt.input), &ni)
			require.NoError(t, err)
			assert.Equal(t, tt.wantValid, ni.Valid)
			assert.Equal(t, tt.wantValue, ni.Int64)
		})
	}
}

func TestNullFloat64_MarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		value    NullFloat64
		expected string
	}{
		{
			name:     "valid float",
			value:    NewNullFloat64(8.5),
			expected: `8.5`,
		},
		{
			name:     "null float",
			value:    NullFloat64{},
			expected: `null`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.value)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, string(data))
		})
	}
}

func TestNullFloat64_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantValid bool
		wantValue float64
	}{
		{
			name:      "valid float",
			input:     `8.5`,
			wantValid: true,
			wantValue: 8.5,
		},
		{
			name:      "null",
			input:     `null`,
			wantValid: false,
			wantValue: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var nf NullFloat64
			err := json.Unmarshal([]byte(tt.input), &nf)
			require.NoError(t, err)
			assert.Equal(t, tt.wantValid, nf.Valid)
			assert.Equal(t, tt.wantValue, nf.Float64)
		})
	}
}

func TestNullBool_MarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		value    NullBool
		expected string
	}{
		{
			name:     "valid true",
			value:    NewNullBool(true),
			expected: `true`,
		},
		{
			name:     "valid false",
			value:    NewNullBool(false),
			expected: `false`,
		},
		{
			name:     "null bool",
			value:    NullBool{},
			expected: `null`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.value)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, string(data))
		})
	}
}

func TestNullBool_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantValid bool
		wantValue bool
	}{
		{
			name:      "valid true",
			input:     `true`,
			wantValid: true,
			wantValue: true,
		},
		{
			name:      "null",
			input:     `null`,
			wantValid: false,
			wantValue: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var nb NullBool
			err := json.Unmarshal([]byte(tt.input), &nb)
			require.NoError(t, err)
			assert.Equal(t, tt.wantValid, nb.Valid)
			assert.Equal(t, tt.wantValue, nb.Bool)
		})
	}
}

func TestNullTime_MarshalJSON(t *testing.T) {
	fixedTime := time.Date(2024, 1, 15, 14, 30, 0, 0, time.UTC)

	tests := []struct {
		name     string
		value    NullTime
		expected string
	}{
		{
			name:     "valid time",
			value:    NewNullTime(fixedTime),
			expected: `"2024-01-15T14:30:00Z"`,
		},
		{
			name:     "null time",
			value:    NullTime{},
			expected: `null`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.value)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, string(data))
		})
	}
}

func TestNullTime_UnmarshalJSON(t *testing.T) {
	t.Run("valid time", func(t *testing.T) {
		var nt NullTime
		err := json.Unmarshal([]byte(`"2024-01-15T14:30:00Z"`), &nt)
		require.NoError(t, err)
		assert.True(t, nt.Valid)
		assert.Equal(t, 2024, nt.Time.Year())
		assert.Equal(t, time.Month(1), nt.Time.Month())
		assert.Equal(t, 15, nt.Time.Day())
	})

	t.Run("null", func(t *testing.T) {
		var nt NullTime
		err := json.Unmarshal([]byte(`null`), &nt)
		require.NoError(t, err)
		assert.False(t, nt.Valid)
	})
}

func TestConstructors(t *testing.T) {
	t.Run("NewNullString", func(t *testing.T) {
		ns := NewNullString("test")
		assert.True(t, ns.Valid)
		assert.Equal(t, "test", ns.String)
	})

	t.Run("NewNullInt64", func(t *testing.T) {
		ni := NewNullInt64(99)
		assert.True(t, ni.Valid)
		assert.Equal(t, int64(99), ni.Int64)
	})

	t.Run("NewNullFloat64", func(t *testing.T) {
		nf := NewNullFloat64(3.14)
		assert.True(t, nf.Valid)
		assert.Equal(t, 3.14, nf.Float64)
	})

	t.Run("NewNullBool", func(t *testing.T) {
		nb := NewNullBool(true)
		assert.True(t, nb.Valid)
		assert.True(t, nb.Bool)
	})

	t.Run("NewNullTime", func(t *testing.T) {
		now := time.Now()
		nt := NewNullTime(now)
		assert.True(t, nt.Valid)
		assert.Equal(t, now, nt.Time)
	})
}

func TestNullTypes_InStruct(t *testing.T) {
	type TestModel struct {
		Name    NullString  `json:"name,omitempty"`
		Score   NullFloat64 `json:"score,omitempty"`
		Count   NullInt64   `json:"count,omitempty"`
		Active  NullBool    `json:"active,omitempty"`
		Updated NullTime    `json:"updated,omitempty"`
	}

	t.Run("valid fields serialize to primitives", func(t *testing.T) {
		m := TestModel{
			Name:  NewNullString("test"),
			Score: NewNullFloat64(8.5),
			Count: NewNullInt64(42),
		}
		data, err := json.Marshal(m)
		require.NoError(t, err)

		var raw map[string]interface{}
		err = json.Unmarshal(data, &raw)
		require.NoError(t, err)

		assert.Equal(t, "test", raw["name"])
		assert.Equal(t, 8.5, raw["score"])
		assert.Equal(t, float64(42), raw["count"])
	})

	t.Run("null fields serialize to null not objects", func(t *testing.T) {
		m := TestModel{}
		data, err := json.Marshal(m)
		require.NoError(t, err)

		var raw map[string]interface{}
		err = json.Unmarshal(data, &raw)
		require.NoError(t, err)

		// Null fields serialize as JSON null, NOT as {"String":"","Valid":false} objects
		for _, key := range []string{"name", "score", "count", "active", "updated"} {
			assert.Nil(t, raw[key], "field %s should be null", key)
		}
	})

	t.Run("scan compatibility", func(t *testing.T) {
		// Verify that Scan interface works through embedding
		ns := &NullString{}
		err := ns.Scan("scanned value")
		require.NoError(t, err)
		assert.True(t, ns.Valid)
		assert.Equal(t, "scanned value", ns.String)

		ni := &NullInt64{}
		err = ni.Scan(int64(123))
		require.NoError(t, err)
		assert.True(t, ni.Valid)
		assert.Equal(t, int64(123), ni.Int64)
	})
}

func TestNullTypes_Value(t *testing.T) {
	t.Run("NullString Value", func(t *testing.T) {
		ns := NewNullString("test")
		v, err := ns.Value()
		require.NoError(t, err)
		assert.Equal(t, "test", v)

		empty := NullString{}
		v, err = empty.Value()
		require.NoError(t, err)
		assert.Nil(t, v)
	})

	t.Run("NullInt64 Value", func(t *testing.T) {
		ni := NewNullInt64(42)
		v, err := ni.Value()
		require.NoError(t, err)
		assert.Equal(t, int64(42), v)
	})
}
