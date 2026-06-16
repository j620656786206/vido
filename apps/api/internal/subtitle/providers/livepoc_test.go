package providers

// Live POC harness — exercises the REAL subtitle providers against the REAL
// upstream sites using vido's own code. Guarded by LIVE_POC=1 so it never runs
// in CI / normal `go test`.
//
//   LIVE_POC=1 go test ./internal/subtitle/providers/ -run TestLiveSubtitlePOC -v -count=1
//
// To extend the POC to the key-gated providers, export before running:
//   ASSRT_API_KEY=...           (free token from assrt.net)
//   OPENSUBTITLES_API_KEY=...   (free key from opensubtitles.com)
//   OPENSUBTITLES_USERNAME=...  OPENSUBTITLES_PASSWORD=...

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"
)

// envSecrets is a SecretsServiceInterface that serves keys from env vars so the
// POC can run the key-gated providers when the user supplies credentials.
type envSecrets struct{ m map[string]string }

func (e envSecrets) Store(context.Context, string, string) error { return nil }
func (e envSecrets) Retrieve(_ context.Context, name string) (string, error) {
	if v, ok := e.m[name]; ok && v != "" {
		return v, nil
	}
	return "", fmt.Errorf("secret %q not configured", name)
}
func (e envSecrets) Delete(context.Context, string) error         { return nil }
func (e envSecrets) Exists(context.Context, string) (bool, error) { return false, nil }
func (e envSecrets) List(context.Context) ([]string, error)       { return nil, nil }

func classifyBody(b []byte) string {
	s := string(b)
	switch {
	case strings.Contains(s, "YunsuoAutoJump") || strings.Contains(s, "stringToHex") || strings.Contains(s, "yunsuo"):
		return "WAF/anti-bot challenge (Yunsuo 雲鎖 JS interstitial)"
	case strings.Contains(s, "captcha") || strings.Contains(s, "验证码") || strings.Contains(s, "驗證碼") || strings.Contains(s, "recaptcha"):
		return "CAPTCHA challenge"
	case strings.Contains(s, "cloudflare") || strings.Contains(s, "cf-browser-verification") || strings.Contains(s, "Just a moment"):
		return "Cloudflare challenge"
	case strings.Contains(s, "search-result") || strings.Contains(s, "subtitle"):
		return "looks like real subtitle HTML"
	default:
		return "unknown HTML (no result markers, no challenge markers)"
	}
}

type posterChild struct {
	title   string
	year    int
	season  int
	episode int
}

func runProvider(t *testing.T, p SubtitleProvider, cases []posterChild) {
	for _, tc := range cases {
		q := SubtitleQuery{Title: tc.title, Year: tc.year, Season: tc.season, Episode: tc.episode, Languages: []string{"zh-Hant", "zh-Hans"}}
		ctx, cancel := context.WithTimeout(context.Background(), 35*time.Second)
		results, err := p.Search(ctx, q)
		cancel()

		label := tc.title
		if tc.season > 0 {
			label = fmt.Sprintf("%s S%02dE%02d", tc.title, tc.season, tc.episode)
		}
		fmt.Printf("\n  --- [%s] search %q ---\n", p.Name(), label)
		if err != nil {
			fmt.Printf("      SEARCH ERROR: %v\n", err)
			continue
		}
		fmt.Printf("      results: %d\n", len(results))
		for i, r := range results {
			if i >= 3 {
				fmt.Printf("      ... (%d more)\n", len(results)-3)
				break
			}
			fmt.Printf("      [%d] id=%s lang=%-8s dl=%-5d group=%q file=%q\n", i, r.ID, r.Language, r.Downloads, r.Group, r.Filename)
		}
		if len(results) == 0 {
			continue
		}
		// Attempt to actually download the top result and preview the content.
		ctx2, cancel2 := context.WithTimeout(context.Background(), 35*time.Second)
		data, derr := p.Download(ctx2, results[0].ID)
		cancel2()
		if derr != nil {
			fmt.Printf("      DOWNLOAD ERROR: %v\n", derr)
			continue
		}
		preview := string(data)
		if len(preview) > 500 {
			preview = preview[:500]
		}
		fmt.Printf("      DOWNLOADED %d bytes. content preview:\n      ┌─────\n", len(data))
		for _, line := range strings.Split(preview, "\n") {
			fmt.Printf("      │ %s\n", strings.TrimRight(line, "\r"))
		}
		fmt.Printf("      └─────\n")
	}
}

func TestLiveSubtitlePOC(t *testing.T) {
	if os.Getenv("LIVE_POC") != "1" {
		t.Skip("set LIVE_POC=1 to run the live subtitle POC")
	}
	ctx := context.Background()
	cases := []posterChild{
		{title: "Breaking Bad", year: 2008, season: 1, episode: 1},
		{title: "Oppenheimer", year: 2023},
		{title: "The Last of Us", year: 2023, season: 1, episode: 1},
		{title: "Friends", year: 1994, season: 1, episode: 1},
	}

	// ---- Provider 1: Zimuku (no key, web scraper) ----
	fmt.Println("\n========== ZIMUKU (字幕庫, keyless scraper) ==========")
	z := NewZimukuProvider()
	z.skipDelays = false // keep anti-ban delays — we want a realistic run
	runProvider(t, z, cases)

	// Raw classification of what the scraper actually receives from /search.
	rawURL := fmt.Sprintf("%s/search?q=%s", zimukuBaseURL, url.QueryEscape("Breaking Bad"))
	if req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil); err == nil {
		req.Header.Set("User-Agent", defaultUserAgents[0])
		req.Header.Set("Accept-Language", "zh-TW,zh;q=0.9,en;q=0.8")
		if resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(req); err == nil {
			body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
			resp.Body.Close()
			fmt.Printf("\n  RAW /search HTTP %d, body classified as: %s\n", resp.StatusCode, classifyBody(body))
			fmt.Printf("  detectCaptcha() would return: %v  (note: WAF page has NO captcha keyword → silent empty result)\n", detectCaptcha(body))
		}
	}

	// ---- Provider 2: Assrt (needs token) ----
	fmt.Println("\n========== ASSRT (射手網, needs API token) ==========")
	assrtSvc := envSecrets{m: map[string]string{assrtSecretKey: os.Getenv("ASSRT_API_KEY")}}
	assrt := NewAssrtProvider(ctx, assrtSvc)
	if assrt.disabled {
		fmt.Println("  DISABLED — no ASSRT_API_KEY in env. (export ASSRT_API_KEY=... to test)")
	} else {
		runProvider(t, assrt, cases)
	}

	// ---- Provider 3: OpenSubtitles (needs API key) ----
	fmt.Println("\n========== OPENSUBTITLES (needs API key + login) ==========")
	osSvc := envSecrets{m: map[string]string{
		openSubAPIKeySecret:   os.Getenv("OPENSUBTITLES_API_KEY"),
		openSubUsernameSecret: os.Getenv("OPENSUBTITLES_USERNAME"),
		openSubPasswordSecret: os.Getenv("OPENSUBTITLES_PASSWORD"),
	}}
	osp := NewOpenSubProvider(ctx, osSvc)
	if osp.disabled {
		fmt.Println("  DISABLED — no OPENSUBTITLES_API_KEY in env. (export OPENSUBTITLES_API_KEY=... to test)")
	} else {
		runProvider(t, osp, cases)
	}

	fmt.Println("\n========== POC DONE ==========")
}
