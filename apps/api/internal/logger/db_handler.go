package logger

import (
	"context"
	"encoding/json"
	"log/slog"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/repository"
)

const (
	// maxBufferSize is the maximum number of log entries to buffer before flushing
	maxBufferSize = 100
	// flushInterval is the time between automatic flushes
	flushInterval = 5 * time.Second
)

// sensitivePatterns matches common sensitive data patterns
var sensitivePatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)(api[_-]?key|apikey|token|secret|password|passwd|pwd)\s*[=:]\s*\S+`),
	regexp.MustCompile(`(?i)(bearer\s+)\S+`),
	regexp.MustCompile(`[a-f0-9]{32,}`), // hex strings >= 32 chars (likely API keys)
}

// sharedState holds the mutable state shared across all child handlers.
type sharedState struct {
	repo   repository.LogRepositoryInterface
	mu     sync.Mutex
	buffer []models.SystemLog
	done   chan struct{}
}

// DBHandler is a custom slog.Handler that writes log records to the database.
// It buffers log entries and batch-writes them periodically to avoid performance impact.
type DBHandler struct {
	shared *sharedState
	attrs  []slog.Attr
	groups []string
}

// NewDBHandler creates a new database log handler.
// It starts a background goroutine that flushes buffered logs periodically.
func NewDBHandler(repo repository.LogRepositoryInterface) *DBHandler {
	h := &DBHandler{
		shared: &sharedState{
			repo:   repo,
			buffer: make([]models.SystemLog, 0, maxBufferSize),
			done:   make(chan struct{}),
		},
	}

	go h.flushLoop()

	return h
}

// Enabled returns true for all levels — filtering happens at the slog level.
func (h *DBHandler) Enabled(_ context.Context, _ slog.Level) bool {
	return true
}

// Handle processes a log record: masks sensitive data, buffers it, and flushes if needed.
func (h *DBHandler) Handle(_ context.Context, r slog.Record) error {
	level := mapLevel(r.Level)

	// Collect structured attributes from handler-level attrs
	attrs := make(map[string]interface{})
	for _, a := range h.attrs {
		attrs[a.Key] = maskSensitiveValue(a.Key, a.Value.String())
	}

	// Collect record attributes
	var source string
	r.Attrs(func(a slog.Attr) bool {
		val := maskSensitiveValue(a.Key, a.Value.String())
		attrs[a.Key] = val
		if a.Key == "source" || a.Key == "package" || a.Key == "module" {
			source = a.Value.String()
		}
		return true
	})

	// Apply group prefixes
	if len(h.groups) > 0 {
		grouped := make(map[string]interface{})
		grouped[strings.Join(h.groups, ".")] = attrs
		attrs = grouped
	}

	var contextJSON string
	if len(attrs) > 0 {
		data, err := json.Marshal(attrs)
		if err == nil {
			contextJSON = string(data)
		}
	}

	message := maskSensitiveString(r.Message)

	entry := models.SystemLog{
		Level:       level,
		Message:     message,
		Source:      source,
		ContextJSON: contextJSON,
		CreatedAt:   r.Time,
	}

	s := h.shared
	s.mu.Lock()
	s.buffer = append(s.buffer, entry)
	shouldFlush := len(s.buffer) >= maxBufferSize
	s.mu.Unlock()

	if shouldFlush {
		h.flush()
	}

	return nil
}

// WithAttrs returns a new handler with the given attributes.
func (h *DBHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	newAttrs := make([]slog.Attr, len(h.attrs)+len(attrs))
	copy(newAttrs, h.attrs)
	copy(newAttrs[len(h.attrs):], attrs)

	return &DBHandler{
		shared: h.shared,
		attrs:  newAttrs,
		groups: h.groups,
	}
}

// WithGroup returns a new handler with the given group name.
func (h *DBHandler) WithGroup(name string) slog.Handler {
	newGroups := make([]string, len(h.groups)+1)
	copy(newGroups, h.groups)
	newGroups[len(h.groups)] = name

	return &DBHandler{
		shared: h.shared,
		attrs:  h.attrs,
		groups: newGroups,
	}
}

// Close flushes remaining logs and stops the background goroutine.
func (h *DBHandler) Close() {
	close(h.shared.done)
	h.flush()
}

// flushLoop periodically flushes buffered logs to the database.
func (h *DBHandler) flushLoop() {
	ticker := time.NewTicker(flushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			h.flush()
		case <-h.shared.done:
			return
		}
	}
}

// flush writes buffered log entries to the database.
func (h *DBHandler) flush() {
	s := h.shared
	s.mu.Lock()
	if len(s.buffer) == 0 {
		s.mu.Unlock()
		return
	}
	// Swap buffer for minimal lock duration
	logs := s.buffer
	s.buffer = make([]models.SystemLog, 0, maxBufferSize)
	s.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := s.repo.CreateLogBatch(ctx, logs); err != nil {
		// Cannot use slog here (infinite recursion), use stderr fallback
		// The logs are dropped; this is acceptable for a logging subsystem
		_ = err
	}
}

// mapLevel converts slog.Level to models.LogLevel
func mapLevel(l slog.Level) models.LogLevel {
	switch {
	case l >= slog.LevelError:
		return models.LogLevelError
	case l >= slog.LevelWarn:
		return models.LogLevelWarn
	case l >= slog.LevelInfo:
		return models.LogLevelInfo
	default:
		return models.LogLevelDebug
	}
}

// maskSensitiveValue masks known sensitive attribute keys
func maskSensitiveValue(key, value string) string {
	lk := strings.ToLower(key)
	if strings.Contains(lk, "key") || strings.Contains(lk, "secret") ||
		strings.Contains(lk, "password") || strings.Contains(lk, "token") ||
		strings.Contains(lk, "passwd") || strings.Contains(lk, "pwd") {
		if len(value) > 4 {
			return value[:4] + "****"
		}
		return "****"
	}
	return maskSensitiveString(value)
}

// maskSensitiveString redacts sensitive patterns in freeform text
func maskSensitiveString(s string) string {
	for _, p := range sensitivePatterns {
		s = p.ReplaceAllStringFunc(s, func(match string) string {
			if len(match) > 8 {
				return match[:8] + "****"
			}
			return "****"
		})
	}
	return s
}
