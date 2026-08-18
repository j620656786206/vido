# Story 5.5: 詞彙庫 auto-harvest —— 翻譯回傳 term map，閉合「翻譯 → 回填名詞庫」迴圈

Status: done

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

**Epic:** `epic-subtitle-pipeline-m3`（M3 第一波，G 群組 —— 最後一支）· **Risk: 🟡 MED（prompt 變更＋雙路徑觸及；trailer 污染 cue 文字與 harvest 覆寫已確認詞是兩條紅線）** · **BACKEND-ONLY**
**Source:** sprint-status `epic-subtitle-pipeline-m3` seed（G:§6.5 glossary auto-harvest 閉環）· spec §6.5（`vido-subtitle-pipeline-spec.md:114-130`，Alexyu 2026-07-23 裁定）
**Cross-stack split check:** backend tasks = 5, frontend tasks = 0 → 單一 story（F6 審核 UI 已由 9R-15 + `GlossaryPanelV2` 交付，本 story 零 FE）

---

## 🔎 現實盤點：seed 提到的零件都在，缺的是「回程」與 pipeline 的「去程」

§6.5 的閉環 = 「名詞庫 → 翻譯」（去程）＋「翻譯 → 名詞庫」（回程）。盤點結果是一張不對稱的矩陣：

| 路徑 | 去程（feed glossary） | 回程（harvest terms） |
| --- | --- | --- |
| **legacy**（`TranscriptionService`，**預設模式** —— `VIDO_SUBTITLE_PIPELINE_MODE` 預設 `legacy`，config.go:160） | ✅ 9R-10：`loadGlossary` → `TranslateWithGlossary`（transcription_service.go:214,837） | ❌ |
| **pipeline**（M1/M2，subtitle package） | ❌ **seam 存在但恆空**：`TranslateContext.Glossary` 欄位＋`BuildGlossarySection` 已在 prompt 組裝鏈裡（pipeline.go:58,835），註解自陳「M1: always empty」；`runVersion()` 硬編 `GlossaryVersion: ""`（segment_cache.go:161） | ❌ |

**兩條路徑在翻譯層收斂於同一組零件**：同一個 `SubtitleTranslatorSystemPrompt` 常數（pipeline.go:832 ＋ translation_service.go:194,290）、同一個 `parseTranslationResponse`。⚖️ **裁定：harvest 指令與 trailer 解析各改一處、雙路生效**；本 story 同時補上 pipeline 缺的去程（含 `GlossaryVersion` 落地）。

**一個危險的既有事實**：`GlossaryRepository.Upsert` 是 `ON CONFLICT DO UPDATE`（term_zh/source/confirmed 全覆寫，註解自陳「a manual edit that races an auto-mine last-writer-wins」）。auto-harvest 若走 Upsert，**每次翻譯都會把使用者改正並確認過的譯名打回機器版本並取消確認** —— 正是 spec「不默默全信、避免抽錯污染」要防的事。回程必須走 insert-if-absent（見 AC #4 紅線）。

---

## Story

As a NAS owner translating a TV series in whatever episode order I please,
I want each translation run to automatically contribute the proper-noun renderings it chose back into that show's glossary,
so that every other episode — earlier or later — renders the same names without me re-typing them, and I only review suggestions instead of authoring from scratch.

---

## Acceptance Criteria

### AC #1 — `[@contract-v1]` Harvest trailer：prompt 指令與 wire 格式

`SubtitleTranslatorSystemPrompt`（共用常數）追加 harvest 指令段：翻譯回應在**所有** `[N]` 行之後，可附一段

```
===TERMS===
<term_src>=><term_zh>
```

- 只列**本 chunk 實際出現且模型做了譯名決策**的專有名詞（人名/地名/術語）；沒有就**整段省略**（trailer 是 optional 的）。
- 分隔符 `=>` 與哨兵行 `===TERMS===` 為固定 wire 格式 —— prompt 指示端與 parser 消費端是跨層契約，兩端必須同步（stamp 註解各留一份互指）。
- **`SubtitleTranslatorPromptVersion` bump `m1-v1` → `m1-v2`**：prompt 語意變了，segment cache 依設計全庫換 key（這正是 RunVersion 機制存在的目的）；pinned-hash 測試（`TestSubtitleTranslatorPromptVersion_PinsPromptText`）同步更新。
- 成本姿態依 spec：「剛翻完的模型最清楚它挑了什麼，近乎零額外成本」—— 不發第二次 LLM 呼叫。

### AC #2 — Parser：trailer 先剝離，cue 文字零污染（紅線 1）

`parseTranslationResponse` 在解析 `[N]` 行**之前**先切出並移除 trailer 段：

- **紅線**：現行 parser 把無 `[N]` 前綴的行**附掛到最近的 cue**（continuation 規則）—— 不先剝離，`===TERMS===` 與詞條行會被縫進**最後一句字幕**。測試必須釘住「有 trailer 時最後一句 cue 的文字與無 trailer 時 byte-identical」。
- 無 trailer 的回應 → 解析結果與今天 **byte-identical**（回溯相容，兩路徑既有測試全綠）。
- 格式錯誤的詞條行（無 `=>`、空 src、空 zh）→ 逐行忽略＋log Debug，合法行照收（fail-soft —— harvest 是機會性收成，不是正確性義務）。
- trailer 出現在回應中段（模型不聽話）→ 自哨兵行起全部視為 trailer；哨兵行之後不再有 cue 行被解析。

### AC #3 — 回傳鏈：terms 從翻譯層流回兩條路徑的呼叫端

- `TranslateChunk` 回傳增加 terms（`map[string]string`）；`ChunkTranslator` port 同步加寬（該 port 未 stamp；實作與測試 fakes 同步）。
- `Pipeline.TranslateTrack` 跨 chunks 合併 terms → `TranslateResult` 增加 `HarvestedTerms` 欄位（**additive** on `[@contract-v1]`，不 bump —— `default_budget_usd` 先例）。同一 src 跨 chunk 重複出現 → 第一次出現者勝（模型後見不推翻先例，跟 glossary 的固定譯名精神一致）。
- legacy 路徑：新增 `TranslateWithGlossaryHarvest`（回傳 blocks + terms），既有 `TranslateWithGlossary` 簽章**不動**、委派之（`Translate` → `TranslateWithGlossary` 同款回溯相容先例）；`translateSRT` 改呼叫新方法。
- segment cache 命中的 cue 不經 LLM ⇒ 無收成 —— 這是**機會性 harvest 的定義**，不補發呼叫（документed，不是 bug）。

### AC #4 — 回程寫入：insert-if-absent，絕不覆寫既有詞條（紅線 2）

- `GlossaryRepository` 新增 `InsertIfAbsent(ctx, term)`（`ON CONFLICT(media_id, term_src, language) DO NOTHING`）。**既有 `Upsert` 一字不動**（9R-15 REST Add 仍用它 —— 手動編輯本來就該 last-writer-wins）。
- harvest 寫入固定：`source='subtitle'`、`confirmed=0`（沿用既有 Confirm/ConfirmAll 審核流，不默默全信）。
- **紅線**：real-sqlite 測試釘住 —— 已存在的詞條（不論 confirmed 狀態、不論 source）經 harvest 再現**一個欄位都不變**；只有全新 term_src 落庫且為未確認。
- 寫入失敗 fail-soft：log Warn 後繼續（Rule 13 case 3 —— 翻譯已成功，收成失敗只是下次再收，絕不讓 run 失敗）。
- 寫入時點：**translate 階段成功之後**（兩路徑各自的成功邊界）；取消或翻譯失敗的 run 不寫入（半途的 terms 沒有隨字幕出貨，餵回去只會污染）。
- media key：episodes → **series id**（pipeline = `MediaItem.ShowKey`；legacy 既有 mediaID 已是 series id）；movies → 自身 id（spec「電影做片內一致性」—— 重譯與 9R-13 NFO localizer 共蒙其利）。

### AC #5 — Pipeline 去程：feed glossary ＋ `GlossaryVersion` 落地

- subtitle package 新增窄港口 `GlossaryStore`（Rule 11；`Lookup(ctx, mediaID) (map[string]string, error)` ＋ `InsertNew(ctx, mediaID string, terms map[string]string) error`），adapter 蓋在 `GlossaryRepository` 上（`SegmentCache`/`NewSegmentCacheRepository` 同款样式），`WithGlossaryStore` pipeline option＋main.go 接線。nil-safe：未接 = 今天的行為。
- `processItem` 在 translate 前 `Lookup`（key 同 AC #4）→ 填 `tctx.Glossary` → prompt 生出 glossary 段（`BuildGlossarySection` 既有鏈路，零新 prompt 組裝碼）。Lookup 失敗 → log Warn、以空 glossary 續跑（fail-soft，9R-10 `loadGlossary` 同款）。
- **`GlossaryVersion` = 實際餵入 prompt 的 pairs 的決定性雜湊**（sorted by source，`fieldSep` 分隔 —— `MetadataHash` 同款紀律）；**空 glossary → `""`**（既有 M1 cache entries 對無 glossary 的 show 保持有效 —— 回溯相容）。雜湊來源是「餵進 prompt 的內容」而非「DB 現況」⇒ cache key 與 prompt 內容**由建構保證一致**。
- 決定性紅線測試：同 pairs 任意順序 → 同 hash；不同 pairs → 不同 hash；空 → `""`。
- `subtitle_runs.glossary_version` 欄位（早已存在，恆存 `""`）自然開始承載真值 —— **零 migration**；`models/subtitle_run.go:75` 的「always "" in M1」註解同步改寫。
- 餵入採 `confirmedOnly=false`（全部詞條含未確認）—— **與 9R-10 legacy 既有姿態一致**（其註解明文預期 auto-mined：「Uses ALL terms (confirmed + auto-mined) for maximum intra-run consistency; the F6 review UI lets users correct mistakes」）。⚖️ 此為 authoring 裁定：spec §6.5 流程第 3 點寫「已(確認的)名詞庫」，但兩路徑一致性＋既有 9R-10 已出貨行為（改它 = AC drift）優先；髒詞條由 F6 審核流修正，修正後（`GlossaryVersion` 變）cache 自動失效重譯。**此裁定寫入 Dev Notes 並於完工回報時明示，Alexyu 可否決**（否決 = 兩路徑同步改 confirmed-only ＋ 9R-10 drift 記錄，一支 follow-up）。

### AC #6 — 可觀測性與範圍圍籬

- 兩路徑的 run 完成 log 各加 `harvested_terms`（新寫入的詞條數，不含 dedupe 掉的）；pipeline 路徑另加 `glossary_fed`（餵入的 pairs 數）。既有欄位一個不動。
- **不改**：`show_glossary` schema（零 migration）、9R-15 REST 全套、`GlossaryPanelV2`、`Upsert`、D6/SSE、任何 endpoint、Rule 7 error code（prefix 維持 16）。
- **不做**（見 Discovery Triage）：F6 對「譯於名詞庫較空時」集數的重譯提示 UI（spec 標 可選；`GlossaryVersion` 進 cache key 後，名詞庫長大 → 既有 FR12 重譯入口自然用新詞庫全量重譯，機制面已閉合，缺的只是 UI 提示）。
- 全回歸閘門：`go test ./...`、`pnpm nx test web`、`pnpm run lint:all`、`format:check`。

---

## Tasks / Subtasks

- [x] **Task 1 — Prompt harvest 段＋trailer wire 格式＋version bump（AC: #1）** 🔴 BE
  - [x] `SubtitleTranslatorSystemPrompt` 追加 harvest 指令（`===TERMS===` / `=>` 格式、只列本 chunk 決策詞、無詞省略整段）
  - [x] `SubtitleTranslatorPromptVersion` `m1-v1`→`m1-v2`＋pinned-hash 測試更新

- [x] **Task 2 — Parser 剝離＋terms 回傳鏈（AC: #2, #3）** 🔴 BE
  - [x] `parseTranslationResponse` 先切 trailer 再解析 cue；紅線測試（最後一句 cue byte-identical、無 trailer 回溯相容、malformed fail-soft、中段哨兵截斷）
  - [x] `TranslateChunk`＋`ChunkTranslator` port 加寬；`TranslateTrack` 跨 chunk 合併（first-wins）→ `TranslateResult.HarvestedTerms`（additive）
  - [x] legacy `TranslateWithGlossaryHarvest`（舊簽章委派保留）；`translateSRT` 切換

- [x] **Task 3 — `InsertIfAbsent` 反覆寫紅線（AC: #4 儲存半）** 🔴 BE
  - [x] repo 方法（`ON CONFLICT DO NOTHING`）＋real-sqlite 紅線測試（已存在詞條零變動；新詞 unconfirmed/source=subtitle）

- [x] **Task 4 — Pipeline 去程＋`GlossaryVersion`＋回程寫入（AC: #4, #5）** 🔴 BE
  - [x] `GlossaryStore` 窄港口＋adapter＋`WithGlossaryStore`＋main.go 接線（nil-safe）
  - [x] `processItem`：translate 前 Lookup → `tctx.Glossary`；translate 成功後 `InsertNew`（fail-soft）；media key = ShowKey∥ref.ID
  - [x] `runVersion`：`GlossaryVersion` = 餵入 pairs 決定性雜湊（空→`""`）；決定性紅線測試；`subtitle_run.go` 註解改寫

- [x] **Task 5 — Legacy 回程＋可觀測性＋全回歸（AC: #4, #6）** 🔴 BE
  - [x] `translateSRT` 成功後 harvest 寫入（既有 glossaryRepo，fail-soft）
  - [x] 兩路徑 log `harvested_terms`（pipeline 另加 `glossary_fed`）
  - [x] 全回歸閘門

（後端 task 5 個、前端 0 個 —— BACKEND-ONLY。）

---

## Dev Notes

### 既有可重用零件（不要重造）

| 需求 | 現成零件 |
| --- | --- |
| 儲存＋審核流 | `show_glossary`（mig 028）＋`GlossaryRepository`（9R-6）＋Confirm/ConfirmAll REST（9R-15）＋`GlossaryPanelV2`（FE 已在） |
| Prompt glossary 段 | `prompts.BuildGlossarySection`（9R-7）—— pipeline 組裝鏈已掛（pipeline.go:835），只缺資料 |
| 去程先例 | `TranscriptionService.loadGlossary`（9R-10）—— fail-soft 姿態照抄 |
| 窄港口＋adapter 樣式 | `subtitle.SegmentCache`／`NewSegmentCacheRepository`（sub-1-5b） |
| 決定性雜湊紀律 | `MetadataHash`（segment_cache.go）—— sorted copy＋`fieldSep` |
| 回溯相容方法擴充先例 | `Translate` → `TranslateWithGlossary` 委派（9R-7） |
| additive-no-bump 先例 | `TranslateResult` 加欄位比照 sub-5-1 `default_budget_usd` |

### 關鍵決策（authoring 已裁）

- **harvest 指令進共用 system prompt 常數**：兩路徑同一常數（pipeline.go:832 / translation_service.go:194,290）、同一 parser —— 各改一處雙路生效。不做 pipeline-only 版本（那會讓預設部署〔legacy 模式〕收不到任何東西）。
- **InsertIfAbsent 而非 Upsert**：Upsert 的 DO UPDATE 會把使用者改正＋確認過的譯名打回機器版並取消確認。spec「去重…merge 累積；新詞以未確認寫入」= 只加新、不動舊。並發同劇兩集同時收成同一新詞 → UNIQUE 約束下第二筆 DO NOTHING，天然安全。
- **餵入 confirmedOnly=false**（見 AC #5 ⚖️）—— 與 9R-10 一致、避免 drift；等 Alexyu 否決再收緊。
- **GlossaryVersion 從「餵入內容」算**：不是 DB 快照 —— cache key 與 prompt 一致性由建構保證；Lookup 失敗餵空 → version=`""` → cache 語意仍正確。
- **順序無關由架構自然成立**：glossary per-show 累積（key=series id）、任一集收成、任一集受惠；名詞庫長大 → `GlossaryVersion` 變 → 舊 cache 失效 → 回頭重譯早集自動用新詞庫（spec「向前補一致性」的機制面免費交付）。
- **PromptVersion bump 的成本**：segment cache 全庫換 key = 已交付字幕的既有快取條目 orphan（30 天 TTL 自然回收）；正常流程不重譯已交付項目，僅顯式重譯多付一次 —— 這是 prompt 變更的設計成本，D4 機制的目的就是讓它「正確地」發生。
- **first-wins 跨 chunk 合併**：同 src 多譯名時取先例 —— 與 glossary「固定譯名」精神一致，且決定性（map 迭代序不參與）。

### 契約姿態（Rule 20）

- **消費**：**confirmed against `[@contract-v1]` (sub-1-5b TranslateContext)** —— `Glossary` 欄位是 D4 預留（「field exists NOW」）、`MetadataHash` 明文排除 glossary（「it has its own RunVersion field」）：填入資料 = 設計內啟用，**不 bump**；ack＋Change Log。**confirmed against `[@contract-v1]` (TranslateResult)** —— `HarvestedTerms` additive，不 bump。
- **產生**：trailer wire 格式 `[@contract-v1]`（prompt 指示端 ↔ parser 消費端跨層契約；兩端 inline stamp 互指）。
- `ChunkTranslator` port 未 stamp（pipeline.go:26-35 驗證過）—— 加寬時同步 fakes 即可。
- `models/subtitle_run.go:75` GlossaryVersion 註解改寫（「always "" in M1」失效）—— 語意文件同步，非 wire 變更。
- 9R-10 `loadGlossary`／9R-15 REST／`Upsert`：零改動 ⇒ 無 drift。

### 已知限制（記錄，不在本 story 解）

- segment cache 命中的 cue 無收成（機會性 harvest 的定義）；一部全命中重譯的劇收成為零 —— 正確，因為沒有新決策發生。
- 模型不聽話（不吐 trailer、格式錯）→ 收成為零或部分，run 照常成功 —— harvest 永遠不是正確性義務。
- Gemini 路徑（`CompleteText` 降級）同樣吃到 harvest 指令與 parser —— 行為一致，無特判。

### Time-dependent visual coverage

`N/A — 100% backend, no apps/web/ files touched.`

### References

- [Source: `_bmad-output/planning-artifacts/vido-subtitle-pipeline-spec.md:114-130`] — §6.5 全文（閉環、順序無關、復用清單、流程 3 步）
- [Source: sprint-status `epic-subtitle-pipeline-m3` seed] — G:sub-5-5 範圍句
- [Source: `9R-6-7-glossary-keystone.md` / `9R-15-glossary-http-api.md`] — schema/repo/REST/審核流
- [Source: `apps/api/internal/subtitle/pipeline.go:26-35,58,580-700,806-845`] — port、TranslateContext、chunk 迴圈、system blocks
- [Source: `apps/api/internal/services/translation_service.go:127-135,194,290,310-360,374+`] — 兩入口、共用 system prompt、`parseTranslationResponse` continuation 規則
- [Source: `apps/api/internal/services/transcription_service.go:210-232,837`] — legacy 去程與寫入點
- [Source: `apps/api/internal/subtitle/segment_cache.go:95-165`] — MetadataHash 紀律、runVersion、GlossaryVersion 預留
- [Source: `apps/api/internal/subtitle/media_store.go:60-130`] — ShowKey 解析（glossary key 來源）
- [Source: `apps/api/internal/repository/glossary_repository.go` Upsert] — DO UPDATE 覆寫語意（紅線 2 的動機）
- [Source: `apps/api/internal/config/config.go:160`] — pipeline mode 預設 legacy（雙路徑必要性）
- [Source: `project-context.md`] — Rule 4/11/13/19/20/24

### Previous Story Intelligence（sub-5-4）

- 紅線測試先行（RED gate 留證據）＋可否證的價值主張斷言（「全命中零探測」式）—— 本 story 對應「有 trailer 時最後一句 cue byte-identical」與「已存在詞條零變動」。
- fail-soft 一律 Rule 13 case 3 註解；port 加寬時 fakes 同步；`gofmt` 只保證自己觸及的檔（`backlog-go-gofmt-not-enforced` 存量勿清）。
- CR 慣例：TTL/type 這類「傳遞鏈」也要測試釘住 —— 本 story 的對應是 `source='subtitle'`/`confirmed=0` 寫入值鏈。

---

## Dev Agent Record

### Agent Model Used

Claude Fable 5 (claude-fable-5) — dev-story workflow, 2026-08-18

### Debug Log References

- RED gate 1（AC #1）：prompt 追加 harvest 段後，`TestSubtitleTranslatorPromptVersion_PinsPromptText` 以 digest mismatch 失敗（`4ca01f92…` → `dd8e754f…`）→ bump `m1-v2` ＋ pinned digest 同編輯更新 → GREEN。
- RED gate 2（AC #2 紅線 1）：`TestParseTranslationResponse_TrailerRedLines` 對「未剝離」的 parser 實跑失敗，實證 continuation 規則把 `===TERMS===\nA=>甲` 縫進最後一句 cue（actual: `{1:"你好\n===TERMS===\nA=>甲", 2:"世界"}`）→ 接上 `splitHarvestTrailer` → GREEN。
- RED gate 3（AC #4 紅線 2）：`TestGlossaryRepository_InsertIfAbsent_*` 於方法不存在時失敗（compile RED）；real-sqlite 行為紅線（已存在 confirmed/manual 詞條經 harvest 再現後 `assert.Equal(before[0], after[0])` 全欄位不變）首跑即須通過 DO NOTHING 實作驗證。
- RED gate 4（AC #5）：`TestGlossaryVersionHash_Deterministic` 於函式不存在時失敗 → 實作 → GREEN（任意順序同 hash／不同 pairs 異 hash／空→`""`／field boundary 不可偽造）。

### Completion Notes List

- **AC #1** — harvest 指令段追加於共用 `SubtitleTranslatorSystemPrompt`（兩路徑、含 Gemini `CompleteText` 降級路徑同吃）；`===TERMS===`／`=>` wire 格式 `[@contract-v1]` inline stamp 兩端互指（prompt 常數 ↔ `splitHarvestTrailer`）。`SubtitleTranslatorPromptVersion` `m1-v1`→`m1-v2`；零額外 LLM 呼叫（terms 從同一回應解析）。
- **AC #2** — `parseTranslationResponse` 簽章不變（內部委派 `parseTranslationResponseWithTerms`，先剝 trailer 再解析 cue），既有 4 個呼叫端（含 9R-13 `TranslateRequest`）自動獲得零污染；malformed 詞條逐行 Debug-log 忽略；中段哨兵起全部視為 trailer。
- **AC #3** — `TranslateChunk` 加寬為 4 回傳值；`ChunkTranslator` port（未 stamp）同步、3 個測試 fakes 同步。`TranslateTrack` 跨 chunk（含 semantic retry 回應）first-wins 合併 → `TranslateResult.HarvestedTerms`（additive on `[@contract-v1]`，不 bump）。legacy 新增 `TranslateWithGlossaryHarvest`，`TranslateWithGlossary` 舊簽章委派保留；`translateSRT` 已切換。segment-cache 命中 cue 無收成＝機會性 harvest 定義（`translateWithCache` 註解記錄）。
- **⚠️ CR 更正**：實作時的宣稱「legacy 既有 mediaID 已是 series id」經 CR 查證為**假**（`asr_adapter.go` 與 `generation_batch_runner.go` 對 episode 傳的都是 episode id）——已由 CR H1 修正（`glossaryMediaKey` 解析 series id），見 Senior Developer Review。
- **AC #4** — `GlossaryRepository.InsertIfAbsent`（`ON CONFLICT(media_id, term_src, language) DO NOTHING`，回傳 inserted bool）；既有 `Upsert` 一字未動。harvest 寫入固定 `source='subtitle'`、`confirmed=0`（adapter 與 legacy helper 兩處值鏈皆有測試釘住）。寫入時點＝translate 階段成功之後（pipeline：`translateWithCache` merge＋invariant 通過後；legacy：翻譯＋place 成功後）；翻譯失敗／取消不寫入（`TestProcessItem_FailedTranslateHarvestsNothing`）。media key：episodes→ShowKey（series id）、series→自身、movies→ref.ID（`glossaryKeyFor`，兩測試釘住）。寫入失敗 fail-soft log Warn 續跑。
- **AC #5** — `subtitle.GlossaryStore` 窄港口＋`NewGlossaryStoreRepository` adapter（SegmentCache 同款樣式）＋`WithGlossaryStore`＋main.go 接線，nil-safe。`feedGlossary` 於 `runVersion` 計算**之前**執行（cache key 與 prompt 內容由建構保證一致）；entries 依 Source 排序（prompt 決定性）；Lookup 失敗 fail-soft 空 glossary 續跑（`GlossaryVersion` 即 `""`，語意自洽）。`GlossaryVersionHash`：sorted-copy＋`fieldSep`＋sha256（`MetadataHash` 紀律）、空→`""`（M1 既有 cache entries 回溯相容）。`subtitle_runs.glossary_version` 零 migration 開始承載真值；`models/subtitle_run.go` 註解改寫。餵入 `confirmedOnly=false`（⚖️ authoring 裁定照案執行，Alexyu 可否決——否決即開 follow-up 兩路徑同步收緊）。
- **AC #6** — pipeline 完成 log 加 `glossary_fed`＋`harvested_terms`；legacy 完成 log 加 `harvested_terms`（既有 `glossary_terms`＝餵入數保留）。`show_glossary` schema／9R-15 REST／`GlossaryPanelV2`／`Upsert`／D6 SSE／endpoint／Rule 7 前綴（16）全部零改動。全回歸：`go test ./...` 全綠、`pnpm nx test web` 233 files / 2653 tests 全綠、`pnpm run lint:all` 0 errors（119 個既有 warnings，屬 retro-11-AI1b 存量）、prettier check 全綠、gofmt 本 story 觸及 20 檔全 clean（存量未格式化檔屬 `backlog-go-gofmt-not-enforced`，未擴 scope）。
- **⚠️ 小幅偏離 story 字面**：AC #5 的 port 簽章寫 `InsertNew(...) error`，實作為 `InsertNew(...) (int, error)` —— AC #6 要求 log `harvested_terms`（新寫入數、不含 dedupe），計數必須由寫入端回傳。語意不變、僅回傳值加寬。
- **Pre-existing fix**（Epic 9c Retro AI-2 選項 1，就地修）：`generation_batch_test.go` 的 `drainEvents` 原以固定 50ms sleep 賭 SSE fan-out 完成，負載下偶發空集合使 `TestGenerationBatch_RequestedBudgetOverridesDefault` flake（本次全回歸期間實際命中 2 次）。改為 bounded-wait（首事件最多等 2s 再 non-blocking drain），9 個共用呼叫端一次修復；與本 story 功能無關。
- 🔗 AC Drift: NONE (checked: 'TranslateWithGlossary|parseTranslationResponse|GlossaryVersion|SubtitleTranslatorSystemPrompt' across _bmad-output/implementation-artifacts/*.md — 8 hits, all REUSE not DRIFT; PromptVersion bump 是 RunVersion 機制的設計內行為、`TranslateWithGlossary`/`Translate` 舊簽章委派保留、`HarvestedTerms` additive 不動 sub-1-5a 契約)
- 📎 Contract Stamps: FOUND (4 stamped-AC 引用 across 3 files — 本 story 產生 trailer wire `[@contract-v1]`（prompt↔parser inline stamp 互指）；消費 confirmed against `[@contract-v1]` (sub-1-5b TranslateContext)——`Glossary` 欄位為 D4 預留「field exists NOW」、`MetadataHash` 明文排除 glossary，填入資料＝設計內啟用不 bump；confirmed against `[@contract-v1]` (sub-1-5a TranslateResult)——`HarvestedTerms` additive 不 bump（`default_budget_usd` 先例）。`ChunkTranslator` port 未 stamp（驗證過），加寬＋fakes 同步即可)
- 🎭 A11y Pre-Flight: N/A (100% backend — no apps/web/ files touched)
- 🎨 UX Verification: SKIPPED — no UI changes in this story

### Discovery Triage

- **Did this story discover any work outside its current scope?**
  - **YES** — filed at authoring time（2026-08-18）：
    - **③ backlog-with-carry-forward-link** — `backlog-glossary-stale-run-retranslate-hint`：spec §6.5 的可選項「對名詞庫還很空時就譯好的集數提供重譯入口」。機制面本 story 已閉合（`GlossaryVersion` 進 cache key ⇒ 名詞庫長大後走既有 FR12 重譯即全量用新詞庫），缺的是**產品面提示 UI**（哪些集譯於舊版詞庫、一鍵重譯）。UX 裁定＋FE 工作，非阻塞。

### File List

- `apps/api/internal/ai/prompts/subtitle_translator.go` — modified：system prompt 追加 Term harvest 段（wire 格式 `[@contract-v1]` stamp）；`SubtitleTranslatorPromptVersion` `m1-v1`→`m1-v2`
- `apps/api/internal/ai/prompts/subtitle_translator_test.go` — modified：pinned-hash 測試更新（version＋digest）
- `apps/api/internal/services/translation_service.go` — modified：`splitHarvestTrailer`（parser 端 stamp）＋`parseTranslationResponseWithTerms`；`TranslateChunk` 加寬（+terms）；`TranslateWithGlossaryHarvest` 新增、`TranslateWithGlossary` 委派；`mergeHarvestedTerms`
- `apps/api/internal/services/translation_service_test.go` — modified：trailer 紅線測試、`splitHarvestTrailer` 測試、`TranslateWithGlossaryHarvest` 測試、`TranslateChunk` 4-值簽章同步
- `apps/api/internal/services/transcription_service.go` — modified：`translateSRT` 切換 harvest 版；`harvestGlossaryTerms`（legacy 回程，fail-soft）；完成 log 加 `harvested_terms`；CR H1 `glossaryMediaKey`（episode→series id）＋CR H2 term_zh 過 OpenCC
- `apps/api/internal/services/transcription_translation_test.go` — modified（CR）：`translateSRT` 簽章同步（+mediaType）
- `apps/api/internal/services/transcription_service_test.go` — modified：`stubGlossaryRepo` 實作 `InsertIfAbsent`；legacy 回程測試（值鏈＋dedupe＋trailer 不入 SRT）
- `apps/api/internal/services/nfo_localizer_service_test.go` — modified：stub 補 `InsertIfAbsent`（介面同步）
- `apps/api/internal/services/generation_batch_test.go` — modified：pre-existing flake 就地修（`drainEvents` bounded-wait，見 Completion Notes）
- `apps/api/internal/subtitle/pipeline.go` — modified：`ChunkTranslator` port 加寬；`TranslateResult.HarvestedTerms`（additive）；`glossary` 欄位＋`WithGlossaryStore`；`processScope.harvestedTerms`；chunk 迴圈 first-wins 合併＋`mergeChunkTerms`
- `apps/api/internal/subtitle/process_item.go` — modified：`feedGlossary`（version 計算前）＋`harvestTerms`（translate 成功後）＋`glossaryKeyFor`；完成 log 加 `glossary_fed`/`harvested_terms`
- `apps/api/internal/subtitle/segment_cache.go` — modified：`GlossaryVersionHash`（決定性雜湊，空→`""`）；`runVersion` 改吃餵入內容；註解改寫
- `apps/api/internal/subtitle/glossary_store.go` — new：`GlossaryStore` 窄港口＋`NewGlossaryStoreRepository` adapter（source=subtitle/confirmed=0 值鏈）
- `apps/api/internal/subtitle/glossary_store_test.go` — new：adapter 值鏈／best-effort／Lookup 測試
- `apps/api/internal/subtitle/pipeline_test.go` — modified：`fakeTranslator` 加 terms hook；跨 chunk first-wins 測試＋nil-harvest 測試
- `apps/api/internal/subtitle/process_item_test.go` — modified：`fakeGlossaryStore`＋6 個 feed/harvest/fail-soft/key 測試；`ctxBudgetSpy` 簽章同步
- `apps/api/internal/subtitle/segment_cache_test.go` — modified：`GlossaryVersionHash` 決定性紅線測試；prompt-version 突變 fixture 撞名修正（`m1-v2`→`m1-v99`）
- `apps/api/internal/repository/glossary_repository.go` — modified：介面＋實作 `InsertIfAbsent`（DO NOTHING，回傳 inserted）
- `apps/api/internal/repository/glossary_repository_test.go` — modified：real-sqlite 紅線測試（已存在詞條零變動／新詞 unconfirmed／validation）
- `apps/api/internal/models/subtitle_run.go` — modified：`GlossaryVersion` 註解改寫（「always "" in M1」失效）
- `apps/api/cmd/api/main.go` — modified：`WithGlossaryStore(NewGlossaryStoreRepository(repos.Glossary))` 接線

---

## Senior Developer Review (AI)

**Date:** 2026-08-18 · **Outcome:** Approve（2 HIGH ＋ 2 MEDIUM 全數當場修復；1 LOW 記錄）
**檢查:** Git↔File List 0 差異 · 🔒 Rule 7: PASS（0 新 error code）· 🔒 Rule 20 Bump: N/A（僅新 v1 stamp）· 🔒 Rule 25: N/A

### Action Items

- [x] **[H1] legacy 路徑 episode 的 glossary key 錯用 episode id** — 實作宣稱「legacy 既有 mediaID 已是 series id」為假：`asr_adapter.go:33` 與 `generation_batch_runner.go:39` 對 episode 傳 episode id → feed 恆空、harvest 落在 F6 面板（用 series id，`LocalDetailV2.tsx:294`）看不到的孤兒 rows，「任一集收成、全劇受惠」在 ASR 路徑失效。**修**：`TranscriptionService.glossaryMediaKey`（episode 經 `episodeReader.FindByID().SeriesID` 解析，nil-safe 回退）；`translateSRT` 穿入 mediaType，feed＋harvest 同 key。測試：`TestTranscriptionService_TranslateSRT_EpisodeGlossaryKeysOnSeries`。順帶修復 9R-10 既有的 episode feed 恆空 gap。
- [x] **[H2] 收成 term_zh 未過 OpenCC → 簡體污染回饋迴圈** — 簡體譯名入庫後成為 MANDATORY 固定譯名 → 品質閘門逢含該詞句必退件 → stubborn 累積可致整集失敗，直到 F6 人工修正。**修**：兩路徑寫入前 s2twp（pipeline `harvestTerms` 用 `p.converter`；legacy `harvestGlossaryTerms` 用 `s.opencc` nil-safe），fail-soft 保留原值。測試：`TestProcessItem_HarvestedTermsGetOpenCC`、`TestTranscriptionService_TranslateSRT_HarvestedTermGetsOpenCC`。
- [x] **[M1] 被整批退件的 attempt 的 trailer 仍被 first-wins 合併** — 錯誤譯名搶先佔位、重試的正確譯名進不來。**修**：merge 移至 verdict 之後，gate＝該 attempt 至少一句 cue 被接受（部分接受仍收成）。測試：`TestTranslateTrack_WhollyRejectedAttemptHarvestsNothing`。
- [x] **[M2] 恆等映射（`Vecna=>Vecna`）未過濾** — 模型不聽話時垃圾 rows 要人工清。**修**：`splitHarvestTrailer` 逐行忽略 src==zh。測試：identity-mapping subtest。
- [ ] **[LOW] `feedGlossary` 在 pre-flight 之前跑** — 早退 item 白付一次 SQLite 點查（設計取捨：version 須先於 preflight 計算以保 resume 分類準確；成本 µs 級）。記錄不修。

### CR 後全回歸

`go test ./...` 全綠 · `pnpm nx test web` 233 files / 2653 tests 全綠 · gofmt 觸及檔全 clean。

## Change Log

| Date | Change |
| --- | --- |
| 2026-08-18 | Code review (adversarial) — 2 HIGH + 2 MEDIUM 全數修復，Status → done。H1：legacy episode glossary key 錯用 episode id（實作宣稱「已是 series id」查證為假）→ `glossaryMediaKey` 解析 series id，feed＋harvest 同修（順帶閉合 9R-10 episode feed 恆空的既有 gap）。H2：harvest term_zh 未過 OpenCC 的簡體污染迴圈 → 兩路徑寫入前 s2twp。M1：整批退件 attempt 的 trailer 不再 first-wins 佔位。M2：恆等映射過濾。LOW（feed 先於 preflight 的 µs 級點查）記錄不修。CR 後全回歸 Go＋web＋gofmt 全綠。 |
| 2026-08-18 | dev-story 實作完成（Task 1–5 全數，Status → review）。RED-first 四道閘（pinned-hash／trailer 縫入實證／InsertIfAbsent／hash 決定性）。`[@contract-v1]` trailer wire 格式新產生（prompt↔parser 互指 stamp，新 v1 無 stale-mark 義務）；消費 ack：confirmed against `[@contract-v1]` (sub-1-5b TranslateContext，D4 預留欄位啟用不 bump)、confirmed against `[@contract-v1]` (sub-1-5a TranslateResult，`HarvestedTerms` additive 不 bump)。`SubtitleTranslatorPromptVersion` m1-v1→m1-v2（segment cache 依設計全庫換 key）。偏離：`GlossaryStore.InsertNew` 回傳 `(int, error)` 非 story 字面的 `error`（AC #6 計數需要）。Pre-existing fix：`drainEvents` SSE flake bounded-wait。全回歸：Go 全綠、web 2653 tests 全綠、lint 0 errors、prettier 全綠。 |
| 2026-08-18 | create-story：M3 G 群組（最後一支）。盤點確立不對稱矩陣：legacy（預設模式）有去程無回程、pipeline 去程 seam 恆空且無回程；兩路徑在 `SubtitleTranslatorSystemPrompt`＋`parseTranslationResponse` 收斂 ⇒ harvest 各改一處雙路生效。兩條紅線：trailer 先剝離（continuation 規則會把詞條縫進最後一句 cue）、`InsertIfAbsent` 反覆寫（既有 Upsert 會打回已確認詞）。`GlossaryVersion` 從餵入內容算（D4 預留欄位啟用，零 migration）。⚖️ 餵入 confirmedOnly=false（與 9R-10 一致，spec 字面偏離已記錄待 Alexyu 否決）。lane ③×1（重譯提示 UI）。 |
