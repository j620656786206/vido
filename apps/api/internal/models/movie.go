package models

import (
	"database/sql"
	"encoding/json"
	"time"
)

// Movie represents a movie entity in the database
type Movie struct {
	// Core fields
	ID             string         `db:"id" json:"id"`
	Title          string         `db:"title" json:"title"`
	OriginalTitle  sql.NullString `db:"original_title" json:"originalTitle,omitempty"`
	ReleaseDate    string         `db:"release_date" json:"releaseDate"`
	Genres         []string       `db:"genres" json:"genres"`
	Rating         sql.NullFloat64 `db:"rating" json:"rating,omitempty"`
	Overview       sql.NullString `db:"overview" json:"overview,omitempty"`
	PosterPath     sql.NullString `db:"poster_path" json:"posterPath,omitempty"`
	BackdropPath   sql.NullString `db:"backdrop_path" json:"backdropPath,omitempty"`
	Runtime        sql.NullInt64  `db:"runtime" json:"runtime,omitempty"`
	OriginalLanguage sql.NullString `db:"original_language" json:"originalLanguage,omitempty"`
	Status         sql.NullString `db:"status" json:"status,omitempty"`
	IMDbID         sql.NullString `db:"imdb_id" json:"imdbId,omitempty"`
	TMDbID         sql.NullInt64  `db:"tmdb_id" json:"tmdbId,omitempty"`

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
