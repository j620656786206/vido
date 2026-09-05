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

// SubtitleStatus represents the subtitle lifecycle state of a media file.
//
// [@contract-v3] (story sub-1-2 AC #2; bumped v1→v2 by sub-2-2a: 9→10 values,
// +untranslated; bumped v2→v3 by sub-3-1: the VALUE SET is UNCHANGED at 10 —
// only `no_text_source`'s TERMINALITY changes, from a permanent verdict to an
// INTERMEDIATE one the ASR fallback leg can recover; `skipped` remains
// terminal) — frontend-consumed wire contract: serialized as
// json:"subtitle_status" on Movie/Series/Episode and used as a URL search
// param on /library. Consumers: sub-1-4 (routing verdicts), sub-1-5 (pipeline
// writes), sub-1-6 (enumeration), sub-1-7b (badge rendering), sub-2-2b
// (untranslated badge), sub-3-1 (ASR leg + sweep terminality). Adding or
// renaming a value is a Rule 20 bump plus a downstream stale-mark.
//
// The four search-flavoured values predate the pipeline (migrations 018/025);
// the five pipeline-flavoured values were added by sub-1-2 because FR5 ("mark
// an item as having no usable text source") could not be expressed at all in
// the search vocabulary.
type SubtitleStatus string

const (
	// --- search-flavoured, pre-existing (migrations 018/025). UNCHANGED. ---

	SubtitleStatusNotSearched SubtitleStatus = "not_searched"
	SubtitleStatusSearching   SubtitleStatus = "searching"
	SubtitleStatusFound       SubtitleStatus = "found"
	SubtitleStatusNotFound    SubtitleStatus = "not_found"

	// --- pipeline-flavoured, added by story sub-1-2. ---

	// SubtitleStatusProbing — ffprobe is enumerating subtitle tracks (FR1).
	SubtitleStatusProbing SubtitleStatus = "probing"
	// SubtitleStatusExtracting — an embedded track is being extracted (FR2/FR3).
	SubtitleStatusExtracting SubtitleStatus = "extracting"
	// SubtitleStatusTranslating — LLM translation is in flight (FR10).
	SubtitleStatusTranslating SubtitleStatus = "translating"
	// SubtitleStatusNoTextSource — INTERMEDIATE since sub-3-1 ([@contract-v2→v3];
	// TERMINAL through v2). The file has no usable text subtitle track at all
	// (image-only tracks, or none) — FR5. The ASR fallback leg recovers it
	// when the transcription service is available; on an ASR-less deployment
	// the sweep-side availability gate keeps it out of re-enumeration, so the
	// pre-M2 once-only behavior is preserved.
	SubtitleStatusNoTextSource SubtitleStatus = "no_text_source"
	// SubtitleStatusSkipped — TERMINAL. A text track exists but the pipeline
	// deliberately declined it: an `und` or non-English tag (FR9 + P0, where
	// `und` is NEVER treated as English). A deliberate routing decision, not a
	// recoverable gap — ASR must not second-guess it (sub-3-1 AC #1).
	// Recoverable by a corrected track tag or the manual flow.
	SubtitleStatusSkipped SubtitleStatus = "skipped"

	// --- added by story sub-2-2a ([@contract-v1→v2]). ---

	// SubtitleStatusUntranslated — TERMINAL. A generated subtitle exists on
	// disk but the EXPECTED translation step did not run (translation key
	// unconfigured, or a non-fatal translate failure). Written ONLY by the
	// generation pipeline — never inferred from embedded tracks: an embedded
	// English track on a foreign film was never owed a translation, and this
	// value names the missing step, not the artifact. Recoverable by
	// configuring a key and re-running 生成字幕, which resumes translate-only
	// (the English SRT on disk is reused; extract+ASR are skipped).
	SubtitleStatusUntranslated SubtitleStatus = "untranslated"
)

// AllSubtitleStatuses is the authoritative ordered value set behind the
// contract stamp above (version-neutral on purpose — the stamp bumps, this
// sentence should not re-stale). Extend it here and nowhere else — every consumer that
// needs to enumerate statuses reads this, so a constant added without a matching
// entry here is a contract bug (guarded by a test).
func AllSubtitleStatuses() []SubtitleStatus {
	return []SubtitleStatus{
		SubtitleStatusNotSearched,
		SubtitleStatusSearching,
		SubtitleStatusFound,
		SubtitleStatusNotFound,
		SubtitleStatusProbing,
		SubtitleStatusExtracting,
		SubtitleStatusTranslating,
		SubtitleStatusNoTextSource,
		SubtitleStatusSkipped,
		SubtitleStatusUntranslated,
	}
}

// IsValid reports whether s is a known subtitle status.
//
// Additive only: no existing call site validates SubtitleStatus, so this cannot
// reject data that used to be written. It is deliberately NOT wired into the
// shipped UpdateSubtitleStatus search path — sub-1-4 onwards use it on the write
// paths they own.
func (s SubtitleStatus) IsValid() bool {
	for _, known := range AllSubtitleStatuses() {
		if s == known {
			return true
		}
	}
	return false
}

// IsTerminal reports whether s is an end state that no further pipeline stage
// will advance.
func (s SubtitleStatus) IsTerminal() bool {
	switch s {
	// no_text_source left this set at [@contract-v3] (sub-3-1): the ASR
	// fallback leg advances it, so calling it terminal would be a lie the
	// moment an ASR key is configured.
	case SubtitleStatusFound, SubtitleStatusNotFound,
		SubtitleStatusSkipped, SubtitleStatusUntranslated:
		return true
	default:
		return false
	}
}

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
	ID            string     `db:"id" json:"id"`
	Title         string     `db:"title" json:"title"`
	OriginalTitle NullString `db:"original_title" json:"original_title,omitempty"`
	ReleaseDate   string     `db:"release_date" json:"release_date"`
	Genres        []string   `db:"genres" json:"genres"` // Simple string array for backward compatibility

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

	// DurationSeconds is the CONTAINER duration ffprobe measured (migration
	// 035, sub-6-10a). Runtime above is TMDb's editorial figure and is absent
	// for anything unmatched; this one comes from the file itself, so it is
	// what the subtitle estimator prices from first. NULL = never measured
	// (pre-035 row, or a failed probe) — never "zero length".
	DurationSeconds NullInt64 `db:"duration_seconds" json:"duration_seconds,omitempty"`

	// Metadata fields
	OriginalLanguage NullString `db:"original_language" json:"original_language,omitempty"`
	Status           NullString `db:"status" json:"status,omitempty"`
	IMDbID           NullString `db:"imdb_id" json:"imdb_id,omitempty"`
	TMDbID           NullInt64  `db:"tmdb_id" json:"tmdb_id,omitempty"`

	// New fields for enhanced TMDb data (Story 2-6)
	CreditsJSON             NullString `db:"credits" json:"-"`              // JSON stored in DB
	ProductionCountriesJSON NullString `db:"production_countries" json:"-"` // JSON stored in DB
	SpokenLanguagesJSON     NullString `db:"spoken_languages" json:"-"`     // JSON stored in DB

	// ProductionCountries is the parsed, wire-exposed form of ProductionCountriesJSON,
	// populated on read (scanMovie) via GetProductionCountries(). It drives the FE §9b
	// CN-subtitle-policy display. db:"-" — computed, never a scan/write target.
	ProductionCountries []ProductionCountry `db:"-" json:"production_countries,omitempty"`

	// Credits is the parsed, wire-exposed form of CreditsJSON, populated on read
	// (scanMovie) via GetCredits() only when cast/crew is non-empty. Manual Metadata-Editor
	// edits are the only writer today; the FE prefers this over live TMDb when the movie's
	// metadata_source is "manual". db:"-" — computed, never a scan/write target.
	Credits *Credits `db:"-" json:"credits,omitempty"`

	// SpokenLanguages is the parsed, wire-exposed form of SpokenLanguagesJSON, populated on
	// read (scanMovie) via GetSpokenLanguages(). Persist-only: exposed on the payload but no
	// UI consumer today (story disc-2026-07-credits-spoken-languages-persist AC #7).
	SpokenLanguages []SpokenLanguage `db:"-" json:"spoken_languages,omitempty"`

	// File tracking fields
	FilePath NullString `db:"file_path" json:"file_path,omitempty"`
	FileSize NullInt64  `db:"file_size" json:"file_size,omitempty"`

	// Parse tracking fields
	ParseStatus    ParseStatus `db:"parse_status" json:"parse_status"`
	MetadataSource NullString  `db:"metadata_source" json:"metadata_source,omitempty"`

	// Subtitle tracking fields
	SubtitleStatus       SubtitleStatus `db:"subtitle_status" json:"subtitle_status"`
	SubtitlePath         NullString     `db:"subtitle_path" json:"subtitle_path,omitempty"`
	SubtitleLanguage     NullString     `db:"subtitle_language" json:"subtitle_language,omitempty"`
	SubtitleLastSearched NullTime       `db:"subtitle_last_searched" json:"subtitle_last_searched,omitempty"`
	SubtitleSearchScore  NullFloat64    `db:"subtitle_search_score" json:"subtitle_search_score,omitempty"`

	// Technical info fields (Story 9c-1)
	VideoCodec      NullString `db:"video_codec" json:"video_codec,omitempty"`
	VideoResolution NullString `db:"video_resolution" json:"video_resolution,omitempty"`
	AudioCodec      NullString `db:"audio_codec" json:"audio_codec,omitempty"`
	AudioChannels   NullInt64  `db:"audio_channels" json:"audio_channels,omitempty"`
	SubtitleTracks  NullString `db:"subtitle_tracks" json:"subtitle_tracks,omitempty"`
	HDRFormat       NullString `db:"hdr_format" json:"hdr_format,omitempty"`

	// Douban rating fields (Story 12-1) — denormalized from douban_cache for fast
	// dual-rating reads on the detail page. Populated by background enrichment.
	DoubanID        NullString  `db:"douban_id" json:"douban_id,omitempty"`
	DoubanRating    NullFloat64 `db:"douban_rating" json:"douban_rating,omitempty"`
	DoubanVoteCount NullInt64   `db:"douban_vote_count" json:"douban_vote_count,omitempty"`

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
