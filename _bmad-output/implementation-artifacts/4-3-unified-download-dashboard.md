# Story 4.3: Unified Download Dashboard

Status: review

## Story

As a **media collector**,
I want a **unified dashboard showing downloads and recent media**,
So that **I can see my complete workflow in one place**.

## Acceptance Criteria

1. **AC1: Dashboard Layout**
   - Given the user opens the homepage
   - When the dashboard loads
   - Then they see:
     - Left panel: qBittorrent download list (compact view)
     - Right panel: Recently added media
     - Bottom: Quick search bar

2. **AC2: Download to Media Flow**
   - Given downloads and media are displayed
   - When a download completes
   - Then the completed item moves from "Downloads" to "Recent Media" after parsing
   - And a notification indicates successful addition

3. **AC3: Disconnected State**
   - Given qBittorrent is disconnected
   - When viewing the dashboard
   - Then the download panel shows connection status
   - And other panels remain functional (NFR-R12)

4. **AC4: Responsive Layout**
   - Given the dashboard is displayed
   - When viewing on mobile
   - Then the layout stacks vertically (downloads, then media)
   - And each section is collapsible

5. **AC5: Quick Actions**
   - Given the dashboard is displayed
   - When the user hovers a download
   - Then quick action buttons appear (view details link)
   - When the user hovers recent media
   - Then quick action buttons appear (view details link)

## Tasks / Subtasks

- [x] Task 1: Create Dashboard Layout Component (AC: 1, 4)
  - [x] 1.1: Create `/apps/web/src/components/dashboard/DashboardLayout.tsx`
  - [x] 1.2: Implement responsive grid (2 columns desktop, stack mobile)
  - [x] 1.3: Use CSS Grid or Tailwind grid utilities
  - [x] 1.4: Write component tests

- [x] Task 2: Create Compact Download List Component (AC: 1, 2, 3, 5)
  - [x] 2.1: Create `/apps/web/src/components/dashboard/DownloadPanel.tsx`
  - [x] 2.2: Show compact view of downloads (name, progress bar, status)
  - [x] 2.3: Display "qBittorrent 未連線" when disconnected
  - [x] 2.4: Add "查看全部" link to full downloads page
  - [x] 2.5: Add hover quick actions
  - [x] 2.6: Write component tests

- [x] Task 3: Create Recent Media Panel (AC: 1, 2, 5)
  - [x] 3.1: Create `/apps/web/src/components/dashboard/RecentMediaPanel.tsx`
  - [x] 3.2: Show recently added media with poster thumbnails
  - [x] 3.3: Show "剛剛新增" badge for newly parsed items
  - [x] 3.4: Add hover quick actions
  - [x] 3.5: Add "查看全部" link to library page
  - [x] 3.6: Write component tests

- [x] Task 4: Create Quick Search Bar (AC: 1)
  - [x] 4.1: Create `/apps/web/src/components/dashboard/QuickSearchBar.tsx`
  - [x] 4.2: Implement search input with TMDb search
  - [x] 4.3: Show recent searches dropdown
  - [x] 4.4: Navigate to search results on submit
  - [x] 4.5: Write component tests

- [x] Task 5: Create Dashboard Page (AC: 1, 2, 3, 4)
  - [x] 5.1: Create `/apps/web/src/routes/index.tsx` (homepage)
  - [x] 5.2: Compose all dashboard components
  - [x] 5.3: Handle loading states for each panel independently
  - [x] 5.4: Handle error states for each panel independently

- [x] Task 6: Create Dashboard API Hooks (AC: 1, 2)
  - [x] 6.1: Create `/apps/web/src/hooks/useDashboardData.ts`
  - [x] 6.2: Combine downloads, recent media, and connection status
  - [x] 6.3: Use parallel queries for independent data fetching
  - [x] 6.4: Implement polling for downloads only

- [x] Task 7: Create Recent Media API Endpoint (AC: 1, 2)
  - [x] 7.1: Implement `GET /api/v1/media/recent?limit=10` handler
  - [x] 7.2: Return recently added media items
  - [x] 7.3: Include "just_added" flag for items added in last 5 minutes
  - [x] 7.4: Add Swagger documentation
  - [x] 7.5: Write handler tests

- [x] Task 8: Create Notification for New Media (AC: 2)
  - [x] 8.1: Create `/apps/web/src/components/notifications/NewMediaToast.tsx`
  - [x] 8.2: Show toast when new media added to library
  - [x] 8.3: Include media poster thumbnail and title
  - [x] 8.4: Write component tests

- [x] Task 9: E2E Tests (AC: all)
  - [x] 9.1: Create `/e2e/dashboard.spec.ts`
  - [x] 9.2: Test dashboard layout
  - [x] 9.3: Test disconnected state
  - [x] 9.4: Test mobile responsive layout
  - [x] 9.5: Test quick actions

## Dev Notes

### Architecture Requirements

**FR30: View download list in unified dashboard**
- Compact download view
- Alongside recent media

**NFR-R12: Partial functionality when disconnected**
- Each panel independent
- Show status when qBittorrent down

**UX-1: Desktop multi-column layout**
- From UX specification
- Responsive design

### Dashboard Layout (Desktop)

```
┌────────────────────────────────────────────────────────┐
│                    Header / Nav                        │
├─────────────────────────┬──────────────────────────────┤
│                         │                              │
│    Downloads Panel      │     Recent Media Panel       │
│    (400px fixed)        │     (flex grow)              │
│                         │                              │
│    ┌─────────────────┐  │  ┌───┐ ┌───┐ ┌───┐ ┌───┐   │
│    │ Download 1  85% │  │  │   │ │   │ │   │ │   │   │
│    ├─────────────────┤  │  │ P │ │ P │ │ P │ │ P │   │
│    │ Download 2  45% │  │  │ O │ │ O │ │ O │ │ O │   │
│    ├─────────────────┤  │  │ S │ │ S │ │ S │ │ S │   │
│    │ Download 3  12% │  │  │ T │ │ T │ │ T │ │ T │   │
│    └─────────────────┘  │  │ E │ │ E │ │ E │ │ E │   │
│    [查看全部下載]        │  │ R │ │ R │ │ R │ │ R │   │
│                         │  └───┘ └───┘ └───┘ └───┘   │
│                         │     [查看全部媒體庫]         │
├─────────────────────────┴──────────────────────────────┤
│                   Quick Search Bar                     │
│           [🔍 搜尋電影或影集...]                        │
└────────────────────────────────────────────────────────┘
```

### Dashboard Layout (Mobile)

```
┌─────────────────────────┐
│       Header / Nav      │
├─────────────────────────┤
│  ▼ 下載中 (3)           │
│  ┌─────────────────────┐│
│  │ Download 1     85%  ││
│  │ Download 2     45%  ││
│  └─────────────────────┘│
│  [查看全部]              │
├─────────────────────────┤
│  ▼ 最近新增 (8)         │
│  ┌───┐ ┌───┐ ┌───┐     │
│  │ P │ │ P │ │ P │     │
│  └───┘ └───┘ └───┘     │
│  [查看全部]              │
├─────────────────────────┤
│ [🔍 搜尋...]            │
└─────────────────────────┘
```

### Frontend Implementation

```tsx
// /apps/web/src/components/dashboard/DashboardLayout.tsx
interface DashboardLayoutProps {
  children: React.ReactNode;
}

export function DashboardLayout({ children }: DashboardLayoutProps) {
  return (
    <div className="container mx-auto px-4 py-6">
      <div className="grid grid-cols-1 lg:grid-cols-[400px_1fr] gap-6">
        {children}
      </div>
    </div>
  );
}
```

```tsx
// /apps/web/src/components/dashboard/DownloadPanel.tsx
interface DownloadPanelProps {
  className?: string;
}

export function DownloadPanel({ className }: DownloadPanelProps) {
  const { data: connectionStatus } = useQBittorrentStatus();
  const { data: downloads, isLoading } = useDownloads({ limit: 5 });

  const isConnected = connectionStatus?.configured && connectionStatus?.connected;

  return (
    <Card className={cn("h-fit", className)}>
      <CardHeader className="pb-2">
        <div className="flex items-center justify-between">
          <CardTitle className="text-lg flex items-center gap-2">
            <Download className="h-5 w-5" />
            下載中
            {isConnected && downloads && (
              <Badge variant="secondary">{downloads.length}</Badge>
            )}
          </CardTitle>
          <ConnectionStatusBadge connected={isConnected} />
        </div>
      </CardHeader>
      <CardContent>
        {!isConnected ? (
          <DisconnectedState />
        ) : isLoading ? (
          <LoadingState />
        ) : downloads?.length === 0 ? (
          <EmptyState message="目前沒有下載任務" />
        ) : (
          <div className="space-y-2">
            {downloads?.slice(0, 5).map((download) => (
              <CompactDownloadItem key={download.hash} download={download} />
            ))}
          </div>
        )}
      </CardContent>
      <CardFooter className="pt-0">
        <Link to="/downloads" className="text-sm text-primary hover:underline">
          查看全部下載 →
        </Link>
      </CardFooter>
    </Card>
  );
}

function DisconnectedState() {
  return (
    <div className="flex flex-col items-center py-6 text-center text-muted-foreground">
      <WifiOff className="h-8 w-8 mb-2" />
      <p>qBittorrent 未連線</p>
      <Link to="/settings/qbittorrent" className="text-sm text-primary hover:underline mt-2">
        前往設定
      </Link>
    </div>
  );
}

function CompactDownloadItem({ download }: { download: Download }) {
  return (
    <div className="group flex items-center gap-3 p-2 rounded-lg hover:bg-muted/50 transition-colors">
      <StatusIcon status={download.status} size="sm" />
      <div className="flex-1 min-w-0">
        <p className="text-sm truncate">{download.name}</p>
        <Progress value={download.progress * 100} className="h-1.5 mt-1" />
      </div>
      <div className="text-xs text-muted-foreground">
        {formatProgress(download.progress)}
      </div>
      {/* Hover action */}
      <Link
        to="/downloads"
        search={{ hash: download.hash }}
        className="opacity-0 group-hover:opacity-100 transition-opacity"
      >
        <ChevronRight className="h-4 w-4" />
      </Link>
    </div>
  );
}
```

```tsx
// /apps/web/src/components/dashboard/RecentMediaPanel.tsx
export function RecentMediaPanel() {
  const { data: recentMedia, isLoading } = useRecentMedia({ limit: 8 });

  return (
    <Card>
      <CardHeader className="pb-2">
        <div className="flex items-center justify-between">
          <CardTitle className="text-lg flex items-center gap-2">
            <Film className="h-5 w-5" />
            最近新增
          </CardTitle>
        </div>
      </CardHeader>
      <CardContent>
        {isLoading ? (
          <MediaGridSkeleton count={8} />
        ) : recentMedia?.length === 0 ? (
          <EmptyState message="媒體庫中還沒有內容" />
        ) : (
          <div className="grid grid-cols-4 gap-3">
            {recentMedia?.map((media) => (
              <MediaCard
                key={media.id}
                media={media}
                showJustAddedBadge={media.justAdded}
              />
            ))}
          </div>
        )}
      </CardContent>
      <CardFooter className="pt-0">
        <Link to="/library" className="text-sm text-primary hover:underline">
          查看全部媒體庫 →
        </Link>
      </CardFooter>
    </Card>
  );
}

interface MediaCardProps {
  media: RecentMedia;
  showJustAddedBadge?: boolean;
}

function MediaCard({ media, showJustAddedBadge }: MediaCardProps) {
  return (
    <Link to="/media/$id" params={{ id: media.id }} className="group relative">
      <div className="aspect-[2/3] rounded-lg overflow-hidden bg-muted">
        <img
          src={media.posterUrl || '/images/placeholder-poster.webp'}
          alt={media.title}
          className="w-full h-full object-cover group-hover:scale-105 transition-transform"
        />
        {showJustAddedBadge && (
          <Badge className="absolute top-1 right-1 text-[10px]" variant="default">
            剛剛新增
          </Badge>
        )}
      </div>
      <p className="mt-1 text-xs truncate">{media.title}</p>
      <p className="text-xs text-muted-foreground">{media.year}</p>
    </Link>
  );
}
```

```tsx
// /apps/web/src/components/dashboard/QuickSearchBar.tsx
export function QuickSearchBar() {
  const navigate = useNavigate();
  const [query, setQuery] = useState('');

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (query.trim()) {
      navigate({ to: '/search', search: { q: query } });
    }
  };

  return (
    <form onSubmit={handleSubmit} className="relative">
      <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
      <Input
        value={query}
        onChange={(e) => setQuery(e.target.value)}
        placeholder="搜尋電影或影集..."
        className="pl-10"
      />
    </form>
  );
}
```

### Backend - Recent Media Endpoint

```go
// /apps/api/internal/handlers/media_handler.go

// GetRecentMedia godoc
// @Summary Get recently added media
// @Tags Media
// @Produce json
// @Param limit query int false "Number of items" default(10) maximum(50)
// @Success 200 {object} response.ApiResponse{data=[]RecentMediaResponse}
// @Router /api/v1/media/recent [get]
func (h *MediaHandler) GetRecentMedia(c *gin.Context) {
    limit := 10
    if l, err := strconv.Atoi(c.Query("limit")); err == nil && l > 0 && l <= 50 {
        limit = l
    }

    media, err := h.service.GetRecentMedia(c.Request.Context(), limit)
    if err != nil {
        ErrorResponse(c, err)
        return
    }

    // Add justAdded flag for items added in last 5 minutes
    fiveMinutesAgo := time.Now().Add(-5 * time.Minute)
    responses := make([]RecentMediaResponse, len(media))
    for i, m := range media {
        responses[i] = RecentMediaResponse{
            ID:        m.ID,
            Title:     m.Title,
            Year:      m.Year,
            PosterUrl: m.PosterUrl,
            MediaType: m.MediaType,
            JustAdded: m.CreatedAt.After(fiveMinutesAgo),
            AddedAt:   m.CreatedAt,
        }
    }

    SuccessResponse(c, responses)
}

type RecentMediaResponse struct {
    ID        string    `json:"id"`
    Title     string    `json:"title"`
    Year      int       `json:"year,omitempty"`
    PosterUrl string    `json:"posterUrl,omitempty"`
    MediaType string    `json:"mediaType"` // movie, tv
    JustAdded bool      `json:"justAdded"`
    AddedAt   time.Time `json:"addedAt"`
}
```

### API Response Format

**Recent Media:**
```
GET /api/v1/media/recent?limit=8
```
Response:
```json
{
  "success": true,
  "data": [
    {
      "id": "media-123",
      "title": "電影名稱",
      "year": 2024,
      "posterUrl": "https://image.tmdb.org/...",
      "mediaType": "movie",
      "justAdded": true,
      "addedAt": "2026-02-09T10:00:00Z"
    }
  ]
}
```

### Project Structure Notes

**Frontend Files to Create:**
```
/apps/web/src/routes/
└── index.tsx (dashboard/homepage)

/apps/web/src/components/dashboard/
├── DashboardLayout.tsx
├── DashboardLayout.spec.tsx
├── DownloadPanel.tsx
├── DownloadPanel.spec.tsx
├── RecentMediaPanel.tsx
├── RecentMediaPanel.spec.tsx
├── QuickSearchBar.tsx
├── QuickSearchBar.spec.tsx
├── CompactDownloadItem.tsx
├── MediaCard.tsx
└── index.ts

/apps/web/src/hooks/
└── useDashboardData.ts

/apps/web/src/components/notifications/
├── NewMediaToast.tsx
└── NewMediaToast.spec.tsx
```

**Backend Files to Modify:**
```
/apps/api/internal/handlers/
└── media_handler.go (add GetRecentMedia)

/apps/api/internal/services/
└── media_service.go (add GetRecentMedia)
```

### Testing Strategy

**Frontend Tests:**
1. Dashboard layout responsiveness tests
2. DownloadPanel disconnected state tests
3. RecentMediaPanel empty state tests
4. QuickSearchBar submission tests

**E2E Tests:**
1. Full dashboard load
2. Navigate to downloads from panel
3. Navigate to library from panel
4. Mobile responsive behavior

**Coverage Targets:**
- Frontend components: ≥70%

### Error Handling

Each panel should handle errors independently:
- Downloads panel: Show disconnected state
- Recent media panel: Show error message, but don't break dashboard
- Search bar: Show error toast on search failure

### Dependencies

**Story Dependencies:**
- Story 4-1 (Connection Configuration) - Connection status
- Story 4-2 (Download Monitoring) - Downloads API
- Story 2-6 (Media Entity) - Media storage

### References

- [Source: _bmad-output/planning-artifacts/epics.md#Story-4.3]
- [Source: _bmad-output/planning-artifacts/architecture.md#FR30]
- [Source: _bmad-output/planning-artifacts/architecture.md#NFR-R12]
- [Source: _bmad-output/planning-artifacts/ux-design-specification.md#UX-1]
- [Source: project-context.md#Rule-5-TanStack-Query]

## Dev Agent Record

### Agent Model Used

Claude Opus 4.6

### Debug Log References

- TanStack Router Link requires full router context in tests (createMemoryHistory + createRouter + RouterProvider)
- "下載中" text collision: heading and StatusIcon labels both render same text → use `findByRole('heading')` instead of `findByText`
- vitest `--testPathPattern` flag doesn't exist → use positional argument pattern
- `models.NullString` doesn't exist → use `database/sql` `sql.NullString` directly

### Completion Notes List

- All 9 tasks implemented following red-green-refactor TDD cycle
- Frontend: 621 tests pass (57 files), zero regressions
- Backend: All packages compile and pass, zero regressions
- Prettier formatting applied to all new files
- Used pure Tailwind CSS (no shadcn/ui) per project conventions
- Used `cn()` utility from clsx + tailwind-merge
- Query key factory pattern with `mediaKeys` for consistent cache management
- Backend handler combines movie + series services, sorts by CreatedAt DESC
- justAdded flag: items added within 5 minutes marked as true

### Change Log

- 2026-02-10: All 9 tasks implemented, tests passing, status → review

### File List

**Created:**
- `apps/web/src/components/dashboard/DashboardLayout.tsx` - Responsive grid layout wrapper
- `apps/web/src/components/dashboard/DashboardLayout.spec.tsx` - 4 tests
- `apps/web/src/components/dashboard/DownloadPanel.tsx` - Compact download list with connection status
- `apps/web/src/components/dashboard/DownloadPanel.spec.tsx` - 11 tests
- `apps/web/src/components/dashboard/RecentMediaPanel.tsx` - Recent media grid with poster cards
- `apps/web/src/components/dashboard/RecentMediaPanel.spec.tsx` - 8 tests
- `apps/web/src/components/dashboard/QuickSearchBar.tsx` - Search input with navigation
- `apps/web/src/components/dashboard/QuickSearchBar.spec.tsx` - 4 tests
- `apps/web/src/components/dashboard/index.ts` - Barrel exports
- `apps/web/src/components/notifications/NewMediaToast.tsx` - Media addition toast notification
- `apps/web/src/components/notifications/NewMediaToast.spec.tsx` - 6 tests
- `apps/web/src/services/mediaService.ts` - Media API service with fetchApi pattern
- `apps/web/src/hooks/useDashboardData.ts` - Query hooks for recent media
- `apps/web/src/hooks/useDashboardData.spec.ts` - 4 tests
- `apps/api/internal/handlers/recent_media_handler.go` - GET /api/v1/media/recent handler
- `apps/api/internal/handlers/recent_media_handler_test.go` - 6 tests
- `tests/e2e/dashboard.spec.ts` - 11 E2E tests

**Modified:**
- `apps/web/src/routes/index.tsx` - Replaced NxWelcome with DashboardPage
- `apps/api/cmd/api/main.go` - Registered recentMediaHandler routes
