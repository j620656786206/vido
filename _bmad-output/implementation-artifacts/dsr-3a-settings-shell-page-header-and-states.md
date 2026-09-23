# Story DSR.3a：設定頁的共用外殼——分頁列、每一頁的標題、共用的「載入失敗」畫面，外觀與效能監控兩頁順手對齊

Status: done

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As the person who opens 設定 on a phone or a desktop,
I want 每一個分頁的標題長得一樣、手機上標題不會吃掉半個螢幕、分頁列兩邊都看得出還有東西可以滑，而且任何一頁載入失敗時都有同一個「重試」可以按,
so that 12 個分頁看起來是同一個地方，而不是 12 個人各自寫的 12 頁。

## Context

`dsr-3`（Flow C 設定）拆出來的**第一張：共用外殼**。**其他五張（`dsr-3b`〜`-3f`）都依賴本張先合併**——它們要用本張新增的 `SettingsPageHeader` 與 `SettingsErrorState`。

### ⚖️ SM 建單裁定：dsr-3 拆成六張（2026-09-23，Bob）

建單時派五個唯讀稽核代理，逐張讀完 Flow C 的 40 張稿（Pencil MCP `Get`＋PNG）並對照 `components/settings/**`、`routes/settings/**`。結果：**稿→碼約 90 處、碼→稿約 35 處、產品做不到的 9 處**，碰到的元件 20 個以上。單張會比 `dsr-6d-b`（8 task）大兩倍以上 → 依 `feedback_split_oversized_stories` **照設定分頁的分組拆**（版面切分線，每一張各自有檔案、測試、價值）：

| 子單 | 範圍（稿） | 主要檔案 |
| --- | --- | --- |
| **`dsr-3a`（本張）** | 所有分頁共用的分頁列、頁面標題、載入失敗樣板；C6 外觀；C14 效能監控 | `SettingsLayout`、新 `SettingsPageHeader`、新 `SettingsErrorState`、12 個 route、`AppearanceSettings`、`SettingsPlaceholder` |
| `dsr-3b` | C4 連線、C23 Sonarr／Radarr、C7 金鑰、C21／C22 金鑰警告 | `QBittorrentForm`、`ArrConnectionForm`、`ApiKeysForm` |
| `dsr-3c` | C8 服務狀態、C15 骨架、C16 載入失敗 | `ServiceStatusDashboard`、`ServiceStatusCard` |
| `dsr-3d` | C9 字幕、C10 自訂首頁 | `LocalizationLevelForm`、`ExploreBlocksSettings` |
| `dsr-3e` | C11 快取、C18 清除確認、C12 日誌、C17 日誌沒結果 | `CacheManagement`、`CacheTypeCard`、`LogsViewer`、`LogFilters`、`LogEntry` |
| `dsr-3f` | C5 備份、C19 還原確認、C20 建立失敗、C13 匯出 | `BackupManagement`、`BackupTable`、`BackupScheduleConfig`、`RestoreConfirmDialog`、`MetadataExport` |

**順序**：`-3a` 先；`-3b`〜`-3f` 互不相依（各自只碰自己的元件），可以任意順序，但**每張都要在 `-3a` 合併後才開工**。

**另外三個範圍裁定**（本張 Context 是傘狀紀錄，其他五張引用這裡）：

1. ⚖️ **C1／C2／C3 不在 dsr-3**。它們掛在 `flow-c-search-settings/` 資料夾，但畫的是**媒體庫**的搜尋篩選面板、批次工具列、檢視偏好選單，不是設定頁。C1 已被 I5-D／A′ 系列取代；C3 畫的功能**在產品裡不存在**（`SettingsGearDropdown` 沒有正式掛載點、存進 localStorage 的「海報大小」沒人讀）；C2 的批次工具列有真實的漂移（稿 3 顆鈕、碼 4 顆）。→ 立 `disc-2026-09-flow-c-library-screens-c1-c3`，等 Alexyu 裁定去留。
2. ⚖️ **C23（Sonarr／Radarr）納入 dsr-3**。原條目寫「⛔ 不含 *arr 設定頁」，理由是「還沒實作所以沒有稿」（2026-09-11）；之後 13-6（PR #434）把功能做出來、C23 也畫了。排除的理由已不成立 → 放進 `dsr-3b`。
3. ⚖️ **稿畫了產品做不到的東西，一律先把稿改成產品的樣子（碼→稿），新功能另立單**。不在 dsr-3 裡加後端。清單見各子單；新立的單子：`disc-2026-09-explore-block-toggle-and-drag`、`disc-2026-09-logs-unfiltered-total`、`disc-2026-09-backup-disk-space-precheck`。

### 這一張做什麼

**碼的部分**：抽兩個共用元件、把 12 個 route 的標題改用它、分頁列補左側淡出與 12px 徽章、外觀頁補頁面標題與寬度、效能監控頁對稿。
**稿的部分**：分頁列母版與 24 個 instance 的一次性清理、所有手機稿的頁面標題與說明字級改成手機規則、效能監控分頁在稿上改畫成停用的樣子。

### 🔴 建單時查到的事（main `b041ea09`；行號皆為現況）

1. **頁面標題沒有共用元件**。10 個 route 各自寫同一段 `<h1 className="mb-2 text-2xl font-bold text-[var(--text-primary)]">`＋`<p className="mb-6 text-sm text-[var(--text-secondary)]">`（`backup.tsx:11`、`cache.tsx:11`、`connection.tsx:18`、`export.tsx:16`、`homepage.tsx:11`、`keys.tsx:12`、`logs.tsx:11`、`scanner.tsx:11`、`status.tsx:11`、`subtitle.tsx:12`）。**外觀**沒有 `<h1>`，元件自己放 `<h2 className="text-lg font-semibold">外觀`（`AppearanceSettings.tsx:33`，18px）；**效能監控**是 `SettingsPlaceholder` 的 `<h2 text-xl>`（`SettingsPlaceholder.tsx:20`）。
2. **手機沒有降一階**。DESIGN.md `:608`「Title 24→20」、`:615`「內文字級不變」。程式碼不分寬度都是 24／14。稿的手機頁面標題綁 `Type/H3`（手機解析成 18）、說明綁 `Label` 12 —— **兩邊都錯**：正確是標題 20（`Type/H2` 的手機值）、說明 14（Body 不變）。
3. **分頁列**（`SettingsLayout.tsx:180-266`）：12 個標籤、順序、4 條分隔線和稿 `iiG1y` 逐字一致，**沒有任何截短**（無 `truncate`／`max-w`／手機短標籤）——sprint 原條目擔心的「標籤自行截短」**不存在**，這條只需要測試釘住。要修的：
   - `:220` 「尚未開放」徽章 `text-[11px]` → 12（`text-xs`）。稿 `cmm2B` 是 Label 12。
   - 只有右側淡出（`:261-265`）；稿兩側都有（`DLjHv`／`JXGqf` 等，但稿用的是**寫死的 `#0c1512`**，日巡會壞）。碼補左側淡出（只在 `scrollLeft > 0` 時顯示）；稿把兩個漸層改成 `$bg-primary` → 透明。
   - `:3-4` 註解「The .pen still shows the RETIRED vertical rail」**過時**（稿早已是橫向分頁列），`:173` 「ten-tab strip measures 1072px」（現在 12 個）。改註解。
   - ~~分隔線：碼 `mx-2`＋`gap-1` ≈ 12px，稿 gap 4。⚖️ **碼→稿**（改母版一次、24 個 instance 自動跟上；碼的 12 讀起來才像分組）。~~ ↪ **dev 推翻（2026-09-23）：稿→碼**——`mx-2` 讓整條在 1440 溢出 1152，見 Completion Notes 的「偏離」。母版維持 gap 4，不要改回去。
4. **效能監控**：碼裡是停用分頁（`SettingsLayout.tsx:131-136`，`aria-disabled` 的 `<span>`，永遠不會有選中樣式），只能手打網址進去。稿 C14-D／C14-M（`aJSKl`／`JUEUD`）卻把它畫成**選中**（`wXZMy` `$accent-subtle`）。⚖️ **碼→稿**：稿的分頁列改成「沒有任何一格選中、效能監控是灰字＋徽章」。佔位內容本身稿→碼：圖示圓 72／圖示 32（手機 64／28）、標題 H3 700（手機 H4 18）、間距一律 16、說明最大寬 420（碼 `max-w-sm` 384）。
5. **載入失敗沒有共用樣板**。C16-D（`uYGBU`）是唯一一張「整頁載入失敗」稿：64 圓 `$error-tint`＋28 `circle-alert`、H3 20／700 `$text-primary` 標題、Body `$text-secondary` 說明、44 高 `$bg-tertiary` `refresh-cw`「重試」。碼裡同樣的情況在五個地方各寫一次、而且都**直接印 `error.message`（後端英文原樣露出）、沒有重試**：`ServiceStatusDashboard.tsx:90-97`、`LogsViewer.tsx:54-69`、`CacheManagement.tsx:38-53`、`BackupManagement.tsx:105-120`、`ApiKeysForm.tsx:218-224`。本張只**做元件**；各頁換上去在各自的子單（`-3b` 金鑰、`-3c` 狀態、`-3e` 快取／日誌、`-3f` 備份）。
6. **外觀頁**（C6-D `B3qPq`／C6-M `XpUjm`）：兩句說明文字與碼的兩種狀態逐字相符（稿桌機畫「跟隨系統」、手機畫「已依你的選擇」）。要修：沒有頁面 `<h1>`（上面 #1）；選項兩張卡在稿上合計 768 寬（`qG91e`），碼 `sm:grid-cols-2` 會撐滿 1152 → 加 `max-w-3xl`。**沒有 `.spec.tsx`、沒有夾具**（全 settings 唯一沒 spec 的元件）。
7. **Rule 21 檔頭**：`components/settings/` 26 個檔**全部有**；`routes/settings/*.tsx` 14 個只有 `connection.tsx` 有。本張補 13 個 route 的檔頭（index／qbittorrent 兩個 redirect 用 no-screen 變體）。
8. **夾具**：settings 的 22 個夾具**全是** `penNode: 'screen-section'`（沒有真的節點 id）、**沒有一個是手機寬**；`SettingsLayout`（分頁列）、`AppearanceSettings` 沒有夾具。
9. **稿的雜項（全部只改稿）**：
   - 手機稿 C12-M／C13-M／C14-M 的畫面與 `tabs-wrap` 沒設 `clip:true`（C10-M、C11-M 有），1099 寬的分頁列在畫布上溢到左邊。
   - C1–C4 的標題格式是「C4 · 設定頁（桌面）」、C5 起是「C6-D · 設定 — 外觀（桌面）」。本張只改 **C4-D／C4-M**（C1–C3 不在範圍）。C4-M 的 y 座標比同列低 100（19581 vs 19481）→ 對齊。
   - 分頁列母版內部 tab 背景是 `#00000000`（`htbx6` 等）——合法（schema 沒有 `transparent`，見 `project_pen_schema_gotchas` #3），**不改**。

### 設計稿節點

| 代號 | 節點 | 說明 |
| --- | --- | --- |
| 母版 | `iiG1y` | `Component/SettingsTabStrip`；分隔線 gap 4→12；`cmm2B` 徽章 Label 12（已是） |
| 手機淡出 | 每張手機稿 `tabs-wrap` 下的 `fade-left`／`fade-right`（例：C4-M `DLjHv`／`JXGqf`、C12-M `Z7otJx`／`ShdpD`） | 寫死 `#0c1512` → `$bg-primary` |
| C4-M 標題／說明 | `cau5D`／`W0v0mj` | 範本：標題改綁 `Type/H2`、說明改綁 `Type/Body`；**其他 12 張手機稿的同名節點同樣改**（dev 用 `Get` 按名稱找） |
| C6-D／C6-M | `B3qPq`／`XpUjm` | 外觀；`qG91e` 選項列 768 |
| C14-D／C14-M | `aJSKl`／`JUEUD` | 效能監控；`wXZMy`／`d6HpA0` 選中覆寫**拿掉**；`L1Eb3R`／`LjMaP` 圖示圓 |
| C16-D | `uYGBU` | `SettingsErrorState` 的正典；`OUqh7` 圖示圓、`eTUqu` 標題、`M5NMY` 說明、`wywZ6` 重試 |
| 群組 | `oqIHS`（Flow C） | 規格註記 `spec-note-dsr-3a` 放群組內手機列下方空白處 |

---

## Acceptance Criteria

1. **設計稿先改（Pencil MCP；每個節點動手前再 `Get` 一次）。**
   - ~~`iiG1y` 母版：分隔線左右間距改成 12（與碼一致）。確認 24 個 instance 跟上（`Get` 數 `ref==="iiG1y"`）。~~ ↪ **dev 推翻：母版不動，碼改成母版的 4px**（見 Completion Notes「偏離」）。
   - 所有手機設定稿（C4-M〜C14-M、C23-M、C15-M、C17-M、C19-M 中**有頁面標題的**）：標題改綁 `Type/H2`（手機解析 20）、說明改綁 `Type/Body`（14）；兩側淡出改 `$bg-primary`→`#00000000`。C12-M／C13-M／C14-M 的畫面與 `tabs-wrap` 補 `clip:true`。
   - C14-D／C14-M：拿掉效能監控的選中覆寫，改成停用樣子（`$text-muted`＋「尚未開放」徽章，與其他 instance 的預設相同）；圖示圓／標題／間距不動（碼去對稿）。
   - C4-D／C4-M 的畫面標題改成 C5 起的格式（「C4-D · 設定 — 連線設定（桌面）」／「C4-M · …（手機）」）；C4-M y 對齊同列。
   - 規格註記 `spec-note-dsr-3a`：
     > 「設定（所有分頁）：頁面標題桌機 24／手機 20，說明兩邊都是 14。分頁列在手機可左右滑，兩側淡出只在那一側還有內容時出現。『效能監控』尚未開放：灰字＋徽章、按不下去。任一頁整頁載入失敗都用 C16 的樣子（圖示、一句人話、重試），不顯示後端的原始錯誤。」
   - 收尾：`problems` **不得增加**；存檔走選單 Save（osascript，同 `dsr-4b-1`）→ `git status --porcelain ux-design.pen` 出現 ` M` → grep 磁碟檔確認 `spec-note-dsr-3a`；**存檔後**才跑 `python3 scripts/export-pen-screenshots.py`；**只 stage 真的有改的** `flow-c-search-settings/*.png` 與 `_bmad-output/pen-tokens.json`，其餘 `git checkout --`。

2. **`SettingsPageHeader`（新，`components/settings/SettingsPageHeader.tsx`）。**
   - Props：`title: string`、`description?: ReactNode`、`testId?`。輸出 `<h1>`＋`<p>`：`<h1>` `text-xl sm:text-2xl font-bold text-[var(--text-primary)]`（手機 20／桌機 24）；`<p>` `text-sm text-[var(--text-secondary)]`（兩邊 14）；間距 `mb-2`／`mb-6` **與現況相同**（桌機零像素變化）。
   - 12 個 route 全部改用它（含 `appearance`：把 `AppearanceSettings` 裡的 `<h2>外觀` 與說明搬上來成為頁面標題，元件內只留選項；`performance`：見 AC #5）。**標題與說明文字一個字都不改**。
   - `settings-page-header-width.spec.ts` 目前讀 connection／keys／subtitle 原始碼確認 `<h1` 在 `max-w-3xl` 之前 → 改成斷言 `SettingsPageHeader` 在 `max-w-3xl` 之前（同一個意圖：標題不被表單寬度限制）。

3. **`SettingsErrorState`（新，`components/settings/SettingsErrorState.tsx`）。**
   - Props：`title: string`、`description: string`、`onRetry: () => void`、`isRetrying?: boolean`、`testId?`。長相照 C16-D：64 圓 `bg-[var(--error-tint)]`＋`CircleAlert` 28 `text-[var(--error-text)]`；標題 `text-xl font-bold`（手機 `text-lg`）`text-primary`；說明 `text-sm text-secondary`；重試鈕 `min-h-11`、`bg-[var(--bg-tertiary)]`、`rounded-[var(--radius-md)]`、`RefreshCw`、文字「重試」；`isRetrying` 時停用並顯示「重試中…」。容器 `role="alert"`。
   - ⚠️ 本張**不替換**任何頁面的錯誤畫面（各子單做），只做元件＋spec＋夾具。
   - 📎 `project_tanstack_refetch_no_data_resets_pending`：沒有快取資料的查詢一 refetch 就退回 pending，「重試中…」在真實頁面上可能根本看不到——元件照做，但**不要**在子單用它當驗收。

4. **分頁列（`SettingsLayout.tsx`）。**
   - 徽章 `text-[11px]` → `text-xs`。
   - 左側淡出：新 `data-testid="settings-tabs-fade-left"`，只在 strip `scrollLeft > 0` 時渲染（監聽 `scroll`；`scrollIntoView` 之後也要更新一次）；右側淡出改成只在「右邊還有內容」時渲染（`scrollLeft + clientWidth < scrollWidth - 1`）。桌機 1440 不溢出 → **兩個都不出現**（今天右側那條在桌機也一直在，這是順手修掉的假訊號）。
   - 改 `:3-4`、`:173` 兩段過時註解。
   - ⛔ 標籤、順序、分組、`aria-label`、`data-status`、44px 觸控、`nav` 語意**一個都不動**。

5. **效能監控與外觀（C14、C6）。**
   - `SettingsPlaceholder`：圖示圓 `size-18`（72）／圖示 32、手機 `max-sm:size-16`／28；標題 `text-xl font-bold`（手機 `text-lg`）；元素間距一律 16（`mb-4`／`mt-4`，拿掉 `mb-2`）；說明 `max-w-[420px]`。**不加頁面 `<h1>`**（稿 `GQoB1`／`O5N9J` 是 `enabled:false`，與碼一致）——⚠️ 這是 AC #2「12 個 route 都用 `SettingsPageHeader`」的**唯一例外**，理由寫進 route 檔的註解。
   - `AppearanceSettings`：選項格加 `max-w-3xl`；頁面標題搬到 route（AC #2）。

6. **Rule 21 檔頭**：13 個 route 補檔頭（`// Design ref: ux-design.pen Screen C?-D (id) · C?-M (id)`，id 用上表／SCREENS dict）；`index.tsx`／`qbittorrent.tsx` 用 no-screen 變體（「redirect only — no screen」）。`SettingsPageHeader`／`SettingsErrorState` 檔頭指 C4-D（標題）／C16-D（錯誤）。

7. **既有的行為不准回歸。**
   - ≥640：既有 22 個 `settings-*` 視覺基準**零變動**（它們是元件夾具，不含 route 標題）。
   - 既有 spec 斷言一條不改；`SettingsLayout.spec.tsx` 既有的標籤／順序／`aria-disabled` 斷言照舊。
   - ⛔ 不改任何分頁的內容元件（`QBittorrentForm`、`ApiKeysForm`… 屬各子單）；⛔ 不改後端。

8. **測試。** 紅／守（Rule 16）。
   - `SettingsPageHeader.spec.tsx`（新）：（紅）`h1` 帶 `text-xl` 與 `sm:text-2xl`；`p` 帶 `text-sm` 且**不帶** `text-xs`；description 缺省時不渲染 `<p>`。
   - `SettingsErrorState.spec.tsx`（新）：（紅）`role="alert"`、標題／說明逐字、按「重試」呼叫 `onRetry` 一次、`isRetrying` 時按鈕 disabled 且文字「重試中…」、**不渲染任何 `error.message` 類 prop**（元件沒有這個 prop——型別測試：`// @ts-expect-error` 傳 `error`）。
   - `SettingsLayout.spec.tsx`：（紅）徽章 class 含 `text-xs` 不含 `text-[11px]`；jsdom 裡 mock `scrollWidth`／`clientWidth`／`scrollLeft` → 左淡出只在 `scrollLeft>0` 出現、右淡出只在還有剩餘時出現。（守）**12 個標籤逐字＋順序**（一個陣列比對，釘住「不截短」）。
   - `AppearanceSettings.spec.tsx`（新，今天零覆蓋）：兩種說明文字的條件、選項格帶 `max-w-3xl`、點選項呼叫主題 setter（讀元件現況決定斷言）。
   - `routes/settings/*`：一條結構測試（讀原始碼，同 `settings-page-header-width.spec.ts` 的做法）——12 個 route 中除 `performance` 外都 import `SettingsPageHeader`，而且**沒有任何一個 route 直接寫 `<h1`**。
   - **視覺夾具**（新）：`settings-layout/tabs`（`SettingsLayout` 包一個空 child；`width: 1200`；`penNode: 'f7u7F6'`——C4-D 的分頁列 instance）、`settings-layout/tabs-mobile`（`width: 390`；`penNode: 'I58Afv'`）、`settings-error-state`（`width: 720`；`penNode: 'uYGBU'`）、`settings-appearance`（`width: 1200`；`penNode: 'B3qPq'`）。既有 `settings-settings-placeholder` 的基準**會變**（AC #5，對稿），在 Completion Notes 寫明。只產 darwin；`-linux` 走 bootstrap PR。
   - **e2e**（新 `tests/e2e/settings-shell.spec.ts`；stub 所有 settings API，照 `tests/e2e/arr-settings.spec.ts` 的 route stub 寫法）：
     - **390**：開 `/settings/keys` → `h1` computed `font-size` 20px、描述 14px；分頁列左淡出**不在**、右淡出**在**；把 strip 捲到底 → 左淡出在、右淡出不在。
     - **1440**：同一頁 `h1` 24px；兩個淡出都不在。
     - **640**（`feedback_measure_the_breakpoint_you_return_to`）：`h1` 24px。
     - **12 頁都有 h1**：依序開 11 個可用分頁，每頁恰好一個 `h1` 且文字等於分頁標籤（`/settings/connection` → 「連線設定」…；外觀 → 「外觀」）。
   - **每一項修法做 mutation check**（拿掉 → 必須紅），結果寫進 Completion Notes。

9. **CI 全綠**：`pnpm run format:check`、`pnpm run lint:all`、`pnpm nx run web:typecheck --skip-nx-cache`、`python3 scripts/check-design-tokens.py`、`pnpm nx test web`。⛔ 絕不 `run_in_background` 跑測試；Nx 一律 `NX_DAEMON=false`。🚨 合併之後看 `main` 的 Tests／Docker／Visual Regression。gh 操作一律帶 `GH_TOKEN=$(gh auth token --user j620656786206)`。

## Tasks / Subtasks

- [x] **Task 1 — 設計稿：母版分隔線、手機標題／說明字級、淡出 token、clip、C14 停用樣子、C4 標題格式、規格註記（AC: #1）**
- [x] **Task 2 — `SettingsPageHeader`＋12 個 route 換上＋外觀頁標題搬家＋Rule 21 檔頭（AC: #2, #5, #6, #8）**
- [x] **Task 3 — `SettingsErrorState`（元件＋spec＋夾具）（AC: #3, #8）**
- [x] **Task 4 — 分頁列：徽章 12、兩側條件淡出、過時註解（AC: #4, #8）**
- [x] **Task 5 — 效能監控佔位對稿、外觀頁寬度與新 spec（AC: #5, #8）**
- [x] **Task 6 — 視覺夾具、e2e、mutation check、收尾（AC: #7, #8, #9）**
  - [x] dev-story Step 9：`c4-d`／`c4-m`／`c6-d`／`c6-m`／`c14-d`／`c14-m`／`c16-d` 截圖比對

## Dev Notes

### 這張的重點

- **這是地基**。兩個新元件的 API 其他五張會直接用；改名或改 props 之前先看它們的 story。
- **桌機零像素變化是驗收的一部分**。`SettingsPageHeader` 在 ≥640 必須和今天 10 個 route 手寫的那段逐 class 相同。
- **稿的手機標題兩個方向都錯**：綁 H3（18）太小、說明綁 Label（12）也太小。規則是「標題降一階、內文不變」→ 20／14。
- **分頁列今天沒有截短**。原條目的擔心已確認不存在；用一條「12 個標籤逐字」的測試把它釘死，不要再改標籤。

### 上游契約（Rule 20）

- 本張不定義也不消費任何 `[@contract-v*]`。

### 建單裁定（2026-09-23，SM；Sally／Alexyu 可在 review 推翻）

1. ⚖️ **拆成六張**（見 Context）。
2. ⚖️ **C1–C3 移出**、**C23 納入**、**產品做不到的一律碼→稿＋另立單**（見 Context）。
3. ⚖️ **分隔線間距碼→稿**（12，改母版）。
4. ⚖️ **效能監控在稿上改成停用**（碼→稿）；佔位內容稿→碼。
5. ⚖️ **錯誤樣板本張只做元件**，替換在各子單——避免本張碰到五個別人的元件。

### 不要做的事

- 不要改任何分頁標籤文字、順序、分組。
- 不要在 `SettingsErrorState` 開一個接 `error`／`message` 的 prop——「不把後端原文露給使用者」是這個元件存在的理由。
- 不要在本張替換任何頁面的載入／錯誤畫面。
- 不要本機產 `-linux.png`；不要改舊 spec 的斷言。

### 已知陷阱

- **`size-18` 在 Tailwind v4 是 4.5rem = 72px**（v4 動態 spacing），不用寫任意值；確認專案 Tailwind 版本後再用。
- **jsdom 沒有版面**：`scrollWidth`／`clientWidth` 要 `Object.defineProperty` mock；真正的淡出行為由 e2e 390／1440 守。
- **`scrollIntoView` 在 jsdom 不存在**——`SettingsLayout.spec.tsx` 應已有 stub，沿用。
- **Pencil**：`execute` 失敗會 rollback；`Replace` 會把沒寫的屬性重設；`Copy` 一般節點的 `descendants` 名稱 key 會被靜默忽略；新 `Insert` 的節點存檔前截圖不出來（`project_pen_schema_gotchas`）。**MCP 讀的是 app 記憶體不是磁碟**——存檔後 grep 磁碟檔確認（`feedback_verify_pen_saved_before_commit`）。
- **匯出是非決定性的**：每張 PNG 都會有位元差，只 stage 真的有改的。
- **gh 帳號會被別的 session 切走**：所有 `gh` 指令帶 `GH_TOKEN=…`。

### Source tree

```
ux-design.pen、_bmad-output/pen-tokens.json、screenshots/flow-c-search-settings/*.png（只 stage 有改的）  ← Task 1
apps/web/src/components/settings/SettingsPageHeader.tsx（新，+spec）                                         ← Task 2
apps/web/src/routes/settings/*.tsx（12 個換標題、13 個補檔頭）；settings-page-header-width.spec.ts          ← Task 2
apps/web/src/components/settings/AppearanceSettings.tsx（+新 spec）                                          ← Task 2/5
apps/web/src/components/settings/SettingsErrorState.tsx（新，+spec）                                         ← Task 3
apps/web/src/components/settings/SettingsLayout.tsx（+spec）                                                 ← Task 4
apps/web/src/components/settings/SettingsPlaceholder.tsx（+spec）                                            ← Task 5
apps/web/src/routes/test/-gallery.fixtures.tsx；tests/e2e/settings-shell.spec.ts（新）                       ← Task 6
```

### Cross-Stack Split Check

後端 task **0**、前端／設計／測試 task 6 → 不觸發跨棧拆分。規模：兩個小新元件、12 個 route 的機械替換、一個元件的條件渲染、稿的一次性清理——與 `dsr-1b-c` 同級。

### Time-dependent visual coverage

- **N/A — no wall-clock-reading components touched.**

### References

- [Source: `apps/web/src/components/settings/SettingsLayout.tsx:1-4, 57-137, 153-162, 173, 185, 205-224, 244-246, 261-265`；`AppearanceSettings.tsx:33-43`；`SettingsPlaceholder.tsx:20-33`；`routes/settings/*.tsx`（標題行見 🔴 #1）；`routes/settings/settings-page-header-width.spec.ts`]
- [Source: `ServiceStatusDashboard.tsx:82-97`、`LogsViewer.tsx:54-69`、`CacheManagement.tsx:38-53`、`BackupManagement.tsx:105-120`、`ApiKeysForm.tsx:218-224`（五個手寫錯誤畫面）]
- [Source: `ux-design.pen` `iiG1y`／`cmm2B`／`htbx6`／`f7u7F6`／`I58Afv`／`cau5D`／`W0v0mj`／`DLjHv`／`JXGqf`／`Z7otJx`／`ShdpD`／`B3qPq`／`XpUjm`／`qG91e`／`aJSKl`／`JUEUD`／`wXZMy`／`d6HpA0`／`GQoB1`／`O5N9J`／`L1Eb3R`／`LjMaP`／`uYGBU`／`OUqh7`／`eTUqu`／`M5NMY`／`wywZ6` —— SM 建單時以 Pencil MCP 讀出（2026-09-23）]
- [Source: `DESIGN.md:364, 604-622`（手機只縮標題、44 觸控）；`scripts/export-pen-screenshots.py:154-193`（Flow C SCREENS）]
- [Source: `sprint-status.yaml` → `dsr-3-flow-c-settings`、`disc-2026-08-settings-mobile-half-the-ia-is-invisible`（分頁列捲動與淡出的由來）]
- [Source: project-context.md#Rule 16／#Rule 20／#Rule 21／#Rule 24；`.claude/memory/feedback_split_oversized_stories.md`、`feedback_measure_the_breakpoint_you_return_to.md`、`feedback_verify_pen_saved_before_commit.md`、`project_pen_schema_gotchas.md`、`project_tanstack_refetch_no_data_resets_pending.md`]

## Dev Agent Record

### Agent Model Used

Claude Opus 5.5 — `claude-opus-5-5`（dev-story，Amelia，2026-09-23）

### Debug Log References

- Pencil：`problems` 67 → **67**（不增）。存檔走選單 Save；`git status` 出現 ` M ux-design.pen`；磁碟檔 grep 到 `spec-note-dsr-3a`（`FtOt1`）。`pen-tokens.json` 重生（67 clipping warnings、196 張、74 母版），`check-design-tokens.py` 一致。
- 匯出兩次（第二次是補 C14-M 捲動位置），每次 196/196；只保留 16 張真的有改的（15 張手機稿＋`c14-d`），其餘 `git checkout --`。
- e2e 需要本機後端：`playwright.config.ts` 的 webServer 用 `go run` 起 api 時第一次撞到 port race 失敗；手動 `VIDO_DATA_DIR=./vido-data VIDO_PORT=8080 go run ./cmd/api` 之後 `reuseExistingServer` 正常。跑完已關掉 8080／4200、`test:cleanup` 乾淨。

### Completion Notes List

- Ultimate context engine analysis completed - comprehensive developer guide created
- 🔗 AC Drift: NONE（checked: `settings-tabs-fade|edge fade|mx-2 h-5|text-[11px]` across `_bmad-output/implementation-artifacts/*.md` — 命中 `feat-settings-tabs-ia.md:83`「scrolls, the clipped edge is visible (an edge fade)」：本張只把「永遠在」收斂成「被切的那一側才在」，被切時一定有淡出 → REUSE 不是 DRIFT；J7-D `settings-page-header-width.spec.ts` 的不變式保留、只把 `<h1` 換成 `<SettingsPageHeader` → REUSE）
- 📎 Contract Stamps: NONE（no [@contract-v*] stamps in this story or upstream refs — 純前端外殼，不定義也不消費 wire contract）
- 🎭 A11y Pre-Flight: PASS（5 components checked — SettingsPageHeader／SettingsErrorState／SettingsLayout／AppearanceSettings／SettingsPlaceholder；0 jsx-a11y warnings on these files；0 introduced。`SettingsErrorState` `role="alert"`、重試鈕 44 高＋`focus-visible` ring、`isRetrying` 真的 disabled；兩條淡出 `aria-hidden`＋`pointer-events-none`；外觀 radiogroup 的 accessible name 由 `titleId` 接到 h1，測試釘住「外觀」；route 檔無 JSX a11y 風險）
- ⚠️ **與 story 字面的一處偏離（分隔線，AC #1／🔴 #3 的 ⚖️）**：story 裁定「碼→稿：母版分隔線改 12」。實作時量出來：分頁列 12 個標籤＋`pr-10` 在碼的 `mx-2` 下是 **1163＋40＝1203px**，超過 1440 時的 1152 內容欄 → **今天在 1440 就有溢出、效能監控被切、右淡出其實是真的**；這跟本張 AC #8「1440 兩個淡出都不在」互斥。改成**稿→碼**：碼拿掉 `mx-2`（分隔線兩側只剩列的 4px，和母版 `iiG1y` 一模一樣），整條 1099＋40＝1139 ≤ 1152，1440 完整放得下。母版我先照 story 改了一輪（包成 17px 的 frame）、量完之後**還原**成原本的 1px rect，16 個手機 instance 的 x／width 也還原。e2e 的 mutation 證實：把 `mx-2` 放回去，1440 那條立刻紅（「strip overflows its 1152px column at 1440」）。
- ⚠️ **偏離 2（外觀的標題位置，AC #2／#5）**：AC 寫「把 `AppearanceSettings` 裡的 `<h2>外觀` 與說明搬到 route」。實作把 `SettingsPageHeader` 放在 `AppearanceSettings` **元件裡**、route 不寫——因為說明文字是 live state（選了主題就要立刻換句子），放在 route 就得把主題狀態往上提一層。代價：若將來有第二個地方掛 `AppearanceSettings`（例如對話框），會多出一個 h1 與重複的 `id="appearance-title"`；結構測試把「外觀是唯一在元件裡渲染 header 的分頁」寫死了，`-3b`〜`-3f` 的頁面一律照 AC 放 route。
- ⚠️ **偏離 3（e2e 沒有 stub API，AC #8）**：AC 寫「stub 所有 settings API」。`settings-shell.spec.ts` 只驗外殼（標題字級、分頁列、淡出），這些不在任何查詢之內，所以直接打本機後端；catch-all stub 反而會碰到首次啟動精靈的檢查、把頁面導去 `/setup`。`-3b`〜`-3f` 追加的頁面內容檢查**要** stub（照 `arr-settings.spec.ts`）。
- ⚠️ **1440 放得下的餘裕只有 13px**（darwin 量 1139／1152）。CI 是 ubuntu chromium、字型後備不同；若 CI 上這條紅了，是字型寬度差，不是回歸——先看 CI 的量測值再決定（最直接的解法是把 `pr-10` 縮小，不是把分隔線加回來）。
- `/settings/performance` 是唯一沒有 `<h1>` 的設定網址（C14 的 page-header `enabled:false`，AC #5 的決定）。
- 🔍 **/ship 對抗式 CR（2026-09-23，獨立 context）0 HIGH／7 MEDIUM／7 LOW，吸收 12 項**：淡出改 `useLayoutEffect`（第一幀就對）、每次導覽都量（不再被「沒溢出就 return」跳過）、`ResizeObserver` 也觀察每個分組（選中分頁是 semibold，導覽會改內容寬但不改盒寬）；新 unit（stub ResizeObserver、寬度變了但沒 scroll → 淡出消失）＋新 e2e（390 開、拉寬到 1440 → 兩側淡出都消失）；`SettingsErrorState` 重試中改 `aria-disabled`（`disabled` 會把鍵盤焦點丟到 `<body>`）、`role="alert"` 只包訊息不包按鈕；分頁列夾具包一層把 `min-h` 歸零（原本 ~700px 空白、高度跟著 viewport 變）；標題夾具改 390 viewport（`penNode` 指的是手機稿）；測試名稱、分隔線 regex。Mutation 再加 3／3 紅（拿掉 ResizeObserver → unit＋e2e 各紅 1；`aria-disabled` 改回 `disabled` → 紅 1）。e2e `--repeat-each=3` **21／21**。未吸收：#5（Linux 寬度，上面那條）、#6（改記偏離 3）。
- ⚠️ **稿的 clip 其實沒問題**：🔴 #9 說 C12-M／C13-M／C14-M 沒設 `clip:true`，實讀 15 張手機稿的 `tabs-wrap` 全部 `clip:true`、畫面 frame 也全是 → 不用改（稽核代理讀錯）。
- ⚠️ **C14-M 捲動位置**：稿原本把手機分頁列捲到最右（x=-757）讓「效能監控」可見，但它是停用分頁、不會有 `data-status="active"`，碼永遠從最左開始 → 稿改成 x=0＋關掉左淡出（`hw6dd` `enabled:false`），與碼一致。
- **Task 1（稿）**：`HMYyz`（C14-D）、`oUeJt`（C14-M）拿掉「效能監控」選中覆寫（`wXZMy` fill → `#00000000`、`d6HpA0` → `$text-muted`／500＝母版預設）；14 張手機稿（C14-M 的 page-header 本來就 `enabled:false`，不動）`page-title` 改綁 `Type/H2`（手機 20）、`page-sub` 改綁 `Type/Body`（14）；29 條淡出漸層 `#0c1512`／`#0c151200` → `$bg-primary`／`#00000000`（剩 0 條 raw）；C4-D／C4-M 畫面標題改成「C4-D · 設定 — 連線設定（桌面）」格式、C4-M y 19581→19481 對齊同列；`spec-note-dsr-3a` 放在手機列下方（x 18020、無重疊）。
- **Task 2**：新 `SettingsPageHeader`（`text-xl sm:text-2xl`／`text-sm`、`mb-2`／`mb-6` 與舊手寫逐 class 相同；沒 description 時 h1 自己帶 `mb-6`；`titleId` 給外觀的 radiogroup 用）。10 個 route 換上；外觀的 h1 放在 `AppearanceSettings` 裡（說明文字是 live state，搬到 route 會失去「選了就換句子」），route 不另寫——結構測試把這個例外寫死。效能監控**不**用 header（C14 的 page-header `enabled:false`），route 檔註解寫明。13 個 route 補 Rule 21 檔頭（redirect 兩個用 no-screen 變體）；`connection.tsx` 補上 C4-M／C23-M。
- **Task 3**：新 `SettingsErrorState`，數值照 C16-D 實讀（`FP7hW` gap 12、圖示圓 64 `$error-tint`＋28 `circle-alert`、標題 20／700（手機 18）、說明 440 寬、重試 44 高 `px-5` 600 `refresh-cw` 16）；沒有任何能帶原始錯誤的 prop（`@ts-expect-error` 測試釘住）。**沒有**替換任何頁面（子單做）。
- **Task 4**：分頁列徽章 `text-[11px]`→`text-xs`；左淡出（`settings-tabs-fade-left`）／右淡出各自只在那一側有被切的內容時渲染（`scroll` 監聽＋`ResizeObserver`＋導覽後 `scrollIntoView` 完立即量一次）；分隔線拿掉 `mx-2`（見上面偏離）；兩段過時註解改掉。
- **Task 5**：`SettingsPlaceholder` 對 C14：`size-18`（72）／圖示 `size-8`，手機 `max-sm:size-16`／`max-sm:size-7`；標題 `text-lg sm:text-xl font-bold`；`gap-4` 取代 `mb-4`／`mb-2`／`mt-4`；說明 `max-w-[420px]`。外觀選項格 `max-w-3xl`、新 `AppearanceSettings.spec.tsx`（6 條，今天之前零覆蓋）。
- **Task 6 測試**：新 unit 6（header）＋6（error）＋6（appearance）＋4（placeholder C14）＋SettingsLayout 淡出 5 條改寫／新增＋labels 3 條＋route 結構 40 條（14 個 route × header／無 h1／檔頭）。`nx test web` **284 files／4162 tests 全綠**。新 e2e `tests/e2e/settings-shell.spec.ts` 6 條（390 字級、640／1440 字級、390 左右淡出三態、390 開後面分頁有左淡出、1440 整條放得下且兩側無淡出、11 個可用分頁各一個 h1 且等於標籤）chromium **`--repeat-each=3` 18/18**。
- **Mutation check：unit 13／13 紅**（h1 拿掉 `text-xl`、右淡出永遠顯示、左淡出永遠顯示、分隔線放回 `mx-2`、徽章回 11px、左淡出算法改 false、重試中字樣、`role="alert"`→`status`、佔位圓回 `p-4`、說明回 `max-w-sm`、外觀拿掉 `max-w-3xl`、外觀拿掉 `titleId`、route 手寫 `<h1`）；**e2e 1／1 紅**（`mx-2` 放回 → 1440 那條紅）。
- **視覺基準**：新 darwin 6 張（`settings-layout/tabs`、`settings-layout/tabs-mobile`（viewport 390×844）、`settings-page-header`、`settings-error-state` default／hover／focus、`settings-appearance`）；`settings-settings-placeholder` 對稿**預期變動**，darwin 更新、stale `-linux` 刪掉讓 CI bootstrap。同時把它的 `penNode` 從 `'utility'` 改成 `'aJSKl'`。全套 visual 連跑三次，新基準穩定。
- ⚠️ **既有的本機視覺差異（非本張）**：`retry-retry-notifications/default` 本機三次都紅，差異全在側軌底部的「儲存空間」區（真的後端資料＋整頁截圖），本張沒有碰 shell／retry；CI 上以 `main` 的結果為準，若 CI 也紅再立單。
- 規格核對（Step 9，`c4-m`／`c6-d`／`c6-m`／`c14-d`／`c14-m`／`c16-d`）：

| Area | Design Spec | Implementation | Match? | Fix Needed |
| --- | --- | --- | --- | --- |
| 手機頁面標題 | H2 手機 20／700 | `text-xl` 20／bold（e2e 量 20px） | ✅ | — |
| 桌機頁面標題 | H2 24 | `sm:text-2xl`（e2e 640／1440 量 24px） | ✅ | — |
| 說明 | Body 14（兩邊） | `text-sm`（e2e 量 14px） | ✅ | — |
| 分頁列 1440 | 12 標籤、4 分隔線、整條 ≤1152 | 1139 ≤ 1152、無淡出 | ✅ | 分隔線 mx-2 拿掉（已修） |
| 分頁列手機淡出 | 兩側、token | 條件渲染、`--bg-primary` | ✅ | — |
| 尚未開放徽章 | Label 12 | `text-xs` | ✅ | — |
| C14 佔位 | 72／32、20／700、gap 16、420 | 同 | ✅ | — |
| C14 分頁列 | 無選中、從左開始 | 同（稿已改） | ✅ | 稿 C14-M 捲動位置（已改） |
| C16 錯誤 | 64 圓、20／700、440、44 重試 | 同 | ✅ | — |
| C6 外觀 | 768 寬兩卡 | `max-w-3xl` | ✅ | — |
| C4-M 分頁列捲動 | 稿把「連線設定」畫在最左 | 碼 `scrollIntoView` 置中，開頭兩個分頁時捲不動（x=0） | ⚠️ 既有 | 不在本張（行為沒變；稿是示意） |

- 🎨 UX Verification: PASS（上表；唯一剩下的 C4-M 捲動示意差異是既有行為，不影響版面）

### File List

- `ux-design.pen`
- `_bmad-output/pen-tokens.json`
- `_bmad-output/screenshots/flow-c-search-settings/{c4,c5,c6,c7,c8,c9,c10,c11,c12,c13,c14,c15,c17,c19,c23}-m.png`、`c14-d.png`
- `apps/web/src/components/settings/SettingsPageHeader.tsx`（新）、`SettingsPageHeader.spec.tsx`（新）
- `apps/web/src/components/settings/SettingsErrorState.tsx`（新）、`SettingsErrorState.spec.tsx`（新）
- `apps/web/src/components/settings/AppearanceSettings.tsx`、`AppearanceSettings.spec.tsx`（新）
- `apps/web/src/components/settings/SettingsLayout.tsx`、`SettingsLayout.spec.tsx`
- `apps/web/src/components/settings/SettingsPlaceholder.tsx`、`SettingsPlaceholder.spec.tsx`
- `apps/web/src/routes/settings/{appearance,backup,cache,connection,export,homepage,index,keys,logs,performance,qbittorrent,scanner,status,subtitle}.tsx`
- `apps/web/src/routes/settings/settings-page-header-width.spec.ts`
- `apps/web/src/routes/test/-gallery.fixtures.tsx`
- `tests/e2e/settings-shell.spec.ts`（新）
- `tests/visual/components.visual.spec.ts-snapshots/components/settings-layout/{tabs,tabs-mobile}/default-visual-darwin.png`（新）
- `tests/visual/components.visual.spec.ts-snapshots/components/settings-page-header/default-visual-darwin.png`（新）
- `tests/visual/components.visual.spec.ts-snapshots/components/settings-error-state/{default,hover,focus}-visual-darwin.png`（新）
- `tests/visual/components.visual.spec.ts-snapshots/components/settings-appearance/default-visual-darwin.png`（新）
- `tests/visual/components.visual.spec.ts-snapshots/components/settings-settings-placeholder/default-visual-darwin.png`（改）、`default-visual-linux.png`（刪，待 CI bootstrap）
- `_bmad-output/implementation-artifacts/sprint-status.yaml`
- `_bmad-output/implementation-artifacts/dsr-3a-settings-shell-page-header-and-states.md`

### Change Log

| Date | Change |
| --- | --- |
| 2026-09-23 | Task 1：設計稿——C14 停用樣子、14 張手機稿標題 20／說明 14、29 條淡出改 token、C4 標題格式、C14-M 捲動位置、`spec-note-dsr-3a`；分隔線母版改動量完後還原（見 Completion Notes） |
| 2026-09-23 | Task 2：`SettingsPageHeader`＋10 個 route＋外觀、13 個 route 檔頭、結構測試 |
| 2026-09-23 | Task 3：`SettingsErrorState`（元件＋spec＋夾具） |
| 2026-09-23 | Task 4：分頁列條件淡出、徽章 12、分隔線拿掉 `mx-2`（1440 放得下） |
| 2026-09-23 | Task 5：效能監控佔位對 C14、外觀 `max-w-3xl`＋新 spec |
| 2026-09-23 | Task 6：6 個夾具、新 e2e 6 條、mutation 14／14、全套 web 4162 綠 |
