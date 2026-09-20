# Story DSR.6f-2：手機上的名詞對照表從底部滑上來，每一列拆成兩行，「編輯」回來了

Status: review

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As the person who opens 名詞對照表 on a phone,
I want 它像管理字幕一樣從底部滑上來、每一列的「確認／編輯／刪除」都看得到也按得到,
so that 我在手機上真的能整理譯名，而不是看著一列被切掉一半、連「編輯」都找不到。

## Context

`dsr-6f`（Flow F 全部手機稿）拆出來的**第二塊**。

**Depends on: `dsr-6f-1`（已 done，PR #488 合併進 main `3058aafd`）。** 本張直接吃它的三樣地基：

| 地基 | 在哪 | 本張怎麼用 |
| --- | --- | --- |
| sheet 共用外殼 | `apps/web/src/components/ui/mobileSheet.tsx`（`MOBILE_SHEET_CONTENT`、`MOBILE_SHEET_CLOSE`、`<SheetGrabber />`） | 名詞表面板與刪除確認都換成它 |
| 由下往上的動畫 | `styles.css` 的 `sheet-enter`／`sheet-exit`（只在 `max-sm:`） | 同上，不用再寫 keyframe |
| 視覺測試的手機 viewport | `GalleryFixture.viewport`＋`gallery.tsx` manifest＋`components.visual.spec.ts` 的 set／reset 與寬度守門 | 新增 390×844 的夾具 |

⚠️ **驗收基準是 `.pen` 節點值**。PNG 只是參考。
⛔ **本張不進** `GenerationBatchDialogV2`／`consent/*`（6f-3）、`GenerationWorkspaceV2`（6f-4）、`ManageSubtitleDialogV2`（6f-1 已做完；本張只在 e2e 裡**經過**它，不改它）。

### ⚖️ 裁定（Alexyu 2026-09-20，建單當下）

**手機上每一列換行成兩行**，不用 ⋯ 溢出選單：

```
┌────────────────────────────────┐
│ Demogorgon  →  魔王獸            │  ← 第一行：原文／箭頭／譯名
│ 字幕                 編輯   刪除 │  ← 第二行：徽章靠左、動作靠右
└────────────────────────────────┘
┌────────────────────────────────┐
│ Hopper  →  霍普                  │
│ 中繼資料 未確認   確認 編輯 刪除 │
└────────────────────────────────┘
```

理由（量測）：第二行最寬的情況 **319px ≤ 可用的 328px**（裁定當下口頭引用的是 309／330，對抗驗證重量後修正為 319／328——結論不變，但餘裕只有 ~9px，所以 AC #3 另加 `flex-wrap` 保險）；而 ⋯ 選單就算把「編輯＋刪除」收進去，未確認的列**仍然差 30–40px**，多半還是得換行——等於兩種做法都要做，而且多欠一個共用元件（今天只有 `media/DetailPanelMenu.tsx` 有自己的一份，不是共用的）。

### 🔴 建單時查到的事（main `48f3e268`；行號皆為現況）

**面板外殼**

1. **手機上它還是「置中對話框」，不是抽屜。** `GlossaryPanelV2.tsx:317` 的 class 是 `flex max-h-[85vh] w-[calc(100vw-2rem)] max-w-[880px] flex-col gap-0 p-0 sm:rounded-[var(--radius-lg)] sm:border sm:border-[var(--border-subtle)]` —— 沒有 `MOBILE_SHEET_CONTENT`、沒有把手、沒有 `closeClassName`。所以在 390 寬它是一個 358 寬、垂直置中、**四角不圓也沒有邊框**（`sm:` 才有）的方塊，進場動畫是桌機的 `dialog-enter`（原地放大 4%）。6f-1 把四個對話框換成 sheet 時**明文排除**了這一個（`dsr-6f-1` AC #3 最後一行），它是唯一漏網的。
2. **關閉鈕是 16×16。** `ui/Dialog.tsx:75-83` 的預設 ✕ 是 `absolute right-4 top-4 … opacity-70`＋`<X className="size-4" />`，面板沒有傳 `closeClassName`。手機上 ✕ 是**唯一**的按鈕式出口（沒有頁尾的「關閉」，手機也沒有 Esc），`DESIGN.md:618-620` 要求命中區 44×44。
3. **稿上連 ✕ 都沒畫。** `Jm7RD`（sheet）的子節點只有：把手 `yEb9p`、標題文字 `I3zyy`、說明 `gG1CX`、列表 `P7NUD`、兩顆滿版鈕 `o938ZZ`／`YMF10`——**沒有標題列、沒有 ✕**。桌機 F6-D 的標題列 `uDU3E` 有 `mvQD3`（44×44）＋`d6llff`（x 18、`$text-secondary`）。→ 稿要補（AC #1）。
4. **兩顆動作鈕的位置相反。** 稿在**列表下面**、各自滿版 h44（`o938ZZ`「全部確認」／`YMF10`「新增詞彙」，都是 `Component/Button/Secondary` `YDPhc` 的 instance）；程式碼在**列表上面**、跟說明擠同一列（`:325-355`）。
5. **手機稿沒有頁尾計數，桌機稿有。** 程式碼有（`:397-404`「共 N 條 · M 條未確認」）；桌機 `GGwqu` 的**第三個**子節點就是頁尾 `VbxVw`（上框線 1px、padding [14,24]、gap 4；數字 `qxyjt`／`eFK5e` 走 Mono、其餘走 Primary——連字型切分都與 `:399-402` 的 `font-mono tabular-nums` 一致，padding 也對得上 `px-6 py-3.5`）。**缺的只有手機**：`Jm7RD` 沒有這一列（見裁定 3）。

**列**

6. **列在 390 寬塞不下。** `GlossaryRowV2.tsx:100` 是不換行的單行 flex（`flex min-h-[54px] items-center gap-3 … px-3.5 py-1.5`），除了譯名（`truncate`）、spacer（`min-w-0 flex-1`）與編輯中的輸入框（`:120` `w-32`，沒有 `shrink-0`）之外**全部 `shrink-0`**。sheet 內寬 390−32＝358，列內可用寬度 358−2（`border` 1px×2）−28（`px-3.5`×2）＝**328**。一個未確認的列要放 8 個元素、7 個 gap(12)：原文、箭頭 14、譯名、來源徽章、未確認徽章、確認、編輯、刪除。量過 `buepS` 的 `RVb6N`（Vecna，**沒有**「編輯」）在 358 的列裡只剩十幾 px 餘裕——也就是說稿是**靠把「編輯」關掉**才塞下的；程式碼每列都有「編輯」（多 48＋gap 12），今天在 390 寬必定溢出。
7. **稿把四列的「編輯」全部關掉了**——不只未確認的兩列。`pwW3l`／`QkHtp`／`ngusL`／`RVb6N` 四個 instance 都帶 `SIp0F:{enabled:false}`（`SIp0F` ＝ `nDSEd` 的 `gr-act-edit`）。⚖️ **Alexyu 2026-09-17（dsr-6c 建單裁定 ②）已裁定所有列都保留「編輯」**：後端允許編輯任何詞、自動流程全是 insert-if-absent 不會覆寫，拿掉「編輯」只會逼人刪掉重打、重打還會把來源變成「手動」。dsr-6c 改了桌機稿（`tvDD3`／`VJ5V8`／`P5j6O` 的覆寫已拿掉），並**明文把手機稿留給本張**（`dsr-6c` AC：「⛔ F6-M 的『編輯』覆寫與 390 寬溢出不動（`dsr-6f`）」）。
8. **兩個檔案今天一個 `max-sm:` 都沒有**（grep 零命中）。
9. **新增表單同樣不換行。** `GlossaryPanelV2.tsx:166-227`：兩個 `w-40`（160）輸入框＋`flex-1` spacer＋「新增」＋「取消」，容器 `flex items-center gap-2 … px-3.5`。光兩個輸入框＋gap 就 328，等於把 328 的可用寬度用光，再加兩顆按鈕一定爆。
10. **刪除確認在手機仍是置中對話框。** `GlossaryRowV2.tsx:210-243`：`max-w-[480px]`、吃 `ui/Dialog` 的預設 `p-6`、16×16 的 ✕。它是**第三層**對話框（管理字幕 sheet → 名詞表 → 刪除確認）。

**測試與夾具（動手前一定要先讀這三條）**

11. 🚨 **`GlossaryRowV2.spec.tsx` 用「class 完全相等」。** `:23-28` 的 `expectExactClasses` 會 `expect(el.classList).toHaveLength(expected.length)`。四個元素被這樣鎖住：來源徽章（`:69-72`）、未確認徽章（`:82-85`）、編輯輸入框（`:202-205`，含 `w-32`）、**刪除鈕**（`:287-290`）。**在這四個元素上多加一個 `max-sm:` token，測試就紅，而且紅的訊息只說長度不符，不會告訴你原因。** 本張的設計是這四個元素**都不用改**（見 AC #3）——真的要改就連同這支測試一起改。
12. 🚨 **`GlossaryPanelV2.spec.tsx:202-214` 斷言面板有不帶前綴的 `max-w-[880px]`。** 換成 sheet 之後桌機寬度必須搬到 `sm:max-w-[880px]`（`MOBILE_SHEET_CONTENT` 裡有 `max-w-none`／`w-full`，不加前綴就會跟它打架），所以這一條要**改寫**，不是只加新的。同 `:216-234` 的 `px-5` 與 `:261-276` 的錯誤框 token 都是 `toHaveClass`（超集容許），加 `max-sm:` 不會紅。
13. **名詞表完全沒有 e2e，也沒有 TestSprite 案例**（`tests/` grep `glossary` 零命中；`testsprite_tests/*` grep `glossary`／`名詞`／`詞彙` 全部 0）。`ManageSubtitleDialogV2.spec.tsx:84-91` 還把 `GlossaryPanelV2` 整個 `vi.mock` 成一個 stub——所以**本張改的東西在管理字幕那支 76 個測試裡完全看不到**，既不會紅也不算覆蓋。
14. **六個視覺夾具全是桌機。** 四個列夾具寫死 `width: 720`（`-gallery.fixtures.tsx:4285`／`4309`／`4333`／`4359`），兩個面板夾具沒有 width（Portal，拍視窗）。`width` 與 `viewport` **互斥**且有守門測試（`routes/test/gallery-fixture-viewport.spec.ts:12-17`），所以手機夾具一定是**新的**，不能改既有的。在頁內（非 Portal）的手機夾具還有寬度守門：`components.visual.spec.ts:277-285` 要求 ≥ `viewport 寬 − 65`（390 → 325）。
15. **巢狀對話框兩層都 `z-50`**，靠 DOM 先後疊（名詞表的 Portal 晚於管理字幕的）；`ManageSubtitleDialogV2.tsx:467-478` 沒有傳 `overlayClassName`。今天可以動——**本張不要動它**。

### 設計稿節點

| 代號 | 節點 | 本張要看的 |
| --- | --- | --- |
| F6-M-v2 | `buepS`（390×844、`theme:{bp:"mobile"}`、`clip:true`；caption `f8u6l`） | sheet `Jm7RD`（x=0 y=359、390 寬、`$bg-secondary`、上緣 `$radius-xl`、padding [8,16,24,16]、gap 14）；把手 `yEb9p`（padding [0,0,2,0]）／`uMwil`（36×4、`$bg-tertiary`、`$radius-sm`）；標題 `I3zyy`（BodyLg 600，**沒有標題列、沒有 ✕**）；說明 `gG1CX`（Body `$text-secondary`）；列表 `P7NUD`（gap 8）四列 `pwW3l`／`QkHtp`／`ngusL`／`RVb6N`（都 h54、`SIp0F` 關掉）；`o938ZZ`「全部確認」／`YMF10`「新增詞彙」（都是 `YDPhc`、`fill_container`、h44） |
| 列母版 | `Component/GlossaryRow-v2` `nDSEd`（720 寬、padding [6,14]、gap 12、`$radius-md`、`$border-subtle` 1px inner） | `rCEl1` gr-src（Mono Body）／`ZMRcC` gr-arrow（lucide `arrow-right` 14 `$text-muted`）／`H6m5F2` gr-zh（Primary Body）／`m6wSed` gr-spacer（`fill_container` h1）／`WyY3x` gr-badge（`$bg-tertiary`、**pill**、padding [4,10]）內 `z5Xrd`（Label `$text-secondary`）／`R353jw` gr-unconf（`$warning-tint`、pill）內 `VKeqh`（Label `$warning-text`）／`wDFKQ` gr-act-confirm（h44、padding [0,10]）內 `u9HLj`（Body 600 `$accent-text`）／`SIp0F` gr-act-edit 內 `KfDYP`（Body `$text-secondary`）／`Q11NpX` gr-act-delete 內 `zl2HB`（Body `$error-text`） |
| 桌機對照 | F6-D-v2 `dlfMR` → 對話框 `GGwqu`（880、`$radius-lg`、shadow）：標題列 `uDU3E`（h56、底線、padding [0,12,0,24]）內 `OOKNV` 標題＋`mvQD3` 44×44／`d6llff` x 18；本體 `Di88y`（padding [20,24] gap 16）內 `oRUF2` header-row（說明＋spacer＋兩顆鈕）與 `gf9el` 列表；**頁尾 `VbxVw`**（上框線、padding [14,24] gap 4、「共 6 條 · 3 條未確認」，數字走 Mono） | 桌機已裁定的事：列都有「編輯」、來源徽章一律中性藥丸、**頁尾計數桌機稿有、手機稿沒有** |
| 狀態規格 | F6-SPEC-STATES `n3vIR`（880 寬，**只有桌機**）：① 載入中 ② 載入失敗 ③ 寫入失敗 ④ 新增表單＋重複詞 ⑤ 編輯中（`l78Hq`：輸入框 `C3MxD` 寬 128、儲存／取消、`Q11NpX` 關掉） ⑥ 刪除確認（`H1OMp`，`Component/DialogFrame` `m6KMPr`） ⑦ 鍵盤規則 | 七個狀態**沒有手機版**——本張用註記補（AC #1），不另畫七張 |
| 規格註記 | 群組 `JzmvC`（F · 規格 Spec）；6f-1 的 `PyB9P` 在 x=17040、y=35161、寬 300 | 新註記放在 `PyB9P` 下方 40px，同 x 同寬——**動手前 `Get` 量 `PyB9P` 的實際高度再算 y**（節距不固定） |
| 母版 | `Component/Button/Secondary` `YDPhc`（h36、`$bg-tertiary`、`$radius-md`、padding [8,20]、label Body 500）、`Component/BottomSheet` `SG1ln`、`Component/DialogFrame` `m6KMPr` | ⛔ **一個都不改**（6f-1 裁定 3 仍有效） |

字階（`DESIGN.md:351-360`）：BodyLg 16、Body 14、Label 12。間距：`$Space/xs`4・`xs-plus`6・`sm`8・`sm-plus`10・`md`12・`md-plus`14・`lg`16・`lg-plus`20・`xl`24。圓角：`radius-sm`4・`md`8・`lg`12・`xl`16・`pill`999。

---

## Acceptance Criteria

1. **設計稿先改（Pencil MCP；動手前每個節點再 `Get` 一次確認 id 與現值）。**
   - **新母版 `Component/GlossaryRow-v2/Mobile`**（兩行版；斜線命名是本檔既有慣例，比照 `ButtonCost/Default`、`PosterCard-v2/Hover`）：
     - 外層 frame：`layout:"vertical"`、`gap:"$Space/sm"`、`padding:["$Space/sm","$Space/md-plus"]`、`fill:"$bg-secondary"`、`cornerRadius:"$radius-md"`、`stroke:"$border-subtle"`、`strokeWidth:1`、`strokeAlignment:"inner"`、`width:358`。（垂直內距取 `$Space/sm`(8) 而非桌機 `nDSEd` 的 `$Space/xs-plus`(6)：兩行的列需要多一點呼吸，而且高度本來就由 44 的動作鈕決定，6 與 8 在桌機那種單行列才有差。）
     - 第一行 frame（`width:"fill_container"`、`gap:"$Space/md"`、`alignItems:"center"`）：gr-src（Mono Body `$text-primary`）、gr-arrow（icon `arrow-right` 14 `$text-muted`）、gr-zh（Primary Body `$text-primary`）。
     - 第二行 frame（同上）：gr-badge（照 `nDSEd` 的 `WyY3x`／`z5Xrd` 逐值複製：`$bg-tertiary`、`$radius-pill`、padding [4,10]、Label `$text-secondary`）、gr-unconf（`$warning-tint`／`$warning-text`、同形狀）、spacer（`width:"fill_container"`、`height:1`）、gr-act-confirm／gr-act-edit／gr-act-delete（各 `height:44`、`padding:["$Space/none","$Space/sm-plus"]`、`justifyContent:"center"`、`alignItems:"center"`，標籤 Body，顏色分別 `$accent-text` 600／`$text-secondary`／`$error-text`）。
     - **放哪**：先 `Get` `nDSEd`（x=17040, y=-6214, 720 寬）與它下方 250px 內的節點 bounds，確認空的再插入；不要蓋到別的母版。
     - 依 `DESIGN.md:788`，新母版要放進 `Components · 元件` 底下對應分類，**並在 Component Library 那一頁補一格**（那頁是手畫的、不會自動收錄母版）——所以 `design-system/component-library.png` **應該**會變，這是預期中的。
   - **`buepS`**：
     - 在 `Jm7RD` 裡、把手 `yEb9p` 之後插一個**標題列** frame（`width:"fill_container"`、`height:44`、`justifyContent:"space_between"`、`alignItems:"center"`、`padding:["$Space/none","$Space/xs","$Space/none","$Space/none"]`、**不要底線**——F1-M 的 `CGvIz` 閒置時就沒有），把既有的 `I3zyy` **Move** 進去，再 `Copy` 桌機的 `mvQD3`（含 `d6llff`）進來當 ✕。sheet 的左右 padding 已經是 16，標題列自己不再加左內距。
     - `P7NUD` 的四列改成新母版的 instance，**`SIp0F`（編輯）一律 enabled**（🔴 #7、⚖️ dsr-6c 裁定 ②）。逐列的覆寫要照抄現況，不要多也不要少：
       - `pwW3l`（Demogorgon／魔王獸）、`QkHtp`（Upside Down／顛倒世界）：只有 `rCEl1`／`H6m5F2` 兩個覆寫，徽章吃母版預設的「字幕」——⛔ **不要替它們補 `WyY3x`／`z5Xrd`**；`R353jw`＋`wDFKQ` 維持 `enabled:false`（已確認的列沒有「未確認」徽章也沒有「確認」鈕）。
       - `ngusL`（Hopper／霍普，`z5Xrd:"中繼資料"`）、`RVb6N`（Vecna／維克那，`z5Xrd:"手動"`）：保留 `WyY3x.fill:$bg-tertiary`＋`z5Xrd`（含 `fill:$text-secondary`）的覆寫，`R353jw`＋`wDFKQ` 維持啟用。
     - sheet 變高之後**重新貼底**：`Jm7RD.y = 844 − 實際高度`（先 `Get` 量再設）。高度上限：`DESIGN.md:630` 寫的是**螢幕的 80%**（844×0.8 ≈ 675），程式碼是 `max-h-[85vh]`（717）——這 5 個百分點的出入已由 `disc-2026-09-bottomsheet-grabber-three-variants` 追蹤（6f-1 裁定 3），本張**對稿用 675**。改完約 650，兩個門檻都過。
   - **規格註記**（x=17040、寬 300，比照 `PyB9P`：`$Type/Label/*`、`fill:"$text-muted"`、`textGrowth:"fixed-width"`；**y 要現場量**——`Get` `PyB9P` 的 bounds 取 `y + height + 40`，建單時量到 35161+147 → 35348，但這個群組的節距本來就不固定，動手前重量一次）：
     > 「F6 名詞對照表（手機）：面板與刪除確認都是由下往上的 sheet，關閉鈕 44×44，沒有頁尾的『關閉』。每一列在手機換成兩行（第一行原文→譯名，第二行徽章靠左、動作靠右）——所有列都保留『編輯』（⚖️ 2026-09-17）。『全部確認』『新增詞彙』在列表下面、各自滿版；新增表單在手機直排（兩個輸入框各佔一行）。F6-SPEC-STATES 的七個狀態只畫桌機，手機沿用同一套元件、照本註記的版面規則直排。頁尾計數（共 N 條 · M 條未確認）桌機稿有（`VbxVw`），手機稿沒畫——程式碼在手機仍然顯示。」
   - ⛔ **不改**母版 `nDSEd`、`YDPhc`、`SG1ln`、`m6KMPr`，也不改 `dlfMR`／`n3vIR`／`A85GFD`（桌機已 done）。
   - 收尾：`Get((n,c)=>c.problems && ...)` 數全檔裁切警告，**基準值是現況 70**（`pen-tokens.json` 的 `counts.clippingWarnings` 也是 70；`DESIGN.md:836` 寫的 74 是 2026-09-11 的舊值，用它等於容許新增 4 個破版），超過 70 就是新破版；存檔並 grep 磁碟上的 `.pen` 確認落盤（`feedback_verify_pen_saved_before_commit`）；跑 `python3 scripts/export-pen-screenshots.py` 後 stage：
     - `_bmad-output/screenshots/flow-f-subtitle-v2/f6-m-v2.png`
     - `_bmad-output/screenshots/design-system/component-library.png`（補格之後必變）
     - 🚨 **`_bmad-output/pen-tokens.json`（必變，一定要 stage）**——新母版讓 `counts.masters` 73→74、`penSha256` 也換；漏 stage 的話 `scripts/check-design-tokens.py` 的快照新鮮度檢查會直接紅。
     - 其餘一律 `git checkout` 還原（全量重跑是非決定性的）。
     - 📎 規格註記不會產生任何 PNG（`JzmvC` 不是 `export-pen-screenshots.py` 的 `SCREENS` key，既有的 `PyB9P`／`bSlYR` 也一樣）。

2. **面板在手機變成 sheet（🔴 #1–#5）。** 一律 `max-sm:`／`sm:` 成對，桌機一個像素都不能變。
   - `GlossaryPanelV2.tsx:310-318` 的 `DialogContent`：
     - `closeClassName={cn(MOBILE_SHEET_CLOSE, 'max-sm:top-4')}`——標題列在手機是 44 高、把手區 16，與 `ManageSubtitleDialogV2.tsx:470` 同幾何，所以同樣是 `top-4`。
     - className 改成「手機外殼 ＋ `sm:` 桌機外殼」的寫法，比照 `ManageSubtitleDialogV2.tsx:472-477`：保留 `flex max-h-[85vh] flex-col gap-0 p-0`、加 **`max-sm:overflow-hidden`**（⚠️ 不加前綴的 `overflow-hidden` 會在所有寬度生效＝桌機變更，與本張的紅線牴觸）、併 `MOBILE_SHEET_CONTENT`，桌機那半段**全部加 `sm:`**（`sm:bottom-auto sm:left-1/2 sm:right-auto sm:top-1/2 sm:w-[calc(100vw-2rem)] sm:max-w-[880px] sm:-translate-x-1/2 sm:-translate-y-1/2 sm:rounded-[var(--radius-lg)] sm:border sm:border-[var(--border-subtle)]`）。
       - 🚨 `MOBILE_SHEET_CONTENT` 裡有不帶前綴的 `w-full`／`max-w-none`／`translate-x-0`／`translate-y-0`——桌機的對應值**不加 `sm:` 就會被 twMerge 丟掉或反過來蓋掉手機值**。這就是 🔴 #12 那條測試要改寫的原因。
     - 第一個子節點補 `<SheetGrabber data-testid="glossary-sheet-grabber" />`。⚠️ **把手、標題列、動作鈕包裝都要給 testid**（`glossary-sheet-grabber`／`glossary-header-row`／`glossary-actions`）——AC #7 要斷言它們的 class，沒有 testid 就只能寫 `button.parentElement` 這種一改 DOM 就碎的查詢。6f-1 的 `manage-sheet-grabber`／`manage-sheet-header` 是先例。
   - 標題列（`:320-322`）：加 `max-sm:h-11 max-sm:border-b-0 max-sm:pl-4`（稿上手機沒有標題列底線；`pr-12`＝48＝`right-1`(4)＋44，不用改）。
   - 內容區（`:324`）：加 `max-sm:px-4 max-sm:pt-1.5 max-sm:gap-3.5`。**底部 padding 只寫一次**：有頁尾計數時（`list.length > 0`）`max-sm:pb-4`，沒有頁尾時 `max-sm:pb-[max(1rem,env(safe-area-inset-bottom))]`。
   - 頁尾計數（`:397-404`；`:396` 是 `{list.length > 0 && (` 守衛）：加 `max-sm:px-4 max-sm:pb-[max(0.875rem,env(safe-area-inset-bottom))]`。（裁定 3：保留它。）
   - 📎 **`env(safe-area-inset-bottom)` 今天恆為 0**（`apps/web/index.html:8` 沒有 `viewport-fit=cover`，追蹤於 `disc-2026-09-viewport-fit-cover-missing`），所以上面兩處 safe-area 實際等於 `pb-3.5`／`pb-4`。照寫，但不要在 Completion Notes 說「safe-area 生效了」。
   - **兩顆動作鈕搬到列表下面、各自滿版**（稿 `o938ZZ`／`YMF10`）：
     - 已查證可行的最小做法——header-row（`:325`）加 `max-sm:contents`（手機上它自己不畫框，子節點直接變成內容區的 flex 子項），兩顆鈕外面包一層 `<div className="flex gap-3 max-sm:order-last max-sm:w-full max-sm:flex-col sm:contents">`（桌機上這層 `contents` 會消失，兩顆鈕仍是 header-row 的直接子項 → 桌機渲染逐位元不變），兩顆鈕各加 `max-sm:w-full`。
     - `hidden flex-1 sm:block` 的 spacer（`:327`）手機本來就不顯示，不動。
     - ⚠️ `SECONDARY_BUTTON`（`:45-46`）是**三顆鈕共用**的常數（`:271` 空狀態、`:338` 全部確認、`:350` 新增詞彙）——⛔ 不要把 `max-sm:w-full` 加進常數裡，要**逐顆**加，否則空狀態那顆也會被拉滿版。
     - ⚠️ `order-last` 只在**同一個 flex 容器**裡有意義——`max-sm:contents` 與這層包裝是一組，少一個就不會動。動手前用瀏覽器量（AC #8），不要只看 class 字串。
     - ⚖️ **已知取捨（刻意接受）**：`order` 只改視覺順序，不改 DOM 與 Tab 順序。手機上讀屏／鍵盤會走「說明 → 全部確認 → 新增詞彙 → 列表」，而畫面上兩顆鈕在列表**下面**。程式順序仍然是有意義的（與桌機同一套），所以不算 WCAG 1.3.2 違規，但 2.4.3 焦點順序確實與視覺分家。**不要為此寫任何 tab-order 斷言**；已掛進 `disc-2026-09-glossary-panel-a11y-leftovers` 的交接註記，Sally／Alexyu 可在 review 推翻（推翻的話就要把兩顆鈕在 DOM 上搬到列表之後，桌機改用別的方式回到同一列）。
   - Rule 21（**兩個檔案都要改**）：
     - `GlossaryPanelV2.tsx:1` 的 `// Design ref:` 接 ` + Screen F6-M-v2 (buepS)`（文法是 ` + Screen …`）。
     - `GlossaryRowV2.tsx:1` 的 `// Implements: Component/GlossaryRow-v2 (nDSEd)` 接 ` + Component/GlossaryRow-v2/Mobile (新母版 id)`——同一支元件實作兩個母版。ESLint 規則 `local/implements-pen-node-id` 接受兩層斜線的名稱，但它**只驗格式、不驗節點存不存在**（規則檔不讀任何檔案），所以新 id 一定要用 Pencil MCP 查證（前例：`sprint-status.yaml:1301` 的 dsr-1 掛了 7 個指向已刪節點的 header）。既有的斜線母版前例是 `ui/ButtonCost.tsx:1`。

3. **列在手機換成兩行（🔴 #6、#7；⚖️ 2026-09-20 裁定）。**
   - `GlossaryRowV2.tsx:97-101` 的列容器：手機直排（`max-sm:flex-col max-sm:items-stretch max-sm:gap-2`），桌機不變。
   - 兩行用**兩層 `sm:contents` 的包裝**做（桌機上包裝消失，現有的單行 flex 逐位元不變）：
     - 第一行包 `rCEl1`／箭頭／譯名三者：`<div data-testid={\`glossary-row-line1-${term.id}\`} className="flex min-w-0 items-center gap-3 sm:contents">`。
     - 第二行包徽章與動作：`<div data-testid={\`glossary-row-line2-${term.id}\`} className="flex flex-wrap items-center gap-3 sm:contents">`——`flex-wrap` 是保險（見下一條），`sm:contents` 之下不生效，桌機零風險。
     - 既有的 `<span className="min-w-0 flex-1" />`（`:126`）留在桌機位置並加 `max-sm:hidden`；在「未確認徽章」與第一顆動作鈕**之間**插一顆手機專用的 `<span aria-hidden="true" className="min-w-0 flex-1 sm:hidden" />`，讓第二行變成「徽章靠左、動作靠右」。
     - 📏 **餘裕只有 ~9px。** 列的可用寬度是 358 − 2（`border` 1px×2）− 28（`px-3.5`×2）＝ **328**；實測最寬的第二行（4 字來源徽章＋未確認＋確認／編輯／刪除）需要 **319**。來源標籤多一個字或字型度量差一點就會爆——所以第二行一定要帶 `flex-wrap`，爆的時候是換行不是切掉。
   - 🚨 **不要碰這四個元素的 class**：來源徽章、未確認徽章、編輯輸入框、刪除鈕——`GlossaryRowV2.spec.tsx` 對它們用 class **完全相等**比對（🔴 #11）。本 AC 的做法刻意不需要動到它們。真的非動不可，就連同 `expectExactClasses` 的期望字串一起改，並在 Completion Notes 說明。
   - 編輯中的那一行：輸入框 `w-32`（128；⚠️ 它是列裡唯一**沒有** `shrink-0` 的動作區元素）＋原文＋箭頭 ≈ 250 ≤ 328，**不用改**；「儲存／取消」落在第二行右側（量過 108 ≤ 328）。
   - 「編輯」在**每一列**都在（已確認與未確認都是）——這是本張最主要的行為差別，要有測試守。
   - ⛔ `text-[11px]` 的徽章字級、`min-h-[54px]`、`GENERATION_STAGES` 之類的凍結項一律不動（11px 追蹤於 `disc-2026-09-11px-micro-label-not-on-type-scale`）。

4. **新增表單在手機不再擠成一行（🔴 #9）。**
   - 表單容器（`:166-170`）：加 `max-sm:flex-col max-sm:items-stretch max-sm:gap-2`。
   - 兩個輸入框用 `TEXT_INPUT`（`:48-49`，`w-40`）——加 `max-sm:w-full`。⚠️ `TEXT_INPUT` 只有這裡用，改它不會波及別處（已查證）。
   - `<span className="flex-1" />`（`:202`）加 `max-sm:hidden`。
   - 「新增」「取消」包一層 `<div className="flex items-center gap-2 max-sm:justify-end sm:contents">`（桌機 `contents` 消失 → 逐位元不變）。
   - 重複詞提示 `:228-238` 不動（它本來就是 `flex-col` 的第二個子項）。
   - 空清單時「新增詞彙」仍然只在空狀態裡那一顆（`:263-275`）——行為不變。

5. **刪除確認在手機也是 sheet（🔴 #10）。** 只換三樣，內容不重排：
   - `GlossaryRowV2.tsx:212-215` 的 `DialogContent`：併 `MOBILE_SHEET_CONTENT`，**桌機的寬度與圓角都要加前綴**——`max-w-[480px]` → `sm:max-w-[480px]`、`rounded-[var(--radius-lg)]` → **`sm:rounded-[var(--radius-lg)]`**；`closeClassName={cn(MOBILE_SHEET_CLOSE, 'max-sm:top-4')}`；第一個子節點補 `<SheetGrabber data-testid="glossary-delete-sheet-grabber" />`。
     - 🚨 **圓角那一項漏了前綴，sheet 的上緣圓角與貼底直角會整個消失。** 用 repo 的 tailwind-merge 3.4.0 實測：不加前綴時 twMerge 會把 `MOBILE_SHEET_CONTENT` 的 `rounded-b-none`／`rounded-t-[var(--radius-xl)]` 全部丟掉（手機變成四角 12px、底部兩角浮在畫面外緣）；把 `MOBILE_SHEET_CONTENT` 挪到最後則換成**桌機**變形。加 `sm:` 是唯一兩邊都對的寫法（`ManageSubtitleDialogV2.tsx:477` 就是這樣寫的）。
   - `DialogFooter` 裡的兩顆鈕加 `max-sm:w-full`（`DialogFooter` 在手機本來就是 `flex-col-reverse`）。
   - ⚠️ `GlossaryRowV2.spec.tsx:309-318` 用的是 `toHaveClass`（超集），所以加 `max-sm:` 不會紅；但它同時斷言 `max-w-[480px]` **與** `rounded-[var(--radius-lg)]`——**這兩項都會紅**，兩條都要改成 `sm:` 版本。
   - ⚠️ ✕ 的垂直位置要**用瀏覽器看過**：這個對話框沒有標題列，`max-sm:top-4` 之下 ✕ 會落在把手與標題之間；若壓到「刪除詞彙」四個字就改 `top` 值，並把實際值寫進 Completion Notes。

6. **既有的行為不准回歸。**
   - 桌機（≥640）：面板、列、新增表單、刪除確認的版面與既有 1280 基準線**完全不變**（六張既有的 glossary 基準線一張都不能動）。
   - 四個狀態（載入中／載入失敗／空／清單）、寫入失敗的 sticky 提示、重複詞擋下、輸入法 Enter／Esc 規則、編輯中 Esc 只取消編輯不關面板（`:312-316` 的 `INLINE_EDITOR` 守衛）——全部照舊。
   - 巢狀關係照舊：從管理字幕的「名詞對照表」那一列打開，關掉之後焦點回到那一列（Radix 預設，沒有 `onCloseAutoFocus`）。
   - ⛔ 不改 `ui/Dialog.tsx`、`ui/mobileSheet.tsx`、`useGlossary.ts`、`glossaryService.ts`、`ManageSubtitleDialogV2.tsx`、任何後端檔案。
   - ⛔ 不動兩層 `z-50` 的疊法（🔴 #15）。

7. **測試。** 分「**紅**」（現在會失敗）與「**守**」（現在就會過，用來擋回歸）——不要把守門測試當紅測試交差（Rule 16）。
   - `GlossaryPanelV2.spec.tsx`
     - （**改寫** `:202-214`）桌機寬度是 `sm:max-w-[880px]`；面板帶 `max-sm:data-[state=open]:animate-sheet-enter`（⚠️ 外殼的其他 token 有些今天就有，拿它們當斷言是假紅）；有把手（`sm:hidden` 的那層）。
     - （紅）✕ 帶 `max-sm:top-4` 與 `max-sm:h-11 max-sm:w-11`。
     - （紅）「全部確認」「新增詞彙」帶 `max-sm:w-full`；`glossary-actions` 帶 `max-sm:order-last`；`glossary-header-row` 帶 `max-sm:contents`；`glossary-sheet-grabber` 存在且帶 `sm:hidden`。（⚠️ 一律用 testid 取節點，不要 `button.parentElement`。）
     - （紅）新增表單的兩個輸入框帶 `max-sm:w-full`；表單容器帶 `max-sm:flex-col`。
     - （守）`:216-234` 的 `px-5`、`:261-276` 的錯誤框 token、四個狀態、重複詞、Esc 規則全數照舊。
   - `GlossaryRowV2.spec.tsx`
     - （紅）**已確認的列也有「編輯」**（`glossary-edit-{id}` 存在）——這條今天就會過（程式碼本來就每列都有），所以它是**守**，要誠實標示；真正的紅是版面。
     - （紅）列容器帶 `max-sm:flex-col`；`glossary-row-line1-{id}`／`glossary-row-line2-{id}` 存在且都帶 `sm:contents`，line2 另帶 `flex-wrap`；手機專用 spacer 帶 `sm:hidden`、原 spacer 帶 `max-sm:hidden`。
     - （紅）刪除確認的 content 帶 `sm:max-w-[480px]`、**`sm:rounded-[var(--radius-lg)]`** 與 `max-sm:data-[state=open]:animate-sheet-enter`；`:309-318` 的 `max-w-[480px]` **與** `rounded-[var(--radius-lg)]` 兩條都要改寫。
     - （守）`expectExactClasses` 的四個元素**逐字不變**——這一條就是 🔴 #11 的守門，不要刪。
   - ⚠️ jsdom 不看 media query，上面全是 token 斷言——`feedback_measure_the_breakpoint_you_return_to` 說得很清楚，class 字串對版面是空話。真的「手機長這樣」由 AC #8 守。

8. **真瀏覽器的驗證（兩層）。**
   - **視覺夾具**（一律 `viewport: { width: 390, height: 844 }`，**不可**同時給 `width`）：
     - `glossary-panel-v2/seeded-mobile`（Portal，拍視窗；props 同 `:4620-4672`，`seedQueries` 至少要有一個**未確認**的詞，才看得到「未確認＋確認＋編輯＋刪除」這條最寬的第二行）。
     - `glossary-row-v2/unconfirmed-mobile`（在頁內；**一張就好**，`term` 用 `source:'metadata'`＋`confirmed:false`——那是第二行**最寬**的組合：4 字來源徽章＋未確認＋確認／編輯／刪除。既有的 `unconfirmed`／`confirmed-metadata` 兩個夾具都拍不到這個組合，不要照抄它們的 props）。
     - 🚨 **頁內的手機夾具只有 326 寬，不是 358。** `/test/gallery` 的頁面容器是 `gallery.tsx:298` 的 `p-8`（32×2），390 − 64 ＝ 326（既有的 `generation-progress-v2/轉錄中-mobile` 基準線就是 326×246）；而 `width` 與 `viewport` 互斥，**撐不到 358**。實測在 326 之下最寬的第二行會溢出約 23px、刪除鈕右緣被切掉，但**在真正的 358 sheet 裡不會**。所以：頁內那張只是「class 有生效」的證據，**不是版面正確的證據**；版面由 Portal 的 `seeded-mobile` 與 AC #8 的 e2e 量測負責。把這句話寫進夾具的註解，免得下一個人去追一個不存在的 bug。
     - ⚠️ 另一個已知陷阱：固定在底部的 `MobileTabBar`（`z-40`、高 84＋safe-area、`sm:hidden`）在 390 寬**會畫出來**，落在視窗底部的頁內夾具會被它蓋到（6f-1 CR L8）。兩行的列實測 **86** 高，應該夠矮，**但要開圖看過**——被蓋到就改成 Portal 或整頁截圖，不要改 harness。
     - **既有 1280 的基準線一張都不能變**——跑完整支 visual，`git status` 只能多出新夾具（本機固定會漂的 `parse-floating-parse-progress-card`／`retry-retry-notifications` 照舊還原）。
   - **e2e**（新，`tests/e2e/glossary-mobile.spec.ts`，tag `@e2e @glossary-mobile`）——照 `tests/e2e/manage-subtitle-mobile.spec.ts` 的寫法：
     - `seedMovie` ＋ `afterEach deleteMovies`；`page.setViewportSize({width:390,height:844})`；打開詳情頁 → `action-manage-subtitle` → `open-glossary`。
     - 假 API：`page.route('**/api/v1/media/*/glossary', …)` 回 `{success:true, data:{terms:[…]}}`（**snake_case**：`term_src`／`term_zh`／`confirmed`／`source`／`created_at`／`updated_at`——`Rule 18` 的轉換在前端做）。至少兩筆：一筆 `confirmed:true`、一筆 `confirmed:false` 且 `source:"metadata"`。
       - ⚠️ glob 是**錨定**的，`**/api/v1/media/*/glossary` 不會吃到 `…/glossary/confirm-all`（6f-1 已查證 glob 行為；反過來，帶 query 的 URL 要寫 `?*`）。
       - ⚠️ `fulfill` 只傳 `status`＋`json`，**不要**把 `response` 一起傳（content-length 會對不上，6f-1 CR L7）。
       - `mediaId` 是**字串**的 local media id（`glossaryService.ts:4-6` 的檔頭說明），`ManageSubtitleDialogV2` 傳的是 `glossaryMediaId ?? mediaId`——用 `*` 通配就好，不要自己拼。
     - 量進場：先等 `getAnimations()` 的 `finished`（`Promise.allSettled(...).then(() => undefined)`，6f-1 CR L1），再量 bounding box。
     - 390 寬要斷言：面板的 computed `animation-name` 是 `sheet-enter`；貼底（`Math.round(box.x) === 0`、`width === 390`、`y + height === 844`）；把手可見；✕ ≥44×44；**每一列都看得到「編輯」**；**列是兩行**（`glossary-delete-{id}` 的 `box.y` > 原文的 `box.y`，且列高 ≥ 76——實測兩行的列是 **86**）；**沒有橫向溢出**（面板內容區的 `scrollWidth <= clientWidth`）；「全部確認」「新增詞彙」寬 ≈ 390−32 且**在列表下面**（`box.y` > 最後一列的 `box.y`）；點「新增詞彙」後兩個輸入框各自 ≈ 滿版、上下排列。
     - **640 寬（斷點的另一側）**重開一次：`animation-name` 是 `dialog-enter`、沒有把手、列是**一行**（刪除鈕與原文的 `box.y` 大致相同）、兩顆鈕沒有被拉寬。
     - 只產 darwin 基準線；`-linux` 交給 CI bootstrap（`project_visual_baseline_intentional_change` 四步）。視覺測試只有**一支** test，`--grep` 濾不到夾具，只能整支跑再還原無關的。

9. **另立的單子（建單時已寫入 sprint-status）。**
   - 新：`disc-2026-09-glossary-footer-count-not-in-design`、`disc-2026-09-glossary-mobile-state-screens-missing`。
   - 既有（補本張的交接註記）：`disc-2026-09-glossary-panel-a11y-leftovers`、`disc-2026-09-11px-micro-label-not-on-type-scale`、`disc-2026-09-dialog-close-target-44px`、`disc-2026-09-viewport-fit-cover-missing`、`dsr-6f-flow-f-mobile`（傘狀）。

10. **CI 全綠**：`pnpm run lint:all`、`pnpm nx run web:typecheck --skip-nx-cache`、`python3 scripts/check-design-tokens.py`、`pnpm nx test web`、`pnpm nx test api`。⛔ 絕不用 `run_in_background` 跑測試；每次跑完 `pnpm run test:cleanup`；Nx 一律 `NX_DAEMON=false`。
    - 🚨 **合併之後要看 `main` 那一次的 CI，不只看 PR 的**（dsr-6e-2 的教訓）。新的非同步測試一律 `findBy*`／`waitFor`。

## Tasks / Subtasks

- [x] **Task 1 — 設計稿（AC: #1）**
  - [x] 新母版 `Component/GlossaryRow-v2/Mobile`（`t28C2s`，358×91）；`buepS` 補標題列＋44×44 ✕、四列改新母版且「編輯」全開、sheet 重新貼底（y 359→193、高 651）
  - [x] Component Library 補格（`ERXTn`）；規格註記（`x1MC3W`）；`ctx.problems` = 70（＝基準）；存檔後 stat 確認落盤；匯出後只留 `f6-m-v2.png`、`component-library.png`、`pen-tokens.json`
- [x] **Task 2 — 面板變 sheet（AC: #2, #7）**
  - [x] 先寫紅測試（5 條，確認全紅）→ 外殼／標題列／內容區／頁尾計數／兩顆鈕搬到下面
- [x] **Task 3 — 列換兩行（AC: #3, #7）**
  - [x] 先寫紅測試（3 條紅＋1 條守）→ 兩層 `sm:contents` 包裝＋兩顆 spacer；`expectExactClasses` 的四個元素一字未動
- [x] **Task 4 — 新增表單與刪除確認（AC: #4, #5, #7）**
- [x] **Task 5 — 夾具、基準線、e2e（AC: #8）**
  - [x] 兩個手機夾具（Portal 的 `seeded-mobile` ＋ 一張頁內列）；`glossary-mobile.spec.ts`（390 與 640 兩側都量，3 條，burn-in 9/9）
- [x] **Task 6 — 收尾（AC: #6, #9, #10）**
  - [x] dev-story Step 9：`f6-m-v2` 對新的手機基準線；sprint-status 補 5 條交接註記＋1 張新單

## Dev Notes

### 這張的重點

- **它是 6f-1 的消費者。** 三樣地基都現成，本張不新增任何共用機制——所有新東西都是 `max-sm:`／`sm:` 的成對 class 與兩層 `sm:contents` 包裝。
- **桌機零變動是紅線。** `sm:contents` 這個手法就是為了這件事：桌機上包裝層整個消失，DOM 盒模型與今天完全一樣，既有六張 1280 基準線才不會動。
- **最容易踩的雷是測試，不是版面。** 🔴 #11（class 完全相等）與 🔴 #12（`max-w-[880px]` 要加前綴）各會讓人卡半小時，兩條都在 AC 裡明寫了。
- **稿比程式碼舊。** 手機稿把「編輯」關掉是 390 寬塞不下的權宜，dsr-6c 已經裁定過產品規則（所有列都有「編輯」）——**先改稿，再對程式碼**，不要反過來把程式碼改成舊稿。

### 上游契約（Rule 20 ack）

- 本張不消費也不改任何線上契約（純呈現層）。`glossaryService` 的六條路由、`useGlossary` 的五個 mutation 一行都不動。

### 建單裁定（2026-09-20，Sally／Alexyu 可在 review 推翻）

1. ⚖️ **列在手機換行成兩行，不用 ⋯ 溢出選單**（**Alexyu 2026-09-20 當場裁定**，採建議選項）。量測見 Context。
2. ⚖️ **新增 `Component/GlossaryRow-v2/Mobile` 母版，而不是在 `buepS` 裡手畫四個本地 frame。** 依據 `DESIGN.md:786`「沒有母版怎麼辦：先補母版，再畫畫面」與 `:788`（新母版要在 Component Library 補一格），以及 `.pen` 既有的斜線變體慣例（`ButtonCost/Default`、`PosterCard-v2/Hover`／`/Selectable`／`/Unmatched`；程式碼端唯一的斜線前例是 `ui/ButtonCost.tsx:1`）。程式碼端仍是**同一個元件兩種外觀**（`max-sm:`），不拆成兩個 React 元件——這與 6f-1 對 sheet 的裁定一致。
3. ⚖️ **保留頁尾計數（共 N 條 · M 條未確認），手機也留。** 桌機稿上有（`VbxVw`），**只有手機稿 `Jm7RD` 沒畫**——也就是說缺的是稿不是功能。一旦清單長到要捲動，它是唯一告訴你「還有幾條沒確認」的地方，而手機正是最容易捲動的場合。本張只把它的左右內距對齊 sheet 並接上 safe-area；要不要補進手機稿由 Sally 裁定（`disc-2026-09-glossary-footer-count-not-in-design`）。
4. ⚖️ **兩顆動作鈕依稿放在列表下面（進捲動區，不做 sticky 頁尾）。** 稿 `Jm7RD` 根本沒有頁尾，`o938ZZ`／`YMF10` 就在列表之後；單片電影的詞表通常很短，而「新增詞彙」不是手機上的主要任務。若日後詞表變長到抱怨，再由 Sally 決定是否改 sticky。
5. ⚖️ **F6-SPEC-STATES 的七個狀態不另畫七張手機稿**，改用一條規格註記說明「手機沿用同一套元件、照本註記的版面規則直排」。理由：七張稿的維護成本遠高於一條規則，而且這七個狀態在手機的差別全部是「直排 vs 橫排」這一件事。
   - ⚠️ **有反向前例**：`disc-2026-08-home-v3-missing-state-frames` 當時 Alexyu 裁的是「三張全補、現在就做」。如果他這次也要補，最可能值得單獨出圖的是「新增表單」與「刪除確認」兩個（它們在手機的版面真的不同，不只是直排）。立 `disc-2026-09-glossary-mobile-state-screens-missing` 記錄這個缺口與這個前例。
6. ⚖️ **刪除確認只換外殼三樣，不重排內容**（比照 6f-1 對批次／同意／確認三個對話框的做法）。

### 不要做的事

- 不要改 `ui/Dialog.tsx`、`ui/mobileSheet.tsx`、`styles.css`、`useGlossary.ts`、`glossaryService.ts`。
- 不要改 `ManageSubtitleDialogV2.tsx`（6f-1 已做完；它的 spec 還把名詞表 mock 掉，本張的改動在那裡看不到）。
- 不要改既有六個 glossary 視覺夾具（尤其不要把 `width: 720` 換成 `viewport`——會撞守門測試，也會毀掉既有基準線）。
- 不要動 `text-[11px]`、`min-h-[54px]`、來源徽章的顏色與形狀（dsr-6c 已裁定：五種來源一律中性藥丸）。
- 不要本機產 `-linux.png`。
- 不要為了拍照在元件上開測試用的後門。

### 已知陷阱

- **`expectExactClasses` 會因為多一個 class 而紅**，錯誤訊息只說長度不符（`GlossaryRowV2.spec.tsx:23-28`）。
- **`MOBILE_SHEET_CONTENT` 帶著不加前綴的 `w-full`／`max-w-none`／`translate-x-0`／`translate-y-0`／`rounded-b-none`／`rounded-t-[var(--radius-xl)]`**：桌機的對應值（寬度、`max-w`、位移、**圓角**）沒加 `sm:` 就會被 twMerge 丟掉；把 `MOBILE_SHEET_CONTENT` 挪到最後則換成桌機變形。加 `sm:` 是唯一兩邊都對的寫法。**圓角是最容易漏的那一個**（AC #5）。
- **頁內的手機夾具只有 326 寬**（`/test/gallery` 的 `p-8`），不是 sheet 的 358；`width` 與 `viewport` 互斥，撐不上去。在那 326 裡看到的溢出，在真 sheet 裡不存在——不要去追。
- **`display: contents` 與 `order` 是一組**：header-row 沒有 `max-sm:contents`，兩顆鈕的 `max-sm:order-last` 就是空話（它們會在 header-row 內部排序，而 header-row 還在原位）。
- **`ml-auto`／`flex-1` 會取消 `align-items: stretch`**（6f-1 AC #5 的教訓）：滿版按鈕沒滿版，先看外面還有沒有一層。
- **同一支 visual test 跑完所有夾具**：viewport 不重設就會污染後面一百多個夾具（harness 已處理，但新增夾具時仍要跑完整支確認）。
- **`MobileTabBar` 在 390 寬會畫出來**（`z-40`、高 84、`sm:hidden`，掛在 `AppShellV2` 上而 `/test/gallery` 也在 shell 裡）：在頁內的手機夾具若落在視窗底部會被它蓋到。
- **e2e 的 glob 是錨定的**，但帶 query 的 URL 要寫 `?*`（6f-1 踩過 `**/transcribe` 對不上 `?translate=true`）。
- **假 API 回 snake_case**：`Rule 18` 的 `snakeToCamel` 在前端做，回 camelCase 的假資料會被再轉一次而失效。
- **行號以建單時為準**（2026-09-20，main `48f3e268`）。

### Source tree

```
ux-design.pen（新母版 + buepS + 規格註記）、_bmad-output/pen-tokens.json、
  _bmad-output/screenshots/flow-f-subtitle-v2/f6-m-v2.png              ← Task 1
apps/web/src/components/subtitle/GlossaryPanelV2.tsx(+spec)            ← Task 2, 4
apps/web/src/components/subtitle/GlossaryRowV2.tsx(+spec)              ← Task 3, 4
apps/web/src/routes/test/-gallery.fixtures.tsx                          ← Task 5（兩個新夾具）
tests/visual/components.visual.spec.ts-snapshots/…（兩張新 darwin 基準） ← Task 5
tests/e2e/glossary-mobile.spec.ts                                       ← Task 5（新）
_bmad-output/implementation-artifacts/sprint-status.yaml                ← Task 6
```

### Cross-Stack Split Check

後端 task **0 個**，前端／設計／測試 task 6 個。→ 不觸發跨棧拆分。規模與 `dsr-6f-1` 相當（兩個元件、一張稿、三個夾具、一支 e2e），不再拆。

### Time-dependent visual coverage

- **N/A — no wall-clock-reading components touched.** `GlossaryPanelV2`、`GlossaryRowV2` 都不讀 `Date.now()`／`new Date()`／`Date.UTC()`／`Date.parse()`（已逐行確認）；`useGlossary.ts` 是純 server state，沒有時間相依。新夾具不需要 `clockTime`。

### References

- [Source: `apps/web/src/components/subtitle/GlossaryPanelV2.tsx:1, 45-49, 164-240, 260-306, 308-408`、`GlossaryRowV2.tsx:1-2, 30, 96-208, 210-243`]
- [Source: `apps/web/src/components/ui/Dialog.tsx:27, 37-58, 63-71, 75-83`、`ui/mobileSheet.tsx:22-44`]
- [Source: `apps/web/src/components/subtitle/ManageSubtitleDialogV2.tsx:208, 467-478, 482, 714-735, 917-922`（本張只讀，不改）]
- [Source: `apps/web/src/components/subtitle/GlossaryRowV2.spec.tsx:23-28, 69-72, 82-85, 202-205, 287-290, 309-318`、`GlossaryPanelV2.spec.tsx:202-214, 216-234, 261-276`]
- [Source: `apps/web/src/routes/test/-gallery.fixtures.tsx:416-430, 4263-4360, 4620-4691`、`routes/test/gallery.tsx:274-276, 348, 364-371, 387-397`、`routes/test/gallery-fixture-viewport.spec.ts:12-25`]
- [Source: `tests/visual/components.visual.spec.ts:72-80, 174-207, 274-285, 343-352`、`tests/e2e/manage-subtitle-mobile.spec.ts:25-78, 87-208`]
- [Source: `apps/web/src/components/shell/MobileTabBar.tsx:26-27`、`shell/AppShellV2.tsx:110`]
- [Source: `apps/web/src/services/glossaryService.ts:84-138`、`hooks/useGlossary.ts:13-58`]
- [Source: `ux-design.pen` `buepS`／`Jm7RD`／`yEb9p`／`uMwil`／`I3zyy`／`gG1CX`／`P7NUD`／`pwW3l`／`QkHtp`／`ngusL`／`RVb6N`／`o938ZZ`／`YMF10`／`nDSEd`／`dlfMR`／`GGwqu`／`uDU3E`／`Di88y`／`n3vIR`／`YDPhc`／`JzmvC`／`PyB9P` —— SM 建單時以 Pencil MCP 逐節點讀出（2026-09-20）；程式碼現況由唯讀稽核代理查證]
- [Source: `DESIGN.md:351-360`（字階）、`:604-615`（手機上什麼會變：覆蓋層一律改 Sheet）、`:618-622`（觸控目標 44×44）、`:626-636`（BottomSheet 規格，最大高度寫的是螢幕 80%）、`:786-788`（先補母版再畫畫面＋Component Library 補格）、`:836`（`ctx.problems` 基準——該處寫 74，現況是 70）]
- [Source: `_bmad-output/implementation-artifacts/dsr-6f-1-mobile-sheet-and-manage-dialog.md`（地基與 CR 教訓）、`dsr-6c-glossary-panel.md:38-42, 66-72, 216-226`（⚖️ 來源徽章中性、⚖️ 保留「編輯」、明文把 F6-M 留給 6f）]
- [Source: `sprint-status.yaml` → `dsr-6f-2-glossary-mobile`（建單稽核）、`dsr-6f-flow-f-mobile`（傘狀）、`disc-2026-09-glossary-panel-a11y-leftovers`、`disc-2026-09-11px-micro-label-not-on-type-scale`、`disc-2026-09-dialog-close-target-44px`、`disc-2026-09-viewport-fit-cover-missing`]
- [Source: project-context.md#Rule 16 / #Rule 20 / #Rule 21 / #Rule 23 / #Rule 24；`.claude/memory/project_visual_baseline_intentional_change.md`、`feedback_measure_the_breakpoint_you_return_to.md`、`feedback_verify_interactive_states_not_read_them.md`、`feedback_verify_pen_saved_before_commit.md`、`feedback_pencil_spec_standalone_screen.md`]

## Dev Agent Record

### Agent Model Used

Claude Opus 5 (1M context) — dev-story (Amelia)

### Debug Log References

- Nx 一律 `NX_DAEMON=false`；單檔 vitest（`cd apps/web && npx vitest run <spec>`）只要幾秒，整包 `nx test web` 約 50 秒——紅綠迴圈用前者，閘門用後者。
- **本機 e2e 起不來**：`playwright.config.ts` 在本機自己起後端，但 `AI_PROVIDER` 預設 `gemini` 而 `.env` 沒有 `GEMINI_API_KEY`，後端 exit 1，Playwright 只說 `Process from config.webServer was not able to start`。繞法 `AI_PROVIDER=claude npx playwright test …`。已立 `preexisting-fail-e2e-local-ai-provider`（CI 不受影響）。
- Pencil `execute` 沒有 `console`／`ctx`／`Save`：要讀值就 `throw new Error(JSON.stringify(Get(...)))`（**會連同這一批編輯一起 rollback**，所以量測與寫入要分兩次呼叫）；存檔要對 Pen.app 送 Cmd+S（`osascript`），再用 `stat` 確認 mtime／size 真的變了。

### Completion Notes List

**做了什麼（對使用者的差別）**

1. **手機上打開「名詞對照表」，它從底部滑上來了。** 以前它是螢幕中間一塊四角不圓、沒有邊框的方塊，進場是原地放大 4%（桌機動畫）。現在是貼底的抽屜：上緣圓角、有把手、由下往上滑入。真瀏覽器量過——390 寬算出來的 `animation-name` 是 `sheet-enter`、貼齊 (0, 390, 844)；640 寬是 `dialog-enter`、置中、下方留白。
2. **關閉鈕從 16×16 變成 44×44。** 手機上它是唯一一顆按得到的出口（沒有頁尾的「關閉」，手機也沒有 Esc）。e2e 量過 ≥44×44。
3. **每一列變成兩行，而且「編輯」回來了。** 第一行是「原文 → 譯名」，第二行是「來源徽章／未確認」靠左、「確認／編輯／刪除」靠右。以前所有東西擠一行，在 390 寬會被切掉；設計稿則是把四列的「編輯」全部關掉來換寬度——那與 ⚖️ 2026-09-17 的裁定（所有列都要能改譯名）牴觸，所以本張改的是**稿**，不是把功能拿掉。e2e 量過：三列都看得到「編輯」，刪除鈕在原文**下方**，列高 86，而且**整列與內容區都沒有橫向捲動**。
4. **「全部確認」「新增詞彙」搬到列表下面、各自滿版**（依稿 `o938ZZ`／`YMF10`）。e2e 量過寬度 358（＝390−32）、位置在最後一列之下。
5. **新增詞彙的表單在手機直排**：兩個輸入框各佔一行、各自滿版，「新增／取消」自己一行。以前兩個 `w-40`（160）輸入框加兩顆按鈕硬塞一行，330 的可用寬度一定爆。
6. **刪除確認也變成抽屜**，兩顆按鈕滿版。

**桌機零變動怎麼做到的**

兩層 `sm:contents` 包裝：在桌機上包裝層整個消失（`display: contents`），DOM 盒模型與 dsr-6c 出貨的版本逐位元相同；在手機上它們才是真正的兩行／兩區塊。所有新 class 都是 `max-sm:`／`sm:` 成對。**六張既有的 glossary 1280 基準線一張都沒變**（整支 visual 跑過，`git status` 只多出兩個新夾具）。

**建單時的三個警告都真的踩到了**

| 警告 | 實際發生 |
| --- | --- |
| `GlossaryRowV2.spec.tsx` 用 class **完全相等**比對 | 四個被鎖住的元素（兩個徽章、編輯輸入框、刪除鈕）**一個字都沒動**，所以沒踩到。AC 的設計是對的。 |
| `max-w-[880px]` 必須加 `sm:` 前綴 | 是。改成 sheet 之後不加前綴，twMerge 會讓 `max-w-none`／`w-full` 與它打架；`:202-214` 依交代改寫。 |
| 刪除確認的**圓角**也要加前綴（對抗驗證 C2） | 是。`rounded-[var(--radius-lg)]` → `sm:rounded-[var(--radius-lg)]`，`:309-318` 兩條一起改寫。 |

**頁內手機夾具：CR 之後拿掉了（M7）**

原本加了一張 `glossary-row-v2/unconfirmed-mobile`。它在 `/test/gallery` 只有 **326** 寬（頁面自己的 `p-8` 吃掉 64），不是真 sheet 的 358，圖上「刪除」會換到第三行。CR 指出把那張圖收成基準線等於**把一個沒有任何畫面會長成這樣的版面當成參考答案**，而且日後任何第二行尺寸微調都會讓它變動、看起來像回歸。已刪除（連同基準線）。真 390 寬的證據由 Portal 的 `glossary-panel-v2/seeded-mobile` 與 e2e 量測負責。

**刻意留下的**

- 頁尾計數（共 N 條 · M 條未確認）手機保留（⚖️ 建單裁定 3）。桌機稿 `GGwqu` 上有這一列（`VbxVw`），**手機稿沒畫**——已立 `disc-2026-09-glossary-footer-count-not-in-design` 給 Sally。
- 兩顆動作鈕的視覺順序與 Tab 順序分家（`order-last` 的本質），已掛進 `disc-2026-09-glossary-panel-a11y-leftovers`。
- 徽章仍是 `text-[11px]`（凍結）、`safe-area` 今天仍是 0。
- 設計稿上把手與標題列之間有 sheet 的 gap 14，程式碼的把手是緊貼標題列的（6f-1 的共用 `SheetGrabber`，四個 sheet 共用，不為這一張改）。

**閘門結果**

| 閘門 | 結果 |
| --- | --- |
| `pnpm run format:check` | ✅ |
| `pnpm run lint:all` | ✅ 0 errors（129 warnings 全是既有批次；**本張碰到的 6 個檔案 0 warning**） |
| `pnpm nx run web:typecheck --skip-nx-cache` | ✅ |
| `python3 scripts/check-design-tokens.py` | ✅（母版 73→74、畫面 196） |
| `pnpm nx test web` | ✅ 3946/3946（274 檔；新增 11 條測試） |
| `pnpm nx test api` | ✅（沒改後端） |
| e2e（chromium） | ✅ 4/4，burn-in `--repeat-each=3` 12/12 |
| 視覺 | ✅ 既有基準線**零變動**；新增 1 張 darwin（`seeded-mobile`），**跑三輪都穩定**。三輪唯一紅的都是本機固定會漂的 `retry-retry-notifications`（已還原，與本張無關） |


**🔍 /ship 對抗式 CR（2026-09-21，fresh-context 代理，只讀；用 repo 自己的 Tailwind 4.1.18 與 tailwind-merge 3.4.0 實編、用 Pencil 逐節點讀 `.pen`、在真瀏覽器量兩種寬度）**——3 HIGH／4 MEDIUM／3 LOW，**修 10、駁回 0**。

| 等級 | 問題 | 處置 |
| --- | --- | --- |
| **H1** | e2e 的「沒有橫向溢出」兩條斷言**不可能失敗**——會換行的 flex 容器永遠不會溢出，它只會變高。本張唯一真正的風險（第二行 9px 餘裕用完）等於沒有守門，而且列高只有下界沒有上界，「刪除」掉到第三行也照樣綠。 | 修：改成量「列高 ≤ 96」＋「來源徽章與刪除鈕在同一條水平線」（跟 640 那側同一招）；拿掉假斷言。 |
| **H2** | **真 bug**：第一行完全沒有保險。`term_src` 是 `shrink-0`、沒有 `truncate`、父層沒有 `flex-wrap`——一個長詞（字幕／中繼資料抽出來的多字專有名詞是常態）就會撐破 328px 的手機行，而 `overflow-y-auto` 的內容區會因此長出橫向捲軸。桌機有約 690px 所以從沒發作；e2e 只餵三個短詞。 | 修：第一行加 `flex-wrap`（先換行、不失字），`term_src` 加 `truncate max-sm:min-w-0 max-sm:shrink` 當最後一道。四個被 class 完全相等鎖住的元素仍未動。 |
| **H3** | AC #5 白紙黑字要求「✕ 的垂直位置要用瀏覽器看過」，而刪除確認那層**從頭到尾沒有被任何瀏覽器打開過**（e2e 不點刪除、沒有夾具、只有 jsdom 的 class 斷言）。 | 修：新增一條 e2e 打開第三層 sheet。**一跑就紅**——✕ 真的壓到「刪除詞彙」。連帶修：標題加 `max-sm:pr-12` 留出 48px（其他三個 sheet 是靠標題列的 `pr-12` 拿到的），測試改用 `Range` 量**文字**而不是整個區塊框（`DialogTitle` 是滿版元素，全 app 每個 `p-6` 對話框的框都會壓到 ✕）。 |
| **M4** | `glossary-actions` 外層是無條件渲染的，但裡面兩顆鈕都是條件的——空清單與載入中在手機會多出一個 0 高的 flex 子項，白吃內容區 14px 的 gap。 | 修：外層跟著條件走；補一條測試。 |
| **M5** | `glossary-header-row` 這個 testid 被裝到**標題列**上，而 AC 指名的是帶 `max-sm:contents` 的工具列——測試跟著改名所以不會紅，但下一張單子照 AC grep 會找錯元素。 | 修：標題列改名 `glossary-title-bar`，工具列拿回 `glossary-header-row`。 |
| **M6** | 手機上「新增詞彙」在列表**下面**，但它打開的表單在列表**上面**——詞表一長到要捲動，按下去表單就開在看不見的地方。 | 修：表單加 `max-sm:order-[9998]`，剛好落在動作鈕（`order-last` ＝ 9999）上面；補一條測試。 |
| **M7** | 頁內的 326 寬夾具把一張沒有任何畫面會長成這樣的版面收成基準線，而且日後動第二行尺寸就會變動、看起來像回歸。 | 修：刪掉夾具與基準線（見上）。 |
| L8 | 刪除確認頁尾兩顆鈕的 `max-sm:w-full` 是空轉的（`DialogFooter` 在手機是 `flex-col-reverse`，預設 `align-items: stretch` 本來就滿版），而且還有一條測試在守這個空轉。 | 修：兩個都拿掉。 |
| L9 | 兩顆 spacer 只有一顆帶 `aria-hidden`；一個沒有插值的樣板字串。 | 修。 |
| L10 | 「已確認的列也有編輯」那條測試被寫成紅測試，但它在 `main` 上本來就會過（**稿**才是把編輯關掉的那一方）——AC 明文要求誠實標成「守」。 | 修：標題改成 `[guard]`，並在註解說明它守的是什麼。 |

代理同時**實測確認**（這些是對的）：`sm:contents`／`max-sm:contents`／`order-last` 在本 repo 的 Tailwind 下產生順序是 base → `max-sm:` → `sm:`，**639/640 沒有死角**；twMerge 對兩個 `cn()` 鏈零丟棄，桌機 ≥640 解析回與 `main` 逐字相同的定位／寬度／圓角（刪除確認的圓角陷阱確實避開了）；四個 class 完全相等的元素逐字未動；Esc 在三層巢狀下只關最上層（Radix `dismissable-layer` 只對 `index === layers.size - 1` 作用），`INLINE_EDITOR` 的 `closest` 穿得過新的包裝層；e2e 的假 API **可證明有被吃到**（沒有 stub 就只會渲染 `glossary-empty`，不會有 `glossary-list`），glob 在 CI 的絕對 URL 下仍然對得上，`webkit-core` 的 `testMatch` 不含這支所以不會撞 `isMobile` 裝置設定；`.pen` 逐節點正確（`t28C2s` 不與任何母版重疊、`Jm7RD` 193+651 = 844 正好貼底且 ≤675、✕ 不壓標題、四個 instance 的覆寫一個不多一個不少、`ctx.problems` 仍是 70）。

- 🔗 AC Drift: NONE（checked: `max-w-\[880px\]|expectExactClasses|glossary-confirm-all|glossary-add-term|rounded-\[var\(--radius-lg\)\]` across `_bmad-output/implementation-artifacts/*.md`——命中 dsr-6c（名詞表桌機）、dsr-6b／6d-b／6e-2（其他對話框的 880／480 寬）。全部是 REUSE：本張只把那些值加上 `sm:` 前綴，**≥640 的幾何完全不變**，而且 `sm:max-w-[880px]` 本來就是 dsr-6b／6d-b 的既有寫法。dsr-6c ⚖️ 裁定 ②「所有列保留編輯」本張是**執行**它（改手機稿），不是推翻。dsr-6f-1 AC #3 明文把 `GlossaryPanelV2`／`GlossaryRowV2` 留給本張。）
- 📎 Contract Stamps: NONE（本張與上游 `dsr-6c`／`dsr-6f-1` 都沒有 `[@contract-v*]` 標記——純呈現層，不定義也不消費線上契約）
- 🎭 A11y Pre-Flight: PASS（2 個元件＋4 個支援檔；`eslint` 對本張碰到的 6 個檔案回 **0 warning**）。四類複查：圖片尺寸 N/A（無 `<img>`）；焦點管理沿用 Radix，巢狀關係與關閉後回焦不變；`aria-live` N/A（沒有新的非同步揭露內容，既有的 `role="alert"` 未動）；自訂元件 N/A。新增的手機 spacer 帶 `aria-hidden="true"`；`sm:contents` 只套在兩個純 `<div>` 上（無隱含語意，不影響無障礙樹）。⚠️ 新增一項：`order-last` 讓視覺順序與 Tab 順序分家（見上）。
- 🎨 UX Verification: PASS —— `f6-m-v2` 對新的手機基準線 `glossary-panel-v2/seeded-mobile` 與 e2e 量測：

| Area | Design Spec (`buepS`) | Implementation | Match? |
| --- | --- | --- | --- |
| 外殼 | 貼底、上緣 `$radius-xl`、把手 36×4 | 同；由下往上滑入（e2e 量過 `sheet-enter`、(0,390,844)） | ✅ |
| 標題列 | 44 高、無底線、BodyLg 600 | 同（`max-sm:h-11 max-sm:border-b-0`） | ✅ |
| 關閉鈕 | 44×44、圖示 18 `$text-secondary` | 同（e2e 量過 ≥44×44） | ✅ |
| 說明 | Body `$text-secondary` | 同 | ✅ |
| 列 | 兩行；padding [8,14] gap 8；徽章靠左、動作靠右 | 同（`max-sm:flex-col max-sm:gap-2`＋兩層 `sm:contents`） | ✅ |
| 「編輯」 | 四列全有（本張改稿後） | 三列全有（夾具三筆） | ✅ |
| 兩顆動作鈕 | 列表下面、各自滿版 h44 | 同（e2e 量過 358、在最後一列之下） | ✅ |
| 頁尾計數 | 手機稿沒畫（桌機稿 `VbxVw` 有） | **有**（⚖️ 裁定 3，已立單給 Sally） | ⚠️ 刻意 |
| 把手與標題的間距 | sheet 的 gap 14 | 把手緊貼標題列（6f-1 共用 `SheetGrabber`） | ⚠️ 刻意 |
| 徽章字級 | Label 12 | `text-[11px]`（凍結） | ⚠️ 既有單 `disc-2026-09-11px-micro-label-not-on-type-scale` |


### Discovery Triage

- **Did this story discover any work outside its current scope?** **YES** —— 建單當下處理如下。

- **① expand-scope-in-place**
  - 手機上名詞表根本不是 sheet、✕ 只有 16×16（🔴 #1、#2）→ **AC #2**
  - 稿上沒有 ✕、也沒有標題列（🔴 #3）→ **AC #1**
  - 稿把四列的「編輯」全關掉，與 ⚖️ 2026-09-17 的裁定牴觸（🔴 #7）→ **AC #1、#3**
  - 新增表單與刪除確認在手機同樣擠成一行（🔴 #9、#10）→ **AC #4、#5**
  - 名詞表沒有任何 e2e 或手機視覺覆蓋（🔴 #13、#14）→ **AC #8**

- **② spawn-blocking-story**：無。

- **補記到既有條目**
  - `disc-2026-09-glossary-panel-a11y-leftovers` — 本張的 `max-sm:order-last` 讓兩顆動作鈕的**視覺順序在列表之後、DOM／Tab 順序仍在列表之前**（刻意接受，見 AC #2）。
  - `disc-2026-09-viewport-fit-cover-missing` — 本張新寫的兩處 `env(safe-area-inset-bottom)` 今天同樣是空轉的。
  - `disc-2026-09-bottomsheet-grabber-three-variants` — 母版說 80%、程式碼 `max-h-[85vh]`（全 repo 14 處）的出入，本張沿用程式碼值、對稿用 675。

- **③ backlog-with-carry-forward-link**（建單時立 ＋ 實作時新增）
  - `disc-2026-09-glossary-footer-count-not-in-design` — 頁尾計數**只有手機稿沒畫**（桌機稿 `GGwqu` 的 `VbxVw` 有；建單裁定 3；owner: Sally）
  - `disc-2026-09-glossary-mobile-state-screens-missing` — F6-SPEC-STATES 的七個狀態只有桌機稿（建單裁定 5；owner: Sally）
  - `preexisting-fail-e2e-local-ai-provider` —（**實作時新立**）本機跑 e2e 時 Playwright 自己起的後端會因為 `AI_PROVIDER` 預設 gemini 而 exit 1，錯誤訊息看起來像測試壞掉。繞法 `AI_PROVIDER=claude`；CI 不受影響
  - `disc-2026-09-api-coverage-not-gitignored` —（**實作時新立**）`pnpm nx test api` 產生的 `apps/api/coverage/` 沒有被忽略（`.gitignore:63` 的 `/coverage` 是根目錄錨定），下次 `git add -A` 會被掃進去。本機已刪，**沒有改 `.gitignore`**（超出範圍）

### File List

**新增**

- `tests/e2e/glossary-mobile.spec.ts`
- `tests/visual/components.visual.spec.ts-snapshots/components/glossary-panel-v2/seeded-mobile/default-visual-darwin.png`
- `_bmad-output/implementation-artifacts/dsr-6f-2-glossary-mobile.md`（建單產物）

**修改**

- `apps/web/src/components/subtitle/GlossaryPanelV2.tsx`（+spec）
- `apps/web/src/components/subtitle/GlossaryRowV2.tsx`（+spec）
- `apps/web/src/routes/test/-gallery.fixtures.tsx`（兩個 390×844 夾具）
- `ux-design.pen`（新母版 `Component/GlossaryRow-v2/Mobile` `t28C2s`；`buepS` 標題列 `A9bh0`＋✕ `HAhWJ`／`J4neyT`、四列 `Z5okpJ`／`H4H0y`／`q6oWR0`／`LCw4G`、sheet 重新貼底；Component Library 新格 `ERXTn`；規格註記 `x1MC3W`）
- `_bmad-output/pen-tokens.json`（masters 73→74、penSha256）
- `_bmad-output/screenshots/flow-f-subtitle-v2/f6-m-v2.png`
- `_bmad-output/screenshots/design-system/component-library.png`
- `_bmad-output/implementation-artifacts/sprint-status.yaml`

## Change Log

| 日期 | 內容 |
| --- | --- |
| 2026-09-21 | 🔍 **/ship 對抗式 CR**：3H／4M／3L，**修 10、駁回 0**。最重要：① e2e 的「沒有橫向溢出」斷言不可能失敗（會換行的容器不會溢出），本張唯一的風險等於沒守門 → 改量列高上界＋第二行同一水平線；② **第一行完全沒有保險**，長 `term_src` 會撐破手機行並讓內容區長出橫向捲軸 → 加 `flex-wrap` ＋ `truncate`；③ AC #5 要求用瀏覽器看的刪除確認 ✕ **從沒被打開過**，補上 e2e **一跑就紅**——✕ 真的壓到「刪除詞彙」，標題補 `max-sm:pr-12`。另修：空清單多出的 14px 空隙、testid 與 AC 對不上、新增表單開在看不見的地方、把一張 326 寬的破版收成基準線、兩個空轉的 class 與一條假紅測試。web 3946/3946、e2e 12/12、既有基準線零變動。 |
| 2026-09-20 | 🚧 **REVIEW**（dev-story, Amelia）。Task 1–6 完成。手機上的名詞對照表變成由下往上的 sheet（e2e 量過 390 是 `sheet-enter`、貼底 (0,390,844)；640 是 `dialog-enter`、置中），✕ 44×44，**每一列換成兩行且「編輯」全部回來**（列高 86、整列零橫向捲動），「全部確認／新增詞彙」搬到列表下面各自滿版 358，新增表單直排，刪除確認也是 sheet。桌機靠兩層 `sm:contents` 包裝做到**逐位元不變**，六張既有 1280 基準線零變動。閘門：format ✅、lint 0 errors（本張 6 個檔案 0 warning）、typecheck ✅、design-tokens ✅、web 3944/3944、api ✅、e2e 3/3（burn-in 9/9）、視覺新增 2 張 darwin。 |
| 2026-09-20 | ✅ **Task 1（設計稿）完成**。新母版 `Component/GlossaryRow-v2/Mobile`（`t28C2s`，358×91，兩行：原文→譯名／徽章靠左＋確認·編輯·刪除靠右）建在 `04 · 列表與進度` 群組裡 `nDSEd` 右邊的空位（x=18540, y=-6214）。`buepS` 補上 44 高的標題列（`A9bh0`）與絕對定位的 44×44 ✕（`HAhWJ`，x=342 y=28，對齊程式碼 `MOBILE_SHEET_CLOSE` 的 `right-1`／`top-4` 幾何）；四列換成新母版的 instance，**「編輯」四列全開**（⚖️ dsr-6c 裁定 ②）；sheet 高度 485→651，y 359→**193** 重新貼底（651 ≤ 675 ＝ 螢幕 80%）。Component Library 補一格（`ERXTn`，`DESIGN.md:788`），規格註記 `x1MC3W` 放在 `PyB9P` 下方 40px（y=35348，與建單預測一致）。`ctx.problems` = **70**（＝現況基準，零新破版）。存檔以 stat 確認落盤（9642163→9654084 bytes），匯出 196/196 後只留 `f6-m-v2.png`、`component-library.png`、`pen-tokens.json`（masters 73→74），其餘 15 張重繪雜訊還原。 |
| 2026-09-20 | 🔍 **建單後對抗驗證**（fresh-context 代理，只讀；逐節點比對 `.pen`、逐行比對程式碼、用 repo 的 Tailwind 4.1.18 與 tailwind-merge 3.4.0 實際編譯候選 class、並在真瀏覽器量過兩種版面）：3 CRITICAL／11 SHOULD FIX／14 NIT，**全部併入**。最重要：① **桌機稿其實有頁尾計數**（`GGwqu` 的第三個子節點 `VbxVw`）——建單初稿寫成「兩張稿都沒有」，那句話原本要被寫進 `.pen` 的規格註記變成假事實；② 刪除確認的 `rounded-[var(--radius-lg)]` 沒加 `sm:` 前綴，twMerge 會把 sheet 的上緣圓角與貼底直角整個丟掉；③ 頁內的手機夾具只有 **326** 寬（頁面 `p-8`）不是 358，最寬的列在那裡會溢出 23px 被切掉，但在真 sheet 裡不會——夾具減成一張並註明它只證明 class 生效。另：可用寬度是 328（不是 330）、最寬第二行 319（餘裕 ~9px，補 `flex-wrap` 保險）、`ctx.problems` 現況是 70（不是 74）、新母版必須 stage `pen-tokens.json`（否則 `check-design-tokens.py` 會紅）並在 Component Library 補格、`GlossaryRowV2.tsx` 的 `// Implements:` 也要接新母版、`order-last` 造成視覺順序與 Tab 順序分家（刻意接受並記錄）。代理同時**實測確認**：`sm:contents`／`max-sm:contents`／`max-sm:order-last` 在本 repo 的 Tailwind 設定下都會生成、桌機 1280 與 640 兩個寬度下 10 個元素的 x/y/w/h 與現況**完全相同**、twMerge 對面板那半段的處方零丟棄、`display:contents` 套在純 `<div>` 上沒有無障礙風險。規模在每個軸上都小於 `dsr-6f-1`，不觸發拆分。 |
| 2026-09-20 | Story 建立（SM Bob, create-story）。SM 以 Pencil MCP 逐節點讀 `buepS`／`nDSEd`／`dlfMR`／`n3vIR`，唯讀代理查程式碼、測試與夾具在手機寬度的現況。找到 15 項；最重要的三個前提——① 名詞表是 6f-1 唯一漏網、在手機仍是置中對話框；② 稿把四列的「編輯」全關掉（不只未確認的兩列），與 dsr-6c 的裁定牴觸；③ `GlossaryRowV2.spec.tsx` 用 class 完全相等比對，多一個 `max-sm:` token 就紅。⚖️ Alexyu 當日裁定：手機用換行成兩行（不用 ⋯ 溢出選單）。另六項建單裁定，新立 2 張 disc 單。 |
