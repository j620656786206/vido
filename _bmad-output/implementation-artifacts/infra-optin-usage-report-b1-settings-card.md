# Story infra-optin-usage-report-b1: 設定頁「連線設定」最底下多一張「匿名使用回報」卡

Status: review

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

**Epic:** standalone（`infra-optin-usage-report` 家族）· **Priority:** P1 · **Size:** S（前端 only）
**Source:** 設計 `infra-optin-usage-report-design`（PR #666）：C25-D `SHogC`／C25-M `bG3l3`／C26-D `FeCfY`；後端 `infra-optin-usage-report-a2-reporter`（PR #665）。
**Split（SM Bob 2026-10-05）：** 原 `infra-optin-usage-report-b-frontend` 依版面拆成 b1（設定卡，本單）與 b2（精靈步驟）——`feedback_split_oversized_stories`：切分線選版面。兩張互不相依。

---

## Story

As a Vido user,
I want a card at the bottom of 設定 → 連線設定 that lets me turn the anonymous weekly report on or off and shows exactly what was sent last time,
so that I can decide with full information and verify the claim myself.

---

## Context —— 查到的事（2026-10-05，SM Bob 逐行查證）

- **頁面**：`apps/web/src/routes/settings/connection.tsx:12-42`——`SettingsPageHeader` 下一個 `flex flex-col gap-4`，依序 qBittorrent 卡（`:22-37`，卡片樣式 `max-w-3xl rounded-[var(--radius-lg)] border border-[var(--border-subtle)] bg-[var(--bg-secondary)] p-4 md:p-6`）、`ArrConnectionForm` sonarr、radarr（`:38-39`）。檔頭 `// Design ref:` 列 C4-D／C4-M／C23-D／C23-M（`:1-2`）。
- **開關範本**：`apps/web/src/components/settings/ArrConnectionForm.tsx:336-373`（`role="switch"`、`aria-checked`、`aria-labelledby`、44px 點擊區、`--accent-primary`／`--bg-tertiary` 軌道）。
- **時間格式範本**：`apps/web/src/components/settings/OfficialSubtitleMiningCard.tsx:23-28` `formatRunTime`（`YYYY-MM-DD HH:mm`，本地時間）。
- **整頁錯誤範本**：`apps/web/src/components/settings/SettingsErrorState.tsx`（不顯示後端原文）。
- **service／hook 範本**：`apps/web/src/services/subtitleLocalizationService.ts:37-62`、`apps/web/src/hooks/useSubtitleLocalization.ts`（PUT 回應直接 `setQueryData`）。
- **後端契約**（a2，已上線）：`GET`／`PUT /api/v1/settings/usage-report` → `{available, enabled, last_sent_at, last_payload}`；從未送出時後兩者為 `null`；`last_payload` 是原樣字串；PUT body `{enabled: boolean}`，缺欄位 400。

## Acceptance Criteria

1. **位置**：`/settings/connection` 在 Radarr 卡下面多一張卡，外觀與 qBittorrent 卡同一套（寬度、底色、圓角、邊框、內距）；`Design ref` 註解補上 C25-D／C25-M／C26-D。
2. **內容（C25）**：標題「匿名使用回報」、副標「每週最多一次，把幾個匿名數字送給 Vido 維護者。」；開關列標籤「每週送一次匿名計數」、提示「預設關閉。關掉之後就不會再送。」；連結「送什麼、不送什麼 →」指向 GitHub 上的 `docs/usage-report.zh-TW.md`（新分頁）。
3. **四種狀態**（C25／C26）：
   - 不可用（`available=false`）：開關停用（視覺 0.4 透明），提示改為「這個版本沒有設定接收端，無法開啟。」，不顯示讀數與原文。
   - 關閉：開關關，無讀數與原文。
   - 已開啟、還沒送過（`enabled` 且 `last_sent_at=null`）：讀數列「上次送出｜還沒送過——打開後一小時內會送出第一份」（`--text-muted`）。
   - 已送過（`last_sent_at` 有值）：讀數列「上次送出｜YYYY-MM-DD HH:mm」（等寬）＋「送出的內容（原文）」方塊，**原樣顯示 `last_payload`**（等寬、`--bg-primary` 底、可換行、不得水平溢出）。關閉後若曾送過，讀數與原文仍顯示（那是事實紀錄）。
4. **開關行為**：按下即呼叫 PUT；送出中開關停用；失敗時退回原狀並在卡內顯示「沒有存到，請再試一次。」（不顯示後端原文）；成功以回應更新畫面（`setQueryData`）。
5. **載入／錯誤**：載入中卡內顯示骨架或「讀取中…」；GET 失敗時卡內顯示一行「讀不到匿名使用回報的狀態。」＋重試按鈕，不影響同頁其他卡。
6. **a11y**：開關 `role="switch"`＋`aria-checked`＋`aria-labelledby`；原文方塊有可讀標籤（`aria-labelledby` 指向「送出的內容（原文）」）。
7. **測試**：service（GET／PUT 形狀、錯誤）、hook（PUT 成功寫 cache）、卡片四種狀態＋開關成功／失敗回滾＋GET 失敗重試；原文顯示與 `last_payload` 逐字相同（用 `docs` 範例那一行當 fixture）。
8. **Gallery＋visual**：`-gallery.fixtures.tsx` 新增四種狀態的 fixture（桌面）＋已送過的手機寬度 fixture；已送過的 fixture 用 Rule 23 固定時間；`-darwin` 基準在本機產生，`-linux` 交給 CI bootstrap。
9. **文件**：`docs/usage-report.md`／`.zh-TW.md` 與 README 中「設定頁開關下一版提供」的說法，改為只保留精靈部分仍待 b2（b2 合併時再移除）。

## Tasks / Subtasks

- [x] Task 1: `usageReportService.ts` + `useUsageReport.ts`（AC #4、#7）
- [x] Task 2: `UsageReportCard.tsx`（四狀態、開關、錯誤、a11y）（AC #2、#3、#4、#5、#6）
- [x] Task 3: 掛到 `connection.tsx`＋Design ref（AC #1）
- [x] Task 4: 單元測試（AC #7）
- [x] Task 5: Gallery fixtures＋visual 基準（AC #8）
- [x] Task 6: 文件措辭（AC #9）

Cross-stack split check：後端 0、前端 6 → 不需拆。

## Dev Notes

- `snakeToCamel` 會轉 key，不會動字串值——`lastPayload` 的內容保持原樣。
- Rule 23：卡片會格式化 `lastSentAt`（本地時間）→ fixture 用固定 ISO 時間並標註 `Clock-injected` 或在 fixture 用 `clockTime`。
- 不要把 `last_payload` 重新 `JSON.parse`／美化：設計與 PRD 都要求「原樣」。

### Time-dependent visual coverage
- 本單元件讀 `new Date(lastSentAt)` 做格式化（不讀現在時間）。依 Rule 23 仍在 fixture 固定 `lastSentAt` 的 ISO 字串；不依賴 `Date.now()`，所以只需一個「已送過」狀態，不需要 recent／stale 兩態。

### References
- 設計：`_bmad-output/screenshots/flow-c-search-settings/c25-d.png`、`c25-m.png`、`c26-d.png`
- `project-context.md` Rule 5、9、16、18、21、23

## Dev Agent Record

### Agent Model Used

Claude Opus 5.5（2026-10-05，dev-story／Amelia）

### Completion Notes List

- Ultimate context engine analysis completed - comprehensive developer guide created（SM Bob，2026-10-05）
- **Task 1**：`services/usageReportService.ts`（`get`／`setEnabled`，錯誤帶 code 的 `UsageReportApiError`）、`hooks/useUsageReport.ts`（PUT 回應 `setQueryData`）。`snakeToCamel` 只改 key，`lastPayload` 字串原樣。
- **Task 2**：`components/settings/UsageReportCard.tsx`——四狀態（不可用：開關 `disabled`＋`opacity-40`＋原因；關閉；已開啟未送過：讀數「還沒送過——打開後一小時內會送出第一份」；已送過：等寬時間＋原文 `<pre>` 原樣、`break-all whitespace-pre-wrap`）；關閉後若曾送過，紀錄仍顯示。開關送出中顯示目標狀態（`save.variables`）並停用；失敗退回並顯示「沒有存到，請再試一次。」；GET 失敗卡內一行＋重試。時間用既有 `formatRunTime`（`OfficialSubtitleMiningCard.tsx`）。`new Date(iso)` 不在 Rule 23 規則範圍內，不需 marker。
- **Task 3**：`routes/settings/connection.tsx` 在 Radarr 卡下掛 `UsageReportCard`，`Design ref` 補 C25-D／C25-M（C26-D 寫在元件檔頭）。
- **Task 4**：service 4 測、卡片 9 測（**真 QueryClient＋fetch mock**，不 mock hook——避免 TanStack refetch 退回 pending 的行為被藏住）：標題／說明／連結、四狀態、關閉後紀錄仍在、開關 PUT 成功、失敗回滾且不顯示後端原文、GET 失敗重試成功；原文用文件範例那一行逐字比對（`textContent === PAYLOAD`）並驗證可讀標籤。
- **Task 5**：gallery 5 個 fixture（C25-D 已送過、C26-D 三態、C25-M 手機 390×844），`lastSentAt` 固定 ISO（與 mining 卡同做法，darwin／linux 可能因時區不同而不同，屬預期）；`test:visual:update-missing` 產生 5 張 `-darwin` 基準，`-linux` 交給 CI bootstrap。
- **Task 6**：兩份文件與 README 改為「設定頁開關已可用，精靈步驟下一版（b2）」。
- **Gate**：`nx test web` 296 檔／4555 測全綠；`go test ./...` 綠；`lint:all` 0 errors（警告 172，與 main 相同、無新增）；`prettier --check .` 綠；`test:cleanup` 無殘留。
- 🔗 AC Drift: NONE（`/settings/connection` 只在 Radarr 卡後追加一張卡；`arr-settings.spec.ts` 的選擇器都 scope 在各自卡片內，不受影響）。
- 📎 Contract Stamps: NONE（無 [@contract-v*] 標記的 AC 被引用或修改）。
- 🎭 A11y Pre-Flight: PASS（1 個元件；開關 `role="switch"`＋`aria-checked`＋`aria-labelledby`；原文 `<pre>` 以 `aria-labelledby` 命名（測試驗 accessible name）；載入 `role="status"`、存檔失敗 `role="alert"`；觸控區 44px；eslint jsx-a11y 對新檔 0 警告）。
- 🎨 UX Verification：

| Area | Design Spec（C25-D／C26-D） | Implementation | Match? |
| --- | --- | --- | --- |
| 卡片外框 | 與連線設定其他卡同（768、`bg-secondary`、`radius-lg`、`border-subtle`） | 同 qBittorrent 卡 class | ✅ |
| 標題／副標 | 16 semibold／12 muted | `text-base font-semibold`／`text-xs text-muted` | ✅ |
| 開關列 | 標籤＋提示、右側 Switch | 沿用 `ArrConnectionForm` 開關 | ✅ |
| 上次送出 | 左標籤、右等寬時間 | `font-mono text-sm`，右對齊 | ✅ |
| 原文方塊 | 等寬、`bg-primary`、`radius-md`、自動換行 | `<pre>` `font-mono text-xs`、`break-all whitespace-pre-wrap` | ✅ |
| 連結 | `accent-text` | `text-[var(--accent-text)]` | ✅ |
| 三種狀態 | C26-D | fixture 三態截圖一致 | ✅ |
| 手機 | 原文折行、無橫向溢出 | 390 寬截圖折行、無溢出 | ✅ |


### Discovery Triage

- N/A — no out-of-scope work discovered

### File List

- apps/web/src/services/usageReportService.ts（新）、usageReportService.spec.ts（新）
- apps/web/src/hooks/useUsageReport.ts（新）
- apps/web/src/components/settings/UsageReportCard.tsx（新）、UsageReportCard.spec.tsx（新）
- apps/web/src/routes/settings/connection.tsx
- apps/web/src/routes/test/-gallery.fixtures.tsx
- tests/visual/components.visual.spec.ts-snapshots/components/settings-usage-report/**/default-visual-darwin.png（新，5 張）
- docs/usage-report.md、docs/usage-report.zh-TW.md、README.md
- _bmad-output/implementation-artifacts/sprint-status.yaml

### Change Log

| Date | Change |
| --- | --- |
| 2026-10-05 | dev-story：連線設定最底下的「匿名使用回報」卡（四狀態、原文原樣、開關回滾）、gallery＋darwin 基準、文件措辭；Status → review |
