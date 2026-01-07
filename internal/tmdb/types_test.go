package tmdb

import (
	"encoding/json"
	"testing"
	"time"
)

func TestDate_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    time.Time
		wantErr bool
	}{
		{
			name:    "valid date",
			input:   `"2023-12-25"`,
			want:    time.Date(2023, 12, 25, 0, 0, 0, 0, time.UTC),
			wantErr: false,
		},
		{
			name:    "empty string",
			input:   `""`,
			want:    time.Time{},
			wantErr: false,
		},
		{
			name:    "null value",
			input:   `null`,
			want:    time.Time{},
			wantErr: false,
		},
		{
			name:    "invalid format",
			input:   `"25-12-2023"`,
			want:    time.Time{},
			wantErr: true,
		},
		{
			name:    "invalid date",
			input:   `"not a date"`,
			want:    time.Time{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var d Date
			err := d.UnmarshalJSON([]byte(tt.input))

			if (err != nil) != tt.wantErr {
				t.Errorf("Date.UnmarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && !d.Time.Equal(tt.want) {
				t.Errorf("Date.UnmarshalJSON() = %v, want %v", d.Time, tt.want)
			}
		})
	}
}

func TestDate_MarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		date    Date
		want    string
		wantErr bool
	}{
		{
			name:    "valid date",
			date:    Date{Time: time.Date(2023, 12, 25, 0, 0, 0, 0, time.UTC)},
			want:    `"2023-12-25"`,
			wantErr: false,
		},
		{
			name:    "zero date",
			date:    Date{},
			want:    `null`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.date.MarshalJSON()

			if (err != nil) != tt.wantErr {
				t.Errorf("Date.MarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && string(got) != tt.want {
				t.Errorf("Date.MarshalJSON() = %v, want %v", string(got), tt.want)
			}
		})
	}
}

func TestDate_JSONRoundTrip(t *testing.T) {
	type TestStruct struct {
		ReleaseDate Date `json:"release_date"`
	}

	tests := []struct {
		name string
		json string
		want time.Time
	}{
		{
			name: "movie with release date",
			json: `{"release_date":"1999-10-15"}`,
			want: time.Date(1999, 10, 15, 0, 0, 0, 0, time.UTC),
		},
		{
			name: "movie with null release date",
			json: `{"release_date":null}`,
			want: time.Time{},
		},
		{
			name: "movie with empty release date",
			json: `{"release_date":""}`,
			want: time.Time{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var s TestStruct
			if err := json.Unmarshal([]byte(tt.json), &s); err != nil {
				t.Fatalf("json.Unmarshal() error = %v", err)
			}

			if !s.ReleaseDate.Time.Equal(tt.want) {
				t.Errorf("ReleaseDate = %v, want %v", s.ReleaseDate.Time, tt.want)
			}

			// Test marshal back
			data, err := json.Marshal(s)
			if err != nil {
				t.Fatalf("json.Marshal() error = %v", err)
			}

			// Unmarshal again to verify roundtrip
			var s2 TestStruct
			if err := json.Unmarshal(data, &s2); err != nil {
				t.Fatalf("json.Unmarshal() roundtrip error = %v", err)
			}

			if !s2.ReleaseDate.Time.Equal(tt.want) {
				t.Errorf("Roundtrip ReleaseDate = %v, want %v", s2.ReleaseDate.Time, tt.want)
			}
		})
	}
}
