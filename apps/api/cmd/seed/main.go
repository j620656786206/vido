// Command seed populates a LOCAL Vido test database with deterministic fixture
// data for TestSprite / manual browser testing (story: testenv-local-seed,
// resolves retro-8-TS2).
//
// It reuses the production migration runner + repository layer, so the fixture
// write path is exactly the app's write path — schema drift breaks this command
// at compile time instead of silently producing ghost rows (the bugfix-a class).
//
// Usage:
//
//	go run ./cmd/seed --data-dir ../../.vido-test-env/data [--media-root ../../.vido-test-env/media] [--reset]
//
// The dataset is intentionally small and shaped for the browse/filter/search
// journeys: zh-TW genres, decade spread (更早→2020s), matched + unmatched +
// soft-deleted movies, a CN-production movie (字幕政策), and 2 series with
// seasons/episodes. Downloads are NOT seedable here (they live in qBittorrent,
// not the DB) — the downloads page shows its empty/fail-soft state.
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/vido/api/internal/config"
	"github.com/vido/api/internal/database"
	"github.com/vido/api/internal/database/migrations"
	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/repository"
)

func main() {
	dataDir := flag.String("data-dir", "./vido-test-data", "directory for the seeded vido.db")
	mediaRoot := flag.String("media-root", "", "if set, create dummy media files here and use it in file_path/library paths")
	reset := flag.Bool("reset", false, "delete an existing database before seeding")
	flag.Parse()

	if err := run(*dataDir, *mediaRoot, *reset); err != nil {
		slog.Error("seed failed", "error", err)
		os.Exit(1)
	}
}

func run(dataDir, mediaRoot string, reset bool) error {
	ctx := context.Background()

	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return fmt.Errorf("create data dir: %w", err)
	}
	dbPath := filepath.Join(dataDir, "vido.db")
	if _, err := os.Stat(dbPath); err == nil {
		if !reset {
			return fmt.Errorf("%s already exists — pass --reset to reseed", dbPath)
		}
		for _, suffix := range []string{"", "-wal", "-shm"} {
			if err := os.Remove(dbPath + suffix); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("remove %s: %w", dbPath+suffix, err)
			}
		}
	}

	if mediaRoot == "" {
		mediaRoot = filepath.Join(dataDir, "media")
	}

	// Same DB bootstrap as cmd/api: config → Initialize → migrations Up.
	if err := os.Setenv("DB_PATH", dbPath); err != nil {
		return err
	}
	dbCfg, err := config.LoadDatabaseConfig()
	if err != nil {
		return fmt.Errorf("load db config: %w", err)
	}
	db, err := database.Initialize(dbCfg)
	if err != nil {
		return fmt.Errorf("init database: %w", err)
	}
	defer db.Close()

	runner, err := migrations.NewRunner(db.Conn())
	if err != nil {
		return fmt.Errorf("create migration runner: %w", err)
	}
	if err := runner.RegisterAll(migrations.GetAll()); err != nil {
		return fmt.Errorf("register migrations: %w", err)
	}
	if err := runner.Up(ctx); err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}

	repos := repository.NewRepositoriesWithCache(db.Conn())

	movieLib, tvLib, err := seedLibraries(ctx, repos, mediaRoot)
	if err != nil {
		return fmt.Errorf("seed libraries: %w", err)
	}
	movieCount, err := seedMovies(ctx, repos, movieLib.ID, filepath.Join(mediaRoot, "movies"))
	if err != nil {
		return fmt.Errorf("seed movies: %w", err)
	}
	seriesCount, episodeCount, err := seedSeries(ctx, repos, tvLib.ID, filepath.Join(mediaRoot, "tv"))
	if err != nil {
		return fmt.Errorf("seed series: %w", err)
	}

	// The seeded env must land on the app, not the first-run wizard: __root.tsx
	// redirects to /setup while `setup_completed` is unset, and the wizard can't
	// complete against fixture paths anyway (they're dummy files, not real
	// media mounts).
	if err := repos.Settings.SetBool(ctx, "setup_completed", true); err != nil {
		return fmt.Errorf("mark setup completed: %w", err)
	}

	slog.Info("seed complete",
		"db", dbPath,
		"libraries", 2,
		"movies", movieCount,
		"series", seriesCount,
		"episodes", episodeCount,
	)
	return nil
}

func seedLibraries(ctx context.Context, repos *repository.Repositories, mediaRoot string) (*models.MediaLibrary, *models.MediaLibrary, error) {
	movieLib := &models.MediaLibrary{
		ID:          "seed-lib-movies",
		Name:        "電影庫",
		ContentType: models.ContentTypeMovie,
		SortOrder:   1,
	}
	tvLib := &models.MediaLibrary{
		ID:          "seed-lib-tv",
		Name:        "影集庫",
		ContentType: models.ContentTypeSeries,
		SortOrder:   2,
	}
	for _, lib := range []*models.MediaLibrary{movieLib, tvLib} {
		if err := repos.MediaLibraries.Create(ctx, lib); err != nil {
			return nil, nil, err
		}
	}
	for _, p := range []*models.MediaLibraryPath{
		{ID: "seed-path-movies", LibraryID: movieLib.ID, Path: filepath.Join(mediaRoot, "movies"), Status: models.PathStatusAccessible},
		{ID: "seed-path-tv", LibraryID: tvLib.ID, Path: filepath.Join(mediaRoot, "tv"), Status: models.PathStatusAccessible},
	} {
		if err := os.MkdirAll(p.Path, 0o755); err != nil {
			return nil, nil, err
		}
		if err := repos.MediaLibraries.AddPath(ctx, p); err != nil {
			return nil, nil, err
		}
	}
	return movieLib, tvLib, nil
}

// movieFixture keeps the fixture table readable; zero values mean "unset".
type movieFixture struct {
	id            string
	title         string
	originalTitle string
	releaseDate   string
	genres        []string
	tmdbID        int64
	posterPath    string
	voteAverage   float64
	parseStatus   models.ParseStatus
	countriesJSON string
	fileName      string
	fileSizeMB    int64
	isRemoved     bool
}

func seedMovies(ctx context.Context, repos *repository.Repositories, libraryID, movieDir string) (int, error) {
	fixtures := []movieFixture{
		// Matched, decade spread 更早→2020s, zh-TW genres (bugfix-d guard: NEVER 简体).
		{id: "seed-mv-001", title: "教父", originalTitle: "The Godfather", releaseDate: "1972-03-14", genres: []string{"犯罪", "劇情"}, tmdbID: 238, posterPath: "/3bhkrj58Vtu7enYsRolD1fZdja1.jpg", voteAverage: 8.7, parseStatus: models.ParseStatusSuccess, fileName: "The.Godfather.1972.1080p.mkv", fileSizeMB: 4200},
		{id: "seed-mv-002", title: "侏羅紀公園", originalTitle: "Jurassic Park", releaseDate: "1993-06-11", genres: []string{"冒險", "科幻"}, tmdbID: 329, posterPath: "/oU7Oq2kFAAlGqbU4VoAE36g4hoI.jpg", voteAverage: 8.2, parseStatus: models.ParseStatusSuccess, fileName: "Jurassic.Park.1993.1080p.mkv", fileSizeMB: 3800},
		{id: "seed-mv-003", title: "駭客任務", originalTitle: "The Matrix", releaseDate: "1999-03-31", genres: []string{"動作", "科幻"}, tmdbID: 603, posterPath: "/f89U3ADr1oiB1s9GkdPOEpXUk5H.jpg", voteAverage: 8.7, parseStatus: models.ParseStatusSuccess, fileName: "The.Matrix.1999.1080p.mkv", fileSizeMB: 4100},
		{id: "seed-mv-004", title: "臥虎藏龍", originalTitle: "Crouching Tiger, Hidden Dragon", releaseDate: "2000-07-06", genres: []string{"動作", "劇情"}, tmdbID: 146, posterPath: "/pDJc7pHIfHXLCFPMbjBIzWyKrmt.jpg", voteAverage: 8.0, parseStatus: models.ParseStatusSuccess, fileName: "Crouching.Tiger.2000.1080p.mkv", fileSizeMB: 3500},
		{id: "seed-mv-005", title: "神隱少女", originalTitle: "千と千尋の神隠し", releaseDate: "2001-07-20", genres: []string{"動畫", "奇幻"}, tmdbID: 129, posterPath: "/39wmItIWsg5sZMyRUHLkWBcuVCM.jpg", voteAverage: 8.5, parseStatus: models.ParseStatusSuccess, fileName: "Spirited.Away.2001.1080p.mkv", fileSizeMB: 3200},
		{id: "seed-mv-006", title: "全面啟動", originalTitle: "Inception", releaseDate: "2010-07-16", genres: []string{"動作", "科幻", "懸疑"}, tmdbID: 27205, posterPath: "/oYuLEt3zVCKq57qu2F8dT7NIa6f.jpg", voteAverage: 8.4, parseStatus: models.ParseStatusSuccess, fileName: "Inception.2010.2160p.mkv", fileSizeMB: 8200},
		{id: "seed-mv-007", title: "讓子彈飛", originalTitle: "让子弹飞", releaseDate: "2010-12-16", genres: []string{"動作", "喜劇"}, tmdbID: 48317, posterPath: "/vFIHbiy55smzi50KmwlV0uhLZYm.jpg", voteAverage: 8.0, parseStatus: models.ParseStatusSuccess, countriesJSON: `[{"iso_3166_1":"CN","name":"China"}]`, fileName: "Let.The.Bullets.Fly.2010.1080p.mkv", fileSizeMB: 3900},
		{id: "seed-mv-008", title: "星際效應", originalTitle: "Interstellar", releaseDate: "2014-11-07", genres: []string{"科幻", "劇情"}, tmdbID: 157336, posterPath: "/gEU2QniE6E77NI6lCU6MxlNBvIx.jpg", voteAverage: 8.4, parseStatus: models.ParseStatusSuccess, fileName: "Interstellar.2014.2160p.mkv", fileSizeMB: 9100},
		{id: "seed-mv-009", title: "寄生上流", originalTitle: "기생충", releaseDate: "2019-05-30", genres: []string{"劇情", "驚悚"}, tmdbID: 496243, posterPath: "/7IiTTgloJzvGI1TAYymCfbfl3vT.jpg", voteAverage: 8.5, parseStatus: models.ParseStatusSuccess, fileName: "Parasite.2019.1080p.mkv", fileSizeMB: 4000},
		{id: "seed-mv-010", title: "媽的多重宇宙", originalTitle: "Everything Everywhere All at Once", releaseDate: "2022-03-24", genres: []string{"動作", "科幻", "喜劇"}, tmdbID: 545611, posterPath: "/w3LxiVYdWWRvEVdn5RYq6jIqkb1.jpg", voteAverage: 7.8, parseStatus: models.ParseStatusSuccess, fileName: "EEAAO.2022.1080p.mkv", fileSizeMB: 4300},
		{id: "seed-mv-011", title: "奧本海默", originalTitle: "Oppenheimer", releaseDate: "2023-07-19", genres: []string{"劇情", "歷史"}, tmdbID: 872585, posterPath: "/8Gxv8gSFCU0XGDykEGv7zR1n2ua.jpg", voteAverage: 8.1, parseStatus: models.ParseStatusSuccess, fileName: "Oppenheimer.2023.2160p.mkv", fileSizeMB: 11000},
		{id: "seed-mv-012", title: "沙丘:第二部", originalTitle: "Dune: Part Two", releaseDate: "2024-02-27", genres: []string{"科幻", "冒險"}, tmdbID: 693134, posterPath: "/1pdfLvkbY9ohJlCjQH2CZjjYVvJ.jpg", voteAverage: 8.2, parseStatus: models.ParseStatusSuccess, fileName: "Dune.Part.Two.2024.2160p.mkv", fileSizeMB: 10400},
		// Unmatched (parse pending / failed) — no tmdb id, no poster.
		{id: "seed-mv-101", title: "Some.Obscure.Film.2023.1080p", parseStatus: models.ParseStatusPending, fileName: "Some.Obscure.Film.2023.1080p.mkv", fileSizeMB: 2100},
		{id: "seed-mv-102", title: "[FanSub] 未知電影 (2021)", parseStatus: models.ParseStatusFailed, fileName: "[FanSub].Unknown.Movie.2021.mkv", fileSizeMB: 1800},
		{id: "seed-mv-103", title: "Home.Video.Collection.Vol1", parseStatus: models.ParseStatusPending, fileName: "Home.Video.Collection.Vol1.mkv", fileSizeMB: 900},
		// Soft-deleted — must NEVER appear in list/count (bugfix-a guard).
		{id: "seed-mv-201", title: "已刪除電影一", originalTitle: "Removed Movie One", releaseDate: "2015-01-01", genres: []string{"劇情"}, tmdbID: 900001, voteAverage: 5.0, parseStatus: models.ParseStatusSuccess, fileName: "Removed.One.2015.mkv", fileSizeMB: 700, isRemoved: true},
		{id: "seed-mv-202", title: "已刪除電影二", originalTitle: "Removed Movie Two", releaseDate: "2016-01-01", genres: []string{"喜劇"}, tmdbID: 900002, voteAverage: 5.5, parseStatus: models.ParseStatusSuccess, fileName: "Removed.Two.2016.mkv", fileSizeMB: 800, isRemoved: true},
	}

	for _, f := range fixtures {
		filePath := filepath.Join(movieDir, f.fileName)
		if err := writeDummyFile(filePath); err != nil {
			return 0, err
		}
		m := &models.Movie{
			ID:          f.id,
			Title:       f.title,
			ReleaseDate: f.releaseDate,
			Genres:      f.genres,
			ParseStatus: f.parseStatus,
			FilePath:    models.NewNullString(filePath),
			FileSize:    models.NewNullInt64(f.fileSizeMB * 1024 * 1024),
			IsRemoved:   f.isRemoved,
			LibraryID:   models.NewNullString(libraryID),
		}
		if f.originalTitle != "" {
			m.OriginalTitle = models.NewNullString(f.originalTitle)
		}
		if f.tmdbID != 0 {
			m.TMDbID = models.NewNullInt64(f.tmdbID)
			m.MetadataSource = models.NewNullString("tmdb")
		}
		if f.posterPath != "" {
			m.PosterPath = models.NewNullString(f.posterPath)
		}
		if f.voteAverage != 0 {
			m.VoteAverage = models.NewNullFloat64(f.voteAverage)
		}
		if f.countriesJSON != "" {
			m.ProductionCountriesJSON = models.NewNullString(f.countriesJSON)
		}
		if err := repos.Movies.Create(ctx, m); err != nil {
			return 0, fmt.Errorf("movie %s: %w", f.id, err)
		}
	}
	return len(fixtures), nil
}

func seedSeries(ctx context.Context, repos *repository.Repositories, libraryID, tvDir string) (int, int, error) {
	type seriesFixture struct {
		id           string
		title        string
		original     string
		firstAirDate string
		genres       []string
		tmdbID       int64
		posterPath   string
		parseStatus  models.ParseStatus
		seasons      int
		epsPerSeason int
	}
	fixtures := []seriesFixture{
		{id: "seed-sr-001", title: "進擊的巨人", original: "進撃の巨人", firstAirDate: "2013-04-07", genres: []string{"動畫", "動作"}, tmdbID: 1429, posterPath: "/hTP1DtLGFamjfu8WqjnuQdP1n4i.jpg", parseStatus: models.ParseStatusSuccess, seasons: 2, epsPerSeason: 3},
		{id: "seed-sr-002", title: "怪奇物語", original: "Stranger Things", firstAirDate: "2016-07-15", genres: []string{"科幻", "懸疑"}, tmdbID: 66732, posterPath: "/49WJfeN0moxb9IPfGn8AIqMGskD.jpg", parseStatus: models.ParseStatusSuccess, seasons: 1, epsPerSeason: 4},
		{id: "seed-sr-101", title: "Unknown.Show.S01", firstAirDate: "", genres: nil, parseStatus: models.ParseStatusPending, seasons: 1, epsPerSeason: 2},
	}

	episodeTotal := 0
	for _, f := range fixtures {
		seriesDir := filepath.Join(tvDir, f.title)
		if err := os.MkdirAll(seriesDir, 0o755); err != nil {
			return 0, 0, err
		}
		s := &models.Series{
			ID:           f.id,
			Title:        f.title,
			FirstAirDate: f.firstAirDate,
			Genres:       f.genres,
			ParseStatus:  f.parseStatus,
			FilePath:     models.NewNullString(seriesDir),
			LibraryID:    models.NewNullString(libraryID),
		}
		if f.original != "" {
			s.OriginalTitle = models.NewNullString(f.original)
		}
		if f.tmdbID != 0 {
			s.TMDbID = models.NewNullInt64(f.tmdbID)
		}
		if f.posterPath != "" {
			s.PosterPath = models.NewNullString(f.posterPath)
		}
		if err := repos.Series.Create(ctx, s); err != nil {
			return 0, 0, fmt.Errorf("series %s: %w", f.id, err)
		}

		for sn := 1; sn <= f.seasons; sn++ {
			seasonID := fmt.Sprintf("%s-s%02d", f.id, sn)
			season := &models.Season{
				ID:           seasonID,
				SeriesID:     f.id,
				SeasonNumber: sn,
				Name:         models.NewNullString(fmt.Sprintf("第 %d 季", sn)),
				EpisodeCount: models.NewNullInt64(int64(f.epsPerSeason)),
			}
			if err := repos.Seasons.Create(ctx, season); err != nil {
				return 0, 0, fmt.Errorf("season %s: %w", seasonID, err)
			}
			for en := 1; en <= f.epsPerSeason; en++ {
				epPath := filepath.Join(seriesDir, fmt.Sprintf("S%02dE%02d.mkv", sn, en))
				if err := writeDummyFile(epPath); err != nil {
					return 0, 0, err
				}
				ep := &models.Episode{
					ID:            fmt.Sprintf("%s-s%02de%02d", f.id, sn, en),
					SeriesID:      f.id,
					SeasonID:      models.NewNullString(seasonID),
					SeasonNumber:  sn,
					EpisodeNumber: en,
					Title:         models.NewNullString(fmt.Sprintf("第 %d 集", en)),
				}
				if err := repos.Episodes.Create(ctx, ep); err != nil {
					return 0, 0, fmt.Errorf("episode %s: %w", ep.ID, err)
				}
				episodeTotal++
			}
		}
	}
	return len(fixtures), episodeTotal, nil
}

// writeDummyFile creates a tiny placeholder so seeded file_path values point at
// real files (keeps library-path health checks and any accidental re-scan
// consistent with the DB instead of manufacturing ghost rows).
func writeDummyFile(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte("vido-seed-dummy"), 0o644)
}
