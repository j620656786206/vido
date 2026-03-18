package repository

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"

	"github.com/vido/api/internal/models"
)

// LogRepository provides data access operations for system logs.
type LogRepository struct {
	db *sql.DB
}

// NewLogRepository creates a new instance of LogRepository.
func NewLogRepository(db *sql.DB) *LogRepository {
	return &LogRepository{db: db}
}

// GetLogs retrieves paginated system logs with optional filters.
func (r *LogRepository) GetLogs(ctx context.Context, filter models.LogFilter) ([]models.SystemLog, int, error) {
	// Validate and default pagination
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PerPage < 1 {
		filter.PerPage = 50
	}
	if filter.PerPage > MaxPageSize {
		filter.PerPage = MaxPageSize
	}

	var conditions []string
	var args []interface{}

	if filter.Level != "" {
		conditions = append(conditions, "level = ?")
		args = append(args, string(filter.Level))
	}

	if filter.Keyword != "" {
		conditions = append(conditions, "(message LIKE ? OR source LIKE ? OR context_json LIKE ?)")
		kw := "%" + filter.Keyword + "%"
		args = append(args, kw, kw, kw)
	}

	where := ""
	if len(conditions) > 0 {
		where = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Count total matching logs
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM system_logs %s", where)
	var total int
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count system logs: %w", err)
	}

	// Query with pagination, newest first
	offset := (filter.Page - 1) * filter.PerPage
	dataQuery := fmt.Sprintf(
		"SELECT id, level, message, COALESCE(source, ''), COALESCE(context_json, ''), created_at FROM system_logs %s ORDER BY created_at DESC LIMIT ? OFFSET ?",
		where,
	)
	dataArgs := append(args, filter.PerPage, offset)

	rows, err := r.db.QueryContext(ctx, dataQuery, dataArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("query system logs: %w", err)
	}
	defer rows.Close()

	var logs []models.SystemLog
	for rows.Next() {
		var log models.SystemLog
		if err := rows.Scan(&log.ID, &log.Level, &log.Message, &log.Source, &log.ContextJSON, &log.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan system log: %w", err)
		}
		logs = append(logs, log)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate system logs: %w", err)
	}

	return logs, total, nil
}

// CreateLog inserts a new log entry into the database.
func (r *LogRepository) CreateLog(ctx context.Context, log *models.SystemLog) error {
	if log == nil {
		return fmt.Errorf("log entry cannot be nil")
	}

	query := `INSERT INTO system_logs (level, message, source, context_json, created_at) VALUES (?, ?, ?, ?, ?)`

	_, err := r.db.ExecContext(ctx, query, string(log.Level), log.Message, log.Source, log.ContextJSON, log.CreatedAt)
	if err != nil {
		return fmt.Errorf("insert system log: %w", err)
	}

	return nil
}

// CreateLogBatch inserts multiple log entries in a single transaction.
func (r *LogRepository) CreateLogBatch(ctx context.Context, logs []models.SystemLog) error {
	if len(logs) == 0 {
		return nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `INSERT INTO system_logs (level, message, source, context_json, created_at) VALUES (?, ?, ?, ?, ?)`)
	if err != nil {
		return fmt.Errorf("prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, log := range logs {
		if _, err := stmt.ExecContext(ctx, string(log.Level), log.Message, log.Source, log.ContextJSON, log.CreatedAt); err != nil {
			return fmt.Errorf("insert log batch entry: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit log batch: %w", err)
	}

	slog.Debug("Log batch inserted", "count", len(logs))
	return nil
}

// DeleteOlderThan removes logs older than the specified number of days.
func (r *LogRepository) DeleteOlderThan(ctx context.Context, days int) (int64, error) {
	if days <= 0 {
		return 0, fmt.Errorf("days must be positive")
	}

	query := `DELETE FROM system_logs WHERE created_at < datetime('now', ?)`
	modifier := fmt.Sprintf("-%d days", days)

	result, err := r.db.ExecContext(ctx, query, modifier)
	if err != nil {
		return 0, fmt.Errorf("delete old system logs: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("get rows affected: %w", err)
	}

	if rowsAffected > 0 {
		slog.Info("Old system logs deleted", "days", days, "deleted", rowsAffected)
	}

	return rowsAffected, nil
}

// Compile-time interface verification
var _ LogRepositoryInterface = (*LogRepository)(nil)
