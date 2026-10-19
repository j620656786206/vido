# Disc：修字幕清單 B2（前端・頁首與工具列）——標題旁寫「37 部缺中文」一點就篩，排序多一個「字幕狀態」

Status: ready-for-dev

**Blocked-by:** `disc-2026-10-library-list-triage-a`（`sort_by=subtitle_status`）、`disc-2026-10-library-list-triage-b1`（列表骨架照 B1 的表格畫；兩張都改 `LibraryBrowseV2.tsx`，B1 先合）。

**Source:** 2026-10-19 `/impeccable critique 媒體庫 列表`（Persona「NAS self-hoster: can't answer how many of 136 still lack 繁中」「Alex: no 缺中文-first sort」；P3：跳到結果連結、ViewToggle 34px、列表載入卻顯示格狀骨架、font-mono 沒吃到 JetBrains Mono）＋ sprint-status「另查：網址 pageSize=20 卻一次 36 列」。⚖️ **1A**（修字幕清單）、⚖️ **2B**（critique 全部處理）。設計：A4p-D `b1H71g`、A4p-M `EkB8g`、`spec-note-a4p`；Sally 裁定 5（「N 部缺中文」泥金連結、點了＝套用缺中文字幕篩選、0 部不顯示）、裁定 6（第五個排序，桌機下拉與手機抽屜同一份）。

⚖️ **SM 裁定（Bob）：** 見 `…-a` 的拆單說明。B2＝列表框**外面**的東西：頁首、工具列、骨架、網址。

## 查到的事（2026-10-19，本 session 讀過的程式碼）

- **「N 部缺中文」不用新 API。** 手機抽屜的「套用篩選 · N 部」就是用 `page_size=1` 問一次、讀 `totalItems`：`sheetCountParams`（`LibraryFilterSheetV2.tsx:44-62`，已 export）＋ `useLibraryList`（`hooks/useLibrary.ts:19-31`；抽屜在 `LibraryFilterSheetV2.tsx:121-122`）。`chinese_subtitle=missing` 篩的是判定 `none`（`apps/api/internal/models/chinese_subtitle.go:71-72`），前端註解寫明它列出的是徽章「缺中文（或 無字幕源／已略過／未翻譯）」的片（`chineseSubtitleFilter.ts:9-14`）。
  - 取捨：重用＝零後端、跟「點了套用的篩選」**定義上一定相同**；代價是每次換篩選多一個 `page_size=1` 請求（資料庫跑一次 COUNT＋一列）。新 endpoint（例如 stats 多一個 `missing_chinese_count`）要另寫一份規則、還要記得失效快取——規則一分叉，數字就會跟篩完的結果對不上。**選重用。**
- **頁首計數**：`library-result-count` 顯示 `{totalItems} 部`（`LibraryBrowseV2.tsx:584-591`），載入中／錯誤時不顯示；E2E 用 `toHaveText('2 部')` 精確比對（`tests/e2e/library-mobile.spec.ts:192`、`:199`、`:211`、`:438`）→ **新連結不能塞進這個 span**。
- **排序選項只有一份**：`SORT_OPTIONS`（`SortSelector.tsx:18-23`），手機抽屜 import 同一份（`LibraryFilterSheetV2.tsx:28`、`:198-213`，testid `library-sort-${field}`）。型別與白名單：`SortField`／`VALID_SORT_FIELDS`（`types/library.ts:3-11`）；`getStoredSort` 用白名單驗 localStorage（`LibraryBrowseV2.tsx:99-109`）——沒加進白名單的話，重新整理就掉回「新增日期」。
- **後端不認得的 `sort_by` 不會 400**，會默默照 `created_at` 排（`apps/api/internal/repository/movie_repository.go:348`）→ A 沒上線前選「字幕狀態」看起來不會壞、只是順序不對；所以 B2 必須在 A 之後。
- **檢視切換 34px**：按鈕 `rounded p-2`＋18px 圖示（`ViewToggle.tsx:32`、`:48`、`:38`、`:54`）。
- **列表載入時顯示格狀骨架**：兩處都寫死 `LibraryGridSkeletonV2`（`LibraryBrowseV2.tsx:779`、`:796`）。
- **沒有跳到結果的連結**：頁首 → 篩選欄（桌機一長串 chip）→ 工具列 → 才到第一張卡／第一列（critique：約 40 次 Tab）。
- **pageSize=20 vs 36 列（找到原因）**：`/library` 的 `validateSearch` 把 `page` 預設 1、`pageSize` 預設 20（`routes/library.tsx:37-38`），但**沒有任何程式讀它**（`grep -rn "search\.page\b\|search\.pageSize" apps/web/src` 只命中 discover／downloads／search 自己的路由）；真正的每頁筆數是無限捲動 hook 寫死的 `PAGE_SIZE = 36`（`hooks/useLibraryInfinite.ts:12`、`:20`）。第一次 `navigate({ search: prev => … })`（例如切到列表）就把這兩個預設值寫進網址。是舊 LibraryPage 留下來的死參數，**資料沒有錯**——36 是刻意的。
- **font-mono 沒吃到 JetBrains Mono（查完了，不在本單修）**：`styles.css:547-549` 在 `@layer base` 寫 `.font-mono { font-family: 'JetBrains Mono', … }`，但 Tailwind v4 的 `font-mono` 工具類在 `@layer utilities`、用預設 `--font-mono: ui-monospace, SFMono-Regular, Menlo, …`（`node_modules/tailwindcss/index.css:1`、`:9-11`），layer 順序 utilities 勝 base → base 那條永遠輸。而且整個 app 沒有載入 JetBrains Mono（`index.html` 沒有字型連結、`styles.css` 沒有 `@font-face`、`package.json` 沒有 fontsource），就算 cascade 對了也只有本機剛好裝了字型才看得到。`font-mono` 全站 72 個檔、180 處 → 改了會動到大半的視覺基準 → 另開單（見 Discovery Triage），需要 Alexyu 選做法。
- **測試現況**：`LibraryBrowseV2.spec.tsx:45-50` 把 `useLibraryList` mock 成 `data: undefined`；`SortSelector.spec.tsx:40-43` 逐一檢查四個選項；`LibraryFilterSheetV2.spec.tsx:53-57` 測試名稱寫「four SORT_OPTIONS」但斷言是 `SORT_OPTIONS.map(...)`（會自動跟著變，名稱要改）。E2E stub 的 `pick` 對 `page_size=1` 回 `totalItems`（`library-mobile.spec.ts:36-46`），新請求會被正確回應。視覺基準受影響：`library-view-toggle`（default／hover／focus）、`library-sort-selector`（open）、`library-mobile-sheets/sort-filter`（多一個 chip）——各有 `-darwin.png`＋`-linux.png`（`tests/visual/components.visual.spec.ts-snapshots/components/…`）。

## Story

身為只有我一個人在用的 NAS 片庫主人，
我想要一打開媒體庫就看到「還有幾部缺中文」、一點就只看那些，而且能依字幕狀態排序，
這樣我不用自己數、也不用一頁一頁翻就知道還剩多少要處理。

## Acceptance Criteria

1. **第五個排序。** confirmed against [@contract-v1] (Story disc-2026-10-library-list-triage-a AC #1)。`SortField` 與 `VALID_SORT_FIELDS` 加 `'subtitle_status'`；`SORT_OPTIONS` 第 5 項 `{ field: 'subtitle_status', label: '字幕狀態', defaultOrder: 'asc' }`。桌機下拉與手機抽屜都是同一份 5 個、同樣的字；選了送出 `sort_by=subtitle_status&sort_order=asc`；再點一次變 `desc`；重新整理後仍是「字幕狀態」（localStorage 白名單）。預設排序不變（新增日期）。
2. **排序順序的前端對表測試。** vitest 列出跟 A 的 Go 表格同一組（判定 × status）組合，斷言 `deriveSubtitleStatus` 的字（或 `null`→「不知道」）依 A 的排名排起來正好是「缺中文 → 無字幕源／未翻譯／已略過 → 不知道 → 簡中 → 中文 → 繁中」。檔頭註明對應的 Go 測試檔與函式名稱（`models.SubtitleTriageRank`）。
3. **「N 部缺中文」連結。** 頁首計數右邊、`library-result-count` 之外另一個元素：「· 37 部缺中文」（`·` 為裝飾、`aria-hidden`），字型同計數（`font-mono` 12 tabular），顏色 `--accent-text`；testid `library-missing-zh-link`；可存取名稱「37 部缺中文，只看缺中文字幕的片」。
   - N＝`useLibraryList(sheetCountParams({ ...目前篩選, chineseSubtitle: ['missing'] }, 類型, 排序, 方向)).data.totalItems`（類型、類別、年份、未匹配都照目前的）。
   - 點了＝`chineseSubtitle: 'missing'`，其他篩選保留；之後連結消失、篩選 chip「缺中文字幕」出現、計數變 N 部。
   - **不顯示**：N＝0、載入中、錯誤、目前已經有任何「字幕」篩選（這時連結沒有工作可做）。
   - `library-result-count` 的文字維持精確的「N 部」（E2E 不改仍綠）。
   - **390×844（A4p-M）**：標題、計數、連結在同一列，右邊 44×44 的篩選鈕不被擠出或縮小（量 bounding box：按鈕寬高 44、右緣 ≤ 374）。
4. **跳到結果。** 頁面內容最前面一個「跳到結果」連結，平常看不到、拿到焦點才出現（左上，`--bg-secondary` 底、焦點框可見）；按下後焦點移到結果區（格狀或列表的容器，`tabIndex={-1}`、`aria-label="媒體庫結果"`），再按一次 Tab 就到第一張卡／第一列。不能用 `href="#…"` 讓 router 換網址——用 `onClick` `preventDefault` 後 `focus()`。
5. **檢視切換 44×44。** 兩顆按鈕都 `size-11`（44px），圖示維持 18；樣式、提示框、`radiogroup` 不變。
6. **列表骨架。** 列表檢視載入中（`LibraryBrowseV2.tsx:779` 與 `:796` 兩處）顯示新的 `LibraryListSkeletonV2`：跟 B1 同一個外框、同一組 `@container` 三層、8 列灰塊（縮圖、片名、meta、徽章位），用 `animate-breathe`（reduced motion 自動停）；格狀檢視仍是格狀骨架。
7. **網址不再出現 `pageSize=20`。** 從 `/library` 的 `validateSearch` 與 `LibrarySearchParams` 拿掉 `page`／`pageSize`；切換檢視、改排序、套篩選後網址都沒有這兩個參數。每頁 36 筆不變。
8. **測試與基準圖。**
   - vitest：`SortSelector.spec`（5 個選項）、`LibraryFilterSheetV2.spec`（測試名稱改「five」）、`LibraryBrowseV2.spec`（`useLibraryList` mock 依參數回值：有篩選字幕時不顯示、N＝0 不顯示、點了 patch 出 `chineseSubtitle=missing`；跳到結果移焦點；列表檢視載入中是列表骨架）、`ViewToggle.spec`、`routes/library.spec.tsx`。
   - E2E `library-mobile.spec.ts` 加：390 連結看得到且篩選鈕 44×44 沒被擠；點連結後送出的列表請求帶 `chinese_subtitle=missing`；抽屜選「字幕狀態」送出 `sort_by=subtitle_status`。原有斷言不改仍綠。
   - 視覺：`library-view-toggle`（3 張）、`library-sort-selector`（open）、`library-mobile-sheets/sort-filter` 重產 `-darwin.png`，**`git rm` 這幾張的 `-linux.png`**（CI bootstrap PR 會補，不要自己產）；新增夾具 `library-list-skeleton`（寬 1200）與 `library-mobile-screens/list-skeleton`（390×844），只產 darwin。
9. `nx test web`、`lint:all`、`typecheck`、`format:check`、`check-design-refs.py` 綠；改過的 E2E 本機跑綠。

## Tasks / Subtasks

- [ ] **T1 排序（AC #1、#2）**：`types/library.ts`、`SortSelector.tsx` 的 `SORT_OPTIONS`；對表測試（建議放 `utils/libraryStatus.spec.ts` 或新 `subtitleTriageOrder.spec.ts`）。
- [ ] **T2 頁首連結（AC #3）**：`LibraryBrowseV2.tsx:571-592` 頁首加連結；重用 `sheetCountParams`＋`useLibraryList`（不要另寫參數組裝，抽屜註解說過鍵差一個欄位就打不到快取）。
- [ ] **T3 跳到結果（AC #4）**：連結放在 `LibraryBrowseV2` 最外層 `div` 的第一個子元素；結果容器（`library-grid-v2`／B1 的 `library-list-v2` 外層）加 `ref`、`tabIndex={-1}`、`aria-label`。
- [ ] **T4 檢視切換（AC #5）**：`ViewToggle.tsx`。
- [ ] **T5 列表骨架（AC #6）**：`LibraryStatesV2.tsx` 加 `LibraryListSkeletonV2`（重用 B1 的 grid template 常數，不要再抄一份欄寬）；檔頭 Design ref 補 `A4p-D (b1H71g) · A4p-M (EkB8g)`；`LibraryBrowseV2.tsx:779`、`:796` 依 `view` 選骨架。
- [ ] **T6 網址參數（AC #7）**：`routes/library.tsx:5-8`、`:37-38`。
- [ ] **T7 測試＋基準圖（AC #8、#9）**。

## Dev Notes

### 不要做的事

- 不要在後端加計數 endpoint（取捨見「查到的事」）。
- 不要把「· N 部缺中文」放進 `library-result-count` 那個 span（E2E 精確比對）。
- 不要改「中文」的顏色與字（⚖️ 2026-10-06 D2）；不要改 `libraryStatus.ts`。
- 不要動列表本身（B1）。
- 不要在本單修 `font-mono`（另開單，見下）。
- 不要 commit `-linux.png`；不要跑 `test:visual:update`。

### 已知語意（不是 bug）

- 「N 部缺中文」數的是篩選 `missing`（判定 `none`），所以包含徽章是「無字幕源／已略過／未翻譯」的片，也包含**正在處理中**、列表顯示「不知道」的片（處理中的徽章是 `null`，`libraryStatus.ts:178-181`）。點下去看到的就是這 N 部——數字與點了之後的清單一定一致，這是選重用的理由。
- A 的排序是依「看得到的徽章」排，跟篩選的分組不完全一樣（例：判定 `unknown`＋`skipped` 徽章是「已略過」、排在第 1 級，但篩選分組是「不知道」）。兩者各自照設計裁定，不要互相「修正」。

### 測試標準

- vitest：`useLibraryList` 的 mock 要看參數（`chineseSubtitle === 'missing'` 才回 N）；斷言具體文字與數字（Rule 16）。
- E2E：`AI_PROVIDER=claude CLAUDE_API_KEY=sk-ant-local-e2e-dummy npx playwright test tests/e2e/library-mobile.spec.ts --project=chromium --reporter=line`。
- 動到 testid／文字前先 `grep -rn "<testid>" tests/`（本 session 查過：`library-result-count` 只在 `library-mobile.spec.ts`；`sort-option-*`、`view-toggle` 在 `tests/` 0 筆）。

### Project Structure Notes

- 改：`LibraryBrowseV2.tsx`、`SortSelector.tsx`、`ViewToggle.tsx`、`LibraryStatesV2.tsx`、`types/library.ts`、`routes/library.tsx`、`routes/test/-gallery.fixtures.tsx`、對應 spec、`tests/e2e/library-mobile.spec.ts`。沒有新檔（除了可能的對表測試檔）。

### Time-dependent visual coverage

- N/A — no wall-clock-reading components touched（`LibraryBrowseV2` 已有的時間讀取是 `Time-bomb-exempt` 的匯出檔名，本單不新增；骨架與連結不讀時間）。

### References

- [Source: _bmad-output/implementation-artifacts/disc-2026-10-library-list-triage-design.md — Sally 裁定 5、6]
- [Source: _bmad-output/implementation-artifacts/disc-2026-10-library-list-triage-a.md AC #1 [@contract-v1]]
- [Source: 篩選分組 disc-2026-10-subtitle-filter-disagrees-with-badges AC #2 [@contract-v1]——confirmed against [@contract-v1] (Story disc-2026-10-subtitle-filter-disagrees-with-badges AC #2)：`chinese_subtitle=has|missing|unknown`，`apps/api/internal/handlers/library_handler.go:151-180`]
- [Source: apps/web/src/components/library/LibraryBrowseV2.tsx:99-109, 556-621, 680-772, 776-842]
- [Source: apps/web/src/components/library/LibraryFilterSheetV2.tsx:28, 44-62, 121-122, 198-213]
- [Source: apps/web/src/hooks/useLibraryInfinite.ts:12-20]、[routes/library.tsx:5-38]
- [Source: apps/web/src/styles.css:547-549]、[node_modules/tailwindcss/index.css:1, 9-11]

## Dev Agent Record

### Agent Model Used

### Debug Log References

### Completion Notes List

### Discovery Triage

- **① expand-scope-in-place** — 網址 `pageSize=20` 卻一次 36 列：查到是 `/library` 路由的死參數（`routes/library.tsx:37-38`），真正的每頁筆數是 `useLibraryInfinite.ts:12` 的 36 → AC #7 拿掉死參數。
- **③ backlog-with-carry-forward-link** — `disc-2026-10-font-mono-not-jetbrains`：`font-mono` 在 Tailwind v4 被預設 `ui-monospace` 蓋掉，而且全站沒載入 JetBrains Mono（見「查到的事」）；`DESIGN.md:337` 寫的是 JetBrains Mono。影響 72 檔／180 處、幾乎所有含數字的基準圖 → 需要 Alexyu 選：(a) 自帶 JetBrains Mono woff2（拉丁子集）＋ `@theme { --font-mono: … }`，全站等寬字變成設計稿那款，大量 `-linux` 基準要 CI 重補；(b) 只改 cascade 不載字型——Mac 會從 SF Mono 變成 Monaco，更糟；(c) 改 DESIGN.md 承認用系統等寬字、刪掉 base 那條死規則。Bob 建議 (a)。

### File List
