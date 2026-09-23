# Story DSR.3c：服務狀態頁對齊設計稿——服務名稱說中文、載入時有骨架、整頁失敗可以按重試、壞掉的服務告訴你該去哪裡修

Status: review

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As the person who opens 服務狀態 because something looks broken,
I want 看到「豆瓣」「維基百科」而不是「Douban Scraper」「Wikipedia API」、每個服務一眼看出是好是壞、壞掉的時候有一句「去哪裡修」，載入中看得出版面，整頁載不出來時有「重試」,
so that 這一頁真的能幫我排除問題，而不是把後端的英文原樣丟給我。

## Context

`dsr-3` 拆出來的第三張：**服務狀態分頁的三張稿**（正常、載入中、整頁失敗）。**依賴 `dsr-3a` 先合併**（`SettingsPageHeader`、`SettingsErrorState`）。與其他子單互不相依。拆單理由見 `dsr-3a` Context。

| 稿 | 節點 | 畫了什麼 |
| --- | --- | --- |
| C8-D／C8-M | `wqcqY`／`qx8Ma` | 服務卡片列表＋列表下方一條整合的錯誤橫幅 |
| C15-D／C15-M | `XwdOH`／`wkUNt` | 載入骨架 5 列 |
| C16-D | `uYGBU` | 整頁載入失敗（`SettingsErrorState` 的正典，`-3a` 已做元件） |

### 🔴 建單時查到的事（main `b041ea09`；行號皆為現況）

1. **服務名稱是後端的英文**。`ServiceStatusCard.tsx:87` 直接渲染 `service.displayName`，後端寫死 `"Douban Scraper"`／`"Wikipedia API"`／`"AI Parser"`（`apps/api/internal/models/degradation.go:254-256`，TMDb 在同一個 map 上方）。稿用中文名＋一行說明（例：「中繼資料與海報」「評分（選配）」——dev 用 `Get` 從 `wqcqY` 逐字抄每個服務的名稱與說明）。⚖️ **稿→碼，前端對照**：新常數 `SERVICE_LABELS: Record<string, { name: string; role: string }>`，key 是 `service.name`（**不是** displayName）；對照表裡沒有的 name → 退回 `displayName`、不顯示說明（未來後端多一個服務不會壞）。後端**不改**。
2. **稿上 AI 那一列的「· Claude」（`h7Pjj`）產品給不出來**：`ServiceStatus` 型別（`serviceStatusService.ts:16-25`）沒有 provider 欄位 → ⚖️ **碼→稿**：稿刪「· Claude」。
3. **卡片版面（稿→碼）**。稿：左「名稱＋說明」；右「回應 N ms／最後檢查」、帶圖示的狀態 pill、36×36 只有圖示的重新檢查鈕。碼（`ServiceStatusCard.tsx:79-115`）：左邊彩色圓形狀態圖示、狀態是有顏色的文字（不是 pill）、「檢查於 X」、文字鈕「測試連線」。
   - 狀態 pill 顏色照狀態詞彙：`connected`＝青碧（`success-tint`／`success-text`）、`rate_limited`＝赭（`warning-*`）、`error`／`disconnected`＝硃砂（`error-*`）、`unconfigured`＝中性（`bg-tertiary`／`text-muted`＋`circle-slash`，碼今天是 `bg-[var(--text-muted)]/10`＋Settings 圖示，`:48-53`）。pill 的中文字 dev 從稿抄；稿沒畫到的狀態用碼現在的字。
   - 重新檢查鈕：36×36（手機 44×44）、只有圖示 `RefreshCw`、**accessible name 必須是「重新檢查 {中文名}」**；`data-testid="test-btn-{name}"` 保留（既有 spec 與 e2e 靠它）；測試中轉圈。
   - 回應時間 `responseTimeMs`、最後檢查 `lastCheckAt`：資料都有。時間格式沿用碼現在的相對時間寫法（dev 讀 `:96` 附近的 formatter），⚠️ 若是讀 `Date.now()` 的相對時間 → 夾具要凍結時間（見 Time-dependent visual coverage）。
4. **錯誤說明（C8 `blwxn`）**：稿在列表**下方**放一條整合橫幅（`$error-tint`），寫可以照做的建議「…請確認下載器已啟動，或到「連線設定」檢查位址」，並有「重新檢查」。碼是每張卡片各自可展開的「顯示詳情」（`:123-132`），內容是後端原始錯誤字串。
   - ⚖️ **稿→碼＋留一條後路**：有任何服務在 `error`／`disconnected` 時，列表下方出現一條橫幅：每個壞掉的服務一行「{中文名}：{建議}」，建議來自同一張 `SERVICE_LABELS`（新欄位 `fixHint`，**只寫前端能保證為真的通用建議**——例：qBittorrent「請確認下載器已啟動，或到「連線設定」檢查位址。」並附 `<Link to="/settings/connection">`；TMDb「到「金鑰設定」確認 TMDB 金鑰。」附 `/settings/keys`；沒有可靠建議的服務不寫建議、只寫「目前無法連線」）。後端原始錯誤字串**不刪**，收進橫幅裡一個預設收起的「技術細節」（`<details>`），因為排錯時它仍有用。
   - 每張卡上的「顯示詳情」**拿掉**（被橫幅的技術細節取代）。⚠️ 既有 spec 若斷言 `detail-toggle-*` → 屬本張刻意退役，改成斷言橫幅裡的技術細節，Completion Notes 列出。
   - 稿補畫「技術細節」收起狀態（碼→稿）。
5. **手機（C8-M `qx8Ma`）**：稿拿掉每張卡的重新檢查鈕、只剩橫幅裡的 → ⚖️ **碼→稿**（手機不砍功能，每張卡保留 44×44 的鈕）；稿的錯誤橫幅 `yNVad` 被 page-body **切掉** 28px（540 > 512）→ 稿修。
6. **載入中（C15）**：碼只有一顆 `Loader2`（`ServiceStatusDashboard.tsx:82-88`）。稿 `UTfMO`：5 列，列距 12，每列 72 高（手機 64）`$bg-secondary`＋`$border-subtle`＋`$radius-lg`＋內距 16；左兩條 160×18／240×14，右一塊 84×24＋一塊 36×36（手機沒有 36×36），全部 `$bg-tertiary`＋`$radius-sm`。`components/ui/Skeleton.tsx` 已有（`animate-pulse`、`bg-tertiary`、預設 `radius-md`）→ 用它，傳 class 改 `radius-sm`。容器 `aria-busy="true"`＋`aria-label="載入中"`。
   - `lLgMu`／`f8FZa`「骨架微光動畫在 prefers-reduced-motion 下停用。」——dev 用 `Get` 確認是不是 frame 外的設計註記（應該是）。碼：確認 `styles.css:300` 的 reduced-motion 區塊有沒有停掉 `animate-pulse`；沒有就**在 `Skeleton` 加 `motion-reduce:animate-none`**（一行，全站受益）。
7. **整頁失敗（C16）**：碼（`:90-97`）是紅字標題＋`text-muted` 印 `{error.message}`、沒有重試 → 換 `SettingsErrorState`：`title`「無法載入服務狀態」、`description`「與後端的連線中斷了。這不影響已在執行的背景工作。」（`eTUqu`／`M5NMY` 逐字）、`onRetry` = `useServiceStatuses` 的 `refetch`。
8. **狀態變化通知**（`ServiceStatusDashboard.tsx:38-48, 113`，「{displayName}：{from} → {to}」）：一樣露出英文名與英文狀態值 → 改用 `SERVICE_LABELS` 的中文名與 pill 的中文狀態字。稿沒畫這條通知（碼→稿不需要：它是暫態提示，`spec-note-dsr-3c` 提一句即可）。

### 設計稿節點

| 代號 | 節點 | 說明 |
| --- | --- | --- |
| C8-D／C8-M | `wqcqY`／`qx8Ma` | 服務卡片（dev `Get` 抄名稱／說明／pill 字）、`h7Pjj`（刪「· Claude」）、`blwxn` 桌機橫幅、`yNVad` 手機橫幅（被切） |
| C15-D／C15-M | `XwdOH`／`wkUNt` | `UTfMO` skeleton-list、`lLgMu`／`f8FZa`（查是否註記） |
| C16-D | `uYGBU` | `eTUqu`／`M5NMY`／`wywZ6` |

---

## Acceptance Criteria

1. **設計稿先改（Pencil MCP）**：刪 `h7Pjj`「· Claude」；C8-M 每張卡補回重新檢查鈕（44×44）；`yNVad` 不再被切（page-body 高度或內容調整，`problems` 不增）；橫幅補畫「技術細節」收起列；規格註記 `spec-note-dsr-3c`：
   > 「服務狀態：服務名稱一律中文（前端對照，後端字串不出現在畫面上）。任一服務壞掉時，列表下方一條橫幅列出每個壞掉的服務與可以照做的建議；後端的原始錯誤收在『技術細節』裡。狀態變化的提示也用中文名。」
   收尾同 `dsr-3a` AC #1（只 stage `c8-d`／`c8-m`／有改到的 `c15-*`＋`pen-tokens.json`）。

2. **中文名稱對照**：🔴 #1、#8。`SERVICE_LABELS` 放 `components/settings/serviceLabels.ts`（新），卡片、橫幅、狀態變化通知**三處都吃它**。

3. **卡片重排**：🔴 #3。

4. **錯誤橫幅**：🔴 #4。橫幅 `role="alert"`（只在第一次出現壞掉的服務時宣讀——用 `aria-live="polite"` 的容器＋條件渲染內容即可，dev 選一種並寫進註解）；橫幅裡的「重新檢查」一次重測所有壞掉的服務（逐一呼叫既有的 `onTest(name)`，**不新增 API**）。

5. **載入骨架**：🔴 #6。

6. **整頁失敗**：🔴 #7。

7. **既有的行為不准回歸。**
   - 測試連線（單一服務）、輪詢、狀態變化偵測、`service-card-{name}`／`test-btn-{name}`／`last-check-{name}` testid 照舊。
   - `ServiceStatusCard.spec.tsx`／`ServiceStatusDashboard.spec.tsx` 的行為斷言不改；`detail-toggle-*`／`detail-panel-*` 相關條目例外（🔴 #4），Completion Notes 逐條列。
   - ⛔ 不改後端、`serviceStatusService.ts` 的型別與請求。

8. **測試。** 紅／守（Rule 16）。
   - `serviceLabels.spec.ts`（新）：已知 name 回中文名；未知 name 退回 displayName、無說明、無建議。
   - `ServiceStatusCard.spec.tsx`：（紅）名稱是中文、**畫面上不出現** `Douban Scraper`／`Wikipedia API`／`AI Parser`；五種狀態各自的 pill token（`connected`→`--success-tint` 等）；重新檢查鈕 accessible name「重新檢查 豆瓣」；沒有「顯示詳情」。（守）測試中轉圈、`onTest(name)`。
   - `ServiceStatusDashboard.spec.tsx`：（紅）有 error 服務時橫幅出現、每個壞掉的服務一行、qBittorrent 那行有連到 `/settings/connection` 的連結、技術細節預設收起且展開後含 mock 的後端錯誤字串；全部正常時沒有橫幅；載入中渲染 5 列骨架且 `aria-busy`；查詢失敗渲染 `SettingsErrorState` 文字逐字、**不含** mock 的 `error.message`、按重試呼叫 `refetch`；狀態變化通知用中文名。（守）既有輪詢與狀態變化偵測。
   - `Skeleton`：若 AC 🔴 #6 加了 `motion-reduce:animate-none`，補一條 class 斷言。
   - **視覺夾具**：既有 `settings-service-status-card`、`settings-service-status-dashboard` 基準**會變**（寫進 Completion Notes），`penNode` 改成 `'wqcqY'`；新增 `settings-service-status-dashboard/loading`（`penNode: 'XwdOH'`）、`/error`（`penNode: 'uYGBU'`）、`/mobile`（`width: 390`、`penNode: 'qx8Ma'`，含一個 error 服務）。
   - **e2e**（追加到 `tests/e2e/settings-shell.spec.ts`）：stub `/settings/services` 回一個 `error` 的 qBittorrent → 頁面上看得到「qBittorrent」與中文建議、點建議裡的連結到 `/settings/connection`；stub 500 → 看得到「無法載入服務狀態」與「重試」、**看不到**英文錯誤。（dev 先確認實際端點路徑，`serviceStatusService.ts`）
   - **每一項修法做 mutation check**，結果寫進 Completion Notes。

9. **CI 全綠**：同 `dsr-3a` AC #9。

## Tasks / Subtasks

- [x] **Task 1 — 設計稿：刪「· Claude」、手機補重新檢查鈕、修被切的橫幅、技術細節、規格註記（AC: #1）**
- [x] **Task 2 — `serviceLabels.ts`＋卡片／通知換中文名（AC: #2, #8）**
- [x] **Task 3 — 卡片重排：pill、回應時間、圖示重新檢查鈕（AC: #3, #8）**
- [x] **Task 4 — 錯誤橫幅＋技術細節＋一次重測（AC: #4, #8）**
- [x] **Task 5 — 載入骨架、整頁失敗換共用元件、Skeleton reduced-motion（AC: #5, #6, #8）**
- [x] **Task 6 — 夾具、e2e、mutation check、收尾（AC: #7, #8, #9）**
  - [x] dev-story Step 9：`c8-d`／`c8-m`／`c15-d`／`c15-m`／`c16-d`

## Dev Notes

### 這張的重點

- **對照表是唯一新增的「知識」**。名稱、說明、建議都在一個檔裡，三處渲染都吃它。
- **建議只寫前端能保證為真的話**。後端不會告訴你錯誤的類型，所以建議必須對「這個服務連不上」的任何原因都成立。寫不出來就不寫。
- **原始錯誤不刪，收起來**。它是使用者排錯的最後一條線索。

### 上游契約（Rule 20）

- 本張不定義也不消費任何 `[@contract-v*]`；只讀既有的服務狀態回應（implicit v0）。

### 建單裁定（2026-09-23，SM；Sally／Alexyu 可在 review 推翻）

1. ⚖️ 中文名稱前端對照、後端不改。
2. ⚖️ 稿刪「· Claude」（產品給不出來）。
3. ⚖️ 整合橫幅（稿→碼）＋技術細節收起（碼→稿）；每卡的「顯示詳情」退役。
4. ⚖️ 手機每張卡保留重新檢查鈕（碼→稿）。
5. ⚖️ `Skeleton` 補 reduced-motion（若 `styles.css` 沒處理）。

### 不要做的事

- 不要改後端的 displayName。
- 不要在建議裡猜錯誤原因（「金鑰錯誤」「網路逾時」這種要後端分類才能說的話）。
- 不要新增 API、不要改輪詢頻率。

### 已知陷阱

- **key 用 `service.name` 不是 `displayName`**：後端的 name 是 `ServiceNameDouban` 等常數字串（`models/degradation.go`），dev 先 grep 出實際值再寫對照表。
- **相對時間**：「檢查於 N 秒前」若讀現在時間，夾具會每天變——凍結時間或傳固定值（Rule 見下）。
- **Pencil／匯出／gh**：同 `dsr-3a`。

### Source tree

```
ux-design.pen、pen-tokens.json、screenshots/flow-c-search-settings/{c8-d,c8-m,c15-d,c15-m}.png   ← Task 1
apps/web/src/components/settings/serviceLabels.ts（新，+spec）                                   ← Task 2
apps/web/src/components/settings/ServiceStatusCard.tsx（+spec）                                  ← Task 2/3
apps/web/src/components/settings/ServiceStatusDashboard.tsx（+spec）                             ← Task 2/4/5
apps/web/src/components/ui/Skeleton.tsx（可能一行）                                              ← Task 5
apps/web/src/routes/test/-gallery.fixtures.tsx；tests/e2e/settings-shell.spec.ts（追加）         ← Task 6
```

### Cross-Stack Split Check

後端 task **0**、前端／設計／測試 task 6 → 不觸發。規模：兩個元件＋一個對照表＋一個骨架——比 `-3b` 小。

### Time-dependent visual coverage

- `ServiceStatusCard` 的「最後檢查」若是相對時間 → **是讀時間的元件**：夾具用固定的 `lastCheckAt` 並凍結 `Date.now()`（照 `-gallery.fixtures.tsx` 既有的凍結寫法），否則基準每天漂。dev 讀完 formatter 後在此處把 N/A 或做法寫進 Completion Notes。

### References

- [Source: `ServiceStatusCard.tsx:48-53, 79-132`；`ServiceStatusDashboard.tsx:17, 38-48, 76, 82-97, 113, 131-134`；`services/serviceStatusService.ts:9-28`；`components/ui/Skeleton.tsx`；`styles.css:300`]
- [Source: `apps/api/internal/models/degradation.go:250-257`]
- [Source: `ux-design.pen` `wqcqY`／`qx8Ma`／`h7Pjj`／`blwxn`／`yNVad`／`XwdOH`／`wkUNt`／`UTfMO`／`lLgMu`／`f8FZa`／`uYGBU`／`eTUqu`／`M5NMY`／`wywZ6` —— SM 建單時以 Pencil MCP 讀出（2026-09-23）]
- [Source: `dsr-3a-settings-shell-page-header-and-states.md`（拆單、`SettingsErrorState` API）；`project_pen_design_token_system`（狀態詞彙）；`project_tanstack_refetch_no_data_resets_pending`]

## Dev Agent Record

### Agent Model Used

Claude Opus 5.5 (1M context)（Amelia / dev-story）

### Debug Log References

- Pencil 裁切警告 67 → 66（少掉的是被切 28px 的手機橫幅 `yNVad`；剩下的 `CpTax` 是分頁列本來就該被切的捲動條）。存檔走 osascript 選單 Save，磁碟 grep 到 `spec-note-dsr-3c`。匯出 196/196，只 stage `c8-d`／`c8-m`／`c15-d`／`c15-m`＋`pen-tokens.json`。
- 本機 e2e／visual 自己起後端與 `nx serve web`；`nx test web` 全套會把兩個程序一起收掉，visual 前要重起（同 `dsr-3b`）。
- 手機視覺夾具被殼層固定的底部分頁列蓋住：gallery 的標題在手機把 fixture 推到 y≈420，5 張卡＋橫幅（708 高）一定壓到分頁列。`gallery-fixture-viewport.spec.ts` 規定手機夾具必須 390×844，所以不能加高視窗 → 手機夾具只放一張壞掉的 qBittorrent（卡＋橫幅＋技術細節＋重新檢查全部入鏡）。

### Completion Notes List

- Ultimate context engine analysis completed - comprehensive developer guide created
- **Task 1（稿）**：刪 `h7Pjj`「· Claude」；「TMDb」→「TMDB」（`xqzRF`，與 `dsr-3b` 品牌拼法一致）；桌機橫幅 `blwxn` 重畫成 `N28x1`（圖示｜一行「qBittorrent：目前無法連線。請確認…」＋收起的「技術細節」｜右側「重新檢查」）——原稿的「連線遭拒（ECONNREFUSED）」是後端分類過的原因，後端並不給，依「不要猜錯誤原因」拿掉；手機 5 張卡重建（每張補 44×44 重新檢查鈕，回應／最後檢查移到第二行），手機橫幅重建（同桌機內容），C8-M 高 844 → 1044，橫幅不再被切；`lLgMu`／`f8FZa` 查證是**畫面裡**的設計註記 → 刪掉，內容併入 `spec-note-dsr-3c`（`BrWLI`）。
- **Task 2**：`serviceLabels.ts`（新）——`SERVICE_LABELS`（key＝`service.name`：`tmdb`／`douban`／`wikipedia`／`ai`／`qbittorrent`，從 `models/degradation.go:29-33` grep 出來）、`getServiceLabel`（未知 name → displayName、無說明無建議）、`STATUS_LABELS`（pill 與狀態變化通知共用）、`isBroken`。卡片、橫幅、通知三處都吃它。
- **Task 3**：卡片重排——左名稱＋說明；中「回應 N ms／最後檢查 X」（等寬，手機 `order-last basis-full` 換到第二行）；pill 照狀態詞彙（`unconfigured` 改 `bg-tertiary`＋`text-muted`＋`CircleSlash`）；只有圖示的重新檢查鈕（`size-11 sm:size-9`、accessible name「重新檢查 {中文名}」、`test-btn-{name}` 保留、測試中 `motion-safe:animate-spin`）。每卡「顯示詳情」退役。
- **Task 4**：列表下方 `aria-live="polite"` 容器常駐、內容條件渲染（不用 `role="alert"`：它會在每 30 秒輪詢重繪時重念）；每個 `error`／`disconnected` 一行「{中文名}：目前無法連線。{建議}」，建議只有 qBittorrent（→ `/settings/connection`）與 TMDB（→ `/settings/keys`）；原始錯誤收進預設收起的 `<details>`「技術細節」；「重新檢查」逐一呼叫既有 `handleTest`，不新增 API。
- **Task 5**：載入 → `ServiceStatusSkeleton`（5 列、手機 64／桌機 72、右側 84×24＋桌機才有的 36×36、`aria-busy`＋`aria-label="載入中"`）；整頁失敗 → `SettingsErrorState`（C16 兩句逐字、`testId="status-error"`、不接 `error.message`），加 `retrying` 旗標讓「重試中…」真的看得到（TanStack 會把沒資料的失敗查詢退回 pending，同 `dsr-3b` CR #1）。**Skeleton 不改**：`styles.css:562-569` 的 reduced-motion 安全網已把所有動畫夾成 1ms×1 次，`animate-pulse` 停在不透明，AC 🔴 #6 的「沒有就加」不成立。
- **Time-dependent visual coverage**：`formatRelativeTime` 讀 `Date.now()` → 夾具沿用既有的 `JUST_NOW`／`MINUTES_AGO`／`HOURS_AGO`（放在時間桶正中），基準不會每天漂；不需要 `clockTime`。
- **既有 spec 改動（刻意退役／改字，逐條）**：
  - `ServiceStatusCard.spec`：`renders disconnected service with detail toggle`、`[P1] renders error status with detail toggle`、`[P1] shows lastSuccessAt in detail panel`、`[P1] collapses detail panel on second click`、`[P1] shows rate limited service with detail panel content`、`[P2] does not show detail toggle for connected service`、`renders unconfigured service without detail toggle` → 顯示詳情退役，改成「卡上沒有顯示詳情、不印後端錯誤」；錯誤內容與最後成功時間的斷言搬到 dashboard 的技術細節測試。`檢查於` → `最後檢查`、`尚未檢查` → `—`、`45ms` → `回應 45 ms`、`TMDb API`／`AI 服務` → 中文名（稿的字）。
  - `ServiceStatusDashboard.spec`：`[P2] shows error message text from API error` **反轉**成「不顯示」（C16 的目的）；`renders loading state` 改斷言 5 列＋aria；通知文字 `TMDb API：` → `TMDB：`。輪詢、狀態變化、測試連線、錯誤訊息等其餘行為斷言未改。
- 🔗 **AC Drift: FOUND**
  - `6-4-service-connection-status-dashboard` AC #2（點狀態看詳細錯誤＋最後成功時間）→ 錯誤與「最後成功」從每卡的顯示詳情搬到列表下方橫幅的「技術細節」；契約保留（加了一條測試釘住「最後成功」還在）。
  - `bugfix-settings-honest-readouts` AC #4（每種狀態都顯示新鮮度、沒有時寫「尚未檢查」）→ 新鮮度仍無條件顯示、testid 照舊；缺值字改成稿上的「—」（「最後檢查 尚未檢查」讀起來重複）。
- 📎 **Contract Stamps: NONE**（本張與上游皆無 `[@contract-v*]`；只讀既有服務狀態回應，implicit v0）。
- 🎭 **A11y Pre-Flight: PASS**（3 個元件；觸碰檔案 jsx-a11y 警告 0 條；圖示鈕有 accessible name、橫幅 polite live region 常駐、`<details>` 原生鍵盤可操作、骨架 `aria-busy`；沒有 modal／combobox）。
- **測試**：新 `serviceLabels.spec.ts` 9 條、`ServiceStatusDashboard.retry.spec.tsx`（真 QueryClient）1 條；`ServiceStatusCard.spec` 22 條、`ServiceStatusDashboard.spec` 27 條。`nx test web` **287 files／4203 tests 全綠**；`nx test api` 綠；`web:typecheck` 綠；touched 檔 eslint 0 error（35 條既有 `no-explicit-any`／1 條既有 `exhaustive-deps` 警告，main 上就有）；prettier 綠。e2e `settings-shell.spec.ts` 追加 3 條，chromium `--repeat-each=3` 整檔 **39／39**。
- **Mutation：unit 10／10 紅、e2e 2／2 紅**（對照表失效→18 紅、拿掉 retrying 旗標、rate_limited 算壞掉、技術細節預設展開、骨架 3 列、未設定 pill 用舊底色、通知用 displayName、一次重測只測第一個、按鈕名用 displayName、拿掉建議；e2e：桌機鈕回 44、拿掉建議）。
- **視覺基準**：`settings-service-status-card`（3 張 darwin）與 `settings-service-status-dashboard/default` 會變（penNode 改 `wqcqY`，dashboard 改成 C8-D 的五種狀態、寬 1152）；dashboard 的 hover／focus 基準刪除（`statesOnly: ['default']`，整頁夾具的 hover 沒意義）；新增 `/loading`（`XwdOH`）、`/mobile`（`qx8Ma`）。過期的 `-linux` 已 `git rm`，等 CI bootstrap（`project_visual_baseline_intentional_change`）。
- ⚠️ **與 story 字面的偏離**：① **沒有 `settings-service-status-dashboard/error` 夾具**——dashboard 的錯誤狀態就是 `SettingsErrorState` 帶這兩句話，而 `dsr-3a` 的 `settings-error-state` 夾具已經以同樣兩句、`penNode: 'uYGBU'`、720 寬拍成基準；再拍一張一模一樣的只是重複（要讓 dashboard 真的進錯誤狀態還得讓查詢失敗，gallery 沒有這種 seed）。② 手機夾具只有一張卡（見 Debug Log）。③ 手機橫幅的「重新檢查」是 44 高（稿 32）——手機觸控高度，跟每張卡的 44 鈕一致。
- 本機 visual 另有一張 `retry-retry-notifications/default-visual-darwin.png` 漂移（main 上就有、與本張無關、只影響 darwin；CI 用 linux）——未收進本張。

### File List

- `ux-design.pen`
- `_bmad-output/pen-tokens.json`
- `_bmad-output/screenshots/flow-c-search-settings/c8-d.png`
- `_bmad-output/screenshots/flow-c-search-settings/c8-m.png`
- `_bmad-output/screenshots/flow-c-search-settings/c15-d.png`
- `_bmad-output/screenshots/flow-c-search-settings/c15-m.png`
- `apps/web/src/components/settings/serviceLabels.ts`（新）
- `apps/web/src/components/settings/serviceLabels.spec.ts`（新）
- `apps/web/src/components/settings/ServiceStatusCard.tsx`
- `apps/web/src/components/settings/ServiceStatusCard.spec.tsx`
- `apps/web/src/components/settings/ServiceStatusDashboard.tsx`
- `apps/web/src/components/settings/ServiceStatusDashboard.spec.tsx`
- `apps/web/src/components/settings/ServiceStatusDashboard.retry.spec.tsx`（新）
- `apps/web/src/routes/test/-gallery.fixtures.tsx`
- `tests/e2e/settings-shell.spec.ts`
- `tests/visual/components.visual.spec.ts-snapshots/components/settings-service-status-card/*-visual-darwin.png`（3 改）、`*-visual-linux.png`（3 刪）
- `tests/visual/components.visual.spec.ts-snapshots/components/settings-service-status-dashboard/default-visual-darwin.png`（改）、`hover|focus-visual-darwin.png`（刪）、`*-visual-linux.png`（3 刪）
- `tests/visual/components.visual.spec.ts-snapshots/components/settings-service-status-dashboard/loading/default-visual-darwin.png`（新）
- `tests/visual/components.visual.spec.ts-snapshots/components/settings-service-status-dashboard/mobile/default-visual-darwin.png`（新）
- `_bmad-output/implementation-artifacts/sprint-status.yaml`
- `_bmad-output/implementation-artifacts/6-4-service-connection-status-dashboard.md`（AC drift reference — see Completion Notes）
- `_bmad-output/implementation-artifacts/bugfix-settings-honest-readouts.md`（AC drift reference — see Completion Notes）

### UX Verification（dev-story Step 9）

| Area | Design Spec | Implementation | Match? | Fix Needed |
|------|------------|----------------|--------|------------|
| 卡片（C8-D） | 名稱 14/600＋說明 12 muted｜等寬 meta 右對齊｜pill｜36 圖示鈕 | 同 | ✅ | — |
| pill 顏色 | 青碧／赭／硃砂／中性＋圖示 | 同（5 種狀態各有測試） | ✅ | — |
| 錯誤橫幅（C8-D） | 列表下方、error-tint、一行建議＋技術細節＋右側重新檢查 | 同（「連線設定」加底線連結） | ✅ | — |
| 手機卡（C8-M） | 44 鈕、meta 第二行 | 同 | ✅ | — |
| 手機橫幅 | 圖示＋建議＋技術細節＋重新檢查 | 同，鈕高 44（稿 32，觸控高度） | ⚠️ 刻意 | 見偏離 ③ |
| 骨架（C15-D／M） | 5 列、72／64、160×18＋240×14、84×24、36×36 只在桌機 | 同 | ✅ | — |
| 整頁失敗（C16-D） | 圖示＋「無法載入服務狀態」＋說明＋重試 | `SettingsErrorState` 逐字 | ✅ | — |

🎨 UX Verification: PASS — 截圖 `c8-d`／`c8-m`／`c15-d`／`c15-m`／`c16-d` 對照視覺基準逐項比對。

## Change Log

| Date | Change |
| --- | --- |
| 2026-09-23 | Task 1：設計稿——刪「· Claude」、TMDB、橫幅改保證為真的建議＋技術細節＋重新檢查、手機每卡 44 鈕、C8-M 加高、骨架註記移進 `spec-note-dsr-3c` |
| 2026-09-23 | Task 2–3：`serviceLabels.ts`；卡片中文名、pill、meta、圖示鈕；顯示詳情退役 |
| 2026-09-23 | Task 4：列表下方錯誤橫幅（建議、技術細節含最後成功、一次重測） |
| 2026-09-23 | Task 5：五列骨架、整頁失敗換 `SettingsErrorState`＋重試中旗標；Skeleton 不改（styles.css 已處理 reduced-motion） |
| 2026-09-23 | Task 6：夾具（card／dashboard 改、loading／mobile 新）、e2e 3 條、mutation 12／12、全套 web 4203 綠 |
