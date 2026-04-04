package models

import (
	"encoding/json"
	"time"
)

// ParseStatus represents the parsing status of a media file
type ParseStatus string

const (
	ParseStatusPending ParseStatus = "pending"
	ParseStatusParsing ParseStatus = "parsing"
	ParseStatusSuccess ParseStatus = "success"
	ParseStatusNeedsAI ParseStatus = "needs_ai"
	ParseStatusFailed  ParseStatus = "failed"
)

// MetadataSource represents the source of metadata
type MetadataSource string

const (
	MetadataSourceTMDb      MetadataSource = "tmdb"
	MetadataSourceDouban    MetadataSource = "douban"
	MetadataSourceWikipedia MetadataSource = "wikipedia"
	MetadataSourceManual    MetadataSource = "manual"
	MetadataSourceNFO       MetadataSource = "nfo"
	MetadataSourceAI        MetadataSource = "ai"
)

// metadataSourcePriority defines the priority of each metadata source.
// Higher value = higher priority. Manual corrections always win.
var metadataSourcePriority = map[MetadataSource]int{
	MetadataSourceManual:    100,
	MetadataSourceNFO:       80,
	MetadataSourceTMDb:      60,
	MetadataSourceDouban:    50,
	MetadataSourceWikipedia: 40,
	MetadataSourceAI:        20,
}

// ShouldOverwrite returns true if the incoming metadata source may overwrite the current source.
// Returns true when current is empty (first data) or incoming priority >= current priority.
func ShouldOverwrite(current, incoming MetadataSource) bool {
	if current == "" {
		return true
	}
	return metadataSourcePriority[incoming] >= metadataSourcePriority[current]
}

// SubtitleStatus represents the subtitle search status of a media file
type SubtitleStatus string

const (
	SubtitleStatusNotSearched SubtitleStatus = "not_searched"
	SubtitleStatusSearching   SubtitleStatus = "searching"
	SubtitleStatusFound       SubtitleStatus = "found"
	SubtitleStatusNotFound    SubtitleStatus = "not_found"
)

// Genre represents a genre with ID and name
type Genre struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// CastMember represents a cast member in credits
type CastMember struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Character   string `json:"character,omitempty"`
	Order       int    `json:"order,omitempty"`
	ProfilePath string `json:"profile_path,omitempty"`
}

// CrewMember represents a crew member in credits
type CrewMember struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Job         string `json:"job,omitempty"`
	Department  string `json:"department,omitempty"`
	ProfilePath string `json:"profile_path,omitempty"`
}

// Credits represents movie/series credits with cast and crew
type Credits struct {
	Cast []CastMember `json:"cast,omitempty"`
	Crew []CrewMember `json:"crew,omitempty"`
}

// ProductionCountry represents a production country
type ProductionCountry struct {
	ISO3166_1 string `json:"iso_3166_1"`
	Name      string `json:"name"`
}

// SpokenLanguage represents a spoken language
type SpokenLanguage struct {
	ISO639_1 string `json:"iso_639_1"`
	Name     string `json:"name"`
}

// Movie represents a movie entity in the database
type Movie struct {
	// Core fields
	ID            string         `db:"id" json:"id"`
	Title         string         `db:"title" json:"title"`
	OriginalTitle NullString `db:"original_title" json:"original_title,omitempty"`
	ReleaseDate   string         `db:"release_date" json:"release_date"`
	Genres        []string       `db:"genres" json:"genres"` // Simple string array for backward compatibility

	// Rating fields (kept for backward compatibility)
	Rating NullFloat64 `db:"rating" json:"rating,omitempty"`

	// TMDb-specific rating fields
	VoteAverage NullFloat64 `db:"vote_average" json:"vote_average,omitempty"`
	VoteCount   NullInt64   `db:"vote_count" json:"vote_count,omitempty"`
	Popularity  NullFloat64 `db:"popularity" json:"popularity,omitempty"`

	// Content fields
	Overview     NullString `db:"overview" json:"overview,omitempty"`
	PosterPath   NullString `db:"poster_path" json:"poster_path,omitempty"`
	BackdropPath NullString `db:"backdrop_path" json:"backdrop_path,omitempty"`
	Runtime      NullInt64  `db:"runtime" json:"runtime,omitempty"`

	// Metadata fields
	OriginalLanguage NullString `db:"original_language" json:"original_language,omitempty"`
	Status           NullString `db:"status" json:"status,omitempty"`
	IMDbID           NullString `db:"imdb_id" json:"imdb_id,omitempty"`
	TMDbID           NullInt64  `db:"tmdb_id" json:"tmdb_id,omitempty"`

	// New fields for enhanced TMDb data (Story 2-6)
	CreditsJSON            NullString `db:"credits" json:"-"`              // JSON stored in DB
	ProductionCountriesJSON NullString `db:"production_countries" json:"-"` // JSON stored in DB
	SpokenLanguagesJSON    NullString `db:"spoken_languages" json:"-"`     // JSON stored in DB

	// File tracking fields
	FilePath NullString `db:"file_path" json:"file_path,omitempty"`
	FileSize NullInt64  `db:"file_size" json:"file_size,omitempty"`

	// Parse tracking fields
	ParseStatus    ParseStatus    `db:"parse_status" json:"parse_status"`
	MetadataSource NullString `db:"metadata_source" json:"metadata_source,omitempty"`

	// Subtitle tracking fields
	SubtitleStatus       SubtitleStatus  `db:"subtitle_status" json:"subtitle_status"`
	SubtitlePath         NullString  `db:"subtitle_path" json:"subtitle_path,omitempty"`
	SubtitleLanguage     NullString  `db:"subtitle_language" json:"subtitle_language,omitempty"`
	SubtitleLastSearched NullTime    `db:"subtitle_last_searched" json:"subtitle_last_searched,omitempty"`
	SubtitleSearchScore  NullFloat64 `db:"subtitle_search_score" json:"subtitle_search_score,omitempty"`

	// Technical info fields (Story 9c-1)
	VideoCodec      NullString `db:"video_codec" json:"video_codec,omitempty"`
	VideoResolution NullString `db:"video_resolution" json:"video_resolution,omitempty"`
	AudioCodec      NullString `db:"audio_codec" json:"audio_codec,omitempty"`
	AudioChannels   NullInt64  `db:"audio_channels" json:"audio_channels,omitempty"`
	SubtitleTracks  NullString `db:"subtitle_tracks" json:"subtitle_tracks,omitempty"`
	HDRFormat       NullString `db:"hdr_format" json:"hdr_format,omitempty"`

	// Soft-delete flag for removed files (Story 7-2)
	IsRemoved bool `db:"is_removed" json:"is_removed"`

	// Library association (Story 7b-5)
	LibraryID NullString `db:"library_id" json:"library_id,omitempty"`

	// Timestamps
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

// ScanGenres handles scanning genres from database (stored as JSON text)
func (m *Movie) ScanGenres(value interface{}) error {
	if value == nil {
		m.Genres = []string{}
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		str, ok := value.(string)
		if !ok {
			m.Genres = []string{}
			return nil
		}
		bytes = []byte(str)
	}

	if err := json.Unmarshal(bytes, &m.Genres); err != nil {
		m.Genres = []string{}
		return err
	}

	return nil
}

// GenresJSON returns genres as JSON string for database storage
func (m *Movie) GenresJSON() (string, error) {
	if m.Genres == nil {
		m.Genres = []string{}
	}

	bytes, err := json.Marshal(m.Genres)
	if err != nil {
		return "[]", err
	}

	return string(bytes), nil
}

// GetCredits parses and returns the credits from JSON
func (m *Movie) GetCredits() (*Credits, error) {
	if !m.CreditsJSON.Valid || m.CreditsJSON.String == "" {
		return &Credits{}, nil
	}

	var credits Credits
	if err := json.Unmarshal([]byte(m.CreditsJSON.String), &credits); err != nil {
		return nil, err
	}

	return &credits, nil
}

// SetCredits serializes credits to JSON and stores in CreditsJSON
func (m *Movie) SetCredits(credits *Credits) error {
	if credits == nil {
		m.CreditsJSON = NullString{}
		return nil
	}

	bytes, err := json.Marshal(credits)
	if err != nil {
		return err
	}

	m.CreditsJSON = NewNullString(string(bytes))
	return nil
}

// GetProductionCountries parses and returns production countries from JSON
func (m *Movie) GetProductionCountries() ([]ProductionCountry, error) {
	if !m.ProductionCountriesJSON.Valid || m.ProductionCountriesJSON.String == "" {
		return []ProductionCountry{}, nil
	}

	var countries []ProductionCountry
	if err := json.Unmarshal([]byte(m.ProductionCountriesJSON.String), &countries); err != nil {
		return nil, err
	}

	return countries, nil
}

// SetProductionCountries serializes production countries to JSON
func (m *Movie) SetProductionCountries(countries []ProductionCountry) error {
	if countries == nil {
		m.ProductionCountriesJSON = NullString{}
		return nil
	}

	bytes, err := json.Marshal(countries)
	if err != nil {
		return err
	}

	m.ProductionCountriesJSON = NewNullString(string(bytes))
	return nil
}

// GetSpokenLanguages parses and returns spoken languages from JSON
func (m *Movie) GetSpokenLanguages() ([]SpokenLanguage, error) {
	if !m.SpokenLanguagesJSON.Valid || m.SpokenLanguagesJSON.String == "" {
		return []SpokenLanguage{}, nil
	}

	var languages []SpokenLanguage
	if err := json.Unmarshal([]byte(m.SpokenLanguagesJSON.String), &languages); err != nil {
		return nil, err
	}

	return languages, nil
}

// SetSpokenLanguages serializes spoken languages to JSON
func (m *Movie) SetSpokenLanguages(languages []SpokenLanguage) error {
	if languages == nil {
		m.SpokenLanguagesJSON = NullString{}
		return nil
	}

	bytes, err := json.Marshal(languages)
	if err != nil {
		return err
	}

	m.SpokenLanguagesJSON = NewNullString(string(bytes))
	return nil
}

// Validate validates the movie fields
func (m *Movie) Validate() error {
	if m.ID == "" {
		return ErrMovieIDRequired
	}
	if m.Title == "" {
		return ErrMovieTitleRequired
	}
	return nil
}

// Movie validation errors
var (
	ErrMovieIDRequired    = &ValidationError{Field: "id", Message: "movie ID is required"}
	ErrMovieTitleRequired = &ValidationError{Field: "title", Message: "movie title is required"}
)

// ValidationError represents a validation error for a specific field
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}
