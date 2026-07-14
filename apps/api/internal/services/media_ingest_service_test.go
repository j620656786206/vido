package services

import (
	"testing"
)

func TestSeriesDirFor(t *testing.T) {
	tests := []struct {
		name     string
		filePath string
		scanRoot string
		want     string
	}{
		{
			name:     "climbs past a Season folder",
			filePath: "/media/tv/鵲刀門傳奇/Season02/Legend.S02E05.2160p.mkv",
			scanRoot: "/media/tv",
			want:     "/media/tv/鵲刀門傳奇",
		},
		{
			name:     "climbs past a bare S02 folder",
			filePath: "/media/tv/Show/S02/Show.S02E01.mkv",
			scanRoot: "/media/tv",
			want:     "/media/tv/Show",
		},
		{
			name:     "climbs past a Chinese season folder",
			filePath: "/media/tv/某劇/第二季/某劇.S02E01.mkv",
			scanRoot: "/media/tv",
			want:     "/media/tv/某劇",
		},
		{
			name:     "climbs past Specials",
			filePath: "/media/tv/Show/Specials/Show.S00E01.mkv",
			scanRoot: "/media/tv",
			want:     "/media/tv/Show",
		},
		{
			name:     "series folder with no season subfolder",
			filePath: "/media/tv/某劇/某劇.S01E01.mkv",
			scanRoot: "/media/tv",
			want:     "/media/tv/某劇",
		},
		{
			// Without the scanRoot guard, climbing here would land on the library root and
			// every series in the library would collapse into a single row.
			name:     "flat layout: never climbs onto the scan root itself",
			filePath: "/media/tv/Season01/Show.S01E01.mkv",
			scanRoot: "/media/tv",
			want:     "/media/tv/Season01",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SeriesDirFor(tt.filePath, tt.scanRoot); got != tt.want {
				t.Errorf("SeriesDirFor(%q, %q) = %q, want %q", tt.filePath, tt.scanRoot, got, tt.want)
			}
		})
	}
}

func TestTitleFromSeriesDir(t *testing.T) {
	if got := TitleFromSeriesDir("/media/tv/鵲刀門傳奇"); got != "鵲刀門傳奇" {
		t.Errorf("TitleFromSeriesDir = %q, want 鵲刀門傳奇", got)
	}
}
