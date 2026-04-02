# Story 5.11: Fallback UI Enhancement for Media Detail Page

Status: ready-for-dev

## Story

As a **Vido user**,
I want to **see a friendly, well-designed fallback interface when viewing a media item without TMDB metadata**,
So that **I understand the system status and can take action to search for metadata, instead of seeing a blank or overly technical page**.

## Acceptance Criteria

1. **AC1: Color Placeholder Poster**
   - Given a media item has no poster image
   - When the detail panel opens
   - Then a color placeholder is displayed using a hash of the filename to generate a gradient background color
   - And the first character of the title is displayed as a large centered letter (Gmail avatar style)
   - And the placeholder maintains 2:3 aspect ratio consistent with real poster cards
   - And the visual style matches the existing detail panel backdrop area

2. **AC2: Pending State (Enrichment In-Progress)**
   - Given the media item has `parseStatus === 'pending'`
   - When the detail panel opens
   - Then the color placeholder poster is shown in the backdrop area
   - And a centered spinner animation is displayed in the content area
   - And the text "正在搜尋電影資訊⋯" is shown as the primary message
   - And "系統正在比對檔案名稱與 TMDb 資料庫" as secondary description
   - And a progress bar indicator is shown below the text
   - And the filename is shown as a subtle hint at the bottom

3. **AC3: Failed/No-Match State**
   - Given the media item has `parseStatus === 'failed'` or `parseStatus === ''`
   - When the detail panel opens
   - Then the color placeholder poster is shown in the backdrop area
   - And `search-x` icon + "我們找不到這部電影的資料" is shown inline (icon + title on same row)
   - And "你可以手動搜尋，或等待系統自動比對" as secondary description
   - And a file info section is displayed with icons (see AC4)
   - And CTA buttons are displayed (see AC5)

4. **AC4: File Info Section with Icons**
   - Given the failed/no-match fallback is displayed
   - When viewing the file info section
   - Then it displays:
     - File name (Lucide: `file` icon)
     - File path (Lucide: `folder` icon)
     - File size in GB (Lucide: `hard-drive` icon)
     - Added date (Lucide: `clock-3` icon)
     - Parse status with color-coded label (Lucide: `circle-alert` icon, warning color)
   - And all labels use `JetBrains Mono` for values, `Noto Sans TC` for labels
   - And the section header reads "檔案資訊"

5. **AC5: CTA Hierarchy**
   - Given the failed/no-match fallback is displayed
   - When viewing the action buttons
   - Then "搜尋 Metadata" is displayed as Primary CTA (large, full-width, `accent-primary` blue, with search icon)
   - And "手動編輯" is displayed as Secondary action (centered text link, `accent-primary` color, no background)
   - And clicking "搜尋 Metadata" navigates to `/search?q={parsed_title}`
   - And clicking "手動編輯" opens the `MetadataEditorDialog`

6. **AC6: All Copy in Traditional Chinese**
   - All user-facing text uses Traditional Chinese (zh-TW)
   - Tone is friendly and helpful, not technical

7. **AC7: Responsive Layout (Desktop & Mobile)**
   - On desktop: fallback renders inside the 460px detail side-panel (matching existing detail panel)
   - On mobile: fallback renders inside the bottom-sheet (matching existing mobile detail pattern)
   - Both layouts follow the same visual hierarchy: poster → message → file info → CTAs

## Tasks / Subtasks

- [ ] Task 1: Create `ColorPlaceholder` component (AC: #1)
  - [ ] 1.1 Create `apps/web/src/components/media/ColorPlaceholder.tsx`
  - [ ] 1.2 Implement filename-to-color hash function (deterministic gradient from string)
  - [ ] 1.3 Render gradient background with centered initial letter
  - [ ] 1.4 Support configurable size via props (desktop vs mobile dimensions)
  - [ ] 1.5 Write unit tests in `ColorPlaceholder.spec.tsx`

- [ ] Task 2: Create `FallbackPending` sub-component (AC: #2)
  - [ ] 2.1 Create pending state UI with spinner + progress bar + messages
  - [ ] 2.2 Use `Loader2` with `animate-spin` for spinner (already used in codebase)
  - [ ] 2.3 Display filename hint at bottom
  - [ ] 2.4 Write unit test

- [ ] Task 3: Create `FallbackFailed` sub-component (AC: #3, #4, #5)
  - [ ] 3.1 Create failed state with inline icon + title row
  - [ ] 3.2 Implement file info section with icon rows (AC #4)
  - [ ] 3.3 Implement CTA section: primary button + secondary link (AC #5)
  - [ ] 3.4 Wire "搜尋 Metadata" to `/search?q={title}` using `<Link>`
  - [ ] 3.5 Wire "手動編輯" to open `MetadataEditorDialog` via existing `setIsEditorOpen`
  - [ ] 3.6 Write unit tests

- [ ] Task 4: Replace existing fallback in detail page (AC: #1-7)
  - [ ] 4.1 Refactor `apps/web/src/routes/media/$type.$id.tsx` lines 251-320
  - [ ] 4.2 Replace gray placeholder with `ColorPlaceholder` in backdrop area
  - [ ] 4.3 Replace inline pending/failed blocks with `FallbackPending`/`FallbackFailed`
  - [ ] 4.4 Ensure desktop detail panel layout preserved (460px side-panel)
  - [ ] 4.5 Ensure mobile bottom-sheet layout preserved

- [ ] Task 5: Responsive styling (AC: #7)
  - [ ] 5.1 Desktop: color poster in 460px panel backdrop (240px height)
  - [ ] 5.2 Mobile: color poster in bottom-sheet backdrop (200px height)
  - [ ] 5.3 Verify both layouts match UX design screenshots

- [ ] Task 6: E2E smoke test (AC: #1-7)
  - [ ] 6.1 Add/update Playwright test for detail page fallback states
  - [ ] 6.2 Test pending state renders spinner + message (use `toBeAttached()` not `toBeVisible()` for animations)
  - [ ] 6.3 Test failed state renders file info + CTAs
  - [ ] 6.4 Test color placeholder renders (check for gradient background)

## Dev Notes

### Architecture Compliance

- **CSS Framework**: Tailwind CSS v3.x utility classes only (no custom CSS files)
- **Component Location**: New components go in `apps/web/src/components/media/`
- **Route File**: `apps/web/src/routes/media/$type.$id.tsx` — the existing fallback block (lines 251-320) is what gets replaced
- **No API/DB changes**: This is purely frontend. All data (`parseStatus`, `filePath`, `fileSize`, `createdAt`, `title`) is already available from the existing `localData` object

### Existing Patterns to Follow

- **Detail panel structure**: The existing detail panel uses `bg-primary` (#1B2336) background, `text-primary` (#F2F2F2) for headings, `text-secondary` (#B3B3B3) for body text, `text-muted` (#808080) for labels
- **Button patterns**: Primary CTA uses `bg-blue-600 hover:bg-blue-700` (maps to `accent-primary`); secondary uses text-only link style
- **Icon usage**: All icons from `lucide-react`, imported at top of file. Existing imports include `Film`, `FileText`, `HardDrive`, `Clock`, `Search`, `Pencil`, `Loader2`
- **Font families**: `JetBrains Mono` for code/data values, `Noto Sans TC` for Chinese text — already configured in Tailwind config as `font-mono` and `font-sans`
- **MetadataEditorDialog**: Already exists and is wired with `isEditorOpen`/`setIsEditorOpen` state in the detail page — reuse directly
- **Link component**: Use TanStack Router `<Link>` (already imported) for navigation

### Color Hash Algorithm

Recommended approach for `ColorPlaceholder`:
```typescript
function filenameToGradient(filename: string): [string, string] {
  let hash = 0;
  for (let i = 0; i < filename.length; i++) {
    hash = filename.charCodeAt(i) + ((hash << 5) - hash);
  }
  const hue = Math.abs(hash) % 360;
  return [
    `hsl(${hue}, 65%, 35%)`,      // darker stop
    `hsl(${(hue + 40) % 360}, 55%, 45%)`,  // lighter stop
  ];
}
```
This produces deterministic, visually consistent gradients. The 65%/35% saturation/lightness keeps it dark enough for white text overlay on the dark theme.

### Design Reference

- **UX Design Screens**: `ux-design.pen` — Screen 4d (Desktop Failed), Screen 4e (Desktop Pending), Screen 5b (Mobile Failed), Screen 5c (Mobile Pending)
- **Screenshots**: `_bmad-output/screenshots/flow-b-hover-detail-desktop/04d-detail-fallback-failed-desktop.png` and `04e-detail-fallback-pending-desktop.png`; `flow-e-interaction-mobile/05b-detail-fallback-failed-mobile.png` and `05c-detail-fallback-pending-mobile.png`
- **Existing Detail Screen Reference**: Screen 4 (Movie Detail Desktop) at `04-detail-panel-movie-desktop.png`

### Anti-Pattern Prevention

- **DO NOT** create a new API endpoint — all data is already on `localData`
- **DO NOT** use `toBeVisible()` for spinner/animation assertions in E2E — use `toBeAttached()` per project gotcha
- **DO NOT** add new Tailwind color values — use existing theme variables (`bg-primary`, `text-primary`, etc.)
- **DO NOT** create a separate mobile component — use responsive Tailwind classes (`md:` prefix) within the same component
- **DO NOT** add a new route — this replaces content within the existing `$type.$id.tsx` route

### Previous Story Intelligence

From **Story 5-6** (Media Detail Page Full Version) and **bugfix-1** (Media Detail Route Refactor):
- Detail page was refactored from tmdbId routing to UUID routing (bugfix-1, commit `332f043`)
- `localData` object provides: `title`, `originalTitle`, `parseStatus`, `filePath`, `fileSize`, `createdAt`, `tmdbId`
- `hasMetadata` is derived from `tmdbId > 0` and determines which branch renders
- The fallback branch is the `else` clause at line 251 of `$type.$id.tsx`
- Mobile uses a different layout path (bottom-sheet vs side-panel) — check for responsive breakpoints in the existing code

### Git Intelligence

Recent commits show:
- `735c67e` — UX design for fallback screens just committed (this story's design reference)
- `545a9eb` — UUID routing fix in LibraryTable links
- `332f043` — Major detail page refactor to UUID + local API (the foundation this story builds on)

### Project Structure Notes

- Component path: `apps/web/src/components/media/ColorPlaceholder.tsx` (new)
- Route file: `apps/web/src/routes/media/$type.$id.tsx` (modify)
- Test files: `apps/web/src/components/media/ColorPlaceholder.spec.tsx` (new), update existing detail page tests
- E2E: `tests/e2e/` — update or add fallback-specific test

### References

- [Source: ux-design.pen — Screen 4d, 4e, 5b, 5c]
- [Source: _bmad-output/screenshots/flow-b-hover-detail-desktop/04d-detail-fallback-failed-desktop.png]
- [Source: apps/web/src/routes/media/$type.$id.tsx#L251-L320 — current fallback]
- [Source: project-context.md — Rule 1 (Tailwind), Rule 6 (slog/AppError), Rule 2 (Testing)]
- [Source: _bmad-output/planning-artifacts/ux-design-specification.md — color system, typography]

## Dev Agent Record

### Agent Model Used

### Debug Log References

### Completion Notes List

### File List
