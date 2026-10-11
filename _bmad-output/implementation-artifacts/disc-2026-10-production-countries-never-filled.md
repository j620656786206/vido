# Disc：片子的「出品國家」從沒存進資料庫——陸港澳片「只轉字」在正式機從沒生效

Status: review

**Source:** 2026-10-11 正式機驗證 `disc-2026-10-hk-mo-mainland-rule`（PR #740）。s2tw 轉換本身是對的，但正式機每一部片的 `production_countries` 都是 NULL，規則永遠不會觸發。

## 查到的事（2026-10-11，Bob 逐一開檔／唯讀查 NAS 確認）

### 正式機現況（`sqlite3 -readonly /mnt/cache/appdata/vido/vido.db`，21:4x 與 22:04 各讀一次）

| | 列數 | 有 tmdb_id | `production_countries` 有值 | `genres` 有值 |
|---|---|---|---|---|
| movies | 55 | 45 | **0** | **1** |
| series | 81 | 81 | **0** | **0** |

- 陸劇也是空的：凡人修仙傳（tmdb 106449）、劍來（259537）、天龍八部（72819），`updated_at` 都是 2026-10-06 22:1x。
- 媒體庫「類別」篩選只剩幾顆（冒險／劇情／動作／西部），就是那 1 部電影的 genres。

### 根因一：比對時有抓到國家，但寫資料庫那一步把它丟掉（新比對的片今天仍是這樣）

- sub-7-2b（PR #631，2026-09-30）讓比對後多抓一次 TMDb 詳情、把國家放進 model：
  - 電影：`applyTMDbMovieFacets` → `movie.SetProductionCountries`（`apps/api/internal/services/enrichment_service.go:1024-1044`），由 `enrichMovieFrom` 呼叫（`:682`）。
  - 影集：`applyTMDbSeriesFacets` → `series.SetProductionCountries`（`:1050-1070`），由 `enrichSeries` 呼叫（`:432`）；國家來自 `seriesCountriesFromDetails`（`:1090-1108`，`production_countries` 空就退回 `origin_country`）。
  - 手動選片與 IMDb 路線走 `applyTMDbMovieDetails`（`:965-1017`，國家在 `:1014-1016`）／`applyTMDbSeriesDetails`（`enrichment_manual_match.go:448-482`，國家在 `:479-481`）。
- **但三條路最後都經過 `persistMovieMatch`／`persistSeriesMatch`（`enrichment_manual_match.go:423-445`），它們只呼叫 `UpdateEnrichedMetadata`＋`UpdateCredits`。**
- `UpdateEnrichedMetadata` **刻意不寫 `production_countries`**：註解明列「Deliberately NOT written here … original_language, status, production_countries, credits, spoken_languages」（`apps/api/internal/repository/enriched_metadata_update.go:44-49`）；電影 SQL（`:65-93`）與影集 SQL（`:132-147`）都沒有這一欄。這段註解是 sub-7-2b 之前寫的（當時 enrichment 確實不碰國家），sub-7-2b 沒有回頭改。
- **為什麼測試沒抓到：** `TestEnrichMovie_SearchPath_FillsGenresAndCountriesFromDetails`（`apps/api/internal/services/enrichment_context_facets_test.go:61-78`）斷言的是假 repo 收到的 model（`repo.updatedMovie`，假 repo 在 `enrichment_nfo_test.go:136` 直接存 model），不是真資料庫。真 DB 的 writer 測試 `apps/api/internal/repository/enriched_metadata_update_test.go` 沒有任何 `production_countries` 斷言（grep `production_countries|ProductionCountries` 該檔 0 筆）。

### 根因二：9/30 之前比對好的片，genres 跟國家都沒有、也不會再被比對

- sub-7-2b 之前，搜尋結果本身沒有 genres，比對時也不抓詳情，所以舊列的 genres 是 `[]`、國家是 NULL。
- 比對只在 `parse_status` 是 pending／空時發生；已比對的列不會重跑（跟 `disc-2026-10-series-credits-empty` 同一個原因，該 story 的根因段有引證）。手動列的「重新整理」走 `refreshManualMovie`／`refreshManualSeries`（`enrichment_manual_match.go:351-381`、`:408-419`），只補 ffprobe 與 credits，不抓詳情。

### 誰在讀國家（修好後會一起變）

- 轉繁的「保留原說法」判斷：`zhtw.Finalize`／`KeepsOwnWording`（`apps/api/internal/zhtw/finalize.go:35-57`）；國家來源：
  - 手動下載／轉換：`handlers/subtitle_countries.go:41`、`:52`（讀 `movie.ProductionCountries`／`show.ProductionCountries`）。
  - 批次：`subtitle/batch.go:547`（電影）、`:590`、`:641-651`（影集）。
  - 請求觸發：`subtitle/request_trigger.go:124`。
  - engine：`subtitle/engine.go:144-157`、`:200`（AI 術語校正跳過與否）。
- **翻譯提示的片子資料**也有國家與類別：`subtitle/media_store.go:105-116`（電影）、`:264`（影集）；`services/transcription_service.go:1868-1878`、`:1913`。
- **⚠️ 會讓翻譯快取換一把鑰匙：** 片子資料的 hash 含 `Genres` 與 `Countries`（`subtitle/segment_cache.go:104-114`）。補上之後，那部片下次「重新生成」時逐句快取不會命中，要整部重翻一次（例：See S01E02 約 $0.40）。只影響重新生成，不影響已經做好的字幕檔。

### 沒有使用者會手改這兩欄的路

- grep `productioncountr|production_countries`（不分大小寫）於 `apps/api/internal/handlers` 與 `services/metadata*.go`：只有 `subtitle_countries.go` 讀取，沒有任何編輯器寫入。genres 的「只補空的、不覆蓋」規則照樣保護使用者手改的類別。

### 先例

- `CreditsBackfillService`（`apps/api/internal/services/credits_backfill_service.go`）：窄介面 `FindMissingCredits`＋`UpdateCredits`（`:29-38`）、一次最多 200 列（`:43`）、`tried` 集合（`:53`）、`RunAfter`（`:142-151`），main 用型別斷言接線、開機 45 秒（`cmd/api/main.go:720-731`）。
- 窄 writer 的寫法：`updateCreditsColumn`（`repository/enriched_metadata_update.go:193-215`）；查詢：`FindMissingCredits`（`:221-270`）。
- 掃描後也觸發的 backfill：`MovieSubtitleTracksBackfillService`（`cmd/api/main.go:1115-1128`，開機 75 秒＋`AppendOnScanComplete`）。

## Story

身為 Vido 的使用者，
我希望每部比對到 TMDb 的片都記得它是哪國出品、是什麼類別，
這樣陸港澳片的簡中字幕轉繁時會保留原本的說法，翻譯也拿得到正確的片子背景，媒體庫的「類別」篩選也有東西可選。

## Acceptance Criteria

1. **比對時真的寫進去：** 電影與影集的比對（搜尋、NFO／IMDb、手動選片三條路，都經過 `persistMovieMatch`／`persistSeriesMatch`）之後，資料庫的 `production_countries` 有值（TMDb 有給的話）。用**真資料庫**測：電影一條、影集一條（含 `origin_country` 退回）。
2. **不覆蓋、不清空：** 這次比對沒拿到國家（詳情呼叫失敗、TMDb 沒給）時，已存的國家不變（NULL 代表「不知道」，不能蓋掉已知答案）。真 DB 測試。
3. **舊列補上：** 新增背景補抓，列出「有 tmdb_id、沒被移除、`production_countries` 是 NULL 或 genres 是空（`'[]'`／`''`）」的電影與影集，每列抓一次 TMDb 詳情（`GetMovieDetails`／`GetTVShowDetails`，走 TMDb 服務自己的快取與限速），**只補空的那一欄**：國家空就寫國家，genres 空就寫 genres；已有值的欄一個字都不動。影集國家用 `seriesCountriesFromDetails`。
4. **補不到的不重問：** TMDb 也沒有國家／類別的列，同一個 process 內不再問（比照 credits 的 `tried`）；抓失敗的下次開機再試。
5. **何時跑：** 開機 60 秒後跑一次，log 一行 `context facets backfill pass finished movies_filled=… series_filled=… tmdb_has_none=… failed=…`（有做事才寫）。不必接掃描完成——新比對的片已由 AC #1 寫入。
6. **註解跟著改：** `enriched_metadata_update.go` 檔頭與 `UpdateEnrichedMetadata` 的「Deliberately NOT written」清單更新成現況（國家改由窄 writer 寫，理由寫明）。
7. **檢查：** `go build／vet／test ./...`、`~/go/bin/staticcheck ./...`、`pnpm run lint:all` 全綠。
8. **正式機實測（部署後補）：** (a) 開機約 1 分鐘後看到 AC #5 那行；(b) 唯讀查 DB：有 tmdb_id 的 45 部電影＋81 部影集，`production_countries` 有值的比例（TMDb 本來就沒國家的除外），凡人修仙傳／劍來 含 `CN`；genres 有值；(c) 回頭重驗 `disc-2026-10-hk-mo-mainland-rule`：找一部 CN 片做一次簡中→繁中（手動下載或片內簡中軌轉換），字變繁體、用語保留（视频→視頻）。通過後該 story 轉 done。

## Tasks / Subtasks

- [x] T1 窄 writer：`MovieRepository.UpdateContextFacets`／`SeriesRepository.UpdateContextFacets`（AC #1、#2、#3）
  - [x] 一個共用 helper：`genres = CASE WHEN ? IS NULL THEN genres ELSE ? END`、`production_countries = COALESCE(?, production_countries)` 這類「沒給就不動」的寫法；空切片視為「沒給」。
  - [x] 真 DB 測試：寫入、只寫一欄、空值不清空、找不到列回錯。
- [x] T2 `persistMovieMatch`／`persistSeriesMatch` 在 `UpdateEnrichedMetadata` 之後呼叫 `UpdateContextFacets`（只傳國家；genres 已由 `UpdateEnrichedMetadata` 寫）（AC #1）
  - [x] 假 repo 補方法；加一條「model 有國家 → writer 被呼叫」的服務測試；AC #1 的真 DB 測試放 T1。
- [x] T3 查詢：`FindMissingContextFacets(ctx, limit)` 電影／影集各一（AC #3）
  - [x] 條件：`tmdb_id > 0 AND is_removed = 0 AND (production_countries IS NULL OR genres IN ('[]',''))`，`ORDER BY updated_at ASC LIMIT ?`；真 DB 測試（有值的不列、被移除的不列、沒 tmdb_id 的不列）。
- [x] T4 `ContextFacetsBackfillService`＋測試＋main 接線（AC #3、#4、#5）
  - [x] 依賴：上面兩個窄介面＋`GetMovieDetails`／`GetTVShowDetails`（`TMDbServiceInterface` 的子集）。
  - [x] 測試：只補空欄、不覆蓋、TMDb 沒給 → tried、抓失敗 → failed 下次再試、ctx 取消就停。
  - [x] `main.go` 比照 credits backfill：型別斷言、自己的可取消 ctx、`RunAfter(ctx, 60*time.Second)`。
- [x] T5 註解（AC #6）＋檢查（AC #7）
- [ ] T6 正式機實測（AC #8，部署後）— ⏳ 刻意延後：要等合併、ghcr 映像更新、NAS 重建後才能量；比照 `disc-2026-10-series-credits-empty` T5 的做法，部署後補「正式機驗證」一節再轉 done。

全部後端 → 不需要拆單（前端 0 個 task）。

## Dev Notes

### 不要做的事

- **不要**把 `production_countries` 加進 `UpdateEnrichedMetadata` 的大 UPDATE——該檔的紀律是「enrichment 算出來的新欄位用自己的窄 writer」（`enriched_metadata_update.go:177-182` 講 credits 為什麼不擴寬），而且大 UPDATE 會把 NULL 當答案寫回。
- 不補 `original_language`：Go 端沒有任何判斷讀它（grep `OriginalLanguage` 於 `apps/api/internal`，排除 models／tmdb／converters／repository：0 筆）。
- 不補 `spoken_languages`、`status`。
- 不碰合拍片規則（`disc-2026-10-coproduction-wording-rule`）。
- 不回頭重做已經轉好的字幕。

### 已知陷阱

- **翻譯快取鑰匙會換**（見上）：補完後重新生成那部片要整部重翻。這是預期中的一次性代價，寫進 Completion Notes 讓 Alexyu 知道。
- 影集 TMDb 詳情常常 `production_countries` 是空的、只有 `origin_country`——一定要用 `seriesCountriesFromDetails`，不要自己讀 `details.ProductionCountries`。
- `genres` 欄是 `NOT NULL DEFAULT '[]'`，空的寫法是 `'[]'`，不是 NULL。
- 沒比對到 TMDb 的列（`tmdb_id` 空，NAS 上 10 部電影，例：芳華、長安的荔枝）不在範圍，也沒辦法補。
- 被 `tried` 記住的列每次開機會再問一次 TMDb（同 credits 的取捨，136 列可接受）。

### Time-dependent visual coverage

N/A — no wall-clock-reading components touched（純後端）。

### References

- 根因與先例：本檔「查到的事」各條的 `path:line`。
- 前一張同型：`_bmad-output/implementation-artifacts/disc-2026-10-series-credits-empty.md`。
- 規則來源：`_bmad-output/implementation-artifacts/disc-2026-10-hk-mo-mainland-rule.md`（含 2026-10-11 正式機驗證不通過的紀錄）、`sub-7-2b-metadata-context-genres-countries.md`。

## Dev Agent Record

### Agent Model Used

Claude Opus 5.5（Amelia，dev-story；2026-10-11）

### Debug Log References

- RED：`TestEnrichSeries_FillsGenresAndCountriesFromDetails`／`TestEnrichSeries_FallsBackToOriginCountry` 加上「國家要經過自己的 writer」斷言後失敗（`map[string]models.NullString(nil) does not contain "series-1"`）；接上 `persistCountries` 後轉綠。

### Completion Notes List

- 2026-10-11 Bob create-story：根因比 sprint-status 立案時的推測多一層——不只舊列沒補，**連 9/30 之後的新比對也沒存進去**（`UpdateEnrichedMetadata` 不寫這欄）。genres 同樣全空（電影 1/55、影集 0/81），同一次 TMDb 詳情就拿得到，併進本單（lane ①，AC #3）。
- **T1 跟 story 寫法不同（刻意）：** story 寫「一個 `UpdateContextFacets`」，實作拆成兩個 writer，因為兩條路的所有權規則不同：
  - `UpdateProductionCountries`（比對用）：新的比對說了算，有值就取代；NULL 不清掉舊值（AC #2）。
  - `FillContextFacets`（補抓用）：只補空欄，SQL 層就保證（`CASE … WHEN 空 THEN`），所以補抓途中有人比對或改了類別，也不會被蓋掉。
  - 兩個都**不動 `updated_at`**（比照 `UpdateSubtitleTracksIfMissing`：背景補的衍生欄位，不是使用者編輯），所以媒體庫「新增日期」排序不會被補抓打亂。
- **T2：** `persistMovieMatch`／`persistSeriesMatch` 在 `UpdateEnrichedMetadata` 之後呼叫 `persistCountries`。用型別斷言拿 `productionCountriesWriter`，不改 `MovieRepositoryInterface`／`SeriesRepositoryInterface`（跟 credits backfill 同理由：加方法會弄壞幾十個測試假物件），並加了編譯期斷言 `var _ productionCountriesWriter = (*repository.MovieRepository)(nil)`，真 repo 少了方法會直接編譯失敗，不會默默跳過。
- **T3：** `FindMissingContextFacets` 用 id 分頁（比照 `FindMissingSubtitleTracks`），不用 `updated_at` 排序——TMDb 真的沒有國家的列每次都會被列出來，固定第一頁會把後面的列擋住。
- **T4：** `ContextFacetsBackfillService`，開機 60 秒跑一次；影集國家走 `seriesCountriesFromDetails`（`origin_country` 退回）；genres／國家用 `models.Movie.GenresJSON`／`SetProductionCountries` 序列化，存的形狀跟比對寫的一模一樣。只有做了事才寫 log。
- **已知限制（CR L3，接受）：** 手動改選成另一部、而 TMDb 上那部沒有國家時，舊的國家會留著（AC #2 的取捨：NULL 不清；genres 本來就是同樣行為）。要修就得在換 tmdb_id 時明確清空，會推翻 AC #2，實務上改選幾乎都是同一部片換版本——記在這裡，不另立單。
- **Code review（2026-10-11，Sonnet 對抗式審查，0H／1M／3L）：**
  - ✅ M1 修：使用者在編輯器清空的類別會被補抓填回 → 類別只補「不是使用者擁有（`metadata_source != manual`）」的列；國家沒有編輯器，照補。新增測試（手動列只補國家、類別不動；查詢只因國家列出手動列）。
  - ✅ L2 修：補抓的 goroutine 可能在 `db.Close()` 之後才寫 → `RunAfter` 改回傳 `stop`（取消並等它結束），main 在關機區塊、`db.Close()` 之前呼叫；新增測試。（credits backfill 也是同樣寫法，傷害只有 warn log，不在本單範圍、不另立單——寫在這裡。）
  - ⏸ L3 接受（見上）。
  - ✅ L4 修：手動選片（`ApplyTMDbMatch`）電影／影集兩條真 DB 測試加上「國家讀回來」斷言；驗證過：把 `persistCountries` 暫時改成直接 return，這兩條會紅。
- 🔗 AC Drift: NONE（checked: `production_countries|國家|countries` across `_bmad-output/implementation-artifacts/*.md` 的 sub-7-2b、hk-mo-mainland-rule、series-credits-empty——sub-7-2b AC #2／#3 本來就要求比對後存下國家，本單是讓它真的成立，REUSE 不是 DRIFT）。
- 📎 Contract Stamps: NONE（本單與 sub-7-2b 都沒有 `[@contract-v*]`；沒有改任何 API 回應形狀）。
- 🎭 A11y Pre-Flight: N/A (100% backend — no apps/web/ files touched)
- 🎨 UX Verification: SKIPPED — no UI changes in this story（媒體庫「類別」篩選會多出選項，但那是資料變多，元件沒改）。
- 檢查：`go build ./...`、`go vet`、`go test ./...` 全綠；`~/go/bin/staticcheck ./...` 0 個問題；`pnpm run lint:all` 0 errors（172 warnings 是既有的）；`pnpm nx test api` 綠；`pnpm nx test web` 300 檔／4677 測試全綠；`test:cleanup` 沒有殘留程序。
- 新測試 9 條（CR 後再加 3 條斷言／測試，見上）：repository 4（真 migration schema：寫入並撐過 `UpdateEnrichedMetadata`、NULL 不清／有值取代／錯 id 報錯、只補空欄不動 updated_at、查詢只列有缺的且分頁）、services 3（backfill 補國家＋類別含 origin 退回、沒東西／失敗／重試、分頁＋取消＋停用＋列表失敗）、既有 3 條 enrichment 測試補上「國家要經過 writer」斷言。

### Discovery Triage

- ① genres 全空（電影 54/55、影集 81/81 是 `[]`）→ 同一個 TMDb 詳情呼叫、同一個翻譯提示欄位，併入 AC #3（T3／T4 已含）。
- 其他：N/A — no other out-of-scope work discovered.

### File List

- `apps/api/internal/repository/context_facets.go`（新）
- `apps/api/internal/repository/context_facets_test.go`（新）
- `apps/api/internal/repository/enriched_metadata_update.go`（改註解）
- `apps/api/internal/services/context_facets_backfill_service.go`（新）
- `apps/api/internal/services/context_facets_backfill_service_test.go`（新）
- `apps/api/internal/services/enrichment_manual_match.go`（`persistCountries`）
- `apps/api/internal/services/enrichment_context_facets_test.go`（斷言）
- `apps/api/internal/services/enrichment_nfo_test.go`（假 repo 補方法）
- `apps/api/internal/services/enrichment_glossary_seed_test.go`（假 repo 補方法）
- `apps/api/internal/services/enrichment_manual_match_test.go`（手動選片真 DB 讀回國家）
- `apps/api/cmd/api/main.go`（接線）
- `_bmad-output/implementation-artifacts/disc-2026-10-production-countries-never-filled.md`
- `_bmad-output/implementation-artifacts/sprint-status.yaml`

## Change Log

| 日期 | 內容 |
|---|---|
| 2026-10-11 | Bob create-story（根因兩層＋genres 併入）。 |
| 2026-10-11 | Amelia dev-story：比對時用窄 writer 存國家；開機 60 秒補抓舊列的國家與類別（只補空欄、不動 updated_at）；9 條新測試；全套測試／staticcheck／lint 綠。T6 正式機實測待部署。 |
| 2026-10-11 | Code review 0H/1M/3L：M1（手動列的類別不補）、L2（關機前停補抓）、L4（手動選片真 DB 讀回國家）已修；L3 接受。 |
