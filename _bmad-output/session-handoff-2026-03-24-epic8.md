# Session Handoff: Epic 8 Subtitle Engine — 2026-03-24

## What Was Done This Session

### Stories Completed (8 of 10)
- **8-1 Assrt API Client** — DONE (CR: 7 findings, all fixed)
- **8-2 Zimuku Web Scraper** — DONE (CR: 7 findings, 6 fixed)
- **8-3 OpenSubtitles API Client** — DONE (CR: partial, retry counter + hash tests added)
- **8-4 Language Detection** — DONE (CR: partial, boundary threshold + shared chars fixed)
- **8-5 OpenCC Integration** — DONE (CR: 7 findings, all fixed)
- **8-6 Subtitle Scoring & Ranking** — DONE (CR: 7 findings, 4 fixed)
- **8-7 Auto-Download Service** — DONE (CR: 7 findings, 3 fixed)
- **8-10 Subtitle File Management** — DONE (CR: 7 findings, 5 fixed)

### Stories In Progress
- **8-8 Manual Subtitle Search UI** — Code complete (status: review), **missing UX design**
  - Backend: `subtitle_handler.go` with POST /search, /download, /preview
  - Frontend: `subtitleService.ts`, `useSubtitleSearch.ts`, `SubtitleSearchDialog.tsx`
  - 10 backend tests pass
  - **BLOCKER**: No UX design in ux-design.pen — must be created before CR

### Stories Remaining
- **8-9 Batch Subtitle Processing** — ready-for-dev
  - May also need UX design if it has UI components (trigger button + progress)

## What Needs to Happen Next

### Priority 1: UX Design (Sally)
1. Open `/bmad:bmm:agents:ux-designer` (Sally)
2. Design screens for:
   - **Flow G: Subtitle Search Dialog** (Story 8-8)
     - Search form with provider checkboxes
     - Results table with score, language, source, group, downloads
     - Preview popover
     - Download button states (idle → loading → success)
   - **Flow G: Batch Subtitle Progress** (Story 8-9)
     - Batch trigger button (in settings or library page)
     - Progress card with item count, current item, success/fail counts
3. Add screens to `ux-design.pen` via Pencil MCP
4. Export screenshots: `python3 scripts/export-pen-screenshots.py`

### Priority 2: Story 8-8 Completion
1. Run code-review with UX verification against new design screenshots
2. Fix any UX discrepancies
3. Mark done

### Priority 3: Story 8-9 Development
1. Run `/bmad:bmm:workflows:dev-story` for 8-9
2. Run code-review
3. Mark done

### Priority 4: Epic 8 Wrap-up
1. Full regression: `go test ./...` + `npx nx test web` + `npx nx e2e web-e2e`
2. Update sprint-status: epic-8 → done
3. Run `/bmad:bmm:workflows:retrospective` for Epic 8

## Sprint Status
```
epic-8: in-progress
8-1 through 8-7, 8-10: done
8-8: review (code complete, needs UX design + CR)
8-9: ready-for-dev
```

## Loop Schedule
- Job `ee5a8de5`: every 2h at :07 → `/bmad:bmm:workflows:dev-story`
- Will auto-trigger 8-9 when 8-8 is marked done
- Re-create if session expired

## Key Files Created This Session
### Backend (apps/api/internal/subtitle/)
- `providers/provider.go` — SubtitleProvider interface + shared types
- `providers/assrt.go` — Assrt API client
- `providers/zimuku.go` — Zimuku web scraper
- `providers/opensub.go` — OpenSubtitles API client
- `detector.go` — 簡繁 language detection
- `converter.go` — OpenCC s2twp integration
- `scorer.go` — Multi-factor scoring engine
- `placer.go` — Atomic file placement + BCP 47 naming
- `manager.go` — Placer + DB coordination
- `engine.go` — Pipeline orchestrator (Search → Score → Download → Convert → Place)

### Backend Handler
- `handlers/subtitle_handler.go` — POST /search, /download, /preview

### Frontend (apps/web/src/)
- `services/subtitleService.ts` — API client
- `hooks/useSubtitleSearch.ts` — TanStack Query mutations + sort
- `components/subtitle/SubtitleSearchDialog.tsx` — Search dialog UI
