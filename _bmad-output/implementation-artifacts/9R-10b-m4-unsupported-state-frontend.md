# Story 9R-10b-M4: 部署未啟用時的誠實狀態（前端落地）

Status: ready-for-dev

## Story

As a **NAS 使用者，我的 Vido 跑在出貨預設的 `legacy` 模式**,
I want **在媒體庫設定裡看到「自動處理免費字幕」這個選項存在、為什麼我現在不能用它，以及我要做什麼才能用**,
so that **我不會勾了一個永遠不會發生的東西，也不會因為它整段消失而根本不知道有這個功能**。

### 這個 story 從哪來

9R-10b 的 80-agent 對抗補審發現 **M4**：`auto_subtitle` 欄位在任何模式都寫得進資料庫，
但真正執行它的 `AutoGenerator` **只在 `main.go` 的 `if cfg.SubtitlePipelineEnabled()` 區塊裡才被建構**，
而 `VIDO_SUBTITLE_PIPELINE_MODE` 的出貨預設是 `legacy`（`apps/api/internal/config/config.go:160`）。

PR #250 的處理是保守的：**legacy 模式下整個欄位隱藏**。不說謊了，但也不再告訴使用者「有這個功能、你可以開」。
Sally 於 2026-08-21 完成設計裁定（PR #252，設計稿 `J5-D`），本 story 是那份裁定的落地。

---

## Acceptance Criteria

### AC #1 —— Modal 欄位從「整段隱藏」改為「永遠渲染」

**Given** `LibraryEditModal` 目前在 `:230` 以 `{autoSubtitleSupported && (...)}` 包住整個
`data-testid="library-auto-subtitle-field"` 區塊，
**When** `autoSubtitleSupported === false`，
**Then** 該區塊**仍然渲染**（`getByTestId('library-auto-subtitle-field')` 找得到），只是進入停用態。

### AC #2 —— 停用態的視覺（J5-D 區塊 D）

**Given** `autoSubtitleSupported === false`，**Then**：

| 元素 | 狀態 |
|---|---|
| checkbox | `disabled`，不可點擊、不可用鍵盤切換 |
| 開關標籤「新檔入庫後，自動完成免費的字幕處理」 | 內容**一字不改**，色彩降為 `--text-disabled` |
| 兩句凍結說明 | 內容**一字不改**，色彩降為 `--text-disabled` |
| **新增**通知列 | `--info-tint` 底、4px 左緣 `--info`、圓角 `--radius-sm` |

通知列位置：**checkbox 列正下方、兩句說明之上**。
理由：使用者的視線先落在灰掉的 checkbox 上問「為什麼」，答案要緊接著出現；
兩句說明接著回答「那它本來會幫我做什麼」。

### AC #3 —— 通知列文案與後端 409 逐字相同

**Given** 停用態，**Then** 通知列包含兩句，且與 `apps/api/internal/handlers/subtitle_pipeline_handler.go:112-113`
**逐字相同**：

```
字幕生成管線尚未啟用，這個選項無法變更。
請將 VIDO_SUBTITLE_PIPELINE_MODE 設為 pipeline 後重啟伺服器。
```

> ⚠️ 第一句的後半「，這個選項無法變更。」是本設計新增的（後端原句是「字幕生成管線尚未啟用」作為
> message、第二句作為 suggestion）。**第二句必須一字不差**。

`VIDO_SUBTITLE_PIPELINE_MODE` 以等寬字（`font-mono`）呈現，讓使用者一眼看出那是要照抄的字串。

**Why**：同一個動作用同一組字。一個透過 API 撞過 409 `AI_NOT_CONFIGURED` 的使用者，
在 modal 裡讀到的會是他認得的句子。這是 sub-2-1a 的先例（同 error code、不同 suggestion，
因為**下一步動作不同**）。

### AC #4 —— `LibraryCard` 取得 capability 並改走三態（J5-D 區塊 E）

**Given** `LibraryCard.tsx:115` 目前**只讀 `library.autoSubtitle`、沒有任何 capability 閘門**，
**When** 伺服器是 `legacy` 而該媒體庫 `auto_subtitle=1`（在 pipeline 模式勾過、或經 API 寫入），
**Then** footer 目前會亮綠字「· 自動處理免費字幕」—— **宣稱一件沒在發生的事**。

修正後三態：

| 伺服器 | 媒體庫勾選 | footer 末段 | 色彩 |
|---|---|---|---|
| 支援 | 已勾 | `· 自動處理免費字幕` | `--success` |
| **不支援** | **已勾** | `· 自動處理免費字幕（伺服器未啟用）` | `--warning` |
| 任一 | 未勾 | 整段不出現 | — |

**括號不可省。** 少了它會被讀成「你沒勾」，而使用者明明勾了 —— 那是最糟的誤讀。

「不支援 ＋ 未勾」刻意留白：使用者沒表達過意願，卡片沒有義務替他擔心。

### AC #5 —— capability 從既有 query 傳進卡片，零新資料來源

**Given** `MediaLibraryManager.tsx:14` 已經有 `const { data } = useMediaLibraries()`，
**Then** 以 `data?.autoSubtitleSupported !== false` 經 **prop** 傳給 `<LibraryCard>`（`:50`）。

**不得**新增 hook、context、prop drilling 中繼層，**不得**新增 API 呼叫。
`auto_subtitle_supported` 已隨 PR #250 出貨在 `GET /api/v1/libraries` 的 list payload。

`!== false` 的語義**必須與 modal 一致**：欄位缺席讀作「未知」→ 維持既有（支援）行為。
把一個已出貨的控制項因為少一個 key 就藏起來，是比顯示更糟的失敗。

### AC #6 —— 既有行為不得回歸

**Given** `autoSubtitleSupported === false`，
**Then** update／create payload 仍**省略** `autoSubtitle`（而非送 `false`）——
以免清掉使用者在 pipeline 模式下做過的選擇。
`LibraryEditModal.tsx:66,72` 的 `...(autoSubtitleSupported ? { autoSubtitle } : {})` **不得改動**，
PR #250 已有測試釘住（`omits autoSubtitle from the update payload when unsupported`）。

### AC #7 —— 測試

1. `LibraryEditModal.spec.tsx:178`「hides the opt-in when the deployment does not run the auto lane」
   **會轉紅，而且應該轉紅** —— 設計裁定改變了，不是回歸。改為斷言**停用態**而非不存在。
2. 新增：停用態下 checkbox `toBeDisabled()`、通知列兩句 `toBeInTheDocument()`、
   env var 以等寬字呈現。
3. `LibraryCard.spec.tsx`（現有 3 例）新增三態覆蓋：綠／琥珀／整段不出現。
4. Rule 16：用最具體的 matcher（`toBeDisabled` / `toBeInTheDocument` / `toHaveTextContent`），
   **不得**用 `toBeTruthy`、不得只斷言 `NotPanics` 型的空殼。

### AC #8 —— 閘門全綠

`pnpm nx test web` ／ `pnpm nx lint web` ／ `pnpm format:check` ／ `tsc --noEmit`（**147 = main 基準，零新增**）。
**BE 零改動** ⇒ `nx test api` 不在必跑範圍，但若順手跑過請記錄。

---

## Tasks / Subtasks

- [ ] **Task 1 —— Modal 停用態（AC #1, #2, #3）**
  - [ ] `LibraryEditModal.tsx:230` 拆掉 `{autoSubtitleSupported && (` 包裹，改為永遠渲染
  - [ ] checkbox 加 `disabled={!autoSubtitleSupported}`
  - [ ] 標籤與兩句說明的 className 依 `autoSubtitleSupported` 切換至 `--text-disabled`
  - [ ] 新增通知列（`--info-tint` 底 / 4px 左緣 `--info` / `--radius-sm`），只在不支援時渲染
  - [ ] 通知列兩句文案照抄，env var 包在 `<code>`／`font-mono` span
  - [ ] Rule 21 檔頭補 `J5-D (alrIw)`
  - [ ] RED 先行：先寫斷言停用態的測試，看它紅，再實作

- [ ] **Task 2 —— 卡片三態（AC #4, #5）**
  - [ ] `LibraryCardProps` 新增 `autoSubtitleSupported: boolean`
  - [ ] `LibraryCard.tsx:115` 改為三態分支
  - [ ] `MediaLibraryManager.tsx:50` 傳 `autoSubtitleSupported={data?.autoSubtitleSupported !== false}`
  - [ ] Rule 21 檔頭補 `J5-D (alrIw)`（**保留**現有的 6UCtX 段落，見 Discovery Triage ③）

- [ ] **Task 3 —— 測試（AC #7）**
  - [ ] 改寫 `LibraryEditModal.spec.tsx:178`
  - [ ] 新增 modal 停用態 3 例
  - [ ] 新增 `LibraryCard.spec.tsx` 三態 3 例
  - [ ] **Fault injection 反證**：每個新守衛各注入一次缺陷，確認對應測試轉紅

- [ ] **Task 4 —— 閘門與 UX 比對（AC #8）**
  - [ ] `nx test web` / `nx lint web` / `format:check` / `tsc --noEmit` 全綠
  - [ ] **Step 9 UX 比對**：對照 `_bmad-output/screenshots/flow-j-specs/j5-d.png`
        （⚠️ 規格頁，非 mockup —— 比對的是文案與色彩規則，不是像素）

**跨端拆分檢查**：FE 4 task ／ BE **0** task ⇒ 門檻（雙邊皆 >3）未達，**不拆**。

---

## Dev Notes

### 資料流（已出貨，不要重建）

```
config.go:160  VIDO_SUBTITLE_PIPELINE_MODE (預設 legacy)
      ↓
cfg.SubtitlePipelineEnabled()                      config/subtitle_pipeline.go:46
      ↓  handlers.WithAutoSubtitleSupport(...)      main.go（PR #250 已接線）
GET /api/v1/libraries → { libraries, auto_subtitle_supported }
      ↓  snakeToCamel
useMediaLibraries() → data.autoSubtitleSupported    hooks/useMediaLibrary.ts:17
      ↓
LibraryEditModal:39（已存在）      MediaLibraryManager:14（已存在 data）→ LibraryCard（❌ 缺這一段）
```

**唯一缺口就是最後那一段。** 其餘全部已出貨。

### 顏色是一套規則，不是三個各自的決定

| | 意思 |
|---|---|
| `--success` | **正在發生** |
| `--warning` | **你要求了，但沒在發生**（系統與使用者意願不一致的唯一一格） |
| 不出現 | **你沒要求** |

改動任何一格前先確認它仍符合這條規則。

### 前一個 story（9R-10b）的教訓，直接適用

1. **fake 不可比真物件寬容** —— CR H2：fake 對查不到的 id 回 `(nil, nil)`，
   真 repo 回包 `sql.ErrNoRows` 的 error，於是測試在斷言一個生產程式碼沒有的行為。
2. **測試要能失敗** —— `assert.NotPanics` 型的空殼測試無法觀察它宣稱釘住的行為（補審 F14/F16）。
3. **每個守衛都要 fault injection 反證** —— 補審修復的六項全部注入過缺陷確認轉紅。
4. **FE mock 的身分穩定性** —— `LibraryEditModal.spec.tsx` 的 `librariesQuery` 是
   **module-level 常數**：TanStack Query 回傳的 data 在 re-render 間是 referentially stable，
   modal 的 hydrate effect 依賴那個 identity。每次 render 重建物件會讓 effect 重跑、
   靜默重置表單狀態，看起來就像元件壞了。
5. **capability mock 要用 `mockReturnValue` 不是 `mockReturnValueOnce`** ——
   元件會 render 超過一次（hydrate effect 觸發 re-render），一次性回傳值會被第一次 render 吃掉，
   後續 render 拿到預設值。這個坑 PR #250 踩過並已在 `beforeEach` 修好，沿用即可。

### 文案是硬性的

- 兩句通知的**第二句**與 `subtitle_pipeline_handler.go:113` 逐字相同
- 卡片新段的**括號不可省**
- 凍結的三句（開關標籤 ＋ 兩段說明）**一字不改**，停用態只改色彩
- 全部新增文案**不得出現「掃描」二字**（sub-4-3 AC #6 / 2026-08-07 誤解的原產地）

### Project Structure Notes

- 三個檔案全在 `apps/web/src/components/settings/`，與既有結構一致，**無新目錄、無新模組**
- 測試 co-location（Rule 9）：`.spec.tsx` 與元件同層，已符合
- **無 API 變更** ⇒ Rule 10（版本）、Rule 20（contract bump）、Rule 7（error code）皆 **N/A**
- Rule 18（API 邊界 case 轉換）：`auto_subtitle_supported` → `autoSubtitleSupported`
  由既有 `snakeToCamel` 處理，**不需新增映射**

### Time-dependent visual coverage

**N/A —— no wall-clock-reading components touched.**
`LibraryEditModal` / `LibraryCard` / `MediaLibraryManager` 皆不讀
`Date.now()` / `new Date()` / `Date.UTC()` / `Date.parse()`。

### 最新技術資訊

**無需研究。** 本 story 不引入任何新 library、framework 或 API；
用的是既有的 React 18 + TanStack Query + Tailwind CSS 變數，版本不變。
刻意不在此填充泛用的 React 最佳實務 —— 那會稀釋上面真正 story-specific 的約束。

### References

- [Source: `_bmad-output/implementation-artifacts/9R-UX-auto-subtitle-unsupported-state-design.md`] —— Sally 兩項裁定、決策矩陣、文案定稿
- [Source: `_bmad-output/screenshots/flow-j-specs/j5-d.png`] —— 設計稿 `J5-D`（規格頁）
- [Source: `_bmad-output/implementation-artifacts/9R-10b-on-add-autotrigger.md#🔎 補審 Medium 逐條複查`] —— M4 的原始發現與 PR #250 的保守處理
- [Source: `apps/api/internal/handlers/subtitle_pipeline_handler.go:112-113`] —— 兩句文案的**唯一權威來源**
- [Source: `apps/api/internal/config/config.go:160`] —— `VIDO_SUBTITLE_PIPELINE_MODE` 預設 `legacy`
- [Source: `docs/deployment.md:80,134`] —— env-only、無設定頁入口、需重啟
- [Source: `project-context.md#Rule 21`] —— 檔頭 design ref
- [Source: `project-context.md#Rule 16`] —— 測試斷言品質
- [Source: `project-context.md#Rule 24`] —— Discovery Triage

---

## Dev Agent Record

### Agent Model Used

_（dev 填寫）_

### Debug Log References

### Completion Notes List

### Discovery Triage

**是的，本 story 準備期間發現了範圍外的工作 —— 1 筆，lane ③。**

- **③ backlog-with-carry-forward-link** —— `settings/` 元件的 design ref 指向錯誤畫面。
  `LibraryCard.tsx:1` 的檔頭是 `Screen 10 Settings Desktop (6UCtX)`，
  但 9R-UX-auto-generation-toggle-design 已查證 **`6UCtX` 實為連線設定（qBittorrent）頁**，
  側欄七項裡沒有「媒體庫掃描」；真正的畫面是 `KvZSc`(E1-D) / `uABWl`(E1-M)。
  **這不是 LibraryCard 專屬的 bug** —— `grep -rl 6UCtX apps/web/src` 命中 **10+ 個 settings 元件**，
  是全域一致的錯誤引用。只修一個檔案反而製造不一致。
  → 已於發現時立案 `bugfix-settings-design-ref-6uctx-sweep`（`sprint-status.yaml`）。
  **本 story 只在檔頭追加 `J5-D (alrIw)`，不動 6UCtX 那一段**，留給 sweep 統一處理。

### File List

_（dev 填寫；預期 5 個檔案）_

| 檔案 | 預期動作 |
|---|---|
| `apps/web/src/components/settings/LibraryEditModal.tsx` | modified |
| `apps/web/src/components/settings/LibraryEditModal.spec.tsx` | modified |
| `apps/web/src/components/settings/LibraryCard.tsx` | modified |
| `apps/web/src/components/settings/LibraryCard.spec.tsx` | modified |
| `apps/web/src/components/settings/MediaLibraryManager.tsx` | modified |

---

## Change Log

| 日期 | 變更 |
|---|---|
| 2026-08-21 | CREATED（create-story, Bob）—— 來源：9R-10b 補審 M4 → Sally 設計裁定（PR #252, `J5-D`）。跨端拆分檢查 FE 4 / BE 0 ⇒ 不拆。Discovery Triage 發現 1 筆 lane ③（`6UCtX` 全域錯誤引用，已立案 sweep）。 |
