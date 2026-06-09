# Automation Summary — Batch Subtitle Search UI

**Date:** 2026-06-09
**Story:** 8-11 (`8-11-batch-subtitle-ui`)
**Mode:** BMad-Integrated (`*automate`)
**Coverage Target:** critical-paths (wire-level integration gap)
**Agent:** Murat (TEA) · TestSprite enhancements off, Playwright-utils off

---

## What was generated

**New file:** `tests/e2e/batch-subtitle.spec.ts` — 6 tests × 4 browser projects (chromium, firefox, mobile-chrome, mobile-safari) = **24 discovered**.

| # | Pri | Test | AC |
|---|-----|------|----|
| 1 | P0 | 批次字幕搜尋 trigger reachable in the selection toolbar | #1 |
| 2 | P0 | Dialog opens, library scope default, season scope absent on `/library` | #2 |
| 3 | P0 | Start sends real `POST {scope:"library"}` (Rule 18) + 202 → processing | #3 |
| 4 | P1 | 409 conflict recovers to the in-progress snapshot, no error | #7 |
| 5 | P1 | Lazy SSE — idle dialog opens **no** EventSource; page hits `networkidle` | #8 |
| 6 | P2 | Cancel confirmation fires the real `POST /subtitles/batch/cancel` | #5 (request half) |

No new fixtures/factories/helpers — reused the repo's established `page.route` + `jsonOk` network-first convention (modeled on `availability-badges.spec.ts`).

---

## Risk-based design rationale

**Why these and not more.** DEV already shipped 42 FE specs + 3 Go tests + 3 visual baselines that thoroughly mock every boundary. Duplicating the component state machine or service transforms at E2E level would be wasted wall-clock. This suite covers **only the wire-level integration those mocks hide** — the real `/library → SelectionToolbar → BatchSubtitleDialog → subtitleService → POST` seam.

**Why the live-SSE journey is deliberately out of scope.**
- This repo **never mocks the `/api/v1/events` stream in E2E** — the lazy-SSE pattern exists precisely so `networkidle` works. Inventing an SSE-mock E2E pattern would add flake risk for coverage already held at the hook/component level.
- `startTracking()` flips to the processing state **optimistically**, so a mocked **202 alone** drives the processing UI — no SSE events needed for AC#3.
- AC#4 (live increments), AC#6 (complete summary + 「查看未找到項目」 deep-link), and AC#5's terminal `cancelled` reflection require the event stream. They are covered by `useSubtitleBatchProgress.spec` (drives `SSE_UPDATE`) + `BatchSubtitleDialog.spec` (full terminal state machine), and are the proper home for a future **TestSprite journey** against the real NAS.

**Deferred (carry-forward, not built here):**
- **TestSprite journey case** for the live batch flow — blocked operationally: no TestSprite MCP/token available in the working session (a stale key sits in the gitignored `testsprite_tests/tmp/config.json`, validity/credits unknown). Tracks `disc-2026-06-batch-subtitle-frontend-ui`.
- AC#6 deep-link **target** still blocked by the deferred backend `subtitle_status` list filter — backlog `disc-2026-06-library-subtitle-status-filter`.

---

## Validation

- **Discovery/compile:** `npx playwright test batch-subtitle --list` → 24 tests, clean TS compile.
- **Lint/format:** `eslint` clean; `prettier --check` clean.
- **Execution (chromium):** 6/6 passed.
- **Burn-in:** `--repeat-each=3` → **18/18 passed, zero flake**.
- One fix during validation: the `0/42` progress bar has `width:0%` (zero-width → not "visible"); switched to `toBeAttached()` per Rule 16. The counter `0 / 42` is the visible processing-state signal.

**Not run locally (covered by CI):** firefox, mobile-chrome, mobile-safari — the sharded E2E CI jobs run all projects. Mobile projects give AC#9 functional coverage for free; the bottom-sheet appearance is held by the existing visual baselines.

---

## Run commands

```bash
# All projects
npx playwright test batch-subtitle

# Local quick loop (primary browser)
npx playwright test batch-subtitle --project=chromium --reporter=list

# Flake check
npx playwright test batch-subtitle --project=chromium --repeat-each=3
```

---

## Definition of Done

- [x] Given-When-Then structure, `[P0]`/`[P1]`/`[P2]` priority tags
- [x] `data-testid` selectors only; no CSS/text coupling
- [x] Network-first (routes installed before `goto`); no hard waits
- [x] Rule 16 matchers (`toBeAttached` for zero-width transition element, `toHaveText` for counters)
- [x] Deterministic — no SSE-stream dependency; `/events` aborted, hermetic
- [x] Burn-in green ×3
- [x] File < 300 lines; no page objects; no shared state between tests

## Next steps

1. Land on `feat/8-11-batch-subtitle-ui`; CI runs all projects sharded.
2. When a TestSprite token is wired, author the live-progress journey case (AC#4/#6).
3. Once the backend `subtitle_status` filter ships, the AC#6 deep-link becomes end-to-end assertable.
