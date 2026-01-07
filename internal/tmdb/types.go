package tmdb

import "time"

// Movie represents a movie from TMDb API
type Movie struct {
	ID               int     `json:"id" example:"550"`
	Title            string  `json:"title" example:"Fight Club"`
	OriginalTitle    string  `json:"original_title" example:"Fight Club"`
	Overview         string  `json:"overview" example:"A ticking-time-bomb insomniac and a slippery soap salesman channel primal male aggression into a shocking new form of therapy."`
	ReleaseDate      string  `json:"release_date" example:"1999-10-15"`
	PosterPath       *string `json:"poster_path" example:"/pB8BM7pdSp6B6Ih7QZ4DrQ3PmJK.jpg"`
	BackdropPath     *string `json:"backdrop_path" example:"/fCayJrkfRaCRCTh8GqN30f8oyQF.jpg"`
	VoteAverage      float64 `json:"vote_average" example:"8.4"`
	VoteCount        int     `json:"vote_count" example:"26280"`
	Popularity       float64 `json:"popularity" example:"61.416"`
	GenreIDs         []int   `json:"genre_ids" example:"18,53"`
	OriginalLanguage string  `json:"original_language" example:"en"`
	Adult            bool    `json:"adult" example:"false"`
	Video            bool    `json:"video" example:"false"`
}

// MovieDetails represents detailed movie information from TMDb API
// This includes additional fields not present in search results
type MovieDetails struct {
	Movie
	Budget            int64         `json:"budget" example:"63000000"`
	Revenue           int64         `json:"revenue" example:"100853753"`
	Runtime           int           `json:"runtime" example:"139"`
	Status            string        `json:"status" example:"Released"`
	Tagline           string        `json:"tagline" example:"Mischief. Mayhem. Soap."`
	Genres            []Genre       `json:"genres"`
	ProductionCountries []Country   `json:"production_countries"`
	SpokenLanguages   []Language    `json:"spoken_languages"`
	ImdbID            string        `json:"imdb_id" example:"tt0137523"`
	Homepage          *string       `json:"homepage"`
}

// TVShow represents a TV show from TMDb API
type TVShow struct {
	ID               int      `json:"id" example:"1396"`
	Name             string   `json:"name" example:"Breaking Bad"`
	OriginalName     string   `json:"original_name" example:"Breaking Bad"`
	Overview         string   `json:"overview" example:"When Walter White, a New Mexico chemistry teacher, is diagnosed with Stage III cancer and given a prognosis of only two years left to live."`
	FirstAirDate     string   `json:"first_air_date" example:"2008-01-20"`
	PosterPath       *string  `json:"poster_path" example:"/ggFHVNu6YYI5L9pCfOacjizRGt.jpg"`
	BackdropPath     *string  `json:"backdrop_path" example:"/tsRy63Mu5cu8etL1X7ZLyf7UP1M.jpg"`
	VoteAverage      float64  `json:"vote_average" example:"8.9"`
	VoteCount        int      `json:"vote_count" example:"12345"`
	Popularity       float64  `json:"popularity" example:"369.594"`
	GenreIDs         []int    `json:"genre_ids" example:"18,80"`
	OriginalLanguage string   `json:"original_language" example:"en"`
	OriginCountry    []string `json:"origin_country" example:"US"`
}

// TVShowDetails represents detailed TV show information from TMDb API
type TVShowDetails struct {
	TVShow
	CreatedBy         []Creator     `json:"created_by"`
	EpisodeRunTime    []int         `json:"episode_run_time" example:"45,47"`
	Genres            []Genre       `json:"genres"`
	Homepage          *string       `json:"homepage"`
	InProduction      bool          `json:"in_production" example:"false"`
	Languages         []string      `json:"languages" example:"en"`
	LastAirDate       string        `json:"last_air_date" example:"2013-09-29"`
	NumberOfEpisodes  int           `json:"number_of_episodes" example:"62"`
	NumberOfSeasons   int           `json:"number_of_seasons" example:"5"`
	ProductionCountries []Country   `json:"production_countries"`
	Seasons           []Season      `json:"seasons"`
	Status            string        `json:"status" example:"Ended"`
	Tagline           string        `json:"tagline" example:"Change the equation."`
	Type              string        `json:"type" example:"Scripted"`
}

// SearchResultMovies represents paginated movie search results from TMDb API
type SearchResultMovies struct {
	Page         int     `json:"page" example:"1"`
	Results      []Movie `json:"results"`
	TotalPages   int     `json:"total_pages" example:"500"`
	TotalResults int     `json:"total_results" example:"10000"`
}

// SearchResultTVShows represents paginated TV show search results from TMDb API
type SearchResultTVShows struct {
	Page         int      `json:"page" example:"1"`
	Results      []TVShow `json:"results"`
	TotalPages   int      `json:"total_pages" example:"500"`
	TotalResults int      `json:"total_results" example:"10000"`
}

// Genre represents a movie or TV show genre
type Genre struct {
	ID   int    `json:"id" example:"18"`
	Name string `json:"name" example:"Drama"`
}

// Country represents a production country
type Country struct {
	ISO31661 string `json:"iso_3166_1" example:"US"`
	Name     string `json:"name" example:"United States of America"`
}

// Language represents a spoken language
type Language struct {
	ISO6391     string `json:"iso_639_1" example:"en"`
	Name        string `json:"name" example:"English"`
	EnglishName string `json:"english_name" example:"English"`
}

// Creator represents a TV show creator
type Creator struct {
	ID          int    `json:"id" example:"66633"`
	CreditID    string `json:"credit_id" example:"52542286760ee313280006ce"`
	Name        string `json:"name" example:"Vince Gilligan"`
	Gender      int    `json:"gender" example:"2"`
	ProfilePath *string `json:"profile_path"`
}

// Season represents a TV show season
type Season struct {
	AirDate      *string `json:"air_date" example:"2008-01-20"`
	EpisodeCount int     `json:"episode_count" example:"7"`
	ID           int     `json:"id" example:"3572"`
	Name         string  `json:"name" example:"Season 1"`
	Overview     string  `json:"overview" example:"High school chemistry teacher Walter White's life is suddenly transformed by a dire medical diagnosis."`
	PosterPath   *string `json:"poster_path" example:"/1BP4xYv9ZG4ZVHkL7ocOziBbSYH.jpg"`
	SeasonNumber int     `json:"season_number" example:"1"`
}

// ImageConfiguration represents TMDb image configuration
// This is used to construct full image URLs
type ImageConfiguration struct {
	BaseURL       string   `json:"base_url" example:"http://image.tmdb.org/t/p/"`
	SecureBaseURL string   `json:"secure_base_url" example:"https://image.tmdb.org/t/p/"`
	BackdropSizes []string `json:"backdrop_sizes" example:"w300,w780,w1280,original"`
	LogoSizes     []string `json:"logo_sizes" example:"w45,w92,w154,w185,w300,w500,original"`
	PosterSizes   []string `json:"poster_sizes" example:"w92,w154,w185,w342,w500,w780,original"`
	ProfileSizes  []string `json:"profile_sizes" example:"w45,w185,h632,original"`
	StillSizes    []string `json:"still_sizes" example:"w92,w185,w300,original"`
}

// Configuration represents the TMDb API configuration response
type Configuration struct {
	Images     ImageConfiguration `json:"images"`
	ChangeKeys []string          `json:"change_keys"`
}

// Image represents image metadata for posters and backdrops
type Image struct {
	AspectRatio float64 `json:"aspect_ratio" example:"0.667"`
	FilePath    string  `json:"file_path" example:"/pB8BM7pdSp6B6Ih7QZ4DrQ3PmJK.jpg"`
	Height      int     `json:"height" example:"3000"`
	ISO6391     *string `json:"iso_639_1" example:"en"`
	VoteAverage float64 `json:"vote_average" example:"5.388"`
	VoteCount   int     `json:"vote_count" example:"4"`
	Width       int     `json:"width" example:"2000"`
}

// ImagesResponse represents the response from the images endpoint
type ImagesResponse struct {
	ID        int     `json:"id" example:"550"`
	Backdrops []Image `json:"backdrops"`
	Logos     []Image `json:"logos"`
	Posters   []Image `json:"posters"`
}

// ErrorResponse represents a TMDb API error response
type ErrorResponse struct {
	StatusCode    int    `json:"status_code" example:"7"`
	StatusMessage string `json:"status_message" example:"Invalid API key: You must be granted a valid key."`
	Success       bool   `json:"success" example:"false"`
}

// Date represents a date field that can be parsed from various formats
type Date struct {
	time.Time
}

// UnmarshalJSON handles parsing dates from TMDb API which may be in different formats
func (d *Date) UnmarshalJSON(b []byte) error {
	s := string(b)
	// Remove quotes
	if len(s) > 1 && s[0] == '"' && s[len(s)-1] == '"' {
		s = s[1 : len(s)-1]
	}

	// Handle empty string
	if s == "" || s == "null" {
		return nil
	}

	// Parse the date (TMDb uses YYYY-MM-DD format)
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return err
	}

	d.Time = t
	return nil
}

// MarshalJSON converts the date back to TMDb format
func (d Date) MarshalJSON() ([]byte, error) {
	if d.Time.IsZero() {
		return []byte("null"), nil
	}
	return []byte(`"` + d.Time.Format("2006-01-02") + `"`), nil
}
