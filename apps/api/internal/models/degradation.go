package models

import (
	"strings"
	"time"
)

// DegradationLevel represents the current system degradation state
type DegradationLevel string

const (
	// DegradationNormal - All services operational
	DegradationNormal DegradationLevel = "normal"

	// DegradationPartial - Some services degraded, full functionality available with delays
	DegradationPartial DegradationLevel = "partial"

	// DegradationMinimal - Multiple services down, reduced functionality
	DegradationMinimal DegradationLevel = "minimal"

	// DegradationOffline - No external services available, cache-only mode
	DegradationOffline DegradationLevel = "offline"
)

// ServiceName represents a known external service
type ServiceName string

const (
	ServiceNameTMDb         ServiceName = "tmdb"
	ServiceNameDouban       ServiceName = "douban"
	ServiceNameWikipedia    ServiceName = "wikipedia"
	ServiceNameAI           ServiceName = "ai"
	ServiceNameQBittorrent  ServiceName = "qbittorrent"
)

// Service status constants
const (
	ServiceStatusHealthy  = "healthy"
	ServiceStatusDegraded = "degraded"
	ServiceStatusDown     = "down"
)

// ErrorThresholdDown is the number of consecutive errors before a service is marked down
const ErrorThresholdDown = 3

// ServiceHealth represents the health status of an external service
type ServiceHealth struct {
	Name           string    `json:"name"`
	DisplayName    string    `json:"displayName"`
	Status         string    `json:"status"`
	LastCheck      time.Time `json:"lastCheck"`
	LastSuccess    time.Time `json:"lastSuccess"`
	ErrorCount     int       `json:"errorCount"`
	Message        string    `json:"message,omitempty"`
	ResponseTimeMs int64     `json:"responseTimeMs"`
}

// ServiceStatus represents a service connection status for the settings dashboard.
// Maps from internal ServiceHealth to user-facing status format.
type ServiceStatus struct {
	Name           string     `json:"name"`
	DisplayName    string     `json:"displayName"`
	Status         string     `json:"status"` // "connected", "rate_limited", "error", "disconnected", "unconfigured"
	Message        string     `json:"message"`
	LastSuccessAt  *time.Time `json:"lastSuccessAt"`
	LastCheckAt    time.Time  `json:"lastCheckAt"`
	ResponseTimeMs int64      `json:"responseTimeMs"`
	ErrorMessage   string     `json:"errorMessage,omitempty"`
}

// ServiceStatus constants for the settings dashboard
const (
	StatusConnected    = "connected"
	StatusRateLimited  = "rate_limited"
	StatusError        = "error"
	StatusDisconnected = "disconnected"
	StatusUnconfigured = "unconfigured"
)

// NewServiceHealth creates a new ServiceHealth with healthy status
func NewServiceHealth(name, displayName string) *ServiceHealth {
	return &ServiceHealth{
		Name:        name,
		DisplayName: displayName,
		Status:      ServiceStatusHealthy,
		ErrorCount:  0,
	}
}

// IsHealthy returns true if the service is healthy
func (h *ServiceHealth) IsHealthy() bool {
	return h.Status == ServiceStatusHealthy
}

// IsDegraded returns true if the service is degraded
func (h *ServiceHealth) IsDegraded() bool {
	return h.Status == ServiceStatusDegraded
}

// IsDown returns true if the service is down
func (h *ServiceHealth) IsDown() bool {
	return h.Status == ServiceStatusDown
}

// RecordError records an error and updates the status
func (h *ServiceHealth) RecordError(message string) {
	h.LastCheck = time.Now()
	h.ErrorCount++
	h.Message = message

	if h.ErrorCount >= ErrorThresholdDown {
		h.Status = ServiceStatusDown
	} else {
		h.Status = ServiceStatusDegraded
	}
}

// RecordSuccess records a successful check and resets the error count
func (h *ServiceHealth) RecordSuccess() {
	now := time.Now()
	h.LastCheck = now
	h.LastSuccess = now
	h.ErrorCount = 0
	h.Message = ""
	h.Status = ServiceStatusHealthy
}

// SetResponseTime sets the response time in milliseconds from the last health check
func (h *ServiceHealth) SetResponseTime(ms int64) {
	h.ResponseTimeMs = ms
}

// ToServiceStatus converts internal ServiceHealth to user-facing ServiceStatus
func (h *ServiceHealth) ToServiceStatus() ServiceStatus {
	status := ServiceStatus{
		Name:           h.Name,
		DisplayName:    h.DisplayName,
		LastCheckAt:    h.LastCheck,
		ResponseTimeMs: h.ResponseTimeMs,
	}

	if !h.LastSuccess.IsZero() {
		t := h.LastSuccess
		status.LastSuccessAt = &t
	}

	switch h.Status {
	case ServiceStatusHealthy:
		status.Status = StatusConnected
		status.Message = "已連線"
	case ServiceStatusDegraded:
		if isUnconfiguredError(h.Message) {
			status.Status = StatusUnconfigured
			status.Message = "未設定"
		} else if isRateLimitError(h.Message) {
			status.Status = StatusRateLimited
			status.Message = "速率限制中"
		} else {
			status.Status = StatusError
			status.Message = h.Message
		}
		status.ErrorMessage = h.Message
	case ServiceStatusDown:
		if isUnconfiguredError(h.Message) {
			status.Status = StatusUnconfigured
			status.Message = "未設定"
		} else {
			status.Status = StatusDisconnected
			status.Message = h.Message
		}
		status.ErrorMessage = h.Message
	default:
		status.Status = StatusUnconfigured
		status.Message = "未設定"
	}

	return status
}

// isRateLimitError detects rate limit errors from error messages
func isRateLimitError(msg string) bool {
	lower := strings.ToLower(msg)
	return strings.Contains(lower, "rate limit") ||
		strings.Contains(lower, "rate_limit") ||
		strings.Contains(lower, "429") ||
		strings.Contains(lower, "too many requests")
}

// isUnconfiguredError detects if a service is not configured
func isUnconfiguredError(msg string) bool {
	lower := strings.ToLower(msg)
	return strings.Contains(lower, "not configured") ||
		strings.Contains(lower, "not enabled")
}

// AllServiceStatuses returns all service statuses in the settings dashboard format
func (s *ServicesHealth) AllServiceStatuses() []ServiceStatus {
	statuses := make([]ServiceStatus, 0, 5)
	for _, svc := range s.AllServices() {
		statuses = append(statuses, svc.ToServiceStatus())
	}
	return statuses
}

// DegradedResult represents a result with missing or fallback data
type DegradedResult struct {
	Data             interface{}      `json:"data"`
	DegradationLevel DegradationLevel `json:"degradationLevel"`
	MissingFields    []string         `json:"missingFields,omitempty"`
	FallbackUsed     []string         `json:"fallbackUsed,omitempty"`
	Message          string           `json:"message,omitempty"`
}

// HasMissingFields returns true if any fields are missing
func (r *DegradedResult) HasMissingFields() bool {
	return len(r.MissingFields) > 0
}

// HasFallbackUsed returns true if any fallback sources were used
func (r *DegradedResult) HasFallbackUsed() bool {
	return len(r.FallbackUsed) > 0
}

// IsDegraded returns true if the result is from a degraded state
func (r *DegradedResult) IsDegraded() bool {
	return r.DegradationLevel != DegradationNormal
}

// FieldAvailability represents the availability of a single field
type FieldAvailability struct {
	Field     string `json:"field"`
	Available bool   `json:"available"`
	Source    string `json:"source,omitempty"`
}

// ServicesHealth holds health status for all external services
type ServicesHealth struct {
	TMDb        *ServiceHealth `json:"tmdb"`
	Douban      *ServiceHealth `json:"douban"`
	Wikipedia   *ServiceHealth `json:"wikipedia"`
	AI          *ServiceHealth `json:"ai"`
	QBittorrent *ServiceHealth `json:"qbittorrent"`
}

// NewServicesHealth creates a new ServicesHealth with all services healthy
func NewServicesHealth() *ServicesHealth {
	return &ServicesHealth{
		TMDb:        NewServiceHealth(string(ServiceNameTMDb), "TMDb API"),
		Douban:      NewServiceHealth(string(ServiceNameDouban), "Douban Scraper"),
		Wikipedia:   NewServiceHealth(string(ServiceNameWikipedia), "Wikipedia API"),
		AI:          NewServiceHealth(string(ServiceNameAI), "AI Parser"),
		QBittorrent: NewServiceHealth(string(ServiceNameQBittorrent), "qBittorrent"),
	}
}

// GetService returns the health of a specific service by name
func (s *ServicesHealth) GetService(name ServiceName) *ServiceHealth {
	switch name {
	case ServiceNameTMDb:
		return s.TMDb
	case ServiceNameDouban:
		return s.Douban
	case ServiceNameWikipedia:
		return s.Wikipedia
	case ServiceNameAI:
		return s.AI
	case ServiceNameQBittorrent:
		return s.QBittorrent
	default:
		return nil
	}
}

// AllServices returns all service health statuses as a slice
func (s *ServicesHealth) AllServices() []*ServiceHealth {
	return []*ServiceHealth{s.TMDb, s.Douban, s.Wikipedia, s.AI, s.QBittorrent}
}

// HealthStatusResponse represents the API response for health status endpoint
type HealthStatusResponse struct {
	DegradationLevel DegradationLevel `json:"degradationLevel"`
	Services         *ServicesHealth  `json:"services"`
	Message          string           `json:"message"`
}
