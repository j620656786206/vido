package qbittorrent

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Default session TTL — skip re-login if last login was within this duration.
const defaultAuthTTL = 30 * time.Minute

// Client communicates with the qBittorrent Web API v2.x.
type Client struct {
	config      *Config
	httpClient  *http.Client
	lastLoginAt time.Time
	// qbtMajorVer caches the qBittorrent server major version (0 = not yet
	// resolved). Used to select the 4.x pause/resume vs 5.0+ stop/start endpoints.
	// Guarded by verMu because a single *Client is cached and shared across
	// concurrent requests (DownloadService.getClient), so concurrent actions
	// would otherwise race the first-time resolution.
	qbtMajorVer int
	verMu       sync.Mutex
}

// NewClient creates a new qBittorrent API client.
// Default timeout is 10 seconds per NFR-I2.
func NewClient(config *Config) *Client {
	jar, _ := cookiejar.New(nil)

	timeout := config.Timeout
	if timeout == 0 {
		timeout = 10 * time.Second
	}

	return &Client{
		config: config,
		httpClient: &http.Client{
			Timeout: timeout,
			Jar:     jar,
		},
	}
}

// buildURL constructs the full API URL for the given path.
// Supports custom base paths for reverse proxy configurations (NFR-I3).
func (c *Client) buildURL(path string) string {
	host := strings.TrimSuffix(c.config.Host, "/")
	basePath := strings.TrimSuffix(c.config.BasePath, "/")
	return fmt.Sprintf("%s%s/api/v2%s", host, basePath, path)
}

// Login authenticates with qBittorrent using the configured credentials.
// On success, the session cookie is stored in the HTTP client's cookie jar.
func (c *Client) Login(ctx context.Context) error {
	loginURL := c.buildURL("/auth/login")

	data := url.Values{}
	data.Set("username", c.config.Username)
	data.Set("password", c.config.Password)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, loginURL, strings.NewReader(data.Encode()))
	if err != nil {
		return &ConnectionError{
			Code:    ErrCodeConnectionFailed,
			Message: "failed to create login request",
			Cause:   err,
		}
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return &ConnectionError{
			Code:    ErrCodeConnectionFailed,
			Message: "login request failed",
			Cause:   err,
		}
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return &ConnectionError{
			Code:    ErrCodeConnectionFailed,
			Message: "failed to read login response",
			Cause:   err,
		}
	}

	if resp.StatusCode != http.StatusOK {
		return &ConnectionError{
			Code:    ErrCodeConnectionFailed,
			Message: fmt.Sprintf("login failed with status %d", resp.StatusCode),
		}
	}

	if string(body) != "Ok." {
		return &ConnectionError{
			Code:    ErrCodeAuthFailed,
			Message: "authentication failed: invalid credentials",
		}
	}

	slog.Info("qBittorrent login successful", "host", c.config.Host)
	c.lastLoginAt = time.Now()
	return nil
}

// ensureAuth checks if the session is still valid and logs in if needed.
// Skips login if last successful login was within the auth TTL.
func (c *Client) ensureAuth(ctx context.Context) error {
	if !c.lastLoginAt.IsZero() && time.Since(c.lastLoginAt) < defaultAuthTTL {
		return nil // Session still fresh
	}
	return c.Login(ctx)
}

// doWithAuth executes an HTTP request with automatic retry on 401/403.
// If the initial request returns 401 or 403, re-authenticates and retries once.
func (c *Client) doWithAuth(ctx context.Context, req *http.Request) (*http.Response, error) {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		resp.Body.Close()
		// Re-authenticate and retry
		if authErr := c.Login(ctx); authErr != nil {
			return nil, authErr
		}
		// Rebuild the request, preserving the body (via GetBody) and headers so a
		// POST-with-form-body action re-sends its payload intact after re-auth.
		// GET callers have a nil GetBody, so body stays nil — identical behavior.
		var body io.Reader
		if req.GetBody != nil {
			rewound, gerr := req.GetBody()
			if gerr != nil {
				return nil, fmt.Errorf("rewind retry request body: %w", gerr)
			}
			body = rewound
		}
		retryReq, err := http.NewRequestWithContext(ctx, req.Method, req.URL.String(), body)
		if err != nil {
			return nil, fmt.Errorf("create retry request: %w", err)
		}
		retryReq.Header = req.Header.Clone()
		return c.httpClient.Do(retryReq)
	}

	return resp, nil
}

// Ping checks if qBittorrent is reachable and authenticated.
// Uses app version endpoint as a lightweight health check.
// If the initial check fails, attempts re-authentication before retrying.
func (c *Client) Ping(ctx context.Context) error {
	_, err := c.getVersion(ctx, "/app/version")
	if err != nil {
		// Try re-authentication
		if authErr := c.Login(ctx); authErr != nil {
			return &ConnectionError{
				Code:    ErrCodeConnectionFailed,
				Message: "qBittorrent unreachable",
				Cause:   authErr,
			}
		}
		// Retry version check after re-auth
		if _, err = c.getVersion(ctx, "/app/version"); err != nil {
			return &ConnectionError{
				Code:    ErrCodeConnectionFailed,
				Message: "qBittorrent health check failed after re-auth",
				Cause:   err,
			}
		}
	}
	return nil
}

// TestConnection verifies connectivity to qBittorrent by authenticating
// and retrieving version information.
func (c *Client) TestConnection(ctx context.Context) (*VersionInfo, error) {
	if err := c.Login(ctx); err != nil {
		return nil, err
	}

	appVersion, err := c.getVersion(ctx, "/app/version")
	if err != nil {
		return nil, &ConnectionError{
			Code:    ErrCodeConnectionFailed,
			Message: "failed to get app version",
			Cause:   err,
		}
	}

	apiVersion, err := c.getVersion(ctx, "/app/webapiVersion")
	if err != nil {
		return nil, &ConnectionError{
			Code:    ErrCodeConnectionFailed,
			Message: "failed to get API version",
			Cause:   err,
		}
	}

	return &VersionInfo{
		AppVersion: appVersion,
		APIVersion: apiVersion,
	}, nil
}

// getVersion fetches a version string from the given API path.
func (c *Client) getVersion(ctx context.Context, path string) (string, error) {
	versionURL := c.buildURL(path)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, versionURL, nil)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response body: %w", err)
	}
	return strings.TrimSpace(string(body)), nil
}

// GetTorrents retrieves all torrents from qBittorrent.
func (c *Client) GetTorrents(ctx context.Context, opts *ListTorrentsOptions) ([]Torrent, error) {
	if err := c.ensureAuth(ctx); err != nil {
		return nil, err
	}

	apiURL := c.buildURL("/torrents/info")

	// Append query parameters
	if opts != nil {
		sep := "?"
		if opts.Filter != "" && opts.Filter != FilterAll {
			apiURL += fmt.Sprintf("%sfilter=%s", sep, string(opts.Filter))
			sep = "&"
		}
		if opts.Sort != "" {
			apiURL += fmt.Sprintf("%ssort=%s", sep, string(opts.Sort))
			sep = "&"
		}
		if opts.Reverse {
			apiURL += fmt.Sprintf("%sreverse=true", sep)
		}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, &ConnectionError{
			Code:    ErrCodeConnectionFailed,
			Message: "failed to create torrents request",
			Cause:   err,
		}
	}

	resp, err := c.doWithAuth(ctx, req)
	if err != nil {
		return nil, &ConnectionError{
			Code:    ErrCodeConnectionFailed,
			Message: "get torrents failed",
			Cause:   err,
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, &ConnectionError{
			Code:    ErrCodeConnectionFailed,
			Message: fmt.Sprintf("get torrents failed with status %d", resp.StatusCode),
		}
	}

	var qbTorrents []qbTorrentInfo
	if err := json.NewDecoder(resp.Body).Decode(&qbTorrents); err != nil {
		return nil, &ConnectionError{
			Code:    ErrCodeConnectionFailed,
			Message: "failed to decode torrents response",
			Cause:   err,
		}
	}

	torrents := make([]Torrent, len(qbTorrents))
	for i, qbt := range qbTorrents {
		torrents[i] = mapQBTorrentInfo(qbt)
	}

	return torrents, nil
}

// GetTorrentDetails retrieves detailed information for a specific torrent.
// Uses the hashes filter to fetch only the target torrent instead of the full list.
func (c *Client) GetTorrentDetails(ctx context.Context, hash string) (*TorrentDetails, error) {
	if err := c.ensureAuth(ctx); err != nil {
		return nil, err
	}

	// Fetch only the target torrent using hashes filter
	apiURL := c.buildURL("/torrents/info") + fmt.Sprintf("?hashes=%s", hash)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, &ConnectionError{
			Code:    ErrCodeConnectionFailed,
			Message: "failed to create torrents request",
			Cause:   err,
		}
	}

	resp, err := c.doWithAuth(ctx, req)
	if err != nil {
		return nil, &ConnectionError{
			Code:    ErrCodeConnectionFailed,
			Message: "get torrents failed",
			Cause:   err,
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, &ConnectionError{
			Code:    ErrCodeConnectionFailed,
			Message: fmt.Sprintf("get torrents failed with status %d", resp.StatusCode),
		}
	}

	var qbTorrents []qbTorrentInfo
	if err := json.NewDecoder(resp.Body).Decode(&qbTorrents); err != nil {
		return nil, &ConnectionError{
			Code:    ErrCodeConnectionFailed,
			Message: "failed to decode torrents response",
			Cause:   err,
		}
	}

	if len(qbTorrents) == 0 {
		return nil, &ConnectionError{
			Code:    ErrCodeTorrentNotFound,
			Message: fmt.Sprintf("torrent not found: %s", hash),
		}
	}

	torrent := mapQBTorrentInfo(qbTorrents[0])

	// Get detailed properties
	propsURL := c.buildURL("/torrents/properties") + fmt.Sprintf("?hash=%s", hash)
	propsReq, err := http.NewRequestWithContext(ctx, http.MethodGet, propsURL, nil)
	if err != nil {
		return nil, &ConnectionError{
			Code:    ErrCodeConnectionFailed,
			Message: "failed to create properties request",
			Cause:   err,
		}
	}

	propsResp, err := c.httpClient.Do(propsReq)
	if err != nil {
		return nil, &ConnectionError{
			Code:    ErrCodeConnectionFailed,
			Message: "get torrent properties failed",
			Cause:   err,
		}
	}
	defer propsResp.Body.Close()

	if propsResp.StatusCode != http.StatusOK {
		return nil, &ConnectionError{
			Code:    ErrCodeConnectionFailed,
			Message: fmt.Sprintf("get properties failed with status %d", propsResp.StatusCode),
		}
	}

	var props qbTorrentProperties
	if err := json.NewDecoder(propsResp.Body).Decode(&props); err != nil {
		return nil, &ConnectionError{
			Code:    ErrCodeConnectionFailed,
			Message: "failed to decode properties response",
			Cause:   err,
		}
	}

	return mapTorrentDetails(&torrent, props), nil
}

// parseQBMajorVersion extracts the major version number from a qBittorrent app
// version string such as "v4.6.7" or "5.0.3". It falls back to 4 (the
// long-established pause/resume API) when the value cannot be parsed, logging a
// warning so an unexpected version string surfaces in observability.
func parseQBMajorVersion(v string) int {
	v = strings.TrimPrefix(strings.TrimSpace(v), "v")
	if dot := strings.IndexByte(v, '.'); dot >= 0 {
		v = v[:dot]
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		slog.Warn("could not parse qBittorrent major version; defaulting to 4.x action endpoints", "version", v)
		return 4
	}
	return n
}

// majorVersion returns the qBittorrent server's major version, caching it on the
// client after the first lookup. Used to select the version-correct pause/resume
// WebAPI endpoints (qBT 5.0 renamed pause/resume → stop/start).
func (c *Client) majorVersion(ctx context.Context) (int, error) {
	c.verMu.Lock()
	defer c.verMu.Unlock()
	if c.qbtMajorVer > 0 {
		return c.qbtMajorVer, nil
	}
	if err := c.ensureAuth(ctx); err != nil {
		return 0, err
	}
	verStr, err := c.getVersion(ctx, "/app/version")
	if err != nil {
		return 0, err
	}
	c.qbtMajorVer = parseQBMajorVersion(verStr)
	return c.qbtMajorVer, nil
}

// doFormAction issues an authenticated POST with an x-www-form-urlencoded body
// to the given API path. The request is built with a rewindable body so a
// 401/403 re-auth retry (see doWithAuth) re-sends the form intact — without
// this, a mid-flight re-auth would silently send an empty body and qBittorrent
// would treat the action as a no-op while still returning 200.
func (c *Client) doFormAction(ctx context.Context, path string, form url.Values) error {
	if err := c.ensureAuth(ctx); err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.buildURL(path), strings.NewReader(form.Encode()))
	if err != nil {
		return &ConnectionError{
			Code:    ErrCodeConnectionFailed,
			Message: "failed to create action request",
			Cause:   err,
		}
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.doWithAuth(ctx, req)
	if err != nil {
		return &ConnectionError{
			Code:    ErrCodeConnectionFailed,
			Message: "action request failed",
			Cause:   err,
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &ConnectionError{
			Code:    ErrCodeConnectionFailed,
			Message: fmt.Sprintf("action failed with status %d", resp.StatusCode),
		}
	}
	return nil
}

// PauseTorrents pauses the given torrents. Accepts a slice so batch operations
// reuse the same method (hashes are pipe-joined per the qBittorrent convention).
// qBT 4.x uses POST /torrents/pause; qBT 5.0+ renamed it to /torrents/stop.
func (c *Client) PauseTorrents(ctx context.Context, hashes []string) error {
	major, err := c.majorVersion(ctx)
	if err != nil {
		return &ConnectionError{Code: ErrCodeConnectionFailed, Message: "failed to resolve qBittorrent version", Cause: err}
	}
	path := "/torrents/pause"
	if major >= 5 {
		path = "/torrents/stop"
	}
	return c.doFormAction(ctx, path, url.Values{"hashes": {strings.Join(hashes, "|")}})
}

// ResumeTorrents resumes the given torrents. qBT 4.x uses POST /torrents/resume;
// qBT 5.0+ renamed it to /torrents/start.
func (c *Client) ResumeTorrents(ctx context.Context, hashes []string) error {
	major, err := c.majorVersion(ctx)
	if err != nil {
		return &ConnectionError{Code: ErrCodeConnectionFailed, Message: "failed to resolve qBittorrent version", Cause: err}
	}
	path := "/torrents/resume"
	if major >= 5 {
		path = "/torrents/start"
	}
	return c.doFormAction(ctx, path, url.Values{"hashes": {strings.Join(hashes, "|")}})
}

// DeleteTorrents removes the given torrents from qBittorrent. When deleteFiles
// is true the downloaded data is also deleted from disk; when false the files
// are kept on disk. POST /torrents/delete is unchanged across qBT 4.x and 5.0+.
func (c *Client) DeleteTorrents(ctx context.Context, hashes []string, deleteFiles bool) error {
	return c.doFormAction(ctx, "/torrents/delete", url.Values{
		"hashes":      {strings.Join(hashes, "|")},
		"deleteFiles": {strconv.FormatBool(deleteFiles)},
	})
}
