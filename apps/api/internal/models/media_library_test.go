package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMediaLibrary_Validate(t *testing.T) {
	tests := []struct {
		name    string
		lib     MediaLibrary
		wantErr bool
		field   string
	}{
		{"valid movie", MediaLibrary{Name: "Movies", ContentType: ContentTypeMovie}, false, ""},
		{"valid series", MediaLibrary{Name: "TV", ContentType: ContentTypeSeries}, false, ""},
		{"empty name", MediaLibrary{Name: "", ContentType: ContentTypeMovie}, true, "name"},
		{"invalid content type", MediaLibrary{Name: "X", ContentType: "anime"}, true, "content_type"},
		{"empty content type", MediaLibrary{Name: "X", ContentType: ""}, true, "content_type"},
		{"name too long", MediaLibrary{Name: string(make([]byte, 256)), ContentType: ContentTypeMovie}, true, "name"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.lib.Validate()
			if tt.wantErr {
				require.Error(t, err)
				var ve *ValidationError
				require.ErrorAs(t, err, &ve)
				assert.Equal(t, tt.field, ve.Field)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestMediaLibraryPath_Validate(t *testing.T) {
	tests := []struct {
		name    string
		path    MediaLibraryPath
		wantErr bool
		field   string
	}{
		{"valid", MediaLibraryPath{LibraryID: "abc", Path: "/media"}, false, ""},
		{"empty library_id", MediaLibraryPath{LibraryID: "", Path: "/media"}, true, "library_id"},
		{"empty path", MediaLibraryPath{LibraryID: "abc", Path: ""}, true, "path"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.path.Validate()
			if tt.wantErr {
				require.Error(t, err)
				var ve *ValidationError
				require.ErrorAs(t, err, &ve)
				assert.Equal(t, tt.field, ve.Field)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
