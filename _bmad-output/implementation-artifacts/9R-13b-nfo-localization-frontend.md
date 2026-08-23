# Story 9R-13b: .nfo 在地化的前端入口（FE 落地）

Status: done  <!-- 2026-08-22 CR（Fable 5，換模型）：2H/4M/1L 全修，+6 測試（共 41）。閘門全綠。 -->

**Epic:** epic-9R-subtitle-route-c · **Priority:** P1（differentiator 的最後一哩 —— 後端出貨兩輪、使用者零觸及）
**Created:** 2026-08-21（SM Bob, create-story）
**Source:** Rule 24 lane ③ `backlog-nfo-localization-frontend-entry`（由 `9R-13a` dev-story 立案）。
🔓 **UNBLOCKED 2026-08-21** —— `9R-UX-nfo-localization-entry-design` **done**（Sally MCP review PASS）。
⚠️ **dev 開工前先確認設計的 PR 已 merge**（Step 9 的 UX 截圖比對需要 `flow-b-detail-v2/b3p-d.png`・`b4p-d.png`・`b3p-m.png` 與 `flow-j-specs/j6-d.png` 在 main 上）。
⚖️ **依 Alexyu 2026-08-20 於 `9R-10b` 的裁定先例**：設計先行，不走 `// Design ref: PENDING` 暫掛。
**Depends:** 9R-13 ✅（movies 後端）、9R-13a ✅（TV 後端）。

---

## Story

As a NAS owner whose player shows English metadata,
I want a button on the detail page that localizes this title's metadata to zh-TW,
so that the differentiator the backend has shipped twice is something I can actually reach — and when it replaces a file, I knew before I pressed it.

---

## 🚨 為什麼這張存在

`grep -rn "localize-nfo" apps/web/src` → **空**。

| 出貨 | 後端 | 前端 |
|---|---|---|
| 9R-13（2026-07） | `POST /movies/:id/localize-nfo` | ❌ |
| 9R-13a（2026-08-21） | `POST /series/:id/localize-nfo[?include_episodes]`、`POST /episodes/:id/localize-nfo` | ❌ |

三條路由**只能用 curl 打**，已經兩個月。

---

## ⛔ 為什麼現在是 `blocked` 而不是 `ready-for-dev`

1. **Rule 21 是 ESLint 強制的**（`local/implements-pen-node-id` 在 `pnpm lint:all` ⇒ CI）。新元件沒有 `.pen` 節點就沒有合法的檔頭。
2. **`ux-design.pen` 完全沒有 nfo 相關畫面**（實證：`scripts/export-pen-screenshots.py` 內 `grep -i nfo` 為空；`flow-b-detail-v2` 只有 b3p/b4p/b6p/b7p/b8p）。
3. **有真正的產品決策要裁**：覆寫確認的逐字文案、電影 vs 影集要不要長得不一樣、`include_episodes` 的預設值（會影響花費）。這些不是工程細節。

⇒ 設計輪先跑。**本 story 的視覺 AC（#2/#3/#4）在設計 done 之後才定稿。**

---

## 🔎 現況查證 @ `3505a153`（開工前必讀）

| 事實 | 位置 | 意義 |
|---|---|---|
| 🔴 **v1 詳情頁是死程式碼** | `src/routes/media/$type.$id.tsx:77` **無旗標**直接渲染 `LocalDetailV2` | `MediaDetailPanel` / `DetailPanelMenu`（`Screen 4c`）**沒有任何正式路由在用**（只剩 dev-only 的 `/test` 展示廊與一支測試引用，兩者檔名皆以 `-` 開頭＝TanStack Router 排除在路由之外）⇒ **不要往那裡加東西**：加了不會顯示，但元件測試會過 |
| v2 動作列 | `LocalDetailV2.tsx:147-184` | 目前三個：管理字幕（primary，需 `filePath`）／修改資訊／複製路徑（icon-only）。**無 overflow 選單** |
| v2 設計錨點 | `LocalDetailV2.tsx:1` | `Component/Detail-Movie-v2 (uRGu2)` + `Component/Detail-TV-v2 (N2fmG6)` |
| 對話框語彙 | `ui/Dialog.tsx`、`library/BatchConfirmDialog.tsx` | 可抄；`DetailPanelMenu` 的 inline confirm **不可**當先例（死程式碼） |
| FE service 慣例 | `src/services/*.ts` | 新增 nfo service 應鏡射既有樣式 |
| 資料抓取 | Rule 5：TanStack Query 管所有 server state | mutation 走 `useMutation`，成功後 invalidate 對應的 media detail query |

---

## Acceptance Criteria

### AC #1 — Service 層（可先於設計完成，但不先合併）

1.1 新增 `nfoLocalizerService`（或併入既有 media service，依 repo 慣例）暴露三個呼叫：
- `localizeMovieNfo(id)` → `POST /movies/:id/localize-nfo`（**無 body**）
- `localizeSeriesNfo(id, { includeEpisodes })` → `POST /series/:id/localize-nfo[?include_episodes=true]`，body `{ confirm_replace: true }`
- `localizeEpisodeNfo(id)` → `POST /episodes/:id/localize-nfo`，body `{ confirm_replace: true }`

1.2 回應型別（**照後端形狀，不要自創**）：
```ts
type NfoLocalizeResult = { path: string; backup_path: string; replaced: boolean };
type NfoSeriesLocalizeResult = {
  show: NfoLocalizeResult;
  episodes: NfoLocalizeResult[];
  succeeded: number; failed: number; skipped: number;
};
```

1.3 錯誤碼對映（Rule 3 信封 `error.code`）：

| 碼 | HTTP | 使用者看到什麼 |
|---|---|---|
| `NFO_REPLACE_NOT_CONFIRMED` | 409 | 🔴 **不該發生** —— FE 一律帶 `confirm_replace: true`。若出現代表 FE 有 bug，要記可診斷的錯誤而不是靜默 |
| `NFO_LOCALIZE_DISABLED` | 503 | 「尚未設定翻譯服務」→ 引導去金鑰設定 |
| `NFO_LOCALIZE_FAILED` | 500 | 一般失敗訊息 |
| `VALIDATION_REQUIRED_FIELD` | 400 | 「請先掃描媒體庫」 |

### AC #2 — 入口（✅ 設計已定稿 2026-08-21）

**Sally 裁定（`J6-D` 與三個詳情 frame 已落地）：**
- 動作列**四顆、順序固定**：`管理字幕`(primary) → `修改資訊` → **`在地化資訊`**(NEW) → `複製檔案路徑`(icon-only)。
- 新按鈕：lucide **`languages`** 圖示 + 文字**「在地化資訊」**，secondary 樣式（與「修改資訊」同一種）。
- 🔴 **不引入 overflow `⋯`** —— 會覆寫檔案的動作不該藏進選單。
- **手機（`< sm`）改成兩排、每排兩顆、四顆等寬且都有文字標籤**：
  上排 `管理字幕｜在地化資訊`、下排 `修改資訊｜複製路徑`。**不得壓成無標籤圖示。**
- 電影與影集的按鈕**完全一致**（同位置、同圖示、同文字），差異全在對話框。
- 📌 前端 `lucide-react` 匯入 `Subtitles`；設計檔用的是 `captions` —— **同一個圖示的新舊名稱，不是不一致**。

**Rule 21 檔頭**（screen frame，用 `Design ref:` 文法而非 `Implements: Component/`）：
```
// Design ref: ux-design.pen Screen B3p-D (uRGu2) + Screen B4p-D (N2fmG6) + Screen B3p-M (SzNRb)
```
確認對話框元件則指向 spec 畫面：
```
// Design ref: ux-design.pen Screen J6-D (zMYsL)
```

### AC #2b — 原本的入口驗收條款

2.1 依 `9R-UX-nfo-localization-entry-design` 的裁定，在 `LocalDetailV2` 的動作區加入在地化入口。
2.2 🔴 **不得**加到 `MediaDetailPanel` / `DetailPanelMenu`（死程式碼）。
2.3 🔴 Rule 21：任何新元件檔頭必須帶設計 story 產出的 `// Implements: Component/{Name} ({nodeId})`。**禁止**用 `Design ref: — no current screen frame` 逃生門（本 story 存在的理由就是不走那條）。
2.4 入口僅在 `filePath` 存在時顯示（與「管理字幕」「複製路徑」同一條件 —— 沒有檔案就沒有地方放 `.nfo`）。

### AC #3 — 確認對話框（✅ 逐字文案已定稿 2026-08-21，截圖 `flow-j-specs/j6-d.png`）

🔴 **Sally 推翻了原本的 AC #3.3：電影也要確認對話框。**
理由：**電影一樣會花錢**（LLM 翻譯）。依 2026-08-19「花錢須同意」裁定，一鍵、無提示、開始計費不可接受。

**共用標題**：`將資訊在地化為繁體中文`

**電影版**（additive，不覆寫）
| 元素 | 逐字文案 |
|---|---|
| 內文 | `Vido 會用 AI 把片名、劇情與角色名翻成繁體中文，寫成播放器讀得到的 .nfo 檔。` |
| 綠色安心區（`--success-tint`／`shield-check`） | `不會覆寫你現有的 .nfo —— 會寫進另一個播放器同樣認得的檔名` |
| 藍色成本區（`--info-tint`／`sparkles`） | `會使用 AI 翻譯額度` |
| 主鍵 / 次鍵 | `開始在地化` / `取消` |

**影集版**（單槽，一定覆寫）
| 元素 | 逐字文案 |
|---|---|
| 內文 | `Vido 會用 AI 把劇名、簡介與角色名翻成繁體中文。` |
| 橘色警示區（`--warning-tint`／`triangle-alert`，兩行） | 第一行（600 粗）`影集只有一個檔名可用，這會覆寫現有的 tvshow.nfo。`<br>第二行（400）`原始檔會先備份成 tvshow.nfo.orig；之後再執行也不會覆蓋這份備份。` |
| 藍色成本區 | `會使用 AI 翻譯額度` |
| Checkbox（🔴 **預設不勾**） | 主標 `連同每一集的集名與劇情`<br>副標（`--text-muted`, 12px）`每一集各翻譯一次，額度用量會明顯增加。` |
| 主鍵 / 次鍵 | `備份並覆寫` / `取消` |

🔴 **主鍵文案不得改成「確定」** —— 必須說出它要做什麼；影集那顆要把兩個動作都講出來（先備份、才覆寫）。

### AC #3b — 原本的確認驗收條款

3.1 影集／單集的觸發**必須**先出確認，使用者確認後才送出 `confirm_replace: true`。
3.2 🔴 **前端不得預設帶 `confirm_replace: true` 就直接送出** —— 那等於把後端刻意設下的閘門繞過去，使用者的檔案會在他不知情時被覆寫。**測試釘住：未經使用者確認的互動，service 呼叫次數為 0。**
3.3 電影**不需**確認（additive，不動原檔）—— 除非設計裁定另有規定。
3.4 `include_episodes` 的控制項與**預設值**依設計裁定（⚠️ 影響花費：一集一次 LLM 呼叫）。

### AC #4 — 結果回饋（✅ 設計已定稿 2026-08-21）

🔴 **用 inline 狀態 pill 就地取代那顆按鈕，不是浮動 toast。**
理由：repo **沒有共用 Toast 元件**（`ui/` 內查無）；既有語彙是 `RequestButton.tsx` 的
inline pill（`role="status"` + `aria-live="polite"` + `rounded-full` + tinted bg）。**沿用，不發明新系統。**

四態（`h-40`、`rounded-full`、icon 18×18）：

| 狀態 | 底色／圖示 | 逐字文案 |
|---|---|---|
| 電影成功 | `--success-tint` / `check` | `已寫入繁中資訊` |
| 影集覆寫成功 | `--success-tint` / `check` | `已覆寫，原檔已備份為 .nfo.orig` |
| 整劇部分成功 | `--warning-tint` / `triangle-alert` | `影集資訊已更新 · 22 集完成、2 集略過`（數字為動態） |
| 未設定金鑰（503） | `--error-tint` / `key-round` | `尚未設定翻譯服務 · 前往設定`（`前往設定`可點，導向金鑰設定） |

📌 `skipped` 對使用者一律說「**略過**」。其語意（spec 註解已定稿）：
`「略過」＝資料庫有這一集，但硬碟上找不到對應的影片檔，所以沒有地方可以放 .nfo。`

### AC #4b — 原本的回饋驗收條款

4.1 成功：顯示寫到哪、是否覆寫、備份在哪（後端已回 `path` / `replaced` / `backup_path`）。
4.2 整劇：`succeeded` / `failed` / `skipped` 三個數字要露到什麼程度依設計；`skipped` 的意思是「資料庫有這集但硬碟上沒有檔案」。
4.3 失敗：依 AC #1.3 的對映；**503 要能把使用者導向設定金鑰**。

### AC #5 — 測試

5.1 service 層：三個呼叫的 URL／body／query 逐一斷言（特別是 `confirm_replace: true` 有帶、`include_episodes` 有正確附加）。
5.2 🔴 **確認閘門**：未確認的互動 → service **零呼叫**（AC #3.2）。
5.3 四種錯誤碼各自的 UI 分支。
5.4 整劇結果的三數字呈現。
5.5 a11y（Rule / Epic 11 retro AI-1）：確認對話框要 trap focus、開啟時移動焦點、關閉時還原、Escape 可關（`ui/Dialog` 若已提供則沿用並斷言）。
5.6 既有 `LocalDetailV2` 測試全綠（動作列新增一個項目不得打破既有斷言）。

### AC #6 — 範圍紅線

- ❌ 不動後端（三條路由與契約已出貨）。
- ❌ 不動 v1 詳情頁。
- ❌ 不做「還原備份」（後端沒有這個端點）。
- ❌ 不做批次／整庫在地化（本案是單一媒體的入口）。
- ❌ 不做自動觸發（是否納入常設同意政策是 `9R-10b` 那條線的問題）。

---

## Tasks / Subtasks

- [x] **Task 1（AC #1）** —— service 層 + 型別 + 錯誤碼對映 + 單元測試
- [x] **Task 2（AC #2）** —— 入口元件（依設計裁定），Rule 21 檔頭帶真節點 ID
- [x] **Task 3（AC #3）** —— 覆寫確認對話框（逐字文案照設計）＋ `include_episodes` 控制項
- [x] **Task 4（AC #4）** —— 結果回饋（單一 / 整劇）
- [x] **Task 5（AC #5）** —— 測試（含確認閘門零呼叫、a11y）
- [x] **Task 6** —— 閘門：`pnpm nx test web`、`pnpm run lint:all`（含 Rule 21 ESLint）、`prettier`
- [x] **Task 7** —— dev-story Step 9 的 UX 截圖比對（對照設計 story 產出的 `flow-b-detail-v2` 畫面）

---

## Dev Notes

### 三個最容易踩的坑

1. 🔴 **加到 v1 詳情頁。** `MediaDetailPanel` / `DetailPanelMenu` 沒有任何**正式**路由在用（只剩 dev-only 的 `/test` 展示廊）—— 加了真實使用者不會看到，而且**測試還是會過**（元件測試不看路由）。
2. 🔴 **在程式碼裡寫死 `confirm_replace: true` 然後直接送。** 後端的閘門是為了保護使用者的檔案，不是為了讓 FE 填一個常數。**確認必須是使用者做的動作。**
3. 🔴 **用 `// Design ref: — no current screen frame` 逃生門。** 本 story 之所以 blocked，就是為了不走那條；走了等於白等一輪設計。

### 架構護欄

- **Rule 5**：所有 server state 走 TanStack Query；mutation 成功後 invalidate 該媒體的 detail query，讓中繼資料重新抓。
- **Rule 3**：錯誤一律讀 `error.code`，不要比對訊息字串。
- **Rule 21**：ESLint 強制（`pnpm lint:all` ⇒ CI）。
- **Rule 18**：API 邊界 case 轉換 —— 注意後端回的是 `backup_path`（snake_case）。確認 repo 現行慣例是自動轉換還是逐欄對應，**不要假設**。
- **Epic 11 retro AI-1**：a11y pre-flight 是 dev-story Step 7 的強制閘門（本 story 有 `apps/web/` 變更 ⇒ **不可** N/A）。

### Time-dependent visual coverage

- **本 story 觸及 `apps/web/src/components/**`。** 但新增的元件**不讀 wall-clock**（`Date.now()` / `new Date()` 等）—— 在地化結果沒有相對時間顯示。
- 若最終設計加入了「上次在地化於 3 分鐘前」這類相對時間，**Rule 23 立刻適用**：需要 ≥2 個 fixture 狀態基準（`recent` / `stale`）＋ `withFixedClock(page, iso)` ＋ `clockTime` fixture 欄位，並標註 `Clock-mocked`。開工時依實際設計重新判斷，**不要沿用這行的預設**。

### Project Structure Notes

- 預期新增：1 個 service（或既有 service 的新方法）、1–2 個元件（入口 + 確認對話框）、對應 `.spec.tsx`。
- 修改：`LocalDetailV2.tsx`（動作列）。
- 無新第三方相依。

### References

- [Source: `apps/web/src/routes/media/$type.$id.tsx#77`] — 🔴 v1 是死程式碼的實證
- [Source: `apps/web/src/components/media/LocalDetailV2.tsx#1,147-184`] — 設計錨點與動作列現況
- [Source: `apps/api/internal/handlers/nfo_localizer_handler.go`] — 三條路由、`confirm_replace`、四個錯誤碼、回傳形狀
- [Source: `apps/api/internal/services/nfo_localizer_service.go`] — `NFOLocalizeResult` / `NFOSeriesLocalizeResult` 欄位
- [Source: `_bmad-output/implementation-artifacts/9R-13a-tv-nfo-localization.md`] — 單槽/覆寫/備份語意
- [Source: `project-context.md#Rule 3 / 5 / 18 / 21 / 23`]
- [Source: `_bmad-output/implementation-artifacts/9R-10c-episode-row-subtitle-cta-frontend.md`] — 同型「blocked → 設計 done → ready-for-dev」的先例

---

## Dev Agent Record

### Agent Model Used

claude-opus-5[1m] (BMAD `dev-story`, 2026-08-22)

### Debug Log References

- **Fault injection 反證確認閘門（AC #3.2 / #5.2）**：把入口按鈕的 `onClick` 從 `setOpen(true)` 改成
  `localize.mutate()`（＝停用閘門）→ **20 條中 18 條轉紅**；還原後 20/20 全綠。
  證明那批測試真的釘住「未經使用者確認就零呼叫」，不是恆真斷言。
- **a11y pre-flight 抓到一個真的 error（會擋 CI）**：`jsx-a11y/label-has-associated-control` ——
  原本用 `<label>` 包住 `<input>` + 巢狀 `<span>`，規則認不出可及文字。
  修：改為 `id` + `htmlFor` 顯式關聯，成本警語用 `aria-describedby` 綁上去（否則它只是一段沒人念的散字）。
  修後 `eslint` 對本案兩個檔案**零輸出**。
- **`tsc` 基準比對**：`npx tsc --noEmit -p apps/web/tsconfig.app.json` → **147 errors**，
  與乾淨 main 的既有基準**完全相同**（bugfix-e 記錄的 148→147）⇒ 本案**零新增型別錯誤**。
- **測試數對得上**：2722（main 基準）+ 15（service）+ 20（component）= **2757** ✅

### Completion Notes List

**實作總結（7/7 task、6/6 AC）**

1. **AC #1 —— service 層。** 新檔 `nfoLocalizerService.ts`：三個呼叫 + `NfoLocalizeApiError`（保留 `error.code`）
   + `NFO_ERROR_CODES` + `isSeriesResult` 型別守衛。
   🔴 **為什麼要自己的 Error 類別**：其他 service 共用的 `parseError` 只丟 `new Error(message)`，**會把 code 丟掉** ——
   那樣四種失敗在 UI 上長得一模一樣，503 就沒辦法給「去設定金鑰」的出口。鏡射 `KeySettingsApiError` 的先例。
   🔴 **`confirmReplace` 是必填參數，不是預設值** —— 服務層刻意不給它預設 `true`，否則後端的檔案保護閘門會被一個常數繞過。
   Rule 18：`snakeToCamel` 在邊界轉換，所以 wire 的 `backup_path` 到 UI 是 `backupPath`
   （⚠️ story AC #1.2 寫的是 snake_case 型別，**依 repo 慣例改為 camelCase**，測試釘住這個轉換）。
2. **AC #2 —— 入口。** 新元件 `NfoLocalizeAction.tsx`（`languages` 圖示 + 「在地化資訊」，secondary 樣式），
   接進 `LocalDetailV2` 的動作列。**未動 `MediaDetailPanel` / `DetailPanelMenu`**（死程式碼）。
   僅在 `hasFilePath` 時渲染。
   **Rule 21 檔頭**：`LocalDetailV2.tsx` 的檔頭由
   `// Implements: Component/Detail-Movie-v2 (uRGu2) + …` 改為
   `// Design ref: ux-design.pen Screen B3p-D (uRGu2) + Screen B4p-D (N2fmG6) + Screen B3p-M (SzNRb)`
   —— 那三個 id 是 **screen frame 不是 Reusable Component**（Sally MCP 實查），`Design ref:` 才是正確文法。
   ⚠️ 這順帶解掉了 `backlog-localdetailv2-rule21-header-grammar` 的**一半**（見 Discovery Triage）。
   新元件檔頭指向 spec 畫面：`// Design ref: ux-design.pen Screen J6-D (zMYsL)`。
3. **AC #2 手機兩排** —— 新增共用的 `actionBasis`：`grow basis-[calc(50%-0.25rem)] sm:grow-0 sm:basis-auto`。
   四顆在 `< sm` 自動折成兩排等寬，`sm` 以上回到一排自然寬度。
   「複製路徑」在手機補上文字標籤（`sm:hidden`），桌面維持 icon-only —— 兩份設計都照顧到。
   ⚠️ **記錄在案的取捨（CR M4 更正）**：設計的手機分組是「管理字幕｜在地化」/「修改資訊｜複製路徑」，
   與桌面順序不同 ⇒ 用 `order-2/order-3` + `sm:order-3/sm:order-2` 交換。
   DOM 順序＝`管理字幕 → 修改資訊 → 在地化 → 複製路徑`＝**桌面視覺順序**（dev 初稿寫反了）。
   代價是**手機**的 DOM 與視覺在中間兩顆上不一致。這是刻意的取捨：桌面是鍵盤 Tab 使用者所在的斷點、
   手機是觸控，所以讓 DOM 對齊桌面。若要根治，需 Sally 統一兩個斷點的順序；改法仍是一行。
4. **AC #3 —— 確認對話框。** 走既有 `ui/Dialog`（Radix）⇒ **focus trap / 初始焦點 / 關閉還原 / Escape 全部免費**。
   兩版逐字文案照 Sally 定稿一字未改。**主鍵不是「確定」**：電影`開始在地化`／影集`備份並覆寫`。
   `include_episodes` checkbox **預設不勾**。
   🔴 **電影也有對話框** —— 依 Sally 裁定（推翻 story 原 AC #3.3）：電影一樣花 LLM 錢。
5. **AC #4 —— 結果回饋。** inline pill **就地取代按鈕**（`role="status"` + `aria-live="polite"`，
   沿用 `RequestButton.tsx` 語彙，**未發明新的 Toast 系統**）。四態＋整劇三數字；
   `skipped` 一律說「略過」，`failed` 另計為「失敗」（兩者語意不同，合併會誤導）。
6. **AC #5 —— 測試 35 條。** service 15 + component 20，含確認閘門四案（點按鈕／取消／Escape／只有確認鍵才送）、
   四種錯誤碼分支、整劇三數字四情境、逐字文案斷言。
7. **AC #6 紅線全守** —— 零後端變更、未動 v1 詳情頁、無「還原備份」、無批次整庫、無自動觸發。

**🎨 UX Design Verification（Step 9，MANDATORY）**

對照 `flow-b-detail-v2/b3p-d.png`・`b4p-d.png`・`b3p-m.png` 與 `flow-j-specs/j6-d.png`：

| 區域 | 設計 | 實作 | Match |
|---|---|---|---|
| 桌面動作列順序 | 管理字幕 → 修改資訊 → 在地化資訊 → 複製路徑 | 同（`sm:order` 還原） | ✅ |
| 手機動作列 | 兩排×兩顆、四顆**都有標籤** | `basis-[calc(50%-0.25rem)]` + 複製路徑補 `sm:hidden` 標籤 | ✅ |
| 新按鈕 | `languages` + 「在地化資訊」+ secondary 底 | `<Languages>` + `bg-[var(--bg-secondary)]` | ✅ |
| 對話框標題 | `將資訊在地化為繁體中文` | 逐字相同 | ✅ |
| 電影安心區 | `--success-tint` / `shield-check` | 同 | ✅ |
| 成本區 | `--info-tint` / `sparkles` | 同 | ✅ |
| 影集警示區 | `--warning-tint` / `triangle-alert`，兩行（600 / 400） | 同 | ✅ |
| Checkbox | 預設不勾、主標 + `--text-muted` 12px 副標 | 同 | ✅ |
| 主鍵文案 | `開始在地化` / `備份並覆寫` | 逐字相同 | ✅ |
| Pill 四態 | h40 / `rounded-full` / 四組 tint + icon | `h-10` + `rounded-full` + 同色同 icon | ✅ |

**🎨 UX Verification: PASS** —— 零差異，無需修正。

**閘門結果（全部實跑）**

| 閘門 | 結果 |
|---|---|
| `pnpm nx test web` | ✅ exit 0，**237 檔 / 2757 測試** |
| `pnpm nx test api` | ✅ 全綠（零後端變更，回歸確認） |
| `pnpm run lint:all` | ✅ **0 errors** / 119 warnings（main 既有基準） |
| `eslint`（本案 2 檔） | ✅ **零輸出**（a11y error 已修） |
| `tsc --noEmit` | ✅ 147＝main 基準，**零新增** |
| `prettier --check .` | ✅ |
| `pnpm run test:cleanup` | ✅ 無 vitest/playwright 殘留（列出的 PID 是使用者的 Edge 瀏覽器） |

**強制稽核項**

- 🔗 **AC Drift: NONE** —— 本案**零後端變更**；FE 側是淨新增 + `LocalDetailV2` 動作列擴充。
  唯一改動既有行為的是 `LocalDetailV2` 的四顆按鈕排版，而那正是 `9R-UX` 設計 story 的定稿（同一條線，非 drift）。
- 📎 **Contract Stamps: NONE** —— 本 story 與 `9R-13`／`9R-13a` 皆無 `[@contract-v*]` 戳記（隱含 v0），
  且本案**未改動任何後端契約**（三條路由、四個錯誤碼、回傳形狀全部照用）。
- 🔒 **Rule 7 Wire Format: N/A** —— 零 Go 錯誤碼檔案在範圍內（純前端）。
- 🔌 **Route Sync: 三條路由皆已註冊並驗證**（9R-13a 已上線）：
  `POST /movies/:id/localize-nfo`・`POST /series/:id/localize-nfo`・`POST /episodes/:id/localize-nfo`
  於 `nfo_localizer_handler.go` `RegisterRoutes` 註冊，`cmd/api/main.go:961` 掛載。
  **本案是它們的第一個前端呼叫者** —— 這正是 story 存在的理由。
- 🎭 **A11y Pre-Flight: FOUND → FIXED** —— `jsx-a11y/label-has-associated-control` **error**（會擋 CI），
  已改為 `id`/`htmlFor` + `aria-describedby`。另手動確認四類：
  響應式圖片 N/A（無 `<img>`）／**modal focus 管理由 Radix `ui/Dialog` 提供**（focus trap + Escape + 還原，測試斷言 Escape 不觸發呼叫）／
  **非同步內容 `role="status"` + `aria-live="polite"`**（四態 pill 全有）／自訂 widget N/A（用原生 `<input type="checkbox">` 與 `<button>`）。
- 🎨 **UX Verification: PASS**（見上方比對表）。
- 🕰️ **Rule 23: N/A** —— 新元件**不讀 wall-clock**（`Date.now()` / `new Date()` 皆未使用），無相對時間顯示。
  story Dev Notes 預留的「若設計加入『上次在地化於 3 分鐘前』則適用」**未發生**。
- **Pre-existing failures: NONE**。

### Discovery Triage

**YES —— 2 項。**

| Lane | 發現 | 追蹤 |
|---|---|---|
| ① expand-scope-in-place | **`LocalDetailV2.tsx` 的 Rule 21 檔頭語法錯**（screen frame 卻用 `Implements: Component/`）—— 本案必須動那個檔頭才能寫對新的設計指向，因此**就地修正** | 已修。⚠️ `backlog-localdetailv2-rule21-header-grammar` **只解掉一半**：條目還提到「ESLint 規則只驗語法形狀、不驗節點實際類型」，那個規則強化**仍未做**，條目保留 |
| ③ backlog-with-carry-forward-link | **桌面 DOM 順序與視覺順序在中間兩顆上不一致**（`order-2/3` ↔ `sm:order-3/2` 交換造成）—— 為了同時滿足設計的桌面與手機兩種分組 | 已在 Completion Notes 記錄取捨與一行改法。若 CR 認為需要根治（例如請 Sally 統一兩個斷點的順序），再開條目；本案不預先立案以免製造空條目 |

### File List

| 檔案 | 變更 |
|---|---|
| `apps/web/src/services/nfoLocalizerService.ts` | **new** —— 三個呼叫、`NfoLocalizeApiError`（保留 code）、`NFO_ERROR_CODES`、`isSeriesResult` |
| `apps/web/src/services/nfoLocalizerService.spec.ts` | **new** —— 15 條 |
| `apps/web/src/components/media/NfoLocalizeAction.tsx` | **new** —— 按鈕 + 兩版確認對話框 + 四態 inline pill |
| `apps/web/src/components/media/NfoLocalizeAction.spec.tsx` | **new** —— 20 條（含閘門四案） |
| `apps/web/src/components/media/LocalDetailV2.tsx` | modified —— Rule 21 檔頭改用 `Design ref:` 文法、`actionBasis` 響應式類別、接入新元件、複製路徑補手機標籤 |
| `_bmad-output/implementation-artifacts/9R-13b-nfo-localization-frontend.md` | modified |
| `_bmad-output/implementation-artifacts/sprint-status.yaml` | modified |

### Change Log

| 日期 | 變更 |
|---|---|
| 2026-08-22 | **Task 1（AC #1）** —— `nfoLocalizerService`：三呼叫 + 保留 `error.code` 的 `NfoLocalizeApiError`（共用 `parseError` 會丟掉 code，四種失敗會長得一樣）+ `confirmReplace` 為必填參數（不給預設值，否則後端閘門被常數繞過）+ Rule 18 邊界轉換。15 條測試。 |
| 2026-08-22 | **Task 2（AC #2）** —— `NfoLocalizeAction` 接入 `LocalDetailV2`；`actionBasis` 讓四顆在手機折兩排等寬、桌面回一排；複製路徑補手機標籤。**Rule 21 檔頭改用 `Design ref:` 文法**（screen frame 非 Reusable Component）。 |
| 2026-08-22 | **Task 3（AC #3）** —— 兩版確認對話框（Radix `ui/Dialog`，focus trap/Escape 免費），逐字文案零改動，主鍵非「確定」，`include_episodes` 預設不勾。電影也有對話框（Sally 裁定）。 |
| 2026-08-22 | **Task 4（AC #4）** —— 四態 inline pill 就地取代按鈕，沿用 `RequestButton` 的 `role="status"`+`aria-live` 語彙；`skipped`（略過）與 `failed`（失敗）分開講。 |
| 2026-08-22 | **Task 5（AC #5）** —— 35 條測試。**Fault injection**：停用確認閘門 → 20 中 18 轉紅，還原 → 全綠。 |
| 2026-08-22 | **Task 6/7** —— a11y pre-flight 抓到並修掉一個會擋 CI 的 `label-has-associated-control` error；UX 截圖比對 10 項全 match。閘門全綠：web 2757、api 綠、lint:all 0 errors、tsc 147＝基準、prettier、cleanup 無殘留。Status → review。 |

---

## Senior Developer Review (AI)

**Reviewer:** Bob (SM) 代跑 `/code-review` —— **Fable 5（與實作者 Opus 5 不同模型）** · **Date:** 2026-08-22 · **Outcome:** APPROVED WITH FIXES

**Git vs Story File List：0 落差**（7 個檔案逐一對上）。

### 強制閘門

| 檢查 | 結果 |
|---|---|
| 🔒 Rule 7 Wire Format | **N/A** —— 純前端，零 Go 錯誤碼檔案 |
| 🔒 Rule 20 Contract Bump | **N/A** —— diff 中 `[@contract-vN→vM]` 命中 0 |
| 🔒 Rule 25 Mega-line | **N/A** —— `project-context.md` 未被修改 |

### findings：2 HIGH / 4 MEDIUM / 1 LOW —— **7/7 全修**

| # | 嚴重度 | 發現 | 處置 |
|---|---|---|---|
| **H1** | 🔴 HIGH | **成功 pill 會說謊。** 文案由 `mediaType` 決定，不是由後端回的 `replaced`。實查後端：影集**沒有**既有 `tvshow.nfo` 時回 `replaced:false`、零備份（`nfo_localizer_service.go:591`）—— 但 UI 照樣說「已覆寫，原檔已備份為 .nfo.orig」，**宣稱一份不存在的備份**；反過來電影兩槽皆滿時會走 backup-and-replace 回 `replaced:true`（`:238`）—— UI 卻說「已寫入繁中資訊」，**把覆寫講成寫入**。AC #4.1 明文「顯示是否覆寫、備份在哪（後端已回 replaced）」，實作漏用了。這是 bugfix-j 剛修掉的「狀態說謊」類。 | ✅ 修 —— `Outcome` 改為 `{kind:'ok', replaced}`，文案隨 `replaced` 走。+2 測試（tv 無原檔**不得**提 `.orig`；movie 兩槽皆滿**要**報覆寫） |
| **H2** | 🔴 HIGH | **錯誤是死胡同。** 任何失敗（500／400／網路）後 `outcome` 被設定且**永不清除**，按鈕消失、沒有任何重試路徑，使用者只能重新整理整頁。503 有「前往設定」所以還好；其他錯誤就卡死。 | ✅ 修 —— 錯誤 pill 加「重試」inline link（沿用 503 pill 的「前往設定」語彙），點了清 `outcome` 讓按鈕回來；測試斷言回來後**仍受閘門保護**（呼叫次數不變） |
| **H3** | 🔴 HIGH（原 M3 升級） | **關閉對話框後焦點掉到 `<body>` —— 真實瀏覽器也一樣。** 本輪先依 AC #5.5 補「焦點還原」斷言，結果**紅**。追到 Radix 源碼：modal content 的 `onCloseAutoFocus` 只會 `context.triggerRef.current?.focus()`（`react-dialog/dist/index.mjs:148`），而 `triggerRef` **只有 `<DialogTrigger>` 會設**。實作用的是 `<Dialog>` 外面一顆普通 `<button>` ⇒ ref 為 null ⇒ Escape 後鍵盤使用者被丟到頁面頂端。Epic 11 retro AI-1 的四類 a11y 之一（modal focus 管理），而 dev 的 a11y pre-flight 記了「Radix 免費提供」—— **免費的前提是要用對元件**。 | ✅ 修 —— 按鈕改包進 `<DialogTrigger asChild>`，開關邏輯收斂到單一 `handleOpenChange`。**修前測試紅、修後綠**，證明斷言真的釘住行為 |
| **M1** | MEDIUM | **`include_episodes` 勾選狀態跨次開啟殘留。** 勾了→取消→再開，還是勾著。Sally 的「預設不勾」是**每次開啟**的預設（一集一次 LLM 費用，24 集＝24 倍）；殘留讓「先做一部再決定」的安全路徑在第二次開啟時失效。 | ✅ 修 —— 每次 `onOpenChange(true)` 重設為 false；+1 測試 |
| **M2** | MEDIUM | **500 把後端的英文直接秀給使用者。** `NFO_LOCALIZE_FAILED` 的後端訊息是 `"Failed to localize metadata"`，實作 `error.message` 原樣顯示。Rule 3 要求使用者可見文字 zh-TW；dev 的 AC #1.3 寫「一般失敗訊息」卻沒對映。409（理論上不該發生）同樣是英文。 | ✅ 修 —— 已知碼（`failed`／`notConfirmed`）對映 zh-TW，**只有未知碼**才透傳原文（保留診斷價值）；409 訊息明講「這是程式錯誤，請回報」。+1 測試（未知碼透傳）、1 測試改寫 |
| **M4** | MEDIUM | **Completion Notes 的取捨寫反了。** 記「桌面 DOM 與視覺不一致」，實查 class：DOM＝`字幕→修改→在地化→複製`，桌面 `sm:order` 後視覺也是這個順序 ⇒ **桌面一致、手機才不一致**。方向反了會讓下一個人修錯邊。而且現行選擇其實是對的（鍵盤 Tab 使用者在桌面）。 | ✅ 修 —— 更正敘述並補上「為什麼讓 DOM 對齊桌面」的理由 |
| **L1** | LOW | **story Dev Notes 要求 mutation 成功後 `invalidateQueries` detail query，實作沒做也沒說為什麼。** 實查：`.nfo` 在地化**寫的是磁碟上的檔案，不是資料庫**（`LocalizeMovieNFO` → `writeAdditiveNFO`），詳情頁顯示的 DB 資料不會變 ⇒ invalidate 是 no-op。結論對，但偏離沒記錄。 | ✅ 修（記錄型）—— 在此載明：**刻意不 invalidate**，因為沒有任何 server state 改變；Dev Notes 的那條假設不成立 |

### 看過但**判定不是** finding

- **service 暴露 `localizeEpisodeNfo` 但零元件呼叫者** —— AC #1.1 明文要求三個呼叫；單集路由的 UI 入口不在本案範圍（詳情頁只有 movie/tv）。service 測試有覆蓋。保留。
- **`ResultPill` 在 `role="status"` 的 span 內放 `<button>`** —— `<span>` 可含 phrasing content，`<button>` 是 phrasing；live region 內有互動元素可接受。
- **`postJson` 在無 body 的 movie 請求也送 `Content-Type: application/json`** —— 無害。
- **成功 pill 也是「一次性」（不能再跑一次）** —— 與 H2 不同：成功後立刻重跑是罕見路徑，且換頁即重置。Sally 的 J6-D 四態 pill 也沒有重跑設計。不改，記錄。
- **`hasFilePath=false` 時整顆消失而非 disabled** —— 與既有「管理字幕」「複製路徑」同一條件、同一行為，一致。

### 修後閘門（全部重跑）

| 閘門 | 結果 |
|---|---|
| `pnpm nx test web` | ✅ exit 0，237 檔 / **2763 測試**（2757 + 6） |
| `eslint`（本案 2 檔，含 jsx-a11y） | ✅ 零輸出 |
| `pnpm run lint:all` | ✅ 0 errors / 119 warnings（既有基準） |
| `tsc --noEmit` | ✅ 147 = main 基準 |
| `prettier --check .` | ✅ |

**本案測試 41 條**（service 15 + component 26）。

### Action Items

無 —— 2H/4M/1L 全數在 review 內修畢。
**給 Sally 的一則追認請求（非阻擋）**：錯誤 pill 新增的「重試」inline link 不在 J6-D 四態規格內，沿用了 503 pill「前往設定」的同一語彙。建議下次 `.pen` 同步時補上。

### Change Log（review 追加）

| 日期 | 變更 |
|---|---|
| 2026-08-22 | **CR 修復 7/7（Fable 5 換模型審）** —— **H1** 成功 pill 改由後端 `replaced` 決定（實查 `:238`/`:591` 兩個分支，原實作對兩種媒體各說謊一次）；**H2** 錯誤 pill 加「重試」、不再是死胡同；**H3** 按鈕改 `<DialogTrigger asChild>`，修掉 Escape 後焦點掉到 `<body>` 的真實 a11y 缺陷（Radix 只還原到 `triggerRef`，測試修前紅修後綠）；**M1** 每次開啟重設 `include_episodes`；**M2** 500/409 對映 zh-TW、未知碼才透傳；**M4** 更正寫反的 DOM/視覺取捨敘述；**L1** 記錄「刻意不 invalidate」的理由（nfo 寫磁碟不寫 DB）。+6 測試（共 41）。閘門全綠。Status review → done。 |
