// Package radarr implements the plugins.DVRPlugin interface against the
// Radarr API v3 (current Radarr v5 still serves /api/v3) — Story 13-4a AC #2.
// Radarr is TMDB-native, so movie adds need no external-id resolution (the
// Sonarr TVDB gotcha is 13-4b's problem).
package radarr

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"golang.org/x/time/rate"

	"github.com/vido/api/internal/plugins"
)

// requestsPerSecond / burst — Radarr publishes no rate ceiling; 10 req/s is
// the LAN-local observed-safe ceiling recorded in the story (Rule 27 ①).
const (
	requestsPerSecond = 10
	burstSize         = 10
)

// maxQueuePages bounds the GetQueue pagination loop so a pathological
// totalRecords from the upstream can never spin us forever.
const maxQueuePages = 20

// Client communicates with the Radarr API v3. One reused http.Client
// (Rule 14) with a 10s timeout, one *rate.Limiter for process life.
type Client struct {
	config     plugins.PluginConfig
	httpClient *http.Client
	limiter    *rate.Limiter
}

// Compile-time interface verification.
var (
	_ plugins.DVRPlugin     = (*Client)(nil)
	_ plugins.ProfileLister = (*Client)(nil)
)

// NewClient creates a new Radarr API client for the given config.
func NewClient(config plugins.PluginConfig) *Client {
	return &Client{
		config: config,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		limiter: rate.NewLimiter(rate.Limit(requestsPerSecond), burstSize),
	}
}

// Name returns the plugin name.
func (c *Client) Name() string {
	return "radarr"
}

// buildURL constructs the full API v3 URL for the given path.
func (c *Client) buildURL(path string) string {
	return strings.TrimSuffix(c.config.URL, "/") + "/api/v3" + path
}

// systemStatus is the subset of GET /system/status we validate.
type systemStatus struct {
	Version string `json:"version"`
}

// TestConnection verifies connectivity using the GIVEN config (not the stored
// one) so the settings PUT test-before-save guard can probe unsaved configs.
// 200 + parseable non-empty version = pass; 401 → DVR_AUTH_FAILED.
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
			Message: "radarr system/status returned no parseable version",
			Cause:   err,
		}
	}
	return nil
}

// addMovieRequest is the POST /movie wire body. Radarr resolves the remaining
// metadata (title etc.) from tmdbId server-side.
type addMovieRequest struct {
	TMDbID           int64           `json:"tmdbId"`
	QualityProfileID int64           `json:"qualityProfileId"`
	RootFolderPath   string          `json:"rootFolderPath"`
	Monitored        bool            `json:"monitored"`
	AddOptions       addMovieOptions `json:"addOptions"`
}

type addMovieOptions struct {
	SearchForMovie bool `json:"searchForMovie"`
}

// addMovieResponse is the subset of the created movie resource we consume.
type addMovieResponse struct {
	ID int64 `json:"id"`
}

// AddMovie adds a movie by TMDb id and returns Radarr's own movie id.
// A 400 (e.g. "already exists") maps to DVR_ADD_FAILED with the upstream message.
func (c *Client) AddMovie(ctx context.Context, tmdbID int64, opts plugins.AddOptions) (int64, error) {
	payload, err := json.Marshal(addMovieRequest{
		TMDbID:           tmdbID,
		QualityProfileID: opts.QualityProfileID,
		RootFolderPath:   opts.RootFolderPath,
		Monitored:        true,
		AddOptions:       addMovieOptions{SearchForMovie: opts.SearchNow},
	})
	if err != nil {
		return 0, &plugins.PluginError{Code: plugins.ErrCodeAddFailed, Message: "encode add-movie request", Cause: err}
	}

	body, err := c.doRequest(ctx, http.MethodPost, c.buildURL("/movie"), c.config.APIKey, payload)
	if err != nil {
		return 0, err
	}

	var created addMovieResponse
	if err := json.Unmarshal(body, &created); err != nil || created.ID == 0 {
		return 0, &plugins.PluginError{
			Code:    plugins.ErrCodeAddFailed,
			Message: "radarr add-movie response carried no movie id",
			Cause:   err,
		}
	}
	return created.ID, nil
}

// AddSeries is not supported — Radarr is movie-only; series routing lands
// with the Sonarr plugin (13-4b).
func (c *Client) AddSeries(ctx context.Context, tmdbID int64, opts plugins.AddOptions) (int64, error) {
	return 0, &plugins.PluginError{
		Code:    plugins.ErrCodeNotSupported,
		Message: "radarr is movie-only; series requests need the sonarr plugin",
	}
}

// queuePage is the paginated GET /queue envelope.
type queuePage struct {
	Page         int           `json:"page"`
	PageSize     int           `json:"pageSize"`
	TotalRecords int           `json:"totalRecords"`
	Records      []queueRecord `json:"records"`
}

// queueRecord is a Radarr movie queue item. size/sizeleft arrive as JSON
// decimals, hence float64 before the int64 normalization.
type queueRecord struct {
	MovieID    int64   `json:"movieId"`
	Title      string  `json:"title"`
	Status     string  `json:"status"`
	Size       float64 `json:"size"`
	SizeLeft   float64 `json:"sizeleft"`
	DownloadID string  `json:"downloadId"`
}

// GetQueue returns the download queue normalized to []plugins.QueueItem,
// following the pagination envelope until all records are collected.
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
				Message: "radarr queue response is not parseable",
				Cause:   err,
			}
		}

		for _, rec := range envelope.Records {
			items = append(items, plugins.QueueItem{
				ExternalID: rec.MovieID,
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

	// No silent caps (13-4a CR L1): a queue larger than maxQueuePages×100 is
	// pathological, but the truncation must be visible, not implied complete.
	slog.Warn("Radarr queue truncated at page cap",
		"collected", len(items), "total_records", totalRecords, "max_pages", maxQueuePages)
	return items, nil
}

// GetQualityProfiles lists the configured quality profiles (client-level
// extra behind plugins.ProfileLister — consumed by config validation + the
// settings passthrough, AC #4 / 13-6 UI).
func (c *Client) GetQualityProfiles(ctx context.Context) ([]plugins.QualityProfile, error) {
	body, err := c.doRequest(ctx, http.MethodGet, c.buildURL("/qualityprofile"), c.config.APIKey, nil)
	if err != nil {
		return nil, err
	}

	var profiles []plugins.QualityProfile
	if err := json.Unmarshal(body, &profiles); err != nil {
		return nil, &plugins.PluginError{
			Code:    plugins.ErrCodeConnectionFailed,
			Message: "radarr qualityprofile response is not parseable",
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
			Message: "radarr rootfolder response is not parseable",
			Cause:   err,
		}
	}
	return folders, nil
}

// doRequest performs a rate-limited authenticated request and maps transport/
// status failures to typed PluginErrors. Success bodies are returned raw.
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
		return nil, &plugins.PluginError{Code: plugins.ErrCodeConnectionFailed, Message: "create radarr request", Cause: err}
	}
	req.Header.Set("X-Api-Key", apiKey)
	req.Header.Set("Accept", "application/json")
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, mapTransportError(err, "radarr request failed")
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, &plugins.PluginError{Code: plugins.ErrCodeConnectionFailed, Message: "read radarr response", Cause: err}
	}

	switch {
	case resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden:
		return nil, &plugins.PluginError{
			Code:    plugins.ErrCodeAuthFailed,
			Message: fmt.Sprintf("radarr rejected the API key (status %d)", resp.StatusCode),
		}
	case resp.StatusCode == http.StatusBadRequest && method == http.MethodPost:
		return nil, &plugins.PluginError{
			Code:    plugins.ErrCodeAddFailed,
			Message: fmt.Sprintf("radarr rejected the add: %s", truncate(string(body), 500)),
		}
	case resp.StatusCode < 200 || resp.StatusCode >= 300:
		return nil, &plugins.PluginError{
			Code:    plugins.ErrCodeConnectionFailed,
			Message: fmt.Sprintf("radarr returned status %d: %s", resp.StatusCode, truncate(string(body), 200)),
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

// truncate bounds upstream error bodies (rune-safe — Radarr messages can
// carry multi-byte UTF-8) so log lines and API messages stay sane.
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
