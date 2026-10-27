package sse

import (
	"bufio"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// disc-2026-10-reparse-dialog-stuck-over-tunnel: through the Cloudflare tunnel
// the stream never opened. The stream must tell proxies not to transform it,
// and announce itself at once.
func TestHandler_StreamIsNotTransformedAndConnectsAtOnce(t *testing.T) {
	gin.SetMode(gin.TestMode)
	hub := NewHub()
	go hub.Run()
	defer hub.Close()

	router := gin.New()
	router.GET("/events", Handler(hub))
	srv := httptest.NewServer(router)
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL+"/events", nil)
	require.NoError(t, err)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, "text/event-stream", resp.Header.Get("Content-Type"))
	assert.Equal(t, "no-cache, no-transform", resp.Header.Get("Cache-Control"))
	assert.Equal(t, "no", resp.Header.Get("X-Accel-Buffering"))
	// The first event arrives before any other event or the 30s ping.
	line, err := bufio.NewReader(resp.Body).ReadString('\n')
	require.NoError(t, err)
	assert.Equal(t, "event: connected\n", line)
}
