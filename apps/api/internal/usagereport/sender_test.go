package usagereport

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func fixture(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile("testdata/" + name)
	require.NoError(t, err)
	return b
}

type captured struct {
	method, path, ua, contentType string
	body                          []byte
}

func umami(t *testing.T, status int, body []byte, got *captured) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		*got = captured{r.Method, r.URL.Path, r.UserAgent(), r.Header.Get("Content-Type"), b}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write(body)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestSender_AcceptedWhenResponseCarriesSessionID(t *testing.T) {
	var got captured
	srv := umami(t, http.StatusOK, fixture(t, "accepted_200.json"), &got)
	s := NewSender(srv.URL, "0.1.2")

	err := s.Send(context.Background(), []byte(`{"type":"event"}`))

	require.NoError(t, err)
	assert.Equal(t, http.MethodPost, got.method)
	assert.Equal(t, "/api/send", got.path)
	assert.Equal(t, "application/json", got.contentType)
	assert.Equal(t, `{"type":"event"}`, string(got.body), "the exact bytes are sent — the settings page shows these same bytes")
	assert.Equal(t, "Mozilla/5.0 (X11; Linux x86_64) Vido/0.1.2", got.ua)
}

// Umami answers a request its bot filter flags with HTTP 200 {"beep":"boop"}
// and records nothing. Treating that as success would mark every dropped
// report as sent.
func TestSender_BotDroppedTwoHundredIsAFailure(t *testing.T) {
	var got captured
	srv := umami(t, http.StatusOK, fixture(t, "bot_dropped_200.json"), &got)

	err := NewSender(srv.URL, "dev").Send(context.Background(), []byte(`{}`))

	assert.ErrorIs(t, err, ErrNotRecorded)
}

func TestSender_WebsiteNotFoundIsAFailure(t *testing.T) {
	var got captured
	srv := umami(t, http.StatusBadRequest, fixture(t, "website_not_found_400.json"), &got)

	err := NewSender(srv.URL, "dev").Send(context.Background(), []byte(`{}`))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "400")
}

func TestSender_UnreachableIsAFailure(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	url := srv.URL
	srv.Close()

	err := NewSender(url, "dev").Send(context.Background(), []byte(`{}`))

	require.Error(t, err)
}

func TestSender_TimesOut(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
		case <-time.After(500 * time.Millisecond):
		}
	}))
	t.Cleanup(srv.Close)
	s := NewSender(srv.URL, "dev")
	s.client.Timeout = 50 * time.Millisecond

	err := s.Send(context.Background(), []byte(`{}`))

	require.Error(t, err)
}

func TestSender_UserAgentIsNotOneUmamiFlagsAsABot(t *testing.T) {
	// The isbot list Umami uses flags "Go-http-client/1.1", "Vido/1.0" and any
	// "compatible;" form; this shape was checked against isbot 5.2.2.
	assert.Equal(t, "Mozilla/5.0 (X11; Linux x86_64) Vido/0.1.2", userAgent("0.1.2"))
	assert.NotContains(t, userAgent("0.1.2"), "compatible")
	assert.NotContains(t, userAgent("0.1.2"), "bot")
}

func TestValidEndpoint(t *testing.T) {
	assert.True(t, ValidEndpoint("https://analytics.example.com"))
	assert.True(t, ValidEndpoint("http://192.168.50.52:3000"))
	assert.False(t, ValidEndpoint("analytics.example.com"), "no scheme — every send would fail quietly")
	assert.False(t, ValidEndpoint("ftp://analytics.example.com"))
	assert.False(t, ValidEndpoint("https://"))
	assert.False(t, ValidEndpoint(""))
}

func TestSender_RedirectIsAFailureNotARepost(t *testing.T) {
	var reposted bool
	elsewhere := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { reposted = true }))
	t.Cleanup(elsewhere.Close)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, elsewhere.URL+"/api/send", http.StatusTemporaryRedirect)
	}))
	t.Cleanup(srv.Close)

	err := NewSender(srv.URL, "dev").Send(context.Background(), []byte(`{}`))

	require.Error(t, err)
	assert.False(t, reposted, "the body must not be re-posted to the redirect target")
}

func TestSender_FailedReplyIsTruncatedInTheError(t *testing.T) {
	var got captured
	srv := umami(t, http.StatusBadGateway, []byte(strings.Repeat("x", 10_000)), &got)

	err := NewSender(srv.URL, "dev").Send(context.Background(), []byte(`{}`))

	require.Error(t, err)
	assert.Less(t, len(err.Error()), 400, "a 10 KB error page must not land in the DB log")
}
