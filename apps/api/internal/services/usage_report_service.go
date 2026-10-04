package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/vido/api/internal/repository"
)

// infra-optin-usage-report-a2 — the opt-in anonymous weekly usage report
// (PRD amendment prd/prd-telemetry-amendment.md, P1-040).
//
// Off by default. When the user turns it on, once per 7 days Vido sends one
// event to the maintainer's receiver carrying ONLY: a random install id, the
// Vido version, and how many subtitles Vido produced on its own in the last 7
// days (total and by source). Library content never leaves the NAS (NFR-S7).

// Settings keys. No migration: settings is a key-value table.
const (
	usageReportKeyEnabled       = "usage_report.enabled"
	usageReportKeyInstallID     = "usage_report.install_id"
	usageReportKeyLastAttemptAt = "usage_report.last_attempt_at"
	usageReportKeyLastSentAt    = "usage_report.last_sent_at"
	usageReportKeyLastPayload   = "usage_report.last_payload"
)

// usageReportInterval is the cadence (P1-040-5). It also limits attempts: a
// failed send waits a full interval too, so a dead receiver never sees a
// retry storm (P1-040-6) — the schedule IS the rate limiter (Rule 27 ①).
const usageReportInterval = 7 * 24 * time.Hour

// Fixed protocol constants of the receiver's event envelope (Umami's
// POST /api/send). None of them is user data.
const (
	usageReportHostname  = "vido"
	usageReportURL       = "/usage-report"
	usageReportEventName = "weekly_usage"
	// usageReportIP is sent as the event's ip so the receiver skips its
	// country lookup — the country is not on the PRD's allow-list.
	usageReportIP = "127.0.0.1"
)

// AutoProducedCounter is the one ledger read the report needs (a1).
type AutoProducedCounter interface {
	AutoProducedBetween(ctx context.Context, from, to time.Time) (repository.AutoProducedCounts, error)
}

// ReportSender delivers a finished body to the receiver (usagereport.Sender).
type ReportSender interface {
	Send(ctx context.Context, body []byte) error
}

// UsageReportConfig is what the build supplies: the version this binary
// reports and the receiver's website id. An empty WebsiteID (or a nil sender)
// means no receiver is configured — the feature is unavailable.
type UsageReportConfig struct {
	Version   string
	WebsiteID string
}

// UsageReportStatus is what the settings page shows (P1-040-1, P1-040-3).
type UsageReportStatus struct {
	Available bool
	Enabled   bool
	// LastSentAt / LastPayload are the last SUCCESSFUL send; nil = never sent.
	LastSentAt  *time.Time
	LastPayload *string
}

// UsageReportService owns the report's settings, body and weekly attempt.
type UsageReportService struct {
	settings repository.SettingsRepositoryInterface
	counter  AutoProducedCounter
	sender   ReportSender
	cfg      UsageReportConfig
	logger   *slog.Logger
	now      func() time.Time
}

// NewUsageReportService builds the service. sender may be nil when no
// receiver is configured.
func NewUsageReportService(settings repository.SettingsRepositoryInterface, counter AutoProducedCounter, sender ReportSender, cfg UsageReportConfig, logger *slog.Logger) *UsageReportService {
	if logger == nil {
		logger = slog.Default()
	}
	return &UsageReportService{
		settings: settings,
		counter:  counter,
		sender:   sender,
		cfg:      cfg,
		logger:   logger,
		now:      time.Now,
	}
}

// Available reports whether this build has a receiver to send to.
func (s *UsageReportService) Available() bool {
	return s.sender != nil && s.cfg.WebsiteID != ""
}

// Status reads the current state. Absent keys are the normal fresh-install
// answer: off, never sent.
func (s *UsageReportService) Status(ctx context.Context) (UsageReportStatus, error) {
	enabled, err := s.enabled(ctx)
	if err != nil {
		return UsageReportStatus{}, err
	}
	st := UsageReportStatus{Available: s.Available(), Enabled: enabled}

	sentAt, err := s.readTime(ctx, usageReportKeyLastSentAt)
	if err != nil {
		return UsageReportStatus{}, err
	}
	st.LastSentAt = sentAt
	payload, err := s.settings.GetString(ctx, usageReportKeyLastPayload)
	switch {
	case err == nil:
		st.LastPayload = &payload
	case !isSettingNotFound(err):
		return UsageReportStatus{}, fmt.Errorf("read last usage report: %w", err)
	}
	return st, nil
}

// SetEnabled turns the report on or off and returns the new state. The first
// enable creates the install id; turning off keeps it, so "the same machine"
// stays the same machine (AC #2).
func (s *UsageReportService) SetEnabled(ctx context.Context, enabled bool) (UsageReportStatus, error) {
	if enabled {
		if err := s.ensureInstallID(ctx); err != nil {
			return UsageReportStatus{}, err
		}
	}
	if err := s.settings.SetBool(ctx, usageReportKeyEnabled, enabled); err != nil {
		return UsageReportStatus{}, fmt.Errorf("save usage report setting: %w", err)
	}
	s.logger.Info("usage report setting saved", "enabled", enabled)
	return s.Status(ctx)
}

// Tick is the scheduler's check. It sends at most once per interval and never
// returns an error: every failure is a Warn and nothing else (P1-040-6).
func (s *UsageReportService) Tick(ctx context.Context) {
	if !s.Available() {
		return
	}
	enabled, err := s.enabled(ctx)
	if err != nil {
		s.logger.Warn("usage report: read setting failed", "error", err)
		return
	}
	if !enabled {
		return
	}
	now := s.now().UTC()
	last, err := s.readTime(ctx, usageReportKeyLastAttemptAt)
	if err != nil {
		s.logger.Warn("usage report: read last attempt failed", "error", err)
		return
	}
	if last != nil && now.Sub(*last) < usageReportInterval {
		return
	}
	// Recorded BEFORE sending: success or failure, this is the one attempt.
	if err := s.settings.SetString(ctx, usageReportKeyLastAttemptAt, now.Format(time.RFC3339Nano)); err != nil {
		s.logger.Warn("usage report: record attempt failed — not sending", "error", err)
		return
	}

	body, err := s.buildBody(ctx, now)
	if err != nil {
		s.logger.Warn("usage report: not sent", "error", err)
		return
	}
	if err := s.sender.Send(ctx, body); err != nil {
		s.logger.Warn("usage report: send failed — next attempt in 7 days", "error", err)
		return
	}
	if err := s.settings.SetString(ctx, usageReportKeyLastPayload, string(body)); err != nil {
		s.logger.Warn("usage report: sent, but saving the shown copy failed", "error", err)
		return
	}
	if err := s.settings.SetString(ctx, usageReportKeyLastSentAt, now.Format(time.RFC3339Nano)); err != nil {
		s.logger.Warn("usage report: sent, but saving the send time failed", "error", err)
		return
	}
	s.logger.Info("usage report sent")
}

// usageReportBody is the exact wire body. Struct fields (not a map) keep the
// key order stable, and the struct IS the allow-list: adding a field here is
// the only way to send more, and the tests pin the field set (NFR-T1).
type usageReportBody struct {
	Type    string             `json:"type"`
	Payload usageReportPayload `json:"payload"`
}

type usageReportPayload struct {
	Website  string          `json:"website"`
	Hostname string          `json:"hostname"`
	URL      string          `json:"url"`
	Name     string          `json:"name"`
	ID       string          `json:"id"`
	IP       string          `json:"ip"`
	Data     usageReportData `json:"data"`
}

type usageReportData struct {
	Version            string `json:"version"`
	SubtitlesAuto7d    int    `json:"subtitles_auto_7d"`
	SubtitlesEmbedded7 int    `json:"subtitles_embedded_7d"`
	SubtitlesOnline7d  int    `json:"subtitles_online_7d"`
	SubtitlesASR7d     int    `json:"subtitles_asr_7d"`
}

func (s *UsageReportService) buildBody(ctx context.Context, now time.Time) ([]byte, error) {
	installID, err := s.settings.GetString(ctx, usageReportKeyInstallID)
	if err != nil {
		return nil, fmt.Errorf("read install id: %w", err)
	}
	counts, err := s.counter.AutoProducedBetween(ctx, now.Add(-usageReportInterval), now)
	if err != nil {
		// An unknown count is not reported as 0.
		return nil, fmt.Errorf("count auto-produced subtitles: %w", err)
	}
	return json.Marshal(usageReportBody{
		Type: "event",
		Payload: usageReportPayload{
			Website:  s.cfg.WebsiteID,
			Hostname: usageReportHostname,
			URL:      usageReportURL,
			Name:     usageReportEventName,
			ID:       installID,
			IP:       usageReportIP,
			Data: usageReportData{
				Version:            s.cfg.Version,
				SubtitlesAuto7d:    counts.Total(),
				SubtitlesEmbedded7: counts.Embedded,
				SubtitlesOnline7d:  counts.Online,
				SubtitlesASR7d:     counts.ASR,
			},
		},
	})
}

func (s *UsageReportService) enabled(ctx context.Context) (bool, error) {
	on, err := s.settings.GetBool(ctx, usageReportKeyEnabled)
	if err != nil {
		if isSettingNotFound(err) {
			return false, nil
		}
		return false, fmt.Errorf("read usage report setting: %w", err)
	}
	return on, nil
}

func (s *UsageReportService) ensureInstallID(ctx context.Context) error {
	_, err := s.settings.GetString(ctx, usageReportKeyInstallID)
	if err == nil {
		return nil
	}
	if !isSettingNotFound(err) {
		return fmt.Errorf("read install id: %w", err)
	}
	// A dashed UUID v4: random, derived from nothing on the machine, and not a
	// 32-hex run the DB log handler would mask.
	if err := s.settings.SetString(ctx, usageReportKeyInstallID, uuid.New().String()); err != nil {
		return fmt.Errorf("save install id: %w", err)
	}
	return nil
}

func (s *UsageReportService) readTime(ctx context.Context, key string) (*time.Time, error) {
	raw, err := s.settings.GetString(ctx, key)
	if err != nil {
		if isSettingNotFound(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read %s: %w", key, err)
	}
	t, err := time.Parse(time.RFC3339Nano, raw)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", key, err)
	}
	t = t.UTC()
	return &t, nil
}

// UsageReportServiceInterface is what the settings handler and the setup
// wizard use (Rule 11: interfaces live in services).
type UsageReportServiceInterface interface {
	Status(ctx context.Context) (UsageReportStatus, error)
	SetEnabled(ctx context.Context, enabled bool) (UsageReportStatus, error)
}

var _ UsageReportServiceInterface = (*UsageReportService)(nil)
