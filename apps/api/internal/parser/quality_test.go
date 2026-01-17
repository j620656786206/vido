package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDetectQuality(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		want     string
	}{
		{"2160p", "Movie.2020.2160p.BluRay.mkv", "2160p"},
		{"4K", "Movie.2020.4K.UHD.mkv", "2160p"},
		{"UHD", "Movie.2020.UHD.BluRay.mkv", "2160p"},
		{"1080p", "Movie.2020.1080p.BluRay.mkv", "1080p"},
		{"1080i", "Movie.2020.1080i.HDTV.mkv", "1080p"},
		{"720p", "Movie.2020.720p.WEB-DL.mkv", "720p"},
		{"576p", "Movie.2020.576p.DVDRip.mkv", "576p"},
		{"480p", "Movie.2020.480p.DVDRip.mkv", "480p"},
		{"SD", "Movie.2020.SD.DVDRip.mkv", "480p"},
		{"no quality", "Movie.2020.BluRay.mkv", ""},
		{"lowercase", "movie.2020.1080p.bluray.mkv", "1080p"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectQuality(tt.filename)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestDetectSource(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		want     string
	}{
		{"BluRay", "Movie.2020.1080p.BluRay.mkv", "BluRay"},
		{"Blu-Ray hyphenated", "Movie.2020.1080p.Blu-Ray.mkv", "BluRay"},
		{"BDRip", "Movie.2020.1080p.BDRip.mkv", "BluRay"},
		{"BRRip", "Movie.2020.1080p.BRRip.mkv", "BluRay"},
		{"WEB-DL", "Movie.2020.1080p.WEB-DL.mkv", "WEB-DL"},
		{"WEBDL", "Movie.2020.1080p.WEBDL.mkv", "WEB-DL"},
		{"WEBRip", "Movie.2020.1080p.WEBRip.mkv", "WEBRip"},
		{"HDTV", "Movie.2020.720p.HDTV.mkv", "HDTV"},
		{"PDTV", "Movie.2020.720p.PDTV.mkv", "PDTV"},
		{"DVDRip", "Movie.2020.480p.DVDRip.mkv", "DVDRip"},
		{"DVD", "Movie.2020.480p.DVD.mkv", "DVDRip"},
		{"HDCAM", "Movie.2020.HDCAM.mkv", "HDCAM"},
		{"CAM", "Movie.2020.CAM.mkv", "CAM"},
		{"TS", "Movie.2020.TS.mkv", "TS"},
		{"Telesync", "Movie.2020.Telesync.mkv", "TS"},
		{"Screener", "Movie.2020.Screener.mkv", "SCR"},
		{"SCR", "Movie.2020.SCR.mkv", "SCR"},
		{"R5", "Movie.2020.R5.mkv", "R5"},
		{"no source", "Movie.2020.1080p.x264.mkv", ""},
		{"lowercase", "movie.2020.1080p.bluray.mkv", "BluRay"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectSource(tt.filename)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestDetectVideoCodec(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		want     string
	}{
		{"x264", "Movie.2020.1080p.BluRay.x264.mkv", "x264"},
		{"H264", "Movie.2020.1080p.BluRay.H264.mkv", "x264"},
		{"H.264", "Movie.2020.1080p.BluRay.H.264.mkv", "x264"},
		{"AVC", "Movie.2020.1080p.BluRay.AVC.mkv", "x264"},
		{"x265", "Movie.2020.1080p.BluRay.x265.mkv", "x265"},
		{"H265", "Movie.2020.1080p.BluRay.H265.mkv", "x265"},
		{"H.265", "Movie.2020.1080p.BluRay.H.265.mkv", "x265"},
		{"HEVC", "Movie.2020.1080p.BluRay.HEVC.mkv", "x265"},
		{"AV1", "Movie.2020.1080p.BluRay.AV1.mkv", "AV1"},
		{"XviD", "Movie.2020.480p.DVDRip.XviD.mkv", "XviD"},
		{"DivX", "Movie.2020.480p.DVDRip.DivX.mkv", "DivX"},
		{"no codec", "Movie.2020.1080p.BluRay.mkv", ""},
		{"lowercase", "movie.2020.1080p.bluray.hevc.mkv", "x265"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectVideoCodec(tt.filename)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestDetectAudioCodec(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		want     string
	}{
		{"AAC", "Movie.2020.1080p.BluRay.AAC.mkv", "AAC"},
		{"AC3", "Movie.2020.1080p.BluRay.AC3.mkv", "AC3"},
		{"DTS", "Movie.2020.1080p.BluRay.DTS.mkv", "DTS"},
		{"DTS-HD", "Movie.2020.1080p.BluRay.DTS-HD.mkv", "DTS"},
		{"TrueHD", "Movie.2020.1080p.BluRay.TrueHD.mkv", "TrueHD"},
		{"Atmos", "Movie.2020.1080p.BluRay.Atmos.mkv", "Atmos"},
		{"FLAC", "Movie.2020.1080p.BluRay.FLAC.mkv", "FLAC"},
		{"MP3", "Movie.2020.720p.WEBRip.MP3.mkv", "MP3"},
		{"EAC3", "Movie.2020.1080p.WEB-DL.EAC3.mkv", "EAC3"},
		{"DD5.1", "Movie.2020.1080p.BluRay.DD5.1.mkv", "EAC3"},
		{"no audio codec", "Movie.2020.1080p.BluRay.mkv", ""},
		{"lowercase", "movie.2020.1080p.bluray.dts.mkv", "DTS"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectAudioCodec(tt.filename)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestDetectReleaseGroup(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		want     string
	}{
		{"standard group", "Movie.2020.1080p.BluRay.x264-SPARKS.mkv", "SPARKS"},
		{"YTS group", "Movie.2020.1080p.BluRay.x265-YTS.mkv", "YTS"},
		{"RARBG group", "Movie.2020.1080p.BluRay.x264-RARBG.mkv", "RARBG"},
		{"no group", "Movie.2020.1080p.BluRay.mkv", ""},
		{"group with numbers", "Movie.2020.1080p.BluRay.x264-GROUP123.mkv", "GROUP123"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectReleaseGroup(tt.filename)
			assert.Equal(t, tt.want, got)
		})
	}
}
