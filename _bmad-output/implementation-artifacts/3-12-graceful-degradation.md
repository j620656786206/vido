# Story 3.12: Graceful Degradation

Status: ready-for-dev

## Story

As a **media collector**,
I want the **system to never completely fail**,
So that **I always have options even in worst-case scenarios**.

## Acceptance Criteria

1. **AC1: All Sources Fail Fallback**
   - Given all metadata sources fail (TMDb, Douban, Wikipedia, AI)
   - When the user views the file
   - Then they see:
     - The original filename
     - "Unable to auto-identify" message
     - Three clear options: Manual search, Edit filename, Skip for now

2. **AC2: AI Service Down Fallback**
   - Given the AI service is down or quota exceeded
   - When parsing a fansub filename
   - Then the system falls back to regex parsing
   - And notifies: "AI 服務暫時無法使用，使用基本解析" (NFR-R11)

3. **AC3: Core Functionality Availability**
   - Given core functionality is needed
   - When all external APIs are unavailable
   - Then the library browsing and search still work (NFR-R13)
   - And only new metadata fetching is affected

4. **AC4: Partial Success Handling**
   - Given metadata is partially retrieved (e.g., title found but no poster)
   - When viewing the result
   - Then the available data is shown
   - And missing items use placeholders with "Data unavailable" message

## Tasks / Subtasks

- [ ] Task 1: Create Degradation State Types (AC: 1, 2, 3, 4)
  - [ ] 1.1: Create `/apps/api/internal/models/degradation.go`
  - [ ] 1.2: Define `DegradationLevel` enum (normal, partial, minimal, offline)
  - [ ] 1.3: Define `ServiceHealth` struct for each external service
  - [ ] 1.4: Define `DegradedResult` struct for partial data
  - [ ] 1.5: Write model tests

- [ ] Task 2: Create Service Health Monitor (AC: 2, 3)
  - [ ] 2.1: Create `/apps/api/internal/health/monitor.go`
  - [ ] 2.2: Track health status of TMDb, Douban, Wikipedia, AI services
  - [ ] 2.3: Implement health check endpoints
  - [ ] 2.4: Emit health change events
  - [ ] 2.5: Write monitor tests

- [ ] Task 3: Implement AI Fallback to Regex (AC: 2)
  - [ ] 3.1: Update AI parser to detect quota/service errors
  - [ ] 3.2: Fallback to regex parser when AI unavailable
  - [ ] 3.3: Set result source to "regex_fallback"
  - [ ] 3.4: Emit notification for UI feedback
  - [ ] 3.5: Write fallback tests

- [ ] Task 4: Create Partial Result Handler (AC: 4)
  - [ ] 4.1: Create `/apps/api/internal/metadata/partial.go`
  - [ ] 4.2: Merge partial data from multiple sources
  - [ ] 4.3: Fill missing fields with placeholders
  - [ ] 4.4: Track which fields are real vs placeholder
  - [ ] 4.5: Write partial result tests

- [ ] Task 5: Create Degradation Service (AC: 1, 2, 3, 4)
  - [ ] 5.1: Create `/apps/api/internal/services/degradation_service.go`
  - [ ] 5.2: Define `DegradationServiceInterface`
  - [ ] 5.3: Implement `GetCurrentLevel()` method
  - [ ] 5.4: Implement `GetServiceHealth()` method
  - [ ] 5.5: Implement `GetDegradedResult()` method
  - [ ] 5.6: Write service tests

- [ ] Task 6: Create Offline Cache System (AC: 3)
  - [ ] 6.1: Cache frequently accessed data locally
  - [ ] 6.2: Serve cached data when APIs unavailable
  - [ ] 6.3: Mark data as "cached" in responses
  - [ ] 6.4: Implement cache invalidation on reconnection
  - [ ] 6.5: Write cache tests

- [ ] Task 7: Create Health Status API (AC: 3)
  - [ ] 7.1: Create `GET /api/v1/health/services` endpoint
  - [ ] 7.2: Return health status of all external services
  - [ ] 7.3: Include degradation level
  - [ ] 7.4: Write handler tests

- [ ] Task 8: Create Fallback UI Components (AC: 1)
  - [ ] 8.1: Create `UnidentifiedFileCard.tsx` component
  - [ ] 8.2: Display original filename
  - [ ] 8.3: Show "Unable to auto-identify" message
  - [ ] 8.4: Add action buttons (Manual search, Edit filename, Skip)
  - [ ] 8.5: Write component tests

- [ ] Task 9: Create Service Health Banner (AC: 2, 3)
  - [ ] 9.1: Create `ServiceHealthBanner.tsx` component
  - [ ] 9.2: Show warning when services degraded
  - [ ] 9.3: Display which services are affected
  - [ ] 9.4: Auto-dismiss when services recover
  - [ ] 9.5: Write component tests

- [ ] Task 10: Create Placeholder Components (AC: 4)
  - [ ] 10.1: Create `PlaceholderPoster.tsx` component
  - [ ] 10.2: Create `MissingDataIndicator.tsx` component
  - [ ] 10.3: Show which data is unavailable
  - [ ] 10.4: Offer "Retry" option for missing data
  - [ ] 10.5: Write component tests

## Dev Notes

### Architecture Requirements

**FR26: Graceful degradation with manual option**
- Never leave user stuck
- Always provide alternatives

**NFR-R11: AI quota exhausted → regex fallback with notification**
- Fallback to simpler parsing when AI unavailable

**NFR-R13: Core functionality works when external APIs down**
- Library browsing always available
- Search within library works

### Degradation Levels

```go
// DegradationLevel represents the current system degradation state
type DegradationLevel string

const (
    // Normal - All services operational
    DegradationNormal DegradationLevel = "normal"

    // Partial - Some services degraded, full functionality available with delays
    DegradationPartial DegradationLevel = "partial"

    // Minimal - Multiple services down, reduced functionality
    DegradationMinimal DegradationLevel = "minimal"

    // Offline - No external services available, cache-only mode
    DegradationOffline DegradationLevel = "offline"
)

// ServiceHealth represents the health status of an external service
type ServiceHealth struct {
    Name        string    `json:"name"`
    Status      string    `json:"status"`      // "healthy", "degraded", "down"
    LastCheck   time.Time `json:"lastCheck"`
    LastSuccess time.Time `json:"lastSuccess"`
    ErrorCount  int       `json:"errorCount"`
    Message     string    `json:"message,omitempty"`
}

// DegradedResult represents a result with missing or fallback data
type DegradedResult struct {
    Data             interface{}       `json:"data"`
    DegradationLevel DegradationLevel  `json:"degradationLevel"`
    MissingFields    []string          `json:"missingFields,omitempty"`
    FallbackUsed     []string          `json:"fallbackUsed,omitempty"`
    Message          string            `json:"message,omitempty"`
}
```

### Service Health Monitoring

```go
// HealthMonitor tracks health of external services
type HealthMonitor struct {
    services map[string]*ServiceHealth
    checker  HealthChecker
    logger   *slog.Logger
    mu       sync.RWMutex
}

type HealthChecker interface {
    CheckTMDb(ctx context.Context) error
    CheckDouban(ctx context.Context) error
    CheckWikipedia(ctx context.Context) error
    CheckAI(ctx context.Context) error
}

func (m *HealthMonitor) GetDegradationLevel() DegradationLevel {
    m.mu.RLock()
    defer m.mu.RUnlock()

    downCount := 0
    degradedCount := 0

    for _, svc := range m.services {
        switch svc.Status {
        case "down":
            downCount++
        case "degraded":
            degradedCount++
        }
    }

    totalServices := len(m.services)

    // All services down
    if downCount == totalServices {
        return DegradationOffline
    }

    // More than half down
    if downCount > totalServices/2 {
        return DegradationMinimal
    }

    // Any service degraded or down
    if downCount > 0 || degradedCount > 0 {
        return DegradationPartial
    }

    return DegradationNormal
}

func (m *HealthMonitor) StartMonitoring(ctx context.Context, interval time.Duration) {
    ticker := time.NewTicker(interval)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            m.checkAllServices(ctx)
        }
    }
}

func (m *HealthMonitor) checkAllServices(ctx context.Context) {
    checks := []struct {
        name    string
        checker func(context.Context) error
    }{
        {"tmdb", m.checker.CheckTMDb},
        {"douban", m.checker.CheckDouban},
        {"wikipedia", m.checker.CheckWikipedia},
        {"ai", m.checker.CheckAI},
    }

    for _, check := range checks {
        go func(name string, fn func(context.Context) error) {
            err := fn(ctx)
            m.updateServiceHealth(name, err)
        }(check.name, check.checker)
    }
}

func (m *HealthMonitor) updateServiceHealth(name string, err error) {
    m.mu.Lock()
    defer m.mu.Unlock()

    svc := m.services[name]
    svc.LastCheck = time.Now()

    if err == nil {
        svc.Status = "healthy"
        svc.LastSuccess = time.Now()
        svc.ErrorCount = 0
        svc.Message = ""
    } else {
        svc.ErrorCount++

        if svc.ErrorCount >= 3 {
            svc.Status = "down"
        } else {
            svc.Status = "degraded"
        }

        svc.Message = err.Error()
    }

    // Emit health change event if status changed
    // ...
}
```

### AI Fallback to Regex

```go
// ParserService with graceful degradation
type ParserService struct {
    aiParser    AIParserInterface
    regexParser RegexParserInterface
    health      *HealthMonitor
    logger      *slog.Logger
}

func (s *ParserService) Parse(ctx context.Context, filename string) (*ParseResult, error) {
    // Check if this looks like a fansub filename
    if !s.isFansubFilename(filename) {
        return s.regexParser.Parse(filename)
    }

    // Check AI service health
    aiHealth := s.health.GetServiceHealth("ai")
    if aiHealth.Status == "down" {
        s.logger.Info("AI service down, using regex fallback", "filename", filename)
        result, err := s.regexParser.Parse(filename)
        if err != nil {
            return nil, err
        }
        result.Source = "regex_fallback"
        result.DegradationMessage = "AI 服務暫時無法使用，使用基本解析"
        return result, nil
    }

    // Try AI parsing
    result, err := s.aiParser.Parse(ctx, filename)
    if err != nil {
        // Check if error is quota or service related
        if isAIServiceError(err) {
            s.logger.Warn("AI service error, falling back to regex",
                "filename", filename,
                "error", err,
            )
            result, err := s.regexParser.Parse(filename)
            if err != nil {
                return nil, err
            }
            result.Source = "regex_fallback"
            result.DegradationMessage = "AI 服務暫時無法使用，使用基本解析"
            return result, nil
        }
        return nil, err
    }

    return result, nil
}

func isAIServiceError(err error) bool {
    errStr := strings.ToLower(err.Error())
    serviceErrors := []string{
        "quota exceeded", "rate limit", "service unavailable",
        "timeout", "connection refused", "503", "429",
    }
    for _, pattern := range serviceErrors {
        if strings.Contains(errStr, pattern) {
            return true
        }
    }
    return false
}
```

### Partial Result Handler

```go
// PartialResultHandler merges and fills incomplete metadata
type PartialResultHandler struct {
    logger *slog.Logger
}

type FieldAvailability struct {
    Field     string `json:"field"`
    Available bool   `json:"available"`
    Source    string `json:"source,omitempty"`
}

func (h *PartialResultHandler) MergePartialResults(results []*MetadataResult) *DegradedResult {
    merged := &MetadataItem{}
    missing := []string{}
    sources := []string{}

    // Define required fields
    requiredFields := []string{"title", "year", "overview", "posterUrl"}

    for _, field := range requiredFields {
        found := false
        for _, result := range results {
            if result != nil && hasField(result, field) {
                setField(merged, field, getField(result, field))
                sources = append(sources, result.Source)
                found = true
                break
            }
        }
        if !found {
            missing = append(missing, field)
            setPlaceholder(merged, field)
        }
    }

    level := DegradationNormal
    if len(missing) > 0 {
        level = DegradationPartial
        if len(missing) > len(requiredFields)/2 {
            level = DegradationMinimal
        }
    }

    return &DegradedResult{
        Data:             merged,
        DegradationLevel: level,
        MissingFields:    missing,
        FallbackUsed:     sources,
        Message:          h.generateMessage(missing),
    }
}

func (h *PartialResultHandler) generateMessage(missing []string) string {
    if len(missing) == 0 {
        return ""
    }

    fieldLabels := map[string]string{
        "title":     "標題",
        "year":      "年份",
        "overview":  "簡介",
        "posterUrl": "海報",
        "genres":    "類型",
        "cast":      "演員",
    }

    var labels []string
    for _, field := range missing {
        if label, ok := fieldLabels[field]; ok {
            labels = append(labels, label)
        }
    }

    return fmt.Sprintf("以下資料暫時無法取得：%s", strings.Join(labels, "、"))
}

func setPlaceholder(item *MetadataItem, field string) {
    switch field {
    case "title":
        item.Title = "未知標題"
    case "overview":
        item.Overview = "暫無簡介"
    case "posterUrl":
        item.PosterUrl = "/images/placeholder-poster.webp"
    }
}
```

### API Response Format

**Health Status Endpoint:**
```
GET /api/v1/health/services
```

Response:
```json
{
  "success": true,
  "data": {
    "degradationLevel": "partial",
    "services": {
      "tmdb": {
        "name": "TMDb API",
        "status": "healthy",
        "lastCheck": "2026-01-18T12:00:00Z",
        "lastSuccess": "2026-01-18T12:00:00Z"
      },
      "douban": {
        "name": "Douban Scraper",
        "status": "degraded",
        "lastCheck": "2026-01-18T12:00:00Z",
        "lastSuccess": "2026-01-18T11:55:00Z",
        "errorCount": 2,
        "message": "Rate limited"
      },
      "wikipedia": {
        "name": "Wikipedia API",
        "status": "healthy",
        "lastCheck": "2026-01-18T12:00:00Z",
        "lastSuccess": "2026-01-18T12:00:00Z"
      },
      "ai": {
        "name": "AI Parser",
        "status": "down",
        "lastCheck": "2026-01-18T12:00:00Z",
        "lastSuccess": "2026-01-18T10:00:00Z",
        "errorCount": 5,
        "message": "Quota exceeded"
      }
    },
    "message": "部分服務降級中：AI 解析暫時使用基本模式"
  }
}
```

### Frontend Components

**UnidentifiedFileCard.tsx:**
```tsx
interface UnidentifiedFileCardProps {
  filename: string;
  attemptedSources: string[];
  onManualSearch: () => void;
  onEditFilename: () => void;
  onSkip: () => void;
}

const UnidentifiedFileCard: React.FC<UnidentifiedFileCardProps> = ({
  filename,
  attemptedSources,
  onManualSearch,
  onEditFilename,
  onSkip,
}) => {
  return (
    <Card className="border-dashed border-2 border-muted">
      <CardContent className="p-6 text-center space-y-4">
        {/* File Icon */}
        <div className="w-16 h-16 mx-auto bg-muted rounded-lg flex items-center justify-center">
          <FileQuestion className="h-8 w-8 text-muted-foreground" />
        </div>

        {/* Filename */}
        <div className="space-y-1">
          <p className="text-sm font-mono text-muted-foreground truncate max-w-[300px] mx-auto">
            {filename}
          </p>
          <p className="text-lg font-medium">無法自動識別</p>
        </div>

        {/* Attempted Sources */}
        <div className="flex items-center justify-center gap-2 text-xs text-muted-foreground">
          <span>已嘗試：</span>
          {attemptedSources.map((source, i) => (
            <React.Fragment key={source}>
              <span className="text-red-500">{source} ✗</span>
              {i < attemptedSources.length - 1 && <span>→</span>}
            </React.Fragment>
          ))}
        </div>

        {/* Action Buttons */}
        <div className="flex flex-col gap-2 max-w-[200px] mx-auto">
          <Button onClick={onManualSearch} className="w-full">
            <Search className="h-4 w-4 mr-2" />
            手動搜尋
          </Button>
          <Button variant="outline" onClick={onEditFilename} className="w-full">
            <Edit className="h-4 w-4 mr-2" />
            編輯檔名
          </Button>
          <Button variant="ghost" onClick={onSkip} className="w-full text-muted-foreground">
            稍後處理
          </Button>
        </div>
      </CardContent>
    </Card>
  );
};
```

**ServiceHealthBanner.tsx:**
```tsx
interface ServiceHealthBannerProps {
  health: ServiceHealthResponse;
}

const ServiceHealthBanner: React.FC<ServiceHealthBannerProps> = ({ health }) => {
  if (health.degradationLevel === 'normal') return null;

  const bannerConfig = {
    partial: {
      variant: 'warning',
      icon: AlertTriangle,
      title: '部分服務降級中',
    },
    minimal: {
      variant: 'destructive',
      icon: AlertCircle,
      title: '多項服務無法使用',
    },
    offline: {
      variant: 'destructive',
      icon: WifiOff,
      title: '離線模式',
    },
  };

  const config = bannerConfig[health.degradationLevel];
  const Icon = config.icon;

  const affectedServices = Object.entries(health.services)
    .filter(([_, svc]) => svc.status !== 'healthy')
    .map(([_, svc]) => svc.name);

  return (
    <Alert variant={config.variant} className="mb-4">
      <Icon className="h-4 w-4" />
      <AlertTitle>{config.title}</AlertTitle>
      <AlertDescription>
        {health.message}
        {affectedServices.length > 0 && (
          <span className="block mt-1 text-sm">
            受影響的服務：{affectedServices.join('、')}
          </span>
        )}
      </AlertDescription>
    </Alert>
  );
};
```

**PlaceholderPoster.tsx:**
```tsx
interface PlaceholderPosterProps {
  title?: string;
  missingReason?: string;
  onRetry?: () => void;
}

const PlaceholderPoster: React.FC<PlaceholderPosterProps> = ({
  title,
  missingReason = '海報暫時無法載入',
  onRetry,
}) => {
  return (
    <div className="relative aspect-[2/3] bg-muted rounded-lg flex items-center justify-center">
      <div className="text-center p-4 space-y-2">
        <ImageOff className="h-8 w-8 mx-auto text-muted-foreground" />
        {title && (
          <p className="text-sm font-medium line-clamp-2">{title}</p>
        )}
        <p className="text-xs text-muted-foreground">{missingReason}</p>
        {onRetry && (
          <Button variant="ghost" size="sm" onClick={onRetry}>
            <RefreshCw className="h-3 w-3 mr-1" />
            重試
          </Button>
        )}
      </div>
    </div>
  );
};
```

### Project Structure Notes

**Backend Files to Create:**
```
/apps/api/internal/models/
└── degradation.go

/apps/api/internal/health/
├── monitor.go
├── monitor_test.go
├── checker.go
└── checker_test.go

/apps/api/internal/metadata/
├── partial.go
└── partial_test.go

/apps/api/internal/services/
└── degradation_service.go

/apps/api/internal/handlers/
└── health_handler.go
```

**Frontend Files to Create:**
```
/apps/web/src/components/degradation/
├── UnidentifiedFileCard.tsx
├── UnidentifiedFileCard.spec.tsx
├── ServiceHealthBanner.tsx
├── ServiceHealthBanner.spec.tsx
├── PlaceholderPoster.tsx
├── PlaceholderPoster.spec.tsx
├── MissingDataIndicator.tsx
└── index.ts

/apps/web/public/images/
└── placeholder-poster.webp
```

### Testing Strategy

**Backend Tests:**
1. Degradation level calculation tests
2. Health monitor tests (status transitions)
3. AI fallback tests
4. Partial result merging tests
5. Cache fallback tests

**Frontend Tests:**
1. UnidentifiedFileCard action tests
2. ServiceHealthBanner visibility tests
3. PlaceholderPoster rendering tests

**E2E Tests:**
1. Full degradation flow (service down → fallback)
2. Recovery flow (service up → normal mode)
3. Offline mode (all services down)

**Coverage Targets:**
- Backend health package: ≥80%
- Backend services: ≥80%
- Frontend components: ≥70%

### Error Codes

Following project-context.md Rule 7:
- `DEGRADATION_OFFLINE` - All services unavailable
- `DEGRADATION_PARTIAL` - Some services degraded
- `AI_FALLBACK_USED` - Fell back to regex parsing
- `DATA_INCOMPLETE` - Partial data returned

### Dependencies

**Story Dependencies:**
- Story 3.1 (AI Provider) - AI health checking
- Story 3.3 (Fallback Chain) - Multi-source orchestration
- Story 3.7 (Manual Search) - Manual search fallback

### References

- [Source: _bmad-output/planning-artifacts/epics.md#Story-3.12]
- [Source: _bmad-output/planning-artifacts/architecture.md#FR26]
- [Source: _bmad-output/planning-artifacts/architecture.md#NFR-R11]
- [Source: _bmad-output/planning-artifacts/architecture.md#NFR-R13]
- [Source: project-context.md#Rule-4-Layered-Architecture]

### Previous Story Intelligence

**From Story 3.3 (Fallback Chain):**
- Multi-source orchestrator provides fallback chain
- Can detect which sources failed

**From Story 3.11 (Auto-Retry):**
- Retry mechanism handles temporary failures
- Graceful degradation is for permanent/prolonged failures

**From Story 3.7 (Manual Search):**
- Manual search is the final fallback
- UI patterns for manual intervention

## Dev Agent Record

### Agent Model Used

{{agent_model_name_version}}

### Debug Log References

### Completion Notes List

### File List
