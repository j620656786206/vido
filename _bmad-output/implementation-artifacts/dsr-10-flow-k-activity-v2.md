# Story DSR.10: Flow K 活動中心——程式碼與設計稿雙向對齊 K1–K4

Status: review

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As someone whose NAS is quietly scanning, translating and downloading in the background,
I want 活動中心顯示的每一個數字都是系統真的量到的，而且設計稿跟出貨的畫面是同一件事,
so that 我看到的「3 個進行中」不是有人在稿上編出來的。

## Context

`epic-dsr` 的第二張（dsr-12 已 done）。規模小——`components/activity/` 只有 4 個檔，扣掉 spec 是 3 個。

⚠️ **這張跟 dsr-12 有一個結構性差別：方向不是單向的。**

dsr-12 是「碼追稿」。這次比對下來，**錯的有一半在稿上**——設計稿畫了一個系統量不到的讀數，還自己編了手機版的縮短文案。所以下面每一條都標了方向：`碼→稿`（程式碼要改）或 `稿→碼`（設計稿要改）。dev 不要無腦把程式碼改成跟稿一樣。

⚠️ **驗收基準是 `.pen` 節點值，不是 `_bmad-output/screenshots/flow-k-activity-v2/` 的 PNG。** 那 5 張現在是 400px 縮圖，讀不到字。Task 1 先修這件事，後面的比對才做得下去。

## Acceptance Criteria

### 設計稿節點（逐字抄，不要重查）

`K1-D-v2 (kMeWS)` · `K1-M-v2 (QIwY1)` · `K2-D-v2 骨架 (suCiI)` · `K3-D-v2 空狀態 (DZnSv)` · `K4-D-v2 區塊載入失敗 (M6ra92)` · `Component/ActivityRow-v2 (fF8nX)`

### 對照表

| # | 項目 | 設計稿 | 程式碼 | 方向 |
| --- | --- | --- | --- | --- |
| 1 | 下載列的速度 | `3 個進行中 · 5 個排隊 · **12.4 MB/s**` | 沒有速度 | **稿→碼** |
| 2 | 下載列的錯誤／暫停數 | 沒畫 | `N 個錯誤`（硃砂）`· N 個暫停`（弱化） | **稿→碼** |
| 3 | 手機版文案 | 自己縮短過（見下） | 同一份字串，CSS `truncate` | **稿→碼** |
| 3b | 進行中第三列說明 | `你的名字 · S1E3 — 簡轉繁 + 語意校正中` | `Detail` 是原始資料（片名），不是造句 | **稿→碼** ⚠️新增 |
| 4 | 手機第五格標籤 | `More`（英文） | `更多` | **稿→碼** |
| 5 | 手機缺「活動記錄」區塊 | 只有三區 | 四區都渲染，不分斷點 | **稿→碼** |
| 6 | 「批次生成字幕」CTA | 兩張稿都沒畫 | header 有，且是 Activity 唯一的批次入口 | **稿→碼** |
| 7 | 待處理列說明 | `8 個檔案無法自動比對中繼資料`（**假的**：那是佇列不是失敗） | `{n} 個項目待處理` | **稿→碼** ⚠️翻向 |
| 8 | 進行中第二列標題 | `批次字幕搜尋` | `批次字幕` | **碼→稿** |
| 9 | 進行中第三列標題 | `AI 字幕校正`（**假的**：transcription 是生成不是校正） | `字幕生成中` | **稿→碼** ⚠️翻向 |
| 10 | 進行中第三列右槽 | `校正中` | `進行中` | **碼→稿** |
| 11 | 空狀態說明 | `掃描、字幕**校正**與下載工作…` | `掃描、字幕與下載工作…` | **稿→碼** ⚠️翻向 |
| 12 | `ActivityHub.tsx` 檔頭 | — | 寫 `Screen A1-D-v2 (kMeWS)`，但 kMeWS 現在叫 **K1-D-v2** | **碼→稿** |
| 13 | `ActivityStates.tsx` 檔頭 | — | 寫 `Screen A4-D-v2 (suCiI)`，但 suCiI 現在叫 **K2-D-v2**；docstring 還寫 A5/A6 | **碼→稿** |
| 14 | `ActivityRow.tsx` 檔頭 | — | `Component/ActivityRow-v2 (fF8nX)` **已驗證正確** | ✅ 不動 |
| 15 | 字級 | Body 14 / Label 12 | 6 處 `text-[13px]` | **碼→稿** |
| 16 | K4 區塊失敗文案 | `無法載入，請稍後再試` ＋ `重試` | 逐字相同 | ✅ |
| 17 | 進行中區塊標題帶計數 | `進行中` ＋ `3` | `SectionShell title count` | ✅ |
| 18 | 頁首副標 | `媒體庫的所有背景工作 — 掃描、字幕、解析與下載` | 逐字相同 | ✅ |

---

1. **Flow K 的稿要讀得到。** `scripts/export-pen-screenshots.py` 的 `READABLE_FLOWS` 加入 `"flow-k-activity-v2"`，重跑匯出，5 張從 400px 長邊變成 2x／最小寬 1400。**只 stage 這 5 張**，其餘 `git checkout` 丟掉。

2. **拿掉設計稿上系統量不到的讀數。** K1-D (kMeWS) 與 K4-D (M6ra92) 的下載列說明寫著 `12.4 MB/s`。`DownloadsSection` 型別（`apps/web/src/services/activityService.ts:35-44`）只有 `downloading` / `queued` / `errored` / `paused` / `total`——**沒有任何速度欄位**，`/api/v1/activity` 也不回傳。一個系統沒有量到的數字不准出現在稿上。改成程式碼真的會渲染的形狀：`3 個錯誤 · 12 個進行中 · 5 個排隊 · 2 個暫停`，並讓「N 個錯誤」用 `$error-text`、「N 個暫停」用 `$text-muted`，對上 `ActivityHub.tsx:225-243` 的分段著色。

3. **手機稿不准自己編縮短文案。** K1-M (QIwY1) 目前寫著 `已處理 1,234 / 5,000`、`尋找繁體中文字幕`、`8 個檔案無法比對`、`處理 →`、`開啟 →`、`3 進行中 · 5 排隊`。這些字串**程式碼永遠不會產生**——`ActivityRow.tsx:61` 用 CSS `truncate`，手機上是同一份完整字串被省略號截掉。全部改回桌機版的完整文案，需要示意截斷就畫省略號，不要另寫一份短句。
   📌 這是 2026-09-11 手機設定分頁列「自訂首頁→首頁」同一類錯誤的再犯，判例已在 `disc-2026-09-flow-a-m-screen-content`。

4. **手機第五格標籤改成「更多」。** K1-M 寫的是英文 `More`。實機是 `MobileTabBar.tsx:70` 的「更多」。

5. **手機補上「活動記錄」區塊。** `ActivityHub` 四個區塊（進行中／待處理／下載／活動記錄）不分斷點都渲染，K1-M 只畫了三個。

6. **兩張稿補上「批次生成字幕」CTA。** `ActivityHub.tsx:330-341` 的 header 右側有這顆按鈕（`data-testid="activity-generation-batch-cta"`，泥金實心、`Captions` 圖示、44px 高），而且它是 **Activity 這一側唯一的批次生成入口**（D4-1 邊界）。K1-D 與 K1-M 都缺。

7. **文案：一處改程式碼，三處改設計稿。**

   ⚠️ **本條在 dev-story 的 AC-drift 檢查階段被大幅修正。** SM 建單時把四處都判成「稿比較準、碼要跟上」，實際查了後端之後**三處是稿在說假話**，只有一處成立。詳見 Dev Agent Record → Completion Notes 的 AC Drift 段。

   **(a) 碼→稿，唯一成立的一條。** `ACTIVE_META.subtitle_batch.title`：`批次字幕` → `批次字幕搜尋`。證據：`apps/api/internal/subtitle/batch.go` 打的是 `providers.SubtitleQuery`、掃的是 `SubtitleStatusNotSearched` / `NotFound`——它是**搜尋**現成字幕，跟 `generation_batch`（生成）是兩件事，原標題分不出來。同步改 `ActivityHub.spec.tsx` 的斷言。

   **(b) 稿→碼：`AI 字幕校正` 是假的。** `transcription` 這個 kind 來自 `TranscriptionService`（`activity_service.go:169-176`），是**單筆生成字幕**，不是校正。而且 `disc-2026-07-transcription-active-jobs` 的 Task 4.2 白紙黑字挑了「字幕生成中」，理由是「要跟既有的字幕批次／生成批次分得開」。改程式碼＝推翻那條已出貨的 AC。**K1-D (kMeWS) 與 K4-D (M6ra92) 的標題改成 `字幕生成中`。**

   **(c) 稿→碼：那一列的說明也是造句。** 稿寫 `你的名字 · S1E3 — 簡轉繁 + 語意校正中`。後端的 `Detail` 欄位放的是**原始資料**（片名／檔名），`disc-2026-07-transcription-active-jobs` 的 Dev Notes 明文禁止「把中文句子塞進 Detail」。**改成 `你的名字 · S1E3`。**

   **(d) 稿→碼：`8 個檔案無法自動比對中繼資料` 是假的。** `parseCount` 來自 `ParseJobRepository.GetPending`，SQL 是 `WHERE status = 'pending'`——那是**佇列**。失敗是 `status = 'failed'`，走的是另一條路（活動記錄區的「解析失敗」）。把排隊講成失敗會讓人以為有八個檔案壞了。**K1-D / K4-D / K1-M 改成程式碼現在的 `8 個項目待處理`。**
   📌 程式碼那句確實是套套邏輯（標題已經寫「待解析項目」），但**「模糊但真」勝過「具體但假」**。改進文案是另一件事，已立案，不在本 story。

   **(e) 稿→碼：空狀態說明的「字幕校正」。** 同 (b)：這裡發生的是搜尋與生成，不是校正。**K3-D (DZnSv) 改成程式碼現在的 `掃描、字幕與下載工作會在這裡顯示。`**

8. **`transcription` 右槽維持「進行中」，稿改成「進行中」。** 稿寫「校正中」（與 (b) 同一列，同一個誤解）。`NO_PERCENT_KINDS` 的右槽文字是 DESIGN.md §Status Rows 的固定形狀——「沒有可量測進度的工作顯示純文字**進行中**」。逐 kind 換字會讓那個位置失去一致含義。方向是**稿→碼**（第 10 列因此翻向）。

9. **三個檔頭改成正確的畫面代號。**
   - `ActivityHub.tsx` → `// Design ref: ux-design.pen Screen K1-D-v2 (kMeWS) · K1-M-v2 (QIwY1)`
   - `ActivityStates.tsx` → `// Design ref: ux-design.pen Screen K2-D-v2 (suCiI) · K3-D-v2 (DZnSv) · K4-D-v2 (M6ra92)`，docstring 內文的 `A4-D-v2` / `A5-D-v2` / `A6-D-v2` 一併改成 K2 / K3 / K4
   - `ActivityRow.tsx` **不動**（`Component/ActivityRow-v2 (fF8nX)` 已驗證仍存在且正確）
   🚨 檔頭那一行必須以 `(節點ID)` 結尾，說明另起一行——`local/implements-pen-node-id` 的 `DESIGN_REF_RE` 要求如此，2026-09-11 有 8 個檔案踩過。

10. **6 處 `text-[13px]` 收斂到字階。** 13 不在字階上（Label 12 / Body 14）。逐處判斷：`ActivityRow.tsx:61` 的 detail 行與 `ActivityHub.tsx:115/119/121` 的右槽讀數屬於次要資訊 → `text-xs`（12）；`ActivityHub.tsx:183/246` 的兩顆 CTA 連結是可點擊的行動 → `text-sm`（14），不要一律降級。改完 `grep -c 'text-\[13px\]' apps/web/src/components/activity/*.tsx` 必須全為 0。

11. **既有測試全數保留通過。** `ActivityHub.spec.tsx` 現有斷言只有第 7 條指名的四處文案可以改，其餘一律不動。特別是 fail-soft 那幾條（一個區塊壞掉時其他區塊照樣渲染）。

12. **CI 全綠。** `pnpm run lint`（0 errors）、`pnpm nx run web:typecheck`、`pnpm run format:check`、`python3 scripts/check-design-tokens.py`、`pnpm nx test web`、`pnpm nx test api`。

## Tasks / Subtasks

- [x] **Task 1 — 先讓稿讀得到（AC: #1）**
  - [x] `READABLE_FLOWS` 加入 `"flow-k-activity-v2"`
  - [x] `python3 scripts/export-pen-screenshots.py`（需 Pencil.app 在跑）
  - [x] 只 stage `_bmad-output/screenshots/flow-k-activity-v2/*.png` 這 5 張，其餘 `git checkout -- _bmad-output/screenshots/`
  - [x] 打開 k1-d.png 跟 k1-m.png 確認字讀得到，再往下做

- [x] **Task 2 — 設計稿側的四項修正（AC: #2, #3, #4, #5, #6, #8）**
  - [x] K1-D (kMeWS) / K4-D (M6ra92)：下載列的 `12.4 MB/s` 換成 `3 個錯誤 · 12 個進行中 · 5 個排隊 · 2 個暫停`，錯誤數 `$error-text`、暫停數 `$text-muted`
  - [x] K1-M (QIwY1)：六處自編短句全部換回完整文案；第五格 `More` → `更多`；補「活動記錄」區塊（照 K1-D 的三列）
  - [x] K1-D / K1-M：header 右側補「批次生成字幕」按鈕（泥金實心、`Captions` 圖示、44px 高）
  - [x] K1-D / K4-D：`AI 字幕校正` 那列 → 標題 `字幕生成中`、說明 `你的名字 · S1E3`、右槽 `進行中`（三處都改，見 AC #7 (b)(c) 與 #8）
  - [x] K1-D / K4-D / K1-M：待處理列說明 `8 個檔案無法自動比對中繼資料` → `8 個項目待處理`（AC #7 (d)）
  - [x] K3-D (DZnSv)：空狀態說明的 `字幕校正` → `字幕`（AC #7 (e)）
  - [x] 每一步之後跑一次溢出檢查：`Get(id,(n,c)=>c.problems&&Print(...))`，Flow K 溢出必須 0
  - [x] ⚠️ 用 Pencil MCP，**不要**用 Read/Grep 碰 `.pen`

- [x] **Task 3 — 三個檔頭（AC: #9）**
  - [x] `ActivityHub.tsx` 第一行改成 K1-D-v2 / K1-M-v2
  - [x] `ActivityStates.tsx` 第一行改成 K2 / K3 / K4，docstring 內文的 A4/A5/A6 一併改
  - [x] `ActivityRow.tsx` 確認後不動
  - [x] `npx eslint apps/web/src/components/activity/` → 0 errors

- [x] **Task 4 — 程式碼只改一處文案（AC: #7a）**
  - [x] 先改 `ActivityHub.spec.tsx` 的斷言讓它變紅，再改 `ACTIVE_META.subtitle_batch.title`：`批次字幕` → `批次字幕搜尋`
  - [x] ⛔ **不要**改 `transcription` 的標題、不要改 `PendingSectionView` 的 detail、不要改 `ActivityEmpty` 的說明——那三處程式碼是對的，錯的是稿（AC #7 (b)(d)(e)）

- [x] **Task 5 — 字級收斂（AC: #10）**
  - [x] `ActivityRow.tsx:61` → `text-xs`
  - [x] `ActivityHub.tsx:115/119/121` → `text-xs`
  - [x] `ActivityHub.tsx:183/246`（兩顆 CTA）→ `text-sm`
  - [x] 新增一個測試守住：活動區沒有任何 `text-[\d+px]` 任意值（照 `LoginForm.spec.tsx` 的 `puts every label on the type scale` 先例）

- [x] **Task 6 — 收尾驗證（AC: #11, #12）**
  - [x] `pnpm run lint` → 0 errors
  - [x] `pnpm nx run web:typecheck --skip-nx-cache`
  - [x] `pnpm run format:check`
  - [x] `python3 scripts/check-design-tokens.py`
  - [x] `pnpm nx test web` 與 `pnpm nx test api`（⛔ 絕不用 `run_in_background` 跑測試）
  - [x] 重跑匯出並確認只有 Flow K 的 5 張有 byte 差異

## Dev Notes

### 這張跟 dsr-12 的差別

dsr-12 的六件事全部是「碼追稿」。這張有 **6 件是稿要改**、**5 件是碼要改**。原因是 Flow K 的稿畫在 `活動中心獨立成 Flow K` 那次重組之前，內容比程式碼舊了一輪，而且當時畫稿的人手上沒有 API 型別。

**不要把程式碼改成跟稿一樣**——特別是下載速度那一條。程式碼沒有那個數字是對的，稿有才是錯的。

### 固定詞彙與誠實讀數

- **`12.4 MB/s` 是這張 story 最重要的一條。** 「沒有可量測進度的工作完全不渲染進度條，而不是渲染一條空的」是 DESIGN.md §Status Rows 寫死的；一個 API 不回傳的速度數字出現在稿上，是同一條規則的反面。
- **下載列的「N 個錯誤」是 bugfix-e 特意加上去的警訊**：在那之前壞掉的 torrent 被掃進「個排隊」，3,068 個壞檔讀起來像健康的佇列。稿上缺這一段，等於把那個修正畫沒了。
- `ActivityRow` 的 `TONE` 只有 `neutral` / `success` / `error` 三色，沒有 warning。本 story 不新增色階。

### 不要做的事

- **不要動 `?view=generation`。** `routes/activity.tsx` 支援它，渲染的是 `GenerationWorkspace`，那張稿是 **F13-D-v2 (F7ohe)**，屬於 Flow F／dsr-6 的範圍，Flow K 不需要補。
- **不要動 `ActivityRow.tsx` 的檔頭。** `fF8nX` 已用 Pencil MCP 驗證仍存在，而且它住在 Design System 群組、是真正的 reusable component。
- **不要改 fail-soft 行為。** 一個區塊 `unavailable` 時只降級那一段、整頁照常渲染（F3），K4-D 畫的就是這個，程式碼也是這個。
- **不要為了對稿而拆 `ActivityHub` 的四個區塊順序。** 進行中 → 待處理 → 下載 → 活動記錄，稿與碼一致。

### Source tree

```
apps/web/src/components/activity/ActivityHub.tsx      ← Task 3,4,5（主要）
apps/web/src/components/activity/ActivityStates.tsx   ← Task 3,4
apps/web/src/components/activity/ActivityRow.tsx      ← Task 5（只改一行字級）
apps/web/src/components/activity/ActivityHub.spec.tsx ← Task 4,5
apps/web/src/services/activityService.ts              ← 只讀，確認沒有速度欄位
scripts/export-pen-screenshots.py                     ← Task 1
ux-design.pen（kMeWS / QIwY1 / M6ra92）                ← Task 2，只能用 Pencil MCP
```

### Project Structure Notes

- `sprint-status.yaml` 的 `dsr-10` 條目寫「ActivityHub.tsx 5 處舊字級」，實際是 **6 處**（ActivityHub 5 ＋ ActivityRow 1）。條目也寫「4 檔只有 2 檔有 Design ref 標頭」，實際是 **3 個非 spec 檔全都有標頭，但其中 2 個寫錯畫面代號**——那比缺標頭更糟，因為它看起來是對的。
- `routes/activity.tsx` 免 Rule 21（route 檔）。

### Time-dependent visual coverage

- **Does this story add/modify any `apps/web/src/components/**/*.{ts,tsx}` that reads the wall clock?**
  - **間接是。** `ActivityHub.tsx:280` 呼叫 `formatRelativeTime(ev.at)` 渲染「2 分鐘前」。那個函式讀 `Date.now()`。
  - 但**本 story 不新增視覺夾具**，也不動 `活動記錄` 區塊的時間渲染，所以不會新增任何會腐爛的基準線。
  - ⚠️ 若 dev 決定順手加一個 `activity-hub` 的 gallery 夾具（**AC 沒有要求，建議不要**），`recent.events[].at` 必須用 `JUST_NOW()` / `MINUTES_AGO()` / `HOURS_AGO()` 這三個既有 helper，**絕不能**寫死絕對時間——`gallery-fixture-time-stability.spec.ts` 會擋，而且 `-linux` 基準線重生要走 CI 一輪。
- Reference: `project-context.md` Rule 23；`tests/visual/.../gallery-fixture-time-stability.spec.ts`。

### References

- [Source: `ux-design.pen` Screen K1-D-v2 (kMeWS) / K1-M-v2 (QIwY1) / K2-D-v2 (suCiI) / K3-D-v2 (DZnSv) / K4-D-v2 (M6ra92)] — 內容用 Pencil MCP `Get(..., {resolveInstances:true})` 展開 instance 後讀出，非依賴縮圖
- [Source: `apps/web/src/services/activityService.ts:35-44`] — `DownloadsSection` 型別，AC #2 的事實基礎
- [Source: `apps/web/src/components/activity/ActivityRow.tsx:61`] — `truncate`，AC #3 的事實基礎
- [Source: `apps/web/src/components/shell/MobileTabBar.tsx:70`] — 「更多」，AC #4 的事實基礎
- [Source: `apps/web/src/components/activity/ActivityHub.tsx:330-341`] — 批次生成字幕 CTA，AC #6 的事實基礎
- [Source: DESIGN.md#Status Rows（signature）] — 右槽三態與「不渲染空進度條」
- [Source: project-context.md#Rule 21 / #Rule 23 / #Rule 24]
- [Source: `sprint-status.yaml` → `epic-dsr` / `dsr-10-flow-k-activity-v2` / `disc-2026-09-flow-a-m-screen-content`（假截斷判例）]

## Dev Agent Record

### Agent Model Used

Claude Opus 5 (1M context) — `claude-opus-5[1m]`，BMAD dev agent（Amelia）2026-09-11

### Debug Log References

- `pnpm nx test web`（RED）：1 failed / 3343 passed —— 翻轉後的 `批次字幕搜尋` 斷言如預期變紅
- `pnpm nx test web`（GREEN）：**3345 / 3345 passed**（+1 新守門測試）
- `pnpm nx test api`：PASS
- `pnpm run lint`：**0 errors**、127 warnings（與改動前同數）
- `pnpm nx run web:typecheck --skip-nx-cache`：PASS
- `python3 scripts/check-design-tokens.py`：一致（188 畫面／72 母版／clipping 72）
- 匯出：188 張，變動 13 張（Flow K 5、Flow F 5、Design System 3——後 8 張是 ActivityRow 母版的進度條顏色）

### Completion Notes List

- 🔗 **AC Drift: FOUND — 這次的重點就是它。** 檢查 `grep -rln "ActivityHub|活動中心|12.4 MB/s|parseCount" _bmad-output/implementation-artifacts/*.md`（10 個檔案命中）之後，**SM 建單時 AC #7 的四條有三條判錯方向**，已在動手前改寫 story：
  - `disc-2026-07-transcription-active-jobs` Task 4.2 白紙黑字挑了「字幕生成中」，理由是要跟字幕批次／生成批次分得開。把它改成稿上的「AI 字幕校正」＝推翻一條已出貨的 AC；而且查 `activity_service.go:169-176`，`transcription` 來自 `TranscriptionService`，是**單筆生成**不是校正。→ 改稿。
  - 同一張單子的 Dev Notes 明文禁止「把中文句子塞進 `Detail`」，`Detail` 放的是原始資料。稿上的「你的名字 · S1E3 — 簡轉繁 + 語意校正中」是造句。→ 改稿。
  - 稿上的「8 個檔案無法自動比對中繼資料」是假的：`ParseJobRepository.GetPending` 的 SQL 是 `WHERE status = 'pending'`，那是**佇列**；失敗是 `status = 'failed'`，走活動記錄區的「解析失敗」。把排隊講成失敗會讓人以為有八個檔案壞了。→ 改稿。
  - 唯一成立的是 `批次字幕` → `批次字幕搜尋`：`subtitle/batch.go` 打 `providers.SubtitleQuery`、掃 `NotSearched`/`NotFound`，它在搜尋不是生成。→ 改碼（本 story 唯一的程式碼文案變更）。
- 📎 **Contract Stamps: NONE**（本 story 與所引用的 `ux3-2-3-activity-frontend.md` 都沒有 `[@contract-v*]` 標記——那是 Rule 20 之前的單子，屬 implicit v0）。
- 🎭 **A11y Pre-Flight: PASS**（3 個 component 檢查：`ActivityHub.tsx`／`ActivityStates.tsx`／`ActivityRow.tsx`，jsx-a11y warning 0，本 story 引入 0）。四類回歸項：沒有新增圖片、沒有新增 aria-modal、沒有新增非同步揭露內容、沒有新增自訂 widget——只改 class 字串、一個標題字串與檔頭。既有的 `role="progressbar"` + `aria-valuenow` 與 `role="alert"` 原封不動。
- ⚠️ **Pre-existing failure — 沿用既有立案**：`CI=1 npx playwright test --project=visual` 在 darwin 上仍是那 3 個既有紅（dsr-12 已用 `git stash` 對照驗證過與改動無關），立案 `preexisting-fail-visual-darwin-three-stale-baselines`。本 story 沒有新增視覺夾具，不影響。
- ✅ Task 1：`flow-k-activity-v2` 進 `READABLE_FLOWS`。**執行順序與 story 寫的略有不同**：因為設計稿內容我一開始就用 Pencil MCP 直接讀（不依賴縮圖），所以把「改腳本」與「改設計稿」併在一起、最後只匯出一次，省掉一輪 10 分鐘的重複匯出。結果相同。
- ✅ Task 2：設計稿六項全修，另外撿到三項一併處理（見下方 Discovery Triage 的 ①）。
- ✅ Task 3：兩個檔頭改成 K 代號，`ActivityRow.tsx` 驗證後未動。
- ✅ Task 4：程式碼只改一處文案。
- ✅ Task 5：6 處 `text-[13px]` 收斂完畢，並加一條守門測試防止長回來。
- 📝 **順手發現並修正的一條母版級問題**：`Component/ActivityRow-v2 (fF8nX)` 的 `progressFill` 是 `$success`（青碧＝有答案了），但程式碼的進度條是 `bg-[var(--accent-primary)]`（泥金＝正在跑）。固定詞彙裡進行中的工作就是泥金，青碧是完成。母版改一次，**126 個 instance 裡有 52 個會顯示進度條**的全部跟著對，橫跨 Design System / Flow F / Flow K。只改 Flow K 那 21 個做不到——那得逐一 override，只會更糟。

### Discovery Triage

- **Did this story discover any work outside its current scope?**
  - **YES**，一項，lane ③：
    - **③ 空狀態的 CTA 是產品問題，不是對齊問題。** 稿寫「掃描媒體庫」，程式碼是「前往媒體庫」連到 `/library`。本 story **兩邊都不改**。已立案 **`disc-2026-09-activity-empty-cta-scan-or-navigate: backlog`**（雙向：該條目寫明由 dsr-10 立案）。
  - 實作中另外撿到三項，全部走 lane **① expand-scope-in-place**（就地吸收，由既有 AC 追蹤，不另開單）：
    - **① `Component/ActivityRow-v2` 的進度條是青碧。** 固定詞彙裡進行中＝泥金、完成＝青碧，程式碼用的是泥金。只修 Flow K 的 21 個 instance 做不到（得逐一 override），改母版才是唯一正解，代價是 Flow F 與 Design System 的 8 張稿一起更新。由 AC #2「稿要對上程式碼真的會渲染的樣子」追蹤。
    - **① K4-D 有三列幽靈 row 停在 y=-6443。** 畫面上看不到（frame `clip:true`），但任何文字讀取都會以為 進行中 那一區有內容——我自己在比對時就被騙過一次。刪除；健康態的那三列 K1-D 已有。由 AC #8 所屬的 K4 區塊修正追蹤。
    - **① K1-M 頂欄寫「活動」。** `AppShellV2` 的手機頂欄只有 vido／主題／搜尋，頁面標題在捲動區內。由 AC #6（補頁首 CTA）連帶追蹤，因為 CTA 本來就住在那個頁首裡。
- Reference: `project-context.md` Rule 24

### File List

**修改（程式碼）：**
- `apps/web/src/components/activity/ActivityHub.tsx` — Rule 21 檔頭、`subtitle_batch` 標題、5 處字級
- `apps/web/src/components/activity/ActivityStates.tsx` — Rule 21 檔頭＋docstring 的舊代號
- `apps/web/src/components/activity/ActivityRow.tsx` — 1 處字級
- `apps/web/src/components/activity/ActivityHub.spec.tsx` — 1 處斷言翻轉＋1 個新守門測試

**修改（設計與腳本）：**
- `ux-design.pen` — K1-D / K1-M / K3-D / K4-D 內容修正；`Component/ActivityRow-v2` 進度條改泥金
- `scripts/export-pen-screenshots.py` — `READABLE_FLOWS` 加入 `flow-k-activity-v2`
- `_bmad-output/pen-tokens.json` — 快照同步
- `_bmad-output/screenshots/flow-k-activity-v2/*.png`（5）、`flow-f-subtitle-v2/*.png`（5）、`design-system/*.png`（3）
- `_bmad-output/implementation-artifacts/sprint-status.yaml` — 狀態流轉＋1 筆新立案

## Change Log

| 日期 | 內容 |
| --- | --- |
| 2026-09-11 | **AC 修正**（動手前）：AC-drift 檢查推翻 SM 對四處文案的方向判斷，三處改判成「稿在說假話」，story 的對照表與 Task 2/4 同步改寫。 |
| 2026-09-11 | Task 1 — `flow-k-activity-v2` 進 `READABLE_FLOWS`，5 張稿從 400px 改為 2x／最小寬 1400。 |
| 2026-09-11 | Task 2 — 設計稿：拿掉 `12.4 MB/s`（系統量不到）；下載列改成三段著色的真實形狀；手機六句自編短句換回完整文案；`More` → `更多`；補「活動記錄」區塊；兩張稿補「批次生成字幕」CTA；`AI 字幕校正` 整列改回 `字幕生成中`；待處理說明改回 `8 個項目待處理`；空狀態說明去掉「校正」。 |
| 2026-09-11 | Task 2 附帶 — K4-D 刪掉三列停在 y=-6443 的幽靈列（區塊失敗時它們不該存在，卻讓任何文字讀取都以為那一區有內容）；同區的計數晶片「3」移除（區塊 unavailable 時程式碼不傳 count）。 |
| 2026-09-11 | Task 2 附帶 — K1-M 頂欄標題從「活動」改成「vido」並補主題切換圖示：`AppShellV2` 的手機頂欄只有 vido／主題／搜尋，頁面標題在捲動區內。 |
| 2026-09-11 | 母版 — `Component/ActivityRow-v2` 進度條 `$success` → `$accent-primary`，52 個會顯示進度條的 instance 一次對齊固定詞彙。 |
| 2026-09-11 | Task 3 — 兩個檔頭從 Flow A 的舊代號改成 K1/K1-M 與 K2/K3/K4。 |
| 2026-09-11 | Task 4 — `批次字幕` → `批次字幕搜尋`（先紅後綠）。 |
| 2026-09-11 | Task 5 — 6 處 `text-[13px]` 收斂（次要讀數 `text-xs`、CTA `text-sm`），加守門測試。 |
| 2026-09-11 | Task 6 — 閘門：web 3345/3345、api PASS、lint 0 errors、typecheck PASS、token 一致、prettier 通過。 |
| 2026-09-11 | 立案 1 筆：`disc-2026-09-activity-empty-cta-scan-or-navigate`（空狀態 CTA 是產品裁定，兩邊都沒動）。 |

## 給 Alexyu 的一個問題（不阻塞開發）

活動中心**空狀態**那顆按鈕，你要哪一個？

1. **維持「前往媒體庫」**（現況）——只是導覽，稿改成跟程式碼一樣。零成本。
2. **改成「掃描媒體庫」並真的觸發掃描**——程式碼要接 `useTriggerScan`，變成一個小功能。稿不用改。

我傾向 **1**，理由是「空狀態」的意思是沒有背景工作在跑，此時最可能的下一步是去看媒體庫有什麼，而不是再掃一次。但如果你覺得空狀態就該給人一個「讓它動起來」的按鈕，那是 2，我另開一張 story。
