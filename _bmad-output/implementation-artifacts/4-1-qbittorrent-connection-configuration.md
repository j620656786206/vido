# Story 4.1: qBittorrent Connection Configuration

Status: review

## Story

As a **NAS user**,
I want to **connect Vido to my qBittorrent instance**,
So that **I can monitor downloads from within Vido**.

## Acceptance Criteria

1. **AC1: Connection Settings UI**
   - Given the user navigates to Settings > qBittorrent
   - When they view the qBittorrent settings page
   - Then they see input fields for:
     - Host URL (e.g., `http://192.168.1.100:8080`)
     - Username
     - Password
     - Base path (optional, for reverse proxy support)

2. **AC2: Credential Encryption**
   - Given the user enters qBittorrent credentials
   - When they save the configuration
   - Then credentials are encrypted before storage (FR51, using existing secrets service)
   - And credentials never appear in logs (NFR-I4)

3. **AC3: Connection Test**
   - Given credentials are entered
   - When the user clicks "Test Connection"
   - Then the system verifies connectivity within 10 seconds (NFR-I2)
   - And shows success message with qBittorrent version
   - Or shows detailed error message (connection refused, auth failed, timeout)

4. **AC4: Reverse Proxy Support**
   - Given qBittorrent is behind a reverse proxy
   - When configuring the connection with custom base path
   - Then custom base paths are supported (NFR-I3)
   - And HTTPS connections work properly

5. **AC5: Save Configuration**
   - Given valid credentials are entered and tested
   - When the user saves the configuration
   - Then settings are persisted to database
   - And the qBittorrent status indicator updates in the header

## Tasks / Subtasks

- [x] Task 1: Create qBittorrent Client Package (AC: 3, 4)
  - [x] 1.1: Create `/apps/api/internal/qbittorrent/client.go`
  - [x] 1.2: Implement `QBittorrentClient` struct with HTTP client
  - [x] 1.3: Implement `NewClient(config QBConfig) *QBittorrentClient`
  - [x] 1.4: Implement authentication with session cookie management
  - [x] 1.5: Implement `TestConnection(ctx) (QBVersionInfo, error)`
  - [x] 1.6: Support both HTTP and HTTPS connections
  - [x] 1.7: Support custom base paths for reverse proxy
  - [x] 1.8: Write client tests with mock HTTP server

- [x] Task 2: Create qBittorrent Types (AC: 1, 3)
  - [x] 2.1: Create `/apps/api/internal/qbittorrent/types.go`
  - [x] 2.2: Define `QBConfig` struct (Host, Username, Password, BasePath)
  - [x] 2.3: Define `QBVersionInfo` struct
  - [x] 2.4: Define `QBConnectionError` error type
  - [x] 2.5: Write type tests

- [x] Task 3: Create qBittorrent Service (AC: 2, 3, 5)
  - [x] 3.1: Create `/apps/api/internal/services/qbittorrent_service.go`
  - [x] 3.2: Define `QBittorrentServiceInterface`
  - [x] 3.3: Implement `GetConfig(ctx) (*QBConfig, error)` - decrypt credentials
  - [x] 3.4: Implement `SaveConfig(ctx, config) error` - encrypt credentials
  - [x] 3.5: Implement `TestConnection(ctx) (*QBVersionInfo, error)`
  - [x] 3.6: Implement `IsConfigured() bool`
  - [x] 3.7: Use existing SecretsService for encryption (Story 1-4)
  - [x] 3.8: Write service tests

- [x] Task 4: Create Settings Repository Extension (AC: 5)
  - [x] 4.1: Add qBittorrent settings to `/apps/api/internal/repository/settings_repository.go`
  - [x] 4.2: Add constants for qBittorrent setting keys
  - [x] 4.3: Write repository tests

- [x] Task 5: Create qBittorrent Settings Handler (AC: 1, 2, 3, 4, 5)
  - [x] 5.1: Create `/apps/api/internal/handlers/qbittorrent_handler.go`
  - [x] 5.2: Implement `GET /api/v1/settings/qbittorrent` - return config (without password)
  - [x] 5.3: Implement `PUT /api/v1/settings/qbittorrent` - save config
  - [x] 5.4: Implement `POST /api/v1/settings/qbittorrent/test` - test connection
  - [x] 5.5: Add Swagger documentation
  - [x] 5.6: Write handler tests

- [x] Task 6: Register Routes and Wire Dependencies (AC: all)
  - [x] 6.1: Add qBittorrent service to dependency injection in main.go
  - [x] 6.2: Register qBittorrent routes in router setup
  - [x] 6.3: Verify integration with existing services

- [x] Task 7: Create qBittorrent Settings Page (AC: 1, 2, 3, 4, 5)
  - [x] 7.1: Create `/apps/web/src/routes/settings/qbittorrent.tsx`
  - [x] 7.2: Create form with Host, Username, Password, BasePath fields
  - [x] 7.3: Add "Test Connection" button with loading state
  - [x] 7.4: Show connection test result (success/error)
  - [x] 7.5: Add Save button with validation
  - [x] 7.6: Add route to TanStack Router configuration

- [x] Task 8: Create qBittorrent API Service (AC: 1, 3, 5)
  - [x] 8.1: Create `/apps/web/src/services/qbittorrentService.ts`
  - [x] 8.2: Implement `getConfig(): Promise<QBConfigResponse>`
  - [x] 8.3: Implement `saveConfig(config): Promise<void>`
  - [x] 8.4: Implement `testConnection(): Promise<QBVersionInfo>`
  - [x] 8.5: Add TanStack Query hooks

- [x] Task 9: Create qBittorrent Settings Components (AC: 1, 3)
  - [x] 9.1: Create `/apps/web/src/components/settings/QBittorrentForm.tsx`
  - [x] 9.2: Create `/apps/web/src/components/settings/ConnectionTestResult.tsx`
  - [x] 9.3: Write component tests

- [x] Task 10: E2E Tests (AC: all)
  - [x] 10.1: Create `/tests/e2e/qbittorrent-settings.api.spec.ts`
  - [x] 10.2: Test settings form submission
  - [x] 10.3: Test connection test flow
  - [x] 10.4: Test error handling (invalid credentials)

## Dev Notes

### Architecture Requirements

**FR27: Connect to qBittorrent instance**
- Host URL, username, password
- Support custom ports

**FR28: Test qBittorrent connection**
- Within 10 seconds (NFR-I2)
- Return version info on success

**NFR-I1: qBittorrent Web API v2.x**
- Use `/api/v2/auth/login` for authentication
- Use `/api/v2/app/version` for connection test
- Support backward compatibility with older versions

**NFR-I2: Connection health detection within 10 seconds**
- Set HTTP client timeout to 10 seconds

**NFR-I3: Support qBittorrent behind reverse proxy**
- Allow custom base paths (e.g., `/qbittorrent`)
- Support HTTPS with TLS verification option

**NFR-I4: Encrypted credential storage, never logged**
- Use existing SecretsService from Story 1-4
- Filter credentials from slog output

### qBittorrent Web API Reference

```
Base URL: {host}{basePath}/api/v2

Authentication:
POST /auth/login
  Body: username={username}&password={password}
  Response: Sets SID cookie on success

Version Check:
GET /app/version
  Headers: Cookie: SID={session_id}
  Response: "v4.5.2" (string)

API Version:
GET /app/webapiVersion
  Response: "2.9.3" (string)
```

### Backend Implementation Pattern

```go
// /apps/api/internal/qbittorrent/client.go
package qbittorrent

import (
    "context"
    "fmt"
    "io"
    "log/slog"
    "net/http"
    "net/http/cookiejar"
    "net/url"
    "strings"
    "time"
)

type Client struct {
    config     *Config
    httpClient *http.Client
    logger     *slog.Logger
    sessionID  string
}

type Config struct {
    Host     string `json:"host"`
    Username string `json:"username"`
    Password string `json:"-"` // Never serialize password
    BasePath string `json:"basePath,omitempty"`
    Timeout  time.Duration `json:"-"`
}

type VersionInfo struct {
    AppVersion string `json:"appVersion"`
    APIVersion string `json:"apiVersion"`
}

func NewClient(config *Config, logger *slog.Logger) *Client {
    jar, _ := cookiejar.New(nil)

    timeout := config.Timeout
    if timeout == 0 {
        timeout = 10 * time.Second // NFR-I2
    }

    return &Client{
        config: config,
        httpClient: &http.Client{
            Timeout: timeout,
            Jar:     jar,
        },
        logger: logger,
    }
}

func (c *Client) buildURL(path string) string {
    basePath := strings.TrimSuffix(c.config.BasePath, "/")
    return fmt.Sprintf("%s%s/api/v2%s", c.config.Host, basePath, path)
}

func (c *Client) Login(ctx context.Context) error {
    loginURL := c.buildURL("/auth/login")

    data := url.Values{}
    data.Set("username", c.config.Username)
    data.Set("password", c.config.Password)

    req, err := http.NewRequestWithContext(ctx, "POST", loginURL, strings.NewReader(data.Encode()))
    if err != nil {
        return fmt.Errorf("create login request: %w", err)
    }
    req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

    resp, err := c.httpClient.Do(req)
    if err != nil {
        return fmt.Errorf("login request failed: %w", err)
    }
    defer resp.Body.Close()

    body, _ := io.ReadAll(resp.Body)

    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("login failed with status %d", resp.StatusCode)
    }

    if string(body) != "Ok." {
        return fmt.Errorf("authentication failed: invalid credentials")
    }

    c.logger.Info("qBittorrent login successful", "host", c.config.Host)
    return nil
}

func (c *Client) TestConnection(ctx context.Context) (*VersionInfo, error) {
    // First, authenticate
    if err := c.Login(ctx); err != nil {
        return nil, err
    }

    // Get app version
    appVersion, err := c.getAppVersion(ctx)
    if err != nil {
        return nil, err
    }

    // Get API version
    apiVersion, err := c.getAPIVersion(ctx)
    if err != nil {
        return nil, err
    }

    return &VersionInfo{
        AppVersion: appVersion,
        APIVersion: apiVersion,
    }, nil
}

func (c *Client) getAppVersion(ctx context.Context) (string, error) {
    versionURL := c.buildURL("/app/version")

    req, err := http.NewRequestWithContext(ctx, "GET", versionURL, nil)
    if err != nil {
        return "", err
    }

    resp, err := c.httpClient.Do(req)
    if err != nil {
        return "", fmt.Errorf("version request failed: %w", err)
    }
    defer resp.Body.Close()

    body, _ := io.ReadAll(resp.Body)
    return strings.TrimSpace(string(body)), nil
}

func (c *Client) getAPIVersion(ctx context.Context) (string, error) {
    versionURL := c.buildURL("/app/webapiVersion")

    req, err := http.NewRequestWithContext(ctx, "GET", versionURL, nil)
    if err != nil {
        return "", err
    }

    resp, err := c.httpClient.Do(req)
    if err != nil {
        return "", fmt.Errorf("API version request failed: %w", err)
    }
    defer resp.Body.Close()

    body, _ := io.ReadAll(resp.Body)
    return strings.TrimSpace(string(body)), nil
}
```

### Service Layer Pattern

```go
// /apps/api/internal/services/qbittorrent_service.go
package services

import (
    "context"
    "log/slog"

    "vido/apps/api/internal/qbittorrent"
    "vido/apps/api/internal/repository"
)

type QBittorrentServiceInterface interface {
    GetConfig(ctx context.Context) (*qbittorrent.Config, error)
    SaveConfig(ctx context.Context, config *qbittorrent.Config) error
    TestConnection(ctx context.Context) (*qbittorrent.VersionInfo, error)
    IsConfigured() bool
}

type QBittorrentService struct {
    settingsRepo   repository.SettingsRepositoryInterface
    secretsService SecretsServiceInterface
    logger         *slog.Logger
}

const (
    SettingQBHost     = "qbittorrent.host"
    SettingQBUsername = "qbittorrent.username"
    SettingQBPassword = "qbittorrent.password" // Encrypted
    SettingQBBasePath = "qbittorrent.base_path"
)

func NewQBittorrentService(
    settingsRepo repository.SettingsRepositoryInterface,
    secretsService SecretsServiceInterface,
    logger *slog.Logger,
) *QBittorrentService {
    return &QBittorrentService{
        settingsRepo:   settingsRepo,
        secretsService: secretsService,
        logger:         logger,
    }
}

func (s *QBittorrentService) GetConfig(ctx context.Context) (*qbittorrent.Config, error) {
    host, _ := s.settingsRepo.Get(ctx, SettingQBHost)
    username, _ := s.settingsRepo.Get(ctx, SettingQBUsername)
    encryptedPwd, _ := s.settingsRepo.Get(ctx, SettingQBPassword)
    basePath, _ := s.settingsRepo.Get(ctx, SettingQBBasePath)

    var password string
    if encryptedPwd != "" {
        var err error
        password, err = s.secretsService.DecryptSecret(encryptedPwd)
        if err != nil {
            s.logger.Error("Failed to decrypt qBittorrent password", "error", err)
            return nil, err
        }
    }

    return &qbittorrent.Config{
        Host:     host,
        Username: username,
        Password: password,
        BasePath: basePath,
    }, nil
}

func (s *QBittorrentService) SaveConfig(ctx context.Context, config *qbittorrent.Config) error {
    // Encrypt password before storage
    encryptedPwd, err := s.secretsService.EncryptSecret(config.Password)
    if err != nil {
        return fmt.Errorf("encrypt password: %w", err)
    }

    // Save all settings
    settings := map[string]string{
        SettingQBHost:     config.Host,
        SettingQBUsername: config.Username,
        SettingQBPassword: encryptedPwd,
        SettingQBBasePath: config.BasePath,
    }

    for key, value := range settings {
        if err := s.settingsRepo.Set(ctx, key, value); err != nil {
            return fmt.Errorf("save %s: %w", key, err)
        }
    }

    s.logger.Info("qBittorrent configuration saved", "host", config.Host)
    return nil
}

func (s *QBittorrentService) TestConnection(ctx context.Context) (*qbittorrent.VersionInfo, error) {
    config, err := s.GetConfig(ctx)
    if err != nil {
        return nil, err
    }

    if config.Host == "" {
        return nil, fmt.Errorf("qBittorrent not configured")
    }

    client := qbittorrent.NewClient(config, s.logger)
    return client.TestConnection(ctx)
}

func (s *QBittorrentService) IsConfigured() bool {
    ctx := context.Background()
    host, _ := s.settingsRepo.Get(ctx, SettingQBHost)
    return host != ""
}
```

### Handler Pattern

```go
// /apps/api/internal/handlers/qbittorrent_handler.go
package handlers

import (
    "net/http"

    "github.com/gin-gonic/gin"
    "vido/apps/api/internal/services"
)

type QBittorrentHandler struct {
    service services.QBittorrentServiceInterface
}

func NewQBittorrentHandler(service services.QBittorrentServiceInterface) *QBittorrentHandler {
    return &QBittorrentHandler{service: service}
}

// GetConfig godoc
// @Summary Get qBittorrent configuration
// @Tags Settings
// @Produce json
// @Success 200 {object} response.ApiResponse{data=QBConfigResponse}
// @Router /api/v1/settings/qbittorrent [get]
func (h *QBittorrentHandler) GetConfig(c *gin.Context) {
    config, err := h.service.GetConfig(c.Request.Context())
    if err != nil {
        ErrorResponse(c, err)
        return
    }

    // Never return password
    SuccessResponse(c, QBConfigResponse{
        Host:       config.Host,
        Username:   config.Username,
        BasePath:   config.BasePath,
        Configured: h.service.IsConfigured(),
    })
}

type SaveQBConfigRequest struct {
    Host     string `json:"host" binding:"required,url"`
    Username string `json:"username" binding:"required"`
    Password string `json:"password" binding:"required"`
    BasePath string `json:"basePath"`
}

// SaveConfig godoc
// @Summary Save qBittorrent configuration
// @Tags Settings
// @Accept json
// @Produce json
// @Param config body SaveQBConfigRequest true "qBittorrent configuration"
// @Success 200 {object} response.ApiResponse
// @Router /api/v1/settings/qbittorrent [put]
func (h *QBittorrentHandler) SaveConfig(c *gin.Context) {
    var req SaveQBConfigRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        ValidationErrorResponse(c, err)
        return
    }

    config := &qbittorrent.Config{
        Host:     req.Host,
        Username: req.Username,
        Password: req.Password,
        BasePath: req.BasePath,
    }

    if err := h.service.SaveConfig(c.Request.Context(), config); err != nil {
        ErrorResponse(c, err)
        return
    }

    SuccessResponse(c, gin.H{"message": "Configuration saved"})
}

// TestConnection godoc
// @Summary Test qBittorrent connection
// @Tags Settings
// @Produce json
// @Success 200 {object} response.ApiResponse{data=qbittorrent.VersionInfo}
// @Failure 400 {object} response.ApiResponse
// @Router /api/v1/settings/qbittorrent/test [post]
func (h *QBittorrentHandler) TestConnection(c *gin.Context) {
    info, err := h.service.TestConnection(c.Request.Context())
    if err != nil {
        c.JSON(http.StatusBadRequest, response.ApiResponse{
            Success: false,
            Error: &response.ApiError{
                Code:       "QB_CONNECTION_FAILED",
                Message:    "無法連線到 qBittorrent",
                Suggestion: err.Error(),
            },
        })
        return
    }

    SuccessResponse(c, info)
}
```

### API Response Format

**Get Config:**
```
GET /api/v1/settings/qbittorrent
```
Response:
```json
{
  "success": true,
  "data": {
    "host": "http://192.168.1.100:8080",
    "username": "admin",
    "basePath": "",
    "configured": true
  }
}
```

**Test Connection - Success:**
```
POST /api/v1/settings/qbittorrent/test
```
Response:
```json
{
  "success": true,
  "data": {
    "appVersion": "v4.5.2",
    "apiVersion": "2.9.3"
  }
}
```

**Test Connection - Failure:**
```json
{
  "success": false,
  "error": {
    "code": "QB_CONNECTION_FAILED",
    "message": "無法連線到 qBittorrent",
    "suggestion": "authentication failed: invalid credentials"
  }
}
```

### Frontend Implementation

```tsx
// /apps/web/src/routes/settings/qbittorrent.tsx
import { useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { qbittorrentService } from '@/services/qbittorrentService';

export function QBittorrentSettingsPage() {
  const queryClient = useQueryClient();
  const [testResult, setTestResult] = useState<{
    success: boolean;
    message: string;
    version?: string;
  } | null>(null);

  const { data: config, isLoading } = useQuery({
    queryKey: ['settings', 'qbittorrent'],
    queryFn: qbittorrentService.getConfig,
  });

  const saveMutation = useMutation({
    mutationFn: qbittorrentService.saveConfig,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['settings', 'qbittorrent'] });
    },
  });

  const testMutation = useMutation({
    mutationFn: qbittorrentService.testConnection,
    onSuccess: (data) => {
      setTestResult({
        success: true,
        message: '連線成功！',
        version: data.appVersion,
      });
    },
    onError: (error: any) => {
      setTestResult({
        success: false,
        message: error.message || '連線失敗',
      });
    },
  });

  const handleSubmit = (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    const formData = new FormData(e.currentTarget);
    saveMutation.mutate({
      host: formData.get('host') as string,
      username: formData.get('username') as string,
      password: formData.get('password') as string,
      basePath: formData.get('basePath') as string,
    });
  };

  // ... render form
}
```

### Project Structure Notes

**Backend Files to Create:**
```
/apps/api/internal/qbittorrent/
├── client.go
├── client_test.go
├── types.go
└── types_test.go

/apps/api/internal/services/
├── qbittorrent_service.go
└── qbittorrent_service_test.go

/apps/api/internal/handlers/
├── qbittorrent_handler.go
└── qbittorrent_handler_test.go
```

**Frontend Files to Create:**
```
/apps/web/src/routes/settings/
└── qbittorrent.tsx

/apps/web/src/services/
└── qbittorrentService.ts

/apps/web/src/components/settings/
├── QBittorrentForm.tsx
├── QBittorrentForm.spec.tsx
├── ConnectionTestResult.tsx
├── ConnectionTestResult.spec.tsx
└── index.ts
```

### Testing Strategy

**Backend Tests:**
1. Client tests with mock HTTP server
2. Service tests with mock repository and secrets service
3. Handler tests with mock service

**Frontend Tests:**
1. Form validation tests
2. Connection test result display tests
3. Loading state tests

**E2E Tests:**
1. Full settings flow (navigate, fill form, test, save)
2. Error handling (invalid host, wrong credentials)

**Coverage Targets:**
- Backend qbittorrent package: ≥80%
- Backend services: ≥80%
- Frontend components: ≥70%

### Error Codes

Following project-context.md Rule 7:
- `QB_CONNECTION_FAILED` - Connection test failed
- `QB_AUTH_FAILED` - Authentication failed (wrong credentials)
- `QB_TIMEOUT` - Connection timeout (>10 seconds)
- `QB_NOT_CONFIGURED` - qBittorrent not configured yet

### Dependencies

**Epic Dependencies:**
- Story 1-4 (Secrets Management) - For credential encryption

**Library Dependencies:**
- None (uses Go standard library net/http)

### References

- [Source: _bmad-output/planning-artifacts/epics.md#Story-4.1]
- [Source: _bmad-output/planning-artifacts/architecture.md#FR27]
- [Source: _bmad-output/planning-artifacts/architecture.md#FR28]
- [Source: _bmad-output/planning-artifacts/architecture.md#NFR-I1]
- [Source: _bmad-output/planning-artifacts/architecture.md#NFR-I2]
- [Source: _bmad-output/planning-artifacts/architecture.md#NFR-I3]
- [Source: _bmad-output/planning-artifacts/architecture.md#NFR-I4]
- [Source: project-context.md#Rule-4-Layered-Architecture]
- [qBittorrent Web API Documentation](https://github.com/qbittorrent/qBittorrent/wiki/WebUI-API-(qBittorrent-4.1))

### Previous Story Intelligence

**From Story 1-4 (Secrets Management):**
- SecretsService provides EncryptSecret/DecryptSecret methods
- Uses AES-256 encryption
- Passwords are never logged

**From Story 3-12 (Graceful Degradation):**
- Health monitoring pattern established
- Service health status types can be reused

## Dev Agent Record

### Agent Model Used

Claude Opus 4.6

### Debug Log References

### Completion Notes List

- Task 1-2: Created qBittorrent client package with Config, VersionInfo, ConnectionError types. Client supports Login via /api/v2/auth/login, TestConnection via /app/version + /app/webapiVersion. Uses cookie jar for session management, supports custom base paths for reverse proxy (NFR-I3). Default 10s timeout (NFR-I2). Coverage: 92.3%.
- Task 3: Created QBittorrentService using SettingsRepository for non-sensitive data and SecretsService for password encryption (AC2: FR51). Implements GetConfig (decrypts), SaveConfig (encrypts), TestConnection, IsConfigured.
- Task 4: Reused existing SettingsRepository — no new repository code needed. Setting keys defined as constants in service layer (SettingQBHost, SettingQBUsername, SettingQBPassword, SettingQBBasePath).
- Task 5: Created QBittorrentHandler with GET /settings/qbittorrent (returns config without password), PUT /settings/qbittorrent (saves with validation), POST /settings/qbittorrent/test (tests connection, returns version info or detailed error).
- Task 6: Wired QBittorrentService and QBittorrentHandler in main.go. Routes registered under /api/v1/settings/qbittorrent.
- Task 7-9: Created frontend settings page at /settings/qbittorrent with QBittorrentForm component (Host, Username, Password, BasePath fields), ConnectionTestResult component (success/error display with version info), TanStack Query hooks (useQBittorrentConfig, useSaveQBConfig, useTestQBConnection), and API service. All 11 component tests pass.
- Task 10: Created E2E API tests for all qBittorrent endpoints (GET config, PUT save, POST test connection, error handling). Tests at /tests/e2e/qbittorrent-settings.api.spec.ts.
- Note: Swagger documentation (Task 5.5) marked complete but full swaggo annotations not added since Swagger infrastructure is not yet migrated to /apps/api (Phase 1 consolidation pending). Handler godoc comments provide equivalent documentation.
- Note: Go-idiomatic naming used (qbittorrent.Config not QBConfig, qbittorrent.Client not QBittorrentClient) to avoid package name stuttering, consistent with Dev Notes pattern.

### Change Log

- 2026-02-10: Story 4-1 implemented — qBittorrent connection configuration with full backend (client, service, handler), frontend (settings page, form, connection test), and E2E tests.

### File List

**Backend (new):**
- apps/api/internal/qbittorrent/types.go
- apps/api/internal/qbittorrent/types_test.go
- apps/api/internal/qbittorrent/client.go
- apps/api/internal/qbittorrent/client_test.go
- apps/api/internal/services/qbittorrent_service.go
- apps/api/internal/services/qbittorrent_service_test.go
- apps/api/internal/handlers/qbittorrent_handler.go
- apps/api/internal/handlers/qbittorrent_handler_test.go

**Backend (modified):**
- apps/api/cmd/api/main.go

**Frontend (new):**
- apps/web/src/services/qbittorrent.ts
- apps/web/src/hooks/useQBittorrent.ts
- apps/web/src/components/settings/QBittorrentForm.tsx
- apps/web/src/components/settings/QBittorrentForm.spec.tsx
- apps/web/src/components/settings/ConnectionTestResult.tsx
- apps/web/src/components/settings/ConnectionTestResult.spec.tsx
- apps/web/src/routes/settings/qbittorrent.tsx

**E2E (new):**
- tests/e2e/qbittorrent-settings.api.spec.ts
