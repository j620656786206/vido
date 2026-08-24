# Story bugfix: SettingsLayout doesn't stretch to fill the v2 shell, leaving a giant blank void

Status: done

## Story

As anyone opening any 設定 (Settings) sub-page on a wide desktop screen,
I want the settings sidebar + form to fill the available content width like every other v2 page does,
so that the page looks like a real desktop layout instead of a shrink-wrapped column with a huge empty void on the right.

## Evidence

- Alexyu screenshotted `/settings/connection` on a wide browser window: the settings sub-nav + form column hugs the left edge, with a large blank area on the right. Compared against the design (`ux-design.pen` frame `6UCtX`/C4-D, 1440px wide) this is not the intended look.
- Root cause: `apps/web/src/components/settings/SettingsLayout.tsx:97`

  ```tsx
  <div className="mx-auto flex max-w-7xl flex-col md:flex-row" data-testid="settings-layout">
  ```

  is missing `w-full`. Its parent is `<main className="flex flex-1 flex-col ...">` (`AppShellV2.tsx`) — a **column flex container**. Without `w-full`, this div (a flex item, `display:flex` itself) sizes to its content's natural width instead of stretching to the container's width and THEN being capped/centered by `max-w-7xl` + `mx-auto`. The result: the whole sidebar+form cluster shrink-wraps to content width and sits at the flex-start (left) edge, instead of centering as a 1280px-wide block.
- Confirmed by comparing against every other page mounted in the same `AppShellV2` shell that uses the identical `mx-auto flex ... max-w-*` shape — **all of them include `w-full`**:
  - `HomeBrowseV2.tsx:36` — `mx-auto flex w-full max-w-7xl flex-col ...`
  - `ActivityHub.tsx:303` — `mx-auto flex w-full max-w-5xl flex-col ...`
  - `DownloadsBrowseV2.tsx:185` — `mx-auto flex w-full max-w-5xl flex-col ...`
  - `ExploreBlock.tsx:89` — `mx-auto w-full max-w-7xl ...`

  `SettingsLayout.tsx` is the sole exception. (Non-flex pages like `HeroBanner`/`LocalDetailV2`/`TMDbDetailV2` correctly omit `w-full` — they aren't `display:flex` themselves, so block-level `width:auto` already fills the parent and `mx-auto` centers normally. The bug is specific to *flex* containers missing `w-full`.)
- Blast radius: `SettingsLayout` is the shared shell for **every** settings sub-page (connection, keys, scanner, homepage, cache, logs, status, backup, export, performance, qbittorrent) — all affected, not just the connection form Alexyu screenshotted.

## Acceptance Criteria

1. `SettingsLayout`'s root div gains `w-full` alongside its existing `mx-auto flex max-w-7xl` classes, matching the established pattern used by `HomeBrowseV2` / `ActivityHub` / `DownloadsBrowseV2` / `ExploreBlock`.
2. A regression test asserts the root element carries `w-full` (or, more robustly, asserts computed/rendered width behavior) so a future edit cannot silently drop it again without a test going red.
3. Verified visually against a running dev server on at least 2 settings sub-pages (`/settings/connection` and one other, e.g. `/settings/cache`) at a wide viewport (≥1600px) — the settings block now centers with balanced side margins instead of hugging the left edge with a void on the right.
4. No visual regression at narrower / mobile viewports — the existing `md:flex-row` / `flex-col` responsive behavior and the `hidden md:block` desktop-sidebar toggle are untouched.
5. Gates: `pnpm nx test web` green, `pnpm nx lint web` clean, `pnpm run format:check` green. Zero backend changes.

## Tasks / Subtasks

- [x] Task 1 — Fix + regression test (AC: #1, #2)
  - [x] 1.1 Add `w-full` to `SettingsLayout.tsx`'s root div
  - [x] 1.2 Add a test to `SettingsLayout.spec.tsx` asserting `w-full` is present on the `settings-layout` testid element
- [x] Task 2 — Manual + automated verification (AC: #3, #4)
  - [x] 2.1 Run the dev server, screenshot `/settings/connection` and one other sub-page at a wide viewport, compare before/after
  - [x] 2.2 Confirm mobile/tablet breakpoint behavior unchanged (existing tests + a narrow-viewport spot check)
- [x] Task 3 — Gates (AC: #5)

## Dev Notes

- **One-line fix, but do not skip the regression test.** This exact class of bug (a missing `w-full` on a flex item inside a `flex-col` parent) is easy to silently reintroduce on any future v2-shell page — a test that would have caught it going forward is the point, not just fixing today's symptom.
- **Do not touch `ConnectionSettingsPage`'s own `max-w-2xl` wrapper** (`routes/settings/connection.tsx:10`) — that is the form's own intentional narrower reading width, nested inside the (now-correctly-centered) `max-w-7xl` shell. Different concern, not part of this bug.
- **Do not "fix" the non-flex pages** (`HeroBanner`, `LocalDetailV2`, `TMDbDetailV2`, `gallery.tsx`) that use `mx-auto max-w-*` without `w-full` — those are correct as-is (see Evidence).
- Rule 7 / 10 / 20: N/A — no error codes, no routes, no wire contract. Rule 23: N/A — no wall-clock component logic touched.
- Discovered via Alexyu's live screenshot + demo (not a spec-reading exercise) — per `feedback_let_user_demo_before_proposing`, the diagnosis was built from what he actually showed, not from assumptions about the bug title.

### Time-dependent visual coverage

- N/A — no `Date.now()`/`new Date()` reading components touched.

### References

- [Source: apps/web/src/components/settings/SettingsLayout.tsx:97 — the missing `w-full`]
- [Source: apps/web/src/components/shell/AppShellV2.tsx — the `<main className="flex flex-1 flex-col ...">` parent that makes this a flex cross-axis issue]
- [Source: apps/web/src/components/homepage/HomeBrowseV2.tsx:36, activity/ActivityHub.tsx:303, downloads/DownloadsBrowseV2.tsx:185 — the established `w-full` pattern]
- [Source: _bmad-output/screenshots/flow-c-search-settings/c4-d.png — the design reference]

## Dev Agent Record

### Agent Model Used

claude-fable-5 (Claude Fable 5) — proposed and prioritized via party-mode consensus (Bob/Amelia/Winston, 2026-08-24)

### Debug Log References

- RED: temporarily reverted `w-full`, ran the new regression test in isolation — failed, `expect.arrayContaining` diff showed `w-full` missing from the class list. Confirms the test actually catches this class of bug.
- GREEN: restored the fix, full `SettingsLayout.spec.tsx` suite 38/38 passing.
- Visual verification: `pnpm nx serve web` + a throwaway Playwright screenshot script (not committed — scratch only) at 1920×1080 on `/settings/connection` and `/settings/cache`, and at 390×844 (mobile) on `/settings/connection`. Desktop: sidebar+form now sits with balanced margins instead of hugging the left edge with a large void on the right. Mobile: horizontal tab bar + full-width form unchanged.
- Gates: `pnpm nx test web` green (both before and after the prettier auto-format), `pnpm run lint:all` 0 errors (119 pre-existing jsx-a11y warnings, unrelated to this change — same count as prior stories today), `pnpm run format:check` green after `prettier --write` on the two touched files.

### Completion Notes List

- One Tailwind class (`w-full`) plus one new regression test. Zero logic changes.
- 🔗 AC Drift: NONE (no prior story specifies `SettingsLayout`'s width behavior; this is the first).
- 📎 Contract Stamps: N/A (pure CSS/test change, no wire contract).
- 🎭 A11y Pre-Flight: N/A — no interactive elements, ARIA attributes, or semantics touched; only a layout width class on an existing `<div>`.
- 🎨 UX Verification: PASS — visually confirmed against the design reference (`ux-design.pen` C4-D, 1440px) at desktop and mobile viewports; the settings shell now centers instead of shrink-wrapping to the left.

### Discovery Triage

- N/A — no out-of-scope work discovered. (Diagnosis, story creation, and this fix were the entire scope; the party-mode discussion that shaped the approach is process, not a separate discovery.)

### File List

- apps/web/src/components/settings/SettingsLayout.tsx
- apps/web/src/components/settings/SettingsLayout.spec.tsx
- _bmad-output/implementation-artifacts/sprint-status.yaml
- _bmad-output/implementation-artifacts/bugfix-settingslayout-missing-w-full.md

## Change Log

| Date | Change |
|------|--------|
| 2026-08-24 | Task 1: added `w-full` to `SettingsLayout.tsx`'s root div; added a regression test proven via a manual RED→GREEN cycle (temporarily reverted the fix, confirmed the test fails, restored, confirmed it passes). |
| 2026-08-24 | Task 2: visually verified at desktop (1920×1080, 2 sub-pages) and mobile (390×844) viewports against a local dev server; no regression to the existing responsive breakpoint behavior. |
| 2026-08-24 | Task 3: gates green — `nx test web`, `lint:all` (0 errors), `format:check`. |
