# Story disc-2026-11-spend-chart-current-month-one-bar：月初的花費圖表不再只有一根撐滿的長條

Status: done

**Source:** sprint-status 同名待辦（P3，PR #946 建單）。⚖️ Alexyu 2026-11-01「處理剩餘的4項代辦」。

## 問題

1. 本月只到第 1 天時，每日長條圖只有一根，撐滿整張圖。後端的 series 只產到今天（刻意不給未來日），`SpendDailyChart.tsx` 照 series 的長度切寬度。
2. 同一天摘要寫 $2.44、圖表「最高」寫 $2.43：圖表把每一類各自四捨五入後再相加，摘要是全部加完才四捨五入。

## 做法

1. 圖表依整個月的天數留位置（`daysInMonth(month)`，純算術、不讀時鐘）。還沒到的日子空白：不畫長條、不畫底線、不能滑過，跟「那天沒花錢」的底線區分開來。X 軸照常標 1、5、10……月底。
2. 後端每天先把總額四捨五入一次，再用「最大餘數法」分回四類，四類加起來剛好等於當天總額。單日的月份，圖表的那一天會等於摘要。

## Acceptance Criteria

1. [x] 11 月 1 日看 11 月：圖表有 30 個位置，只有第 1 天有長條，X 軸到 30。
2. [x] 已經過完的月份照舊（31 天都有位置、都有資料）。
3. [x] 同一天的四類金額加起來等於當天總額；只有一天的月份，當天總額等於摘要總額。
4. [x] 測試：`allocateCents`（11/1 那個差一分錢的例子、整數分、全 0、兩分剩餘分給最大的兩個）、單日月份圖表＝摘要、前端 31 個位置但只有 2 天能滑、`daysInMonth`。

## Dev Agent Record

- BE：`subtitle_run_history_service.go` 新增 `allocateCents`，`runHistorySeries` 改用它。
- FE：`SpendDailyChart` 多一個 `month` prop，寬度與 X 軸用整個月的天數，長條依日期放到對應位置。`daysInMonth` 不用 `Date`（Rule 23 lint 會擋讀時鐘的元件，這只是曆法計算）。
- 驗證：`go test ./internal/services/`、`go vet`；`nx run web:test`（4,995 個）、`web:typecheck`、`nx lint web`（0 errors）。本機用正式機資料副本截 iPhone 14 寬度：30 個位置、只有 1 號有長條，摘要 $3.36＝圖表最高 $3.36。

## Change Log

- 2026-11-01：建立＋實作。
