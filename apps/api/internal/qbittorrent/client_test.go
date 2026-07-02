package qbittorrent

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// qbtLoginOK is a mock login handler that sets a session cookie and returns Ok.
func qbtLoginOK(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{Name: "SID", Value: "test-session"})
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, "Ok.")
}

// newTestServer creates a mock qBittorrent API server for testing.
func newTestServer(t *testing.T, handler http.Handler) *httptest.Server {
	t.Helper()
	return httptest.NewServer(handler)
}

func TestNewClient(t *testing.T) {
	cfg := &Config{
		Host:     "http://localhost:8080",
		Username: "admin",
		Password: "password",
	}

	client := NewClient(cfg)

	assert.NotNil(t, client)
	assert.Equal(t, cfg, client.config)
	assert.NotNil(t, client.httpClient)
	assert.Equal(t, 10*time.Second, client.httpClient.Timeout)
}

func TestNewClient_CustomTimeout(t *testing.T) {
	cfg := &Config{
		Host:    "http://localhost:8080",
		Timeout: 5 * time.Second,
	}

	client := NewClient(cfg)

	assert.Equal(t, 5*time.Second, client.httpClient.Timeout)
}

func TestClient_BuildURL(t *testing.T) {
	tests := []struct {
		name     string
		config   *Config
		path     string
		expected string
	}{
		{
			name:     "basic URL without base path",
			config:   &Config{Host: "http://192.168.1.100:8080"},
			path:     "/auth/login",
			expected: "http://192.168.1.100:8080/api/v2/auth/login",
		},
		{
			name:     "URL with base path",
			config:   &Config{Host: "http://192.168.1.100:8080", BasePath: "/qbittorrent"},
			path:     "/auth/login",
			expected: "http://192.168.1.100:8080/qbittorrent/api/v2/auth/login",
		},
		{
			name:     "URL with trailing slash in base path",
			config:   &Config{Host: "http://192.168.1.100:8080", BasePath: "/qbittorrent/"},
			path:     "/app/version",
			expected: "http://192.168.1.100:8080/qbittorrent/api/v2/app/version",
		},
		{
			name:     "HTTPS URL",
			config:   &Config{Host: "https://nas.example.com"},
			path:     "/app/version",
			expected: "https://nas.example.com/api/v2/app/version",
		},
		{
			name:     "host with trailing slash",
			config:   &Config{Host: "http://localhost:8080/"},
			path:     "/auth/login",
			expected: "http://localhost:8080/api/v2/auth/login",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClient(tt.config)
			result := client.buildURL(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestClient_Login_Success(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v2/auth/login", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "application/x-www-form-urlencoded", r.Header.Get("Content-Type"))

		err := r.ParseForm()
		require.NoError(t, err)
		assert.Equal(t, "admin", r.FormValue("username"))
		assert.Equal(t, "password123", r.FormValue("password"))

		http.SetCookie(w, &http.Cookie{Name: "SID", Value: "test-session-id"})
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "Ok.")
	})

	server := newTestServer(t, mux)
	defer server.Close()

	client := NewClient(&Config{
		Host:     server.URL,
		Username: "admin",
		Password: "password123",
	})

	err := client.Login(context.Background())
	assert.NoError(t, err)
}

func TestClient_Login_InvalidCredentials(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v2/auth/login", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "Fails.")
	})

	server := newTestServer(t, mux)
	defer server.Close()

	client := NewClient(&Config{
		Host:     server.URL,
		Username: "admin",
		Password: "wrong",
	})

	err := client.Login(context.Background())
	assert.Error(t, err)

	var connErr *ConnectionError
	assert.ErrorAs(t, err, &connErr)
	assert.Equal(t, ErrCodeAuthFailed, connErr.Code)
}

func TestClient_Login_ServerError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v2/auth/login", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		fmt.Fprint(w, "Forbidden")
	})

	server := newTestServer(t, mux)
	defer server.Close()

	client := NewClient(&Config{
		Host:     server.URL,
		Username: "admin",
		Password: "password",
	})

	err := client.Login(context.Background())
	assert.Error(t, err)

	var connErr *ConnectionError
	assert.ErrorAs(t, err, &connErr)
	assert.Equal(t, ErrCodeConnectionFailed, connErr.Code)
}

func TestClient_Login_ConnectionRefused(t *testing.T) {
	client := NewClient(&Config{
		Host:    "http://127.0.0.1:1",
		Timeout: 1 * time.Second,
	})

	err := client.Login(context.Background())
	assert.Error(t, err)

	var connErr *ConnectionError
	assert.ErrorAs(t, err, &connErr)
	assert.Equal(t, ErrCodeConnectionFailed, connErr.Code)
}

func TestClient_TestConnection_Success(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v2/auth/login", func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{Name: "SID", Value: "test-session"})
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "Ok.")
	})
	mux.HandleFunc("/api/v2/app/version", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "v4.5.2")
	})
	mux.HandleFunc("/api/v2/app/webapiVersion", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "2.9.3")
	})

	server := newTestServer(t, mux)
	defer server.Close()

	client := NewClient(&Config{
		Host:     server.URL,
		Username: "admin",
		Password: "password",
	})

	info, err := client.TestConnection(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "v4.5.2", info.AppVersion)
	assert.Equal(t, "2.9.3", info.APIVersion)
}

func TestClient_TestConnection_AuthFailure(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v2/auth/login", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "Fails.")
	})

	server := newTestServer(t, mux)
	defer server.Close()

	client := NewClient(&Config{
		Host:     server.URL,
		Username: "admin",
		Password: "wrong",
	})

	info, err := client.TestConnection(context.Background())
	assert.Nil(t, info)
	assert.Error(t, err)

	var connErr *ConnectionError
	assert.ErrorAs(t, err, &connErr)
	assert.Equal(t, ErrCodeAuthFailed, connErr.Code)
}

func TestClient_TestConnection_WithBasePath(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/qbt/api/v2/auth/login", func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{Name: "SID", Value: "session"})
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "Ok.")
	})
	mux.HandleFunc("/qbt/api/v2/app/version", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "v4.6.0")
	})
	mux.HandleFunc("/qbt/api/v2/app/webapiVersion", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "2.10.0")
	})

	server := newTestServer(t, mux)
	defer server.Close()

	client := NewClient(&Config{
		Host:     server.URL,
		Username: "admin",
		Password: "password",
		BasePath: "/qbt",
	})

	info, err := client.TestConnection(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "v4.6.0", info.AppVersion)
	assert.Equal(t, "2.10.0", info.APIVersion)
}

func TestClient_TestConnection_ContextCancelled(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v2/auth/login", func(w http.ResponseWriter, r *http.Request) {
		// Simulate slow response
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "Ok.")
	})

	server := newTestServer(t, mux)
	defer server.Close()

	client := NewClient(&Config{
		Host:     server.URL,
		Username: "admin",
		Password: "password",
		Timeout:  100 * time.Millisecond,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	info, err := client.TestConnection(ctx)
	assert.Nil(t, info)
	assert.Error(t, err)
}

func TestClient_Ping_Success(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v2/app/version", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "v4.5.2")
	})

	server := newTestServer(t, mux)
	defer server.Close()

	client := NewClient(&Config{Host: server.URL})

	err := client.Ping(context.Background())
	assert.NoError(t, err)
}

func TestClient_Ping_FailsWithReAuth(t *testing.T) {
	callCount := 0
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v2/app/version", func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount == 1 {
			// First call fails (session expired)
			w.WriteHeader(http.StatusForbidden)
			return
		}
		// Second call after re-auth succeeds
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "v4.5.2")
	})
	mux.HandleFunc("/api/v2/auth/login", func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{Name: "SID", Value: "new-session"})
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "Ok.")
	})

	server := newTestServer(t, mux)
	defer server.Close()

	client := NewClient(&Config{
		Host:     server.URL,
		Username: "admin",
		Password: "password",
	})

	err := client.Ping(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, 2, callCount)
}

func TestClient_Ping_ConnectionRefused(t *testing.T) {
	client := NewClient(&Config{
		Host:    "http://127.0.0.1:1",
		Timeout: 1 * time.Second,
	})

	err := client.Ping(context.Background())
	assert.Error(t, err)
}

func TestClient_Ping_ReAuthFails(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v2/app/version", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	})
	mux.HandleFunc("/api/v2/auth/login", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "Fails.")
	})

	server := newTestServer(t, mux)
	defer server.Close()

	client := NewClient(&Config{
		Host:     server.URL,
		Username: "admin",
		Password: "wrong",
	})

	err := client.Ping(context.Background())
	assert.Error(t, err)
	var connErr *ConnectionError
	assert.ErrorAs(t, err, &connErr)
	assert.Equal(t, ErrCodeConnectionFailed, connErr.Code)
}

func TestClient_TestConnection_VersionEndpointFailure(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v2/auth/login", func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{Name: "SID", Value: "session"})
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "Ok.")
	})
	mux.HandleFunc("/api/v2/app/version", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	server := newTestServer(t, mux)
	defer server.Close()

	client := NewClient(&Config{
		Host:     server.URL,
		Username: "admin",
		Password: "password",
	})

	info, err := client.TestConnection(context.Background())
	assert.Nil(t, info)
	assert.Error(t, err)
}

func TestClient_EnsureAuth_SkipsLoginWhenSessionFresh(t *testing.T) {
	loginCallCount := 0
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v2/auth/login", func(w http.ResponseWriter, r *http.Request) {
		loginCallCount++
		http.SetCookie(w, &http.Cookie{Name: "SID", Value: "session"})
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "Ok.")
	})
	mux.HandleFunc("/api/v2/torrents/info", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "[]")
	})

	server := newTestServer(t, mux)
	defer server.Close()

	client := NewClient(&Config{
		Host:     server.URL,
		Username: "admin",
		Password: "password",
	})

	ctx := context.Background()

	// First call — should login (no session yet)
	_, err := client.GetTorrents(ctx, nil)
	require.NoError(t, err)
	assert.Equal(t, 1, loginCallCount, "first call should trigger login")

	// Second call — should skip login (session still fresh)
	_, err = client.GetTorrents(ctx, nil)
	require.NoError(t, err)
	assert.Equal(t, 1, loginCallCount, "second call should NOT trigger login — session is fresh")

	// Third call — still fresh
	_, err = client.GetTorrents(ctx, nil)
	require.NoError(t, err)
	assert.Equal(t, 1, loginCallCount, "third call should NOT trigger login")
}

func TestClient_DoWithAuth_RetriesOn401(t *testing.T) {
	apiCallCount := 0
	loginCallCount := 0

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v2/auth/login", func(w http.ResponseWriter, r *http.Request) {
		loginCallCount++
		http.SetCookie(w, &http.Cookie{Name: "SID", Value: fmt.Sprintf("session-%d", loginCallCount)})
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "Ok.")
	})
	mux.HandleFunc("/api/v2/torrents/info", func(w http.ResponseWriter, r *http.Request) {
		apiCallCount++
		if apiCallCount == 1 {
			// First API call returns 401 (session expired)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		// Subsequent calls succeed
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "[]")
	})

	server := newTestServer(t, mux)
	defer server.Close()

	client := NewClient(&Config{
		Host:     server.URL,
		Username: "admin",
		Password: "password",
	})

	// Pre-set lastLoginAt so ensureAuth doesn't trigger initial login
	client.lastLoginAt = time.Now()

	ctx := context.Background()
	torrents, err := client.GetTorrents(ctx, nil)
	require.NoError(t, err)
	assert.Empty(t, torrents)

	// Should have: 1 initial login (from doWithAuth retry), 2 API calls (first 401, second ok)
	assert.Equal(t, 1, loginCallCount, "should re-login once after 401")
	assert.Equal(t, 2, apiCallCount, "should retry API call after re-auth")
}

func TestParseQBMajorVersion(t *testing.T) {
	cases := []struct {
		in   string
		want int
	}{
		{"v4.6.7", 4},
		{"4.5.2", 4},
		{"v5.0.3", 5},
		{"5.1.0", 5},
		{"v10.2.0", 10},
		{"", 4},        // fallback
		{"garbage", 4}, // fallback
		{"v0.0.0", 4},  // non-positive → fallback
	}
	for _, c := range cases {
		assert.Equal(t, c.want, parseQBMajorVersion(c.in), "version %q", c.in)
	}
}

// AC4/AC5: pause on qBT 4.x hits POST /torrents/pause with pipe-joined hashes.
func TestClient_PauseTorrents_4xEndpoint(t *testing.T) {
	var gotHashes string
	pausePath := ""
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v2/auth/login", qbtLoginOK)
	mux.HandleFunc("/api/v2/app/version", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "v4.6.0")
	})
	mux.HandleFunc("/api/v2/torrents/pause", func(w http.ResponseWriter, r *http.Request) {
		pausePath = r.URL.Path
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "application/x-www-form-urlencoded", r.Header.Get("Content-Type"))
		require.NoError(t, r.ParseForm())
		gotHashes = r.FormValue("hashes")
		w.WriteHeader(http.StatusOK)
	})
	server := newTestServer(t, mux)
	defer server.Close()

	client := NewClient(&Config{Host: server.URL, Username: "admin", Password: "password"})
	err := client.PauseTorrents(context.Background(), []string{"h1", "h2"})
	require.NoError(t, err)
	assert.Equal(t, "/api/v2/torrents/pause", pausePath)
	assert.Equal(t, "h1|h2", gotHashes, "hashes must be pipe-joined for batch ops")
}

// AC4: pause on qBT 5.0+ hits POST /torrents/stop (renamed endpoint), never /pause.
func TestClient_PauseTorrents_5xUsesStop(t *testing.T) {
	stopCalled, pauseCalled := false, false
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v2/auth/login", qbtLoginOK)
	mux.HandleFunc("/api/v2/app/version", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "v5.0.1")
	})
	mux.HandleFunc("/api/v2/torrents/stop", func(w http.ResponseWriter, r *http.Request) {
		stopCalled = true
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/api/v2/torrents/pause", func(w http.ResponseWriter, r *http.Request) {
		pauseCalled = true
		w.WriteHeader(http.StatusOK)
	})
	server := newTestServer(t, mux)
	defer server.Close()

	client := NewClient(&Config{Host: server.URL, Username: "admin", Password: "password"})
	err := client.PauseTorrents(context.Background(), []string{"h1"})
	require.NoError(t, err)
	assert.True(t, stopCalled, "qBT 5.x must use /torrents/stop")
	assert.False(t, pauseCalled, "qBT 5.x must NOT use the 4.x /torrents/pause name")
}

// AC4: resume routes to /torrents/resume (4.x) and /torrents/start (5.0+).
func TestClient_ResumeTorrents_VersionRouting(t *testing.T) {
	cases := []struct {
		version   string
		wantPath  string
		otherPath string
	}{
		{"v4.6.0", "/api/v2/torrents/resume", "/api/v2/torrents/start"},
		{"v5.0.1", "/api/v2/torrents/start", "/api/v2/torrents/resume"},
	}
	for _, c := range cases {
		t.Run(c.version, func(t *testing.T) {
			wantCalled, otherCalled := false, false
			mux := http.NewServeMux()
			mux.HandleFunc("/api/v2/auth/login", qbtLoginOK)
			mux.HandleFunc("/api/v2/app/version", func(w http.ResponseWriter, r *http.Request) {
				fmt.Fprint(w, c.version)
			})
			mux.HandleFunc(c.wantPath, func(w http.ResponseWriter, r *http.Request) {
				wantCalled = true
				w.WriteHeader(http.StatusOK)
			})
			mux.HandleFunc(c.otherPath, func(w http.ResponseWriter, r *http.Request) {
				otherCalled = true
				w.WriteHeader(http.StatusOK)
			})
			server := newTestServer(t, mux)
			defer server.Close()

			client := NewClient(&Config{Host: server.URL, Username: "admin", Password: "password"})
			err := client.ResumeTorrents(context.Background(), []string{"h1"})
			require.NoError(t, err)
			assert.True(t, wantCalled, "expected %s to be hit", c.wantPath)
			assert.False(t, otherCalled)
		})
	}
}

// AC5: delete posts hashes + deleteFiles form body to /torrents/delete (version-agnostic).
func TestClient_DeleteTorrents_FormBody(t *testing.T) {
	for _, deleteFiles := range []bool{true, false} {
		t.Run(fmt.Sprintf("deleteFiles=%v", deleteFiles), func(t *testing.T) {
			var gotHashes, gotDeleteFiles string
			mux := http.NewServeMux()
			mux.HandleFunc("/api/v2/auth/login", qbtLoginOK)
			mux.HandleFunc("/api/v2/torrents/delete", func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodPost, r.Method)
				require.NoError(t, r.ParseForm())
				gotHashes = r.FormValue("hashes")
				gotDeleteFiles = r.FormValue("deleteFiles")
				w.WriteHeader(http.StatusOK)
			})
			server := newTestServer(t, mux)
			defer server.Close()

			client := NewClient(&Config{Host: server.URL, Username: "admin", Password: "password"})
			err := client.DeleteTorrents(context.Background(), []string{"a", "b"}, deleteFiles)
			require.NoError(t, err)
			assert.Equal(t, "a|b", gotHashes)
			assert.Equal(t, fmt.Sprintf("%v", deleteFiles), gotDeleteFiles)
		})
	}
}

// AC6 (correctness): a 401 mid-flight forces re-auth; the retried POST MUST still
// carry the form body — otherwise the pause/delete silently no-ops while 200-ing.
func TestClient_PauseTorrents_BodySurvivesReauth(t *testing.T) {
	var bodies, contentTypes []string
	pauseCalls := 0
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v2/auth/login", qbtLoginOK)
	mux.HandleFunc("/api/v2/app/version", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "v4.6.0")
	})
	mux.HandleFunc("/api/v2/torrents/pause", func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		bodies = append(bodies, string(b))
		contentTypes = append(contentTypes, r.Header.Get("Content-Type"))
		pauseCalls++
		if pauseCalls == 1 {
			w.WriteHeader(http.StatusForbidden) // force a re-auth retry
			return
		}
		w.WriteHeader(http.StatusOK)
	})
	server := newTestServer(t, mux)
	defer server.Close()

	client := NewClient(&Config{Host: server.URL, Username: "admin", Password: "password"})
	client.lastLoginAt = time.Now() // skip the initial login so the 403 drives re-auth

	err := client.PauseTorrents(context.Background(), []string{"abc123"})
	require.NoError(t, err)
	require.Len(t, bodies, 2, "pause should be attempted twice (403 then retry)")
	assert.Equal(t, "hashes=abc123", bodies[0])
	assert.Equal(t, "hashes=abc123", bodies[1], "AC6: retried POST must still carry the form body")
	// L2: the retried request must ALSO carry the form Content-Type (header clone),
	// else real qBittorrent would fail to parse the re-sent body.
	assert.Equal(t, "application/x-www-form-urlencoded", contentTypes[1],
		"AC6: retried POST must preserve the form Content-Type header")
}

// L3: a failed version lookup surfaces as a *ConnectionError and never reaches
// the action endpoint (pause is intentionally unregistered here).
func TestClient_PauseTorrents_VersionFetchFails(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v2/auth/login", qbtLoginOK)
	mux.HandleFunc("/api/v2/app/version", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	server := newTestServer(t, mux)
	defer server.Close()

	client := NewClient(&Config{Host: server.URL, Username: "admin", Password: "password"})
	err := client.PauseTorrents(context.Background(), []string{"h1"})
	require.Error(t, err)

	var connErr *ConnectionError
	assert.ErrorAs(t, err, &connErr)
	assert.Equal(t, ErrCodeConnectionFailed, connErr.Code)
}

// M1: the shared client's first-time version resolution must be race-free under
// concurrent actions. lastLoginAt is preset so ensureAuth never re-Logins —
// isolating this test to the qbtMajorVer path (the field this story added).
// Run under `-race`: fails without verMu, passes with it.
func TestClient_MajorVersion_ConcurrentNoRace(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v2/auth/login", qbtLoginOK)
	mux.HandleFunc("/api/v2/app/version", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "v4.6.0")
	})
	mux.HandleFunc("/api/v2/torrents/pause", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	server := newTestServer(t, mux)
	defer server.Close()

	client := NewClient(&Config{Host: server.URL, Username: "admin", Password: "password"})
	client.lastLoginAt = time.Now() // keep the session fresh → no Login → no lastLoginAt write

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			assert.NoError(t, client.PauseTorrents(context.Background(), []string{"h1"}))
		}()
	}
	wg.Wait()
	assert.Equal(t, 4, client.qbtMajorVer)
}

// AC5: a non-2xx action response surfaces as a *ConnectionError.
func TestClient_DeleteTorrents_NonSuccessStatus(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v2/auth/login", qbtLoginOK)
	mux.HandleFunc("/api/v2/torrents/delete", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	server := newTestServer(t, mux)
	defer server.Close()

	client := NewClient(&Config{Host: server.URL, Username: "admin", Password: "password"})
	err := client.DeleteTorrents(context.Background(), []string{"a"}, false)
	require.Error(t, err)

	var connErr *ConnectionError
	assert.ErrorAs(t, err, &connErr)
	assert.Equal(t, ErrCodeConnectionFailed, connErr.Code)
}
