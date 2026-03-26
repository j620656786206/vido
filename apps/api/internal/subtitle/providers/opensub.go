package providers

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"

	"github.com/vido/api/internal/secrets"
)

const (
	openSubBaseURL          = "https://api.opensubtitles.com/api/v1"
	openSubAPIKeySecret     = "opensubtitles_api_key"
	openSubUsernameSecret   = "opensubtitles_username"
	openSubPasswordSecret   = "opensubtitles_password"
	openSubHTTPTimeout      = 30 * time.Second
	openSubMaxResponseBytes = 1 << 20  // 1 MB for JSON responses
	openSubMaxDownloadBytes = 50 << 20 // 50 MB for subtitle files
	openSubTokenBuffer      = 5 * time.Minute // refresh 5 min before expiry
	openSubHashChunkSize    = 64 * 1024       // 64 KB
	openSubUserAgent        = "Vido/1.0 (NAS Media Manager)"
)

// openSubMaxRetries is the maximum number of 429 retries before giving up.
const openSubMaxRetries = 2

const (
	openSubRateLimit = 5 // requests per second
	openSubRateBurst = 5 // token bucket burst size
)

// OpenSubProvider implements SubtitleProvider for the OpenSubtitles REST API v1.
type OpenSubProvider struct {
	apiKey      string
	username    string
	password    string
	disabled    bool
	httpClient  *http.Client
	rateLimiter *rate.Limiter
	testBaseURL string // override for testing

	mu          sync.RWMutex
	authToken   string
	tokenExpiry time.Time
}

// NewOpenSubProvider creates an OpenSubtitles subtitle provider.
// Disabled mode if API key or credentials are not configured.
func NewOpenSubProvider(ctx context.Context, secretsSvc secrets.SecretsServiceInterface) *OpenSubProvider {
	p := &OpenSubProvider{
		httpClient:  &http.Client{Timeout: openSubHTTPTimeout},
		rateLimiter: rate.NewLimiter(rate.Limit(openSubRateLimit), openSubRateBurst),
	}

	apiKey, err := secretsSvc.Retrieve(ctx, openSubAPIKeySecret)
	if err != nil {
		slog.Info("OpenSubtitles provider disabled — API key not configured",
			"secret_key", openSubAPIKeySecret)
		p.disabled = true
		return p
	}
	p.apiKey = apiKey

	username, _ := secretsSvc.Retrieve(ctx, openSubUsernameSecret)
	password, _ := secretsSvc.Retrieve(ctx, openSubPasswordSecret)
	p.username = username
	p.password = password

	slog.Info("OpenSubtitles provider initialized")
	return p
}

// Name returns the provider identifier.
func (p *OpenSubProvider) Name() string {
	return "opensubtitles"
}

func (p *OpenSubProvider) baseURL() string {
	if p.testBaseURL != "" {
		return p.testBaseURL
	}
	return openSubBaseURL
}

// authenticate performs a login to get a JWT token.
func (p *OpenSubProvider) authenticate(ctx context.Context) error {
	if err := p.rateLimiter.Wait(ctx); err != nil {
		return fmt.Errorf("opensubtitles rate limiter: %w", err)
	}

	body, _ := json.Marshal(map[string]string{
		"username": p.username,
		"password": p.password,
	})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL()+"/login", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("opensubtitles: create login request: %w", err)
	}
	req.Header.Set("Api-Key", p.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", openSubUserAgent)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("opensubtitles: login request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, openSubMaxResponseBytes))
	if err != nil {
		return fmt.Errorf("opensubtitles: read login response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("opensubtitles: login returned HTTP %d", resp.StatusCode)
	}

	var loginResp struct {
		Token  string `json:"token"`
		Status int    `json:"status"`
	}
	if err := json.Unmarshal(respBody, &loginResp); err != nil {
		return fmt.Errorf("opensubtitles: parse login response: %w", err)
	}

	if loginResp.Token == "" {
		return fmt.Errorf("opensubtitles: login returned empty token")
	}

	p.authToken = loginResp.Token
	// OpenSubtitles tokens are valid for 24 hours; refresh 5 min early
	p.tokenExpiry = time.Now().Add(24*time.Hour - openSubTokenBuffer)

	return nil
}

// ensureAuth checks token validity and re-authenticates if needed.
func (p *OpenSubProvider) ensureAuth(ctx context.Context) error {
	// Fast path: check with read lock first.
	p.mu.RLock()
	valid := p.authToken != "" && time.Now().Before(p.tokenExpiry)
	p.mu.RUnlock()
	if valid {
		return nil
	}

	// Slow path: acquire write lock and re-check.
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.authToken != "" && time.Now().Before(p.tokenExpiry) {
		return nil
	}

	return p.authenticate(ctx)
}

// openSubSearchResponse represents the API search response.
type openSubSearchResponse struct {
	Data []openSubSearchItem `json:"data"`
}

type openSubSearchItem struct {
	ID         string                 `json:"id"`
	Attributes openSubSearchAttrs     `json:"attributes"`
}

type openSubSearchAttrs struct {
	SubtitleID    string               `json:"subtitle_id"`
	Language      string               `json:"language"`
	DownloadCount int                  `json:"download_count"`
	Release       string               `json:"release"`
	UploadDate    string               `json:"upload_date"`
	Uploader      *openSubUploader     `json:"uploader"`
	Files         []openSubFile        `json:"files"`
	FeatureDetails *openSubFeature     `json:"feature_details"`
}

type openSubUploader struct {
	Name string `json:"name"`
}

type openSubFile struct {
	FileID   int    `json:"file_id"`
	FileName string `json:"file_name"`
}

type openSubFeature struct {
	Title string `json:"title"`
}

// Search queries the OpenSubtitles API for subtitles.
func (p *OpenSubProvider) Search(ctx context.Context, query SubtitleQuery) ([]SubtitleResult, error) {
	return p.searchWithRetry(ctx, query, 0)
}

func (p *OpenSubProvider) searchWithRetry(ctx context.Context, query SubtitleQuery, retryCount int) ([]SubtitleResult, error) {
	if p.disabled {
		slog.Info("OpenSubtitles search skipped — provider disabled")
		return nil, nil
	}

	if query.Title == "" && query.ImdbID == "" {
		return nil, fmt.Errorf("opensubtitles: search requires title or IMDB ID")
	}

	if err := p.ensureAuth(ctx); err != nil {
		return nil, fmt.Errorf("opensubtitles: auth failed: %w", err)
	}

	u, err := url.Parse(p.baseURL() + "/subtitles")
	if err != nil {
		return nil, fmt.Errorf("opensubtitles: invalid URL: %w", err)
	}

	q := u.Query()
	if query.ImdbID != "" {
		q.Set("imdb_id", query.ImdbID)
	} else {
		q.Set("query", query.Title)
	}
	if query.FileHash != "" {
		q.Set("moviehash", query.FileHash)
	}
	if len(query.Languages) > 0 {
		q.Set("languages", strings.Join(query.Languages, ","))
	} else {
		q.Set("languages", "zh-cn,zh-tw")
	}
	if query.Season > 0 {
		q.Set("season_number", strconv.Itoa(query.Season))
	}
	if query.Episode > 0 {
		q.Set("episode_number", strconv.Itoa(query.Episode))
	}
	u.RawQuery = q.Encode()

	if err := p.rateLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("opensubtitles rate limiter: %w", err)
	}

	resp, err := p.doRequest(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, openSubMaxResponseBytes))
	if err != nil {
		return nil, fmt.Errorf("opensubtitles: read search response: %w", err)
	}

	if resp.StatusCode == http.StatusTooManyRequests {
		if retryCount >= openSubMaxRetries {
			return nil, fmt.Errorf("opensubtitles: rate limited after %d retries", retryCount)
		}
		retryAfter := parseRetryAfter(resp.Header.Get("Retry-After"))
		select {
		case <-time.After(retryAfter):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
		return p.searchWithRetry(ctx, query, retryCount+1)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("opensubtitles: search returned HTTP %d", resp.StatusCode)
	}

	var searchResp openSubSearchResponse
	if err := json.Unmarshal(body, &searchResp); err != nil {
		slog.Debug("OpenSubtitles: malformed JSON response", "body", string(body))
		return nil, fmt.Errorf("opensubtitles: parse search response: %w", err)
	}

	results := make([]SubtitleResult, 0, len(searchResp.Data))
	for _, item := range searchResp.Data {
		result := SubtitleResult{
			Source:    "opensubtitles",
			Language:  item.Attributes.Language,
			Downloads: item.Attributes.DownloadCount,
		}

		if len(item.Attributes.Files) > 0 {
			result.ID = strconv.Itoa(item.Attributes.Files[0].FileID)
			result.Filename = item.Attributes.Files[0].FileName
		}

		if item.Attributes.Uploader != nil {
			result.Group = item.Attributes.Uploader.Name
		}

		uploadDate, _ := time.Parse("2006-01-02T15:04:05Z", item.Attributes.UploadDate)
		result.UploadDate = uploadDate

		// Extract format from filename
		if idx := strings.LastIndex(result.Filename, "."); idx >= 0 {
			result.Format = strings.TrimPrefix(result.Filename[idx:], ".")
		}

		results = append(results, result)
	}

	return results, nil
}

// openSubDownloadResponse represents the download API response.
type openSubDownloadResponse struct {
	Link     string `json:"link"`
	FileName string `json:"file_name"`
}

// Download fetches a subtitle file by file ID.
func (p *OpenSubProvider) Download(ctx context.Context, id string) ([]byte, error) {
	return p.downloadWithRetry(ctx, id, 0)
}

func (p *OpenSubProvider) downloadWithRetry(ctx context.Context, id string, retryCount int) ([]byte, error) {
	if p.disabled {
		return nil, fmt.Errorf("opensubtitles: provider is disabled")
	}

	if err := p.ensureAuth(ctx); err != nil {
		return nil, fmt.Errorf("opensubtitles: auth failed: %w", err)
	}

	// Step 1: Get download link
	fileID, err := strconv.Atoi(id)
	if err != nil {
		return nil, fmt.Errorf("opensubtitles: invalid file ID %q: %w", id, err)
	}

	if err := p.rateLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("opensubtitles rate limiter: %w", err)
	}

	reqBody, _ := json.Marshal(map[string]int{"file_id": fileID})
	resp, err := p.doRequest(ctx, http.MethodPost, p.baseURL()+"/download", bytes.NewReader(reqBody))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, openSubMaxResponseBytes))
	if err != nil {
		return nil, fmt.Errorf("opensubtitles: read download response: %w", err)
	}

	if resp.StatusCode == http.StatusTooManyRequests {
		if retryCount >= openSubMaxRetries {
			return nil, fmt.Errorf("opensubtitles: download rate limited after %d retries", retryCount)
		}
		retryAfter := parseRetryAfter(resp.Header.Get("Retry-After"))
		select {
		case <-time.After(retryAfter):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
		return p.downloadWithRetry(ctx, id, retryCount+1)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("opensubtitles: download API returned HTTP %d", resp.StatusCode)
	}

	var dlResp openSubDownloadResponse
	if err := json.Unmarshal(body, &dlResp); err != nil {
		return nil, fmt.Errorf("opensubtitles: parse download response: %w", err)
	}

	if dlResp.Link == "" {
		return nil, fmt.Errorf("opensubtitles: download response has no link for file %s", id)
	}

	// Step 2: Fetch actual subtitle file
	if err := p.rateLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("opensubtitles rate limiter: %w", err)
	}

	dlReq, err := http.NewRequestWithContext(ctx, http.MethodGet, dlResp.Link, nil)
	if err != nil {
		return nil, fmt.Errorf("opensubtitles: create file request: %w", err)
	}
	dlReq.Header.Set("User-Agent", openSubUserAgent)

	dlFileResp, err := p.httpClient.Do(dlReq)
	if err != nil {
		return nil, fmt.Errorf("opensubtitles: file download failed: %w", err)
	}
	defer dlFileResp.Body.Close()

	if dlFileResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("opensubtitles: file download returned HTTP %d", dlFileResp.StatusCode)
	}

	data, err := io.ReadAll(io.LimitReader(dlFileResp.Body, openSubMaxDownloadBytes))
	if err != nil {
		return nil, fmt.Errorf("opensubtitles: read file: %w", err)
	}

	if len(data) == 0 {
		return nil, fmt.Errorf("opensubtitles: downloaded empty file for ID %s", id)
	}

	return data, nil
}

// doRequest makes an authenticated API request with standard headers.
func (p *OpenSubProvider) doRequest(ctx context.Context, method, reqURL string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, reqURL, body)
	if err != nil {
		return nil, fmt.Errorf("opensubtitles: create request: %w", err)
	}

	p.mu.RLock()
	token := p.authToken
	p.mu.RUnlock()

	req.Header.Set("Api-Key", p.apiKey)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", openSubUserAgent)

	return p.httpClient.Do(req)
}

// parseRetryAfter parses the Retry-After header value (seconds).
func parseRetryAfter(value string) time.Duration {
	if value == "" {
		return 5 * time.Second // default
	}
	seconds, err := strconv.Atoi(value)
	if err != nil || seconds <= 0 {
		return 5 * time.Second
	}
	if seconds > 60 {
		seconds = 60 // cap at 60 seconds
	}
	return time.Duration(seconds) * time.Second
}

// CalculateOpenSubHash computes the OpenSubtitles hash for a media file.
// Algorithm: file_size + sum_of_first_64KB + sum_of_last_64KB (as 64-bit little-endian words).
func CalculateOpenSubHash(filePath string) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("opensub hash: open file: %w", err)
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		return "", fmt.Errorf("opensub hash: stat file: %w", err)
	}

	fileSize := fi.Size()
	if fileSize < int64(openSubHashChunkSize*2) {
		return "", fmt.Errorf("opensub hash: file too small (%d bytes, need at least %d)", fileSize, openSubHashChunkSize*2)
	}

	hash := uint64(fileSize)

	// Read first 64KB
	buf := make([]byte, openSubHashChunkSize)
	if _, err := io.ReadFull(f, buf); err != nil {
		return "", fmt.Errorf("opensub hash: read first chunk: %w", err)
	}
	hash += sumChunk(buf)

	// Read last 64KB
	if _, err := f.Seek(-int64(openSubHashChunkSize), io.SeekEnd); err != nil {
		return "", fmt.Errorf("opensub hash: seek last chunk: %w", err)
	}
	if _, err := io.ReadFull(f, buf); err != nil {
		return "", fmt.Errorf("opensub hash: read last chunk: %w", err)
	}
	hash += sumChunk(buf)

	return fmt.Sprintf("%016x", hash), nil
}

// sumChunk sums a byte slice as 64-bit little-endian words.
func sumChunk(buf []byte) uint64 {
	var sum uint64
	for i := 0; i+8 <= len(buf); i += 8 {
		sum += binary.LittleEndian.Uint64(buf[i : i+8])
	}
	return sum
}

// Compile-time interface verification.
var _ SubtitleProvider = (*OpenSubProvider)(nil)
