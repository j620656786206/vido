package services

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─── parseFfprobeJSON tests ────────────────────────────────────────────────

func TestParseFfprobeJSON_H265_4K_DTS_HDR10(t *testing.T) {
	input := []byte(`{
		"streams": [
			{"codec_type":"video","codec_name":"hevc","width":3840,"height":2160,"color_transfer":"smpte2084"},
			{"codec_type":"audio","codec_name":"dts","channels":6},
			{"codec_type":"subtitle","codec_name":"subrip","tags":{"language":"eng"}}
		],
		"format": {"filename":"movie.mkv","size":"12345678"}
	}`)

	info, err := parseFfprobeJSON(input)
	require.NoError(t, err)

	assert.Equal(t, "H.265", info.VideoCodec)
	assert.Equal(t, "3840x2160", info.VideoResolution)
	assert.Equal(t, "DTS", info.AudioCodec)
	assert.Equal(t, 6, info.AudioChannels)
	assert.Equal(t, "HDR10", info.HDRFormat)
	assert.Len(t, info.SubtitleTracks, 1)
	assert.Equal(t, "eng", info.SubtitleTracks[0].Language)
	assert.Equal(t, "subrip", info.SubtitleTracks[0].Format)
	assert.False(t, info.SubtitleTracks[0].External)
}

func TestParseFfprobeJSON_H264_1080p_AAC(t *testing.T) {
	input := []byte(`{
		"streams": [
			{"codec_type":"video","codec_name":"h264","width":1920,"height":1080},
			{"codec_type":"audio","codec_name":"aac","channels":2}
		],
		"format": {}
	}`)

	info, err := parseFfprobeJSON(input)
	require.NoError(t, err)

	assert.Equal(t, "H.264", info.VideoCodec)
	assert.Equal(t, "1920x1080", info.VideoResolution)
	assert.Equal(t, "AAC", info.AudioCodec)
	assert.Equal(t, 2, info.AudioChannels)
	assert.Empty(t, info.HDRFormat) // SDR
}

func TestParseFfprobeJSON_AV1_VP9(t *testing.T) {
	tests := []struct {
		codec string
		want  string
	}{
		{"av1", "AV1"},
		{"vp9", "VP9"},
		{"mpeg4", "MPEG-4"},
	}

	for _, tt := range tests {
		t.Run(tt.codec, func(t *testing.T) {
			input := []byte(`{
				"streams": [{"codec_type":"video","codec_name":"` + tt.codec + `","width":1920,"height":1080}],
				"format": {}
			}`)
			info, err := parseFfprobeJSON(input)
			require.NoError(t, err)
			assert.Equal(t, tt.want, info.VideoCodec)
		})
	}
}

func TestParseFfprobeJSON_AudioCodecs(t *testing.T) {
	tests := []struct {
		codec string
		want  string
	}{
		{"ac3", "AC-3"},
		{"eac3", "E-AC-3"},
		{"truehd", "TrueHD Atmos"},
		{"flac", "FLAC"},
		{"mp3", "MP3"},
		{"opus", "Opus"},
	}

	for _, tt := range tests {
		t.Run(tt.codec, func(t *testing.T) {
			input := []byte(`{
				"streams": [{"codec_type":"audio","codec_name":"` + tt.codec + `","channels":2}],
				"format": {}
			}`)
			info, err := parseFfprobeJSON(input)
			require.NoError(t, err)
			assert.Equal(t, tt.want, info.AudioCodec)
		})
	}
}

// ─── HDR detection tests ───────────────────────────────────────────────────

func TestParseFfprobeJSON_HDR_DolbyVision(t *testing.T) {
	input := []byte(`{
		"streams": [{
			"codec_type":"video","codec_name":"hevc","width":3840,"height":2160,
			"side_data_list":[{"side_data_type":"DOVI configuration record"},{"side_data_type":"Dolby Vision Metadata"}]
		}],
		"format": {}
	}`)

	info, err := parseFfprobeJSON(input)
	require.NoError(t, err)
	assert.Equal(t, "Dolby Vision", info.HDRFormat)
}

func TestParseFfprobeJSON_HDR_HLG(t *testing.T) {
	input := []byte(`{
		"streams": [{"codec_type":"video","codec_name":"hevc","width":3840,"height":2160,"color_transfer":"arib-std-b67"}],
		"format": {}
	}`)

	info, err := parseFfprobeJSON(input)
	require.NoError(t, err)
	assert.Equal(t, "HLG", info.HDRFormat)
}

func TestParseFfprobeJSON_SDR(t *testing.T) {
	input := []byte(`{
		"streams": [{"codec_type":"video","codec_name":"h264","width":1920,"height":1080,"color_transfer":"bt709"}],
		"format": {}
	}`)

	info, err := parseFfprobeJSON(input)
	require.NoError(t, err)
	assert.Empty(t, info.HDRFormat) // SDR — no HDR format
}

// ─── Graceful degradation test ─────────────────────────────────────────────

func TestFFprobeService_NotAvailable(t *testing.T) {
	// Create a service with a path that won't have ffprobe
	svc := &FFprobeService{
		semaphore: make(chan struct{}, 1),
		available: false,
	}

	assert.False(t, svc.IsAvailable())

	_, err := svc.Probe(t.Context(), "/some/file.mkv")
	assert.ErrorIs(t, err, ErrFFprobeNotAvailable)
}

// ─── Empty/malformed JSON ──────────────────────────────────────────────────

func TestParseFfprobeJSON_EmptyStreams(t *testing.T) {
	input := []byte(`{"streams":[],"format":{}}`)
	info, err := parseFfprobeJSON(input)
	require.NoError(t, err)
	assert.Empty(t, info.VideoCodec)
	assert.Empty(t, info.AudioCodec)
}

func TestParseFfprobeJSON_MalformedJSON(t *testing.T) {
	_, err := parseFfprobeJSON([]byte(`{invalid json}`))
	assert.Error(t, err)
}

// ─── Multiple streams (takes first) ───────────────────────────────────────

func TestParseFfprobeJSON_MultipleVideoStreams(t *testing.T) {
	input := []byte(`{
		"streams": [
			{"codec_type":"video","codec_name":"hevc","width":3840,"height":2160},
			{"codec_type":"video","codec_name":"h264","width":1920,"height":1080},
			{"codec_type":"audio","codec_name":"dts","channels":6},
			{"codec_type":"audio","codec_name":"aac","channels":2}
		],
		"format": {}
	}`)

	info, err := parseFfprobeJSON(input)
	require.NoError(t, err)

	// Should take first video and first audio stream
	assert.Equal(t, "H.265", info.VideoCodec)
	assert.Equal(t, "3840x2160", info.VideoResolution)
	assert.Equal(t, "DTS", info.AudioCodec)
	assert.Equal(t, 6, info.AudioChannels)
}

// ─── Subtitle with no language tag ─────────────────────────────────────────

func TestParseFfprobeJSON_SubtitleNoLanguage(t *testing.T) {
	input := []byte(`{
		"streams": [
			{"codec_type":"subtitle","codec_name":"ass","tags":{}}
		],
		"format": {}
	}`)

	info, err := parseFfprobeJSON(input)
	require.NoError(t, err)
	assert.Len(t, info.SubtitleTracks, 1)
	assert.Equal(t, "und", info.SubtitleTracks[0].Language)
}

// ─── Unknown codec normalization ───────────────────────────────────────────

func TestNormalizeVideoCodec_Unknown(t *testing.T) {
	assert.Equal(t, "RAWVIDEO", normalizeVideoCodec("rawvideo"))
}

func TestNormalizeAudioCodec_Unknown(t *testing.T) {
	assert.Equal(t, "WMAV2", normalizeAudioCodec("wmav2"))
}

// ─── External subtitle detection ───────────────────────────────────────────

func TestDetectExternalSubtitles_Found(t *testing.T) {
	dir := t.TempDir()
	videoPath := filepath.Join(dir, "Movie.2024.mkv")
	require.NoError(t, os.WriteFile(videoPath, []byte("video"), 0o644))

	// Create sidecar files
	require.NoError(t, os.WriteFile(filepath.Join(dir, "Movie.2024.eng.srt"), []byte("sub"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "Movie.2024.zh-Hant.ass"), []byte("sub"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "Movie.2024.srt"), []byte("sub"), 0o644))

	tracks := DetectExternalSubtitles(videoPath)
	assert.Len(t, tracks, 3)

	// Verify all are external
	for _, track := range tracks {
		assert.True(t, track.External)
	}
}

func TestDetectExternalSubtitles_NoMatches(t *testing.T) {
	dir := t.TempDir()
	videoPath := filepath.Join(dir, "Movie.mkv")
	require.NoError(t, os.WriteFile(videoPath, []byte("video"), 0o644))

	// Create subtitle for a DIFFERENT movie
	require.NoError(t, os.WriteFile(filepath.Join(dir, "Other.eng.srt"), []byte("sub"), 0o644))

	tracks := DetectExternalSubtitles(videoPath)
	assert.Empty(t, tracks)
}

func TestDetectExternalSubtitles_EmptyPath(t *testing.T) {
	tracks := DetectExternalSubtitles("")
	assert.Nil(t, tracks)
}

func TestDetectExternalSubtitles_LanguageExtraction(t *testing.T) {
	dir := t.TempDir()
	videoPath := filepath.Join(dir, "Film.mkv")
	require.NoError(t, os.WriteFile(videoPath, []byte("v"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "Film.jpn.srt"), []byte("s"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "Film.srt"), []byte("s"), 0o644))

	tracks := DetectExternalSubtitles(videoPath)
	require.Len(t, tracks, 2)

	// Find by language
	langs := map[string]bool{}
	for _, t := range tracks {
		langs[t.Language] = true
	}
	assert.True(t, langs["jpn"])
	assert.True(t, langs["und"]) // No language tag → "und"
}

// ─── MergeSubtitleTracks ───────────────────────────────────────────────────

func TestMergeSubtitleTracks(t *testing.T) {
	embedded := []SubtitleTrack{
		{Language: "eng", Format: "subrip", External: false},
	}
	external := []SubtitleTrack{
		{Language: "zh-Hant", Format: "srt", External: true},
	}

	merged := MergeSubtitleTracks(embedded, external)
	assert.Len(t, merged, 2)
	assert.False(t, merged[0].External)
	assert.True(t, merged[1].External)
}

func TestMergeSubtitleTracks_Empty(t *testing.T) {
	merged := MergeSubtitleTracks(nil, nil)
	assert.Empty(t, merged)
}
