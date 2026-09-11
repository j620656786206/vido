# Story DSR.10: Flow K 活動中心——程式碼與設計稿雙向對齊 K1–K4

Status: ready-for-dev

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
| 4 | 手機第五格標籤 | `More`（英文） | `更多` | **稿→碼** |
| 5 | 手機缺「活動記錄」區塊 | 只有三區 | 四區都渲染，不分斷點 | **稿→碼** |
| 6 | 「批次生成字幕」CTA | 兩張稿都沒畫 | header 有，且是 Activity 唯一的批次入口 | **稿→碼** |
| 7 | 待處理列說明 | `8 個檔案無法自動比對中繼資料` | `{n} 個項目待處理` | **碼→稿** |
| 8 | 進行中第二列標題 | `批次字幕搜尋` | `批次字幕` | **碼→稿** |
| 9 | 進行中第三列標題 | `AI 字幕校正` | `字幕生成中` | **碼→稿** |
| 10 | 進行中第三列右槽 | `校正中` | `進行中` | **碼→稿** |
| 11 | 空狀態說明 | `掃描、字幕**校正**與下載工作…` | `掃描、字幕與下載工作…` | **碼→稿** |
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

7. **四處文案改程式碼去對稿。** 稿的說法比程式碼準確，程式碼要跟上：
   - `PendingSectionView` 的 detail 從 `${section.parseCount} 個項目待處理` 改成 `${section.parseCount} 個檔案無法自動比對中繼資料`。原句是套套邏輯——標題已經寫了「待解析項目」，說明再說一次「待處理」等於沒說；`ActivityRow` 的 detail 欄位在元件註解裡就叫 **explain-why 行**。
   - `ACTIVE_META.subtitle_batch.title`：`批次字幕` → `批次字幕搜尋`（與 `generation_batch` 的「批次生成」區分開；一個是找現成的、一個是生成）。
   - `ACTIVE_META.transcription.title`：`字幕生成中` → `AI 字幕校正`。
   - `ActivityEmpty` 的說明：`掃描、字幕與下載工作會在這裡顯示。` → `掃描、字幕校正與下載工作會在這裡顯示。`
   ⚠️ **這四條都要同步改 `ActivityHub.spec.tsx` 裡對應的斷言**，不要只改元件。

8. **`transcription` 右槽維持「進行中」，稿改成「進行中」。** 稿寫「校正中」。`NO_PERCENT_KINDS` 的右槽文字是 DESIGN.md §Status Rows 的固定形狀——「沒有可量測進度的工作顯示純文字**進行中**」。逐 kind 換字會讓那個位置失去一致含義。方向是**稿→碼**（第 10 列因此翻向）。

9. **三個檔頭改成正確的畫面代號。**
   - `ActivityHub.tsx` → `// Design ref: ux-design.pen Screen K1-D-v2 (kMeWS) · K1-M-v2 (QIwY1)`
   - `ActivityStates.tsx` → `// Design ref: ux-design.pen Screen K2-D-v2 (suCiI) · K3-D-v2 (DZnSv) · K4-D-v2 (M6ra92)`，docstring 內文的 `A4-D-v2` / `A5-D-v2` / `A6-D-v2` 一併改成 K2 / K3 / K4
   - `ActivityRow.tsx` **不動**（`Component/ActivityRow-v2 (fF8nX)` 已驗證仍存在且正確）
   🚨 檔頭那一行必須以 `(節點ID)` 結尾，說明另起一行——`local/implements-pen-node-id` 的 `DESIGN_REF_RE` 要求如此，2026-09-11 有 8 個檔案踩過。

10. **6 處 `text-[13px]` 收斂到字階。** 13 不在字階上（Label 12 / Body 14）。逐處判斷：`ActivityRow.tsx:61` 的 detail 行與 `ActivityHub.tsx:115/119/121` 的右槽讀數屬於次要資訊 → `text-xs`（12）；`ActivityHub.tsx:183/246` 的兩顆 CTA 連結是可點擊的行動 → `text-sm`（14），不要一律降級。改完 `grep -c 'text-\[13px\]' apps/web/src/components/activity/*.tsx` 必須全為 0。

11. **既有測試全數保留通過。** `ActivityHub.spec.tsx` 現有斷言只有第 7 條指名的四處文案可以改，其餘一律不動。特別是 fail-soft 那幾條（一個區塊壞掉時其他區塊照樣渲染）。

12. **CI 全綠。** `pnpm run lint`（0 errors）、`pnpm nx run web:typecheck`、`pnpm run format:check`、`python3 scripts/check-design-tokens.py`、`pnpm nx test web`、`pnpm nx test api`。

## Tasks / Subtasks

- [ ] **Task 1 — 先讓稿讀得到（AC: #1）**
  - [ ] `READABLE_FLOWS` 加入 `"flow-k-activity-v2"`
  - [ ] `python3 scripts/export-pen-screenshots.py`（需 Pencil.app 在跑）
  - [ ] 只 stage `_bmad-output/screenshots/flow-k-activity-v2/*.png` 這 5 張，其餘 `git checkout -- _bmad-output/screenshots/`
  - [ ] 打開 k1-d.png 跟 k1-m.png 確認字讀得到，再往下做

- [ ] **Task 2 — 設計稿側的四項修正（AC: #2, #3, #4, #5, #6, #8）**
  - [ ] K1-D (kMeWS) / K4-D (M6ra92)：下載列的 `12.4 MB/s` 換成 `3 個錯誤 · 12 個進行中 · 5 個排隊 · 2 個暫停`，錯誤數 `$error-text`、暫停數 `$text-muted`
  - [ ] K1-M (QIwY1)：六處自編短句全部換回完整文案；第五格 `More` → `更多`；補「活動記錄」區塊（照 K1-D 的三列）
  - [ ] K1-D / K1-M：header 右側補「批次生成字幕」按鈕（泥金實心、`Captions` 圖示、44px 高）
  - [ ] K1-D / K4-D：`AI 字幕校正` 那列右槽 `校正中` → `進行中`
  - [ ] 每一步之後跑一次溢出檢查：`Get(id,(n,c)=>c.problems&&Print(...))`，Flow K 溢出必須 0
  - [ ] ⚠️ 用 Pencil MCP，**不要**用 Read/Grep 碰 `.pen`

- [ ] **Task 3 — 三個檔頭（AC: #9）**
  - [ ] `ActivityHub.tsx` 第一行改成 K1-D-v2 / K1-M-v2
  - [ ] `ActivityStates.tsx` 第一行改成 K2 / K3 / K4，docstring 內文的 A4/A5/A6 一併改
  - [ ] `ActivityRow.tsx` 確認後不動
  - [ ] `npx eslint apps/web/src/components/activity/` → 0 errors

- [ ] **Task 4 — 四處文案改程式碼（AC: #7）**
  - [ ] 先改 `ActivityHub.spec.tsx` 的四處斷言讓它變紅，再改元件
  - [ ] `PendingSectionView` detail、`ACTIVE_META.subtitle_batch.title`、`ACTIVE_META.transcription.title`、`ActivityEmpty` 說明

- [ ] **Task 5 — 字級收斂（AC: #10）**
  - [ ] `ActivityRow.tsx:61` → `text-xs`
  - [ ] `ActivityHub.tsx:115/119/121` → `text-xs`
  - [ ] `ActivityHub.tsx:183/246`（兩顆 CTA）→ `text-sm`
  - [ ] 新增一個測試守住：活動區沒有任何 `text-[\d+px]` 任意值（照 `LoginForm.spec.tsx` 的 `puts every label on the type scale` 先例）

- [ ] **Task 6 — 收尾驗證（AC: #11, #12）**
  - [ ] `pnpm run lint` → 0 errors
  - [ ] `pnpm nx run web:typecheck --skip-nx-cache`
  - [ ] `pnpm run format:check`
  - [ ] `python3 scripts/check-design-tokens.py`
  - [ ] `pnpm nx test web` 與 `pnpm nx test api`（⛔ 絕不用 `run_in_background` 跑測試）
  - [ ] 重跑匯出並確認只有 Flow K 的 5 張有 byte 差異

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

（dev agent 填寫）

### Debug Log References

### Completion Notes List

### Discovery Triage

- **Did this story discover any work outside its current scope?**
  - **YES**，一項，lane ③：
    - **③ 空狀態的 CTA 是產品問題，不是對齊問題。** 稿寫「掃描媒體庫」（暗示按了會開始掃描），程式碼是「前往媒體庫」連到 `/library`（只是導覽）。本 story **兩邊都不改**——把它變成觸發掃描是新功能不是對齊，而把稿改成「前往媒體庫」等於幫 Alexyu 做了產品裁定。留給他決定。**待立案：若 dev 執行時此條目尚未存在，開 `disc-2026-09-activity-empty-cta-scan-or-navigate: backlog` 並在此列回填 ID。**
- Reference: `project-context.md` Rule 24

### File List

（dev agent 填寫）

## 給 Alexyu 的一個問題（不阻塞開發）

活動中心**空狀態**那顆按鈕，你要哪一個？

1. **維持「前往媒體庫」**（現況）——只是導覽，稿改成跟程式碼一樣。零成本。
2. **改成「掃描媒體庫」並真的觸發掃描**——程式碼要接 `useTriggerScan`，變成一個小功能。稿不用改。

我傾向 **1**，理由是「空狀態」的意思是沒有背景工作在跑，此時最可能的下一步是去看媒體庫有什麼，而不是再掃一次。但如果你覺得空狀態就該給人一個「讓它動起來」的按鈕，那是 2，我另開一張 story。
