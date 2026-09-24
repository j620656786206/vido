# Story bugfix：沒有片子在用的海報檔，會被定期清掉

Status: review

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As the person whose NAS keeps Vido's data folder,
I want poster files that no film or series points at any more to be removed on their own,
so that deleting films, re-uploading, or a crash halfway through an upload does not slowly fill `data/posters` with files nothing can show.

## Context

升級自 `disc-2026-09-poster-orphan-files`（P3，`bugfix-custom-posters-served-and-not-cache` 的 /ship CR L4 立案）。`poster-upload-b-poster-field`（PR #542）已修掉「上傳時留檔」的兩個主因（先查該類型的片子再寫檔；DB 寫入失敗會刪新檔、還原舊檔）。**剩下的是「片子被刪了，海報檔還在」**。

### 🔴 查到的事（main `66adf26b`；行號皆為現況）

1. **刪片子的路徑至少有 7 條，沒有一條會刪海報檔**：
   - `MovieService.Delete`（`services/movie_service.go:126`）、`SeriesService.Delete`（`series_service.go:149`）
   - `LibraryService.DeleteMovie`／`DeleteSeries`／`BatchDelete`（`library_service.go:478, 491, 757`，媒體庫的單筆與批次刪除）
   - 掃描時刪掉誤歸類的電影列（`scanner_service.go:560`）
   - 刪媒體庫並移除媒體（`media_library_service.go:207-216`，`DeleteByLibraryID` 電影＋影集）
   - 資料庫是**硬刪除**（`movie_repository.go:308` `DELETE FROM movies`）；另有軟移除旗標 `is_removed`（migration 019），被軟移除的列**仍在表裡**。
2. **`ImageProcessor.DeletePoster` 只被上傳失敗的還原路徑用**（`metadata_edit_service.go` `parkPoster`，PR #542）。
3. **現成的定期清理機制**：`CacheSweepScheduler`（`services/cache_sweep_scheduler.go`）開機先跑一次、之後每 45 分鐘（設定 `cache_sweep_interval_minutes`），`main.go:497-513` 用 `services.SweepFunc(name, fn)` 註冊額外的清理項目（`ai_cache`、`system_logs_retention` 就是這樣掛上去的）；每個項目有自己的 recover，一個壞掉不影響其他。
4. **海報檔名規則**：`handlers/poster_file_handler.go:16` 的白名單 `^[A-Za-z0-9_-]+\.jpg$`；上傳會寫 `<id>.jpg`＋`<id>-thumb.jpg`，DB 存 `/posters/<id>.jpg?v=<unix 毫秒>`（PR #542 起；更早的列沒有 `?v=`）；上傳中暫存的舊檔叫 `<name>.jpg.bak`。
5. **備份不含海報**（`backup_service.go:79-141` 只打包資料庫）→ 另立 `disc-2026-09-backup-excludes-uploaded-posters`，不在本張。

### ⚖️ 建單裁定（2026-09-25，SM；Alexyu 可在 review 推翻）

1. **用「定期對帳」而不是在 7 條刪除路徑各補一刀。** 對帳＝「資料夾裡只留資料庫有指到的海報」。理由：刪除路徑有 7 條、以後還會再多；每條各自記得刪檔，漏一條就漏檔，而且沒有測試抓得到「新的刪除路徑忘了刪檔」。對帳一個地方就涵蓋全部，連當機留下的 `.bak`、e2e 留下的檔也一起收（`feedback_architecture_prefer_long_solutions`）。代價：刪片子後檔案最多留 45 分鐘——那段時間它沒有任何列指到，頁面不會顯示它，無害。
2. **掛在既有的 `CacheSweepScheduler`**，不另開排程器。它不是「快取」，但那個排程器本來就是「定期維護」的載體（系統日誌保留也掛在上面）；名稱用 `poster_orphans`，**不會**出現在設定 → 快取管理頁（那頁讀的是 `CacheStatsService`，本張不碰）。
3. **寧可少刪，不可錯刪**：資料庫查詢失敗 → 一個都不刪；檔名不合白名單 → 不碰；剛寫入的檔（10 分鐘內）→ 不碰（上傳是「先寫檔、後寫 DB」，中間那一刻檔案確實沒人指到）；被軟移除（`is_removed`）的片子仍算「有指到」，還原後海報還在。

## Acceptance Criteria

1. **對帳規則（新 `services/poster_orphan_sweeper.go`，或同等位置）。**
   - 「有指到」的集合：`movies` 與 `series` **所有列**（含 `is_removed = 1`）中 `poster_path` 以 `/posters/` 開頭者，去掉 `?` 之後的查詢字串，取檔名。`<id>.jpg` 有指到 → `<id>.jpg` 與 `<id>-thumb.jpg` 都保留。
   - 只處理資料夾**第一層的一般檔案**（不進子資料夾、不跟符號連結）；檔名必須符合 `^[A-Za-z0-9_-]+\.jpg$` 或 `^[A-Za-z0-9_-]+\.jpg\.bak$`，其他檔名一律不碰（白名單規則與 `poster_file_handler.go` 共用同一個來源，不要複製一份正則）。
   - 沒人指到、且修改時間早於 **10 分鐘前**的 → 刪除；`.bak` 一律視為沒人指到（同樣受 10 分鐘保護）。
   - 查「有指到」集合失敗 → 回錯誤、**一個都不刪**。單一檔案刪除失敗 → 記 Warn、繼續下一個。
   - 回傳刪除數（`SweepFunc` 的 `(int64, error)` 形狀），並以 Info 記錄「刪了幾個、保留幾個」。
2. **SQL 放在 repository 層。** `MovieRepository`／`SeriesRepository` 各加一個只讀方法（例如 `ListUploadedPosterPaths(ctx) ([]string, error)`），查 `poster_path LIKE '/posters/%'`；**不加** `is_removed` 條件（Context 裁定 #3）。
3. **接上排程**：`main.go` 以 `services.SweepFunc("poster_orphans", sweeper.Sweep)` 加進 `cacheSweepExtra`；開機先跑一次、之後跟著既有間隔。`posterDir` 沿用 `main.go:204` 的同一個值。
4. **測試（Go）**——每一條裁定都要有一條會紅的測試：
   - 沒人指到且舊 → 刪；有人指到（含 `?v=` 與沒有 `?v=` 的舊寫法）→ 留，且其 `-thumb.jpg` 也留；
   - `is_removed = 1` 的片子的海報 → 留；
   - 10 分鐘內的新檔 → 留；舊的 `.bak` → 刪、新的 `.bak` → 留；
   - `notes.txt`、`x.png`、子資料夾、指向資料夾外的符號連結 → 都不碰；
   - 查詢失敗（mock）→ 回錯誤且資料夾不變；
   - repository 方法用真的 SQLite（`setupTestDB` 那一套）測：只回 `/posters/` 開頭的值，TMDb 路徑與絕對網址不回。
   - 時間用可注入的時鐘（`now func() time.Time`），不要 `time.Sleep`。
5. **不准回歸**：`cache_sweep_scheduler` 既有測試、`poster_file_handler` 測試、`custom-poster` e2e 全綠；設定 → 快取管理頁的類型與數字不變（`CacheStatsService` 不碰）。
6. **CI**：`pnpm nx test api`、`pnpm run lint:all` 綠；紅／守（Rule 16）與 mutation check（至少：拿掉 10 分鐘保護、查詢失敗時仍刪、`is_removed` 過濾、白名單放寬成任何 `.jpg` 以外）。

## Tasks / Subtasks

- [x] **Task 1 — repository：列出有被指到的上傳海報（AC: #2, #4）**＋SQLite 測試
- [x] **Task 2 — `PosterOrphanSweeper`（AC: #1, #4）**：對帳規則、白名單共用、可注入時鐘＋測試
- [x] **Task 3 — 接上 `CacheSweepScheduler`、收尾（AC: #3, #5, #6）**

## Dev Notes

### 這張的重點

- **一個地方管全部**：不要去 7 條刪除路徑補 `DeletePoster`。
- **錯刪比漏刪嚴重**：使用者上傳的海報沒有第二份（備份也不含），任何不確定都選「不刪」。

### 上游契約（Rule 20）

- 本張不定義也不消費任何 `[@contract-v*]`；沒有 API 變更。

### 不要做的事

- 不要在各刪除路徑呼叫 `DeletePoster`。
- 不要讓海報出現在快取管理頁、不要改 `CacheStatsService`／`CacheCleanupService`（PR #536 才把它們分開）。
- 不要把 `is_removed` 當成「可以刪」。
- 不要新增排程器或設定項。

### 已知陷阱

- **白名單要共用**：`poster_file_handler.go` 的 `posterFileName` 是 handlers 套件私有；把規則搬到 `internal/images`（例如 `images.IsPosterFileName`）讓兩邊都用，handler 的既有測試要照樣綠。
- **`?v=`**：DB 值可能是 `/posters/<id>.jpg?v=…` 或舊的 `/posters/<id>.jpg`，兩種都要算「有指到」。
- **符號連結**：用 `os.ReadDir` ＋ `DirEntry.Type().IsRegular()`，不要 `os.Stat`（會跟著連結）。
- **CacheSweepScheduler 的 recover**：每個項目各自 recover；sweeper 本身不要 panic 也不要吞掉查詢錯誤（要回錯誤讓排程器記下來）。

### Source tree

```
apps/api/internal/repository/movie_repository.go、series_repository.go（+tests）   ← Task 1
apps/api/internal/images/（共用檔名規則）、internal/handlers/poster_file_handler.go  ← Task 2
apps/api/internal/services/poster_orphan_sweeper.go（新，+test）                   ← Task 2
apps/api/cmd/api/main.go                                                          ← Task 3
```

### Cross-Stack Split Check

後端 3、前端 0 → 不拆。

### Time-dependent visual coverage

- N/A — 沒有前端；後端的時間判斷用注入時鐘測。

### References

- [Source: `apps/api/internal/services/cache_sweep_scheduler.go:32-60, 94-102, 171-200`；`cmd/api/main.go:204, 497-513`]
- [Source: `apps/api/internal/services/movie_service.go:126`、`series_service.go:149`、`library_service.go:478, 491, 757`、`scanner_service.go:560`、`media_library_service.go:207-216`；`repository/movie_repository.go:308`；migration `019_add_is_removed_field.go`]
- [Source: `apps/api/internal/handlers/poster_file_handler.go:12-16`；`internal/services/metadata_edit_service.go`（`parkPoster`、`.bak`）；`internal/images/processor.go`（`DeletePoster`）]
- [Source: `sprint-status.yaml` → `disc-2026-09-poster-orphan-files`]

## Dev Agent Record

### Agent Model Used

### Debug Log References

### Completion Notes List

- Ultimate context engine analysis completed - comprehensive developer guide created
- **Task 1（偏離 AC #2 的寫法，意圖不變）**：沒有在 `MovieRepository`／`SeriesRepository` 各加一個方法，改成新的 `repository/poster_reference_repository.go`（一條 `UNION ALL` 查兩張表）。理由：兩個 repo 在 `Registry` 裡是以介面型別存放，加方法就得同步改所有實作那兩個介面的 mock；而這個問題本來就是「兩張表合起來」問的，放一處更不會漏表。SQL 仍在 repository 層；**不加** `is_removed` 條件。測試用真的 migration 鏈（照 `glossary_repository_test` 的寫法）：只回 `/posters/` 開頭的值（含軟移除的列），TMDb 路徑、絕對網址、空值不回；查詢失敗回錯誤。
- **Task 2**：檔名規則搬到 `images.IsPosterFileName`（`internal/images/filename.go`），`poster_file_handler.go` 改用它——「可以被服務的」與「可以被刪的」永遠同一套規則；handler 既有測試全綠。`services/poster_orphan_sweeper.go`：有指到的集合去掉 `?` 之後取檔名，`<id>.jpg` 有指到則連 `-thumb.jpg` 一起保留；只看第一層、`DirEntry.Type().IsRegular()`（不進資料夾、不跟符號連結）；`.bak` 一律可刪；10 分鐘保護；查詢失敗回錯誤不刪；資料夾不存在回 0；單檔刪除失敗 Warn 後繼續。時鐘可注入（`now`），測試不 sleep。
- **Task 3**：`main.go` 以 `SweepFunc("poster_orphans", …)` 掛進 `cacheSweepExtra`，`posterDir` 同一個值。**本機冒煙**：在本機開發用的 `apps/api/vido-data/posters` 放一個舊檔後啟動 API → 開機那一輪記錄 `Poster orphan sweep removed=93 kept=0`（那 93 個是先前 e2e 跑完留下的測試海報，本機資料庫已無列指到；正式環境不受本機動作影響），放進去的檔也被清掉。冒煙用的 API（port 8081）與 nx daemon 已關閉。
- 🔗 **AC Drift: NONE**（檢查 `grep -rn "posters\|DeletePoster\|poster_file_handler" _bmad-output/implementation-artifacts/*.md`：`bugfix-custom-posters-served-and-not-cache` 的白名單契約不變——只是搬到共用位置；`poster-upload-b` 的 `.bak` 行為不變，本張只多了「久了會被清」）。
- 📎 **Contract Stamps: NONE**（無 API 變更、無 `[@contract-v*]`）。
- 🎭 **A11y Pre-Flight: N/A（100% backend — no apps/web/ files touched）**。
- 🎨 **UX Verification: SKIPPED — no UI changes in this story**。
- **測試**：`nx test api` 綠、`lint:all`（0 error）綠；新增 Go 測試 repository 2、images 1、sweeper 6。
- **Mutation 7／7 紅**：拿掉 10 分鐘保護、查詢失敗仍照刪、repo 加上 `is_removed = 0`、拿掉白名單、改成會跟符號連結、縮圖不一起保留、不去掉 `?v=`。（第一輪有 3 條因改壞後編譯不過而「紅」，不算證據，改成可編譯的改壞方式重跑後才算。）

### 🔍 /ship Adversarial Review（2026-09-25）

0 HIGH／2 MEDIUM，全修：
- **M1** 上傳時把舊海報改名成 `.bak`，但改名會保留**舊的**修改時間 → 對帳眼中它是「舊檔」，10 分鐘保護罩不住；若上傳剛好卡在對帳那一刻，`.bak` 可能被清掉、之後 DB 寫入失敗就還原不了 → `parkPoster` 改名後把 `.bak` 的時間蓋成現在。測試 +1。
- **M2** 使用者在「改用圖片網址」貼了本站自己的海報網址（`http://nas:8080/api/v1/posters/<id>.jpg`）→ 原查詢只認 `/posters/` 開頭，那個檔會被當成沒人用 → 查詢放寬成 `%/posters/%`、取最後一個 `/posters/` 之後的檔名。寧可多留：外部網址剛好含 `/posters/x.jpg` 頂多讓一個同名的本地檔多留一陣子。測試 +2 斷言。

### Discovery Triage

- N/A — no out-of-scope work discovered（`disc-2026-09-backup-excludes-uploaded-posters` 已於建單時立案）。

### File List

- `apps/api/internal/repository/poster_reference_repository.go`（新，+test）
- `apps/api/internal/images/filename.go`（新，+test）
- `apps/api/internal/handlers/poster_file_handler.go`
- `apps/api/internal/services/poster_orphan_sweeper.go`（新，+test）
- `apps/api/cmd/api/main.go`
- `_bmad-output/implementation-artifacts/sprint-status.yaml`

## Change Log

| Date | Change |
| --- | --- |
| 2026-09-25 | 建單（SM）：由 `disc-2026-09-poster-orphan-files` 升級；裁定用定期對帳而非逐條刪除路徑補刀；另立 `disc-2026-09-backup-excludes-uploaded-posters` |
| 2026-09-25 | 實作完成：`PosterReferenceRepository`、共用檔名規則、`PosterOrphanSweeper`、掛進 `CacheSweepScheduler`；本機冒煙清掉 93 個 e2e 殘檔；狀態 review |
| 2026-09-25 | /ship CR：`.bak` 改名後蓋新時間、認得貼上的本站海報網址 |
