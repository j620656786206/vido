# Story 6.12: 同意畫面金錢陷阱 —— 全選範圍、錯誤位置、砍線、總額 ≈、字級（前端）

Status: ready-for-review

## Story

As a BYOK NAS owner,
I want 全選 to select what I can see, failures to appear where I am looking, and the budget cut line to be visible in the list,
so that the number I consent to is the number I meant.

## Context

critique P1「篩選＋全選＝金錢陷阱」「開始失敗的錯誤埋在清單底」＋ P2「砍線不可見／總額假精準／10px」。

## Acceptance Criteria

1. **全選作用於可見集合。** `handleToggleAll`（`GenerationConsentView.tsx:277-281`）改為對 `visibleIds`（route chip × 搜尋，sub-6-11）操作；工具列文案「已選 x / n」改為「已選 x / 顯示 n（全部 N）」；全選 aria-label「選取顯示的 n 部」。`onSelectAllExtract` 語意不變（本來就是路線集合）。

2. **錯誤搬到看得到的地方。** `startError`（`CandidateListPanel.tsx:402-406`）移到 sticky footer 上方、與超支橫幅同區（`role="alert"`），並保留在清單捲動之外。

3. **砍線可見。** 超支時，在提交順序的第 `feasibleCount+1` 列**之前**畫分隔列「── 到此為止約 $上限，之後的項目會暫停 ──」，其後列 `opacity-60`；橫幅的「約 N 部」加「（清單中已標示）」。排序改變顯示順序時分隔列跟著提交順序的列走（sub-6-11 AC #2）。

4. **總額 ≈。** 已選項目中任一 `runtime_source=fallback` → 摘要、頁尾、確認框三處總額前加 `≈`（同源 selector 加 `hasEstimatedRows`）；確認框 F16 加一行「其中 n 部片長未知，以 45 分鐘估算」。

5. **字級底線。** `text-[10px]`（`:269`、`:274`）→ `text-xs`；內容用途的 `text-[11px]`（`:99`、`:189`、`:491`）→ `text-xs`；殼層用途維持。偵測器 `design-system-font-size` 歸零。

6. **扣誰的錢。** 摘要條末尾加一行「使用：Claude（你的金鑰）· 語音辨識：OpenAI（你的金鑰）／自架」——來源 `AnalysisSnapshot.self_hosted_asr` 與 keys 狀態（`ApiKeysForm` 已有的 `source` 資料）；critique 專案 persona 紅旗。

7. **設計 + 測試。** `.pen` F15/F18 更新（分隔列、錯誤位置、來源行）；重出截圖。specs：全選只選可見、錯誤在 footer 區、砍線位置＝`feasibleCount`、`≈` 三處同步、字級無 10/11px 內容、來源行三種狀態。

## Tasks / Subtasks

- [x] **Task 1 — 全選語意 + 文案（AC: #1）**
- [x] **Task 2 — 錯誤區與砍線（AC: #2, #3）**
- [x] **Task 3 — `≈` 總額與來源行（AC: #4, #6）**
- [x] **Task 4 — 字級（AC: #5）**；設計更新提示詞已產出，`.pen` 執行待 Alexyu（AC #7 前半）
- [x] **Task 5 — 測試（AC: #7 後半）**

（全前端。AC #6 若需後端曝露 key source 至 snapshot，為 additive 欄位，可併入 sub-6-10a。）

## Dev Notes

- 「三處金額同源」與「三序同源」兩條紅線都在 `consentSelection.ts`；新旗標與砍線索引都加在 `ConsentTotals`，不得在元件裡另算。
- Rule 21 header 不變。

### Time-dependent visual coverage

- N/A。

### References

- critique P1/P2；`GenerationConsentView.tsx:277-281`、`CandidateListPanel.tsx:191-199,269,274,402-433`

## Dev Agent Record

### Agent Model Used

Claude Opus 5（1M context）

### Completion Notes List

**AC #1 — 全選作用於可見集合（含群組標頭，Alexyu 2026-09-09 裁定）。**
`visibleIds` 從 panel 搬到 container（`GenerationConsentView`），因為它現在決定的是
「勾了會選到什麼」而不只是「畫什麼」，而選取狀態是 container state。`handleToggleAll`
改成只加減可見集合，所以在別的篩選下勾好的選取不會被抹掉。
`computeTotals` 多收一個 `visibleIds` 參數，回傳 `visibleSelectableCount` /
`visibleSelectedCount` / `visibleSelectedTotalUsd` —— 元件裡不另外算（Dev Notes 紅線）。

**超出 AC 的一項裁定：群組標頭也改成只勾可見。** sub-6-11 明文交棒了這個危害
（搜尋只看到一集、勾整劇標頭卻同意整部 9 集、約 $2.79），並記在 `onToggleGroup` 的註解。
問過 Alexyu，裁定「跟全選一樣，只勾看得到的」。代價是改掉 sub-5-3 寫過測試的語意 ——
那條測試已改寫成新語意並標明是誰裁定的。

**文案**：沒被篩選時工具列維持出貨的「已選 x / n」；一旦被篩窄才變成
「已選 x / 顯示 n（全部 N）」，其中 x 是**可見已選**，讓分子分母講同一件事
（全庫的「已選 N 部」在上方摘要條，不受任何篩選影響）。群組標頭同一套規則，
而且被篩窄時**金額也跟著只算可見的** —— 否則會出現「已選 0 · $1.55」這種
數字與金額各說各話的列。

**AC #2 — 錯誤搬家。** `startError` 從捲動區最後一個子節點搬到超支橫幅與頁尾之間，
`role="alert"`，加 `CircleAlert` 圖示與 `--error-tint` 底。順序是橫幅 → 錯誤 → 頁尾
（錯誤是剛發生的事，離按鈕最近）。

**AC #3 — 砍線。** `ConsentTotals` 多了 `cutMediaId`（**media id 不是索引** —— 清單可被
排序，索引會指到別部片）與 `pausedIds`。`consentRows` 多一種 `kind: 'cut'` 的列，插在
那個 id 的列前面，所以換排序時分隔列跟著它的片子走。線後的列 `opacity-60`。
一個誠實的邊界：選了一部比整個上限還貴的片時 `overBudget` 為真但 `cutMediaId` 是 null
（後端是每次付費呼叫**之前**檢查，所以那部片還是會跑）—— 這時不畫線，橫幅也不加
「（清單中已標示）」。

**AC #4 — `≈`。** `hasEstimatedRows` / `estimatedRowCount` 進 `ConsentTotals`，
`usdWithEstimate()` 一個 formatter 供摘要／頁尾／確認框三處共用（三處金額同源延伸到
**標記**本身）。F16 多一行「其中 n 部片長未知，以 45 分鐘估算」。
`isRuntimeApproximate` 從 panel 搬進 `consentSelection` —— 它現在是金錢事實，不能有第二份。

**AC #5 — 字級。** consent 目錄下 `text-[10px]` ×2（chip 標記）、`text-[11px]` ×3
（列的路線徽章、群組路線徽章、預算提示）全部改 `text-xs`；順手把 `ModelPicker` 的
三處 `text-[11px]` 也改掉，讓偵測器在整個目錄歸零（`grep text-[1[01]px]` = 0 筆）。

**AC #6 — 扣誰的錢。** `spendSourceLabel()` 是純函式，吃 `KeyState.source`
（跟金鑰設定頁同一個 query，`enabled: open`）與 sweep 自己的 `summary.selfHostedAsr`，
不猜。三種狀態：`secret` →「你的金鑰」、`env` →「環境變數金鑰」、其餘 →「尚未設定金鑰」；
自架 ASR →「語音辨識：自架（不另計費）」。清單沒有 ASR 列時整個 ASR 半句不出現。

**AC #7 — 設計與測試。** `.pen` 依 `feedback_pen_inline_agent_workflow` 走完整輪：
Claude 產節點錨定提示詞 `sub-6-12-f15-f18-money-traps-pen-prompt.md` → Alexyu 跑 Pencil
Inline AI Agent → Claude 以 MCP 複審。四張畫面全部落地（F15-D／F15-M 加來源行、
F18-D 加砍線＋淡化＋橫幅字＋來源行、新增獨立規格畫面 `f18-spec-err` 講錯誤位置 ——
依 `feedback_pencil_spec_standalone_screen` 不塞進 F18 正常狀態）。
Inline agent 多做且**正確**的一件事：加了來源行後 F15 桌機對話框變高、底部被切 12px，
它自行改為置中（F18 一併）—— 複審量過上下留白對稱（F15 各 19px、F18 各 41px）並追認。
複審抓到一項沒落地：`start-error-wrap` 的 8px 上下留白（Alexyu 授權後由 Claude 直接
MCP `Update` + AppleScript 存檔，並 grep 磁碟檔確認 `padding: [8, 24]` 真的落盤 ——
這次改動前後等長，`feedback_verify_pen_saved_before_commit` 的「看 size」再次失效）。
截圖全量重出 174 張，只 commit 真的改到的 4 張。
測試 +53（consent 目錄 195 → 248），web 全套 3,339 全綠（261 檔），lint 0 error。
視覺基準：11 張 darwin 重生、對應 `-linux` 已 `git rm` 走 CI bootstrap
（`project_visual_baseline_intentional_change` 的功能分支流程）。

### Discovery Triage

- **`--update-snapshots` 的預設模式會漏掉小改動。** Playwright 的 `changed` 模式用
  `maxDiffPixelRatio: 0.001` 當門檻，一個字元從「2」變「1」大約 40px，低於 900×768 的
  0.1%（691px），所以基準檔**不會被重寫** —— 留下一張數字是舊的參考圖。
  這次是用 `--update-snapshots=all` 全量重寫再把非本次改動的 checkout 還原。
  值得記成 memory。
- **`list-mobile` 視覺 fixture 的頁尾本來就是壞的**（「已 1 部 · 預 選 估」直排）。
  HEAD 的基準圖同樣如此，不是這次造成的。成因是 fixture 只把**元素**縮到 390px，
  但 `sm:` 斷點看的是**視窗**寬（1280），所以頁尾走了桌機的 `sm:flex-row`。
  真手機上視窗就是 390，會正常直排。屬於 fixture 失真，另立案較妥。
- **`useKeySettings` 在 consent 流程被叫用**，多一支 `GET /settings/keys`（`enabled: open`、
  `staleTime` 5 分鐘）。與金鑰設定頁共用 query key，所以多半直接命中快取。

### File List

- `apps/web/src/components/subtitle/consent/consentSelection.ts`（M）
- `apps/web/src/components/subtitle/consent/consentSelection.spec.ts`（M）
- `apps/web/src/components/subtitle/consent/consentRows.ts`（M）
- `apps/web/src/components/subtitle/consent/consentRows.spec.ts`（M）
- `apps/web/src/components/subtitle/consent/CandidateListPanel.tsx`（M）
- `apps/web/src/components/subtitle/consent/CandidateListPanel.spec.tsx`（M）
- `apps/web/src/components/subtitle/consent/GenerationConsentView.tsx`（M）
- `apps/web/src/components/subtitle/consent/GenerationConsentView.spec.tsx`（M）
- `apps/web/src/components/subtitle/consent/ConfirmGenerationDialog.tsx`（M）
- `apps/web/src/components/subtitle/consent/ConfirmGenerationDialog.spec.tsx`（M）
- `apps/web/src/components/subtitle/consent/ModelPicker.tsx`（M，AC #5 字級）
- `apps/web/src/routes/test/-gallery.fixtures.tsx`（M）
- `_bmad-output/implementation-artifacts/sub-6-12-f15-f18-money-traps-pen-prompt.md`（A）
- `tests/visual/.../generation-consent/*/default-visual-darwin.png`（M ×11）
- `tests/visual/.../generation-consent/*/default-visual-linux.png`（D ×11，走 CI bootstrap）
