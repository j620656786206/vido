# Story：季標頭補上 TMDb 的季資料——不再寫「劇集數未知」、也有季海報

Status: done

**Source:** `disc-2026-10-season-episode-count-unknown`（P3，filed 2026-10-05，Alexyu 截圖：《末日光明》第 1 季標頭寫「劇集數未知」，下面卻列了 8 集）。Alexyu 2026-10-08：「接著做下一張」（候選清單第 3 張）。

> ⚖️ **SM 裁定（Bob 2026-10-08）：** 原單給兩條路——「補寫 episode_count」或「沒有時改用列出的集數」。查下去發現不是一部片的問題：**正式機 134 個季，全部沒有集數、名稱、海報、TMDb id**。所以走長解：把 TMDb 的季資料真的寫進 `seasons` 表，不在前端補數字。

## Story

身為 Vido 的使用者，
我希望影集的每一季標頭都顯示「8 集 · 2019-11-01」和季海報，
而不是「劇集數未知」和「無圖」。

## 查到的事（2026-10-08）

- 正式機唯讀：`seasons` 134 列，`episode_count`／`name`／`poster_path`／`tmdb_id` **全部是 NULL**。
- 原因：`MediaIngestService.UpsertSeason`（`media_ingest_service.go:204-251`）建季時去讀 `series.GetSeasons()`——但那是早已沒人讀寫的 `series.seasons` JSON 欄（`series_repository.go` 的 SELECT／UPDATE 都沒有它，bugfix-20-1 也記過），永遠是空的；季已經存在時直接回傳、不再補。掃描時影集還沒對上 TMDb，之後也沒有任何地方回頭補。
- 季標頭：`SeasonAccordion.tsx:108` 只看 `season.episodeCount`；`GetSeasons` 讀 `seasons` 表（`series_season.go:60-82`）。
- 打開一季時，`GetSeasonEpisodes` 已經向 TMDb 拿了這季的完整資料（名稱、海報、首播日、每一集），用完就丟。
- 前例：開機後一次的背景補齊 `CreditsBackfillService.RunAfter`。

## Acceptance Criteria

1. **背景補齊**：開機後（延遲）與每次掃描完成，對「已對上 TMDb、且有季缺資料（`episode_count` 或 `tmdb_id` 是 NULL）」的影集，各向 TMDb 拿一次影集資料，把每季的名稱、簡介、海報、首播日、集數、TMDb id 寫進 `seasons`。TMDb 沒有那一季就跳過，這個 process 內不重試那部。失敗只記 log。log：補了幾部、幾季。
2. **打開一季時順手寫回**：`GetSeasonEpisodes` 拿到的 TMDb 季資料，若跟 `seasons` 那列不同（或那列是空的），寫回那一列；寫回失敗只記 log，不影響回應。
3. **不蓋掉已有的值成空的**：TMDb 某欄是空的，不把資料庫的值改成空。
4. **測試**：補齊（有缺的才拿、TMDb 沒有那季跳過、失敗不重試、不蓋成空）、寫回（不同才寫）、repository 查詢（只列出有缺的、已對上 TMDb 的影集）。
5. `go build／vet／test ./...`、`staticcheck`、`lint:all`（看「✖ N problems (0 errors」那行）全綠。

## Tasks / Subtasks

- [x] T1 repository：列出「有季缺資料」的影集
- [x] T2 `SeasonDetailsBackfillService`＋開機／掃描後接線（AC #1、#3）
- [x] T3 `GetSeasonEpisodes` 寫回（AC #2）
- [x] T4 測試＋閘門（AC #4、#5）

全後端，4 項。

## Dev Notes

- 不修 `UpsertSeason` 讀死欄位的那段——它在掃描時跑，那時影集多半還沒對上；補齊交給背景工作。死欄位的清理另立條目。
- TMDb 呼叫走 `TMDbService.GetTVShowDetails`（有快取、有限流）。81 部影集 → 81 次。

## Dev Agent Record

### Agent Model Used

Claude Opus 5.5（dev）；對抗式 CR 換模型（Sonnet 5.5）。

### Completion Notes List

- 2026-10-08 Bob（SM）：建單；正式機唯讀查證 134 季全空，改走長解（SM 裁定）。
- 2026-10-08 Amelia（dev）：T1–T4 完成。
  - `SeasonRepository.FindSeriesNeedingSeasonDetails`：已對上 TMDb、沒被移除、有季缺集數或 TMDb id 的影集（正式機唯讀跑同一段 SQL：81 部）。
  - `SeasonDetailsBackfillService`：開機 90 秒後＋每次掃描完成；每部影集向 TMDb 拿一次影集資料（有快取、有限流），照季號把名稱／簡介／海報／首播日／集數／TMDb id 寫進 `seasons`；`MergeSeasonDetails` 不會把已有的值蓋成空的。
  - `GetSeasonEpisodes`：打開一季時把剛拿到的 TMDb 季資料寫回那一列，回應裡的季標頭也用寫回後的資料。
  - 關機時先取消補齊再關資料庫。
- 2026-10-08 對抗式 CR（Sonnet 5.5）：0H／2M／4L。修 M1（缺資料的影集改成「休息 12 小時」再問，不是整個 process 都不問——還沒播的季會播、TMDb 暫時壞掉會好）、M2（一輪上限 5000，休息中的影集不會擠掉其他影集）、L3（影集層級的集數只補空的，不蓋掉打開季清單時寫的精確數字）、補 Trigger 合併測試。L4（寬的 Update）、L5（關機只取消不等待）記錄不修。
  - 閘門：`go vet`、`go test ./...` 全綠、本單測試 `-race` ×2、`staticcheck` 乾淨、`lint:all` 0 errors。

- 2026-10-08 /ship：PR #723。

### Discovery Triage

- **③（不修）** `MediaIngestService.UpsertSeason` 讀的 `series.seasons` JSON 欄是死欄位（沒有 SELECT／寫入），那段永遠拿不到資料；可清掉。留在 Dev Notes，不另立。

### File List

- apps/api/internal/repository/season_repository.go、season_details_gap_test.go（新）
- apps/api/internal/services/season_details_backfill_service.go（新）＋ _test.go（新）、series_season.go
- apps/api/cmd/api/main.go
