# Story DSR.3e：快取管理與系統日誌對齊設計稿——清除前先講清楚會發生什麼，日誌在手機上讀得到訊息，篩到沒東西時有「清除篩選」

Status: review

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As the person who clears cache or digs through logs when something went wrong,
I want 按下「清除」之後、真的清掉之前，看得到一句話告訴我會刪什麼、不會刪什麼；日誌在手機上是「等級＋時間」一行、訊息一行，而不是被時間戳擠到剩 80px；篩到沒東西時告訴我是篩選的關係、一鍵清掉,
so that 我不會因為怕刪錯而不敢按，也不會以為日誌是空的。

## Context

`dsr-3` 拆出來的第五張：**「維護」分組的快取管理、系統日誌**。**依賴 `dsr-3a` 先合併**。與其他子單互不相依。拆單理由見 `dsr-3a` Context。

| 稿 | 節點 | 元件 |
| --- | --- | --- |
| C11-D／C11-M 快取管理 | `TrU8k`／`aYEWP` | `routes/settings/cache.tsx`、`CacheManagement.tsx`、`CacheTypeCard.tsx` |
| C18-D 清除確認 | `dfwSb` | 同上（按第一次之後的狀態） |
| C12-D／C12-M 系統日誌 | `K28SdR`／`dOEbF` | `routes/settings/logs.tsx`、`LogsViewer.tsx`、`LogFilters.tsx`、`LogEntry.tsx` |
| C17-D／C17-M 篩選後沒結果 | `Gw61P`／`J186P` | 同上 |

### 🔴 建單時查到的事（main `b041ea09`；行號皆為現況）

**快取（C11、C18）**
1. 標題、說明、五種快取名稱與後端逐字一致（`apps/api/internal/services/cache_stats_service.go:54-102`）。
2. 快取卡片（稿→碼）：稿 `ceRJp` 實色 `$bg-secondary`、`$radius-lg`、內距 16、高 76；碼 `bg-[var(--bg-secondary)]/50`、`px-4 py-3`（約 60 高）、`rounded-lg`（8px）（`CacheTypeCard.tsx:44`）。容量／筆數那行（`Len0R`／`p7OvlA`）稿用等寬；碼無 `font-mono`（`:54`）。「清除」小鈕稿 600（`x8OUE`），碼 `font-medium`（`:72`）。
3. 手機「清除」鈕：稿 36（`QruOf`），碼 `min-h-[44px]`（`:63`、`:72`，符合觸控規則）→ ⚖️ **碼→稿**（稿改 44）。
4. 「清除 30 天前的快取」主鈕：稿 40 高、500、`clock-3`；碼 `Clock`、約 36（`CacheManagement.tsx:91`）→ 高度 40（`min-h-10`）、圖示 `Clock3`。手機稿文字縮成「清除 30 天前」（`lOTqc`）→ ⚖️ **稿→碼**：`<span className="sm:hidden">清除 30 天前</span><span className="hidden sm:inline">清除 30 天前的快取</span>`，**accessible name 兩個寬度都是完整句**（`aria-label`）。
5. 區塊間距：碼 `space-y-6`（`:56`）→ 稿 16 `space-y-4`。
6. **C18 缺警告條（稿→碼）**：稿 `N0tZx` 在主鈕進入「再按一次」狀態時出現：`$error-tint`、`$radius-md`、16px `triangle-alert`、文字 `aldeK`（Label、`$error-text`）「再按一次才會真的清除。這會刪掉 30 天前的所有快取，之後第一次瀏覽會比較慢，但不會影響影片與字幕檔案。」。碼（`:66-94`）只把按鈕換色。確認鈕 `EBlze`（`$error` 底、`text-on-scrim`、`clock-3`、40）與「取消」`BipN6` **碼已一致**。
   - ⚖️ **警告句要先查證**：dev 讀 `cache_stats_service.go` 與清除 30 天前的 handler，確認 (a) 清的真的是「30 天前」的、(b) 五種快取裡沒有任何一種是影片或字幕檔本身。兩者為真才照稿逐字加；否則改寫成真的句子、稿同步，Completion Notes 寫查證（檔案:行號）。
   - 警告條 `role="status"`（兩段式確認的說明，不是錯誤）、與確認鈕 `aria-describedby` 連起來。
   - 單張卡片的「清除」（`JR7Nz`／確認態 `jfGtx`）**不加**警告條（稿沒畫；單一快取的影響範圍按鈕上已寫）。
7. 載入失敗（`CacheManagement.tsx:38-53`）直接印錯誤 → `SettingsErrorState`：「無法載入快取資訊」／「與後端的連線中斷了。快取本身不受影響。」／`refetch`（SM 擬，待 Sally）。

**日誌（C12、C17）**
8. 每列欄位順序（稿→碼）：稿「展開箭頭 → 等級徽章 → 時間 → 訊息」；碼時間在最右（`LogEntry.tsx:82`）。
9. 時間格式（稿→碼）：稿等寬「2026-09-11 09:42:18」（手機只顯示「09:42:18」）；碼 `toLocaleString('zh-TW')`（`:25`）→ 「2026/9/11 上午9:42:18」、非等寬。新 formatter 用**本地時區**的 `YYYY-MM-DD HH:mm:ss`（不要用 `toISOString`，那是 UTC）；`<time dateTime={iso}>` 包起來。
10. 等級徽章（稿→碼）：稿固定寬 64、等寬（`hOQyo`／`fm0RW`）；碼寬度隨字（`:59`）。DEBUG 稿 `$bg-tertiary`＋`$text-muted`，碼 `text-muted/10` 底＋`text-secondary`（`:13`）。
11. 列樣式（稿→碼）：稿內距 12／16、**列與列之間沒有分隔線**；碼 `py-2.5`＋`border-b`（`:36`）。清單外框：碼 `/50` 半透明（`LogsViewer.tsx:121`）→ 實色、`rounded-[var(--radius-lg)]`。
12. 來源標籤「[source]」：碼有（`LogEntry.tsx:73-77`），稿沒有 → **碼→稿**（真實資料）。
13. **手機排版（稿→碼，這是真的使用 bug）**：稿兩行——第一行徽章＋時間（`HH:mm:ss`）、第二行 12px 訊息（`Sc3vW`）。碼在手機仍是一行：44px 箭頭＋完整時間戳（約 130px），訊息只剩約 80px。→ `<640` 改兩行：第一行箭頭＋徽章＋短時間、第二行訊息 `text-xs`（⚠️ 這是 12px 的**日誌訊息**——DESIGN.md「內文不縮」指的是 Body 內文；日誌訊息屬於等寬資料列，稿明確畫 12。⚖️ **照稿**，理由寫進註解）。
14. 手機搜尋框：碼有（`LogFilters.tsx:53`），手機稿沒畫 → **碼→稿**。
15. 等級篩選 chip（稿→碼）：選中的 ERROR 稿有 `$error-text` 描邊（`cKXI5`／`eWtSQ`），碼 `border-transparent`（`LogFilters.tsx:10`）——每個等級選中時都用自己的 `-text` 色描邊（WARN→`warning-text`、INFO→`info-text`、DEBUG→`text-muted`；dev 讀 `:12-61` 現有的等級色對照沿用）。chip 間距碼 `gap-1.5`→`gap-2`；高度 28（`h-7`）。
16. 搜尋框（稿→碼）：稿 12px、圖示 14、等寬（`xayc0`／`GrTvl`）；碼 `text-sm`、圖示 16、非等寬（`:93`）。⚠️ iOS Safari 會對 <16px 的輸入框 focus 時自動放大——這是既有全站問題不在本張處理，但 e2e 不要在 iOS project 斷言 zoom。
17. **C17 空狀態太簡陋（稿→碼）**：碼（`LogsViewer.tsx:124-130`）只有一行 `text-muted` 置中字，而且**不分**「篩選後沒結果」與「日誌本來就是空的」。稿 `V6r6i9`：56 圓 `$bg-tertiary`＋24 `search-x` `$text-muted`；標題 `UZf5I`「沒有符合條件的日誌記錄」H4 18／600；說明 `Ywq1y`「目前篩選：ERROR ＋ 關鍵字「qbittorrent」。放寬等級或清除關鍵字再試一次。」（依目前篩選組句）；按鈕 `kB52D`「清除篩選」`$bg-tertiary`、40 高。
    - 篩選後沒結果 → 上述完整空狀態；說明句的組法：`目前篩選：{等級們用「、」連}{有關鍵字時「 ＋ 關鍵字「{kw}」」}。放寬等級或清除關鍵字再試一次。`；只有等級沒有關鍵字時說明後半改「放寬等級再試一次。」。
    - 日誌本來就是空的（沒有任何篩選）→ 同樣的圖示結構、標題「還沒有日誌記錄」、沒有按鈕（稿沒畫這一態 → 碼→稿不需要新稿，`spec-note-dsr-3e` 提一句）。
    - 手機說明句（`yfdzz`）稿只到第一個「。」→ ⚖️ **碼→稿**（一套文案）。
18. **計數「共 18,402 筆記錄 · 篩選後 0 筆」產品給不出來**：後端 `Total` 是**套用篩選後**的數（`apps/api/internal/services/log_service.go:98-105`），沒有未篩選的總數。碼今天篩選時顯示「共 0 筆記錄」（`LogsViewer.tsx:78`）——**會誤導**（看起來像日誌全空）。⚖️ **碼改字、稿改字、另立單**：有任何篩選時顯示「符合條件 {N} 筆」，沒有篩選時「共 {N} 筆記錄」；稿 `ZDOda` 改成「符合條件 0 筆」；立 `disc-2026-09-logs-unfiltered-total`（要不要後端多回一個總數）。
19. 「清除 30 天前」的日誌鈕在 C17 的標題列（`WNlRO`）沒畫、碼一直顯示（`LogsViewer.tsx:81-108`）→ ⚖️ **碼→稿**：清舊日誌與目前的篩選無關，篩到沒東西時也該能按。
20. 載入失敗（`LogsViewer.tsx:54-69`）→ `SettingsErrorState`：「無法載入系統日誌」／「與後端的連線中斷了。日誌仍在記錄中。」／`refetch`（SM 擬，待 Sally；「仍在記錄中」要 dev 確認日誌寫入不依賴這支 API——應為真，寫入是後端本機）。

### 設計稿節點

| 代號 | 節點 | 說明 |
| --- | --- | --- |
| C11-D／C11-M | `TrU8k`／`aYEWP` | `ceRJp` 卡片、`Len0R` 容量、`x8OUE` 清除、`QruOf` 手機清除、`lOTqc` 手機主鈕字 |
| C18-D | `dfwSb` | `N0tZx`／`aldeK` 警告條、`EBlze` 確認、`BipN6` 取消、`p7OvlA`、`JR7Nz`／`jfGtx` |
| C12-D／C12-M | `K28SdR`／`dOEbF` | `hOQyo`／`fm0RW` 徽章、`Sc3vW` 手機列、`cKXI5` chip、`xayc0` 搜尋 |
| C17-D／C17-M | `Gw61P`／`J186P` | `V6r6i9` 空狀態、`UZf5I`／`Ywq1y`／`yfdzz`／`kB52D`、`ZDOda` 計數、`WNlRO` 標題列、`eWtSQ`、`GrTvl` |

---

## Acceptance Criteria

1. **設計稿先改（Pencil MCP）**：🔴 #3（手機清除 44）、#12（來源標籤）、#14（手機搜尋框）、#17（手機說明句全句）、#18（`ZDOda`「符合條件 0 筆」）、#19（C17 標題列補「清除 30 天前」）、#6（若警告句查證後要改字）。規格註記 `spec-note-dsr-3e`：
   > 「快取：『清除 30 天前』按第一次後出現一條硃砂說明，講清楚會刪什麼、不會刪什麼，再按一次才清。日誌：每列 箭頭→等級→時間→訊息（手機兩行：等級＋時間 / 訊息）。有篩選時計數寫『符合條件 N 筆』（後端沒有未篩選的總數）。篩到沒東西＝圖示＋一句說明目前的篩選＋清除篩選；日誌本來就空＝『還沒有日誌記錄』、沒有按鈕。」
   收尾同 `dsr-3a` AC #1（只 stage `c11-*`／`c12-*`／`c17-*`／有改到的 `c18-d`＋`pen-tokens.json`）。

2. **快取**：🔴 #2、#4–#7。⛔ 兩段式確認的行為（第一次只換狀態、0 個請求；取消回原狀）**不動**——那是 `disc-2026-08-settings-destructive-ceremony-inverted` 的修法。

3. **日誌列**：🔴 #8–#13。新 formatter 放 `LogEntry.tsx` 同檔或 `lib/`（dev 看有沒有既有的日期工具可重用，**有就重用**）。

4. **日誌篩選列**：🔴 #15、#16。

5. **日誌空狀態與計數**：🔴 #17、#18；「清除篩選」一次清掉等級與關鍵字（呼叫 `LogFilters` 既有的 reset 路徑；沒有就在 `LogsViewer` 加一個 `onClearFilters` 把兩個 state 設回初始值）。

6. **載入失敗**：🔴 #7、#20。

7. **既有的行為不准回歸。**
   - `CacheManagement.spec.tsx`、`CacheTypeCard.spec.tsx`、`LogsViewer.spec.tsx`、`LogFilters.spec.tsx`、`LogEntry.spec.tsx` 的行為斷言不改；時間格式與計數文字的斷言屬本張刻意變更 → Completion Notes 逐條列。
   - 兩段式確認、清舊日誌的確認、展開單筆日誌、分頁／載入更多（若有）行為照舊。
   - ⛔ 不改後端。

8. **測試。** 紅／守（Rule 16）。
   - `CacheManagement.spec.tsx`：（紅）第一次按下後出現警告條、逐字（依查證結果）、`role="status"`、確認鈕 `aria-describedby` 指向它；取消後警告條消失；兩個寬度的 accessible name 都是完整句；區塊 `space-y-4`；載入失敗渲染 `SettingsErrorState`、不含英文錯誤。（守）兩段式確認 0 請求。
   - `CacheTypeCard.spec.tsx`：（紅）實色底、`font-mono` 容量、`font-semibold` 清除。
   - `LogEntry.spec.tsx`：（紅）DOM 順序 徽章→時間→訊息；時間文字符合 `/^\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}$/` 且是**本地時區**（用 `vi.stubEnv('TZ', 'Asia/Taipei')` 或固定 offset 的時間戳驗證）、`<time dateTime>` 是 ISO；徽章 `w-16 font-mono`；DEBUG 的 token；沒有 `border-b`；來源標籤仍在。
   - `LogFilters.spec.tsx`：（紅）選中的 ERROR chip 帶 `--error-text` 框；`gap-2`；搜尋框 `text-xs font-mono`。
   - `LogsViewer.spec.tsx`：（紅）有篩選且 0 筆 → 空狀態標題、說明句三種組法（等級＋關鍵字／只有等級／只有關鍵字）、按「清除篩選」後等級與關鍵字都回初始、再次查詢；無篩選且 0 筆 → 「還沒有日誌記錄」且沒有清除篩選鈕；計數在有篩選時是「符合條件 N 筆」、沒篩選時「共 N 筆記錄」；「清除 30 天前」在空狀態仍在；載入失敗渲染 `SettingsErrorState`。
   - **視覺夾具**：既有 `settings-cache-management`、`settings-cache-type-card`、`settings-logs-viewer`、`settings-log-entry`、`settings-log-filters` 基準**會變**、`penNode` 改真節點；新增 `settings-cache-management/confirm`（`penNode: 'dfwSb'`）、`settings-logs-viewer/filtered-empty`（`penNode: 'Gw61P'`）、`settings-logs-viewer/mobile`（`width: 390`、`penNode: 'dOEbF'`）。⚠️ `LogEntry` 讀時間戳但不讀「現在」→ 夾具用固定時間戳即可；**但時區**：CI（linux）與本機（darwin）的 TZ 可能不同 → 夾具的時間戳要在 visual project 固定 TZ 下渲染（見 `disc-2026-09-visual-project-pin-timezone`；若該單未修，夾具改用 UTC 午夜附近以外的時刻並在 Completion Notes 記錄兩平台結果）。
   - **e2e**（追加到 `tests/e2e/settings-shell.spec.ts`）：390 開 `/settings/logs`（stub 兩筆）→ 每筆訊息的寬度 ≥ 視窗寬 − 64；勾 ERROR＋打關鍵字、stub 回 0 筆 → 看到「沒有符合條件的日誌記錄」與「清除篩選」，按下後兩個條件都清掉。
   - **每一項修法做 mutation check**，結果寫進 Completion Notes。

9. **CI 全綠**：同 `dsr-3a` AC #9。

## Tasks / Subtasks

- [x] **Task 1 — 查證快取警告句（清什麼、不清什麼）與「日誌仍在記錄中」（AC: #2, #6）**
- [x] **Task 2 — 設計稿：手機清除 44、來源標籤、手機搜尋框、C17 全句與計數字與標題列、規格註記（AC: #1）**
- [x] **Task 3 — 快取：卡片、主鈕（手機短字）、C18 警告條、間距、載入失敗（AC: #2, #6, #8）**
- [x] **Task 4 — 日誌列：順序、時間格式、徽章、列樣式、手機兩行（AC: #3, #8）**
- [x] **Task 5 — 篩選列、空狀態兩態、計數文字、清除篩選、載入失敗（AC: #4, #5, #6, #8）**
- [x] **Task 6 — 夾具、e2e、mutation check、收尾（AC: #7, #8, #9）**
  - [x] dev-story Step 9：`c11-d`／`c11-m`／`c12-d`／`c12-m`／`c17-d`／`c17-m`／`c18-d`

## Dev Notes

### 這張的重點

- **手機日誌是真 bug**：訊息被擠到 80px。兩行排版是這張最有價值的修正。
- **「共 0 筆記錄」是誤導**：改成「符合條件 0 筆」就修掉了，不需要後端。
- **警告句是承諾**：先查證。

### 上游契約（Rule 20）

- 本張不定義也不消費任何 `[@contract-v*]`；讀既有的 logs／cache API（implicit v0）。

### 建單裁定（2026-09-23，SM；Sally／Alexyu 可在 review 推翻）

1. ⚖️ 手機快取清除鈕 44（碼→稿）；主鈕手機短字（稿→碼），accessible name 完整。
2. ⚖️ 日誌訊息手機 12px 照稿（等寬資料列，不是 Body 內文）。
3. ⚖️ 計數文字：有篩選「符合條件 N 筆」；未篩選總數另立 `disc-2026-09-logs-unfiltered-total`。
4. ⚖️ 空狀態分兩態；「本來就空」不需新稿。
5. ⚖️ 來源標籤、手機搜尋框、C17 的清舊日誌鈕都碼→稿。

### 不要做的事

- 不要改兩段式確認的請求時機。
- 不要用 `toISOString()` 當顯示時間（UTC）。
- 不要把後端錯誤字串放回畫面上。

### 已知陷阱

- **時區**：jsdom／CI／本機的 TZ 可能不同，時間斷言一律固定 TZ。
- **`Clock3` 是 lucide-react 的名稱**；`.pen` 裡叫 `clock-3`（`project_pen_schema_gotchas` #4）。
- **Pencil／匯出／gh**：同 `dsr-3a`。

### Source tree

```
ux-design.pen、pen-tokens.json、screenshots/flow-c-search-settings/{c11,c12,c17}-{d,m}.png、c18-d.png   ← Task 2
apps/web/src/components/settings/CacheManagement.tsx、CacheTypeCard.tsx（+spec）                         ← Task 3
apps/web/src/components/settings/LogEntry.tsx（+spec）                                                   ← Task 4
apps/web/src/components/settings/LogFilters.tsx、LogsViewer.tsx（+spec）                                 ← Task 5
apps/web/src/routes/test/-gallery.fixtures.tsx；tests/e2e/settings-shell.spec.ts（追加）                 ← Task 6
```

### Cross-Stack Split Check

後端 task **0**（Task 1 只讀）、前端／設計／測試 task 6 → 不觸發。規模：五個元件、多數是 class；日誌列重排與空狀態是實質邏輯——與 `-3b` 同級。

### Time-dependent visual coverage

- `LogEntry` 顯示時間戳（不讀「現在」）→ 固定時間戳＋固定 TZ（見 AC #8）。`CacheTypeCard` 若顯示「N 天前」→ dev 確認；若讀 `Date.now()` 則凍結。

### References

- [Source: `CacheManagement.tsx:38-94`；`CacheTypeCard.tsx:44-86`；`LogsViewer.tsx:54-130`；`LogFilters.tsx:10-93`；`LogEntry.tsx:13-82`]
- [Source: `apps/api/internal/services/cache_stats_service.go:54-102`；`log_service.go:98-105`]
- [Source: `ux-design.pen` 節點見上表 —— SM 建單時以 Pencil MCP 讀出（2026-09-23）]
- [Source: `sprint-status.yaml` → `disc-2026-08-settings-destructive-ceremony-inverted`、`disc-2026-09-visual-project-pin-timezone`；`dsr-3a-settings-shell-page-header-and-states.md`]

## Dev Agent Record

### Agent Model Used

Claude Opus 5.5 (1M context)（Amelia / dev-story）

### Debug Log References

- Pencil：裁切警告 66 → 66；選單 Save 後磁碟 grep 到 `spec-note-dsr-3e`（`wxa4B`）；匯出 196/196，只 stage `c11-m`／`c12-d`／`c12-m`／`c17-d`／`c17-m`＋`pen-tokens.json`（`c11-d`、`c18-d` 稿沒動）。
- 本機 8080 又有前次留下的 `api` 程序，直接沿用。

### Completion Notes List

- Ultimate context engine analysis completed - comprehensive developer guide created
- **Task 1（查證）**：
  - 快取警告句**成立、照稿逐字**。(a) 前端 `CacheManagement` 呼叫 `clearByAge.mutateAsync(30)` → `DELETE /settings/cache?older_than_days=30`（`cache_handler.go:39-58`）→ `ClearCacheByAge`（`cache_cleanup_service.go:48-90`）：刪 `cache_entries`／`ai_cache`／`douban_cache`／`wikipedia_cache` 裡時間早於 cutoff 的列，以及 `imageDir` 裡 mtime 早於 cutoff 的檔案。(b) `imageDir` 是 `data/posters`（`cmd/api/main.go:204-206`），是海報快取，不是影片或字幕檔 →「不會影響影片與字幕檔案」為真。⚠️ 順帶發現：`posters` 目錄同時是中繼資料編輯器上傳海報的位置（`main.go:339`），**使用者上傳超過 30 天的自訂海報也會被這個鈕刪掉**——這是後端既有行為、不在本張範圍 → 已立 `disc-2026-09-cache-clear-deletes-uploaded-posters`（P1，待查證）。
  - 「日誌仍在記錄中」**不成立**：日誌由後端自己的 `slog` DB handler 寫入（`internal/logger/db_handler.go`、`log_repository.go:98`），讀取 API 失敗通常代表後端本身連不上，那時也不會在記錄。→ 改成一定為真的「已記錄的日誌不受影響。」（story 標了 SM 擬、待 Sally）。
- **Task 2（稿）**：C11-M 五顆「清除」與「清除 30 天前」36→44；C12-D 五列、C12-M 四列補來源標籤 `[source]`（等寬、muted），桌機訊息改 `fill_container`；C12-M 補 44 高搜尋框、清除鈕 44；C17-D 計數改「符合條件 0 筆」、標題列補「清除 30 天前」；C17-M 補標題列（計數＋清除）、說明句改全句；`spec-note-dsr-3e`。C18-D 不用改（警告句查證成立）。
- **Task 3（快取）**：卡片實色 `bg-secondary`、`radius-lg`、`p-4`、容量行 `font-mono`、清除 600＋`radius-md`；主鈕 `min-h-10`（手機 44）、`Clock3`、手機短字「清除 30 天前」／桌機全句，`aria-label` 兩個寬度都是全句；區塊 `space-y-4`；**C18 警告條**：`role="status"`、`id` 由確認鈕 `aria-describedby` 指向、取消後消失、第一次按下仍 0 個請求；載入失敗 → `SettingsErrorState`「無法載入快取資訊／與後端的連線中斷了。快取本身不受影響。」（沿用 `dsr-3c` 的「沒資料才換錯誤頁、重抓不閃轉圈」）。
- **Task 4（日誌列，真 bug）**：順序 箭頭→等級→時間→訊息→[來源]；時間**重用 `dsr-3f` 的 `utils/formatLocalDateTime`**（`precision: 'second'`，本地時區，`<time dateTime>`）；手機只顯示 `HH:mm:ss`；徽章 `w-16` 等寬、DEBUG `bg-tertiary`／`text-muted`；列 `px-4 py-3`、無分隔線；清單外框實色 `radius-lg`。**手機兩行**：`flex-wrap`＋訊息 `order-last basis-full`（DOM 順序不變），訊息 12px（照稿；註解寫明這是資料列不是 Body 內文）。
- **Task 5（篩選、空狀態、計數）**：選中的等級 chip 用自己的 `-text` 描邊（ERROR／WARN／INFO；DEBUG 用 `text-muted`）、`gap-2`、`h-7`（手機內距 8 讓五顆一排放得下）；搜尋框 12px 等寬、圖示 14、`radius-md`、手機 44 高。空狀態兩態（`LogsEmpty`）：有篩選＝`search-x`＋「沒有符合條件的日誌記錄」＋依篩選組的說明句（三種組法）＋「清除篩選」；沒篩選＝`scroll-text`＋「還沒有日誌記錄」、沒有按鈕。「清除篩選」同時清等級、關鍵字、頁碼，並以 `key` 重新掛載 `LogFilters` 清掉輸入框。計數：有篩選「符合條件 N 筆」、沒篩選「共 N 筆記錄」。「清除 30 天前」在空狀態仍在。載入失敗 → `SettingsErrorState`「無法載入系統日誌／與後端的連線中斷了。已記錄的日誌不受影響。」；有舊資料時的背景重抓失敗保留畫面。
- **Time-dependent visual coverage**：`LogEntry` 不讀「現在」；夾具用不帶時區的 ISO（當地時間），darwin 與 linux 都印同一個字串（`dsr-3f` 同一招），不需要等 `disc-2026-09-visual-project-pin-timezone`。`CacheTypeCard` 沒有相對時間，N/A。
- **既有 spec 改動（刻意，逐條）**：`CacheManagement.spec` —「renders error state」「displays error message text」改斷言**不出現**英文錯誤並斷言中文說明；「shows dash when stats are not yet loaded」→ 改成「shows the loading state」（沒資料、沒抓完＝載入中，舊的「總計 —」是一頁空殼）。`LogEntry.spec` — DEBUG 徽章 `text-secondary` → `text-muted`＋`bg-tertiary`（稿）。`LogsViewer.spec` —「shows empty state when no logs」（沒篩選）改斷言「還沒有日誌記錄」且沒有清除篩選鈕；「shows error state」加斷言中文說明、不出現原文。
- 🔗 **AC Drift: NONE**（checked: `6-2-cache-management` AC #1–#3（分類顯示、清 30 天前、單類清除要確認）與 `6-3-system-logs-viewer` AC #1–#4（時間／等級／訊息、色碼、分頁、篩選與搜尋、遮罩與提示）——全部 REUSE：欄位都還在、只換格式與排版）。
- 📎 **Contract Stamps: NONE**（本張與上游皆無 `[@contract-v*]`）。
- 🎭 **A11y Pre-Flight: PASS**（5 個元件；觸碰檔 jsx-a11y 警告 0；主鈕 accessible name 全句、警告條 `role="status"`＋`aria-describedby`、空狀態按鈕有字、錯誤頁共用元件）。
- **測試**：`CacheManagement.spec` +4、`CacheTypeCard.spec` +1、`LogEntry.spec` +4、`LogFilters.spec` +2、`LogsViewer.spec` +5（含三種說明句、清除篩選清兩個條件）。`nx test web` **288 files／4267 tests 全綠**；`lint:all`、typecheck 綠。e2e `settings-shell.spec.ts` 追加 2 條（390 每筆訊息寬度 ≥ 視窗 − 96；篩到沒東西→說明→清除篩選兩個條件都清掉），`--repeat-each=3` **6／6**。
- **Mutation：unit 17／17 紅、e2e 2／2 紅**（拿掉警告條、`aria-describedby`、全句名稱、間距、重抓閃轉圈、容量非等寬、UTC 時間、手機一行、舊 DEBUG、分隔線、chip 描邊、搜尋字型、誤導計數、單一空狀態、清除篩選漏關鍵字、說明句、拿掉載入態；e2e：手機一行、清除篩選不重掛輸入框）。
- **視覺基準**：`settings-cache-management`、`settings-cache-type-card`（3）、`settings-logs-viewer`、`settings-log-entry`（3）、`settings-log-filters`（3）的 darwin 基準改變、`penNode` 改真節點；cache-management／logs-viewer 的 hover／focus 刪除（改 `statesOnly: ['default']`）；新增 `settings-cache-management/confirm`（`open` 狀態點主鈕，拍 C18 的警告條）、`settings-logs-viewer/mobile`（390×844）、`settings-logs-viewer/filtered-empty`（`LogsEmpty` 單獨拍，篩選是元件內部 state、夾具無法預設）。過期 `-linux` 全部 `git rm`，等 CI bootstrap。
- ⚠️ **與 story／稿的偏離**：① 載入失敗說明改「已記錄的日誌不受影響。」（story 擬的「仍在記錄中」不保證為真，見 Task 1）。② 手機沒有可展開內容的列仍保留隱形的 44px 箭頭位，第一行比稿多一格縮排（保持每列徽章對齊）。③ 手機篩選 chip 內距 8（稿），桌機 12。

- 🔍 **/ship 對抗式 CR（2026-09-24，獨立 context）0 HIGH／4 MEDIUM／4 LOW／3 NIT，吸收 4M／4L／1N**：
  ① 🟠 手機上報讀器聽不到日誌時間（短時間 `aria-hidden`、完整時間 `display:none`）→ 短時間不再 `aria-hidden`。
  ② 🟠 快取警告條與文字同時掛上，報讀器多半不念 → 常駐、平時空的 `role="status"` 容器，確認時才塞文字（空時 `empty:mb-0` 不佔間距）。
  ③ 🟠 換篩選的瞬間，`keepPreviousData` 的舊資料讓計數寫「符合條件 18,402 筆」、清除篩選後可能閃「還沒有日誌記錄」→ `isPlaceholderData` 期間計數寫「載入中…」、舊列變淡、舊空結果改顯示轉圈，不下結論。
  ④ 🟠 在第 3 頁清掉舊日誌，會同時出現「共 60 筆記錄」「還沒有日誌記錄」「第 3 / 2 頁」→ 清除成功回第 1 頁，另加「頁碼超過總頁數就拉回」的防護。
  ⑤ 清除進行中按鈕的 `aria-describedby` 指向不存在的 id → 條件加上非 pending。⑥ 按「清除篩選」後焦點掉到 body → 移到「全部」chip。⑦ 錯誤頁註解不準（換篩選後失敗也會整頁錯誤）→ 改註解。NIT：只有空白的關鍵字不算篩選（trim）。
  未改：C1（清除篩選的測試從快取拿結果，沒驗「再次查詢」——同一個 query key 本來就不會重打）、C2（快取錯誤頁測試寫法）、C3（e2e 門檻已在 Completion Notes 說明）。
  CR 後：mutation 再 7／7 紅；`nx test web` **288 files／4273 tests** 綠；`lint:all`、typecheck 綠；e2e 整檔 `--repeat-each=3` 57／57；視覺基準不變。

### File List

- `ux-design.pen`
- `_bmad-output/pen-tokens.json`
- `_bmad-output/screenshots/flow-c-search-settings/{c11-m,c12-d,c12-m,c17-d,c17-m}.png`
- `apps/web/src/components/settings/CacheManagement.tsx`（+spec）
- `apps/web/src/components/settings/CacheTypeCard.tsx`（+spec）
- `apps/web/src/components/settings/LogEntry.tsx`（+spec）
- `apps/web/src/components/settings/LogFilters.tsx`（+spec）
- `apps/web/src/components/settings/LogsViewer.tsx`（+spec）
- `apps/web/src/routes/test/-gallery.fixtures.tsx`
- `tests/e2e/settings-shell.spec.ts`
- `tests/visual/components.visual.spec.ts-snapshots/components/settings-{cache-management,cache-type-card,logs-viewer,log-entry,log-filters}/…`（darwin 改／新增、linux 刪除、hover／focus 刪除）
- `_bmad-output/implementation-artifacts/sprint-status.yaml`

### UX Verification（dev-story Step 9）

| Area | Design Spec | Implementation | Match? | Fix Needed |
|------|------------|----------------|--------|------------|
| 快取卡片（C11） | 實色、radius-lg、內距 16、等寬容量、600 清除 | 同 | ✅ | — |
| 主鈕＋警告條（C11／C18） | 40 高、clock-3、手機短字；硃砂說明條逐字 | 同（名稱全句） | ✅ | — |
| 日誌列（C12-D） | 箭頭→等級(64 等寬)→時間(等寬)→訊息，無分隔線 | 同＋[來源]（稿已補） | ✅ | — |
| 日誌列（C12-M） | 兩行：等級＋時間 / 12px 訊息 | 同 | ✅ | 箭頭位見偏離 ② |
| 篩選列 | 選中描邊、gap 8、28 高、12px 等寬搜尋 | 同 | ✅ | — |
| 空狀態（C17） | 圖示＋標題＋說明句＋清除篩選；計數「符合條件 0 筆」；清除 30 天前仍在 | 同；另有「還沒有日誌記錄」態 | ✅ | — |

🎨 UX Verification: PASS — `c11-d`／`c11-m`／`c12-d`／`c12-m`／`c17-d`／`c17-m`／`c18-d` 對照視覺基準逐項比對。

## Change Log

| Date | Change |
| --- | --- |
| 2026-09-24 | Task 1：查證快取警告句（成立）與「日誌仍在記錄中」（不成立→改句） |
| 2026-09-24 | Task 2：設計稿——手機 44、來源標籤、手機搜尋框、C17 計數／標題列／全句、`spec-note-dsr-3e` |
| 2026-09-24 | Task 3：快取卡片、主鈕手機短字、C18 警告條、間距、載入失敗 |
| 2026-09-24 | Task 4：日誌列重排、本地時間、徽章、手機兩行（真 bug） |
| 2026-09-24 | Task 5：篩選 chip 與搜尋框、空狀態兩態、計數文字、清除篩選、載入失敗 |
| 2026-09-24 | /ship CR：手機日誌時間可報讀、警告條常駐 live region、換篩選時不下錯的結論、清舊日誌回第 1 頁、焦點不掉 |
| 2026-09-24 | Task 6：夾具 5 改 3 新、e2e 2 條、mutation 19／19、全套 web 4267 綠 |
