# Story 5.6: Media Detail Page (Full Version)

Status: ready-for-dev

## Story

As a **media collector**,
I want to **view comprehensive details about media in my library**,
So that **I can access all information including cast, trailers, and metadata source**.

## Acceptance Criteria

1. **AC1: Enhanced Detail Panel Content**
   - Given the user clicks on a library item
   - When the detail panel opens (Spotify-style slide-in)
   - Then it displays all information from Story 2.4 plus:
     - Full cast list with roles and profile photos
     - Embedded trailer (YouTube)
     - Metadata source indicator (TMDb/Douban/Wikipedia/AI/Manual)
     - File information (filename, size, quality)
     - Date added to library

2. **AC2: YouTube Trailer Embed**
   - Given trailers are available for the media
   - When the user clicks "觀看預告片" (Watch Trailer)
   - Then the YouTube video plays in an embedded player
   - And doesn't navigate away from the page
   - And uses privacy-enhanced mode (youtube-nocookie.com)

3. **AC3: Metadata Source Badge**
   - Given the metadata source is displayed
   - When viewing the source badge
   - Then it shows the source name with icon (TMDb/Douban/Wikipedia/AI/Manual)
   - When hovering over the badge
   - Then tooltip shows: "資料來源：TMDb，於 2026-01-10 取得"

4. **AC4: File Information Display**
   - Given a media item has associated file info
   - When viewing the detail panel
   - Then file info shows: filename, file size (formatted), detected quality
   - And file path is displayed (truncated with tooltip for full path)

5. **AC5: TV Show Enhanced Details**
   - Given the media is a TV series
   - When viewing details
   - Then additional info shows: number of seasons, episode count, production company
   - And season list with episode counts per season

6. **AC6: Detail Panel Context Menu**
   - Given the detail panel is open
   - When the user clicks the `...` (three-dot) icon (top-right, next to close button)
   - Then a context menu opens with the following items (Epic 5 scope):
     - Re-parse Metadata (Lucide: `RefreshCw`) — re-parse this item (FR40)
     - Export Metadata (Lucide: `Download`) — export JSON/YAML/NFO (FR40)
     - *(separator)*
     - Delete (Lucide: `Trash2`, `--error` red color) — remove from library, requires confirmation dialog
   - And the menu dismisses when clicking outside
   - And on mobile, the menu presents as a bottom sheet
   - And single-item operations reuse the same API endpoints as PosterCard context menu (Story 5.1)

## Tasks / Subtasks

- [ ] Task 1: Add Trailer Data to Backend (AC: 2)
  - [ ] 1.1: Add `videos` field to movie/series detail API response
  - [ ] 1.2: Store YouTube trailer key from TMDb API in models (or fetch on-demand)
  - [ ] 1.3: Add `GET /api/v1/library/movies/:id/videos` endpoint (proxy to TMDb)
  - [ ] 1.4: Write handler tests

- [ ] Task 2: Add File Info to Library Detail API (AC: 4)
  - [ ] 2.1: Ensure `file_path`, `file_size` returned in library item detail
  - [ ] 2.2: Add file size formatting utility (bytes → human-readable)
  - [ ] 2.3: Parse quality from filename if available (1080p, 4K, etc.)
  - [ ] 2.4: Write tests

- [ ] Task 3: Enhance MediaDetailPanel Component (AC: 1, 3, 4, 5)
  - [ ] 3.1: Extend `/apps/web/src/components/media/MediaDetailPanel.tsx`
  - [ ] 3.2: Add metadata source badge with icon + tooltip
  - [ ] 3.3: Add file info section (filename, size, quality)
  - [ ] 3.4: Add date added display
  - [ ] 3.5: Enhance TV show details with season list
  - [ ] 3.6: Write updated component tests

- [ ] Task 4: Create YouTube Trailer Component (AC: 2)
  - [ ] 4.1: Create `/apps/web/src/components/media/TrailerEmbed.tsx`
  - [ ] 4.2: Use privacy-enhanced embed: `https://www.youtube-nocookie.com/embed/{key}`
  - [ ] 4.3: Lazy load iframe (only render when "觀看預告片" clicked)
  - [ ] 4.4: Responsive aspect ratio (16:9)
  - [ ] 4.5: Write component tests

- [ ] Task 5: Create Metadata Source Badge Component (AC: 3)
  - [ ] 5.1: Create `/apps/web/src/components/media/MetadataSourceBadge.tsx`
  - [ ] 5.2: Icons per source: TMDb (blue), Douban (green), Wikipedia (gray), AI (purple), Manual (orange)
  - [ ] 5.3: Tooltip with source details and fetch date
  - [ ] 5.4: Write component tests

- [ ] Task 6: Create File Info Component (AC: 4)
  - [ ] 6.1: Create `/apps/web/src/components/media/FileInfo.tsx`
  - [ ] 6.2: Display filename (truncated), file size (formatted), quality badge
  - [ ] 6.3: Full path in tooltip on hover
  - [ ] 6.4: Write component tests

- [ ] Task 7: Add Trailer Hook (AC: 2)
  - [ ] 7.1: Add `useMediaTrailers(type, id)` hook
  - [ ] 7.2: Query key: `['library', type, id, 'videos']`
  - [ ] 7.3: Fetch on-demand (not preloaded)

- [ ] Task 8: Create DetailPanelMenu Component (AC: 6)
  - [ ] 8.1: Create `/apps/web/src/components/media/DetailPanelMenu.tsx`
  - [ ] 8.2: Add `...` (MoreHorizontal) icon button to detail panel header, next to close button
  - [ ] 8.3: Menu items with Lucide icons: RefreshCw (Re-parse), Download (Export), Trash2 (Delete)
  - [ ] 8.4: Delete uses `--error` red color, separated by divider, appears last
  - [ ] 8.5: Delete triggers confirmation dialog (reuse pattern from Story 5.7)
  - [ ] 8.6: Reuse single-item API endpoints from Story 5.1 Task 10
  - [ ] 8.7: Mobile: bottom sheet menu presentation
  - [ ] 8.8: Menu dismisses on outside click
  - [ ] 8.9: After delete, close detail panel and invalidate library query
  - [ ] 8.10: Write component tests

## Dev Notes

### Architecture Requirements

**FR39:** View media detail pages with cast info, trailers, complete metadata
**FR40:** Single-item operations via context menu (delete, re-parse, export metadata)
**FR42:** Display metadata source indicators (TMDb/Douban/Wikipedia/AI/Manual)
**PRD UI Component Interaction Specs:** Detail Panel Context Menu (#3)

### Existing Code to Reuse (DO NOT Reinvent)

**Backend — Already exists:**
- `MediaDetailPanel` in `/apps/web/src/components/media/MediaDetailPanel.tsx` — EXTEND, don't recreate
- `CreditsSection` in `/apps/web/src/components/media/CreditsSection.tsx` — cast/crew already done
- `TVShowInfo` in `/apps/web/src/components/media/TVShowInfo.tsx` — TV details done
- `SidePanel` in `/apps/web/src/components/ui/SidePanel.tsx` — slide-in panel done
- `useMovieDetails`, `useTVShowDetails`, `useMovieCredits` hooks — all exist
- Models already have: `MetadataSource`, `FilePath`, `FileSize`, `Credits` JSON fields
- `getImageUrl()` in `/apps/web/src/lib/image.ts` — TMDb image URL builder

**Key: Story 2.4 built the basic detail page. This story ENHANCES it with:**
- Trailer embed (new)
- Metadata source badge (new)
- File info display (new)
- Enhanced TV show seasons detail (new)

### Metadata Source Mapping

| Source | Icon | Color | Badge Text |
|--------|------|-------|------------|
| tmdb | 🎬 | Blue (#0d253f) | TMDb |
| douban | 📗 | Green (#00b51d) | 豆瓣 |
| wikipedia | 📖 | Gray (#636466) | Wikipedia |
| ai | 🤖 | Purple (#7c3aed) | AI 解析 |
| manual | ✏️ | Orange (#f59e0b) | 手動輸入 |

### YouTube Privacy Embed Pattern

```tsx
// MUST use privacy-enhanced mode per architecture requirement
const YOUTUBE_EMBED_BASE = 'https://www.youtube-nocookie.com/embed/';

interface TrailerEmbedProps {
  videoKey: string; // YouTube video ID
  title: string;
}

export function TrailerEmbed({ videoKey, title }: TrailerEmbedProps) {
  const [showPlayer, setShowPlayer] = useState(false);

  if (!showPlayer) {
    return (
      <button onClick={() => setShowPlayer(true)} className="...">
        ▶ 觀看預告片
      </button>
    );
  }

  return (
    <div className="aspect-video w-full">
      <iframe
        src={`${YOUTUBE_EMBED_BASE}${videoKey}`}
        title={`${title} 預告片`}
        allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope"
        allowFullScreen
        className="w-full h-full rounded-lg"
      />
    </div>
  );
}
```

### File Size Formatting

```typescript
export function formatFileSize(bytes: number): string {
  if (bytes === 0) return '0 B';
  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return `${parseFloat((bytes / Math.pow(k, i)).toFixed(1))} ${sizes[i]}`;
}
```

### Project Structure Notes

```
Backend (extend):
/apps/api/internal/handlers/library_handler.go ← ADD videos endpoint

Frontend (new):
/apps/web/src/components/media/TrailerEmbed.tsx             ← NEW
/apps/web/src/components/media/TrailerEmbed.spec.tsx        ← NEW
/apps/web/src/components/media/MetadataSourceBadge.tsx      ← NEW
/apps/web/src/components/media/MetadataSourceBadge.spec.tsx ← NEW
/apps/web/src/components/media/FileInfo.tsx                 ← NEW
/apps/web/src/components/media/FileInfo.spec.tsx            ← NEW
/apps/web/src/components/media/DetailPanelMenu.tsx          ← NEW
/apps/web/src/components/media/DetailPanelMenu.spec.tsx     ← NEW

Frontend (modify):
/apps/web/src/components/media/MediaDetailPanel.tsx ← EXTEND with new sections
/apps/web/src/hooks/useLibrary.ts                   ← ADD useMediaTrailers
```

### Dependencies

- Story 2-4 (Media Detail Page) — base detail panel exists
- Story 5-1 (Media Library Grid View) — library context

### Testing Strategy

- TrailerEmbed: renders button initially, click shows iframe, uses youtube-nocookie
- MetadataSourceBadge: renders correct icon/color per source, tooltip on hover
- FileInfo: formats file size, truncates filename, shows quality badge
- MediaDetailPanel: renders all new sections when data available, hides when not

### References

- [Source: _bmad-output/planning-artifacts/epics.md#Story-5.6]
- [Source: _bmad-output/planning-artifacts/prd.md#FR39]
- [Source: _bmad-output/planning-artifacts/prd.md#FR42]
- [Source: _bmad-output/planning-artifacts/ux-design-specification.md#Slide-in-Detail-Panel]
- [Source: _bmad-output/planning-artifacts/prd.md#UI-Component-Interaction-Specifications]

## Dev Agent Record

### Agent Model Used

{{agent_model_name_version}}

### Debug Log References

### Completion Notes List

### File List
