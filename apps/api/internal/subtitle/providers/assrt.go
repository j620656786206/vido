package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"golang.org/x/time/rate"

	"github.com/vido/api/internal/secrets"
)

const (
	assrtBaseURL     = "https://api.assrt.net/v1"
	assrtSecretKey   = "assrt_api_key"
	assrtRateLimit   = 2 // requests per second
	assrtHTTPTimeout = 30 * time.Second
)

// AssrtProvider implements SubtitleProvider for the Assrt (射手網) subtitle source.
type AssrtProvider struct {
	apiKey      string
	disabled    bool
	httpClient  *http.Client
	rateLimiter *rate.Limiter
	testBaseURL string // override for testing; empty = use assrtBaseURL
}

// NewAssrtProvider creates an Assrt subtitle provider.
// If the API key is not configured in the secrets service, the provider is created
// in disabled mode and will return empty results instead of errors.
func NewAssrtProvider(ctx context.Context, secretsSvc secrets.SecretsServiceInterface) *AssrtProvider {
	p := &AssrtProvider{
		httpClient: &http.Client{
			Timeout: assrtHTTPTimeout,
		},
		rateLimiter: rate.NewLimiter(rate.Limit(assrtRateLimit), 1),
	}

	apiKey, err := secretsSvc.Retrieve(ctx, assrtSecretKey)
	if err != nil {
		slog.Info("Assrt provider disabled — API key not configured, skipping Assrt source",
			"secret_key", assrtSecretKey,
		)
		p.disabled = true
		return p
	}

	p.apiKey = apiKey
	slog.Info("Assrt provider initialized")
	return p
}

// Name returns the provider identifier.
func (p *AssrtProvider) Name() string {
	return "assrt"
}

// assrtSearchResponse is the top-level API response for search.
type assrtSearchResponse struct {
	Status int              `json:"status"`
	Sub    *assrtSearchSub  `json:"sub"`
}

type assrtSearchSub struct {
	Subs []assrtSearchItem `json:"subs"`
}

type assrtSearchItem struct {
	ID         int    `json:"id"`
	NativeName string `json:"native_name"` // P1-011 fix: use native_name, NOT name
	VideoName  string `json:"videoname"`
	Lang       string `json:"lang"`
	Upload     string `json:"upload_time"`
}

// assrtDetailResponse is the API response for subtitle detail.
type assrtDetailResponse struct {
	Status int             `json:"status"`
	Sub    *assrtDetailSub `json:"sub"`
}

type assrtDetailSub struct {
	Subs []assrtDetailItem `json:"subs"`
}

type assrtDetailItem struct {
	ID       int               `json:"id"`
	Filename string            `json:"filename"`
	URL      string            `json:"url"`
	Filelist []assrtDetailFile `json:"filelist"`
}

type assrtDetailFile struct {
	URL      string `json:"url"`
	Filename string `json:"f"`
	Size     int64  `json:"s"`
}

func (p *AssrtProvider) baseURL() string {
	if p.testBaseURL != "" {
		return p.testBaseURL
	}
	return assrtBaseURL
}

// Search queries the Assrt API for subtitles matching the query.
// Returns empty results (not an error) when the provider is disabled.
func (p *AssrtProvider) Search(ctx context.Context, query SubtitleQuery) ([]SubtitleResult, error) {
	if p.disabled {
		slog.Info("Assrt search skipped — provider disabled")
		return nil, nil
	}

	if err := p.rateLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("assrt rate limiter: %w", err)
	}

	u, err := url.Parse(p.baseURL() + "/sub/search")
	if err != nil {
		return nil, fmt.Errorf("assrt: invalid URL: %w", err)
	}

	q := u.Query()
	q.Set("q", query.Title)
	q.Set("token", p.apiKey)
	if query.Year > 0 {
		q.Set("q", fmt.Sprintf("%s %d", query.Title, query.Year))
	}
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("assrt: create request: %w", err)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("assrt: search request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("assrt: read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("assrt: search returned HTTP %d for %s", resp.StatusCode, "/sub/search")
	}

	var searchResp assrtSearchResponse
	if err := json.Unmarshal(body, &searchResp); err != nil {
		slog.Debug("Assrt search: malformed JSON response", "body", string(body))
		return nil, fmt.Errorf("assrt: failed to parse search response: %w", err)
	}

	if searchResp.Sub == nil {
		return nil, nil
	}

	results := make([]SubtitleResult, 0, len(searchResp.Sub.Subs))
	for _, item := range searchResp.Sub.Subs {
		uploadDate, _ := time.Parse("2006-01-02 15:04:05", item.Upload)

		results = append(results, SubtitleResult{
			ID:         fmt.Sprintf("%d", item.ID),
			Source:     "assrt",
			Filename:   item.NativeName, // P1-011: use native_name
			Language:   item.Lang,
			UploadDate: uploadDate,
		})
	}

	return results, nil
}

// Download fetches the subtitle file content by ID.
// It first retrieves the subtitle detail to find the download URL, then downloads the file.
func (p *AssrtProvider) Download(ctx context.Context, id string) ([]byte, error) {
	if p.disabled {
		return nil, fmt.Errorf("assrt: provider is disabled — no API key configured")
	}

	// Step 1: Get subtitle detail to find download URL
	if err := p.rateLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("assrt rate limiter: %w", err)
	}

	u, err := url.Parse(p.baseURL() + "/sub/detail")
	if err != nil {
		return nil, fmt.Errorf("assrt: invalid URL: %w", err)
	}

	q := u.Query()
	q.Set("id", id)
	q.Set("token", p.apiKey)
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("assrt: create detail request: %w", err)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("assrt: detail request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("assrt: read detail response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("assrt: detail returned HTTP %d for /sub/detail", resp.StatusCode)
	}

	var detailResp assrtDetailResponse
	if err := json.Unmarshal(body, &detailResp); err != nil {
		slog.Debug("Assrt detail: malformed JSON response", "body", string(body))
		return nil, fmt.Errorf("assrt: failed to parse detail response: %w", err)
	}

	if detailResp.Sub == nil || len(detailResp.Sub.Subs) == 0 {
		return nil, fmt.Errorf("assrt: no subtitle found for id %s", id)
	}

	// Find the download URL from the first result
	detail := detailResp.Sub.Subs[0]
	downloadURL := detail.URL
	if downloadURL == "" && len(detail.Filelist) > 0 {
		downloadURL = detail.Filelist[0].URL
	}
	if downloadURL == "" {
		return nil, fmt.Errorf("assrt: no download URL found for id %s", id)
	}

	// Step 2: Download the actual subtitle file
	if err := p.rateLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("assrt rate limiter: %w", err)
	}

	dlReq, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		return nil, fmt.Errorf("assrt: create download request: %w", err)
	}

	dlResp, err := p.httpClient.Do(dlReq)
	if err != nil {
		return nil, fmt.Errorf("assrt: download request failed: %w", err)
	}
	defer dlResp.Body.Close()

	if dlResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("assrt: download returned HTTP %d", dlResp.StatusCode)
	}

	data, err := io.ReadAll(dlResp.Body)
	if err != nil {
		return nil, fmt.Errorf("assrt: read download response: %w", err)
	}

	if len(data) == 0 {
		return nil, fmt.Errorf("assrt: downloaded empty subtitle file for id %s", id)
	}

	return data, nil
}

// Compile-time interface verification.
var _ SubtitleProvider = (*AssrtProvider)(nil)
