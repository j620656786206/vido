# Disc：批次執行中，筆電寬度（1024／1280）看生成畫面，右邊「即時活動」被擠成一字一行

Status: blocked: waiting Pencil

**Source:** 2026-10-20 #874（`disc-2026-10-gen-queue-done-row-1024`）Dev Agent Record「附帶發現」：批次執行中、佇列裡有一部正在處理時，1024 寬右邊即時活動欄被擠到 **54px**（一字一行，等於不能讀）；1280 也只剩 **310px**（設計是 400）。#874 之前就這樣，用 main 的程式量也是 54。只要有一部在處理就會發生，也就是批次執行的大部分時間。⚖️ Alexyu 2026-10-20：開單，請 Sally 決定怎麼修。

## 為什麼會被擠

- 處理中那列底下的六步進度條：六格各 72 寬＋五條連接線各 26 寬＝**562**，再加左邊縮排 54＝**616**，再加列的左右內距與框線 34＝整列至少 **650**。
- 左欄佇列是 `flex-1`，沒有 `min-w-0`，所以縮不到 650 以下；右欄寫的是 `lg:w-[400px]`，卻可以被壓縮。結果是右欄讓位。
- 內容區寬度（扣掉側欄 240 與左右邊距各 32）：1024 → **720**、1280 → **976**、1440 → **1136**。
  - 1024：720 − 650 − 16（欄距）＝ **54**。
  - 1280：976 − 650 − 16 ＝ **310**。
  - 1440：1136 − 400 − 16 ＝ 720 ≥ 650，所以 1440（設計稿的寬度）看起來沒事。
- 預算上限終態（沒有處理中的列）1024 時右欄是 400，所以現在同一個畫面「執行中」跟「結束」兩種狀態右欄寬度差很多，批次一結束版面會跳。

## ⚖️ Sally 裁定（2026-10-20）

**選 (d)：(a) 右欄固定 400、不讓寬度 ＋ (b) 處理中那列的進度條看「列本身」夠不夠寬，不夠就改成直排（跟手機同一個樣子）。**

白話：

- 右邊「即時活動」在桌機寬度永遠是 400，跟設計稿、也跟批次結束後的畫面一樣寬。空間不夠時，讓左邊佇列變窄。
- 佇列變窄後，處理中那列的六步進度條橫排放不下，就改成直排：一步一行，圓點＋步驟名＋右邊百分比。這個直排在手機（F11-M-v2 `n79vZ`）已經在用，使用者看過，不是新樣子。
- 只看一個切換點：**列內可用寬度 616 以上用橫排**（跟現在一模一樣），**不到 616 用直排**。用這列自己的寬度判斷，不看視窗寬度，所以側欄收合、不同螢幕都會自己算對。

各寬度結果：

| 視窗 | 佇列欄 | 列內寬 | 進度條 | 即時活動 |
|---|---|---|---|---|
| 1440 | 720 | 686 | 橫排（不變） | 400（不變） |
| 1280 | 560 | 526 | **直排** | **400**（原本 310） |
| 1024 | 304 | 270 | **直排** | **400**（原本 54） |
| 390／320 | 全寬 | — | 直排（不變） | 在佇列下面（不變） |

為什麼不選其他做法：

- **只做 (a)（右欄固定、左欄 `min-w-0`）不做 (b)**——不選。1024 時列內只剩 270，橫排要 562，進度條會超出列、被裁掉或整頁橫捲。問題只是換到左邊。
- **只做 (b)，進度條縮成只有圓點、或步驟名折行**——不選。1024 列內 270：六個步驟名（最長「提取音訊」4 字約 48px）加五條連接線，最少也要約 330，放不下；步驟名折成兩行，「AI校正」這種會斷得很難看。只留圓點，就看不出「已經做完哪些、還剩哪些」，等於這條進度條沒用（現在在哪一步，列的說明行和徽章已經講了）。
- **(c) 1440 以下即時活動改放到佇列下面**——不選。1280 是很常見的筆電寬度；即時活動是設計成在旁邊一直看得到的欄，放到佇列下面就要捲很遠才看得到，等於在常見寬度把這欄拿掉。而且問題只出在「有一列在處理」，為這個改整個頁面的排法不划算。
- **做兩個切換點（1280 用縮小版橫排、1024 才直排）**——不選。1280 列內 526，把每格縮到約 64 寬勉強放得下，但要多一套樣子、多一組量測和基準；直排在 1280 只是讓處理中那列高一點（約多 210px，一次只會有一列在處理），可以接受。之後如果真的覺得 1280 太高，再另外裁。

**設計稿要補（所以本單先 blocked）：** 1440 的 F11-D-v2 不用改，但「窄桌機右欄固定 400、進度條改直排」是新的版面規則，設計稿裡沒有任何一張畫面記錄。照規則（規格說明要放獨立的規格畫面，不擠進既有畫面），新增一張 **F11-SPEC-WIDTH**：1024 的示意（304 寬佇列放一列直排進度條的處理中列＋400 寬即時活動框）＋各寬度規則文字。提示詞見下方「Pencil 提示詞」。

## Story

身為在筆電（1024／1280 寬）上看批次生成的使用者，
我想要批次跑到一半時，右邊「即時活動」跟平常一樣寬、一行一行看得懂，
這樣我不用把視窗拉寬，也能看到每一部片做到哪、哪一部失敗了、為什麼。

## Acceptance Criteria

0. **開工前：** 確認設計稿已補（Pencil MCP 唯讀 `Get`：F11-SPEC-WIDTH 存在、裡面的 `demo-row-running-vertical` 是直排進度條、`demo-live-400` 寬 400），截圖 PR 已合併（`flow-f-subtitle-v2/f11-spec-width.png`，`scripts/export-pen-screenshots.py` 的 `SCREENS` 已加新畫面、`_bmad-output/pen-node-ids.json` 已更新）。沒補就停下來等。
1. **右欄不讓寬度：** 工作區 body 在 `lg:` 以上左右並排時，右欄（`workspace-event-log`）固定 400、不會被壓縮（加 `lg:shrink-0`），左欄佇列加 `min-w-0`，讓左欄縮。640～1023（右欄在下面）與手機的排法不動。
2. **進度條看列本身的寬度：** 佇列處理中那列的進度條用容器查詢（`@container`，前例 `LibraryListV2`）判斷：
   - 列內寬度 **≥ 616**：橫排＋左邊縮排 54，計算出來的樣子跟現在 `sm:` 以上**完全一樣**（1440 不能有任何差異）。
   - 列內寬度 **< 616**：直排，計算出來的樣子跟現在手機（`max-sm`）的直排**完全一樣**（一步一行、圓點 22、步驟名 Body 14、百分比靠右），沒有縮排。
   - 五步版（抽取字幕路線）用同一個切換點。
   - **對話框（F3／F4 生成進度、F8 批次對話框）裡的進度條不受影響**：元件預設仍看視窗寬度，只有工作區佇列這一處改看容器。開工時 `git grep GenerationProgressV2` 列出所有使用處，確認只有 `QueueRow` 傳新設定。
3. **量測合格線（真瀏覽器，改完量；照 #874 的做法：狀態查詢＋假 SSE stub，量完刪掉、不進版控）：** 頁面 `/activity?view=generation`。佇列放：一部完成、一部處理中（翻譯中 45%，要讓第二條以後的 EventSource 也收到 `translation_progress`）、一部排隊中、一部失敗（「影片所在的資料夾寫不進字幕」，片名用魔戒長名）、一部已略過（19 字原因）。另外跑一次預算上限終態。每個寬度都記下數字寫進 Dev Agent Record。
   - **1024（執行中）：** 即時活動欄寬 **400**（±1）；即時活動每一列的階段字、片名、原因都沒有一字一行（每個文字元素寬 ≥ 120px）；處理中列的進度條是**直排**（六個 `gen-stage-*` 的左緣 x 相同、由上往下排），每個步驟名 `scrollWidth <= clientWidth`；處理中列的片名欄寬記下來（< 64px 就停下來回報 Sally，不要順手改）；所有說明行 `scrollWidth <= clientWidth`；頁面無橫捲（`document.documentElement.scrollWidth === 1024`）。
   - **1024（預算上限終態）：** 即時活動欄寬 400（跟改前一樣，沒有退步）；無橫捲。
   - **1280（執行中）：** 即時活動欄寬 **400**；進度條直排、步驟名整字；說明行沒有「…」；無橫捲。
   - **1440（執行中）：** 即時活動欄寬 400；進度條**橫排**、縮排 54；六個 `gen-stage-*` 的位置（x、y、寬）跟改前**逐一相同**（改前先量一次記下來）；無橫捲。
   - **390、320（執行中）：** 跟改前一樣——進度條直排、即時活動在佇列下面、步驟名整字；無橫捲。
   - 選做：1024 把側欄收合後再量一次（內容區變寬），記下列內寬度與進度條排法，確認切換點是看列寬、不是看視窗。
4. **測試（先改測試看它紅，再改程式看它綠）：**
   - `GenerationWorkspaceV2.spec.tsx`：右欄有 `lg:shrink-0`、左欄有 `min-w-0`；處理中那列的外層有 `@container`，進度條外框用容器變體縮排（不再是 `sm:pl-[54px]`）。
   - `GenerationProgressV2.spec.tsx`：預設（不傳新設定）的 class 跟現在逐字一樣（對話框不變）；傳容器設定時，橫排相關 class 改用容器變體、直排相關改用容器 max 變體。
   - jsdom 沒有版面，寬度靠 AC #3 真瀏覽器量。
5. **視覺基準：**
   - 既有六張 `generation-workspace-v2/*`（畫廊 1200 寬，內容區 1136，跟 1440 一樣佇列 720、列內 686 ≥ 616，仍是橫排）**應該不變**；重產後 byte 有差但內容沒變的當雜訊還原。對話框相關（`GenerationProgressV2`、F3／F4／F8）基準也不應變。**任何一張內容真的變了就停下來回報**，不要直接換基準。
   - 新增一個畫廊 fixture `generation-workspace-v2/running-narrow`：跟 `running` 同一份資料，`width: 720`（＝1024 的內容區），用來守住「右欄 400＋直排進度條」。darwin 本機 `update-missing` 產生；`-linux` 不要本機產，交給 CI 的 `chore(visual): bootstrap` PR。
6. **UX 驗收：** 1024、1280 截圖對 F11-SPEC-WIDTH 的 `demo-1024`（右欄 400、處理中列直排）；1440 截圖對 F11-D-v2 `l8FsB`（不變）；390 對 F11-M-v2 `n79vZ`。截圖留在本機 session scratchpad，PR 描述寫量測數字。
7. **全綠：** `pnpm nx run web:test`、`web:typecheck`、`lint:all`（看「✖ N problems (X errors」那行）、`format:check`、`scripts/check-design-refs.py`（`GenerationWorkspaceV2.tsx` 檔頭 Design ref 加 F11-SPEC-WIDTH 的新 id）。沒有刪改 testid；E2E 先 `git grep -n "gen-stage-\|workspace-event-log" tests/e2e` 看有沒有寫死寬度或排法的斷言，再跑 `generation-workspace.spec.ts`、`generation-workspace-mobile.spec.ts`。

## Tasks / Subtasks

- [ ] T0：確認設計稿與截圖已補（AC #0）
- [ ] T1：改前在 1440 量一次六個 `gen-stage-*` 的位置，記下來（AC #3 的比對基準）
- [ ] T2：先改兩支 spec 的期待值，跑一次看紅（AC #4）
- [ ] T3：右欄 `lg:shrink-0`、左欄 `min-w-0`（AC #1）
- [ ] T4：`GenerationProgressV2` 加「看容器」設定，`QueueRow` 外層 `@container`、縮排改容器變體（AC #2）；跑 spec 看綠；DevTools 確認容器變體的 CSS 真的有產生
- [ ] T5：真瀏覽器量 1024／1280／1440／390／320（AC #3）
- [ ] T6：視覺基準＋新 fixture（AC #5）、UX 截圖（AC #6）
- [ ] T7：全綠（AC #7）

## Dev Notes

- 相關程式位置（origin/main，2026-10-20）：
  - `GenerationWorkspaceV2.tsx:848-849`：body `lg:flex-row`；左欄 `flex min-h-0 flex-1 flex-col gap-3`（沒有 `min-w-0`）。
  - `GenerationWorkspaceV2.tsx:520`：右欄 `<aside data-testid="workspace-event-log">` 的 `lg:w-[400px]`（沒有 `shrink-0`）。
  - `GenerationWorkspaceV2.tsx:324`：`QueueRow` 的 `<li>`（`px-4 py-3.5`＋框線）；`:370` 進度條外框 `sm:pl-[54px]`。
  - `GenerationProgressV2.tsx`：步驟 `sm:w-[72px]`、連接線 `sm:w-[26px]`、`ol` 的 `sm:flex-row sm:items-start sm:gap-0`、`sm:justify-start/center`、步驟 `max-sm:p-1 sm:flex-col sm:gap-1.5`、步驟名 `sm:text-xs sm:leading-normal`、百分比 `sm:ml-0`、連接線 `hidden sm:block`。
- 做法建議（細節由 dev 決定）：`GenerationProgressV2` 加一個選填 prop（例如 `widthSource?: 'viewport' | 'container'`，預設 `viewport`）。`container` 時，上面那些 `sm:` 換成 `@min-[616px]:`、`max-sm:` 換成 `@max-[616px]:`。**Tailwind 只認寫死的完整 class 字串**，兩套 class 都要整串寫出來，不能用字串拼接。`QueueRow` 的 `<li>` 加 `@container`，進度條外框 `@min-[616px]:pl-[54px]`。
  - 切換點用 616（＝562 橫排＋54 縮排），量的是 `<li>` 的內容寬（扣掉 `px-4` 與框線）。1440 列內 686、1280 是 526、1024 是 270。
  - Tailwind v4 內建容器查詢；arbitrary 值 `@min-[616px]:`／`@max-[616px]:` 請在 DevTools 確認規則真的有產生再往下（記憶 `feedback_css_verify_before_iterate`）。
  - `@container` 會讓 `<li>` 的寬度不再被內容撐開，這本身也幫忙解掉「列把左欄撐到 650」；但 AC #1 的 `min-w-0`／`shrink-0` 還是要加，不要只靠這個。
- 1024 時佇列只剩 304：處理中列的片名欄大約 80px（約 5 字一行，兩行後才「…」）。這跟現在預算上限終態的 1024 一樣（#874 量過、可接受），不在本單處理；量到 < 64px 才回報。
- 處理中列改直排後高度約多 210px（六行×30＋五個間距）。一次只會有一列在處理，佇列本來就會捲，可以接受。
- 螢幕報讀：直排、橫排是同一個 `<ol aria-label="字幕生成進度">`，只換排法，讀出來一樣。

### 不要做的事

- 不改對話框（F3／F4／F8）的進度條——元件預設行為逐字不變。
- 不做縮小版橫排、不隱藏步驟名、不讓步驟名折行。
- 不改右欄內容與即時活動的列規則（F11-SPEC-LOG）、不改說明行的 class（#871／#874 的規則）。
- 不加新的視窗斷點、不改 640～1023 的上下排法。
- 不本機產 `-linux` 基準。

### 設計稿 / Design ref

- **F11-SPEC-WIDTH**（新增，id 待 Pencil 建好後回填）：窄桌機版面規則＋1024 示意。
- F11-D-v2 `l8FsB`：1440 執行中，處理中列 `OkdGK`（橫排進度條 `gkGTR` → 元件 `XkGvG`），右欄 `DUvwI`（400）——不變。
- F11-M-v2 `PXB0z`：手機，處理中列 `n79vZ`（直排進度條 `p8No2j`）——窄桌機直排照這個樣子。
- F12-D-v2 `iH98f`：預算上限終態——不變。
- F11-SPEC-LOG `tT3JX`：即時活動列規則——不變。
- 元件 Component/GenerationProgress-v2 `XkGvG`、Component/GenQueueRow-v2 `aw4Qr`——不變。

### Pencil 提示詞（Alexyu 貼給 Pen 的 Inline AI Agent）

```
在 ux-design.pen 新增一張獨立的規格畫面「F11-SPEC-WIDTH」，記錄生成工作區在窄桌機（1024／1280）的版面規則。只新增節點，不要改任何現有節點：F11-D-v2（l8FsB）、F11-M-v2（PXB0z）、F12-D-v2（iH98f）、F11-SPEC-LOG（tT3JX）、F8-SPEC-ROW（NfHKO）、元件母版 aw4Qr、XkGvG 都不要動。

位置
1. 放在群組「F · 規格 Spec」（id JzmvC）裡，F8-SPEC-ROW（id NfHKO，x 24740、y 34393、寬 880）的右邊。
2. 先檢查 x 25720～26700、y 34300 以下有沒有別的節點；有的話整組往右移 980 再檢查，回報最後用的座標。

新增的節點（每個都照這個 name 命名）
3. 標題文字 cap-F11-SPEC-WIDTH：x 25720、y 34348，內容「F11 · 生成工作區・窄桌機（1024／1280）版面規格」。fill $text-muted、fontFamily $Type/Family/Primary、fontSize $Type/Body/Size、lineHeight $Type/Body/Line、fontWeight $Type/H4/Weight。
4. 畫面 frame F11-SPEC-WIDTH：x 25720、y 34393、寬 880、高度隨內容。layout vertical、gap $Space/lg、padding [$Space/lg-plus, $Space/xl]、fill $bg-secondary、cornerRadius $radius-lg。下面 5～8 都放在它裡面，照順序由上往下。
5. 文字 spec-intro：width fill_container、textGrowth fixed-width，內容是下面「文字 A」。
6. frame demo-1024：width fill_container、layout horizontal、gap $Space/lg、padding $Space/none、alignItems start。裡面兩個子節點，左右排：
   6a. frame demo-queue-304：寬 304、layout vertical、gap $Space/sm-plus、padding $Space/none。
       把 F11-M-v2 裡的 row-進行中（id n79vZ，有直排進度條的那列）Copy 一份放進 demo-queue-304。Copy 完先 Get demo-queue-304 拿到新那列的 id，再用新 id Update：name 改成 demo-row-running-vertical、width 改成 fill_container。它裡面的字和樣式都不要改。
   6b. frame demo-live-400：寬 400、layout vertical、gap $Space/sm、padding [$Space/md, $Space/md-plus]、fill $bg-primary、cornerRadius $radius-md。裡面兩段文字：
       - demo-live-title：「即時活動（固定 400 寬）」。fill $text-primary、fontFamily $Type/Family/Primary、fontSize $Type/Body/Size、lineHeight $Type/Body/Line、fontWeight $Type/H4/Weight。
       - demo-live-note：width fill_container、textGrowth fixed-width，「1024 寬時內容區只有 720：佇列 304＋間距 16＋即時活動 400。空間不夠時讓佇列變窄，右欄不讓。」樣式用下面的「說明文字樣式」。
7. 文字 demo-caption：width fill_container、textGrowth fixed-width，「▲ 1024 寬・批次執行中：處理中那列的進度條改成直排（跟手機 F11-M-v2 同一個樣子），右邊即時活動維持 400。」
8. 五段文字 spec-note-1440、spec-note-1280、spec-note-1024、spec-note-phone、spec-note-why：都是 width fill_container、textGrowth fixed-width，內容依序是下面「文字 B」的 ①～⑤。

說明文字樣式（5、7、8 與 demo-live-note 共用）：fill $text-secondary、fontFamily $Type/Family/Primary、fontSize $Type/Label/Size、lineHeight $Type/Label/Line、fontWeight $Type/Body/Weight。

文字 A：
生成工作區在桌機寬度（1024 以上）左右並排：右欄「即時活動」固定 400 寬，任何時候都不會被擠窄；左欄佇列拿剩下的寬度。處理中那列的六步進度條，看「這一列本身」的寬度決定排法：列內可用寬度 616 以上用橫排（F11-D-v2，跟現在一樣）；不到 616 改用直排，跟手機 F11-M-v2 同一個樣子——一步一行，圓點 22＋步驟名＋右邊百分比，沒有左邊縮排。只有這一個切換點，不另做縮小版橫排。對話框（F3／F4／F8）裡的進度條不適用這條規則。

文字 B：
① 1440（F11-D-v2）：佇列 720，列內 686，616 以上 → 橫排，跟現在一模一樣；即時活動 400。
② 1280：佇列 560，列內 526，不到 616 → 直排；即時活動 400（改之前被擠成 310）。
③ 1024：佇列 304，列內 270 → 直排；即時活動 400（改之前被擠成 54，一字一行）。處理中那列會變高約 210，一次只有一列在處理，可以接受。
④ 手機 390／320：本來就是直排、即時活動在佇列下面，不變。
⑤ 為什麼不縮小橫排：1024 列內只剩 270，六個步驟名（最長 4 字約 48px）加五條連接線最少要約 330，放不下；只留圓點就看不出做完哪些、還剩哪些。直排手機已經在用，不用學新樣子。

規則
9. 樣式值只能用檔案裡現有的 $ 變數；數值 0 一律寫 $Space/none，不要寫裸數字或色碼（x、y、寬度這類尺寸數字除外）。
10. 改完自己檢查：F11-SPEC-WIDTH 跟其他節點沒有重疊；cap-F11-SPEC-WIDTH 在畫面框上方、不壓到框；demo-row-running-vertical 的直排六步（提取音訊、轉錄中、翻譯中、簡轉繁、AI校正、完成）都整字顯示、沒有超出 304 寬；demo-queue-304 和 demo-live-400 沒有互相壓到；所有文字都沒有超出各自的框。

不要切換分支、不要 git commit、不要開 PR；改完存檔（⌘S 或選單 File → Save），回報改了哪些節點 id。
```

存檔後：Sally 用 Pencil MCP 唯讀 `Get` 複審（F11-SPEC-WIDTH 位置不重疊、`demo-row-running-vertical` 是直排、`demo-live-400` 寬 400、文字逐字對）；在 `scripts/export-pen-screenshots.py` 的 `SCREENS` 加 `"<新 id>": ("flow-f-subtitle-v2", "f11-spec-width")`，出這一張截圖並跑 `--snapshot-only` 更新 `pen-tokens.json`／`pen-node-ids.json`（另一個 session 占用工作目錄時，照記憶 `feedback_pen_inline_agent_workflow` 在 scratch 複本出圖、plumbing commit），**只 stage 新截圖與兩份快照**，其餘重產雜訊不進版控。截圖 PR 合併後，本單 Design ref 回填新 id、改 `ready-for-dev`。

### References

- [Source: apps/web/src/components/subtitle/GenerationWorkspaceV2.tsx:324-395（QueueRow 與進度條外框 sm:pl-[54px]）、:520（右欄 lg:w-[400px]）、:848-849（body／左欄）]
- [Source: apps/web/src/components/subtitle/GenerationProgressV2.tsx（橫排 72×6＋26×5＝562、手機直排）]
- [Source: apps/web/src/components/library/LibraryListV2.tsx:34、LibraryListRowV2.tsx（@container 前例）]
- [Source: apps/web/src/routes/test/-gallery.fixtures.tsx:7206（generation-workspace-v2/running，width 1200）]
- [Source: _bmad-output/implementation-artifacts/disc-2026-10-gen-queue-done-row-1024.md Dev Agent Record「附帶發現」（1024 右欄 54、1280 右欄 310、整列 650）]

## Change Log

| 日期 | 變更 |
|---|---|
| 2026-10-20 | 建單（Bob）。Sally 裁定 (d)：右欄桌機固定 400、左欄讓寬度；處理中列的進度條看列寬，列內不到 616 改直排（手機同款）。設計稿新增獨立規格畫面 F11-SPEC-WIDTH，先 blocked 等 Pencil。 |

## Dev Agent Record

### Agent Model Used

### Debug Log References

### Completion Notes List

### File List
