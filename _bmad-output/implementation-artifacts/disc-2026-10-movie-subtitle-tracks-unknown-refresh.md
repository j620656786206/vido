# Disc：電影「不知道有沒有中文字幕」的那些，背景補讀一次

Status: done

**Source:** filed 2026-10-06 by `disc-2026-10-subtitle-filter-disagrees-with-badges`（Rule 24 lane ③）。⚖️ 2026-10-08 Alexyu 選 C 開工。

## Story

身為片庫的使用者，
我要每部電影都知道「有沒有中文字幕」，
這樣篩「缺中文字幕」時不會漏掉 NAS 上那 12 部被標成「不知道」的電影。

## 問題

NAS 有 12 部電影 `subtitle_tracks` 是 NULL（＝不知道）。兩個原因，都在 `enrichment_service.go` 的 `applyFFprobeTechInfo`：

1. 舊的掃描讀到「沒有任何字幕」時不寫；上游單改成寫 `[]`，只對之後的掃描有效。
2. 影片技術資訊已經有值（通常來自 NFO 的 streamdetails）時，整段 ffprobe 跳過、也不寫字幕；沒有旁邊的字幕檔時就一直是 NULL。**這條到現在還會產生新的 NULL。**

已經有值的列不會被重讀，所以這些 NULL 永遠不會自己好。

## Acceptance Criteria

1. 新的背景補讀：開機約 75 秒後跑一次，之後每次**有新增或變動檔案的**掃描完成再跑一次（掃描器本來就只在這種時候通知）（原因 2 會持續產生新的 NULL）。同時間只跑一輪；跑的中途又被觸發，就在跑完後再補一輪。
2. 只處理「有檔案路徑、沒被移除、`subtitle_tracks` 是 NULL」的電影。已經有值的（包括 `[]`）一律不碰。
3. 每部：檔案不在（NAS 還沒掛載、被刪）→ 跳過，下次再試；ffprobe 讀成功 → 片內字幕軌＋旁邊字幕檔合併寫入，跟掃描時同一套規則（沒有任何字幕就寫 `[]`＝「缺中文」）；ffprobe 失敗 → 維持 NULL（不知道），同一個檔案在這次程式執行期間不再重試，檔案換了就會再試。
4. 寫入只在該列還是 NULL 時生效（`WHERE subtitle_tracks IS NULL`）：同時間掃描已經寫了值，就不覆蓋。
5. ffprobe 不能用時整輪不做（維持「不知道」，不亂猜）。
6. log：一輪開始／結束各一行（補了幾部、失敗幾部、檔案不在幾部）；什麼都沒得補就不寫 log。
7. 關機時先停掉補讀、等它結束，再關資料庫。
8. 「有沒有中文字幕」的結論是查詢時從 `subtitle_tracks` 算的（`vido_chinese_subtitle`），不用另外改欄位。
9. 檢查：`go build／vet`、`go test ./...`、`staticcheck` 全綠；新服務與兩個 repository 方法都有測試。

## Tasks / Subtasks

- [x] T1（AC #2、#4）：`MovieRepository.FindMissingSubtitleTracks(afterID, limit)`、`UpdateSubtitleTracksIfMissing(id, json)`＋repository 測試。
- [x] T2（AC #1、#3、#5、#6）：`services.MovieSubtitleTracksBackfillService`（Run／Trigger／RunAfter／Wait）＋測試。
- [x] T3（AC #1、#7）：`cmd/api/main.go` 接線：掃描完成觸發、開機 75 秒、關機前停。
- [x] T4（AC #9）：全套檢查。

## Dev Notes

- 參考同樣形狀的 `episode_subtitle_tracks_service.go`（集數版，含檔案簽章、失敗記憶、掃描後觸發）。電影不需要簽章欄位：只補 NULL，補完就不在名單裡。
- 字幕合併沿用 `MergeSubtitleTracks` ＋ `DetectExternalSubtitles`（跟掃描同一套）。
- 開工前原本要唯讀查 NAS 這 12 部的 `video_codec`／`metadata_source`；雲端 session 連不到 NAS，所以設計成兩個原因都涵蓋。正式機更新後看 log「movie subtitle tracks backfill: pass finished」的 filled 數（預期 12 左右）。
- 不做：繁簡判斷抽樣（集數版有，電影的掃描路徑本來就沒有，另議）；原因 2 的根治（讓掃描在有 NFO 技術資訊時也讀字幕）——背景補讀在每次掃描後都會跑，效果相同，不必動掃描。

## Dev Agent Record

**Agent:** Opus 5.5（create-story／dev）；CR：Sonnet（換模型慣例）。

### Completion Notes

- Repository：`FindMissingSubtitleTracks(afterID, limit)`（依 id 分頁）、`UpdateSubtitleTracksIfMissing`（`WHERE subtitle_tracks IS NULL`，不動 `updated_at`）（AC #2、#4）。
- 服務 `MovieSubtitleTracksBackfillService`：stat → ffprobe → `MergeSubtitleTracks(片內, DetectExternalSubtitles)` → 只補 NULL；檔案不在跳過、探測失敗依檔案簽章記住；ffprobe 不能用整輪不做（AC #1、#3、#5、#6）。
- `main.go`：掃描完成觸發、開機 75 秒、關機在 `db.Close()` 前停（AC #1、#7）。
- 檢查：`go build／vet` 綠；`go test ./...` 只有 `TestBackupService_Restore_PosterFailureKeepsTheRestoredDatabase` 紅（main 一樣，雲端容器 root）；`staticcheck 2026.1` 0 項；新測試 `-race -count=3` 綠（AC #9）。

### CR（Sonnet，0H／3M／4L，修 3M＋2L）

- **M1**：掃描（enrichment）走 NFO 那條時手上拿著舊的 NULL，晚一步寫回會把補讀剛寫的答案蓋成「不知道」。修：`UpdateEnrichedMetadata` 與 wide `Update` 的 `subtitle_tracks` 改 `COALESCE(?, subtitle_tracks)`——「不知道」蓋不掉已知答案；有值照樣覆寫。repository 測試釘住。
- **M2**：原本一輪只取最舊 1000 列，一直讀不到的列會永遠擋在前面。修：依 id 分頁走完所有頁。測試：每頁 2 列、前 3 列檔案不在，第 4 列仍補到。
- **M3**：關機後才結束的掃描仍會 `Trigger`。修：ctx 已取消就不啟動。
- L4：補「關機中斷的探測不記成失敗」「關機後觸發不動作」測試；編碼失敗補 log。
- 未改：L1（探測失敗到重啟前不重試——AC #3 本來就這樣定，換檔會重試）；L2（掃描器只在有新增／變動檔案時通知——AC #1 文字已改成照實寫）；L3（ffprobe 已有 3 格的共用限流，補讀一次只用一格）。

### 交付後待辦

- 正式機更新後看 log「movie subtitle tracks backfill: pass finished」的 `filled`（預期約 12），再用「缺中文字幕／不知道」篩選確認那 12 部有了答案。

### File List

- `apps/api/internal/services/movie_subtitle_tracks_backfill_service.go`（新）
- `apps/api/internal/services/movie_subtitle_tracks_backfill_service_test.go`（新）
- `apps/api/internal/repository/enriched_metadata_update.go`
- `apps/api/internal/repository/enriched_metadata_update_test.go`
- `apps/api/internal/repository/movie_repository.go`
- `apps/api/cmd/api/main.go`
- `_bmad-output/implementation-artifacts/disc-2026-10-movie-subtitle-tracks-unknown-refresh.md`
- `_bmad-output/implementation-artifacts/sprint-status.yaml`
