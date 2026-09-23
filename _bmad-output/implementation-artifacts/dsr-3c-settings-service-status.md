# Story DSR.3c：服務狀態頁對齊設計稿——服務名稱說中文、載入時有骨架、整頁失敗可以按重試、壞掉的服務告訴你該去哪裡修

Status: ready-for-dev

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

- [ ] **Task 1 — 設計稿：刪「· Claude」、手機補重新檢查鈕、修被切的橫幅、技術細節、規格註記（AC: #1）**
- [ ] **Task 2 — `serviceLabels.ts`＋卡片／通知換中文名（AC: #2, #8）**
- [ ] **Task 3 — 卡片重排：pill、回應時間、圖示重新檢查鈕（AC: #3, #8）**
- [ ] **Task 4 — 錯誤橫幅＋技術細節＋一次重測（AC: #4, #8）**
- [ ] **Task 5 — 載入骨架、整頁失敗換共用元件、Skeleton reduced-motion（AC: #5, #6, #8）**
- [ ] **Task 6 — 夾具、e2e、mutation check、收尾（AC: #7, #8, #9）**
  - [ ] dev-story Step 9：`c8-d`／`c8-m`／`c15-d`／`c15-m`／`c16-d`

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

### Debug Log References

### Completion Notes List

- Ultimate context engine analysis completed - comprehensive developer guide created

### File List
