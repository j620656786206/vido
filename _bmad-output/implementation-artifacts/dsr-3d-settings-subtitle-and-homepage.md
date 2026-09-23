# Story DSR.3d：字幕設定與自訂首頁對齊設計稿——選項卡片看得出選了哪個，首頁區塊列表不再畫產品沒有的開關與拖曳

Status: ready-for-dev

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As the person who tunes 字幕的在地化程度 and 首頁要出現哪些區塊,
I want 三個在地化選項是三張看得出選中哪張的卡、首頁區塊每一列一眼看出是電影還是影集、排什麼、幾部，而且稿上畫的每個控制項都真的存在,
so that 我改設定時知道自己改了什麼，也不會去找一個不存在的「關掉這個區塊」開關。

## Context

`dsr-3` 拆出來的第四張：**「媒體庫」分組裡的字幕設定、自訂首頁**（媒體庫掃描是 Flow E，`dsr-5` 已對齊，不在這裡）。**依賴 `dsr-3a` 先合併**。與其他子單互不相依。拆單理由見 `dsr-3a` Context。

| 稿 | 節點 | 元件 |
| --- | --- | --- |
| C9-D／C9-M 字幕設定 | `NR3zK`／`AWYm0` | `routes/settings/subtitle.tsx`、`LocalizationLevelForm.tsx` |
| C10-D／C10-M 自訂首頁 | `wnmGh`／`ZjsVs` | `routes/settings/homepage.tsx`、`ExploreBlocksSettings.tsx`（＋`ExploreBlockEditModal.tsx` 只改檔頭） |

### 🔴 建單時查到的事（main `b041ea09`；行號皆為現況）

**字幕設定（C9）**
1. 碼多包了一層外框：一張大卡（`bg-[var(--bg-secondary)]/50`，`LocalizationLevelForm.tsx:85`），裡面有小標「AI 字幕的在地化程度」與引言「這是口味，不是對錯…」，再放三個選項。稿沒有外框，三個選項各自是卡片，底部一句說明 `rUcFk`「改了之後只影響之後生成的字幕；已經翻好的不會重跑。」。
   - ⚖️ **外框拿掉（稿→碼）；小標與引言保留（碼→稿）**——引言是這一頁唯一在說「三個選項沒有對錯」的話，拿掉會讓使用者以為有標準答案。稿補畫小標＋引言在選項上方。
   - ⚖️ **底部說明加（稿→碼），但先查證**：dev 讀 `apps/api/internal/subtitle/pipeline.go`／`process_item.go`／`segment_cache.go` 確認在地化程度是**生成當下**讀取、已完成的字幕不會被重跑。**查證為真才加**；若 `segment_cache` 會因程度改變而失效或觸發重跑 → 不加、稿刪 `rUcFk`，Completion Notes 寫查證結果（檔案:行號）。
2. 選項卡片（稿→碼）：稿 `$radius-lg`、內距 16、`$bg-secondary`＋`$border-subtle`；選中 `$accent-subtle` 底＋`$accent-primary` 框＋標題 `$accent-text`；**自繪 20px 圓形 radio**（選中時裡面打勾）。碼（`:115-137`）：`radius-sm`、`px-3 py-3`、未選中框透明、選中 `bg-tertiary`、**原生 16px radio**、標題 `font-medium text-primary`＋另外一顆綠勾。
   - 自繪 radio 要**保留原生 `<input type="radio">`**（`sr-only` 或 `appearance-none`），鍵盤上下鍵、`name` 群組、`checked` 行為全部由原生負責；外觀用相鄰元素畫。⛔ 不要改成 `div role="radio"`。
3. 說明文字：碼 `text-sm`（`:139`），稿 Label 12 `$text-secondary` → `text-xs`。範例：稿 Mono 12、沒有「例：」前綴；碼非 mono、有「例：」（`:142-143`）→ ⚖️ **稿→碼**，但把「例」的語意留給螢幕閱讀器（`<span className="sr-only">例：</span>`）。
4. 碼→稿：來自環境變數時的說明（`:94-101`）稿沒畫 → 稿補。手機稿把說明／描述／範例都縮短了（例如「「grocery store」→「超市」」）→ ⚖️ **碼→稿**（只有一套文案）；手機稿的 radio 18／圖示 10–11 → 與桌機一致（20）。

**自訂首頁（C10）**
5. **每個區塊的開關（`b8XMg` 等）產品做不到**：後端 `ExploreBlock`（`apps/api/internal/models/explore_block.go:26-38`）沒有 enabled 欄位，前端型別也沒有。連帶不成立：關掉時整列變灰（`a6kOtj`）、底部說明第一句「關掉的區塊不會出現在首頁」（`zN7oY`）。⚖️ **碼→稿：稿刪開關、刪灰列、刪第一句**；第二句「已擁有的作品不會出現」是真的（`homepage/ExploreBlock.tsx:72` 有過濾）→ 保留並**加到碼**（稿→碼）。立 `disc-2026-09-explore-block-toggle-and-drag`。
6. **拖曳排序產品沒有**：稿畫拖曳把手 `kvoQ8`＋「拖曳可調整順序」`u5ByTQ`；碼是上移／下移按鈕（`ExploreBlocksSettings.tsx:150-169`）。⚖️ **碼→稿**：稿換成上移／下移，刪把手與那句話。拖曳併入同一張 disc。
7. 標頭左側「6 個區塊」計數：碼沒有（只有新增按鈕靠右，`:74`）→ 稿→碼（`{blocks.length} 個區塊`，`text-sm text-secondary`）。
8. 每列的描述：稿「人氣排序 · 20 部 · zh-TW 台灣」「類型：動畫」；碼「電影 · 20 個項目 · 類型 16 · 地區 TW」（`:125-146`），**類型印原始 TMDb ID**、沒顯示排序。
   - ⚖️ **排序方式與語言：稿→碼**（`sortBy`、`language` 資料都有；排序的中文字用 `ExploreBlockEditModal` 下拉選項的同一份標籤——dev 找出來、抽成共用常數，**不要**寫第二份）。「20 部」取代「20 個項目」。
   - ⚖️ **類型名稱：碼→稿**。前端**沒有** TMDb genre ID → 中文名的來源（編輯框本身也要使用者手打 ID，`ExploreBlockEditModal.tsx:174`「類型 ID（逗號分隔 TMDb genre IDs，可留空）」）。做對照表是新功能 → 立 `disc-2026-09-explore-block-genre-ids-raw`（兩處一起解）；本張碼維持「類型 16」，稿改成碼的寫法。
   - 手機稿描述用等寬、桌機用一般字 → 稿統一一般字。
9. 類型圖示：稿是文字左邊獨立 18px film／tv（`XmC7a`）；碼是塞在描述行內的 14px 圖示＋「電影／影集」字樣 → 稿→碼（圖示提出來、描述行首的「電影／影集」保留成純文字）。
10. 區塊名稱字重：稿 600（`sytMB`），碼 `font-medium`（`:122`）→ `font-semibold`。
11. 編輯／刪除鈕：稿 32×32 實心 `$bg-tertiary`、`$radius-md`、圖示 14（`rRMJl`）；碼透明底、hover 才有底、4px 圓角、圖示 16（`:175`、`:184`）→ 稿→碼（桌機 32、手機維持 44——碼已有 `sm:` 縮小的觸控寫法，保留）。上移／下移鈕同一套樣式。
12. **手機稿每列只剩開關**（沒有編輯、刪除、排序）→ ⚖️ **碼→稿**：手機稿畫回四顆鈕（44）。
13. 間距：列距碼 `space-y-2`（8）→ 稿 12 `space-y-3`；區塊間距碼 `space-y-6`（24）→ 稿 16 `space-y-4`。
14. 新增按鈕：稿 40 高、左右 16、600、`$radius-md`（`fK1tP`）；碼約 36、`rounded-md`（6px，非 token）、500（`:79`）→ 稿→碼。
15. 列卡片圓角：碼 `rounded-lg`（8px）→ `rounded-[var(--radius-lg)]`（12）。
16. **檔頭過時**：`ExploreBlocksSettings.tsx:1` 寫 `H5-D (Y5XvRv) · H3`——`H5-D` 於 2026-09-16 由 dsr-7 更名 `H9-SPEC`（CLAUDE.md）→ 改成 `C10-D (wnmGh) · C10-M (ZjsVs) · H9-SPEC (Y5XvRv) · H3 (Paqlk)`（dev 用 `Get` 確認 `Y5XvRv` 仍是 H9-SPEC）。`ExploreBlockEditModal.tsx` 檔頭指 `H3 (Paqlk)` 正確，不動。
17. 刪除確認框（`:205-225`）與 `ExploreBlockEditModal` 是自己寫的 fixed overlay，但**都有** `role="dialog"`、`aria-modal`、Esc；陰影是 token（`shadow-[var(--shadow-xl)]`）、真的浮在頁面上 → ⚖️ **不動**（原條目的「raw shadow 要逐一判定」已判：全部合法）。

### 設計稿節點

| 代號 | 節點 | 說明 |
| --- | --- | --- |
| C9-D／C9-M | `NR3zK`／`AWYm0` | 選項卡片（dev `Get`）、`rUcFk` 底部說明 |
| C10-D／C10-M | `wnmGh`／`ZjsVs` | `b8XMg`（開關，刪）、`a6kOtj`（灰列，刪）、`zN7oY`（說明，刪第一句）、`kvoQ8`／`u5ByTQ`（拖曳，刪）、`H5rMVk`（描述）、`XmC7a`（類型圖示）、`sytMB`（名稱）、`rRMJl`（動作鈕）、`fK1tP`（新增） |

---

## Acceptance Criteria

1. **設計稿先改（Pencil MCP）**：🔴 #1（小標＋引言；`rUcFk` 視查證結果留或刪）、#4、#5、#6、#8（類型寫法、描述統一一般字）、#12。規格註記 `spec-note-dsr-3d`：
   > 「字幕設定：三個選項各自一張卡，選中＝金框＋淡金底＋打勾的圓；範例用等寬字。自訂首頁：區塊用上移／下移排序（沒有拖曳）、沒有個別開關（要隱藏就刪除）；每列顯示類型圖示、名稱、排序 · 數量 · 語言。類型目前顯示 TMDb ID（對照表另案）。」
   收尾同 `dsr-3a` AC #1（只 stage `c9-*`／`c10-*`＋`pen-tokens.json`）。

2. **字幕設定**：🔴 #1–#3。⛔ 儲存行為、選項值、env 鎖定邏輯不動。

3. **自訂首頁**：🔴 #7–#11、#13–#16，底部加一行「已擁有的作品不會出現在首頁。」（`text-xs text-muted`；稿 `zN7oY` 第二句的字，dev 逐字抄）。⛔ 新增／編輯／刪除／上下移的行為不動；⛔ 不改 `ExploreBlockEditModal` 的欄位（包含手打 ID 那格）。

4. **既有的行為不准回歸。**
   - `LocalizationLevelForm.spec.tsx`、`ExploreBlocksSettings.spec.tsx`、`ExploreBlockEditModal.spec.tsx`、`e2e/explore-blocks.spec.ts`（9 條，都開 `/settings/homepage`）的行為斷言不改。描述文字從「N 個項目」改成「N 部」若撞到既有斷言 → 屬本張刻意變更，Completion Notes 列出。
   - ⛔ 不改後端。

5. **測試。** 紅／守（Rule 16）。
   - `LocalizationLevelForm.spec.tsx`：（紅）沒有外框容器（`bg-secondary/50` 不存在）、每個選項是自己的卡片、選中那張帶 `--accent-subtle` 與 `--accent-primary` 框；原生 radio 仍存在且鍵盤方向鍵能換選（`userEvent.keyboard('{ArrowDown}')`）；說明 `text-xs`；範例 `font-mono` 且可見文字不含「例：」但 accessible text 含；底部說明依查證結果存在或不存在（寫死期望值）。（守）儲存、env 鎖定。
   - `ExploreBlocksSettings.spec.tsx`：（紅）標頭「N 個區塊」；描述含排序中文字（用共用常數比對）與「20 部」；類型圖示在描述行之外；動作鈕 `bg-[var(--bg-tertiary)]`；列距 `space-y-3`；底部那句逐字。（守）新增／編輯／刪除／上移下移。
   - 排序標籤共用常數：一條測試斷言 `ExploreBlockEditModal` 的下拉選項與列描述用的是同一個 export。
   - **視覺夾具**：既有 `settings-explore-blocks-settings` 基準**會變**、`penNode` 改 `'wnmGh'`；新增 `settings-localization-level-form`（今天沒有夾具；`width: 1200`、`penNode: 'NR3zK'`）、`settings-explore-blocks-settings/mobile`（`width: 390`、`penNode: 'ZjsVs'`）。
   - **e2e**：`explore-blocks.spec.ts` 跑綠即可（不新增）；追加到 `tests/e2e/settings-shell.spec.ts` 一條——390 開 `/settings/subtitle`，選項卡片寬 = 內容寬、按鍵盤方向鍵選中下一張。
   - **每一項修法做 mutation check**，結果寫進 Completion Notes。

6. **CI 全綠**：同 `dsr-3a` AC #9。

## Tasks / Subtasks

- [ ] **Task 1 — 查證「只影響之後生成的字幕」（讀 pipeline／segment_cache），結論寫進 Completion Notes（AC: #1, #2）**
- [ ] **Task 2 — 設計稿：C9 小標引言／env／手機文案、C10 刪開關與拖曳、類型寫法、手機補鈕、規格註記（AC: #1）**
- [ ] **Task 3 — 字幕設定：拿掉外框、選項卡片、自繪 radio（保留原生 input）、說明與範例（AC: #2, #5）**
- [ ] **Task 4 — 自訂首頁：計數、描述（排序共用常數）、類型圖示、動作鈕、間距、新增鈕、底部說明、檔頭（AC: #3, #5）**
- [ ] **Task 5 — 夾具、e2e、mutation check、收尾（AC: #4, #5, #6）**
  - [ ] dev-story Step 9：`c9-d`／`c9-m`／`c10-d`／`c10-m`

## Dev Notes

### 這張的重點

- **自訂首頁的稿大部分是錯的**：畫了開關、拖曳、類型中文名——三樣產品都沒有。先把稿改成產品的樣子，碼只做真的缺的（計數、排序字、樣式）。
- **自繪 radio 不能丟掉原生 radio**。
- **「只影響之後生成」是一個承諾**，先查證再寫。

### 上游契約（Rule 20）

- 本張不定義也不消費任何 `[@contract-v*]`。

### 建單裁定（2026-09-23，SM；Sally／Alexyu 可在 review 推翻）

1. ⚖️ 字幕：外框拿掉、小標＋引言保留、底部說明查證後才加。
2. ⚖️ 首頁：開關與拖曳碼→稿＋`disc-2026-09-explore-block-toggle-and-drag`。
3. ⚖️ 類型名稱碼→稿＋`disc-2026-09-explore-block-genre-ids-raw`。
4. ⚖️ 排序中文字抽共用常數（不寫第二份）。
5. ⚖️ 首頁的兩個自製 dialog 不動（已合法）。

### 不要做的事

- 不要加 enabled 欄位、拖曳、genre 對照表。
- 不要把 radio 改成 `div role="radio"`。
- 不要改 `ExploreBlockEditModal` 的表單欄位。

### 已知陷阱

- **自繪 radio 的焦點框**：原生 input 藏起來後，`focus-visible` 要畫在相鄰的可見元素上（`peer-focus-visible:`），不然鍵盤使用者看不到焦點。
- **`C9-D` 與 `dsr-3f` 的 `C13` 都有自繪 radio**（20 vs 18）。兩張互不相依；**後做的那張**若看到先做的已抽出元件，就重用並把尺寸統一成 20（Completion Notes 註明）。
- **Pencil／匯出／gh**：同 `dsr-3a`。

### Source tree

```
ux-design.pen、pen-tokens.json、screenshots/flow-c-search-settings/{c9,c10}-{d,m}.png        ← Task 2
apps/web/src/components/settings/LocalizationLevelForm.tsx（+spec）                         ← Task 3
apps/web/src/components/settings/ExploreBlocksSettings.tsx（+spec）、ExploreBlockEditModal.tsx（抽排序常數） ← Task 4
apps/web/src/routes/test/-gallery.fixtures.tsx；tests/e2e/settings-shell.spec.ts（追加）    ← Task 5
```

### Cross-Stack Split Check

後端 task **0**（Task 1 只讀後端）、前端／設計／測試 task 5 → 不觸發。規模與 `-3c` 同級。

### Time-dependent visual coverage

- **N/A — no wall-clock-reading components touched.**

### References

- [Source: `LocalizationLevelForm.tsx:85-143`；`ExploreBlocksSettings.tsx:1, 74-79, 122-184, 205-225`；`ExploreBlockEditModal.tsx:45-91, 174-180`；`homepage/ExploreBlock.tsx:72`]
- [Source: `apps/api/internal/models/explore_block.go:26-38`；`apps/api/internal/subtitle/pipeline.go`、`process_item.go`、`segment_cache.go`；`services/localization_settings_service.go`]
- [Source: `ux-design.pen` 節點見上表 —— SM 建單時以 Pencil MCP 讀出（2026-09-23）]
- [Source: `dsr-3a-settings-shell-page-header-and-states.md`；CLAUDE.md（H5-D → H9-SPEC 更名）；`dsr-7-flow-h-homepage-v3.md`]

## Dev Agent Record

### Agent Model Used

### Debug Log References

### Completion Notes List

- Ultimate context engine analysis completed - comprehensive developer guide created

### File List
