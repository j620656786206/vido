# Story 6.2: Cache Management

Status: ready-for-dev

## Story

As a **system administrator**,
I want to **view and manage cached data**,
So that **I can reclaim disk space when needed**.

## Acceptance Criteria

1. **Given** the user opens Settings > Cache, **When** viewing cache information, **Then** they see breakdown by type: Image cache (X.X GB), AI parsing cache (X MB), Metadata cache (X MB), Total (X.X GB)
2. **Given** cache information is displayed, **When** the user clicks "Clear cache older than 30 days", **Then** old cache is removed and space reclaimed is shown
3. **Given** individual cache types are shown, **When** the user clicks "Clear" on a specific type, **Then** only that cache type is cleared and a confirmation is required

## Tasks / Subtasks

- [ ] Task 1: Create Cache Stats Service (AC: 1)
  - [ ] 1.1: Create `/apps/api/internal/services/cache_stats_service.go` with `CacheStatsServiceInterface`
  - [ ] 1.2: Implement `GetCacheStats(ctx) (*CacheStats, error)` - query sizes from `cache_entries`, `ai_cache`, `douban_cache`, `wikipedia_cache` tables
  - [ ] 1.3: Implement `GetImageCacheSize(ctx) (int64, error)` - calculate image cache dir size
  - [ ] 1.4: Write unit tests `cache_stats_service_test.go` (≥80% coverage)

- [ ] Task 2: Create Cache Cleanup Service (AC: 2, 3)
  - [ ] 2.1: Create `/apps/api/internal/services/cache_cleanup_service.go`
  - [ ] 2.2: Implement `ClearCacheByAge(ctx, days int) (*CleanupResult, error)` - delete entries older than N days
  - [ ] 2.3: Implement `ClearCacheByType(ctx, cacheType string) (*CleanupResult, error)` - clear specific cache type
  - [ ] 2.4: Write unit tests (≥80% coverage)

- [ ] Task 3: Create Cache Management Endpoints (AC: 1, 2, 3)
  - [ ] 3.1: Create `/apps/api/internal/handlers/cache_handler.go`
  - [ ] 3.2: `GET /api/v1/settings/cache` → returns cache stats by type
  - [ ] 3.3: `DELETE /api/v1/settings/cache` → clear all cache (with query params: `?older_than_days=30`)
  - [ ] 3.4: `DELETE /api/v1/settings/cache/:type` → clear specific cache type (image, ai, metadata, tmdb)
  - [ ] 3.5: Write handler tests (≥70% coverage)

- [ ] Task 4: Create Cache Management UI (AC: 1, 2, 3)
  - [ ] 4.1: Create `/apps/web/src/routes/settings.tsx` (or extend if exists) with tab navigation
  - [ ] 4.2: Create `/apps/web/src/components/settings/CacheManagement.tsx` - cache stats display with clear buttons
  - [ ] 4.3: Create `/apps/web/src/components/settings/CacheTypeCard.tsx` - individual cache type card
  - [ ] 4.4: Implement confirmation dialog before clearing (UX modal pattern)
  - [ ] 4.5: Show space reclaimed result after clearing

- [ ] Task 5: Create API Client & Hooks (AC: all)
  - [ ] 5.1: Create `/apps/web/src/services/settingsService.ts` - cache API client
  - [ ] 5.2: Create `/apps/web/src/hooks/useCacheStats.ts` - TanStack Query hook
  - [ ] 5.3: Create clear cache mutation with optimistic updates

- [ ] Task 6: Wire Up (AC: all)
  - [ ] 6.1: Register services and handlers in `main.go`
  - [ ] 6.2: Add settings route to TanStack Router
  - [ ] 6.3: Write component tests

## Dev Notes

### Architecture Requirements

**FR53: Manage cache** - View cache size, clear old cache
**ARCH-5: Cache Management System** - Multi-tier caching

### Existing Codebase Context

**Cache tables already exist:**
- `cache_entries` (migration 004) - general metadata cache
- `ai_cache` (migration 007) - AI parsing cache
- `douban_cache` (migration 008) - Douban metadata cache
- `wikipedia_cache` (migration 009) - Wikipedia metadata cache

**Existing cache module:** `/apps/api/internal/cache/offline_cache.go` - offline cache implementation. New cache stats service should query the database tables directly via repository pattern.

### Cache Types Mapping

| Display Name | DB Table | Cache Type Key |
|---|---|---|
| 圖片快取 | File system (images dir) | `image` |
| AI 解析快取 | `ai_cache` | `ai` |
| TMDb 中繼資料 | `cache_entries` | `metadata` |
| 豆瓣快取 | `douban_cache` | `douban` |
| 維基百科快取 | `wikipedia_cache` | `wikipedia` |

### API Response Format

```json
// GET /api/v1/settings/cache
{
  "success": true,
  "data": {
    "cacheTypes": [
      { "type": "image", "label": "圖片快取", "sizeBytes": 1073741824, "entryCount": 450 },
      { "type": "ai", "label": "AI 解析快取", "sizeBytes": 52428800, "entryCount": 120 },
      { "type": "metadata", "label": "TMDb 中繼資料", "sizeBytes": 10485760, "entryCount": 300 },
      { "type": "douban", "label": "豆瓣快取", "sizeBytes": 5242880, "entryCount": 50 },
      { "type": "wikipedia", "label": "維基百科快取", "sizeBytes": 3145728, "entryCount": 30 }
    ],
    "totalSizeBytes": 1144965172
  }
}

// DELETE /api/v1/settings/cache/:type
{
  "success": true,
  "data": {
    "type": "ai",
    "entriesRemoved": 85,
    "bytesReclaimed": 42000000
  }
}
```

### Error Codes

- `CACHE_TYPE_INVALID` - Unknown cache type provided
- `CACHE_CLEAR_FAILED` - Failed to clear cache entries

### Project Structure Notes

```
/apps/api/internal/services/
├── cache_stats_service.go
├── cache_stats_service_test.go
├── cache_cleanup_service.go
└── cache_cleanup_service_test.go

/apps/api/internal/handlers/
├── cache_handler.go
└── cache_handler_test.go

/apps/web/src/components/settings/
├── CacheManagement.tsx
├── CacheManagement.spec.tsx
├── CacheTypeCard.tsx
└── CacheTypeCard.spec.tsx
```

### Dependencies

- Story 1-1 (Repository Pattern) - database access
- Existing cache tables (migrations 004, 007-009)

### References

- [Source: _bmad-output/planning-artifacts/epics.md#Story-6.2]
- [Source: _bmad-output/planning-artifacts/prd.md#FR53]
- [Source: _bmad-output/planning-artifacts/architecture.md#ARCH-5-Cache-Management]
- [Source: project-context.md#Rule-4-Layered-Architecture]

## Dev Agent Record

### Agent Model Used

{{agent_model_name_version}}

### Debug Log References

### Completion Notes List

### File List
