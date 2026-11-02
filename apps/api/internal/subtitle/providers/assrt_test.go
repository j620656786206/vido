package providers

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"golang.org/x/time/rate"
)

// --- Test helpers ---

func searchResponse(subs []assrtSearchItem) assrtSearchResponse {
	return assrtSearchResponse{
		Status: 0,
		Sub:    &assrtSearchSub{Subs: subs},
	}
}

func detailResponse(subs []assrtDetailItem) assrtDetailResponse {
	return assrtDetailResponse{
		Status: 0,
		Sub:    &assrtDetailSub{Subs: subs},
	}
}

// --- Tests ---

func TestAssrtProvider_Name(t *testing.T) {
	p := NewAssrtProvider(staticAssrtKey("test-key"))
	assert.Equal(t, "assrt", p.Name())
}

func TestAssrtProvider_SearchSuccess_NativeName(t *testing.T) {
	// P1-011 fix: verify native_name is used, not name
	items := []assrtSearchItem{
		{
			ID:         12345,
			NativeName: "進擊的巨人 第01話",
			VideoName:  "Attack on Titan S01E01",
			Lang:       assrtLang{Desc: "Chn"},
			Upload:     "2024-01-15 10:30:00",
		},
		{
			ID:         12346,
			NativeName: "進擊的巨人 第02話",
			VideoName:  "Attack on Titan S01E02",
			Lang:       assrtLang{Desc: "Chn"},
			Upload:     "2024-01-16 11:00:00",
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/sub/search", r.URL.Path)
		assert.Equal(t, "test-key", r.URL.Query().Get("token"))

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(searchResponse(items))
	}))
	defer server.Close()

	p := newTestAssrtProvider("test-key", server.URL)

	results, err := p.Search(context.Background(), SubtitleQuery{Title: "進擊的巨人"})
	require.NoError(t, err)
	require.Len(t, results, 2)

	// Verify P1-011: Filename uses native_name
	assert.Equal(t, "進擊的巨人 第01話", results[0].Filename)
	assert.Equal(t, "12345", results[0].ID)
	assert.Equal(t, "assrt", results[0].Source)
	assert.Equal(t, "Chn", results[0].Language)

	assert.Equal(t, "進擊的巨人 第02話", results[1].Filename)
}

// The live API's real shapes (captured 2026-11-02, trimmed): an index row with
// a string `fileid` and `m_langn`, a documented row with `lang` as an object,
// and `"subs": {}` for no match. The old decoder failed on all three.
func TestAssrtProvider_SearchLiveShapes(t *testing.T) {
	const live = `{"status":0,"sub":{"action":"search","subs":[
	  {"fileid":"714057","video_chinese_name":"天蝎 第一季","videoname":"Scorpion.S01","uploadtime":"2026-02-19 16:39:33","m_langn":["langeng","langchs","langdou"]},
	  {"id":635634,"native_name":"天蝎 第一季 WEB精校版","videoname":"Scorpion.S01.720p","upload_time":"2019-07-16 22:02:00","lang":{"desc":"繁 简 双语","langlist":{"langcht":true,"langchs":true,"langdou":true}}},
	  {"id":1,"native_name":"雙語","lang":{"desc":"双语","langlist":{"langdou":true}}},
	  {"native_name":"no id at all"}
	],"result":"succeed"}}`
	const empty = `{"status":0,"sub":{"action":"search","subs":{},"result":"succeed","keyword":"天蠍行動 2014"}}`

	body := live
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}))
	defer server.Close()
	p := newTestAssrtProvider("test-key", server.URL)

	results, err := p.Search(context.Background(), SubtitleQuery{Title: "Scorpion"})
	require.NoError(t, err)
	require.Len(t, results, 3, "a row without any id is skipped")

	assert.Equal(t, "714057", results[0].ID)
	assert.Equal(t, "天蝎 第一季", results[0].Filename)
	assert.Equal(t, "zh-Hans", results[0].Language)
	assert.Equal(t, 2026, results[0].UploadDate.Year())

	assert.Equal(t, "635634", results[1].ID)
	assert.Equal(t, "天蝎 第一季 WEB精校版", results[1].Filename)
	assert.Equal(t, "zh-Hant", results[1].Language, "Traditional wins when a pack has both")

	assert.Equal(t, "zh", results[2].Language, "bilingual-only is Chinese of unknown script")

	body = empty
	results, err = p.Search(context.Background(), SubtitleQuery{Title: "天蠍行動"})
	require.NoError(t, err, `"subs": {} means no match, not a parse error`)
	assert.Empty(t, results)
}

// disc-2026-11-online-subs-for-episodes: a season query is "Title Sxx" with no
// year — 「Scorpion S01」 finds the 天蝎 pack, 「Scorpion 2014」 finds nothing.
func TestAssrtProvider_SearchSeasonQuery(t *testing.T) {
	var gotQ string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQ = r.URL.Query().Get("q")
		_, _ = w.Write([]byte(`{"status":0,"sub":{"subs":{}}}`))
	}))
	defer server.Close()
	p := newTestAssrtProvider("test-key", server.URL)

	_, err := p.Search(context.Background(), SubtitleQuery{Title: "Scorpion", Year: 2014, Season: 1, Episode: 2})
	require.NoError(t, err)
	assert.Equal(t, "Scorpion S01", gotQ)
}

func TestAssrtProvider_SearchEmptyTitle(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("should not reach server with empty title")
	}))
	defer server.Close()

	p := newTestAssrtProvider("test-key", server.URL)

	results, err := p.Search(context.Background(), SubtitleQuery{Title: ""})
	assert.Error(t, err)
	assert.Nil(t, results)
	assert.Contains(t, err.Error(), "title is required")
}

func TestAssrtProvider_SearchDisabled(t *testing.T) {
	// When API key is not configured, provider should return empty, not error
	p := NewAssrtProvider(staticAssrtKey(""))

	results, err := p.Search(context.Background(), SubtitleQuery{Title: "test"})
	assert.NoError(t, err)
	assert.Nil(t, results)
}

func TestAssrtProvider_SearchWithYear(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query().Get("q")
		assert.Contains(t, q, "2024")

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(searchResponse(nil))
	}))
	defer server.Close()

	p := newTestAssrtProvider("test-key", server.URL)

	results, err := p.Search(context.Background(), SubtitleQuery{Title: "Dune", Year: 2024})
	require.NoError(t, err)
	assert.Empty(t, results)
}

func TestAssrtProvider_SearchHTTPError(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
	}{
		{"400 Bad Request", http.StatusBadRequest},
		{"404 Not Found", http.StatusNotFound},
		{"500 Internal Server Error", http.StatusInternalServerError},
		{"503 Service Unavailable", http.StatusServiceUnavailable},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
			}))
			defer server.Close()

			p := newTestAssrtProvider("test-key", server.URL)

			results, err := p.Search(context.Background(), SubtitleQuery{Title: "test"})
			assert.Error(t, err)
			assert.Nil(t, results)
			assert.Contains(t, err.Error(), fmt.Sprintf("HTTP %d", tt.statusCode))
		})
	}
}

func TestAssrtProvider_SearchMalformedJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{invalid json`))
	}))
	defer server.Close()

	p := newTestAssrtProvider("test-key", server.URL)

	results, err := p.Search(context.Background(), SubtitleQuery{Title: "test"})
	assert.Error(t, err)
	assert.Nil(t, results)
	assert.Contains(t, err.Error(), "failed to parse")
}

func TestAssrtProvider_SearchEmptySubField(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(assrtSearchResponse{Status: 0, Sub: nil})
	}))
	defer server.Close()

	p := newTestAssrtProvider("test-key", server.URL)

	results, err := p.Search(context.Background(), SubtitleQuery{Title: "nonexistent"})
	assert.NoError(t, err)
	assert.Nil(t, results)
}

func TestAssrtProvider_DownloadSuccess(t *testing.T) {
	subtitleContent := []byte("1\n00:00:01,000 --> 00:00:03,000\n你好世界\n")

	var requestCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&requestCount, 1)

		if count == 1 {
			// Detail request
			assert.Equal(t, "/v1/sub/detail", r.URL.Path)
			assert.Equal(t, "42", r.URL.Query().Get("id"))

			resp := detailResponse([]assrtDetailItem{
				{
					ID: 42,
					Filelist: []assrtDetailFile{
						{URL: fmt.Sprintf("http://%s/download/42.srt", r.Host), Filename: "test.srt"},
					},
				},
			})

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
		} else {
			// Download request
			assert.Equal(t, "/download/42.srt", r.URL.Path)
			w.Write(subtitleContent)
		}
	}))
	defer server.Close()

	p := newTestAssrtProvider("test-key", server.URL)

	data, err := p.Download(context.Background(), "42")
	require.NoError(t, err)
	assert.Equal(t, subtitleContent, data)
}

// pickAssrtFile — a season pack must yield the requested episode's file
// (Traditional first), never the .rar, never a neighbouring episode.
func TestPickAssrtFile(t *testing.T) {
	pack := assrtDetailItem{
		Filename: "Scorpion.S01.rar",
		URL:      "http://x/download/1/Scorpion.S01.rar",
		Filelist: []assrtDetailFile{
			{Filename: "Scorpion.S01E01.chs.srt", URL: "http://x/e01chs"},
			{Filename: "Scorpion.S01E02.eng.srt", URL: "http://x/e02eng"},
			{Filename: "Scorpion.S01E02.chs.srt", URL: "http://x/e02chs"},
			{Filename: "Scorpion.S01E02.cht.ass", URL: "http://x/e02cht"},
			{Filename: "readme.txt", URL: "http://x/readme"},
		},
	}

	cases := []struct {
		name    string
		detail  assrtDetailItem
		media   string
		want    string
		wantErr string
	}{
		{name: "episode match, Traditional first", detail: pack, media: "/tv/Scorpion/Season 1/Scorpion.S01E02.1080p.mkv", want: "http://x/e02cht"},
		{name: "s1e1 spelled loosely", detail: pack, media: "/tv/天蠍/天蠍行動 s01.e01.mkv", want: "http://x/e01chs"},
		{name: "pack lacks the episode", detail: pack, media: "/tv/Scorpion.S01E09.mkv", wantErr: "S01E09"},
		{name: "no video path: best file overall", detail: pack, media: "", want: "http://x/e02cht"},
		{name: "single file, no episode tags", media: "/movies/Film.2020.mkv",
			detail: assrtDetailItem{Filelist: []assrtDetailFile{{Filename: "Film.srt", URL: "http://x/film"}}}, want: "http://x/film"},
		{name: "plain file without a list", media: "/movies/Film.mkv",
			detail: assrtDetailItem{Filename: "Film.srt", URL: "http://x/film.srt"}, want: "http://x/film.srt"},
		{name: "archive without a list is refused", media: "/movies/Film.mkv",
			detail: assrtDetailItem{Filename: "Film.zip", URL: "http://x/film.zip"}, wantErr: "archive"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := pickAssrtFile(tc.detail, tc.media)
			if tc.wantErr != "" {
				assert.ErrorContains(t, err, tc.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want, got.URL)
		})
	}
}

// The real 天蝎 S01E02 entries (BS pack, 2026-11-02): pure Traditional .srt
// beats the bilingual and .ass variants; Simplified beats English-only.
func TestAssrtFileRank_RealPackNames(t *testing.T) {
	const base = "Scorpion.S01E02.Single.Point.of.Failure.720p.WEB-DL.DD5.1.H.264-BS."
	order := []string{"繁体.srt", "繁体.ass", "繁体&英文.srt", "繁体&英文.ass", "简体.srt", "简体&英文.srt", "英文.srt"}
	for i := 1; i < len(order); i++ {
		assert.Less(t, assrtFileRank(base+order[i-1]), assrtFileRank(base+order[i]),
			"%s should rank before %s", order[i-1], order[i])
	}
}

// Assrt cannot extract from RAR packs on the fly (HTTP 493); the provider
// then downloads the pack and extracts the file itself.
func TestAssrtProvider_DownloadForFallsBackToThePack(t *testing.T) {
	var zipped bytes.Buffer
	zw := zip.NewWriter(&zipped)
	for name, body := range map[string]string{
		"Show.S01E01.繁体.srt": "1\n00:00:01,000 --> 00:00:02,000\n第一集\n",
		"Show.S01E02.繁体.srt": "1\n00:00:01,000 --> 00:00:02,000\n第二集\n",
		"readme.txt":         "x",
	} {
		w, err := zw.Create(name)
		require.NoError(t, err)
		_, _ = w.Write([]byte(body))
	}
	require.NoError(t, zw.Close())

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/sub/detail":
			fmt.Fprintf(w, `{"status":0,"sub":{"subs":[{"id":7,"filename":"Show.S01.zip","url":"http://%[1]s/download/pack.zip",
			  "filelist":[{"f":"Show.S01E01.繁体.srt","url":"http://%[1]s/onthefly/1","s":"1KB"},{"f":"Show.S01E02.繁体.srt","url":"http://%[1]s/onthefly/2","s":"1KB"}]}]}}`, r.Host)
		case "/onthefly/2":
			w.WriteHeader(493)
		case "/download/pack.zip":
			_, _ = w.Write(zipped.Bytes())
		default:
			t.Errorf("unexpected request %s", r.URL.Path)
		}
	}))
	defer server.Close()
	p := newTestAssrtProvider("test-key", server.URL)

	var seen []float64
	ctx := WithProgress(context.Background(), func(f float64) { seen = append(seen, f) })
	data, err := p.DownloadFor(ctx, "7", "/tv/Show/Show.S01E02.mkv")
	require.NoError(t, err)
	assert.Contains(t, string(data), "第二集")

	// disc-2026-11-online-sub-download-progress: the long pack path reports
	// all the way through, never going backwards, and ends at 100%.
	require.NotEmpty(t, seen)
	for i := 1; i < len(seen); i++ {
		assert.GreaterOrEqual(t, seen[i], seen[i-1], "progress never goes backwards: %v", seen)
	}
	assert.Contains(t, seen, 0.4, "the switch to the pack is reported")
	assert.Equal(t, 1.0, seen[len(seen)-1])
}

// A pack whose member names no longer match the listing (GBK-encoded names)
// still yields the right episode by the same choice rule.
func TestExtractSubtitle_RenamedMembers(t *testing.T) {
	var zipped bytes.Buffer
	zw := zip.NewWriter(&zipped)
	for name, body := range map[string]string{"a.S01E01.srt": "one", "a.S01E02.srt": "two"} {
		w, _ := zw.Create(name)
		_, _ = w.Write([]byte(body))
	}
	require.NoError(t, zw.Close())

	data, err := extractSubtitle(zipped.Bytes(), "name-from-listing.srt", "/tv/a.S01E02.mkv")
	require.NoError(t, err)
	assert.Equal(t, "two", string(data))

	_, err = extractSubtitle([]byte("not an archive"), "x.srt", "")
	assert.ErrorContains(t, err, "neither")
}

// The live detail sends file sizes as "138KB"; the decoder must not choke.
func TestAssrtProvider_DownloadForLiveDetailShape(t *testing.T) {
	subtitleContent := []byte("1\n00:00:01,000 --> 00:00:03,000\n你好世界\n")
	var gotPaths []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPaths = append(gotPaths, r.URL.Path)
		if r.URL.Path == "/v1/sub/detail" {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{"status":0,"sub":{"action":"detail","subs":[{"id":635634,"filename":"Show.S01.rar","url":"http://%[1]s/download/pack.rar","size":763266,
			  "filelist":[{"f":"Show.S01E01.ass","url":"http://%[1]s/onthefly/1","s":"231KB"},{"f":"Show.S01E02.ass","url":"http://%[1]s/onthefly/2","s":"142KB"}]}]}}`, r.Host)
			return
		}
		_, _ = w.Write(subtitleContent)
	}))
	defer server.Close()
	p := newTestAssrtProvider("test-key", server.URL)

	data, err := p.DownloadFor(context.Background(), "635634", "/tv/Show/Show.S01E02.mkv")
	require.NoError(t, err)
	assert.Equal(t, subtitleContent, data)
	assert.Equal(t, []string{"/v1/sub/detail", "/onthefly/2"}, gotPaths)
}

func TestAssrtProvider_DownloadDisabled(t *testing.T) {
	p := NewAssrtProvider(staticAssrtKey(""))

	data, err := p.Download(context.Background(), "42")
	assert.Error(t, err)
	assert.Nil(t, data)
	assert.Contains(t, err.Error(), "disabled")
}

func TestAssrtProvider_DownloadNoURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := detailResponse([]assrtDetailItem{
			{ID: 42, URL: "", Filelist: nil},
		})
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p := newTestAssrtProvider("test-key", server.URL)

	data, err := p.Download(context.Background(), "42")
	assert.Error(t, err)
	assert.Nil(t, data)
	assert.Contains(t, err.Error(), "no download URL")
}

func TestAssrtProvider_DownloadDetailHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	p := newTestAssrtProvider("test-key", server.URL)

	data, err := p.Download(context.Background(), "42")
	assert.Error(t, err)
	assert.Nil(t, data)
	assert.Contains(t, err.Error(), "HTTP 500")
}

func TestAssrtProvider_DownloadEmptyFile(t *testing.T) {
	var requestCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&requestCount, 1)
		if count == 1 {
			resp := detailResponse([]assrtDetailItem{
				{ID: 42, URL: fmt.Sprintf("http://%s/download/42.srt", r.Host)},
			})
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
		} else {
			// Return empty body
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	p := newTestAssrtProvider("test-key", server.URL)

	data, err := p.Download(context.Background(), "42")
	assert.Error(t, err)
	assert.Nil(t, data)
	assert.Contains(t, err.Error(), "empty subtitle file")
}

func TestAssrtProvider_RateLimiter(t *testing.T) {
	var requestCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requestCount, 1)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(searchResponse(nil))
	}))
	defer server.Close()

	// A scaled-down limiter proves every Search consults the limiter without
	// paying the production 13s spacing in the test suite.
	p := newTestAssrtProvider("test-key", server.URL)
	p.rateLimiter = rate.NewLimiter(rate.Every(300*time.Millisecond), 1)

	// 4 sequential requests: 1st from burst, then 3 × 300ms waits ≈ 900ms.
	start := time.Now()
	for i := 0; i < 4; i++ {
		_, _ = p.Search(context.Background(), SubtitleQuery{Title: "test"})
	}
	elapsed := time.Since(start)

	assert.GreaterOrEqual(t, elapsed, 800*time.Millisecond, "rate limiter should throttle requests")
	assert.Equal(t, int32(4), atomic.LoadInt32(&requestCount))
}

// TestAssrtProvider_ProductionRateLimitMatchesQuota pins the production limiter
// to Assrt's REAL quota: 5 req/min per token (live-verified 2026-07-31; the
// official docs' 20/min is wrong). A regression here (e.g. restoring the old
// 2 req/s) would burn the whole quota on the first search and fail every
// subsequent call for the rest of the minute.
func TestAssrtProvider_ProductionRateLimitMatchesQuota(t *testing.T) {
	p := NewAssrtProvider(staticAssrtKey("test-key"))

	assert.Equal(t, rate.Every(assrtRateInterval), p.rateLimiter.Limit())
	assert.Equal(t, 1, p.rateLimiter.Burst())
	// Worst-case calls in any 60s window: burst + floor(60s/interval) must stay ≤ 5.
	worstCase := 1 + int((60*time.Second)/assrtRateInterval)
	assert.LessOrEqual(t, worstCase, 5, "limiter config must fit the 5 req/min quota")
}

func TestAssrtProvider_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Use the request context to detect client disconnect, avoiding
		// a long sleep that blocks server.Close() and slows CI.
		select {
		case <-r.Context().Done():
		case <-time.After(5 * time.Second):
		}
	}))
	defer server.Close()

	p := newTestAssrtProvider("test-key", server.URL)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err := p.Search(ctx, SubtitleQuery{Title: "test"})
	assert.Error(t, err)
}

// disc-2026-11-assrt-key-in-settings: the token is read on every call, so a
// token saved on the settings page applies to the next search without a
// restart, and clearing it idles the provider again.
func TestAssrtProvider_TokenReadPerCall(t *testing.T) {
	var gotTokens []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotTokens = append(gotTokens, r.URL.Query().Get("token"))
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(searchResponse(nil))
	}))
	defer server.Close()

	current := ""
	p := newTestAssrtProvider("", server.URL)
	p.apiKey = func(context.Context) string { return current }

	results, err := p.Search(context.Background(), SubtitleQuery{Title: "Scorpion"})
	require.NoError(t, err)
	assert.Nil(t, results)
	assert.Empty(t, gotTokens, "no token configured → no request")

	current = "saved-on-settings-page"
	_, err = p.Search(context.Background(), SubtitleQuery{Title: "Scorpion"})
	require.NoError(t, err)
	assert.Equal(t, []string{"saved-on-settings-page"}, gotTokens)

	current = ""
	_, err = p.Download(context.Background(), "42")
	assert.ErrorContains(t, err, "disabled")
	assert.Len(t, gotTokens, 1, "cleared token → back to idle")
}

// --- Helper ---

func staticAssrtKey(key string) func(context.Context) string {
	return func(context.Context) string { return key }
}

// newTestAssrtProvider creates an AssrtProvider pointing at a test server.
// It uses an effectively-unlimited rate limiter: the production limiter is
// 5 req/min (13s spacing), which would stall every multi-call test; limiter
// behaviour is covered by TestAssrtProvider_RateLimiter and the production
// config by TestAssrtProvider_ProductionRateLimitMatchesQuota.
func newTestAssrtProvider(apiKey, baseURL string) *AssrtProvider {
	p := &AssrtProvider{
		apiKey:      staticAssrtKey(apiKey),
		httpClient:  &http.Client{Timeout: 10 * time.Second},
		rateLimiter: rate.NewLimiter(rate.Inf, 1),
	}
	p.testBaseURL = baseURL + "/v1"
	return p
}
