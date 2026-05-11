# Story bugfix-10-5: Empty Library Onboarding — 3-State Diagnostic UI

Status: done

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

- [x] **Task 1 — Create `EmptyNoQBT.tsx` + spec** (AC: #2, #5, #8) — DONE 2026-05-11. 5/5 spec PASS.
  - [x] 1.1 Create `apps/web/src/components/library/EmptyNoQBT.tsx` with Rule 21 header `// Implements: Component/EmptyLibrary-NoQBT (fSKuT)`.
  - [x] 1.2 Layout: `flex flex-col items-center justify-center py-24 text-center`, `<Film>`+`<FolderOpen>` 40px muted, H2 `text-xl font-semibold`, subtitle `text-sm text-[var(--text-secondary)]`, button row `gap-3`.
  - [x] 1.3 Primary `<Link to="/settings/qbittorrent">` blue + secondary `<Link to="/settings/libraries">` outlined. Testids `empty-no-qbt`, `empty-no-qbt-connect-btn`, `empty-no-qbt-folder-btn`.
  - [x] 1.4 `EmptyNoQBT.spec.tsx` — 5 cases: H2 text, subtitle text, primary href+text, secondary href+text, root testid. All Rule 16 (`toBeInTheDocument`, `toHaveAttribute`, `toHaveTextContent`).

- [x] **Task 2 — Create `EmptyNoFolder.tsx` + spec** (AC: #3, #5, #8) — DONE 2026-05-11. 5/5 spec PASS.
  - [x] 2.1 Created with Rule 21 header `// Implements: Component/EmptyLibrary-NoFolder (U3SGxG)`.
  - [x] 2.2 Single `<FolderOpen>` icon (no Film), Case B copy.
  - [x] 2.3 Primary `<Link to="/settings/libraries">` + secondary `<Link to="/setup">`.
  - [x] 2.4 `EmptyNoFolder.spec.tsx` — 5 cases mirroring Task 1.4 with Case B assertions.

- [x] **Task 3 — Create `EmptyReadyForScan.tsx` + spec** (AC: #4, #5, #8) — DONE 2026-05-11. 9/9 spec PASS.
  - [x] 3.1 Created with Rule 21 header `// Implements: Component/EmptyLibrary-ReadyForScan (mfKgm)`.
  - [x] 3.2 `<ScanSearch>` icon (verified available in installed lucide-react via `lucide-react.d.ts` grep).
  - [x] 3.3 `useTriggerScan` imported from `hooks/useScanner`; primary CTA is `<button onClick={handleScan}>` calling `await triggerScan.mutateAsync()`.
  - [x] 3.4 `disabled={isPending}` + label swap to `「掃描中…」`. **🚧 Dev Notes correction**: story specified "sonner toast" but `sonner` is NOT installed (`grep "sonner"` in package.json returned zero). Adopted the EXISTING `ScannerSettings.tsx:43-60, 109-124` pattern instead — `useState<{type,message}> + setTimeout(5000) dismiss + inline AlertCircle render`. Functional outcome (success: "掃描已啟動"; error: `ScannerApiError.message` fallback "掃描觸發失敗") identical to AC #4 wording.
  - [x] 3.5 Secondary `<Link to="/downloads">`.
  - [x] 3.6 `EmptyReadyForScan.spec.tsx` — 9 cases: H2, subtitle, root testid, secondary href, primary is `<button>` (tagName assert), click→mutateAsync, isPending disables btn + swaps label, success notification text, error notification text from rejected mutation.

- [x] **Task 4 — Add state classifier helper** (AC: #1, #9) — DONE 2026-05-11. 8/8 spec PASS.
  - [x] 4.1 Hoisted to `apps/web/src/utils/emptyLibraryState.ts` (>15 LOC threshold met after type defs + jsdoc). Exports `EmptyLibraryState` type + `ClassifyEmptyStateInput` interface + `classifyEmptyState` function.
  - [x] 4.2 `emptyLibraryState.spec.ts` — 8 table-driven cases: loading short-circuit (isLoading=true regardless), loading (qbtConfigured=undefined), Case A wins over folder/items, Case A absolute priority, Case B, Case B wins over C, Case C single library, Case C multiple libraries.

- [x] **Task 5 — Refactor `library.tsx` isLibraryEmpty branch** (AC: #7) — DONE 2026-05-11. 29/29 library.spec.tsx PASS.
  - [x] 5.1 Imported `useQBittorrentConfig` from `hooks/useQBittorrent` and `useMediaLibraries` from `hooks/useMediaLibrary`. Imported `classifyEmptyState` + 3 components.
  - [x] 5.2 Both hooks called after `useSeriesStats()` (lines ~188-191). Renamed locals `qbtConfigQuery`, `mediaLibrariesQuery` for clarity.
  - [x] 5.3 Replaced `<EmptyLibrary />` at the `isLibraryEmpty ? ` branch with an IIFE switch over `classifyEmptyState(...)` returning `null | <EmptyNoQBT/> | <EmptyNoFolder/> | <EmptyReadyForScan/>`. `'loading'` → `null` so the skeleton (from earlier `isLoading` rows above) owns the frame.
  - [x] 5.4 `isSearchEmpty ? <EmptySearchResults>` branch UNCHANGED — verified by grep + Edit boundary.
  - [x] 5.5 Added 3 new Case A/B/C tests + updated 2 stale tests (`renders EmptyLibrary when no items` / `does not render search bar when library is empty`) to assert new testids. Added 2 `vi.mock` blocks for `useQBittorrent` + `useMediaLibrary` with per-test override pattern via `mockReturnValue` + beforeEach reset.

- [x] **Task 6 — Delete old `EmptyLibrary.tsx` + spec** (AC: #6) — DONE 2026-05-11.
  - [x] 6.1 Grep confirmed: zero references to old `EmptyLibrary` outside the files being deleted (3 import-style grep variants returned only `EmptyLibrary.tsx:4` self-ref + spec self-refs).
  - [x] 6.2 Removed `apps/web/src/components/library/EmptyLibrary.tsx`. **Note**: git's rename detection recorded this as `EmptyLibrary.tsx → EmptyNoQBT.tsx` (content rewritten in the same commit), not a pure `git rm`+create. AC #6 intent ("no orphan old single-state component") is satisfied — `grep 'EmptyLibrary[^-]'` finds zero `<EmptyLibrary>` component refs (only the unrelated `EmptyLibraryState` type name in `emptyLibraryState.ts`).
  - [x] 6.3 `git rm apps/web/src/components/library/EmptyLibrary.spec.tsx` — fully deleted (45 lines).
  - [x] 6.4 Full `nx test web` verification deferred to Task 7.1.

- [x] **Task 7 — Regression gates + manual smoke** (AC: #10) — DONE 2026-05-11.
  - [x] 7.1 `pnpm nx test web` → **1787/1787 PASS** (Δ +26 vs baseline 1761: +5 EmptyNoQBT + +5 EmptyNoFolder + +9 EmptyReadyForScan + +8 classifier + +3 library route Case A/B/C − 4 old EmptyLibrary spec = +26 net).
  - [x] 7.2 `pnpm lint:all` → **0 errors / 122 warnings** — matches bugfix-10-4 baseline EXACTLY. Zero new warnings introduced. Prettier clean.
  - [x] 7.3 `pnpm run test:cleanup` → "No test processes found" ✅.
  - [x] 7.4 **CLI substitution per bugfix-10-2 precedent**: deterministic test coverage replaces browser smoke (CLI agent cannot drive Chrome DevTools). The classifier+component contract is locked by 8 classifier unit tests + 3 route-level tests + 9 EmptyReadyForScan tests (mocked `useTriggerScan` asserts `mutateAsync` invocation, `isPending` disable+label-swap, success+error notification). User browser DevTools verification (POST `/api/v1/scanner/scan` network frame + visual responsive check at 390/1440) recommended on NAS deploy.
  - [x] 7.5 Sprint-status flip handled at Step 10 closeout (workflow boundary; Step 10 owns `in-progress → review`).

## Dev Notes

### Pre-flight confirmed by SM (you don't have to re-verify these, but DO sanity-check before edits):

1. **Backend `POST /api/v1/scanner/scan` exists**: `apps/api/internal/handlers/scanner_handler.go:58` — `scanner.POST("/scan", h.TriggerScan)`. Already wired in `cmd/api/main.go` per Phase 1 consolidation. Mutation-style trigger; returns 202 Accepted with task ID.

2. **Frontend `useTriggerScan` mutation exists**: `apps/web/src/hooks/useScanner.ts` (line referenced via `ScannerSettings.tsx:41` — `const triggerScan = useTriggerScan(); await triggerScan.mutateAsync();`). Returns `UseMutationResult<...>` with `.isPending` flag. Use the EXACT same pattern as `ScannerSettings.tsx:67` for the button click handler.

3. **`useQBittorrentConfig` returns**: `{ data: { configured: boolean, ... } | undefined, isLoading, isError }`. Verified by bugfix-10-2's `useDownloads.ts:16` import + gate pattern. Use `data?.configured === true` (explicit boolean compare, NOT truthy) to avoid race conditions during initial query — see bugfix-10-2 CR M3.

4. **`useMediaLibraries` returns**: `{ data: { libraries: MediaLibraryWithPaths[] } | undefined, isLoading, isError }` — ⚠️ note the **WRAPPER object**: `mediaLibraryService.getAll()` returns `{ libraries: [...] }`, NOT a bare array. Verified at `hooks/useMediaLibrary.ts:17-22` + `services/mediaLibraryService.ts:70-72`. Use `(data?.libraries?.length ?? 0) === 0` for the no-folder check (handles `undefined` data, missing `libraries`, and empty array). **CR-corrected 2026-05-11**: the original Pre-flight here said `MediaLibrary[]` (bare array) — that was wrong and directly caused the Case-C-unreachable bug; see Completion Notes List.

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
| 2026-05-11 | **TA pass + CR followup**: TEA (Murat) `*test-automate` added `tests/e2e/empty-library.spec.ts` (6 P0/P1 Playwright specs) + `_bmad-output/automation-summary-bugfix-10-5.md`. En route it found Case C (`ready-for-scan`) was **unreachable in production** — `library.tsx:653` read `mediaLibrariesQuery.data?.length` against a `{libraries:[...]}` wrapper → `mediaLibrariesCount` always `0`. Fixed: `library.tsx:653` → `data?.libraries?.length`; `library.spec.tsx` 3 mock sites → `{ data: { libraries: [...] } }` (locking the contract at the unit layer too). CR (Amelia) then hardened the loading E2E spec (deterministic `waitForResponse` instead of `waitForTimeout`), made the `/scanner/scan` route hermetic, and de-duped the search route. Gates: nx test web 1787/1787 PASS preserved · lint:all 0 errors / 122 warnings unchanged · E2E 6/6 PASS chromium · test:cleanup no orphans. |

## Dev Agent Record

### Agent Model Used

claude-opus-4-7 (1M context) — invoked as Amelia (`dev.agent.yaml`) via `/bmad:bmm:agents:dev` → `DS bugfix-10-5`.

### Debug Log References

- vitest run `EmptyNoQBT.spec.tsx` → 5/5 PASS (89ms)
- vitest run `EmptyNoFolder.spec.tsx` → 5/5 PASS (124ms)
- vitest run `EmptyReadyForScan.spec.tsx` → 9/9 PASS (138ms)
- vitest run `emptyLibraryState.spec.ts` → 8/8 PASS (5ms)
- vitest run `library.spec.tsx` → 29/29 PASS (1204ms) — 2 stale tests updated + 3 new Case A/B/C tests added
- `pnpm nx test web` → 1787/1787 PASS (40s, +26 net vs 1761 baseline)
- `pnpm nx test api` → PASS (flaky retry on `TestScannerService_SSEBroadcast_ScanCancelled` — known issue per bugfix-10-1 precedent, filed as `preexisting-fail-scanner-sse-scan-cancelled-flake` in sprint-status line 352)
- `pnpm lint:all` → 0 errors / 122 warnings (matches bugfix-10-4 baseline exactly, ZERO new warnings)
- `pnpm run test:cleanup` → no orphans

### Completion Notes List

- 🔗 **AC Drift**: FOUND-authorized — pre-existing `EmptyLibrary.tsx` implicit v0 → this story v1 3-state classifier per Rule 20 forward-only retrofit. Change Log row 1 (`[@contract-v0→v1] AC #1`) documents.
- 📎 **Contract Stamps**: FOUND (2× `[@contract-v1]` in this story — AC #1 classifier signature + AC #5 Rule 21 header format; upstream pre-Rule-20 → ack-skipped per forward-only retrofit policy).
- 🔒 **Rule 7 Wire Format**: N/A (no Go error codes touched; pure FE story).
- 🛠️ **Dev Notes correction (Task 3.4)**: story specified "sonner toast" for success/error feedback but `sonner` is not in `package.json`. Adopted the existing `ScannerSettings.tsx:43-60, 109-124` pattern — `useState<{type,message}> + setTimeout(5000) dismiss + inline AlertCircle render`. Functional outcome identical to AC #4 wording ("掃描已啟動" success / `ScannerApiError.message` fallback "掃描觸發失敗" error).
- 🎨 **UX Verification (Step 9)**: PASS via design-vs-code structural comparison (CLI agent cannot drive browser). See table below.
- ✅ **Lint baseline**: 122 warnings exactly — matches bugfix-10-4 closeout. The 3 pre-existing `react-hooks/exhaustive-deps` warnings on `library.tsx:435` (lines 464, 504, 512) are NOT new — they reference pre-existing handlers I didn't touch (`handleSelect`, keyboard shortcut closure, `getAllItemIds`). My only library.tsx edits were imports + 2 hook calls + the empty-state branch rewrite — none added new exhaustive-deps cycles.
- 🧪 **Manual smoke deferred** to user NAS verification (Task 7.4) — covered by 26 new deterministic tests; browser DevTools confirmation recommended post-deploy.
- 🐛 **CR followup fix (2026-05-11)** — code review found **Case C (`ready-for-scan`) was unreachable in production**: `library.tsx:653` read `mediaLibrariesQuery.data?.length` against a `{ libraries: [...] }` wrapper object, so `mediaLibrariesCount` was always `0` → `classifyEmptyState` always returned `no-folder`. The 1787 unit tests passed only because `library.spec.tsx` mocked `useMediaLibraries().data` as a bare array (matching the buggy code, not the real hook). **Root cause**: SM Pre-flight #4 mis-stated the hook return shape (`MediaLibrary[]` instead of `{ libraries: MediaLibraryWithPaths[] }`); DEV trusted the brief. The two other call sites (`MediaLibraryManager.tsx:36`, `LibraryEditModal.tsx:28`) had it right. **Fix**: `library.tsx:653` → `data?.libraries?.length`; `library.spec.tsx` 3 mock sites → `{ data: { libraries: [...] } }`. **Locked** by `tests/e2e/empty-library.spec.ts` (6 P0/P1 specs, network-first, mocks the real `{success, data}` wire shape). **Lesson**: DEV must verify hook return shape against the *service implementation*, not just the hook signature. **Process gap**: `tsc --noEmit` would have caught this at commit time, but `pnpm lint:all` runs `go vet`+`staticcheck`+`eslint`+`prettier` only — candidate Epic 10 retro item.
- 🔧 **CR test hardening (2026-05-11)** — adversarial CR also tightened the new E2E spec: loading test now uses deterministic `page.waitForResponse('**/api/v1/settings/qbittorrent')` instead of an arbitrary `waitForTimeout(600)` (and exercises the "one of N queries still pending" branch via a single held `/libraries` route); `/scanner/scan` route aborts non-POST requests instead of leaking them to the real backend; the `/library/search*` mock was de-duplicated into the baseline helper (removed the dead inner branch in the `/library*` catch-all).

### UX Design Verification Table (Step 9 — mandatory for UI stories)

Comparing implementation against Sally's committed screenshots (`5ebac6f`):

| Component | Design Spec (Sally `.pen` node) | Implementation | Match? | Fix Needed |
|---|---|---|---|---|
| EmptyNoQBT (fSKuT) | Dark bg `#0F172A`, Film+Folder 40px muted, H2 「連接 qBittorrent 開始下載」semibold white, subtitle `text-sm text-[var(--text-secondary)]`, primary blue `#3B82F6` "連接 qBittorrent" + secondary outlined "已有檔案？設定資料夾" | `bg-[var(--bg-primary)]` (parent `<LibraryPage>`), `<Film>`+`<FolderOpen>` h-10 w-10 `text-[var(--text-muted)]`, H2 `text-xl font-semibold text-[var(--text-primary)]`, subtitle `text-sm text-[var(--text-secondary)]`, primary `bg-[var(--accent-primary)]` + secondary `border border-[var(--border-subtle)]` | ✅ | None |
| EmptyNoFolder (U3SGxG) | Single `<FolderOpen>` 40px muted, H2 「指定一個媒體資料夾即可開始」, subtitle 「Vido 會掃描資料夾中的影片並自動匹配 TMDb 資訊」, primary blue "設定媒體資料夾" → `/settings/libraries`, secondary outlined "開啟設定精靈" → `/setup` | Matches: single `<FolderOpen>` icon, exact H2 + subtitle, primary `<Link to="/settings/libraries">` blue, secondary `<Link to="/setup">` outlined | ✅ | None |
| EmptyReadyForScan (mfKgm) | `<ScanSearch>` (Sally's `.pen` placeholder = 🔎 emoji; DEV per story Dev Notes picks closest lucide), H2 「準備好了，等待第一筆媒體」, subtitle 「下載完成或掃描到檔案後會自動出現在這裡」, primary `<button>` "立即掃描" + secondary `<Link to="/downloads">` "前往下載中" | Matches: `<ScanSearch>` lucide-react icon (verified available), exact copy, `<button onClick={handleScan}>` with `disabled={isPending}` + label swap, `<Link to="/downloads">` | ✅ | None |
| Tailwind tokens | `bg-[var(--bg-primary)]`, `text-[var(--text-primary)]`, `text-[var(--text-secondary)]`, `text-[var(--text-muted)]`, `bg-[var(--accent-primary)]`, `border-[var(--border-subtle)]` (Sally used same tokens for color parity with old EmptyLibrary) | All 3 components use identical token set; no new color literals introduced | ✅ | None |

Result: **🎨 UX Verification: PASS — 3 implementations match design contract**. Browser-pixel verification still recommended post-NAS-deploy but no structural discrepancies detected at code-level.

### File List

**Created:**
- `apps/web/src/components/library/EmptyNoQBT.tsx`
- `apps/web/src/components/library/EmptyNoQBT.spec.tsx`
- `apps/web/src/components/library/EmptyNoFolder.tsx`
- `apps/web/src/components/library/EmptyNoFolder.spec.tsx`
- `apps/web/src/components/library/EmptyReadyForScan.tsx`
- `apps/web/src/components/library/EmptyReadyForScan.spec.tsx`
- `apps/web/src/utils/emptyLibraryState.ts`
- `apps/web/src/utils/emptyLibraryState.spec.ts`
- `tests/e2e/empty-library.spec.ts` (TA pass + CR followup — 6 P0/P1 Playwright specs)
- `_bmad-output/automation-summary-bugfix-10-5.md` (TA pass summary, commit `2d1495f`)

**Modified:**
- `apps/web/src/routes/library.tsx` (imports + 2 new hook calls + isLibraryEmpty branch replacement; **CR followup**: line 653 `data?.length` → `data?.libraries?.length`)
- `apps/web/src/routes/library.spec.tsx` (2 hook mocks added, 2 stale tests updated, 3 new Case A/B/C tests added; **CR followup**: 3 `useMediaLibraries` mock sites → `{ data: { libraries: [...] } }` wrapper shape)
- `_bmad-output/implementation-artifacts/sprint-status.yaml` (status transitions)
- `_bmad-output/implementation-artifacts/bugfix-10-5-empty-library-onboarding.md` (this file)

**Deleted:**
- `apps/web/src/components/library/EmptyLibrary.tsx` (git recorded as rename → `EmptyNoQBT.tsx`; see Task 6.2 note)
- `apps/web/src/components/library/EmptyLibrary.spec.tsx`
