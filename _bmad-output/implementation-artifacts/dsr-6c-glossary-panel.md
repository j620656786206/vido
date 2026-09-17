# Story DSR.6c：名詞對照表對齊設計稿，順手修掉「存失敗不講」「Esc 關掉整個面板」「打中文按 Enter 送出半個字」

Status: review

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As the person who opens 「名詞對照表」 from 管理字幕 to fix how a show's names get translated,
I want 面板長得跟設計稿一樣、來源標籤不再亂用狀態色，而且新增／編輯／確認／刪除失敗時會老實說、用注音輸入時按 Enter 選字不會把半個字送出去,
so that 我改過的譯名真的有存到，沒存到我也會知道，下一次生成字幕就會用對的名字。

## Context

`epic-dsr` 的 `dsr-6`（Flow F 字幕）拆出來的**第三張**。前兩張已完成：`dsr-6a`（付費按鈕帶金額，PR #455）、`dsr-6b`（管理字幕對話框桌機，PR #458——它把分集的名詞面板改成收**整部劇的 id**，本張不用再處理 id）。

| 單子 | 範圍 |
| --- | --- |
| `dsr-6b` ✅ | 管理字幕對話框桌機 F1–F5＋進度母版 |
| **`dsr-6c`（本張）** | 名詞對照表：F6-D-v2 `dlfMR`、F7-D-v2 `A85GFD`、母版 `Component/GlossaryRow-v2` `nDSEd`、規格條 `r7rxg0` |
| `dsr-6d` | 批次生成＋生成工作區 |
| `dsr-6e` | 同意流程 F14–F20 |
| `dsr-6f` | Flow F 所有手機稿（含 F6-M-v2 `buepS`） |

⚠️ **驗收基準是 `.pen` 節點值**（文字逐字、字級變數、顏色變數、間距）。PNG 只是參考。

### 🔴 建單時查到的事

三個唯讀稽核代理（2026-09-17，main `70462433`）：一個用 Pencil MCP 逐節點讀 F6／F7／`nDSEd`／`r7rxg0`，一個讀程式碼與測試，一個查 DESIGN.md 規則與歷來裁定。

1. **新增、編輯、確認、全部確認、刪除失敗時，畫面什麼都不說（真的 bug）。** `useGlossary.ts:29-55` 五個 mutation 只有 `onSuccess: invalidate`，沒有 `onError`；全域也沒有 `MutationCache`（`queryClient.ts:5-12`）。面板只顯示「列表載入失敗」（`GlossaryPanelV2.tsx:163-181`）。更糟的是**編輯**：`GlossaryRowV2.tsx:54-58` 的 `saveEdit` 先離開編輯模式、才等結果——存失敗時那一列默默變回舊譯名，使用者打的字不見了，也不知道沒存到。
2. **編輯某一列時按 Esc，整個名詞面板被關掉（真的 bug）。** 該列的輸入框本來要用 Esc 取消編輯（`GlossaryRowV2.tsx:76-79`），但 Radix Dialog 在 `document` 的 capture 階段先收到 Esc，面板是最上層，就直接關掉（`@radix-ui/react-use-escape-keydown`、dismissable-layer）。`GlossaryPanelV2` 沒有傳 `onEscapeKeyDown`。`GlossaryRowV2.spec` 把列渲染在 Dialog 外面，所以測不到。
3. **用注音（或任何輸入法）打譯名、按 Enter 選字，會把還沒選完的字送出去（真的 bug）。** 三個 Enter 送出點（`GlossaryPanelV2.tsx:79, 89`、`GlossaryRowV2.tsx:75`）都沒檢查輸入法組字狀態；全 `apps/web/src` 找不到任何 `isComposing`。Chrome／Firefox 在組字中按 Enter 會帶 `isComposing: true`，Safari 在確認選字那一下是 `keyCode 229`。這個面板**唯一的用途就是打中文譯名**。
4. **新增一個「已經在表裡」的詞，會默默蓋掉舊的。** 新增走 `Upsert`（`glossary_service.go:77` → `glossary_repository.go:128-152`，唯一鍵 `(scope, term_src COLLATE NOCASE, language)`，`migrations/036…go:68-69`），撞到時覆寫 `term_zh`、把 `source` 改成手動、`confirmed` 改成 true，回 201。畫面沒有任何提示——例如「官方字幕」來源的詞被打錯的新譯名蓋掉，來源也跟著不見。
5. **Enter 不看 `busy`。** `submitAdd`（`:53-68`）與 `saveEdit` 都沒檢查 `busy`，連按兩下 Enter 會送兩次（upsert 讓資料不壞，但會多跑一次）。
6. **背景重抓失敗，已經載入的列表整個被換成錯誤框。** `:161-163` 是 `isLoading ? … : isError ? … : …`；有資料時重抓失敗，列表消失、但「共 N 條」footer、「全部確認」「新增詞彙」還在（它們看 `list.length`）。
7. **小狀態問題。** 空狀態下按「新增詞彙」，表單和「尚無詞彙」插圖同時出現（`:158` 與 `:183-198`）；「取消」不清空草稿（`:107`）；關掉面板再打開，表單還開著、草稿還在；表單打開時游標不會跳進第一個輸入框。
8. **來源徽章亂用狀態色，而且設計稿自己打架。** 程式碼 `GlossaryRowV2.tsx:17-29`：字幕＝靛青、中繼資料＝泥金、官方字幕＝青碧、社群＝赭、手動＝中性。DESIGN.md:300「狀態色不得被挪用為強調、裝飾或分類」、:652「`outline`／`secondary`…用於分類而非狀態」、:716＋⚖️ 2026-09-10 TechBadge 裁定「屬性不是『發生了什麼』，本來就不該穿狀態色」。稿的 F6 把中繼資料畫成泥金（`P5j6O`、`T8nYpg`、F6-M `ngusL` 是 instance 覆寫），規格條 `r7rxg0` 卻畫成青碧（`L2XAH`）；**官方字幕、社群兩種整份稿都沒畫**（今天也沒有任何程式會寫入這兩種來源——`sub-7-5`、`sub-8-1` 都還是 ready-for-dev）。`GlossaryRowV2.tsx:21-23` 的註解把青碧當「最可信」的信任等級，正是 DESIGN.md:249「絕不當作一般的肯定」禁止的用法。
9. **徽章形狀也錯。** DESIGN.md:568／:650（⚖️ 2026-09-10 `disc-2026-09-radius-pill-rule-vs-reality`）：徽章＝藥丸。母版 `nDSEd` 與程式碼（`:95`、`:105`）都是 `radius-sm`。同一個詳情頁的 `DetailTechInfoV2.tsx:55` 與管理字幕對話框 `:393` 已經是 `rounded-full`。
10. **設計稿與程式碼的其他差異**（全表見 AC #3、#4、#5）：面板寬 880（程式碼 768）、圓角 `$radius-lg`（程式碼繼承 `ui/Dialog.tsx:60` 的 16px）、髮絲框（程式碼沒有）、內容 padding [20,24]、兩顆 Secondary 按鈕**沒有圖示**、刪除是紅字「刪除」不是垃圾桶圖示、16 處 `text-[13px]` 要收成 Body 14、footer 範例字「共 12 條 · 6 條未確認」跟畫出來的 6 列 3 條未確認對不上、「Eleven」那列的譯名被拆成寫死 13px 的「11」＋「號」（`eyc3W`）。
11. **程式碼有、稿沒畫的狀態**：新增表單、編輯中、刪除確認、載入骨架、載入失敗。本張還要新增「寫入失敗」與「重複的詞」兩個提示——**都要先有稿**（`feedback_pencil_spec_standalone_screen`：狀態規格獨立成一張畫面）。
12. **確認過的列要不要有「編輯」——稿與程式碼不同。** F6-D-v2 把三列已確認的 `SIp0F` 關掉，只剩「刪除」；程式碼每列都有「編輯」。查證（詳見建單裁定 ②）：後端允許編輯任何詞、任何自動流程都不會覆寫既有詞（全是 insert-if-absent）、sub-5-5 AC 明文「髒詞條由 F6 審核流修正」；拿掉「編輯」只會逼人刪掉重打、而且重打會把來源變成「手動」。F6-M 連**未確認**列的「編輯」也關了，看起來是 390 寬塞不下的權宜，不是產品規則。
13. **11px 凍結。** `disc-2026-09-11px-micro-label-not-on-type-scale`：兩個徽章的 `text-[11px]`（`GlossaryRowV2.tsx:95, 105`）**原樣保留**，只改內距與圓角。
14. **行高不在本張處理。** 稿的 Body 14／BodyLg 16 行高是 1.625，Tailwind `text-sm`／`text-base` 是 20／24px。dsr-6b 等前例都只換字級、不加 `leading-*`，本張比照，不要自己發明慣例。
15. **陰影。** 稿的對話框（與母版 `Component/DialogFrame` `m6KMPr`）是 0 16 48 50%，程式碼共用 `--shadow-xl`（0 12 24 60%，`styles.css:121`）。全 app 對話框都一樣，**本張不動**，另立單（AC #12）。

### 設計稿節點

| 代號 | 節點 | 本張要看的 |
| --- | --- | --- |
| F6-D-v2 | `dlfMR`（對話框 `GGwqu`） | 標題列 `uDU3E`／`OOKNV`、內容 `Di88y`、說明 `HdXSb`、按鈕 `REsxv`／`iNRPj`、列表 `gf9el`、列 `EFHsP`／`oBH2c`／`T8nYpg`（未確認）、`tvDD3`／`VJ5V8`／`P5j6O`（已確認）、footer `VbxVw`／`qxyjt`／`eFK5e` |
| F7-D-v2 | `A85GFD`（對話框 `B84cF`） | 空狀態 `k0vQH`：圖示 `QxQJ6`、標題 `gORoO`、說明 `n9OYb`、按鈕包裝 `hGlj0`／`O0Hf18` |
| 母版 | `Component/GlossaryRow-v2` `nDSEd` | 原文 `rCEl1`、箭頭 `ZMRcC`、譯名 `H6m5F2`、來源徽章 `WyY3x`／`z5Xrd`、未確認 `R353jw`／`VKeqh`、確認 `wDFKQ`、編輯 `SIp0F`、刪除 `Q11NpX`／`zl2HB` |
| 規格條 | `r7rxg0`（在 `JzmvC` Spec 群組） | 三個徽章 `k8X8FN`／`L2XAH`／`Ot9AS`、說明 `D9y6PK` |
| 元件庫 | `Fx24g` | instance `yOVMl`、說明 `lb5mu` |
| 手機（只改顏色） | F6-M-v2 `buepS` | `ngusL` 的泥金覆寫 |
| 會用到的母版 | `Component/Skeleton` `m0kOOB`、`Component/TextField/Default` `aaiLz`、`Component/DialogFrame` `m6KMPr`、`Component/Button/Secondary` `YDPhc`、`Component/Button/Destructive` `GhcG4`、TechBadge `L9m19`（藥丸圓角參考） | AC #2 新的狀態規格畫面 |

字階（`DESIGN.md:390-394`）：Label 12／1.5、Body 14／1.625、BodyLg 16／1.625。

---

## Acceptance Criteria

1. **設計稿：F6、母版、規格條（Pencil MCP；動手前每個節點再 `Get` 一次）。**
   - **來源徽章一律中性**（⚖️ 裁定 ①）：母版 `nDSEd` 的來源徽章（`WyY3x` 底、`z5Xrd` 字）→ `$bg-tertiary`／`$text-secondary`。F6-D-v2 的 `P5j6O`、`T8nYpg` 與 F6-M-v2 的 `ngusL` **是 instance 覆寫**——只改母版不會變。⚠️ **不要整個拿掉覆寫**：這三處的 `z5Xrd` 覆寫同時帶著 `content:"中繼資料"`，整個拿掉標籤會變回母版的「字幕」。做法是在覆寫裡把 `WyY3x.fill` 改 `$bg-tertiary`、`z5Xrd.fill` 改 `$text-secondary`、**保留 `z5Xrd.content`**，改完 `Get` 確認文字仍是「中繼資料」。`oBH2c`（F6-D）與 `RVb6N`（F6-M）本來就是中性覆寫，不用動。「未確認」徽章（`R353jw`／`VKeqh`）維持 `$warning-tint`／`$warning-text`。
   - **徽章形狀與內距對齊 DESIGN.md:650**（「藥丸、4px／10px 內距」，TechBadge 母版 `L9m19` 就是 `[$Space/xs, $Space/sm-plus]`）：來源徽章與未確認徽章的 `cornerRadius` → `$radius-pill`（變數存在，值 999）、`padding` → `[$Space/xs, $Space/sm-plus]`（原本 [4, 8]）。
   - **確認過的列保留「編輯」**（⚖️ 裁定 ②）：拿掉 F6-D-v2 `tvDD3`、`VJ5V8`、`P5j6O` 上把 `SIp0F` 關掉的覆寫。⛔ F6-M 的「編輯」覆寫與 390 寬溢出不動（`dsr-6f`）。
   - **「Eleven」那列**（`T8nYpg`）：把拆開的 `eyc3W`（寫死 13px 的「11」＋「號」）換回跟其他列一樣的單一譯名文字「11 號」（Body 14 變數，與 `H6m5F2` 同設定）。程式碼不拆數字。
   - **「Eleven」列改完後**再 `Get` 一次 `T8nYpg`，確認 `descendants.H6m5F2` 是單純的文字覆寫，不是一個分離出來的替代 frame。
   - **footer 範例字**：「共 12 條 · 6 條未確認」→「共 6 條 · 3 條未確認」（`qxyjt` → `6`、`eFK5e` → `3`，跟畫出來的列一致）。
   - **規格條 `r7rxg0`**：徽章補成五種（字幕／中繼資料／手動／官方字幕／社群，**全部中性、藥丸**），旁邊加一顆「未確認」（赭）；說明 `D9y6PK` 設 `textGrowth:"fixed-width"`、寬約 480，改成：「來源徽章五種：字幕／中繼資料／手動／官方字幕／社群。一律中性（$bg-tertiary 底、$text-secondary 字、藥丸），只靠文字區分——來源是屬性，不是狀態（DESIGN.md:300、TechBadge 裁定 2026-09-10）。顏色只留給『未確認』＝赭：未確認的詞已經被拿去翻譯（confirmedOnly=false，sub-5-5），使用者還沒看過。」
   - **元件庫說明 `lb5mu`**（⚠️ 它是 585 寬放在 640 的格子 `Fx24g` 裡，而整排格子 `ISilG` 是橫排——文字變長會把右邊三格 PosterCard 往右推、`component-library.png` 被裁掉更多；**先設 `textGrowth:"fixed-width"`、`width:"fill_container"` 再改字**）：「來源徽章×3：字幕/中繼資料/手動」→「來源徽章×5：字幕/中繼資料/手動/官方字幕/社群（中性藥丸）」，「確認/編輯/刪除」後補「（刪除是文字，不是圖示）」。
   - **新節點一律綁變數**（`fontSize: $Type/Label/Size`、`fill: $…`）——`check-design-tokens.py:273` 會擋裸數字。

2. **設計稿：新增一張狀態規格畫面 `F6-SPEC-STATES`**（獨立畫面，放 Flow F 的 `F · 規格 Spec` 子群組；畫布標題「F6 · 名詞對照表・狀態規格（桌面）」；位置用 `FindEmptySpace` 錨在同群組最後一張之後；**一律 instance 母版**，DESIGN.md §SOP 第 3 步）。寬 880 的面板內容，由上而下畫七格，每格上方一行 Label 12 `$text-secondary` 小標：
   - ① **載入中**：**一個** `Component/Skeleton`（`m0kOOB`，它本身就是直排三條、間距 `$Space/sm`）instance，寬 `fill_container`；把三條（`JGgiL`／`YKfdF`／`w2wmH`）覆寫成寬 `fill_container`、高 54、`cornerRadius $radius-md`——跟 `SkeletonRows` 一模一樣。⛔ 不要放三個 instance（會變九條）。
   - ② **載入失敗**：`$error-tint` 底框（padding `$Space/md`、gap `$Space/sm`、`$radius-md`，對應 `GlossaryPanelV2.tsx:166`）、`circle-alert` 16 `$error-text`、Body 14 `$error-text`「名詞對照表載入失敗」、右側「重試」Body 14／600 `$accent-text`。小標註明：「有已載入的列表時，這個框放在列表上方，列表不消失。」
   - ③ **寫入失敗（新）**：同 ② 的框，**沒有重試鈕**，文字「「Demogorgon」的新譯名沒有存到，請再試一次」。小標：「新增／編輯／確認／全部確認／刪除失敗時出現在工具列下方；下一次操作開始時消失。」
   - ④ **新增表單**：`$bg-secondary` 底、`$border-subtle` 框、`$radius-md`、padding `[$Space/sm, $Space/md-plus]`、gap `$Space/sm` 的列（對應 `:73`），裡面兩個 `Component/TextField/Default`（寬 160；placeholder「原文（例：Demogorgon）」Mono、「譯名（例：魔王獸）」）、「新增」Body 14／600 `$accent-text`、「取消」Body 14 `$text-secondary`。下方一行 Body 14 `$warning-text`：「「Demogorgon」已經在表裡了（→ 魔王獸）。要改譯名，請按那一列的「編輯」。」（重複的詞，AC #8）
   - ⑤ **編輯中**：一列 `nDSEd` instance，譯名位置換成 `Component/TextField/Focus`（寬 128，值「魔神獸」），動作換成「儲存」（600 `$accent-text`）＋「取消」（`$text-secondary`），徽章照常。
   - ⑥ **刪除確認**：`Component/DialogFrame` instance，標題「刪除詞彙」、說明「確定要刪除「Demogorgon → 魔王獸」嗎？此操作無法復原。」、footer `Component/Button/Secondary`「取消」＋`Component/Button/Destructive`「刪除」。
   - ⑦ **鍵盤規則**（純文字註記，Body 14 `$text-secondary`）：「Enter：送出（輸入法組字中的 Enter 只選字，不送出）。Esc：焦點在新增表單或正在編輯的那一列裡（包括輸入框與「新增／儲存／取消」按鈕）時，取消新增或編輯，不關閉面板；輸入法組字中的 Esc 只取消組字。其他時候 Esc 才關閉面板。」
   - `scripts/export-pen-screenshots.py` 的 `SCREENS` 在 `"VPT7l": ("flow-f-subtitle-v2", "f18-spec-err")` 旁邊加 `"{新節點 id}": ("flow-f-subtitle-v2", "f6-spec-states")`。
   - **重排流程群組間距**：Flow F 目前底部 y≈34753、Flow H 頂部 36665（差 1912，已經不到 2000）。新畫面約 1000 高，加進 Spec 子群組後，用 `Update(groupId,{y})` 把 Flow H 以下（H／I／J／K／L／M／N）整批往下移，讓每個最外層群組之間**回到 2000px**（DESIGN.md §畫布版面）。移動 y 不影響匯出 PNG。
   - **動手前先數一次裁切問題數**（稽核當天略過 instance 數到 70；DESIGN.md 記的基準是 74），改完再數，不能變多。新畫面依 DESIGN.md SOP 7.3 補日巡主題的截圖檢查。
   - 每張改完 `ctx.problems` 掃裁切；存檔後確認 ` M ux-design.pen` 並讀磁碟檔確認新文字落盤（`feedback_verify_pen_saved_before_commit`；`Insert` 的新節點存檔前量測會假警告，`project_pen_schema_gotchas` #6）。**跑一次最外層與同層畫面的重疊檢查**（DESIGN.md §畫布版面）。
   - 匯出後只 stage：`flow-f-subtitle-v2/{f6-d-v2,f6-m-v2,f6-spec-states}.png`、`design-system/component-library.png`、`_bmad-output/pen-tokens.json`（若有變），其餘重繪雜訊還原。**若 F1／F10 或其他 flow 的圖也變了，代表動到了不該動的母版——回頭檢查。**

3. **名詞一列（`GlossaryRowV2.tsx`，對齊 `nDSEd`）。**
   - 原文 `:65` `text-[13px]` → `text-sm`（`rCEl1` Mono Body 14）；譯名 `:88` 與編輯輸入框 `:85` `text-[13px]` → `text-sm`（`H6m5F2`）。
   - **來源徽章**：`SOURCE_BADGE`（`:17-29`）改成只存文字標籤；五種來源共用一組樣式 `bg-[var(--bg-tertiary)] text-[var(--text-secondary)]`；刪掉 `:21-23` 「官方字幕 wears the success tint」的信任等級註解，改寫成「來源是屬性不是狀態，一律中性（DESIGN.md:300；dsr-6c）」。
   - **兩個徽章**（`:95`、`:105`）：`rounded-[var(--radius-sm)]` → `rounded-full`、`px-2 py-0.5` → `px-2.5 py-1`（DESIGN.md:650 的 4／10，與 `ManageSubtitleDialogV2.tsx:393` 相同）；**`text-[11px]` 不動**（🔴 #13）。未確認維持 `bg-[var(--warning-tint)] text-[var(--warning-text)]`。
   - **刪除改成文字按鈕**（`Q11NpX`／`zl2HB`）：拿掉 `Trash2`（`:12` import 一起刪）；`className` → `flex min-h-[44px] shrink-0 items-center px-2.5 text-sm text-[var(--error-text)] hover:underline disabled:opacity-50`，文字「刪除」；**保留** `data-testid={`glossary-delete-${term.id}`}` 與 `aria-label={`刪除 ${term.termSrc}`}`。
   - 確認 `:141`、編輯 `:154`、儲存 `:118`、取消 `:128` 的 `text-[13px]` → `text-sm`。
   - **確認、編輯也要帶詞的無障礙名稱**：`aria-label={`確認 ${term.termSrc}`}`、`aria-label={`編輯 ${term.termSrc}`}`（目前每列都叫「確認」「編輯」，螢幕報讀分不出是哪一列；不要用「編輯 X 的譯名」——那是編輯輸入框 `:81` 的名稱，兩個會撞名）。
   - **確認過的列保留「編輯」**（⚖️ 裁定 ②；程式碼不改，`GlossaryPanelV2.spec.tsx:156-174` 編輯已確認列的測試照舊通過）。
   - 檔頭 `:4-9` 的「source badge（字幕/中繼資料/手動）」→ 五種、中性；「row actions edit/confirm/delete」補「delete is a text button (nDSEd Q11NpX)」。
   - `:52` 未知來源退回「手動」的行為不變。
   - **列的編輯輸入框**（`:85`）：`rounded-[var(--radius-sm)] … bg-[var(--bg-primary)] px-2 py-1` → `rounded-[var(--radius-md)] bg-[var(--bg-secondary)] px-3 py-2`，`w-32`、`border border-[var(--border-subtle)]`、`focus:border-[var(--accent-primary)] focus:outline-none` 保留（`Component/TextField/Default` `aaiLz`：`$bg-secondary`、`$radius-md`、[8, 12]、1px `$border-subtle`；**不要**照抄母版的 280×36 尺寸）。
   - **刪除確認對話框**（`:172-199`，對齊 AC #2 ⑥ 與 `m6KMPr`／`YDPhc`／`GhcG4`；只在這個 `DialogContent` 的 `className` 覆寫，⛔ 不改 `ui/Dialog.tsx`）：`DialogContent` 加 `max-w-[480px] rounded-[var(--radius-lg)] flex flex-col gap-4`；`DialogTitle` 加 `text-xl`（H3 20）；「取消」`px-4` → `px-5` 並加 `font-medium`；「刪除」`px-4` → `px-5`、`font-medium` → `font-semibold`。手機寬度下 `max-w-[480px]` 不會比現在的 `max-w-lg` 更寬，沒有回歸。

4. **名詞面板外框與工具列（`GlossaryPanelV2.tsx`，對齊 F6／F7）。**
   - `DialogContent`（`:119`）：`max-w-3xl` → `max-w-[880px]`；加 `sm:rounded-[var(--radius-lg)] sm:border sm:border-[var(--border-subtle)]`（`GGwqu`：`$radius-lg`、1px `$border-subtle`）。**圓角與框只加在 `sm:`**——比照 dsr-6b 建單驗證的結論（`ManageSubtitleDialogV2.tsx:473-474`），手機版的樣子屬 `dsr-6f`（F6-M 是底部抽屜），不要先給它一個兩邊都沒有的外觀。`w-[calc(100vw-2rem)]`、陰影不動（🔴 #15）。
   - 內容區 `:126` `p-6` → `px-6 py-5`（`Di88y` [20, 24]）；`gap-4` 不動。
   - 說明 `:128` `text-[13px]` → `text-sm`（`HdXSb`）。
   - 「全部確認」`:133-143`、「新增詞彙」`:144-155`、空狀態「新增詞彙」`:187-197`：`px-4` → `px-5`（`REsxv`／`iNRPj`／`O0Hf18` [8, 20]），**拿掉 `CheckCheck`、`Plus` 圖示**（`:11` import 一起清）。`gap-1.5` 沒有圖示後可一併拿掉。
   - 空狀態說明 `:186` `text-[13px]` → `text-sm`（`n9OYb`）。
   - footer `:225` `text-[13px]` → `text-sm`（`VbxVw`）；文案與 Mono 數字不變。
   - 載入失敗 `:172`、`:177` 與新增表單 `:83`、`:93`、`:101`、`:108` 的 `text-[13px]` → `text-sm`（對應 AC #2 ②④）。
   - **新增表單的兩個輸入框**（`:83`、`:93`，對齊 `Component/TextField/Default` `aaiLz`：`$bg-secondary`、`$radius-md`、[8, 12]、1px `$border-subtle`）：`rounded-[var(--radius-sm)] … bg-[var(--bg-primary)] px-2 py-1.5` → `rounded-[var(--radius-md)] bg-[var(--bg-secondary)] px-3 py-2`；`w-40`、框、focus 樣式、原文的 `font-mono` 保留。**不要**照抄母版的 280×36 尺寸。
   - 工具列的「新增詞彙」（`:144`）顯示條件 `list.length > 0 || terms.isError` → `list.length > 0 || (terms.isError && !terms.data)`——否則「資料是空陣列＋重抓失敗」時，工具列與空狀態會同時出現兩顆「新增詞彙」（AC #9 的新分支會讓空狀態照常顯示）。
   - 檔頭 `:1` 補 `+ Screen F7-D-v2 (A85GFD) + Screen F6-SPEC-STATES ({新節點 id})`（一行、以 `)` 結尾）。
   - 改完 `grep -n 'text-\[13px\]' GlossaryPanelV2.tsx GlossaryRowV2.tsx` 必須為 0；`grep -n 'text-\[11px\]'` 仍是 2 處。

5. **寫入失敗要說出來（🔴 #1）。**
   - 面板新增一個寫入錯誤狀態：`add`、`edit`、`confirm`、`confirmAll`、`remove` 任一失敗時，在工具列下方（新增表單之上）顯示 `data-testid="glossary-write-error"`、`role="alert"` 的框：樣式同載入失敗框（`bg-[var(--error-tint)]`、`CircleAlert`、`text-sm text-[var(--error-text)]`），**沒有重試鈕**。
   - 文案（全形標點，`{src}` 是該詞的原文）：
     - 新增：「新增「{src}」失敗，請再試一次」
     - 編輯：「「{src}」的新譯名沒有存到，請再試一次」
     - 確認：「確認「{src}」失敗，請再試一次」
     - 全部確認：「全部確認失敗，請再試一次」
     - 刪除：「刪除「{src}」失敗，請再試一次」
   - 任何一個 mutation **開始**時清掉舊訊息；面板關閉時清掉。只顯示最近一次的失敗。
   - 用 `mutate(vars, { onError })` 在面板裡處理即可（Rule 5：寫入仍是 mutation）；**不要**加全域 `MutationCache`，也不要改 `useGlossary.ts` 的 invalidate 行為。
   - **編輯失敗時保留使用者打的字**——契約**定死**，不要自選：
     - `GlossaryRowV2Props.onEdit: (termId: string, termZh: string) => Promise<void>`。
     - 面板：`onEdit={async (termId, termZh) => { clear 錯誤; try { await edit.mutateAsync({ termId, termZh, confirmed: … }) } catch (e) { 設定編輯失敗訊息; throw e } }}`。用 `mutateAsync`，**不要**用 `mutate(vars, { onSuccess, onError })` 包 Promise——TanStack v5 的 per-call callback 在同一個 observer 又被 `mutate` 一次、或元件卸載時會被丟掉，Promise 可能永遠不 resolve，列就卡在編輯模式。
     - 列：`saveEdit` 改 async——草稿 trim 後是空的、或跟原譯名一樣 → 照舊直接離開編輯模式、**不呼叫** `onEdit`；否則 `try { await onEdit(term.id, next); setEditing(false) } catch { /* 留在編輯模式，輸入框保留新字 */ }`。
     - 既有測試：`GlossaryRowV2.spec.tsx` 的 `const noop = () => undefined` 當 `onEdit` 傳入約 10 次，型別會不合（`tsconfig.spec.json` 含 spec）——`onEdit` 一律改傳 `() => Promise.resolve()` 或 `vi.fn().mockResolvedValue(undefined)`；`:85` 的 `toHaveBeenCalledWith('t1','魔神獸')` 維持兩個參數、照舊通過。gallery 夾具型別寬鬆，不用改。
   - **錯誤框要看得到**：列表區會捲動（`max-h-[85vh] overflow-y-auto`，新配對的影集約 20 列），在下面幾列編輯失敗時，頂端的錯誤框可能在畫面外。錯誤框出現時呼叫 `scrollIntoView({ block: 'nearest' })`（jsdom 沒有這個方法——測試裡 `Element.prototype.scrollIntoView = vi.fn()` 並斷言被呼叫）。
   - 新增失敗時表單維持開著、草稿保留（現況已如此，補測試鎖住）。

6. **Esc 只取消正在做的事（🔴 #2）。**
   - **標記放在容器上，不是只放輸入框**——焦點在「新增／取消」「儲存／取消」按鈕上時按 Esc 也不能關掉面板：
     - 新增表單的容器（`data-testid="glossary-add-form"` 那個 div，`:71`）加 `data-glossary-inline-editor=""`。
     - 列的根 div（`:61`）**只在編輯中**加：`data-glossary-inline-editor={editing ? '' : undefined}`。
   - `GlossaryPanelV2` 的 `DialogContent` 傳 `onEscapeKeyDown={(e) => { if (e.target instanceof Element && e.target.closest('[data-glossary-inline-editor]')) e.preventDefault(); }}`（Radix 只在 `preventDefault` 時不關，`@radix-ui/react-dismissable-layer` `dist/index.mjs:59-66`）；其餘照常關閉。只有最上層會處理 Esc，外層的管理字幕對話框不會跟著關。
   - Esc 的處理搬到**容器的** `onKeyDown`（按鈕上按 Esc 也會冒泡到容器）：列 → 取消編輯、還原草稿；新增表單 → 關閉表單並清空草稿。刪掉輸入框層級的 Escape 分支（`GlossaryRowV2.tsx:76-79`），避免跑兩次。
   - **輸入法組字中的 Esc 只取消組字**：容器的 Esc 處理也要先檢查 `!isImeComposing(e)`（AC #7），否則用注音按 Esc 放棄選字會把整個編輯取消掉。
   - 測試**必須用真的 Radix Dialog**（渲染整個 `GlossaryPanelV2`，不是單獨的列）：① 在編輯輸入框按 Esc → `onOpenChange` 沒有被呼叫 `false`、輸入框消失、譯名是舊的；② **焦點在「儲存」按鈕上**按 Esc → 同樣不關面板、編輯取消；③ 新增表單的「取消」按鈕上按 Esc → 表單關閉、面板不關；④ 組字中按 Esc（`isComposing: true`）→ 編輯**不**取消；⑤ 不在編輯也不在新增時按 Esc → `onOpenChange(false)`。

7. **Enter：組字中不送出、忙碌中不送出（🔴 #3、#5）。**
   - 新增 `apps/web/src/utils/keyboard.ts`（＋spec）：`export function isImeComposing(e: React.KeyboardEvent): boolean`，回傳 `e.nativeEvent.isComposing || e.keyCode === 229`（Safari 確認選字那一下 `isComposing` 是 false、`keyCode` 是 229）。
   - 三個 Enter 送出點（`GlossaryPanelV2.tsx:79, 89`、`GlossaryRowV2.tsx:75`）改成 `e.key === 'Enter' && !isImeComposing(e)` 才送出。
   - `submitAdd` 與 `saveEdit` 開頭：`busy` 時直接 return（按鈕本來就 disabled，這是補 Enter 那條路）。
   - 測試：`fireEvent.keyDown(input, { key: 'Enter', isComposing: true })` 與 `{ key: 'Enter', keyCode: 229 }` 都**不**觸發送出；一般 Enter 會送出；`busy` 時 Enter 不送出。
   - **「busy 時 Enter 不送出」的測試做法**：busy 時「新增詞彙」按鈕是 disabled，所以要**先**打開表單、打好兩個字，**再**觸發一個永遠不 resolve 的 mutation（例如按某列的「確認」，mock 回傳 `new Promise(() => {})`），最後在輸入框按 Enter，斷言 `addTerm` 沒被呼叫。列的編輯同理（先進編輯模式再讓別的 mutation 卡住）。
   - ⛔ 全 app 其他 6 個 Enter 送出點不在本張改（AC #12 立單）。

8. **新增重複的詞時擋下來、指路（🔴 #4）。**
   - 送出前在已載入的列表裡找同一個詞：`language === 'zh-Hant'`（`models/glossary.go:48` 的預設；新增表單不送 language）且原文相同——比對規則要跟 SQLite `COLLATE NOCASE` 一樣**只忽略 ASCII 大小寫**（`A–Z`），並先 `trim()` 草稿。
   - 找到時**不呼叫 API**，在表單下方顯示 `data-testid="glossary-add-duplicate"`、`text-sm text-[var(--warning-text)]`：「「{existing.termSrc}」已經在表裡了（→ {existing.termZh}）。要改譯名，請按那一列的「編輯」。」（赭＝你要求了但沒發生，DESIGN.md:250）
   - 原文輸入框內容一改變就清掉這行。
   - 列表還沒載入完（`terms.data` 為 undefined）時不擋，照常送出。
   - 測試：大小寫不同（`demogorgon` vs `Demogorgon`）要擋；`É` vs `é` **不**擋（跟 NOCASE 一致）；語言不同不擋。

9. **其他狀態修正（🔴 #6、#7）。**
   - **有資料時重抓失敗不要換掉列表**：錯誤框只在 `terms.isError && !terms.data` 時取代列表；`terms.isError && terms.data` 時錯誤框（含重試）放在列表**上方**，列表照常顯示。
   - 空狀態下正在新增時（`adding`），不顯示「尚無詞彙」插圖。
   - 「取消」清空兩個草稿。
   - 面板關閉時重設 `adding`、草稿、寫入錯誤、重複提示（面板重新打開是乾淨的）。
   - 表單打開時游標在原文輸入框（`autoFocus` 加上跟 `GlossaryRowV2.tsx:83` 同樣的 eslint 註解理由）。

10. **視覺夾具與基準線。**
    - 會變的：`glossary-row-v2/{unconfirmed,confirmed-metadata,manual}`（徽章中性＋藥丸、字級、刪除變文字）、`glossary-panel-v2/{seeded,empty}`（外框、字級、按鈕無圖示）。
    - **`glossary-panel-v2/seeded` 的 darwin 基準本機原本就紅**（`preexisting-fail-visual-darwin-three-stale-baselines`），`glossary-panel-v2/empty` 的 darwin 也一樣舊（PR #146）。本張刻意改這兩個畫面，所以**兩個都要重生 darwin 基準**。依據：`glossary-panel-v2/seeded/default-visual-linux.png`（CI 正在用、全綠）本身就是**側軌沒有電影／影集數字、儲存空間 `–`**——也就是無後端的樣子；舊 darwin 圖之所以不同，是夜行改色前的調色盤加上有後端時才有的 5／2 數字（`–` 在舊 darwin 圖裡也有）。**重生前確認本機沒有 Go API 在跑**（`lsof -i :8080` 必須是空的），重生後打開圖確認側軌「電影」「影集」旁邊**沒有數字**。重生後在該條目補一句「`glossary-panel-v2/seeded` 已由 dsr-6c 刻意重生（無後端），剩兩個」。
    - 新增 `glossary-row-v2/official-subtitle`：一列 `source: 'official_subtitle'`、已確認（例如 `Mind Flayer` → `奪心魔`）、`penNode: 'nDSEd'`，其餘照 `glossary-row-v2/confirmed-metadata`（`-gallery.fixtures.tsx:4190`）——鎖住「五種來源長得一樣」。
    - 基準線流程照 `project_visual_baseline_intentional_change.md`：`--update-snapshots=all` 後只留上列、其餘還原；改過的 `-linux` 一律 `git rm` 交給 CI bootstrap。⛔ 不要本機產 `-linux.png`。

11. **既有測試保留通過**，刻意改的只有：
    - `GlossaryRowV2.spec.tsx`：`onEdit` 一律改傳會 resolve 的函式（AC #5）；刪除按鈕改文字（`glossary-delete-*` testid 與 aria-label 不變，補一條斷言按鈕文字是「刪除」且沒有 svg）；新增徽章五種共用同一組 class 的斷言（逐字比對 class，不用子字串）；確認／編輯的 aria-label；Enter 組字；`onEdit` 失敗時留在編輯模式。
    - `GlossaryPanelV2.spec.tsx`：寫入失敗五種文案（逐字）、訊息清除時機、Esc 行為（真 Dialog）、Enter 組字與 busy、重複詞規則、有資料時重抓失敗、空狀態＋新增、取消清草稿、關閉再打開是乾淨的、按鈕沒有圖示。**`:156-174` 編輯已確認列的測試照舊**。
    - `utils/keyboard.spec.ts`（新）。
    - `styles-contrast.spec.ts:212, 217, 262` 的註解引用 `GlossaryRowV2.tsx:185`（早已過期，現在是 `:193`，本張又會移動）→ 改成引用符號（「`GlossaryRowV2` 刪除確認對話框的『刪除』鈕」），不要寫行號。同一段 `:198-199` 說 `GlossaryRowV2` 把 `--accent-hover` 當文字色——現在已經沒有，一併改掉。
    - 特別守住：`ManageSubtitleDialogV2.spec.tsx`（它 stub 掉面板，名詞入口測試 `:243-255` 等不應受影響）、`glossaryService.spec.ts`。

12. **另立的單子（建單時已寫入 sprint-status，不在本張做）。**
    - `disc-2026-09-glossary-edit-keeps-machine-source`（P2）：編輯譯名不會改來源、也不會改確認狀態——使用者改過的「字幕」詞仍掛「字幕」＋「未確認」。**`sub-8-1` 匯入的「保留我的」判斷依賴這個語意，要在 sub-8-1 做 UI 之前裁定**。
    - `disc-2026-09-ime-enter-submits-mid-composition`（P2）：其他 6 個 Enter 送出點（`LogFilters.tsx:42`、`SavePresetDialog.tsx:140`、`CastEditor.tsx:34`、`MetadataEditorDialog.tsx:421`、`QuickSearchBar.tsx:109`、`SubtitleSearchDialog.tsx:209`）改用 `isImeComposing`。
    - `disc-2026-09-glossary-term-routes-ignore-media-id`（P3）：`PUT`／`confirm`／`DELETE` 只看 term id，不驗證它屬於網址上的那部片（`glossary_service.go:80-112`）。
    - `disc-2026-09-dialogframe-shadow-vs-shadow-xl`（P3）：🔴 #15。
    - `disc-2026-09-design-md-frontmatter-badge-running-stale`（P3）：`DESIGN.md:172-176` 的 `badge-running` 還是 `success-tint`＋`rounded.sm`，跟 StatusBadge/Running 母版（泥金、藥丸）不一致。
    - `disc-2026-09-glossary-panel-a11y-leftovers`（P3）：儲存或刪除後焦點掉回對話框容器；列不是清單語意；一個 `busy` 鎖住所有列。
    - 補記到既有條目：`disc-2026-09-dialog-close-target-44px`（關閉鈕的 sr-only 文字是英文「Close」）、`sub-8-1-glossary-export-import`（UI 排在 dsr-6c 之後，而且要先有四顆按鈕的工具列、匯入摘要、衝突清單設計稿）、`dsr-6f-flow-f-mobile`（F6-M 在未確認列關掉「編輯」是寬度權宜，要照 dsr-6c 的裁定處理；F6-M 的徽章顏色本張已改中性，其餘手機項目仍歸 dsr-6f）。

13. **CI 全綠**：`pnpm run lint:all`、`pnpm nx run web:typecheck --skip-nx-cache`、`python3 scripts/check-design-tokens.py`、`pnpm nx test web`。⛔ 絕不用 `run_in_background` 跑測試；每次跑完 `pnpm run test:cleanup`。⛔ 局部 vitest 綠之後一定要跑 typecheck。本張沒有後端改動，`pnpm nx test api` 不必跑。

## Tasks / Subtasks

- [x] **Task 1 — 設計稿（AC: #1, #2）**
  - [x] F6／母版／規格條／元件庫說明（AC #1）
  - [x] 新畫面 `F6-SPEC-STATES` 七格（AC #2）；`SCREENS` 加一行
  - [x] 動手前後各數一次裁切問題；`ctx.problems`、重疊檢查；Flow H 以下整批重排回 2000px 間距；存檔並確認落盤；匯出；只 stage 該變的 PNG
- [x] **Task 2 — 共用鍵盤判斷（AC: #7 前半）**
  - [x] 先寫紅測試 `utils/keyboard.spec.ts` → `isImeComposing`
- [x] **Task 3 — 名詞一列（AC: #3, #5 編輯部分, #6 列部分, #7 列部分）**
  - [x] 先寫紅測試：徽章五種同 class、藥丸＋內距、刪除是文字、aria-label、Enter 組字／busy、`onEdit` 回傳 Promise 失敗時留在編輯模式、編輯輸入框與刪除確認對話框的 class
  - [x] 實作；檔頭與註解
- [x] **Task 4 — 名詞面板（AC: #4, #5, #6, #7 面板部分, #8, #9）**
  - [x] 先寫紅測試：寫入失敗五種文案、清除時機、捲進畫面；Esc 五種情境（真 Dialog）；Enter 組字／busy；重複詞；重抓失敗保留列表；不會出現兩顆「新增詞彙」；空狀態＋新增；取消清草稿；關閉再開是乾淨的；按鈕無圖示
  - [x] 實作外框與字級；寫入錯誤；Esc；重複詞；狀態修正
- [x] **Task 5 — 夾具與基準線（AC: #10）**
  - [x] 新夾具 `glossary-row-v2/official-subtitle`；重生 darwin；`git rm` 對應 `-linux`；更新 `preexisting-fail-visual-darwin-three-stale-baselines` 條目
- [x] **Task 6 — 收尾（AC: #11, #13）**
  - [x] `styles-contrast.spec.ts` 註解；全套閘門；dev-story Step 9 截圖比對（`f6-d-v2`、`f7-d-v2`、`f6-spec-states`）

## Dev Notes

### 這張的重點

- **三個真的 bug 比對齊更重要**：存失敗不講（編輯失敗還會吃掉使用者打的字）、編輯時按 Esc 關掉整個面板、注音按 Enter 選字就送出半個字。這個面板是「修正譯名」唯一的地方——修不成還不知道，下一次生成字幕就繼續用錯的名字、而且要再付一次錢。
- **狀態色只講狀態**：來源是屬性，全部中性；「未確認」留赭，因為未確認的詞**現在就被拿去翻譯**（`transcription_service.go:284-292`、`subtitle/glossary_store.go:70`、`nfo_localizer_service.go:114` 都是 `confirmedOnly=false`）。
- **新的畫面行為先有稿**：寫入失敗、重複的詞、Esc 規則都畫在 `F6-SPEC-STATES`，dev-story Step 9 對照它驗收。

### 建單裁定（2026-09-17）

1. **① 來源徽章顏色**（⚖️ **Alexyu 2026-09-17 裁定**，採建議選項）：五種來源一律中性（`bg-tertiary`／`text-secondary`、藥丸），「未確認」維持赭。
   - 依據：DESIGN.md:300、:652、:716；⚖️ 2026-09-10 TechBadge 裁定（「屬性不是發生了什麼」）；dsr-6b 把「已生成」從泥金改中性；2026-09-10 狀態色稽核已把「未確認」算作赭的正確用法（`disc-2026-09-status-vocabulary-missing-done-and-money`）。
   - 另一個選項（不採用）：「未確認」也拿掉赭色。缺點是忽略未確認的詞已經在翻譯裡生效；新配對的影集一開始會有最多約 20 列未確認（TMDb 播種，`glossary_seeder.go`），但那正是需要人看的狀態。
   - ⚠️ 如果日後改成只用確認過的詞翻譯（推翻 sub-5-5 的 authoring 裁定），「未確認」就只是「等你看」，應改靛青或中性。
2. **② 確認過的詞保留「編輯」**（⚖️ **Alexyu 2026-09-17 裁定**，採建議選項；改稿不改碼）。
   - 依據：後端允許編輯任何詞（`glossary_repository.go:227-239`）；所有自動流程都是 insert-if-absent，確認旗標什麼都沒保護；sub-5-5 AC「髒詞條由 F6 審核流修正」；「全部確認」讓確認變得很便宜，確認≠逐條看過；拿掉編輯就只能刪掉重打，重打會把來源變成「手動」。
   - 另一個選項（不採用）：照稿拿掉。要改 `GlossaryRowV2.tsx:146-157` 與 `GlossaryPanelV2.spec.tsx:156-174`，AC #8 的指路文案也要改成「先刪除再新增」。
3. SM 定、Sally／Alexyu 可在 review 推翻：寫入失敗的文案（AC #5）、重複詞的文案與擋法（AC #8）、Esc 規則（AC #6）。
4. 行高不動（🔴 #14）、陰影不動（🔴 #15）、F6-M 只改徽章顏色（其他歸 `dsr-6f`）。

### 不要做的事

- **不要改 `ui/Dialog.tsx`**（關閉 X、陰影、圓角預設）——共用元件；本張只在 `GlossaryPanelV2` 的 `className` 覆寫圓角與框。
- **不要把 `text-[11px]` 改成 12**。
- **不要做手機版**（F6-M bottom sheet、滿版按鈕、溢出）——`dsr-6f`。
- **不要做匯出／匯入**——`sub-8-1`。
- **不要改後端**（編輯語意、路由驗證、upsert 行為都另立單）。
- **不要加全域 `MutationCache` 或 toast**——錯誤在面板內說。
- **不要改其他 6 個 Enter 送出點**——另立單。
- **不要加 `leading-*`**。

### 已知陷阱

- **Radix 的 Esc 在 capture 階段**：在輸入框的 React `onKeyDown` 裡 `stopPropagation()` **擋不住**面板關閉；一定要在 `DialogContent` 的 `onEscapeKeyDown` 裡 `preventDefault()`。`preventDefault` 之後同一個 keydown 仍會送到輸入框的 `onKeyDown`，取消編輯的邏輯照常跑。
- **Radix Dialog 走 Portal**：spec 用 `screen` 找；刪除確認是第三層 Dialog。
- **`fireEvent.keyDown` 的 `isComposing`**：jsdom 的 `KeyboardEvent` 支援 `isComposing` 初始化參數；`keyCode` 也要能帶進去——先寫一條確認測試環境真的收得到，再寫行為測試。
- **per-call `mutate` callback 會被丟掉**：同一個 mutation observer 再呼叫一次 `mutate`、或元件卸載時，前一次呼叫傳入的 `onSuccess`／`onError` 不會執行（TanStack v5）。要拿結果就用 `mutateAsync`（AC #5）；寫入錯誤訊息用 per-call `onError` 可以接受（最多漏掉「被下一次操作蓋掉」的那則，而下一次操作開始時本來就要清掉）。
- **`useMutation` 的 `isPending` 會等 `onSuccess` 的 invalidate 完成**（`useGlossary.ts` 回傳 promise），所以 `busy` 期間會包含重抓。
- **Pencil**：`nDSEd` 的泥金在 F6／F6-M 是 **instance 覆寫**，只改母版不會變（`project_pen_schema_gotchas`）；`Replace` 會重設沒寫的屬性；`Insert` 的新節點存檔前量測不準；**不准刪母版節點**。新畫面一律 instance 母版。
- **`toHaveTextContent` 是子字串比對**——文案要逐字斷言；class 也要逐字（dsr-6b CR 教訓：`toContain('sm:gap-1')` 會被 `sm:gap-1.5` 騙過）。
- **視覺 CI 沒有後端**：新夾具的名詞列表要 `seedQueries` 預塞（照 `glossary-panel-v2/seeded`，`-gallery.fixtures.tsx:4414-4465`）；列夾具不打 API。
- **`--update-snapshots` 預設模式小於門檻不會重寫**（`project_visual_update_snapshots_threshold`）——用 `=all` 再還原。
- **行號以建單時為準**（2026-09-17，main `70462433`）。

### Source tree

```
apps/web/src/utils/keyboard.ts(+spec)                                   ← Task 2（新）
apps/web/src/components/subtitle/GlossaryRowV2.tsx(+spec)               ← Task 3
apps/web/src/components/subtitle/GlossaryPanelV2.tsx(+spec)             ← Task 4
apps/web/src/hooks/useGlossary.ts                                       ← 只讀（mutation 形狀）
apps/web/src/styles-contrast.spec.ts                                    ← Task 6（只改註解）
apps/web/src/routes/test/-gallery.fixtures.tsx                          ← Task 5
tests/visual/components.visual.spec.ts-snapshots/components/{glossary-row-v2/*,glossary-panel-v2/*}/** ← Task 5
apps/api/internal/repository/glossary_repository.go、migrations/036_*.go、models/glossary.go ← 只讀（upsert 衝突規則、預設語言）
scripts/export-pen-screenshots.py                                       ← Task 1（SCREENS 一行）
ux-design.pen、_bmad-output/pen-tokens.json、_bmad-output/screenshots/flow-f-subtitle-v2/{f6-d-v2,f6-m-v2,f6-spec-states}.png、design-system/component-library.png ← Task 1
_bmad-output/implementation-artifacts/sprint-status.yaml                ← Task 5（preexisting 條目補記）、完工改狀態
```

### Rule 21 檔頭

- `GlossaryRowV2.tsx`：`// Implements: Component/GlossaryRow-v2 (nDSEd)`（不變）
- `GlossaryPanelV2.tsx`：`// Design ref: ux-design.pen Screen F6-D-v2 (dlfMR) + Screen F7-D-v2 (A85GFD) + Screen F6-SPEC-STATES ({新節點 id})`
- `utils/keyboard.ts`：utils 免標註。

### Cross-Stack Split Check

後端 task **0 個**，前端／設計 task 6 個。→ **本張不拆**。

### Time-dependent visual coverage

- **N/A — no wall-clock-reading components touched.** 名詞面板與列不顯示 `createdAt`／`updatedAt`，不讀時鐘；新夾具不含日期。

### References

- [Source: `ux-design.pen` `dlfMR`／`GGwqu`／`A85GFD`／`B84cF`／`nDSEd`／`r7rxg0`／`Fx24g`／`buepS`／`m6KMPr`／`m0kOOB`／`aaiLz`／`L9m19`] — 稽核代理以 Pencil MCP 讀出（2026-09-17）
- [Source: `apps/web/src/components/subtitle/GlossaryPanelV2.tsx:1, 11, 38-68, 70-113, 119-230`、`GlossaryRowV2.tsx:1-29, 52-58, 61-199`]
- [Source: `apps/web/src/hooks/useGlossary.ts:18-55`、`services/glossaryService.ts:24-63`、`queryClient.ts:5-12`]
- [Source: `apps/api/internal/services/glossary_service.go:48-139`、`repository/glossary_repository.go:113-152, 227-239`、`database/migrations/036_rebuild_show_glossary_with_scope.go:66-70`、`models/glossary.go:14-35, 48`]
- [Source: `apps/api/internal/services/transcription_service.go:284-292`、`subtitle/glossary_store.go:70-98`、`services/nfo_localizer_service.go:114`、`services/glossary_seeder.go:294-303`]
- [Source: `DESIGN.md:172-176, 223, 249-257, 293-300, 390-394, 568, 650-652, 714-720, 746+`（SOP）]
- [Source: `sprint-status.yaml` → `disc-2026-09-techbadge-uses-status-colors-as-taxonomy`／`disc-2026-09-radius-pill-rule-vs-reality`／`disc-2026-09-status-vocabulary-missing-done-and-money`／`disc-2026-09-11px-micro-label-not-on-type-scale`／`preexisting-fail-visual-darwin-three-stale-baselines`／`sub-8-1-glossary-export-import`]
- [Source: `_bmad-output/implementation-artifacts/sub-5-5-glossary-auto-harvest.md:72, 84`（insert-if-absent 紅線、confirmedOnly=false 裁定）、`9R-UX-subtitle-v2-design.md:124-126`（使用者確認／修正）、`dsr-6b-manage-subtitle-dialog-desktop.md`（前一張的做法與 CR 教訓）]
- [Source: project-context.md#Rule 5 / #Rule 13 / #Rule 21 / #Rule 23 / #Rule 24]

## Dev Agent Record

### Agent Model Used

Claude Opus 5 (1M context) — Amelia（dev-story），2026-09-17

### Debug Log References

- **Pencil**：動手前全檔裁切警告 70（略過 instance 也是 70；DESIGN.md 基準 74），改完 70，沒有變多。`Update(instance,{descendants})` 實測是**合併**不是覆蓋，所以 `P5j6O`／`ngusL` 只改 `WyY3x.fill`／`z5Xrd.fill`，`z5Xrd.content`「中繼資料」保留（改完 `Get` 確認）。「Eleven」列原本的 `H6m5F2` 被整個替換成 frame `eyc3W`，無法用 `Update` 改回文字，改用 `Replace` 整個 instance（新 id `r8PoZ4`，`descendants` 只剩三個 content 覆寫；列表順序不變）。新畫面 `n3vIR` 存檔前量到的 `+50` 位移與 fully clipped 是已知假警告（`project_pen_schema_gotchas` #6），AppleScript 存檔後重量為 0 個問題。日巡檢查：`n3vIR`／`GGwqu` 暫加 `theme:{mode:"light"}` 截圖看過，沒有黑字壓黑底；之後改回 `theme:{}`（Pencil 無法把屬性設成 undefined——空物件等於沒有覆寫，畫面照夜行渲染，已截圖確認）。
- **畫布重排**：Flow F 規格群組底部 35408，Flow H 標題原本 36665；46 個最外層節點整批下移 743，現在 H 標題 37408（間距 2000）。重疊檢查只剩一組：`C · 桌面 Desktop` 與 `C · 手機 Mobile`——兩者都在 F 之上、本張沒動，是既有問題，另立單（Discovery Triage）。
- **匯出**：195 張中，會變的是 `component-library`、`f6-d-v2`、`f6-m-v2`、新的 `f6-spec-states`，加 `pen-tokens.json`；`flow-i-discover-v2` 四張是已知重繪雜訊，已還原。`component-library.png` 尺寸 2728×4096 不變（`lb5mu` 改固定寬後沒有把格子推出去）。
- **視覺基準**：沒有單一夾具的篩選參數，所以跑整套 `CI=1 VISUAL_BUCKETS=4 npx playwright test --project=visual --update-snapshots=all --workers=4`（`nx serve web` 先起、:8080 無人聽），之後只留 glossary 的 5 張＋新的 1 張，其餘 168 張 `git checkout` 還原。重生後打開 `glossary-panel-v2/seeded`：側軌電影／影集沒有數字、儲存空間 `–`。
- **測試陷阱**：TanStack 的 `mutationFn` 在 `mutate` 之後才非同步執行，所以「沒有呼叫 addTerm」這類斷言第一次寫時是**假綠**（修正前就會過）。補了 `flush()`（`setTimeout 30`）後再跑，確認它們在修正前是紅的。

### Completion Notes List

- Ultimate context engine analysis completed - comprehensive developer guide created
- 🔗 AC Drift: FOUND — `ux3-subtitle-v2` AC #4「Panel supports: list, add, …」：新增一個已經在表裡的詞，原本會被後端 upsert 默默覆寫（譯名、來源、確認狀態都換掉），現在在前端擋下來、指路「編輯」（AC #8）。其餘命中皆為 REUSE：ux3-subtitle-v2 AC #4「unconfirmed visually distinct / row actions edit/confirm/delete」、AC #7「Radix Dialog for destructive confirms」仍成立；sub-7-1 AC #4 只規定官方字幕／社群的**文字**，顏色是當時開發者自選，本張依裁定 ① 改中性；dsr-6b AC #4 面板 id 不受影響。（grep：`GlossaryRowV2\|GlossaryPanelV2\|glossary-row\|GlossaryRow-v2` across `_bmad-output/implementation-artifacts/*.md`，9 個檔案）
- 📎 Contract Stamps: NONE（本張、ux3-subtitle-v2、sub-7-1、dsr-6b 都沒有 `[@contract-v*]`；本張不改任何 API 形狀）
- 🎭 A11y Pre-Flight: PASS（2 個元件；`pnpm run lint:all` 0 errors、128 warnings 與開工前相同，觸及的檔案 0 個 jsx-a11y 警告）。手動四項：modal 焦點——新增表單打開時焦點進原文輸入框（`autoFocus`＋理由註解），Esc 只取消編輯中的東西、不關面板；寫入失敗與重複詞提示帶 `role="alert"`；確認／編輯帶詞的無障礙名稱；刪除改文字按鈕仍保留 `aria-label="刪除 {原文}"`。Esc 的處理放在輸入框與按鈕自己的 `onKeyDown`，**不放容器**——`jsx-a11y/no-static-element-interactions` 是 error 級，div 掛 `onKeyDown` 會擋 CI；標記 `data-glossary-inline-editor` 仍在容器上，給面板的 `onEscapeKeyDown` 判斷用。
- **設計稿（AC #1、#2）**：母版徽章中性＋`$radius-pill`＋`[$Space/xs,$Space/sm-plus]`；三處泥金覆寫改中性（保留文字）；三列已確認恢復「編輯」；「Eleven」改回單一文字「11 號」；footer 範例「共 6 條 · 3 條未確認」；`r7rxg0` 補成五種中性藥丸＋一顆赭色「未確認」、說明改寫並固定寬 480；`lb5mu` 固定寬後改字。新畫面 `F6-SPEC-STATES`（`n3vIR`，標題 `jGabE`）七格全用母版 instance：`m0kOOB`（三條覆寫成 54 高）、`aaiLz`×2、`nDSEd`（譯名換成 `rGYSZ`）、`m6KMPr`（按鈕換成 `YDPhc`／`GhcG4`、高 44）。`SCREENS` 加 `n3vIR → f6-spec-states`。
- **名詞一列（AC #3、#5、#6、#7）**：五種來源共用 `bg-tertiary`／`text-secondary` 藥丸（`data-testid="glossary-source-{id}"` 新增，給測試逐字比對 class）；徽章形狀抽成常數 `BADGE_SHAPE`，所以 `grep 'text-\[11px\]'` 現在是 **1 處**（兩顆徽章共用），11px 仍凍結、沒有改成 12。刪除改紅字文字鈕；確認／編輯帶詞名；編輯輸入框對齊 `aaiLz`；刪除確認對話框對齊 `m6KMPr`——除了 AC 寫的 480／radius-lg／gap-4／`text-xl`／按鈕 px-5 與字重，另把標題與說明包進 `flex flex-col gap-1`（母版 header gap `$Space/xs`，否則 `gap-4` 會把標題和說明拉開 16px）。`onEdit` 改回傳 Promise：成功才離開編輯模式，失敗留著打的字。Esc（輸入框、儲存、取消）取消編輯；組字中的 Enter／Esc 不動作；`busy` 時 Enter 不送。
- **名詞面板（AC #4、#5、#6、#7、#8、#9）**：外框 880、`sm:` 圓角與框、`px-6 py-5`；兩顆 Secondary 無圖示 px-5；所有 13px → `text-sm`（0 處殘留）。**寫入一律用 `mutateAsync`**（AC #5 允許 `mutate(vars,{onError})`，但編輯已定死要 `mutateAsync`，五種寫入同一條 `runWrite` 路徑較單純，也避開 per-call callback 被丟掉）：開始時清掉舊訊息、失敗時顯示五種逐字文案之一，出現時 `scrollIntoView({block:'nearest'})`（**CR L7 後改掉**：錯誤框放進 `sticky top-0` 的不透明底座，不再捲動）。`onEscapeKeyDown` 在標記容器內 `preventDefault`。重複詞比對只折 ASCII 大小寫、語言限 `zh-Hant`、列表沒載入時不擋。重抓失敗但有資料時錯誤框放列表上方；工具列「新增詞彙」只在「有列表」或「錯誤且完全沒資料」時出現；新增時空狀態插圖讓位；取消清草稿；面板關閉時用 render 期間比對上一個 `open` 重設（不用 effect，關閉後再開不會先閃一下舊表單，父層直接改 `open` 也涵蓋；**CR L6 後改在打開時重設**，並以「場次」擋掉舊一次開啟的失敗回報）。
- **夾具與基準線（AC #10）**：新增 `glossary-row-v2/official-subtitle`；5 張 darwin 重生＋1 張新增；5 張 `-linux` `git rm`，交給 CI bootstrap；`preexisting-fail-visual-darwin-three-stale-baselines` 條目補記（剩兩個）。
- **其他（AC #11）**：`styles-contrast.spec.ts` 三處 `GlossaryRowV2.tsx:185` 改成符號引用；`:198-199` 的 accent-hover 名單註明 GlossaryRowV2 已不再使用。
- **閘門（AC #13）**：`pnpm run lint:all` exit 0（0 errors／128 warnings，與開工前相同）；`pnpm nx run web:typecheck --skip-nx-cache` exit 0；`python3 scripts/check-design-tokens.py` 一致（82 變數、195 張、73 母版）；`pnpm nx test web --skip-nx-cache` **267 檔／3636 條全過**（dsr-6b 收尾時 3584）；`pnpm nx test api` exit 0（workflow Step 7 的全套閘門；本張沒有後端改動）。每次測試後 `pnpm run test:cleanup`。
- 🎨 UX Verification: PASS — 對照 `f6-d-v2`、`f7-d-v2`、`f6-spec-states`（見下表）；差異只有三項，都是建單時已裁定或已知：列高 54 vs 56、行高、編輯輸入框的 focus 樣式。

| 區域 | 設計稿 | 實作 | 相符 | 需要修 |
| --- | --- | --- | --- | --- |
| 面板外框 | 880、`$radius-lg`、1px `$border-subtle` | `max-w-[880px]`、`sm:rounded-[var(--radius-lg)]`、`sm:border` | ✅ | — |
| 標題列／內容區 | h56、BodyLg 600；內容 [20,24] gap 16 | `h-14`、`text-base font-semibold`；`px-6 py-5 gap-4` | ✅ | — |
| 工具列按鈕 | Button/Secondary h44 [8,20] 500，無圖示 | `min-h-[44px] px-5 text-sm font-medium`，無 svg | ✅ | — |
| 列 | Mono 14 原文、arrow 14、譯名 14 | `font-mono text-sm`、`h-3.5`、`text-sm` | ✅ | — |
| 來源徽章 | 中性、藥丸、[4,10] | `bg-tertiary`／`text-secondary`、`rounded-full px-2.5 py-1`（字 11px 凍結） | ✅ | — |
| 未確認 | `$warning-tint`／`$warning-text` 藥丸 | 同 | ✅ | — |
| 列動作 | 確認 600 accent、編輯 secondary、刪除 error-text（文字） | 同；已確認列保留編輯 | ✅ | — |
| footer | Body 14 secondary、數字 Mono | `text-sm`、`font-mono tabular-nums` | ✅ | — |
| 空狀態（F7） | book-open 40、BodyLg 600、Body 14、Secondary 無圖示 | 同 | ✅ | — |
| ① 載入中 | 三條 54、radius-md、gap 8 | `SkeletonRows` 同 | ✅ | — |
| ②③ 錯誤框 | error-tint、padding 12、gap 8、radius-md；② 有重試 | `p-3 gap-2`；寫入失敗無重試 | ✅ | — |
| ④ 新增表單＋重複詞 | [8,14] gap 8；TextField 160；warning-text 提示 | `px-3.5 py-2 gap-2`；`w-40` 輸入框；提示同 | ✅ | — |
| ⑤ 編輯中 | TextField/Focus 128、儲存／取消 | `w-32`；focus 是 1px `accent-primary` 框（AC #3 指定保留），稿是 2px `$focus-ring` | ⚠️ 已知 | 否（AC 指定） |
| ⑥ 刪除確認 | DialogFrame 480、radius-lg、H3、header gap 4、按鈕 h44 | `max-w-[480px]`、`rounded-[var(--radius-lg)]`、`text-xl`、`gap-1`、`min-h-[44px] px-5` | ✅ | — |
| 列高 | 已確認列固定 54 | 每列有 44px 按鈕，實際 56 | ⚠️ 已知 | 否（建單稽核：稿的固定 54 擠掉了 1px 內距） |
| 行高 | Body／BodyLg 1.625 | Tailwind `text-sm`／`text-base` 預設 | ⚠️ 已知 | 否（🔴 #14，不在本張） |

### Discovery Triage

- **Did this story discover any work outside its current scope?** **YES** —— 建單當下處理如下。

- **① expand-scope-in-place**
  - 寫入失敗不講、編輯失敗吃掉輸入（🔴 #1）→ **AC #5**
  - 編輯時 Esc 關掉整個面板（🔴 #2）→ **AC #6**
  - 輸入法組字中按 Enter 送出（🔴 #3，僅本面板三處）→ **AC #7**
  - 新增重複的詞默默覆寫（🔴 #4）→ **AC #8**
  - Enter 不看 busy（🔴 #5）→ **AC #7**
  - 重抓失敗換掉列表、空狀態＋表單、取消不清草稿、重開不乾淨、表單不聚焦（🔴 #6、#7）→ **AC #9**
  - 來源徽章狀態色、徽章形狀（🔴 #8、#9）→ **AC #1、#3**
  - 稿沒畫的狀態（🔴 #11）→ **AC #2**
  - 確認／編輯按鈕沒有帶詞的無障礙名稱 → **AC #3**
  - 刪除確認對話框與 `Component/DialogFrame` 不一致（寬、圓角、標題字級、間距、按鈕）→ **AC #3**
  - 「資料為空＋重抓失敗」時出現兩顆「新增詞彙」→ **AC #4**
  - Flow F 與 Flow H 的畫布間距已經不到 2000px → **AC #2**
  - `styles-contrast.spec.ts` 過期行號註解 → **AC #11**

- **② spawn-blocking-story**：無。

- **③ backlog-with-carry-forward-link**（建單時立）
  - `disc-2026-09-glossary-edit-keeps-machine-source` — 編輯不改來源與確認狀態；sub-8-1 之前要裁定
  - `disc-2026-09-ime-enter-submits-mid-composition` — 其他 6 個 Enter 送出點
  - `disc-2026-09-glossary-term-routes-ignore-media-id` — 單詞路由不驗證所屬
  - `disc-2026-09-dialogframe-shadow-vs-shadow-xl` — 對話框陰影稿與碼不同
  - `disc-2026-09-design-md-frontmatter-badge-running-stale` — DESIGN.md frontmatter 過期
  - `disc-2026-09-glossary-panel-a11y-leftovers` — 焦點、清單語意、單一 busy
  - 補記：`disc-2026-09-dialog-close-target-44px`、`sub-8-1-glossary-export-import`、`dsr-6f-flow-f-mobile`
  - **dev-story 時新立**：`disc-2026-09-pen-flow-c-desktop-mobile-overlap` — 畫布重排後跑重疊檢查，`C · 桌面 Desktop` 與 `C · 手機 Mobile` 兩個群組互相重疊（本張沒動 Flow C）

- Reference: `project-context.md` Rule 24

### File List

**新增：**
- `apps/web/src/utils/keyboard.ts`（＋`keyboard.spec.ts`）— `isImeComposing`
- `_bmad-output/screenshots/flow-f-subtitle-v2/f6-spec-states.png`
- `tests/visual/components.visual.spec.ts-snapshots/components/glossary-row-v2/official-subtitle/default-visual-darwin.png`
- `_bmad-output/implementation-artifacts/dsr-6c-glossary-panel.md`（本檔）

**修改：**
- `apps/web/src/components/subtitle/GlossaryRowV2.tsx`（＋spec）
- `apps/web/src/components/subtitle/GlossaryPanelV2.tsx`（＋spec）
- `apps/web/src/routes/test/-gallery.fixtures.tsx` — 新夾具 `glossary-row-v2/official-subtitle`
- `apps/web/src/styles-contrast.spec.ts` — 只改註解
- `scripts/export-pen-screenshots.py` — `SCREENS` 加 `n3vIR`
- `ux-design.pen`、`_bmad-output/pen-tokens.json`
- `_bmad-output/screenshots/flow-f-subtitle-v2/{f6-d-v2,f6-m-v2}.png`、`_bmad-output/screenshots/design-system/component-library.png`
- `tests/visual/components.visual.spec.ts-snapshots/components/{glossary-panel-v2/{empty,seeded},glossary-row-v2/{confirmed-metadata,manual,unconfirmed}}/default-visual-darwin.png`
- `_bmad-output/implementation-artifacts/sprint-status.yaml` — 本張 → review；新立 `disc-2026-09-pen-flow-c-desktop-mobile-overlap`；`preexisting-fail-visual-darwin-three-stale-baselines` 補記

**刪除：**
- 上列 5 個夾具的 `default-visual-linux.png`（交給 CI bootstrap）

**AC drift reference（未修改）：** `_bmad-output/implementation-artifacts/ux3-subtitle-v2.md`（AC #4 新增語意，見 Completion Notes）

## 對抗式 Code Review（/ship，2026-09-17）

獨立 reviewer（fresh context，只讀；自己跑三支 spec 74 條，另做 33 個變異版本，32 個有效、抓到 29 個）回報 **0 HIGH、1 MED、8 LOW＋1 註記**。**修 7、交代 1、不修 1**。每條修正都先確認測試在修正前是紅的（把修正拿掉再跑，5 條新測試轉紅）。修完三支 spec 86 條全綠、沒有 act() 警告；全套 `pnpm nx test web` **267 檔／3648 條全過**、lint 0 errors、typecheck、token 一致（第一次重跑時 Nx daemon 建專案圖逾時，三個指令都在建圖階段失敗、沒有跑到測試；停掉 daemon 用 `NX_DAEMON=false` 重跑後全綠）。

| # | 等級 | 問題 | 處置 |
| --- | --- | --- | --- |
| M1 | MED | 存檔（或新增）還在跑時按 Esc 或「取消」，編輯列／表單先關掉、打的字清掉，之後失敗才跳訊息——AC #5「失敗保留打的字」被繞過；成功時則是「取消了卻存進去」。另一條：點「儲存」後按鈕變 `disabled`，瀏覽器把焦點丟到 `<body>`，這時按 Esc 會關掉整個面板 | 列加本地 `saving`，存檔中 `cancelEdit` 不動作；新增用 `add.isPending` 擋「取消」與 Esc。「儲存」「新增」「取消」在忙碌時改用 `aria-disabled`（不是 `disabled`），焦點留在按鈕上，Esc 仍在標記容器內。測試：列與面板各一條「送出中按 Esc → 失敗 → 字還在」、`aria-disabled` 不掉焦點、第二次點擊不重送 |
| L2 | LOW | 重複詞提示存的是舊物件：那一列被刪掉或改了譯名，提示照舊；刪掉後再新增失敗，畫面同時出現兩個互相矛盾的 alert | 只存 id，文字從目前列表推出（列不在就不顯示）；通過檢查時先清掉。測試：刪掉那一列後提示消失、新增失敗時只有一個 alert |
| L3 | LOW | 面板關掉前送出、重新打開後才失敗的寫入，會把錯誤報進新的一次開啟 | 每次打開算一個新場次，舊場次的失敗不回報。測試 |
| L4 | LOW | 變異沒抓到：新增表單的輸入法 Esc、「取消」清重複提示、寫入錯誤框的樣式；刪除確認裡的 Esc、面板層的「編輯中別的寫入卡住按 Enter」沒有測試 | 五條都補上 |
| L5 | LOW | 列 spec「Enter saves」在 act 外面離開編輯模式（act 警告）；失敗測試用兩次 `await Promise.resolve()` 賭時機 | 改成 `waitFor`／`await act(async () => {})` |
| L6 | LOW | 在「關閉時」重設，會在淡出動畫一開始就拿掉表單，面板在淡出途中跳一下 | 改在「打開時」重設（仍在 render 期間，舊表單不會先閃一下）。AC #9 的「面板關閉時重設」→ 使用者看到的效果相同：每次打開都是乾淨的 |
| L7 | LOW | AC #5 規定的 `scrollIntoView` 在編輯下方某列失敗時，會把那一列（和正在打字的輸入框）捲出畫面 | 改成**不捲動**：錯誤框放在一個 `sticky top-0` 的不透明底座裡，列表往下捲時它黏在內容區頂端，編輯中的列不動。這是對 AC #5 寫法的偏離，效果是 AC 的原意（錯誤一定看得到）。拿掉 `scrollIntoView` 與其測試，改測 sticky 與錯誤框樣式 |
| L8 | LOW | 同一個重複詞再送一次，提示不會重新掛載，螢幕報讀不會再唸 | 每次被擋都換 key 重新掛載。測試 |
| L9 | LOW | 工作區有不相關的 memory、skills、coverage 檔 | 不修——ship 時只 stage File List 上的檔案 |
| 註 | — | `text-[11px]` 字面次數 2 → 1（兩顆徽章共用常數） | 交代：在 `disc-2026-09-11px-micro-label-not-on-type-scale` 補記計數方式 |

查過不成立：舊寫入的錯誤會蓋掉新寫入（`busy` 含重抓，前一個沒結束開不了新的）、per-call callback 被丟掉（全走 `mutateAsync`）、unhandled rejection、render 期間重設在 StrictMode 出錯、同一句錯誤連續兩次不重新掛載、Esc 讓外層管理字幕對話框跟著關、`keyCode 229` 傳不到元件、重複詞規則與後端不一致、「新增詞彙」在任何狀態重複或消失、AC #3／#4 的 class／文案／testid、設計稿節點值、darwin 基準圖、sprint-status YAML。

## Change Log

| 日期 | 內容 |
| --- | --- |
| 2026-09-17 | 🔍 **/ship 對抗式 CR**：0 HIGH／1 MED／8 LOW，修 7、交代 1、不修 1。最重要：存檔或新增還在跑的時候按 Esc／取消，原本會先把打的字清掉、之後才跳失敗——現在會等它跑完；忙碌中的按鈕改用 aria-disabled，焦點不會掉到頁面上導致 Esc 關掉整個面板。另外：重複詞提示跟著目前的列表走、舊的一次開啟裡的失敗不報到新的一次、錯誤框改成黏在內容區頂端而不是把正在編輯的列捲走、每次打開才重設（關閉動畫不跳）。 |
| 2026-09-17 | ✅ **dev-story 完成 → review**（Amelia）。web 3636/3636、api PASS、lint 0 errors、typecheck、token 一致。修掉三個真的問題：新增／編輯／確認／刪除失敗時畫面不說（編輯失敗還會吃掉打的字）、編輯時按 Esc 關掉整個面板、注音按 Enter 選字就送出；另外擋下「新增已存在的詞會默默覆寫」。對齊：面板 880＋框、按鈕無圖示、刪除改文字、字級收成 14、來源徽章全中性藥丸（未確認留赭）。設計稿：母版與 F6 改色、已確認列恢復編輯、新畫面 `F6-SPEC-STATES` 七種狀態、規格條補成五種；Flow H 以下畫布下移 743 恢復 2000 間距。新立 1 張單（Flow C 畫布重疊，既有）。 |
| 2026-09-17 | 🔍 **建單後對抗驗證**（fresh-context 驗證代理，只讀；抽查約 45 個行號、全部 .pen 節點存在、`nDSEd` 只在 F6-D／F6-M／`Fx24g` 被 instance）：1 項 CRITICAL、10 項 SHOULD FIX、11 項 NIT，**全部併入**。最重要：① Esc 的標記原本只放輸入框——焦點在「儲存」「取消」按鈕上按 Esc 仍會關掉整個面板，改成標在新增表單容器與編輯中的列上；② `onEdit` 契約定死成回傳 Promise、面板用 `mutateAsync`（per-call callback 會被丟掉，列可能卡在編輯模式），並處理既有 spec 的型別；③ 拿掉泥金覆寫時要保留「中繼資料」文字（覆寫同時帶著標籤）；④ `Component/Skeleton` 本身就是三條，原本會畫成九條；⑤ 刪除確認對話框在列裡用 className 對齊 `DialogFrame`；⑥ 面板的圓角與框只加在 `sm:`；⑦ 徽章內距改成 DESIGN.md 的 4／10；⑧ 元件庫說明變長會把整排格子推出畫面，先設固定寬；⑨ 新畫面讓 Flow F 與 H 的間距跌破 2000，要整批重排；⑩ 編輯失敗的錯誤框可能在捲動區外，要捲進畫面。另外：組字中的 Esc、darwin 重生前確認沒跑後端、無障礙名稱撞名、busy 測試做法、重複的「新增詞彙」。 |
| 2026-09-17 | Story 建立（SM Bob, create-story）。三個唯讀稽核代理（Pencil 節點、程式碼與測試、DESIGN.md 規則與歷來裁定）。找到三個真的 bug（寫入失敗不講、編輯時 Esc 關掉整個面板、輸入法 Enter 送出半個字）與重複詞默默覆寫；兩個待裁定事項 ⚖️ Alexyu 當日裁定，都採建議選項（來源徽章全中性、未確認留赭；確認過的詞保留編輯、改稿）。新立 6 張 disc 單、補記 3 個既有條目。 |
