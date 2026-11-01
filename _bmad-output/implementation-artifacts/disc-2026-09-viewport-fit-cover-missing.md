# Story disc-2026-09-viewport-fit-cover-missing：iPhone 上底部彈窗沒貼到底

Status: review

**Source:** sprint-status 同名待辦（2026-09-20 dsr-6f-1 建單，2026-11-01 調成 P2）。⚖️ Alexyu 2026-11-01「處理剩餘的4項代辦」。

## 問題

2026-11-01 Alexyu 用 iPhone 的 Edge（底層是 WebKit）看背景生成進度的底部彈窗（`ManageSubtitleDialogV2`），有時彈窗沒貼到螢幕底，彈窗和灰色遮罩下面都露出一條沒變暗的頁面。

- `index.html` 的 viewport 沒有 `viewport-fit=cover`，所以全站的 `env(safe-area-inset-*)` 都是 0，固定在底部的圖層也不會畫到真正的螢幕邊緣。
- iPhone 的工具列縮放時，版面視窗的底部會比螢幕底慢一步，釘在 `bottom-0` 的彈窗和 `inset-0` 的遮罩短暫停在比螢幕底高的地方。

## 做法

1. `index.html` 加 `viewport-fit=cover`。
2. 底部彈窗（`MOBILE_SHEET_CONTENT`，手機限定）和 `ui/Sheet` 的陰影多一層「裙襬」：往下 6rem、跟彈窗同色的實心陰影，工具列縮放露出來的那條會是彈窗的顏色。用陰影不用 `::after`，因為每個彈窗都有 `overflow-hidden`，陰影不會被自己裁掉。
3. `ui/Dialog` 和 `ui/Sheet` 的灰色遮罩往下多延伸 6rem（`-bottom-24`），在螢幕外，不影響畫面。
4. safe-area 開始生效後，底部分頁列是「總高 84px、safe-area 包在裡面」（border-box），主內容的下邊距和掃描進度條的位置原本寫成 84px＋safe-area，會多出一截空白，改成 84px。

## Acceptance Criteria

1. [x] viewport meta 有 `viewport-fit=cover`（測試讀 `index.html`）。
2. [x] 手機底部彈窗有同色裙襬，桌機置中的對話框不受影響（測試＋瀏覽器量到 computed box-shadow `rgb(240,231,211) 0 96px 0 0`）。
3. [x] 兩種遮罩往下延伸 96px，其他三邊不變（測試＋瀏覽器量到 `bottom: -96px`）。
4. [x] 主內容下邊距、掃描進度條位置不再重複加 safe-area。
5. [ ] Alexyu 用 iPhone Edge 實測：生成進度彈窗打開後上下捲動、工具列縮放，底下不再露出一條頁面。

## Dev Agent Record

- 改的檔：`apps/web/index.html`、`ui/mobileSheet.tsx`、`ui/Dialog.tsx`、`ui/Sheet.tsx`、`shell/AppShellV2.tsx`、`scanner/ScanProgress.tsx`、`ManageSubtitleDialogV2.tsx`（註解）。
- 測試：`mobileSheet.spec.tsx` 加裙襬與 viewport 兩條；`Dialog.spec.tsx` 加遮罩一條；`Sheet.spec.tsx` 逐字比對的 class 字串照新陰影更新。`nx run web:test`（4,993 個）、`web:typecheck`、`nx lint web`（0 errors）通過。
- 沒做：橫放手機時左右瀏海的 safe-area（全站沒有左右 padding），手機橫放很少用，等有人遇到再處理。

## Change Log

- 2026-11-01：建立＋實作。
