# Disc：影片從 NAS 刪掉後，那一集要變回「沒有本地檔」

Status: done

**Source:** filed 2026-10-08 by `disc-2026-10-subtitle-filter-series-phase-2`（CR M1，Rule 24 lane ③）。⚖️ 2026-10-08 party mode（Winston／Amelia／John〔兼 founder-pm 角度〕／Murat 一致推薦）→ **Alexyu 裁定 A：清掉檔案路徑，不加 is_removed**。

## Story

身為片庫的使用者，
我要刪掉 NAS 上某一集的影片後，Vido 也知道那一集已經沒有檔案，
這樣季清單不會說「有本地檔」，影集「有沒有中文字幕」也不會把已經不存在的那集算進去。

## 裁定理由（party mode 摘要）

- 全程式「這一集有沒有本地檔」本來就看 `file_path` 空不空（季清單、影集中文字幕彙總、生成字幕、NFO 在地化、單集轉文字）。清掉路徑＝回到既有的「知道有這集、沒有檔」，一行查詢都不用改；加 `is_removed` 每處都要加過濾，漏一處就對不存在的檔案按生成。
- 掃描以「劇＋季＋集」對回同一列，檔案放回來（甚至改名）下次掃描自動補回路徑，不需要另寫恢復邏輯。
- 只有一個使用者、沒有任何地方需要「被刪過的集數」紀錄；真要時再加。

## Acceptance Criteria

1. 掃描結束時，除了電影，也檢查每一集有路徑的集數：檔案確定不存在（`os.IsNotExist`）→ 清掉該集的 `file_path`、`subtitle_tracks`、`subtitle_tracks_file_sig`。
2. **掛載保護沿用電影**：這輪掃描讀不到、讀到一半失敗、或整個變空的資料夾底下的集數一律不動（以路徑判斷，也以所屬影集的媒體庫判斷）。權限等其他讀取錯誤也不動。
3. 清除是窄寫入，只在路徑仍是掃描讀到的那個值時生效（`WHERE file_path = ?`）：同時間被重新掃進新路徑的集數不會被清掉。
4. 字幕狀態欄（`subtitle_status`／`subtitle_path`…）不動——那是別的流程的資料，沒有路徑的集數那些流程本來就跳過。
5. 檔案放回原處後，下一次掃描把路徑補回同一列（既有的 Upsert 行為），測試釘住。
6. 清掉後：影集「有沒有中文字幕」彙總不再算那一集；季清單那一集變成「沒有本地檔」（兩者原本就看路徑，測試釘住彙總）。
7. 掃描結果的「移除檔案數」把清掉的集數也算進去；log 分別寫電影與集數的數字。
8. 檢查：`go build／vet`、`go test ./...`、`staticcheck` 全綠。

## Tasks / Subtasks

- [x] T1（AC #1、#3）：`EpisodeRepository.FindFilesForRemovalCheck`（id、路徑、所屬影集的媒體庫）、`ClearMissingFile(id, path)`＋測試。
- [x] T2（AC #1、#2、#7）：`ScannerService.detectRemovedEpisodeFiles`，接在 `detectRemovedFiles` 之後＋測試。
- [x] T3（AC #5、#6）：恢復與彙總測試。
- [x] T4（AC #8）：全套檢查。

## Dev Notes

- 掃描器的 `episodeRepo` 是 `repository.EpisodeRepositoryInterface`；新方法用窄介面型別判斷接上，不擴大既有介面（避免改所有 mock）。
- 保護判斷沿用 `underAnyRoot(path, untrustedRoots)`＋`untrustedLibraries`；集數的媒體庫取自 `series.library_id`。

## Dev Agent Record

**Agent:** Opus 5.5（create-story／dev）；CR：Sonnet（換模型慣例）。

### Completion Notes

- Repository：`FindFilesForRemovalCheck`（id、路徑、`series.library_id`）、`ClearMissingFile(id, path)`（`WHERE file_path = ?`，清 `file_path`／`subtitle_tracks`／`subtitle_tracks_file_sig`）（AC #1、#3、#4）。
- `ScannerService.detectRemovedEpisodeFiles`：接在電影的 `detectRemovedFiles` 之後；`underAnyRoot`＋`untrustedLibraries` 保護；非 `IsNotExist` 的錯誤不動；`FilesRemoved`＝電影＋集數，log 分開寫（AC #1、#2、#7）。窄介面型別判斷，production 的 `repos.Episodes` 是 `*repository.EpisodeRepository`，測試釘住。
- 測試：repository 3 個（真 migration 鏈：列出含媒體庫、清除後彙總不再算、CAS 不清新路徑、狀態欄保留、檔案回來寫回同一列）；scanner 7 個（健康資料夾只清那一集、讀不到／空掛載不清、不可信媒體庫保留、沒有新方法時不做事、掃描中被移到新路徑不算、電影＋集數一起計數）（AC #3、#5、#6、#7）。
- 檢查：`go build／vet` 綠；`go test ./...` 只有 `TestBackupService_Restore_PosterFailureKeepsTheRestoredDatabase` 紅（main 一樣，雲端容器 root）；`staticcheck 2026.1` 0 項；scanner 測試 `-race` 綠（AC #8）。

### CR（Sonnet，0H／0M／5L）

- 補測試：掃描中被移到新路徑不清不算（AC #3）、電影＋集數合計（AC #7）。
- 未補：權限錯誤那條（雲端容器 root 模擬不出，同備份測試的原因；程式邏輯與電影同一行判斷）。
- 記錄、不改（與電影行為一致）：
  - L1：媒體庫被刪或掃描退回環境變數資料夾時，不在這輪掃描範圍的集數若檔案也不在，會被清——電影同樣會被標移除，屬既有行為。
  - L2：一部劇的集數全被清時影集本身仍在清單上，中文字幕結論退回影集列自己的（可能舊的）判斷；沒有影集層的 is_removed，另議。
  - L3：清掉的集數保留字幕狀態；檔案回來但沒有字幕檔時，狀態要等下次找字幕才會更新（AC #4 本就如此定）。

### 交付後待辦

- 正式機刪掉一集（或看 log「episode file gone — now shown as no local file」），確認季清單變「沒有本地檔」、影集中文字幕結論不再被它拉歪。

### File List

- `apps/api/internal/repository/episode_repository.go`
- `apps/api/internal/repository/episode_removed_file_test.go`（新）
- `apps/api/internal/services/scanner_service.go`
- `apps/api/internal/services/scanner_episode_removed_test.go`（新）
- `_bmad-output/implementation-artifacts/disc-2026-10-episode-rows-outlive-deleted-files.md`（新）
- `_bmad-output/implementation-artifacts/sprint-status.yaml`
