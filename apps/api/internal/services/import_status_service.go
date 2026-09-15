package services

import (
	"context"
	"fmt"
	"log/slog"
	"path"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/plugins"
)

// ImportState is where a finished download stands on its way into the
// library (dl-import-1 [@contract-v1], consumed by dl-import-2).
type ImportState string

const (
	// ImportStateInLibrary — Sonarr/Radarr imported it and Vido has it: the
	// movie (any non-removed copy with a file), or every episode the torrent
	// imported.
	ImportStateInLibrary ImportState = "in_library"
	// ImportStateAwaitingScan — Sonarr/Radarr imported it, but Vido does not
	// have it. True whether or not the next scan will find it.
	ImportStateAwaitingScan ImportState = "awaiting_scan"
	// ImportStateAwaitingImport — grabbed, and nothing has happened since.
	ImportStateAwaitingImport ImportState = "awaiting_import"
	// ImportStateImportFailed — Sonarr/Radarr marked the download failed.
	ImportStateImportFailed ImportState = "import_failed"
	// ImportStateImportIgnored — someone told Sonarr/Radarr to ignore it.
	ImportStateImportIgnored ImportState = "import_ignored"
)

// DownloadImportStatus is the import status of one torrent.
type DownloadImportStatus struct {
	State ImportState `json:"state"`
	// Source is the plugin that knows the torrent: "radarr" | "sonarr".
	Source    string `json:"source"`
	MediaType string `json:"media_type"` // "movie" | "tv"
	// MediaID is the Vido movie/series id when Vido has the title — a link
	// target, set for a partly scanned season pack too.
	MediaID string `json:"media_id,omitempty"`
	// Sonarr, once imported: episodes the torrent imported, and how many of
	// those Vido has.
	EpisodesImported  *int `json:"episodes_imported,omitempty"`
	EpisodesInLibrary *int `json:"episodes_in_library,omitempty"`
}

// ImportStatusServiceInterface resolves import status for a page of torrents.
type ImportStatusServiceInterface interface {
	// Resolve returns a status per hash it can vouch for; everything else is
	// absent. It never fails the caller and never blocks it for long.
	Resolve(ctx context.Context, hashes []string) map[string]*DownloadImportStatus
}

// ImportHistoryClients is the slice of plugins.Manager the service needs.
// Health is an in-memory read; GetClient decrypts the API key, so it is only
// called when a refresh is actually due.
type ImportHistoryClients interface {
	Health(name string) plugins.PluginHealth
	GetClient(ctx context.Context, name string) (plugins.DVRPlugin, error)
}

// ImportMovieFinder finds the library copy of a movie: not removed, with a file.
type ImportMovieFinder interface {
	FindWithFileByTMDbID(ctx context.Context, tmdbID int64) (*models.Movie, error)
}

// ImportSeriesFinder finds a library series that has not been removed.
type ImportSeriesFinder interface {
	FindActiveByTMDbID(ctx context.Context, tmdbID int64) (*models.Series, error)
}

// ImportEpisodeFinder lists a series' episodes.
type ImportEpisodeFinder interface {
	FindBySeriesID(ctx context.Context, seriesID string) ([]models.Episode, error)
}

// Rule 27 pillars ② cache and ③ degrade. The history snapshot per plugin is
// served stale-while-revalidate: a request never waits for a refresh once a
// snapshot exists, and waits at most importFirstFetchWait for the first one.
// Import events never un-happen, so a stale snapshot can only report something
// late, never wrongly — up to importStaleMax, after which it says nothing.
const (
	importHistoryTTL     = time.Minute
	importFirstFetchWait = 3 * time.Second
	importFetchTimeout   = 90 * time.Second
	importStaleMax       = 30 * time.Minute
	importBackoffBase    = time.Minute
	importBackoffMax     = 5 * time.Minute
)

// historyIndex maps an upper-case download id to its records, newest first.
type historyIndex map[string][]plugins.ImportHistoryRecord

type pluginHistory struct {
	index     historyIndex
	client    plugins.DVRPlugin // the client that produced index
	fetchedAt time.Time
	failures  int
	failedAt  time.Time
}

// ImportStatusService answers "did this finished download make it into the
// library?" from Sonarr/Radarr's own history, because they rename files on
// import — matching by file name finds 31 of 717 files on a real install.
type ImportStatusService struct {
	clients  ImportHistoryClients
	movies   ImportMovieFinder
	series   ImportSeriesFinder
	episodes ImportEpisodeFinder
	now      func() time.Time
	// firstFetchWait bounds how long a request waits when no snapshot exists yet.
	firstFetchWait time.Duration

	mu       sync.Mutex
	history  map[string]*pluginHistory
	inflight map[string]chan struct{}
}

var _ ImportStatusServiceInterface = (*ImportStatusService)(nil)

// NewImportStatusService creates the service. clients is normally the
// plugins.Manager.
func NewImportStatusService(
	clients ImportHistoryClients,
	movies ImportMovieFinder,
	series ImportSeriesFinder,
	episodes ImportEpisodeFinder,
) *ImportStatusService {
	return &ImportStatusService{
		clients:        clients,
		movies:         movies,
		series:         series,
		episodes:       episodes,
		now:            time.Now,
		firstFetchWait: importFirstFetchWait,
		history:        map[string]*pluginHistory{},
		inflight:       map[string]chan struct{}{},
	}
}

// Resolve implements ImportStatusServiceInterface.
func (s *ImportStatusService) Resolve(ctx context.Context, hashes []string) map[string]*DownloadImportStatus {
	if len(hashes) == 0 {
		return nil
	}

	radarr := s.historyFor(ctx, "radarr")
	sonarr := s.historyFor(ctx, "sonarr")
	if radarr == nil && sonarr == nil {
		return nil
	}

	lookups := &importLookups{
		service:  s,
		movies:   map[int64]*models.Movie{},
		series:   map[int64]*models.Series{},
		episodes: map[string][]models.Episode{},
	}
	out := make(map[string]*DownloadImportStatus)
	for _, hash := range hashes {
		key := strings.ToUpper(hash)
		movieRecords, seriesRecords := radarr[key], sonarr[key]

		var status *DownloadImportStatus
		switch {
		// Whichever plugin imported it speaks; otherwise whichever knows it.
		case hasImport(movieRecords):
			status = lookups.movieStatus(ctx, movieRecords)
		case hasImport(seriesRecords):
			status = lookups.seriesStatus(ctx, seriesRecords)
		case len(movieRecords) > 0:
			status = lookups.movieStatus(ctx, movieRecords)
		case len(seriesRecords) > 0:
			status = lookups.seriesStatus(ctx, seriesRecords)
		}
		if status != nil {
			out[hash] = status
		}
	}
	return out
}

// historyFor returns the plugin's download-indexed history without making
// the request wait on *arr, starting a background refresh when one is due.
func (s *ImportStatusService) historyFor(ctx context.Context, name string) historyIndex {
	health := s.clients.Health(name)
	now := s.now()

	s.mu.Lock()
	if health.Status == plugins.HealthStatusUnconfigured {
		// Switched off or not set up: nothing it once said applies any more.
		delete(s.history, name)
		s.mu.Unlock()
		return nil
	}
	state := s.history[name]
	var index historyIndex
	if state != nil && state.index != nil && now.Sub(state.fetchedAt) < importStaleMax {
		index = state.index
	}
	due := health.Status == plugins.HealthStatusHealthy &&
		(state == nil || state.index == nil || now.Sub(state.fetchedAt) >= importHistoryTTL) &&
		(state == nil || state.failures == 0 || now.Sub(state.failedAt) >= importBackoff(state.failures))
	s.mu.Unlock()

	if !due {
		return index
	}
	done := s.refresh(name)
	if index != nil {
		return index // stale-while-revalidate
	}

	timer := time.NewTimer(s.firstFetchWait)
	defer timer.Stop()
	select {
	case <-done:
		s.mu.Lock()
		defer s.mu.Unlock()
		if st := s.history[name]; st != nil {
			return st.index
		}
		return nil
	case <-ctx.Done():
		return nil
	case <-timer.C:
		return nil // the fetch keeps going; the next request gets it
	}
}

// refresh starts (or joins) the plugin's history fetch. It runs on its own
// context: a request that gives up must not cancel the fetch others wait on.
func (s *ImportStatusService) refresh(name string) <-chan struct{} {
	s.mu.Lock()
	if done, ok := s.inflight[name]; ok {
		s.mu.Unlock()
		return done
	}
	done := make(chan struct{})
	s.inflight[name] = done
	s.mu.Unlock()

	go func() {
		defer func() {
			s.mu.Lock()
			delete(s.inflight, name)
			s.mu.Unlock()
			close(done)
		}()
		ctx, cancel := context.WithTimeout(context.Background(), importFetchTimeout)
		defer cancel()
		s.fetch(ctx, name)
	}()
	return done
}

func (s *ImportStatusService) fetch(ctx context.Context, name string) {
	client, err := s.clients.GetClient(ctx, name)
	if err == nil {
		reader, ok := client.(plugins.ImportHistoryReader)
		if !ok {
			return
		}
		records, fetchErr := reader.GetImportHistory(ctx)
		if fetchErr == nil {
			s.mu.Lock()
			if prev := s.history[name]; prev != nil && prev.failures > 0 {
				slog.Info("import status: history readable again", "plugin", name)
			}
			s.history[name] = &pluginHistory{index: buildHistoryIndex(records), client: client, fetchedAt: s.now()}
			s.mu.Unlock()
			return
		}
		err = fetchErr
	}

	s.mu.Lock()
	state := s.history[name]
	if state == nil {
		state = &pluginHistory{}
		s.history[name] = state
	}
	if client != nil && state.client != nil && state.client != client {
		// The config now points at another server; the old index describes
		// somebody else's downloads.
		state.index, state.client = nil, nil
	}
	state.failures++
	state.failedAt = s.now()
	failures := state.failures
	s.mu.Unlock()

	if failures == 1 {
		slog.Warn("import status: cannot read history — downloads show no new import status until it recovers",
			"plugin", name, "error", err)
	} else {
		slog.Debug("import status: history still unreadable", "plugin", name, "failures", failures, "error", err)
	}
}

// importBackoff is 1, 2, 4 … minutes after consecutive failures, capped at 5.
func importBackoff(failures int) time.Duration {
	delay := importBackoffBase
	for i := 1; i < failures && delay < importBackoffMax; i++ {
		delay *= 2
	}
	if delay > importBackoffMax {
		return importBackoffMax
	}
	return delay
}

func buildHistoryIndex(records []plugins.ImportHistoryRecord) historyIndex {
	index := make(historyIndex)
	for _, record := range records {
		key := strings.ToUpper(record.DownloadID)
		if key == "" {
			continue
		}
		index[key] = append(index[key], record)
	}
	for key := range index {
		records := index[key]
		sort.SliceStable(records, func(i, j int) bool { return records[i].Date.After(records[j].Date) })
	}
	return index
}

func hasImport(records []plugins.ImportHistoryRecord) bool {
	for _, record := range records {
		if record.EventType == plugins.HistoryEventImported {
			return true
		}
	}
	return false
}

// importLookups caches library reads for one Resolve call: a page of season
// packs from one show reads that show once, not once per torrent × episode.
type importLookups struct {
	service  *ImportStatusService
	movies   map[int64]*models.Movie
	series   map[int64]*models.Series
	episodes map[string][]models.Episode
}

func (l *importLookups) movie(ctx context.Context, tmdbID int64) *models.Movie {
	if movie, ok := l.movies[tmdbID]; ok {
		return movie
	}
	movie, err := l.service.movies.FindWithFileByTMDbID(ctx, tmdbID)
	if err != nil {
		slog.Debug("import status: movie lookup failed", "tmdb_id", tmdbID, "error", err)
		movie = nil
	}
	l.movies[tmdbID] = movie
	return movie
}

func (l *importLookups) show(ctx context.Context, tmdbID int64) *models.Series {
	if series, ok := l.series[tmdbID]; ok {
		return series
	}
	series, err := l.service.series.FindActiveByTMDbID(ctx, tmdbID)
	if err != nil {
		slog.Debug("import status: series lookup failed", "tmdb_id", tmdbID, "error", err)
		series = nil
	}
	l.series[tmdbID] = series
	return series
}

func (l *importLookups) seriesEpisodes(ctx context.Context, seriesID string) []models.Episode {
	if episodes, ok := l.episodes[seriesID]; ok {
		return episodes
	}
	episodes, err := l.service.episodes.FindBySeriesID(ctx, seriesID)
	if err != nil {
		slog.Debug("import status: episode lookup failed", "series_id", seriesID, "error", err)
		episodes = nil
	}
	l.episodes[seriesID] = episodes
	return episodes
}

// notImported names the state of a download with no import event, from its
// newest event.
func notImported(records []plugins.ImportHistoryRecord) ImportState {
	switch records[0].EventType {
	case plugins.HistoryEventDownloadFailed:
		return ImportStateImportFailed
	case plugins.HistoryEventDownloadIgnored:
		return ImportStateImportIgnored
	default:
		return ImportStateAwaitingImport
	}
}

// importedTMDbID is the TMDb id of the newest import that carries one.
func importedTMDbID(records []plugins.ImportHistoryRecord) int64 {
	for _, record := range records {
		if record.EventType == plugins.HistoryEventImported && record.TMDbID != 0 {
			return record.TMDbID
		}
	}
	return 0
}

func (l *importLookups) movieStatus(ctx context.Context, records []plugins.ImportHistoryRecord) *DownloadImportStatus {
	status := &DownloadImportStatus{Source: "radarr", MediaType: "movie"}
	if !hasImport(records) {
		status.State = notImported(records)
		return status
	}
	tmdbID := importedTMDbID(records)
	if tmdbID == 0 {
		return nil // no way to check the library — say nothing rather than guess
	}
	if movie := l.movie(ctx, tmdbID); movie != nil {
		status.State = ImportStateInLibrary
		status.MediaID = movie.ID
		return status
	}
	status.State = ImportStateAwaitingScan
	return status
}

func (l *importLookups) seriesStatus(ctx context.Context, records []plugins.ImportHistoryRecord) *DownloadImportStatus {
	status := &DownloadImportStatus{Source: "sonarr", MediaType: "tv"}
	if !hasImport(records) {
		status.State = notImported(records)
		return status
	}
	tmdbID := importedTMDbID(records)
	if tmdbID == 0 {
		return nil
	}

	// Group imported episodes by the file they landed in: a multi-episode file
	// (S01E01E02) is two import events but one library file.
	type file struct {
		path     string
		episodes []string // "season:episode", or the file itself when *arr sent no numbers
	}
	files := map[string]*file{}
	episodeIDs := map[string]struct{}{}
	for _, record := range records {
		if record.EventType != plugins.HistoryEventImported {
			continue
		}
		episodeID := fmt.Sprintf("%d:%d", record.SeasonNumber, record.EpisodeNumber)
		if record.SeasonNumber == 0 && record.EpisodeNumber == 0 {
			episodeID = "file:" + record.ImportedPath
		}
		key := record.ImportedPath
		if key == "" {
			key = "episode:" + episodeID
		}
		if files[key] == nil {
			files[key] = &file{path: record.ImportedPath}
		}
		files[key].episodes = append(files[key].episodes, episodeID)
		episodeIDs[episodeID] = struct{}{}
	}

	total, present := len(episodeIDs), 0
	status.EpisodesImported = &total
	status.EpisodesInLibrary = &present
	status.State = ImportStateAwaitingScan

	series := l.show(ctx, tmdbID)
	if series == nil {
		return status
	}
	status.MediaID = series.ID

	withFile := map[string]bool{}
	fileNames := map[string]bool{}
	for _, episode := range l.seriesEpisodes(ctx, series.ID) {
		if !hasFile(episode.FilePath) {
			continue
		}
		withFile[fmt.Sprintf("%d:%d", episode.SeasonNumber, episode.EpisodeNumber)] = true
		fileNames[path.Base(episode.FilePath.String)] = true
	}

	counted := map[string]bool{}
	for _, f := range files {
		// Sonarr renamed the file on import, so the library copy has the same
		// name — that also covers numbering Vido stores differently (absolute).
		inLibrary := f.path != "" && fileNames[path.Base(f.path)]
		for _, episodeID := range f.episodes {
			if withFile[episodeID] {
				inLibrary = true
			}
		}
		if !inLibrary {
			continue
		}
		for _, episodeID := range f.episodes {
			if !counted[episodeID] {
				counted[episodeID] = true
				present++
			}
		}
	}
	if total > 0 && present == total {
		status.State = ImportStateInLibrary
	}
	return status
}

func hasFile(p models.NullString) bool {
	return p.Valid && p.String != ""
}
