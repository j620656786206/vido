# Story bugfix-10-6: ExploreBlock Polish Bundle — Scroll-Chevron Contrast, Empty State, lucide Icons

Status: ready-for-dev

<!-- Created 2026-05-11 by SM Bob /create-story (YOLO). Sally UX delivery committed 936fdb0 (Screen HP-5 spec). Bundles the 3 small UX calls that were filed separately as 10-6/10-7/10-8 before the 2026-04-20 consolidation. -->

## Story

**As a** Vido user browsing the homepage TV wall (and the admin opening Settings → 自訂首頁),
**I want** the explore-block scroll arrows to actually be visible on the dark theme, the "沒有符合條件的內容" message to not be hidden behind a scroll arrow, and the block-list to use clean lucide icons instead of 🎬/📺 emoji,
**so that** the homepage and its admin feel polished and intentional rather than "AI 感太重" — the 3 small papercuts Alexyu flagged during the retro-10 local smoke test.

## Context & Why This Is Bundled

These are 3 single-digit-line CSS/copy changes that share one UX design pass (Sally, committed `936fdb0` — Screen HP-5 in `ux-design.pen`) and should ship in one DEV commit + one code review. They were originally 10-6 / 10-7 / 10-8 in the 2026-04-20 retro-10 backlog; the SM consolidated them after Sally's pass. **This is a pure-frontend story — ZERO Go edits, ZERO migrations, ZERO swagger.** Cross-stack split check: BE = 0 tasks, FE = 6 tasks → single story (the `>3 each side` split threshold is not met).

## Acceptance Criteria

1. **[@contract-v1] ExploreBlock scroll-chevron visibility & contrast invariant** (`apps/web/src/components/homepage/ExploreBlock.tsx`, the two `<button>`s at ~L103-120):
   - The chevron buttons MUST have **guaranteed contrast against whatever is behind them** — `bg-black/70` on the near-black `$bg-primary` page (`#1B2336`) is the bug. Replace with: `bg-[var(--bg-secondary)]/95 backdrop-blur-sm ring-1 ring-[var(--border-subtle)]/70 text-[var(--text-primary)] shadow-lg` (or visually-equivalent tokens — DO NOT introduce raw color literals). The chevron glyph color flips from `text-white` to `text-[var(--text-primary)]`.
   - Each chevron MUST sit on a **left / right edge gradient scrim** so it stays legible over poster art too: a `~w-14` (`3.5rem`) overlay using `bg-gradient-to-r from-[var(--bg-primary)] to-transparent` (left side) / `bg-gradient-to-l from-[var(--bg-primary)] to-transparent` (right side), behind the chevron button, in front of the scroller. `pointer-events-none` on the scrim so it doesn't eat scroll/click.
   - **Hover-reveal**: the chevrons are `opacity-0 transition-opacity` by default and `lg:group-hover:opacity-100` — i.e. they fade in only while the user hovers the block (Netflix/Disney+ pattern). Implement by adding `group/scroller` (or just `group`) to the `relative` wrapper around the scroller; the chevron buttons use `group-hover/scroller:opacity-100` (or `group-hover:` if the unnamed group is unambiguous). ⚠️ **Cascade gotcha (bugfix-10-4 CR H2 lesson)**: ensure NO ancestor of the chevron buttons also has a `lg:group-hover:opacity-0` rule on the *same* group — a child with `opacity-0 group-hover:opacity-100` nested under a parent with `group-hover:opacity-0` cascades to effective `0` in BOTH states. Use a named group (`group/scroller`) if there's any ambiguity with other `group` usages in the subtree.
   - **Touch behaviour UNCHANGED**: keep the existing `hidden lg:block` so the chevrons never appear on touch devices (native horizontal scroll handles it there). The `data-testid="explore-block-scroll-left"` and `data-testid="explore-block-scroll-right"` MUST remain on the buttons, and the `scroll('left'|'right')` click handler MUST keep working.
   - **OPTIONAL (DEV judgment, not required)** — overflow-awareness: a side's chevron MAY additionally be hidden when that direction has no scroll room (`scrollLeft <= 0` → hide left; `scrollLeft + clientWidth >= scrollWidth` → hide right), tracked via an `onScroll` handler + `ResizeObserver` on the scroller. **IF** implemented, it MUST default to *visible* when scroll metrics are `0`/unavailable (jsdom / SSR) so the existing `ExploreBlock.spec.tsx` chevron test (`findByTestId('explore-block-scroll-left')` on a populated block) stays green WITHOUT mocking layout metrics. If this adds meaningful complexity, skip it — the contrast + hover-reveal + scrim are the contract; overflow-awareness is a nice-to-have.

2. **Scroll chevrons unchanged when the block has items** (regression guard for AC #1): when `items.length > 0`, both `explore-block-scroll-left` and `explore-block-scroll-right` MUST still be in the DOM (so the existing `ExploreBlock.spec.tsx:278-280` test passes), `hidden lg:block`, with the new styling. Clicking them MUST still call `scrollerRef.current.scrollBy(...)`.

3. **`<option>` content-type labels lose the emoji** (AC3 spill-over — native `<option>` can't render an SVG, so they go to plain text, not lucide):
   - `apps/web/src/components/settings/ExploreBlockEditModal.tsx:165-166`: `<option value="movie">🎬 電影</option>` → `<option value="movie">電影</option>`; `<option value="tv">📺 影集</option>` → `<option value="tv">影集</option>`.
   - `apps/web/src/components/settings/LibraryEditModal.tsx:132-133`: `<option value="movie">🎬 電影</option>` → `<option value="movie">電影</option>`; `<option value="series">📺 影集</option>` → `<option value="series">影集</option>`.
   - `apps/web/src/components/setup/MediaLibrarySetupStep.tsx:87-88`: same — `🎬 電影` → `電影`, `📺 影集` → `影集`.
   - The `value` attributes (`movie`/`tv`/`series`) and `data-testid`s are UNCHANGED.

4. **`[@contract-v1]` Settings → 自訂首頁 block-row uses lucide icons, not emoji** (`apps/web/src/components/settings/ExploreBlocksSettings.tsx:127`):
   - Replace `{block.contentType === 'movie' ? '🎬 電影' : '📺 影集'}` with a lucide icon + plain text: `movie` → `<Film className="inline h-3.5 w-3.5 text-[var(--text-muted)]" aria-hidden="true" />` immediately before `電影`; `tv` → `<Tv className="inline h-3.5 w-3.5 text-[var(--text-muted)]" aria-hidden="true" />` before `影集`. Keep the existing ` · {block.maxItems} 個項目` suffix and the genre/region suffixes exactly as-is.
   - `import { Film, Tv } from 'lucide-react'` is ADDED to the existing `import { Plus, Pencil, Trash2, ArrowUp, ArrowDown } from 'lucide-react'` line at the top of the file.
   - Icon size/color MUST match the SettingsLayout idiom (`h-3.5 w-3.5` … `h-4 w-4` lucide line icons, `text-[var(--text-muted)]` / `text-[var(--text-secondary)]`). DO NOT introduce a new color literal.
   - This is the canonical "no decorative emoji in admin chrome — use lucide consistent with SettingsLayout" invariant; future settings rows that need a content-type marker reuse `<Film>`/`<Tv>`.

5. **[@contract-v1] ExploreBlock empty state renders NO scroll chevrons** (`ExploreBlock.tsx`, the `items.length === 0` branch at ~L154-161 and the `<button>`s at ~L103-120):
   - When `items.length === 0` (block fetched OK but has zero matching results), the two scroll-chevron `<button>`s MUST NOT be rendered at all (`queryByTestId('explore-block-scroll-left')` / `...-right` return `null`). Rationale: nothing to scroll ⇒ no scroll affordance ⇒ the `沒有符合條件的內容` message at the left edge can no longer be clipped/overlapped by the left chevron. This is the fix — NOT a `z-index` bump and NOT a left-`padding` hack.
   - The `data-testid="explore-block-empty"` element keeps its content `沒有符合條件的內容`, stays left-aligned, `text-[var(--text-muted)]`, `py-8` (unchanged). The empty `<div>` MUST get normal left rhythm now that the chevron is gone (it already lives inside the `explore-block-scroller` flex with no left chevron over it).
   - **Block still renders** when empty — only `isError` makes `ExploreBlock` `return null` (unchanged). An empty block with the message is informative for a user who configured an over-narrow filter.

6. **Test coverage** (extend existing specs — no new spec files needed):
   - `apps/web/src/components/homepage/ExploreBlock.spec.tsx`:
     - The existing populated-block chevron test (`renders scroll chevrons`, ~L278-280) MUST still pass — verifies AC #2.
     - ADD a test: render a block whose content query resolves to `{ movies: [], tvShows: [] }` (empty), then `expect(screen.queryByTestId('explore-block-scroll-left')).toBeNull()` AND `...('explore-block-scroll-right')).toBeNull()` AND `expect(await screen.findByTestId('explore-block-empty')).toHaveTextContent('沒有符合條件的內容')` — verifies AC #5. (The existing empty-state test at ~L200-201 already asserts the message text; this new one adds the chevron-absence assertion — either extend that test or add a sibling.)
     - Use Rule 16 specific matchers (`toBeInTheDocument` / `toBeNull` / `toHaveTextContent`, never `toBeTruthy`).
   - `apps/web/src/components/settings/ExploreBlocksSettings.spec.tsx`:
     - ADD an assertion in the existing "renders block list" test (~L82-93): after the rows render, the movie row's meta line shows `電影` WITHOUT the `🎬` glyph (e.g. `expect(screen.getByText(/^電影 ·/)).toBeInTheDocument()` or assert the row text does NOT contain `🎬`), and similarly `影集` without `📺`. Optionally assert the row contains an `svg` (`container.querySelector('svg')`) — lucide renders as inline `<svg>`; loose assertion is fine, don't over-couple to lucide internals.
   - `apps/web/src/components/settings/ExploreBlockEditModal.spec.tsx` / `LibraryEditModal.spec.tsx` / `MediaLibrarySetupStep.spec.tsx`: only touch these if an existing test asserts the `🎬 電影` / `📺 影集` option text — grep first; if a test references the emoji'd option label, update it to the plain `電影` / `影集`. If no test references it, no change needed (the `value` attributes are what tests query by, and those are unchanged).
   - **Anti-pattern guard**: ZERO new `as any` casts. If a test needs to mock a hook, use the `Partial<ReturnType<typeof useX>>` typed-mock pattern (bugfix-10-2 CR M3 precedent).

7. **Design-traceability note (Rule 21, soft)**: Sally's design contract for this story is **Screen HP-5 "ExploreBlock Polish (bugfix-10-6)"** in `ux-design.pen` (frame `Y5XvRv`, committed `936fdb0`) — section A `FjisT` = chevron treatment, section B `MAwOp` = empty state, section C `PhBJ8` = lucide icon Before/After. Rule 21's strict `// Implements: Component/{Name} ({pen-node-id})` header format does **not** cleanly apply here because HP-5's sub-sections are *screen frames*, not Reusable Components. DEV MAY add a soft reference comment near the top of `ExploreBlock.tsx` and `ExploreBlocksSettings.tsx` — e.g. `// Design ref: ux-design.pen Screen HP-5 (Y5XvRv) — bugfix-10-6 polish` — but this is **optional / DEV+CR judgment**, not a hard AC. (Filing the canonical Rule-21-format backfill for these two pre-Rule-21 files is out of scope here → that's epic-19-8's full sweep.)

8. **AC Drift / Rule 20**: This story introduces NEW `[@contract-v1]` stamps on **AC #1** (ExploreBlock scroll-chevron visibility/contrast invariant), **AC #4** (no-emoji-use-lucide invariant for admin block rows), and **AC #5** (ExploreBlock empty state renders no chevrons). No upstream stamped ACs to acknowledge — `ExploreBlock.tsx` (Story 10.3) and `ExploreBlocksSettings.tsx` (Story 10.3) predate Rule 20 → implicit `v0`, forward-only retrofit per Rule 20. Change Log MUST record the `[@contract-v0→v1]` rows (see Change Log section). The visual changes here do not break any downstream code or test (the `data-testid`s are stable; no test asserts `bg-black/70` or the `🎬 電影` string today — DEV verifies this with a grep before claiming completion).

9. **Regression gates** (Definition of Done):
   - `pnpm nx test web` PASS — baseline **1787** tests (post-bugfix-10-5) + the new assertions from AC #6. No removals expected.
   - `pnpm nx test api` PASS (no Go changes; run anyway per Epic 9 Retro AI-1 mandatory-gate rule — if `TestScannerService_SSEBroadcast_ScanCancelled` flakes on the full-suite run, retry once and reference `preexisting-fail-scanner-sse-scan-cancelled-flake` in sprint-status line 352; do NOT file a new entry).
   - `pnpm lint:all` → **0 errors / 122 warnings** — matches the bugfix-10-5 closeout baseline EXACTLY. ZERO new warnings. `prettier --check` (or `pnpm format:check`) clean on every touched file.
   - `pnpm run test:cleanup` verified — no orphaned vitest workers (Epic 9c retro lesson).
   - **No `.pen` edits** in this story — Sally's `936fdb0` locked the design contract. DEV does NOT run `scripts/export-pen-screenshots.py`.
   - Manual smoke (Task 6 substitute, CLI precedent): `pnpm nx serve web` against any backend → on the homepage, hover an explore block and confirm the left/right chevrons fade in with visible contrast (and that they're absent on a block with zero results — if you can produce one; otherwise the unit test covers it); open Settings → 自訂首頁 and confirm the block rows show a small lucide film/tv icon instead of 🎬/📺. Browser-pixel verification against HP-5 screenshot deferred to user / NAS deploy. **Per bugfix-10-2/10-5 CLI precedent**, deterministic vitest assertions (AC #6) substitute for the browser smoke since a CLI agent can't drive Chrome DevTools.

## Tasks / Subtasks

- [ ] **Task 1 — ExploreBlock chevron contrast + scrim + hover-reveal** (AC: #1, #2)
  - [ ] 1.1 Add `group/scroller` (named group) to the `relative` wrapper that contains the chevrons + the scroller (currently `<div className="relative">` at ~L101). Verify no ancestor in that subtree has a conflicting `*group-hover*:opacity-0` rule (bugfix-10-4 CR H2 cascade trap).
  - [ ] 1.2 Restyle both chevron `<button>`s: `bg-black/70 ... hover:bg-black/90` → `bg-[var(--bg-secondary)]/95 backdrop-blur-sm ring-1 ring-[var(--border-subtle)]/70 text-[var(--text-primary)] shadow-lg hover:bg-[var(--bg-tertiary)]`; glyph `text-white` → `text-[var(--text-primary)]`; ADD `opacity-0 transition-opacity group-hover/scroller:opacity-100`. Keep `hidden lg:block`, keep `absolute left-0/right-0 top-1/2 z-10 -translate-x-1/2/-translate-y-1/2` positioning, keep `rounded-full p-2`, keep `data-testid` + `aria-label` + `onClick`.
  - [ ] 1.3 Add the left/right edge gradient scrims: two `<div>`s inside the `relative` wrapper, behind the chevrons (`z-0` or just earlier in DOM order, but still above the scroller — actually put them at `z-[5]` between scroller and chevron, OR simpler: render scrim then chevron as siblings after the scroller so paint order handles it). `pointer-events-none`, `absolute inset-y-0 left-0 w-14 bg-gradient-to-r from-[var(--bg-primary)] to-transparent` (and the right mirror with `bg-gradient-to-l` + `right-0`). They should also be `hidden lg:block` and `opacity-0 transition-opacity group-hover/scroller:opacity-100` so the whole arrow-affordance fades together. (If the scrim feels too heavy visually, it MAY be left always-visible at a low strength — DEV judgment, but the hover-together version matches HP-5.)
  - [ ] 1.4 (OPTIONAL per AC #1 last bullet) overflow-awareness: `useState` for `canScrollLeft`/`canScrollRight`, an `onScroll` handler on the scroller `<div>` + a `ResizeObserver`, update the flags; chevron buttons get `enabled={canScrollX}`-style conditional render. **Default both flags to `true`** when `scroller.scrollWidth === 0` (jsdom). If this balloons the diff, skip it and leave a `// TODO(bugfix-10-7?): hide chevron when no scroll room` comment instead.

- [ ] **Task 2 — ExploreBlock empty state drops the chevrons** (AC: #5)
  - [ ] 2.1 Make the two scroll-chevron `<button>`s conditional on `items.length > 0` (or equivalently on `!showSkeleton && items.length > 0` — they're already inside the non-skeleton path's sibling space, but currently rendered unconditionally inside `<div className="relative">`). When empty, render neither chevron nor the scrims (Task 1.3) — only the `explore-block-scroller` flex with the `explore-block-empty` message.
  - [ ] 2.2 Verify the `explore-block-empty` `<div>` keeps `py-8 text-sm text-[var(--text-[var(--text-muted)]]` (actually `text-[var(--text-muted)]`) and is left-aligned within the scroller flex (it already is — just confirm no left padding/margin was needed to dodge the now-removed chevron).

- [ ] **Task 3 — `<option>` emoji → plain text** (AC: #3)
  - [ ] 3.1 `ExploreBlockEditModal.tsx:165-166`: `🎬 電影` → `電影`, `📺 影集` → `影集`.
  - [ ] 3.2 `LibraryEditModal.tsx:132-133`: `🎬 電影` → `電影`, `📺 影集` → `影集`.
  - [ ] 3.3 `MediaLibrarySetupStep.tsx:87-88`: `🎬 電影` → `電影`, `📺 影集` → `影集`.
  - [ ] 3.4 Grep `🎬|📺` across `apps/web/src/components/settings/` + `apps/web/src/components/setup/` to confirm no other `<option>` was missed; do NOT touch `RecentMediaPanel.tsx:115` poster-fallback 🎬, `MetadataSourceBadge.tsx:11` `{ icon: '🎬' }`, or the notification toasts (`NewMediaToast`/`ParseCompleteToast`/`NewMediaNotifications`) — out of scope.

- [ ] **Task 4 — ExploreBlocksSettings block-row lucide icons** (AC: #4)
  - [ ] 4.1 Add `Film, Tv` to the existing `import { Plus, Pencil, Trash2, ArrowUp, ArrowDown } from 'lucide-react'` line.
  - [ ] 4.2 Replace the `{block.contentType === 'movie' ? '🎬 電影' : '📺 影集'}` ternary at L127 with a `<>{...}</>` fragment: `{block.contentType === 'movie' ? (<><Film className="inline h-3.5 w-3.5 text-[var(--text-muted)]" aria-hidden="true" /> 電影</>) : (<><Tv className="inline h-3.5 w-3.5 text-[var(--text-muted)]" aria-hidden="true" /> 影集</>)}` — keep the ` · {block.maxItems} 個項目` and the genre/region suffixes that follow. Mind whitespace: a `{' '}` or a space inside the fragment between icon and label so it reads `🎞 電影` not `🎞電影`.
  - [ ] 4.3 (HP-5 has `aRow1Act`/`aRow2Act` showing the existing `ArrowUp/ArrowDown/Pencil/Trash2` action icons on the right — those are ALREADY lucide in the code, no change needed; just don't accidentally remove them.)

- [ ] **Task 5 — Tests** (AC: #6)
  - [ ] 5.1 `ExploreBlock.spec.tsx`: keep the existing populated-block chevron test green (AC #2). Add the empty-block chevron-absence test (AC #5): mock `useExploreBlockContent` to resolve `{ movies: [], tvShows: [] }`, assert both `queryByTestId('explore-block-scroll-left'|'...-right')` are `null` + `explore-block-empty` has the message text.
  - [ ] 5.2 `ExploreBlocksSettings.spec.tsx`: in the existing block-list render test, assert the movie row's meta shows `電影` without `🎬` and the tv row shows `影集` without `📺` (and optionally `container.querySelector('svg')` is non-null on a row). Use Rule 16 matchers.
  - [ ] 5.3 Grep `ExploreBlockEditModal.spec.tsx` / `LibraryEditModal.spec.tsx` / `MediaLibrarySetupStep.spec.tsx` for `🎬|📺|電影|影集` text assertions on `<option>` labels; update any that break (likely none — tests query `<select>` by `value`/`data-testid`).
  - [ ] 5.4 `pnpm nx test web` → expect ≥ 1787 PASS (baseline + new assertions).

- [ ] **Task 6 — Regression gates + closeout** (AC: #9)
  - [ ] 6.1 `pnpm nx test web` PASS (≥1787). `pnpm nx test api` PASS (retry once if the SSE flake hits — reference existing backlog entry, don't file new).
  - [ ] 6.2 `pnpm lint:all` → 0 errors / 122 warnings (matches bugfix-10-5 closeout EXACTLY — ZERO new). `prettier --check` / `pnpm format:check` clean on touched files. (Reminder: subagent edits skip Prettier — run format before commit per `feedback_format_before_commit`.)
  - [ ] 6.3 `pnpm run test:cleanup` → no orphaned vitest workers.
  - [ ] 6.4 Confirm NO `.pen` file changes in `git status` and `scripts/export-pen-screenshots.py` was NOT run (this story doesn't touch the design).
  - [ ] 6.5 Manual smoke (CLI substitute per bugfix-10-2/10-5): the AC #6 deterministic tests substitute for the browser smoke. Browser DevTools verification (hover-fade timing, scrim contrast over poster art, Settings row icons at 390/1440) recommended on NAS deploy. Sprint-status `in-progress → review` flip handled at dev-story Step closeout (workflow boundary).

## Dev Notes

### Pre-flight confirmed by SM (sanity-check before edits, don't re-derive):

1. **`ExploreBlock.tsx` current chevron code** is at `apps/web/src/components/homepage/ExploreBlock.tsx:101-120` — a `<div className="relative">` wrapper, two `<button>`s with `className="absolute left-0/right-0 top-1/2 z-10 hidden -translate-x-1/2 -translate-y-1/2 rounded-full bg-black/70 p-2 text-white hover:bg-black/90 lg:block"`, each containing a `<ChevronLeft/ChevronRight className="h-5 w-5" />` from `lucide-react`. The `scroll('left'|'right')` handler calls `scrollerRef.current.scrollBy({ left: ±clientWidth*0.8, behavior: 'smooth' })`. The scroller `<div data-testid="explore-block-scroller">` and the empty `<div data-testid="explore-block-empty">沒有符合條件的內容</div>` are siblings of the chevrons inside that `relative` wrapper (the empty `<div>` is actually *inside* the scroller flex, after the `items.map`). `ChevronLeft`, `ChevronRight` are already imported from `lucide-react` — no new import for Task 1.

2. **`ExploreBlocksSettings.tsx` current emoji** is at `apps/web/src/components/settings/ExploreBlocksSettings.tsx:127`: `{block.contentType === 'movie' ? '🎬 電影' : '📺 影集'} · {block.maxItems} 個項目{block.genreIds && ...}{block.region && ...}`. The file ALREADY does `import { Plus, Pencil, Trash2, ArrowUp, ArrowDown } from 'lucide-react'` (top of file) — Task 4.1 just appends `Film, Tv` to that line. The 4 action `<button>`s on the right of each row already use `ArrowUp/ArrowDown/Pencil/Trash2` — leave them.

3. **The 3 `<option>` files** (Task 3): `ExploreBlockEditModal.tsx:165-166`, `LibraryEditModal.tsx:132-133`, `MediaLibrarySetupStep.tsx:87-88` — each has `<option value="movie">🎬 電影</option>` + `<option value="tv|series">📺 影集</option>`. The `value`s differ (`tv` in ExploreBlockEditModal, `series` in the other two) — leave `value`s and `data-testid`s alone, only strip the emoji + leading space from the visible label text.

4. **Test baselines** (post-bugfix-10-5): `pnpm nx test web` = **1787** tests PASS; `pnpm lint:all` = **0 errors / 122 warnings**. Any deviation from those baselines (beyond the new test count delta from AC #6) is a regression you introduced — fix it, don't paper over it.

5. **`useInViewport.ts` pattern** (`apps/web/src/hooks/useInViewport.ts`) — if you do the optional overflow-awareness in Task 1.4, mirror its SSR-safe shape: guard `typeof ResizeObserver === 'undefined'` (jsdom doesn't have it by default — actually jsdom DOES have a stub since vitest, but it doesn't fire; defaulting `canScroll*` to `true` covers it), clean up the observer on unmount. Do NOT add a new dependency.

6. **lucide `Film` / `Tv` availability**: both are in the installed lucide-react (used widely — `Film` appears in `EmptyNoQBT.tsx` from bugfix-10-5; `Tv` … verify with `import { Film, Tv } from 'lucide-react'` — if `Tv` TS-errors, fall back to `Tv2` or `MonitorPlay`, but `Tv` should be there). If verifying via `node_modules/lucide-react/dist/lucide-react.d.ts` grep is faster, do that.

7. **No backend changes. No `.pen` changes. No new files.** Everything in this story is an edit to 5 existing `.tsx` source files + their 2-3 existing `.spec.tsx` files.

### Sally's design contract (Screen HP-5)

Committed `936fdb0` (2026-05-11). Screen HP-5 "ExploreBlock Polish (bugfix-10-6)" in `ux-design.pen` — frame `Y5XvRv`, screenshot `_bmad-output/screenshots/flow-g-homepage-desktop/hp5-exploreblock-polish-bugfix-10-6.png`. Three sections:

| HP-5 section | Node | AC | What it shows |
|---|---|---|---|
| A — scroll chevron | `FjisT` | #1, #2 | Before (`bg-black/70`, invisible) vs After (`bg-secondary/95` + `backdrop-blur` + `ring` + `shadow` on a `from-[var(--bg-primary)]→transparent` edge scrim) contrast chips, plus a realistic hover-state demo block ("動作冒險精選") with both chevrons on scrims and the last poster cut off (overflow) |
| B — empty state | `MAwOp` | #5 | An empty block ("我的院線追蹤") demo card — `沒有符合條件的內容` left-aligned, `text-muted`, `py-8`, **no left/right chevrons** |
| C — lucide icons | `PhBJ8` | #3, #4 | Before column (🎬/📺 emoji rows) vs After column (`<Film>`/`<Tv>` lucide rows + the existing `ArrowUp/Pencil/Trash2` action icons) |

HP-5 also confirms HP-1's block titles dropped their decorative emoji prefixes — Sally already did that in the `.pen`; it's NOT something DEV touches (the block names come from the backend `熱門電影` / `熱門影集` / `近期新片` defaults which never had emoji; the `.pen` mockup had added decorative ones).

### Cross-cutting Rule compliance checklist

- **Rule 4 (Layered Architecture)**: N/A — pure UI, no service layer touched.
- **Rule 5 (TanStack Query for server state)**: N/A — no new server state. The optional overflow flags in Task 1.4 are *UI* state (`useState`), not server state — that's allowed (TanStack Query is for server state only).
- **Rule 6 (Naming)**: ✅ Components stay `PascalCase.tsx`, testids `kebab-case`. New `useState` flags `canScrollLeft`/`canScrollRight` are camelCase.
- **Rule 11 (Interface Location)**: N/A — no new interfaces.
- **Rule 12 (lint:all baseline)**: ✅ Task 6.2 enforces 0/122.
- **Rule 13 (Error Handling Completeness)**: N/A — no error paths added.
- **Rule 14 (Resource Lifecycle)**: ⚠️ IF Task 1.4 adds a `ResizeObserver` / `onScroll` listener, it MUST be torn down on unmount (`useEffect` cleanup). If you skip Task 1.4, N/A.
- **Rule 15 (Pre-commit self-verification)**: Wiring/DB/Swagger/Route-Client-sync all N/A (no API change). Run the lint + test gates (Task 6).
- **Rule 16 (Test Assertion Quality)**: ✅ AC #6 mandates `toBeInTheDocument` / `toBeNull` / `toHaveTextContent` — never `toBeTruthy`.
- **Rule 18 (API Boundary Case Transformation)**: N/A — no new API service code.
- **Rule 19 (Package Boundaries)**: N/A — pure frontend.
- **Rule 20 (AC Contract Versioning)**: ✅ AC #1, #4, #5 stamped `[@contract-v1]`. Change Log records the `[@contract-v0→v1]` retrofit rows. Forward-only: `ExploreBlock.tsx` / `ExploreBlocksSettings.tsx` predate Rule 20 → implicit v0, no upstream stamps to ack.
- **Rule 21 (Component-to-Design Node Traceability)**: SOFT for this story (AC #7) — HP-5 sub-sections are *screen frames* not Reusable Components, so the strict `// Implements: Component/X (id)` format doesn't apply; a `// Design ref: ... Screen HP-5 (Y5XvRv)` comment is optional. The canonical Rule-21 backfill of these two pre-Rule-21 files is epic-19-8's job.
- **Rule 22 (Epic Retro Design-Drift Audit)**: Deferred to Epic 10 retro (fires at retro time, not story closeout).

### Previous story intelligence

- **bugfix-10-5 (empty-library-onboarding)** — just completed. Lessons: (1) CR found Pre-flight #4 had a wrong hook return shape → DEV trusted it → bug. **For this story**: the Pre-flight section above gives you exact file:line — but still grep/open the file before editing to confirm the line numbers haven't drifted. (2) CR M1 was about File List omitting TA-pass files — keep your File List truthful, list every file you touched. (3) Candidate Epic 10 retro item: add `tsc --noEmit` to CI — for this story, since you're touching `.tsx`, a quick `pnpm exec tsc -p apps/web/tsconfig.app.json --noEmit` before commit is cheap insurance (note: there are 2 pre-existing `localData.releaseDate` errors in `LocalDetailView.tsx` per bugfix-10-1 CR — those are NOT yours; any NEW tsc error is).
- **bugfix-10-4 (hover-preview-viewport-flip)** — the **`group-hover` opacity cascade trap** (CR H2): `PosterCard.tsx:171` had a child with `opacity-0 lg:group-hover:opacity-100` nested under a parent with `lg:group-hover:opacity-0` → the child's effective opacity was `0` in BOTH default and hover states (parent multiplies down). **Directly relevant to AC #1 Task 1.1/1.3** — your chevron buttons + scrims get `opacity-0 group-hover/scroller:opacity-100`; make sure nothing between them and the `group/scroller` element has a competing `group-hover:opacity-0`. Use the *named* group `group/scroller` to be unambiguous. Also: bugfix-10-4 CR H1 was File-List-lying — don't.
- **bugfix-10-2 (qbt-downloads-http-status)** — established the `Partial<ReturnType<typeof useX>>` typed-mock pattern (vs `as any`) referenced in AC #6.
- **bugfix-10-3 (skeleton-flicker)** — `## 🧪 Known dev-mode artifacts` section in `project-context.md`: StrictMode double-mounts components in dev. If you do any manual smoke, `pnpm nx run web:preview` (prod build, StrictMode stripped) is the truthful target — but for this story the unit tests are the gate.

### Project Structure Notes

**Modified (production):**
- `apps/web/src/components/homepage/ExploreBlock.tsx` (chevron restyle + scrims + hover-reveal + empty-state-no-chevrons; optional overflow flags)
- `apps/web/src/components/settings/ExploreBlocksSettings.tsx` (import `Film, Tv`; emoji → lucide at L127)
- `apps/web/src/components/settings/ExploreBlockEditModal.tsx` (L165-166 `<option>` emoji → plain text)
- `apps/web/src/components/settings/LibraryEditModal.tsx` (L132-133 `<option>` emoji → plain text)
- `apps/web/src/components/setup/MediaLibrarySetupStep.tsx` (L87-88 `<option>` emoji → plain text)

**Modified (tests):**
- `apps/web/src/components/homepage/ExploreBlock.spec.tsx` (add empty-block-no-chevrons test; keep populated chevron test)
- `apps/web/src/components/settings/ExploreBlocksSettings.spec.tsx` (assert lucide icon / no-emoji on rows)
- *(possibly)* `ExploreBlockEditModal.spec.tsx` / `LibraryEditModal.spec.tsx` / `MediaLibrarySetupStep.spec.tsx` — only if an existing test asserts the `🎬 電影` option label (grep first; likely no change)

**Untouched (DO NOT TOUCH — out of scope):**
- All backend code (`apps/api/**`), all migrations, swagger.
- `ux-design.pen` (Sally's `936fdb0` is the locked contract for this story).
- `_bmad-output/screenshots/**` (no `.pen` change → no regen).
- `scripts/export-pen-screenshots.py`.
- `RecentMediaPanel.tsx:115` poster-fallback `🎬`, `MetadataSourceBadge.tsx:11` `{ icon: '🎬', label: 'TMDb' }`, `NewMediaToast.tsx` / `ParseCompleteToast.tsx` / `NewMediaNotifications.tsx` `🎬` — different contexts (poster placeholder, source badge, notification glyph); NOT in this story's scope (filed loosely under the "full emoji sweep" idea in the sprint-status note's OUT OF SCOPE list).
- `ExploreBlocksList.tsx` (parent — the `flex flex-col gap-6 md:gap-8` wrapper; no group-hover there, leave it).
- `ExploreBlockSkeleton.tsx` (loading skeleton — unrelated to the populated/empty states this story touches).
- `HeroBanner.tsx`, `RecentMediaPanel.tsx`, the homepage route — unrelated.

### References

- Sally's UX pass + design contract: `_bmad-output/implementation-artifacts/sprint-status.yaml` (bugfix-10-6 entry, post-commit `936fdb0`); `ux-design.pen` Screen HP-5 (`Y5XvRv`); screenshot `_bmad-output/screenshots/flow-g-homepage-desktop/hp5-exploreblock-polish-bugfix-10-6.png`.
- Code under change: `apps/web/src/components/homepage/ExploreBlock.tsx:101-161`; `apps/web/src/components/settings/ExploreBlocksSettings.tsx:127`; `apps/web/src/components/settings/ExploreBlockEditModal.tsx:165-166`; `apps/web/src/components/settings/LibraryEditModal.tsx:132-133`; `apps/web/src/components/setup/MediaLibrarySetupStep.tsx:87-88`.
- Icon idiom precedent: `apps/web/src/components/settings/SettingsLayout.tsx` (lucide `Plug/Database/FileText/Activity/HardDrive/ArrowUpDown/Gauge/ScanLine/LayoutGrid` at `h-4 w-4` / `h-3.5 w-3.5`, `text-[var(--text-secondary)]`).
- `group-hover` cascade trap: `apps/web/src/components/library/PosterCard.tsx` (bugfix-10-4 CR H2 fix); story `bugfix-10-4-hover-preview-viewport-flip.md`.
- Typed-mock pattern: bugfix-10-2 CR M3 — `Partial<ReturnType<typeof useX>>`.
- Project rules: `project-context.md` Rule 5, Rule 16, Rule 20, Rule 21 (soft here), Rule 22; `## 🧪 Known dev-mode artifacts` section.
- Test/lint baselines: sprint-status line 491 (bugfix-10-5 closeout: 1787 tests, 0 errors / 122 warnings).

## Change Log

| Date | Change |
|---|---|
| 2026-05-11 | [@contract-v0→v1] AC #1 (new): ExploreBlock scroll-chevron visibility/contrast invariant — `bg-black/70` (invisible on `$bg-primary`) → `bg-[var(--bg-secondary)]/95 backdrop-blur ring shadow` on an edge gradient scrim, `opacity-0 lg:group-hover:opacity-100` hover-reveal, `hidden lg:block` touch-suppression preserved, `explore-block-scroll-left`/`-right` testids stable. Future homepage redesigns must not regress to invisible chevrons (bump v2 + Change Log). ExploreBlock.tsx (Story 10.3) was implicit v0 (pre-Rule-20) → forward-only retrofit. No downstream code/test breaks (no test asserts `bg-black/70`). |
| 2026-05-11 | [@contract-v0→v1] AC #4 (new): Settings → 自訂首頁 block-row content-type marker uses lucide `<Film>` / `<Tv>` (matching SettingsLayout's lucide idiom, `h-3.5 w-3.5 text-[var(--text-muted)]`) instead of 🎬/📺 emoji; native `<option>` labels (ExploreBlockEditModal/LibraryEditModal/MediaLibrarySetupStep) drop the emoji to plain `電影`/`影集`. No downstream code/test breaks (no test asserts the `🎬 電影` string; `<select>` tests query by `value`/`data-testid`). |
| 2026-05-11 | [@contract-v0→v1] AC #5 (new): ExploreBlock empty state (`items.length === 0`) renders NO scroll chevrons (nothing to scroll ⇒ no affordance ⇒ `沒有符合條件的內容` message can't be clipped). The message div keeps `text-[var(--text-muted)] py-8` left-aligned; block still renders when empty (only `isError` → `return null`). Future changes that re-introduce chevrons-when-empty must bump v2. |

## Dev Agent Record

### Agent Model Used

_(to be filled by DEV)_

### Debug Log References

_(to be filled by DEV)_

### Completion Notes List

_(to be filled by DEV — must include: 🔗 AC Drift, 📎 Contract Stamps, 🔒 Rule 7 Wire Format = N/A pure FE, 🎨 UX verification result, lint baseline confirmation)_

### File List

_(to be filled by DEV — list every modified production + test file; this story creates no new files and touches no `.pen` / screenshots)_
