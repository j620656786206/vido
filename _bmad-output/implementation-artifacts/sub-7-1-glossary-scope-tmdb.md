# Story 7.1: `show_glossary` 改綁 TMDb ID（scope 層）—— 詞彙表跨機器對得上（後端）

Status: ready-for-dev

## Story

As a NAS owner whose glossary is the only asset that compounds,
I want my show glossaries keyed by a world-wide id instead of this machine's random id,
so that seeding, sharing and re-scanning all land on the same drawer.

## Context

eval-1「架構評估：`show_glossary` 從本機 `media_id` 改綁 TMDb ID」（含 A→B 銜接檢查與彈性檢查）—— **本 story 照該評估實作，不重新設計**。現況：`show_glossary.media_id TEXT`、unique `(media_id, term_src, language)`、`source` CHECK enum 三值、無 FK（migration 028）；key 由 `glossaryKeyFor`（`process_item.go:801`）決定；四個消費者吃 `mediaID`（`glossary_store.go:44`、`glossary_service.go`、`nfo_localizer_service.go:97`、`transcription_service.go:245`）；HTTP `/media/:id/glossary`（`glossary_handler.go:30`）。`TMDbID` 三個 model 都是 `NullInt64`。

## Acceptance Criteria

1. **migration 034（重建表）。** 新欄 `scope TEXT NOT NULL`（`tmdb:tv:<id>`／`tmdb:movie:<id>`／`local:<media_id>`）；unique `(scope, term_src COLLATE NOCASE, language)`；`media_id` 保留一版供稽核（下一 migration 移除，Rule 24 superseded-mechanism 註記）；**拿掉 `source` CHECK**，enum 改由 `models.GlossaryTerm.Validate` 驗證並擴成 `subtitle|metadata|manual|official_subtitle|community`；回填：series／movies 有 `tmdb_id` 者寫 `tmdb:*`，其餘 `local:`。`term_src` 寫入前 `TrimSpace`。

2. **`GlossaryScopeResolver`（services）。** `Resolve(ctx, mediaID) (scope string, err)`：查 movie／series（episode → series）的 `tmdb_id`；無效 → `local:<mediaID>`。**不快取**（比對可能晚於入庫）。四個消費者改先 resolve 再查 repo；repo 方法由 `ByMedia` 改為 `ByScope`（舊名保留一版委派）。

3. **local → tmdb 搬家。** 當 resolver 首次為某 media 解出 `tmdb:*` 且存在 `local:<id>` 列 → 同交易內 UPDATE scope（insert-if-absent 語意保留：既有 `tmdb:*` 同詞不覆寫）。

4. **HTTP／web 不動。** `/media/:id/glossary` 照舊；handler 內 resolve。`GlossaryRowV2` 的 `SOURCE_BADGE` 加 `official_subtitle`（「官方字幕」）與 `community`（「社群」）兩個徽章（既有 `字幕／中繼資料／手動` 保留）—— 這是 P1-10 的殘餘，UI 已有 source 徽章機制。

5. **測試。** (a) migration up：回填分流、NOCASE 去重（`Demogorgon`／`demogorgon` 合併時保留先者並 log）、CHECK 移除後五值可寫；(b) resolver 三種 media 與無效 id；(c) 搬家：local→tmdb 且不覆寫；(d) 四個消費者各一條 resolve 路徑測試；(e) segment cache `GlossaryVersion` 不受 key 影響（既有斷言）；(f) FE 徽章 spec。

## Tasks / Subtasks

- [ ] **Task 1 — migration 034 + model enum + repo ByScope（AC: #1）**
- [ ] **Task 2 — resolver + 四消費者接線（AC: #2）**
- [ ] **Task 3 — 搬家邏輯（AC: #3）**
- [ ] **Task 4 — FE 徽章（AC: #4）**
- [ ] **Task 5 — 測試（AC: #5）**

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

### Completion Notes List

### Discovery Triage

- （dev 填）

### File List
