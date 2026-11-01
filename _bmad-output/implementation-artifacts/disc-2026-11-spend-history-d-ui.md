# Story disc-2026-11-spend-history-d-ui：花費歷程頁（前端）

Status: backlog（等 -b API 合併）

**Source:** 傘狀 `disc-2026-11-spend-history`。照 `disc-2026-11-spend-history-c-design.md` 的 Sally 設計決定（**沒有 .pen 稿**，⚖️ Alexyu 2026-11-01：Pencil 要收費，先用現有 design system 做）實作新頁和活動頁入口，讀 -b 的 `GET /api/v1/subtitles/runs`。

## Story

身為想知道錢花在哪裡的人，我希望打開一頁就能看到每天花了多少、每一次跑了哪支片、用了哪些模型、實際和預估差多少。

## Acceptance Criteria

1. `SpendCard` 加「看明細 →」連到 `/activity/spend`。
2. `/activity/spend`：標題列（返回、月份切換、不能切到未來）、摘要列（總額、次數、預估準度）、每日堆疊長條圖（inline SVG，四類顏色＋圖例，桌機 hover／手機點擊顯示當日明細）、逐筆列表（桌機表格、手機卡片、處理方式 chip 篩選、「載入更多」）——全部照 -c 的設計決定。
3. 處理方式的字照 -c 的對照表；「語音辨識」不寫成「聽寫」。
4. 誠實規則：沒記錄的寫「沒記錄」，不顯示 US$0.00；實際超過預估 50% 加 ⚠；失敗／中途停止照列照算。
5. 載入／空／錯誤三種狀態，錯誤可重試。
6. 測試：元件 spec（含沒記錄、超出預估、失敗列、空狀態、手機版面）、gallery fixture（不要自己產 `-linux` 基準，CI 會開 bootstrap PR）、e2e 一條（從活動頁點「看明細」到看到列表）。
7. 完成前用瀏覽器實際打開頁面（桌機＋手機寬度）截圖，跟 K5 截圖並排看是同一個家族（取代原本對照 .pen 稿的 UX 驗證）。

## Change Log

- 2026-11-01：建立（Bob）。
- 2026-11-01：-c 改成不出稿，AC 依 Sally 設計決定補齊。
