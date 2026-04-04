package services

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─── Parse: XML movie format ───────────────────────────────────────────────

func TestNFOReaderService_Parse_XMLMovie(t *testing.T) {
	dir := t.TempDir()
	nfoPath := filepath.Join(dir, "movie.nfo")
	content := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<movie>
  <title>功夫</title>
  <originaltitle>Kung Fu Hustle</originaltitle>
  <year>2004</year>
  <plot>A plot about kung fu.</plot>
  <uniqueid type="tmdb" default="true">10196</uniqueid>
  <uniqueid type="imdb">tt0373074</uniqueid>
  <fileinfo>
    <streamdetails>
      <video><codec>h265</codec><width>3840</width><height>2160</height></video>
      <audio><codec>dts</codec><channels>6</channels></audio>
      <subtitle><language>chi</language></subtitle>
      <subtitle><language>eng</language></subtitle>
    </streamdetails>
  </fileinfo>
</movie>`
	require.NoError(t, os.WriteFile(nfoPath, []byte(content), 0o644))

	svc := NewNFOReaderService(nil)
	data, err := svc.Parse(nfoPath)
	require.NoError(t, err)

	assert.Equal(t, "10196", data.TMDbID)
	assert.Equal(t, "tt0373074", data.IMDbID)
	assert.Equal(t, "功夫", data.Title)
	assert.Equal(t, "Kung Fu Hustle", data.OriginalTitle)
	assert.Equal(t, "2004", data.Year)
	assert.Equal(t, "A plot about kung fu.", data.Plot)
	assert.Equal(t, NFOSourceFormatXML, data.SourceFormat)
	assert.Equal(t, "movie", data.MediaType)

	// Streamdetails
	assert.Equal(t, "h265", data.VideoCodec)
	assert.Equal(t, "4K", data.VideoResolution)
	assert.Equal(t, "dts", data.AudioCodec)
	assert.Equal(t, 6, data.AudioChannels)
	assert.Len(t, data.Subtitles, 2)
	assert.Equal(t, "chi", data.Subtitles[0].Language)
	assert.Equal(t, "eng", data.Subtitles[1].Language)
}

// ─── Parse: XML tvshow format ──────────────────────────────────────────────

func TestNFOReaderService_Parse_XMLTVShow(t *testing.T) {
	dir := t.TempDir()
	nfoPath := filepath.Join(dir, "tvshow.nfo")
	content := `<tvshow>
  <title>進擊的巨人</title>
  <originaltitle>進撃の巨人</originaltitle>
  <year>2013</year>
  <plot>Humanity fights titans.</plot>
  <uniqueid type="tmdb">1429</uniqueid>
</tvshow>`
	require.NoError(t, os.WriteFile(nfoPath, []byte(content), 0o644))

	svc := NewNFOReaderService(nil)
	data, err := svc.Parse(nfoPath)
	require.NoError(t, err)

	assert.Equal(t, "1429", data.TMDbID)
	assert.Empty(t, data.IMDbID)
	assert.Equal(t, "tvshow", data.MediaType)
	assert.Equal(t, "進擊的巨人", data.Title)
	assert.Equal(t, NFOSourceFormatXML, data.SourceFormat)
}

// ─── Parse: XML episodedetails format ──────────────────────────────────────

func TestNFOReaderService_Parse_XMLEpisode(t *testing.T) {
	dir := t.TempDir()
	nfoPath := filepath.Join(dir, "episode.nfo")
	content := `<episodedetails>
  <title>Episode 1</title>
  <plot>Pilot episode.</plot>
  <uniqueid type="tmdb">99999</uniqueid>
  <fileinfo>
    <streamdetails>
      <video><codec>h264</codec><width>1920</width><height>1080</height></video>
      <audio><codec>aac</codec><channels>2</channels></audio>
    </streamdetails>
  </fileinfo>
</episodedetails>`
	require.NoError(t, os.WriteFile(nfoPath, []byte(content), 0o644))

	svc := NewNFOReaderService(nil)
	data, err := svc.Parse(nfoPath)
	require.NoError(t, err)

	assert.Equal(t, "episodedetails", data.MediaType)
	assert.Equal(t, "99999", data.TMDbID)
	assert.Equal(t, "h264", data.VideoCodec)
	assert.Equal(t, "1080p", data.VideoResolution)
	assert.Equal(t, "aac", data.AudioCodec)
	assert.Equal(t, 2, data.AudioChannels)
}

// ─── Parse: URL format (TMDB) ──────────────────────────────────────────────

func TestNFOReaderService_Parse_URLFormatTMDb(t *testing.T) {
	dir := t.TempDir()
	nfoPath := filepath.Join(dir, "movie.nfo")
	content := "https://www.themoviedb.org/movie/12345\n"
	require.NoError(t, os.WriteFile(nfoPath, []byte(content), 0o644))

	svc := NewNFOReaderService(nil)
	data, err := svc.Parse(nfoPath)
	require.NoError(t, err)

	assert.Equal(t, "12345", data.TMDbID)
	assert.Empty(t, data.IMDbID)
	assert.Equal(t, NFOSourceFormatURL, data.SourceFormat)
}

// ─── Parse: URL format (IMDB) ──────────────────────────────────────────────

func TestNFOReaderService_Parse_URLFormatIMDb(t *testing.T) {
	dir := t.TempDir()
	nfoPath := filepath.Join(dir, "movie.nfo")
	content := "https://www.imdb.com/title/tt1234567/\n"
	require.NoError(t, os.WriteFile(nfoPath, []byte(content), 0o644))

	svc := NewNFOReaderService(nil)
	data, err := svc.Parse(nfoPath)
	require.NoError(t, err)

	assert.Empty(t, data.TMDbID)
	assert.Equal(t, "tt1234567", data.IMDbID)
	assert.Equal(t, NFOSourceFormatURL, data.SourceFormat)
}

// ─── Parse: URL format (TMDB TV) ──────────────────────────────────────────

func TestNFOReaderService_Parse_URLFormatTMDbTV(t *testing.T) {
	dir := t.TempDir()
	nfoPath := filepath.Join(dir, "show.nfo")
	content := "https://www.themoviedb.org/tv/1429\n"
	require.NoError(t, os.WriteFile(nfoPath, []byte(content), 0o644))

	svc := NewNFOReaderService(nil)
	data, err := svc.Parse(nfoPath)
	require.NoError(t, err)

	assert.Equal(t, "1429", data.TMDbID)
	assert.Equal(t, NFOSourceFormatURL, data.SourceFormat)
}

// ─── Parse: Malformed XML ──────────────────────────────────────────────────

func TestNFOReaderService_Parse_MalformedXML(t *testing.T) {
	dir := t.TempDir()
	nfoPath := filepath.Join(dir, "bad.nfo")
	content := `<notamovie><broken>stuff</notamovie>`
	require.NoError(t, os.WriteFile(nfoPath, []byte(content), 0o644))

	svc := NewNFOReaderService(nil)
	_, err := svc.Parse(nfoPath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unrecognized XML root element")
}

// ─── Parse: Empty file ─────────────────────────────────────────────────────

func TestNFOReaderService_Parse_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	nfoPath := filepath.Join(dir, "empty.nfo")
	require.NoError(t, os.WriteFile(nfoPath, []byte(""), 0o644))

	svc := NewNFOReaderService(nil)
	_, err := svc.Parse(nfoPath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "empty nfo file")
}

// ─── Parse: Unrecognized format ────────────────────────────────────────────

func TestNFOReaderService_Parse_UnrecognizedFormat(t *testing.T) {
	dir := t.TempDir()
	nfoPath := filepath.Join(dir, "random.nfo")
	content := "This is just some random text\nwith no URLs"
	require.NoError(t, os.WriteFile(nfoPath, []byte(content), 0o644))

	svc := NewNFOReaderService(nil)
	_, err := svc.Parse(nfoPath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no TMDB/IMDB URL found")
}

// ─── Parse: File not found ─────────────────────────────────────────────────

func TestNFOReaderService_Parse_FileNotFound(t *testing.T) {
	svc := NewNFOReaderService(nil)
	_, err := svc.Parse("/nonexistent/path.nfo")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "read nfo")
}

// ─── Parse: Streamdetails extraction ───────────────────────────────────────

func TestNFOReaderService_Parse_StreamDetails_720p(t *testing.T) {
	dir := t.TempDir()
	nfoPath := filepath.Join(dir, "movie.nfo")
	content := `<movie>
  <title>Test</title>
  <fileinfo>
    <streamdetails>
      <video><codec>h264</codec><width>1280</width><height>720</height></video>
      <audio><codec>ac3</codec><channels>2</channels></audio>
    </streamdetails>
  </fileinfo>
</movie>`
	require.NoError(t, os.WriteFile(nfoPath, []byte(content), 0o644))

	svc := NewNFOReaderService(nil)
	data, err := svc.Parse(nfoPath)
	require.NoError(t, err)

	assert.Equal(t, "h264", data.VideoCodec)
	assert.Equal(t, "720p", data.VideoResolution)
	assert.Equal(t, "ac3", data.AudioCodec)
	assert.Equal(t, 2, data.AudioChannels)
}

func TestNFOReaderService_Parse_StreamDetails_1440p(t *testing.T) {
	dir := t.TempDir()
	nfoPath := filepath.Join(dir, "movie.nfo")
	content := `<movie>
  <title>Test</title>
  <fileinfo>
    <streamdetails>
      <video><codec>hevc</codec><width>2560</width><height>1440</height></video>
    </streamdetails>
  </fileinfo>
</movie>`
	require.NoError(t, os.WriteFile(nfoPath, []byte(content), 0o644))

	svc := NewNFOReaderService(nil)
	data, err := svc.Parse(nfoPath)
	require.NoError(t, err)

	assert.Equal(t, "1440p", data.VideoResolution)
}

func TestNFOReaderService_Parse_NoStreamDetails(t *testing.T) {
	dir := t.TempDir()
	nfoPath := filepath.Join(dir, "movie.nfo")
	content := `<movie><title>Minimal</title></movie>`
	require.NoError(t, os.WriteFile(nfoPath, []byte(content), 0o644))

	svc := NewNFOReaderService(nil)
	data, err := svc.Parse(nfoPath)
	require.NoError(t, err)

	assert.Empty(t, data.VideoCodec)
	assert.Empty(t, data.VideoResolution)
	assert.Empty(t, data.AudioCodec)
	assert.Equal(t, 0, data.AudioChannels)
}

// ─── FindNFOSidecar ────────────────────────────────────────────────────────

func TestNFOReaderService_FindNFOSidecar_Exists(t *testing.T) {
	dir := t.TempDir()
	videoPath := filepath.Join(dir, "Movie.2024.mkv")
	nfoPath := filepath.Join(dir, "Movie.2024.nfo")

	require.NoError(t, os.WriteFile(videoPath, []byte("video"), 0o644))
	require.NoError(t, os.WriteFile(nfoPath, []byte("nfo"), 0o644))

	svc := NewNFOReaderService(nil)
	result := svc.FindNFOSidecar(videoPath)
	assert.Equal(t, nfoPath, result)
}

func TestNFOReaderService_FindNFOSidecar_NotExists(t *testing.T) {
	dir := t.TempDir()
	videoPath := filepath.Join(dir, "Movie.2024.mkv")
	require.NoError(t, os.WriteFile(videoPath, []byte("video"), 0o644))

	svc := NewNFOReaderService(nil)
	result := svc.FindNFOSidecar(videoPath)
	assert.Empty(t, result)
}

func TestNFOReaderService_FindNFOSidecar_EmptyPath(t *testing.T) {
	svc := NewNFOReaderService(nil)
	result := svc.FindNFOSidecar("")
	assert.Empty(t, result)
}

func TestNFOReaderService_FindNFOSidecar_VariousExtensions(t *testing.T) {
	dir := t.TempDir()

	tests := []struct {
		videoExt string
	}{
		{".mkv"},
		{".mp4"},
		{".avi"},
		{".wmv"},
	}

	for _, tt := range tests {
		t.Run(tt.videoExt, func(t *testing.T) {
			base := filepath.Join(dir, "test"+tt.videoExt)
			nfo := filepath.Join(dir, "test.nfo")

			// Create NFO file
			require.NoError(t, os.WriteFile(nfo, []byte("nfo"), 0o644))

			svc := NewNFOReaderService(nil)
			result := svc.FindNFOSidecar(base)
			assert.Equal(t, nfo, result)

			// Cleanup for next iteration
			os.Remove(nfo)
		})
	}
}

// ─── extractTMDbID / extractIMDbID ─────────────────────────────────────────

func TestExtractTMDbID(t *testing.T) {
	tests := []struct {
		input string
		want  string
		ok    bool
	}{
		{"https://www.themoviedb.org/movie/12345", "12345", true},
		{"https://themoviedb.org/movie/99999", "99999", true},
		{"https://www.themoviedb.org/tv/1429", "1429", true},
		{"http://themoviedb.org/movie/42", "42", true},
		{"not a url", "", false},
		{"https://imdb.com/title/tt1234567", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, ok := extractTMDbID(tt.input)
			assert.Equal(t, tt.ok, ok)
			if ok {
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestExtractIMDbID(t *testing.T) {
	tests := []struct {
		input string
		want  string
		ok    bool
	}{
		{"https://www.imdb.com/title/tt1234567/", "tt1234567", true},
		{"https://imdb.com/title/tt0000001", "tt0000001", true},
		{"not a url", "", false},
		{"https://www.themoviedb.org/movie/12345", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, ok := extractIMDbID(tt.input)
			assert.Equal(t, tt.ok, ok)
			if ok {
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

// ─── resolveResolution ─────────────────────────────────────────────────────

func TestResolveResolution(t *testing.T) {
	tests := []struct {
		width, height int
		want          string
	}{
		{3840, 2160, "4K"},
		{1920, 1080, "1080p"},
		{1280, 720, "720p"},
		{720, 480, "480p"},
		{640, 360, "360p"},
		{2560, 1440, "1440p"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := resolveResolution(tt.width, tt.height)
			assert.Equal(t, tt.want, got)
		})
	}
}

// ─── XML with only IMDB ID (no TMDB) ──────────────────────────────────────

func TestNFOReaderService_Parse_XMLMovie_IMDbOnly(t *testing.T) {
	dir := t.TempDir()
	nfoPath := filepath.Join(dir, "movie.nfo")
	content := `<movie>
  <title>Test Movie</title>
  <uniqueid type="imdb">tt9876543</uniqueid>
</movie>`
	require.NoError(t, os.WriteFile(nfoPath, []byte(content), 0o644))

	svc := NewNFOReaderService(nil)
	data, err := svc.Parse(nfoPath)
	require.NoError(t, err)

	assert.Empty(t, data.TMDbID)
	assert.Equal(t, "tt9876543", data.IMDbID)
	assert.Equal(t, "movie", data.MediaType)
}

// ─── XML with whitespace in uniqueid values ───────���────────────────────────

func TestNFOReaderService_Parse_XMLMovie_WhitespaceUniqueID(t *testing.T) {
	dir := t.TempDir()
	nfoPath := filepath.Join(dir, "movie.nfo")
	content := `<movie>
  <title>Whitespace Test</title>
  <uniqueid type="tmdb">  12345  </uniqueid>
  <uniqueid type="imdb">  tt0000001  </uniqueid>
</movie>`
	require.NoError(t, os.WriteFile(nfoPath, []byte(content), 0o644))

	svc := NewNFOReaderService(nil)
	data, err := svc.Parse(nfoPath)
	require.NoError(t, err)

	assert.Equal(t, "12345", data.TMDbID)     // Trimmed
	assert.Equal(t, "tt0000001", data.IMDbID)  // Trimmed
}

// ──�� XML with both TMDB and IMDB — both extracted ─────────────────────────

func TestNFOReaderService_Parse_XMLMovie_BothIDs(t *testing.T) {
	dir := t.TempDir()
	nfoPath := filepath.Join(dir, "movie.nfo")
	content := `<movie>
  <title>Both IDs</title>
  <uniqueid type="tmdb">42</uniqueid>
  <uniqueid type="imdb">tt0042</uniqueid>
</movie>`
	require.NoError(t, os.WriteFile(nfoPath, []byte(content), 0o644))

	svc := NewNFOReaderService(nil)
	data, err := svc.Parse(nfoPath)
	require.NoError(t, err)

	// Both IDs should be extracted — enrichment service decides precedence
	assert.Equal(t, "42", data.TMDbID)
	assert.Equal(t, "tt0042", data.IMDbID)
}

// ─── XML with case-insensitive uniqueid type ──────────────────────────────

func TestNFOReaderService_Parse_XMLMovie_CaseInsensitiveType(t *testing.T) {
	dir := t.TempDir()
	nfoPath := filepath.Join(dir, "movie.nfo")
	content := `<movie>
  <title>Case Test</title>
  <uniqueid type="TMDB">777</uniqueid>
  <uniqueid type="IMDB">tt0000777</uniqueid>
</movie>`
	require.NoError(t, os.WriteFile(nfoPath, []byte(content), 0o644))

	svc := NewNFOReaderService(nil)
	data, err := svc.Parse(nfoPath)
	require.NoError(t, err)

	assert.Equal(t, "777", data.TMDbID)
	assert.Equal(t, "tt0000777", data.IMDbID)
}

// ─── 480p resolution boundary ─────────────────────────────────────────────

func TestNFOReaderService_Parse_StreamDetails_480p(t *testing.T) {
	dir := t.TempDir()
	nfoPath := filepath.Join(dir, "movie.nfo")
	content := `<movie>
  <title>SD</title>
  <fileinfo>
    <streamdetails>
      <video><codec>mpeg4</codec><width>720</width><height>480</height></video>
    </streamdetails>
  </fileinfo>
</movie>`
	require.NoError(t, os.WriteFile(nfoPath, []byte(content), 0o644))

	svc := NewNFOReaderService(nil)
	data, err := svc.Parse(nfoPath)
	require.NoError(t, err)

	assert.Equal(t, "480p", data.VideoResolution)
}

// ─── XML with no uniqueid at all ──────────────────────────────────────────

func TestNFOReaderService_Parse_XMLMovie_NoUniqueID(t *testing.T) {
	dir := t.TempDir()
	nfoPath := filepath.Join(dir, "movie.nfo")
	content := `<movie>
  <title>No IDs Movie</title>
  <year>2020</year>
</movie>`
	require.NoError(t, os.WriteFile(nfoPath, []byte(content), 0o644))

	svc := NewNFOReaderService(nil)
	data, err := svc.Parse(nfoPath)
	require.NoError(t, err)

	assert.Empty(t, data.TMDbID)
	assert.Empty(t, data.IMDbID)
	assert.Equal(t, "No IDs Movie", data.Title)
	assert.Equal(t, "2020", data.Year)
}

// ─── URL format with blank lines ──────────────────────────────────────────

func TestNFOReaderService_Parse_URLFormat_BlankLines(t *testing.T) {
	dir := t.TempDir()
	nfoPath := filepath.Join(dir, "movie.nfo")
	content := "\n\n  https://www.themoviedb.org/movie/54321  \n\n"
	require.NoError(t, os.WriteFile(nfoPath, []byte(content), 0o644))

	svc := NewNFOReaderService(nil)
	data, err := svc.Parse(nfoPath)
	require.NoError(t, err)

	assert.Equal(t, "54321", data.TMDbID)
	assert.Equal(t, NFOSourceFormatURL, data.SourceFormat)
}
