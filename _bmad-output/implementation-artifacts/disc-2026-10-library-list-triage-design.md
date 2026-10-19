# Disc：媒體庫列表改成「修字幕清單」（設計稿）

Status: done

**Source:** 2026-10-19 `/impeccable critique 媒體庫 列表`（21／40；快照 `.impeccable/critique/2026-10-19T00-18-51Z__ps-web-src-components-library-librarylistrowv2-tsx.md`）＋ Alexyu 看正式機列表：「左邊有片名、年份跟類別，右手邊是評比星號跟字幕訊息，中間一大片空白，看起來很奇怪」「要不要每張卡片加 border，比較看得懂邊界」。

查證：空白不只是程式碼的問題——設計稿 A4p-D（`b1H71g`）本身就把片名欄 `info` 設成 `fill_container`，標籤被推到最右邊。

⚖️ **2026-10-19 Alexyu 裁定：**
- **1A：列表的工作是「修字幕清單」**——字幕狀態緊跟在片名後面、可以依字幕狀態排序、比對失敗的列有出路；格狀留給逛片。
- **2B：critique 的問題全部一起處理**（P0 手機看不到字幕狀態、P1 重點擺反、P1 失敗列死路、P2 繁中／中文、P3 小地方）。
- 看過 ASCII 草圖（表格欄位＋外框一道＋列間細線）後回「可以」。

## Sally 的設計裁定

1. **邊框：整個列表一道外框，列與列一條髮絲線，列本身不加框。** Alexyu 提的「每張卡片加 border」方向對（邊界不清楚），但 36 列各自有框、只隔 2px 會變成一疊格子；`DESIGN.md:679` 也規定「同一層級只准一道框，容器有框，其內的列用 `--border-subtle` 分隔線」。細線另一個作用：把左邊的片名和右邊的資訊連成同一列。
2. **空白的解法：欄位往左靠，多出的空間留在最右邊。** 欄序＝海報 → 片名（固定 360）→ 字幕（104）→ 評分（64）→ 規格（填滿、內容靠左）→ ›。重要的東西全在左半邊，空白只出現在規格灰字之後、› 之前，不再夾在兩個有意義的東西中間。上面加一列欄名（片名／字幕／評分／規格），讓它讀起來是表格而不是一排浮著的標籤。
3. **規格改成一段灰字** `4K · H.265 · DTS`（`$text-secondary`、等寬、Label 字級），不再是三顆藥丸；解析度寫 4K／1080p／720p，不寫原始寬高（B10p 已經是這個寫法）。
4. **字幕欄**沿用海報那套徽章與顏色（`deriveSubtitleStatus`：繁中／簡中／中文／缺中文／無字幕源／已略過／未翻譯）。不知道的時候寫灰字「不知道」（跟篩選器同一個詞），不留白——欄位留白會被讀成「沒問題」。
   - 「中文」維持 ⚖️ 2026-10-06 D2 裁定（算「有」，青碧）。critique P2 的疑慮只用提示框處理：「字幕軌沒標示繁簡」。不改顏色、不改字。
5. **標題計數後加「N 部缺中文」**（泥金連結色），點了等於套用「缺中文字幕」篩選。N＝0 時不顯示。
6. **排序多一個「字幕狀態」**：缺中文 → 無字幕源／未翻譯／已略過 → 不知道 → 簡中 → 中文 → 繁中。桌機下拉與手機抽屜都是同一份五個選項。
7. **比對失敗的列**：第一行「沒有找到這部片的資料」＋「失敗」徽章（跟 B10p 同一句）；第二行是檔名，從中間截斷、保留副檔名；海報位用 `PosterCard-v2/Unmatched` 的底。整列仍是一個連結，點進去就是 B10p（手動選片在那裡）——列裡不放第二個按鈕，免得一列兩個點擊目標。字幕欄照實顯示（失敗的片也可能真的有字幕）。
8. **手機（<640）**：欄名與規格藏起來；每列三行——片名、年份 · 類型、字幕徽章＋★評分。**字幕狀態在任何寬度都不藏。**

## 範圍（只動 `.pen`）

1. **A4p-D 原地改版**（保留節點 `b1H71g`，`a4p-d.png` 路徑不變，程式碼的 `Design ref` 不用改）。
2. **新稿 A4p-M**（手機列表，之前沒有）放在 Flow A 手機列。
3. **規格說明** `spec-note-a4p`（照 Flow A 既有的 spec-note 文字節點樣式，獨立放，不塞進畫面）。
4. `SCREENS` 加 `A4p-M` 一筆、重出截圖。

## 不做

- 程式實作：另開 `disc-2026-10-library-list-triage`（blocked-by 本單；Bob 依 Story Splitting 再決定要不要拆前後端——「字幕狀態」排序要動後端）。
- 「中文」的顏色與字（2026-10-06 已裁定）。

## Acceptance Criteria

1. A4p-D：表格容器一道外框＋列間髮絲線、欄名列、8 列真實資料（排序＝字幕狀態），片名 360、字幕 104、評分 64、規格填滿；列沒有自己的底色（第 1 列畫 hover：`$bg-secondary`）。
2. A4p-D 標題列寫「1,284 部 · 37 部缺中文」，排序鈕寫「字幕狀態」。
3. 比對失敗列照裁定 7。
4. A4p-M：7 列，三行結構，358 寬內不溢出，字幕徽章每列都看得到。
5. Sally 用 MCP 唯讀複審：文案逐字對、`ctx.problems` 為空、全部用變數。
6. 截圖：只 stage `a4p-d.png`、`a4p-m.png`；`check-design-refs.py` 與 token 快照通過。

## Inline Agent 提示詞（Sally，2026-10-19）

把下面整段貼給 Pencil 的 Inline AI Agent。節點 id 都是 2026-10-19 用 MCP 讀出來的。

```text
在 ux-design.pen 改「媒體庫列表檢視」。全部用既有變數，不寫死顏色、字級、間距（0 用 $Space/none）。所有新節點都要取名。

【A. 原地改 A4p-D（節點 b1H71g）】
1. 標題列計數 h7LED 文字改成「1,284 部」不變，在它右邊（同一個 hLeft 容器 gLK18 內）加 text「count-missing」：「· 37 部缺中文」，字型同 h7LED，fill $accent-text。
2. 排序鈕 ljXQ4（Component/SortDropdown 實例）的文字改成「字幕狀態」。
3. 刪掉清單容器 KuyJc 底下全部 8 個 row（w4Oo2、mURjL、TyMAE、Avq9r、p8vDv、ipIiD、kFLBb、CQLhY）。KuyJc 本身的 padding、gap 不動。
4. 在 KuyJc 底下加 frame「table」：vertical、width fill_container、gap $Space/none、cornerRadius $radius-md、clip true、1px 內側描邊 $border-subtle、無填色。
5. table 第一個子節點 frame「thead」：horizontal、width fill_container、height 36、alignItems center、gap $Space/lg、padding [$Space/none, $Space/md]。子節點依序：
   - frame「h-thumb」寬 48、高 1
   - text「h-title」「片名」寬 360
   - text「h-sub」「字幕」寬 104
   - text「h-rate」「評分」寬 64
   - text「h-spec」「規格」width fill_container
   - frame「h-chev」寬 20、高 1
   四段文字都用 $Type/Family/Primary、$Type/Label/Size、$Type/Label/Line、fontWeight 600、fill $text-muted。
6. 然後每一列前面先放一條 rectangle「div」：width fill_container、height 1、fill $border-subtle。再放 frame「row」：horizontal、width fill_container、gap $Space/lg、padding [$Space/sm-plus, $Space/md]、alignItems center、無填色。row 子節點依序：
   - frame「thumb」48×72、cornerRadius $radius-md、clip true（圖片沿用原本那幾列的海報圖；第 3 列用 PosterCard-v2/Unmatched 裡 poster-fallback 的 $bg-tertiary 底，中間放 lucide film 圖示 20×20 fill $text-muted）
   - frame「info」：vertical、寬 360、gap $Space/xs；內含 text「t」（$Type/Family/Primary、$Type/BodyLg/Size、$Type/BodyLg/Line、600、fill $text-primary）與 text「m」（$Type/Body/Size、$Type/Body/Line、fill $text-secondary）
   - frame「sub」寬 104、alignItems center：放字幕徽章（藥丸：cornerRadius $radius-pill、padding [$Space/xs, $Space/sm-plus]、文字 $Type/Label/Size 600）或灰字「不知道」（$Type/Label/Size、fill $text-muted、不是藥丸）
   - frame「rate」寬 64、gap $Space/xs、alignItems center：lucide star 12×12 fill $warning，text 評分（$Type/Family/Mono、$Type/Label/Size、fill $text-secondary）
   - text「spec」width fill_container：$Type/Family/Mono、$Type/Label/Size、$Type/Label/Line、fill $text-secondary
   - frame「chev」20×20 置中：lucide chevron-right 16×16 fill $text-muted
7. 8 列內容（依序；徽章顏色：缺中文＝$warning-tint 底／$warning-text 字；中文、繁中＝$success-tint／$success-text）：
   1) 執法記錄｜2026 · 恐怖｜缺中文｜6.0｜4K · H.265 · DTS
   2) 鏢人：風起大漠｜2026 · 126 分 · 動作｜不知道（灰字）｜8.1｜—
   3) t＝「沒有找到這部片的資料」，t 右邊同一行加「失敗」藥丸（$error-tint 底、$error-text 字、cornerRadius $radius-pill、padding [$Space/2xs, $Space/xs-plus]、$Type/Label/Size 600）——把 t 和藥丸包進 horizontal frame「t-row」gap $Space/sm；m＝「[ItachiUchiha] Chainsaw Man – The Mov…Reze-hen.mkv」改用 $Type/Family/Mono、$Type/Label/Size、fill $text-muted｜不知道（灰字）｜評分欄留空（只放 frame 不放內容）｜—
   4) 牧師的最後誘惑｜2018 · 劇情｜中文｜7.0｜4K · H.265 · DTS
   5) 阿凡達：火與燼｜2025 · 科幻｜繁中｜7.6｜4K · H.265 · E-AC-3
   6) 捍衛任務：復仇芭蕾｜2025 · 動作｜繁中｜7.3｜4K · H.265 · AAC
   7) F1電影｜2025 · 動作｜繁中｜7.8｜1080p · H.265 · AAC
   8) 四海好傢伙｜1990 · 劇情｜繁中｜8.5｜4K · H.265 · AC-3
   第 1 列的 row 填色 $bg-secondary（hover 示意），其他列無填色。
8. 檢查：table 不能超出 KuyJc；8 列放不下時讓 A4p-D 的 KuyJc 照原樣裁切（跟原稿一樣只露出畫面內的列）。

【B. 新稿 A4p-M（手機列表）】
1. 複製 A3p-M（h1v1U6），放在群組 JUkwx 裡 x=19000、y=10993，frame 改名「A4p-M」。
2. 複製標題 cap-A6p-M（qItRC），放在 x=19000、y=10948，文字改成「A′4 · 列表檢視（手機）」。
3. 在複製出的 A4p-M 裡：
   - topbar 的 count 文字改成「1,284 部 · 37 部缺中文」，「· 37 部缺中文」用 $accent-text（可拆成兩個 text）。整個 title-row 不得把右邊 44×44 的篩選鈕擠出去。
   - 刪掉 active-chips。
   - 刪掉 grid 的全部子節點，把 grid 改名「list」：vertical、gap $Space/none、padding [$Space/sm, $Space/md]。
   - list 裡放 frame「table」：vertical、width fill_container、gap $Space/none、cornerRadius $radius-md、clip、1px 內側描邊 $border-subtle。沒有欄名列。
   - 每列前放 rectangle「div」（第 1 列前不用）：width fill_container、height 1、fill $border-subtle。row：horizontal、width fill_container、gap $Space/md、padding [$Space/sm-plus, $Space/md]、alignItems center。子節點：
     · thumb 40×60、cornerRadius $radius-sm、clip
     · info：vertical、width fill_container、gap $Space/2xs：t（$Type/Body/Size、$Type/Body/Line、600、$text-primary，單行）、m（$Type/Label/Size、$text-secondary）、line3（horizontal、gap $Space/sm、alignItems center：字幕徽章或「不知道」灰字，再接 star＋評分，樣式同 A-6）
     · chev：lucide chevron-right 16×16 $text-muted
   - 放 A-7 的前 7 列（同樣內容，不放規格）。第 3 列：t＝「沒有找到這部片的資料」（單行，放不下就截斷），m＝「[ItachiUchiha] Chain…Reze-hen.mkv」（Mono、$text-muted），line3＝「失敗」藥丸＋「不知道」灰字。
4. 檢查：每列寬 ≤ 358，沒有文字超出 row；tabbar 不動。

【C. 規格說明 spec-note-a4p】
複製文字節點 GMxQV（spec-note-dsr-1b-b），放在 x=17760、y=11897，改名 spec-note-a4p，文字換成：
「A4p（列表檢視，2026-10-19 改版）：列表是『修字幕清單』，字幕狀態緊跟在片名後面。桌機是有欄名的表格：片名（固定寬）→ 字幕 → 評分 → 規格（灰字一段，寫 4K／1080p，不寫原始寬高）→ ›；多出的空間留在最右邊，不夾在片名和狀態中間。整個列表一道外框、列與列一條髮絲線，列本身不加框（同一層只准一道框）。字幕欄沿用海報同一套徽章；不知道就寫灰字『不知道』，不留白。『中文』提示框：字幕軌沒標示繁簡。標題計數後的『N 部缺中文』點了等於套用『缺中文字幕』篩選，0 部時不顯示。排序多一個『字幕狀態』：缺中文 → 無字幕源／未翻譯／已略過 → 不知道 → 簡中 → 中文 → 繁中。比對失敗的列：第一行『沒有找到這部片的資料』＋失敗徽章，第二行檔名從中間截斷、保留副檔名；整列是一個連結，點進去是 B10p（手動選片在那裡）。手機 <640：藏欄名與規格，字幕徽章和評分在第三行，字幕狀態在任何寬度都不藏。」

完成後不要自己匯出截圖。
```

## Dev Agent Record

**2026-10-19 Alexyu 跑完 inline agent（三段都做）；Sally MCP 唯讀複審通過。**

- 文案逐字對：A4p-D 8 列、A4p-M 7 列、欄名、「· 37 部缺中文」、「字幕狀態」、失敗列兩行，全部與提示詞一致。
- 全部用變數：A4p-D 掃描 fill／fontSize／gap／padding／cornerRadius，沒有寫死的數值。
- `ctx.problems`：只有 A4p-D 的 `table` 被 `list` 裁切約 8px——這是提示詞第 8 點要的「照原樣裁切」，追認。
- **追認一個偏差：** 手機列寬是 366，不是 AC 寫的 358。是我的提示詞把 list 左右 padding 寫成 `$Space/md`（12），inline agent 照做；不溢出。**實作照 A3p-M 網格的 16px 頁邊**（列寬 358），不照稿上的 12px。
- 小差異交給實作：星號在稿上是外框星，程式碼現行是實心（`fill-[var(--warning)]`），維持程式碼。
- 新節點：A4p-M `EkB8g`（`SCREENS` 已加 `a4p-m`）、`spec-note-a4p`。截圖只 stage `a4p-d.png`、`a4p-m.png`；`check-design-refs.py` 通過（385 個檔頭）；token 快照 raw 全 0。
