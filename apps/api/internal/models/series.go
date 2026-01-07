package models

import (
	"database/sql"
	"encoding/json"
	"time"
)

// Series represents a TV series entity in the database
type Series struct {
	// Core fields
	ID               string         `db:"id" json:"id"`
	Title            string         `db:"title" json:"title"`
	OriginalTitle    sql.NullString `db:"original_title" json:"originalTitle,omitempty"`
	FirstAirDate     string         `db:"first_air_date" json:"firstAirDate"`
	LastAirDate      sql.NullString `db:"last_air_date" json:"lastAirDate,omitempty"`
	Genres           []string       `db:"genres" json:"genres"`
	Rating           sql.NullFloat64 `db:"rating" json:"rating,omitempty"`
	Overview         sql.NullString `db:"overview" json:"overview,omitempty"`
	PosterPath       sql.NullString `db:"poster_path" json:"posterPath,omitempty"`
	BackdropPath     sql.NullString `db:"backdrop_path" json:"backdropPath,omitempty"`
	NumberOfSeasons  sql.NullInt64  `db:"number_of_seasons" json:"numberOfSeasons,omitempty"`
	NumberOfEpisodes sql.NullInt64  `db:"number_of_episodes" json:"numberOfEpisodes,omitempty"`
	Status           sql.NullString `db:"status" json:"status,omitempty"`
	OriginalLanguage sql.NullString `db:"original_language" json:"originalLanguage,omitempty"`
	IMDbID           sql.NullString `db:"imdb_id" json:"imdbId,omitempty"`
	TMDbID           sql.NullInt64  `db:"tmdb_id" json:"tmdbId,omitempty"`
	InProduction     sql.NullBool   `db:"in_production" json:"inProduction,omitempty"`

	// Timestamps
	CreatedAt time.Time `db:"created_at" json:"createdAt"`
	UpdatedAt time.Time `db:"updated_at" json:"updatedAt"`
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
