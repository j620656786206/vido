# Story 13.2b: 部分請求（選季/選集）—— 前端：L3 季/集樹選取器與 TV 請求流接線

Status: done

**Depends on:** `13-2a-partial-request`（backend API must be ready — selection wire 形狀、coverage 端點、season 路由）

**Epic:** `epic-13`（Request System，G-2/P3-002，artery #4）· **Risk: 🟡 MED（Winston 點名的最高設計風險面——L3 樹）** · **FRONTEND-ONLY（cross-stack 拆分之 b 半）**
**Source:** `epics/epic-13-request-system.md` §13-2 · 設計 13-0（GATE A 已過：L3-D-v2 已繪，Sally MCP review PASS 2026-07-04）

---

## Story

As a user on a TV show's detail page,
I want a season/episode tree to pick exactly which seasons or episodes to request,
so that I only ask Vido to acquire what I'm missing — with what I already own or requested clearly reflected.

---

## 🎨 設計（GATE A 已過）

- **主 frame：`L3-D-v2`（node `He04g`）** —— 部分請求季/集樹 dialog。13-0 已確立：整部影集 master checkbox（Indeterminate 狀態）、season 列可展開至集、**已入庫/已請求 badge＋disabled checkbox**（`Checkbox` 元件的 `Indeterminate`/`DisabledChecked`/`DisabledEmpty` 三態 2026-07-04 已元件化並在 L3 換成 instances）、footer 選取摘要＋取消/確認請求、44px touch floor、數字一律 JetBrains Mono（Rule TY-3）。
- 相關 frames：`L2-D-v2`（`VH3Tq`，想要按鈕三態——13-1b 已實作）、`L4-M-v2`（`n7isVa`，mobile）、`L8`（toast——13-1b 已實作）。
- 截圖：`_bmad-output/screenshots/flow-l-requests-v2/l3-d-v2.png`（低解析——**實作時 MUST 以 Pencil MCP 讀 `He04g` 取得精確 spec**；authoring 時 Pencil.app 未開啟，本 story 記載的是 13-0 story 文字規格＋截圖可辨結構）。
- Step 9 UX verification 對照 flow-l 截圖為 MANDATORY。

---

## Acceptance Criteria

### AC #1 — SeasonEpisodeTree dialog（L3-D-v2，He04g）

- 新元件 `components/requests/SeasonEpisodeTreeDialog.tsx`（`// Design ref:` header 指向 He04g，Rule 21）：Base UI dialog primitive（13-0 規範）、v2 tokens only、N4 四態（loading skeleton／error／empty=無季資料／content）。
- 樹結構：頂部「整部影集」master checkbox（全選/全不選/Indeterminate）→ season 列（checkbox＋`第 N 季`＋集數 count，Mono 數字）→ 展開後 per-episode checkbox 列。
- 集清單 **lazy per-season 載入**（展開才打 `GET /tmdb/tv/:id/season/:n`——13-2a 補登的路由；TanStack Query per-season key，Rule 5）。
- **已入庫 → `DisabledChecked` 樣式＋$success 語彙 badge；已請求 → disabled＋$info 語彙 badge**（資料源 AC #2 coverage）。已入庫/已請求列不可選取——確保送出的 selection 不含重疊（13-2a AC #3 的 400 在正常流程不會發生）。
- footer：已選摘要（`N 季 · M 集` Mono）＋「取消」＋「確認請求」primary（無選取時 disabled）；Escape 關閉＋initial focus（11-2 CR a11y 先例）；`aria-modal` focus trap。
- 選取語意與 13-2a canonical 一致：整季勾選 → `seasons`；季內部分勾選 → `episodes[season]`；全部勾選 → whole-series（**不帶** selection——與一鍵 想要 同款 create）。

### AC #2 — coverage 反映（confirmed against 13-2a AC #5 `[@contract-v1]`）

- 開啟 dialog 時打 `GET /api/v1/requests/tv/:tmdb_id/coverage` → owned/requested per-episode 反映到樹（badge＋disabled）；`whole_series_requested` 或 `active_request` 為 true → 樹不應被開啟（防 race/直接 URL：dialog 顯示「已有進行中的請求」狀態而非樹，附 查看清單 連結——⚖️ A 裁定 v1 不追加）。
- coverage 失敗 → fail-soft：樹照開、無 badge（請求送出時 13-2a 後端 guard 仍把關）——log console.warn 級即可。
- 整季「已入庫」的判定：該季全部集 owned → season 列整列 DisabledChecked。

### AC #3 — TV 請求流接線（confirmed against 13-2a AC #1 `[@contract-v1]`）

- **detail page（`TMDbDetailV2`）的 TV 想要按鈕 → 開樹 dialog**（取代直接 create）；movie 與卡片 contexts（PosterCard/MediaGrid/Discover）維持既有 one-click whole-title 行為零改動。⚠️ 實作時以 MCP 讀 L2/L3 確認此互動邊界與 13-0 意圖一致；若設計另有指示（如卡片 TV 也開樹），以設計為準並記錄。
- `requestService.createRequest` 加 optional `seasons`/`episodes`（camelToSnake，Rule 18）；`useRequestActions` optimistic 更新照舊（selection 不改變 requested-state 快取語意——⚖️ A：任何 active request 都讓按鈕變 pill）。
- 確認請求成功 → 既有 L8 toast（已加入想要清單＋查看清單）；`REQUEST_DUPLICATE` → 沿用 13-1b 靜默 settle 慣例；`REQUEST_INVALID_SELECTION`/`REQUEST_ALREADY_IN_LIBRARY` → error toast 顯示後端 zh-TW message。

### AC #4 — requests list 的部分範圍顯示

- `RequestRow` 對帶 selection 的 request 顯示範圍摘要（如 `第 1 季`、`第 2 季 · 3 集`，Mono 數字）；whole-series 列零改動。⚠️ 實作時 MCP 讀 L5/L6/L7 frames 確認設計是否已畫此摘要——**有畫照畫；沒畫則以最小文字列（text-secondary 一行）實作並在 Completion Notes 記錄設計缺口**（Rule 24 ③ 候選，交 Sally 補圖決定）。

### AC #5 — 測試與範圍圍籬

- 元件 spec：樹三態勾選邏輯（master/season/episode 級聯與 Indeterminate）、lazy season 載入、coverage badge/disabled、selection→payload 映射（含「全勾 = whole 不帶 selection」紅線）、active_request 擋樹、error codes 分流。Rule 16 斷言紀律。
- **不改**：13-1b 按鈕三態語意、13-3b RequestsView/SSE、13-7b action-area（ready-for-dev 中，勿碰其檔案）、Discover reserved entries。
- jsx-a11y：本 story 新增元件零新警告（A11y pre-flight，dialog 屬 aria-modal 四類檢查全跑）。
- 全回歸閘門：`pnpm nx test web`、`go test ./...`、`pnpm run lint:all`、`format:check`＋Step 9 UX verification（l3-d-v2 對照）。

---

## Tasks / Subtasks

- [x] **Task 1 — SeasonEpisodeTreeDialog 元件（AC: #1）** 🎨 FE
  - [x] MCP 讀 `He04g` 精確 spec → dialog＋樹＋checkbox 三態級聯＋footer；N4 四態；a11y（Escape/initial focus/trap）
  - [x] lazy per-season episodes（TanStack Query＋新 season 端點 service 方法）
- [x] **Task 2 — coverage 反映（AC: #2）** 🎨 FE
  - [x] `requestService.getCoverage`＋hook；owned/requested badge＋disabled；active → 擋樹視圖；fail-soft
- [x] **Task 3 — TV 請求流接線（AC: #3）** 🎨 FE
  - [x] TMDbDetailV2 TV 按鈕 → 開樹；`createRequest` selection 參數（camelToSnake）；error-code 分流＋toast
- [x] **Task 4 — RequestRow 範圍摘要（AC: #4）** 🎨 FE
  - [x] MCP 讀 L5/L6/L7 確認設計 → 摘要顯示；設計缺口時最小實作＋記錄
- [x] **Task 5 — 測試＋a11y＋全回歸＋UX verification（AC: #5）** 🎨 FE
  - [x] 元件 spec 全套（含 whole=無 selection 紅線）；jsx-a11y；全回歸；Step 9 對照 flow-l

（前端 task 5 個、後端 0 個 —— a 半見 `13-2a-partial-request`。）

---

## Dev Notes

### 既有可重用零件（不要重造）

| 需求 | 現成零件 |
| --- | --- |
| 按鈕三態＋toast | `RequestButton.tsx`（13-1b）—— 樹只接手 detail-TV 的「可請求」分支 |
| API client 慣例 | `requestService.ts`（RequestApiError code 分流、camelToSnake、[@contract-v1] ack 註解樣式） |
| requested/owned 快取 | `useRequestedMedia`/`useOwnedMedia` hooks（13-1b） |
| dialog 先例 | 13-0 規範 Base UI primitives；11-2/11-3 CR 的 a11y 四類（focus trap/Escape/initial focus/aria） |
| checkbox 三態 | `.pen` `Checkbox` 元件 `Indeterminate`/`DisabledChecked`/`DisabledEmpty`（13-0 已元件化）——FE 對應實作跟隨 |
| TMDB types | `types/tmdb.ts` ApiResponse＋season types（如缺 SeasonDetails type 則補，Rule 18 邊界轉換） |

### 關鍵決策（authoring 已裁）

- **⚖️ A（Alexyu 2026-08-19）**：v1 不追加——active request 存在時按鈕已是 pill 不會開樹；coverage 的 requested 反映主要防 race/直接觸發，仍照設計畫。
- **全勾 = whole-series 不帶 selection**：讓「開樹全選確認」與「一鍵想要」在 wire 上等價（後端單一 whole 語意，資料不分裂）。
- **部分重疊防線在 FE disable、後端 400 兜底**：樹永不讓使用者選到已擁有/已請求的列。
- **設計缺口（L5-L7 無範圍摘要時）走最小實作＋記錄**：不自行發明視覺（identical-rendering=Sally's-decision 的精神），缺口交 Sally。

### 契約姿態（Rule 20）

- **消費（實作時 verbatim 記錄）**：confirmed against `[@contract-v1]` (13-2a AC #1 selection 形狀)；confirmed against `[@contract-v1]` (13-2a AC #5 coverage 形狀)；confirmed against `[@contract-v1]` (13-1a AC #2/#3 resource shape——seasons/episodes 開始有值)。
- 產生：無新 stamp（純消費端）。

### Time-dependent visual coverage

`N/A — 新元件無 wall-clock 讀取（Rule 23 trigger 不成立）。`

### References

- [Source: `_bmad-output/implementation-artifacts/13-0-requests-design.md` L3 段] — 樹 spec 文字（granularity／already-owned/requested 反映／confirm）
- [Source: `_bmad-output/screenshots/flow-l-requests-v2/l3-d-v2.png`] — 結構佐證（低解析，MCP 讀 He04g 為準）
- [Source: `apps/web/src/components/requests/RequestButton.tsx`] — 三態＋toast＋guard 慣例
- [Source: `apps/web/src/services/requestService.ts`] — client 慣例＋[@contract-v1] ack 樣式
- [Source: `apps/web/src/components/media/TMDbDetailV2.tsx:84`] — detail 按鈕接點
- [Source: `_bmad-output/implementation-artifacts/13-2a-partial-request.md` AC #1/#5] — 上游契約
- [Source: `project-context.md`] — Rule 5/16/18/21/26

### Previous Story Intelligence（13-1b／13-3b）

- Portal-to-body：卡片 context 的 fixed 定位元素會被 clip-path/transform 容器困住（13-1b CR H1）——dialog 若在卡片 context 觸發需同款 portal。
- `RequestApiError` code 分流是 13-1b 特意建的 seam——新 error codes 走同一條。
- 13-3b 目前 status=review——其 RequestsView/RequestRow 檔案可能仍在收 CR 修正；Task 4 動 `RequestRow` 前先確認 13-3b 已 done（未 done → 與其 owner 序列化，避免同檔衝突）。

---

## Dev Agent Record

### Agent Model Used

Claude Opus 5.5（2026-10-02）

### Debug Log References

### Completion Notes List

- **Contract acks**：confirmed against [@contract-v1]（13-2a AC #1 selection 形狀：`seasons` int 陣列、`episodes` 以季號字串為鍵）；confirmed against [@contract-v1]（13-2a AC #5 coverage 形狀）；confirmed against [@contract-v1]（13-1a AC #2/#3 resource——`seasons`/`episodes` 為 JSON 文字）。
- **設計對照**：Pencil MCP 讀 `He04g`（modal 560 寬、header padding [20,16,12,24]、樹 padding [8,12] gap 2、列 44px、集列左縮 48、「顯示其餘 N 集」左縮 80、footer padding [16,24]）與五個 Checkbox 元件（`4EHFN`/`Wd9AL`/`NfHDL`/`Fn5MZ`/`VSXl5`）。新增 `ui/CheckboxBox.tsx` 畫這五態，語意由同列的原生 `<input type="checkbox">`（sr-only，indeterminate 以 effect 設定）承擔——RadioDot 先例。
- **選取模型** 抽到 `utils/requestSelection.ts`（純函式、單元測試）：季分「整季」與「部分集」兩種；整季不需要知道集號（可能從沒展開過），只有「整季但內含已入庫／已請求集」送出前才補抓該季集號，改送逐集。**紅線**：沒有任何鎖定且全部整季 → 不帶 selection（與一鍵想要同一個 wire）。season 0（特別篇）與 0 集的季不列。
- **lazy 載入**：展開才打 `GET /tmdb/tv/:id/season/:n`（key `['tmdb','tv',id,'season',n]`，10 分鐘）。coverage key 掛在 `requestKeys.all` 底下（`['requests','coverage',id]`，30 秒），任何建立／取消／重試 invalidate 都會順帶刷新；SSE 合併只動 `['requests','list']`，不受影響。
- **⚖️ A**：coverage 顯示有進行中的請求 → 不開樹，顯示「這部影集已有進行中的請求」＋查看清單；此時不顯示 footer。coverage 失敗 → 照開、無徽章、`console.warn`（fail-soft）。
- **AC #3**：`RequestButton` 新增 `pickEpisodes`，只有詳情頁（`TMDbDetailV2`）的影集帶；卡片與電影維持一鍵。樹的「確認請求」→ 關閉對話框 → 走原本的 `create`（toast、`REQUEST_DUPLICATE` 靜默、其他錯誤顯示後端中文訊息）。optimistic 列直接帶同樣的 JSON 文字，範圍標籤立即出現。
- **AC #4**：L1／L5–L7 沒畫範圍 → 最小實作：meta 列加「第 2 季 · 第 1 季 3 集」（數字 Mono），已立案 `disc-2026-10-request-row-range-undrawn`。
- **手機**：沒有 L3-M 稿，沿用置中對話框（寬 = 螢幕減 2rem、最高 85vh、樹區可捲），390px 截圖基準有收；已立案 `disc-2026-10-season-tree-mobile-undrawn`。
- **瀏覽器實測**（`scripts/serve-test-env.sh`＋Playwright 攔截 TMDb／coverage／POST）：詳情頁按想要 → 樹開、焦點落在對話框內 → 展開 S1、勾 E03–E05、勾整季 S2 → 摘要「已選 1 季 · 3 集」→ 確認送出 `{"tmdb_id":1429,"media_type":"tv","seasons":[2],"episodes":{"1":[3,4,5]}}` → 對話框關閉、toast「已加入想要清單」；390px 打開、Esc 關閉。
- **⚠️ 重大範圍限制（已立案，需 Alexyu 裁定）**：樹只從「想要」按鈕打開，而**只要本機有這部影集的任何一集**，詳情頁就顯示「已入庫」膠囊、沒有按鈕；媒體庫詳情頁（LocalDetailV2）也沒有請求入口。所以「有第 1 季、想補第 2 季」這個最主要的用途目前**打不開樹**，樹只對完全沒有的影集有用。後端早就支援。依故事範圍（不改 13-1b 三態）沒做，立案 `disc-2026-10-partial-request-entry-for-owned-series`（P2）。gallery 的「已入庫集鎖定」基準就是為那張單先收的狀態。
- **Adversarial CR（2026-10-02，獨立 agent，含暫時 probe spec）**：
  - ✅ 修：部分選取在「該季列出的集全部已鎖定」時會變成空 selection，而空 selection 在 wire 上等於整部 → 現在對話框直接擋下並提示「選到的集數都已入庫或已請求」。
  - ✅ 修：送出時正在補抓集號，使用者按取消／Esc，請求仍會在背景送出 → 卸載後不再送。
  - ✅ 修：`{open && …}` 讓關閉動畫期間對話框是空的 → 拿掉，交給 Radix 在動畫結束後卸載。
  - ✅ 修（加固）：範圍標籤加 `min-w-0` 才能真的截斷；`episodes` 為 JSON `null` 時不再讓清單崩潰；背景 refetch 失敗但有快取時不再顯示錯誤畫面；樹的捲動區加 `relative`。
  - 📝 不修：確認後焦點回到 body（觸發按鈕變成膠囊，沒地方回）；建立失敗時對話框已關，要重選；鎖定數算進 TMDb 沒列出的集號（絕對集數動畫）會讓季誤顯示為已入庫——三者都只在上面那個入口缺口解決後才常見，併入 `disc-2026-10-partial-request-entry-for-owned-series` 一起處理。

### Discovery Triage

- **Did this story discover any work outside its current scope?**
  - 見 13-2a（⚖️ A 裁定 → `disc-2026-07-arr-already-exists-loop` 擴充）。
  - **③ `disc-2026-10-partial-request-entry-for-owned-series`**（P2，dev 時發現）：已入庫影集打不開樹。
  - **③ `disc-2026-10-request-row-range-undrawn`**（P3）：範圍標籤的最小實作待 Sally 定稿。
  - **③ `disc-2026-10-season-tree-mobile-undrawn`**（P3）：手機版樹沒有稿。

### File List

- apps/web/src/components/requests/SeasonEpisodeTreeDialog.tsx（新）、SeasonEpisodeTreeDialog.spec.tsx（新）
- apps/web/src/components/ui/CheckboxBox.tsx（新）
- apps/web/src/utils/requestSelection.ts（新）、requestSelection.spec.ts（新）
- apps/web/src/components/requests/RequestButton.tsx、RequestButton.spec.tsx
- apps/web/src/components/requests/RequestRow.tsx、RequestRow.spec.tsx
- apps/web/src/components/media/TMDbDetailV2.tsx
- apps/web/src/hooks/useRequestActions.ts
- apps/web/src/services/requestService.ts、requestService.spec.ts
- apps/web/src/services/tmdb.ts、apps/web/src/types/tmdb.ts
- apps/web/src/routes/test/-gallery.fixtures.tsx（4 個新 fixture）
- tests/visual/…/request-season-tree/{content,content-mobile,active}/、request-row/partial/ 的 darwin 基準（linux 待 CI bootstrap）

---

## Change Log

| Date | Change |
| --- | --- |
| 2026-10-02 | dev 完成（Opus 5.5）→ review；adversarial CR 修 4 項＋加固 4 項；立案 3 筆設計／入口缺口。 |
| 2026-08-19 | create-story：Epic 13 artery #4 之 b 半（FE）。GATE A 已過（L3-D-v2 已繪＋Sally review PASS）；authoring 時 Pencil 未開啟——實作時 MUST MCP 讀 He04g。⚖️ A 裁定納入（active 擋樹視圖、requested 反映防 race）。「全勾=whole 不帶 selection」wire 等價裁定。依賴 13-2a（selection/coverage/season 路由三契約）。 |
