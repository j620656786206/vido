# Story 9R-10b-M4: 部署未啟用時的誠實狀態（前端落地）

Status: review

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

- [x] **Task 1 —— Modal 停用態（AC #1, #2, #3）**
  - [x] `LibraryEditModal.tsx:230` 拆掉 `{autoSubtitleSupported && (` 包裹，改為永遠渲染
  - [x] checkbox 加 `disabled={!autoSubtitleSupported}`
  - [x] 標籤與兩句說明的 className 依 `autoSubtitleSupported` 切換至 `--text-disabled`
  - [x] 新增通知列（`--info-tint` 底 / 4px 左緣 `--info` / `--radius-sm`），只在不支援時渲染
  - [x] 通知列兩句文案照抄，env var 包在 `<code>`／`font-mono` span
  - [x] Rule 21 檔頭補 `J5-D (alrIw)`
  - [x] RED 先行：先寫斷言停用態的測試，看它紅，再實作

- [x] **Task 2 —— 卡片三態（AC #4, #5）**
  - [x] `LibraryCardProps` 新增 `autoSubtitleSupported: boolean`
  - [x] `LibraryCard.tsx:115` 改為三態分支
  - [x] `MediaLibraryManager.tsx:50` 傳 `autoSubtitleSupported={data?.autoSubtitleSupported !== false}`
  - [x] Rule 21 檔頭補 `J5-D (alrIw)`（**保留**現有的 6UCtX 段落，見 Discovery Triage ③）

- [x] **Task 3 —— 測試（AC #7）**
  - [x] 改寫 `LibraryEditModal.spec.tsx:178`
  - [x] 新增 modal 停用態 3 例
  - [x] 新增 `LibraryCard.spec.tsx` 三態 3 例
  - [x] **Fault injection 反證**：每個新守衛各注入一次缺陷，確認對應測試轉紅

- [x] **Task 4 —— 閘門與 UX 比對（AC #8）**
  - [x] `nx test web` / `nx lint web` / `format:check` / `tsc --noEmit` 全綠
  - [x] **Step 9 UX 比對**：對照 `_bmad-output/screenshots/flow-j-specs/j5-d.png`
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

Claude Opus 5 (1M context) — `claude-opus-5[1m]`

### Debug Log References

無阻斷性失敗。唯一一次誤判記於 Completion Notes「方法論」。

### Completion Notes List

**🔗 AC Drift: FOUND** —— `9R-10b-on-add-autotrigger` 補審 M4（該檔 `:698`）：
「FE 在 `!== false` 時**才渲染**該欄位」（＝不支援就隱藏）
→ 本 story 改為「**永遠渲染**，不支援時為停用態」。
同一個 `autoSubtitleSupported` 述詞、**相反的可觀察行為**。
這是刻意的設計裁定變更（Sally 2026-08-21，PR #252），不是回歸。
`!== false` 的**語義本身未變**（缺席＝未知＝視為支援），改的是「不支援時要顯示什麼」。
grep pattern：`整段隱藏|autoSubtitleSupported && |才渲染該欄位|hides the opt-in`。

**📎 Contract Stamps: NONE** —— 本 story 無 `[@contract-v*]` 戳記，也未消費任何帶戳記的契約。
上游 `9R-10b` 的 `[@contract-v1]` 戳在 `ProcessItemOptions`（Go 內部 struct），
本 FE story 不碰它。本 story 讀的 wire 欄位 `auto_subtitle_supported`
（`media_libraries_handler.go:88`）**未帶戳記** —— 它是 PR #250 的 additive-on-v1 新增，
形狀未變。零 API 變更 ⇒ Rule 7／10／20 皆 N/A。

**🎭 A11y Pre-Flight: PASS**（3 個元件檢查，touched 檔案 jsx-a11y 警告 **0**，本 story 引入 0）
- 響應式圖片：N/A（無圖片）
- Modal 焦點管理：本 story 未改動焦點行為
- `aria-live`：N/A（非非同步揭露內容）
- 自訂 widget 鍵盤／ARIA：N/A（原生 checkbox）
- **本 story 自己引入的一項，已修**：`disabled` 的 input **完全不進 tab order**，
  所以解釋「為什麼是灰的」那段通知，鍵盤／螢幕閱讀器使用者根本觸及不到。
  已補 `aria-describedby` 把 checkbox 綁到通知列（含測試，拿掉即轉紅）。

**🎨 UX Verification: PASS** —— 以 Pencil MCP **直接讀 `J5-D` 的文字節點**，
與實作字串逐條程式化比對（12/12 命中），非目測：

| 區域 | 設計稿（J5-D） | 實作 | 相符 |
|---|---|---|---|
| 通知列第 1 句 | `字幕生成管線尚未啟用，這個選項無法變更。` | 同 | ✅ |
| 通知列第 2 句 | `請將 ` + `VIDO_SUBTITLE_PIPELINE_MODE` + ` 設為 pipeline 後重啟伺服器。` | 同（env var 獨立 span + `font-mono`） | ✅ |
| 凍結三句 | 一字不改 | 一字不改，僅降色 | ✅ |
| 停用色 | `$text-disabled` | `var(--text-disabled)` | ✅ |
| 通知列 | `$info-tint` 底 + 4px 左緣 `$info` | `bg-[var(--info)]/10` + `border-l-4 border-[var(--info)]` | ✅ |
| 卡片綠態 | `自動處理免費字幕` `$success` | 同 | ✅ |
| 卡片琥珀態 | `自動處理免費字幕（伺服器未啟用）` `$warning` | 同 | ✅ |
| 卡片未勾 | 整段不出現 | 同 | ✅ |

**Pre-existing failures**：無。`nx test web` 與 `nx test api` 在改動前後皆全綠。

**方法論 —— 一次誤判，記錄下來**：
反證 `MediaLibraryManager` 的 prop 接線時，我先跑了
`vitest run .../MediaLibraryManager.spec.tsx` —— **那個檔案當時並不存在**，
vitest 對「找不到測試檔」回傳非零，於是它看起來像「RED ✅」。
**那是假的紅**，等於用一個不存在的測試證明守衛有效。
發現後補了真的 `MediaLibraryManager.spec.tsx`（3 例）再重跑，三個變異才是真的轉紅。
這正是 9R-10b 補審「測試要能失敗」那條教訓的變形：
**先確認測試檔存在且會跑，再相信它的紅。**

**Fault injection 全紀錄（10/10 轉紅）**

| 注入的缺陷 | 轉紅的測試檔 |
|---|---|
| checkbox 不再 `disabled` | LibraryEditModal.spec |
| 通知列永不渲染 | LibraryEditModal.spec |
| 通知列永遠渲染（支援時也出現） | LibraryEditModal.spec |
| env var 不用等寬字 | LibraryEditModal.spec |
| 凍結說明改回硬編色 | LibraryEditModal.spec |
| 開關標籤改回硬編色 | LibraryEditModal.spec |
| 拿掉 `aria-describedby` | LibraryEditModal.spec |
| 卡片忽略 capability（永遠綠） | LibraryCard.spec |
| 卡片括號被省略 | LibraryCard.spec |
| 未勾時也顯示狀態 | LibraryCard.spec |
| manager 硬傳 `true` | MediaLibraryManager.spec |
| manager 改成 `=== true`（把 `undefined` 當不支援） | MediaLibraryManager.spec |

**一個查過但不是問題的東西**：`LibraryCard.tsx:160` 有「掃描」二字
（`同時移除已掃描的媒體資料`）—— 那是**刪除確認對話框**的既有文案，
講的是「已掃描的媒體資料」，與字幕 opt-in 無關且在自身脈絡下正確。
sub-4-3 AC #6 的約束範圍是 opt-in 文案，既有測試釘的也正是 footer 範圍（已綠）。
我的第一版 grep 掃全檔太粗，查證後未改動。

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

| 檔案 | 動作 |
|---|---|
| `apps/web/src/components/settings/LibraryEditModal.tsx` | modified —— 永遠渲染＋停用態＋通知列＋`aria-describedby`；Rule 21 檔頭補 `J5-D (alrIw)` |
| `apps/web/src/components/settings/LibraryEditModal.spec.tsx` | modified —— 改寫 1 例、新增 6 例（13 → 19） |
| `apps/web/src/components/settings/LibraryCard.tsx` | modified —— `autoSubtitleSupported` prop ＋ footer 三態；Rule 21 檔頭補 `J5-D (alrIw)` |
| `apps/web/src/components/settings/LibraryCard.spec.tsx` | modified —— 既有 3 例補新 prop、新增 4 例（3 → 7） |
| `apps/web/src/components/settings/MediaLibraryManager.tsx` | modified —— 傳遞 capability（一個 prop，沿用既有 `data`） |
| `apps/web/src/components/settings/MediaLibraryManager.spec.tsx` | **new** —— 3 例，接線的**唯一**覆蓋（沒有它，硬傳 `true` 不會被任何測試發現） |
| `_bmad-output/implementation-artifacts/9R-10b-on-add-autotrigger.md` | （AC drift reference — see Completion Notes） |

---

## Change Log

| 日期 | 變更 |
|---|---|
| 2026-08-21 | **Task 1（AC #1/#2/#3）** —— `LibraryEditModal` 拆掉 `{autoSubtitleSupported && (` 整段隱藏，改為永遠渲染；checkbox `disabled`、標籤與凍結兩句降 `--text-disabled`、新增 `--info` 通知列（第二句與 `subtitle_pipeline_handler.go:113` 逐字相同、env var 獨立 `font-mono` span）。RED 先行：3 例先紅再實作。 |
| 2026-08-21 | **Task 2（AC #4/#5）** —— `LibraryCardProps` 新增 `autoSubtitleSupported`，footer 改三態（`$success`／`$warning`／不出現）；`MediaLibraryManager` 以既有 `data` 傳遞，零新 hook、零新請求。 |
| 2026-08-21 | **Task 3（AC #7）** —— 改寫 `hides the opt-in...` 為斷言停用態；modal 新增 6 例、card 新增 4 例、**新建** `MediaLibraryManager.spec.tsx` 3 例。**10 項 fault injection 全數轉紅**。過程中發現並修正一次**假紅**（測試檔不存在時 vitest 回非零）。 |
| 2026-08-21 | **Task 4（AC #8）** —— `nx test web` 全綠／`nx test api` 全綠／`nx lint web` 綠／`format:check` 綠／`tsc --noEmit` **147＝main 基準**。A11y pre-flight 補 `aria-describedby`（disabled input 不進 tab order）。UX 比對以 MCP 直讀 `J5-D` 文字節點程式化比對 12/12。 |
| 2026-08-21 | **CR 自審（Amelia）** —— 0H/4M/3L ＋ 1 項交回設計。M1 改用 `--info-tint`（設計系統既有 token，勿手工兌色）／M2 gallery fixture 補必填 prop（cast 使 tsc 抓不到，PR #250 CR M3 同類）／M3 補 AC #2 順序測試／M4 修正自己違反的 Rule 16 `toBeTruthy`／L3 提出重複 class。L1（className 斷言證明不了樣式）與 L2（`text-[13px]`）記錄為已知取捨。`--text-disabled` 的 sub-AA 對比問題**交回 Sally**（J5-D 明訂，非 dev 缺陷）。 |
| 2026-08-21 | CREATED（create-story, Bob）—— 來源：9R-10b 補審 M4 → Sally 設計裁定（PR #252, `J5-D`）。跨端拆分檢查 FE 4 / BE 0 ⇒ 不拆。Discovery Triage 發現 1 筆 lane ③（`6UCtX` 全域錯誤引用，已立案 sweep）。 |

---

## Senior Developer Review (AI)

**日期：** 2026-08-21 ｜ **審查者：** Amelia（⚠️ **自審** —— 實作者與審查者同一 context，結構上弱於換模型／換 context 審查）
**結果：** **APPROVED-WITH-FIXES** —— 0 High / 4 Medium / 3 Low ＋ 1 項交回設計裁定
**修復：** M1–M4 ＋ L3 全數修復並以 fault injection 反證；L1／L2 記錄為已知取捨

### 強制檢查

| 檢查 | 結果 |
|---|---|
| 🔒 Rule 7 Wire Format | **N/A**（範圍內無 Go error-code 檔） |
| 🔒 Rule 20 Contract Bump | **N/A**（無戳記 bump） |
| 🔒 Rule 25 Mega-line | **N/A**（未觸及 `project-context.md`） |
| Git vs Story File List | **相符，零落差** |
| AC 實作查核 | **8/8 IMPLEMENTED** |
| Task `[x]` 稽核 | **4/4 真的做完** |

### Findings

**🟡 M1 —— 手工兌了一個設計系統已經有的顏色** · ✅ FIXED
實作寫 `bg-[var(--info)]/10`，但 `styles.css:45` 已定義 `--info-tint: #06b6d41f`，
設計稿寫的也正是 `$info-tint`。最接近的鄰居 `ApiKeysForm.tsx`（同為設定頁提示橫幅）
用 `bg-[var(--warning-tint)]` / `bg-[var(--info-tint)]`。
等於自行調了個近似值（10% vs 12.2%）並偏離同類元件慣例。
**修復**：改用 `bg-[var(--info-tint)]`。

**🟡 M2 —— gallery fixture 沒補上新的必填 prop** · ✅ FIXED
`autoSubtitleSupported` 在 `-gallery.fixtures.tsx` 出現 **0 次**，
但它已是 `LibraryCard` 的**必填** prop。該檔把元件 cast 成
`ComponentType<Record<string, unknown>>`，**tsc 因此抓不到缺漏**。
今天沒事純屬運氣 —— fixture 是 `autoSubtitle: false`，狀態段根本不渲染；
一旦有人改成 `true`，gallery 會靜靜畫出琥珀色「未啟用」態，視覺基準跟著漂移。
**前科**：PR #250 CR M3 是同一個檔案、同一類問題。
**修復**：補 `autoSubtitleSupported: true` ＋ 檔內註記說明為何要手動同步。

**🟡 M3 —— AC #2 的「順序」有理由卻沒有測試** · ✅ FIXED
Story 明寫通知列要在 checkbox 與兩句說明**之間**，並寫了理由；但零測試釘住。
同一個 spec 檔裡就有現成寫法（`places the opt-in LAST` 用 `compareDocumentPosition`）。
**修復**：新增 `puts the notice between the control and the description, in that order`。
**反證**：通知列改為不渲染 → 轉紅。

**🟡 M4 —— 實作者自己違反 Rule 16** · ✅ FIXED
`expect(notice.id).toBeTruthy()` —— Rule 16 白紙黑字把 `toBeTruthy` 列為 WRONG。
**修復**：改為斷言確切 id 字串兩端（notice 的 `id` 與 checkbox 的 `aria-describedby`）。
**反證**：改掉 notice 的 id → 轉紅（原寫法**不會**紅，因為任何非空 id 都 truthy）。

**🟢 L3 —— 三元式兩邊重複 `font-medium`** · ✅ FIXED（提出到樣板字串外層）

**🟢 L1 —— className 斷言證明不了樣式真的算出來** · ⏭️ 記錄為已知取捨
`className.toContain('font-mono')` / `toContain('text-[var(--text-disabled)]')`
測的是 class **字串**，Tailwind 沒生出規則它照樣綠
（`feedback_css_verify_before_iterate` 那條的變形）。
jsdom 下沒有更好的觀察點；真正的守衛是 visual regression 基準。不修，記錄。

**🟢 L2 —— `text-[13px]` 是檔內唯一的任意字級** · ⏭️ 記錄為已知取捨
其餘皆 `text-xs`/`text-sm`。設計稿明訂 13px，故保留而非四捨五入到既有級距。

### ⚖️ 交回設計裁定（不由 dev 自行處理）

**`--text-disabled` 用在兩句凍結說明上，可能讓「該去開這個功能的理由」讀不清。**

`styles.css:47` 對該 token 的註解是 **「intentionally sub-AA (TC-1)」** ——
設計系統自己標記它對比度不足。而 J5-D 明訂停用態要把兩句說明降到這個色。

問題在於：**那兩句正是說服使用者去改 env var 的理由**。
WCAG 對「停用元件」的對比豁免給的是**控制項本身**，不涵蓋旁邊的說明散文。
換句話說，我們把「你為什麼該去開這個功能」用一個已知讀不清的顏色印出來。

這是 Sally 的裁定（J5-D 區塊 D 明文），**不是 dev 缺陷** ⇒ 不自行修改。
→ 立案 `9R-UX-disabled-state-description-contrast`（Rule 24 ②）。

**⚖️ 已裁定並修復（2026-08-21，同 PR）** —— Sally **自我更正**：初版把兩句說明也降色是錯的。

實測（背景 `#1b2336`）：`$text-disabled` **3.55:1**（12px 內文需 4.5 ⇒ 不通過）／
`$text-muted` 6.71 ✅／`$text-secondary` **7.47** ✅。

裁定：**停用的是「控制項」，不是「說明」**。checkbox 與**開關標籤**降色**保留**
（標籤是該 checkbox 的可及名稱，屬 WCAG 1.4.3 明文豁免的 inactive user interface component）；
**兩句說明維持 `$text-secondary` 不降色**。
不折衷用 `$text-muted` 的理由：與 `$text-secondary` 視覺差幾乎看不出來，
會是「程式碼裡有、眼睛看不到」的區別。
**落地**：兩個說明 `<p>` 改為無條件 `text-[var(--text-secondary)]`；
測試改名為 `dims the CONTROL but never the description`，**兩側都釘**
（把說明降回去 → 紅；把標籤的降色拿掉 → 也紅）。
`.pen` 的 `J5-D` 同步更正（4 節點 ＋ 區塊F 理由 ⑥），`j5-d.png` 重匯。

### 修後閘門（全綠）

`pnpm nx test web` ✅ ｜ `pnpm nx lint web` ✅ ｜ `pnpm format:check` ✅ ｜
`tsc --noEmit` **147 ＝ main 基準，零新增**。
修復後 fault injection **3/3 轉紅**（通知列不渲染／拿掉 `aria-describedby`／notice id 改名）。

