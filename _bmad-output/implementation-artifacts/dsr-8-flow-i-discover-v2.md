# Story DSR.8: Flow I 探索與進階搜尋——程式碼與設計稿雙向對齊

Status: review

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As the person who opens 探索 to find something new to watch,
I want 探索頁與搜尋頁在 TMDb 連不上時老實說「暫時算不出來」，錯誤代碼用跟其他頁一樣的形狀出現，設計稿指到的畫面也真的存在,
so that 我不會看到「符合 0 部」「找不到結果」這種其實是在說謊的讀數，而且設計稿能拿來當真的參考。

## Context

`epic-dsr` 的第十一張（已收 1、2、4、5、7、9、10、11、12、13）。

⚠️ **這張有一個是剛剛造成的問題。** dsr-2（#444，今天合併）讓 `services/tmdb.ts` 開始把錯誤代碼帶出來，探索頁的錯誤句因此**第一次會真的顯示代碼**——形狀是句尾括號「…請稍後再試（TMDB_TIMEOUT）」，正是 dsr-1 CR 第 5 項否決過的樣子。AC #3 修。

⚠️ **驗收基準是 `.pen` 節點值，不是 PNG。** Flow I 兩個資料夾都不在 `READABLE_FLOWS` 裡。Task 1 先修。

⚠️ **方向不是單向的，而且這次「碼對稿錯」佔多數。** 探索頁 v2 是 ux3-3-2 照 ux3-3-1 的稿做出來的，之後 13-1b 把「想要清單」做活了、篩選的實際選項也跟稿分岔了，稿沒跟上。每一條都標了方向，**dev 不要無腦把程式碼改成跟稿一樣**。

### 🔴 建單時查到、sprint-status 條目沒寫的四件事

1. **TMDb 連不上時，篩選軌寫「符合 0 部」。** `useDiscoverResults.ts:105-107` 在查詢失敗時把總數當成 0，`DiscoverFilterRail.tsx:64` 照印；手機的套用鈕也寫「套用篩選（0 部結果）」（`FilterBottomSheet.tsx:142`）。這不是「沒有結果」，是「算不出來」——`disc-2026-08-discover-live-filters-while-source-down` 點名的品牌級誠實問題，數字這一半本張修（AC #4）。
2. **`/search` 頁與頂欄搜尋下拉完全沒有錯誤狀態。** 查詢失敗時顯示「找不到符合的結果」／「找不到「X」的結果」——伺服器壞了卻說你打的字搜不到（AC #5）。
3. **12 個元件檔頭裡有 6 個指向已刪除的節點。** `NWxok`／`TMaw5`／`i74p2`／`oypj1`／`KNI8F`／`dPbq2` 都已不存在（Pencil MCP 驗過）；另有 4 個指向 `rsAxf`，但那是 **Flow C 的 C1-D（媒體庫搜尋＋篩選）**，檔頭卻寫「AS-1 Advanced Filter Chips」「Screen 7」。
4. **設計稿停在「想要清單即將推出」。** 5 張桌機稿的工具列都畫著灰掉的「想要清單 · 即將推出」，但 13-1b 早就把它做活了（`DiscoverBrowseV2.tsx:231` → `RequestsView`）。手機篩選 sheet 也把「地區」「串流平台」畫成即將推出，程式碼兩個都能用。

### 設計稿節點（逐字抄，不要重查）

| 代號 | 節點 | 內容 |
| --- | --- | --- |
| `I1-D-v2` | `fxCVk` | 探索 v2（桌機，篩選軌展開） |
| `I2-M-v2` | `hi6WD` | 探索 v2（手機） |
| `I3-D-v2` | `m0Zew` | 即時搜尋建議（頂欄下拉） |
| `I4-D-v2` | `m4fY7c` | 篩選軌收合 |
| `I4-M-v2` | `kzzjc` | 篩選 sheet（手機） |
| `I5-D-v2` | `nLrzc` | 儲存篩選對話框 |
| `I6-D-v2` | `YYEBd` | 載入骨架 |
| `I7-D-v2` | `S3qke` | 無結果 |
| `I8-D-v2` | `KdnVw` | 區段 fail-soft |
| `I7-D` | `SgncH` | 篩選軌狀態 spec（舊） |
| `I5-D` | `vpDLh` | 篩選軌常駐（舊；**媒體庫**的 `LibraryFilterRail` 指向它，dsr-1 定的，本張不動） |
| `C1-D` | `rsAxf` | ⚠️ Flow C 的媒體庫搜尋＋篩選，**不是探索頁** |
| `Component/Pagination` | `Me2in` | ⛔ 本張不動（見對照表 #20） |

### 程式碼地圖（全部都有正式掛載）

```
routes/discover.tsx → DiscoverBrowseV2
  ├─ DiscoverFilterRail › ui/FilterRailShell（媒體庫也共用）› search/FilterPanel
  ├─ MediaTypeTabs · PresetChips · FilterChipBar · SavePresetDialog
  ├─ DiscoverStatesV2（骨架／無結果／區段錯誤）
  ├─ media/MediaGrid › media/PosterCard(v1) › requests/RequestButton · AvailabilityBadge
  ├─ ui/Pagination · requests/RequestsView（想要清單）
  └─ FilterBottomSheet（手機）› search/FilterPanel
routes/search.tsx → SearchBar · MediaTypeTabs · SearchResults › MediaGrid · Pagination
shell/AppShellV2 → InstantSearchBar › SearchSuggestions（頂欄搜尋，每一頁都有）
```

### 對照表

| # | 項目 | 設計稿 | 程式碼 | 方向 |
| --- | --- | --- | --- | --- |
| 1 | 錯誤代碼 | I8-D-v2 不顯示代碼 | `DiscoverStatesV2.tsx:89-94` 句尾括號 `（CODE）` | **兩邊都改**：獨立等寬膠囊（dsr-1 A8p-D、dsr-2 的同一個形狀） |
| 2 | 部分失敗的用字 | — | `DiscoverBrowseV2.tsx:143`「**節目**結果暫時無法載入」，分頁寫「影集」 | **碼要修**（節目 → 影集） |
| 3 | TMDb 連不上時的總數 | — | 軌道「符合 0 部」、手機「套用篩選（0 部結果）」 | **碼要修** |
| 4 | `/search` 查詢失敗 | 沒有稿 | 顯示「找不到符合的結果」 | **碼要修** |
| 5 | 頂欄搜尋下拉查詢失敗 | 沒有稿 | 顯示「找不到「X」的結果」 | **碼要修** |
| 6 | 篩選軌 sticky 位置 | — | `FilterRailShell.tsx:48` `top-16`（64px），頂欄是 `h-14`（56px）→ 永久 8px 縫 | **碼要修**（媒體庫也共用） |
| 7 | 篩選軌標題 | `篩選` BodyLg（16） | `h3`、`text-[15px]`；上面是 h1，**跳過 h2** | **稿→碼**（`h2`、`text-base`） |
| 8 | 收合／展開鈕 | — | 兩顆鈕互換時焦點掉到 body | ⛔ 立案（補記到 rails a11y 條目；AC #6 不加 `aria-expanded`） |
| 9 | 想要清單 | 灰掉＋「即將推出」 | 活的（13-1b） | **碼→稿** |
| 10 | 地區選項 | 美國／日本／韓國／台灣／香港 | 國旗＋台灣／日本／韓國／美國／中國（`discoverFilters.ts:60-64`、`FilterPanel.tsx:236`） | **碼→稿** |
| 11 | 串流平台選項 | Netflix／Disney+／Apple TV+／HBO／Prime | Netflix／Disney+／KKTV（`:72-74`） | **碼→稿** |
| 12 | 評分選項 | 7+／8+／9+ | `★6+`／`★7+`／`★8+`／`★9+`（`discoverFilters.ts:78`、`FilterPanel.tsx:288`），標題「最低評分」 | **碼→稿** |
| 13 | 年份 | 年代 chip（2020s…90s） | 兩個數字輸入「最早年份／最晚年份」，標題「年份範圍」 | **碼→稿** |
| 14 | 手機 sheet | 地區／串流平台「即將推出」；標題「篩選」；「重設」；「套用篩選」 | 兩者都能用；`篩選條件`；`清除全部`；`套用篩選（N 部結果）` | **碼→稿** |
| 15 | 搜尋建議分區 | 媒體庫／TMDb；底部「查看所有「你的」的結果 →」 | 媒體庫／電影／影集／人物；「按 Enter 查看所有結果 →」 | **碼→稿** |
| 16 | 搜尋建議浮層陰影 | 寫死 `#00000066` | `shadow-[var(--shadow-xl)]` | **碼→稿**（改吃 token） |
| 17 | 儲存篩選對話框 | 預填「週末動作片」、區塊「目前篩選」 | 標籤「預設名稱」、placeholder「例：高評分韓劇」、「包含的篩選條件：」 | **碼→稿** |
| 18 | 無結果 | 「沒有符合「復仇者聯盟 外傳」及目前篩選條件的結果」＋「調整搜尋」；chip 是「科幻」但摘要寫「動作」 | 探索頁**沒有文字查詢**；只有「目前篩選：…」＋「清除篩選」 | **碼→稿**（拿掉查詢句與「調整搜尋」、修假資料） |
| 19 | I8 區段 fail-soft | 「媒體庫結果 4 部」一區 ＋ TMDb 一區錯誤 | **沒有媒體庫區**；錯誤分電影／影集 | **碼→稿**（⚖️ 裁定 A，AC #12） |
| 20 | 頁碼 | 母版 `Me2in`：32×32、目前頁 `$accent-subtle` 底；`disc-2026-09-pagination-44px-drift` 說 D1 的 instance 是 44×44 | `ui/Pagination.tsx` 40×40、目前頁泥金實心 | ⛔ **不動**：三個版本互相矛盾，是設計系統決定不是對齊（補記到該條目） |
| 21 | chip 標籤冒號 | — | `類型: 動作`（半形，`discoverFilters.ts:141-183`）、`快速篩選:`（`PresetChips.tsx:90`） | **碼要修**（全形 `：`） |
| 22 | 進行中省略號 | — | `刪除中...`／`儲存中...`（半形三點），同頁的 `計算中…` 是 `…` | **碼要修** |
| 23 | 手機 sheet 無障礙名稱 | — | `FilterBottomSheet.tsx:87` dialog 沒有名稱 | **碼要修** |
| 24 | 分頁 tab | — | `MediaTypeTabs.tsx:47` `aria-controls` 指向不存在的 id | **碼要修**（拿掉） |
| 25 | 死註解 | — | `DiscoverBrowseV2.tsx:182` 還說想要清單是 inert；`FilterBottomSheet.tsx:95` 三元運算兩邊一樣 | **碼要修** |
| 26 | `text-[13px]` | — | `SavePresetDialog.tsx:110,128` | ⛔ **不動**（等 `disc-2026-09-type-scale-even-migration`，dsr-2 判例） |
| 27 | 「已有／已擁有／已入庫」三種說法、`TMDB`／`TMDb` 混用 | — | 見 Dev Notes | ⛔ 立案，文案裁定 |

---

## Acceptance Criteria

1. **Flow I 的稿要讀得到。** `READABLE_FLOWS` 加入 `"flow-i-discover-v2"` 與 `"flow-i-advanced-search"`，每一筆附日期與理由。匯出併到最後一次跑。

2. **檔頭：12 個檔案指向真的存在、而且對的畫面。** 已用 Pencil MCP 驗證：`NWxok`／`dPbq2`／`TMaw5`／`i74p2`／`oypj1`／`KNI8F` **不存在**；`rsAxf` 存在但是 **C1-D**；`fxCVk`／`hi6WD`／`m0Zew`／`m4fY7c`／`kzzjc`／`nLrzc`／`YYEBd`／`S3qke`／`KdnVw` 存在。

   | 檔案 | 現在 | 改成 |
   | --- | --- | --- |
   | `search/DiscoverStatesV2.tsx` | 只寫 I6 | `Design ref: ux-design.pen Screen I6-D-v2 (YYEBd) + Screen I7-D-v2 (S3qke) + Screen I8-D-v2 (KdnVw)` |
   | `search/DiscoverBrowseV2.tsx` | I1 | `Design ref: ux-design.pen Screen I1-D-v2 (fxCVk) + Screen I4-D-v2 (m4fY7c) + Screen I2-M-v2 (hi6WD)` |
   | `search/FilterBottomSheet.tsx` | `oypj1`（已刪）＋ `Source:` 行 | `Design ref: ux-design.pen Screen I4-M-v2 (kzzjc)` |
   | `search/FilterPanel.tsx` | `rsAxf`（C1-D，名稱寫錯） | `Design ref: ux-design.pen Screen I1-D-v2 (fxCVk) + Screen I4-M-v2 (kzzjc)` |
   | `search/FilterChipBar.tsx` | 第 2 行 `rsAxf` | 第 2 行改 `Design ref: ux-design.pen Screen I1-D-v2 (fxCVk)`；第 1 行 `Implements: Component/FilterChip (jD7gF)` 不動 |
   | `search/PresetChips.tsx` | `NWxok`（已刪）＋ `dPbq2`（已刪） | `Design ref: ux-design.pen Screen I1-D-v2 (fxCVk)`；刪掉 `dPbq2` 那行 |
   | `search/SavePresetDialog.tsx` | `i74p2`（已刪） | `Design ref: ux-design.pen Screen I5-D-v2 (nLrzc)` |
   | `search/InstantSearchBar.tsx`、`search/SearchSuggestions.tsx` | `TMaw5`（已刪） | `Design ref: ux-design.pen Screen I3-D-v2 (m0Zew)`（`InstantSearchBar` 第 1 行的 `Implements: Component/SearchInput (6MxLT)` 不動） |
   | `search/SearchResults.tsx` | `rsAxf`「Screen 7」 | 逐字：`// Design ref: ux-design.pen — no current screen frame; /search（TMDb 搜尋結果頁）沒有設計稿，見 disc-2026-09-search-page-no-design`（形狀照 `ui/TmdbAttribution.tsx:1`） |
   | `media/MediaGrid.tsx` | `KNI8F`（已刪） | `Design ref: ux-design.pen Screen I1-D-v2 (fxCVk)`——它是探索頁結果格線的活實作（卡片是 v1，見 `disc-2026-09-related-content-card-v1-vs-v2`） |
   | `routes/discover.tsx` | `rsAxf`「AS-1」 | `Design ref: ux-design.pen Screen I1-D-v2 (fxCVk)`（routes 不受 lint 管，但錯的指標一樣會誤導人） |

   🚨 節點 ID 在 `Design ref:` 那一行、以 `)` 結尾，`)` 後面不准有任何字；no-screen 變體理由寫在同一行分號後（`implements-pen-node-id.js` 的 `DESIGN_REF_RE`）。
   🚨 **lint 只要開頭註解裡「任一行」合格就放行，而且只驗形狀不驗存在**——`FilterChipBar`、`InstantSearchBar` 靠第 1 行的 `Implements:` 就會過，第 2 行寫錯 lint 抓不到。每一行都要自己對表。
   - 其餘的 `// Source: ux-design.pen (Pencil app)` 行（`FilterPanel:2`、`FilterChipBar:3`、`InstantSearchBar:3`、`SavePresetDialog:2`、`SearchSuggestions:2`）**保留**；只有 `PresetChips` 帶節點 id 的那行（`dPbq2`）要刪。`FilterBottomSheet` 的 `Source:` 行同樣保留。
   ✅ 不用改：`DiscoverFilterRail.tsx`、`ui/FilterRailShell.tsx`（`fxCVk` 對）、`MediaTypeTabs.tsx`、`SearchBar.tsx`（元件 id 都在）。
   收尾自檢：本張改過的檔頭裡每個 `disc-*`／`dsr-*` 都要在 sprint-status 找得到。

3. **錯誤代碼改成獨立膠囊，重試有進行中狀態。** 對照表 #1、#2。
   - `DiscoverSectionErrorV2`（`DiscoverStatesV2.tsx:80-104`）：把 `（{code}）` 從句子的 `<span>` 裡拿出來，改成跟 `LibraryStatesV2.tsx:119`、`DetailStatesV2.tsx:130` **同一組 class** 的等寬膠囊（`rounded-[var(--radius-sm)] bg-[var(--bg-tertiary)] px-2 py-0.5 font-mono text-[11px] text-[var(--text-muted)]`），`data-testid="discover-section-error-code"`。這是橫幅不是整頁：膠囊放在句子之後、重試鈕之前的同一列。
   - 新增 `retrying` prop：`DiscoverBrowseV2` 傳 `isFetching`；為 true 時重試鈕停用、寫「重試中…」（TanStack v5 重試期間 `isError` 仍為 true，dsr-2 CR #5 的同一件事）。
   - `DiscoverBrowseV2.tsx:143` 的「節目」→「影集」。
   - 先改 `DiscoverStatesV2.spec.tsx` 讓它紅：膠囊獨立存在、句子裡沒有全形括號、`retrying` 停用。
   - **設計稿 I8-D-v2**：錯誤區塊加一個膠囊（示範 `TMDB_TIMEOUT`）。

4. **TMDb 連不上時，不准印出算不出來的數字。** 對照表 #3。
   - **判斷條件（寫死，不要自己推）**：`countUnavailable = (mediaType !== 'tv' && moviesQuery.isError && !moviesQuery.data) || (mediaType !== 'movie' && tvQuery.isError && !tvQuery.data)`。在 `DiscoverBrowseV2` 這等於既有的 `moviesErr || tvErr`（`:90-91`）。**部分失敗也算**——那時的數字只算了一半。已驗證：TanStack 5.90 只在 pending 時用 placeholder 資料，失敗後 `data` 是 undefined，不會殘留舊數字；重試期間 `isFetching` 會先顯示「計算中…」。
   - `DiscoverFilterRail` 新增**可選** prop `countUnavailable`（預設 false）。為 true 時底部寫 `暫時無法計算`，不顯示數字；「計算中…」優先序不變。
   - `FilterBottomSheet.tsx:45,142`：它自己呼叫的 `useDiscoverResults` 已經回傳 `moviesQuery`／`tvQuery`，用上面同一個條件（sheet 與頁面共用快取，目前分頁用不到的 query 也可能是錯誤狀態，所以一定要看 `mediaType`）；為 true 時按鈕只寫 `套用篩選`。**不要改 hook 的回傳型別。**
   - `MediaTypeTabs.tsx:28-29` 把缺的計數當 0，「全部 N」在影集失敗時只算了電影：`DiscoverBrowseV2` 在 `countUnavailable` 時傳 `movieCount`／`tvCount` 為 undefined，讓分頁不顯示數字。
   - chip 本身**不動**。「chip 要不要帶理由或不顯示」留在 `disc-2026-08-discover-live-filters-while-source-down`。
   - 先寫紅測試：`DiscoverFilterRail.spec.tsx`（`countUnavailable` → 有「暫時無法計算」、沒有「符合」）、`DiscoverBrowseV2.spec.tsx`（兩區都失敗、以及只有影集失敗 → 軌道暫時無法計算、分頁沒有數字）、`FilterBottomSheet.spec.tsx`（失敗 → 按鈕沒有數字；既有 mock `:17-24` 的 `moviesQuery: {}` 不會壞）。
   - **設計稿 I8-D-v2**：篩選軌底部的「符合 N 部」改成「暫時無法計算」。

5. **`/search` 頁與頂欄搜尋下拉：整個請求失敗時不准說「找不到」。** 對照表 #4、#5。
   - **`/search` 頁的接線（寫死）**：
     - 查詢在 `routes/search.tsx:32-33`（`useSearchMovies`／`useSearchTVShows`，兩個都一直開著）。`SearchResults` 目前只收 `movies/tvShows/isLoading/type/currentPage/onPageChange`，新增 props：`isError`、`errorCode`、`onRetry`、`retrying`、`partialFailure`。
     - **錯誤條件只看目前分頁需要的 query，而且要 `isError && !data`**（dsr-2 CR #2：背景重新整理失敗會保留快取資料）。
     - **全部失敗**：`role="alert"` 區塊，`搜尋暫時無法使用，請稍後再試` ＋ 有代碼時的等寬膠囊（讀法同 `DiscoverBrowseV2.tsx:47` 的 `errCode`）＋ `重試`（只 refetch 失敗的 query；`retrying` 時停用、寫「重試中…」）。不顯示「找不到符合的結果」。
     - **部分失敗**（全部分頁、電影成功影集失敗）：照常顯示成功的結果，上方加一行 `影集結果暫時無法載入，其他結果不受影響` ＋ `重試`；「找到 N 個結果」（`SearchResults.tsx:70`）**不印**（數字只算了一半）。
     - 紅測試**寫在 `routes/-search.spec.tsx`**（它已經 mock `tmdbService` 且 `retry: false`）——要穿過路由，不要只餵葉子元件 props（dsr-1 CR #11 的教訓）。
   - **頂欄搜尋下拉的接線（寫死）**：`InstantSearchBar.tsx:52` 目前只取 `{data, isLoading}`，加取 `isError` 並傳給 `SearchSuggestions` 的新 prop `isError`。為 true 且沒有資料時那一列寫 `搜尋暫時無法使用`，不做重試（下一個字就會重查）。紅測試寫在 `SearchSuggestions.spec.tsx`＋`InstantSearchBar.spec.tsx` 各一條。
   - ⚠️ **這只涵蓋「整個請求失敗」（API 掛了、網路斷、5xx）。** 只有 TMDb 斷線時，`search_service.go:105-176` 刻意把 TMDb 的部分當空清單、整體回 200（testsprite TC092 依賴這個行為），所以下拉還是會說「找不到」。那一半要後端加旗標，**本張不做**，已立 `disc-2026-09-instant-search-tmdb-outage-silent`。**不要**讓後端改回錯誤——會打壞 TC092。
   - 副作用（預期的）：`tests/e2e/search.spec.ts:166` 用真的 TMDb（CI 有 `TMDB_API_KEY`），改完之後 TMDb 斷線時它會明確紅，而不是安靜地綠。
   - `/search` 沒有設計稿，不改稿。

6. **共用的篩選軌外殼與標題層級（探索與媒體庫都受影響）。** 對照表 #6、#7。
   - `ui/FilterRailShell.tsx:48`：`sticky top-16` → `top-14`、`h-[calc(100vh-4rem)]` → `h-[calc(100vh-3.5rem)]`（對上 `AppShellV2.tsx:72` 的 `h-14`）。
   - 標題 `h3` → `h2`、`text-[15px]` → `text-base`（稿是 BodyLg 16）。
   - **區塊標題跟著升一階，否則只是把跳階從 h2 搬到 h3**：`search/FilterPanel.tsx:199,223,247,275,297,321` 的 `h4` → `h3`（順帶修好手機 sheet 的 h2→h4）；`library/FilterPanel.tsx` 在篩選軌（instant）模式下的區塊標題 `h4` → `h3`（`:195` 附近，先讀清楚兩種模式怎麼分）。
   - 測試**要明文斷言層級**（現在沒有任何 spec 斷言）：`FilterRailShell.spec.tsx`（h2「篩選」）、`DiscoverFilterRail.spec.tsx` 與媒體庫篩選軌的 spec（區塊是 h3、頁面上沒有 h4）。
   - ⛔ **不加 `aria-expanded`**：收合鈕按下去會把自己拿掉、換成另一顆展開鈕（`DiscoverBrowseV2.tsx:202-217`、`LibraryBrowseV2.tsx:607-625`），焦點掉到 body——真正的問題是焦點交接，不是屬性。補記到 `disc-2026-08-rails-a11y-landmarks-and-unreachable-disabled`。
   - 收單時在 `disc-2026-08-rails-24px-accidental-gap`（(a) 8px 縫）與 `disc-2026-08-rails-minor-drift`（(c) 15px）補一行。

7. **文案一致——只改這份清單，不要照 grep 全改。** 對照表 #21、#22。
   - **程式碼**：`lib/discoverFilters.ts:128`（文件註解）、`:141, 149, 151, 153, 166, 174, 183` 的 `: ` → 全形 `：`（`類型：動作`，冒號後不空格）；`PresetChips.tsx:90` `快速篩選:` → `快速篩選：`；`PresetChips.tsx:164` `刪除中...` → `刪除中…`；`SavePresetDialog.tsx:187` `儲存中...` → `儲存中…`。
   - **要跟著改的斷言**：`lib/discoverFilters.spec.ts:85-89, 110`；`FilterChipBar.spec.tsx:24, 25, 37`；`SavePresetDialog.spec.tsx:33-35`；`PresetChips.spec.tsx:48`；`DiscoverStatesV2.spec.tsx:20, 24`；e2e `tests/e2e/discover-filters.spec.ts:163, 201, 231, 232, 259, 347`、`tests/e2e/saved-filter-presets.spec.ts:150`（`:201` 與 `:150` 比對的是移除鈕名稱 `移除${label}篩選`，不改一定紅）。
   - ⛔ **grep 還會掃到 Flow A／C／N 的 `...`**（`settings/LibraryCard.tsx:180`、`settings/ExploreBlocksSettings.tsx:236`、`library/LibraryBrowseV2.tsx:401`、`settings/LibraryEditModal.tsx:361`、`settings/ExploreBlockEditModal.tsx:251`、`setup/CompleteStep.tsx:69`、`library/BatchProgress.spec.tsx:10`、`-gallery.fixtures.tsx:1616` 會動到 `library-batch-progress` 基準線）——**一個都不要碰**，已立 `disc-2026-09-ascii-ellipsis-sweep`。
   - ⛔ placeholder `搜尋媒體庫...` 不動（e2e 依賴，見 `disc-2026-09-search-page-no-design`）。
   - chip 標籤沒有被存起來（預設存的是篩選 JSON，`PresetChips.tsx:24-31`），改格式不會壞資料。

8. **無障礙小修。** 對照表 #23、#24。
   - `FilterBottomSheet.tsx:87` 的 dialog 加 `aria-labelledby` 指向 `:118` 的 h2。
   - `MediaTypeTabs.tsx:47` 拿掉 `aria-controls`（指向的 id 不存在；沒有 spec 斷言它）。

9. **死註解與死程式碼。** 對照表 #25。`DiscoverBrowseV2.tsx:182` 的 inert 註解改寫成現況；`FilterBottomSheet.tsx:95` 兩邊一樣的三元運算收成一個值。

10. **設計稿跟上程式碼（碼→稿）。** 對照表 #9–#18。全程 Pencil MCP，每張改完用 `ctx.problems` 掃裁切。
    - **想要清單**：I1-D-v2／I3-D-v2／I4-D-v2／I7-D-v2／I8-D-v2 工具列的「想要清單」改成可按的樣子（對照 `DiscoverBrowseV2.tsx:231` 的樣式），刪掉「即將推出」。
    - **篩選軌選項**（I1-D-v2，以及 I3／I7／I8 裡同一條軌道）——照 `search/FilterPanel.tsx` 真正渲染的樣子：
      - 地區：`🇹🇼 台灣`／`🇯🇵 日本`／`🇰🇷 韓國`／`🇺🇸 美國`／`🇨🇳 中國`（chip 前有國旗 emoji，`:236`）；
      - 串流平台：Netflix／Disney+／KKTV，區塊標題「平台」；
      - 評分：`★6+`／`★7+`／`★8+`／`★9+`（`:288`），區塊標題「最低評分」；
      - 年份：兩個輸入框、placeholder「不限」（「最早年份／最晚年份」只是螢幕閱讀器標籤，不要畫成可見文字），區塊標題「年份範圍」；
      - 另有一個「排序方式」下拉（`:321`），確認軌道模式下是否渲染再決定畫不畫；
      - 計數數字跟著選項重排，被移除的選項連同計數一起刪，不要留孤兒數字。類型 chip 維持稿上的 8 個代表。
    - **I4-M-v2 手機 sheet**：地區、平台改成可選 chip（同上）、拿掉兩個「即將推出」；標題「篩選條件」；「重設」→「清除全部」；按鈕「套用篩選（412 部結果）」。
    - **I3-D-v2 搜尋建議**：分區改 媒體庫／電影／影集／人物（人物區至少一列）；底部「按 Enter 查看所有結果 →」；擁有標記「已擁有」；浮層陰影從寫死 `#00000066` 改吃設計稿既有的陰影變數（照 dsr-9，不新增）。
    - **I5-D-v2 儲存篩選**：輸入框上方加標籤「預設名稱」、placeholder「例：高評分韓劇」；「目前篩選」→「包含的篩選條件：」。
    - **I7-D-v2 無結果**：標題維持「找不到相符的結果」（`DiscoverStatesV2.tsx:47`）；刪掉帶查詢字串的句子與「調整搜尋」（探索頁沒有文字查詢）；「目前篩選：」後的標籤跟上面的 chip 一致（現在 chip 是科幻、摘要寫動作）。
    - **I8-D-v2**：見 AC #12（裁定 A，改畫部分失敗）。
    - **寫死色**：掃一次 Flow I 群組（`OkokD`）所有 fill／stroke／effect，列出字面 hex（`#00000000` 除外），能對上 token 的改吃變數；對不上的記進 Dev Notes。

11. **補視覺夾具（三個都要，不准跳過）。** 探索頁的狀態元件與篩選軌都沒有夾具，AC #3／#4／#6 的改動會零像素覆蓋。在 `routes/test/-gallery.fixtures.tsx` 新增（id 保留元件名的 `-v2`，照 dsr-2 的 `media-detail-load-error-v2`）：
    - `search-discover-section-error-v2`：`DiscoverSectionErrorV2`，`code: 'TMDB_TIMEOUT'`。
    - `search-discover-no-result-v2`：`DiscoverNoResultV2`，篩選標籤用 `activeFilterChips()` 產生（不要手打——AC #7 改了格式，手打會跟真實輸出分岔）。
    - `search-discover-filter-rail-unavailable`：`DiscoverFilterRail` 帶 `countUnavailable`。它在 `:48` 呼叫 `useDiscoverFacetCounts`，用夾具既有的 `seedQueries`（`:395`）預先塞 `['tmdb','discover','facet-counts', buildFacetCountParams(filters).toString()]` → `{ counts: {}, partial: false }`：資料 5 分鐘內新鮮、`partial: false` 不會輪詢、debounce 初值就等於這個 key，**不會打網路**。這個夾具同時守住 AC #6 的 h2 標題。
    - 全部 `penNode: 'screen-section'`、`statesOnly: ['default']`、不含日期。基準線照 dsr-2 AC #14 的五步流程（`.claude/memory/project_visual_baseline_intentional_change.md`）。⛔ 不要本機產 `-linux.png`。
    - 已確認：AC #3／#6／#7／#8 不會讓任何**既有**夾具出現像素差異（只要遵守 AC #7 的「不要碰」清單）。

12. **⚖️ 探索頁不做「媒體庫結果」區，設計稿收回（Alexyu 2026-09-16 裁定 A）。** 對照表 #19。I8-D-v2 畫了「媒體庫結果 4 部」一區，ux3-3-2 AC #8 也寫了「local results still render」，但程式碼從來沒做——探索頁只有 TMDb 的電影／影集兩個來源。
    - **理由**：探索＝TMDb，你自己的片去媒體庫頁找；而且探索頁的地區、平台篩選在媒體庫資料上大多沒有對應欄位。
    - **設計稿 I8-D-v2**：拿掉「媒體庫結果」一區（`section-local`，`jR2Jk`），改畫程式碼真的會出現的**部分失敗**：「電影」結果照常顯示一排卡片，錯誤橫幅 `影集結果暫時無法載入，其他結果不受影響` ＋ AC #3 的代碼膠囊 ＋ `重試`（位置照 `DiscoverBrowseV2` 的實際順序）；篩選軌底部照 AC #4 寫「暫時無法計算」；分頁不顯示數字。
    - **程式碼不動。**
    - 開發時的 AC Drift 要記：`ux3-3-2-discover-frontend` AC #8「local results still render」從沒出貨，本裁定正式撤銷。

13. **既有測試保留通過**，只有 AC #3／#4／#5／#6／#7／#8 指名的斷言可以改。特別守住 `DiscoverBrowseV2.spec.tsx` 的 11 條、`tests/e2e/discover-filters.spec.ts`（含 `:302` 的 facet counts unavailable）、`saved-filter-presets.spec.ts`、`instant-search.spec.ts`、`search.spec.ts`。

14. **CI 全綠**：`pnpm run lint:all`、`pnpm nx run web:typecheck --skip-nx-cache`、`python3 scripts/check-design-tokens.py`、`pnpm nx test web`、`pnpm nx test api`。⛔ 絕不用 `run_in_background` 跑測試。⛔ 用 `nx test web` 跑全套；局部跑 `cd apps/web && npx vitest run <檔>` 之後**一定要跑 typecheck**（dsr-2 的假綠教訓）。

## Tasks / Subtasks

- [x] **Task 1 — 稿讀得到（AC: #1）**
- [x] **Task 2 — 12 個檔頭（AC: #2）**
  - [x] 照表改；`npx eslint apps/web/src/components/search/ apps/web/src/components/media/MediaGrid.tsx` → 0 errors
  - [x] 收尾自檢 disc-*／dsr-* 引用
- [x] **Task 3 — 錯誤代碼膠囊＋重試進行中＋節目→影集（AC: #3）**：紅 → 綠
- [x] **Task 4 — 不准印算不出來的數字（AC: #4）**：三支 spec 先紅 → `DiscoverFilterRail`／`DiscoverBrowseV2`（含分頁計數）／`FilterBottomSheet` → 綠
- [x] **Task 5 — 搜尋錯誤狀態（AC: #5）**：`routes/-search.spec.tsx` 先紅（全部失敗、部分失敗）→ `SearchResults` 新 props → 綠；`InstantSearchBar`／`SearchSuggestions` 各一條紅 → 綠
- [x] **Task 6 — 篩選軌外殼與標題層級（AC: #6）**：層級斷言先紅 → `FilterRailShell`＋兩個 `FilterPanel` → 綠；跑媒體庫相關 spec
- [x] **Task 7 — 文案一致（AC: #7）**：只照 AC 清單改斷言讓它紅 → 改碼 → 綠；不碰清單外的 `...`
- [x] **Task 8 — 無障礙與死碼（AC: #8, #9）**
- [x] **Task 9 — 設計稿（AC: #10）**：六組改動逐張做、逐張掃裁切
- [x] **Task 10 — I8-D-v2 收回媒體庫結果區、改畫部分失敗（AC: #12）** ⚖️ 已裁定 A
- [x] **Task 11 — 視覺夾具與基準線（AC: #11）**：3 個新夾具＋darwin 基準線；`-linux` 待推分支後 `gh workflow run "Visual Regression" --ref fix/dsr-8-flow-i-discover`（本機無法產生，交給 /ship）
- [x] **Task 12 — 收尾（AC: #13, #14）**
  - [x] 全套閘門；確認 `ux-design.pen` 真的存檔（`git status` 看到 ` M`）再重跑匯出；只有 Flow I 兩個資料夾有變動
  - [x] sprint-status：補記本張點名的條目

## Dev Notes

### 這張的重點

- **AC #4 是最有感的一條。** 「符合 0 部」在 TMDb 斷線時出現，跟「你的篩選太嚴了」長得一模一樣——使用者會去放寬篩選，而問題根本不在篩選。
- **AC #3 是在收自己的尾巴。** 代碼是 dsr-2 讓它出現的，形狀要跟媒體庫、詳情頁一樣。
- **AC #10 工作量最大，但每一條都有程式碼當真值**，不需要設計判斷——唯一要判斷的 AC #12 已經裁定（A：收回稿）。

### 不要做的事

- **不要改 `ui/Pagination.tsx`**——母版 32px、條目說 instance 44px、程式碼 40px，三個版本互相矛盾，是設計系統決定（對照表 #20）。
- **不要動 `text-[13px]`、`text-[11px]`**（dsr-2 判例）。
- **不要把 `MediaGrid` 換成 `PosterCardV2`**（`disc-2026-09-related-content-card-v1-vs-v2`）。
- **不要改 placeholder `搜尋媒體庫...`**（e2e 依賴，另立案）。
- **不要讓篩選 chip 在 TMDb 斷線時停用或消失**（AC #4 只修數字，chip 的問題留在原條目）。
- **不要改後端 `search_service.go` 讓 TMDb 失敗變成錯誤**——testsprite TC092 依賴「TMDb 壞了本地結果照常」（AC #5）。
- **不要照 grep 把全站的 `...` 都改掉**（AC #7 的清單外一律不碰）。
- **不要動 `I5-D`（`vpDLh`）**——媒體庫的篩選軌指向它。
- **不要動 `RequestButton`／`RequestRow`／`RequestsView`**——Flow L（dsr-11 已收）。

### sprint-status 條目已過期的兩處

- 「① FilterBottomSheet／PresetChips／SavePresetDialog／SearchSuggestions 有 raw shadow-*——PresetChips 是 chip 不是浮層」——**已過期**：四個都已經是 `shadow-[var(--shadow-xl)]`（dsr-9 收斂過），而且全部掛在**真的浮層**上（sheet／刪除確認對話框／儲存對話框／建議下拉）。`PresetChips.tsx:136` 的陰影在確認對話框上，不在 chip 上。
- 「② SavePresetDialog.tsx 3 處舊字級」——實際 2 處 `text-[13px]`＋1 處 `text-[11px]`；13px 等型別階裁定，11px 是標準，本張都不動。`FilterRailShell` 的 `text-[15px]` 由 AC #6 修（稿有明確值）。

### 已知陷阱

- **`FilterRailShell` 是共用的**：AC #6 會改到媒體庫頁。
- **`DiscoverBrowseV2.tsx:47` 的 `errCode` 用 `e?.code`**，dsr-2 之後 TMDb 錯誤是 `ApiError`；網路斷線（fetch 直接 reject）沒有 code，膠囊不顯示——這是對的。
- **`useDiscoverResults.ts:105-107` 在失敗時把總數當 0**：AC #4 在呼叫端判斷錯誤，**不要**改 hook 的回傳型別（其他地方可能依賴 number）。
- **`e2e/discover-filters.spec.ts:302`「facet counts unavailable → single total」**：那是 facet 計數失敗、結果成功的情境，總數仍有效；別跟 AC #4 混在一起改壞。
- **行號以本單建立時（main `e6634c5d`）為準**，動手前重新確認。

### 已查過、不用做的

- 陰影：`components/search/` 內 4 個 `shadow-` 全在真浮層上（見上）；`RequestButton` 的 toast 另算，屬 Flow L。
- 寫死色：程式碼範圍內**沒有**任何 hex／rgb；只有 token 加 `/NN` 透明度（Tailwind v4 支援）。
- 牆鐘：範圍內元件沒有讀時鐘（`hooks/useRequestActions.ts:46-47` 的 `new Date()` 屬 Flow L，本張不碰）。
- 播放／加入片單：範圍內沒有。

### Source tree

```
apps/web/src/components/search/DiscoverStatesV2.tsx      ← Task 2, 3
apps/web/src/components/search/DiscoverBrowseV2.tsx      ← Task 2, 3, 4, 6, 9
apps/web/src/components/search/DiscoverFilterRail.tsx    ← Task 4
apps/web/src/components/search/FilterBottomSheet.tsx     ← Task 2, 4, 8, 9
apps/web/src/components/search/SearchResults.tsx         ← Task 2, 5（新 props）
apps/web/src/components/search/SearchSuggestions.tsx     ← Task 2, 5
apps/web/src/components/search/InstantSearchBar.tsx      ← Task 2, 5（傳 isError）
apps/web/src/components/search/FilterPanel.tsx           ← Task 2, 6（h4→h3）
apps/web/src/components/search/{FilterChipBar,PresetChips,SavePresetDialog}.tsx ← Task 2, 7
apps/web/src/components/search/MediaTypeTabs.tsx         ← Task 8
apps/web/src/components/ui/FilterRailShell.tsx           ← Task 6
apps/web/src/components/library/FilterPanel.tsx          ← Task 6（篩選軌模式的 h4→h3）
apps/web/src/components/media/MediaGrid.tsx              ← Task 2
apps/web/src/lib/discoverFilters.ts                      ← Task 7
apps/web/src/routes/discover.tsx                         ← Task 2
apps/web/src/routes/search.tsx、routes/-search.spec.tsx  ← Task 5
apps/web/src/routes/test/-gallery.fixtures.tsx           ← Task 11
tests/e2e/discover-filters.spec.ts、saved-filter-presets.spec.ts ← Task 7（AC #7 點名的行）
ux-design.pen（fxCVk/m0Zew/m4fY7c/kzzjc/nLrzc/S3qke/KdnVw，Flow I 群組 OkokD）← Task 9
scripts/export-pen-screenshots.py                        ← Task 1
```

### Cross-Stack Split Check

後端 task **0 個**，前端／設計 task 約 11 個 → **不拆**。

### Time-dependent visual coverage

- **N/A — no wall-clock-reading components touched.** 範圍內元件都不讀時鐘；AC #11 的夾具不含日期。
- Reference: `project-context.md` Rule 23。

### References

- [Source: `ux-design.pen` Flow I 12 張（表格）] — Pencil MCP 逐節點讀出；`Me2in` 母版值（32×32、`$accent-subtle`）
- [Source: Pencil MCP 節點存在性驗證] — `NWxok`／`dPbq2`／`TMaw5`／`i74p2`／`oypj1`／`KNI8F` 查無；`rsAxf` = C1-D
- [Source: `apps/web/src/components/search/DiscoverStatesV2.tsx:80-104`、`DiscoverBrowseV2.tsx:47, 90-91, 131-149, 160-231`] — AC #3／#4
- [Source: `apps/web/src/hooks/useDiscoverResults.ts:105-107`、`DiscoverFilterRail.tsx:64`、`FilterBottomSheet.tsx:45, 87, 95, 118, 142`] — AC #4／#8／#9
- [Source: `apps/web/src/routes/search.tsx`、`SearchResults.tsx:69-84`、`SearchSuggestions.tsx:106-115`] — AC #5
- [Source: `apps/web/src/components/ui/FilterRailShell.tsx:45-73`、`shell/AppShellV2.tsx:72`] — AC #6
- [Source: `apps/web/src/lib/discoverFilters.ts:54-83, 141-183`] — 真實選項與 chip 標籤，AC #7／#10
- [Source: `apps/web/src/components/library/LibraryStatesV2.tsx`、`media/DetailStatesV2.tsx`] — 錯誤代碼膠囊的既有形狀
- [Source: `_bmad-output/implementation-artifacts/ux3-3-2-discover-frontend.md` AC #8] — 「local results still render」的原始規格（AC #12）
- [Source: `_bmad-output/implementation-artifacts/dsr-2-flow-b-detail-v2.md` AC #14、Debug Log] — 基準線流程、局部 vitest 假綠
- [Source: project-context.md#Rule 21 / #Rule 23 / #Rule 24、`apps/web/src/eslint-rules/implements-pen-node-id.js`]
- [Source: `sprint-status.yaml` → `epic-dsr` / `dsr-8-flow-i-discover-v2` / `disc-2026-08-discover-live-filters-while-source-down` / `disc-2026-08-rails-24px-accidental-gap` / `disc-2026-08-rails-minor-drift` / `disc-2026-08-rails-a11y-landmarks-and-unreachable-disabled` / `disc-2026-09-pagination-44px-drift` / `disc-2026-09-dangling-design-refs-outside-library` / `disc-2026-09-related-content-card-v1-vs-v2` / `disc-2026-09-type-scale-even-migration` / `ux3-discover-v2`]

## Dev Agent Record

### Agent Model Used

Claude Opus 5 (1M context) — `claude-opus-5[1m]`，BMAD dev agent（Amelia）2026-09-17

### Debug Log References

- `pnpm nx test web --skip-nx-cache`：**3451 / 3451 passed**（259 files）。⚠️ 第一次全套跑在機器高負載下（視覺測試剛跑完、系統常駐程序吃滿 CPU）出現 11 條失敗：`eslint-rules/*.spec.ts` 三支與 `ScanProgress.spec.tsx` 一條；單獨跑全綠，機器空下來後重跑全套 **3451/3451 綠**（耗時 982 秒，平常約一半）——負載造成的逾時，不是本張的回歸
- `pnpm nx test api --skip-nx-cache`：PASS
- `pnpm run lint:all`：**0 errors**、128 warnings（既有）；本張改過的 web 檔案單獨 eslint：0 problems · `typecheck`：PASS · prettier：PASS · `check-design-tokens.py`：一致（190 張畫面、73 個母版）
- 視覺：第一次 `--update-snapshots=all`（單一 bucket）跑了 17.3 分鐘，撞到測試本身的 10 分鐘上限而失敗（dsr-2 時同一支是 8.8 分鐘，這次多了 3 個夾具、機器也比較忙）。改用檔案說明裡的 `VISUAL_BUCKETS=4 --workers=4` 重跑。
- ⚠️ **Pencil 的 `Insert` 新建 frame 在這次的篩選軌上會位移**：新插入的 row／input 容器在版面計算裡多出約 50px，畫面上整排 chip 被裁掉看不見（`ctx.problems` 回報 fully clipped、截圖一片空白）。換成 `Copy` 既有節點（原本的 row、chip、`O6ho8I` 輸入框）再改內容就正常。四條篩選軌全部用 Copy 重建。之後改稿如果「插進去的東西不見了」，先懷疑這個。
- ⚠️ 篩選軌的 spacer 設 `fill_container` 時位置算錯（底部「符合 N 部／清除全部篩選」被推出 900px 外），改成依實測算出的固定高度。
- `.pen` 存檔：第一次點 File › Save 沒有寫入（mtime 沒動）；照記憶檔先對節點做同值 `Update` 標髒再存，確認 ` M ux-design.pen`。

### Completion Notes List

- 🔗 **AC Drift: FOUND**（`grep -rn "local results still render\|DiscoverSectionErrorV2\|符合 .* 部\|discover-rail-count\|找不到符合的結果"` 全部 story 檔）
  - `ux3-3-2-discover-frontend` AC #8「local results still render」→ **撤銷**（⚖️ AC #12 裁定 A；該條從未出貨）
  - `ux3-3-2-discover-frontend` AC #3「single live total `符合 N 部`」→ 查詢失敗時改顯示 `暫時無法計算`（成功時行為不變）
  - `tech-spec-ux3-discover-facet-aggregation`「facet 計數失敗時退回單一總數」→ **REUSE**（那是 facet 計數失敗、結果成功的情境，總數仍有效，本張沒改）
  - `dsr-1-flow-a-browse-v2` 的「找不到符合的結果」是媒體庫頁文案 → **REUSE**
- 📎 **Contract Stamps: NONE**（範圍內的上游單都沒有 `[@contract-v*]`；13-1b 的 v1 蓋在請求 API，本張沒碰）
- 🎭 **A11y Pre-Flight: PASS**（改過的 15 個 web 檔案 eslint 0 problems。①圖片：沒有新增；②modal：`FilterBottomSheet` 補上 `aria-labelledby`，焦點與 Escape 原本就有；③非同步揭露：`search-error`／`search-partial-error`／`search-suggestions-error` 都帶 `role="alert"`，探索頁橫幅原本就有；④自訂 widget：`MediaTypeTabs` 拿掉指向不存在 id 的 `aria-controls`。標題層級：探索與媒體庫的篩選軌都改成 h1 → h2 → h3，有測試守住。）
- ✅ **Pre-existing failures: NONE**
- **/search 分頁計數**：AC #5 沒明寫，但跟 AC #4 同一個道理——影集失敗時「全部 N」只算了電影——所以 `/search` 在有一側失敗時也不傳計數給分頁。`/search` 的兩個查詢**不分分頁一直在跑**，所以這裡看兩側，不看目前分頁（CR #2 抓到原本只看目前分頁，「電影」分頁時「全部」還是會印半數）。
- **設計稿的陰影沒有 token**：`.pen` 裡沒有任何陰影變數（實測 0 個），浮層陰影都是字面值。I3-D-v2 的建議浮層改成跟 `Component/DialogFrame`／`Component/BottomSheet` 同一組（`#00000080` y16 blur48，全檔 22 處在用），也就是程式碼 `--shadow-xl` 對應的那一組。
- **設計稿選項換成程式碼的真值**：四條篩選軌（I1／I3／I7／I8）照 `FilterPanel` 的順序重建成 類型 → 地區 → 年份範圍 → 最低評分 → 平台 → 排序方式；只有類型與平台的選中 chip 有打勾（程式碼就是這樣）。類型 chip 從 8 個減到 4 個代表，否則加了排序方式之後底部的計數與清除鈕會被擠出 900px。
- **I4-M-v2 手機 sheet 高度 550 → 760**：六個區塊放不下原本的高度；程式碼是 `max-h-[85vh]` 可捲動（844 的 85% ≈ 717），稿為了不裁切畫到 760。
- **寫死色**：Flow I 群組掃出來只有 `#222222`／`#666666` 兩個，都是畫布上的流程標題與說明文字（不是畫面內容），照 dsr-7 的判例不動。
- **沒做、另立案**（Rule 24 ③）：設計稿工具列跟程式碼的結構差異——稿沒有「全部／電影／影集」分頁、排序在工具列（程式碼在篩選軌）、I3／I4 有「找到 312 部作品」（程式碼沒有）→ `disc-2026-09-discover-toolbar-structure-drift`。
- **設計稿上方的篩選 chip 列**（I1／I3／I4／I7／I8 與兩張手機稿，共 19 顆）也改成程式碼的標籤格式 `類型：動作`／`年份：2020-2025`／`評分：7+`，與 AC #7 的程式碼文案一致。
- **視覺基準線**：新增 3 個夾具的 darwin 圖；驗證跑（不更新）時只有 `retry-retry-notifications`、`parse-floating-parse-progress-card` 兩張有差異，屬既有的本機限定問題（`preexisting-fail-visual-darwin-three-stale-baselines`、`preexisting-fail-parse-progress-darwin-baseline`），與本張無關；另有幾個 bucket 因開發伺服器變慢逾時，重開伺服器後確認。
- 🔍 **對抗式 code review（/ship，fresh-context 只讀代理）**：0 HIGH、1 MED、6 LOW，**全部處理**，5 條新測試都先在還原修正的程式碼上確認會紅：
  1. MED：部分失敗、成功的那一側剛好 0 筆時，橫幅下面又出現「找不到符合的結果，請嘗試使用不同的關鍵字搜尋」→ `/search` 不再顯示那句；**探索頁同樣的情況**（出現「找不到相符的結果＋清除篩選」，等於怪篩選）也一起修。
  2. LOW：`/search` 分頁計數只看目前分頁 → 改看兩側（見上）。
  3. LOW：探索頁「影集」分頁全部失敗時，錯誤代碼可能取到快取裡舊的電影錯誤 → 改取目前失敗那一側的。
  4. LOW：`retrying` 用整頁的 `isFetching`，健康那一側背景重抓時重試鈕會誤顯「重試中…」→ 只看失敗那一側。
  5. LOW：`library/FilterPanel.tsx` 註解說「兩種模式都不跳階」是錯的——手機 sheet（`LibraryFilterSheetV2`）仍是 h2 → h4 → h3 → h4。註解更正，並補記到 `disc-2026-08-rails-a11y-landmarks-and-unreachable-disabled`（本張不修）。
  6. LOW：測試缺口——補上 `role="alert"`、「電影」分頁無計數、重試進行中不說謊。
  7. LOW：AC #6 要在 `disc-2026-08-rails-24px-accidental-gap`／`disc-2026-08-rails-minor-drift` 補一行 → 已補。
- ⚠️ **AC #3 的前提在 query-core 5.90 不成立**：「重試期間 `isError` 仍為 true」只在查詢**已有資料**時才對。錯誤橫幅只在沒有資料時出現，而沒有資料的查詢一 refetch 就退回 `pending`（`query.js` `fetchState`）——所以實際上按下重試，錯誤區塊會**立刻換成載入中**，沒有東西可以連按，也不會在重試中說謊（`-search.spec.tsx` 穿過真的 QueryClient 守住）。`retrying` 保留當防呆；dsr-2 詳情頁的同名 prop 同理（重試→骨架），不算缺陷，不立案。

### Discovery Triage

- **Did this story discover any work outside its current scope?** **YES** —— 建單當下處理如下。

- **① expand-scope-in-place**
  - `disc-2026-08-discover-live-filters-while-source-down` 的**數字那一半** → **AC #4**（chip 那一半留在原條目）
  - `disc-2026-08-rails-24px-accidental-gap` 的 (a) 8px 縫 → **AC #6**
  - `disc-2026-08-rails-minor-drift` 的 (c) `text-[15px]` → **AC #6**
  - `disc-2026-09-dangling-design-refs-outside-library` 的 `MediaGrid.tsx` → **AC #2**
  - `disc-2026-08-rails-a11y-landmarks-and-unreachable-disabled` 的 (2)「篩選軌 aside 沒有名稱」——**已過期**：`FilterRailShell.tsx:47` 現在有 `aria-labelledby`，建單時已在該條目補記；同時補記「收合／展開鈕互換時焦點掉到 body」（AC #6 不加 `aria-expanded` 的理由）

- **② spawn-blocking-story**：無。

- **③ backlog-with-carry-forward-link**
  - **`disc-2026-09-search-page-no-design`** — `/search`（TMDb 搜尋結果頁）沒有任何設計稿：標題「搜尋媒體」、placeholder「搜尋媒體庫...」（但搜的是 TMDb，不是媒體庫）、空狀態用 emoji 🔍。本張只補錯誤狀態。
  - **`disc-2026-09-owned-and-tmdb-wording`** — 「擁有」有三種說法（海報角標「已有」、搜尋建議「已擁有」、想要按鈕「已入庫」），`TMDB` 與 `TMDb` 在同一頁混用。文案裁定。
  - **`disc-2026-09-instant-search-tmdb-outage-silent`** — 頂欄搜尋在「只有 TMDb 斷線」時仍說「找不到「X」的結果」：`search_service.go:105-176` 刻意把 TMDb 失敗當空清單、整體回 200（testsprite TC092）。要誠實得在 `UnifiedSearchResult` 加一個 `tmdb_unavailable` 之類的旗標＋一列提示，不能讓整個請求失敗。AC #5 只修整個請求失敗的情況。
  - **`disc-2026-09-ascii-ellipsis-sweep`** — 全站還有 7 處半形 `...`（`settings/LibraryCard.tsx:180`、`settings/ExploreBlocksSettings.tsx:236`、`library/LibraryBrowseV2.tsx:401`、`settings/LibraryEditModal.tsx:361`、`settings/ExploreBlockEditModal.tsx:251`、`setup/CompleteStep.tsx:69`＋其 spec、`library/BatchProgress.spec.tsx:10`，以及 `-gallery.fixtures.tsx:1616` 會動 `library-batch-progress` 基準線）與 placeholder `搜尋媒體庫...`。AC #7 只改探索頁的。
  - **`disc-2026-09-discover-live-region-noise`** — 探索格線上每一張「已請求／已入庫」卡片的 `RequestButton` 都帶 `role="status" aria-live="polite"`（`RequestButton.tsx:90-91, 113-114`），在 `lg+` 只是 `opacity-0` 仍在無障礙樹裡；翻頁或套篩選時螢幕閱讀器會連念一串。屬 Flow L。
  - **補記 `disc-2026-09-pagination-44px-drift`**：母版 `Me2in` 實測是 **32×32、目前頁 `$accent-subtle` 底＋`$accent-text`**，跟條目寫的「44×44、泥金實心」不同；程式碼 40×40 泥金實心。三個版本，要先裁定正典。

  - **`disc-2026-09-discover-toolbar-structure-drift`**（dev 時立）— 稿的工具列沒有分頁、排序在工具列、I3／I4 多一行「找到 312 部作品」、國旗 emoji 在 Pencil 匯出圖上是方框字母。

- Reference: `project-context.md` Rule 24

### File List

**程式碼：**
- `apps/web/src/components/search/DiscoverStatesV2.tsx`（＋spec）— 代碼膠囊、`retrying`、檔頭
- `apps/web/src/components/search/DiscoverBrowseV2.tsx`（＋spec）— `countUnavailable`、分頁計數、節目→影集、`retrying`、檔頭、註解
- `apps/web/src/components/search/DiscoverFilterRail.tsx`（＋spec）— `countUnavailable`、層級測試
- `apps/web/src/components/search/FilterBottomSheet.tsx`（＋spec）— 失敗時不帶數字、`aria-labelledby`、死三元、檔頭
- `apps/web/src/components/search/SearchResults.tsx` — 錯誤／部分失敗狀態、檔頭
- `apps/web/src/routes/search.tsx`（＋`routes/-search.spec.tsx`）— 錯誤條件接線、分頁計數
- `apps/web/src/components/search/SearchSuggestions.tsx`（＋spec）— `isError` 列、檔頭
- `apps/web/src/components/search/InstantSearchBar.tsx`（＋spec）— 傳 `isError`、檔頭
- `apps/web/src/components/search/FilterPanel.tsx` — 區塊 h4→h3、檔頭
- `apps/web/src/components/library/FilterPanel.tsx`（＋`library/LibraryFilterRail.spec.tsx`）— 篩選軌模式區塊 h3
- `apps/web/src/components/ui/FilterRailShell.tsx`（＋spec）— `top-14`、h2、`text-base`
- `apps/web/src/components/search/MediaTypeTabs.tsx`（＋spec）— 拿掉 `aria-controls`
- `apps/web/src/components/search/PresetChips.tsx`（＋spec）、`SavePresetDialog.tsx`（＋spec）— 全形冒號與省略號、檔頭
- `apps/web/src/components/search/FilterChipBar.tsx`（＋spec）— 檔頭、標籤斷言
- `apps/web/src/lib/discoverFilters.ts`（＋spec）— chip 標籤全形冒號
- `apps/web/src/components/media/MediaGrid.tsx`、`apps/web/src/routes/discover.tsx` — 檔頭
- `apps/web/src/routes/test/-gallery.fixtures.tsx` — 3 個新夾具
- `tests/e2e/discover-filters.spec.ts`、`tests/e2e/saved-filter-presets.spec.ts` — 標籤斷言

**視覺基準線：**
- 新增 `tests/visual/components.visual.spec.ts-snapshots/components/{search-discover-section-error-v2,search-discover-no-result-v2,search-discover-filter-rail-unavailable}/default-visual-darwin.png`

**設計與腳本：**
- `ux-design.pen` — Flow I：五張桌機稿的想要清單、四條篩選軌重建（選項／順序／排序方式）、I4-M-v2 sheet、I3-D-v2 建議分區與陰影、I5-D-v2 對話框文案、I7-D-v2 無結果、I8-D-v2 部分失敗（裁定 A）＋代碼膠囊＋暫時無法計算、19 顆篩選 chip 標籤
- `scripts/export-pen-screenshots.py` — `READABLE_FLOWS` ＋ Flow I 兩個資料夾
- `_bmad-output/pen-tokens.json`、`_bmad-output/screenshots/flow-i-discover-v2/*`（9）、`flow-i-advanced-search/*`（2）
- `_bmad-output/implementation-artifacts/sprint-status.yaml`、本檔

## Change Log

| 日期 | 內容 |
| --- | --- |
| 2026-09-17 | 🔍 **/ship 對抗式 CR**：0 HIGH／1 MED／6 LOW 全修（詳見 Completion Notes）。最重要的一條：部分失敗而成功那一側剛好沒結果時，搜尋頁與探索頁都會在「影集暫時無法載入」下面接一句「找不到／清除篩選」，等於又怪使用者——兩頁一起修。另更正 AC #3 對 TanStack 重試狀態的前提。 |
| 2026-09-17 | ✅ **dev-story 完成 → review**（Amelia）。web 3451/3451、api PASS、lint 0 errors、typecheck、prettier、token 一致。最有感的三件：① TMDb 斷線時**不再寫「符合 0 部」**（篩選軌、手機套用鈕、分頁計數都不再印算不出來的數字）；② `/search` 與頂欄搜尋在整個請求失敗時**不再說「找不到結果」**；③ 探索頁的錯誤代碼改成跟其他頁一樣的獨立膠囊。探索與媒體庫的篩選軌標題層級改成 h1→h2→h3。設計稿四條篩選軌照程式碼重建。途中踩到 Pencil `Insert` 新建 frame 會位移的問題（改用 Copy）已記在 Debug Log。 |
| 2026-09-16 | ⚖️ **AC #12 裁定 A**（Alexyu）：探索頁不做「媒體庫結果」區。I8-D-v2 拿掉那一區、改畫程式碼真的會出現的部分失敗（電影照常、影集失敗）；程式碼不動；ux3-3-2 AC #8「local results still render」正式撤銷。 |
| 2026-09-16 | 🔍 **建單後對抗驗證**（fresh-context 驗證代理，只讀）：4 項 CRITICAL、9 項 SHOULD FIX，**全部併入**。最重要的是 AC #5 原本**會假裝修好**：頂欄搜尋在「只有 TMDb 斷線」時後端刻意回 200（TC092），前端的錯誤旗標根本不會亮——dev 用假的失敗請求測會綠，然後回報修好了。現在 AC 明寫只涵蓋整個請求失敗，TMDb-only 的情況另立案。另外：AC #5 原本太模糊做不出來（補上真正的 props、錯誤條件、部分失敗、測試位置）；AC #6 只升軌道標題會把跳階從 h2 搬到 h3（兩個 `FilterPanel` 的區塊標題一起升）；AC #7 照 grep 全改會溢出到 Flow A／C／N 並動到別人的基準線（改成逐行清單）；篩選軌夾具其實可以不打網路地渲染（`seedQueries`）；分頁「全部 N」也有同樣的半數謊；重試要有進行中狀態；稿的選項描述補上國旗、★ 與 placeholder 等真實細節。 |
| 2026-09-16 | Story 建立（SM Bob, create-story）。Flow I 12 張稿以 Pencil MCP 逐節點讀出，與 `components/search/` 14 個檔案＋兩條路由的掛載鏈比對（背景代理盤點檔頭、陰影、字級、寫死色、狀態文案、測試、夾具、標題層級）。查到條目沒寫的四件事：TMDb 斷線時寫「符合 0 部」、`/search` 與搜尋下拉沒有錯誤狀態、6 個檔頭指向已刪節點＋4 個指向 Flow C 的畫面、設計稿停在「想要清單即將推出」。條目裡的陰影一項已過期。一項待裁定（AC #12 媒體庫結果區）。 |

## 裁定紀錄

### ⚖️ 2026-09-16 · 探索頁不做「媒體庫結果」區

I8-D-v2 畫了一區「媒體庫結果」，TMDb 壞掉時照常顯示；ux3-3-2 AC #8 也寫了，但從沒做過。SM 建議收回稿。

**Alexyu 選：A，收回稿。** 探索＝TMDb；探索頁的地區、平台篩選在媒體庫資料上大多沒有對應欄位，做出來也只能套一部分篩選。

落點：AC #12 / Task 10；AC #10 的 I8-D-v2 項。

