# Story disc-2026-11-spend-history-c-design：花費歷程頁設計稿 K7

Status: ready-for-dev

**Source:** 傘狀 `disc-2026-11-spend-history`。Sally 出 Pen Inline Agent 提示詞（格式照 `pen-prompts-solo-progress-cancel.md`），Alexyu 在 Pencil 跑，Sally 用 MCP 唯讀 review，再跑 `scripts/export-pen-screenshots.py` 出截圖。不做：程式。

## 要畫的

| 畫面 | 內容 |
|---|---|
| **K7-D**（桌機）、**K7-M**（手機） | 新頁「AI 花費明細」：上方圖表（每日／每週花費，依處理方式分顏色：片內字幕、片旁英文字幕、聽寫再翻譯）；下方逐筆列表（日期時間、片名、處理方式、用了哪些模型、實際花費、當初預估、狀態） |
| **K7s**（規格屏） | 空狀態、載入、錯誤；「沒記錄」的顯示（例：預估欄寫「沒記錄」不是 US$0.00）；失敗／中途停止的列怎麼標；實際比預估高很多時要不要提醒 |
| **K5-D／K5-M 改** | 「本月 AI 花費」區塊加「看明細」入口 |

## 要 Sally 決定、寫在規格屏上的

- 圖表用堆疊長條（每日）還是折線（累積）。
- 手機上列表要收成卡片還是可橫捲的表。
- 篩選：日期範圍、處理方式；要不要依影片搜尋。

## Acceptance Criteria

1. K7-D、K7-M、K7s 三張存進 `ux-design.pen`，K5 兩張加入口。
2. `SCREENS` 加新節點、截圖存 `_bmad-output/screenshots/flow-k-activity-v2/`，只提交真的有改的 PNG。
3. 欄位名稱和 -b 的回應一一對得上（畫面上有的欄位，API 都有）。

## Change Log

- 2026-11-01：建立（Bob）。
