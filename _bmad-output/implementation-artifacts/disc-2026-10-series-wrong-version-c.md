# disc-2026-10-series-wrong-version-c：影集資料夾改名或刪掉後，舊的那一筆影集會從媒體庫消失

Status: done

<!-- SM Bob create-story 2026-10-23；行號為 main `2dfc100c`。傘狀條目 disc-2026-10-series-wrong-version 的第三張。 -->

**Source:** 2026-10-23 查「將夜 60 集」時在正式機資料庫看到（`sqlite3 -readonly /mnt/cache/appdata/vido/vido.db`）：除了正在用的 `/media/tv/將夜`，還有一筆 `將夜`（`7e28b051…`，路徑 `/media/tv/將夜 (2026)`，2026-08-24 建立）。那個資料夾已經不在了（Alexyu 之前改過名），它底下 3 集都沒有檔案，可是 `is_removed = 0`，所以媒體庫會列出兩張將夜。Alexyu 選 A 時一起涵蓋。

## 為什麼會這樣

- 掃描結束時，電影找不到檔案會標成已移除（`detectRemovedFiles`）。
- 集數找不到檔案會清掉路徑（`detectRemovedEpisodeFiles`，`disc-2026-10-episode-rows-outlive-deleted-files`）。
- **影集本身沒有人管**：資料夾不見、集數全都沒檔案，那一筆影集還是照樣列在媒體庫。

## Acceptance Criteria

1. **標成已移除**：掃描結束時，一筆影集同時符合下面三點，就標 `is_removed = 1`（從媒體庫消失）：
   - 它的資料夾（`file_path`）在磁碟上不存在；
   - 它底下沒有任何一集還有檔案路徑；
   - 它的資料夾不在這次掃描「信不過」的根目錄底下，它的媒體庫這次也不是信不過的（掛載保護，跟電影、集數同一套）。
2. **資料夾還在就不動**：資料夾存在但裡面沒影片（例如還沒下載完）→ 不標。
3. **不搶寫**：標記時再確認一次「沒有任何一集有檔案」，掃描途中剛好有集數寫回來，就不標。只改 `is_removed` 和 `updated_at`。
4. **回來就恢復**：之後掃描又在同一個資料夾找到影片（例如改回原本的名字）→ 同一筆影集的 `is_removed` 清回 0，跟電影一樣。
5. **只動影集這一筆**：集數、季、TMDb 資料、字幕狀態、詞彙表都不刪。
6. **測試**：
   - 資料夾不在＋集數都沒檔案 → 標記；資料夾還在 → 不標；還有一集有檔案 → 不標；信不過的根目錄／媒體庫 → 不標。
   - 標記時有集數剛寫回 → 不標。
   - 已移除的影集，資料夾回來後再掃 → 恢復。
   - repository 的兩個查詢用真的 SQLite 測。

## 不要做的事

- 不合併兩筆影集、不把舊那一筆的集數搬到新的那一筆（舊那一筆的集數本來就沒檔案）。
- 不改「媒體庫只列沒被移除的影集」這條既有規則。
- 不做 -b（手動選片）。

## Tasks

- [x] Task 1 — `SeriesRepository.FindFoldersForRemovalCheck`／`MarkRemovedIfNoFiles`／`RestoreRemoved`（AC 1、3、5）
- [x] Task 2 — 掃描器 `detectRemovedSeries`，接在集數那一輪之後（AC 1、2）
- [x] Task 3 — `UpsertSeries` 找到已移除的影集時恢復（AC 4）
- [x] Task 4 — 測試、vet、staticcheck（AC 6）

## Dev Notes

- 掃描結束的移除流程：`scanner_service.go:440-455`；集數版本 `detectRemovedEpisodeFiles`（`:939`），用窄介面斷言（`episodeRemovalRepo`），不擴充寬介面和它的 mock。
- 電影的窄寫入：`repository/scan_state_update.go:75-90`（`MarkRemoved`／`RestoreRemoved`）。
- 影集找到既有那一筆：`media_ingest_service.go` 的 `UpsertSeries`（`FindByFilePath` 不濾已移除，`FindByTMDbID` 會濾）。
- 正式機部署後下一次掃描，`7e28b051…` 應該會被標記。

### Time-dependent visual coverage

- N/A — 只有後端。

## Dev Agent Record

### Agent Model Used

Claude Opus 5.5（claude-opus-5-5），2026-10-23。

### Completion Notes List

- repository：`FindFoldersForRemovalCheck`（沒被移除、有資料夾路徑、沒有任何一集有檔案）、`MarkRemovedIfNoFiles`（同一句 SQL 再檢查一次，只改 `is_removed`／`updated_at`）、`RestoreRemoved`（窄寫入）。
- 掃描器：`detectRemovedSeries` 接在集數那一輪之後，窄介面斷言（`seriesRemovalRepo`），掛載保護沿用 `underAnyRoot`＋信不過的媒體庫。標記的影集數只寫 log，不算進「移除的檔案數」（那是檔案）。
- 匯入：`UpsertSeries` 用資料夾找到已移除的影集時恢復（`seriesRestorer`）。
- 反向驗證：拿掉恢復那段 → `TestUpsertSeries_RestoresARemovedShowWhoseFolderIsBack` 紅。
- **正式機資料庫唯讀試算（2026-10-23）**：候選查詢在正式資料庫跑出 11 筆，11 筆的資料夾在 NAS 上都不存在 → 部署後下一次掃描會隱藏這 11 筆：`將夜 (2026)`（改名留下的舊那筆），以及夜魔俠、書卷一夢、The First Order、街頭餐廳鬥士、Overlord、雪中悍刀行、棋魂、Undercover Chef Korea、有本事換你來做啊、顯微鏡下的大明之絲絹案（`/mnt/user/data` 底下四層內都找不到這些資料夾，所有集數也都沒有檔案）。
- `go test ./internal/services/... ./internal/repository/... ./internal/handlers/... ./cmd/api/...` 全過；vet、staticcheck 乾淨。

### File List

- `apps/api/internal/repository/series_repository.go`
- `apps/api/internal/repository/scan_state_update.go`
- `apps/api/internal/repository/series_removed_folder_test.go`（新）
- `apps/api/internal/services/scanner_service.go`
- `apps/api/internal/services/media_ingest_service.go`
- `apps/api/internal/services/series_removed_folder_test.go`（新）
