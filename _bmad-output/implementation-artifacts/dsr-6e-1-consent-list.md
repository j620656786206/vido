# Story DSR.6e-1：產生字幕的候選清單對齊設計稿——金額不再穿狀態色，少一個金額也不會讓整個畫面壞掉

Status: done

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As the person who opens 產生字幕 and decides which films to spend money on,
I want 清單上的每個金額都是同一種顏色、每一列長得跟設計稿一樣、後端少給一個金額時畫面照樣打得開,
so that 我看到的顏色只在說「狀態」（不是在說「這筆比較貴」），而且不會因為一筆資料有缺就整個對話框炸掉、什麼都選不了。

## Context

`dsr-6e`（同意流程 F14–F20）拆出來的**第一塊**。

⚖️ **SM 裁定（2026-09-20，拆單）**：建單稽核在 F14–F20 找到 30 多處落差，外加一張 P1、三張設計稿漂移單子與兩個裁定；`CandidateListPanel.tsx` 一支就 1,155 行、14 處 13px。一張做完會超過 `dsr-6d-c-2`（目前最大的一張）。依 `feedback_split_oversized_stories`「該拆就拆」，**切分線選版面**：

| 單子 | 範圍 |
| --- | --- |
| **`dsr-6e-1`（本張）** | 候選清單 F15-D-v2 `pwMzT`／超出上限 F18-D-v2 `zBik1`（＋規格 `VPT7l`）→ `CandidateListPanel.tsx`、`consentRows.ts`、`consentSelection.ts`；同意對話框的**外殼**（`GenerationConsentView.tsx` 的 `DialogContent`）；P1 `disc-2026-09-candidate-usd-missing-guard` |
| `dsr-6e-2` | 分析中 F14 `nBT3M`、金額確認 F16 `gmOt6`／F19 `KThbY`、空狀態 F20 `D7MOm`、掃描完成入口 F17 `I3Wb0p`（稿）→ `AnalysisProgressPanel`、`ConfirmGenerationDialog`、`ModelPicker`、`ConsentEmptyState`。**依賴本張**（外殼寬度） |

P1 放本張：它壞的是清單（每一列、每個小計都經過 `candidateUsd`），而且它是 P1，要先出貨。

⛔ **手機稿（F15-M `fdu4y`、F16-M、F19-M）屬 `dsr-6f`**。本張的程式碼是桌機與手機共用的，所以手機的畫面**會跟著變**（字重、金額顏色、徽章顏色）——這是預期的，但手機的**排版**（`@max-xl:` 那一組）一律不動。
⚠️ **驗收基準是 `.pen` 節點值**（文字逐字、字級變數、顏色變數、間距）。PNG 只是參考。

### 🔴 建單時查到的事（main `c4d34187`；行號皆為現況）

1. **P1：少一個金額，整個對話框炸掉。** `candidateUsd`（`consentSelection.ts:269-281`）直接 `roundUsd(… c.estimatedUsd)`；`estimatedUsd` 是 `undefined` 時 decimal.js 丟 `[DecimalError] Invalid argument: undefined`，每一列、每個群組小計、摘要列、頁尾、F18 的可完成部數都經過它——**沒有降級路徑，只有 render 失敗**。（sprint 條目寫的 `:262` 是舊行號。）今天的後端 `estimated_usd` 沒有 `omitempty`（`generation_candidates.go:246`），所以要靠「舊後端／資料不完整／e2e mock 少一欄」才會觸發；但金額是這個畫面唯一的承諾，一欄缺值就讓使用者什麼都做不了，這不是可以賭的地方。
2. **P1 另一半：換了模型，有些列偷偷還是舊模型的價錢。** `candidateUsd` 在「選中的模型的 `perCandidate` 沒有這一列」時，**靜默退回** `estimatedUsd`（＝伺服器預設模型的價）。後端 `generation_candidates.go:834-846` **只替可寫入的列報價**，所以今天**每一列「資料夾無法寫入」的片，在你選 Haiku 時顯示的仍是 Sonnet 的價錢**，而且畫面上看不出來（設計稿 `DesXa` 在那一列畫的也是 $0.31）。後端刻意不替它們報價（sub-6-1：「報了只會灌大每個模型的總額」），所以那一格**任何數字都是錯的**。
3. **金額穿了狀態色**（違反 `DESIGN.md:308` 金錢是事實規則，⚖️ Alexyu 2026-09-10 裁定 B）：列的金額 `:369-374` 依路線穿青碧／赭；摘要列 `:768-774` 與頁尾 `:1081-1087` 超過上限時翻成赭；砍線文字 `:407-408` 整句（含金額）是赭。設計稿的每一個金額都已經是 `$text-primary`（F15 `FDay3`／`X7T8UI`／`XyNOv`、F18 `h8P1C9`／`CJDcb`／砍線 `lQGAX`）。
4. **路線分類穿了狀態色**（dsr-6 原單子的 ③）：列的路線徽章 `:357-367`（抽取＝青碧、語音辨識＝赭）、群組標題的路線徽章 `:541-552`、篩選 chip 的費用標記 `:740-749`（僅翻譯費＝青碧、付費＝赭）。「抽取」「語音辨識」是**這一列走哪條路**，不是「發生了什麼」——`DESIGN.md:300`「狀態色不得被挪用為強調、裝飾或**分類**」；2026-09-10 Alexyu 已經為同一類東西（TechBadge 的 H.265／HDR／繁中）裁定過「一律中性」（`DESIGN.md:716-718`）。**設計稿也是錯的**：列徽章 F15 `FYu5G`／`u5guM`／`Qofei`／`RkJNq`／`w9oXL`、F18 `rXaYQ`／`FuzMU`／`Eo0FX`／`k6Pfd`／`KahBc`，群組徽章 `Jo8Lz`／`H38QvC`／`Sfdxh`（＋手機 `dYPUR`），chip 標記 F15 `J3kT6`(`p0hBTX`)／`mw17e`(`tpVXR`)、F18 `L2vpy`(`ts6OZ`)／`iak8P`(`AMl15`)、**手機 F15-M `BZwGh`(`s2yz6N`)／`Ouuhe`(`E05nyI`)**——這三組 chip 標記還寫死 **10px**（字階沒有這一階）而且用**基礎語意色當文字**（`$success`／`$warning`，`DESIGN.md:285` 明文禁止）。⚖️ **Alexyu 2026-09-20 裁定：改中性**（同 TechBadge）。第四處同類（F16／F19 的「品質 A」青碧徽章）在 `dsr-6e-2`——sprint 條目說的「全 app 共 4 處」就是這四處：列的路線徽章、群組的路線徽章、chip 的費用標記、模型的品質徽章。
5. **F18 的稿還停在舊列型**（`backlog-f18-pen-row-format-drift`）：海報是 40×60 空灰塊、副標是 14px `$text-secondary` 而且沒有片長、`FPqBM` 的副標還是被 Sally 殺掉的那句「片長未知，以 45 分鐘估算」、chip 標記寫「**免費**」（`ts6OZ`，§5-sexies 明文不准）、沒有搜尋／排序列（程式碼一直都有）。
6. **片名字級三邊各說各話**（`backlog-consent-row-title-tier-drift`）：程式碼 `:316` 是 `text-sm`（14/400）；設計稿在 2026-09-10 的字階整理後已定案——桌機 F15／F18 是 **BodyLg 16／600**（`lnmY6` 等），手機 F15-M 是 **Body 14／600**（`u3Wf2` 等）。那張單子等的「裁定」已經在稿上了，剩程式碼。
7. **群組標題**（`backlog-f15-group-header-pen-code-drift`）：(1) 路線徽章——設計稿**已經補上了**（`Jo8Lz`／`H38QvC`／`Sfdxh`），這一半結案；(2) 單季影集——稿上是「怪奇物語 · 第 4 季」（`yQgKh`、手機 `Vj22Z`），程式碼只寫「怪奇物語」（`consentRows.ts:346`，在 `pushSeriesRows` `:331-377` 裡；季標籤的 `seasonLabel` 在 `:380`）。多季影集的兩層結構稿上沒畫，程式碼是唯一的規格。
8. **對話框寬 768，稿上 960**（`GenerationConsentView.tsx:525` `sm:max-w-3xl`；F15 `DBPu5`、F18 `ckLed`、F20 `WqnVQ` 都是 960）。F14 `h60Flj` 是 560——但 F14 與 F15 是**同一個 `DialogContent`**（`:518-623`，phase 切換不重新掛載），照稿走就是「分析跑完，對話框在使用者眼前從 560 撐到 960」。見建單裁定 2。
9. **字級**：本張範圍內 14 處 `text-[13px]`（`:371`、`:534`、`:732`、`:765`、`:807`、`:834`、`:854`、`:902`、`:911`、`:996`、`:1016`、`:1079`、`:1104`、`:1107`）。
10. **列的幾何**：`:263` `rounded-[var(--radius-lg)] … bg-[var(--bg-secondary)] px-3.5 py-3`；稿（`u4sSoJ`）是 `$radius-md`、**沒有底色**（只有 1px `$border-subtle` 框）、padding [10,12]。路線徽章 `:360` 是 `radius-sm px-2 py-0.5` 400；稿（`FYu5G`）是 **pill**、padding [4,10]、Label **600**。
11. **群組標題幾何**：`:491-494` `radius-lg px-3.5`；稿（`LupMK`）`$radius-md`、padding [8,12]、gap 10。
12. **控制列**：搜尋框 `:795` 是 `bg-tertiary radius-sm`、圖示 16、13px；稿是 `Component/SearchInput`（`6MxLT`：`$bg-secondary`＋1px `$border-subtle`、`$radius-md`、padding [10,16]、圖示 18、Body）。排序 `:824` 同樣是 `bg-tertiary radius-sm`；稿是 `Component/SortDropdown`（`955EZ`：`$bg-secondary`＋框、`$radius-md`、padding [8,12]），而且**稿上的標籤覆寫 `lJppJ` 寫死 13**（稿要改）。篩選 chip `:732` 是 `radius-sm`、13px、選中 `--accent-subtle`；稿是 `Component/FilterChip`（`jD7gF`：**pill**、Label 12／500、padding [6,12]），選中覆寫 `$accent-tint`＋`$accent-text`（`J51aSC`）。
13. **工具列**：`:853` `gap-x-3`；稿 `U2UYW3` gap 16。「清除選取」`:911` 是 `--accent-text`；稿 `HBSgk` 是 **`$text-secondary`**／500。
14. **頁尾**：「已選 N 部 · 預估」`:1079` 是 13px／400；稿 `RQVra` 是 Body 14／**600** `$text-primary`。預算欄 `:1107` 是 `bg-secondary radius-sm px-2 py-1.5` 13px；稿 `cLlZL` 是 **96×36**、`$bg-primary`、`$radius-md`、padding [0,12]、Body mono。「開始產生」`:1142` 是 `px-6 font-medium`；稿是 `Component/Button/Primary`（`otvKh`：14／**600**、padding [8,20]）。
15. **F18 橫幅**：`:1013` 是縮排的圓角卡片（`mx-6 mb-2 … rounded p-3`、`CircleAlert`、13px）；稿 `RrCEy` 是**貼齊兩側的整條**（padding [10,24]、無圓角）、`triangle-alert`、Body 14 `$text-primary`。
16. **設計稿把規格註記塞進了畫面裡**：F15 對話框 body 裡的 `UF8ic`「群組 checkbox 為三態；影集預設收合，搜尋命中時自動展開。」長得像 UI 文字（違反 `feedback_pencil_spec_standalone_screen`：規格註記要獨立，不塞進 mockup）。
17. **`:295-296` 的註解會變成假話**：它說「手機上路線徽章收起，因為金額的顏色已經說了抽取（綠）還是語音辨識（橘）」——第 3 點改完之後金額沒有顏色了。手機上的路線仍在副標的文字裡（「內嵌英文字幕 → 翻譯」／「無文字字幕軌 → 語音辨識 + 翻譯」），所以行為不用改，**註解要改**。
18. **原生 checkbox**（16px、`accent-color`）vs 設計稿的 `Component/Checkbox/*`（20×20、`$radius-sm`、實心泥金）——全 app 的原生 checkbox 都是這樣，不在本張改（見 Discovery Triage ③）。

### 設計稿節點

| 代號 | 節點 | 本張要看的 |
| --- | --- | --- |
| F15-D-v2 | `pwMzT`（對話框 `DBPu5` 960、`$border-subtle` 1px、`$radius-lg`） | 標題列 `geiGH`（h56、padding [0,12,0,24]）／標題 `G6vQEv`；body `RTmxa` [16,24] gap 12；摘要 `JVGKf`（gap 2）：`njQZt` gap 6、`v30eJ` Body `$text-secondary`、`FDay3` Body 600 mono `$text-primary`、`y4LQ5` Label `$text-muted`；搜尋列 `UmqHL` gap 8：`g6z5aY`（SearchInput h44）、`ky1ns`（SortDropdown h44，標籤 `lJppJ`）；chip 列 `NiXrE` gap 8：`J51aSC`（選中）／`MQwxG`＋標記 `J3kT6`(`p0hBTX`)／`Lm25e`＋標記 `mw17e`(`tpVXR`)；工具列 `pB5IP`：`FbSbF` gap 8（`tR9dk`、`u4xtZr` Body `$text-secondary`）、`U2UYW3` gap 16（`M2xP4` Body 500 `$accent-text`、`HBSgk` Body 500 `$text-secondary`）；清單 `T3rfXQ` gap 8；群組標題 `LupMK`／`Fmsn2`／`o7L2K`（[8,12] gap 10 `$bg-tertiary` `$radius-md`；標題 `DKr3p`／`EGhCw`／`yQgKh` Body 600；徽章 `Jo8Lz`／`H38QvC`／`Sfdxh` [2,6] `$radius-sm` Label 400；已選 `y0GMzT`／`bjP6Y`／`i1GmG` Label mono `$text-muted`）；列 `u4sSoJ`（沙丘）／`bvqel`／`S3aO4`（不可寫入，opacity 0.7）／`hxctk`（未匹配＋首字海報 `Nm2A6`）／`Jht9P`（S4E7，`enabled:false`）：[10,12] gap 12、`$border-subtle` 框、`$radius-md`、無底色；海報 38×54；文字欄 gap 4；片名 BodyLg 600；副標 Label `$text-muted`；右側 gap 10：路線徽章 pill [4,10] Label 600、金額 Body 600 mono `$text-primary`、不可寫入標 `mm34N`；說明 `LjZL8`；頁尾 `rrOWa` [14,24]：`RQVra`／`XyNOv` Body 600、`cMziP` Label muted、預算 `mTADq` Body `$text-secondary`＋欄 `cLlZL`、提示 `twwOH`、按鈕 `iK0BE`（Primary h44） |
| F18-D-v2 | `zBik1`（對話框 `ckLed` 960） | body `MqUky`；chip `i2hwp`／`S8usm`＋`L2vpy`(`ts6OZ`「免費」)／`uG6xH`＋`iak8P`(`AMl15`)；列 `cEsaG`／`LZiaq`／`mMhPj`／`t20SOB`（opacity 0.6）／`FPqBM`（opacity 0.6）與它們的 `poster`、`sub`、`route-badge`；砍線 `iQY76`（`mTXmS`／`RlUZZ` `$warning` 1px、`lQGAX` Label `$text-primary`）；橫幅 `RrCEy`（[10,24] gap 8 `$warning-tint`、`Jx2Pe` triangle-alert 16、`tHo1i` Body `$text-primary`）；頁尾 `A9foYw`（預算欄 `fxVRp` 框 `$warning`）、按鈕 `vSsLp`「開始產生（將於上限暫停）」 |
| F18 規格 | `VPT7l`（f18-spec-err） | 開始失敗訊息 `TBbai`（外 [8,24]）／`C2mDb`（[10,12] gap 8 `$error-tint` `$radius-md`、`bA4UX` circle-alert 16、`Wuju3` Body `$error-text`） |
| F15-M-v2 | `fdu4y`（**只改顏色**，排版歸 dsr-6f） | 群組徽章 `dYPUR` |
| 母版 | `Component/SearchInput` `6MxLT`、`Component/SortDropdown` `955EZ`、`Component/FilterChip` `jD7gF`、`Component/Button/Primary` `otvKh` | 見上 |
| 規格群組 | Flow F `JzmvC`，既有註記 `c4FIoB`（`note-gen-capability`） | AC #1 的註記 append 在這裡（dsr-6b／6d-b 的做法） |

字階（`DESIGN.md:355-358`）：H4 18／1.5、BodyLg 16／1.625、Body 14／1.625、Label 12／1.5。**行高不在本張**（`text-sm` 自帶 1.429，要 1.625 得寫 `leading-relaxed`——全 app 一起處理，追蹤於 `disc-2026-09-type-scale-even-migration`）。

---

## Acceptance Criteria

1. **設計稿先改（Pencil MCP；動手前每個節點再 `Get` 一次）。順序照這裡寫的走**——`lJppJ` 要在複製 `UmqHL` **之前**改，否則 F18 複製到的是還沒修的 13px 標籤。
   - **排序標籤先修**：F15 `lJppJ`（`ky1ns` 的覆寫）`fontSize: 13` → `$Type/Body/Size`，補 `lineHeight: $Type/Body/Line`；手機 F15-M 的同一個覆寫 `v05oXD`（也是 13）一起改。
   - **路線與費用標記改中性**（⚖️ Alexyu 2026-09-20 裁定）：列的路線徽章（F15 `FYu5G`／`u5guM`／`Qofei`／`RkJNq`／`w9oXL`、F18 `rXaYQ`／`FuzMU`／`Eo0FX`／`k6Pfd`／`KahBc`）與群組徽章（`Jo8Lz`／`H38QvC`／`Sfdxh`、手機 `dYPUR`）：底 `$bg-tertiary`、字 `$text-secondary`。chip 標記（F15 `J3kT6`(`p0hBTX`)／`mw17e`(`tpVXR`)、F18 `L2vpy`(`ts6OZ`)／`iak8P`(`AMl15`)、手機 F15-M `BZwGh`(`s2yz6N`)／`Ouuhe`(`E05nyI`)）：底 `$bg-secondary`（chip 本身是 `$bg-tertiary`，同色會看不見）、字 `$text-secondary`、`fontSize: $Type/Label/Size`＋`lineHeight: $Type/Label/Line`（10 → 12）、`fontWeight: 500`。
   - **徽章一律藥丸**（`DESIGN.md:566-568` 的 2026-09-10 圓角新規則：藥丸＝資料標記，`radius-sm` 只給「不可互動的小色標」）：上面那幾顆群組徽章與 chip 標記 `cornerRadius` → `$radius-pill`（chip 標記順手取代寫死的 `100`）；同一個叢集裡的 `mm34N`（資料夾無法寫入）與 `otBhu`（未匹配 chip）也一起改藥丸，免得同一列出現兩種形狀。列的路線徽章本來就是藥丸。
   - **F18 的「免費」**：`ts6OZ` 內容改「**僅翻譯費**」（與 F15 `p0hBTX` 相同；§5-sexies）。
   - **F18 的五列改成 F15 的列型**（建單裁定 8；同一個 `CandidateRow`）：海報 40×60 空灰塊一律換成 38×54 的**首字海報**（照 F15 `Nm2A6` 的形狀：`$bg-tertiary`、`$radius-sm`、置中、H4 `$text-secondary`；字是片名第一個字：怪／怪／全／教／星）；副標 `M2pQt`／`KxYWD`／`f7PE3l`／`I1OKTn`／`q65gXI` 改 Label `$text-muted`，內容補上片長，S4E7 用 F15 的「無文字字幕軌 → 語音辨識 + 翻譯 · 1 小時 38 分」、全面啟動用 F15 的「… · 2 小時 28 分」，S4E8「… · 1 小時 25 分」、教父「… · 2 小時 55 分」、星際效應「無文字字幕軌 → 語音辨識 + 翻譯 · 片長未知（估 45 分）」並照 F15 `MSJfU` 把片名包進 titleRow 補上「未匹配」chip。
   - **F18 補上搜尋／排序列**（在 `lJppJ` 修好之後）：`Copy("UmqHL", "MqUky")` 後 `Move` 到 index 1（摘要之後、chip 之前），與 F15 同位置。
   - **不可寫入的列不顯示金額**（建單裁定 7）：F15 `DesXa`（全面啟動那一列的「$0.31」）與**手機 F15-M `YqUYl`**（同一部片、同一列 `eODso`）`enabled:false`。
   - **規格註記搬出畫面**：刪掉 F15 body 裡的 `UF8ic`，在**規格群組 `JzmvC` 裡新增一個中性的 text 節點**（像 `dufQr` 那顆 spec-note：`$Type/Label/Size`＋`$Type/Label/Line`，`fill: $text-muted`——⛔ 不要學 `dufQr` 的寫死 `#888888`；先 `Get("dufQr")` 讀它的 bounds，把新節點放在它下方 40px、同寬），內容：「F15 群組：checkbox 三態；影集預設收合，搜尋命中時自動展開。單季影集的標題寫『劇名 · 第 N 季』；多季影集是兩層（影集列＋縮排的季列）。路線徽章與 chip 的費用標記是分類，一律中性；金額一律 `$text-primary`。沒有報價的列（後端沒給 `estimated_usd`）選不到，右側寫『無法估價』；資料夾無法寫入的列不顯示金額（後端不替它們報價）。」⛔ **不要 append 進 `c4FIoB`**：那顆是赭色的「BE 能力邊界」盒，而 `DESIGN.md:253` 明文說設計稿的後設註記不得穿赭色（見 Discovery Triage ③ 的 `disc-2026-09-pen-meta-notes-wear-ochre`）。
   - 新節點一律綁變數；每張改完 `ctx.problems` 掃裁切；存檔後確認 ` M ux-design.pen` 並讀磁碟檔確認落盤（`feedback_verify_pen_saved_before_commit`）；匯出後只 stage `flow-f-subtitle-v2/{f15-d-v2,f18-d-v2,f15-m-v2}.png` ＋ `_bmad-output/pen-tokens.json`（若變），其餘重繪雜訊還原。**若 `f18-spec-err`、`f14-d-v2` 或元件庫變了，代表動到了不該動的節點——回頭檢查。**
   - ⛔ F14 `h60Flj` 的寬度**本張不改稿**（F14 的重畫在 `dsr-6e-2`）。

2. **P1：少一個金額不再讓畫面壞掉，也不再偷換模型的價錢（🔴 #1、#2、#15）。**
   - `consentSelection.ts` 新增 `isPriced(c)`：`typeof c.estimatedUsd === 'number' && Number.isFinite(c.estimatedUsd)`；新增 `isSelectable(c) = isWritable(c) && isPriced(c)`。**選得到的列＝可寫入而且有報價**（與模型無關，所以換模型不會讓已選的列消失）。
   - `candidateUsd(c, prices)` 改回傳 `number | null`（三條規則，順序固定）：
     1. `!isWritable(c)` → **一律 `null`**。後端不替不可寫入的列報價（`generation_candidates.go:833-843`），所以那一格任何數字都是別的模型的價或假設。⚠️ 這條要放最前面，否則「有沒有載到模型清單」會改變同一列的排序位置（`consentRows.ts:81-89` 的金額排序）。
     2. 有 `prices`（某個模型的 `perCandidate` 正在生效）→ 該列在表裡而且是有限數字才回傳它（`roundUsd` 後），**否則 `null`**——⛔ 不准再退回 `estimatedUsd`（那就是 🔴 #2 的偷換）。
     3. 沒有 `prices`（舊後端、或沒有任何模型可選）→ `isPriced(c)` 才回傳 `roundUsd(c.estimatedUsd)`，否則 `null`。這時批次送出的 `model_id` 是 `''`＝伺服器預設模型，而 `estimatedUsd` 正是那個模型的價，所以數字與實際收費是同一個來源。
   - `consentSelection.ts:255-260`（`ModelPrices` 的 JSDoc）現在寫著「a row missing from it … `estimatedUsd` stands」——那正是本 AC 禁止的行為，**要一起改掉**，否則下一個讀檔案的人會照著註解把它改回去。
   - `sumSelected`（`:610-620`）遇到 `null` 跳過（照涵蓋規則不可能發生，但不要讓它變成 `NaN`）；金額排序（`consentRows.ts:81-89`）遇到 `null` **一律排在最後**（兩個方向都是），⛔ 不要讓 `cmpUsd(null, …)` 進 decimal.js。
   - 所有「能不能選」的地方從 `isWritable` 換成 `isSelectable`：`consentSelection.ts:52`（`defaultSelection`）、`:89`（`selectableIds`）、`:110`（`visibleSelectableIds`）、`:313`／`:321`（`computeTotals`）、`:617`（`sumSelected`）；`GenerationConsentView.tsx:206`（預選交集）、`:498`（送出）。⛔ **`:668`（`modelChoices` 的片長分母）維持 `isWritable`**——後端的 `estimated_minutes_by_model` 是對**所有可寫入的列**加總的（`generation_candidates.go:834-846`，`:663-664` 的註解就是在說這件事），分母換掉會讓「約 N 分鐘」系統性偏大。`:415` 的 `writableIdSet` 是用 `selectableIds()` 算的，會自動跟著改；順手改名 `selectableIdSet`（`:419`、`:427`、`:433`、`:443` 一併）。⛔ `CandidateRow` 的 `data-writable`（`:260`）與「資料夾無法寫入」標**仍只看 `isWritable`**——兩個原因要分開說。
   - `modelChoices`：有 `estimatesByModel` 時，**只提供 `perCandidate` 對每一個 `isSelectable` 的候選都有有限數字的模型**（後端本來就保證這件事；不保證時寧可不給選，也不要讓那個模型的總額混進別的模型的價錢）。🚨 **預設模型沒過關就整組回 `[]`**——`effectiveModelId`（`GenerationConsentView.tsx:377-380`）在預設模型不在清單時會退到 `choices[0]`，那會讓整份清單**沒人按過就改用另一個模型的價**（畫面上還不會有「（預設）」那一列）。全部回 `[]` 之後，`prices` 不帶、`model_id` 送 `''`，一律照伺服器預設模型計價（與上一條同一個來源）。舊後端（沒有 `estimatesByModel`）的路徑不變。
   - `ConsentTotals` 新增 `unpricedCount`（可寫入但沒有報價的列數）；`unwritableCount` 改成**只數** `!isWritable`（今天是 `candidates.length - selectableCount`，改完會把沒報價的也算進去）。防禦用：`computeTotals` 遇到**已選**而 `candidateUsd` 為 `null` 的列（照上面的規則不該發生），**只記在 `unpricedSelectedCount`**，不進 `selectedCount`、不進抽取／語音辨識的計數與金額、不進 `feasibleCount`／`pausedIds`／`estimatedRowCount`——一列沒有價錢就不能參與任何「這批要花多少」的說法。
   - ⚠️ `ConsentTotals` 是**型別化**的：`ConfirmGenerationDialog.spec.tsx:9-28` 的 `baseTotals`、`-gallery.fixtures.tsx:554` 的 `confirmTotals()` 與 `:4820-4860` 兩個手寫 totals 都要補上兩個新欄位（`unpricedCount: 0, unpricedSelectedCount: 0`），否則 `web:typecheck` 紅。**這是「不要碰 `ConfirmGenerationDialog`」的唯一例外，而且只動它的 spec 檔、不動元件。**
   - 畫面（**一條規則決定金額畫不畫**：`candidateUsd(candidate, prices) !== null` 才畫金額那一格；「無法估價」標的條件是 `isWritable(c) && !isPriced(c)`——所以不可寫入的列是「沒有金額、沒有無法估價標、只有既有的『資料夾無法寫入』」）：
     - 可寫入但**沒有報價**的列：checkbox `disabled`、金額那一格**不畫**、叢集裡加一顆與「資料夾無法寫入」同樣式的標（`--error-tint`／`--error-text`，Label，藥丸——見 AC #4），文字「**無法估價**」，`title="後端沒有給這一部的估價，所以不能選。重新分析一次通常就會補上。"`，`data-testid="consent-row-unpriced-{id}"`，列 `opacity-70`（與不可寫入相同）。
     - **不可寫入的列不畫金額**（建單裁定 7；🔴 #2 的根）。
     - 工具列在 `unwritableCount` 那句旁邊，`unpricedCount > 0` 時加「（N 部無法估價）」，同樣式，`data-testid="consent-unpriced-count"`。
     - `unpricedSelectedCount > 0` 時「開始產生」停用，並顯示原因（`DESIGN.md:314`：停用＋原因）。⚠️ 預算提示那一行今天只在 `budgetInvalid || !overBudget` 時才 render（`:1131`）——**原因那一行要獨立於那個條件**（`unpricedSelectedCount > 0` 就顯示，而且蓋過另外兩句）：「有 N 部沒有報價，請重新分析」，`data-testid="consent-unpriced-hint"`。
   - ⛔ 不准把缺值補成 `0`：`$0.00` 在這個畫面是一個讀數（「這一部不花錢」），用它代替「不知道」就是在說謊（`DESIGN.md:320`）。
   - ⚠️ **這一條與 `DESIGN.md:318` 的字面有出入**：那裡寫「畫面上唯一合法的無金額狀態只有兩個：估價請求還在路上、估價請求失敗（停用＋原因）」。本張多出兩種「沒有金額」：一整列沒有報價（＝這一列的估價失敗，**停用＋原因**，符合第二種的精神）、以及不可寫入的列（後端刻意不報價，那一列本來就不會跑）。AC #4 改 `DESIGN.md` 時一併把這兩種寫進去，不要讓規則與出貨的畫面對不上。

3. **金額不穿狀態色（🔴 #3；⚖️ Alexyu 2026-09-10 裁定 B，`DESIGN.md:308`）。**
   - 列的金額 `:369-374`：一律 `text-[var(--text-primary)]`，連同字級改 `text-xs font-semibold @xl:text-sm`（桌機 Body 600、手機 Label 600——稿 F15 `X7T8UI` 是 Body、F15-M `Z2Tnx` 是 Label，與片名同一個 container query 斷點）。
   - 摘要列 `:768-774`、頁尾 `:1081-1087`：拿掉 `overBudget ? --warning-text : …` 的分支，一律 `--text-primary`。
   - 砍線 `:407-408`：文字改 `text-[var(--text-primary)]`（稿 `lQGAX`）；兩側的線維持 `bg-[var(--warning)]`——**狀態押在線上，數字保持中性**（`DESIGN.md:310` 那一段的原則）。
   - 超過上限的訊號仍然完整：赭色橫幅、預算欄的赭色框、砍線、按鈕上的「（將於上限暫停）」。⛔ 不要為了「補回訊號」把金額又塗回去。
   - **把會變成假話的註解與測試名一起改掉**（🔴 #17）：`:295-296`（手機上的路線由副標文字承載，不是金額的顏色）、`:10-11` 的檔頭（「the extract route shows its small translation fee (success green)」）、`:553-558` 的群組註解、`CandidateListPanel.spec.tsx:735` 的測試名（「the amount colour already says the route」）與 `:142`（「warning amounts」）。⛔ 測試名要跟著行為改，不要留著一個名字說綠色、身體斷言中性的測試。

4. **路線與費用標記改中性（🔴 #4；⚖️ Alexyu 2026-09-20 裁定）。**
   - 列的路線徽章 `:357-367`：`bg-[var(--bg-tertiary)] text-[var(--text-secondary)]`，形狀改稿上的藥丸：`rounded-full px-2.5 py-1 text-xs font-semibold`。⛔ `hidden … @xl:inline` 兩個 class 保留（spec `:739-740` 在守）。
   - 群組標題的路線徽章 `:543`／`:548`：`rounded-full bg-[var(--bg-tertiary)] px-1.5 py-0.5 text-[var(--text-secondary)]`（稿 `Jo8Lz` 的 padding [2,6]、Label 400；形狀依 AC #1 改藥丸）。
   - chip 的費用標記 `:741`／`:746`：`rounded-full bg-[var(--bg-secondary)] px-1.5 py-0.5 text-xs font-medium text-[var(--text-secondary)]`；文字「僅翻譯費」「付費」不變。
   - 同一個叢集裡的另外兩顆跟著改藥丸（AC #1 的稿也改了）：「資料夾無法寫入」`:352` 與「未匹配」`:328` 的 `rounded-[var(--radius-sm)]` → `rounded-full`；顏色不動。新的「無法估價」標比照「資料夾無法寫入」。
   - 「資料夾無法寫入」與新的「無法估價」**維持硃砂**——那是「壞了」，是真的狀態。F18 橫幅**維持赭**：你要了 96 部、實際只會跑約 18 部，這正是赭的定義（`DESIGN.md:250`「使用者要求了某件事，而它沒有在發生」），不是「按下去會怎樣」的事前警語（`:252`）。
   - `DESIGN.md` 兩處要改：
     - 在 TechBadge 那段裁定（`:716-720`）之後補一句：候選清單的路線徽章（抽取／語音辨識）、群組標題的路線徽章、chip 的費用標記與 F16／F19 的品質徽章同屬分類，一律中性（⚖️ Alexyu 2026-09-20；四處的第四處在 `dsr-6e-2`）。
     - `:318` 那句「畫面上唯一合法的無金額狀態只有兩個」補上第三與第四種：**一整列的估價缺席**（停用那一列＋寫出原因，就是同一條規則套用在列上）與**不可能執行的列**（資料夾無法寫入，後端刻意不報價）。

5. **對話框外殼（🔴 #8；建單裁定 2）。**
   - `GenerationConsentView.tsx:525`：`sm:max-w-3xl` → `sm:max-w-[960px]`；加 `sm:border sm:border-[var(--border-subtle)]`（`DBPu5`；先例 dsr-6d-b 的 `Fkiqd`）。`sm:w-[calc(100vw-4rem)]` 保留（視窗窄時仍會縮）。
   - 這個寬度是**整個同意對話框**的（F14／F15／F18／F20 同一個 `DialogContent`，phase 切換不重新掛載）；F14 與 F20 的內容排版歸 `dsr-6e-2`。⛔ 不要碰 `ConfirmGenerationDialog`（F16／F19 是另一個 Dialog root，寬度歸 `dsr-6e-2`），也不要碰批次面板的 880（dsr-6d-b）。
   - 標題列（`:533` `h-14 pl-6 pr-12`）不動：稿上的 44×44 關閉鈕在 `ui/Dialog.tsx`，歸 `disc-2026-09-dialog-close-target-44px`。

6. **清單的排版與字級（🔴 #9–#15；只動桌機值，`@max-xl:` 那一組不動）。**
   - **摘要**：`:765` → `text-sm text-[var(--text-secondary)]`，`gap-[3px]` **保留**（那是把「候選 142 部 · 已選 46 部 · 預估」這串字與中間的 mono 數字黏在一起的間距；稿上那一整串是**一個** text 節點，6px 的 gap 只在金額之前）——改成 `gap-1.5` 會讓每個詞都隔 6px。金額那個 span `:769` 改加 `ml-1.5`（稿 `njQZt` 的 `$Space/xs-plus`）。外層 `:764` `gap-0.5` 保留。
   - **搜尋框** `:795`：`rounded-[var(--radius-md)] border border-[var(--border-subtle)] bg-[var(--bg-secondary)] px-4`，高度維持 `h-11`；圖示 `:798` `h-[18px] w-[18px] mr-2.5`；輸入 `:807` `text-sm`。
   - **排序** `:824`：`rounded-[var(--radius-md)] border border-[var(--border-subtle)] bg-[var(--bg-secondary)] pl-3 pr-1 gap-1.5`；圖示 `:827` `h-3.5 w-3.5`；`<select>` `:834` `text-sm text-[var(--text-secondary)]`。⛔ 仍然是原生 `<select>`（sub-6-11 AC #5 的理由還在）。
   - **chip** `:732`：`h-11 rounded-full px-3 text-xs font-medium`（Label 500）；選中 `bg-[var(--accent-tint)] text-[var(--accent-text)]`（稿 `J51aSC`；`--accent-tint` 是「chip 底色」那一階，`styles.css:56`）——⛔ 拿掉選中時的 `font-semibold`（稿是 500）；數字 span `:739` 的 `font-semibold` 一併拿掉。高度保留 44 是觸控下限（稿的 30 是視覺，程式碼的 44 是點擊區），不改成稿的 30。
   - **工具列**：`:853` `gap-x-3` → `gap-x-4`；`:854` `text-[13px]` → `text-sm`；兩顆文字按鈕 `:902`／`:911` → `text-sm font-medium`，「清除選取」`:911` 顏色改 `text-[var(--text-secondary)] hover:text-[var(--text-primary)]`（稿 `HBSgk`）。
   - **列** `:263`：`rounded-[var(--radius-md)] border border-[var(--border-subtle)] px-3 py-2.5`，**拿掉 `bg-[var(--bg-secondary)]`**（稿無底色，對話框本身就是 `--bg-secondary`，所以視覺上只少了一層重疊的同色）。
   - **片名** `:316`：`truncate text-sm font-semibold text-[var(--text-primary)] @xl:text-base`（桌機 BodyLg 600、手機 Body 600；建單裁定 5）。虛擬清單的 `ESTIMATED_ROW_PX`（`:58`）不用改——真實高度由 `measureElement` 回報。
   - **群組標題** `:491-494`：`rounded-[var(--radius-md)] px-3 py-2 gap-2.5`；`min-h-[44px]`／季標題的 `min-h-[40px]` 保留（觸控）。季標題的標籤 `:534` `text-[13px]` → `text-sm`（顏色不動）。
   - **頁尾**：`:1078` 的欄加 `gap-1`（稿 `zKqAw` gap 4）；`:1079` → `text-sm font-semibold text-[var(--text-primary)]`，`gap-[3px]` 保留、金額 span `:1082` 加 `ml-1.5`（與摘要同一條理由）；`:1093` 的 `gap-[3px]` 保留（明細是「抽取 46 部 $1.84 · 語音辨識 0 部 $0.00」一整串）；預算標籤 `:1104` → `text-sm`；預算欄 `:1107` → `flex h-9 w-24 items-center rounded-[var(--radius-md)] border bg-[var(--bg-primary)] px-3 font-mono text-sm tabular-nums text-[var(--text-primary)]`（`style` 的 `borderColor` 邏輯不動），裡面的 `<input>` `:1124` `w-16` → `min-w-0 flex-1`。
   - **開始產生** `:1142`：`px-6 font-medium` → `px-5 font-semibold`（Primary 14／600、[8,20]），`min-h-[44px]` 保留。
   - **F18 橫幅** `:1013`：`flex items-center gap-2 bg-[var(--warning-tint)] px-6 py-2.5`（**拿掉** `mx-6 mb-2 rounded-[var(--radius-md)] p-3`，貼齊兩側）；圖示 `:1015` 換 `TriangleAlert`（lucide，稿 `Jx2Pe`），顏色維持 `--warning-text`（⚠️ 稿是 `$warning`，但 `local/no-base-semantic-as-text` 擋 `text-[var(--warning)]`——**以程式碼為準**，dsr-6d-b 的 `Bc8Ps` 先例）；文字 `:1016` → `text-sm`，`gap-[3px]` 保留（標點黏字的理由見 `:1030-1036`）。
   - **開始失敗** `:1066-1073`：`mx-6 my-2 flex items-center gap-2 rounded-[var(--radius-md)] bg-[var(--error-tint)] px-3 py-2.5 text-sm text-[var(--error-text)]`（稿 `C2mDb`）。
   - **搜尋沒結果** `:996`：`text-sm`。
   - `CandidateListPanel.tsx:1` 的 Rule 21 標頭補上 `· F18-spec-err (VPT7l)`（開始失敗的位置來自那張規格）。

7. **單季影集的群組標題（🔴 #7；建單裁定 6）。** `consentRows.ts:346`（`pushSeriesRows`，`:331-377`）：`showSeasonHeaders === false` 而且只有一季時，`label` 改成 `` `${seriesTitle || '未知影集'} · ${seasonLabel(n)}` ``（`seasonLabel` 在 `:380`，0 → 「特別篇」）。`selectLabel`「選取整部 X」與 `testid` 不動；多季影集的兩層結構不動。

8. **既有的行為不准回歸**（測試都要留著）：
   - 三序同源：顯示、送出、F18 可完成部數走同一個順序（spec `GenerationConsentView.spec.tsx:700`、`:804`、`:826`）。
   - 全選與群組 checkbox 只管**看得到的**列（sub-6-12 AC #1；`:860`、`:875`、`:890`）；群組標題在被篩選縮窄時說「顯示 n」。
   - `≈` 一列最多一個；片長未知寫「片長未知（估 N 分）」；摘要、頁尾的 `≈` 由 `usdWithEstimate` 決定。
   - 砍線、變暗、橫幅**同時出現或同時不出現**（`showCut`，`:635`）；排序不是「群組」時橫幅加「（依提交順序計算，不是目前的排序）」。
   - WYSIWYG 預算、預填 `default_budget_usd`（`:573`–`:616`）。
   - 虛擬清單門檻 80 列、`gap` 告知虛擬器、換篩選／搜尋／排序捲回頂端。
   - 手機列型的 container query（spec `CandidateListPanel.spec.tsx:714-752` 的 class 斷言）、首字海報的 class（`:596`）、未匹配標與它的 `title`。
   - 開始失敗的訊息貼在按鈕上方、不在捲動區裡（sub-6-12 AC #2）。

9. **測試（先寫紅測試）。**
   - `consentSelection.spec.ts`：
     - 🚨 **`:326-329` 那條現有測試斷言的正是本張禁止的行為**（標題「falls back to the row's own estimate…」，`:328` 斷言 `candidateUsd(row, { [B]: 0.02 })` 是 `0.05`）。**改寫**：`:328` → `toBeNull()`、測試名改成「有 prices 卻缺這一列 → null，不退回預設模型價」；`:329`（沒有 `prices` → `0.05`）保留。AC #8「測試都要留著」不涵蓋這一條——它守的是被裁掉的行為。
     - `candidateUsd`：`estimatedUsd` 是 `undefined`／`NaN` → `null` 且**不丟例外**；不可寫入的列 → `null`（不論有沒有 `prices`）；有 `prices` 但缺這一列 → `null`（⛔ 不是 `estimatedUsd`）；有 `prices` 有這一列 → 表裡的數字。
     - `isSelectable`；`defaultSelection`／`selectableIds`／`visibleSelectableIds` 排除沒報價的列。
     - `computeTotals`：`unpricedCount` 與 `unwritableCount` 分開算；已選但 `null` 的列不進任何金額並記在 `unpricedSelectedCount`。
     - `modelChoices`：某個**非預設**模型的 `perCandidate` 缺一個可選的列 → 那個模型不在選項裡；**預設模型**缺 → 整組 `[]`（⛔ 不准自動改用別的模型）；舊後端路徑不變（既有測試留著）。
     - 片長分母仍是可寫入的列（沒報價但可寫入的列**要**算進 `sweepRuntime` 與 `selectedRuntime`）——拿掉 `isWritable` 會讓「約 N 分鐘」偏大。
   - `consentRows.spec.ts`：單季影集的標籤是「劇名 · 第 N 季」、季 0 是「劇名 · 特別篇」、多季影集的影集列仍只有劇名（`:299` 的 `selectLabel` 斷言不動）。
   - `CandidateListPanel.spec.tsx`：
     - **一列沒有 `estimatedUsd` 時整個面板照樣 render**（🔴 #1 的回歸測試；現在會丟 `DecimalError`），那一列有「無法估價」、checkbox 停用、沒有金額；工具列有「（1 部無法估價）」。
     - 不可寫入的列沒有金額（`consent-row-usd-{id}` 不存在）。
     - 金額中性：列、摘要、頁尾的金額在 `overBudget` 時**都沒有** `--warning-text`／`--success-text`（用 `className` 逐字比對，dsr-6b 的教訓：`toHaveClass` 會漏掉「多了一個」）；砍線文字是 `--text-primary`。
     - 路線徽章、群組徽章、chip 標記都是中性，而且**沒有** `-tint` 語意底色；「資料夾無法寫入」仍是 `--error-tint`。
     - 片名有 `font-semibold` 與 `@xl:text-base`。
     - `unpricedSelectedCount > 0` → 開始產生停用而且有原因那一行。
   - `GenerationConsentView.spec.tsx`：快照裡一列沒有 `estimated_usd` → 清單照樣出現、那一列不在預設選取裡、送出的 id 不含它；`estimates_by_model` 裡某模型缺一列 → 那個模型不出現在確認框的選項裡。**保留不動**：AC #8 列的那些。
   - `GenerationConsentView.spec.tsx:721` 的 `toHaveTextContent('怪奇物語')` 是子字串比對，單季標籤改完仍會過——**另外加**一條逐字斷言新標籤（否則這條改動沒有測試）。

10. **視覺夾具與基準線。**
    - 桌機清單夾具（`-gallery.fixtures.tsx`）`generation-consent/{list, grouped, over-budget, f15-collapsed, f15-search-hit, f15-sorted-cost}` 的 `width: 900` → `960`（對話框內容寬度）。
    - `ConfirmGenerationDialog.spec.tsx:9` 的 `baseTotals`、`-gallery.fixtures.tsx:554` 的 `confirmTotals()` 與 `:4820-4860` 兩個手寫 totals 補 `unpricedCount: 0, unpricedSelectedCount: 0`（AC #2 的型別；⛔ 只動 spec／夾具，不動 `ConfirmGenerationDialog.tsx`）。
    - 重生 darwin：上面六張＋`list-mobile`、`list-mobile-groups`（共用程式碼，字重與顏色會變）。照 `project_visual_baseline_intentional_change` 的四步：自己先 `nx serve web`，再 `CI=1 npx playwright test --project=visual --update-snapshots --grep generation-consent`（本機是 darwin，只會產 darwin）→ `git rm` 過期的 `-linux` → 把 main 併進分支 → `gh workflow run "Visual Regression" --ref <branch>` 開 bootstrap PR 合回分支。⛔ 不要本機產 `-linux.png`。
    - 夾具的 `props` 是 `Record<string, unknown>`——`ConsentTotals` 加欄位時 typecheck 抓不到夾具裡手寫的 totals；改完一定要看圖確認工具列與頁尾有畫出來。

11. **另立的單子（建單時已寫入 sprint-status，不在本張做）。**
    - 新：`disc-2026-09-native-checkbox-vs-checkbox-master`（🔴 #18）、`disc-2026-09-pen-meta-notes-wear-ochre`（設計稿裡寫給設計師看的後設註記 `c4FIoB`／`CJVC5`／`NtMLG` 用 `$warning-tint`＋`$warning-text`，`DESIGN.md:253` 明文說那不合法——本張改用中性的新註記，既有三顆不動）。
    - 既有：`disc-2026-09-dialog-close-target-44px`、`disc-2026-09-dialogframe-shadow-vs-shadow-xl`、`disc-2026-09-type-scale-even-migration`（行高）、`disc-2026-09-generation-resume-c-estimate-deduction`（清單「已翻部分不重算」的註記是那張的事）、`backlog-consent-policy-implementation`。

12. **CI 全綠**：`pnpm run lint:all`、`pnpm nx run web:typecheck --skip-nx-cache`、`python3 scripts/check-design-tokens.py`、`pnpm nx test web`、`pnpm nx test api`（本張不改後端，但全套閘門照跑）。e2e `pnpm run test:e2e -- --grep @batch-subtitle`（同意流程在那支裡走一遍：`tests/e2e/batch-subtitle.spec.ts:258-262`）本機跑一次，結果寫進 Completion Notes。⛔ 絕不用 `run_in_background` 跑測試；每次跑完 `pnpm run test:cleanup`。Nx daemon 建圖逾時就用 `NX_DAEMON=false` 重跑並記在 Completion Notes。

## Tasks / Subtasks

- [x] **Task 1 — 設計稿（AC: #1）**
  - [x] 路線／費用標記中性、F18「免費」→「僅翻譯費」、F18 五列改列型、F18 補搜尋列、`lJppJ` 字級、`DesXa` 關掉、`UF8ic` 搬進 `c4FIoB`
  - [x] `ctx.problems`；存檔並確認落盤；匯出後只 stage f15-d-v2／f18-d-v2／f15-m-v2（＋pen-tokens）
- [x] **Task 2 — P1：報價缺值（AC: #2, #9 前兩項）**
  - [x] 先寫紅測試（`candidateUsd` 不丟例外、不偷換模型；`isSelectable`；`modelChoices` 覆蓋率；`computeTotals` 新欄位）→ `consentSelection.ts`
  - [x] 容器三處換 `isSelectable`；面板：無法估價標、不可寫入不畫金額、工具列計數、停用＋原因
- [x] **Task 3 — 金額與分類不穿狀態色（AC: #3, #4, #9）**
  - [x] 先寫紅測試 → 列／摘要／頁尾／砍線中性；三組徽章中性；`DESIGN.md` 補一句
- [x] **Task 4 — 外殼、排版、字級、單季標籤（AC: #5, #6, #7）**
  - [x] 960＋框；14 處 13px；搜尋、排序、chip、工具列、列、片名、群組標題、頁尾、預算欄、按鈕、橫幅、開始失敗
  - [x] `consentRows.ts` 單季標籤（先寫紅測試）
- [x] **Task 5 — 夾具與基準線（AC: #10）**
- [x] **Task 6 — 收尾（AC: #8, #11, #12）**
  - [x] 全套閘門、e2e；dev-story Step 9 截圖比對（`f15-d-v2`、`f18-d-v2`）

## Dev Notes

### 這張的重點

- **P1 的本質是「不知道」被寫成了「崩潰」或「別人的數字」**。改完之後，不知道的那一列會說它不知道、而且選不到；選得到的列一定有一個屬於**你選的那個模型**的價錢。
- **顏色只說狀態**：金額（事實）與路線（分類）都不穿顏色。整個畫面上的顏色剩下：泥金（按鈕、選中的 chip）、赭（超過上限）、硃砂（資料夾無法寫入、無法估價、開始失敗）。
- **設計稿大多已經對了**：F15 在 2026-09-10 的整理裡已經改成中性金額、16／600 片名、有群組徽章。本張是把程式碼追上去，外加修掉稿上還錯的三件事（徽章顏色、F18 過時、10px）。

### 上游契約（Rule 20 ack）

- confirmed against [@contract-v1] (Story sub-4-1 AC #7) —— `GET /api/v1/subtitles/generation-candidates` 的 `result.candidates[].estimated_usd`（今天恆存在，本張只是讓前端在它缺席時不崩）。
- sub-6-8a AC #3（`result.estimates_by_model[<id>].per_candidate`）**本身沒有 stamp**——它是加在 sub-4-1 AC #7 `[@contract-v1]` 上的 additive 欄位（該 story 的 Dev Notes 寫明 additive ack、不 bump），所以 ack 掛在 AC #7 上：`omitempty`，而且後端**只替可寫入的列報價**（`generation_candidates.go:834-846`）——本張 AC #2 的涵蓋規則就是建立在這件事上。
- confirmed against [@contract-v1] (Story sub-6-8a AC #2) —— `GET /api/v1/settings/models`（本張不改它的消費方式）。

### 建單裁定（2026-09-20，Sally／Alexyu 可在 review 推翻）

1. **拆成 6e-1／6e-2**（見 Context），切在版面：清單 vs 其他四個小畫面。
2. ✅ **同意對話框整段固定 960**（F14／F15／F18／F20）——**⚖️ Alexyu 2026-09-20 裁定（建單時提問，選「整段固定 960」）**。理由：它們是同一個 `DialogContent`，照稿（F14 560 → F15 960）等於在使用者眼前把對話框撐大；F16／F19 是另一個 Dialog root，維持稿上的 480；執行面板 880（dsr-6d-b）不動——同意 → 執行是兩個 Dialog root 的切換（本來就重新掛載）。F14 的稿改成 960（內容欄仍是 480）歸 `dsr-6e-2`。
3. ✅ **路線徽章與 chip 的費用標記改中性**——**⚖️ Alexyu 2026-09-20 裁定（建單時提問，選「改成灰色中性」）**，也就是 dsr-6 原單子 ③ 要「一起裁定」的那件事。依據是既有的兩條：`DESIGN.md:300` 狀態色不得用於分類，以及 Alexyu 2026-09-10 對 TechBadge 的同類裁定（`DESIGN.md:716-720`：「它們已經有文字，顏色沒有承載額外資訊」）。「抽取」「語音辨識」是這一列走哪條路，跟 H.265 是這個檔案的編碼一樣是屬性。**四處同類**：列的路線徽章、群組的路線徽章、chip 的費用標記（本張）、模型的品質徽章（`dsr-6e-2`）。
4. **金額中性不是本張的裁定**，是 Alexyu 2026-09-10 裁定 B（`disc-2026-09-money-has-no-color`，done）。
5. **片名：桌機 16／600、手機 14／600**。`backlog-consent-row-title-tier-drift` 等的「Body 400 還是 600」已經在 2026-09-10 的稿上定案（F15／F18 BodyLg 600、F15-M Body 600），本張把程式碼追上去並結案該單。
6. **單季影集標題「劇名 · 第 N 季」，多季維持兩層**；註記寫進 `c4FIoB`。結案 `backlog-f15-group-header-pen-code-drift`（(1) 徽章稿上已補，(2) 由本張 AC #7 對齊）。
7. **不可寫入的列不顯示金額**：後端刻意不替它們報價（sub-6-1），那一格顯示任何數字都是別的模型的價錢或假設。它仍顯示路線徽章與「資料夾無法寫入」。
8. **F18 採 F15 的列型**：兩張稿畫的是同一個 `CandidateRow`，這不是設計決定而是同步。結案 `backlog-f18-pen-row-format-drift`。
9. **F18 橫幅維持赭**（理由見 AC #4）。
10. **徽章一律藥丸**：`DESIGN.md:566-568` 2026-09-10 的新圓角規則就是「看語意不看高度」——資料標記藥丸、`radius-sm` 只給不可互動的小色標。本張反正要動這些節點，順手一起對齊（群組徽章、chip 標記、資料夾無法寫入、未匹配、新的無法估價）。
11. **沒有報價的列選不到、不可寫入的列不畫金額**（AC #2）——這比 `DESIGN.md:318` 的字面多出兩種「沒有金額」的狀態，所以規則本身也要補（AC #4）。

### 不要做的事

- **不要碰 `ConfirmGenerationDialog`、`ModelPicker`、`AnalysisProgressPanel`、`ConsentEmptyState`**（`dsr-6e-2`）。例外：本張 AC #2 讓 `modelChoices` 少給不完整的模型，那是 `consentSelection.ts` 的事，確認框不用改。
- **不要做手機排版**（`@max-xl:`、手機 sheet 的形狀歸 `dsr-6f`）。
- **不要改 `ui/Dialog.tsx`**（關閉鈕、陰影——兩張既有單子）。
- **不要把缺的金額補成 0**，也不要在 `candidateUsd` 裡退回 `estimatedUsd`。
- **不要改後端**：`estimated_usd` 補 `omitempty` 與否、`per_candidate` 要不要涵蓋不可寫入的列，都不是本張的事。
- **不要換掉原生 `<select>` 與原生 checkbox**。
- **不要加行高**（`leading-relaxed` 全 app 一起處理）。

### 已知陷阱

- **`candidateUsd` 的呼叫者很多**：`computeTotals`、`sumSelected`、`sortForDisplay`（`consentRows.ts:81-89`，金額排序）、`CandidateRow`（`:375-377`）。（`GroupHeaderRow` 的小計是經過 `computeTotals` 的，不直接呼叫。）回傳型別改成 `number | null` 後 typecheck 會全部點出來——**金額排序遇到 `null` 要排在最後**（兩個方向都是），不要讓 `cmpUsd(null, …)` 丟例外。
- **`unwritableCount` 的語意改了**：今天是 `candidates.length - selectableCount`，會把沒報價的列算成「資料夾無法寫入」。
- **`toHaveTextContent` 是子字串比對**：「怪奇物語 · 第 4 季」包含「怪奇物語」，舊斷言照樣過；class 也要逐字（dsr-6b／6c 的教訓）。
- **視覺 CI 沒有後端**：夾具全靠 props。
- **`--accent-tint` ≠ `--accent-subtle`**（`styles.css:55-56`）：一個是 chip 底、一個是導覽與選中列的洗色。
- **行號以建單時為準**（2026-09-20，main `c4d34187`）。

### Source tree

```
apps/web/src/components/subtitle/consent/consentSelection.ts(+spec)     ← Task 2
apps/web/src/components/subtitle/consent/consentRows.ts(+spec)          ← Task 2（金額排序的 null）、Task 4（單季標籤）
apps/web/src/components/subtitle/consent/CandidateListPanel.tsx(+spec)  ← Task 2, 3, 4
apps/web/src/components/subtitle/consent/GenerationConsentView.tsx(+spec) ← Task 2（isSelectable 三處）、Task 4（外殼）
apps/web/src/components/subtitle/consent/ConfirmGenerationDialog.spec.tsx  ← Task 2（只補 ConsentTotals 的兩個新欄位，⛔ 不動元件）
apps/web/src/routes/test/-gallery.fixtures.tsx                          ← Task 5
tests/visual/components.visual.spec.ts-snapshots/components/generation-consent/** ← Task 5
DESIGN.md                                                               ← Task 3
ux-design.pen、_bmad-output/pen-tokens.json、_bmad-output/screenshots/flow-f-subtitle-v2/{f15-d-v2,f18-d-v2,f15-m-v2}.png ← Task 1
```

### Cross-Stack Split Check

後端 task **0 個**，前端／設計 task 6 個。→ 不觸發跨棧拆分；規模拆分已在 Context 說明。

### Time-dependent visual coverage

- **N/A — no wall-clock-reading components touched.** `CandidateListPanel`、`consentRows`、`consentSelection`、`GenerationConsentView` 都不讀時鐘（檔頭 Rule 23 註記；`analyzed_at` 刻意不顯示）。搜尋的 200ms debounce 是等待，不是讀時鐘。

### References

- [Source: `apps/web/src/components/subtitle/consent/CandidateListPanel.tsx:1-18, 54-63, 226-383, 397-414, 424-575, 700-751, 753-1155`]
- [Source: `apps/web/src/components/subtitle/consent/consentSelection.ts:50-112, 188-281, 283-373, 604-716`、`consentRows.ts:54-100, 331-382`]
- [Source: `apps/web/src/components/subtitle/consent/GenerationConsentView.tsx:178-213, 377-415, 490-504, 518-535`]
- [Source: `apps/api/internal/services/generation_candidates.go:246, 319-350, 834-846`（報價只涵蓋可寫入的列）]
- [Source: `apps/web/src/routes/test/-gallery.fixtures.tsx:4575-4815`、`tests/e2e/batch-subtitle.spec.ts:170-190, 258-262`]
- [Source: `ux-design.pen` `pwMzT`／`DBPu5`／`zBik1`／`ckLed`／`VPT7l`／`fdu4y`／`6MxLT`／`955EZ`／`jD7gF`／`otvKh`／`c4FIoB` —— SM 建單時以 Pencil MCP 逐節點讀出（2026-09-20）]
- [Source: `DESIGN.md:246-253`（赭的定義與 2026-09-11 裁定）、`:285`（基礎語意色不可當文字）、`:289-300`（固定詞彙）、`:308-322`（金錢是事實、會花錢的動作、`:318` 合法的無金額狀態、金額的寫法）、`:355-358`（字階 H4／BodyLg／Body／Label）、`:566-568`（2026-09-10 圓角新規則）、`:716-720`（TechBadge 中性裁定）]
- [Source: `sprint-status.yaml` → `dsr-6-flow-f-subtitle-v2`（③ 的歸屬）、`dsr-6e-consent-flow`、`disc-2026-09-candidate-usd-missing-guard`、`disc-2026-09-money-has-no-color`、`backlog-f18-pen-row-format-drift`、`backlog-f15-group-header-pen-code-drift`、`backlog-consent-row-title-tier-drift`]
- [Source: `_bmad-output/implementation-artifacts/dsr-6d-b-batch-dialog.md`（寬度裁定的交接、`Fkiqd` 框、`$warning` 圖示以程式碼為準的先例）、`sub-6-12-consent-money-traps.md`、`sub-6-10b-candidate-identity-frontend.md`、`sub-6-8b-per-run-model-selection-frontend.md`]
- [Source: project-context.md#Rule 5 / #Rule 16 / #Rule 20 / #Rule 21 / #Rule 23 / #Rule 24]

## Dev Agent Record

### Agent Model Used

Claude Fable 5.1 — dev-story (Amelia)

### Debug Log References

- Pencil 在開工時「沒有開啟的檔案」（隔夜被關掉），`open -a Pen ux-design.pen` 重新打開後 MCP 才接得上。
- Nx 一律加 `NX_DAEMON=false`。視覺測試只有**一支** test 包全部夾具，`--grep generation-consent` 會回「No tests found」——只能整支跑（8.7 分鐘），再把無關的還原。
- `pnpm run test:cleanup` 會順手把 `nx serve web` 砍掉（exit 144），不是錯誤。
- e2e 要 `AI_PROVIDER=claude` 後端才起得來；`pnpm run test:e2e -- --grep …` 會因為多個 project 回「No tests found」，改用 `npx playwright test tests/e2e/batch-subtitle.spec.ts --project=chromium`。

### Completion Notes List

**做了什麼（對使用者的差別）**

1. **少一個金額，對話框照樣打得開（P1）。** `candidateUsd` 改回傳 `number | null`，三條規則依序：不可寫入 → 一律 `null`；有模型價目表 → 表裡的數字或 `null`（**不再退回預設模型的價**）；沒有表 → 這一列自己的 `estimatedUsd`（要是有限數字）。新增 `isPriced`／`isSelectable`，所有「能不能選」的路徑（預設選取、全選、群組、預選交集、送出）都改用 `isSelectable`。沒報價的列：checkbox 停用、右側「無法估價」、沒有金額格、工具列多一句「（N 部無法估價）」。
2. **換模型不再偷換價錢。** 不可寫入的列不畫金額（後端本來就不替它們報價）；`modelChoices` 只提供「每一個選得到的列都有報價」的模型；**預設模型的報價有洞就整組回 `[]`**，不會無聲改用別的模型。片長分母刻意維持 `isWritable`（與後端的加總一致，有測試守著）。
3. **金額沒有顏色了。** 列、摘要、頁尾的金額一律 `--text-primary`（超過上限也一樣）；砍線的赭色留在線上，句子與金額中性。
4. **分類也沒有顏色了**（⚖️ Alexyu 2026-09-20）。列的路線徽章、群組的路線徽章、chip 的「僅翻譯費／付費」都是中性藥丸。畫面上剩下的顏色：泥金（按鈕、選中的 chip）、赭（超過上限）、硃砂（資料夾無法寫入、無法估價、開始失敗）。
5. **對話框 960 寬＋髮絲框**，分析中、清單、空狀態同一個寬度。
6. **排版對稿**：14 處 13px 全收；列（radius-md、無底色、[10,12]）、片名（手機 14/600、桌機 16/600）、金額（手機 12、桌機 14）、群組標題、搜尋框、排序、chip（藥丸、Label 500、`--accent-tint`）、工具列、頁尾、預算欄（96×36）、開始產生（14/600）、F18 橫幅（貼齊兩側＋三角圖示）、開始失敗訊息。
7. **單季影集的標題**寫「劇名 · 第 N 季」（季 0＝特別篇），多季維持兩層。
8. **停用要說原因**：已選的列若沒有價錢，「開始產生」停用，並顯示「有 N 部沒有報價，請清除選取後重選」（CR M2 後改的文案）——這一行**不受**「超過上限就藏起預算提示」那個條件影響。

**設計稿（Task 1）**

- 排序標籤 `lJppJ`／`v05oXD` 13 → Body（**先改、再複製**搜尋列進 F18）；10 顆列徽章、4 顆群組徽章、6 顆 chip 標記（含手機 `BZwGh`／`Ouuhe`）改中性藥丸，chip 標記 10px → Label 500；`mm34N`／`otBhu` 改藥丸；F18 `ts6OZ`「免費」→「僅翻譯費」；F18 五列換成首字海報 38×54＋Label muted 副標含片長，星際效應補「未匹配」chip；`DesXa`／`YqUYl` `enabled:false`；`UF8ic` 刪除，新的中性註記 `gwS55` 放在 `JzmvC`（`dufQr` 下方 40px，無重疊）。
- 🎨 **偏離單子的一處**：群組徽章的底我用 **`$bg-secondary`**（稿與程式碼一致），不是單子寫的 `$bg-tertiary`——群組標題本身就是 `$bg-tertiary`，同色的藥丸等於沒有邊。理由與單子自己對 chip 標記的說法相同（「同色會看不見」）。
- F18 加了搜尋列後對話框高 862，超出 900 的畫布 3px → `ckLed` `y: 41 → 19` 置中。
- `ctx.problems`：三張畫面裡只剩既有的背景裁切（`sec-活動記錄`）與手機 `route-chips`（橫向捲動列，本來就會裁）。`pen-tokens.json` 的 `clippingWarnings` 68 → 70，多的兩個就是被關掉的 `DesXa`／`YqUYl`（停用節點會被算成 fully clipped）。
- 存檔用 AppleScript 點 File ▸ Save；已讀磁碟檔確認 `Jo8Lz`／`dYPUR` 的 fill、新註記與「僅翻譯費」都落盤。匯出後只留 `f15-d-v2`／`f15-m-v2`／`f18-d-v2`＋`pen-tokens.json`；其餘 17 張（含 `component-library`）是 Pencil 重開後的重繪雜訊——我沒有動任何母版，全部還原。

**閘門結果**

| 閘門 | 結果 |
| --- | --- |
| `pnpm run lint:all` | ✅ 0 errors／127 warnings（全部既有）、prettier clean |
| `pnpm nx run web:typecheck --skip-nx-cache` | ✅ |
| `python3 scripts/check-design-tokens.py` | ✅ 夜行 33／日巡 33、82 變數、196 畫面、73 母版 |
| `pnpm nx test web` | ✅ 3905/3905（270 檔；CR 後） |
| `pnpm nx test api` | ✅（本張沒改後端） |
| e2e `batch-subtitle.spec.ts`（chromium） | 7 條中 5 綠；2 條紅是本機 `AI_PROVIDER=claude` 讓請求主體多一個 `model_id`（已看 diff 確認只差這一欄）——與 dsr-6d-b 記錄的是同一個本機環境副作用，CI 上是綠的 |
| 視覺 | 8 張 darwin 重生、8 張過期 `-linux` 已 `git rm`；本機固定會漂的 `parse-floating-parse-progress-card`／`retry-retry-notifications` 已還原 |

- 🔗 AC Drift: FOUND —— `sub-6-8b` AC #3「沒有 per-model 報價的列退回 `estimatedUsd`」→ 現在是「沒有金額」（這正是本張的 P1）；`sub-6-1` CR M4 的 `unwritableCount = candidates − selectable` → 現在只數 `!isWritable`，沒報價的另計 `unpricedCount`；`sub-4-3` AC #2／`sub-6-12` 的「抽取綠、語音辨識橘」金額與徽章 → 中性（Alexyu 2026-09-10 裁定 B＋2026-09-20 裁定）。三者都是刻意的、由本張的 AC 明文取代。
- 📎 Contract Stamps: FOUND（上游 sub-4-1 AC #7 `[@contract-v1]`、sub-6-8a AC #2 `[@contract-v1]`；本張只改前端對缺值的容忍度，沒有改任何線上形狀，ack 已在 Dev Notes）
- 🎭 A11y Pre-Flight: PASS（2 個元件；touched files 上 1 個 warning，是既有的 `react-hooks/incompatible-library`，0 個由本張引入）。停用的 checkbox 用原生 `disabled`（不可聚焦是對的——那一列本來就不能選），原因在同列的徽章與 `title`；「開始產生」停用時原因是可見文字。
- 🎨 UX Verification: PASS —— 對照 `f15-d-v2`／`f18-d-v2` 與新的 `grouped`／`over-budget` 基準線：

| Area | Design Spec | Implementation | Match? |
| --- | --- | --- | --- |
| 對話框 | 960、1px `$border-subtle` | `sm:max-w-[960px] sm:border` | ✅ |
| 列 | radius-md、無底色、[10,12] | 同 | ✅ |
| 片名 | BodyLg 600（手機 Body 600） | `text-sm font-semibold @xl:text-base` | ✅ |
| 金額 | Body 600 mono `$text-primary` | 同（手機 Label） | ✅ |
| 路線徽章 | 中性藥丸 [4,10] Label 600 | 同 | ✅ |
| chip | 藥丸 Label 500、選中 `$accent-tint` | 同；高度 44（觸控，稿 30——單子明文保留） | ✅（刻意） |
| 搜尋／排序 | `$bg-secondary`＋框、radius-md | 同 | ✅ |
| F18 橫幅 | 貼齊兩側、triangle-alert、Body | 同；圖示色 `--warning-text`（lint 擋 `--warning`，單子明文） | ✅（刻意） |
| 頁尾 | 14/600、預算欄 96×36 `$bg-primary` | 同 | ✅ |
| checkbox | 母版 20×20 | 原生 16px | ❌ 已立案 `disc-2026-09-native-checkbox-vs-checkbox-master` |

**🔍 /ship 對抗式 CR（2026-09-20，fresh-context 代理，只讀）**——0 HIGH／3 MEDIUM／8 LOW，**修 7、立案 4、駁回 0**。

| 等級 | 問題 | 處置 |
| --- | --- | --- |
| **M1** | 規則 2 沒看 `isPriced`：沒有 `estimatedUsd`、但模型價目表裡有這一列時，畫面會出現「$0.31 無法估價」（金額＋停用的 checkbox＋說不能估價的標）。而且這正是 P1 最可能的形狀——後端在同一個迴圈裡填兩者，掉了 `estimated_usd` 的 payload 通常還留著 per-model 那一筆。 | 修：規則 1 改成 `!isSelectable(c)` 一律 `null`；spec 那行改 `toBeNull()`，另加 panel 測試 |
| **M2** | 文案叫使用者「重新分析」，但對話框裡沒有這個入口（關掉重開讀的是同一份快照）。 | 修文案（拿掉承諾）；入口要不要加 → 立案 |
| **M3** | 測試名說有測群組徽章，身體沒有讀它——把群組徽章改回綠／橘整套仍然綠。新的「無法估價」標也沒斷言樣式。 | 修：補群組徽章與標的斷言，測試名改實話 |
| L1 | 預設模型**完全沒有**報價（別的模型有）時仍會掉到 `choices[0]`，新註解卻說這條路關了。 | 修：同樣回 `[]`＋測試 |
| L3 | 「不進任何數字」那條測試沒斷言 `estimatedRowCount`／`visibleSelectedCount`／`pausedIds`／`cutMediaId`。 | 修 |
| L4 | 金額排序測試的兩個 null 列本來就在輸入的最後，抓不到「null 回 0」的 mutant。 | 修：`sortForDisplay` 單元測試，null 列放最前、兩個方向 |
| L8 | `sumSelected` 的註解還寫 writable。 | 修 |
| L2 | 只有總額、沒有 `perCandidate` 的報價 → 該模型不提供；是預設就整組 `[]`。退路安全。 | 確認無誤，不改 |
| L5／L6／L7 | 已選不可寫入列的原因誤說、送出與 totals 的條件不同源、停用原因沒連到按鈕。 | 立案 `disc-2026-09-consent-unpriced-row-followups` |

**沒做的事**

- `-gallery.fixtures.tsx` 裡 `confirm`／`confirm-over-budget` 兩個**手寫**的 totals 沒補新欄位：它們只餵給 `ConfirmGenerationDialog`，那個元件不讀新欄位，而且它們本來就只有部分欄位（`dsr-6e-2` 會整理）。型別化的 `confirmTotals()` 與 `ConfirmGenerationDialog.spec.tsx` 的 `baseTotals` 已補。
- 預設模型「完全沒有報價」（不是有洞）的情況維持原行為——那是既有路徑，不在本張的涵蓋規則裡。

### Discovery Triage

- **Did this story discover any work outside its current scope?** **YES** —— 建單當下處理如下。

- **① expand-scope-in-place**
  - 報價缺值讓畫面崩潰、換模型偷換價錢（P1 `disc-2026-09-candidate-usd-missing-guard`，🔴 #1、#2）→ **AC #2**（結案該單）
  - F18 稿停在舊列型、「免費」（`backlog-f18-pen-row-format-drift`，🔴 #5）→ **AC #1**（結案該單）
  - 片名字級（`backlog-consent-row-title-tier-drift`，🔴 #6）→ **AC #6**（結案該單）
  - 群組標題的兩處差異（`backlog-f15-group-header-pen-code-drift`，🔴 #7）→ **AC #1、#7**（結案該單）
  - 稿上的規格註記塞在畫面裡（🔴 #16）→ **AC #1**
  - `:295-296` 會變成假話的註解（🔴 #17）→ **AC #3**

- **② spawn-blocking-story**：無。

- **③ backlog-with-carry-forward-link**（建單時立）
  - `disc-2026-09-native-checkbox-vs-checkbox-master` — 原生 checkbox 與 `Component/Checkbox/*` 母版不一致，全 app（🔴 #18）
  - `disc-2026-09-pen-meta-notes-wear-ochre` — 設計稿的後設註記盒 `c4FIoB`／`CJVC5`／`NtMLG` 穿赭色，違反 `DESIGN.md:253`（建單後對抗驗證撿到；本張只是不把新註記放進去）
  - **手機（`dsr-6f`）**：本張改的是桌機手機共用的程式碼，所以 F15-M 的片名字重、金額顏色與徽章顏色會跟著變；手機的**排版**仍歸 `dsr-6f`（已寫進它的 sprint 條目，含「金額尺寸 `text-xs @xl:text-sm`」與「不可寫入的列不畫金額」兩件事）

  - `disc-2026-09-consent-unpriced-row-followups` — 沒有「重新分析」入口、停用原因的無障礙、兩個只有直接餵 props 才到得了的邊角（/ship CR M2 後半＋L5／L6／L7）

- Reference: `project-context.md` Rule 24

### File List

**修改**

- `apps/web/src/components/subtitle/consent/consentSelection.ts`（+spec）
- `apps/web/src/components/subtitle/consent/consentRows.ts`（+spec）
- `apps/web/src/components/subtitle/consent/CandidateListPanel.tsx`（+spec）
- `apps/web/src/components/subtitle/consent/GenerationConsentView.tsx`（+spec）
- `apps/web/src/components/subtitle/consent/ConfirmGenerationDialog.spec.tsx`（只補 `ConsentTotals` 兩個新欄位；元件沒動）
- `apps/web/src/routes/test/-gallery.fixtures.tsx`（六個清單夾具 900 → 960；`confirmTotals()` 補欄位）
- `DESIGN.md`（無金額的列兩種、分類一律中性的 2026-09-20 延伸）
- `ux-design.pen`、`_bmad-output/pen-tokens.json`、`_bmad-output/screenshots/flow-f-subtitle-v2/{f15-d-v2,f15-m-v2,f18-d-v2}.png`
- `tests/visual/components.visual.spec.ts-snapshots/components/generation-consent/{list,grouped,over-budget,f15-collapsed,f15-search-hit,f15-sorted-cost,list-mobile,list-mobile-groups}/default-visual-darwin.png`
- `_bmad-output/implementation-artifacts/sprint-status.yaml`

**新增**

- `_bmad-output/implementation-artifacts/dsr-6e-1-consent-list.md`、`dsr-6e-2-consent-small-surfaces.md`（建單產物，隨本分支進版）

**刪除**

- 上述 8 個夾具的 `default-visual-linux.png`（過期基準；交給 CI bootstrap）

**AC drift reference — see Completion Notes**：`sub-6-8b-per-run-model-selection-frontend.md`、`sub-6-1-preflight-writable-target.md`、`sub-6-12-consent-money-traps.md`

⛔ **沒有碰**：`ConfirmGenerationDialog.tsx`、`ModelPicker.tsx`、`AnalysisProgressPanel.tsx`、`ConsentEmptyState.tsx`、`ui/Dialog.tsx`、任何後端檔案。

## Change Log

| 日期 | 內容 |
| --- | --- |
| 2026-09-20 | ✅ **DONE** —— PR #481 合併進 main（commit `5955a563`）。CI 全綠：Lint、Unit、Go、4 個 E2E shard、3 個 Build、Serve Smoke、4 個視覺 shard。8 張 `-linux` 基準由手動觸發的 Visual Regression workflow 開 bootstrap PR #482（只有 8 張 PNG＋一行稽核紀錄）合回分支後一次過。本機 e2e 紅的那 2 條（`model_id`）在 CI 是綠的。 |
| 2026-09-20 | 🔍 **/ship 對抗式 CR**：0H／3M／8L，修 7、立案 4。最重要：沒報價的列在有模型價目表時會畫出「$0.31 無法估價」→ `candidateUsd` 第一條規則改成「選不到的列一律沒有金額」；文案不再承諾對話框裡做不到的「重新分析」；補上原本名不副實的群組徽章測試。web 3905/3905。 |
| 2026-09-20 | 🚧 **REVIEW**（dev-story, Amelia）。Task 1–6 全數完成。P1：`candidateUsd` 回 `number \| null`、`isSelectable`、`modelChoices` 涵蓋規則（預設模型有洞 → `[]`）；金額與三種分類標記中性；對話框 960；14 處 13px 與整張清單對稿；單季標題。一處偏離單子：群組徽章底用 `$bg-secondary`（標題列本身是 `$bg-tertiary`，同色看不見）。閘門：lint 0 errors、typecheck ✅、design-tokens ✅、web 3901/3901、api ✅；e2e 本機 5/7（2 條紅是 `AI_PROVIDER` 的 `model_id`，已看 diff 確認）。 |
| 2026-09-20 | 🔍 **建單後對抗驗證**（fresh-context 驗證代理，只讀；逐節點比對 `.pen`、逐行比對程式碼）：2 CRITICAL／14 SHOULD FIX／多項 NIT，**全部併入**。最重要：① `ConfirmGenerationDialog.spec.tsx:9` 的 `baseTotals` 是型別化的 `ConsentTotals`，加欄位不補它 typecheck 直接紅；② `consentSelection.spec.ts:326-329` 現有測試斷言的正是本張禁止的「靜默退回預設模型價」，要改寫而不是留著；③ `candidateUsd` 對不可寫入的列要**一律**回 null，否則同一列的排序位置會隨「模型清單載到了沒」改變；④ 預設模型沒通過涵蓋檢查時不能自動改用別的模型（會無聲換價）；⑤ `modelChoices` 的片長分母要維持 `isWritable`（後端就是這樣加總的）；⑥「開始產生停用的原因」那一行在超過上限時不會 render；⑦ 手機稿還有四組同類節點沒點名；⑧ 新註記不該塞進赭色的 `c4FIoB`；⑨ `consentRows.ts` 的行號與 `DESIGN.md` 的字階行號修正。 |
| 2026-09-20 | ⚖️ **Alexyu 裁定（建單提問）**：對話框整段固定 960；路線徽章與費用標記改中性。兩項建單裁定因此從「待確認」變成已定案。 |
| 2026-09-20 | Story 建立（SM Bob, create-story）。`dsr-6e` 依版面拆成 6e-1（本張：候選清單＋外殼＋P1）與 6e-2（F14／F16／F19／F20／F17）。建單稽核以 Pencil MCP 逐節點讀 F14–F20，找到 18 項現況問題；吸收 1 張 P1 與 3 張設計稿漂移單（lane ①），新立 1 張 disc。九項建單裁定，其中兩項（對話框寬度、路線徽章中性）待 Alexyu 確認。 |
