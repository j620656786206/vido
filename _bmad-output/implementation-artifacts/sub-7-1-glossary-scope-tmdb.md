# Story 7.1: `show_glossary` 改綁 TMDb ID（scope 層）—— 詞彙表跨機器對得上（後端）

Status: done（Alexyu 2026-09-06 裁定；PR #395 squash 6a4358bf，main 全綠）— 原 review 註記：5 task 全數交付（migration 號碼是 **036**，story 寫「下一個空號」時 034/035 已被 sub-6-2／sub-6-10a 用掉）。

## Story

As a NAS owner whose glossary is the only asset that compounds,
I want my show glossaries keyed by a world-wide id instead of this machine's random id,
so that seeding, sharing and re-scanning all land on the same drawer.

## Context

eval-1「架構評估：`show_glossary` 從本機 `media_id` 改綁 TMDb ID」（含 A→B 銜接檢查與彈性檢查）—— **本 story 照該評估實作，不重新設計**。現況：`show_glossary.media_id TEXT`、unique `(media_id, term_src, language)`、`source` CHECK enum 三值、無 FK（migration 028）；key 由 `glossaryKeyFor`（`process_item.go:801`）決定；四個消費者吃 `mediaID`（`glossary_store.go:44`、`glossary_service.go`、`nfo_localizer_service.go:97`、`transcription_service.go:245`）；HTTP `/media/:id/glossary`（`glossary_handler.go:30`）。`TMDbID` 三個 model 都是 `NullInt64`。

## Acceptance Criteria

1. **migration（下一個空號，重建表）。** 新欄 `scope TEXT NOT NULL`（`tmdb:tv:<id>`／`tmdb:movie:<id>`／`local:<media_id>`）；unique `(scope, term_src COLLATE NOCASE, language)`；`media_id` 保留一版供稽核（下一 migration 移除，Rule 24 superseded-mechanism 註記）；**拿掉 `source` CHECK**，enum 改由 `models.GlossaryTerm.Validate` 驗證並擴成 `subtitle|metadata|manual|official_subtitle|community`；回填：series／movies 有 `tmdb_id` 者寫 `tmdb:*`，其餘 `local:`。`term_src` 寫入前 `TrimSpace`。

2. **`GlossaryScopeResolver`（services）。** `Resolve(ctx, mediaID) (scope string, err)`：查 movie／series（episode → series）的 `tmdb_id`；無效 → `local:<mediaID>`。**不快取**（比對可能晚於入庫）。四個消費者改先 resolve 再查 repo；repo 方法由 `ByMedia` 改為 `ByScope`（舊名保留一版委派）。

3. **local → tmdb 搬家。** 當 resolver 首次為某 media 解出 `tmdb:*` 且存在 `local:<id>` 列 → 同交易內 UPDATE scope（insert-if-absent 語意保留：既有 `tmdb:*` 同詞不覆寫）。

4. **HTTP／web 不動。** `/media/:id/glossary` 照舊；handler 內 resolve。`GlossaryRowV2` 的 `SOURCE_BADGE` 加 `official_subtitle`（「官方字幕」）與 `community`（「社群」）兩個徽章（既有 `字幕／中繼資料／手動` 保留）—— 這是 P1-10 的殘餘，UI 已有 source 徽章機制。

5. **測試。** (a) migration up：回填分流、NOCASE 去重（`Demogorgon`／`demogorgon` 合併時保留先者並 log）、CHECK 移除後五值可寫；(b) resolver 三種 media 與無效 id；(c) 搬家：local→tmdb 且不覆寫；(d) 四個消費者各一條 resolve 路徑測試；(e) segment cache `GlossaryVersion` 不受 key 影響（既有斷言）；(f) FE 徽章 spec。

## Tasks / Subtasks

- [x] **Task 1 — migration 036 + model enum + repo ByScope（AC: #1）** — `036_rebuild_show_glossary_with_scope.go`：新表 + `(scope, term_src COLLATE NOCASE, language)` unique、LEFT JOIN series/movies 一次算出 scope、Go 端逐列 trim + NOCASE 去重（先建立者留、被丟的 id 進 log）、`source` CHECK 拿掉、`media_id` 留一版；`models.GlossarySource*` 五值 + `GlossaryScope*` 輔助函式；repo `ListByScope`／`LookupByScope`／`ConfirmAllByScope`／`MigrateScope`，舊名三個留在 struct 上委派到 `local:` 抽屜
- [x] **Task 2 — resolver + 四消費者接線（AC: #2）** — `services.GlossaryScopeResolver`（series → movie → episode→series；不快取）；pipeline 的 `GlossaryStore` adapter、`TranscriptionService`、`NFOLocalizerService`、`GlossaryService` 全部先 resolve；main.go 一個 resolver 注入四處
- [x] **Task 3 — 搬家邏輯（AC: #3）** — `MigrateScope` 單一交易；resolver 解出 `tmdb:*` 時對 `local:<show id>`（與 `local:<episode id>`）各掃一次
- [x] **Task 4 — FE 徽章（AC: #4）** — `GlossarySource` 型別 + `SOURCE_BADGE` 加 官方字幕／社群
- [x] **Task 5 — 測試（AC: #5）** — (a) migration 6 個；(b)(c) resolver 7 個；(c) repo `MigrateScope` 2 個；(d) store／transcription／nfo／service 各有 resolve 路徑測試；(e) `GlossaryVersion` 既有斷言未動、全綠；(f) `GlossaryRowV2.spec` 來源表加兩列

（後端 4 task、前端 1 —— 不觸發拆分。）

## Dev Notes

- 評估原文（含「為什麼不做 `tmdb_id INTEGER` 欄」「唯一單行道是顆粒度」）在 eval-1 story 檔尾，dev 開工前整節讀完。
- Rule 15：SELECT／scan 同步；Rule 19：resolver 在 services，subtitle 透過既有 `GlossaryStore` port 注入 scope（port 簽名改吃 scope）。
- 與 sub-6-7 無衝突；與 sub-7-3／7-5／8-1／8-2 是前置。

### Time-dependent visual coverage

- N/A。

### References

- eval-1 story「架構評估」「A → B 銜接檢查」「彈性檢查」三節；migration 028；`glossary_repository.go`

## Dev Agent Record

### Agent Model Used

Claude Fable 5.1（claude-fable-5-1），2026-09-06

### Completion Notes List

- **正式資料乾跑**（NAS `vido.db` 用 `.backup` 取一致快照後在本機跑 033→036）：261 列 → 259 列；220 列進 `tmdb:*`（133 tv / 87 movie）、39 列留 `local:`（一部沒比對到的電影 `b7def516…`）；被合併的 2 組正好是《火星任務》（tmdb:movie:687163）的 `Astrophage`／`astrophage`（留「蟲洞菌」）與 `xenonite`／`Xenonite`（留「氙石」）——eval-1 拿 Demogorgon 舉的例在真資料裡真的存在。乾跑用的 test 檔沒進 repo。
- **裁量 1（偏離 Dev Notes 的「port 簽名改吃 scope」）— ⚖️ Alexyu 2026-09-06 裁定採 A（adapter 內解 scope、port 不動）**：`subtitle.GlossaryStore` 兩個方法**仍吃 pipeline 的 glossary key（本機 show id）**，改在 adapter 內先 resolve 再打 repo。理由：(a) pipeline 不需要知道 scope 是什麼；(b) harvest 寫入需要本機 id 填 `media_id` 稽核欄；(c) `subtitle` 不能 import `services`（反向已存在），所以 resolver 以 `subtitle.GlossaryScopeResolver` port 注入 adapter。效果等同 AC #2「四個消費者先 resolve 再查 repo」，pipeline 程式碼零改動。
- **裁量 2（AC #3 的「不覆寫」怎麼落地）**：搬家時，shared 抽屜已有同詞（NOCASE）的 local 列**留在 `local:` 不刪**，計入 `skipped` 並 log。不刪是因為那可能是使用者確認過的譯名；它從此不會被讀到（resolver 只回 tmdb），但資料還在，之後要做「分歧攤開」（B 路線）時有材料。
- **裁量 3**：resolver 對 episode id 除了掃 `local:<series id>`，也掃 `local:<episode id>`——sub-5-5 之前的舊路徑曾用 episode id 當 key 寫過。
- **裁量 4**：`media_id` 留 `NOT NULL`（每條寫入路徑都有本機 id）；`GlossaryTerm` 的 JSON 多一個 `scope` 欄（additive，web 端沒用）。
- **舊名委派的語意**：`ListByMedia`／`LookupByMedia`／`ConfirmAll` 留在 struct（不在 interface）上，委派到 `local:<id>`——意思是「沒 resolve 的呼叫者只看得到本機抽屜」；repo test 釘了這一點。
- Rule 15：`glossaryColumns` 同步含 `scope`；ON CONFLICT 目標必須逐字等於 unique index（含 `COLLATE NOCASE`），常數 `glossaryConflictTarget` 集中一處。
- 驗證：`go test ./...` 全綠、`nx lint api`（vet + staticcheck）乾淨；web：GlossaryRowV2 + services spec 207 passed、typecheck 0、eslint／prettier 綠。

### Discovery Triage

- ③ `backlog-sqlite-timestamps-carry-go-monotonic-suffix` — 乾跑時看到 `show_glossary.created_at` 存的是 Go `time.Time.String()`：`2026-09-03 01:31:37.706771431 +0800 CST m=+1411.298300644`（含單調時鐘讀數）。不只詞彙表：正式庫 `movies` 55/55、`series` 也是；`subtitle_runs` 因為有 `.UTC()` 而是乾淨的 `+0000 UTC`。後果：SQLite 的 `date()`／`datetime()` 讀不懂、跨時區排序只是字串序、`m=+` 是純垃圾。本 story 的 036 逐列**原樣複製**（不修、不破壞排序），搬家的 tie-break 用字串序仍成立。非本 story 範圍，立案。

### File List

- apps/api/internal/models/glossary.go
- apps/api/internal/database/migrations/036_rebuild_show_glossary_with_scope.go（+ _test）
- apps/api/internal/repository/glossary_repository.go（+ _test）
- apps/api/internal/services/glossary_scope_resolver.go（+ _test，新）
- apps/api/internal/services/glossary_service.go（+ _test，新）
- apps/api/internal/services/transcription_service.go（+ _test）
- apps/api/internal/services/nfo_localizer_service.go（+ _test）
- apps/api/internal/subtitle/glossary_store.go（+ _test）
- apps/api/cmd/api/main.go
- apps/web/src/services/glossaryService.ts
- apps/web/src/components/subtitle/GlossaryRowV2.tsx（+ .spec）
- _bmad-output/implementation-artifacts/sub-7-1-glossary-scope-tmdb.md、sprint-status.yaml
