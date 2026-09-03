# Story 6.10a: 候選列身分 —— 封面、真實片長、未匹配標題（後端）

Status: ready-for-dev

## Story

As a BYOK NAS owner looking at the 產生字幕 consent list,
I want every row to carry a poster, the file's real duration and an honest title,
so that I can recognise what I am about to pay for — consent without recognition is not consent.

## Context — 這個 story 為什麼存在

`/impeccable critique` 2026-09-03（20/40，`.impeccable/critique/2026-09-03T15-07-46Z__apps-web-src-components-subtitle-consent.md`）P1「列身分崩塌」。正式環境截圖：2399 列全部灰方塊、全部「片長未知，以 45 分鐘估算」、多列標題是 `[bitsearch.to] Predator.Badlands.2025.2160p…`。

根因（已查證）：

| 症狀 | 根因 |
| --- | --- |
| 封面空白 | `GenerationCandidate`（`generation_candidates.go:93-118`）沒有 poster 欄位；FE 畫佔位 span |
| 片長全未知 | `candidateRow.runtimeMinutes()`（`:689`）只讀 TMDb `runtime`；ffprobe 在 `applyFFprobeTechInfo`（`enrichment_service.go:470`）量到的 `DurationSeconds`（`ffprobe_service.go:33`）**從未持久化、從未被消費** |
| 檔名當標題 | 未比對的 `movie.Title` 就是檔名（同 sub-6-7 的 prompt 側問題，這裡是 UI 側） |

**Consumed by:** sub-6-10b（前端）。

## Acceptance Criteria

1. **持久化容器時長。** migration 033：`movies` 與 `episodes` 加 `duration_seconds INTEGER NULL`（Rule 15：repo INSERT/UPDATE/SELECT/scan 全同步；`enriched_metadata_update.go` 的 tech-info 寫入加欄）。`applyFFprobeTechInfo` 把 `info.DurationSeconds` 寫入 movie；episodes 若無 ffprobe 路徑，則在候選分析的路線探測（本來就對每個 episode 跑 ffprobe 取軌道）同時擷取時長並回寫，快取語意沿用 `routeCachePlan`。

2. **估價改讀真實時長。** `runtimeMinutes()` 優先序：`duration_seconds/60`（容器）→ TMDb `runtime` → 45 分鐘 fallback；`RuntimeKnown=true` 於前兩者。additive 欄位 `runtime_source: "ffprobe"|"tmdb"|"fallback"`（sub-4-1 `[@contract-v1]` additive 不 bump，ack + Change Log）。

3. **封面。** `GenerationCandidate` additive `poster_path string`：電影用 `movies.poster_path`；集數用**影集**的 `series.poster_path`（`resolveSeriesTitle` 的 memo 擴成 `resolveSeriesMeta` 一次取 title + poster，仍是每影集一次查詢）。空字串＝無封面。

4. **未匹配標題誠實化。** additive `tmdb_matched bool` + `display_title`：已比對 → TMDb 標題；未比對 → 走既有檔名解析器（`internal/parser`）產出的乾淨標題（片名＋年份），失敗才退回檔名。`title` 既有欄位語意不動（舊 FE 不壞）。

5. **測試。** (a) migration up/down 與 SELECT/scan 同步（真 sqlite）；(b) `runtimeMinutes` 三級優先序表格；(c) 封面：電影自有、集數繼承影集、影集缺列時空字串且只 log 一次；(d) `display_title` 三種情況；(e) 候選分析 episode 時長回寫；(f) 全回歸。

## Tasks / Subtasks

- [ ] **Task 1 — migration 033 + repo 同步（AC: #1）**
- [ ] **Task 2 — ffprobe 時長寫入（movie 路徑 + episode 分析路徑）（AC: #1）**
- [ ] **Task 3 — 估價優先序 + `runtime_source`（AC: #2）**
- [ ] **Task 4 — `poster_path` / `tmdb_matched` / `display_title`（AC: #3, #4）+ Swagger + 契約 ack**
- [ ] **Task 5 — 測試（AC: #5）**

（後端 5 task → 跨端拆分，FE 為 sub-6-10b。）

## Dev Notes

- Rule 15 精準先例：bugfix-20-1（欄位存在但 SELECT/scan 沒載 → 永遠零值）。本 story 加的是同一類欄位，測試要含真 DB 讀回。
- Rule 24 superseded-mechanism：加了 `duration_seconds` 後，`unknownRuntimeMinutes=45` 只剩 fallback 角色——註解要改寫，不得留兩套語意。
- 與 sub-6-7 的關係：6-7 管 prompt 不吃檔名，本 story 管 UI 不顯示檔名；共用 `LooksLikeFilename`。

### Time-dependent visual coverage

- N/A — 純後端。

### References

- critique snapshot（上）；`apps/api/internal/services/generation_candidates.go:93-118,665-765`、`enrichment_service.go:466-535`、`ffprobe_service.go:28-45`、`repository/enriched_metadata_update.go:84-88`

## Dev Agent Record

### Agent Model Used

### Completion Notes List

### Discovery Triage

- （dev 填）

### File List
