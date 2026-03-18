package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/repository"
)

// LogServiceInterface defines the contract for system log operations.
type LogServiceInterface interface {
	// GetLogs retrieves paginated system logs with optional filters
	GetLogs(ctx context.Context, filter models.LogFilter) (*LogsResponse, error)

	// ClearLogs removes logs older than the specified number of days
	ClearLogs(ctx context.Context, days int) (*LogClearResult, error)
}

// LogsResponse represents the paginated logs response
type LogsResponse struct {
	Logs    []models.SystemLog `json:"logs"`
	Total   int                `json:"total"`
	Page    int                `json:"page"`
	PerPage int                `json:"perPage"`
}

// LogClearResult represents the result of clearing logs
type LogClearResult struct {
	EntriesRemoved int64 `json:"entriesRemoved"`
	Days           int   `json:"days"`
}

// troubleshootingHints maps error codes to actionable hints in Traditional Chinese
var troubleshootingHints = map[string]string{
	"TMDB_TIMEOUT":     "檢查網路連線，或稍後重試。TMDb API 可能暫時不可用。",
	"AI_QUOTA_EXCEEDED": "AI API 配額已用完。請檢查帳戶或等待配額重置。",
	"DB_QUERY_FAILED":  "資料庫查詢失敗。請檢查磁碟空間是否充足。",
	"QBT_CONNECTION":   "無法連線到 qBittorrent。請確認服務是否正在運行。",
	"TMDB_NOT_FOUND":   "在 TMDb 找不到此影片。請確認標題或 ID 是否正確。",
	"TMDB_RATE_LIMIT":  "TMDb API 請求已達上限。請稍後再試。",
	"AI_TIMEOUT":       "AI 服務回應超時。請稍後再試，或切換至其他 AI 提供者。",
	"AUTH_TOKEN_EXPIRED": "認證已過期。請重新登入。",
}

// LogService provides business logic for system log operations.
type LogService struct {
	repo repository.LogRepositoryInterface
}

// NewLogService creates a new LogService.
func NewLogService(repo repository.LogRepositoryInterface) *LogService {
	return &LogService{repo: repo}
}

// GetLogs retrieves paginated logs, enriching ERROR entries with troubleshooting hints.
func (s *LogService) GetLogs(ctx context.Context, filter models.LogFilter) (*LogsResponse, error) {
	// Validate level filter if provided
	if filter.Level != "" && !models.IsValidLogLevel(filter.Level) {
		return nil, fmt.Errorf("invalid log level: %s", filter.Level)
	}

	// Defaults
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PerPage < 1 {
		filter.PerPage = 50
	}

	logs, total, err := s.repo.GetLogs(ctx, filter)
	if err != nil {
		slog.Error("Failed to query system logs", "error", err)
		return nil, fmt.Errorf("query logs: %w", err)
	}

	// Enrich logs: parse context JSON and add hints for ERROR entries
	for i := range logs {
		// Parse context_json into Context field
		if logs[i].ContextJSON != "" {
			var ctxMap map[string]interface{}
			if err := json.Unmarshal([]byte(logs[i].ContextJSON), &ctxMap); err == nil {
				logs[i].Context = ctxMap
				// Add troubleshooting hint for ERROR entries
				if logs[i].Level == models.LogLevelError {
					logs[i].Hint = findHint(ctxMap, logs[i].Message)
				}
			}
		}
	}

	return &LogsResponse{
		Logs:    logs,
		Total:   total,
		Page:    filter.Page,
		PerPage: filter.PerPage,
	}, nil
}

// ClearLogs removes logs older than the specified number of days.
func (s *LogService) ClearLogs(ctx context.Context, days int) (*LogClearResult, error) {
	if days <= 0 {
		return nil, fmt.Errorf("days must be positive")
	}

	removed, err := s.repo.DeleteOlderThan(ctx, days)
	if err != nil {
		slog.Error("Failed to clear system logs", "error", err, "days", days)
		return nil, fmt.Errorf("clear logs: %w", err)
	}

	slog.Info("System logs cleared", "days", days, "removed", removed)

	return &LogClearResult{
		EntriesRemoved: removed,
		Days:           days,
	}, nil
}

// findHint searches for a matching troubleshooting hint based on context or message
func findHint(ctxMap map[string]interface{}, message string) string {
	// Check error_code in context
	if code, ok := ctxMap["error_code"]; ok {
		if codeStr, ok := code.(string); ok {
			if hint, found := troubleshootingHints[codeStr]; found {
				return hint
			}
		}
	}

	// Check code in context
	if code, ok := ctxMap["code"]; ok {
		if codeStr, ok := code.(string); ok {
			if hint, found := troubleshootingHints[codeStr]; found {
				return hint
			}
		}
	}

	// Fallback: check message for known keywords
	upper := strings.ToUpper(message)
	for code, hint := range troubleshootingHints {
		if strings.Contains(upper, strings.ReplaceAll(code, "_", " ")) ||
			strings.Contains(upper, code) {
			return hint
		}
	}

	return ""
}

// Compile-time interface verification
var _ LogServiceInterface = (*LogService)(nil)
