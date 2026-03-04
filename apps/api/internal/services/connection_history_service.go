package services

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/repository"
)

// ConnectionHistoryServiceInterface defines the contract for connection history operations.
type ConnectionHistoryServiceInterface interface {
	// RecordEvent persists a connection status change event.
	RecordEvent(ctx context.Context, service string, eventType models.ConnectionEventType, status string, message string) error

	// GetHistory retrieves recent connection events for a service.
	GetHistory(ctx context.Context, service string, limit int) ([]models.ConnectionEvent, error)
}

// ConnectionHistoryService implements ConnectionHistoryServiceInterface.
type ConnectionHistoryService struct {
	repo   repository.ConnectionHistoryRepositoryInterface
	logger *slog.Logger
}

// Compile-time interface verification.
var _ ConnectionHistoryServiceInterface = (*ConnectionHistoryService)(nil)

// NewConnectionHistoryService creates a new ConnectionHistoryService.
func NewConnectionHistoryService(repo repository.ConnectionHistoryRepositoryInterface) *ConnectionHistoryService {
	return &ConnectionHistoryService{
		repo:   repo,
		logger: slog.Default(),
	}
}

// ValidServiceNames returns the set of known service names for validation.
func ValidServiceNames() map[string]bool {
	return map[string]bool{
		string(models.ServiceNameTMDb):        true,
		string(models.ServiceNameDouban):      true,
		string(models.ServiceNameWikipedia):   true,
		string(models.ServiceNameAI):          true,
		string(models.ServiceNameQBittorrent): true,
	}
}

// IsValidServiceName checks if a service name is known.
func IsValidServiceName(name string) bool {
	return ValidServiceNames()[name]
}

// RecordEvent persists a connection status change event.
func (s *ConnectionHistoryService) RecordEvent(ctx context.Context, service string, eventType models.ConnectionEventType, status string, message string) error {
	event := &models.ConnectionEvent{
		ID:        uuid.New().String(),
		Service:   service,
		EventType: eventType,
		Status:    status,
		Message:   message,
	}

	if err := s.repo.Create(ctx, event); err != nil {
		s.logger.Error("Failed to record connection event",
			"service", service,
			"event_type", eventType,
			"error", err,
		)
		return err
	}

	return nil
}

// GetHistory retrieves recent connection events for a service.
func (s *ConnectionHistoryService) GetHistory(ctx context.Context, service string, limit int) ([]models.ConnectionEvent, error) {
	return s.repo.GetHistory(ctx, service, limit)
}
