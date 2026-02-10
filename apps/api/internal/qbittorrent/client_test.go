package qbittorrent

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
