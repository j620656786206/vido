package models

import (
	"encoding/json"
	"time"
)

// SeasonSummary represents a season summary for series
type SeasonSummary struct {
	ID           int    `json:"id"`
	SeasonNumber int    `json:"season_number"`
	Name         string `json:"name,omitempty"`
	Overview     string `json:"overview,omitempty"`
	PosterPath   string `json:"poster_path,omitempty"`
	AirDate      string `json:"air_date,omitempty"`
	EpisodeCount int    `json:"episode_count,omitempty"`
}

// Network represents a TV network
type Network struct {
	ID            int    `json:"id"`
	Name          string `json:"name"`
	LogoPath      string `json:"logo_path,omitempty"`
	OriginCountry string `json:"origin_country,omitempty"`
}

// Series represents a TV series entity in the database
type Series struct {
	// Core fields
	ID            string         `db:"id" json:"id"`
	Title         string         `db:"title" json:"title"`
	OriginalTitle NullString `db:"original_title" json:"original_title,omitempty"`
	FirstAirDate  string         `db:"first_air_date" json:"first_air_date"`
	LastAirDate   NullString `db:"last_air_date" json:"last_air_date,omitempty"`
	Genres        []string       `db:"genres" json:"genres"`

	// Rating fields (kept for backward compatibility)
	Rating NullFloat64 `db:"rating" json:"rating,omitempty"`

	// TMDb-specific rating fields
	VoteAverage NullFloat64 `db:"vote_average" json:"vote_average,omitempty"`
	VoteCount   NullInt64   `db:"vote_count" json:"vote_count,omitempty"`
	Popularity  NullFloat64 `db:"popularity" json:"popularity,omitempty"`

	// Content fields
	Overview         NullString `db:"overview" json:"overview,omitempty"`
	PosterPath       NullString `db:"poster_path" json:"poster_path,omitempty"`
	BackdropPath     NullString `db:"backdrop_path" json:"backdrop_path,omitempty"`
	NumberOfSeasons  NullInt64  `db:"number_of_seasons" json:"number_of_seasons,omitempty"`
	NumberOfEpisodes NullInt64  `db:"number_of_episodes" json:"number_of_episodes,omitempty"`

	// Metadata fields
	Status           NullString `db:"status" json:"status,omitempty"`
	OriginalLanguage NullString `db:"original_language" json:"original_language,omitempty"`
	IMDbID           NullString `db:"imdb_id" json:"imdb_id,omitempty"`
	TMDbID           NullInt64  `db:"tmdb_id" json:"tmdb_id,omitempty"`
	InProduction     NullBool   `db:"in_production" json:"in_production,omitempty"`

	// New fields for enhanced TMDb data (Story 2-6)
	CreditsJSON  NullString `db:"credits" json:"-"`  // JSON stored in DB
	SeasonsJSON  NullString `db:"seasons" json:"-"`  // JSON stored in DB
	NetworksJSON NullString `db:"networks" json:"-"` // JSON stored in DB

	// File tracking fields
	FilePath NullString `db:"file_path" json:"file_path,omitempty"`
	FileSize NullInt64  `db:"file_size" json:"file_size,omitempty"`

	// Technical info fields (Story 9c-1)
	VideoCodec      NullString `db:"video_codec" json:"video_codec,omitempty"`
	VideoResolution NullString `db:"video_resolution" json:"video_resolution,omitempty"`
	AudioCodec      NullString `db:"audio_codec" json:"audio_codec,omitempty"`
	AudioChannels   NullInt64  `db:"audio_channels" json:"audio_channels,omitempty"`
	SubtitleTracks  NullString `db:"subtitle_tracks" json:"subtitle_tracks,omitempty"`
	HDRFormat       NullString `db:"hdr_format" json:"hdr_format,omitempty"`

	// Parse tracking fields
	ParseStatus    ParseStatus    `db:"parse_status" json:"parse_status"`
	MetadataSource NullString `db:"metadata_source" json:"metadata_source,omitempty"`

	// Subtitle tracking fields
	SubtitleStatus       SubtitleStatus  `db:"subtitle_status" json:"subtitle_status"`
	SubtitlePath         NullString  `db:"subtitle_path" json:"subtitle_path,omitempty"`
	SubtitleLanguage     NullString  `db:"subtitle_language" json:"subtitle_language,omitempty"`
	SubtitleLastSearched NullTime    `db:"subtitle_last_searched" json:"subtitle_last_searched,omitempty"`
	SubtitleSearchScore  NullFloat64 `db:"subtitle_search_score" json:"subtitle_search_score,omitempty"`

	// Soft-delete flag for removed files (Story 7-2)
	IsRemoved bool `db:"is_removed" json:"is_removed"`

	// Timestamps
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

// ScanGenres handles scanning genres from database (stored as JSON text)
func (s *Series) ScanGenres(value interface{}) error {
	if value == nil {
		s.Genres = []string{}
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		str, ok := value.(string)
		if !ok {
			s.Genres = []string{}
			return nil
		}
		bytes = []byte(str)
	}

	if err := json.Unmarshal(bytes, &s.Genres); err != nil {
		s.Genres = []string{}
		return err
	}

	return nil
}

// GenresJSON returns genres as JSON string for database storage
func (s *Series) GenresJSON() (string, error) {
	if s.Genres == nil {
		s.Genres = []string{}
	}

	bytes, err := json.Marshal(s.Genres)
	if err != nil {
		return "[]", err
	}

	return string(bytes), nil
}

// GetCredits parses and returns the credits from JSON
func (s *Series) GetCredits() (*Credits, error) {
	if !s.CreditsJSON.Valid || s.CreditsJSON.String == "" {
		return &Credits{}, nil
	}

	var credits Credits
	if err := json.Unmarshal([]byte(s.CreditsJSON.String), &credits); err != nil {
		return nil, err
	}

	return &credits, nil
}

// SetCredits serializes credits to JSON and stores in CreditsJSON
func (s *Series) SetCredits(credits *Credits) error {
	if credits == nil {
		s.CreditsJSON = NullString{}
		return nil
	}

	bytes, err := json.Marshal(credits)
	if err != nil {
		return err
	}

	s.CreditsJSON = NewNullString(string(bytes))
	return nil
}

// GetSeasons parses and returns the seasons from JSON
func (s *Series) GetSeasons() ([]SeasonSummary, error) {
	if !s.SeasonsJSON.Valid || s.SeasonsJSON.String == "" {
		return []SeasonSummary{}, nil
	}

	var seasons []SeasonSummary
	if err := json.Unmarshal([]byte(s.SeasonsJSON.String), &seasons); err != nil {
		return nil, err
	}

	return seasons, nil
}

// SetSeasons serializes seasons to JSON
func (s *Series) SetSeasons(seasons []SeasonSummary) error {
	if seasons == nil {
		s.SeasonsJSON = NullString{}
		return nil
	}

	bytes, err := json.Marshal(seasons)
	if err != nil {
		return err
	}

	s.SeasonsJSON = NewNullString(string(bytes))
	return nil
}

// GetNetworks parses and returns the networks from JSON
func (s *Series) GetNetworks() ([]Network, error) {
	if !s.NetworksJSON.Valid || s.NetworksJSON.String == "" {
		return []Network{}, nil
	}

	var networks []Network
	if err := json.Unmarshal([]byte(s.NetworksJSON.String), &networks); err != nil {
		return nil, err
	}

	return networks, nil
}

// SetNetworks serializes networks to JSON
func (s *Series) SetNetworks(networks []Network) error {
	if networks == nil {
		s.NetworksJSON = NullString{}
		return nil
	}

	bytes, err := json.Marshal(networks)
	if err != nil {
		return err
	}

	s.NetworksJSON = NewNullString(string(bytes))
	return nil
}

// Validate validates the series fields
func (s *Series) Validate() error {
	if s.ID == "" {
		return ErrSeriesIDRequired
	}
	if s.Title == "" {
		return ErrSeriesTitleRequired
	}
	return nil
}

// Series validation errors
var (
	ErrSeriesIDRequired    = &ValidationError{Field: "id", Message: "series ID is required"}
	ErrSeriesTitleRequired = &ValidationError{Field: "title", Message: "series title is required"}
)
