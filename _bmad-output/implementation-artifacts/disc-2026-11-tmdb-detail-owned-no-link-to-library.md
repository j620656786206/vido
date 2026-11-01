# Story disc-2026-11-tmdb-detail-owned-no-link-to-library：從搜尋點進已入庫的影集，直接到媒體庫版詳情頁

Status: done

**Source:** sprint-status 同名待辦（P2，PR #946 建單）。⚖️ Alexyu 2026-11-01「處理剩餘的4項代辦」。

## 問題

2026-11-01 Alexyu 在外面用手機從搜尋點進《基地》，看不到季和集數，也就沒有「生成字幕」。搜尋結果的連結帶的是 TMDb 的數字 id，`routes/media/$type.$id.tsx` 因此開了 `TMDbDetailV2`（只有簡介、觀看平台、推薦）。這頁知道《基地》已入庫（顯示「已入庫」），卻沒有帶去媒體庫版 `LocalDetailV2`。

## 設計決定（Sally）

待辦寫了兩條路：自動導過去，或加一個連結。選**自動導過去**：

- 媒體庫版詳情頁有 TMDb 版的全部區塊（簡介、演員、預告、觀看平台、推薦），另外還有季、集數、字幕。對已入庫的片來說，TMDb 版沒有任何東西是媒體庫版沒有的，多一個按鈕只是多按一下。
- 用「取代」（replace）導過去，按返回不會回到 TMDb 版再被導一次。
- 查詢中顯示骨架畫面，不會先閃一下 TMDb 版。
- 查詢失敗就留在 TMDb 版，跟以前一樣。

## Acceptance Criteria

1. [x] 新 API `GET /api/v1/media/library-id?type=movie|tv&tmdb_id=N` → `{"id": "<uuid>"}`，沒入庫回 `{"id": null}`（200）。只看沒被移除的資料，電影和影集分開查（TMDb 兩邊編號各自獨立）。type 或 tmdb_id 不對回 400。
2. [x] TMDb 版詳情頁查到 id 就 `navigate({ to: '/media/$type/$id', replace: true })` 到媒體庫版；查詢中顯示骨架。
3. [x] 沒入庫或查詢失敗，TMDb 版照舊顯示。
4. [x] 測試：repository（有入庫／已移除／沒有／無效 id）、service（依 type 分表、沒接上時回空）、handler（200／null／400／500）、前端（導過去、查詢中不閃、沒入庫留下）。

## Dev Agent Record

- BE：`MovieRepository`／`SeriesRepository` 新增 `FindLibraryIDByTMDbID`（同一部掃進來兩次取最早的）。`AvailabilityService.LibraryID` 透過一個只有這個方法的小 interface 取用，不動大的 repository interface 和它們一堆測試替身；編譯期斷言兩個正式 repository 都有實作。`AvailabilityHandler` 掛 `GET /media/library-id`。
- FE：`availabilityService.libraryId`、`hooks/useLibraryId.ts`、`TMDbDetailV2` 查到就取代導頁。
- 驗證：`go test ./...`、`go vet`、`staticcheck`（services／handlers／repository）乾淨；`nx run web:test`（4,993 個）、`web:typecheck`、`nx lint web`（0 errors）通過。本機用正式機資料副本起新後端：`type=tv&tmdb_id=93740` → 《基地》的 id；`type=movie&tmdb_id=93740` → null（同編號不同類型不會串錯）；壞參數 → 400。

## Change Log

- 2026-11-01：建立＋實作。
