package services

import (
	"encoding/xml"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// NFOSourceFormat indicates how the NFO file was formatted
type NFOSourceFormat string

const (
	NFOSourceFormatXML NFOSourceFormat = "xml"
	NFOSourceFormatURL NFOSourceFormat = "url"
)

// NFOData holds the parsed data from an NFO sidecar file
type NFOData struct {
	TMDbID          string
	IMDbID          string
	Title           string
	OriginalTitle   string
	Year            string
	Plot            string
	SourceFormat    NFOSourceFormat
	MediaType       string // "movie", "tvshow", "episodedetails"
	VideoCodec      string
	VideoResolution string
	AudioCodec      string
	AudioChannels   int
	Subtitles       []NFOSubtitleTrack
}

// NFOSubtitleTrack represents a subtitle track in NFO streamdetails
type NFOSubtitleTrack struct {
	Language string
}

// nfoXMLMovie is the XML struct for parsing Kodi <movie> NFO files
type nfoXMLMovie struct {
	XMLName       xml.Name         `xml:"movie"`
	Title         string           `xml:"title"`
	OriginalTitle string           `xml:"originaltitle"`
	Year          string           `xml:"year"`
	Plot          string           `xml:"plot"`
	UniqueIDs     []nfoXMLUniqueID `xml:"uniqueid"`
	FileInfo      *nfoXMLFileInfo  `xml:"fileinfo"`
}

// nfoXMLTVShow is the XML struct for parsing Kodi <tvshow> NFO files
type nfoXMLTVShow struct {
	XMLName       xml.Name         `xml:"tvshow"`
	Title         string           `xml:"title"`
	OriginalTitle string           `xml:"originaltitle"`
	Year          string           `xml:"year"`
	Plot          string           `xml:"plot"`
	UniqueIDs     []nfoXMLUniqueID `xml:"uniqueid"`
}

// nfoXMLEpisode is the XML struct for parsing Kodi <episodedetails> NFO files
type nfoXMLEpisode struct {
	XMLName   xml.Name         `xml:"episodedetails"`
	Title     string           `xml:"title"`
	Plot      string           `xml:"plot"`
	UniqueIDs []nfoXMLUniqueID `xml:"uniqueid"`
	FileInfo  *nfoXMLFileInfo  `xml:"fileinfo"`
}

// nfoXMLUniqueID represents a <uniqueid> element
type nfoXMLUniqueID struct {
	Type    string `xml:"type,attr"`
	Default bool   `xml:"default,attr"`
	Value   string `xml:",chardata"`
}

// nfoXMLFileInfo wraps <fileinfo><streamdetails>
type nfoXMLFileInfo struct {
	StreamDetails nfoXMLStreamDetails `xml:"streamdetails"`
}

// nfoXMLStreamDetails contains video/audio/subtitle stream info
type nfoXMLStreamDetails struct {
	Video    *nfoXMLVideo      `xml:"video"`
	Audio    *nfoXMLAudio      `xml:"audio"`
	Subtitle []nfoXMLSubtitle  `xml:"subtitle"`
}

// nfoXMLVideo contains video stream properties
type nfoXMLVideo struct {
	Codec  string `xml:"codec"`
	Width  int    `xml:"width"`
	Height int    `xml:"height"`
}

// nfoXMLAudio contains audio stream properties
type nfoXMLAudio struct {
	Codec    string `xml:"codec"`
	Channels int    `xml:"channels"`
}

// nfoXMLSubtitle contains subtitle stream language
type nfoXMLSubtitle struct {
	Language string `xml:"language"`
}

// NFOReaderService reads and parses Kodi-compatible .nfo sidecar files
type NFOReaderService struct {
	logger *slog.Logger
}

// NewNFOReaderService creates a new NFOReaderService
func NewNFOReaderService(logger *slog.Logger) *NFOReaderService {
	if logger == nil {
		logger = slog.Default()
	}
	return &NFOReaderService{
		logger: logger.With("service", "nfo_reader"),
	}
}

// regex patterns for URL-format NFO files
var (
	tmdbURLPattern = regexp.MustCompile(`themoviedb\.org/(?:movie|tv)/(\d+)`)
	imdbURLPattern = regexp.MustCompile(`imdb\.com/title/(tt\d+)`)
)

// FindNFOSidecar checks if a .nfo sidecar file exists for the given video path.
// Returns the NFO path if found, empty string otherwise.
func (s *NFOReaderService) FindNFOSidecar(videoPath string) string {
	if videoPath == "" {
		return ""
	}
	ext := filepath.Ext(videoPath)
	nfoPath := strings.TrimSuffix(videoPath, ext) + ".nfo"

	if _, err := os.Stat(nfoPath); err == nil {
		s.logger.Debug("NFO sidecar found", "video", videoPath, "nfo", nfoPath)
		return nfoPath
	}
	return ""
}

// maxNFOFileSize is the maximum size of an NFO file we'll read (1MB).
// NFO files are typically small XML or single-line URLs; anything larger is suspect.
const maxNFOFileSize = 1 << 20 // 1MB

// Parse reads and parses an NFO file, detecting format (XML or URL).
func (s *NFOReaderService) Parse(nfoPath string) (*NFOData, error) {
	info, err := os.Stat(nfoPath)
	if err != nil {
		return nil, fmt.Errorf("read nfo: %w", err)
	}
	if info.Size() > maxNFOFileSize {
		return nil, fmt.Errorf("nfo file too large (%d bytes, max %d): %s", info.Size(), maxNFOFileSize, nfoPath)
	}

	content, err := os.ReadFile(nfoPath)
	if err != nil {
		return nil, fmt.Errorf("read nfo: %w", err)
	}

	if len(content) == 0 {
		return nil, fmt.Errorf("empty nfo file: %s", nfoPath)
	}

	trimmed := strings.TrimSpace(string(content))

	// Detect format: XML starts with '<' or '<?xml'
	if strings.HasPrefix(trimmed, "<") {
		return s.parseXML([]byte(trimmed))
	}

	// Try URL format (single line with TMDB/IMDB URL)
	return s.parseURL(trimmed)
}

// parseXML parses a Kodi-style XML NFO file
func (s *NFOReaderService) parseXML(content []byte) (*NFOData, error) {
	data := &NFOData{SourceFormat: NFOSourceFormatXML}

	// Try <movie>
	var movie nfoXMLMovie
	if err := xml.Unmarshal(content, &movie); err == nil && movie.XMLName.Local == "movie" {
		data.MediaType = "movie"
		data.Title = movie.Title
		data.OriginalTitle = movie.OriginalTitle
		data.Year = movie.Year
		data.Plot = movie.Plot
		extractUniqueIDs(movie.UniqueIDs, data)
		extractStreamDetails(movie.FileInfo, data)
		return data, nil
	}

	// Try <tvshow>
	var tvshow nfoXMLTVShow
	if err := xml.Unmarshal(content, &tvshow); err == nil && tvshow.XMLName.Local == "tvshow" {
		data.MediaType = "tvshow"
		data.Title = tvshow.Title
		data.OriginalTitle = tvshow.OriginalTitle
		data.Year = tvshow.Year
		data.Plot = tvshow.Plot
		extractUniqueIDs(tvshow.UniqueIDs, data)
		return data, nil
	}

	// Try <episodedetails>
	var episode nfoXMLEpisode
	if err := xml.Unmarshal(content, &episode); err == nil && episode.XMLName.Local == "episodedetails" {
		data.MediaType = "episodedetails"
		data.Title = episode.Title
		data.Plot = episode.Plot
		extractUniqueIDs(episode.UniqueIDs, data)
		extractStreamDetails(episode.FileInfo, data)
		return data, nil
	}

	return nil, fmt.Errorf("unrecognized XML root element in NFO")
}

// parseURL parses a URL-format NFO file (single line with TMDB/IMDB URL)
func (s *NFOReaderService) parseURL(content string) (*NFOData, error) {
	// Check each line — some NFO files may have the URL on a single line
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if id, ok := extractTMDbID(line); ok {
			return &NFOData{
				TMDbID:       id,
				SourceFormat: NFOSourceFormatURL,
			}, nil
		}

		if id, ok := extractIMDbID(line); ok {
			return &NFOData{
				IMDbID:       id,
				SourceFormat: NFOSourceFormatURL,
			}, nil
		}
	}

	return nil, fmt.Errorf("no TMDB/IMDB URL found in NFO")
}

// extractTMDbID extracts a TMDB ID from a URL string
func extractTMDbID(line string) (string, bool) {
	matches := tmdbURLPattern.FindStringSubmatch(line)
	if len(matches) >= 2 {
		return matches[1], true
	}
	return "", false
}

// extractIMDbID extracts an IMDB ID from a URL string or raw ID
func extractIMDbID(line string) (string, bool) {
	matches := imdbURLPattern.FindStringSubmatch(line)
	if len(matches) >= 2 {
		return matches[1], true
	}
	return "", false
}

// extractUniqueIDs extracts TMDB/IMDB IDs from <uniqueid> elements
func extractUniqueIDs(ids []nfoXMLUniqueID, data *NFOData) {
	for _, uid := range ids {
		switch strings.ToLower(uid.Type) {
		case "tmdb":
			data.TMDbID = strings.TrimSpace(uid.Value)
		case "imdb":
			data.IMDbID = strings.TrimSpace(uid.Value)
		}
	}
}

// extractStreamDetails extracts video/audio/subtitle info from <fileinfo>
func extractStreamDetails(fi *nfoXMLFileInfo, data *NFOData) {
	if fi == nil {
		return
	}
	sd := fi.StreamDetails

	if sd.Video != nil {
		data.VideoCodec = sd.Video.Codec
		if sd.Video.Width > 0 && sd.Video.Height > 0 {
			data.VideoResolution = resolveResolution(sd.Video.Width, sd.Video.Height)
		}
	}

	if sd.Audio != nil {
		data.AudioCodec = sd.Audio.Codec
		data.AudioChannels = sd.Audio.Channels
	}

	for _, sub := range sd.Subtitle {
		if sub.Language != "" {
			data.Subtitles = append(data.Subtitles, NFOSubtitleTrack(sub))
		}
	}
}

// resolveResolution converts width×height to a standard resolution label.
// Uses height as the primary indicator, with width as a secondary check
// for standard aspect ratios. This avoids misclassifying ultra-wide formats.
func resolveResolution(width, height int) string {
	if width <= 0 && height <= 0 {
		return ""
	}
	switch {
	case height >= 2160 || (width >= 3840 && height >= 1080):
		return "4K"
	case height >= 1440 || (width >= 2560 && height >= 1080):
		return "1440p"
	case height >= 1080:
		return "1080p"
	case height >= 720:
		return "720p"
	case height >= 480:
		return "480p"
	default:
		return strconv.Itoa(height) + "p"
	}
}
