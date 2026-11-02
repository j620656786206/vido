package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"golang.org/x/time/rate"
)

const (
	assrtBaseURL = "https://api.assrt.net/v1"
	// Assrt's real quota is 5 requests/minute PER TOKEN (live-verified 2026-07-31
	// via /v1/user/quota; the official docs' 20/min is wrong — trust the panel).
	// 13s spacing with burst 1 caps any 60s window at 5 calls (t=0,13,26,39,52)
	// with headroom. Do NOT raise burst: burst 2 allows 6 calls in a window.
	assrtRateInterval     = 13 * time.Second
	assrtRateBurst        = 1 // token bucket burst size
	assrtHTTPTimeout      = 30 * time.Second
	assrtMaxResponseBytes = 1 << 20  // 1 MB max for API JSON responses
	assrtMaxDownloadBytes = 50 << 20 // 50 MB max for subtitle file downloads
	assrtUserAgent        = "Vido/1.0 (NAS Media Manager)"
)

// AssrtProvider implements SubtitleProvider for the Assrt (射手網) subtitle source.
type AssrtProvider struct {
	// apiKey returns the current token, "" when none is configured. Asked on
	// every call so a token saved on the settings page applies immediately.
	apiKey      func(ctx context.Context) string
	httpClient  *http.Client
	rateLimiter *rate.Limiter
	testBaseURL string // override for testing; empty = use assrtBaseURL
}

// NewAssrtProvider creates an Assrt subtitle provider that reads its token
// from apiKey on every search and download. With no token the provider is
// idle: Search returns empty results and Download returns an error.
func NewAssrtProvider(apiKey func(ctx context.Context) string) *AssrtProvider {
	return &AssrtProvider{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: assrtHTTPTimeout,
		},
		rateLimiter: rate.NewLimiter(rate.Every(assrtRateInterval), assrtRateBurst),
	}
}

// token returns the configured token, or "" when there is none.
func (p *AssrtProvider) token(ctx context.Context) string {
	if p.apiKey == nil {
		return ""
	}
	return strings.TrimSpace(p.apiKey(ctx))
}

// Name returns the provider identifier.
func (p *AssrtProvider) Name() string {
	return "assrt"
}

// assrtSearchResponse is the top-level API response for search.
type assrtSearchResponse struct {
	Status int             `json:"status"`
	Sub    *assrtSearchSub `json:"sub"`
}

type assrtSearchSub struct {
	Subs assrtSearchItems `json:"subs"`
}

// assrtSearchItems is the `subs` list. The live API answers an empty search
// with `"subs": {}` (an empty PHP array serialised as an object), which a plain
// slice refuses to decode — live-verified 2026-11-02, the reason every
// no-match search used to fail with a parse error.
type assrtSearchItems []assrtSearchItem

func (l *assrtSearchItems) UnmarshalJSON(b []byte) error {
	trimmed := bytes.TrimSpace(b)
	if len(trimmed) == 0 || trimmed[0] != '[' {
		*l = nil
		return nil
	}
	var items []assrtSearchItem
	if err := json.Unmarshal(trimmed, &items); err != nil {
		return err
	}
	*l = items
	return nil
}

// assrtSearchItem covers the two row shapes the live search mixes in one list
// (live-verified 2026-11-02): the documented one (`id`, `native_name`,
// `upload_time`, `lang` as {desc, langlist}) and an index row (`fileid` as a
// string, `video_chinese_name`, `uploadtime`, `m_langn` as a list). Both ids
// work with /sub/detail.
type assrtSearchItem struct {
	ID               int       `json:"id"`
	FileID           string    `json:"fileid"`
	NativeName       string    `json:"native_name"` // P1-011 fix: use native_name, NOT name
	VideoChineseName string    `json:"video_chinese_name"`
	VideoName        string    `json:"videoname"`
	Lang             assrtLang `json:"lang"`
	LangFlags        []string  `json:"m_langn"`
	Upload           string    `json:"upload_time"`
	UploadAlt        string    `json:"uploadtime"`
}

// assrtLang is `lang`: an object {desc, langlist:{langcht:true,…}} on the live
// API, a plain string in older fixtures. Either decodes.
type assrtLang struct {
	Desc  string
	Flags map[string]bool
}

func (l *assrtLang) UnmarshalJSON(b []byte) error {
	trimmed := bytes.TrimSpace(b)
	if len(trimmed) == 0 || string(trimmed) == "null" {
		return nil
	}
	if trimmed[0] == '"' {
		return json.Unmarshal(trimmed, &l.Desc)
	}
	var obj struct {
		Desc     string          `json:"desc"`
		LangList map[string]bool `json:"langlist"`
	}
	if err := json.Unmarshal(trimmed, &obj); err != nil {
		return nil // an unexpected shape costs the language hint, not the row
	}
	l.Desc, l.Flags = obj.Desc, obj.LangList
	return nil
}

// id returns the identifier /sub/detail takes, "" when the row has none.
func (it assrtSearchItem) id() string {
	if it.ID > 0 {
		return strconv.Itoa(it.ID)
	}
	return strings.TrimSpace(it.FileID)
}

func (it assrtSearchItem) name() string {
	for _, n := range []string{it.NativeName, it.VideoChineseName, it.VideoName} {
		if strings.TrimSpace(n) != "" {
			return n
		}
	}
	return ""
}

// language maps Assrt's flags onto the tags the scorer knows: Traditional
// wins, then Simplified; a bilingual-only row is Chinese of unknown script.
// A legacy string `lang` is passed through unchanged.
func (it assrtSearchItem) language() string {
	flags := map[string]bool{}
	for k, v := range it.Lang.Flags {
		flags[k] = v
	}
	for _, k := range it.LangFlags {
		flags[k] = true
	}
	switch {
	case flags["langcht"]:
		return "zh-Hant"
	case flags["langchs"]:
		return "zh-Hans"
	case flags["langdou"]:
		return "zh"
	case flags["langeng"]:
		return "en"
	case len(flags) == 0:
		return it.Lang.Desc
	default:
		return ""
	}
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

// assrtDetailFile is one file inside a result. The live API sends its size as
// a display string ("138KB"), so the size is not decoded at all — a typed
// int64 here made every multi-file detail response fail to parse.
type assrtDetailFile struct {
	URL      string `json:"url"`
	Filename string `json:"f"`
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
	token := p.token(ctx)
	if token == "" {
		slog.Info("Assrt search skipped — no API token configured")
		return nil, nil
	}

	if query.Title == "" {
		return nil, fmt.Errorf("assrt: search title is required")
	}

	if err := p.rateLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("assrt rate limiter: %w", err)
	}

	u, err := url.Parse(p.baseURL() + "/sub/search")
	if err != nil {
		return nil, fmt.Errorf("assrt: invalid URL: %w", err)
	}

	q := u.Query()
	q.Set("token", token)
	switch {
	case query.Season > 0:
		// A season tag finds both season packs and single episodes; a year
		// would only narrow a show's index rows to nothing.
		q.Set("q", fmt.Sprintf("%s S%02d", query.Title, query.Season))
	case query.Year > 0:
		q.Set("q", fmt.Sprintf("%s %d", query.Title, query.Year))
	default:
		q.Set("q", query.Title)
	}
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("assrt: create request: %w", sanitizeTokenError(err))
	}
	req.Header.Set("User-Agent", assrtUserAgent)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("assrt: search request failed: %w", sanitizeTokenError(err))
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, assrtMaxResponseBytes))
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
		id := item.id()
		if id == "" {
			continue // nothing /sub/detail could fetch
		}
		upload := item.Upload
		if upload == "" {
			upload = item.UploadAlt
		}
		uploadDate, _ := time.Parse("2006-01-02 15:04:05", upload)

		results = append(results, SubtitleResult{
			ID:         id,
			Source:     "assrt",
			Filename:   item.name(), // P1-011: native_name first
			Language:   item.language(),
			UploadDate: uploadDate,
		})
	}

	return results, nil
}

// Download fetches the subtitle file content by ID, picking the best file when
// the result is a pack (see DownloadFor).
func (p *AssrtProvider) Download(ctx context.Context, id string) ([]byte, error) {
	return p.DownloadFor(ctx, id, "")
}

// DownloadFor fetches the subtitle for one video. It reads the result's detail,
// then downloads a single subtitle file from its file list — the one whose
// SxxEyy matches mediaFilePath when the result is a season pack, Traditional
// Chinese first. The archive itself is never returned: nothing downstream can
// unpack it.
func (p *AssrtProvider) DownloadFor(ctx context.Context, id, mediaFilePath string) ([]byte, error) {
	token := p.token(ctx)
	if token == "" {
		return nil, fmt.Errorf("assrt: provider is disabled — no API key configured")
	}

	// Step 1: Get subtitle detail to find download URL
	ReportProgress(ctx, 0.05)
	if err := waitWithProgress(ctx, p.rateLimiter, 0.05, 0.2); err != nil {
		return nil, fmt.Errorf("assrt rate limiter: %w", err)
	}

	u, err := url.Parse(p.baseURL() + "/sub/detail")
	if err != nil {
		return nil, fmt.Errorf("assrt: invalid URL: %w", err)
	}

	q := u.Query()
	q.Set("id", id)
	q.Set("token", token)
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("assrt: create detail request: %w", sanitizeTokenError(err))
	}
	req.Header.Set("User-Agent", assrtUserAgent)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("assrt: detail request failed: %w", sanitizeTokenError(err))
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, assrtMaxResponseBytes))
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

	detail := detailResp.Sub.Subs[0]
	file, err := pickAssrtFile(detail, mediaFilePath)
	if err != nil {
		return nil, fmt.Errorf("assrt: subtitle %s: %w", id, err)
	}
	ReportProgress(ctx, 0.25)

	data, fetchErr := p.fetchFile(ctx, file.URL, 0.25, 0.4)
	if fetchErr == nil {
		ReportProgress(ctx, 1)
		return data, nil
	}
	// Assrt extracts one file from a pack on the fly, but not from RAR packs:
	// those answer HTTP 493「服务器抽风」(live-verified 2026-11-02 on 天蝎 S01,
	// a 197-file .rar). Fall back to the pack itself and extract the file here.
	packURL := detail.URL
	isPack := hasExt(detail.Filename, assrtArchiveExts) || hasExt(packURL, assrtArchiveExts)
	if !isPack || packURL == "" || packURL == file.URL {
		return nil, fmt.Errorf("assrt: subtitle %s: %w", id, fetchErr)
	}
	slog.Info("Assrt single-file download failed, extracting from the pack",
		"id", id, "file", file.Filename, "error", fetchErr)
	ReportProgress(ctx, 0.4)
	archive, err := p.fetchFile(ctx, packURL, 0.4, 0.9)
	if err != nil {
		return nil, fmt.Errorf("assrt: subtitle %s: file failed (%v), pack failed: %w", id, fetchErr, err)
	}
	data, err = extractSubtitle(archive, file.Filename, mediaFilePath)
	if err != nil {
		return nil, fmt.Errorf("assrt: subtitle %s: %w", id, err)
	}
	ReportProgress(ctx, 1)
	return data, nil
}

// fetchFile downloads one file from Assrt's file server, reporting the body's
// progress as from→to of the whole download. A 200 that carries Assrt's
// XML/HTML error page instead of a file counts as a failure.
func (p *AssrtProvider) fetchFile(ctx context.Context, fileURL string, from, to float64) ([]byte, error) {
	// The quota wait takes the first part of the span, the body the rest.
	mid := from + (to-from)*0.4
	if err := waitWithProgress(ctx, p.rateLimiter, from, mid); err != nil {
		return nil, fmt.Errorf("assrt rate limiter: %w", err)
	}
	from = mid

	dlReq, err := http.NewRequestWithContext(ctx, http.MethodGet, fileURL, nil)
	if err != nil {
		return nil, fmt.Errorf("assrt: create download request: %w", err)
	}
	dlReq.Header.Set("User-Agent", assrtUserAgent)

	dlResp, err := p.httpClient.Do(dlReq)
	if err != nil {
		return nil, fmt.Errorf("assrt: download request failed: %w", err)
	}
	defer dlResp.Body.Close()

	if dlResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("assrt: download returned HTTP %d", dlResp.StatusCode)
	}

	body := &progressReader{r: dlResp.Body, ctx: ctx, total: dlResp.ContentLength, from: from, to: to}
	data, err := io.ReadAll(io.LimitReader(body, assrtMaxDownloadBytes))
	if err != nil {
		return nil, fmt.Errorf("assrt: read download response: %w", err)
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("assrt: downloaded empty subtitle file")
	}
	head := strings.ToLower(string(bytes.TrimSpace(data[:min(len(data), 64)])))
	if strings.HasPrefix(head, "<?xml") || strings.HasPrefix(head, "<html") || strings.HasPrefix(head, "<!doctype") {
		return nil, fmt.Errorf("assrt: download returned an error page")
	}
	return data, nil
}

var (
	assrtSubtitleExts = []string{".srt", ".ass", ".ssa", ".vtt"}
	assrtArchiveExts  = []string{".rar", ".zip", ".7z"}
	assrtEpisodeTag   = regexp.MustCompile(`(?i)s(\d{1,2})[ ._-]?e(\d{1,3})`)
	// Markers release groups put in subtitle file names (matched lower-cased).
	assrtTraditionalMarks = []string{"繁", "cht", "big5", "zh-tw", "zh-hant", ".tc.", "_tc.", "-tc."}
	assrtChineseMarks     = []string{"简", "簡", "chs", "gb.", "zh-cn", "zh-hans", ".sc.", "_sc.", "-sc.", "中文"}
	assrtEnglishMarks     = []string{".eng.", ".en.", "_eng.", "-eng.", "英文", "english"}
	assrtBilingualMarks   = []string{"双语", "雙語", "中英"}
)

func hasExt(name string, exts []string) bool {
	lower := strings.ToLower(name)
	for _, ext := range exts {
		if strings.HasSuffix(lower, ext) {
			return true
		}
	}
	return false
}

func containsAny(name string, marks []string) bool {
	lower := strings.ToLower(name)
	for _, m := range marks {
		if strings.Contains(lower, m) {
			return true
		}
	}
	return false
}

// episodeKey returns "s01e02" for a name carrying an SxxEyy tag, else "".
func episodeKey(name string) string {
	m := assrtEpisodeTag.FindStringSubmatch(name)
	if m == nil {
		return ""
	}
	season, _ := strconv.Atoi(m[1])
	episode, _ := strconv.Atoi(m[2])
	return fmt.Sprintf("s%02de%02d", season, episode)
}

// pickAssrtFile chooses the one file worth downloading from a detail row.
//
//   - Only subtitle files from the file list are candidates.
//   - When the video names an episode and the candidates name episodes, only
//     that episode's files stay; a pack without it is an error, never a
//     neighbouring episode's subtitle.
//   - Among what is left, the lowest assrtFileRank wins (Traditional, single
//     language, .srt).
//   - No listed subtitle file: the row's own URL, unless it is an archive.
func pickAssrtFile(detail assrtDetailItem, mediaFilePath string) (assrtDetailFile, error) {
	var candidates []assrtDetailFile
	for _, f := range detail.Filelist {
		if f.URL != "" && hasExt(f.Filename, assrtSubtitleExts) {
			candidates = append(candidates, f)
		}
	}

	if want := episodeKey(filepath.Base(mediaFilePath)); want != "" && len(candidates) > 0 {
		var matching []assrtDetailFile
		named := false
		for _, f := range candidates {
			key := episodeKey(f.Filename)
			named = named || key != ""
			if key == want {
				matching = append(matching, f)
			}
		}
		switch {
		case len(matching) > 0:
			candidates = matching
		case named:
			return assrtDetailFile{}, fmt.Errorf("pack has no file for %s", strings.ToUpper(want))
		}
	}

	if len(candidates) > 0 {
		best := candidates[0]
		for _, f := range candidates[1:] {
			if assrtFileRank(f.Filename) < assrtFileRank(best.Filename) {
				best = f
			}
		}
		return best, nil
	}

	if detail.URL == "" {
		return assrtDetailFile{}, fmt.Errorf("no download URL")
	}
	if hasExt(detail.Filename, assrtArchiveExts) || hasExt(detail.URL, assrtArchiveExts) {
		return assrtDetailFile{}, fmt.Errorf("archive %q lists no subtitle file", detail.Filename)
	}
	return assrtDetailFile{URL: detail.URL, Filename: detail.Filename}, nil
}

// assrtFileRank orders a pack's files for a zh-TW viewer, lower is better:
// Traditional Chinese, then other Chinese, then unmarked, then English only;
// within each, a single-language file before a bilingual one, .srt first.
func assrtFileRank(name string) int {
	traditional := containsAny(name, assrtTraditionalMarks)
	chinese := traditional || containsAny(name, assrtChineseMarks)
	english := containsAny(name, assrtEnglishMarks)

	rank := 15 // unmarked
	switch {
	case traditional:
		rank = 0
	case chinese:
		rank = 10
	case english:
		rank = 20
	}
	if chinese && (english || strings.Contains(name, "&") || containsAny(name, assrtBilingualMarks)) {
		rank += 2
	}
	if !hasExt(name, []string{".srt"}) {
		rank++
	}
	return rank
}

// sanitizeTokenError removes the API token from error messages to prevent
// accidental secret leakage in logs or error responses.
func sanitizeTokenError(err error) error {
	if err == nil {
		return nil
	}
	msg := err.Error()
	// Look for token= query parameter and redact its value
	if idx := strings.Index(msg, "token="); idx >= 0 {
		end := strings.IndexAny(msg[idx:], "&\" ")
		if end < 0 {
			end = len(msg) - idx
		}
		msg = msg[:idx] + "token=REDACTED" + msg[idx+end:]
	}
	return fmt.Errorf("%s", msg)
}

// Compile-time interface verification.
var _ SubtitleProvider = (*AssrtProvider)(nil)
