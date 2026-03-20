package services

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"

	"github.com/vido/api/internal/models"
)

// NFOGenerator generates Kodi-compatible NFO files
type NFOGenerator struct{}

// MovieNFO represents a Kodi-compatible movie NFO structure
type MovieNFO struct {
	XMLName       xml.Name    `xml:"movie"`
	Title         string      `xml:"title"`
	OriginalTitle string      `xml:"originaltitle,omitempty"`
	Year          string      `xml:"year"`
	Plot          string      `xml:"plot,omitempty"`
	Genres        []string    `xml:"genre"`
	Directors     []string    `xml:"director,omitempty"`
	Actors        []NFOActor  `xml:"actor,omitempty"`
	UniqueIDs     []NFOUniqueID `xml:"uniqueid,omitempty"`
	Thumb         string      `xml:"thumb,omitempty"`
	Rating        float64     `xml:"rating,omitempty"`
}

// SeriesNFO represents a Kodi-compatible TV show NFO structure
type SeriesNFO struct {
	XMLName       xml.Name    `xml:"tvshow"`
	Title         string      `xml:"title"`
	OriginalTitle string      `xml:"originaltitle,omitempty"`
	Year          string      `xml:"year"`
	Plot          string      `xml:"plot,omitempty"`
	Genres        []string    `xml:"genre"`
	Actors        []NFOActor  `xml:"actor,omitempty"`
	UniqueIDs     []NFOUniqueID `xml:"uniqueid,omitempty"`
	Thumb         string      `xml:"thumb,omitempty"`
	Rating        float64     `xml:"rating,omitempty"`
	Status        string      `xml:"status,omitempty"`
}

// NFOActor represents an actor entry in NFO
type NFOActor struct {
	Name string `xml:"name"`
	Role string `xml:"role,omitempty"`
}

// NFOUniqueID represents a unique identifier in NFO
type NFOUniqueID struct {
	Type  string `xml:"type,attr"`
	Value string `xml:",chardata"`
}

// CreditsPerson is used for parsing the CreditsJSON field
type CreditsPerson struct {
	Name      string `json:"name"`
	Character string `json:"character,omitempty"`
	Job       string `json:"job,omitempty"`
}

// CreditsData is used for parsing the CreditsJSON field
type CreditsData struct {
	Cast []CreditsPerson `json:"cast"`
	Crew []CreditsPerson `json:"crew"`
}

// GenerateMovieNFO creates a Kodi-compatible movie NFO XML
func (g *NFOGenerator) GenerateMovieNFO(movie models.Movie) []byte {
	nfo := MovieNFO{
		Title: movie.Title,
		Year:  movie.ReleaseDate,
	}

	if movie.OriginalTitle.Valid {
		nfo.OriginalTitle = movie.OriginalTitle.String
	}
	if movie.Overview.Valid {
		nfo.Plot = movie.Overview.String
	}
	if movie.Genres != nil {
		nfo.Genres = movie.Genres
	}
	if movie.VoteAverage.Valid {
		nfo.Rating = movie.VoteAverage.Float64
	}
	if movie.PosterPath.Valid {
		nfo.Thumb = movie.PosterPath.String
	}

	// Parse credits for directors and actors
	if movie.CreditsJSON.Valid {
		var credits CreditsData
		if err := json.Unmarshal([]byte(movie.CreditsJSON.String), &credits); err == nil {
			for _, person := range credits.Crew {
				if person.Job == "Director" {
					nfo.Directors = append(nfo.Directors, person.Name)
				}
			}
			for _, person := range credits.Cast {
				if len(nfo.Actors) >= 10 {
					break
				}
				nfo.Actors = append(nfo.Actors, NFOActor{
					Name: person.Name,
					Role: person.Character,
				})
			}
		}
	}

	// Unique IDs
	if movie.TMDbID.Valid {
		nfo.UniqueIDs = append(nfo.UniqueIDs, NFOUniqueID{Type: "tmdb", Value: fmt.Sprintf("%d", movie.TMDbID.Int64)})
	}
	if movie.IMDbID.Valid {
		nfo.UniqueIDs = append(nfo.UniqueIDs, NFOUniqueID{Type: "imdb", Value: movie.IMDbID.String})
	}

	return marshalNFO(nfo)
}

// GenerateSeriesNFO creates a Kodi-compatible TV show NFO XML
func (g *NFOGenerator) GenerateSeriesNFO(series models.Series) []byte {
	nfo := SeriesNFO{
		Title: series.Title,
		Year:  series.FirstAirDate,
	}

	if series.OriginalTitle.Valid {
		nfo.OriginalTitle = series.OriginalTitle.String
	}
	if series.Overview.Valid {
		nfo.Plot = series.Overview.String
	}
	if series.Genres != nil {
		nfo.Genres = series.Genres
	}
	if series.VoteAverage.Valid {
		nfo.Rating = series.VoteAverage.Float64
	}
	if series.PosterPath.Valid {
		nfo.Thumb = series.PosterPath.String
	}
	if series.Status.Valid {
		nfo.Status = series.Status.String
	}

	// Parse credits for actors
	if series.CreditsJSON.Valid {
		var credits CreditsData
		if err := json.Unmarshal([]byte(series.CreditsJSON.String), &credits); err == nil {
			for _, person := range credits.Cast {
				if len(nfo.Actors) >= 10 {
					break
				}
				nfo.Actors = append(nfo.Actors, NFOActor{
					Name: person.Name,
					Role: person.Character,
				})
			}
		}
	}

	// Unique IDs
	if series.TMDbID.Valid {
		nfo.UniqueIDs = append(nfo.UniqueIDs, NFOUniqueID{Type: "tmdb", Value: fmt.Sprintf("%d", series.TMDbID.Int64)})
	}
	if series.IMDbID.Valid {
		nfo.UniqueIDs = append(nfo.UniqueIDs, NFOUniqueID{Type: "imdb", Value: series.IMDbID.String})
	}

	return marshalNFO(nfo)
}

func marshalNFO(v interface{}) []byte {
	data, err := xml.MarshalIndent(v, "", "  ")
	if err != nil {
		return []byte(fmt.Sprintf("<!-- NFO generation failed: %v -->", err))
	}
	return append([]byte(xml.Header), data...)
}

func writeFile(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create dir: %w", err)
	}
	return os.WriteFile(path, data, 0o644)
}
