# Story 6.11: Performance Metrics Dashboard

Status: ready-for-dev

## Story

As a **system administrator**,
I want to **view performance metrics**,
So that **I can monitor system health and identify issues**.

## Acceptance Criteria

1. **Given** the user opens Settings > Performance, **When** metrics are displayed, **Then** they see: Query latency (p50, p95), Cache hit rate, API response times, Library item count
2. **Given** metrics show concerning values, **When** p95 latency > 500ms or items > 8,000, **Then** a warning is displayed (NFR-SC2) and recommendation: "Consider PostgreSQL migration"
3. **Given** the metrics page is open, **When** viewing trends, **Then** charts show 24-hour and 7-day trends

## Tasks / Subtasks

- [ ] Task 1: Create Metrics Collection Service (AC: 1)
  - [ ] 1.1: Create `/apps/api/internal/services/metrics_service.go` with `MetricsServiceInterface`
  - [ ] 1.2: Implement in-memory metrics collector using sync.Map for thread safety
  - [ ] 1.3: Track: query latency (histogram), cache hits/misses (counter), API response times (histogram)
  - [ ] 1.4: Implement `RecordQueryLatency(duration time.Duration)`
  - [ ] 1.5: Implement `RecordCacheAccess(hit bool)`
  - [ ] 1.6: Implement `RecordAPIResponseTime(endpoint string, duration time.Duration)`
  - [ ] 1.7: Write unit tests (≥80% coverage)

- [ ] Task 2: Create Metrics Storage (AC: 3)
  - [ ] 2.1: Create migration for `performance_metrics` table (timestamp, metric_type, value, context_json)
  - [ ] 2.2: Aggregate and persist metrics every 5 minutes (p50, p95, avg, count)
  - [ ] 2.3: Retain 7 days of minute-level data, 90 days of hourly aggregates
  - [ ] 2.4: Write cleanup job for expired metrics data

- [ ] Task 3: Add Metrics Middleware (AC: 1)
  - [ ] 3.1: Create `/apps/api/internal/middleware/metrics_middleware.go`
  - [ ] 3.2: Gin middleware that records request duration per endpoint
  - [ ] 3.3: Integrate cache hit/miss tracking into existing cache module
  - [ ] 3.4: Integrate query latency tracking into repository layer

- [ ] Task 4: Create Scalability Warning Service (AC: 2)
  - [ ] 4.1: Add `CheckScalabilityWarnings(ctx) ([]Warning, error)` to metrics service
  - [ ] 4.2: Check p95 latency > 500ms → warning
  - [ ] 4.3: Check library item count > 8,000 → PostgreSQL migration recommendation
  - [ ] 4.4: Check cache hit rate < 50% → cache configuration recommendation

- [ ] Task 5: Create Metrics API Endpoints (AC: 1, 2, 3)
  - [ ] 5.1: Create `/apps/api/internal/handlers/metrics_handler.go`
  - [ ] 5.2: `GET /api/v1/settings/metrics` → current metrics snapshot
  - [ ] 5.3: `GET /api/v1/settings/metrics/history?period=24h|7d` → historical data for charts
  - [ ] 5.4: `GET /api/v1/settings/metrics/warnings` → active warnings
  - [ ] 5.5: Write handler tests (≥70% coverage)

- [ ] Task 6: Create Metrics Dashboard UI (AC: 1, 2, 3)
  - [ ] 6.1: Create `/apps/web/src/components/settings/PerformanceDashboard.tsx`
  - [ ] 6.2: Create metric cards: latency (p50/p95), cache hit rate (%), API times, library count
  - [ ] 6.3: Create `/apps/web/src/components/settings/MetricCard.tsx` - individual metric display
  - [ ] 6.4: Implement simple trend charts (24h / 7d toggle) using canvas or lightweight chart library
  - [ ] 6.5: Create `/apps/web/src/components/settings/ScalabilityWarning.tsx` - warning banner
  - [ ] 6.6: Auto-refresh metrics every 30 seconds

- [ ] Task 7: Wire Up & Tests (AC: all)
  - [ ] 7.1: Register metrics middleware in Gin router
  - [ ] 7.2: Register metrics service and handler in `main.go`
  - [ ] 7.3: Start metrics persistence goroutine
  - [ ] 7.4: Write component tests

## Dev Notes

### Architecture Requirements

**FR65: Display performance metrics (query latency, cache hit rate)**
**FR66: Warn when approaching scalability limits**
**NFR-SC2:** System must warn when media library exceeds 8,000 items
**NFR-M12:** Performance metrics must be queryable via system dashboard
**ARCH-10:** Performance Monitoring Dashboard

### Metrics Model

```go
type MetricsSnapshot struct {
    Timestamp       time.Time          `json:"timestamp"`
    QueryLatency    LatencyMetric      `json:"queryLatency"`
    CacheHitRate    float64            `json:"cacheHitRate"`    // 0.0 - 1.0
    APIResponseTime map[string]LatencyMetric `json:"apiResponseTime"` // per endpoint
    LibraryCount    int                `json:"libraryCount"`
    Warnings        []Warning          `json:"warnings"`
}

type LatencyMetric struct {
    P50    float64 `json:"p50"`    // milliseconds
    P95    float64 `json:"p95"`    // milliseconds
    Avg    float64 `json:"avg"`    // milliseconds
    Count  int64   `json:"count"`  // number of measurements
}

type Warning struct {
    Type        string `json:"type"`        // "high_latency", "scalability_limit", "low_cache_hit"
    Severity    string `json:"severity"`    // "warning", "critical"
    Message     string `json:"message"`     // 繁體中文 message
    Suggestion  string `json:"suggestion"`  // Actionable recommendation
}
```

### Scalability Warnings

| Condition | Warning Message | Suggestion |
|---|---|---|
| p95 > 500ms | 查詢延遲偏高 (p95: Xms) | 考慮優化查詢或遷移至 PostgreSQL |
| items > 8,000 | 媒體庫已超過 8,000 項目 | 建議遷移至 PostgreSQL 以維持效能 |
| cache hit < 50% | 快取命中率偏低 (X%) | 檢查快取配置，考慮增加快取 TTL |

### Chart Library Decision

Use a lightweight chart library. Options:
- **recharts** (if already in deps) - React-native charting
- **Chart.js** with `react-chartjs-2` - lightweight, well-documented
- **Canvas-based custom** - minimal deps, most lightweight

Check existing `package.json` for already-installed chart libraries before adding new ones.

### API Response Format

```json
// GET /api/v1/settings/metrics
{
  "success": true,
  "data": {
    "timestamp": "2026-02-10T14:30:00Z",
    "queryLatency": { "p50": 12.5, "p95": 45.2, "avg": 18.3, "count": 1500 },
    "cacheHitRate": 0.85,
    "apiResponseTime": {
      "tmdb": { "p50": 150, "p95": 350, "avg": 180, "count": 200 },
      "ai": { "p50": 2500, "p95": 8000, "avg": 3200, "count": 50 }
    },
    "libraryCount": 1250,
    "warnings": []
  }
}

// GET /api/v1/settings/metrics/history?period=24h
{
  "success": true,
  "data": {
    "period": "24h",
    "dataPoints": [
      { "timestamp": "2026-02-09T14:30:00Z", "queryLatencyP95": 42.1, "cacheHitRate": 0.87 },
      { "timestamp": "2026-02-09T14:35:00Z", "queryLatencyP95": 38.5, "cacheHitRate": 0.89 }
    ]
  }
}
```

### Error Codes

- `METRICS_QUERY_FAILED` - Failed to query metrics
- `METRICS_PERIOD_INVALID` - Invalid time period

### Project Structure Notes

```
/apps/api/internal/services/
├── metrics_service.go
└── metrics_service_test.go

/apps/api/internal/middleware/
├── metrics_middleware.go
└── metrics_middleware_test.go

/apps/api/internal/handlers/
├── metrics_handler.go
└── metrics_handler_test.go

/apps/web/src/components/settings/
├── PerformanceDashboard.tsx
├── PerformanceDashboard.spec.tsx
├── MetricCard.tsx
├── MetricCard.spec.tsx
└── ScalabilityWarning.tsx
```

### Dependencies

- Existing cache module (for hit/miss tracking)
- Existing repository layer (for query latency)
- Media repository (for library count)

### References

- [Source: _bmad-output/planning-artifacts/epics.md#Story-6.11]
- [Source: _bmad-output/planning-artifacts/prd.md#FR65]
- [Source: _bmad-output/planning-artifacts/prd.md#FR66]
- [Source: _bmad-output/planning-artifacts/prd.md#NFR-SC2]
- [Source: _bmad-output/planning-artifacts/prd.md#NFR-M12]
- [Source: _bmad-output/planning-artifacts/architecture.md#ARCH-10-Performance-Monitoring]

## Dev Agent Record

### Agent Model Used

{{agent_model_name_version}}

### Debug Log References

### Completion Notes List

### File List
