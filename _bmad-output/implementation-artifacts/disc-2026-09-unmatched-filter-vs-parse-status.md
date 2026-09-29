# Story: disc-2026-09-unmatched-filter-vs-parse-status — 「未匹配」只算真的找不到資料的片

Status: done

<!-- SM Bob create-story 2026-09-29。⚖️ Alexyu 裁定兩次（同日）：
     ①「完全未匹配的時候才顯示『未匹配』」——任何來源（TMDb／豆瓣／Wikipedia／NFO／手動）拿到資料就不算；
     ② 選 A：剛掃進來、還在「整理中」的片**不算**未匹配。
     建單前開了顧問討論（BMAD Winston／Sally／Amelia，Fable 模型，兩輪互評），結論見 Dev Notes。
     行號為 main `5983668e`，改動前；每條「查到的事」都是 SM 或顧問本次親自讀過的位置（retro-dsr-AI4）。 -->

## Story

身為會按媒體庫「未匹配」篩選來找「要我手動處理的片」的人，
我要這個篩選和它旁邊的數字**只**列出系統真的找不到任何資料的片，
而不是把豆瓣／NFO 找到、我手動改過、或正在整理中的片也算進去。

## 背景（查到的事，附親自讀過的位置）

**兩套判準打架：**

- 「未匹配」篩選看 `tmdb_id`：`apps/api/internal/repository/movie_repository.go:650-652`、`series_repository.go:667-669` → `(tmdb_id IS NULL OR tmdb_id = 0)`。
- 「未匹配 (N)」的 N 也是同一條件：`movie_repository.go:561-572`、`series_repository.go:582-590`（`GetStats`）。前端 `LibraryBrowseV2.tsx:250-251` 相加後給 `FilterPanel.tsx:357-366`。
- `git grep -n 'tmdb_id IS NULL\|tmdb_id = 0' -- apps/api` 只命中上述 4 行（Amelia 驗證）。四個端點 `/library`、`/library/search`、`/movies`、`/series` 都經由這兩個 `*ListFilterConditions`（handler 只轉發：`library_handler.go:115`、`movie_handler.go:79`、`series_handler.go:83`）。
- 海報徽章看 `parse_status`：`apps/web/src/utils/libraryStatus.ts:55-70`（success＝已入庫、pending＝整理中、failed＝失敗）。

**`parse_status` 是「流程狀態」，不是「資料狀態」（Winston 的反例，三位都親自驗證）：**

- 批次重新解析只寫 `parse_status='pending'`（`library_service.go:820,828` → `repository/scan_state_update.go:85-88`）；檔案大小變了也只寫 size＋pending（`scanner_service.go:495` → `scan_state_update.go:68`）。`tmdb_id`／`metadata_source` 原封不動。
- 之後若搜尋失敗（例如 TMDb 暫時斷線），`persistFailedWithLocalAnalysis`（`enrichment_service.go:715-727`）用寬寫 `UpdateEnrichedMetadata`（`enriched_metadata_update.go:70-85` 含 `tmdb_id = ?`、`metadata_source = ?`）把整列連同舊資料寫回並標 `failed`。→ **有資料卻是 failed**。所以不能只看 `parse_status`。

**`metadata_source` 是可靠的「有沒有資料」訊號：**

- 每個寫入 `ParseStatusSuccess` 的路徑都同時寫 `metadata_source`：`converters.go:108,213`（TMDb）、`enrichment_service.go:486,1087-1088`（搜尋命中，含豆瓣／Wikipedia）、`:874-875`（NFO）、`enrichment_manual_match.go:95-96,122-123`（手動選片）、`metadata_edit_service.go:150-153,243-245`（修改資訊）、`media_ingest_service.go:200`、`parse_queue_service.go:227-228`。
- 掃描新建的列不寫來源（`scanner_service.go:504-513`）→ NULL，`parse_status='pending'`。
- `'ai'` 只存在於 parser 層的型別（`parser/types.go:30`、`parser_service.go:318`），`git grep -n 'MetadataSourceAI' -- apps/api ':!*_test.go'` 在 movies／series 表**沒有寫入點**。
- 缺口：`metadata_source` 欄位是 migration 006 才加的（`006_media_entities_enhancement.go:33,52` 只 `ADD COLUMN`，沒回填）。006 之前就入庫、有 `tmdb_id` 的舊列，`metadata_source` 可能是 NULL。

**測試與 seed 釘住舊語意：**

- `movie_repository_test.go:1625-1756`（`TestMovieGetStats`、`TestMovieListUnmatchedFilter` 用 tmdb NULL／0 造「未匹配」）、`:2400-2460`（FTS 的 `subtitle_status + unmatched`，`fts-nf-unmatched` 用 tmdb 0）。
- `series_repository_test.go:1171-1260`（`TestSeriesGetStats`、`TestSeriesListUnmatchedFilter`）。
- `cmd/seed/main.go:227-230`：「Unmatched」示範列是 2 筆 pending＋1 筆 failed、無來源。新判準下只有那筆 failed 算未匹配，2 筆 pending 是「整理中」——正是裁定 ② 要的示範。
- E2E 用假資料（`tests/support/helpers/library-stubs.ts:93`、`tests/e2e/library-mobile.spec.ts:416-427`），不受語意影響。

## Acceptance Criteria

1. **判準 `[@contract-v1]`**：「未匹配」＝`parse_status = 'failed' AND (metadata_source IS NULL OR metadata_source = '')`。只寫在**一個**共用的 SQL 片段裡，篩選（movie／series 的 `*ListFilterConditions`，含 FTS 搜尋路徑）與 `GetStats` 的 `unmatched_count` 都用它——篩選出來的筆數永遠等於 N。
   - 本 AC 定義「未匹配」的**語意**。API 形狀不變：`?unmatched=true` 參數（`dsr-1b-a2` AC #1 `[@contract-v1]`，形狀未改、不 bump）與 `unmatched_count` 欄位名都照舊。
2. 依裁定：TMDb／豆瓣／Wikipedia／NFO／手動 任一來源有資料的片**不算**；「整理中」（pending）**不算**；「失敗但舊資料還在」（重新解析時斷線）**不算**；只有「失敗而且沒有任何來源」才算。
3. **Migration 039**：`UPDATE movies/series SET metadata_source = 'tmdb' WHERE tmdb_id > 0 AND (metadata_source IS NULL OR metadata_source = '')`。冪等、`Down` 不還原（同 038 理由：沒有要還原的真實狀態）。
4. 測試（後端）：
   - movie 與 series 各一組表格測試，同一批資料同時驗篩選與 `GetStats`：tmdb success（不算）、豆瓣 success 無 tmdb（不算）、manual success（不算）、pending 無來源（不算）、failed 有舊 tmdb＋來源（不算）、failed 無來源（**算**）、failed 來源為 `''`（**算**）、軟刪除的 failed 無來源（不算、不計數）。
   - 既有 4 個測試（`TestMovieGetStats`、`TestMovieListUnmatchedFilter`、series 兩個）與 FTS `subtitle_status + unmatched` 改用新判準造資料，斷言照原本的意圖。
   - Migration 039 測試：有 tmdb 無來源 → 'tmdb'；已有來源（douban／manual）不動；tmdb 0／NULL 不動；再跑一次結果相同。
5. seed：`cmd/seed/main.go` 的註解改成新語意（pending＝整理中、failed＝未匹配），資料不動。
6. 前端與畫面零改動；篩選文字維持「未匹配」。

## Tasks / Subtasks

- [x] Task 1 — 共用判準片段＋ movie／series 篩選與 `GetStats` 改用它（AC #1, #2）
- [x] Task 2 — Migration 039 回填＋測試（AC #3, #4）
- [x] Task 3 — repository 測試：新增表格測試、改寫既有 5 處（AC #4）
- [x] Task 4 — seed 註解；`pnpm nx test api`、`pnpm nx test web`、`lint:all` 全綠（AC #5, #6）

## Dev Notes

### 顧問討論結論（2026-09-29，Winston／Sally／Amelia，Fable，兩輪）

- 第一輪三人三種判準：`failed`（Sally）、`!= 'success'`（Amelia）、`metadata_source` 為空（Winston）。
- Winston 的反例（上方「流程狀態」段）讓 Sally、Amelia 撤回只看 `parse_status` 的方案；Sally 的「按這個篩選是在找要我動手的片」讓 Winston 把 pending 排除。第二輪收斂成 AC #1 的複合條件。Alexyu 裁定 ② 選 A 確認排除 pending。
- **文字維持「未匹配」**（SM 裁定，Sally 第二輪同意）：徽章「失敗」描述這次流程、篩選描述有沒有資料；若篩選改叫「失敗」，帶舊資料的「失敗」徽章片不會出現在「失敗」篩選裡，反而更混亂。
- **不建 `metadata_source` 索引**：條件以 `parse_status = 'failed'` 開頭，已有 `idx_movies_parse_status`／`idx_series_parse_status`（`006_media_entities_enhancement.go:93,95`）。
- **活動中心不在本張**：「待解析項目 N → 前往處理」（`ActivityHub.tsx:173-190`）的 N 是解析佇列（`activity_service.go:183-191`），點過去是媒體庫的列，本來就不是同一批 → `disc-2026-09-activity-pending-count-vs-unmatched`（見 Discovery Triage）。

### 實作提示

- 共用片段放 repository 套件（例如 `unmatched_condition.go`），用函式接受欄位前綴（FTS 路徑用 `col()` 加表別名，見 `movie_repository.go:650-652` 現寫法），GetStats 用無前綴版本。
- `GetStats` 的 `COUNT(CASE WHEN … THEN 1 END)` 直接嵌同一片段。
- 測試造資料要**明確**設 `ParseStatus` 與 `MetadataSource`；`repo.Create` 不會自動補。

### Project Structure Notes

- 純後端：`internal/repository/`（2 個 repo＋共用片段＋2 個測試檔）、`internal/database/migrations/039_*`、`cmd/seed/main.go` 註解。後端 4 個 task、前端 0 → 單張。

### Time-dependent visual coverage

- N/A — no wall-clock-reading components touched.

### References

- 原條目：`sprint-status.yaml` `disc-2026-09-unmatched-filter-vs-parse-status`（dsr-2b-a 立案）
- 相關：`disc-2026-09-apply-non-tmdb-sources`（該條「套用後仍算未匹配」的顧慮由本張解除）

## Dev Agent Record

### Agent Model Used

Claude Opus 5.5（Amelia）

### Debug Log References

- RED：新增 `unmatched_condition_test.go`（movie／series 各一組 9 列表格，同時驗 List、FullTextSearch、GetStats），舊碼跑出 6 列（豆瓣／NFO／手動／整理中全被算進去）vs 期望 2 列；migration 039 測試在實作前編譯失敗。
- GREEN：repository、migrations 套件全綠；舊語意的 6 個測試（`TestMovieGetStats`、`TestMovieListUnmatchedFilter`、`TestMovieFullTextSearchAppliesFilters`、series 三個）如預期變紅，改成用新判準造資料後全綠，斷言數字不變。

### Completion Notes List

- 判準集中在 `repository/unmatched_condition.go` 的 `unmatchedCondition(alias)`；movie／series 的 `*ListFilterConditions`（List 與 FTS 共用）與兩個 `GetStats` 都呼叫它。`git grep -n 'tmdb_id IS NULL\|tmdb_id = 0' -- apps/api ':!*_test.go'` 改動後零命中。
- Migration 039 回填 `tmdb_id > 0` 且無來源的列為 `'tmdb'`；已有來源的不動；冪等。
- seed：只改註解（2 筆 pending＝整理中、1 筆 failed＝未匹配），資料不動。
- API 形狀、前端、畫面零改動；文字維持「未匹配」。
- 全量回歸：`pnpm nx test api` 綠；`pnpm nx test web` 291 檔／4,346 條綠；`lint:all` 0 errors。
- /ship 對抗式 CR（fresh-context 子代理）：0 HIGH／3 MED／4 LOW；SQL、所有失敗與成功寫入路徑、migration 039 編號與冪等皆確認無誤。已修：MED-1（表格測試少了「failed＋非 TMDb 來源＋無 tmdb_id」→ 加 `failed-keeps-douban-data`、`failed-keeps-nfo-data`；暫時把判準改成錯的 tmdb 版，movie／series 測試都會紅，已還原）、LOW-6（`scripts/seed-test-data.sh:100,111` 註解、`disc-2026-09-apply-non-tmdb-sources` 條目的「仍算未匹配」）。其餘見 Discovery Triage。
- 🔗 AC Drift: FOUND — `9c-4-tech-badges-ui-unmatched-filter`（未匹配篩選與計數的原始 story）AC #5, #6「`tmdb_id IS NULL OR tmdb_id = 0`」→ 本張 AC #1「failed 且無任何來源」。這是 Alexyu 裁定的語意修正；`dsr-1b-a2` AC #1 `[@contract-v1]` 只定義 `?unmatched=true` 的參數形狀，形狀未變，不 bump。
- 📎 Contract Stamps: FOUND（本張 AC #1 定義「未匹配」語意 `[@contract-v1]`；上游 `dsr-1b-a2` AC #1 `[@contract-v1]` 參數形狀，已於 AC #1 註明未變）。
- 🎭 A11y Pre-Flight: N/A（100% backend — no apps/web/ files touched）。
- 🎨 UX Verification: SKIPPED — no UI changes in this story。

### Discovery Triage

- ③ backlog-with-carry-forward-link — 活動中心「待解析項目 N 個 → 前往處理」的 N 是解析佇列（`activity_service.go:183-191` `parse.GetPending`），連結卻是媒體庫 `?unmatched=true`（`ActivityHub.tsx:184`），點過去的片數對不上 → `disc-2026-09-activity-pending-count-vs-unmatched`（P3，建單時立案）。
- ③ — CR MED-2：詳情頁只看 `parseStatus`（`LocalDetailV2.tsx:179-180`），「失敗但舊資料還在」的片會顯示「比對失敗」並藏掉資料 → `disc-2026-09-detail-failed-hides-kept-data`（P2）。
- ③ — CR MED-3：字幕生成同意框的「未匹配」是「沒有 TMDb id」（`generation_candidates.go:1204`、`consent/consentRows.ts:263-269`），同一個詞兩種意思 → `disc-2026-09-consent-unmatched-means-no-tmdb`（P3）。
- ③ — CR LOW-5：`POST /movies` 不寫 `metadata_source` → `disc-2026-09-api-created-row-no-source`（P4）。
- ③ — CR LOW-7：「未匹配 (N)」不看其他篩選（既有行為）→ `disc-2026-09-unmatched-count-ignores-other-filters`（P4）。
- 不立案：seed 影集有 TMDb id 卻沒寫來源（`cmd/seed/main.go` 影集段）——都是 success，不影響未匹配；CR LOW-6 已記錄。

### File List

- apps/api/internal/repository/unmatched_condition.go（新）
- apps/api/internal/repository/unmatched_condition_test.go（新）
- apps/api/internal/repository/movie_repository.go
- apps/api/internal/repository/movie_repository_test.go
- apps/api/internal/repository/series_repository.go
- apps/api/internal/repository/series_repository_test.go
- apps/api/internal/database/migrations/039_backfill_tmdb_metadata_source.go（新）
- apps/api/internal/database/migrations/039_backfill_tmdb_metadata_source_test.go（新）
- apps/api/cmd/seed/main.go
- scripts/seed-test-data.sh
- _bmad-output/implementation-artifacts/9c-4-tech-badges-ui-unmatched-filter.md（AC drift reference — see Completion Notes；未修改）
- _bmad-output/implementation-artifacts/disc-2026-09-unmatched-filter-vs-parse-status.md
- _bmad-output/implementation-artifacts/sprint-status.yaml

## Change Log

| Date       | Change                                                                                                          |
| ---------- | --------------------------------------------------------------------------------------------------------------- |
| 2026-09-29 | create-story（SM Bob）：依 Alexyu 裁定 ①② 與顧問兩輪討論建單；立案 `disc-2026-09-activity-pending-count-vs-unmatched`。 |
| 2026-09-29 | dev-story（Amelia）：共用判準＋ migration 039 ＋ 表格測試；全量回歸綠 → review。 |
| 2026-09-29 | /ship 對抗式 CR：補 2 個表格案例（豆瓣／NFO 失敗但有資料）、修註解；立案 4 條。 |
