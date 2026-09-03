# Story 7.6: 成本收據 —— 單次、本月、以及「省了多少」（全端，BE 3 / FE 3）

Status: ready-for-dev

## Story

As a BYOK NAS owner,
I want to see what each run cost, what this month cost, and what the router and cache saved me,
so that I trust the number — and see the moat working for me.

## Context

party-mode Sally：「讓使用者親眼看到護城河在替他省錢，他才會留下來。」現況：`subtitle_runs` 有 `cost_usd`／`spend` 索引（migration 032）、`LatestWithSpend`（`subtitle_run_repository.go:300`）、首頁 readout 只在有 live batch 時顯示 spend 三聯（`HomeReadoutBand.tsx:18-66`）。沒有月彙總、沒有「略過省下」。

## Acceptance Criteria

1. **後端彙總端點。** `GET /api/v1/subtitles/spend?period=month` `[@contract-v1]` 回 `{period, translated_runs, translated_usd, asr_runs, asr_usd, skipped_deliver_count, skipped_saved_usd_estimate, cache_hit_cues, cache_saved_usd_estimate, by_model:[{model_id, runs, usd}]}`。`skipped_saved_usd_estimate` = deliver 路線集數 × 該集依 sub-6-8a 估價公式的翻譯費（明示為估算）；`cache_saved` = segment cache 命中 cue 數 × 平均每 cue 費（run 記錄的 usage 反推）。需要 `subtitle_runs` additive 欄 `cache_hit_cues`（sub-1-5b 若已記則沿用）。
2. **單次收據。** run 完成事件（F8 terminal）與 `subtitle_runs` 詳情回 `{cost_usd, model_id, cue_count, cache_hit_cues, elapsed}`；SSE terminal payload additive 加 `cost_usd`／`model_id`。
3. **收據 UI。** F8 terminal 卡片：「本次 $0.53 · claude-sonnet-5 · 844 句 · cache 命中 12%」；活動頁（Activity hub）新增「本月 AI 花費」區塊：翻譯／語音辨識兩列 + 「略過 N 集省下約 $X」+「cache 省下約 $Y」，all `≈` 標示估算；依模型的小表。
4. **首頁 readout。** `HomeReadoutBand` 的 spend 三聯改讀月彙總（有 live batch 時仍顯示 live），absent ≠ $0 規則不變。
5. **誠實規則。** 不追蹤的東西不報數字（PRODUCT.md 原則 1）：沒有 cache 統計的舊 run 顯示「—」不顯示 0；估算一律 `≈`。
6. **測試。** repo 月彙總 SQL（真 sqlite，含跨月邊界）、估算公式、端點 shape、SSE additive、FE 三處元件 spec + gallery fixtures。

## Tasks / Subtasks

- [ ] **Task 1 — 彙總查詢 + 端點（AC: #1）**
- [ ] **Task 2 — run 收據欄位 + SSE additive（AC: #2）**
- [ ] **Task 3 — 後端測試（AC: #6）**
- [ ] **Task 4 — F8 收據卡 + 活動頁月區塊（AC: #3, #5）**
- [ ] **Task 5 — 首頁 readout 改讀月彙總（AC: #4）**
- [ ] **Task 6 — FE 測試 + fixtures（AC: #6）**

（後端 3 task、前端 3 task —— 剛好在門檻內，不拆；若 authoring 發現任一側超過 3 即拆 a/b。）

## Dev Notes

- 設計：活動頁 v2（flow-k-activity-v2）已有版位語彙；Sally 需為「本月 AI 花費」區塊與 F8 收據卡各出一張，走 `.pen` 流程。
- 與 sub-6-8a 的估價公式同源（`PricingFor`）——不得另寫一份費率。

### Time-dependent visual coverage

- **YES**：「本月」以月為界會讀 `new Date()` → 依 Rule 23 用 `Clock-injected`（`now` prop，預設 `Date.now()`）；fixtures：`month-start`／`month-end` 兩態。

### References

- party-mode 2026-09-03（Sally 收據）；`subtitle_run_repository.go:300`；`HomeReadoutBand.tsx`；migration 032

## Dev Agent Record

### Agent Model Used

### Completion Notes List

### Discovery Triage

- （dev 填）

### File List
