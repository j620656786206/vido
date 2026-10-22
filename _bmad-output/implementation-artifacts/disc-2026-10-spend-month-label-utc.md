# disc-2026-10-spend-month-label-utc：「本月 AI 花費」標對月份，10 月不會再寫「9 月」

Status: done

**Source:** 2026-10-22 `testenv-seed-2026-10` 在本機 :8090 看活動頁時發現（Rule 24 lane ③ 立案）。Alexyu 排序：三個 bug 裡第三個修。

## 為什麼要做

活動頁「本月 AI 花費」右上角會標月份。今天是 10 月 22 日，畫面卻寫「9 月」。

原因：後端把「這個月從什麼時候開始」換成 UTC 時間才回傳。台灣的 10 月 1 日 0 點，等於 UTC 的 9 月 30 日 16 點，所以回傳的是 `2026-09-30T16:00:00Z`。前端直接讀這串字裡的月份，就讀到「09」。

只要伺服器的時區不是 UTC（例如 NAS 容器設了 `TZ=Asia/Taipei`），每個月都會標成上個月。金額本身是對的，只有月份標錯；但使用者會以為看到的是上個月的帳。

## Acceptance Criteria

1. `GET /api/v1/subtitles/spend?period=month` 回傳的 `from`、`to` 用**伺服器時區**寫（例如台灣：`2026-10-01T00:00:00+08:00`），代表的時間點跟現在一樣。
2. 查資料用的區間不變（照舊用 UTC 比對資料庫）。
3. 活動頁在台灣時區的伺服器上，10 月顯示「10 月」；UTC 伺服器照舊。
4. 測試：服務層斷言 `from` 的時區是伺服器時區、JSON 字串的月份是當月、時間點跟查詢用的一樣；月初半小時仍算這個月（既有斷言保留）。

## 不要做的事

- 不改前端（`spendMonthLabel` 的測試本來就假設 `+08:00` 的字串）。
- 不加新欄位。

## Tasks

- [x] `SubtitleSpendService.MonthSummary`：`out.From`／`out.To` 用伺服器時區（AC 1、2）
- [x] 改既有測試、補 JSON 斷言（AC 4）
- [x] :8090 本機看活動頁月份（AC 3）

## Dev Notes

- `apps/api/internal/services/subtitle_spend_service.go`：`monthWindow()` 回傳 `from.UTC(), to.UTC()`，`MonthSummary` 直接把它塞進 `out.From/To`。
- 既有測試 `TestSpendSummary_MonthWindowIsTheServerZoneCalendarMonth` 斷言 `got.From` 等於 UTC 的值——等於把這個 bug 寫成了規格，要改。
- 前端 `apps/web/src/components/activity/SpendSection.tsx` `spendMonthLabel` 讀 `from.slice(5, 7)`。

## Dev Agent Record

### Agent Model Used

Claude Opus 5.5（claude-opus-5-5），2026-10-22。

### Completion Notes List

- `MonthSummary` 查資料照舊用 UTC 區間；回傳前把 `from`／`to` 換成伺服器時區（`s.now().Location()`），時間點不變。
- 改既有測試：不再斷言 `got.From` 是 UTC，改成「時間點相同、時區是伺服器的、JSON 是 `2026-10-01T00:00:00+08:00`」。把服務改回舊版時這條會紅。
- 本機 :8090（伺服器時區 +08:00）：API 回 `from: 2026-10-01T00:00:00+08:00`，活動頁顯示「10 月」，金額不變（翻譯 $0.24）。
- 驗證：services／handlers 的 Spend 測試、staticcheck 綠。
- 正式機容器的時區仍沒查到（這次連不到 NAS）；不論是不是 UTC，修完都會標對。

### File List

- `apps/api/internal/services/subtitle_spend_service.go`、`subtitle_spend_service_test.go`
- 本檔、`sprint-status.yaml`（也把上一張 `disc-2026-10-reparse-dialog-misses-instant-pass` 標 done，PR #901）
