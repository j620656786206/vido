# Story bugfix-10-5: Empty Library Onboarding — 3-State Diagnostic UI

Status: ready-for-dev

<!-- Created 2026-05-11 by SM Bob /create-story (YOLO). Sally UX delivery committed 5ebac6f. -->

## Story

**As a** Vido user whose library is currently empty (first launch, post-setup, or post-cleanup),
**I want** the empty-state UI to tell me **why** my library is empty AND show me the correct next action specific to **my** state,
**so that** I don't get trapped on a misleading "請連接 qBittorrent" CTA when qBT is already connected but my media folder hasn't been configured — the bug Alexyu identified during retro-10 walkthrough.

## Acceptance Criteria

1. **[@contract-v1] State classifier**: `library.tsx` (or a hoisted helper) MUST classify the empty-state into exactly one of three buckets, in this priority order, given the inputs (qbtConfigured: boolean | undefined, mediaLibraries: MediaLibrary[] | undefined, items: LibraryItem[]):
   - **Case A — `no-qbt`**: `qbtConfigured === false` (qBT not connected). Wins regardless of folder/items state, because user can't even start downloading.
   - **Case B — `no-folder`**: `qbtConfigured === true` AND `(mediaLibraries?.length ?? 0) === 0` (qBT OK, no media folder). Wins over case C because items can't populate without a folder to scan.
   - **Case C — `ready-for-scan`**: `qbtConfigured === true` AND `(mediaLibraries?.length ?? 0) > 0` AND `items.length === 0` AND `!isLoading` (everything wired, library genuinely empty).
   - While any of (qbtConfig, mediaLibraries, items) is still **loading**, classifier MUST return `loading` (do NOT render any empty-state — let the existing skeleton/loader own that frame).

2. **Case A renders `EmptyNoQBT`**: When classifier returns `no-qbt`, render `<EmptyNoQBT />`. Component MUST display:
   - Icons: `<Film>` + `<FolderOpen>` from lucide-react, side-by-side, ~40px each, muted color (`text-[var(--text-muted)]`).
   - H2: `「連接 qBittorrent 開始下載」`
   - Subtitle: `「Vido 會自動追蹤你的下載並建立媒體庫」`
   - Primary CTA: `「連接 qBittorrent」` → `<Link to="/settings/qbittorrent">` styled with `bg-[var(--accent-primary)]` (same blue as old EmptyLibrary primary).
   - Secondary CTA: `「已有檔案？設定資料夾」` → `<Link to="/settings/libraries">` styled with `border` (transparent fill, subtle border).
   - `data-testid="empty-no-qbt"` on the root container; `data-testid="empty-no-qbt-connect-btn"` and `data-testid="empty-no-qbt-folder-btn"` on the two CTAs.

3. **Case B renders `EmptyNoFolder`**: When classifier returns `no-folder`, render `<EmptyNoFolder />`. Component MUST display:
   - Icon: single `<FolderOpen>` from lucide-react, ~40px, muted color.
   - H2: `「指定一個媒體資料夾即可開始」`
   - Subtitle: `「Vido 會掃描資料夾中的影片並自動匹配 TMDb 資訊」`
   - Primary CTA: `「設定媒體資料夾」` → `<Link to="/settings/libraries">`.
   - Secondary CTA: `「開啟設定精靈」` → `<Link to="/setup">`.
   - `data-testid="empty-no-folder"` + CTA testids `empty-no-folder-libraries-btn`, `empty-no-folder-wizard-btn`.

4. **Case C renders `EmptyReadyForScan`**: When classifier returns `ready-for-scan`, render `<EmptyReadyForScan />`. Component MUST display:
   - Icon: `<ScanSearch>` (or `<FolderSearch>`) from lucide-react, ~40px, muted color. (Sally's .pen uses 🔎 emoji as placeholder; DEV picks the closest lucide equivalent — `ScanSearch` preferred for kinetic feel.)
   - H2: `「準備好了，等待第一筆媒體」`
   - Subtitle: `「下載完成或掃描到檔案後會自動出現在這裡」`
   - Primary CTA: `「立即掃描」` — **not** a `<Link>`; it's a `<button>` that calls `useTriggerScan().mutateAsync()`. Button MUST disable while `triggerScan.isPending === true` and show a spinner/text swap (e.g., `「掃描中…」`). On success → toast `「掃描已啟動」` (or equivalent). On error → toast with error message from `AppError.message`.
   - Secondary CTA: `「前往下載中」` → `<Link to="/downloads">`.
   - `data-testid="empty-ready-for-scan"` + `data-testid="empty-ready-for-scan-trigger-btn"` + `data-testid="empty-ready-for-scan-downloads-btn"`.

5. **[@contract-v1] Rule 21 component-to-design traceability**: Each of the 3 new component files MUST begin with the exact header comment format:
   - `apps/web/src/components/library/EmptyNoQBT.tsx`: `// Implements: Component/EmptyLibrary-NoQBT (fSKuT)`
   - `apps/web/src/components/library/EmptyNoFolder.tsx`: `// Implements: Component/EmptyLibrary-NoFolder (U3SGxG)`
   - `apps/web/src/components/library/EmptyReadyForScan.tsx`: `// Implements: Component/EmptyLibrary-ReadyForScan (mfKgm)`
   - Header MUST be on a line by itself, immediately above the component declaration (per `bugfix-10-4` precedent on `PosterCard.tsx`).

6. **Old EmptyLibrary.tsx + spec DELETED**: `apps/web/src/components/library/EmptyLibrary.tsx` and `EmptyLibrary.spec.tsx` MUST be removed from the file system (not commented out, not renamed — fully deleted). No references to `<EmptyLibrary>` (the old single-state component) may remain anywhere in `apps/web/src/`. This is the explicit "no orphans" rule per Alexyu's component-strategy decision (split into 3).

7. **library.tsx integration**: The `isLibraryEmpty` branch at `library.tsx:638-642` MUST be replaced with a state-classifier branch that:
   - Reads `useQBittorrentConfig()`, `useMediaLibraries()` (NEW imports for this story), and the existing `listQuery`/`searchResult` chain.
   - When loading state is true (any of the 3 queries pending while items === 0 in a non-search context), continues to render the existing grid/table skeleton (do NOT replace skeleton with an empty-state). The classifier MUST gate the empty-state ONLY for the `!isSearchActive && items.length === 0 && !isLoading` path.
   - Switches on classifier output and renders the appropriate `<EmptyNoQBT />` / `<EmptyNoFolder />` / `<EmptyReadyForScan />`.
   - When `isSearchActive && isEmpty`, the existing `<EmptySearchResults>` branch is UNCHANGED — this story does not touch search-empty UX.

8. **Test coverage**: New `*.spec.tsx` files for each of the 3 components, co-located. Each spec MUST cover:
   - Renders correct H2 text (`getByText` with exact match).
   - Renders both CTAs with correct `href` (or `onClick`) and exact text content.
   - Uses Rule 16 specific matchers (`toBeInTheDocument`, not `toBeTruthy`).
   - For `EmptyReadyForScan.spec.tsx` ONLY: mock `useTriggerScan` (`vi.mock` with `Partial<ReturnType<typeof useTriggerScan>>` typed pattern per bugfix-10-2 CR M3) and assert (a) click triggers `mutateAsync`, (b) button disables when `isPending`, (c) success/error toast paths.
   - Additionally, `library.spec.tsx` (existing) MUST gain at least 3 new test cases — one per state — asserting the classifier picks the right component given mocked hook returns.

9. **AC Drift / Rule 20**: This story introduces NEW `[@contract-v1]` stamps on AC #1 (state classifier signature) and AC #5 (Rule 21 component header format). No upstream stamped ACs to acknowledge (the pre-existing `EmptyLibrary.tsx` is pre-Rule-20 → implicit `v0`, forward-only retrofit per Rule 20). Change Log MUST record: `| 2026-05-11 | [@contract-v0→v1] AC #1: single-state empty UI → 3-state classifier (no-qbt/no-folder/ready-for-scan), downstream callers that import EmptyLibrary will fail to resolve — replace with state-classifier branch in library.tsx | bugfix-10-5 |`.

10. **Regression gates** (Definition of Done):
    - `pnpm nx test web` PASS (existing 1761 tests + new tests for 3 components + new classifier tests in library.spec.tsx).
    - `pnpm lint:all` matches baseline (0 errors / 122 warnings as of bugfix-10-4 closeout — see sprint-status line 490). ZERO new warnings.
    - `pnpm run test:cleanup` verified — no orphaned vitest workers (Epic 9c retro lesson).
    - Manual smoke (Task 7): start `pnpm nx serve web` against a backend with **(a)** qBT disconnected → verify Case A renders + CTAs route correctly; **(b)** qBT connected + zero libraries → verify Case B; **(c)** qBT connected + 1+ libraries + zero items → verify Case C + scan trigger fires (check network tab POST `/api/v1/scanner/scan`).

## Tasks / Subtasks

- [ ] **Task 1 — Create `EmptyNoQBT.tsx` + spec** (AC: #2, #5, #8)
  - [ ] 1.1 Create `apps/web/src/components/library/EmptyNoQBT.tsx` with Rule 21 header `// Implements: Component/EmptyLibrary-NoQBT (fSKuT)`.
  - [ ] 1.2 Implement layout matching Sally's screenshot at `_bmad-output/screenshots/flow-a-browse-desktop/09a-1-empty-library-no-qbt.png` — `flex flex-col items-center justify-center py-24 text-center` container, icons row (Film + FolderOpen, gap-3), H2, subtitle, button row (gap-3).
  - [ ] 1.3 Wire `<Link to="/settings/qbittorrent">` primary + `<Link to="/settings/libraries">` secondary with the testids from AC #2.
  - [ ] 1.4 Create `EmptyNoQBT.spec.tsx` co-located, covering H2 text, both CTA hrefs + text content, Rule 16 matchers.

- [ ] **Task 2 — Create `EmptyNoFolder.tsx` + spec** (AC: #3, #5, #8)
  - [ ] 2.1 Create `apps/web/src/components/library/EmptyNoFolder.tsx` with Rule 21 header `// Implements: Component/EmptyLibrary-NoFolder (U3SGxG)`.
  - [ ] 2.2 Same layout pattern as Task 1, but single `<FolderOpen>` icon (no Film) and Case B copy.
  - [ ] 2.3 Wire `<Link to="/settings/libraries">` primary + `<Link to="/setup">` secondary.
  - [ ] 2.4 Create `EmptyNoFolder.spec.tsx` mirroring Task 1.4 structure with Case B assertions.

- [ ] **Task 3 — Create `EmptyReadyForScan.tsx` + spec** (AC: #4, #5, #8)
  - [ ] 3.1 Create `apps/web/src/components/library/EmptyReadyForScan.tsx` with Rule 21 header `// Implements: Component/EmptyLibrary-ReadyForScan (mfKgm)`.
  - [ ] 3.2 Same layout pattern, with `<ScanSearch>` (or `<FolderSearch>` if `ScanSearch` is not in installed lucide-react version — check `package.json`) icon and Case C copy.
  - [ ] 3.3 Import `useTriggerScan` from `hooks/useScanner` and wire it to the primary `<button>` via `await triggerScan.mutateAsync()` — DO NOT use `<Link>` for the primary CTA (it's an action, not navigation).
  - [ ] 3.4 Add `disabled={triggerScan.isPending}` + loading text swap on the button. Wire toast on success/error (reuse existing `sonner` toast pattern — check `apps/web/src/main.tsx` for `<Toaster />`).
  - [ ] 3.5 Wire `<Link to="/downloads">` secondary CTA.
  - [ ] 3.6 Create `EmptyReadyForScan.spec.tsx` with the deeper coverage from AC #8 (mock `useTriggerScan`, assert click→mutate, isPending disable, success+error paths).

- [ ] **Task 4 — Add state classifier helper** (AC: #1, #9)
  - [ ] 4.1 Add a pure helper function `classifyEmptyState({ qbtConfigured, mediaLibrariesCount, itemsCount, isLoading })` returning `'loading' | 'no-qbt' | 'no-folder' | 'ready-for-scan'`. Place it either inline at the top of `library.tsx` (preferred, since it's the only caller) OR at `apps/web/src/utils/emptyLibraryState.ts` (only if DEV judges it worth co-testing in isolation). Decision criterion: if the helper >15 lines, hoist; else inline.
  - [ ] 4.2 Add unit tests for the classifier (table-driven, ≥7 cases covering all 4 returns including the loading short-circuit and priority order Case A > B > C). If hoisted, put tests in `emptyLibraryState.spec.ts`; if inline, add `describe('classifyEmptyState')` to `library.spec.tsx`.

- [ ] **Task 5 — Refactor `library.tsx` isLibraryEmpty branch** (AC: #7)
  - [ ] 5.1 Import `useQBittorrentConfig` from `hooks/useQBittorrent` and `useMediaLibraries` from `hooks/useMediaLibrary`.
  - [ ] 5.2 Call both hooks at the top of `LibraryPage()` (near the existing `useLibraryList`, `useLibraryStats` calls around lines 186-208). Capture `data?.configured`, `data` (library array).
  - [ ] 5.3 At line 638-642 (or wherever the `isLibraryEmpty` branch lives post-refactor), replace `<EmptyLibrary />` with `switch (classifyEmptyState({...}))` rendering the 3 components. `loading` branch returns null (skeleton owns it).
  - [ ] 5.4 Verify the `isSearchActive && isEmpty` path (line 638 `isSearchEmpty ? <EmptySearchResults>`) is UNTOUCHED.
  - [ ] 5.5 Add 3+ new tests to `library.spec.tsx` (or co-located route spec) for the 3-state branch coverage. Mock `useQBittorrentConfig`, `useMediaLibraries`, `useLibraryList` per state combo.

- [ ] **Task 6 — Delete old `EmptyLibrary.tsx` + spec** (AC: #6)
  - [ ] 6.1 Verify no remaining imports of `EmptyLibrary` (the old single-state component) anywhere in `apps/web/src/`. Use `grep -rn "EmptyLibrary[^N]" apps/web/src/` — match should be ZERO after Task 5.
  - [ ] 6.2 `git rm apps/web/src/components/library/EmptyLibrary.tsx`.
  - [ ] 6.3 `git rm apps/web/src/components/library/EmptyLibrary.spec.tsx`.
  - [ ] 6.4 Confirm `pnpm nx test web` still passes (deleted spec's 4 tests are SUPERSEDED by the new 3 component specs + classifier tests).

- [ ] **Task 7 — Regression gates + manual smoke** (AC: #10)
  - [ ] 7.1 Run `pnpm nx test web` → expect green, count delta within ±10 of baseline 1761.
  - [ ] 7.2 Run `pnpm lint:all` → expect 0 errors / 122 warnings (match bugfix-10-4 baseline). If ANY new warning appears, fix it (do NOT push baseline upward — Rule 12 / Agreement 2 left-shift).
  - [ ] 7.3 Run `pnpm run test:cleanup` → expect no orphaned processes (Epic 9c retro AI-2 rule).
  - [ ] 7.4 Manual smoke matrix:
    - **Case A**: stop backend with VIDO_DISABLE_QBT or wipe qBT config in DB → start `pnpm nx serve web` → navigate `/library` → verify EmptyNoQBT renders, both CTAs route correctly.
    - **Case B**: qBT connected but `media_libraries` table empty → verify EmptyNoFolder renders, both CTAs route.
    - **Case C**: qBT connected, library exists, items empty → verify EmptyReadyForScan renders, click `「立即掃描」` and check network tab for `POST /api/v1/scanner/scan` 200 response + success toast.
  - [ ] 7.5 Update sprint-status.yaml entry: `backlog → ready-for-dev → in-progress → review` (DEV will handle subsequent transitions per workflow).

## Dev Notes

### Pre-flight confirmed by SM (you don't have to re-verify these, but DO sanity-check before edits):

1. **Backend `POST /api/v1/scanner/scan` exists**: `apps/api/internal/handlers/scanner_handler.go:58` — `scanner.POST("/scan", h.TriggerScan)`. Already wired in `cmd/api/main.go` per Phase 1 consolidation. Mutation-style trigger; returns 202 Accepted with task ID.

2. **Frontend `useTriggerScan` mutation exists**: `apps/web/src/hooks/useScanner.ts` (line referenced via `ScannerSettings.tsx:41` — `const triggerScan = useTriggerScan(); await triggerScan.mutateAsync();`). Returns `UseMutationResult<...>` with `.isPending` flag. Use the EXACT same pattern as `ScannerSettings.tsx:67` for the button click handler.

3. **`useQBittorrentConfig` returns**: `{ data: { configured: boolean, ... } | undefined, isLoading, isError }`. Verified by bugfix-10-2's `useDownloads.ts:16` import + gate pattern. Use `data?.configured === true` (explicit boolean compare, NOT truthy) to avoid race conditions during initial query — see bugfix-10-2 CR M3.

4. **`useMediaLibraries` returns**: `{ data: MediaLibrary[] | undefined, isLoading, isError }`. Verified at `hooks/useMediaLibrary.ts:17-22`. Use `(data?.length ?? 0) === 0` for explicit no-folder check (handles both `undefined` and empty array).

5. **lucide-react `ScanSearch` icon**: If `ScanSearch` is not in the installed lucide-react version, fall back to `<FolderSearch>` (definitely available). Verify via `import { ScanSearch } from 'lucide-react'` — if TypeScript errors, switch to `FolderSearch`.

6. **Toast library**: Vido uses `sonner`. Pattern: `import { toast } from 'sonner'; toast.success('掃描已啟動'); toast.error(error.message);`. Check `apps/web/src/main.tsx` for the `<Toaster />` mount.

7. **No backend changes required**: ZERO Go file edits. ZERO migrations. ZERO swagger updates. This is a pure-frontend story.

### Sally's .pen Reusable Components (Rule 21 anchors)

Sally committed 3 Reusable Components in `ux-design.pen` (commit `5ebac6f`, 2026-05-11):

| Component (.pen) | Node ID | DEV file | Rule 21 header |
|---|---|---|---|
| `Component/EmptyLibrary-NoQBT` | `fSKuT` | `EmptyNoQBT.tsx` | `// Implements: Component/EmptyLibrary-NoQBT (fSKuT)` |
| `Component/EmptyLibrary-NoFolder` | `U3SGxG` | `EmptyNoFolder.tsx` | `// Implements: Component/EmptyLibrary-NoFolder (U3SGxG)` |
| `Component/EmptyLibrary-ReadyForScan` | `mfKgm` | `EmptyReadyForScan.tsx` | `// Implements: Component/EmptyLibrary-ReadyForScan (mfKgm)` |

Visual reference screenshots (committed in same commit):
- `_bmad-output/screenshots/flow-a-browse-desktop/09a-1-empty-library-no-qbt.png`
- `_bmad-output/screenshots/flow-a-browse-desktop/09a-2-empty-library-no-folder.png`
- `_bmad-output/screenshots/flow-a-browse-desktop/09a-3-empty-library-ready-for-scan.png`

Sally's design uses dark bg `#0F172A`, white H2 `font-semibold text-xl`, muted subtitle `text-sm text-[var(--text-secondary)]`, primary button `bg-[var(--accent-primary)]` blue `#3B82F6`, secondary button transparent fill + `border-[var(--border-subtle)]`. DEV reuses these Tailwind tokens — DO NOT introduce new color literals.

### Cross-cutting Rule compliance checklist

- **Rule 4 (Layered Architecture)**: N/A — no service layer touched (pure UI + hook composition).
- **Rule 5 (TanStack Query for server state)**: ✅ Use `useQBittorrentConfig`, `useMediaLibraries`, `useTriggerScan` — all already TanStack Query hooks. NO Zustand.
- **Rule 6 (Naming)**: ✅ Components `PascalCase.tsx`, hooks `useXxx`, testids `kebab-case`.
- **Rule 11 (Interface Location)**: N/A — no new interfaces.
- **Rule 12 (lint:all baseline)**: ✅ Task 7.2 enforces.
- **Rule 13 (Error Handling Completeness)**: ✅ `useTriggerScan` mutation error MUST surface via toast (Task 3.4). No swallowed errors.
- **Rule 14 (Resource Lifecycle)**: N/A — TanStack Query handles lifecycle.
- **Rule 15 (Pre-commit self-verification)**:
  - Wiring: no new handlers/services → N/A.
  - DB columns: no schema change → N/A.
  - Swagger: no API change → N/A.
  - HTTP Route ↔ Client Method sync: `POST /api/v1/scanner/scan` ↔ `useTriggerScan` mutation. SM verified BOTH sides exist via grep (`scanner_handler.go:58` server + `ScannerSettings.tsx:41` client usage). ✅
- **Rule 16 (Test Assertion Quality)**: ✅ Specs MUST use `toBeInTheDocument` (not `toBeTruthy`), `toEqual` (not `toBe` for objects).
- **Rule 18 (API Boundary Case Transformation)**: N/A — no new API service code. `mediaLibraryService` and `scannerService` already implement Rule 18.
- **Rule 19 (Package Boundaries)**: N/A — pure frontend.
- **Rule 20 (AC Contract Versioning)**: ✅ AC #1 and AC #5 stamped `[@contract-v1]`. Change Log entry per AC #9.
- **Rule 21 (Component-to-Design Node Traceability)**: ✅ All 3 new components carry the `// Implements:` header per AC #5. This is the SECOND inaugural Rule 21 enforcement after bugfix-10-4's PosterCard.tsx — DEV is strictly bound to format and node IDs.
- **Rule 22 (Epic Retro Design-Drift Audit)**: Deferred to Epic 10 retro (not in scope of this story — Rule 22 fires at retro time, not at story closeout).

### Previous story intelligence

**bugfix-10-4 (hover-preview-viewport-flip)** — Sally + Bob + Winston + Amelia + Murat Party Mode set the precedent for Rule 21 enforcement:
- `PosterCard.tsx` got the inaugural `// Implements: Component/PosterCardHover (MQbvp)` header.
- DEV (Amelia) implementation was clean — single file edit + Rule 21 compliance trivially achieved.
- CR found 7 issues (2 HIGH + 3 MED + 2 LOW + 1 bonus) — most about File List truthfulness and component conflict edge cases. None about Rule 21 itself.
- **Lesson for THIS story**: When deleting `EmptyLibrary.tsx`, DEV MUST verify File List explicitly lists it under `## Deleted` (not just `## Modified`) — bugfix-10-4 CR H1 was about File List lying. Same trap applies here.

**bugfix-10-2 (qbt-downloads-http-status-semantics)** — Established the typed mock pattern that AC #8 references:
- `Partial<ReturnType<typeof useQBittorrentConfig>>` cast pattern (vs `as any`) — Task 3.6's `useTriggerScan` mock should follow the same shape.

**bugfix-10-3 (skeleton-flicker-on-load)** — Established the `## 🧪 Known dev-mode artifacts` doc pattern. Not directly relevant to this story but: when DEV writes manual smoke notes (Task 7.4), verify in `nx run web:preview` (production build) NOT just `nx serve web` — StrictMode in dev double-mounts the components, which could mask classifier bugs. Cross-reference `project-context.md` `## 🧪 Known dev-mode artifacts` section.

### Project Structure Notes

**New files** (all under `apps/web/src/components/library/`):
- `EmptyNoQBT.tsx` + `EmptyNoQBT.spec.tsx`
- `EmptyNoFolder.tsx` + `EmptyNoFolder.spec.tsx`
- `EmptyReadyForScan.tsx` + `EmptyReadyForScan.spec.tsx`

**Possibly new** (DEV judgment per Task 4.1):
- `apps/web/src/utils/emptyLibraryState.ts` + `emptyLibraryState.spec.ts` (only if classifier >15 lines)

**Modified**:
- `apps/web/src/routes/library.tsx` (lines 186-208 imports/hook calls; lines 638-642 render branch)
- `apps/web/src/routes/library.spec.tsx` (existing — add 3+ classifier test cases per AC #8)

**Deleted**:
- `apps/web/src/components/library/EmptyLibrary.tsx`
- `apps/web/src/components/library/EmptyLibrary.spec.tsx`

**Untouched** (DO NOT TOUCH — out of scope):
- All backend code (`apps/api/**`)
- `EmptySearchResults.tsx` (different empty-state, handles search-no-results case)
- `RecentlyAdded.tsx` (only renders when `isCleanBrowse && !isEmpty`)
- Pencil `.pen` file (Sally's 3 Reusable Components are locked from bugfix-10-5 perspective)
- Existing `09a-empty-library-{desktop,mobile}.png` screenshots (kept as bug evidence)

### References

- Sprint-status delivery note: `_bmad-output/implementation-artifacts/sprint-status.yaml` line 491 (post-commit `5ebac6f`).
- Visual contracts: `_bmad-output/screenshots/flow-a-browse-desktop/09a-{1,2,3}-empty-library-*.png`.
- Project rules: `project-context.md` Rule 5, Rule 16, Rule 18, **Rule 21 (especially)**.
- Hook precedents:
  - `apps/web/src/hooks/useDownloads.ts:16` — `useQBittorrentConfig` gate pattern.
  - `apps/web/src/components/settings/ScannerSettings.tsx:41-67` — `useTriggerScan().mutateAsync()` pattern.
  - `apps/web/src/hooks/useMediaLibrary.ts:17-22` — `useMediaLibraries` shape.
- Rule 21 inaugural precedent: `apps/web/src/components/library/PosterCard.tsx` (bugfix-10-4 commit `24dc1a0`).
- AC Contract Versioning spec: `_bmad-output/implementation-artifacts/spike-10-AI5-ac-contract-versioning.md`.
- Existing buggy code under refactor: `apps/web/src/components/library/EmptyLibrary.tsx` + `library.tsx:451, 638-642`.

## Change Log

| Date | Change |
|---|---|
| 2026-05-11 | [@contract-v0→v1] AC #1: single-state empty UI → 3-state classifier (no-qbt/no-folder/ready-for-scan), downstream callers that import EmptyLibrary will fail to resolve — replace with state-classifier branch in library.tsx |
| 2026-05-11 | [@contract-v1] AC #5 (new): Rule 21 component header format `// Implements: Component/EmptyLibrary-{NoQBT,NoFolder,ReadyForScan} ({fSKuT,U3SGxG,mfKgm})` — DEV deviations from exact format break design-traceability audit |

## Dev Agent Record

### Agent Model Used

_To be filled by DEV during /dev-story._

### Debug Log References

_To be filled by DEV._

### Completion Notes List

_To be filled by DEV._

### File List

_To be filled by DEV. Expected structure (DEV verify before marking review):_

**Created:**
- `apps/web/src/components/library/EmptyNoQBT.tsx`
- `apps/web/src/components/library/EmptyNoQBT.spec.tsx`
- `apps/web/src/components/library/EmptyNoFolder.tsx`
- `apps/web/src/components/library/EmptyNoFolder.spec.tsx`
- `apps/web/src/components/library/EmptyReadyForScan.tsx`
- `apps/web/src/components/library/EmptyReadyForScan.spec.tsx`
- _(optional)_ `apps/web/src/utils/emptyLibraryState.ts` + spec

**Modified:**
- `apps/web/src/routes/library.tsx`
- `apps/web/src/routes/library.spec.tsx`

**Deleted:**
- `apps/web/src/components/library/EmptyLibrary.tsx`
- `apps/web/src/components/library/EmptyLibrary.spec.tsx`
