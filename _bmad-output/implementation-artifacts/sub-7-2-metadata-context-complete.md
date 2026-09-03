# Story 7.2: 翻譯 prompt 的 metadata 補齊 —— Cast 死碼、genres、countries（後端）

Status: ready-for-dev

## Story

As a BYOK NAS owner,
I want the show context I already paid TMDb calls for to actually reach the translator,
so that character names and register stop being guessed from a one-line overview.

## Context

eval-1「Metadata 注入實況」：`BuildMetadataSection` 七欄實際只送 3～4 欄——**Genres 九部全 `[]`、Production countries 全空且 `seriesContext` 未填、Cast 從未被賦值**（`TranslateContext.Cast` 只在 `pipeline.go:955` 被讀，`media_store.go` 兩條 load 路徑都沒寫 → `MetadataCastLimit=10` 是死碼）。而 cast **已經存在 DB**：`movie.GetCredits().Cast`／`series.GetCredits().Cast`（`models/movie.go:169-189,245`、`series.go:65`）。

**前置：** sub-6-7（未比對不送檔名）先落地，讓本 story 只加訊號不加雜訊。

## Acceptance Criteria

1. **Cast 接上。** `media_store.go` 電影與 `seriesContext` 填 `Cast`：每項「`Name`（`Character`）」（角色名有值才加括號），取前 `MetadataCastLimit` 位（credits 已依 order 排序）。
2. **Genres／Countries。** 查清為何九部 `genres=[]`（enrichment 沒寫？scan 掃描寫空？）並修根因；`seriesContext` 補 `Countries: countryCodes(series.ProductionCountries)`（若 series 無此欄，加欄或從 origin_country 取，Rule 15 同步）。
3. **集數層級 title。** `loadEpisode` 仍套 series context（prompt cache 跨集命中的前提，**不變**），但 additive 傳 `EpisodeTitle` 給 `TranslateContext`，`BuildMetadataSection` 以獨立一行「Episode: SxxEyy Title」渲染在 **cache 前綴之外**（放 user prompt 首行而非 system block），不破 cache。
4. **`metadata_hash` 語意。** Cast／genres 進 hash → 既有 completed run 的 `FindCompletedRun` 版本不再命中屬預期（重跑會付費）；pre-flight 的 sidecar 閘門仍優先（`preflightSkip`）所以**不會自動重跑**，只有 force 時才會。文件於 story 記錄，`docs/deployment*.md` 加一句。
5. **測試。** metadata section 快照測試：有 cast／無 cast、genres、countries、episode 行位置（在 user prompt 不在 system）；hash 變更斷言；eval 用 `eval/boys` 的 metadata 值當 fixture（不含對白）。

## Tasks / Subtasks

- [ ] **Task 1 — Cast 與 countries 接線（AC: #1, #2）**
- [ ] **Task 2 — genres 根因（AC: #2）**
- [ ] **Task 3 — Episode 行（AC: #3）+ 文件（AC: #4）**
- [ ] **Task 4 — 測試（AC: #5）**

## Dev Notes

- `TranslateContext` `[@contract-v1]`：加 `EpisodeTitle` 是 additive，不 bump、Change Log 記。
- Prompt 規則 3「人名保留英文」與 `===TERMS===` 語意拉扯（eval-1 已知槓桿 4）**不在本 story**——屬 sub-7-9／prompt 版本工作。

### Time-dependent visual coverage

- N/A。

### References

- eval-1「Metadata 注入實況」「發現 10、11」；`prompts/subtitle_translator.go:110-175`；`subtitle/media_store.go:95-112,140-215`

## Dev Agent Record

### Agent Model Used

### Completion Notes List

### Discovery Triage

- （dev 填）

### File List
