# Story 5.8: Recently Added Section

Status: ready-for-dev

## Story

As a **media collector**,
I want to **see recently added media prominently**,
So that **I can quickly access my newest additions**.

## Acceptance Criteria

1. **AC1: Recently Added Section Display**
   - Given the user opens the Library page
   - When the page loads
   - Then a "最近新增" (Recently Added) section shows the newest 10-20 items
   - And items are sorted by date added (newest first)
   - And section appears above the main library grid

2. **AC2: Auto-Refresh on New Media**
   - Given new media is added to the library
   - When the library updates within 30 seconds (NFR-P9)
   - Then the new item appears at the top of "Recently Added"
   - And a subtle fade-in animation highlights the new addition

3. **AC3: See All Navigation**
   - Given the user clicks "查看全部" (See All)
   - When navigating to the full library
   - Then the sort is set to "Date Added (newest)"
   - And all items are visible

4. **AC4: New Badge**
   - Given a media item was added within the last 7 days
   - When displayed in the library
   - Then a "新增" (New) badge appears on the card
   - And the badge auto-removes after 7 days

## Tasks / Subtasks

- [ ] Task 1: Create Recently Added API Endpoint (AC: 1)
  - [ ] 1.1: Add `GET /api/v1/library/recent?limit=20` endpoint
  - [ ] 1.2: Query movies + series sorted by created_at DESC, limited to N items
  - [ ] 1.3: Return combined list with type indicator (movie/series)
  - [ ] 1.4: Write handler and service tests

- [ ] Task 2: Create Recently Added Section Component (AC: 1, 3)
  - [ ] 2.1: Create `/apps/web/src/components/library/RecentlyAdded.tsx`
  - [ ] 2.2: Horizontal scrollable row of poster cards (not full grid)
  - [ ] 2.3: Section header: "最近新增" with "查看全部 >" link
  - [ ] 2.4: Click "查看全部" navigates to `/library?sortBy=created_at&sortOrder=desc`
  - [ ] 2.5: Skeleton loading state (horizontal card placeholders)
  - [ ] 2.6: Write component tests

- [ ] Task 3: Create useRecentlyAdded Hook (AC: 1, 2)
  - [ ] 3.1: Add `useRecentlyAdded(limit)` hook
  - [ ] 3.2: Query key: `['library', 'recent', limit]`
  - [ ] 3.3: staleTime: 30s (NFR-P9: updates within 30 seconds)
  - [ ] 3.4: refetchInterval: 30_000 for auto-refresh
  - [ ] 3.5: Add `getRecentlyAdded(limit)` to libraryService.ts

- [ ] Task 4: Add New Badge to PosterCard (AC: 4)
  - [ ] 4.1: Add optional `isNew` prop to PosterCard
  - [ ] 4.2: When isNew: show "新增" badge (top-right, accent color)
  - [ ] 4.3: Calculate isNew: `Date.now() - createdAt < 7 * 24 * 60 * 60 * 1000`
  - [ ] 4.4: Write updated PosterCard tests

- [ ] Task 5: Integrate into Library Page (AC: 1, 2, 3)
  - [ ] 5.1: Add RecentlyAdded section above main grid in library.tsx
  - [ ] 5.2: Only show when not searching/filtering (clean browse mode)
  - [ ] 5.3: Animate new items with fade-in when data refreshes

## Dev Notes

### Architecture Requirements

**FR41:** View recently added media items
**NFR-P9:** Library updates reflect new items within <30 seconds

### Existing Code to Reuse (DO NOT Reinvent)

- `PosterCard` — reuse for recently added cards (add isNew badge prop)
- `PosterCardSkeleton` — loading state
- `LibraryService` — extend with GetRecentlyAdded method
- Query pattern from `useLibraryList` — same service, different params
- `getImageUrl()` for poster images

### Backend Implementation

```go
// /apps/api/internal/services/library_service.go (extend)
func (s *LibraryService) GetRecentlyAdded(ctx context.Context, limit int) (*LibraryListResult, error) {
    params := repository.ListParams{
        Page:      1,
        PageSize:  limit,
        SortBy:    "created_at",
        SortOrder: "desc",
    }
    return s.ListLibrary(ctx, params)
}
```

### Frontend Horizontal Scroll Pattern

```tsx
// /apps/web/src/components/library/RecentlyAdded.tsx
export function RecentlyAdded() {
  const { data, isLoading } = useRecentlyAdded(20);

  return (
    <section className="mb-8">
      <div className="flex items-center justify-between mb-4">
        <h2 className="text-xl font-semibold">最近新增</h2>
        <Link
          to="/library"
          search={{ sortBy: 'created_at', sortOrder: 'desc' }}
          className="text-sm text-primary hover:underline"
        >
          查看全部 &gt;
        </Link>
      </div>
      <div className="flex gap-4 overflow-x-auto pb-4 scrollbar-thin">
        {isLoading
          ? Array.from({ length: 8 }).map((_, i) => <PosterCardSkeleton key={i} />)
          : data?.items.map((item) => (
              <PosterCard
                key={item.id}
                item={item}
                isNew={isWithin7Days(item.createdAt)}
                className="flex-shrink-0 w-[180px]"
              />
            ))}
      </div>
    </section>
  );
}

function isWithin7Days(dateStr: string): boolean {
  return Date.now() - new Date(dateStr).getTime() < 7 * 24 * 60 * 60 * 1000;
}
```

### Auto-Refresh Pattern

```typescript
export function useRecentlyAdded(limit: number = 20) {
  return useQuery({
    queryKey: libraryKeys.recent(limit),
    queryFn: () => libraryService.getRecentlyAdded(limit),
    staleTime: 30 * 1000,      // 30s
    refetchInterval: 30 * 1000, // Auto-refresh every 30s (NFR-P9)
  });
}
```

### Project Structure Notes

```
Backend (extend):
/apps/api/internal/services/library_service.go  ← ADD GetRecentlyAdded
/apps/api/internal/handlers/library_handler.go   ← ADD recent endpoint

Frontend (new):
/apps/web/src/components/library/RecentlyAdded.tsx       ← NEW
/apps/web/src/components/library/RecentlyAdded.spec.tsx  ← NEW

Frontend (modify):
/apps/web/src/components/media/PosterCard.tsx    ← ADD isNew badge prop
/apps/web/src/routes/library.tsx                 ← ADD RecentlyAdded section
/apps/web/src/hooks/useLibrary.ts                ← ADD useRecentlyAdded
/apps/web/src/services/libraryService.ts         ← ADD getRecentlyAdded
```

### Dependencies

- Story 5-1 (Media Library Grid View) — library page, API, PosterCard

### Testing Strategy

- Backend: returns items sorted by created_at DESC, respects limit
- RecentlyAdded: renders cards, shows skeleton loading, "查看全部" links correctly
- PosterCard: "新增" badge shown when isNew=true, hidden when false
- Auto-refresh: hook refetches every 30 seconds

### References

- [Source: _bmad-output/planning-artifacts/epics.md#Story-5.8]
- [Source: _bmad-output/planning-artifacts/prd.md#FR41]
- [Source: _bmad-output/planning-artifacts/prd.md#NFR-P9]
- [Source: _bmad-output/planning-artifacts/ux-design-specification.md#Recently-Added-Sections]

## Dev Agent Record

### Agent Model Used

{{agent_model_name_version}}

### Debug Log References

### Completion Notes List

### File List
