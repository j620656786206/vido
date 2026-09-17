# Story DSR.2b-a：沒認出來的片救得回來——「手動選片」真的寫進去、「重新比對」真的會跑（後端）

Status: review

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As the person whose NAS has files the scanner could not match to a movie or series,
I want 我在詳情頁選的那部片真的被存進去、按「重新比對」真的會重新比對、自己填的資料不會被下一次自動比對蓋掉,
so that 一部沒認出來的片有真正能走的路變回正常的片，而不是按了「成功」卻什麼都沒變。

## Context

`epic-dsr` 的 `dsr-2b`（由 dsr-2 拆出）。**建單時查到後端三條救援路全是空殼**，Alexyu 2026-09-17 裁定「連後端一起修，拆兩張」（見裁定紀錄）：

- **本張（dsr-2b-a）＝後端**：套用、單筆重新比對、修改資訊三個端點做成真的，並讓自動比對不再蓋掉使用者的選擇。**沒有任何 `apps/web` 工作**（只動 `tests/` 底下的 e2e helper）。
- **`dsr-2b-b`＝前端**：v2 詳情頁的「比對失敗」「整理中」兩個狀態＋手動選片對話框＋設計稿。**依賴本張。**

⚠️ **正式環境目前沒有任何畫面呼叫「套用」或「單筆重新比對」**（`ManualSearchDialog` 只掛在 dev 用的 `/test/manual-search` 與視覺夾具；`useReparseItem` 只在死元件裡）。所以本張可以放心改回應形狀——唯一的外部呼叫者是 `tests/e2e/manual-search.api.spec.ts` 與 `/test/manual-search` 頁。

### 🔴 建單時查到的事（sprint-status 條目都沒寫）

1. **「套用」是靜默的空操作。** `POST /api/v1/metadata/apply` 需要 `SetMediaUpdaters`（`services/metadata_service.go:682`），**只有測試呼叫過**。正式環境走 nil 分支：記一行 Warn、`title="Unknown"`、**照樣回成功**（`:745-753`）。就算接上，`MediaUpdater` 介面（`:301-304`）也只有 `UpdateMetadataSource`——**形狀本身寫不進一整筆配對結果**。已知條目 `wire-set-media-updaters-test-harness` 只看到「沒接線」，本張吸收。
2. **單筆重新比對是 stub。** `ReparseMovie`／`ReparseSeries`（`handlers/library_handler.go:158-177, 180-198`）只確認片子存在就回 `{status:"reparse_queued"}`。
3. **「修改資訊」改完徽章還是「失敗」。** `metadata_edit_service.go:151, 243` 只設 `metadata_source=manual`，`parse_status` 原封不動寫回。**已經有片子卡在 `manual`＋`failed`**（編輯過的失敗片），批次比對只撿 pending／空狀態（`enrichment_service.go:457-469`），永遠不會修好它們。
4. **自動比對會蓋掉使用者的資料。** 比對成功路徑完全不看 `ShouldOverwrite`（只有演員表 `enrichment_service.go:413` 與 NFO `:678` 有擋）；掃描器發現檔案變動就把狀態打回 pending（`scanner_service.go:478-495`），批次重新解析也會（`library_service.go:780-816`），下次比對直接覆寫。
5. **重新比對拿的是 `movie.Title`，不是檔名**（`enrichment_service.go:473`）。電影被手動改過或配錯後，標題已不是檔名，重跑只會搜到同一部錯的片。
6. **套用請求沒說「選的結果是電影還是影集」。** `SelectedMetadataItem{id, source}`（`metadata_service.go:132-135`）。`tmdb-550` 在電影與影集兩邊都存在而且是不同的作品。
7. **比對逾時不會回錯誤。** context 取消時 `orchestrator.Search` 回 `(nil, status)` 沒有 error（`metadata/orchestrator.go:278-300`），`SearchMetadata` 把 nil error 當正常（`metadata_service.go:420-434`），`enrichMovie` 最後回報「沒找到」。**`errors.Is(err, context.DeadlineExceeded)` 永遠不會中。**

### 程式碼地圖

```
POST /api/v1/metadata/apply          handlers/metadata_handler.go:218（路由 :453）
  └─ MetadataService.ApplyMetadata   services/metadata_service.go:693-776   ← 空殼
     └─ ApplyMetadataRequest.Validate :146-161（預設 media_type=movie）
POST /api/v1/library/movies/:id/reparse  handlers/library_handler.go:158（路由 :426）← stub
POST /api/v1/library/series/:id/reparse  handlers/library_handler.go:180（路由 :434）← stub
PUT  /api/v1/media/:id/metadata      handlers/metadata_handler.go:301（路由 :459）
  └─ MetadataEditService             services/metadata_edit_service.go:91-180（電影）/ :183-270（影集）
EnrichmentService                    services/enrichment_service.go
  ├─ CancelEnrichment                :110-118（close(cancelChan)）
  ├─ StartEnrichment                 :122（isEnriching :123-138；一開始就載入全部 pending 列 :143-148）
  ├─ enrichMovie                     :472（NFO :476-483 → 解析 :486 → 搜尋 :507-513 → applyMetadataToMovie :866-937）
  ├─ enrichSeries                    :278-321（解析用 context.Background :280；applyMetadataToSeries :324-366）
  ├─ persistFailedWithLocalAnalysis  :561-575（空路徑已能處理：applyFFprobeTechInfo :582-584 提早返回）
  ├─ tryNFOEnrichment                :667-739（已知 tmdb id → GetMovieDetails → applyTMDbMovieDetails :794-841 → 收尾 :709-730）
TMDbService.GetMovieDetails / GetTVShowDetails   services/tmdb_service.go:237 / :264（都有快取）
窄寫入器 UpdateEnrichedMetadata       repository/enriched_metadata_update.go:65-90（電影）/ :128-142（影集）
main.go   metadataService :328-336、enrichmentService :450-459、post-scan 比對 :465-486、
          演員表／詞彙表 :628-641、metadataHandler :875、libraryHandler :890、scannerHandler :917
```

---

## Acceptance Criteria

1. **AC #1 [@contract-v1]：套用使用者選的 TMDb 結果，真的寫進那部片。** `POST /api/v1/metadata/apply`。
   - **請求**：`{ media_id, media_type: "movie"|"series", selected_item: { id, source, media_type: "movie"|"tv" }, learn_pattern? }`。`selected_item.media_type` **新增、必填**（手動搜尋結果本來就帶 `media_type`，`metadata_service.go:105-116`）。
   - **驗證 → 400 `APPLY_METADATA_INVALID_REQUEST`**：`source` 不是 `tmdb`；`id` 不是 `tmdb-<正整數>`；`selected_item.media_type` 缺漏；`media_type=movie` 配 `≠movie`、`series` 配 `≠tv`。message 說清楚是哪一條。
   - **找不到**：`media_id` 不存在 → 404 `APPLY_METADATA_NOT_FOUND`；TMDb 查無該 id → 404 `TMDB_NOT_FOUND`；TMDb 逾時／限流 → 沿用 `TMDB_*` 碼與正確狀態碼。判斷**一律 `errors.As`**（dsr-2 CR #1：TMDb 錯誤被 `%w` 包過一層，型別斷言永遠不中）。
   - **批次比對進行中** → 409 `ENRICHMENT_ALREADY_RUNNING`（呼叫 `IsEnrichmentActive()`；**不要**讓套用去搶 `isEnriching`，見已知陷阱「共用鎖」）。
   - **電影**：`GetMovieDetails(id)` → `applyTMDbMovieDetails`（已存在）；**影集**：`GetTVShowDetails(id)` → 新增 `applyTMDbSeriesDetails`，欄位照 `applyMetadataToSeries`（`:324-366`）：title（優先 zh-TW）、original_title、`tmdb_id`、poster_path、backdrop_path、overview、first_air_date、vote_average、genres。
   - **兩者都**：`parse_status = success`、**`metadata_source = manual`**（使用者親手選的＝優先序 100，NFO 80 與自動比對都蓋不掉；見 AC #4）；用**窄寫入器** `UpdateEnrichedMetadata` 寫（**不准用寬的 `Update`**，`enriched_metadata_update.go:12-41`）；接著跑演員表（`matchedCredits`／`persistCredits`——**演員表要寫進本地資料列**，因為 `LocalDetailV2.tsx:86-90` 對 `manual` 的片改讀本地演員表，沒寫就會空白）與 `touchGlossaryScope`，照 `tryNFOEnrichment:709-730` 的順序。**抽成一個共用函式**讓 NFO 路徑與套用路徑走同一段。
   - **回應 200**：`{ success: true, media_id, media_type, title, source: "tmdb", tmdb_id, parse_status: "success" }`——`title` 是寫進去之後讀回的片名。
   - **`MediaUpdater` 介面與 `SetMediaUpdaters` 刪掉**，換成 setter 注入的套用器（例如 `SetMatchApplier`；跟 `SetMetadataEditors`（`main.go:345`）同一種做法——enrichment 建在 metadata 之後，只能用 setter）。**套用器沒接線時回 500 `APPLY_METADATA_FAILED`，不准再靜默成功。** 寫入資料庫失敗 → 500 `DB_QUERY_FAILED`。
   - `learn_pattern`：**行為不變**（仍是 TODO，`:760`）。
   - **只支援 TMDb**：豆瓣／Wikipedia 沒有「用 id 取細節」，而且預設關閉（`config/config.go:191-192`）。已立 `disc-2026-09-apply-non-tmdb-sources`。

2. **AC #2 [@contract-v1]：單筆重新比對真的會跑。** `POST /api/v1/library/movies/:id/reparse`、`/series/:id/reparse`。
   - `EnrichmentService` 新增公開方法（例如 `EnrichOne(ctx, kind, id)`）：`FindByID` → 比對 → 讀回資料列。用 setter 注入 `LibraryHandler`（`main.go:890` 之後）。**沒注入時回 500**（跟 `scanner_handler.go:170-174` 的 `TriggerEnrich` 同一個做法）。
   - **電影的比對輸入用檔名**（🔴 #5）：`filepath.Base(FilePath)`；`FilePath` 空的時候退回 `Title`（API 建的片沒有路徑——e2e 就是這樣種資料；空路徑的本地分析已能處理，`:582-584`）。在 `enrichMovie` 開一個「解析輸入」的縫，**只有單筆重新比對傳檔名，批次路徑不變**。
   - **影集的比對輸入不變**（仍用 `series.Title`）：影集標題本來就不是檔名——匯入時用分集檔名解析出的片名、只在失敗時退回資料夾名（`media_ingest_service.go:289-302`），而扁平目錄下 `SeriesDirFor` 會回傳媒體庫根目錄（搜尋字會變成 `tv`）。
   - **影集解析改吃 ctx**：`enrichSeries:280` 用 `ParseFilename`（底層 `context.Background`）→ 改用帶 ctx 的版本，否則 60 秒上限對影集無效。
   - **`metadata_source = manual` 的片**：**不搜尋、不覆寫**，只做 AC #4 的「本地檔案重新分析＋設回 success」，回 200。理由：媒體庫批次重新解析會把 manual 片打回 pending（沒人會再跑它），而詳情頁會對 pending 片顯示「立即比對」——這時按下去必須把它修好，不能永遠 409。
   - **批次比對進行中** → 409 `ENRICHMENT_ALREADY_RUNNING`（呼叫 `IsEnrichmentActive()`，不搶鎖）。
   - **同步執行、60 秒上限**（`context.WithTimeout`）。**逾時判斷用比對結束後的 `ctx.Err()`**，不是看回傳的 error（🔴 #7）→ 504 `METADATA_TIMEOUT`；資料列維持逾時前最後寫入的狀態。
   - **回應 200 一律帶 `success` 或 `failed`**：`{ id, parse_status: "success"|"failed", title, tmdb_id }`（讀回的資料列）。**比對又失敗也是 200、`"failed"`**——「跑了但沒找到」是結果，不是錯誤。寫入資料庫失敗 → 500 `DB_QUERY_FAILED`（**不准**回 200 帶舊的 `pending`）。
   - **Swaggo 註解**：兩條路由補 `@Summary`／`@Param`／`@Success`／`@Failure`／`@Router`。⛔ **不要跑 `swag init`**：`docs/swagger.json` 只有 6 條舊路徑、程式碼有 70 個 `@Router`，重生會帶出巨大的無關 diff。已立 `disc-2026-09-swagger-docs-stale`。
   - 💰 **AI 花費**：重新比對可能觸發 AI 檔名解析與 AI 關鍵字。依 `ai_service.go:1-10` 的既有裁定（sub-5-1 AC #4），解析路徑**刻意不計量**，而且結果依檔名快取 30 天——同一個檔名重跑拿到同一個答案。**本張不改這個裁定、不加 Budget。**

3. **AC #3：「修改資訊」存檔後，片子不再是「失敗」。**
   - `PUT /api/v1/media/:id/metadata` 的電影與影集兩條路徑都設 `parse_status = success`（`metadata_source = manual` 本來就有）。
   - **一次性資料修正**：新 migration（下一號是 `038`）把 `metadata_source = 'manual' AND parse_status IN ('failed', 'pending')` 的電影與影集設成 `success`（🔴 #3 那批卡住的片）。migration 要有測試。
   - ⚠️ 已知不一致（**不在本張修**）：「未匹配」篩選看 `tmdb_id`（`movie_repository.go:377-378, 569`；`series_repository.go:370, 590`），不是 `parse_status`。已立 `disc-2026-09-unmatched-filter-vs-parse-status`。
   - 測試：電影、影集各一條「failed → 編輯 → success」。

4. **AC #4：自動比對不再蓋掉你選的或你填的資料。**
   - `enrichMovie` 與 `enrichSeries`：**在寫入之前重新從資料庫讀一次 `metadata_source`**（不能只看批次一開始載入的舊副本——批次可能跑好幾分鐘，使用者在這期間編輯或套用的片會被蓋掉）。是 `manual` → **不搜尋結果、不覆寫 metadata**。
   - manual 片的處理：**先跑本地檔案分析**（`applyFFprobeTechInfo`＋字幕偵測，跟 `persistFailedWithLocalAnalysis:561-575` 同一段——掃描器打回 pending 正是因為檔案變了，解析度、編碼、字幕軌要更新），再設 `parse_status = success`，用窄寫入器寫，記一行 `slog.Info`。**抽成共用函式**，AC #2 的 manual 分支也呼叫它。
   - 實作建議：在比對**開頭**讀一次（manual 就直接走上面那段，省掉搜尋），並在**寫入前**再讀一次（擋住「比對進行中使用者剛好編輯」）。兩次讀之間的空窗只剩寫入前那幾毫秒，記進 Dev Notes。
   - 測試（**要打真的 SQLite，不能只 mock repo**——Rule 15 的 bugfix-20-1 判例）：
     - ① 一部 `manual` 的片 → 打回 pending → 跑批次比對 → 片名、tmdb_id、海報沒變、狀態 success、技術資訊有重新分析（用假 ffprobe 驗證有被呼叫）；
     - ② **批次載入快照之後**才把片改成 `manual`（在測試裡用 stub 搜尋器的回呼插入這一步）→ 批次寫入前讀到 manual → 沒覆寫。
   - ⚠️ 已知取捨：使用者把檔案換成**另一部電影**時，manual 鎖會保留舊資料；解法是再手動選一次。已立 `disc-2026-09-manual-lock-survives-file-replacement`。

5. **main.go 接線（Rule 15）。**
   - 套用器在演員表與詞彙表接好**之後**注入 `metadataService`（`:628-641` 之後），而且**不能放進 `if creditsClient != nil` 區塊裡**——沒有演員表 client 的環境也要能套用。
   - enrichment 注入 `libraryHandler`（`:890` 之後）。
   - `grep` 確認三條路由仍由 `RegisterRoutes` 註冊、方法與路徑沒變。

6. **錯誤碼（Rule 7）。** 本張**不新增任何錯誤碼**。`APPLY_METADATA_*` 與 `ENRICHMENT_ALREADY_RUNNING` 是**既有的舊字面值**（前綴不在 Rule 7 的 17 個權威前綴裡），**照原樣重用、不另定義新常數**；`TMDB_*`、`METADATA_TIMEOUT`、`DB_QUERY_FAILED` 在權威清單裡。CR 的 Rule 7 檢查若對舊字面值報錯，以本條為準（記進 Completion Notes）。

7. **測試。**
   - **服務層**：套用電影、套用影集、類型不符、`selected_item.media_type` 缺漏、來源不支援、id 格式錯、TMDb 404（**包過一層與兩層**）、TMDb 逾時、批次進行中、套用器未接線、寫入失敗；單筆重新比對成功、又失敗（200＋failed）、manual 片（200＋success＋本地分析）、批次進行中、**真的逾時**（stub 搜尋器等到 ctx 到期；**不准**只餵一個包好的 `DeadlineExceeded` 錯誤——那樣會假綠）、空 `FilePath`、寫入失敗 → 500、未注入 → 500。
   - **handler 層**：每個狀態碼與錯誤碼各一條（Rule 16：斷言真正的 `code` 與 HTTP 狀態）。
   - **整合測試（真 SQLite）**：套用後**讀回** `tmdb_id`／`title`／`poster_path`／`parse_status`／`metadata_source`／演員表；AC #3 migration；AC #4 兩條。
   - **e2e API**（`tests/e2e/manual-search.api.spec.ts`）：
     - `:246`（電影 `tmdb-550`）、`:270`（影集 `tmdb-1396`）、`:313`（NOT_FOUND，**解除 skip**）、`:342`（learn_pattern）四條請求都補 `selected_item.media_type`。`:246`／`:270` 的斷言改成**讀回片子**確認 `tmdb_id`、`parse_status=success`、片名不是 `Unknown`。CI 的 e2e 後端有 `TMDB_API_KEY`（`.github/workflows/test.yml:428`）。
     - 新增：類型不符 → 400；重新比對一部「亂碼標題、無路徑」的片 → 200、`parse_status: "failed"`（這也是 `dsr-2b-b` 在 e2e 種「失敗」片的方法）。
   - `tests/support/helpers/api-helpers.ts:496` 的 `applyMetadata` 型別補 `media_type`；新增 `reparseMovie`／`reparseSeries` helper，**遇到 409 自動重試幾次**（e2e 在 chromium 與 webkit-core 兩個專案平行跑、共用同一個後端，`playwright.config.ts:120-130`）。

8. **既有測試保留通過**，只有下列是本張刻意改的：
   - `metadata_service_test.go`：`ApplyMetadata_*`（`:820, 855, 882, 901, 929`——`_NoUpdater` 要反過來斷言 500）、`mockMediaUpdater`（`:740-757`）、`TestApplyMetadataRequest_Validate_*`（`:760-805`，含 `_DefaultsMediaType`）
   - `metadata_handler_test.go`：`ApplyMetadata_*`（`:904, 956, 988, 1009, 1027, 1065, 1105, 1129`）
   - `library_handler_test.go`：`TestLibraryHandler_ReparseMovie`（`:331`）
   - 上面點名的 e2e
   - **`TestMetadataService_ManualSearch_ResultIDFormat`（`:669`）鎖住的 `tmdb-<id>` 格式不准改；批次重新解析（`TestLibraryService_BatchReparse_*`）行為不准改。**

9. **CI 全綠**：`pnpm nx test api`、`pnpm run lint:all`、`pnpm nx test web`、e2e API spec。⛔ 絕不用 `run_in_background` 跑測試。

## Tasks / Subtasks

- [x] **Task 1 — 共用的「用已知 TMDb id 套用」與「manual 只重新分析檔案」（AC: #1, #4）**
  - [x] 先寫紅測試（真 SQLite 讀回）
  - [x] 從 `tryNFOEnrichment:709-730` 抽出共用收尾；新增 `applyTMDbSeriesDetails`；NFO 路徑改呼叫它
  - [x] manual 分支共用函式（本地分析＋success）；`enrichMovie`／`enrichSeries` 開頭與寫入前各讀一次
- [x] **Task 2 — 套用端點（AC: #1, #5, #6）**
  - [x] 請求加 `selected_item.media_type`、驗證、錯誤碼（先紅；含 Validate 測試改寫）
  - [x] 刪 `MediaUpdater`／`SetMediaUpdaters`，setter 注入套用器；main.go 接線（不放進 `creditsClient` 區塊）
- [x] **Task 3 — 單筆重新比對（AC: #2, #5）**
  - [x] `EnrichOne`＋電影檔名輸入縫；影集解析吃 ctx
  - [x] manual 分支、409、`ctx.Err()` 判逾時、寫入失敗 500、未注入 500（先紅，含真逾時測試）
  - [x] handler、main.go 注入、Swaggo 註解（不跑 `swag init`）
- [x] **Task 4 — 修改資訊（AC: #3）**
  - [x] 電影／影集編輯設 success（先紅）
  - [x] migration 038＋測試
- [x] **Task 5 — e2e 與收尾（AC: #7, #8, #9）**
  - [x] `manual-search.api.spec.ts` 四條改寫＋解除 skip＋兩條新增；helper（含 409 重試）
  - [x] 全套閘門
  - [x] 收單時：關 `wire-set-media-updaters-test-harness`；在 `dsr-2b-b` 條目標「後端已就緒」

## Dev Notes

### 這張的重點

- **最有感的是 AC #1。** 今天「套用」回傳成功卻什麼都沒寫——這是會讓人以為修好了的謊。
- **AC #4 是 AC #1 與 AC #3 的保險。** 沒有它，你選好的片在下次檔案變動或批次重新解析之後會被蓋回去。
- **本張不碰 `apps/web`。** 前端型別、`titleZhTW` 大小寫 bug、套用後不刷新，全部在 `dsr-2b-b`。

### 不要做的事

- **不要用寬的 `movieRepo.Update`／`seriesRepo.Update` 寫比對結果**（`bugfix-wide-update-stale-copy-other-callers` 判例）。
- **不要讓套用器沒接線時靜默成功。**
- **不要支援豆瓣／Wikipedia 的套用。**
- **不要改 `tmdb-<id>` 結果 id 格式**、不要改手動搜尋端點。
- **不要替解析路徑加 AI Budget**（`ai_service.go:1-10`）。
- **不要改批次重新解析**（`library_service.go:780-816`，只設 pending 是 `LibraryBrowseV2` 批次按鈕的既有契約）。它「設完沒人跑」記在 `disc-2026-09-batch-reparse-never-runs`。
- **不要改批次比對的輸入**（仍用標題）——檔名輸入只給單筆重新比對。
- **不要處理重複 `tmdb_id`**（同一部電影 1080p＋4K 兩個檔案共用 id 是合法的）。
- **不要碰 `learn_pattern`**（`disc-2026-09-learn-pattern-never-pins-id`）。
- **不要跑 `swag init`**（AC #2）。
- **不要新增錯誤碼**（AC #6）。

### 已知陷阱

- **共用鎖。** 不要讓單筆操作去設 `isEnriching`：
  - `CancelEnrichment` 會 `close(s.cancelChan)`（`:110-118`），單筆操作期間那個 channel 是 nil 或已關閉 → **panic**；
  - 掃描完的自動批次（`main.go:465-486`）在單筆操作期間會拿到 ALREADY_RUNNING，**那次掃描的比對就被默默丟掉**。
  - 所以：單筆操作只**檢查** `IsEnrichmentActive()`，不設旗標；批次與單筆同時寫同一部片的風險由 AC #4 的「寫入前重讀」擋掉。
- **TMDb 錯誤會被包過一層**，一律 `errors.As`，handler 測試餵包一層與兩層的錯誤。
- **逾時不會變成 error**（🔴 #7）。用 `ctx.Err()`。
- **`applyMetadataToMovie` 只接受 TMDb 電影結果寫 `tmdb_id`**（`tmdbIDFromMatch`，`:981-989`）——套用路徑直接拿使用者給的 id，不要繞回這個函式。
- **換片會留下舊資料**（imdb、runtime、backdrop、`douban_id`、詞彙表 scope）。本張的對象是沒資料的片，舊值通常是空的。`disc-2026-09-rematch-leaves-stale-fields`。
- **NFO 會短路重新比對**（`:476-483`），錯的 NFO 救不回來；manual 鎖在 NFO 之前生效。`disc-2026-09-wrong-nfo-short-circuits-rematch`。
- **使用者上傳的海報會被 TMDb 海報取代**（`metadata_edit_service.go:293-305`）。`dsr-2b-b` 的確認文案會說清楚。
- **AI 解析結果依檔名快取 30 天**：因 AI 解析錯而失敗的片，重新比對通常還是失敗；重新比對真正有用的是「上次 TMDb 或網路暫時出錯」與「還沒跑過的 pending 片」。
- **`Exists` 查兩張表、`UpdateMetadata` 依 media_type 分支**：影集 id 配 `media_type=movie` 會 500。`disc-2026-09-metadata-edit-type-mismatch-500`。
- **行號以建單時（2026-09-17，main `2a172a3a`）為準**，動手前重新確認。

### Source tree

```
apps/api/internal/services/enrichment_service.go        ← Task 1, 3（主要）
apps/api/internal/services/metadata_service.go          ← Task 2
apps/api/internal/services/metadata_edit_service.go     ← Task 4
apps/api/internal/handlers/metadata_handler.go          ← Task 2
apps/api/internal/handlers/library_handler.go           ← Task 3
apps/api/internal/database/migrations/038_*.go（＋registry.go）← Task 4
apps/api/cmd/api/main.go                                ← Task 2, 3
apps/api/internal/**/*_test.go                          ← 各 Task
tests/e2e/manual-search.api.spec.ts                     ← Task 5
tests/support/helpers/api-helpers.ts                    ← Task 5
```

### Cross-Stack Split Check

後端 task **5 個**、前端 task **0 個**（前端全部在 `dsr-2b-b`）。前端 ≤ 3 → **本張不再拆**。

### Time-dependent visual coverage

- **N/A — no wall-clock-reading components touched.** 本張不改任何 `apps/web/src/components/**`。

### References

- [Source: `apps/api/internal/services/metadata_service.go:105-116, 132-161, 164-170, 301-304, 420-434, 573-658, 682, 693-776`] — 手動搜尋結果、套用請求／回應、空殼分支、搜尋吞掉取消
- [Source: `apps/api/internal/metadata/orchestrator.go:278-300`] — 取消時回 `(nil, status)`
- [Source: `apps/api/internal/handlers/metadata_handler.go:144-265, 301`、`library_handler.go:158-198, 426, 434`、`scanner_handler.go:170-185`] — 路由、錯誤碼、stub、未注入時的做法
- [Source: `apps/api/internal/services/enrichment_service.go:110-148, 278-366, 399-419, 457-472, 476-486, 507-513, 561-584, 667-739, 794-841, 866-937, 981-989`] — 比對內部
- [Source: `apps/api/internal/services/tmdb_service.go:237, 264`、`repository/enriched_metadata_update.go:12-41, 65-90, 128-142`] — 取資料、窄寫入器
- [Source: `apps/api/internal/services/metadata_edit_service.go:61-88, 91-180, 183-270, 293-305`] — 修改資訊
- [Source: `apps/api/internal/services/library_service.go:780-816`、`scanner_service.go:478-510`、`media_ingest_service.go:115-126, 289-302`] — 打回 pending 的兩條路、影集標題來源
- [Source: `apps/api/internal/models/movie.go:12-47`] — `ParseStatus` 值、`metadataSourcePriority`／`ShouldOverwrite`
- [Source: `apps/api/internal/services/ai_service.go:1-10`] — 解析路徑不計量的裁定
- [Source: `apps/web/src/components/media/LocalDetailV2.tsx:86-90`] — manual 片讀本地演員表
- [Source: `apps/api/cmd/api/main.go:328-346, 450-486, 628-641, 875, 890, 917`] — 接線位置
- [Source: `tests/e2e/manual-search.api.spec.ts:240-360`、`tests/support/helpers/api-helpers.ts:496`、`playwright.config.ts:120-130`、`.github/workflows/test.yml:428`] — e2e
- [Source: `_bmad-output/implementation-artifacts/dsr-2-flow-b-detail-v2.md` 對抗式 CR #1] — `errors.As` 判例
- [Source: project-context.md#Rule 7 / #Rule 15 / #Rule 16 / #Rule 20 / #Rule 24 / #Rule 27]
- [Source: `sprint-status.yaml` → `wire-set-media-updaters-test-harness` / `dsr-2b-b-no-metadata-states-frontend` / `backlog-parse-path-ai-metering` / `bugfix-wide-update-stale-copy-other-callers`]

## Dev Agent Record

### Agent Model Used

Claude Opus 5 (1M context) — `claude-opus-5[1m]`，BMAD dev agent（Amelia）2026-09-17

### Debug Log References

- `pnpm nx test api --skip-nx-cache`：**PASS**（exit 0，無任何 FAIL）
- `pnpm nx test web`：**3461 / 3461 passed**（260 files；本張沒動 `apps/web`）
- `pnpm run lint:all`：**0 errors**（web 128 warnings 既有、root 126 warnings 既有）；prettier：PASS；`go vet ./...`：PASS
- **e2e API 對真的後端＋真的 TMDb 跑**：本機建 `vido-api`，`VIDO_DATA_DIR` 指到 scratchpad、TMDb key 從 `.env` 讀，`CI=1 npx playwright test tests/e2e/manual-search.api.spec.ts --project=chromium` → **17 / 17 passed**（含解除 skip 的 NOT_FOUND、類型不符 400、讀回資料列、重新比對亂碼片 → 200 failed）。
- **手動煙霧測試（curl，同一個本機後端）**：API 建一部 `Fight.Club.1999.1080p.BluRay.x264.mkv` → `POST /library/movies/:id/reparse` → `{"parse_status":"success","title":"鬥陣俱樂部","tmdb_id":550}`，讀回有海報；接著 `PUT /media/:id/metadata` 改片名 → 再重新比對 → 回 `success` 且片名仍是「我自己填的」（manual 鎖生效）。
- ⚠️ 本機跑 API 時有個元件寫了 repo 根目錄的 `data/vido.db`（沒吃 `VIDO_DATA_DIR`）；是本次手動測試的副產品，已刪除、沒進 commit。
- ⚠️ `gofmt` 陷阱：doc comment 裡的 `''` 會被 gofmt 改成 `”`。migration 038 的註解改寫成「empty-status」避開。
- **變異測試**（把修正拿掉確認測試會紅）：拿掉 manual 判斷、拿掉 `ctx.Err()` 判逾時、影集改回不帶 ctx 的解析 → 7 條測試紅；逾時那條原本不會紅（過期 ctx 讓寫入也失敗，錯誤鏈裡剛好也有 `DeadlineExceeded`），補上「不是 `ErrEnrichPersist`」的斷言後會紅。

### Completion Notes List

- Ultimate context engine analysis completed - comprehensive developer guide created
- 🔗 **AC Drift: FOUND**（`grep -rln "reparse_queued\|metadata/apply\|ApplyMetadata"` 與 `grep -rn "reparse"` 全部 story 檔）
  - `3-7-manual-metadata-search-and-selection` AC3「Selection and Application」→ 請求多了必填的 `selected_item.media_type`、只接受 TMDb 結果；回應多 `tmdb_id`／`parse_status`；**真的寫進資料列**（原本在正式環境是空操作）。
  - `5-1-media-library-grid-view` Task 8.5／10（單筆 reparse 端點）→ 回應從 `{id, status:"reparse_queued"}` 改成 `{id, parse_status, title, tmdb_id}`，而且真的會比對。
  - `sub-7-3`（掃描比對時種詞彙表）的測試 `TestEnrichMovie_ManualSourceOutranksMatch_CreditsKeptButGlossaryStillSeeds` → **改寫**成 `TestEnrichMovie_ManualSource_NeverSearched_GlossaryStillResolves`：manual 的片不再被搜尋與覆寫（原本片名與海報會被蓋掉、只保留演員表——正是本張 AC #4 要修的 bug）；詞彙表 scope 照舊會 resolve（`refreshManual*` 裡保留 `touchGlossaryScope`）。這條不在 AC #8 的清單上，但它斷言的正是 AC #4 禁止的行為。
  - `story-20-2-e2e-test-data-seeding` → 它記錄的「apply 是空操作、NOT_FOUND 測試只能 skip」已不成立；e2e 註解已更新、skip 解除。
  - `3-8-metadata-editor` → 存檔後 `parse_status` 變 `success`（原本不動）。
- 📎 **Contract Stamps: FOUND**（本張 v1×2：AC #1 套用請求／回應、AC #2 重新比對回應；程式碼註解同步標在 `ApplyMetadataRequest.Validate`、`ApplyMetadataResponse`、`EnrichedItem`。上游 3-7、5-1 沒有 stamp＝implicit v0。下游 `dsr-2b-b` 已有 `confirmed against [@contract-v1] (Story dsr-2b-a AC #1)`／`AC #2` 兩行。）
- 🎭 **A11y Pre-Flight: N/A**（100% backend——沒有 `apps/web/` 檔案；只動 `tests/` 底下的 e2e helper）
- 🎨 **UX Verification: SKIPPED — no UI changes in this story**
- ✅ **Pre-existing failures: NONE**
- **錯誤碼（AC #6）**：沒有新增任何碼。`APPLY_METADATA_*` 與 `ENRICHMENT_ALREADY_RUNNING` 是前綴不在 Rule 7 權威清單裡的舊字面值，照原樣重用（handler 註解已標明）；`TMDB_*`、`METADATA_TIMEOUT`、`DB_QUERY_FAILED`、`DB_NOT_FOUND` 在清單裡。
- **manual 鎖的實作**（AC #4）：`enrichMovieFrom`／`enrichSeries` 開頭重讀資料列（批次快照可能是幾分鐘前的），是 manual 就走 `refreshManualMovie`／`refreshManualSeries`（電影重跑本地分析；影集沒有技術欄位，只設 success）；每一個寫入點前再用 `userTookOver*` 重讀一次——成功寫入、`persistFailedWithLocalAnalysis`、NFO 路徑、影集三個寫入點都擋。兩次讀之間的空窗只剩寫入前那幾毫秒。重讀失敗（mock 或資料庫暫時讀不到）時照舊寫入，不因為讀不到就失敗。
- **manual 片會真的重新 probe**（CR #2 更正）：`applyFFprobeTechInfo` 在 `VideoCodec` 已有值時整段跳過（連字幕重新偵測也跳過）。原本的註記「字幕軌會重新偵測」在有裝 ffmpeg 的正式環境**不成立**（當時的測試環境沒有 ffprobe 才會綠）。`refreshManualMovie` 現在在 ffprobe 可用時先清掉 `VideoCodec` 再 probe、probe 不出來就還原舊值；用 PATH 上的假 ffprobe 測試守住。非 manual 片的既有跳過規則沒改。
- **單筆操作不搶批次旗標**：`EnrichOne`／`ApplyTMDbMatch` 只檢查 `IsEnrichmentActive()`。測試 `TestEnrichOne_RefusedWhileBatchRuns_AndDoesNotTakeTheBatchLock` 同時守住「單筆進行中 `IsEnrichmentActive()` 是 false、`CancelEnrichment` 不 panic」。
- **逾時判斷**（AC #2）：`EnrichOne` 在比對結束後**先看 `ctx.Err()`** 才看寫入錯誤——過期的 ctx 會讓寫入本身也失敗，那是逾時，不是儲存故障。handler 同樣先比對 `context.DeadlineExceeded`。
- **200 的保證**：`EnrichOne` 讀回資料列後，狀態不是 `success`／`failed` 就回 `ErrEnrichPersist`（500），不會回 200 帶 `pending`。
- **影集套用的欄位**：新增的 `applyTMDbSeriesDetails` 用 `details.Name`（TMDb 服務以 zh-TW 查詢），欄位集合跟 `applyMetadataToSeries` 一致。
- **main.go**：`metadataService.SetMatchApplier(enrichmentService)` 放在演員表／詞彙表接好之後、`if creditsClient != nil` 區塊**外面**；`libraryHandler.SetItemEnricher(enrichmentService)` 緊接在 handler 建立之後。本機啟動 log 有「Match applier configured for metadata service」。
- **Swagger**：只補註解（兩條 reparse 路由原本沒有；apply 補 409／504 與描述），**沒跑 `swag init`**（`disc-2026-09-swagger-docs-stale`）。

### Discovery Triage

- **Did this story discover any work outside its current scope?** **YES** —— 建單當下處理如下。

- **① expand-scope-in-place**
  - `wire-set-media-updaters-test-harness` → **AC #1**（介面形狀也不對；收單時關掉）
  - 影集解析用 `context.Background` 取消不了 → **AC #2**
  - 手動改過的資料會被自動比對蓋掉（含批次進行中的競態）→ **AC #4**
  - 已經卡在 `manual`＋`failed` 的片 → **AC #3** 的 migration

- **② spawn-blocking-story**：無（`dsr-2b-b` 依賴本張，不是本張依賴它）。

- **③ backlog-with-carry-forward-link**
  - **`disc-2026-09-apply-non-tmdb-sources`** — 豆瓣／Wikipedia 結果不能套用。
  - **`disc-2026-09-unmatched-filter-vs-parse-status`** — 「未匹配」看 `tmdb_id`、徽章看 `parse_status`。
  - **`disc-2026-09-rematch-leaves-stale-fields`** — 換配對時舊欄位殘留。
  - **`disc-2026-09-duplicate-tmdb-id-series-ingest`** — 兩筆影集同 tmdb id 時新分集掛到任意一筆。
  - **`disc-2026-09-wrong-nfo-short-circuits-rematch`** — 錯的 NFO 讓重新比對救不回來。
  - **`disc-2026-09-learn-pattern-never-pins-id`** — `learn_pattern` 是 TODO；`LearnedTmdbID` 沒人用。
  - **`disc-2026-09-metadata-retry-queue-never-writes`** — 重試佇列成功了也不寫回。
  - **`disc-2026-09-batch-reparse-never-runs`** — 批次重新解析設完 pending 沒人跑。
  - **`disc-2026-09-metadata-edit-type-mismatch-500`** — 影集 id 配 movie 回 500。
  - **`disc-2026-09-swagger-docs-stale`**（建單驗證時立）— `docs/swagger.json` 只有 6 條舊路徑，程式碼有 70 個 `@Router`。
  - **`disc-2026-09-manual-lock-survives-file-replacement`**（建單驗證時立）— 檔案換成另一部片時 manual 鎖保留舊資料。

- Reference: `project-context.md` Rule 24

### File List

**新增：**
- `apps/api/internal/services/enrichment_manual_match.go` — `MediaKind`、`EnrichedItem` [@contract-v1]、`ItemEnricherInterface`、`MatchApplier`、`ApplyTMDbMatch`、`EnrichOne`、manual 鎖（`freshMovie`／`userTookOver*`／`refreshManual*`）、共用收尾 `persistMovieMatch`／`persistSeriesMatch`、`applyTMDbSeriesDetails`、錯誤 `ErrEnrichItemNotFound`／`ErrEnrichmentAlreadyRunning`／`ErrEnrichPersist`
- `apps/api/internal/services/enrichment_manual_match_test.go` — 24 個測試函式（/ship CR 後補 7 個；其中 TMDb 錯誤那條有兩個子測試；真 SQLite：套用電影／影集讀回、TMDb 錯誤包一層兩層、批次中拒絕；manual 片批次不覆寫＋重新偵測字幕、批次中途變 manual 的成功／失敗兩條、manual 影集；單筆：檔名輸入、無路徑退回標題＋沒找到是結果、manual 只刷新、影集用標題＋帶 ctx、找不到、批次中拒絕且不搶旗標、真逾時、寫入失敗）
- `apps/api/internal/database/migrations/038_manual_rows_parse_status_success.go`（＋`_test.go`）— 卡在 manual＋failed／pending 的片改 success

**修改：**
- `apps/api/internal/services/enrichment_service.go` — `enrichMovie` → `enrichMovieFrom`（解析輸入縫）；開頭重讀＋manual 分支；所有寫入點前的 manual 閘門；寫入錯誤包 `ErrEnrichPersist`（原本兩處 `_ =` 吞掉）；`enrichSeries` 改 `ParseFilenameWithContext`；NFO 收尾改走 `persistMovieMatch`
- `apps/api/internal/services/metadata_service.go` — `SelectedMetadataItem.MediaType`；`Validate` 新規則；`ApplyMetadataResponse` 加 `tmdb_id`／`parse_status`；五個驗證錯誤＋`IsApplyMetadataInvalidRequest`；刪 `MediaUpdater`／`SetMediaUpdaters`；新 `SetMatchApplier`；`ApplyMetadata` 改交給套用器
- `apps/api/internal/services/metadata_service_test.go` — apply 測試整段改寫（mock 套用器、驗證表格、未接線是錯誤、錯誤直通）
- `apps/api/internal/services/metadata_edit_service.go`（＋`_test.go`）— 電影／影集存檔設 success；mock 與真 SQLite 各一條
- `apps/api/internal/services/enrichment_glossary_seed_test.go` — 改寫 sub-7-3 那條 manual 測試（見 AC Drift）
- `apps/api/internal/handlers/metadata_handler.go`（＋`_test.go`）— 請求帶 `selected_item.media_type`；錯誤對應（400／404／409／TMDb `errors.As`／500 DB／500 未接線）；Swaggo 註解
- `apps/api/internal/handlers/library_handler.go`（＋`_test.go`）— `SetItemEnricher`、60 秒上限、兩條 reparse 共用 `reparse`（504／404／409／500、未注入 500）；Swaggo 註解
- `apps/api/cmd/api/main.go` — `SetMatchApplier`、`SetItemEnricher` 接線
- `tests/e2e/manual-search.api.spec.ts` — apply 區塊改寫（讀回資料列、類型不符、解除 NOT_FOUND skip）＋新增重新比對區塊
- `tests/support/helpers/api-helpers.ts` — apply 型別、`ReparseResponse`、`reparseMovie`／`reparseSeries`（409 重試）
- `_bmad-output/implementation-artifacts/sprint-status.yaml` — 狀態流轉；關 `wire-set-media-updaters-test-harness`；`dsr-2b-b` 補註
- `_bmad-output/implementation-artifacts/dsr-2b-a-manual-match-backend.md` — 本檔
- `_bmad-output/implementation-artifacts/3-7-manual-metadata-search-and-selection.md`、`5-1-media-library-grid-view.md`、`story-20-2-e2e-test-data-seeding.md`、`3-8-metadata-editor.md`（AC drift reference — see Completion Notes；未修改）

## 對抗式 Code Review（/ship，2026-09-17）

獨立 reviewer（fresh context，只讀）回報 **0 HIGH、2 MED、6 LOW**。**修 7、立案 1（附在既有條目）**。每一條修正都先寫測試，並在「把修正拿掉」的程式碼上確認會紅（5 條新測試全紅）。

| # | 等級 | 問題 | 處置 |
| --- | --- | --- | --- |
| 1 | MED | 失敗路徑先檢查「使用者是否接手」、**再** probe：probe 在要喚醒硬碟的 NAS 上可能要好幾秒，使用者在那幾秒存檔，舊的「失敗」還是會蓋掉（片名變回檔名、manual 鎖消失） | 檢查移到 probe 之後、寫入之前；測試在檢查當下確認本地分析已跑完 |
| 2 | MED | manual 片的「重新分析檔案」在正式環境（有 ffmpeg、已有 codec）其實什麼都不做；測試會綠只因為測試環境沒有 ffprobe | 清掉 codec 後重新 probe（失敗還原）；PATH 上放假 ffprobe 測試 |
| 3 | LOW | 批次開始後才被處理的片（例如單筆重新比對剛配好），批次仍會用它的**新片名**重新解析，可能把對的配對換掉 | 批次每筆先重讀，已不是 pending／空狀態就跳過（計入 skipped）；測試 |
| 4 | LOW | 逾時落在「寫入之後」（演員表／詞彙表那段）時，已經成功的重新比對會回 504，前端不會刷新 | 逾時時用獨立的短 context 讀回；本次有寫入且狀態是 success／failed 就回 200；測試 |
| 5 | LOW | reparse 的預設錯誤分支把「使用者離開頁面」「沒接線」都標成 `DB_QUERY_FAILED` | 只有 `ErrEnrichPersist` 是 `DB_QUERY_FAILED`；`context.Canceled` 記 Info、其他 `INTERNAL_ERROR`；測試 |
| 6 | LOW | 套用時如果新配對拿不到演員表（沒有 credits client 或抓失敗），舊的錯片演員表會留在 manual 片上，詳情頁會顯示錯的演員 | 拿不到就寫空演員表，詳情頁改用新 tmdb id 的即時演員表；測試。**另一半**（套用期間兩次網路呼叫的空窗，並行的重新比對寫的技術欄位可能被舊副本蓋回）→ 附在 `disc-2026-09-rematch-leaves-stale-fields` |
| 7 | LOW | 測試缺口：未接線只看 `success:false`；服務層沒測套用 TMDb 逾時與寫入失敗；migration「冪等」測試其實沒重跑 SQL | 補 code 斷言、兩條服務層測試、直接對 tx 再跑一次 `Up` |
| 8 | LOW | `/test/manual-search` 開發頁的套用現在會 400（沒送 `selected_item.media_type`） | 刻意的：只掛在開發頁與夾具，`tests/e2e/manual-search.spec.ts` 沒有真的套用；`dsr-2b-b` 的新對話框會送 |

## Change Log

| 日期 | 內容 |
| --- | --- |
| 2026-09-17 | 🔍 **/ship 對抗式 CR**：0 HIGH／2 MED／6 LOW，修 7、附案 1（見上表）。最重要的兩條：① 失敗路徑的「使用者是否接手」檢查放錯位置，probe 期間的存檔還是會被蓋掉；② manual 片的「重新分析檔案」在有 ffmpeg 的正式環境其實什麼都沒做，原本的完成註記寫錯了（已更正）。修完後 api 全綠、lint 0 errors、e2e API 17/17（重建後端再跑一次）。 |
| 2026-09-17 | ✅ **dev-story 完成 → review**（Amelia）。api 全綠、web 3461/3461、lint 0 errors、e2e API 17/17（真後端＋真 TMDb）。最有感的三件：① **「套用」真的寫進去了**——選 `tmdb-550` 之後資料列就是鬥陣俱樂部、有海報、有演員、狀態成功，而且記成你親手選的；② **「重新比對」真的會跑**，用檔名重新解析，60 秒內回來，沒找到也老實說 `failed`；③ **你填的或你選的，自動比對不再蓋掉**——連批次比對跑到一半你剛好存檔的情況都擋住。另外修改資訊存檔會清掉「失敗」，並用 migration 038 修好已經卡住的片。一條 sub-7-3 的舊測試斷言的正是被修掉的 bug，已改寫並記在 AC Drift。 |
| 2026-09-17 | 🔍 **建單後對抗驗證**（fresh-context 驗證代理，只讀）：本張相關 5 項 CRITICAL、6 項 SHOULD FIX，**全部併入**。最重要的三項：① 原本對 manual 片的重新比對回 409——但批次重新解析會把 manual 片打回 pending，詳情頁會一直顯示「立即比對」而永遠 409（改成「只重新分析檔案＋設回 success」，並加 migration 修好已經卡住的片，不再新增錯誤碼）；② 「不蓋 manual」原本只看批次一開始的舊副本，批次跑到一半使用者編輯還是會被蓋（改成寫入前重讀，並加一條「快照之後才改成 manual」的測試）；③ 逾時根本不會變成 error，照原寫法會回 200 failed 而且 mock 測試照樣綠（改用 `ctx.Err()`、要求真逾時測試）。另外：單筆操作不准搶 `isEnriching`（會讓取消 panic、吃掉掃描後的比對）；影集輸入不能用檔名；manual 片也要重新分析檔案；`:313` 與 Validate 測試漏列；不跑 `swag init`；e2e helper 對 409 重試。 |
| 2026-09-17 | Story 建立（SM Bob, create-story）。原本要做的 `dsr-2b` 盤點時查到**後端三條救援路全是空殼**。⚖️ Alexyu 裁定連後端一起修、拆兩張（本張後端、`dsr-2b-b` 前端）。 |

## 裁定紀錄

### ⚖️ 2026-09-17 · dsr-2b 連後端一起修，拆兩張

SM 建單時查到：沒認出來的片在 app 裡沒有任何能用的修法——「手動選片」套用後什麼都沒寫、「重新比對」是空殼、「修改資訊」改完徽章還是「失敗」。只做畫面的話，頁面上的按鈕都是假的。

**Alexyu 選：連後端一起修，拆兩張。** `dsr-2b-a`（本張，後端）→ `dsr-2b-b`（前端，依賴本張）。

落點：本張 AC #1–#4；`dsr-2b-b` 全部。
