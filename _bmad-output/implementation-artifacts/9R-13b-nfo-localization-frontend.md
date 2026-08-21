# Story 9R-13b: .nfo 在地化的前端入口（FE 落地）

Status: ready-for-dev  <!-- 🔓 UNBLOCKED 2026-08-21：9R-UX-nfo-localization-entry-design done + Sally MCP review PASS。設計節點與逐字文案見下方 AC。 -->

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

- [ ] **Task 1（AC #1）** —— service 層 + 型別 + 錯誤碼對映 + 單元測試
- [ ] **Task 2（AC #2）** —— 入口元件（依設計裁定），Rule 21 檔頭帶真節點 ID
- [ ] **Task 3（AC #3）** —— 覆寫確認對話框（逐字文案照設計）＋ `include_episodes` 控制項
- [ ] **Task 4（AC #4）** —— 結果回饋（單一 / 整劇）
- [ ] **Task 5（AC #5）** —— 測試（含確認閘門零呼叫、a11y）
- [ ] **Task 6** —— 閘門：`pnpm nx test web`、`pnpm run lint:all`（含 Rule 21 ESLint）、`prettier`
- [ ] **Task 7** —— dev-story Step 9 的 UX 截圖比對（對照設計 story 產出的 `flow-b-detail-v2` 畫面）

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

{{agent_model_name_version}}

### Debug Log References

### Completion Notes List

### Discovery Triage

- **本 story 是否發現任何超出當前範圍的工作？**
  - 若 **NO**：寫 `N/A — no out-of-scope work discovered`。
  - 若 **YES**：每項一列，歸入**恰好一條** lane（① / ② / ③）。

**已預見的候選：**

- **v1 詳情頁死程式碼的清理**（`MediaDetailPanel` / `DetailPanelMenu` / `Screen 4c` 設計節點 / gallery fixture）—— 本 story 只是繞過它。要刪是**獨立的清理案**（lane ③）。
- **「還原 `.nfo.orig` 備份」沒有後端端點** —— 若設計輪認為使用者需要一鍵還原，那是**後端新 story**（lane ③），不可在 FE 硬幹。

### File List

<!-- 待實作填寫 -->

### Change Log
