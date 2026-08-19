# Story 9R-10b: 入庫自動觸發 —— 常設同意下的「早上起來字幕就好了」

Status: ready-for-dev

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story.
     ⚠️ AC #1（常設同意政策）需 Alexyu 裁定後 dev-story 才可動工 —— 其餘 AC 不受影響可先讀。 -->

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

## AC #1 —— 🔴 常設同意政策（需 Alexyu 裁定；以下為 authoring 提案）

| 政策項 | 提案值 | 理由 |
|---|---|---|
| 預設狀態 | **OFF**（現行行為 byte-unchanged） | opt-in 是紅線的直接推論 |
| 開關粒度 | 每 library 一個開關（設定頁） | 動漫庫要、家庭錄影庫不要 |
| 免費層 | extract-only 自動觸發可獨立開啟（零 ASR 花費；LLM 翻譯仍計費故**不屬**免費層） | 「免費」的定義必須誠實：只有純抽取＋既有繁中軌 passthrough 是零花費 |
| 付費層 | 需再設 **常設預算上限**（每月 USD，沿用 sub-4-1 估價機制與 `ai.Governor` 預算池）方可開啟 | 數字先行，同 2026-08-07 裁定精神 |
| 觸頂行為 | 當月剩餘預算不足以跑該項估價 → 跳過並記 `pending-budget`，Activity 顯示「等待下月預算」；**絕不**部分執行 | 與 F14 budget_ceiling 語意一致 |
| 每次觸發上限 | 單次 scan 自動 enqueue 上限 N 項（提案 N=20，防首掃海嘯；超出項留待下次或手動批次） | 1,026 項事故的直接防呆 |
| 事後可見性 | 每次自動觸發在 Activity 產生一列（觸發源=auto、項數、估價、實際花費） | 無人值守 ≠ 不可稽核 |

## Acceptance Criteria（AC #2 起為實作，前提 = AC #1 裁定通過）

### AC #2 — 觸發線
- scan-complete 掛點以**組合**方式接入：包裹既有 `postScanEnrichment`（`main.go:431-449` —— setter 只有一個 slot，sub-1-6 AC #2 明文「must WRAP this body, never call the setter a second time」），enrichment 先行、自動觸發後行（metadata/語言路由依賴 enrichment 結果）。
- 觸發集合 = 本次 scan 新增/變更且 `subtitle_status ∈ {not_searched, not_found, untranslated}` 的項目 ∩ 開啟自動化的 library，經 AC #1 政策過濾（免費層/付費層/單次上限）。
- 排程掃描（`scan_scheduler.go`）與手動掃描走同一條線 —— 政策在 callback 內判定，不在觸發源判定。

### AC #3 — repo guard 交接（不是刪除，是升級）
- 依 guard 檔頭指示刪除 `TestScanMustNotAutoEnqueueSubtitleGeneration`，**同一 commit** 內以新 guard 取代：`TestAutoEnqueueRequiresStandingConsent` —— 斷言自動 enqueue 呼叫點被常設同意檢查包裹（無政策/OFF/無預算 → 零 enqueue），並在測試註解保留 2026-08-07 事故全文與本 story 的裁定鏈。PR 描述需引用裁定（guard 檔頭的第三個要求）。

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
