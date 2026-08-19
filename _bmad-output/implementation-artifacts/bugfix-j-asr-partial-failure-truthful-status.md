# Story: Bugfix J — ASR 路徑部分翻譯失敗的真實狀態（hasPartialFailure 貫穿到 verdict）

Status: review

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

- [x] Task 1: AC #1 訊號貫穿（translation_service.go + wrapper + 呼叫點）
- [x] Task 2: AC #2 verdict 三態改判（transcription_service.go）
- [x] Task 3: AC #3 SSE/log 可觀測性
- [x] Task 4: AC #4 測試（含回歸釘）

---

## Dev Agent Record

### Implementation Plan
- AC #1 以具名結果型別落地：`TranslationOutcome{EnglishKeptBlocks, TotalBlocks}` + `Partial()`（story 的「或等價的具名結果型別」選項）—— 比裸 bool 多帶計數，AC #3 的訊息與 log 同源，且為 9R-8 的 metadata 參數擴充預留簽名空間（協調條款：本 story 只加回傳值，9R-8 加參數，零衝突）。
- partial 判定涵蓋兩路：批次整批失敗（`englishKept += len(batch)`）＋逐 cue 缺漏（response 缺 index → `englishKept++`）。
- verdict 三態：`partial → untranslated+EN 路徑（混合檔仍 place）`；寫入失敗**傳播**（Rule 13 —— 此為 verdict write 非 best-effort，與 found/untranslated 分支一致；budget-pause 分支的 best-effort 語意原樣保留）。
- `Translate`（POC 專用入口）明示丟棄 outcome＋註解；`TranslateWithGlossary` 透傳（AC #1 紅線）。

### Debug Log
- 簽名變更使嚴格 compile-broken RED 不可行 → 以行為斷言達成紅綠：新測試先斷言 outcome/verdict 行為，實作前該行為不存在（found 誤判）。
- `pnpm nx test api` 首輪紅 → 定位為兩個與本 story 無關的環境性問題（見 Completion Notes），非回歸。

### Completion Notes
- 🔗 AC Drift: FOUND — 9R-16 AC 12（經 sub-2-2a 已為 [@contract-v2]）的 verdict 契約再次改變：`found` 由「zh 路徑非空」收窄為「完整翻譯成功」，partial 降級 untranslated+EN 路徑 → 本 story 執行 **[@contract-v2→v3] bump**（Change Log 列）。grep 下游 v2 ackers（sub-2-2b/sub-3-1/sub-4-3/sub-5-1/ux3-ai-1/ux3-ai-2 等）：**全部 done → frozen，無 stale-mark 欠務**。9R-10a（in-dev，另一 session 持有）僅消費 `untranslated` 的 resume 語意 —— 本改動為相容性擴充（更多列成為 resume-eligible），**未改其檔案**（避免跨 session 檔案衝突），義務以本列記錄。
- 📎 Contract Stamps: FOUND（上游 9R-16 AC 12 [@contract-v2] 於 transcription_service.go:698 註解區 —— 本 story bump 至 v3，程式碼註解同步改寫；本 story 自身不新增 stamp）。
- 🎭 A11y Pre-Flight: N/A (100% backend — no apps/web/ files touched)。
- Pre-existing fix: `TestScannerService_StartScan_PermissionDenied` 在 root 容器 4/4 必敗（root 無視權限位 → ErrorCount=0）→ 加標準 root-skip guard（`os.Geteuid()==0 → t.Skip`），乾淨 main stash 驗證為既有問題。
- Pre-existing FILED ×2（stash 驗證皆於乾淨 main 重現）：`preexisting-flake-generation-batch-cancel-sse-drain`（CancelMidItem 容器內 2/4 flake，SSE drain 時序）、`preexisting-fail-web-vitest-nonzero-exit-container`（web 2653/2653 全過但 vitest 因 jsdom unhandled navigation error 非零退出；CI 綠）。
- AC #3 SSE 訊息：completeMsg 於 partial 終局附「（部分翻譯失敗，N 句保留英文）」＋additive `english_kept_blocks` key（僅 partial 出現）；無既有 SSE hub 測試 fake，訊息組裝由 outcome 單元測試間接覆蓋（repo 既有慣例 —— complete 事件測試僅釘常數名）。
- AC #5 留白照案記錄：重試重付費（backlog-translate-budget-partial-progress）、attach 徽章時序，均不在本案。
- 測試結果：`go test ./...` 34 packages 全綠（exit 0）；`pnpm nx test web` 2653/2653 全過（非零退出為已立案的容器環境問題）；gofmt 觸及檔案全乾淨；新增測試 6（outcome ×3、verdict ×3）。

### File List
- apps/api/internal/services/translation_service.go（TranslationOutcome 型別、簽名 ×3、計數、Warn 欄位）
- apps/api/internal/services/transcription_service.go（translateSRT/translateAndPersist 簽名與 verdict 三態、[@contract-v2→v3] 註解、complete 訊息＋additive key、Info 欄位）
- apps/api/internal/services/translation_service_test.go（呼叫點更新＋新測試 ×3）
- apps/api/internal/services/transcription_generation_test.go（呼叫點更新＋新測試 ×3）
- apps/api/internal/services/transcription_service_test.go（呼叫點更新）
- apps/api/internal/services/transcription_translation_test.go（呼叫點更新）
- apps/api/internal/services/transcription_episode_test.go（呼叫點更新）
- apps/api/internal/services/scanner_service_test.go（pre-existing fix：root-skip guard）
- _bmad-output/implementation-artifacts/sprint-status.yaml（status 流轉＋2 筆 pre-existing 立案）
- _bmad-output/implementation-artifacts/9R-16-batch-generation-endpoint.md（AC drift reference — see Completion Notes）

## Change Log

| Date | Change |
|------|--------|
| 2026-08-19 | Task 1-4 實作：TranslationOutcome 貫穿（批次失敗＋逐 cue 缺漏計數）、verdict 三態（partial→untranslated+EN）、SSE partial 訊息＋english_kept_blocks additive key、單元/整合測試 ×6 |
| 2026-08-19 | [@contract-v2→v3] 9R-16 AC 12: what changed —— `found` 收窄為「完整翻譯成功」，partial 結果降級為 `untranslated`+EN 路徑（混合檔仍 place）；what breaks downstream —— 消費 verdict 的徽章/批次列舉會看到更多 `untranslated`（不再有謊報的 `found`），resume-eligible 集合擴大；9R-10a（in-dev）的 resume gate 相容（僅新增輸入案例），其餘 v2 ackers 全 done/frozen 無 stale-mark |
| 2026-08-19 | Pre-existing: scanner PermissionDenied root-skip fix；FILED generation-batch cancel SSE flake ＋ web vitest 容器非零退出 |
