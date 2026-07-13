package services

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/vido/api/internal/metadata"
	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/parser"
	"github.com/vido/api/internal/repository"
)

// MediaIngestService is the single place that turns a TV episode file into
// series/seasons/episodes rows.
//
// Before this existed the scanner wrote EVERY scanned file into `movies` — the
// media-type decision was computed and thrown away (scanDir.contentType was never
// threaded past StartScan) — while the only correct TV-ingest code in the tree lived
// in ParseQueueService, which main.go never constructs. The result on a real library
// was one `movies` row per episode, all wearing the same series' TMDb metadata, and
// series/seasons/episodes permanently empty. Both callers now share this service so
// the logic cannot drift apart again.
type MediaIngestService struct {
	seriesRepo  repository.SeriesRepositoryInterface
	seasonRepo  repository.SeasonRepositoryInterface
	episodeRepo repository.EpisodeRepositoryInterface
	logger      *slog.Logger
}

// NewMediaIngestService creates a MediaIngestService.
func NewMediaIngestService(
	seriesRepo repository.SeriesRepositoryInterface,
	seasonRepo repository.SeasonRepositoryInterface,
	episodeRepo repository.EpisodeRepositoryInterface,
	logger *slog.Logger,
) *MediaIngestService {
	if logger == nil {
		logger = slog.Default()
	}
	return &MediaIngestService{
		seriesRepo:  seriesRepo,
		seasonRepo:  seasonRepo,
		episodeRepo: episodeRepo,
		logger:      logger,
	}
}

// SeriesInput identifies a series to find-or-create.
//
// A series is keyed by TMDbID when the caller already resolved metadata (the parse-queue
// path), and by SeriesDir otherwise (the scanner path, which runs before any TMDb lookup).
// SeriesDir is the series' own folder — stable across a rename of any single episode file,
// and unique per series in every NAS layout we support.
type SeriesInput struct {
	TMDbID    int64  // 0 when unknown (scanner runs before enrichment)
	SeriesDir string // e.g. /media/tv/鵲刀門傳奇 — the identity when TMDbID is 0
	Title     string // best-effort title; the enrichment pass replaces it with TMDb's
	LibraryID string

	// Metadata is set when the caller already resolved the show against a provider
	// (the parse-queue path). The series is then created complete, with parse_status
	// success. The scanner leaves it nil — it runs before any TMDb lookup — and the
	// enrichment pass fills the row in afterwards.
	Metadata *metadata.MetadataItem
	Source   models.MetadataSource
}

// EpisodeInput identifies one episode file.
type EpisodeInput struct {
	SeriesID      string
	SeasonID      string
	SeasonNumber  int
	EpisodeNumber int
	Title         string
	FilePath      string
}

// seasonDirPattern matches a season folder so SeriesDirFor can climb past it:
// Season 02, Season.2, S02, 第二季, Specials.
var seasonDirPattern = regexp.MustCompile(`(?i)^(season[\s._-]*\d+|s\d{1,2}|specials?|第.{1,3}季)$`)

// SeriesDirFor derives the series folder for an episode file.
//
//	/media/tv/鵲刀門傳奇/Season02/....S02E05.mkv  →  /media/tv/鵲刀門傳奇
//	/media/tv/某劇/某劇.S01E01.mkv                →  /media/tv/某劇
//
// scanRoot guards the flat-layout case: if climbing would land on the library root
// itself, every series would collapse into one row, so we stop at the file's own parent.
func SeriesDirFor(filePath, scanRoot string) string {
	dir := filepath.Dir(filePath)

	if seasonDirPattern.MatchString(filepath.Base(dir)) {
		parent := filepath.Dir(dir)
		if !sameDir(parent, scanRoot) {
			return parent
		}
	}
	return dir
}

func sameDir(a, b string) bool {
	if b == "" {
		return false
	}
	return filepath.Clean(a) == filepath.Clean(b)
}

// TitleFromSeriesDir is the fallback series title when the filename parser gives nothing
// usable: the folder name is what the user themselves called the show.
func TitleFromSeriesDir(seriesDir string) string {
	return strings.TrimSpace(filepath.Base(seriesDir))
}

// UpsertSeries finds an existing series or creates one, and returns its ID.
func (s *MediaIngestService) UpsertSeries(ctx context.Context, in SeriesInput) (string, error) {
	if in.TMDbID > 0 {
		existing, err := s.seriesRepo.FindByTMDbID(ctx, in.TMDbID)
		if err == nil && existing != nil {
			return existing.ID, nil
		}
	}

	if in.SeriesDir != "" {
		existing, err := s.seriesRepo.FindByFilePath(ctx, in.SeriesDir)
		if err != nil {
			return "", fmt.Errorf("look up series by dir: %w", err)
		}
		if existing != nil {
			return existing.ID, nil
		}
	}

	title := in.Title
	if title == "" {
		title = TitleFromSeriesDir(in.SeriesDir)
	}

	series := &models.Series{
		ID:             uuid.New().String(),
		Title:          title,
		Genres:         []string{},
		FilePath:       models.NewNullString(in.SeriesDir),
		ParseStatus:    models.ParseStatusPending,
		SubtitleStatus: models.SubtitleStatusNotSearched,
	}
	if in.TMDbID > 0 {
		series.TMDbID = models.NewNullInt64(in.TMDbID)
	}
	if in.LibraryID != "" {
		series.LibraryID = models.NewNullString(in.LibraryID)
	}
	if in.Metadata != nil {
		applyMetadataItemToSeries(series, in.Metadata, in.Source)
		series.ParseStatus = models.ParseStatusSuccess
	}

	if err := s.seriesRepo.Create(ctx, series); err != nil {
		return "", fmt.Errorf("create series: %w", err)
	}

	s.logger.Info("series created", "series_id", series.ID, "title", series.Title, "dir", in.SeriesDir)
	return series.ID, nil
}

// applyMetadataItemToSeries copies a resolved provider match onto a new series row.
func applyMetadataItemToSeries(series *models.Series, item *metadata.MetadataItem, source models.MetadataSource) {
	if item.Title != "" {
		series.Title = item.Title
	}
	if item.OriginalTitle != "" {
		series.OriginalTitle = models.NewNullString(item.OriginalTitle)
	}
	if item.PosterURL != "" {
		series.PosterPath = models.NewNullString(item.PosterURL)
	}
	if item.Overview != "" {
		series.Overview = models.NewNullString(item.Overview)
	}
	if item.ReleaseDate != "" {
		series.FirstAirDate = item.ReleaseDate
	}
	if item.Rating > 0 {
		series.VoteAverage = models.NewNullFloat64(item.Rating)
	}
	if len(item.Genres) > 0 {
		series.Genres = item.Genres
	}
	series.MetadataSource = models.NewNullString(string(source))
}

// UpsertSeason finds-or-creates the season row and returns its ID.
//
// Season detail (name, poster, air date, episode count) is read off the series' own
// TMDb seasons summary when it has one. That makes this caller-agnostic: the scanner
// creates the season bare because it runs before enrichment, and the same code fills it
// in on the parse-queue path where the series arrives already resolved.
func (s *MediaIngestService) UpsertSeason(ctx context.Context, seriesID string, seasonNumber int) (string, error) {
	existing, err := s.seasonRepo.FindBySeriesAndNumber(ctx, seriesID, seasonNumber)
	if err == nil && existing != nil {
		return existing.ID, nil
	}

	season := &models.Season{
		ID:           uuid.New().String(),
		SeriesID:     seriesID,
		SeasonNumber: seasonNumber,
	}

	if series, err := s.seriesRepo.FindByID(ctx, seriesID); err == nil && series != nil {
		if seasons, err := series.GetSeasons(); err == nil {
			for i := range seasons {
				if seasons[i].SeasonNumber != seasonNumber {
					continue
				}
				summary := &seasons[i]
				if summary.ID > 0 {
					season.TMDbID = models.NewNullInt64(int64(summary.ID))
				}
				if summary.Name != "" {
					season.Name = models.NewNullString(summary.Name)
				}
				if summary.Overview != "" {
					season.Overview = models.NewNullString(summary.Overview)
				}
				if summary.PosterPath != "" {
					season.PosterPath = models.NewNullString(summary.PosterPath)
				}
				if summary.AirDate != "" {
					season.AirDate = models.NewNullString(summary.AirDate)
				}
				if summary.EpisodeCount > 0 {
					season.EpisodeCount = models.NewNullInt64(int64(summary.EpisodeCount))
				}
				break
			}
		}
	}

	if err := s.seasonRepo.Create(ctx, season); err != nil {
		return "", fmt.Errorf("create season: %w", err)
	}
	return season.ID, nil
}

// UpsertEpisode writes the episode row. episodeRepo.Upsert keys on
// (series_id, season_number, episode_number), so a re-scan of the same file is idempotent
// and a re-encode that changes the filename updates in place rather than duplicating.
func (s *MediaIngestService) UpsertEpisode(ctx context.Context, in EpisodeInput) error {
	episode := &models.Episode{
		ID:             uuid.New().String(),
		SeriesID:       in.SeriesID,
		SeasonNumber:   in.SeasonNumber,
		EpisodeNumber:  in.EpisodeNumber,
		FilePath:       models.NewNullString(in.FilePath),
		SubtitleStatus: models.SubtitleStatusNotSearched,
	}
	if in.SeasonID != "" {
		episode.SeasonID = models.NewNullString(in.SeasonID)
	}
	if in.Title != "" {
		episode.Title = models.NewNullString(in.Title)
	}

	if err := s.episodeRepo.Upsert(ctx, episode); err != nil {
		return fmt.Errorf("upsert episode: %w", err)
	}
	return nil
}

// IngestEpisodeFile is the whole series → season → episode chain for one scanned file.
func (s *MediaIngestService) IngestEpisodeFile(
	ctx context.Context,
	filePath, scanRoot, libraryID string,
	parseResult *parser.ParseResult,
) (seriesID string, err error) {
	seriesDir := SeriesDirFor(filePath, scanRoot)

	title := ""
	season, episode := 1, 0
	if parseResult != nil {
		title = parseResult.CleanedTitle
		if parseResult.Season > 0 {
			season = parseResult.Season
		}
		episode = parseResult.Episode
	}
	if title == "" {
		title = TitleFromSeriesDir(seriesDir)
	}

	seriesID, err = s.UpsertSeries(ctx, SeriesInput{
		SeriesDir: seriesDir,
		Title:     title,
		LibraryID: libraryID,
	})
	if err != nil {
		return "", err
	}

	seasonID, err := s.UpsertSeason(ctx, seriesID, season)
	if err != nil {
		return "", err
	}

	if err := s.UpsertEpisode(ctx, EpisodeInput{
		SeriesID:      seriesID,
		SeasonID:      seasonID,
		SeasonNumber:  season,
		EpisodeNumber: episode,
		FilePath:      filePath,
	}); err != nil {
		return "", err
	}

	return seriesID, nil
}
