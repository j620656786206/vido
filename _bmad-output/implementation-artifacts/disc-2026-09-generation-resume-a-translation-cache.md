# Story disc-2026-09-generation-resume-a-translation-cache：翻譯到一半被預算擋下，下次只翻沒翻過的句子——語音辨識這條線接上既有的逐句快取

Status: done

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As the person whose batch hit the budget ceiling at cue 1,200 of 1,910,
I want the next 下次繼續 to translate only the 710 cues that were never translated,
so that the $1.70 already spent on the first 1,200 is not spent again — and the run finishes instead of hitting the same ceiling from cue 1 forever.

## Context

`disc-2026-09-generation-resume-from-checkpoint` 拆出的**第一張**（⚖️ SM 2026-09-18：三張互不相依，各自可出貨——**A 翻譯續跑**（本張）、**B 語音辨識續跑**（`…-b-asr-chunk-store`，同時建單）、**C 估價扣減**（`…-c-estimate-deduction`，依賴 A＋B，只建 sprint 條目））。

### 🔴 建單時查到的事（main `ef224134`；行號為現況）

1. **翻譯被預算擋下時，翻好的全部丟掉。** `TranslateWithGlossaryHarvest`（`translation_service.go:266-412`）碰到 `ai.ErrBudgetExceeded` 回 `nil, nil, TranslationOutcome{}, err`（`:346-351`）——已翻的 `result` 在函式邊界就消失，沒有任何呼叫端能存。`transcription_service.go:1059-1073` 只寫 DB `untranslated`＋EN 路徑（讓下次跳過辨識），**翻譯本身從第 1 批重來**。以 157 分鐘那部為例：翻譯 191 批、$2.71；擋在第 120 批 → 下次重付 $1.70。
2. **這條線沒有逐句快取。** `translation_service.go:98-99` 明講 `this (ASR) leg has no segment cache at all — splitCachedCues is pipeline-only`。抽內嵌字幕那條線有（`subtitle/segment_cache.go`）：`cache_entries` 表（migration 004）、type `subtitle_segment`、TTL 30 天、key ＝ `subseg:v1:` ＋ sha256(句子內容 ＋ `RunVersion{MetadataHash, GlossaryVersion, PromptVersion, ModelID}`)（`segment_cache.go:158-165`）；`splitCachedCues` 一次 `GetMany` 整軌（`:201-233`，讀失敗＝全部 miss、只 Warn）、`storeCachedCues` 翻完寫入（`:243-265`，寫失敗只 Warn）、`mergeCues` 合回（`:281`）。**而且它也是整軌成功才寫**（`process_item.go:92-102` 註解、`backlog-translate-budget-partial-progress`）——本張只做語音辨識這條線，pipeline 那條線的「逐批寫」留在那張 backlog（見 AC #8）。
3. **`main.go:755` 只在 pipeline 模式接快取**（`if cfg.SubtitlePipelineEnabled()` `:741`）；legacy 模式（預設）什麼快取都沒有。
4. **Rule 19**：`services` 不能 import `subtitle`（循環）。key 函式 `segmentKey`／`MetadataHash`／`GlossaryVersionHash` 現在住在 `subtitle`；`MetadataHash` 吃 `subtitle.TranslateContext`（`pipeline.go:81-95`），欄位與 `prompts.MediaMetadata`（`prompts/subtitle_translator.go:121-129`）**完全相同**（Title／OriginalTitle／Year／Genres／Overview／Cast／Countries）。
5. **`RunVersion` 四個欄位在這條線的來源**：`MetadataHash` ← `s.mediaMetadataFor(ctx, mediaType, mediaID, glossaryKey)`（`transcription_service.go:1375`）；`GlossaryVersion` ← `s.loadGlossary(glossaryKey)` 的 pairs（`:384-414`）＋ `prompts.LexiconVersion()`；`PromptVersion` ← `prompts.PromptVersionFor(s.localizationLevel(ctx))`（`prompts/localization.go:101`；level 已折進 PromptVersion，不另列）；**`ModelID` 沒有來源**——`TranslationService.provider` 是 `ai.TextCompleter`（`translation_service.go:154`），沒有 `EffectiveModel()`；`ai.ModelIDFromContext(ctx)` 只在使用者明選模型時才有（`generation_batch.go:435`），預設模型的批次是空字串；`transcription_service.go` 從不呼叫 `ai.WithModelID`。`*ClaudeProviderHolder` 有 `EffectiveModel()`（`claude_provider_holder.go:172`）而 `main.go:699` 傳的就是它。
6. **翻譯用前 5 句的譯文當上下文**（`translation_service.go:302-316`，`contextWindow=5`）：`result` 一開始是英文複本（`:279-280`），翻過的批會覆寫。續跑時若把快取命中先填進 `result`，第一個 miss 批的上下文自然是命中的譯文。
7. **OpenCC 與台灣詞彙表是對整份組好的 zh SRT 跑的**（`transcription_service.go:1549-1560`，`serializeTranslationBlocksToSRT` 之後），不是逐句——所以快取存**模型原始輸出**即可，命中的句子照樣會經過同一道 OpenCC＋詞彙表。
8. **進度百分比**：`progressFn(pct)` 由 `processedBlocks/totalBlocks` 算（`:365-369`），SSE 每批一次。
9. **既有測試縫**：`services/route_cache_test.go:415-447` 的 `fakeCacheRepo`（7 個方法都有，**在 `services` 套件內**）；`subtitle/segment_cache_test.go:38-72` 的 `memorySegmentCache`；`newMigratedTestDB`（`segment_cache_test.go:25-35`）跑真 migration 的 `:memory:` SQLite。
10. **清理**：`cache_sweep_scheduler.go` 每 45 分鐘 `ClearExpired`；⚠️ `CacheCleanupService.ClearCacheByType("metadata")` 會**整張** `cache_entries` truncate（`cache_cleanup_service.go:110-116`）——既有行為、也影響 pipeline 的快取，本張不修，記錄。
11. **規模**：這部片 1,910 句 EN SRT 118 KB → 每句約 62 bytes；一列（key 74 B ＋ value ≈ 63 B ＋ type／時間戳）約 250–300 B → **一部片約 0.5 MB，1,000 部約 500 MB**，30 天 TTL 自動清。`GetMany` 每 500 個 key 一句 `IN (…)`（`cache_repository.go:60`），PK 查詢。

### ⚖️ 設計裁定（SM，2026-09-18）

1. **每翻完一批就寫快取，不等整軌成功**——這是「續跑」的全部；`ErrBudgetExceeded` 那條路不需要再改回傳值（`nil` 照舊），因為已寫進去的不會丟。
2. **開跑前一次 `GetMany` 整份 SRT**，命中的句子直接填進 `result`，**只把 miss 的句子分批送模型**，上下文取 `result` 中該批前 5 句（命中或剛翻的都算）。進度從命中比例起算（1,200/1,910 → 一開始就 63%）。
3. **key 與 pipeline 那條線位元相同**：同一個 `segmentKey`、同一個 `RunVersion` 定義、同一個 TTL、同一個 type tag。把 `segmentKey`／`MetadataHash`／`GlossaryVersionHash` 搬到 `services` 與 `subtitle` 都能 import 的地方（建議新 leaf 套件 `internal/segkey`，只 import `models`＋`prompts`；`MetadataHash` 改吃 `prompts.MediaMetadata`，`subtitle.MetadataHash(tctx)` 變成一行包裝）。`subtitle` 既有測試**一行不改**照跑。
4. **`ModelID` 一律有值**：`ai.ModelIDFromContext(ctx)` 有就用，否則問 provider——`TranslationService` 對 provider 做**可選介面探測** `interface{ EffectiveModel() string }`（`ai.DetailedTranscriber` 的先例），`*ClaudeProviderHolder` 已實作；兩者皆無 → `ai.DefaultClaudeModel`（pipeline `currentModelID` 的同一條退路，`pipeline.go:417`）。
5. **快取存模型原始輸出**（pre-OpenCC）；命中句照樣走整份 OpenCC＋詞彙表。
6. **legacy 與 pipeline 模式都接**：`main.go` 無條件建 store 給 `TranscriptionService`（`SetSegmentStore`）。
7. **快取失敗＝省錢的加值沒了，不是錯**：讀失敗 → 全部 miss、Warn 一次；寫失敗 → 第一次 Warn＋最後總數 Warn（`storeCachedCues` 的形狀）；⛔ 絕不讓付費 run 因快取失敗而失敗。
8. **不動 pipeline 的 `TranslateTrack`／`translateWithCache`**（stamped 面，`backlog-translate-budget-partial-progress` 另案）；不動 `cache_cleanup_service.go`。

## Acceptance Criteria

1. **共用 key 套件。** 新增 `internal/segkey`（或 dev 證明更好的位置，但**必須**同時能被 `services` 與 `subtitle` import 且不進 Rule 19 禁止邊）：`SegmentKey(cueText string, v models.RunVersion) string`、`MetadataHash(m prompts.MediaMetadata) string`、`GlossaryVersionHash([]prompts.GlossaryEntry) string`、常數 `Prefix="subseg:v1:"`、`Type="subtitle_segment"`、`TTL=30d`。`subtitle/segment_cache.go` 改為委派（`MetadataHash(tctx)` 組 `prompts.MediaMetadata` 後呼叫）。**位元相同**由測試釘住：`subtitle` 既有 key 測試全綠不改 ＋ 新的 parity 測試（外部測試套件，同時 import 兩邊，同一組輸入兩邊 key 相等）。`boundaries_test.go` 全綠；若把 `segkey` 列進 leaf 名單，`TestLeafPackagesHaveNoInternalDeps` 要跟著證明。

2. **`services` 的 store 與 split／merge。** `services.SegmentStore` 介面（`GetMany`／`Set`，`subtitle.SegmentCache` 的同形）＋ `NewSegmentStore(repository.CacheRepositoryInterface)`（type tag ＝ `segkey.Type`）。`TranscriptionService.SetSegmentStore(SegmentStore)`；`main.go` 用 `repos.Cache` 無條件接上（legacy 模式也要）。`subtitle.NewSegmentCacheRepository` 可改為委派 `services.NewSegmentStore`（少一份重複）——可選。

3. **`RunVersion` 在這條線算得出來。** `translateSRT` 在呼叫翻譯前組 `models.RunVersion{MetadataHash: segkey.MetadataHash(metadata), GlossaryVersion: segkey.GlossaryVersionHash(glossary pairs), PromptVersion: prompts.PromptVersionFor(level), ModelID: <裁定 4>}`。`TranslationService` 加 `effectiveModelID(ctx) string`：ctx → provider 可選介面 → `ai.DefaultClaudeModel`；並提供給 `translateSRT`（或由 `TranslateOption` 傳入版本）。**測試**：三條退路各一條；`ModelID` 永不為空。

4. **翻譯前分 hit／miss，翻譯中逐批寫。** `TranslateWithGlossaryHarvest` 加 `TranslateOption`：`WithSegmentCache(store SegmentStore, version models.RunVersion)`。有設定時：
   - 開跑前**一次** `GetMany`（key 依 `blocks[i].Text`）；命中 → `result[i].Text = 譯文`，計入 `processedBlocks`，第一次 `progressFn` 就反映；
   - 只把 miss 的句子（保持來源順序）10 句一批送模型；每批的上下文 ＝ 該批第一句在來源中的前 5 句的 **`result` 內容**（命中或已翻）；
   - **每批成功解析後立刻** `Set` 該批每一句（模型原始輸出，pre-OpenCC）；保留英文的句子（`englishKept`）**不寫**；
   - `ErrBudgetExceeded`／ctx 到期時的回傳**不變**（`nil`＋err）——已寫的批就是進度；
   - 無 store（nil）→ 行為與今天位元相同（既有測試全綠不改）。
   - `harvested` 只來自真的送模型的批（命中的句子沒有 trailer）——這是可接受的，記在註解。

5. **可觀察行為（測試要蓋到的情境）。**
   - 1,910 句、擋在第 120 批（fake completer 第 120 次呼叫回 `ErrBudgetExceeded`）→ store 內恰好 1,200 句；再跑一次（completer 正常）→ **模型只被呼叫 71 次**、譯文 1,910 句完整、順序與時間軸不變、SSE 第一個進度事件 ≥ 62%。
   - 第二次跑時 `GetMany` 回錯 → 191 次呼叫（全部 miss）、只 Warn、run 成功。
   - `Set` 全部回錯 → run 成功、Warn 兩則（第一次＋總數）。
   - 換模型（ctx `WithModelID` 不同）或詞彙表多一個詞 → key 不同 → 全部 miss（換設定就重來，刻意）。
   - 命中句的 OpenCC：store 存簡體的模型輸出，命中後整份輸出仍是繁體（fake OpenCC 記錄輸入含該句）。
   - **規模門檻**：用 `newMigratedTestDB` 塞 **200,000** 列（type 混合），對 2,000 個 key `GetMany` **< 100 ms**（`testing.Short()` 時跳過）；另在 Completion Notes 記一次本機 200 萬列的實測數字（一次性腳本，不進 repo）。
   - `translation_service.go:84-106` 那段「這條線沒有 segment cache」的註解改成實話；`TestSubtitleTranslatorPromptVersion_NotBumpedBy9R8` 照舊綠（本張**不**動 prompt version）。

6. **真機驗證（done 的門檻）。** 照 `.claude/memory/project_unraid_nas_access.md` 的隔離容器食譜，用**已有英文字幕、狀態 `untranslated`** 的片（可先跑一次 A 之前的流程做出來，或用這次 157 分鐘那部的 `.en.srt` 複本），`AI_RUN_BUDGET_USD=1.0` 跑到被擋（約第 70 批）→ log 有寫入筆數 → 改回 5.0 再跑 → log 顯示 `segment cache hits=N`、模型呼叫次數 ≈ 191−N、`spent_usd` 只有剩餘部分（估 ≈ $1.7）。正式 Vido 不動、跑完清理。花費估計 ≈ $2.7 總計。

7. **不做的事。** 不動 pipeline 的 `TranslateTrack`／`translateWithCache`／`storeCachedCues` 時機（`backlog-translate-budget-partial-progress` 另案，見 AC #8）；不動 `cache_cleanup_service.go` 的整表 truncate（記錄）；不做估價扣減（story C）；不改 SSE／API 形狀（無 contract 變動）；不做語音辨識段（story B）。

8. **交代。** `backlog-translate-budget-partial-progress` 補記：A 之後 pipeline 那條線可用同一個 store／同一個「逐批寫」做法，差別只在 `TranslateTrack` 內部；本張不碰。

9. **CI 全綠**：`pnpm nx test api`、`pnpm nx test web`（無前端改動仍照跑）、`pnpm run lint:all`、`pnpm run format:check`。⛔ 測試不 `run_in_background`；跑完 `pnpm run test:cleanup`。

## Tasks / Subtasks

- [x] **Task 1 — `segkey` 套件與 parity（AC: #1）**：先寫紅測試（parity、boundaries）→ 搬 key 函式 → `subtitle` 委派
- [x] **Task 2 — `services.SegmentStore`＋接線（AC: #2）**：先寫紅測試（adapter 的 type／TTL／miss-by-absence，抄 `route_cache_test.go:449-462`）→ 實作 → `main.go` 無條件接
- [x] **Task 3 — `RunVersion` 與 `effectiveModelID`（AC: #3）**：先寫紅測試（三條退路）→ 實作
- [x] **Task 4 — 分 hit／miss、逐批寫（AC: #4, #5）**：先寫紅測試（120 批被擋→再跑只呼叫 71 次；讀失敗；寫失敗；換模型；OpenCC；nil store 位元相同）→ 實作
- [x] **Task 5 — 規模門檻與註解（AC: #5）**：200k 列 `GetMany` < 100 ms；2M 列一次性實測記錄；改 `translation_service.go:84-106` 註解
- [x] **Task 6 — 收尾（AC: #7, #8, #9）**：全套閘門；mutation check；`backlog-translate-budget-partial-progress` 補記
- [x] **Task 7 — 真機驗證（AC: #6）**

## Dev Notes

### 這張的重點

- 「續跑」＝「**每批寫、開跑前查**」兩件事，回傳值與錯誤路徑都不用改。最容易做錯的是**上下文**：miss 批的前 5 句要從 `result` 取（含命中），不是從 miss 清單取。
- key 必須與 pipeline **位元相同**——不是為了共用命中（兩條線通常不會翻同一部片），是為了只有一套定義；parity 測試是門檻。
- `ModelID` 永不為空，否則預設模型的批次會與明選同一個模型的批次 key 不同，命中率無故砍半。

### 上游契約（Rule 20）

- confirmed against [@contract-v1] (Story sub-1-5b AC #1 / sub-1-3 AC #1) —— `RunVersion` 四欄位語意不變，只是多了一個消費者；`segmentKey` 格式（`subseg:v1:`）不變。
- `TranslateOption` 是純加寬；SSE `translation_progress` 形狀不變（只是第一個 pct 可能不是 0）。無 bump。

### 已知陷阱

- `services ↛ subtitle`（Rule 19）——`segkey` 不能 import `subtitle`；`subtitle` 可以 import `segkey`。
- `TranslateWithGlossary`（無 harvest）是 `Harvest` 的薄包裝（`:259`）——選項要兩邊都通。
- `result[i]` 命中後 `parseTranslationResponseWithTerms` 的 index 對應仍以來源 `Index` 為準（`:377`），不要用位置。
- `fakeCacheRepo` 在 `services` 套件內可直接用；`memorySegmentCache` 在 `subtitle`，不能跨包。
- 進度事件：命中先計入會讓 `translation_progress` 第一個 frame 不是 0%——前端 `useGenerationProgress` 只顯示數字，無影響；`dsr-6d-c-2` 的紀錄原地更新也無影響。

### Source tree

```
apps/api/internal/segkey/segkey.go(+test)                 ← Task 1（新）
apps/api/internal/subtitle/segment_cache.go               ← Task 1（委派；測試不動）
apps/api/internal/services/segment_store.go(+test)        ← Task 2（新）
apps/api/internal/services/translation_service.go(+test)  ← Task 3, 4, 5
apps/api/internal/services/transcription_service.go(+test)← Task 3, 4
apps/api/cmd/api/main.go                                  ← Task 2
apps/api/internal/boundaries_test.go                      ← Task 1（若加 leaf）
```

### Cross-Stack Split Check

後端 task 7、前端 0 → 不觸發。規模與 `disc-2026-09-transcription-run-5min-hard-timeout`（6 task）相當。

### Time-dependent visual coverage

- **N/A — no wall-clock-reading components touched.**

### References

- [Source: `apps/api/internal/subtitle/segment_cache.go:31-93, 110-121, 158-190, 201-265, 281`、`process_item.go:64-66, 92-102, 385-455`、`pipeline.go:81-95, 417`]
- [Source: `apps/api/internal/services/translation_service.go:84-106, 110-118, 154, 163, 259-412`、`transcription_service.go:333-343, 384-414, 1059-1073, 1375, 1480-1560`、`claude_provider_holder.go:172`、`route_cache.go:78-128`、`route_cache_test.go:415-462`]
- [Source: `apps/api/internal/repository/cache_repository.go:35-136, 150-202`、`interfaces.go:349-385`、`database/migrations/004_create_cache_entries_table.go`]
- [Source: `apps/api/internal/ai/prompts/subtitle_translator.go:26-34, 121-129`、`prompts/localization.go:101`、`ai/model_context.go:30-33`]
- [Source: `cmd/api/main.go:699, 741, 755`]
- [Source: project-context.md#Rule 13 / #Rule 19 / #Rule 20]

## Dev Agent Record

### Agent Model Used

Claude Opus 5（Dev Amelia，2026-09-19）

### Debug Log References

- 紅→綠：`go test ./internal/services/ ./internal/subtitle/ ./internal/ -run 'TestEffectiveModelID|TestTranslate_|TestSegmentStore_|TestTranslationRunVersion_|TestTranslateSRT_|TestSegkeyParity|TestSegkeyDependsOnlyOnLeaves'`
- Mutation check（八刀全部被殺）：不讀快取 → 2 紅；不寫快取 → 1 紅；連保留英文的句子也寫 → 1 紅；上下文取自來源而非 `result` → 1 紅；進度不計命中 → 1 紅；`ModelID` 可為空 → 2 紅；key 不含 model → 2 紅；`RunVersion` 丟掉詞彙表 → 1 紅。
- 閘門：`pnpm nx test api` 綠、`pnpm nx test web` 270 files 綠、`pnpm run lint:all` 0 errors（129 warnings 既有浮動）、Prettier 綠、`pnpm run test:cleanup` 已跑。

### Completion Notes List

- Ultimate context engine analysis completed - comprehensive developer guide created（SM Bob，2026-09-18；一個唯讀稽核代理查快取表／估價／續跑語意，SM 逐項複核）。

### Discovery Triage

- **Did this story discover any work outside its current scope?** **YES**
- **① expand-scope-in-place**：`ModelID` 在這條線沒有來源 → AC #3；legacy 模式沒接快取 → AC #2。
- **② spawn-blocking-story**：無。
- **③ backlog-with-carry-forward-link**：
  - `disc-2026-09-generation-resume-b-asr-chunk-store`（同時建單）— 語音辨識段的續跑。
  - `disc-2026-09-generation-resume-c-estimate-deduction`（sprint 條目）— 估價扣掉已有的段與句。
  - `backlog-translate-budget-partial-progress`（既有）— pipeline 那條線的逐批寫，同一個做法。
  - `cache_cleanup_service.go:110-116` 整表 truncate 會清掉所有字幕快取 — 既有、記錄（不另立，等 C 一起看是否值得）。
- **`segkey` 沒有列進 leaf 名單**（它 import `models` 與 `ai/prompts`，不是零內部相依），改在 Rule 19 新增「Shared key package」段落，並加一條 `TestSegkeyDependsOnlyOnLeaves` 釘住它只能相依這兩個套件——長胖就會有一條線編不過，而那時最誘人的「修法」正是再複製一份 hash。
- **`splitCachedBlocks` 回傳「來源位置」而不是 block 複本**：批次迴圈用 `pending []int` 索引 `blocks`／`result`／`keys`，三者永遠對齊，不需要第二張對照表；無 store 時 `pending` 就是 0..n-1，所以舊路徑位元相同（`TestTranslate_NoStoreIsTheOldBehaviour` 直接比對兩次呼叫的回傳值）。
- **上下文取自 `result`**：命中的譯文會成為第一個 miss 批的前文（`TestTranslate_ResumedBatchSeesCachedContext` 斷言 prompt 裡是「舊譯文10」而不是「Line 10」）。這是本張最容易做錯的一點（Dev Notes 有點名）。
- **保留英文的句子不寫快取**：否則下一輪會把英文當成譯文送出去。
- **存模型原始輸出（pre-OpenCC）**：OpenCC＋台灣詞彙表是對整份組好的 SRT 跑的，命中句照樣會經過同一道處理（`TestTranslateSRT_CachedCuesStillGetConverted` 用 fake OpenCC 證明簡體的快取值進得去、繁體出得來）。
- **`ErrBudgetExceeded` 的回傳值一行沒改**：已寫進快取的批就是進度；錯誤訊息裡的 block 編號改成該批的**來源**位置（`batchAt[0]`），否則續跑時會報一個沒意義的 miss 序號。
- **2,000,000 列一次性實測**（檔案型 SQLite，暫存目錄，不進 repo）：插入 2M 列 15.7 s、DB 762.9 MB；`SELECT ... IN (500 keys)` **223 ms（冷啟，第一次查）**、`SELECT ... IN (1910 keys)` **11.8 ms**。repo 內的門檻測試是 200k 列 / 2,000 keys < 100 ms（`TestSegmentStore_ReadStaysFastAgainst200kRows`，`-short` 時跳過）。
- ✅ **Task 7 真機驗證（2026-09-19 18:10–18:26，NAS 隔離容器 `vido-xlate`：正式 image ＋ 交叉編譯二進位覆蓋；DB 副本、scratch symlink；正式 Vido 全程未動、跑完清理、正式片庫 0 個 srt）**。用 story B 那輪產出的 `.en.srt`（1,935 句）複本 ＋ row 設成 `untranslated`，所以走 translate-only：不抽音訊、不花語音辨識的錢。

  | 回合 | 上限 | log | 結果 |
  | --- | --- | --- | --- |
  | 1 | $1.0 | `segment cache hits=0 total=1935 to_translate=1935` | 71 批後 `AI budget exhausted`、`translation stopped at block 710`、`spent_usd=1.002`。`cache_entries` 的 `subtitle_segment` 由 20,432 → **21,064（+632 列）** |
  | 2 | $5.0 | **`segment cache hits=787 total=1935 to_translate=1148`** | `llm_calls=115`（＝1,148÷10 進位）、**`spent_usd=1.696`**、`transcription complete`、row 翻成 `found / zh-Hant`、`.zh-Hant.srt` 1,935 句 |

  兩輪合計 **$2.70**，與一次跑完的 $2.71 幾乎相同——**續跑沒有額外成本**（AC #6 估的 ≈$1.7 剛好命中）。
  ⚖️ 兩個真實世界的細節：
  - **命中 787 > 第一輪付費的 710**：快取是**照句子內容**當 key，所以片子後半重複出現的短句（`Yes.`／`No.`／`Come on.`）也一起命中。同理第一輪 710 句只寫出 632 列（重複句共用一列）。
  - **正式 DB 本來就有 20,432 列 `subtitle_segment`**（抽內嵌字幕那條線長期累積的），本張新增的列與它們同族、同 type、同 TTL——證明「一套定義」不是紙上談兵。
  - 第二輪有 1 句保留英文（`english_kept_blocks=1`），既有的 partial 語意，未入快取（符合裁定）。
- ⚠️ **兩條線共用 key 但存的語意不同**（SM 裁定 3 已知並接受）：pipeline 存 **post-OpenCC**、本條線存 **pre-OpenCC**（因為裁定 1 要求每批就寫，而 OpenCC 只在整份組好後才跑）。同一部片只會走其中一條線，所以實務上不會撞；但理論上同片同 `RunVersion` 下兩條線若各跑過一次，pipeline 可能讀到未轉換的值。已立 `disc-2026-09-segment-cache-cross-leg-semantics`。
- Reference: `project-context.md` Rule 24

### File List

- `apps/api/internal/segkey/segkey.go`（新）— `SegmentKey`／`MetadataHash`／`GlossaryVersionHash`／`Prefix`／`Type`／`TTL`
- `apps/api/internal/subtitle/segment_cache.go` — 改為委派 `segkey`（常數與三個函式），刪掉搬走的 `fieldSep`／`payloadSep`／`sortedCopy`
- `apps/api/internal/subtitle/segkey_parity_test.go`（新）— 位元相同的 parity 測試（AC #1）
- `apps/api/internal/boundaries_test.go` — `TestSegkeyDependsOnlyOnLeaves`
- `apps/api/internal/services/segment_store.go`（新）— `SegmentStore` port ＋ `cache_entries` adapter
- `apps/api/internal/services/segment_store_test.go`（新）— type／TTL／共用資料列／200k 列規模門檻
- `apps/api/internal/services/translation_service.go` — `WithSegmentCache`、`splitCachedBlocks`、批次迴圈改走 `pending`、每批 `Set`、`EffectiveModelID`、`modelNamer`、改掉過時註解
- `apps/api/internal/services/translation_cache_test.go`（新）— AC #3／#4／#5 行為測試
- `apps/api/internal/services/transcription_service.go` — `segmentStore` 欄位＋`SetSegmentStore`、`translationRunVersion`、`translateSRT` 接上
- `apps/api/internal/services/transcription_translation_resume_test.go`（新）— 服務層縫合（OpenCC、RunVersion）
- `apps/api/cmd/api/main.go` — `SetSegmentStore(NewSegmentStore(repos.Cache))`（legacy 與 pipeline 皆接）
- `project-context.md` — Rule 19 新增 Shared key package 段落
- `_bmad-output/implementation-artifacts/sprint-status.yaml` — `backlog-translate-budget-partial-progress` 補記（AC #8）＋ 新 disc

## Change Log

| 日期 | 內容 |
| --- | --- |
| 2026-09-19 | ✅ **DONE**（PR #479 已合併進 main；CI **13 pass / 0 fail**，視覺檢查因無前端改動 skip）。 |
| 2026-09-19 | ✅ **真機驗證通過**（Task 7）：$1.0 跑到第 710 句被擋、632 列入庫；$5.0 續跑 `hits=787/1935`、`llm_calls=115`、`spent_usd=1.696`、row 翻 `found/zh-Hant`。兩輪合計 $2.70 ≈ 一次跑完的 $2.71，續跑不額外花錢。 |
| 2026-09-19 | 🚧 **REVIEW**（dev-story, Amelia；branch `feat/generation-resume-translation-cache`）。Task 1–6 完成：key 搬到 `internal/segkey`（parity 測試釘住位元相同、`subtitle` 既有測試一行沒改）、`services.SegmentStore`＋無條件接線、`RunVersion` 與 `EffectiveModelID` 三條退路、開跑前一次 `GetMany`＋每批 `Set`、上下文取自 `result`、保留英文的不寫、快取失敗只 Warn。Mutation 八刀全殺。實測：200k 列 / 2,000 keys 門檻綠；2M 列一次性量測 11.8 ms（冷啟第一次 223 ms）。Task 7 真機驗證待跑。 |
| 2026-09-18 | Story 建立（SM Bob, create-story；main `ef224134`）。`disc-2026-09-generation-resume-from-checkpoint` 拆三張的第一張。裁定：每批寫快取、開跑前一次查、只翻 miss、上下文從 `result` 取；key 與 pipeline 位元相同（搬到共用 leaf 套件）；`ModelID` 永不為空；存模型原始輸出；legacy 模式也接；快取失敗只 Warn。規模門檻：200k 列 `GetMany` < 100 ms。 |
