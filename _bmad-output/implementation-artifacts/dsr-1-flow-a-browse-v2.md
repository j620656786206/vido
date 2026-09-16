# Story DSR.1: Flow A 瀏覽 v2——程式碼與設計稿雙向對齊（桌機 A1p–A8p）

Status: done

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As the person who opens 媒體庫 more than any other page in this product,
I want 瀏覽頁的每一句話都說出它真正知道的事，而且設計稿指到的畫面真的還存在,
so that 載入失敗時我知道「檔案沒事」，篩不到東西時我知道是哪個篩選擋住的，
而不是讀到一句「請稍後再試」然後自己猜。

## Context

`epic-dsr` 的第九張（已收 4、5、7、9、10、11、12、13）。

⚠️ **這是 dsr 系列裡第二大的一塊，已拆成兩張。** `components/library/` 有 24 個元件、光是主要 10 個檔就 2,066 行，設計稿 10 張。

- **本張（dsr-1）＝桌機**：A1p-D 空白 / A2p-D 骨架 / A3p-D 網格 / A4p-D 列表 / A7p-D 無結果 / A8p-D 錯誤，＋ `Component/PosterCard-v2` 的三個新狀態母版，＋ `library/` 裡 7 個壞掉的 Rule 21 檔頭。
- **拆出 `dsr-1b`＝手機**：A1p-M / A2p-M / A3p-M / A6p-M（排序＋篩選 sheet），＋ E4-M。手機 sheet 是最大的一塊獨立工作（設計稿畫了四個分區、程式碼只有兩個），不該跟桌機擠在一起。

⚠️ **驗收基準是 `.pen` 節點值，不是 `_bmad-output/screenshots/flow-a-browse-v2/` 的 PNG。** Flow A 不在 `READABLE_FLOWS` 裡，那 10 張是 400px 縮圖，讀不到字。Task 1 先修。

⚠️ **方向仍然不是單向的。** 這次**文案幾乎全是「稿比碼好」**（稿會說「你的檔案沒有受影響」、會說出是哪個篩選擋住的；碼只說「請稍後再試」），但**結構有幾處是碼比稿新**（篩選軌、三態空白畫面）。逐條標了方向。

### 設計稿節點（逐字抄，不要重查）

| 代號 | 節點 | 內容 |
| --- | --- | --- |
| `A1p-D` | `vZpT8` | 空白資料庫（桌面） |
| `A2p-D` | `EsoIv` | 載入骨架（桌面） |
| `A3p-D` | `LcHBs` | 內容網格（桌面） |
| `A4p-D` | `b1H71g` | 列表檢視（桌面） |
| `A7p-D` | `R3FqJc` | 無結果（桌面） |
| `A8p-D` | `dVGIa` | 錯誤／重試（桌面） |
| `Component/PosterCard-v2` | `hD7Tw` | 母版 |
| `…/Hover` · `…/Selectable` · `…/Unmatched` | `L2ynz` · `fpKEv` · `n6Crb` | 三個新狀態母版 |

### 對照表

| # | 項目 | 設計稿 | 程式碼 | 方向 |
| --- | --- | --- | --- | --- |
| 1 | **7 個檔頭指向已刪除的節點** | `KNI8F`／`LZ8Ds`／`YEqii` 都不存在 | 7 個檔案照樣寫著它們 | **碼→稿** |
| 2 | 兩個檔頭還掛著 19-8 掃描的暫時佔位 | — | `LibraryListRowV2`／`LibraryStatesV2` 寫 `<screen-section — pending epic-19-8 mapping>` | **碼→稿** |
| 3 | 錯誤說明 | `媒體庫資料查詢失敗，你的檔案沒有受影響。` | `請稍後再試` | **稿→碼** |
| 4 | 無結果標題 | `找不到符合的結果` | `沒有符合條件的項目` | **稿→碼** |
| 5 | 無結果說明 | `沒有電影符合目前的篩選條件（4K）。試著調整或清除篩選。`（說出類型與生效的篩選） | `試著調整或清除目前的篩選條件。`（通用句） | **稿→碼** |
| 6 | 結果計數的**位置** | 在頁首、標題右邊 | 在工具列、`ml-auto` 靠右 | ⚖️**需裁定**（見 AC #7） |
| 7 | 結果計數的**單位** | `1,284 部` | `1,284 項` | ⚖️**需裁定**（同上） |
| 8 | 空白畫面 | 只畫一張（`片庫還是空的`） | **三態分類器**（NoFolder／NoQBT／ReadyForScan），文案都不同 | **碼→稿** |
| 9 | 搜尋框 placeholder | `搜尋媒體庫、TMDB…` | 待逐字核對 | 待驗 |
| 10 | 側欄導覽項 | 首頁／媒體庫／電影／影集／**動畫**／探索／活動／下載／系統／設定 | 待核對有沒有「動畫」 | 待驗 |
| 11 | PosterCard 三個狀態母版 | Hover／Selectable／Unmatched 已建 | `PosterCardV2` 的 hover／selectable／unmatched 形狀要對照 | 待驗 |
| 12 | 海報徽章形狀 | 母版：左上、`$overlay-scrim` 底、帶打勾圖示 | `PosterCardV2.tsx:144-155`：**右上**、不透明 `--bg-secondary` 墊底、**無圖示** | **稿→碼**（dsr-7 立案） |
| 13 | `text-[11px]` | — | `library/` 有 10 處 | ✅ **不動**（⚖️ 2026-09-16 裁定：11px 是全站微標籤標準） |
| 14 | `PosterCardV2` 的 `shadow-md` | — | **已不存在**（dsr-9 修掉了，只剩註解提到） | ✅ 條目過期 |
| 15 | 錯誤代碼膠囊 | `DB_QUERY_FAILED`，與說明同一行 | `（CODE）`，`text-[11px]`、`text-muted` | 待對形狀 |

---

## Acceptance Criteria

1. **Flow A 的稿要讀得到。** `READABLE_FLOWS` 加入 `"flow-a-browse-v2"`，重跑匯出，只 stage Flow A 的 10 張（其餘 `git checkout`）。

2. **7 個指向已刪除節點的檔頭修好。** 這是本張最重要的一條——它們**看起來是對的，其實指向虛空**，比缺標頭更糟（dsr-7 已在 `sAaCR` 上踩過同一顆雷）。已用 Pencil MCP 逐一驗證：
   - `KNI8F`（`Screen 1 Library Grid Desktop`）**不存在** → `LibraryGrid.tsx`／`LibrarySearchBar.tsx`／`ViewToggle.tsx`／`ParseFailureCard.tsx`／`RecentlyAdded.tsx` 共 5 個檔
   - `LZ8Ds`（`Screen 6 List View Desktop`）**不存在** → `LibraryTable.tsx`
   - `YEqii`（`Screen I5-D`）**不存在**；I5-D 這張稿還在，但節點是 `vpDLh` → `LibraryFilterRail.tsx`
   全部改成本張範圍內真實存在的畫面（網格類 → `A3p-D (LcHBs)`、列表 → `A4p-D (b1H71g)`、篩選軌 → `I5-D (vpDLh)`）。
   🚨 節點 ID **必須在 `Design ref:` 那一行**，不能放到下面的說明文字裡——`local/implements-pen-node-id` 只驗那一行（dsr-7 CR 第 11 項的判例）。

3. **兩個 19-8 的暫時佔位要收掉。** `LibraryListRowV2.tsx` 與 `LibraryStatesV2.tsx` 仍寫 `// Implements: <screen-section — pending epic-19-8 mapping>`，但 `project-context.md` Rule 21 白紙黑字寫著「As of story 19-8 (2026-05-20) **NO** `components/` file should still carry the pending placeholder」。這兩個是漏網的。改成 `Design ref:` 形式（列表列 → `A4p-D`；狀態元件 → `A2p-D`／`A7p-D`／`A8p-D`）。

4. **錯誤畫面要說「你的檔案沒有受影響」。** `LibraryStatesV2.tsx` 的 `LibraryErrorV2` 說明只有「請稍後再試」；稿（`KvrJp`）是 **`媒體庫資料查詢失敗，你的檔案沒有受影響。`**。媒體庫載不出來時，第一個閃過腦中的念頭是「我的檔案還在嗎」——稿回答了，碼沒有。改程式碼。

5. **無結果畫面要說出是哪個篩選擋住的。** 稿（`ys0uC`／`ttToQ`）是 `找不到符合的結果` ＋ `沒有電影符合目前的篩選條件（4K）。試著調整或清除篩選。`；碼是 `沒有符合條件的項目` ＋ 通用句。改程式碼：`LibraryNoResultV2` 接受目前的媒體類型與生效中的篩選摘要，組出同樣具體的句子。**沒有生效篩選時退回通用句**（那時具體化沒有意義）。

6. **空白畫面：稿要補上三態。** 程式碼有 `EmptyNoFolder`／`EmptyNoQBT`／`EmptyReadyForScan` 三個**文案完全不同**的空狀態（由容器分類），A1p-D 只畫了一張、而且文案是第四種寫法。**方向是碼→稿**：A1p-D 改成畫程式碼真的會渲染的那一張（以 `EmptyNoFolder` 為主稿，因為那是全新使用者一定先遇到的），另外兩態在同一張稿上加註記說明何時出現。⛔ **不要為了對稿把三態合併成一態**——那會刪掉一個已出貨的分類器。

7. **⚖️ 結果計數搬到標題旁、單位改「部」（Alexyu 2026-09-16 裁定：照稿）。** 稿把計數放在**頁首、標題右邊**、寫 `1,284 部`；碼放在**工具列、靠右**、寫 `1,284 項`。
   - **理由（Alexyu）**：「電影 1,284 部」讀起來是一句話，而且首頁讀數帶已經在用「3 部」「2 部失敗」——媒體庫用「項」會變成同一個產品兩套說法。
   - **碼**：把 `library-result-count` 從工具列搬到頁首標題右側，單位 `項` → `部`，`data-testid` 維持不變（既有斷言不動）。`ml-auto` 拿掉，改成標題那一行的 flex 成員。
   - **驗收**：既有引用 `library-result-count` 的測試全部仍綠；頁首在窄螢幕不換行擠壞（手機版面屬 dsr-1b，本張只確認不回歸）。

8. **PosterCard 的三個新狀態母版要跟程式碼對上。** `L2ynz`(Hover)／`fpKEv`(Selectable)／`n6Crb`(Unmatched) 是這次新增的，程式碼端 `PosterCardV2.tsx` 有對應的 hover／selectable／unmatched 分支。逐一比對**形狀與狀態色**（不是只看顏色）：hover 的浮起方式、selectable 的勾選記號位置與 `aria-pressed`、unmatched 的 fallback 版面。差異逐條記進 Dev Notes 並修對的那一邊。

9. **海報徽章的位置／底色／圖示三處不一致，本張修程式碼側還是母版側要先判。** dsr-7 已立案 `disc-2026-09-postercard-v2-badge-shape-drift`：稿的徽章在**左上**、`$overlay-scrim` 底、帶打勾圖示；碼在 `absolute right-1.5 top-1.5`（**右上**）、不透明 `--bg-secondary` 墊底、**沒有圖示**。
   - 不透明墊底是 critique R1 P0 特意加的（12% tint 直接壓在任意海報上量到 1.58:1），**碼是對的，稿要跟**。
   - 打勾圖示在「整理中」上是錯的（打勾＝完成），**碼是對的，稿要拿掉**。
   - 左上 vs 右上是純版面決定，**照碼**（右上，避開海報左上角常見的片名排版）。
   改完關掉該立案條目。

10. **既有測試全數保留通過**，只有 AC #4／#5 指名的文案可以改。特別是 `LibraryBrowseV2` 的篩選、選取模式、鍵盤操作那幾條。

11. **CI 全綠**：`pnpm run lint:all`、`pnpm nx run web:typecheck --skip-nx-cache`、`python3 scripts/check-design-tokens.py`、`pnpm nx test web`、`pnpm nx test api`。⛔ 絕不用 `run_in_background` 跑測試。

12. **視覺基準**：`poster-card`／`library-*` 有既有夾具，AC #9 若動到 `PosterCardV2` 會改到基準線。**預期會有 `-linux` bootstrap PR**——這與 dsr-7 不同，不要當成意外。⛔ 不要在本機跑 `test:visual:update` 產 `-linux.png`。

## Tasks / Subtasks

- [x] **Task 1 — 先讓稿讀得到（AC: #1）**
  - [x] `READABLE_FLOWS` 加入 `"flow-a-browse-v2"`
  - [x] 匯出併到最後一次跑（設計稿內容全程用 Pencil MCP 直讀，照 dsr-7／dsr-10 先例）

- [x] **Task 2 — 九個檔頭（AC: #2, #3）**
  - [x] 5 個 `KNI8F` → `A3p-D (LcHBs)`（`RecentlyAdded.tsx` 要先判它屬不屬於 Flow A，它是首頁也在用的元件）
  - [x] `LibraryTable.tsx` `LZ8Ds` → `A4p-D (b1H71g)`
  - [x] `LibraryFilterRail.tsx` `YEqii` → `I5-D (vpDLh)`
  - [x] `LibraryListRowV2.tsx`／`LibraryStatesV2.tsx` 的 19-8 佔位改成 `Design ref:` 形式
  - [x] `npx eslint apps/web/src/components/library/` → **0 errors**

- [x] **Task 3 — 程式碼：兩處文案（AC: #4, #5）**
  - [x] 先寫紅測試，再改 `LibraryErrorV2` 的說明
  - [x] `LibraryNoResultV2` 接篩選摘要，組出具體句子；無篩選時退回通用句
  - [x] 呼叫端 `LibraryBrowseV2` 傳入目前類型與生效篩選

- [x] **Task 4 — 設計稿：空白三態與徽章（AC: #6, #9 部分）**
  - [x] A1p-D 改成 `EmptyNoFolder` 的文案，另兩態加註記
  - [x] 徽章改不透明 `$bg-secondary` 墊底、停用打勾圖示；**位置沒改**（試過又還原，理由見 Dev Agent Record）
  - [x] ⚠️ 母版被 Flow A／B／H／L 大量 instance——改完**必須**掃一次全檔溢出與各 flow 的視覺影響
  - [x] ⚠️ 全程 Pencil MCP

- [x] **Task 5 — 結果計數（AC: #7）** ⚖️ 已裁定，直接做
  - [x] 先改測試（位置與文字）讓它紅，再把計數搬到頁首標題右側、`項` → `部`
  - [x] 確認 `data-testid="library-result-count"` 的既有斷言全綠

- [x] **Task 6 — 三個狀態母版逐一比對（AC: #8）**
  - [x] Hover／Selectable／Unmatched 各自的形狀與狀態色——**10 項差異全部列出**
  - [x] 差異逐條記進 Dev Agent Record；**修掉 3 項、延後 7 項**（各有理由，並立案）

- [x] **Task 7 — 收尾（AC: #10, #11, #12）**
  - [x] 全套閘門
  - [x] 重跑匯出——母版改動讓 **32 個檔案**變動（Flow A 10 ＋ B/C/E/F/H/I/design-system 22），全部 stage（＋若動了母版，被影響的其他 flow 也要 stage）
  - [x] 視覺基準若紅，確認是母版改動造成的**真差異**，走 CI 的 bootstrap PR，不要本機產 `-linux.png`

## Dev Notes

### 這張的重點不在長相，在兩句話和七個壞掉的指標

- **AC #2 是這張最有價值的一條。** 七個檔頭指向三個**已經不存在**的節點。dsr-7 才剛在 `sAaCR` 上證明過：一個看起來正確、其實指向虛空的標頭，比沒有標頭更危險——它讓下一個人以為對齊過了。
- **AC #4 是最有感的一條。** 「媒體庫載不出來」時，使用者第一個念頭是「我的檔案還在嗎」。稿回答了（`你的檔案沒有受影響`），碼只說「請稍後再試」。

### sprint-status 條目已過期的兩處

- 「① `PosterCardV2.tsx` 仍掛 `shadow-md`」——**已被 dsr-9 修掉**，現在全檔只剩一句註解提到它。`library/` 剩下的陰影都是 `shadow-[var(--shadow-xl)]`，掛在下拉選單與對話框等真浮層上，合法。
- 「② `PosterCardV2` 3 處、`LibraryListRowV2` 5 處 `text-[1Xpx]`」——**⚖️ 2026-09-16 已裁定 11px 是全站微標籤標準**（44 處在用），這條作廢。見 `disc-2026-09-11px-micro-label-not-on-type-scale`。

### 不要做的事

- **不要動 `components/library/` 裡屬於 Flow C 的檔案。** `FilterPanel`／`EmptySearchResults` → `C1-D (rsAxf)`、`BatchConfirmDialog`／`BatchProgress`／`SelectionToolbar` → `C2-D (dcf67)`、`SettingsGearDropdown` → `C3-D (7fE0b)`。它們住在 `library/` 但畫的是 Flow C 的畫面，屬 **dsr-3**。（三個節點都還存在，只是檔頭寫的是舊畫面名，一併留給 dsr-3。）
- **不要動手機的四張稿與 sheet** —— 那是 `dsr-1b`。
- **不要把空白三態合併成一態**（AC #6）。
- **不要動 `text-[11px]`**（AC #13 的判例）。
- **不要碰 `components/media/`** —— dsr-2 的範圍，即使 `MediaGrid.tsx` 也指向已刪的 `KNI8F`（已立案，見 Discovery Triage）。

### Source tree

```
apps/web/src/components/library/LibraryGrid.tsx            ← Task 2
apps/web/src/components/library/LibrarySearchBar.tsx       ← Task 2
apps/web/src/components/library/ViewToggle.tsx             ← Task 2
apps/web/src/components/library/ParseFailureCard.tsx       ← Task 2
apps/web/src/components/library/RecentlyAdded.tsx          ← Task 2
apps/web/src/components/library/LibraryTable.tsx           ← Task 2
apps/web/src/components/library/LibraryFilterRail.tsx      ← Task 2
apps/web/src/components/library/LibraryListRowV2.tsx       ← Task 2
apps/web/src/components/library/LibraryStatesV2.tsx        ← Task 2, 3（主要）
apps/web/src/components/library/LibraryBrowseV2.tsx        ← Task 3（呼叫端）、Task 5
apps/web/src/components/library/PosterCardV2.tsx           ← Task 6（只讀比對，改動看 AC #9 判定）
ux-design.pen（LcHBs/b1H71g/vZpT8/EsoIv/R3FqJc/dVGIa/hD7Tw/L2ynz/fpKEv/n6Crb）← Task 4,6
scripts/export-pen-screenshots.py                          ← Task 1
```

### Time-dependent visual coverage

- **Does this story add/modify any `apps/web/src/components/**/*.{ts,tsx}` that reads the wall clock?**
  - **間接是。** `RecentlyAdded.tsx` 的檔頭自己標著 `Clock-mocked`（gallery 夾具 `library-recently-added` 用 `page.clock.setFixed`），`LibraryBrowseV2.tsx` 標著 `Time-bomb-exempt`（唯一的時鐘讀取是批次匯出的檔名時間戳）。
  - **本 story 對這兩個檔只改檔頭註解，不動任何時間邏輯，也不新增視覺夾具。** 兩個既有標記維持原樣。
- Reference: `project-context.md` Rule 23。

### References

- [Source: `ux-design.pen` A1p-D (vZpT8) / A2p-D (EsoIv) / A3p-D (LcHBs) / A4p-D (b1H71g) / A7p-D (R3FqJc) / A8p-D (dVGIa) / Component/PosterCard-v2 (hD7Tw) ＋三個狀態母版] — 以 Pencil MCP `Get(..., {resolveInstances:true})` 讀出
- [Source: Pencil MCP 節點存在性驗證] — `KNI8F`／`LZ8Ds`／`YEqii` 三個節點在全文件搜尋中**查無**，AC #2 的事實基礎
- [Source: `apps/web/src/components/library/LibraryStatesV2.tsx:56-81`] — `LibraryErrorV2`／`LibraryNoResultV2` 現有文案，AC #4/#5 的事實基礎
- [Source: `apps/web/src/components/library/LibraryBrowseV2.tsx:588-592`] — 結果計數的位置與單位，AC #7 的事實基礎
- [Source: `apps/web/src/components/library/PosterCardV2.tsx:144-155`] — 徽章形狀，AC #9 的事實基礎
- [Source: project-context.md#Rule 21（19-8 之後不得再有 pending 佔位）/ #Rule 23 / #Rule 24]
- [Source: `sprint-status.yaml` → `epic-dsr` / `dsr-1-flow-a-browse-v2` / `dsr-5`（E4 耦合）/ `disc-2026-09-postercard-v2-badge-shape-drift` / `disc-2026-09-11px-micro-label-not-on-type-scale`]

## Dev Agent Record

### Agent Model Used

Claude Opus 5 (1M context) — `claude-opus-5[1m]`，BMAD dev agent（Amelia）2026-09-16

### Debug Log References

- `pnpm nx test web`（GREEN）：**3403 / 3403 passed**（+2 新測試）
- `pnpm nx test api`：PASS · `pnpm run lint:all`：**0 errors**、128 warnings · `typecheck`：PASS · `prettier`：PASS · `check-design-tokens.py`：一致
- 匯出：192 張，**32 個檔案變動**（Flow A 10、design-system 4、Flow B 2、C 2、E 4、F 1、H 3、I 6）——母版改動的預期擴散
- ⚠️ **一次自找的假警報**：我用 `npx vitest run --root apps/web` 跑，`eslint-rules/*.spec.ts` 16 紅。那三支 spec 會動態 import repo 根目錄的 `eslint.config.mjs`，而 `--root apps/web` 讓 vite 拒絕載入 root 之外的檔案（`Failed to load url … Does the file exist?`）。**用 `nx test web` 全綠**。`git stash` 對照確認與改動無關。教訓：這個 repo 的測試要用 `nx test web`，不要自己換 root。

### Completion Notes List

- 🔗 **AC Drift: NONE**（檢查 `grep -rn "library-result-count\|LibraryNoResultV2\|LibraryErrorV2" _bmad-output/implementation-artifacts/*.md` —— 除了本單與 `ux2-2-browse-v2`（定義這三個元件的原始單子，其 AC 只規定「三態要存在且各自可辨識」，未凍結文案）之外無命中，本次改動不牴觸任何已出貨 AC。計數從工具列搬到頁首確實改變了「選取模式下計數消失」這個既有行為，該行為**不是任何 AC 指定的**，只是版位的副作用；已在測試裡翻轉並註明。）
- 📎 **Contract Stamps: NONE**（本 story 與 `ux2-2-browse-v2` 都沒有 `[@contract-v*]` 標記——Rule 20 之前的單子，屬 implicit v0）
- 🎭 **A11y Pre-Flight: PASS**（`npx eslint apps/web/src/components/library/` → 0 errors、1 warning（既有）。新增的 `<h1>` 是本頁第一個標題，補上了原本**完全缺少的頁面標題**——這本身是無障礙改善。四類回歸項：沒有新增圖片、沒有 aria-modal、沒有非同步揭露內容、沒有自訂 widget。）

#### Task 6 —— 三個狀態母版 vs 程式碼，10 項差異

| # | 元素 | 設計稿母版 | 程式碼 | 處置 |
| --- | --- | --- | --- | --- |
| 1 | `play-btn`（Hover 中央 48×48） | **有** | **全 app 沒有任何可達的播放按鈕**（`LocalDetailV2.tsx:12` 明寫沒有；唯一一顆在 `MediaDetailPanel.tsx:264`，那是無掛載點的死碼） | ✅ **刪除** |
| 2 | 徽章打勾圖示 | `check` 圖示 | 徽章**沒有任何圖示** | ✅ **停用**（不是刪除——dsr-7 在 Flow H 的 instance 上留有 `niwYf` 的 fill override，刪節點會留下懸空引用） |
| 3 | 徽章底色 | `$overlay-scrim`（半透明） | **不透明 `--bg-secondary` 墊底**再疊 tint | ✅ **改稿**（不透明墊底是 critique R1 P0 特意加的：12% tint 直接壓在任意海報上量到 1.58:1） |
| 4 | 徽章／勾選框**位置** | 徽章左上、勾選框右上 | 徽章**右上**、勾選框**左上**（完全互換） | ⏸️ **試過，還原了**。`poster-image` 是 `layout:"none"` ＋ `width:"fill_container"`，絕對定位的子節點**只能貼左邊**——改成貼右邊之後，在比母版（220px）窄的 instance 上（6 欄網格）徽章直接被切出卡片外，匯出圖上看得見。設計稿當初放左上就是因為這個限制。要真的對齊得改母版的排版模型或改程式碼，兩者都超出本單，已立案。 |
| 5 | `hover-scrim`（整片遮罩） | 有 | **沒有遮罩**，改用 `group-hover/card:scale-[var(--motion-lift)]` 放大 | ⏸️ 動態詞彙決定（`--motion-lift` 是 feat/motion 的既有語彙），不是對齊工作 |
| 6 | `rating-badge` 位置 | 左下 (8,206) | `bottom-1.5 left-1.5` 左下 | ✅ 一致 |
| 7 | `menu-btn` | Hover／Selectable 右下 | 存在，但由 `LibraryGrid.tsx` 掛載（不在卡片元件內） | ✅ 功能一致 |
| 8 | Unmatched 海報 fallback | `$bg-tertiary` ＋ `file-search` 圖示 ＋「無中繼資料」 | hash 漸層 ＋ 片名首字（`filenameToGradient`） | ⏸️ 差異大，是版面決定 |
| 9 | Unmatched 狀態晶片 | `$warning-tint` ＋ `triangle-alert` ＋「未比對」 | 走 `pickPosterBadge`，字是「未**匹配**」 | ⏸️ 已立案 `disc-2026-09-unmatched-two-words`（dsr-5 立，待文案裁定） |
| 10 | 勾選框描邊 | Selectable 有 `$text-on-scrim` 描邊 | `border-[var(--text-on-accent)]`（未選）／`border-[var(--accent-primary)]`（已選） | ⏸️ 併入第 4 項一起處理 |

**AC #8／#9 因此只完成一部分**：10 項裡修了 3 項（1、2、3），7 項延後並各自記了理由與立案。這是刻意的——第 4／5／8 項都要動版面或動態語彙，而且會改到 `poster-card` 的視覺基準，屬於「設計決定」而非「把碼追回稿」。

### Discovery Triage

- **Did this story discover any work outside its current scope?** **YES** —— 建單當下立三案。

- **③ backlog-with-carry-forward-link**
  - **`disc-2026-09-dangling-design-refs-outside-library`** — 除了 `library/` 那七個，另有 **三個檔案**也指向已刪除的 `KNI8F`：`components/degradation/ServiceHealthBanner.tsx`、`components/notifications/NewMediaNotifications.tsx`、`components/media/MediaGrid.tsx`。三個都不在本張範圍（前兩個沒有任何 dsr 單子涵蓋，第三個屬 dsr-2）。**建議做成一次性掃描**：把全 `components/` 的 `Design ref:` 節點 ID 丟給 Pencil MCP 驗存在性，而不是等每個 flow 各自撞到。
  - **`disc-2026-09-rule21-node-existence-unguarded`** — `local/implements-pen-node-id` 只驗**格式**（一行、結尾是 `(id)`），**不驗那個 id 在 `.pen` 裡還在不在**。`sAaCR`（dsr-7）與 `KNI8F`／`LZ8Ds`／`YEqii`（本張）合計 **12 個檔案**指向空無，全部通過 lint。要根治得有一道會讀 `.pen` 的檢查（`export-pen-screenshots.py` 已經會 dump 節點清單，可以順便輸出所有存在的 id，讓一支 test 比對）。這是 `disc-2026-09-pen-hardcoded-hex-unguarded` 的姊妹案——同樣是「設計稿那一側沒有機器在看」。
  - **`disc-2026-09-library-empty-three-states-one-frame`** — 程式碼有三個文案完全不同的空狀態（NoFolder／NoQBT／ReadyForScan），設計稿只有一張 A1p-D，而且文案是第四種寫法。本張只把 A1p-D 對到 `EmptyNoFolder` 並加註記；**另外兩態要不要各給一張稿**是 Sally 的版面決定，不是對齊工作。

  - **`disc-2026-09-postercard-master-cannot-right-anchor`** — `Component/PosterCard-v2` 的 `poster-image` 是 `layout:"none"` ＋ `width:"fill_container"`，子節點絕對定位。**貼左邊的東西在任何寬度都對，貼右邊的東西只在母版寬度（220px）對**——6 欄網格的 instance 比母版窄，貼右的節點會被切出卡片外（我實測過並還原）。所以「徽章左上 vs 程式碼右上」這個差異**不是有人畫錯，是母版的排版模型畫不出來**。要解得改母版（poster-image 換成能右對齊的排版）或改程式碼（徽章搬左上，但會撞到勾選框，也會改視覺基準）。連帶影響第 4／10 項與任何未來想貼右的裝飾。
  - **`disc-2026-09-library-has-filters-the-product-lacks`** — A3p-D 的工具列畫了 `類型／年份/**解析度**` 三個 chip，A7p-D 的無結果句子舉的例子是「（4K）」，A6p-M 的 sheet 還畫了 **解析度**（4K/1080p/720p）與 **字幕**（繁中／簡轉繁／缺字幕）兩整組。實際上：`FilterValues` 只有 `genres` / `yearMin` / `yearMax` / `unmatched`，**前端完全沒有解析度篩選**；字幕篩選的參數 `subtitleStatus` 存在於 route，但 `routes/library.tsx:20` 自己註明 **not yet wired**。這是 dsr-10「12.4 MB/s」的同一類問題——**稿上畫了一個系統做不到的控制項**。是要補功能還是把稿收回來，是產品決定。本單的 AC #5 因此只用真的存在的篩選（類型／年代／未匹配）組句子，不承諾 4K。
  - **`disc-2026-09-design-system-ghost-count-node`** — Design System 群組裡有一個 `A4JAM`（名為 `count`，在 `hLeft` 底下）是 **fully clipped**——畫面上看不到，但任何文字讀取都會撈到它。與 dsr-10 在 K4-D 抓到的「幽靈列」同一類。屬 dsr-9 的範圍，本單只記錄。

- Reference: `project-context.md` Rule 24

### File List

**修改（程式碼）：**
- `apps/web/src/components/library/LibraryStatesV2.tsx` — Rule 21 檔頭＋錯誤說明＋無結果具體化
- `apps/web/src/components/library/LibraryStatesV2.spec.tsx` — ＋2 測試、2 處斷言翻轉
- `apps/web/src/components/library/LibraryBrowseV2.tsx` — 頁首標題＋計數搬位改單位、`activeFilterLabels`
- `apps/web/src/components/library/LibraryBrowseV2.spec.tsx` — 計數斷言＋頁面標題斷言＋選取模式行為翻轉
- `apps/web/src/components/library/LibraryGrid.tsx`／`LibrarySearchBar.tsx`／`ViewToggle.tsx`／`LibraryTable.tsx`／`LibraryFilterRail.tsx`／`LibraryListRowV2.tsx` — Rule 21 檔頭
- `apps/web/src/components/library/ParseFailureCard.tsx`／`RecentlyAdded.tsx` — Rule 21 檔頭改成「無對應畫面」變體（兩者都沒有掛載點）

**修改（設計與腳本）：**
- `ux-design.pen` — A1p-D 對齊 `EmptyNoFolder` ＋三態註記；`Component/PosterCard-v2` 母版三個狀態：刪 `play-btn`、停用徽章打勾、徽章改不透明 `$bg-secondary` 底
- `scripts/export-pen-screenshots.py` — `READABLE_FLOWS` ＋ `flow-a-browse-v2`
- `_bmad-output/pen-tokens.json`、`_bmad-output/screenshots/**`（32 個檔案）
- `_bmad-output/implementation-artifacts/sprint-status.yaml` — 狀態流轉＋dsr-1b 拆出＋6 筆立案

## 對抗式 Code Review（/ship，2026-09-16）

獨立 reviewer（fresh context）回報 **14 項**。**修掉 12 項、2 項記錄成裁定項**。

### 🔴 最難堪的三項：這個 commit 犯了它自己在罵的錯

| # | 問題 | 處置 |
| --- | --- | --- |
| 1 | **我在兩個檔頭裡引用了一張根本沒立的單子**（`disc-2026-09-unmounted-components`）。這張 story 的 Dev Notes 白紙黑字寫著「一個看起來正確、其實指向虛空的標頭，比沒有標頭更危險」——然後我在同一個 commit 裡製造了兩個。 | 立案 `disc-2026-09-unmounted-v1-components` 並把所有引用指向它；**收尾前加了一道自檢**：把 `library/*.tsx` 裡所有 `disc-2026-09-*` 抓出來逐一確認 sprint-status 有對應條目 |
| 2 | **`LibraryBrowseV2.tsx` 自己的檔頭形式是錯的**——寫 `Implements: Component/Browse-Grid-v2 (LcHBs)`，但 `LcHBs` 是**畫面** A3p-D，`.pen` 裡根本沒有叫 `Browse-Grid-v2` 的元件。它通過 lint 是因為規則只檢查形狀。而這是本單改動最多的檔案。 | 改成 `Design ref: … Screen A3p-D (LcHBs) · A4p-D (b1H71g)`。「修了 9 個檔頭」應該說「11 個裡修了 10 個」（`LibraryFilterSheetV2` 同類問題留給 dsr-1b） |
| 3 | **`LibraryGrid`／`LibraryTable`／`LibrarySearchBar` 也沒有掛載點**，我卻把它們指向**活的** A3p-D／A4p-D——等於把「壞掉的指標」換成「假的指標」：現在問「誰實作 A3p-D」會答出三個死的 v1 元件。 | 三個都改成誠實的「沒有對應畫面」變體。`ViewToggle` 是五個裡唯一真的有掛載的，保留畫面指向 |

### 🔴 其他修掉的

| # | 問題 | 處置 |
| --- | --- | --- |
| 4 | **安慰的那句話被漆成警報紅。** 「你的檔案沒有受影響」用 `--error-text`，稿上是中性灰。用紅色說「別緊張」是在跟自己打架。 | 改 `--text-secondary` |
| 5 | 錯誤碼內嵌在句子裡 → 「…沒有受影響。（DB_QUERY_FAILED）」，句號後面掛一個括號。稿上是獨立一行的等寬膠囊。 | 改成獨立膠囊 |
| 6 | **半開年份標籤是胡說的**：`yearMin=2010` 與 `yearMax=2010` 都渲染成「2010 年」——一個字串兩個相反意思。而且兩端都給時我寫 `2010–2019`（沒有「年」），`FilterChips` 同一畫面上方寫的是 `2010–2019 年`。我的註解還宣稱「不可能跟徽章打架」——那對數字成立，對用字不成立。 | 抽出共用的 `yearFilterLabel()`，`FilterChips` 與句子都吃它 |
| 7 | **「部」的裁定被同一畫面上的「項目」打臉**：頁首「媒體庫 0 部」，四行下面「沒有**項目**符合…」。裁定的理由正是「一個產品不該有兩套說法」。 | `TYPE_NOUN.all` 改「內容」，與頁首同一量詞。**但裁定本身只有電影那一張稿有證據**，已立案 |
| 8 | 「全部／電影／影集」在程式碼裡有**四份**定義、對 `all` 有兩個答案，而我新增的兩份註解都宣稱「跟側欄與篩選軌同一組」——對篩選軌是假的（它說「全部」）。 | 拿掉不實的句子並立案 |
| 9 | **新的 `<h1>` 不在稿放的位置**：稿畫在頂欄、跟 omnisearch 同一列、釘住不動；我放在內容欄，會跟著捲走。 | 註解記下差異並立案（要動 `AppShellV2` 的標題插槽，超出本單） |
| 10 | **標題階層變成倒的**：篩選軌的 `h3` 渲染在新的 `h1` 之前，所以文件第一個標題是 h3。而且 repo 裡**沒有任何 axe 檢查**會抓到。 | `h1` 移到篩選軌之前；兩個區塊標題 `h3` → `h2` |
| 11 | **新增的程式碼沒有任何測試碰得到**：唯一穿過 `LibraryBrowseV2` 到 `LibraryNoResultV2` 的測試只斷言「存在」，改動前就會綠；葉子元件的測試餵的是手寫標籤陣列，抓不到生產端的 bug（第 6 項就是生產端的）。 | 補 3 條**穿過容器**的測試：完整句子、半開年份、計數在載入／錯誤時缺席 |
| 12 | 計數新增了 `!isLoading && !isError` 的顯示條件——AC 沒提、Dev Notes 沒寫、沒有測試。（它其實是對的，對得上 A2p-D／A8p-D／A1p-D。） | 註解寫明依據，並由第 11 項的測試守住 |
| 13 | `ParseFailureCard` 的新檔頭一句話自相矛盾：說「mounted NOWHERE」又說「only routes/test/manual-search.tsx」——後者是 route，而且有 e2e 在跑。 | 改寫成「只掛在 dev-only route 與視覺夾具，沒有生產掛載點」 |
| 14 | `READABLE_FLOWS` 的每一筆都有日期與理由註解，我加的那筆沒有。 | 補上 |

### 📌 reviewer 的兩個提醒（已納入）

- **本次 diff 只是改動的一半**：`ux-design.pen` 與 32 張 PNG 不在 diff 裡，AC #1/#6/#8/#9 全在那一側，需要人看圖。
- **AC #12 的預測不會從程式碼側成真**：六個對應的視覺基準都是純註解改動；`LibraryStatesV2` 與 `LibraryBrowseV2` **根本沒有夾具**，所以兩處文案改寫與新頁首**零像素覆蓋**。`-linux` bootstrap 只可能來自 `.pen` 母版改動。

## Change Log

| 日期 | 內容 |
| --- | --- |
| 2026-09-16 | ✅ 收單 —— PR #442 合併進 main（commit 2adc3974），**CI 17 項一次全綠**，視覺基準零變動。 |
| 2026-09-16 | 🔴 **對抗式 CR 回報 14 項，修 12 記 2**。最難堪的是前三項——這個 commit 犯了它自己在罵的錯：我在檔頭引用了一張**沒立的單子**，還把三個同樣沒有掛載點的檔案指向**活的**畫面（把壞指標換成假指標）。另外：安慰的句子被漆成警報紅、半開年份標籤一個字串兩個意思、「部」的裁定被同畫面的「項目」打臉、標題階層變成 h3→h1 倒序、以及**新增的程式碼原本沒有任何測試碰得到**（補了 3 條穿過容器的測試）。收尾前加了一道自檢：所有 `disc-2026-09-*` 引用都必須在 sprint-status 找得到條目。 |
| 2026-09-16 | ✅ Task 7 閘門全綠：web **3403/3403**、api PASS、lint 0 errors、typecheck PASS、prettier PASS、token 一致。匯出 32 個檔案變動（母版改動的預期擴散）。 |
| 2026-09-16 | Task 4／6 — 三個狀態母版與程式碼比出 **10 項差異，修 3 延 7**。刪掉 Hover 的 `play-btn`（全 app 沒有任何可達的播放按鈕）、停用徽章打勾（程式碼的徽章沒有圖示，而打勾在「整理中」上等於說「做完了」）、徽章底改不透明。**徽章位置試過改成右上又還原**：母版的 `poster-image` 是絕對定位排版，貼右的節點在比母版窄的 instance 上會被切出卡片——設計稿放左上是因為這個限制，不是畫錯。A1p-D 的文案對齊 `EmptyNoFolder` 並加上三態註記。 |
| 2026-09-16 | Task 5 — ⚖️ 計數搬到頁首、`項` → `部`。順帶發現**這頁原本根本沒有頁面標題**（`<h1>` 是新加的，媒體庫／電影／影集三個字沿用側欄與篩選軌的同一組）。副作用：計數現在**在選取模式下不會消失**了（以前它跟工具列一起被換掉），測試已翻轉並註明。 |
| 2026-09-16 | Task 3 — 錯誤畫面改說「媒體庫資料查詢失敗，**你的檔案沒有受影響**」（原本只有「請稍後再試」）；無結果改說「沒有**電影**符合目前的篩選條件（**動畫、2010s**）」，篩選標籤與 rail 的計數讀同一個 `filters` 物件，句子不可能跟徽章打架。無篩選時退回通用句。 |
| 2026-09-16 | Task 2 — **9 個檔頭**。7 個指向三個已刪除的節點（`KNI8F` ×5、`LZ8Ds`、`YEqii`），2 個還掛著 19-8 的 `pending` 佔位（Rule 21 明文說 19-8 之後不該再有）。其中 `ParseFailureCard` 與 `RecentlyAdded` 查證後**沒有任何掛載點**，改成「無對應畫面」的誠實變體而不是硬指一張稿。 |
| 2026-09-16 | Story 建立（SM Bob, create-story）。Flow A 十張稿以 Pencil MCP 逐節點讀出，與 `components/library/` 24 個元件比對。**拆成 dsr-1（桌機）／dsr-1b（手機）**：24 個元件、主要 10 檔 2,066 行、10 張稿，單張過大。sprint-status 條目的兩處過期描述已在 Dev Notes 更正（`shadow-md` 已被 dsr-9 修掉；`text-[1Xpx]` 已被 2026-09-16 的 11px 裁定作廢）。 |

## 裁定紀錄

### ⚖️ 2026-09-16 · 結果計數：搬到標題旁，單位用「部」

稿放在頁首標題右邊寫「1,284 部」，碼放在工具列靠右寫「1,284 項」。

**Alexyu 選：照稿。**「電影 1,284 部」讀起來是一句話；而且首頁讀數帶已經在用「3 部」「2 部失敗」，媒體庫用「項」會變成同一個產品兩套說法。

落點：AC #7 / Task 5。
