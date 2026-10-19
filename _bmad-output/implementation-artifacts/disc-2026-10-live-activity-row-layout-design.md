# disc-2026-10-live-activity-row-layout-design：即時活動失敗列把原因放到第二行，片名先完整顯示

- 狀態：done（2026-10-19 Alexyu 跑完 inline agent 並存檔；Sally 用 MCP 唯讀複審通過：④／⑤／⑤b 都是 evt-line＋evt-reason-row 兩行、原因左緣與階段字對齊（x=38）、新增 ④ 手機示範 T6WrrI（358 寬）、F11-D-v2 即時活動多一列 evt-fail（mHyTb）、F11-M-v2 佇列失敗列改「和平使者 S01E01」＋最長原因、三張稿沒有 clipped 節點）
- 來源：2026-10-19 PR #846（disc-2026-10-live-activity-failure-reasons-b）dev 回報。即時活動改講具體原因之後，失敗列「失敗 · 片名 · 原因」擠在同一行。最長的原因「影片所在的資料夾寫不進字幕」在 390 寬時佔 182px，片名只剩 34px，變成「和…」「魔…」「花…」（「花月殺手」只剩一個字）；1440 寬的 400px 側欄也把「和平使者 S01E01」截成「和平使者 …」，看不到集數。失敗了卻認不出是哪部片。⚖️ Alexyu 裁定 1A：請 Sally 出版面方案。
- 一併涵蓋：Bob 正在開的單——左邊「生成佇列」的失敗項目也改講具體原因（現在寫「生成失敗」）。
- 範圍：這張只改 `ux-design.pen`（三張稿），不動程式。程式另開實作單（見文末「實作單要做的事」）。
- 備註：dev 回報寫的是「F11-SPEC-LOG ⑥」，稿上失敗列其實是 **④**（⑥ 是「批次完成」）。-b 的 sprint-status 條目也寫成「⑤／⑤b／⑥」，以本檔為準。

## 現況量測（Pencil MCP 唯讀＋程式＋截圖）

| 地方 | 節點 | 現況 |
| --- | --- | --- |
| 規格稿 F11-SPEC-LOG | `tT3JX`（880 寬，`F · 規格 Spec` 群組） | 十三種列的示範。每列 `row-box` 400 寬，裡面一個水平 `evt`：圖示 16 · 階段字 · `·` · 項目 · `·` · 原因 · 撐開 · 尾端。④ 失敗 `dNQa0`→`stf9c`、⑤ 已停止 `xPTom`→`cu3CY`、⑤b 帳號沒錢 `xkmMv`→`c5WP3` 都是「片名＋原因」同一行。沒有手機寬度的示範。 |
| 桌機執行中 F11-D-v2 | `l8FsB`，側欄 `DUvwI`（400 寬），清單 `dPQ9b` | 四列：完成×2、提取音訊（已過）、轉錄中。**沒有失敗列**，但同一張左邊佇列有一列失敗（`f14sNh`「特技玩家」）。 |
| 桌機預算上限 F12-D-v2 | `iH98f`，側欄 `LTW74` | 五列，沒有失敗列，不用改。 |
| 手機 F11-M-v2 | `PXB0z` | 即時活動是收合狀態（`rMD73`），沒有畫展開的清單。佇列有一列失敗 `Ah8fN`（358 寬）。 |
| 佇列元件 | `Component/GenQueueRow-v2`（`aw4Qr`） | 第一行片名（BodyLg 600）、**第二行**說明（`lUZol`，Body）、右邊徽章。原因本來就在自己那一行。 |
| 程式：即時活動 | `GenerationWorkspaceV2.tsx` `FeedRowItem`；文字在 `generationEventCopy.ts` `feedRowView` | 一個 `flex` 單行；片名 `truncate` 可縮，原因 `shrink-0` 不縮——所以原因越長，片名越短。 |
| 程式：佇列 | `GenerationWorkspaceV2.tsx` `QueueRow`；文字在 `generationQueueRow.ts` | 片名 `truncate` 單行；說明行桌機 `truncate`、手機會折行；失敗說明是 `--text-secondary`（稿子畫成 `$error-text`）。 |

寬度試算（14px 中文一字約 14px）：

- 手機 390：即時活動清單 358 寬。現在片名能用的寬度 = 358 − 邊框與內距 30 − 圖示＋間距 24 − 「失敗」28 − 兩個「·」與間距 ≈ 34px（等於 dev 量到的）。改成兩行後，片名那一行約 **240px（約 17 字）**，原因那一行約 **300px**。
- 桌機 400 側欄：片名那一行約 **290px（約 20 字）**。
- 手機佇列：說明行約 193px，最長原因 182px，**一行放得下**（放不下時手機本來就會折行）。片名（16px）一行約 12 字。

## Sally 的裁定

1. **原因換到第二行。** 即時活動裡「某一部片的結果＋原因」這三種列——④ 失敗、⑤ 已停止（預算上限）、⑤b 已停止（帳號沒錢）——改成兩行：
   - 第一行：圖示 · 階段字 · `·` · 片名。
   - 第二行：原因。左緣對齊階段字（往右縮 24px＝圖示 16＋間距 8），**前面不加「·」**，樣式不變（Body 14、`$text-secondary`），**永遠完整顯示**，太長就折行。
   - 理由：片名是「是哪一部」，原因是「為什麼」，兩個都不能丟，同一行一定有一個要被犧牲。DESIGN.md 的 Status Rows 本來就是「標題＋原因說明行」，活動記錄頁（同一句原因）和左邊佇列也都是原因在第二行——三個地方排法一致，使用者不用學兩套。
2. **片名在所有列最多兩行，超過才在結尾加「…」。** 不只失敗列，進行中、已過、中斷、完成列也一樣（同一個欄位一個規則）。兩行在手機約 34 字、桌機約 40 字，「影集名＋SxxEyy＋集名」幾乎都放得下；真的超過時被切的是集名的尾巴，不是集數。截斷只是視覺，螢幕閱讀器仍唸全名（播報字串不變）。
   - 不選「中間截斷、集數硬保留」：要在前端拆字串（單部任務送來的是「和平使者 S01E01」一整串，批次送來的是影集名＋「S01E01 集名」兩段），規則多、會壞；兩行就夠了。
3. **批次層的列不動。** ⑦「批次完成 · 本批次 · 完成 1 部、失敗 2 部」、⑧「批次已取消 · 本批次 · 完成 3 部」維持一行——第二段是整批的統計，不是某一部片的原因，而且「本批次」固定 3 個字，沒有片名要保護（358 寬實測仍放得下一行）。⑩⑪ 的尾端金額／廠商也不動。
4. **圖示對齊第一行。** 兩行的列，圖示垂直置中在第一行，不是整列置中。列的上下內距不變（8／14），兩行之間不加間距（行高 1.625 已經夠）。
5. **桌機和手機同一個排法。** 1440 的 400px 側欄一樣會把「和平使者 S01E01」截掉，所以不只手機改。
6. **佇列失敗項目：排法不用改，只對齊兩件事。**
   - 原因照 Bob 那張單放在說明行（第二行），用跟即時活動、活動記錄同一份文字。手機最長的原因一行放得下。
   - 說明行用 `$text-secondary`，**不是紅色**——「失敗」兩個字已經在紅色徽章上，原因是說明、不是狀態（DESIGN.md「狀態押在標籤與圖示上」）。app 現在就是次要色，**改的是稿子**（`f14sNh`、`Ah8fN` 原本畫 `$error-text`）。
   - 佇列片名也套裁定 2：最多兩行。桌機 712 寬幾乎不會用到，手機長影集名才會。
7. **稿子補一列真的失敗。** F11-D-v2 左邊佇列有一部失敗，右邊即時活動卻沒有對應的失敗列，兩邊對不起來。在即時活動補一列失敗（「和平使者 S01E01」＋「影片所在的資料夾寫不進字幕」），佇列那一列改成同一部片、同一句原因。手機 F11-M-v2 的佇列失敗列也改成同一部片、最長的原因，證明 358 寬放得下。
8. **不開新的規格畫面。** 改的是既有規格稿 F11-SPEC-LOG 自己的示範列，再在 ④ 下面加一列「④ 手機（358 寬）」示範。這是在規格畫面裡加規格，不是把規格塞進畫面稿，符合「spec 畫面獨立」的規則；`SCREENS` 也不用加。

## 已知 app 差異（改完稿子之後，給實作單用）

- **即時活動**：app 是單行，片名會被原因擠掉——這張稿子定案後要開實作單（見下）。
- **佇列片名**：app 是單行 `truncate`，稿子改成最多兩行。
- **佇列失敗說明的顏色**：app 已經是次要色，這次是稿子改成跟 app 一樣，**不用改程式**。
- 其他佇列說明行的顏色（排隊中稿子是 `$text-muted`、app 是次要色）不在這張範圍。

## 實作單要做的事（給 Bob 建單參考，不是這張的範圍）

- `feedRowView` 把原因跟片名分開回傳（例如多一個 `detail` 欄位），④⑤⑤b 的 `parts` 只放片名；`announce` 播報字串不變。
- `FeedRowItem` 改成「圖示＋內容欄」：內容欄第一行是階段字 · 片名（片名 `line-clamp-2`、可折行）· 尾端；有 `detail` 時第二行顯示原因（不加「·」、不截斷）。圖示置中在第一行。
- `QueueRow` 片名 `truncate` 改成最多兩行。
- 可以跟 Bob 正在開的「佇列失敗講具體原因」那張併成同一張，或排在它後面；兩張都改 `GenerationWorkspaceV2.tsx`，分開做容易衝突。
- testid 不變；`generation-workspace-v2` 的視覺基準若有失敗列的夾具會變，`-linux` 基準交給 CI 開的 bootstrap PR。
- 手機 390 驗收：「和平使者 S01E01」「花月殺手」要整個看得到，原因整句看得到。

## Inline agent 提示詞（Alexyu 在 Pencil 裡貼上執行，跑完按 ⌘S）

```
在 ux-design.pen 改三張稿：F11-SPEC-LOG（tT3JX）、F11-D-v2（l8FsB）、F11-M-v2（PXB0z）。目的是把即時活動「失敗／已停止」列的原因移到第二行，讓片名先完整顯示。

全域規則：
- padding／gap 的 0 一律寫 $Space/none，不要寫數字 0。
- 顏色、字級、間距只用 $ 變數，不寫裸數字。
- 每個新節點都要設 name。
- 不要改任何其他列、不要改畫布上的說明文字（caption）、不要動 components。
- 新增的文字不能跟其他內容重疊。

【共用做法：把一個「evt」列改成兩行】
一個 evt 列原本是水平 frame，子節點依序是：圖示、階段字、點1、片名、點2、原因、撐開（evt-sp）。改法：
1. 在這個 evt 裡新增 frame，name「evt-line」：layout 水平，width fill_container，gap $Space/sm，alignItems center，padding $Space/none。放在 evt 的第 1 個位置。
2. 把「圖示、階段字、點1、片名」依序 Move 進 evt-line。
3. 片名改 width fill_container、textGrowth fixed-width。
4. 刪掉「點2」和「撐開」。
5. 在 evt 裡新增 frame，name「evt-reason-row」：layout 水平，width fill_container，padding [$Space/none, $Space/none, $Space/none, $Space/xl]（左邊 24，讓原因跟階段字左緣對齊）。放在 evt-line 後面。
6. 把「原因」Move 進 evt-reason-row，改 width fill_container、textGrowth fixed-width。原因的字型、顏色不變（$text-secondary、$Type/Body/*）。
7. evt 本身改 layout vertical、gap $Space/none、alignItems start；padding 維持 [$Space/sm, $Space/md-plus] 不變。

一、F11-SPEC-LOG（tT3JX）
1. ④ 失敗：evt 是 stf9c（在 dNQa0 → Yn4Z4 裡）。子節點：圖示 r38GQ、階段字 q3gyb、點1 SXR7z、片名 ft0Nk、點2 xpUlw、原因 Ti6lM、撐開 b8nqiD。照共用做法改兩行。
   再把 ft0Nk 的文字改成「和平使者 S01E01」，Ti6lM 的文字改成「影片所在的資料夾寫不進字幕」。
2. ⑤ 已停止：evt 是 cu3CY（在 xPTom → P7LB9z 裡）。圖示 CL367、階段字 Z8vfp、點1 qlPCe、片名 bVg7L、點2 eMGG9、原因 XCzdu、撐開 tmAst。照共用做法改兩行，文字不變。
3. ⑤b 帳號沒錢：evt 是 c5WP3（在 xkmMv → VkXSm 裡）。圖示 LyqLI、階段字 aqtme、點1 ArrvU、片名 oZ6xi、點2 iz5kl、原因 r2yv5、撐開 U88mXO。照共用做法改兩行，文字不變。
4. 新增「④ 手機」示範：Copy 整個 dNQa0（做完第 1 步之後的），放進 tT3JX，位置緊接在 dNQa0 後面（tT3JX 的第 7 個子節點，在 xPTom 前面）。新節點 name「spec-entry-m」。
   Copy 完先 Get 這個新節點拿到子節點的新 id（一般節點 Copy 的 descendants 用名稱改不會生效），再逐一 Update：
   - 它的 row-box 改 width 358（手機 390 螢幕上清單的寬度），name「row-box-m」。
   - 它的 spec-note 文字改成：
     「④ 手機｜390 寬的螢幕上清單只有 358 寬。片名那一行扣掉圖示、「失敗」與「·」還有約 240px（約 17 個中文字）；最長的原因「影片所在的資料夾寫不進字幕」約 182px，放在第二行有約 300px。改版前兩者擠同一行，片名只剩 34px，會變成「和…」。」
5. 改說明文字（只改 content）：
   - VMUYs 改成：
     「右欄「即時活動」只會出現下面十三種列。每一列＝圖示 16 · 階段字（Body 600）· 「·」· 項目（Body 次要色）· 撐開 · 尾端（Mono，可選）。④⑤⑤b 是「某一部片＋原因」：原因換到第二行，左緣對齊階段字，前面不加「·」，永遠完整顯示；片名在所有列都可以折成兩行，超過兩行才在結尾加「…」。圖示對齊第一行。同一部片同一個階段只佔一列，翻譯百分比原地更新；批次裡每一部的結果只看批次事件（changed_item），不看單部的轉錄事件。」
   - I2WeeC 改成：
     「④ 失敗｜批次成員：changed_item = failed，原因來自 failure_reason（reason 是 error 時）或 reason；單部：transcription_failed 的 failure_reason。文字跟活動記錄同一份。原因放第二行，片名先完整顯示。後端原字串不上畫面。帳號沒錢不算失敗，見⑤b。」
   - I4V8GW 改成：
     「⑤ 已停止（單部）｜詳情頁單獨生成時碰到預算上限：你要求了但沒發生，赭色，不算失敗。原因放第二行，同④。」
   - A4WJFd 改成：
     「⑤b 已停止・帳號沒錢（單部）｜OpenAI 或 Anthropic 帳號沒有額度了。跟預算上限一樣用赭色：片子沒壞，儲值後再按一次。後端原字串不上畫面。原因放第二行，同④。」

二、F11-D-v2（l8FsB）
1. 即時活動補一列失敗：Copy 第一步改好的 stf9c，放進 log-body（dPQ9b），再 Move 到 dPQ9b 的第 3 個位置（在 jque4「完成 · 芭比」後面、a7UpAm「提取音訊 · 奧本海默」前面）。新節點 name「evt-fail」。文字維持「失敗」「和平使者 S01E01」「影片所在的資料夾寫不進字幕」。
2. 佇列的失敗列 f14sNh（ref 元件 aw4Qr）用 descendants 更新：
   - bSNmA content「和平使者 S01E01」
   - lUZol content「影片所在的資料夾寫不進字幕」，fill $text-secondary（原本是 $error-text）
   - 其他（徽章 IcPTo、圖示 CQhw2、徽章字 OLg1J「失敗」）不變。

三、F11-M-v2（PXB0z）
1. 佇列的失敗列 Ah8fN（ref 元件 aw4Qr）用 descendants 更新：
   - bSNmA content「和平使者 S01E01」
   - lUZol content「影片所在的資料夾寫不進字幕」，fill $text-secondary
   - 其他不變。

做完檢查：
- tT3JX 裡 stf9c、cu3CY、c5WP3、spec-entry-m 的 evt 都是兩個子節點（evt-line、evt-reason-row）；原因文字的左緣 x 跟階段字的左緣 x 一樣（都是 row-box 內 38）；原因前面沒有「·」。
- spec-entry-m 的 row-box 是 358 寬；片名「和平使者 S01E01」整個看得到，原因整句一行。
- dPQ9b 有五列，evt-fail 在第 3 列；log-body 沒有東西超出。
- f14sNh、Ah8fN 的說明行是一行、次要色；Ah8fN 仍是 358 寬、88 高。
- 三張稿都沒有 partially clipped／fully clipped 的節點；沒有裸數字 0。
- 新增的 frame 存檔前截圖可能是空白（Pencil 的已知現象），先存檔再截。
- 最後對 tT3JX、l8FsB 的 DUvwI、PXB0z 的 Ruyx7 各截一張圖。

不要切分支、不要 commit，存檔（⌘S）後回報。
```

## 跑完之後（Sally／我）

1. **先確認分支沒被切走**：`git branch --show-current`（背景可能有 dev agent 在用工作目錄）。被切走就切回並通知 dev agent。
2. **確認已存檔**：`git hash-object ux-design.pen` 對 `git rev-parse HEAD:ux-design.pen`；MCP 讀的是 app 記憶體，不是磁碟。
3. **MCP 唯讀複審**：
   - `Get("tT3JX", …)` 列出 ④⑤⑤b 與 spec-entry-m 的結構，逐字比對四段說明文字；用 `ctx.bounds` 確認原因的 x 等於階段字的 x、`ctx.problems` 為空。
   - `Get("dPQ9b")` 確認 evt-fail 在第 3 列；`Get("f14sNh")`、`Get("Ah8fN")` 確認文字與 `$text-secondary`。
   - 掃新節點有沒有裸數字（`padding` 陣列裡的數字、`typeof fontSize === "number"`）。
   - 本機跑 `python3 scripts/check-design-tokens.py` 和 `python3 scripts/check-design-refs.py`（被刪的 `xpUlw`／`b8nqiD`／`eMGG9`／`tmAst`／`iz5kl`／`U88mXO` 已確認程式沒有引用）。
4. **出截圖**：`python3 scripts/export-pen-screenshots.py`，只留：
   - `_bmad-output/screenshots/flow-f-subtitle-v2/f11-spec-log.png`
   - `_bmad-output/screenshots/flow-f-subtitle-v2/f11-d-v2.png`
   - `_bmad-output/screenshots/flow-f-subtitle-v2/f11-m-v2.png`
   - `_bmad-output/pen-node-ids.json`（有新節點、也有刪掉的節點，一定會變）；`pen-tokens.json` 有變才留。
   - 其他 PNG 都是重繪雜訊，`git checkout --` 還原。
5. **`SCREENS` 不用加**：沒有新的最上層畫面，三張稿都已登錄（`tT3JX`、`l8FsB`、`PXB0z`）。加之前仍可用 grep 確認沒有重複 key。
6. **sprint-status**：本條改 `done`，並加實作單條目（或併進 Bob 正在開的佇列原因單）；這份檔改狀態、補「複審結果」。跟 `.pen`、截圖同一個 PR，用 plumbing commit，不碰工作目錄分支。
7. **通知**：dev 那邊 `generationEventCopy.ts` 第 6 行與 `GenerationWorkspaceV2.tsx` `FeedRowItem` 的註解要在實作單裡一起更新（`Design ref` 仍是 `tT3JX`／`DUvwI`，不用改 id）。
