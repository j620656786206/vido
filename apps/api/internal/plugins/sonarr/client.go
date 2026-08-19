// Package sonarr implements the plugins.DVRPlugin interface against the
// Sonarr v4 API (served under /api/v3) — Story 13-4b AC #2, mirroring the
// Radarr client structure.
//
// THE TVDB GOTCHA (AC #1, web-verified 2026-07-04): Sonarr's POST /series
// hard-requires tvdbId (Sonarr#7565) and series/lookup officially supports
// only name and `tvdb:{id}` terms — so AddSeries resolves TMDB→TVDB via the
// injected TVDBResolver (the existing shared TMDb client underneath; Rule 27
// reuse) before talking to Sonarr. A title with no TVDB entry is a typed
// terminal DVR_TVDB_NOT_FOUND.
package sonarr

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"golang.org/x/time/rate"

	"github.com/vido/api/internal/plugins"
)

// requestsPerSecond / burst — same LAN-local observed-safe ceiling as the
// Radarr client (Rule 27 ①).
const (
	requestsPerSecond = 10
	burstSize         = 10
)

// maxQueuePages bounds the GetQueue pagination loop (Radarr-client parity).
const maxQueuePages = 20

// minimumMajorVersion gates TestConnection: Sonarr v3 requires
// languageProfileId on series adds (removed in v4) — half-working v3 support
// is rejected loudly (AC #2 version note).
const minimumMajorVersion = 4

// TVDBResolver resolves a TMDB tv id to its TVDB id. 0 means the title has
// no TVDB entry. Implemented in main.go over services.TMDbService — defined
// here so this package needs no services import (Rule 19 direction).
type TVDBResolver interface {
	ResolveTVDBID(ctx context.Context, tmdbID int64) (int64, error)
}

// TVDBResolverFunc adapts a function to TVDBResolver.
type TVDBResolverFunc func(ctx context.Context, tmdbID int64) (int64, error)

// ResolveTVDBID implements TVDBResolver.
func (f TVDBResolverFunc) ResolveTVDBID(ctx context.Context, tmdbID int64) (int64, error) {
	return f(ctx, tmdbID)
}

// Client communicates with the Sonarr v4 API. One reused http.Client
// (Rule 14) with a 10s timeout, one *rate.Limiter for process life.
type Client struct {
	config     plugins.PluginConfig
	httpClient *http.Client
	limiter    *rate.Limiter
	resolver   TVDBResolver
}

// Compile-time interface verification.
var (
	_ plugins.DVRPlugin     = (*Client)(nil)
	_ plugins.ProfileLister = (*Client)(nil)
)

// NewClient creates a new Sonarr API client for the given config.
func NewClient(config plugins.PluginConfig, resolver TVDBResolver) *Client {
	return &Client{
		config: config,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		limiter:  rate.NewLimiter(rate.Limit(requestsPerSecond), burstSize),
		resolver: resolver,
	}
}

// Name returns the plugin name.
func (c *Client) Name() string {
	return "sonarr"
}

// buildURL constructs the full API v3 URL for the given path.
func (c *Client) buildURL(path string) string {
	return strings.TrimSuffix(c.config.URL, "/") + "/api/v3" + path
}

// systemStatus is the subset of GET /system/status we validate.
type systemStatus struct {
	Version string `json:"version"`
}

// TestConnection verifies connectivity using the GIVEN config and enforces
// the v4 gate: 200 + parseable version ≥ 4 = pass; 401 → DVR_AUTH_FAILED;
// v3 → DVR_TEST_FAILED with a clear zh-TW upgrade message.
func (c *Client) TestConnection(ctx context.Context, config plugins.PluginConfig) error {
	statusURL := strings.TrimSuffix(config.URL, "/") + "/api/v3/system/status"
	body, err := c.doRequest(ctx, http.MethodGet, statusURL, config.APIKey, nil)
	if err != nil {
		return err
	}

	var status systemStatus
	if err := json.Unmarshal(body, &status); err != nil || status.Version == "" {
		return &plugins.PluginError{
			Code:    plugins.ErrCodeConnectionFailed,
			Message: "sonarr system/status returned no parseable version",
			Cause:   err,
		}
	}

	major, err := strconv.Atoi(strings.SplitN(status.Version, ".", 2)[0])
	if err != nil || major < minimumMajorVersion {
		return &plugins.PluginError{
			Code: plugins.ErrCodeTestFailed,
			Message: fmt.Sprintf(
				"需要 Sonarr v4（偵測到 %s）— v3 series adds require the removed languageProfileId",
				status.Version),
		}
	}
	return nil
}

// AddMovie is not supported — Sonarr is series-only; movie routing is the
// Radarr plugin's job (13-4a).
func (c *Client) AddMovie(ctx context.Context, tmdbID int64, opts plugins.AddOptions) (int64, error) {
	return 0, &plugins.PluginError{
		Code:    plugins.ErrCodeNotSupported,
		Message: "sonarr is series-only; movie requests need the radarr plugin",
	}
}

// addSeriesResponse is the subset of the created series resource we consume.
type addSeriesResponse struct {
	ID int64 `json:"id"`
}

// AddSeries adds a series by TMDB id — [@contract-v2] (13-2a bump of the
// 13-4b whole-series [@contract-v1]): with an empty selection the flow is
// byte-identical to v1 (whole series, `monitor:"all"`, searchForMissing);
// with opts.Seasons/Episodes it becomes selection-aware — only selected
// seasons monitored, `monitor:"none"` at add time, explicit episode
// monitoring + SeasonSearch/EpisodeSearch commands afterwards.
//
// Resolution flow is unchanged: resolve TMDB→TVDB → lookup `tvdb:{id}` →
// POST the lookup-shaped object enriched with config fields. Returns Sonarr's
// own series id.
func (c *Client) AddSeries(ctx context.Context, tmdbID int64, opts plugins.AddOptions) (int64, error) {
	tvdbID, err := c.resolver.ResolveTVDBID(ctx, tmdbID)
	if err != nil {
		return 0, &plugins.PluginError{
			Code:    plugins.ErrCodeConnectionFailed,
			Message: fmt.Sprintf("resolve tvdb id for tmdb %d", tmdbID),
			Cause:   err,
		}
	}
	if tvdbID == 0 {
		return 0, &plugins.PluginError{
			Code:    plugins.ErrCodeTVDBNotFound,
			Message: fmt.Sprintf("tmdb series %d has no TVDB entry; sonarr cannot search it", tmdbID),
		}
	}

	series, err := c.lookupSeries(ctx, tvdbID)
	if err != nil {
		return 0, err
	}

	partial := opts.Partial()
	selectedSeasons := selectedSeasonSet(opts)

	// Enrich the lookup object in place — POSTing a hand-built minimal body
	// is the classic source of Sonarr 400s (Dev Notes idiom).
	series["qualityProfileId"] = opts.QualityProfileID
	series["rootFolderPath"] = opts.RootFolderPath
	series["monitored"] = true
	if seasons, ok := series["seasons"].([]any); ok {
		for _, s := range seasons {
			if season, ok := s.(map[string]any); ok {
				monitored := true
				if partial {
					// Sonarr season numbers arrive as JSON float64.
					number, _ := season["seasonNumber"].(float64)
					_, monitored = selectedSeasons[int(number)]
				}
				season["monitored"] = monitored
			}
		}
	}
	if partial {
		// Episode monitoring is set EXPLICITLY below rather than trusted to a
		// monitor strategy: `none` guarantees a clean slate, then the selected
		// episodes are promoted — deterministic regardless of how Sonarr's
		// monitor options interact with season flags. Search fires via
		// explicit commands, never searchForMissingEpisodes (which is a
		// whole-series lever and would over-search).
		series["addOptions"] = map[string]any{
			"monitor":                  "none",
			"searchForMissingEpisodes": false,
		}
	} else {
		series["addOptions"] = map[string]any{
			"monitor":                  "all",
			"searchForMissingEpisodes": opts.SearchNow,
		}
	}

	payload, err := json.Marshal(series)
	if err != nil {
		return 0, &plugins.PluginError{Code: plugins.ErrCodeAddFailed, Message: "encode add-series request", Cause: err}
	}

	body, err := c.doRequest(ctx, http.MethodPost, c.buildURL("/series"), c.config.APIKey, payload)
	if err != nil {
		return 0, err
	}

	var created addSeriesResponse
	if err := json.Unmarshal(body, &created); err != nil || created.ID == 0 {
		return 0, &plugins.PluginError{
			Code:    plugins.ErrCodeAddFailed,
			Message: "sonarr add-series response carried no series id",
			Cause:   err,
		}
	}

	if partial {
		if err := c.applySelection(ctx, created.ID, opts); err != nil {
			// The series row exists in Sonarr but the selection did not fully
			// land — an honest DVR_ADD_FAILED (request stays pending for the
			// 13-3a reconcile). A subsequent retry re-hits POST /series and
			// gets already-exists: the KNOWN class tracked as
			// disc-2026-07-arr-already-exists-loop (extended by 13-2a's ⚖️ A
			// ruling to own the existing-series update path).
			return 0, err
		}
	}
	return created.ID, nil
}

// selectedSeasonSet unions fully-selected seasons with the seasons that carry
// individual episode selections — the set whose season flags stay monitored.
func selectedSeasonSet(opts plugins.AddOptions) map[int]struct{} {
	set := make(map[int]struct{}, len(opts.Seasons)+len(opts.Episodes))
	for _, n := range opts.Seasons {
		set[n] = struct{}{}
	}
	for n := range opts.Episodes {
		set[n] = struct{}{}
	}
	return set
}

// sonarrEpisode is the subset of GET /api/v3/episode we consume.
type sonarrEpisode struct {
	ID            int64 `json:"id"`
	SeasonNumber  int   `json:"seasonNumber"`
	EpisodeNumber int   `json:"episodeNumber"`
}

// applySelection promotes the selected episodes to monitored and triggers the
// per-selection searches (13-2a AC #4): one GET /episode per selected season,
// one PUT /episode/monitor for ALL selected episode ids, then SeasonSearch
// per fully-selected season and a single EpisodeSearch for listed episodes
// (only when SearchNow).
//
// NUMBERING CAVEAT (recorded, not solved here): the request was validated
// against TMDB numbering while Sonarr speaks TVDB numbering; for the rare
// show where they diverge, a selected number missing from Sonarr's list is
// skipped with a warning — an entirely-unresolvable season fails loudly.
func (c *Client) applySelection(ctx context.Context, seriesID int64, opts plugins.AddOptions) error {
	var monitorIDs []int64 // every selected episode id (whole seasons + listed)
	var episodeIDs []int64 // listed-episode ids only (EpisodeSearch payload)

	fetch := func(season int) ([]sonarrEpisode, error) {
		body, err := c.doRequest(ctx, http.MethodGet,
			c.buildURL(fmt.Sprintf("/episode?seriesId=%d&seasonNumber=%d", seriesID, season)),
			c.config.APIKey, nil)
		if err != nil {
			return nil, err
		}
		var episodes []sonarrEpisode
		if err := json.Unmarshal(body, &episodes); err != nil {
			return nil, &plugins.PluginError{
				Code:    plugins.ErrCodeAddFailed,
				Message: fmt.Sprintf("sonarr episode list for season %d is not parseable", season),
				Cause:   err,
			}
		}
		return episodes, nil
	}

	for _, season := range opts.Seasons {
		episodes, err := fetch(season)
		if err != nil {
			return err
		}
		if len(episodes) == 0 {
			// CR 13-2a H2: symmetry with the listed-episodes branch below. A
			// season Sonarr knows nothing about (numbering mismatch, or the
			// season not yet populated) would otherwise monitor nothing and
			// fire a no-op SeasonSearch while the request happily reports
			// 搜尋中 forever.
			return &plugins.PluginError{
				Code:    plugins.ErrCodeAddFailed,
				Message: fmt.Sprintf("sonarr lists no episodes for season %d (numbering mismatch?)", season),
			}
		}
		for _, ep := range episodes {
			monitorIDs = append(monitorIDs, ep.ID)
		}
	}
	for season, numbers := range opts.Episodes {
		episodes, err := fetch(season)
		if err != nil {
			return err
		}
		byNumber := make(map[int]int64, len(episodes))
		for _, ep := range episodes {
			byNumber[ep.EpisodeNumber] = ep.ID
		}
		resolved := 0
		for _, n := range numbers {
			if id, ok := byNumber[n]; ok {
				monitorIDs = append(monitorIDs, id)
				episodeIDs = append(episodeIDs, id)
				resolved++
			} else {
				slog.Warn("Sonarr has no matching episode for the selected number — skipping",
					"series_id", seriesID, "season", season, "episode", n)
			}
		}
		if resolved == 0 {
			return &plugins.PluginError{
				Code:    plugins.ErrCodeAddFailed,
				Message: fmt.Sprintf("none of the selected episodes of season %d exist in sonarr (numbering mismatch?)", season),
			}
		}
	}

	if len(monitorIDs) > 0 {
		payload, err := json.Marshal(map[string]any{"episodeIds": monitorIDs, "monitored": true})
		if err != nil {
			return &plugins.PluginError{Code: plugins.ErrCodeAddFailed, Message: "encode episode-monitor request", Cause: err}
		}
		if _, err := c.doRequest(ctx, http.MethodPut, c.buildURL("/episode/monitor"), c.config.APIKey, payload); err != nil {
			return err
		}
	}

	if !opts.SearchNow {
		return nil
	}
	for _, season := range opts.Seasons {
		if err := c.postCommand(ctx, map[string]any{
			"name": "SeasonSearch", "seriesId": seriesID, "seasonNumber": season,
		}); err != nil {
			return err
		}
	}
	if len(episodeIDs) > 0 {
		if err := c.postCommand(ctx, map[string]any{
			"name": "EpisodeSearch", "episodeIds": episodeIDs,
		}); err != nil {
			return err
		}
	}
	return nil
}

// postCommand POSTs one Sonarr command payload.
func (c *Client) postCommand(ctx context.Context, command map[string]any) error {
	payload, err := json.Marshal(command)
	if err != nil {
		return &plugins.PluginError{Code: plugins.ErrCodeAddFailed, Message: "encode sonarr command", Cause: err}
	}
	_, err = c.doRequest(ctx, http.MethodPost, c.buildURL("/command"), c.config.APIKey, payload)
	return err
}

// lookupSeries fetches the full lookup-shaped series object for a TVDB id.
func (c *Client) lookupSeries(ctx context.Context, tvdbID int64) (map[string]any, error) {
	lookupURL := c.buildURL("/series/lookup?term=" + url.QueryEscape(fmt.Sprintf("tvdb:%d", tvdbID)))
	body, err := c.doRequest(ctx, http.MethodGet, lookupURL, c.config.APIKey, nil)
	if err != nil {
		return nil, err
	}

	var results []map[string]any
	if err := json.Unmarshal(body, &results); err != nil {
		return nil, &plugins.PluginError{
			Code:    plugins.ErrCodeConnectionFailed,
			Message: "sonarr series/lookup response is not parseable",
			Cause:   err,
		}
	}
	if len(results) == 0 {
		return nil, &plugins.PluginError{
			Code:    plugins.ErrCodeAddFailed,
			Message: fmt.Sprintf("sonarr lookup returned no results for tvdb:%d", tvdbID),
		}
	}
	return results[0], nil
}

// queuePage is the paginated GET /queue envelope.
type queuePage struct {
	Page         int           `json:"page"`
	PageSize     int           `json:"pageSize"`
	TotalRecords int           `json:"totalRecords"`
	Records      []queueRecord `json:"records"`
}

// queueRecord is a Sonarr series queue item. size/sizeleft arrive as JSON
// decimals, hence float64 before the int64 normalization.
type queueRecord struct {
	SeriesID   int64   `json:"seriesId"`
	Title      string  `json:"title"`
	Status     string  `json:"status"`
	Size       float64 `json:"size"`
	SizeLeft   float64 `json:"sizeleft"`
	DownloadID string  `json:"downloadId"`
}

// GetQueue returns the download queue normalized to []plugins.QueueItem
// (seriesId → ExternalID per the 13-4b queue mapping), following the
// pagination envelope until all records are collected.
func (c *Client) GetQueue(ctx context.Context) ([]plugins.QueueItem, error) {
	items := []plugins.QueueItem{}
	totalRecords := 0
	for page := 1; page <= maxQueuePages; page++ {
		body, err := c.doRequest(ctx, http.MethodGet,
			c.buildURL(fmt.Sprintf("/queue?page=%d&pageSize=100", page)), c.config.APIKey, nil)
		if err != nil {
			return nil, err
		}

		var envelope queuePage
		if err := json.Unmarshal(body, &envelope); err != nil {
			return nil, &plugins.PluginError{
				Code:    plugins.ErrCodeConnectionFailed,
				Message: "sonarr queue response is not parseable",
				Cause:   err,
			}
		}

		for _, rec := range envelope.Records {
			items = append(items, plugins.QueueItem{
				ExternalID: rec.SeriesID,
				Title:      rec.Title,
				Status:     rec.Status,
				Size:       int64(rec.Size),
				SizeLeft:   int64(rec.SizeLeft),
				DownloadID: rec.DownloadID,
			})
		}

		totalRecords = envelope.TotalRecords
		if len(envelope.Records) == 0 || len(items) >= totalRecords {
			return items, nil
		}
	}

	// No silent caps (13-4a CR L1 parity).
	slog.Warn("Sonarr queue truncated at page cap",
		"collected", len(items), "total_records", totalRecords, "max_pages", maxQueuePages)
	return items, nil
}

// GetQualityProfiles lists the configured quality profiles (plugins.ProfileLister).
func (c *Client) GetQualityProfiles(ctx context.Context) ([]plugins.QualityProfile, error) {
	body, err := c.doRequest(ctx, http.MethodGet, c.buildURL("/qualityprofile"), c.config.APIKey, nil)
	if err != nil {
		return nil, err
	}

	var profiles []plugins.QualityProfile
	if err := json.Unmarshal(body, &profiles); err != nil {
		return nil, &plugins.PluginError{
			Code:    plugins.ErrCodeConnectionFailed,
			Message: "sonarr qualityprofile response is not parseable",
			Cause:   err,
		}
	}
	return profiles, nil
}

// GetRootFolders lists the configured root folders — see GetQualityProfiles.
func (c *Client) GetRootFolders(ctx context.Context) ([]plugins.RootFolder, error) {
	body, err := c.doRequest(ctx, http.MethodGet, c.buildURL("/rootfolder"), c.config.APIKey, nil)
	if err != nil {
		return nil, err
	}

	var folders []plugins.RootFolder
	if err := json.Unmarshal(body, &folders); err != nil {
		return nil, &plugins.PluginError{
			Code:    plugins.ErrCodeConnectionFailed,
			Message: "sonarr rootfolder response is not parseable",
			Cause:   err,
		}
	}
	return folders, nil
}

// doRequest performs a rate-limited authenticated request and maps transport/
// status failures to typed PluginErrors (Radarr-client parity — the shared
// helper extraction is deliberately deferred per ADR Decision 3 until a third
// client re-hand-rolls this).
func (c *Client) doRequest(ctx context.Context, method, fullURL, apiKey string, payload []byte) ([]byte, error) {
	// Rule 27 ① — limiter.Wait is the first line of every request path.
	if err := c.limiter.Wait(ctx); err != nil {
		return nil, mapTransportError(err, "rate limiter wait aborted")
	}

	var bodyReader io.Reader
	if payload != nil {
		bodyReader = bytes.NewReader(payload)
	}
	req, err := http.NewRequestWithContext(ctx, method, fullURL, bodyReader)
	if err != nil {
		return nil, &plugins.PluginError{Code: plugins.ErrCodeConnectionFailed, Message: "create sonarr request", Cause: err}
	}
	req.Header.Set("X-Api-Key", apiKey)
	req.Header.Set("Accept", "application/json")
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, mapTransportError(err, "sonarr request failed")
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, &plugins.PluginError{Code: plugins.ErrCodeConnectionFailed, Message: "read sonarr response", Cause: err}
	}

	switch {
	case resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden:
		return nil, &plugins.PluginError{
			Code:    plugins.ErrCodeAuthFailed,
			Message: fmt.Sprintf("sonarr rejected the API key (status %d)", resp.StatusCode),
		}
	case resp.StatusCode == http.StatusBadRequest && method == http.MethodPost:
		return nil, &plugins.PluginError{
			Code:    plugins.ErrCodeAddFailed,
			Message: fmt.Sprintf("sonarr rejected the add: %s", truncate(string(body), 500)),
		}
	case resp.StatusCode < 200 || resp.StatusCode >= 300:
		return nil, &plugins.PluginError{
			Code:    plugins.ErrCodeConnectionFailed,
			Message: fmt.Sprintf("sonarr returned status %d: %s", resp.StatusCode, truncate(string(body), 200)),
		}
	}
	return body, nil
}

// mapTransportError distinguishes deadline/timeout failures (DVR_TIMEOUT)
// from plain connectivity failures (DVR_CONNECTION_FAILED).
func mapTransportError(err error, message string) *plugins.PluginError {
	code := plugins.ErrCodeConnectionFailed
	var netErr interface{ Timeout() bool }
	if errors.Is(err, context.DeadlineExceeded) || (errors.As(err, &netErr) && netErr.Timeout()) {
		code = plugins.ErrCodeTimeout
	}
	return &plugins.PluginError{Code: code, Message: message, Cause: err}
}

// truncate bounds upstream error bodies (rune-safe) so log lines and API
// messages stay sane.
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n]) + "…"
}
