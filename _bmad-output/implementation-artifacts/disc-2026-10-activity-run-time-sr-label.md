# Disc：活動記錄的「花多久 · 花多少錢」前面，讀螢幕軟體多唸一句「花了」——畫面不變

Status: ready-for-dev

**Source:** `disc-2026-10-activity-subtitle-run-time-b`（PR #828）合併後，dev 回報：讀螢幕軟體把桌機右側兩行「3 分鐘 · US$0.27」和「2 分鐘前」連在一起唸成「3 分鐘 US$0.27 2 分鐘前」，聽的人分不出哪個是「花了多久」、哪個是「多久以前」。
⚖️ **Alexyu 2026-10-19 裁定 1A：** 在「花多久 · 花多少錢」前面加一段只給讀螢幕軟體唸的「花了」（sr-only），**畫面不變**。桌機和手機都要（手機那段在說明下一行——同一個節點，改一次兩邊都有）。

設計稿：**無變更，沿用 K1-D-v2 `kMeWS`（`LndN2`、`EVrLx`）／K1-M-v2 `QIwY1`（`JLTBV`、`akt4i`）**。sr-only 文字不畫在設計稿上（同 `DownloadCardV2.tsx:137` 的「預估剩餘」、`ImportStatusChip.tsx:163` 的前例）。

**依賴：** 無。`disc-2026-10-activity-subtitle-run-time-b` 已合併（`origin/main` 9331d1e2）。跟進行中的 `disc-2026-10-activity-subtitle-run-time-a`（後端）不碰同一個檔案，誰先合都可以。

## 查到的事（2026-10-19，讀 `origin/main`）

- **meta 節點只有一個**：`SubtitleRow`（`apps/web/src/components/activity/RecentSection.tsx:45-70`）在 `:57-66` 輸出 `<span data-testid="activity-recent-meta" className="font-mono text-xs text-[var(--text-secondary)] sm:text-sm">{meta}</span>`；`ActivityRow` 用 grid 把**同一個節點**桌機放右上、手機放說明下面（`apps/web/src/components/activity/ActivityRow.tsx:94-111`），DOM 順序是「標題 → 說明 → meta → 時間」（`:87-93` 註解）。所以讀螢幕軟體在兩種寬度都是「…說明、3 分鐘 · US$0.27、2 分鐘前」。在 meta 節點**裡面**、文字**前面**放一個 sr-only，就兩邊一起解決。
- **meta 文字怎麼來**：`recentSubtitleMeta(ev)`（`apps/web/src/components/activity/recentEventCopy.ts:90-96`）＝ `formatRunDuration`（`:34-39`）與 `formatRunSpend`（`:42-55`）兩段用 ` · ` 接，缺的段省略，兩段都缺回 `null`（那時 `RecentSection.tsx:58` 不輸出 meta 節點）。可能出現的組合：
  | durationSeconds | spentUsd | meta（畫面） | 讀螢幕軟體 |
  |---|---|---|---|
  | 180 | 0.27 | 3 分鐘 · US$0.27 | 花了3 分鐘 · US$0.27 |
  | 4 | 0 | 4 秒 · 沒花錢 | 花了4 秒 · 沒花錢 |
  | 180 | 沒有（舊紀錄 NULL） | 3 分鐘 | 花了3 分鐘 |
  | 沒有（時鐘倒退，A 單 Dev Notes） | 0.27 | US$0.27 | 花了US$0.27 |
  | 沒有 | 0.004 | 不到 US$0.01 | 花了不到 US$0.01 |
  | 沒有 | 0 | 沒花錢 | **沒花錢**（不加「花了」——「花了沒花錢」不通） |
  | 沒有 | 沒有 | （沒有 meta 節點） | （什麼都不唸） |
- **專案的 sr-only 寫法**：Tailwind `sr-only` class，前例 `DownloadCardV2.tsx:99`、`:137`、`ImportStatusChip.tsx:163`、`LocalizationLevelForm.tsx:172`（`<span className="sr-only">例：</span>`，其 spec `LocalizationLevelForm.spec.tsx:145` 用 `querySelector('.sr-only')` 斷言）。
- **既有測試不會因為多一個子節點而壞**：`RecentSection.spec.tsx:64`、`:66` 用 `toHaveTextContent('3 分鐘 · US$0.27')`（子字串比對，「花了3 分鐘 · US$0.27」仍含它）；`:69` 的 `getAllByText('3 分鐘 · US$0.27')` 只看元素**自己的文字節點**（Testing Library `getNodeText`），sr-only 是子元素，meta 自己的文字節點仍是「3 分鐘 · US$0.27」。→ 這三條照綠，**所以它們抓不到本單的改動**，要另寫會變紅的斷言（AC #4）。
- **E2E**：`git grep -n "activity-recent" origin/main -- tests` 沒有命中；E2E 不碰活動記錄。`tests/e2e/activity-spend.spec.ts:43` 的 stub 是空的 `events: []`。→ **E2E 與 data-testid 不受影響**（不新增、不改名、不移除 testid）。
- **視覺基準**：活動記錄有三組基準，darwin＋linux 都已在 main：`tests/visual/components.visual.spec.ts-snapshots/components/activity-recent/{subtitle-runs,subtitle-runs-mobile,subtitle-failed-mobile}/default-visual-{darwin,linux}.png`。`sr-only`＝`position:absolute; width:1px; height:1px; overflow:hidden; clip:rect(0,0,0,0)`，不佔版面也不出像素 → **這六張圖都不應該變**。

## Story

身為只有我一個人在用的 NAS 片庫主人（也可能哪天用讀螢幕軟體聽這一頁），
我想要讀螢幕軟體唸活動記錄時，在「3 分鐘 · US$0.27」前面先唸「花了」，
這樣我聽得出這是「這次花了多久、花了多少錢」，不會跟後面的「2 分鐘前」混在一起。

## Acceptance Criteria

1. **讀到的文字：** 字幕列的 meta 節點（`data-testid="activity-recent-meta"`）的 `textContent` 依「查到的事」那張表：
   - 有「花多久」那段，或「花多少錢」那段不是「沒花錢」→ 開頭是「花了」，後面接畫面上的字，例如 `花了3 分鐘 · US$0.27`、`花了4 秒 · 沒花錢`、`花了3 分鐘`、`花了US$0.27`、`花了不到 US$0.01`。
   - 只有「沒花錢」一段 → `textContent` 就是 `沒花錢`，沒有「花了」。
   - 兩段都沒有 → 跟今天一樣，沒有 meta 節點。
2. **只給讀螢幕軟體：** 「花了」包在一個 `<span className="sr-only">` 裡，放在 meta 節點裡面、可見文字前面。畫面上看不到它，可見文字一個字都不變（meta 自己的文字節點仍是 `3 分鐘 · US$0.27`）。
3. **桌機、手機同一處：** 不另外在手機或桌機各放一份；meta 節點仍然整個 DOM 只有一個（`RecentSection.spec.tsx:63`、`:65`、`:69` 照綠）。解析列與其他非字幕列沒有「花了」。
4. **會變紅的測試（`RecentSection.spec.tsx` 新增）：**
   - 完成列：`expect(meta.textContent).toBe('花了3 分鐘 · US$0.27')`；失敗列：`toBe('花了4 秒 · 沒花錢')`。——拿掉 sr-only span → 變成 `3 分鐘 · US$0.27`，紅。
   - `within(meta).getByText('花了')` 有 `sr-only` class（`toHaveClass('sr-only')`）。——把 class 拿掉（字會露出來）→ 紅。
   - 「花了」在 meta 節點**裡面**、在時間節點**前面**：`row.textContent` 裡 `indexOf('花了') < indexOf('3 分鐘')`，而且 `within(time).queryByText('花了')` 是 null。——放到時間節點或列的最後 → 紅。
   - 只有「沒花錢」的字幕列（`durationSeconds: undefined, spentUsd: 0`）：`meta.textContent` 是 `沒花錢`，`within(meta).queryByText('花了')` 是 null。——無條件加「花了」→ 紅。
   - 解析列：`within(parseRow).queryByText('花了')` 是 null。
5. **純函式（`recentEventCopy.ts`）：** 加一個回傳要不要唸「花了」的函式（建議 `recentSubtitleMetaLead(ev): string | null`，回 `'花了'` 或 `null`），規則照 AC #1，**用 `formatRunDuration`／`formatRunSpend` 的結果判斷**，不要自己重比錢（`currency.ts:10-21` 規則：錢不准用 `===`／`<`）。`recentEventCopy.spec.ts` 為表格的每一列各寫一條，含 `spentUsd: 0.004`（「不到 US$0.01」也要「花了」）。
6. **視覺基準不變：** 本機跑 `AI_PROVIDER=claude pnpm test:visual --grep activity-recent`（不加 `--update-snapshots`；`AI_PROVIDER` 不帶的話 Go API 起不來）三組全綠；**不重產、不 commit 任何 png**。如果紅了，代表 sr-only 影響了版面——那是 bug，修程式，不是更新基準。
7. `pnpm nx run web:test`（activity 相關 spec）、`pnpm run lint:all`（看「✖ N problems (X errors」那行）、typecheck、`pnpm run format:check` 綠。

## Tasks / Subtasks

- [ ] **T1 純函式（AC #5）**：`recentEventCopy.ts` 在 `recentSubtitleMeta`（`:90-96`）旁加 `recentSubtitleMetaLead`；常數 `'花了'` 放檔內（同 `MISSING_TITLE` `:22` 的寫法，附一行註解寫 ⚖️ Alexyu 2026-10-19 1A）；`recentEventCopy.spec.ts` 加一個 `describe('recentSubtitleMetaLead (sr-only 花了)')`，表格七列。
- [ ] **T2 元件（AC #1–#3）**：`RecentSection.tsx:57-66` 的 meta span 內、`{meta}` 前面加 `{lead && <span className="sr-only">{lead}</span>}`；檔頭註解（`:11-14`）補一句「meta 前有 sr-only 花了，讀螢幕軟體才分得出跟相對時間」。
- [ ] **T3 測試（AC #4）**：`RecentSection.spec.tsx` 在「each subtitle row has exactly one meta node」（`:60-70`）後面加一個 `it`，涵蓋 AC #4 五點。
- [ ] **T4 收尾（AC #6、#7）**：visual 不更新跑一次；lint／typecheck／format。

## Dev Notes

### 不要做的事

- 不要把「花了」做成看得見的字，或改 meta 的 class／字級／顏色（裁定是「畫面不變」）。
- 不要用 `aria-label` 蓋掉 meta 節點：`aria-label` 放在沒有 role 的 `<span>` 上，多數讀螢幕軟體不唸，而且會讓 `getByText` 類的斷言失真。用 sr-only 子節點（專案既有寫法）。
- 不要在時間節點（`activity-recent-time`）加「之前」「時間」之類的 sr-only：裁定只有「花了」。要的話另開單。
- 不要動 `ActivityRow.tsx`：這是 `RecentSection` 字幕列自己的文字，不是列元件的通用行為。
- 不要跑 `test:visual:update`，不要產或 commit `-linux.png`（CLAUDE.md）。

### 實作提示

- 寫法示意（不是唯一寫法）：
  ```tsx
  <span data-testid="activity-recent-meta" className="font-mono text-xs text-[var(--text-secondary)] sm:text-sm">
    {lead && <span className="sr-only">{lead}</span>}
    {meta}
  </span>
  ```
  `lead` 在 `SubtitleRow`（`:45-48`）跟 `meta` 一起算。
- 「花了」與「3 分鐘」之間不加空白：中文唸起來是「花了三分鐘」；`textContent` 斷言就照 `花了3 分鐘 · US$0.27` 寫。
- 規則的白話：**「花了」後面一定要接一個量（時間或金額）**。只有「沒花錢」時沒有量可以接，所以不加。

### 測試標準

- Vitest＋Testing Library；斷言用 testid 與 `textContent`（Rule 16）。visual 只驗證「沒變」。
- 不要背景跑測試（會留下孤兒 vitest）。

### E2E data-testid

- 沒有新增、改名或移除 testid。E2E 不受影響（grep 結果見「查到的事」）。

### 視覺基準

- **不影響。** 預期 6 張（3 組 × darwin／linux）都不變；本機只驗 darwin。

### Time-dependent visual coverage

- 不新增夾具，不改時間來源；`RecentSection` 已是 Rule 23 (ii) `Clock-injected`（`RecentSection.tsx:2`），不需要動。

### Project Structure Notes

- 動到：`apps/web/src/components/activity/recentEventCopy.ts`（＋`.spec.ts`）、`apps/web/src/components/activity/RecentSection.tsx`（＋`.spec.tsx`）、`_bmad-output/implementation-artifacts/sprint-status.yaml`。
- 跟 `disc-2026-10-live-activity-failure-reasons-b` 沒有共用檔案（那張動 `generationEventCopy.ts`；本單只 import 它間接用到的 `failureCopyForReason`，不改）。

### References

- [Source: apps/web/src/components/activity/RecentSection.tsx:1-15, 45-70]
- [Source: apps/web/src/components/activity/recentEventCopy.ts:22, 34-55, 90-96]
- [Source: apps/web/src/components/activity/ActivityRow.tsx:87-111]
- [Source: apps/web/src/components/activity/RecentSection.spec.tsx:60-70]
- [Source: apps/web/src/components/downloads/DownloadCardV2.tsx:137；ImportStatusChip.tsx:163；settings/LocalizationLevelForm.tsx:172 + .spec.tsx:145（sr-only 前例）]
- [Source: _bmad-output/implementation-artifacts/disc-2026-10-activity-subtitle-run-time-b.md Completion Notes「範圍外」第 2 點]
- [Source: project-context.md Rule 16、Rule 23]

## Dev Agent Record

### Agent Model Used

### Debug Log References

### Completion Notes List

### File List
