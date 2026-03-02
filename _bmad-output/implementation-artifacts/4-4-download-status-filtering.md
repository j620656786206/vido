# Story 4.4: Download Status Filtering

Status: done

## Story

As a **media collector**,
I want to **filter downloads by status**,
So that **I can focus on specific download states**.

## Acceptance Criteria

1. **AC1: Filter Buttons Display**
   - Given the download list is displayed
   - When filter buttons are shown
   - Then options include: All, Downloading, Paused, Completed, Seeding
   - And each button shows the count of items

2. **AC2: Filter Application**
   - Given the user selects "Downloading" filter
   - When the filter is applied
   - Then only actively downloading torrents are shown
   - And the count updates in the filter button

3. **AC3: Real-time Filter Updates**
   - Given filters are applied
   - When the list updates (polling)
   - Then new items matching the filter appear
   - And items no longer matching disappear

4. **AC4: URL Persistence**
   - Given a filter is applied
   - When the filter state is applied
   - Then the filter is persisted in URL (e.g., `/downloads?filter=downloading`)
   - And reloading the page maintains the filter

5. **AC5: Multiple Statuses in One Filter**
   - Given the user wants to see multiple statuses
   - When they select multiple filter options (optional multiselect mode)
   - Then items matching any selected status are shown

## Tasks / Subtasks

- [x] Task 1: Extend Downloads API with Filter Support (AC: 1, 2)
  - [x] 1.1: Add `filter` query parameter to `GET /api/v1/downloads`
  - [x] 1.2: Support values: `all`, `downloading`, `paused`, `completed`, `seeding`, `error`
  - [x] 1.3: Implement filtering logic using qBittorrent's filter parameter
  - [x] 1.4: Update Swagger documentation
  - [x] 1.5: Write handler tests

- [x] Task 2: Add Count Endpoint (AC: 1)
  - [x] 2.1: Create `GET /api/v1/downloads/counts` endpoint
  - [x] 2.2: Return counts by status: `{ all: 10, downloading: 3, paused: 2, completed: 4, seeding: 1 }`
  - [x] 2.3: Add Swagger documentation
  - [x] 2.4: Write handler tests

- [x] Task 3: Create Filter Tabs Component (AC: 1, 2, 3)
  - [x] 3.1: Create `/apps/web/src/components/downloads/DownloadFilterTabs.tsx`
  - [x] 3.2: Display filter tabs with status icons
  - [x] 3.3: Show count badge on each tab
  - [x] 3.4: Highlight active filter
  - [x] 3.5: Write component tests

- [x] Task 4: Integrate Filter with URL State (AC: 4)
  - [x] 4.1: Use TanStack Router search params for filter state
  - [x] 4.2: Update URL when filter changes
  - [x] 4.3: Read filter from URL on page load
  - [x] 4.4: Ensure polling continues with filter active

- [x] Task 5: Update Downloads Page (AC: 2, 3, 4)
  - [x] 5.1: Add FilterTabs to Downloads page
  - [x] 5.2: Pass filter to API calls
  - [x] 5.3: Update query key to include filter
  - [x] 5.4: Ensure optimistic filter updates

- [x] Task 6: Create Download Count Hook (AC: 1, 3)
  - [x] 6.1: Create `/apps/web/src/hooks/useDownloadCounts.ts`
  - [x] 6.2: Poll counts at same interval as downloads
  - [x] 6.3: Share polling state with downloads

- [x] Task 7: E2E Tests (AC: all)
  - [x] 7.1: Create `/tests/e2e/download-filtering.api.spec.ts`
  - [x] 7.2: Test filter selection
  - [x] 7.3: Test URL persistence
  - [x] 7.4: Test count updates

## Dev Notes

### Architecture Requirements

**FR31: Filter downloads by status**
- Status-based filtering
- Client-side for responsiveness (optional server-side)

### Filter Values Mapping

| UI Filter | qBittorrent API Filter | Description |
|-----------|----------------------|-------------|
| All | `all` | All torrents |
| Downloading | `downloading` | Actively downloading |
| Paused | `paused` | Download or upload paused |
| Completed | `completed` | Finished downloading |
| Seeding | `seeding` | Upload only (completed + active) |
| Error | `errored` | Has errors |

### Backend Implementation

```go
// /apps/api/internal/handlers/download_handler.go

// GetDownloads godoc
// @Summary Get download list
// @Tags Downloads
// @Produce json
// @Param filter query string false "Filter by status" Enums(all, downloading, paused, completed, seeding, error)
// @Param sort query string false "Sort field" Enums(added_on, name, progress, size)
// @Param order query string false "Sort order" Enums(asc, desc)
// @Success 200 {object} response.ApiResponse{data=[]qbittorrent.Torrent}
// @Router /api/v1/downloads [get]
func (h *DownloadHandler) GetDownloads(c *gin.Context) {
    filter := c.DefaultQuery("filter", "all")
    sort := c.DefaultQuery("sort", "added_on")
    order := c.DefaultQuery("order", "desc")

    // Validate filter
    validFilters := map[string]bool{
        "all": true, "downloading": true, "paused": true,
        "completed": true, "seeding": true, "error": true,
    }
    if !validFilters[filter] {
        filter = "all"
    }

    downloads, err := h.service.GetAllDownloads(c.Request.Context(), filter, sort, order)
    if err != nil {
        ErrorResponse(c, err)
        return
    }

    SuccessResponse(c, downloads)
}

// GetDownloadCounts godoc
// @Summary Get download counts by status
// @Tags Downloads
// @Produce json
// @Success 200 {object} response.ApiResponse{data=DownloadCountsResponse}
// @Router /api/v1/downloads/counts [get]
func (h *DownloadHandler) GetDownloadCounts(c *gin.Context) {
    counts, err := h.service.GetDownloadCounts(c.Request.Context())
    if err != nil {
        ErrorResponse(c, err)
        return
    }

    SuccessResponse(c, counts)
}

type DownloadCountsResponse struct {
    All         int `json:"all"`
    Downloading int `json:"downloading"`
    Paused      int `json:"paused"`
    Completed   int `json:"completed"`
    Seeding     int `json:"seeding"`
    Error       int `json:"error"`
}
```

```go
// /apps/api/internal/services/download_service.go

func (s *DownloadService) GetAllDownloads(ctx context.Context, filter, sort, order string) ([]qbittorrent.Torrent, error) {
    config, err := s.qbService.GetConfig(ctx)
    if err != nil {
        return nil, err
    }

    if config.Host == "" {
        return nil, fmt.Errorf("qBittorrent not configured")
    }

    client := qbittorrent.NewClient(config, s.logger)

    // Map filter to qBittorrent filter
    qbFilter := mapToQBFilter(filter)

    opts := &qbittorrent.ListTorrentsOptions{
        Filter:  qbFilter,
        Sort:    qbittorrent.TorrentsSort(sort),
        Reverse: order == "desc",
    }

    return client.GetTorrents(ctx, opts)
}

func mapToQBFilter(filter string) qbittorrent.TorrentsFilter {
    switch filter {
    case "downloading":
        return qbittorrent.FilterDownloading
    case "paused":
        return qbittorrent.FilterPaused
    case "completed":
        return qbittorrent.FilterCompleted
    case "seeding":
        return qbittorrent.FilterSeeding
    case "error":
        return qbittorrent.TorrentsFilter("errored")
    default:
        return qbittorrent.FilterAll
    }
}

func (s *DownloadService) GetDownloadCounts(ctx context.Context) (*DownloadCounts, error) {
    // Get all downloads to count by status
    downloads, err := s.GetAllDownloads(ctx, "all", "", "")
    if err != nil {
        return nil, err
    }

    counts := &DownloadCounts{
        All: len(downloads),
    }

    for _, d := range downloads {
        switch d.Status {
        case qbittorrent.StatusDownloading:
            counts.Downloading++
        case qbittorrent.StatusPaused:
            counts.Paused++
        case qbittorrent.StatusCompleted:
            counts.Completed++
        case qbittorrent.StatusSeeding:
            counts.Seeding++
        case qbittorrent.StatusError:
            counts.Error++
        }
    }

    return counts, nil
}

type DownloadCounts struct {
    All         int `json:"all"`
    Downloading int `json:"downloading"`
    Paused      int `json:"paused"`
    Completed   int `json:"completed"`
    Seeding     int `json:"seeding"`
    Error       int `json:"error"`
}
```

### Frontend Implementation

```tsx
// /apps/web/src/components/downloads/DownloadFilterTabs.tsx
import { useNavigate, useSearch } from '@tanstack/react-router';

type FilterStatus = 'all' | 'downloading' | 'paused' | 'completed' | 'seeding' | 'error';

interface FilterConfig {
  value: FilterStatus;
  label: string;
  icon: React.ComponentType<{ className?: string }>;
  variant: 'default' | 'destructive' | 'secondary';
}

const filters: FilterConfig[] = [
  { value: 'all', label: '全部', icon: List, variant: 'default' },
  { value: 'downloading', label: '下載中', icon: ArrowDown, variant: 'default' },
  { value: 'paused', label: '已暫停', icon: Pause, variant: 'secondary' },
  { value: 'completed', label: '已完成', icon: Check, variant: 'default' },
  { value: 'seeding', label: '上傳中', icon: ArrowUp, variant: 'default' },
  { value: 'error', label: '錯誤', icon: AlertCircle, variant: 'destructive' },
];

interface DownloadFilterTabsProps {
  counts?: DownloadCounts;
}

export function DownloadFilterTabs({ counts }: DownloadFilterTabsProps) {
  const navigate = useNavigate();
  const { filter = 'all' } = useSearch({ from: '/downloads' });

  const handleFilterChange = (newFilter: FilterStatus) => {
    navigate({
      search: (prev) => ({ ...prev, filter: newFilter === 'all' ? undefined : newFilter }),
      replace: true,
    });
  };

  return (
    <div className="flex flex-wrap gap-2 mb-4">
      {filters.map((f) => {
        const Icon = f.icon;
        const count = counts?.[f.value] ?? 0;
        const isActive = filter === f.value || (f.value === 'all' && !filter);

        // Don't show error tab if no errors
        if (f.value === 'error' && count === 0) return null;

        return (
          <Button
            key={f.value}
            variant={isActive ? 'default' : 'outline'}
            size="sm"
            onClick={() => handleFilterChange(f.value)}
            className={cn(
              'gap-1.5',
              f.value === 'error' && count > 0 && 'border-destructive text-destructive'
            )}
          >
            <Icon className="h-4 w-4" />
            {f.label}
            <Badge
              variant={isActive ? 'secondary' : 'outline'}
              className="ml-1 h-5 min-w-[20px] text-xs"
            >
              {count}
            </Badge>
          </Button>
        );
      })}
    </div>
  );
}
```

```tsx
// /apps/web/src/hooks/useDownloadCounts.ts
import { useQuery } from '@tanstack/react-query';
import { downloadService } from '@/services/downloadService';

export function useDownloadCounts(enabled = true) {
  return useQuery({
    queryKey: ['downloads', 'counts'],
    queryFn: downloadService.getCounts,
    enabled,
    refetchInterval: 5000, // Same as downloads polling
  });
}
```

```tsx
// /apps/web/src/routes/downloads.tsx - Updated
import { createFileRoute } from '@tanstack/react-router';

interface DownloadsSearch {
  filter?: 'all' | 'downloading' | 'paused' | 'completed' | 'seeding' | 'error';
  sort?: string;
  order?: 'asc' | 'desc';
}

export const Route = createFileRoute('/downloads')({
  validateSearch: (search: Record<string, unknown>): DownloadsSearch => {
    return {
      filter: search.filter as DownloadsSearch['filter'],
      sort: (search.sort as string) || 'added_on',
      order: (search.order as 'asc' | 'desc') || 'desc',
    };
  },
  component: DownloadsPage,
});

function DownloadsPage() {
  const { filter = 'all', sort, order } = Route.useSearch();
  const [isVisible, setIsVisible] = useState(true);

  useEffect(() => {
    const handleVisibilityChange = () => {
      setIsVisible(document.visibilityState === 'visible');
    };
    document.addEventListener('visibilitychange', handleVisibilityChange);
    return () => document.removeEventListener('visibilitychange', handleVisibilityChange);
  }, []);

  const { data: downloads, isLoading } = useQuery({
    queryKey: ['downloads', filter, sort, order],
    queryFn: () => downloadService.getDownloads({ filter, sort, order }),
    refetchInterval: isVisible ? 5000 : false,
  });

  const { data: counts } = useDownloadCounts(isVisible);

  return (
    <div className="container mx-auto px-4 py-6">
      <h1 className="text-2xl font-bold mb-4">下載任務</h1>

      <DownloadFilterTabs counts={counts} />

      {isLoading ? (
        <LoadingState />
      ) : downloads?.length === 0 ? (
        <EmptyFilterState filter={filter} />
      ) : (
        <DownloadList downloads={downloads} />
      )}
    </div>
  );
}

function EmptyFilterState({ filter }: { filter: string }) {
  const messages: Record<string, string> = {
    all: '目前沒有下載任務',
    downloading: '沒有正在下載的任務',
    paused: '沒有已暫停的任務',
    completed: '沒有已完成的任務',
    seeding: '沒有正在上傳的任務',
    error: '沒有發生錯誤的任務',
  };

  return (
    <div className="flex flex-col items-center justify-center py-12 text-muted-foreground">
      <Inbox className="h-12 w-12 mb-4" />
      <p>{messages[filter] || messages.all}</p>
    </div>
  );
}
```

### API Response Format

**Download Counts:**
```
GET /api/v1/downloads/counts
```
Response:
```json
{
  "success": true,
  "data": {
    "all": 10,
    "downloading": 3,
    "paused": 2,
    "completed": 4,
    "seeding": 1,
    "error": 0
  }
}
```

**Filtered Downloads:**
```
GET /api/v1/downloads?filter=downloading
```
Response:
```json
{
  "success": true,
  "data": [
    {
      "hash": "abc123",
      "name": "Movie.mkv",
      "status": "downloading",
      "progress": 0.45
    }
  ]
}
```

### Project Structure Notes

**Backend Files to Modify:**
```
/apps/api/internal/handlers/
└── download_handler.go (add filter, counts endpoints)

/apps/api/internal/services/
└── download_service.go (add filter, counts methods)
```

**Frontend Files to Create:**
```
/apps/web/src/components/downloads/
├── DownloadFilterTabs.tsx
├── DownloadFilterTabs.spec.tsx
└── index.ts (update exports)

/apps/web/src/hooks/
└── useDownloadCounts.ts

/apps/web/src/routes/
└── downloads.tsx (update with search validation)
```

### Testing Strategy

**Backend Tests:**
1. Filter parameter validation tests
2. Filter mapping tests
3. Count calculation tests

**Frontend Tests:**
1. FilterTabs render tests
2. Active state tests
3. URL persistence tests

**E2E Tests:**
1. Filter selection and list update
2. URL reload maintains filter
3. Counts update with polling

**Coverage Targets:**
- Backend: ≥80%
- Frontend: ≥70%

### Dependencies

**Story Dependencies:**
- Story 4-2 (Download Monitoring) - Downloads API

### References

- [Source: _bmad-output/planning-artifacts/epics.md#Story-4.4]
- [Source: _bmad-output/planning-artifacts/architecture.md#FR31]
- [Source: project-context.md#Rule-5-TanStack-Query]
- [qBittorrent API - Torrent filters](https://github.com/qbittorrent/qBittorrent/wiki/WebUI-API-(qBittorrent-4.1)#get-torrent-list)

## Dev Agent Record

### Agent Model Used

Claude Opus 4.6

### Debug Log References

None - clean implementation, no debugging needed.

### Completion Notes List

- **Task 1:** Extended `GET /api/v1/downloads` with `filter` query parameter. Added `TorrentsFilter` type and constants to qbittorrent package. Updated `ListTorrentsOptions` to include `Filter` field. Updated client to pass filter to qBittorrent API. Updated service interface to accept filter parameter (3 params: filter, sort, order). Swagger docs updated. Handler and service tests updated for new signatures, plus new filter-specific tests added.

- **Task 2:** Created `GET /api/v1/downloads/counts` endpoint. `DownloadCounts` type placed in qbittorrent package for shared access between handlers and services. Counts calculated by fetching all torrents and grouping by normalized status. Handler and service tests cover success and not-configured cases.

- **Task 3:** Created `DownloadFilterTabs.tsx` with 6 filter tabs (All, Downloading, Paused, Completed, Seeding, Error). Uses text icons matching existing StatusIcon patterns. Error tab auto-hides when count is 0. ARIA roles (tablist/tab) for accessibility. 7 component tests covering rendering, counts, active state, click handling, and ARIA.

- **Task 4:** Integrated filter with TanStack Router `validateSearch`. Filter persisted in URL as `?filter=downloading`. "all" filter removes query param for clean URLs. Filter read from URL on page load.

- **Task 5:** Downloads page updated to use `DownloadFilterTabs` and pass active filter to `useDownloads`. Query key includes filter for proper cache isolation. Added `EmptyFilterState` component with filter-specific messages.

- **Task 6:** `useDownloadCounts` hook added to `useDownloads.ts` (not separate file - better code organization). Polls at same 5-second interval. Shares visibility detection via `usePageVisibility` helper. Tests added for counts hook.

- **Task 7:** Created `download-filtering.api.spec.ts` E2E test file. Tests filter parameter acceptance, all filter values, invalid filter fallback, filter + sort combination, and counts endpoint response format.

- **AC5 (Multiple Statuses):** Marked as optional in story. Not implemented as multiselect since single-filter covers core use case. Can be added as future enhancement.

### Change Log

- 2026-03-01: Initial implementation of all 7 tasks for download status filtering
- 2026-03-02: Code review fixes — removed duplicate interface (Rule 11), refactored usePageVisibility to useSyncExternalStore singleton, used FilterErrored constant, updated File List with missing dashboard files

### File List

**Backend - Modified:**
- `apps/api/internal/qbittorrent/torrent.go` — Added `TorrentsFilter` type, filter constants, `DownloadCounts` type, `Filter` field to `ListTorrentsOptions`
- `apps/api/internal/qbittorrent/client.go` — Updated `GetTorrents` to pass filter parameter to qBittorrent API
- `apps/api/internal/services/download_service.go` — Updated `DownloadServiceInterface` (added filter param, `GetDownloadCounts`), added `mapToQBFilter`, `validFilters`, `GetDownloadCounts` method
- `apps/api/internal/handlers/download_handler.go` — Imports `services.DownloadServiceInterface` (Rule 11), added filter param to `ListDownloads`, added `GetDownloadCounts` handler, registered `/counts` route
- `apps/api/internal/handlers/download_handler_test.go` — Updated mock for new signatures, added filter tests, counts tests
- `apps/api/internal/services/download_service_test.go` — Updated tests for new filter param, added `MapToQBFilter` tests, `ValidFilters` tests, `GetDownloadCounts` tests

**Frontend - Modified:**
- `apps/web/src/services/downloadService.ts` — Added `FilterStatus`, `DownloadCounts` types, `filter` param to `GetDownloadsParams`, `getDownloadCounts` method
- `apps/web/src/hooks/useDownloads.ts` — Added filter param to `useDownloads`, added `useDownloadCounts` hook, `usePageVisibility` via `useSyncExternalStore` singleton, updated `downloadKeys`
- `apps/web/src/hooks/useDownloads.spec.ts` — Updated tests for new signatures, added counts hook tests
- `apps/web/src/routes/downloads.tsx` — Added `validateSearch` for URL filter persistence, integrated `DownloadFilterTabs`, added `EmptyFilterState` component
- `apps/web/src/components/downloads/index.ts` — Added `DownloadFilterTabs` export

**Frontend - Created:**
- `apps/web/src/components/downloads/DownloadFilterTabs.tsx` — Filter tabs component with counts and ARIA
- `apps/web/src/components/downloads/DownloadFilterTabs.spec.tsx` — 7 component tests

**Frontend - Formatting only (Prettier):**
- `apps/web/src/components/dashboard/CollapsibleSection.tsx` — Prettier formatting fix
- `apps/web/src/components/dashboard/DownloadPanel.tsx` — Prettier formatting fix
- `apps/web/src/components/dashboard/RecentMediaPanel.tsx` — Prettier formatting fix

**E2E - Created:**
- `tests/e2e/download-filtering.api.spec.ts` — API E2E tests for filter and counts endpoints
