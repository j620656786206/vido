# Story 9R-10b: 入庫自動觸發 —— 常設同意下的「早上起來字幕就好了」

Status: ready-for-dev

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story.
     ⚖️ AC #1 已裁定 2026-08-19（Alexyu）：「花錢須同意」—— 付費動作不得自動執行。story 可直接 dev。 -->

**Epic:** 9R 尾款（Rule-24 ③ from 9R-10, 2026-07-05）· **Risk: 🔴 HIGH —— 花費同意紅線**（2026-08-07 裁定 + `cost_consent_test.go` repo guard）
**Source:** sprint-status `9R-10b-on-add-autotrigger`（backlog, needs product call）；2026-08-19 What'Sub 對抗重審列為無悔四項之 N4 —— 同時是 P（每日體驗）、O（最小可愛 demo）、C（購買觸發點）三情境的共同核心。
**機器盤點：全部就緒，只缺最後一條線** —— 管線（9R-10）、批次協奏（9R-16）、成本預估＋同意＋預算（sub-4-1/4-2/4-3, M2.5）、詞彙庫閉環（sub-5-5）、scan-complete 掛點（`scanner_service.go:82 SetOnScanComplete`）皆已出貨。

---

## Story

As a NAS owner who adds new episodes at night,
I want items that arrive missing zh-TW subtitles to be generated automatically under a standing budget I approved once,
so that by morning the library is subtitled and truthfully badged —— without me pressing anything, and without the system ever spending money I didn't consent to.

---

## Context —— 為什麼這是 HIGH risk，以及紅線的原文

**2026-08-07 事故與裁定**（`apps/api/internal/cost_consent_test.go` 檔頭原文）：pipeline 模式首次上線時 scan-complete 曾直接掛整庫 sweep，一次掃描 enqueue 1,026 項、約 2/3 走付費 ASR、估算 ~US$200，使用者全程沒看到數字。裁定：「**scanning updates metadata and nothing else；paid generation is chosen explicitly on a screen that shows the estimate first**」。repo guard `TestScanMustNotAutoEnqueueSubtitleGeneration` 令 `.EnqueueMissing(` 在生產程式碼零呼叫者，並明文寫下解除條件：「Re-enabling it deliberately is still possible —— but **it must come with a consent surface**, and whoever does it has to delete this test and explain why in the PR.」

本 story 就是那個 consent surface。自動觸發**不是**繞過同意 —— 是把「每次同意」升級為「**常設同意**（standing consent）」：使用者一次性設定政策＋預算上限，此後系統只在政策範圍內花錢，且每一筆都可追溯到那次明示設定。

## AC #1 —— ⚖️ 花錢須同意（Alexyu 裁定 2026-08-19，取代原 authoring 提案）

裁定原文：「**9R-10b 花錢須同意**」。落地語意（本檔記錄的解讀 —— 若 Alexyu 意指「常設月預算＝一次性同意」的較寬模式，修此節即可，架構不變）：

| 動作類別 | 自動觸發行為 |
|---|---|
| **零花費**：繁中內嵌軌 passthrough、簡中軌 OpenCC s2twp、軌道探測/語言路由、候選分析＋估價 | ✅ **可全自動執行**（per-library 開關，預設 OFF） |
| **付費**：LLM 翻譯、ASR 轉錄 | 🔴 **自動只到「入待同意清單」為止** —— scan-complete 自動分析＋估價＋掛入既有同意流程（F14–F20）的候選清單，經 Activity/badge 通知；**執行永遠等使用者在估價畫面按下確認**。零自動扣款 |
| 常設預算上限（standing budget）模式 | ❌ 本次不做 —— 若未來想要「設一次月上限、之後全自動」，另立 story 重新裁定 |

淨效果：**「免費的自動做完，要花錢的排好隊等你一鍵同意」** —— 早上起來：繁中/簡中軌的項目已完工、需要翻譯/ASR 的項目帶著金額在同意清單裡等一次點擊。

## Acceptance Criteria（AC #2 起為實作，前提 = AC #1 裁定通過）

### AC #2 — 觸發線
- scan-complete 掛點以**組合**方式接入：包裹既有 `postScanEnrichment`（`main.go:431-449` —— setter 只有一個 slot，sub-1-6 AC #2 明文「must WRAP this body, never call the setter a second time」），enrichment 先行、自動觸發後行（metadata/語言路由依賴 enrichment 結果）。
- 觸發集合 = 本次 scan 新增/變更且 `subtitle_status ∈ {not_searched, not_found, untranslated}` 的項目 ∩ 開啟自動化的 library。零花費項目直接執行；付費項目**只做分析＋估價＋入待同意清單**（AC #1 裁定）。單次自動處理上限 N=20 防首掃海嘯（超出留待下次或手動批次）。
- 排程掃描（`scan_scheduler.go`）與手動掃描走同一條線 —— 政策在 callback 內判定，不在觸發源判定。

### AC #3 — repo guard **保留並加固**（裁定後不再需要交接）
- 「花錢須同意」裁定下付費 sweep 依然零自動呼叫 → `TestScanMustNotAutoEnqueueSubtitleGeneration` **原樣保留**（其守護的不變量未變）。
- 新增互補 guard：自動觸發路徑僅允許零花費操作＋分析/估價＋入待同意清單 —— 以測試斷言 auto 路徑沒有任何 `TranslateWithGlossaryHarvest`/ASR/`TranslateTrack` 直呼點；測試註解引用本裁定（2026-08-19「花錢須同意」）與 2026-08-07 事故鏈。

### AC #4 — 誠實狀態
- 自動觸發的項目走既有 SSE/狀態機（`subtitle_status` 生命週期、Activity 生成列）；徽章語意零新增。
- 與 bugfix-j 串接：partial-failure verdict（`untranslated`）在自動路徑同樣生效 —— 自動化**不得**放大不誠實。

### AC #5 — 測試
- 政策矩陣 table-driven：OFF/免費層/付費層×預算足/不足/觸頂×單次上限內/外 → enqueue 集合斷言。
- 組合 callback 測試：enrichment 先行且 byte-unchanged（sub-1-6 AC #2 回歸釘）。
- 新 guard 測試（AC #3）紅→綠演示：裸 enqueue 呼叫 = 紅。
- 既有 26 Claude-touching 測試＋cost-consent 相關測試（除被交接者外）全綠。

### AC #6 — 範圍紅線
- 不動每次同意流程（F14-F20）—— 手動批次照舊。
- 不做下載完成觸發（那是 Epic 13 story 13-5 的職權；本案只接 scan/import）。
- 不做 per-item 重試自動化（sub-5-3 已交付手動重試）。

## Tasks / Subtasks

- [ ] Task 0: 🔴 AC #1 政策提案送 Alexyu 裁定（在 dev-story session 開場）
- [ ] Task 1: 設定模型＋設定頁（library 開關、預算上限、免費/付費層）
- [ ] Task 2: AC #2 組合 callback＋政策過濾器
- [ ] Task 3: AC #3 guard 交接
- [ ] Task 4: AC #4 狀態串接驗證＋AC #5 測試矩陣
