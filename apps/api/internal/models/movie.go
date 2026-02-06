package models

import (
	"database/sql"
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
	ProfilePath string `json:"profilePath,omitempty"`
}

// CrewMember represents a crew member in credits
type CrewMember struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Job         string `json:"job,omitempty"`
	Department  string `json:"department,omitempty"`
	ProfilePath string `json:"profilePath,omitempty"`
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
	OriginalTitle sql.NullString `db:"original_title" json:"originalTitle,omitempty"`
	ReleaseDate   string         `db:"release_date" json:"releaseDate"`
	Genres        []string       `db:"genres" json:"genres"` // Simple string array for backward compatibility

	// Rating fields (kept for backward compatibility)
	Rating sql.NullFloat64 `db:"rating" json:"rating,omitempty"`

	// TMDb-specific rating fields
	VoteAverage sql.NullFloat64 `db:"vote_average" json:"voteAverage,omitempty"`
	VoteCount   sql.NullInt64   `db:"vote_count" json:"voteCount,omitempty"`
	Popularity  sql.NullFloat64 `db:"popularity" json:"popularity,omitempty"`

	// Content fields
	Overview     sql.NullString `db:"overview" json:"overview,omitempty"`
	PosterPath   sql.NullString `db:"poster_path" json:"posterPath,omitempty"`
	BackdropPath sql.NullString `db:"backdrop_path" json:"backdropPath,omitempty"`
	Runtime      sql.NullInt64  `db:"runtime" json:"runtime,omitempty"`

	// Metadata fields
	OriginalLanguage sql.NullString `db:"original_language" json:"originalLanguage,omitempty"`
	Status           sql.NullString `db:"status" json:"status,omitempty"`
	IMDbID           sql.NullString `db:"imdb_id" json:"imdbId,omitempty"`
	TMDbID           sql.NullInt64  `db:"tmdb_id" json:"tmdbId,omitempty"`

	// New fields for enhanced TMDb data (Story 2-6)
	CreditsJSON            sql.NullString `db:"credits" json:"-"`              // JSON stored in DB
	ProductionCountriesJSON sql.NullString `db:"production_countries" json:"-"` // JSON stored in DB
	SpokenLanguagesJSON    sql.NullString `db:"spoken_languages" json:"-"`     // JSON stored in DB

	// File tracking fields
	FilePath sql.NullString `db:"file_path" json:"filePath,omitempty"`
	FileSize sql.NullInt64  `db:"file_size" json:"fileSize,omitempty"`

	// Parse tracking fields
	ParseStatus    ParseStatus    `db:"parse_status" json:"parseStatus"`
	MetadataSource sql.NullString `db:"metadata_source" json:"metadataSource,omitempty"`

	// Timestamps
	CreatedAt time.Time `db:"created_at" json:"createdAt"`
	UpdatedAt time.Time `db:"updated_at" json:"updatedAt"`
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
		m.CreditsJSON = sql.NullString{Valid: false}
		return nil
	}

	bytes, err := json.Marshal(credits)
	if err != nil {
		return err
	}

	m.CreditsJSON = sql.NullString{String: string(bytes), Valid: true}
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
		m.ProductionCountriesJSON = sql.NullString{Valid: false}
		return nil
	}

	bytes, err := json.Marshal(countries)
	if err != nil {
		return err
	}

	m.ProductionCountriesJSON = sql.NullString{String: string(bytes), Valid: true}
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
		m.SpokenLanguagesJSON = sql.NullString{Valid: false}
		return nil
	}

	bytes, err := json.Marshal(languages)
	if err != nil {
		return err
	}

	m.SpokenLanguagesJSON = sql.NullString{String: string(bytes), Valid: true}
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
