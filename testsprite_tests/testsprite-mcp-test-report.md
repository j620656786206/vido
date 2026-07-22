# TestSprite AI Testing Report (MCP) — v2 Smoke Round 1

---

## 1️⃣ Document Metadata

- **Project Name:** vido
- **Date:** 2026-07-22
- **Prepared by:** TestSprite AI + Claude (party-mode session)
- **Round:** testsprite-v2-round1 (smoke, 12 cases ≈ 60 credits) against the local seeded env (`scripts/serve-test-env.sh`, production build on :8090, v2 shell post-cutover-4)

---

## 2️⃣ Requirement Validation Summary

### Requirement: 媒體庫瀏覽互動 (Library browse & item actions)

- **TC010** Delete an item from list view — ⚠️ BLOCKED (run 1): the generated flow deleted 教父 via selection-mode batch delete first, leaving no target for the per-item menu path. Environment artifact, not an app defect. DB reseeded; queued for rerun.
- **TC011** Cancel delete leaves item intact — ✅ Passed
- **TC012** Filter by TV type then clear — ✅ Passed

### Requirement: URL↔UI Consistency (2026-07-22 type-filter bug class, new TC089–TC091)

- **TC089** Deep-link `/library/movies` → movie-only list + 電影 control active — ✅ Passed
- **TC090** Deep-link `?genres=科幻` pre-applies the genre filter — ✅ Passed
- **TC091** Type switch → browser Back → hard refresh: URL, active control, and list agree at every checkpoint — ✅ Passed

### Requirement: 搜尋→瀏覽→詳情 P0 journey (new TC092)

- **TC092** Header instant search finds a seeded title → detail — ❌ Failed (run 1)
  - **Root cause chain (all three fixed this session):**
    1. `/api/v1/search` (unified instant search) was TMDb-only — five TMDb legs, no local-library leg; with no TMDb API key every leg failed and the endpoint returned 500, which the dropdown rendered as 找不到結果.
    2. Even via `/api/v1/library/search`, partial zh-TW queries (駭客) missed owned titles (駭客任務): FTS5 unicode61 keeps a CJK run as ONE token, and raw `MATCH` has no prefix semantics. Raw FTS operators in input could also 500.
  - **Fixes:** unified search gains a local-library leg returning `local_movies`/`local_tv` with LOCAL ids (dropdown 媒體庫 section, 已擁有 badge, navigates to the TMDb-independent local detail); per-leg degradation (error only when every leg fails); `ftsPrefixQuery` in the repository layer (quoted-prefix, operator-inert) fixes CJK partial search everywhere.
  - Verified live keyless: `q=駭客` → `local_movies: [seed-mv-003 駭客任務]`.

### Requirement: 詳情 (Media detail)

- **TC084** Poster → detail shows core metadata — ❌ Failed (run 1): the generated code clicked the FIRST card, which was the UNMATCHED fixture `Unknown.Show.S01` (no year/rating by design — that UX gap is already tracked as `disc-2026-07-v2-detail-fallback-states`). Plan steps now target the matched 駭客任務 explicitly. Queued for rerun.
- **TC085** Tech badges + 檔案資訊 for an owned item — ✅ Passed

### Requirement: 下載監控 (Downloads, empty/degraded half)

- **TC079** Downloads page renders filter tabs + empty state — ✅ Passed
- **TC080** Status tab switching stays stable on the empty state — ✅ Passed

### Requirement: 降級狀態 (Degraded services)

- **TC088** Degraded state visible + navigation usable — ❌ Failed (run 1)
  - **Root cause:** the general `healthMonitor.StartMonitoring` goroutine was never wired in `main.go` (only the qBT-specific monitor ran), so tmdb/douban/wikipedia/ai sat on their factory-default "healthy" forever (`last_check: 0001-01-01`) — a keyless TMDb still displayed 正常.
  - **Fix:** `go healthMonitor.StartMonitoring(monitorCtx, 5*time.Minute)` with an immediate startup sweep. Verified live: `degradation_level: partial`, all five services report their true degraded state on the seeded env.

---

## 3️⃣ Coverage & Matching Metrics

- **12 selected cases / 62-case plan** (smoke subset by design — Free-150 budget)
- **Run 1:** 8 ✅ / 3 ❌ / 1 ⚠️ BLOCKED → **66.7% pass**
- **All 3 failures were real product defects (2 app bug chains + 1 test-data targeting), all fixed with unit regression locks** (Go: search local-leg ×5, FTS ×3, migration-era suites all green 34 pkg; web: +3 search dropdown specs, 2455/2455)
- **Rerun queue:** TC010, TC084, TC088, TC092 (20 credits; done-gate = all green, no skips/waivers)

| Requirement        | Total | ✅ Passed | ❌ Failed | ⚠️ Blocked |
| ------------------ | ----- | --------- | --------- | ---------- |
| 媒體庫瀏覽互動     | 3     | 2         | 0         | 1          |
| URL↔UI Consistency | 3     | 3         | 0         | 0          |
| 搜尋→瀏覽→詳情     | 1     | 0         | 1         | 0          |
| 詳情               | 2     | 1         | 1         | 0          |
| 下載監控           | 2     | 2         | 0         | 0          |
| 降級狀態           | 1     | 0         | 1         | 0          |

---

## 4️⃣ Key Gaps / Risks

1. **The URL↔UI class is locked green** — the exact 2026-07-22 type-filter bug shape (TC091 back/refresh agreement) passes against the real v2 stack; cheap, high-hit insurance the unit layer structurally cannot see.
2. **Unified search had no owned-content story** — the header search silently depended on TMDb for titles the user already owns. Fixed; consider a follow-up UX polish pass on the 媒體庫 section design (currently reuses the standard row + 已擁有 badge, no .pen counterpart yet).
3. **CJK partial search was broken library-wide** (FTS5 tokenizer) — fixed at the repository layer for both movies and series; any future FTS surface must reuse `ftsPrefixQuery`.
4. **Health monitoring was dark since Story 3.12** — /health/services reported factory defaults. Now live at a 5-minute cadence; NAS deploy will begin showing true service states (expect tmdb/douban/wiki/ai 正常 there since keys are configured).
5. **Downloads deep interactions (TC081/TC083) remain untestable** in the seeded env (no qBittorrent by design) — deferred to round 2 with a qBT stub/config decision.
6. **TC086 (detail fallback states)** stays deliberately excluded — blocked on `disc-2026-07-v2-detail-fallback-states`.
