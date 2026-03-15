# Story 5-10: UX Design Compliance — Phase 5 (Detail Page & Batch Operations)

## Origin
UX Designer (Sally) verification of Stories 5-6 and 5-7 against design screenshots.

## Status: review

---

## Issues from Story 5-6 (Media Detail Page Full Version)

### Issue 1 — 🔴 Critical: Missing "播放" and "加入清單" Buttons
- **Design**: Desktop and mobile both show a prominent green "播放" (Play) button and blue outlined "加入清單" (Add to List) button as the primary CTAs
- **Implementation**: Neither button exists. Only a "觀看預告片" (Watch Trailer) button at the bottom, which serves a different purpose
- **Screenshots**: `flow-b/04-detail-panel-movie-desktop.png`, `flow-e/05-detail-panel-mobile.png`
- **Files**: `apps/web/src/components/media/MediaDetailPanel.tsx`
- **Fix**: Add "播放" and "加入清單" buttons below the genre tags, matching design placement and styling. "播放" = green solid button, "加入清單" = blue outlined button. Wire to `onPlay` and `onAddToList` callbacks (may be no-op stubs until future stories implement playback/collections).

### Issue 2 — 🟠 Medium: Missing Cast/Actors Display
- **Design**: Shows "主演：神木隆之介、上白石萌音" beneath director info
- **Implementation**: Only displays "導演" (Director). `credits` prop is passed and contains `cast` data but it is not rendered
- **Screenshots**: `flow-b/04-detail-panel-movie-desktop.png`, `flow-b/04b-detail-panel-tv-series-desktop.png`
- **Files**: `apps/web/src/components/media/MediaDetailPanel.tsx`
- **Fix**: Add "主演" section after director, showing top 3-5 cast member names from `credits.cast`

### Issue 3 — 🟡 Low: Poster/Backdrop Overlap Layout
- **Design**: Poster image overlaps the backdrop area with negative margin positioning, creating a layered visual effect
- **Implementation**: Poster is placed in a separate row below the backdrop with no overlap
- **Screenshots**: `flow-b/04-detail-panel-movie-desktop.png`, `flow-e/05-detail-panel-mobile.png`
- **Files**: `apps/web/src/components/media/MediaDetailPanel.tsx`
- **Fix**: Apply negative top margin on the poster/info row to overlap the backdrop gradient area

### Issue 4 — 🟡 Low: Context Menu Wording Inconsistency (中英混雜)
- **Design**: "匯出中繼資料"
- **PosterCardMenu**: "匯出中繼資料" ✅
- **DetailPanelMenu**: "匯出 Metadata" ❌ (mixed Chinese/English)
- **Files**: `apps/web/src/components/media/DetailPanelMenu.tsx` line 69
- **Fix**: Change "匯出 Metadata" → "匯出中繼資料"

---

## Issues from Story 5-7 (Batch Operations)

### Issue 5 — 🟡 Low: Simplified Chinese Terminology
- **Design**: Uses Traditional Chinese throughout
- **Implementation**: `SelectionToolbar.tsx` line 48 uses "匯出**元數據**" — "元數據" is Simplified Chinese
- **Files**: `apps/web/src/components/library/SelectionToolbar.tsx`
- **Fix**: Change "匯出元數據" → "匯出中繼資料"

### Issue 6 — 🟡 Low: Mobile Batch Toolbar Not Responsive
- **Design**: Mobile batch toolbar shows a compact layout with buttons arranged for small screens
- **Implementation**: Desktop and mobile share the same `SelectionToolbar` component without responsive breakpoints
- **Files**: `apps/web/src/components/library/SelectionToolbar.tsx`
- **Fix**: Add responsive styles — stack buttons vertically or use icon-only mode on mobile (`sm:` breakpoint)

---

## Items Without Design Coverage (informational)

The following implemented components have no corresponding design screenshots. These may be intentional enhancements beyond the original design scope:

| Component | Status |
|-----------|--------|
| `MetadataSourceBadge` | No design — consider adding to design if keeping |
| `TrailerEmbed` (YouTube) | No design — consider adding to design if keeping |
| `TVShowSeasons` expanded list | No design — design only shows summary season info |
| `BatchConfirmDialog` | No design — standard UX pattern, acceptable |
| `BatchProgress` | No design — standard UX pattern, acceptable |

---

## Acceptance Criteria

- [x] AC1: "播放" and "加入清單" buttons visible on both desktop and mobile detail panel, matching design colors and placement
- [x] AC2: Cast actors displayed as "主演：actor1, actor2, ..." using credits.cast data
- [x] AC3: Poster image overlaps backdrop area with layered layout matching design
- [x] AC4: All instances of "Metadata" and "元數據" replaced with "中繼資料"
- [x] AC5: Mobile batch toolbar has responsive layout adjustments
- [x] AC6: All existing tests still pass after changes
- [ ] AC7: UX Designer verification PASS on all 7 relevant design screenshots

## Tasks/Subtasks

- [x] Task 1: Add "播放" and "加入清單" CTA buttons to MediaDetailPanel (AC1)
  - [x] 1.1: Add onPlay and onAddToList callback props to MediaDetailPanelProps
  - [x] 1.2: Add green "播放" button and blue outlined "加入清單" button after genre tags
  - [x] 1.3: Update tests for new buttons
- [x] Task 2: Add cast/actors display to MediaDetailPanel (AC2)
  - [x] 2.1: Render top cast members from credits.cast after director section
  - [x] 2.2: Update tests for cast display
- [x] Task 3: Poster/Backdrop overlap layout (AC3)
  - [x] 3.1: Apply negative top margin to poster/info row for layered effect
- [x] Task 4: Fix terminology — all "Metadata" and "元數據" → "中繼資料" (AC4)
  - [x] 4.1: Fix DetailPanelMenu.tsx "匯出 Metadata" → "匯出中繼資料"
  - [x] 4.2: Fix SelectionToolbar.tsx "匯出元數據" → "匯出中繼資料"
  - [x] 4.3: Update any affected tests (DetailPanelMenu.spec.tsx)
- [x] Task 5: Mobile batch toolbar responsive layout (AC5)
  - [x] 5.1: Add responsive breakpoints to SelectionToolbar for mobile (flex-col on small screens, icon-only buttons, hidden separator)

## File List

- `apps/web/src/components/media/MediaDetailPanel.tsx` — Modified (CTA buttons, cast display, poster overlap)
- `apps/web/src/components/media/MediaDetailPanel.spec.tsx` — Modified (new tests for CTA buttons, cast)
- `apps/web/src/components/media/DetailPanelMenu.tsx` — Modified (terminology fix)
- `apps/web/src/components/media/DetailPanelMenu.spec.tsx` — Modified (terminology fix in test)
- `apps/web/src/components/library/SelectionToolbar.tsx` — Modified (terminology fix, responsive layout)

## Change Log

- 2026-03-15: Implemented all 6 UX design compliance fixes from Sally's verification report

## Dev Agent Record

### Implementation Plan
- Task 1-3: All modify MediaDetailPanel.tsx — implemented together for efficiency
- Task 4: Simple text replacements in DetailPanelMenu.tsx and SelectionToolbar.tsx
- Task 5: Responsive Tailwind classes for SelectionToolbar

### Completion Notes
- 🎨 UX Fix: Added "播放" (emerald-600) and "加入清單" (blue-500 outlined) CTA buttons with onPlay/onAddToList callbacks
- 🎨 UX Fix: Added "主演" cast display showing top 5 actors from credits.cast, joined with Chinese separator "、"
- 🎨 UX Fix: Applied -mt-12 and relative z-10 to content area for poster/backdrop overlap
- 🎨 UX Fix: Unified all export terminology to "匯出中繼資料" across DetailPanelMenu and SelectionToolbar
- 🎨 UX Fix: SelectionToolbar now uses flex-col on mobile, icon-only buttons below sm breakpoint, hidden divider
- All 1089 existing tests pass, 4 new tests added (cast display, CTA buttons)

### Debug Log
- No issues encountered during implementation

- Story created: 2026-03-15
- Created by: Bob (SM) based on Sally (UX Designer) verification report
