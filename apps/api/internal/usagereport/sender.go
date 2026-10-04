// Package usagereport sends the opt-in anonymous usage report to its receiver
// (infra-optin-usage-report-a2). It knows HTTP and the receiver's protocol
// (Umami's POST /api/send) and nothing about Vido's data — the caller hands it
// the finished body, so what is sent and what the settings page shows are the
// same bytes.
package usagereport

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

// ErrNotRecorded is a 2xx reply that does not carry a session id: Umami
// silently drops what its bot filter flags and still answers 200 {"beep":"boop"}.
var ErrNotRecorded = errors.New("usage report: receiver answered but did not record the event")

// sendTimeout bounds one weekly send. Nothing waits on it; it only keeps a
// dead receiver from holding a goroutine.
const sendTimeout = 15 * time.Second

// Sender posts a report body to the receiver. Build it once and reuse it
// (Rule 14): it owns one http.Client.
type Sender struct {
	endpoint  string
	userAgent string
	client    *http.Client
}

// NewSender builds a Sender for the receiver at endpoint (base URL, no
// trailing slash) reporting as the given Vido version.
func NewSender(endpoint, version string) *Sender {
	return &Sender{
		endpoint:  endpoint,
		userAgent: userAgent(version),
		client:    &http.Client{Timeout: sendTimeout},
	}
}

// userAgent is a browser-shaped UA. Umami runs every request through isbot
// and drops the matches; Go's default "Go-http-client/1.1", a bare
// "Vido/<v>" and any "(compatible; …)" form are all flagged, this shape is not.
func userAgent(version string) string {
	return "Mozilla/5.0 (X11; Linux x86_64) Vido/" + version
}

// Send posts body to {endpoint}/api/send. Success means a 2xx reply that
// carries a sessionId — the only proof the event was recorded.
func (s *Sender) Send(ctx context.Context, body []byte) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.endpoint+"/api/send", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("usage report: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", s.userAgent)

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("usage report: send: %w", err)
	}
	defer resp.Body.Close()

	reply, err := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
	if err != nil {
		return fmt.Errorf("usage report: read reply: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("usage report: receiver answered %d: %s", resp.StatusCode, bytes.TrimSpace(reply))
	}
	var accepted struct {
		SessionID string `json:"sessionId"`
	}
	if err := json.Unmarshal(reply, &accepted); err != nil || accepted.SessionID == "" {
		return ErrNotRecorded
	}
	return nil
}
