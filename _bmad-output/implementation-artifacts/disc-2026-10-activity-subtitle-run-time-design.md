# Disc：活動記錄每一筆字幕生成都寫出「花了多久、花了多少錢」（設計稿）

Status: ready-for-pen

**Source:** 2026-10-19 Alexyu：「目前生成出來的字幕只有在活動會看到本月 AI 花費，但是字幕生成花了多少時間有地方紀錄嗎？」

查證：
- 每次生成的開始與結束時間都有記：`subtitle_runs.started_at`／`completed_at`，同一列也有 `spent_usd`、`cue_count`、`route`、`status`。正式機最近 6 次翻譯花 2.0～5.1 分鐘。
- **但沒有任何畫面顯示。** 活動頁「活動記錄」目前只放解析完成／失敗（`activity_service.go:173-180` 的 `RecentEvent` 註解寫「v1 sources parse jobs only」；前端 `ActivityHub.tsx` 的 `RecentSectionView` 標題只有 解析完成／解析失敗）。字幕生成根本沒進活動記錄。
- 設計稿 K1-D-v2（`kMeWS`）的活動記錄早就畫了一列字幕，但內容還是 v1 的「字幕已下載／蒼鷺 · 繁體中文」——Vido 現在是生成字幕，不是下載。

⚖️ **2026-10-19 Alexyu 裁定：紀錄放在活動頁每一筆紀錄的後面。**

## Sally 的設計裁定

1. **字幕生成進活動記錄。** 每次生成（成功或失敗）一列，跟解析事件按時間混排。
2. **列的內容：**
   - 標題：成功「字幕完成」；失敗「字幕沒做成」。
   - 說明：「片名 S01E02 · 翻譯 568 句」（電影不寫集數；ASR 路徑寫「聽寫＋翻譯 N 句」）；失敗時改寫原因，用管理字幕視窗同一句人話。
   - **右側兩行**：上行「3 分鐘 · US$0.27」（等寬、`$text-secondary`），下行「2 分鐘前」（`$text-muted`、Label 字級）。花多久和花多少錢放在同一行，是因為使用者看這兩個數字的理由相同：值不值得。
3. **時間寫法：** 不到 1 分鐘寫「N 秒」；不到 1 小時寫「N 分鐘」（四捨五入）；1 小時以上寫「1 小時 5 分」。
4. **錢的寫法：** 沿用 SpendSection 的 `usd()`；**0 元寫「沒花錢」**（跟管理字幕視窗的「這次沒有花到錢」同一個意思）。沒有完成時間的不顯示花多久。
5. **只有字幕列有右側兩行**；掃描、解析列維持一行相對時間。
6. ~~手機一樣是右側兩行~~ → **Sally 複審改判（2026-10-19）：手機右側只留相對時間，「花多久 · 花多少錢」移到說明下面當第三行。** 第一輪照右側兩行畫，手機上右欄佔 129px，標題與說明被擠到 131px 寬，失敗原因要折成四行。

## 範圍（只動 `.pen`）

1. K1-D-v2（`kMeWS`）活動記錄：把舊的字幕列改成新內容，再加一列失敗的例子。
2. K1-M-v2（`QIwY1`）同步改。這一段在手機稿裡本來就被裁切在畫面外（`sec-活動記錄` fully clipped），所以截圖看不到，但節點內容要一致。
3. 規格說明 `spec-note-k-run-time`，獨立放，不塞進畫面。

## 不做

- 程式實作：另開 `disc-2026-10-activity-subtitle-run-time`（blocked-by 本單；前後端都要動，Bob 依 Story Splitting 決定拆不拆）。

## Acceptance Criteria

1. K1-D-v2 的活動記錄有：字幕完成（《末日光明》S01E02）、字幕沒做成（S01E01）、掃描完成、解析失敗。前兩列右側是兩行。
2. K1-M-v2 的對應列內容一致。
3. Sally 用 MCP 唯讀複審：文案逐字對、`ctx.problems` 沒有新增項目（手機 `sec-活動記錄` 原本就被裁切，不算）、全部用變數。
4. 截圖：只 stage `k1-d.png`（手機截圖看不到這段，若位元相同就不 stage）；`check-design-refs.py` 與 token 快照通過。

## Inline Agent 提示詞（Sally，2026-10-19）

把下面整段貼給 Pencil 的 Inline AI Agent。節點 id 都是 2026-10-19 用 MCP 讀出來的。

```text
在 ux-design.pen 改「活動中心」的活動記錄：字幕生成那幾列要寫出花了多久、花了多少錢。全部用既有變數，不寫死顏色、字級、間距（0 用 $Space/none）。所有新節點都要取名。不要改元件 fF8nX（Component/ActivityRow-v2）本身，只改實例。

【A. 桌面 K1-D-v2（節點 kMeWS）的 sec-活動記錄（qWzds）】
1. 實例 LndN2（row-字幕完成）：
   - 改名 row-字幕完成
   - rowTitle（xWxRJ）：「字幕完成」
   - rowSubtitle（ko2Lf）：「末日光明 S01E02 · 翻譯 568 句」
   - 把 rowRight（tLXCT）整個換成 frame「rowRight」：vertical、gap $Space/2xs、alignItems end（靠右）。裡面兩個 text：
     · 「rowRightMeta」：「3 分鐘 · US$0.27」，$Type/Family/Mono、$Type/Body/Size、$Type/Body/Line、fill $text-secondary
     · 「rowRightTime」：「2 分鐘前」，$Type/Family/Primary、$Type/Label/Size、$Type/Label/Line、fill $text-muted
   - 圖示維持 circle-check、iconWrap 維持成功色（跟原本一樣）。
2. 複製 LndN2，放在它正下方（sec-活動記錄 裡的第 2 列），改名 row-字幕沒做成：
   - iconWrap（EtPLt）fill $error-tint；rowIcon（B5edWI）icon triangle-alert、fill $error（跟 hJhqs 同一組）
   - rowTitle：「字幕沒做成」
   - rowSubtitle：「末日光明 S01E01 · 影片所在的資料夾寫不進字幕」
   - rowRightMeta：「4 秒 · 沒花錢」
   - rowRightTime：「3 小時前」
3. Wq46Q（掃描完成）與 hJhqs（解析失敗）不動。
4. sec-活動記錄 多了一列會變高：如果 K1-D-v2 的高度（1360）裝不下，把 K1-D-v2 的高度加到剛好裝得下（底部留跟原本一樣的空白），不要讓任何東西被裁切。

【B. 手機 K1-M-v2（節點 QIwY1）的 sec-活動記錄（aZeU5）】
1. 實例 JLTBV：照 A-1 改（名稱、標題、說明、右側兩行）。說明在 358 寬裡可以換行。
2. 複製 JLTBV 放在它下面，照 A-2 改成 row-字幕沒做成。
3. 這一段本來就在手機畫面的可視範圍外，不用移動它，也不要改 K1-M-v2 的高度。

【C. 規格說明 spec-note-k-run-time】
在群組「K · 桌面 Desktop」裡、最右邊那張畫面的右邊 100px，頂端對齊 K1-D-v2，放一個 text 節點 spec-note-k-run-time，寬 300，樣式照 Flow A 的 spec-note-dsr-1b-b（GMxQV），不得跟任何東西重疊。文字：
「K1（活動記錄，2026-10-19）：每次字幕生成（成功或失敗）都是一列，跟解析事件按時間混排。標題：成功『字幕完成』、失敗『字幕沒做成』。說明：『片名 S01E02 · 翻譯 N 句』（電影不寫集數；有用語音辨識寫『聽寫＋翻譯 N 句』），失敗時寫原因，用管理字幕視窗同一句。右側兩行：上行『花多久 · 花多少錢』（等寬），下行相對時間。花多久：不到 1 分鐘寫『N 秒』，不到 1 小時寫『N 分鐘』，以上寫『1 小時 5 分』；沒有完成時間就不寫。花多少錢：US$0.27；0 元寫『沒花錢』。只有字幕列有右側兩行，掃描、解析列維持一行時間。手機一樣。」

完成後不要自己匯出截圖。
```

## 第二輪修正提示詞（Sally 複審後，2026-10-19）

第一輪複審發現：① 兩個失敗列（桌機 `EVrLx`、手機 `akt4i`）右側文字沒改到，還是「3 分鐘 · US$0.27／2 分鐘前」；② 手機右側兩行把說明擠到 131px 寬（見裁定 6 改判）。

```text
在 ux-design.pen 修「活動中心」活動記錄的字幕列。全部用既有變數，不寫死數值。不要改元件 fF8nX 本身。

【1. 桌面失敗列 EVrLx（row-字幕沒做成）】
它的 rowRight（SWSZr）裡兩個 text：rowRightMeta 改成「4 秒 · 沒花錢」，rowRightTime 改成「3 小時前」。

【2. 手機兩列 JLTBV（mrow-字幕完成）與 akt4i（mrow-字幕沒做成）】
每一列都做：
- 刪掉 rowRight 裡的 rowRightMeta，只留 rowRightTime（相對時間）。rowRight 維持靠右。
- 在 textCol 裡、rowSubtitle 下面加 text「rowMeta」：$Type/Family/Mono、$Type/Label/Size、$Type/Label/Line、fill $text-secondary、width fill_container。
- JLTBV：rowMeta「3 分鐘 · US$0.27」，rowRightTime「2 分鐘前」。
- akt4i：rowMeta「4 秒 · 沒花錢」，rowRightTime「3 小時前」。

【3. 規格說明 XYUmV（spec-note-k-run-time）】
把最後一句「手機一樣。」換成：「手機：右側只留相對時間，『花多久 · 花多少錢』移到說明下面一行（右側兩行會把說明擠到太窄）。」

改完按 ⌘S 存檔。不要匯出截圖。
```

## Dev Agent Record

（待 Alexyu 跑完 inline agent、Sally 複審後填寫。）
