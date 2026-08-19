# Story: Bugfix J — ASR 路徑部分翻譯失敗的真實狀態（hasPartialFailure 貫穿到 verdict）

Status: ready-for-dev

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

**Source:** 2026-08-19 What'Sub 競品對抗重審（party-mode）—— code-gap lens 發現、skeptic 查證確認。無悔四項之 N1（P/O/C 三情境交集，第一順位）。
**Risk: 🟡 MED-LOW** —— 純誠實性修復；一個跨層訊號貫穿＋一個 verdict 分支改判。不新增狀態值、不 bump 契約。
**Exposure:** sub-3-1 完工紀錄：抽樣媒體庫 **68.3%（142/208）走 ASR 路徑**（sprint-status `sub-3-1` 條目）—— 本缺陷坐落在唯一繁中來源的**多數路徑**上。

---

## Story

As a NAS owner whose poster badges are the product's core honesty promise,
I want an ASR-generated subtitle whose translation PARTIALLY failed to be recorded as `untranslated` instead of `found`/`zh-Hant`,
so that a file that still contains whole batches of raw English is never presented as「繁中字幕已就緒」—— the badge tells the truth on the majority path, not just the extract path.

---

## Context —— 缺陷機制（已逐行查證）

1. `TranslationService.TranslateWithGlossaryHarvest`（`translation_service.go:148`）在批次翻譯失敗時**整批保留英文原文並繼續**（AC #5 of 9-2b，`:223-241`：`hasPartialFailure = true; continue`），逐 cue 缺漏也靜默保留英文（`:252-258`「If translation not found for this block, original English text is kept」）。
2. `hasPartialFailure` 在函式尾端**只降級成一行 `slog.Warn` 就被丟棄**（`:273-278`），回傳 `err == nil` —— 呼叫端拿不到任何訊號。
3. `TranscriptionService.translateSRT`（`transcription_service.go:823`，呼叫點 `:863`）拿到「成功」，照常 OpenCC、照常 place，回傳非空 `zhSRTPath`。
4. `translateAndPersist` 的 verdict switch（`transcription_service.go:679-693`）看到 `zhSRTPath != ""` → 寫入 `SubtitleStatusFound` + `zh-Hant`。**混了整批英文的檔案，徽章顯示繁中已就緒。**
5. 對照組：extract 路徑有逐 cue 品質閘門＋FR17 時間戳斷言（`subtitle/quality_gate.go:50`、`subtitle/pipeline.go:694,748`）；ASR 路徑**完全不經過**那套機制（`cmd/api/asr_adapter.go:33` 橋接進 `TranscriptionService`，legacy 與 pipeline 兩模式同路）。本 story 是最小誠實性修復，**不是**把 ASR 路徑併入閘門化 `TranslateTrack` 的那個大重構（後者另案，見 sprint-status `backlog-asr-leg-unify-gated-pipeline`）。

### 既有語意可直接復用（不發明新狀態）

`SubtitleStatusUntranslated`（`models/movie.go:102-110`，sub-2-2a）的定義正是本案需要的：「generated subtitle exists on disk but the EXPECTED translation step did not run … Recoverable by re-running 生成字幕, which resumes **translate-only**（English SRT on disk is reused; extract+ASR are skipped）」。部分失敗 = 翻譯步驟沒有完整跑完 → 語意吻合，且 resume 路徑（translate-only、免重付 ASR）現成。

---

## Acceptance Criteria

### AC #1 — 訊號貫穿：`TranslateWithGlossaryHarvest` 回傳 partial 訊號
- 簽名改為 `(result []TranslationBlock, harvested map[string]string, partial bool, err error)`（或等價的具名結果型別；擇一，全 repo 一致）。
- `partial == true` 的判定涵蓋**兩種**既有靜默保英路徑：(a) 批次整批失敗（`:223-241`）；(b) 逐 cue 缺漏 —— 回應中缺某 block 的譯文（`:252-258`；以 `len(translations)` 對 `len(batch)` 之差計入）。
- Wrapper `TranslateWithGlossary`（`:136-141`）同步更新（照舊丟棄 harvest；partial 訊號**必須**透傳，不得吞掉）。
- 兩個生產呼叫點全部更新；`grep -rn "TranslateWithGlossaryHarvest\|TranslateWithGlossary(" apps/api --include="*.go"` 無遺漏編譯錯誤。

### AC #2 — verdict 改判：partial → `untranslated`，混合檔仍落地
- `translateSRT` 將 partial 訊號回傳給 `translateAndPersist`。
- **檔案照常 place**（混合檔今晚仍有觀看價值；OpenCC 照跑 —— 英文 cue 過 OpenCC 是 no-op）。
- verdict switch 改為三態：
  - `zhSRTPath != "" && !partial` → `found` + `zh-Hant`（現行成功路徑，byte-unchanged）；
  - `zhSRTPath != "" && partial` → **`untranslated` + `srtPath`（EN 路徑）** —— 與 budget-pause 分支（`:657-663`，CR sub-2-2a M1）同款寫法：EN 路徑是 resume enabler，resume 走 translate-only 重譯後以 placer 覆寫混合檔；
  - `translate && zhSRTPath == ""` → `untranslated`（現行，不動）。
- 🔴 紅線：不新增 `SubtitleStatus` 枚舉值（`partial` 狀態 = 契約 bump + FE 徽章工程，明確 out of scope）。

### AC #3 — 可觀測性
- partial 終局的 SSE complete 事件 message 含「部分翻譯失敗」字樣與保英 cue 數（沿用既有 `broadcastEvent` 欄位型態，additive，不動既有 key）。
- `slog.Warn`（`:273-277`）保留，補上 `english_kept_blocks` 計數欄位。

### AC #4 — 測試
- `translation_service_test.go`：(a) 批次失敗 → `partial=true`；(b) 逐 cue 缺漏 → `partial=true`；(c) 全成功 → `partial=false`（回歸釘）。
- `transcription_service_test.go`：partial → 寫 `untranslated`+EN 路徑且檔案已 place；非 partial → `found`（回歸釘）。
- 既有 26 個 Claude-touching 測試與 retry guards 全綠（媒體掃描回歸紅線，subtitle-pipeline epics 慣例）。

### AC #5 — 已知留白（記錄，不解）
- 重試會對已成功批次**重新付費**（legacy 路徑無 segment cache）—— 既有已立案問題，見 `backlog-translate-budget-partial-progress`（sprint-status，2026-08-12）；本 story 不解，僅在 Dev Notes 引用。
- attach 模式（items[] 缺席）的徽章更新時序不在本案範圍。

## Tasks / Subtasks

- [ ] Task 1: AC #1 訊號貫穿（translation_service.go + wrapper + 呼叫點）
- [ ] Task 2: AC #2 verdict 三態改判（transcription_service.go）
- [ ] Task 3: AC #3 SSE/log 可觀測性
- [ ] Task 4: AC #4 測試（含回歸釘）
