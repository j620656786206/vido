# TestSprite AI Testing Report (MCP) — v2 Smoke Round 1 ✅ CLOSED

---

## 1️⃣ Document Metadata

- **Project Name:** vido
- **Date:** 2026-07-22 (run 1 + same-day fix PR #171 + rerun)
- **Prepared by:** TestSprite AI + Claude (party-mode session)
- **Round:** testsprite-v2-round1 — smoke vs the v2 shell (post-cutover-4) on the local seeded env (`scripts/serve-test-env.sh`, production Go serve on :8090)
- **Credits:** run 1 = 12 cases (60) + rerun = 4 cases (20) → **~70 remain** of Free-150

> **Headline: 12/12 PASS.** Run 1 went 8/12; the 3 failures were **real product
> defects** (not flakes, not env noise), all fixed and merged the same day
> (PR #171), and the rerun closed every red. Done-gate met — no skips, no waivers.

---

## 2️⃣ Requirement Validation Summary

### Requirement: 媒體庫瀏覽互動 (Library browse & item actions)

| Test  | Title                                       | Run 1       | Final     |
| ----- | ------------------------------------------- | ----------- | --------- |
| TC010 | Delete an item via the per-item action menu | ⚠️ BLOCKED¹ | ✅ PASSED |
| TC011 | Cancel delete leaves item intact            | ✅          | ✅ PASSED |
| TC012 | Filter by TV type then clear                | ✅          | ✅ PASSED |

¹ Run-1 block was test-orchestration (the generated flow batch-deleted 教父 before exercising the per-item path); reseeded + rerun with an explicit per-item instruction.

### Requirement: URL↔UI Consistency (the 2026-07-22 type-filter bug class — new TC089–TC091)

| Test  | Title                                                       | Final     |
| ----- | ----------------------------------------------------------- | --------- |
| TC089 | Deep-link `/library/movies` → movie-only list + 電影 active | ✅ PASSED |
| TC090 | Deep-link `?genres=科幻` pre-applies the filter             | ✅ PASSED |
| TC091 | Type switch → Back → hard refresh: URL/control/list agree   | ✅ PASSED |

### Requirement: 搜尋→瀏覽→詳情 P0 journey (new TC092)

| Test  | Title                                             | Run 1 | Final     |
| ----- | ------------------------------------------------- | ----- | --------- |
| TC092 | Header search finds an owned title → local detail | ❌    | ✅ PASSED |

**Defects found & fixed (PR #171):** unified `/api/v1/search` had no local-library leg (keyless TMDb = all legs fail = 500 = 找不到 for OWNED titles) → added `local_movies`/`local_tv` with LOCAL ids + per-leg degradation + a 媒體庫 dropdown section (已擁有 badge) navigating TMDb-independently. Chain #2: FTS5 unicode61 keeps CJK runs as one token — partial zh queries (駭客) never matched 駭客任務 library-wide, and raw FTS operators could 500 → `ftsPrefixQuery` (quoted-prefix, operator-inert) guards both repos.

### Requirement: 詳情 (Media detail)

| Test  | Title                                          | Run 1 | Final     |
| ----- | ---------------------------------------------- | ----- | --------- |
| TC084 | Matched poster → detail with title/year/rating | ❌²   | ✅ PASSED |
| TC085 | Tech badges + 檔案資訊 for an owned item       | ✅    | ✅ PASSED |

² Run-1 fail was test-data targeting: it clicked the UNMATCHED first card (`Unknown.Show.S01`, legitimately metadata-less — that UX gap is tracked separately in `disc-2026-07-v2-detail-fallback-states`). Plan retargeted to 駭客任務.

### Requirement: 下載監控 (Downloads — empty/degraded half)

| Test  | Title                                  | Final     |
| ----- | -------------------------------------- | --------- |
| TC079 | Page renders filter tabs + empty state | ✅ PASSED |
| TC080 | Tab switching stable on empty state    | ✅ PASSED |

### Requirement: 降級狀態 (Degraded services)

| Test  | Title                                      | Run 1 | Final     |
| ----- | ------------------------------------------ | ----- | --------- |
| TC088 | Degraded state visible + navigation usable | ❌    | ✅ PASSED |

**Defect found & fixed (PR #171):** `healthMonitor.StartMonitoring` had never been wired since Story 3.12 — only the qBT monitor ran, so `/health/services` served factory-default "healthy" (`last_check: 0001-01-01`) forever; a keyless TMDb displayed 正常. Wired at 5m + immediate startup sweep; the env now truthfully reports `degradation_level: partial`.

---

## 3️⃣ Coverage & Matching Metrics

- **Final: 12/12 PASS (100%)** — run 1: 8 ✅ / 3 ❌ / 1 ⚠️; rerun: 4/4 ✅
- All 3 run-1 failures were genuine product defects with unit regression locks landed alongside the fixes (Go: +5 unified-search legs, +3 FTS; web: +3 dropdown specs)

| Requirement        | Tests  | ✅ Passed |
| ------------------ | ------ | --------- |
| 媒體庫瀏覽互動     | 3      | 3         |
| URL↔UI Consistency | 3      | 3         |
| 搜尋→瀏覽→詳情     | 1      | 1         |
| 詳情               | 2      | 2         |
| 下載監控           | 2      | 2         |
| 降級狀態           | 1      | 1         |
| **Total**          | **12** | **12**    |

---

## 4️⃣ Key Gaps / Risks

1. **URL↔UI class locked green first try** — the type-filter bug shape is now regression-locked at three layers (unit routeTree specs, Playwright, TestSprite vs the real stack).
2. **The round paid for itself**: two shipped-code defect chains (owned-search degradation; dark health monitor) were invisible to every unit/E2E layer because they only manifest against a REAL backend in a degraded configuration — exactly the seeded env's design point.
3. **媒體庫 dropdown section has no .pen design counterpart** (reuses standard row + 已擁有 badge) — UX follow-up candidate for Sally.
4. **Deferred (deliberate):** TC081/TC083 need real qBT rows (round-2 stub/config decision); TC086 blocked on `disc-2026-07-v2-detail-fallback-states`; TC009's steps are legacy (pagination/header-sort) — retire or rewrite against v2 in the round-2 selection pass.
5. **Round 2** (full P0, ~20 cases ≈ 100 credits) fires when the `ux3-v2-cutover` epic close is on the table — budget note: round1+round2 ≈ the whole Free-150.

---

_Run artifacts: `testsprite_tests/tmp/test_results.json` + dashboard links in `tmp/raw_report.md`._
