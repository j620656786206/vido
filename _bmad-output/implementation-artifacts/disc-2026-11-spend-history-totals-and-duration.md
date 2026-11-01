# Story disc-2026-11-spend-history-totals-and-duration：花費明細改成「預估總額 vs 實際花費」，每筆列出花了多久

Status: done

**Source:** ⚖️ Alexyu 2026-11-01：「不要說預估準度，應該說總預估多少錢、實際花費多少錢。另外明細應該也要列出每項跑了多久完成。」

## 問題

1. 摘要列寫「預估準度：實際是預估的 N%」，Alexyu 要看的是兩個金額本身，不是比例。
2. 逐筆紀錄看不到每一次跑了多久。

## 做法

- API `summary` 新增 `estimate_usd`（這個月所有有記預估的執行，預估加總；沒有任何一筆時是 null，不是 0）和 `estimated_runs`（幾筆有預估）。失敗、免費的也算進去，因為那是按下去之前看到的報價。`comparable_*` 保留，畫面不再用。
- 摘要列：「這個月實際花費 $X · N 次」＋「這個月預估 $Y」。附註：有幾次沒記錄金額、幾次沒有預估，各自沒算進哪一個總額。
- 每筆的「花了多久」＝完成時間 − 開始時間：「不到 1 秒」「45 秒」「8 分 3 秒」「1 小時 5 分」；沒有完成時間寫「沒記錄」，不寫 0 秒。電腦版多一欄「花了多久」；手機放在影片名下面那行：「時間 · 處理方式 · 花了 X」。
- API 的時間有 9 位小數（Go 的奈秒），iPhone 的瀏覽器不保證讀得懂，前端先截到毫秒再算。

## Acceptance Criteria

1. [x] 摘要不再出現「準度」；顯示實際花費與預估兩個金額。沒有預估紀錄時顯示「沒有預估紀錄」，不顯示 $0.00。
2. [x] 附註說明沒記錄金額／沒有預估的筆數。
3. [x] 電腦版有「花了多久」欄；手機版影片名下面那行有「花了 X」；沒有完成時間寫「沒記錄」。
4. [x] 測試：後端 `estimate_usd`／`estimated_runs`（含失敗與免費的、沒有時為 null）；前端摘要兩種狀態、花了多久兩種狀態、`formatRunDuration`（秒／分秒／時分／不到 1 秒／缺值／倒退／奈秒）。

## Dev Agent Record

- BE：`subtitle_run_history_service.go` `RunHistorySummary` 加兩個欄位、`summarizeRunHistory` 加總；handler 的 swagger 說明更新。
- FE：`subtitleRunHistoryService.ts` 型別；`SpendHistoryPage.tsx` 摘要列重寫；`spendHistoryCopy.ts` 刪掉 `estimateAccuracyPercent`；新增 `utils/formatRunDuration.ts`；`SpendRunList.tsx` 加欄位與手機小字；gallery fixture、e2e mock 補欄位。
- 驗證：`go test`、`go vet`、`staticcheck`；`nx run web:test`（5,001 個）、`web:typecheck`、`nx lint web`（0 errors）。本機用正式機資料副本：11 月「實際花費 $3.36 · 10 次、預估 $3.27」；天蠍 E3 花了 7 分 59 秒、E2 10 分 5 秒、基地 10～13 秒、鋼鐵之淚（快取）0 秒 → 改成「不到 1 秒」。
- 截圖基準：手機與桌機的花費明細畫面改了，刪掉這兩張舊的 `-linux` 基準，由 CI 在這個分支重新產生。

## Change Log

- 2026-11-01：建立＋實作。
