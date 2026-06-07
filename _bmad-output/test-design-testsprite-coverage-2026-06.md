# Test Design — TestSprite Coverage Expansion (2026-06)

**Author:** Murat (Master Test Architect) · **Date:** 2026-06-07
**Mode:** Targeted coverage-gap design (TestSprite journey layer)
**Scope decision (Alexyu):** Tier 0 cleanup + Tier 1 P0-journey gaps · Budget: **stay on Free 150 credits, add P0 only**
**Authoritative sources:** `testsprite_tests/testsprite_frontend_test_plan.json` (v4, 50 IDs), `testsprite_tests/standard_prd.json` (12 features), `_bmad-output/audit/testsprite-queue.yaml`

---

## 1. Baseline after Tier 0 cleanup

The `testsprite_tests/` directory held **48 `.py` files mixing v3 + v4 generations** (duplicate TC numbers — two `TC009`, two `TC010`, …). This inflated apparent coverage and would produce false results if the v3 orphans were ever executed.

- **Removed:** 30 v3 orphan files (`git rm`, staged — not committed).
- **Retained:** 18 canonical v4 files matching the plan.
- **Plan defines:** 50 v4 IDs — `TC009–014`, `TC035–078` (permanent gaps at `TC024–026`, `TC030–032`).
- **Honest execution status:** per `testsprite-queue.yaml`, only `TC035` has been generated+passed; `TC036` blocked; the other **48 IDs have `last_status: null` (never executed)**. The "50-case catalog" is currently *aspirational* — the monthly cron lazily generates each case on first pass.

> ⚠️ **Reality:** before adding cases, the existing 50 have not completed one full rotation. The cron sweeps ~24 cases/run (80% of 150 credits ÷ ~5/case) → ~3 months per full rotation. Coverage is **budget-bound**: every new case dilutes rotation frequency.

---

## 2. Coverage map (PRD v4 — 12 features)

| PRD Feature | v4 plan TCs | Status |
|---|---|---|
| Media Library | TC009–014 | ✅ Good |
| qBittorrent Settings | TC035–042 | ✅ Good (8 cases — over-weighted) |
| Connection Health Monitoring | TC043–048 | ✅ Good (states + history modal) |
| Manual Metadata Search | TC049–056 | ✅ Good |
| Pattern Learning | TC057–062 | ✅ Good |
| Scanner Settings | TC063–070 | ✅ Good |
| Subtitle Search | TC071–078 | ⚠️ Happy-path UI only (see §3) |
| **Downloads Monitor** | — | 🔴 **Zero** — route 256 LOC, P0 journey #3 |
| **Media Detail Panel** | only TC077 grazes it | 🔴 **Near-zero** — panel 40+ testids, P0 journey #1 |
| **Batch Subtitle Processing** | — | ✅ **Out of TestSprite scope** — see §3 (Go-tested) |
| Search (TMDB basic query) | — | 🟡 folded into Manual Search (v4 dropped basic) |
| Settings Hub nav | — | 🟡 partial |

---

## 3. Test-level routing (the core decision)

> **Principle:** prefer the lowest level that can assert the behavior. TestSprite is a **black-box browser journey** tool against the deployed NAS — it cannot inspect files, force backend state, or assert conversion internals. Push correctness *down*; keep TestSprite for *journeys a user actually drives in the browser*.

| Behavior | Right level | Why NOT TestSprite | Current state |
|---|---|---|---|
| **Batch Subtitle Processing** | Go integration | No journey-level UI trigger (library batch = delete/reparse/export only; `subtitle_batch_progress` SSE has no web consumer) | ✅ `subtitle/batch_test.go` — **28 test funcs** |
| **OpenCC `s2twp` conversion correctness** | Go unit/integration | Output is file content, not browser DOM | ✅ `subtitle/converter_test.go`, `engine_test.go` |
| **CN-content policy (skip conversion for mainland)** | Go integration | Depends on `production_countries`; assert on engine output | ✅ covered in `engine_test.go` (verify depth) |
| **Content-based language detection** | Go unit | Reads file bytes, not DOM | ✅ engine tests (verify depth) |
| **`.zh-Hant/.zh-Hans` extension naming** | Go integration | Filesystem assertion | ✅ engine/batch tests (verify depth) |
| **Connection degrade → recover *transition*** | Playwright (mock health API) | TestSprite can't flip backend health mid-test | ❌ gap — Playwright candidate |
| **Connection degraded *state* + graceful nav** | TestSprite ✅ | static state, browser-observable | 🔴 gap → **TC088** below |
| **Downloads / Media-Detail journeys** | TestSprite ✅ | pure browser journeys | 🔴 gap → **TC079–087** below |

**Net:** the only true *TestSprite* gaps are the journey flows in §4. The "subtitle engine differentiator" coverage that looked missing is actually present at the correct (Go) level — a depth-audit of those Go tests is a separate, cheaper task than new journey cases.

---

## 4. New TestSprite scenarios (P0 only) — ready to merge

IDs continue after `TC078` (avoids the permanent-gap IDs). All steps use **real `data-testid`s** verified in source so TestSprite's `generateCodeAndExecute` produces working selectors. Each lists a **seed-data prerequisite** (the deployed NAS must contain the right fixtures — see §6).

### 4.1 Downloads Monitor — P0 journey #3 (`/downloads`)

| ID | Title | Prio | Key testids / anchors | Seed prereq |
|---|---|---|---|---|
| **TC079** | Downloads page loads with filter tabs, list, and per-page count | High | filter tabs `all/downloading/paused/completed/seeding/error` (role=tab), `#download-list`, `每頁 N 筆` count | ≥1 download present |
| **TC080** | Filter by status tab updates visible list + count | High | click `downloading` tab → list reflects only downloading; `error`-tab empty-state text `沒有發生錯誤的任務` | mixed-status downloads |
| **TC081** | Expand a download item to view details | High | `download-item-{hash}` (first) → `download-details` visible | ≥1 download |
| **TC083** | Parse-failed download surfaces status badge + recovery actions | High | `download-parse-status-badge`, `parse-failed-actions` (re-parse / manual entry) | ≥1 parse-failed download |

*Deferred (NOT P0 — Medium):* `TC082` sort field/order changes ordering — defer per budget.

### 4.2 Media Detail Panel — P0 journey #1 (`/library` → poster → panel)

| ID | Title | Prio | Key testids / anchors | Seed prereq |
|---|---|---|---|---|
| **TC084** | Click poster opens detail panel with core metadata | High | `poster-card` → `media-detail-panel`, `detail-title`, `detail-year`, `detail-rating` | ≥1 matched movie |
| **TC085** | Detail panel renders tech badges + file info for an owned item | High | `tmdb-detail-owned-badge`/`detail-cta-buttons`, `TechBadgeGroup`, `file-info` (`file-info-name/size/path/status`) | ≥1 owned item w/ file |
| **TC086** | Metadata fallback states render with recovery CTA | High | `fallback-pending` (`pending-spinner`) OR `fallback-failed` (`fallback-failed-title`, `fallback-cta` / `cta-search-metadata` / `cta-manual-edit`) | ≥1 unmatched/failed item |

*Already covered:* `TC077` opens Subtitle Search from the detail panel — do not duplicate.
*Deferred (Medium):* `TC087` detail-panel menu (`detail-menu-trigger` → `menu-reparse/export/delete`) — overlaps library context-menu coverage.

### 4.3 Connection Health — P0 journey #4 (graceful-degradation slice)

| ID | Title | Prio | Key testids / anchors | Seed prereq |
|---|---|---|---|---|
| **TC088** | Degraded state shows banner AND core navigation stays usable | High | dashboard `degraded state indicator` visible → navigate `/library` → media grid still renders (graceful degradation) | health API reporting degraded |

> The full **degrade→recover transition** is routed to **Playwright** (mock the health endpoint) — TestSprite cannot flip backend state mid-run.

**New P0 total: 8 cases** (TC079, TC080, TC081, TC083, TC084, TC085, TC086, TC088).

---

## 5. Budget reconciliation (stay on Free 150)

Adding 8 cases takes the catalog 50 → 58 (≈290 credits ≈ 2.4 cron runs/rotation). To honor "P0 only, no plan upgrade," **demote 4 Low-priority cases** so net rotation stays ~flat:

| Demote (Low prio) | Rationale |
|---|---|
| `TC013` Export confirmation | Low-value happy-path; export covered structurally |
| `TC048` Indicator/modal stable after filter toggles | Pure UI-stability, Low |
| `TC061` Pattern delete cancel-if-confirm | Conditional/Low |
| `TC062` Pattern stats refresh | Low |

**Net: 50 − 4 + 8 = 54 cases** (~270 credits ≈ 2.25 runs/rotation). Demoted cases move to a `parked:` list in the queue (not deleted) — re-promote if the plan is upgraded.

---

## 6. Execution plan / next steps

1. **Seed data** (gating prerequisite — same blocker as the 2026-03 first run): the deployed NAS at `http://192.168.50.52:8088` must contain fixtures for the new cases. **`scripts/seed-test-data.sh` now exists** and seeds the DB-backed media fixtures via SQLite — an **owned matched movie** (TC084/085) + **pending/failed-metadata** items (TC086), idempotent and collision-safe. The two **non-DB** fixtures must be produced qBT-side (the script documents both): mixed-status incl. a **parse-failed** download (TC079-083) = add torrents to qBittorrent; **degraded** health (TC088) = point the qBittorrent host at an unreachable address.
2. **Wire into the catalog:** merge the 8 entries (see `testsprite-new-cases-2026-06.json`) into `testsprite_frontend_test_plan.json`; add them to `testsprite-queue.yaml` `queue:` and move the 4 demoted IDs to a `parked:` block. (Queue schema is Rule-20 `[@contract-v2]` — a shape change beyond appending entries would need a bump; appending queue entries does not.)
3. **Generate + execute:** let the monthly cron lazily generate, OR a one-shot local `npx @testsprite/testsprite-mcp@0.0.37 generateCodeAndExecute` for the 8 new IDs (cost ≈ 40 credits).
4. **Run the existing 50 to completion first** (Tier 0 follow-through) to surface real breakage before judging new cases.
5. **Separate cheaper task:** depth-audit the Go subtitle tests (`engine_test.go` / `converter_test.go` / `batch_test.go`) to confirm CN-policy + extension-naming + content-detection assertions exist — this is where the "differentiator" coverage lives, not TestSprite.
6. **Optional Playwright:** add a connection degrade→recover transition spec (mock health API) — out of TestSprite scope.

---

## Appendix — risk basis

P0 = (high user impact) × (revenue/trust-critical journey). Downloads (#3), Media Detail (#1), and Connection health (#4) are the three under-covered P0 journeys from the integration plan; Library browse (#5), QB settings (#6), and parse→match (#2) are already covered. Subtitle *engine* value is correctly tested below the journey layer (Go), so it carries no TestSprite risk debt.
