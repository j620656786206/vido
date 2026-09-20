# Story DSR.6e-2：產生字幕的分析中、金額確認、空狀態對齊設計稿——確認框只寫「約 $4.50」，總額不再變成橘色

Status: ready-for-dev

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As the person who presses 產生字幕 and then 開始產生,
I want 分析中、金額確認、沒有可做的項目這三個畫面都跟設計稿一樣清楚——金額是一般的字、取消鍵在我找得到的地方、螢幕報讀不會每秒念四次數字,
so that 在我按下「確認並開始」之前，我讀到的每個數字都只是一個數字，而不是一個被上色的警告。

## Context

`dsr-6e`（同意流程 F14–F20）拆出來的**第二塊**。**Depends on: `dsr-6e-1`**（同意對話框的外殼寬度在那張設定；本張的 F14／F20 內容排在那個外殼裡）。

| 單子 | 範圍 |
| --- | --- |
| `dsr-6e-1` | 候選清單 F15／F18＋對話框外殼＋P1 金額缺值 |
| **`dsr-6e-2`（本張）** | 分析中 F14-D-v2 `nBT3M`、金額確認 F16-D-v2 `gmOt6`、超出上限確認 F19-D-v2 `KThbY`、空狀態 F20-D-v2 `D7MOm`、掃描完成入口 F17-D-v2 `I3Wb0p`（**只改稿**）→ `AnalysisProgressPanel.tsx`、`ConfirmGenerationDialog.tsx`、`ModelPicker.tsx`、`ConsentEmptyState.tsx`、`GenerationConsentView.tsx`（只有空狀態頁尾那一段） |

帶入 dsr-6 原單子的 **②：F16 金額標籤只顯示「約 $4.5」**（Alexyu 2026-09-11 拿掉「這批…」後綴；稿上是「約 $4.50」，兩位小數是 `DESIGN.md:320` 的規定）。

⛔ **手機稿（F16-M `x45wBO`、F19-M `IMQO6`）屬 `dsr-6f`**：本張只改 `sm:` 以上的值，手機 sheet 的形狀不動。
⛔ **不進 `CandidateListPanel`／`consentRows`／`consentSelection`**（`dsr-6e-1`）。
⚠️ **驗收基準是 `.pen` 節點值**。PNG 只是參考。

### 🔴 建單時查到的事（main `c4d34187`；行號皆為現況）

**F14 分析中（`AnalysisProgressPanel.tsx`）**

1. **取消鍵的位置不對**：`:57-68` 放在內容正中間；稿是**獨立的頁尾**（`srMNm`：[14,24]、上框線、靠右、`Component/Button/Secondary` h44「取消」`tv6My`），跟 F15、F20 同一個位置。
2. **螢幕報讀每秒念四次**：`:45` 把 `aria-live="polite"` 掛在計數那一行上，而那一行由 SSE 每 250ms 更新一次（檔頭 `:4-5`）。容器已經有一個只在 phase 改變時說話的 `consent-phase-live`（`GenerationConsentView.tsx:538-548`），進度本身由 `role="progressbar"` 的 `aria-valuenow` 承載——計數行不該再是即時區域。
3. **取消中焦點會掉**：`:60` `disabled={cancelling}`——被聚焦的按鈕變成 disabled，焦點掉到 `<body>`（dsr-6c／dsr-6d-b 同一個問題，同一個修法：`aria-disabled`＋文案「取消中…」）。
4. **字級與排版**：說明 `:53` 13px（稿 `L0QAf` Body 14 `$text-secondary` 置中）；標籤 `:46` 是 400（稿 `QS5df` Body **600** 置中）；外層 `:27` `px-6 py-12 gap-5`（稿 `zqp9y` [48,40] gap 16）；軌道與填色 `:36`／`:40` 是 `radius-sm`（稿 `nggm5`／`xPKNe` 是 **pill**）。
5. **稿的寬度**：`h60Flj` 是 560，但 `dsr-6e-1` 建單裁定 2 把整個同意對話框定成 960（F14 與 F15 是同一個 `DialogContent`，照稿等於在使用者眼前撐大）。稿要改成 960，內容欄維持 480 寬置中。

**F16／F19 金額確認（`ConfirmGenerationDialog.tsx`、`ModelPicker.tsx`）**

6. **② 金額標籤**：`ModelPicker.tsx:102`「這批約 $X」；稿 `W1yhi`／`H4tpEe`／`Y9tGlo`／`Z4vD5` 是「**約 $4.50**」Body **700** mono `$text-primary`（程式碼是 13px／600）。
7. **合計穿了狀態色**：`:146-153` 超過上限時合計翻成 `--warning-text`；稿 F19 的合計 `CoYHC` 仍是 `$text-primary`（金錢是事實規則，`DESIGN.md:308`；⚖️ Alexyu 2026-09-10 裁定 B）。合計的字級也不對：程式碼 13px／600，稿是 **BodyLg 16／700** mono；「合計預估」標籤程式碼 600，稿 `iJlz7` **700**。
8. **明細的金額是灰的**：`:131-133`、`:140-142` 把「預估 $X」整串放在一個 mono span 裡，顏色繼承父層的 `--text-secondary`；稿是兩個節點——「預估」`J7ELF5` Body `$text-secondary`、金額 `CqONI` Body mono **`$text-primary`**。明細之間的間距程式碼 `gap-2`（8），稿 `KVGPG` gap 10。
9. **標題下那一句太小**：`:102`「即將為 N 部影片產生字幕」是 `text-sm` 400；稿 `hI6cN` 是 **BodyLg 16／600**。
10. **外框**：`:89` `sm:max-w-md`（448）、`sm:rounded-[var(--radius-xl)]`、沒有框線；稿 `S5mHWw`／`MYP82` 是 **480**、`$radius-lg`、1px `$border-subtle`。內容區 `:101` `sm:p-6`；稿 `b4DwH` [20,24] gap 16。
11. **13px**：`ConfirmGenerationDialog.tsx:119`（模型清單載入失敗那句）、`:125`（明細）、`:173`（提示框）；`ModelPicker.tsx:62`（「選擇翻譯模型」，稿 `i2Sdh` Body **600**，程式碼 500）、`:92`（模型名，稿 Body 400）、`:100`（金額）。共 6 處。
12. **「確認並開始」**`:211` `px-6 font-medium`；稿 `FzOyw` 是 `Component/Button/Primary`（14／**600**、padding [8,20]）。
13. **F19 提示框的字色三邊不一樣**：程式碼 `:175` 是 `--text-primary`；稿 F19-D `NMVfO` 是 `$text-secondary`；手機 F19-M `orgWi` 是 `$text-primary`；同一個「超過上限」的訊息在 F18 的橫幅 `tHo1i` 也是 `$text-primary`。F19-D 是唯一的例外 → 改稿。
14. **`gap-[3px]`** 三處：`:102`、`:127`、`:136`。

**F20 空狀態（`ConsentEmptyState.tsx`）**

15. **圖示與字級**：圓 `:52` 64（稿 `EqgR0` **96**）、圖示 `:55`／`:58` 32（稿 `A6t1tD` **36**）、`SquareCheck`（稿 `circle-check`）；標題 `:64`／`:73` `text-base`（稿 `jz9Yt` **H4 18／600**）；說明 `:67`／`:76` 13px（稿 `xnmxG` Body 14）；外層 `:48` `px-6 py-14 gap-4`（稿 `y84AZ` [48,32] gap 12）。
16. **「沒有找到可以產生字幕的項目」這一版沒有稿**：`disc-2026-08-consent-empty-state-asserted-false-completion`（done）為了不說謊加上的第二種空狀態，稿上只有「所有影片都有繁中字幕了」。字級照 F20 同一套，並在規格註記裡寫明它的存在（本張不另畫一張）。

**F17 掃描完成入口（只改稿）**

17. **F17 的稿還停在 dsr-5 之前**：`AobjC`「找到 1,247 檔案 · 比對成功 1,198 · 未比對 42 · 錯誤 7」、兩個連結「查看未比對項目」`jCfH4`／「查看錯誤」`q1QKm`、自動關閉進度條 `u6VL3` 是 `$accent-primary`。dsr-5 已經把 E3-D（`szzaW`）與程式碼改成「找到 1,247 檔案 · 新增 30 · 更新 1,168 · 無法匯入 42 · 錯誤 7」、**一個**連結「查看無法匯入與錯誤」、進度條 `$text-muted`（E3-D `NI87i`；倒數不是正在跑的工作）。F17 是同一張吐司多了「缺繁中字幕」那一行與「產生字幕 →」，那兩樣程式碼已經對（`ScanProgressCard.tsx:208-252`）。
18. **Rule 21 追溯缺一個**：`ScanProgressCard.tsx:1` 只寫 E2-D／E3-D，沒有 F17（缺字幕那一行與連結是 F17 畫的）。

**跨畫面**

19. **「品質 A」是分類，卻穿著青碧**：`ModelPicker.tsx:30-34` 的 `gradeTint` 給 A 級 `--success-tint`／`--success-text`，稿上 `xKKKz`（F16-D）／`VBvLY`（F19-D）／`zl86k`（F16-M）／`Msfhy`（F19-M）也是 `$success-tint`。品質等級是這個模型的**屬性**（而且下面還有一行說明它從哪來），不是「發生了什麼」。這是 sprint 條目 ③ 說的「全 app 共 4 處」的**第四處**——另外三處（列的路線徽章、群組的路線徽章、chip 的費用標記）在 `dsr-6e-1`。⚖️ Alexyu 2026-09-20 已裁定這一類一律中性。
20. **會花錢的按鈕沒有金額**：`DESIGN.md:314`「只要按鈕可以按，金額（含 `$`）就一定在場」。F16 的「確認並開始」、F19 的「仍要開始」、F15 的「開始產生」都沒有金額，稿也沒有——金額就在按鈕正上方的明細裡。這是規則與稿之間的問題，要 Sally／Alexyu 裁定，不在本張改（見 Discovery Triage ③）。

### 設計稿節點

| 代號 | 節點 | 本張要看的 |
| --- | --- | --- |
| F14-D-v2 | `nBT3M`（對話框 `h60Flj`，560 → 960） | 標題 `bsV9h`；body `zqp9y` [48,40] gap 16；進度群組 `ClRNE` gap 12：軌道 `nggm5`（h6 `$bg-tertiary` pill）／填色 `xPKNe`（`$accent-primary` pill）、標籤 `QS5df`（Body 600 置中「分析字幕軌 234 / 1,247」）；說明 `L0QAf`（Body `$text-secondary` 置中）；頁尾 `srMNm` [14,24] 靠右：`tv6My`（Secondary h44「取消」） |
| F16-D-v2 | `gmOt6`（對話框 `S5mHWw` 480、`$radius-lg`、1px `$border-subtle`） | 標題列 `Qbop7`／`Y8SDR`「確認產生字幕」；body `b4DwH` [20,24] gap 16；`hI6cN` BodyLg 600；模型 `auwq8` gap 8：`i2Sdh` Body 600、清單 `ZoGgC`（padding 6 gap 6 框 `$radius-md`）、列 `IYzk9`／`cLNkD`（[8,10] gap 4 `$radius-sm`，選中 `$bg-tertiary`）、名稱 `NusMp` Body、`（預設）` `Y6mj24` Label muted、金額 `W1yhi`／`H4tpEe`「約 $4.50」Body **700** mono、品質 `xKKKz`、分鐘 `wLlSG` Label mono `$text-secondary`、說明 `Z1ie5` Label muted；明細 `KVGPG` gap 10：`S8A3nL`／`FWlgm` Body `$text-secondary`、「預估」`J7ELF5`／`znUCO` Body `$text-secondary`、金額 `CqONI`／`tXHCJ` Body mono `$text-primary`、分隔線 `ElrY5`、合計 `iJlz7` Body **700**＋`tWF5e` BodyLg **700** mono；提示 `v34WID`（padding 12 `$bg-tertiary` `$radius-md`，`l5oicD` Body `$text-secondary`）；頁尾 `kjzT0` [14,24] gap 12：`aYn21` Secondary「取消」、`FzOyw` Primary「確認並開始」 |
| F19-D-v2 | `KThbY`（對話框 `MYP82` 480） | 同 F16，差在：`OeRlK`「即將為 96 部影片產生字幕」、Haiku 選中（`w5L1S3`）＋說明 `vSwAv`「比 Claude Sonnet 5 省 $16.08（62%）」、合計 `CoYHC` **`$text-primary`**、提示 `wal5S` `$warning-tint`＋`NMVfO`（**`$text-secondary` → 改 `$text-primary`**）、`wOwu7` Primary「仍要開始」 |
| F20-D-v2 | `D7MOm`（對話框 `WqnVQ` 960） | body `y84AZ` [48,32] gap 12 置中；`EqgR0`（96 圓 `$bg-tertiary`）＋`A6t1tD`（circle-check 36 `$success-text`）；`jz9Yt` H4；`xnmxG` Body `$text-secondary`；頁尾 `tDQhB` [14,24]：`VrrPl` Secondary「關閉」 |
| F17-D-v2 | `I3Wb0p`（吐司 `fL8DD` 480） | `AobjC`（統計）、`ZRgj7`（缺字幕那一行）、`THX2G` 內 `jCfH4`／`q1QKm`／`M6d0z`、`u6VL3`（倒數條）；對照 E3-D `szzaW` 的 `TSeyg`／`NDCHi`／`NI87i` |
| 規格群組 | Flow F `JzmvC`，既有註記 `c4FIoB` | AC #1 的註記 append 在這裡 |
| 母版 | `Component/Button/Secondary` `YDPhc`（h36 → 稿上覆寫 44、`$bg-tertiary`、14／500、padding [8,20]）、`Component/Button/Primary` `otvKh`（14／600） | 頁尾按鈕 |

字階（`DESIGN.md:355-358`）：H4 18／1.5、BodyLg 16／1.625、Body 14／1.625、Label 12／1.5。**行高不在本張**（`disc-2026-09-type-scale-even-migration`）。

---

## Acceptance Criteria

1. **設計稿先改（Pencil MCP；動手前每個節點再 `Get` 一次）。**
   - **F14 改成 960**（`dsr-6e-1` 建單裁定 2）：`h60Flj` `width: 960`；body `zqp9y` 加 `alignItems: center`，`ClRNE` 與 `L0QAf` 的 `width` 從 `fill_container` 改成 **480**（今天 560 − 左右各 40 的實際寬度），讓內容欄與今天一樣窄。改完 `TakeScreenshot` 確認對話框仍置中於 1440 的畫布上（`h60Flj` 的 `x` 可能要跟著改）。
   - **F19 提示字色**：`NMVfO` `fill` → `$text-primary`（🔴 #13）。
   - **品質徽章改中性**（🔴 #19；⚖️ Alexyu 2026-09-20 裁定）：`xKKKz`（F16-D）／`VBvLY`（F19-D）／`zl86k`（F16-M）／`Msfhy`（F19-M）的 `fill` → `$bg-tertiary`，裡面的 `xcJIE`／`nUvTU`／`ZwUz4`／`QnlRg` `fill` → `$text-secondary`；`cornerRadius` → `$radius-pill`（`DESIGN.md:566-568`：徽章是資料標記＝藥丸，`radius-sm` 只給不可互動的小色標）。B 級那幾顆本來就是中性，只改圓角。
   - **F17 對齊 E3-D**（🔴 #17）：`AobjC` 內容 →「找到 1,247 檔案 · 新增 30 · 更新 1,168 · 無法匯入 42 · 錯誤 7」；`jCfH4` 內容 →「查看無法匯入與錯誤」（`name` **不改**——E3-D 對應的那顆 `NDCHi` 名字也還是 `actionUnmatched`，兩邊保持一致比各自改名好）；`Delete("q1QKm")`；`u6VL3` `fill` → `$text-muted`（`opacity` 0.6 保留，與 `NI87i` 一致）；**吐司圖示 `jvPNk` → `triangle-alert`、`fill: $warning-text`**（🔴 #12 的延伸：統計裡有「無法匯入 42 · 錯誤 7」，E3-D 的 `9u1DU` 與程式碼 `ScanProgressCard.tsx:183-185` 在有問題時畫的都是赭色三角，綠勾會說這次掃描一切順利）。`ZRgj7`「142 部影片缺繁中字幕」與 `M6d0z`「產生字幕 →」不動。
   - **補一段註記**：在**規格群組 `JzmvC` 裡新增一個中性的 text 節點**（像 `dufQr`：`$Type/Label/*`＋`fill: $text-muted`；⛔ 不要 append 進赭色的 `c4FIoB`，`DESIGN.md:253` 說後設註記不得穿赭——見 `dsr-6e-1` 立的 `disc-2026-09-pen-meta-notes-wear-ochre`；位置放在 6e-1 那顆新註記下方 40px）：「F20 另有一種空狀態沒有畫出來：分析有結果但沒有任何一部能抽取或做語音辨識時（例如只有影集的片庫），標題是『沒有找到可以產生字幕的項目』、圖示是灰色的 search-x、說明寫分析了幾部——它不能說『都有字幕了』，因為那不是真的。F16／F19 的合計下方另有一行『其中 N 部片長未知，以 45 分鐘估算』（sub-6-12），稿上沒畫，只有在真的有估算列時才出現。F14 的『分析字幕軌 N / M』不是即時播報區，進度由進度條承載。F17 的缺字幕數字用等寬字，以程式碼為準。」
   - 新節點綁變數；`ctx.problems` 掃裁切；存檔後確認 ` M ux-design.pen` 並讀磁碟檔確認落盤；匯出後只 stage `flow-f-subtitle-v2/{f14-d-v2,f16-d-v2,f16-m-v2,f17-d-v2,f19-d-v2,f19-m-v2}.png`（＋`pen-tokens.json` 若變），其餘重繪雜訊還原。**若 `f20-d-v2` 或 `flow-e-scanner/*` 變了，代表動到了不該動的節點——回頭檢查。**（F20 的稿本來就對，本張只改它的程式碼。）

2. **F14 分析中（🔴 #1–#4）。**
   - 元件改成回傳**兩塊**：內容（`flex flex-1 flex-col items-center justify-center px-10 py-12`，內層 `flex w-full max-w-[480px] flex-col gap-4`，其中進度群組 `flex flex-col gap-3`）與**頁尾**（`flex shrink-0 items-center justify-end border-t border-[var(--border-subtle)] px-6 py-3.5`，跟 `GenerationConsentView.tsx:593` 空狀態的頁尾同一組 class）。取消鍵搬進頁尾，樣式不變（Secondary、`min-h-[44px] px-5`）。⚠️ 容器 `GenerationConsentView.tsx:550-557` 把它直接放在 `DialogContent` 的 flex 欄裡——元件用 fragment 回傳兩個兄弟節點即可，容器不用改。
   - 標籤 `:46`：`text-sm font-semibold text-[var(--text-primary)]`（計數的 mono span 保留）；**拿掉 `aria-live="polite"`**（🔴 #2）。`role="progressbar"` 的 `aria-valuenow`／`aria-valuemax`／`aria-label` 不動。
   - 說明 `:53`：`text-sm`。軌道 `:36` 與填色 `:40`：`rounded-full`。
   - 取消中（🔴 #3）：把 `disabled={cancelling}` 換成 `aria-disabled={cancelling}`，送出中文案「取消中…」、`onClick` 在 `cancelling` 時直接 return；`Loader2` 保留。⛔ 不要用 `disabled`。⚠️ 變灰也要跟著換 variant：`:62` 的 `disabled:cursor-not-allowed disabled:opacity-50` 在 `aria-disabled` 下**完全不會生效**，改成 `aria-disabled:cursor-not-allowed aria-disabled:opacity-50`（先例 `GlossaryRowV2.tsx:155`）。
   - **測試掛鉤**：`data-testid="consent-analysis-panel"` **留在內容那一塊**（`GenerationConsentView.spec.tsx:193`、`:203`、`:485`、`:499` 都靠它找畫面）；頁尾另給 `data-testid="consent-analysis-footer"`。
   - `:1` 的 Design ref 不變。

3. **F16／F19 金額確認（🔴 #6–#12、#14）。只改 `sm:` 以上。**
   - 外框 `:89`：`sm:max-w-md` → `sm:max-w-[480px]`；`sm:rounded-[var(--radius-xl)]` → `sm:rounded-[var(--radius-lg)]`；加 `sm:border sm:border-[var(--border-subtle)]`。
   - 內容區 `:101`：`sm:p-6` → `sm:px-6 sm:py-5`（`gap-4` 不動）。
   - 開頭那句 `:102`：`flex items-center gap-1 text-base font-semibold text-[var(--text-primary)]`（mono 計數保留）。
   - 模型清單載入失敗 `:119`：`text-sm`（**維持 `--warning-text`**：你要選模型、而這件事沒發生，正是赭的定義）。
   - 明細 `:125`：`flex flex-col gap-2.5 text-sm text-[var(--text-secondary)]`。兩列的金額 `:131-133`、`:140-142` 拆成兩個節點：`<span className="flex items-center gap-1.5">預估<span className="font-mono tabular-nums text-[var(--text-primary)]" data-testid="consent-confirm-asr-usd">{usd(…)}</span></span>`（`data-testid` 移到**只包金額**的那個 span，抽取那一列同理 `consent-confirm-extract-usd`）。`gap-[3px]` `:127`／`:136` → `gap-1`。
   - 合計 `:144-155`：標籤 `font-bold`；金額 `font-mono text-base font-bold tabular-nums text-[var(--text-primary)]`，**拿掉 `overBudget` 的赭色分支**（🔴 #7）。分隔線維持在這一列的 `border-t`，`pt-2` → `pt-2.5`。
   - 「其中 N 部片長未知，以 45 分鐘估算」`:162-167`（sub-6-12，稿上沒有）維持 `text-xs`，不動。
   - 提示框 `:170-178`：`text-[13px]` → `text-sm`；兩個分支的顏色**不動**（F16 `--bg-tertiary`＋`--text-secondary`；F19 `--warning-tint`＋`--text-primary`，稿依 AC #1 改成一致）。
   - 「確認並開始／仍要開始」`:211`：`px-6 font-medium` → `px-5 font-semibold`；`min-h-[44px]` 保留。「取消」`:202` 不動。
   - `ModelPicker.tsx`：`:62` → `text-sm font-semibold`；`:92` → `text-sm`；`:100-103` → `text-sm font-bold`，文字**「約 {usd(choice.totalUsd)}」**（② 拿掉「這批」）。**品質徽章改中性**（🔴 #19）：`gradeTint`（`:30-34`）不論等級一律 `bg-[var(--bg-tertiary)] text-[var(--text-secondary)]`（沒有等級時維持 `--text-muted`），徽章與「尚未評測」`:119-125` 的 `rounded-[var(--radius-sm)]` → `rounded-full`；檔頭 `:8-12` 那段「the MEASURED quality grade」的說明保留，但把「只有 MEASURED grade 才有顏色」那個意思改掉（顏色已經不承載這件事，`aria-label` 與文字才是）。`（預設）`、分鐘、說明不動。
   - **改掉會變成假話的註解**：`ConfirmGenerationDialog.tsx:5-7` 寫「Over-budget flips the tint to warning … **the total to warning orange**」——合計改中性之後後半句就是假的。
   - ⛔ 標題列、手機 sheet（`:84-95` 非 `sm:` 的那些 class）、`ModelPicker` 的 radio 行為、`selectionNote` 的文案都不動。

4. **F20 空狀態（🔴 #15、#16）。**
   - `ConsentEmptyState.tsx:48`：`px-8 py-12 gap-3`（`text-center` 保留）。
   - `:52`：`h-24 w-24`；`:55`／`:58`：`h-9 w-9`；`SquareCheck` → `CircleCheck`（lucide；`SearchX` 不變）。⚠️ spec `:28`／`:31` 用 `.text-[var(--success-text)]` 找圖示——class 保留就不會紅。
   - `:64`／`:73`：`text-lg font-semibold`；`:67`／`:76`：`text-sm`（`max-w-sm` 保留）。
   - 頁尾（`GenerationConsentView.tsx:593-602`）不動——已經對稿。

5. **F17 追溯（🔴 #18）。** `ScanProgressCard.tsx:1` → `// Design ref: ux-design.pen Screen E2-D (wyuhF) · E3-D (szzaW) · F17-D-v2 (I3Wb0p)`。其餘程式碼不動（已對稿；缺字幕的數字用 mono 是程式碼的慣例，以程式碼為準）。

6. **既有的行為不准回歸**（測試都要留著）：
   - F14：分析中每 5 秒輪詢一次當作漏掉 `ready` 的保險（`GenerationConsentView.spec.tsx:503`）；取消後關閉（`:475`；取消失敗也關閉——分析是免費、在本機跑，繼續跑完也無害）；409 是加入不是錯誤（`:196`）。
   - F16／F19：確認框在開始送出中保持開啟、失敗才關（`:524`，CR L8）；送出中模型清單鎖住（`ModelPicker.spec.tsx:123`）；換模型連動摘要列、頁尾與合計（`:299`、`:338`）；「品質最穩」只給真的拿到最高分的預設列（`ModelPicker.spec.tsx:100`）；目錄載入失敗與沒設金鑰是兩件事（`:435`、`:448`）；「仍要開始／確認並開始」不因換模型而改變（`ConfirmGenerationDialog.spec.tsx:134`）。
   - F20：沒有 props 時**不**宣稱都有字幕了；綠色勾勾只在真的都有時出現（`ConsentEmptyState.spec.tsx:16`、`:26`）。

7. **測試（先寫紅測試）。**
   - **新增** `AnalysisProgressPanel.spec.tsx`（今天沒有）：計數那一行**沒有** `aria-live`；進度條的 `aria-valuenow`／`aria-valuemax`；取消鍵在 `consent-analysis-footer` 裡；`cancelling` 時按鈕是 `aria-disabled="true"`、**不是** `disabled`、文案「取消中…」、再點不會呼叫 `onCancel`、而且帶 `aria-disabled:opacity-50`；軌道與填色是 `rounded-full`。
   - `ModelPicker.spec.tsx`：A 級徽章的 class **沒有** `--success-tint`／`--success-text`（品質徽章中性，🔴 #19）；`:100` 那條「品質最穩」的斷言不動（那是文字，不是顏色）。
   - `ModelPicker.spec.tsx:51`、`:58`：斷言改成「約 $0.53」「約 $0.21」，並**另加** `not.toHaveTextContent('這批')`——⚠️ `toHaveTextContent` 是子字串比對，「這批約 $0.53」本來就包含「約 $0.53」，只改字串的話舊文案照樣會過。
   - `ConfirmGenerationDialog.spec.tsx`：F19（`overBudget: true`）的合計金額 `className` **沒有** `--warning-text` 而有 `--text-primary`；明細金額是 `--text-primary`；開頭那句是 `text-base font-semibold`；F19 提示框仍是 `--warning-tint`。`:76-77` 今天是 `.textContent).toContain('$4.32')`／`('$0.18')`（testid 包著「預估 $4.32」，所以用 `toContain`）——testid 移到只包金額的 span 之後，改成 `.toBe('$4.32')`／`.toBe('$0.18')`，讓這次移位有測試守著（否則舊寫法照樣過）。
   - `ConsentEmptyState.spec.tsx`：兩種狀態的標題都是 `text-lg`；既有四條不動。
   - `GenerationConsentView.spec.tsx`：F14 的取消鍵在 `consent-analysis-footer` 裡（`consent-analysis-panel` 仍然存在、仍然是內容那一塊）；**保留不動**：AC #6 列的那些。

8. **視覺夾具與基準線。**
   - `generation-consent/analyzing` 與 `generation-consent/empty` 的 `width: 720` → `960`（對話框寬度）。
   - **新增** `generation-consent/empty-all-covered`（`ConsentEmptyState`，`props: { allCovered: true, analyzed: 1247 }`，`width: 960`，`penNode: 'screen-section'` 註記 F20-D-v2 `D7MOm`）——今天唯一的 `empty` 夾具畫的是「沒有找到」那一版，而稿 F20 畫的是「都有字幕了」，沒有夾具可以做 Step 9 比對。
   - 重生 darwin：`analyzing`、`empty`、`empty-all-covered`（新）、`confirm`、`confirm-over-budget`、`f16-model-default`、`f16-model-haiku`、`f19-over-budget-haiku`。照 `project_visual_baseline_intentional_change` 的四步：`nx serve web` → `CI=1 npx playwright test --project=visual --update-snapshots --grep generation-consent` → `git rm` 過期的 `-linux` → 併 main → `gh workflow run "Visual Regression" --ref <branch>`。⛔ 不要本機產 `-linux.png`。
   - ⚠️ 新夾具 `empty-all-covered` 只畫得出 `ConsentEmptyState` 本身——標題列與「關閉」頁尾住在容器（`GenerationConsentView`）裡，所以它跟 F20 比對時**只能比中間那一塊**。
   - ⚠️ `confirm`／`confirm-over-budget` 兩個夾具的 `totals` 是手寫的部分物件（`-gallery.fixtures.tsx:4820-4860`），typecheck 抓不到漏欄位；改完一定要看圖。

9. **另立的單子（建單時已寫入 sprint-status，不在本張做）。**
   - 新：`disc-2026-09-batch-consent-buttons-without-amount`（🔴 #20）。
   - 既有：`disc-2026-09-dialog-close-target-44px`、`disc-2026-09-dialogframe-shadow-vs-shadow-xl`、`disc-2026-09-type-scale-even-migration`、`disc-2026-09-generation-resume-c-estimate-deduction`（續跑後單部估價扣減）、`sub-7-8-model-ratings-feed`（「可花約 $0.01 試跑 20 句」變成真的按鈕是那張的事）。

10. **CI 全綠**：`pnpm run lint:all`、`pnpm nx run web:typecheck --skip-nx-cache`、`python3 scripts/check-design-tokens.py`、`pnpm nx test web`、`pnpm nx test api`。e2e `pnpm run test:e2e -- --grep @batch-subtitle`（同意流程走到 `consent-confirm-start`，`tests/e2e/batch-subtitle.spec.ts:258-262`）本機跑一次，結果寫進 Completion Notes。⛔ 絕不用 `run_in_background` 跑測試；每次跑完 `pnpm run test:cleanup`。Nx daemon 建圖逾時就 `NX_DAEMON=false`。

## Tasks / Subtasks

- [ ] **Task 1 — 設計稿（AC: #1）**
  - [ ] F14 960＋內容欄 480、F19 提示字色、F17 對齊 E3-D、補註記
  - [ ] `ctx.problems`；存檔並確認落盤；匯出後只 stage f14-d-v2／f17-d-v2／f19-d-v2（＋pen-tokens）
- [ ] **Task 2 — F14（AC: #2, #7 第一項）**
  - [ ] 先寫紅測試 `AnalysisProgressPanel.spec.tsx` → 頁尾、`aria-live`、`aria-disabled`、pill、字級
- [ ] **Task 3 — F16／F19（AC: #3, #7）**
  - [ ] 先寫紅測試（合計中性、明細金額、「約 $X」不含「這批」）→ `ConfirmGenerationDialog.tsx`、`ModelPicker.tsx`
- [ ] **Task 4 — F20＋F17 追溯（AC: #4, #5）**
- [ ] **Task 5 — 夾具與基準線（AC: #8）**
- [ ] **Task 6 — 收尾（AC: #6, #9, #10）**
  - [ ] 全套閘門、e2e；dev-story Step 9 截圖比對（`f14-d-v2`、`f16-d-v2`、`f19-d-v2`、`f20-d-v2`）

## Dev Notes

### 這張的重點

- **確認框是花錢前的最後一個畫面**：這裡的數字最不該被上色、也最不該寫得比稿更囉嗦。②「這批約」→「約」是 Alexyu 親自拿掉的字。
- **F14 的兩個修正都是給看不到畫面的人的**：計數行不再每 250ms 被念一次；取消中焦點不再掉到 `<body>`。
- **F17 只改稿**：dsr-5 改了掃描吐司，但同一張吐司在 Flow F 還有一份複本沒跟上。

### 上游契約（Rule 20 ack）

- confirmed against [@contract-v1] (Story sub-4-1 AC #8) —— `generation_candidates_progress` SSE 的 `{status, analyzed, total}`（F14 只改呈現，不改消費）。
- confirmed against [@contract-v1] (Story sub-6-8a AC #2) —— `GET /api/v1/settings/models`（`ModelPicker` 只改字級與文案）。

### 建單裁定（2026-09-20，Sally／Alexyu 可在 review 推翻）

1. **F14 的稿改成 960、內容欄 480**——承接 `dsr-6e-1` 建單裁定 2（✅ ⚖️ Alexyu 2026-09-20 已裁定「整段固定 960」）。
2. **F19 提示框用 `--text-primary`**，改稿不改程式碼：F18 橫幅、F19-M 都是 `$text-primary`，只有 F19-D 例外。
3. **模型清單載入失敗那一句維持赭**：使用者要的「選模型」沒有發生、這批會照預設模型收費——這是現在的狀態，不是事前警語（`DESIGN.md:250-252`）。
4. **F20 的「沒有找到」版本不另畫一張稿**，用註記說明；字級照 F20。
5. **F14 取消失敗仍然關閉對話框**（不改）：分析是免費、本機執行，背景繼續跑完也無害，下次打開會直接接上結果。
6. **品質徽章改中性**（🔴 #19）：⚖️ Alexyu 2026-09-20 對「分類不穿狀態色」的裁定涵蓋四處，這是第四處；另外三處在 `dsr-6e-1`。等級的差別由字（品質 A／品質 B／尚未評測）與下面那行出處承載。
7. **會花錢的按鈕要不要帶金額**不在本張決定（🔴 #20，另立單子）。

### 不要做的事

- **不要進 `CandidateListPanel`／`consentRows`／`consentSelection`**（`dsr-6e-1`）。
- **不要改同意對話框的外殼寬度**（`GenerationConsentView.tsx:525`，`dsr-6e-1` 已改）。
- **不要做手機版**（F16-M／F19-M 歸 `dsr-6f`）；`ConfirmGenerationDialog` 裡非 `sm:` 的 class 一律不動。
- **不要改 `ui/Dialog.tsx`**。
- **不要改 `ScanProgressCard` 的行為與樣式**，只補標頭。
- **不要把合計或任何金額塗回赭色**來表達超過上限——那由提示框與「仍要開始」承載。

### 已知陷阱

- **`toHaveTextContent` 是子字串比對**：「這批約 $0.53」包含「約 $0.53」。
- **`data-testid` 移位**：`consent-confirm-asr-usd`／`-extract-usd` 今天包著「預估 $X」，改完只包金額——grep spec 與 e2e 有沒有人斷言那段文字。
- **`AnalysisProgressPanel` 回傳 fragment**：夾具 `generation-consent/analyzing` 直接 render 它，頁尾會一起畫進基準線（這是要的）。
- **Radix Dialog 走 Portal**：spec 用 `screen` 找。
- **行號以建單時為準**（2026-09-20，main `c4d34187`）；`dsr-6e-1` 合併後 `GenerationConsentView.tsx` 的行號會位移。

### Source tree

```
apps/web/src/components/subtitle/consent/AnalysisProgressPanel.tsx(+spec 新)  ← Task 2
apps/web/src/components/subtitle/consent/ConfirmGenerationDialog.tsx(+spec)   ← Task 3
apps/web/src/components/subtitle/consent/ModelPicker.tsx(+spec)               ← Task 3
apps/web/src/components/subtitle/consent/ConsentEmptyState.tsx(+spec)         ← Task 4
apps/web/src/components/subtitle/consent/GenerationConsentView.spec.tsx       ← Task 2（F14 頁尾的斷言）
apps/web/src/components/scanner/ScanProgressCard.tsx                          ← Task 4（只改標頭）
apps/web/src/routes/test/-gallery.fixtures.tsx                                ← Task 5
tests/visual/components.visual.spec.ts-snapshots/components/generation-consent/** ← Task 5
ux-design.pen、_bmad-output/pen-tokens.json、_bmad-output/screenshots/flow-f-subtitle-v2/{f14-d-v2,f17-d-v2,f19-d-v2}.png ← Task 1
```

### Cross-Stack Split Check

後端 task **0 個**，前端／設計 task 6 個。→ 本張不拆。

### Time-dependent visual coverage

- **N/A — no wall-clock-reading components touched.** `AnalysisProgressPanel`（檔頭：zero wall-clock reads）、`ConfirmGenerationDialog`、`ModelPicker`、`ConsentEmptyState` 都不讀時鐘。`ScanProgressCard` 有 10 秒自動關閉，但本張只改它的標頭註解。

### References

- [Source: `apps/web/src/components/subtitle/consent/AnalysisProgressPanel.tsx:1-71`、`ConfirmGenerationDialog.tsx:1-225`、`ModelPicker.tsx:1-153`、`ConsentEmptyState.tsx:1-89`]
- [Source: `apps/web/src/components/subtitle/consent/GenerationConsentView.tsx:476-488, 537-557, 590-604`]
- [Source: `apps/web/src/components/scanner/ScanProgressCard.tsx:1, 180-252`]
- [Source: `apps/web/src/routes/test/-gallery.fixtures.tsx:4580-4589, 4806-4935`、`tests/e2e/batch-subtitle.spec.ts:258-262`]
- [Source: `ux-design.pen` `nBT3M`／`h60Flj`／`gmOt6`／`S5mHWw`／`KThbY`／`MYP82`／`D7MOm`／`WqnVQ`／`I3Wb0p`／`fL8DD`／`szzaW`／`c4FIoB`／`YDPhc`／`otvKh` —— SM 建單時以 Pencil MCP 逐節點讀出（2026-09-20）]
- [Source: `DESIGN.md:250-253`（赭的定義、2026-09-11 裁定、後設註記不得穿赭）、`:308-322`（金錢是事實、會花錢的動作、`:314` 沒有金額就沒有按鈕、金額的寫法）、`:355-358`（字階）、`:566-568`（圓角新規則）、`:716-720`（TechBadge 中性裁定）]
- [Source: `sprint-status.yaml` → `dsr-6-flow-f-subtitle-v2`（② 的歸屬）、`dsr-6e-consent-flow`、`dsr-6d-b-batch-dialog`（寬度交接）、`disc-2026-08-consent-empty-state-asserted-false-completion`、`dsr-5-flow-e-scanner`（E3-D 的改動）]
- [Source: `_bmad-output/implementation-artifacts/dsr-6e-1-consent-list.md`（建單裁定 2）、`dsr-6d-b-batch-dialog.md`（`aria-disabled` 與逐字斷言的做法）、`sub-6-8b-per-run-model-selection-frontend.md`、`sub-4-3-cost-consent-frontend.md`]
- [Source: project-context.md#Rule 5 / #Rule 16 / #Rule 20 / #Rule 21 / #Rule 23 / #Rule 24]

## Dev Agent Record

### Agent Model Used

{{agent_model_name_version}}

### Debug Log References

### Completion Notes List

### Discovery Triage

- **Did this story discover any work outside its current scope?** **YES** —— 建單當下處理如下。

- **① expand-scope-in-place**
  - F14 計數行每 250ms 被螢幕報讀念一次（🔴 #2）→ **AC #2**
  - F14 取消中焦點掉到 `<body>`（🔴 #3）→ **AC #2**
  - F19-D 提示字色與 F18／F19-M 不一致（🔴 #13）→ **AC #1**
  - F20「沒有找到」版本沒有稿、也沒有能對 F20 的夾具（🔴 #16）→ **AC #1（註記）、#8（新夾具）**
  - F17 稿停在 dsr-5 之前（🔴 #17）→ **AC #1**
  - `ScanProgressCard` 的 Rule 21 標頭少了 F17（🔴 #18）→ **AC #5**

- **② spawn-blocking-story**：無（`dsr-6e-1` 是排序上的依賴，不是阻擋）。

- **① expand-scope-in-place（續）**
  - 品質徽章穿青碧（🔴 #19，⚖️ Alexyu 2026-09-20 裁定涵蓋的第四處）→ **AC #1、#3、#7**
  - F17 的吐司圖示是綠勾，但統計裡有「無法匯入 42 · 錯誤 7」→ **AC #1**

- **③ backlog-with-carry-forward-link**（建單時立）
  - `disc-2026-09-batch-consent-buttons-without-amount` — 「開始產生」「確認並開始」「仍要開始」都沒有帶金額，與 `DESIGN.md:314` 不一致（🔴 #20）

- Reference: `project-context.md` Rule 24

### File List

## Change Log

| 日期 | 內容 |
| --- | --- |
| 2026-09-20 | 🔍 **建單後對抗驗證**（fresh-context 驗證代理，只讀；逐節點比對 `.pen`、逐行比對程式碼）：本張的改動併入 4 項——① `aria-disabled` 換上去之後 `disabled:opacity-50` 完全失效，要改成 `aria-disabled:` variant；② `consent-analysis-panel` 這個 testid 有四條既有測試在用，不能跟著搬；③ F17 的綠勾與「無法匯入 42 · 錯誤 7」互相矛盾（E3-D 與程式碼都是赭色三角）；④ F16／F19 的「品質 A」青碧徽章就是 sprint 條目說的「4 處」的第四處，本張一起收。另修 `DESIGN.md` 字階行號與匯出清單（動到 F16-M／F19-M 的徽章）。 |
| 2026-09-20 | ⚖️ **Alexyu 裁定（建單提問）**：對話框整段固定 960（F14 的稿改 960、內容欄 480）；分類徽章一律中性（本張的品質徽章）。 |
| 2026-09-20 | Story 建立（SM Bob, create-story）。`dsr-6e` 的第二塊：F14／F16／F19／F20 的程式碼對齊，F17 只改稿。找到 19 項現況問題（其中兩項是給螢幕報讀的：計數行每 250ms 被念一次、取消中焦點掉到 body）；帶入 dsr-6 的 ②（「約 $4.50」）。新立 1 張 disc。依賴 `dsr-6e-1`。 |
