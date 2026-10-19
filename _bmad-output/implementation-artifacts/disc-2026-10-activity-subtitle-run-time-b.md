# Disc：活動記錄寫出字幕花多久 B（前端）——字幕列後面寫「花多久 · 花多少錢」，手機移到說明下一行

Status: ready-for-dev

**Source:** 同 A 單。2026-10-19 Alexyu：「目前生成出來的字幕只有在活動會看到本月 AI 花費，但是字幕生成花了多少時間有地方紀錄嗎？」⚖️ **Alexyu 裁定：紀錄放在活動頁的每一筆紀錄後面。** Bob 依 Story Splitting Rule 拆出（見 A 單的 SM 裁定）。
設計稿：K1-D-v2 `kMeWS`（字幕完成列 `LndN2`、字幕沒做成列 `EVrLx`）、K1-M-v2 `QIwY1`（`JLTBV`、`akt4i`，說明下多一行 `rowMeta`）、元件 `Component/ActivityRow-v2`（`fF8nX`，沒改元件本身）、規格說明 `spec-note-k-run-time`（`XYUmV`）。桌機截圖 `_bmad-output/screenshots/flow-k-activity-v2/k1-d.png`；手機這一段在 `k1-m.png` 畫面外（被裁切），要看節點。

**依賴：** confirmed against `[@contract-v1]` (Story disc-2026-10-activity-subtitle-run-time-a AC #1)。照那張表寫，**不用等 A 合併才開工**。
⛔ **合併順序：本單先合併（或跟 A 同一次部署）。** 今天的 `RecentSectionView` 不看 `kind`（`ActivityHub.tsx:312-320`），A 先上線會把字幕事件畫成「解析完成」。本單先上線則完全無害：後端還沒送 `kind: "subtitle"`，畫面不變。

## 查到的事（2026-10-19，本 session 讀過的程式碼）

- **活動記錄的畫法**：`RecentSectionView`（`apps/web/src/components/activity/ActivityHub.tsx:301-332`）——`failed` → `AlertTriangle`＋error 色＋「解析失敗」，其餘 → `CircleCheck`＋success 色＋「解析完成」；說明 `ev.detail`；右側一個 `<span className="whitespace-nowrap text-xs text-[var(--text-muted)]">{formatRelativeTime(ev.at)}</span>`；沒有事件就 `return null`（`:309`）。它沒有 export，畫廊目前沒有活動記錄的夾具。
- **列元件**：`ActivityRow`（`ActivityRow.tsx:38-81`）＝ 圖示 · 標題＋說明 · 右側一格。**說明一律 `truncate` 單行**（`:61`）；右側 `shrink-0 text-right`（`:63`）。沒有第三行的位置。全 repo 沒有任何視覺基準用到 `ActivityRow`（`tests/visual/components.visual.spec.ts-snapshots/components/` 下只有 `activity-spend-card`，它不用 `ActivityRow`，`SpendSection.tsx:8`）。
- **型別**：`RecentEvent`（`apps/web/src/services/activityService.ts:61-67`）只有 `kind`／`result`／`detail?`／`at`；回應經 `snakeToCamel`（`:100`），所以 `spent_usd` → `spentUsd`、`duration_seconds` → `durationSeconds`、`failure_reason` → `failureReason`、`media_title` → `mediaTitle`、`season_number` → `seasonNumber`、`episode_number` → `episodeNumber`、`cue_count` → `cueCount`、`media_type` → `mediaType`。
- **錢**：`usd()` 回 `$0.27`（`apps/web/src/lib/currency.ts:47-50`，decimal.js 四捨五入到分）；比大小一律用 `gtUsd`／`ltUsd`（`:94`、`:102`），不准 `===`／`<`（`:10-21`）。設計稿寫「US$0.27」，所以是 `US` ＋ `usd(v)`（同字樣已出現在 `ApiKeysForm.tsx:97`）。
- **時間**：`formatRuntime(minutes)`（`apps/web/src/lib/formatMedia.ts:19-25`）已經是「47 分鐘／1 小時／2 小時 5 分」，正好是裁定 3 的「不到 1 小時寫 N 分鐘、以上寫 1 小時 5 分」。秒數那一段要自己補。`formatRelativeTime(iso, now = Date.now())`（`apps/web/src/utils/relativeTime.ts:6-22`）已經能注入 `now`。
- **失敗原因的人話**：`failureCopy`（`apps/web/src/components/subtitle/generationEventCopy.ts:28-71`）——子字串比對原始錯誤，文字：已達預算上限／找不到可用的音軌／伺服器缺少 ffmpeg／語音辨識未設定／AI 帳號沒有額度／API 金鑰無效／處理逾時／（其餘）生成失敗。**沒有「寫不進資料夾」那一條**；那句只在後端 409（`apps/api/internal/handlers/transcription_handler.go:350-351`：「影片所在的資料夾寫不進字幕，這次沒有花到錢。…」）。活動記錄只取前半句「影片所在的資料夾寫不進字幕」（設計稿 `EVrLx`），**不能帶「這次沒有花到錢」**：正式機和平使者 S01E01 是翻完才寫不進，花了 $0.196（A 單「查到的事」）。
- **路線字彙**的既有中文：`completeMessageForRoute`（`apps/api/internal/subtitle/solo_runner.go:424-437`）——deliver_direct「直接使用片內中文字幕，沒有花錢」、convert_then_deliver「片內簡體字幕已轉成繁體」、translate「翻譯片內英文字幕」、asr「轉錄完成」。
- **字級規則**：`ActivityHub.spec.tsx:589-592` 斷言整頁沒有 `text-[Npx]` 任意字級；設計稿桌機右上行是 `$Type/Body/Size`＝14（`text-sm`）、右下行與手機 `rowMeta` 是 `$Type/Label/Size`＝12（`text-xs`）（`_bmad-output/pen-tokens.json` 的 `Type/Body/Size`、`Type/Label/Size`）。
- **Rule 23**：`RecentSection` 會呼叫相對時間（讀現在時間）。用 `now` 注入（`// Clock-injected:`）就只需要一張基準；ESLint 規則 `local/time-dependent-fixture-stability` 會檢查檔頭（`project-context.md` Rule 23）。
- **畫廊夾具欄位**：`width?`／`viewport?`／`penNode`（`apps/web/src/routes/test/-gallery.fixtures.tsx:689-706`）；鄰近的花費卡夾具（`:6846-6875`）用 `width: 896` 與 `viewport: { width: 390, height: 844 }`。
- **E2E**：`git grep -n "activity-recent" -- apps tests` 只命中 `ActivityHub.tsx` 與 `ActivityHub.spec.tsx`；E2E 沒有斷言活動記錄。`tests/e2e/activity-spend.spec.ts:43` 的 stub 是 `recent: { status: 'ok', events: [] }`（不受影響）。`tests/support/helpers/generation-workspace-stubs.ts:67` 的 stub 寫成 `recent: { status: 'ok', items: [] }`（鍵名錯，應為 `events`）——那幾支只開 `?view=generation`，活動記錄不渲染，所以今天沒炸。

## Story

身為只有我一個人在用的 NAS 片庫主人，
我想要活動記錄裡每一次字幕生成，後面直接寫「花了幾分鐘 · 花了多少錢」，沒做成的寫為什麼，
這樣我掃一眼就知道哪一集值不值得、哪一集要處理。

## Acceptance Criteria

1. **型別（ack A 單 AC #1）：** `RecentEvent` 加 `mediaType?`、`mediaTitle?`、`seasonNumber?`、`episodeNumber?`、`route?`、`cueCount?`、`durationSeconds?`、`spentUsd?`、`failureReason?`；`kind` 型別寫 `'parse' | 'subtitle' | string`。`activityService.ts:61` 的「v1 sources parse events only」註解改成現況。
2. **標題與圖示：** `kind === 'subtitle'`：`completed` →「字幕完成」＋`CircleCheck`＋success；`failed` →「字幕沒做成」＋`AlertTriangle`＋error。`kind === 'parse'` 與其他 kind 的列**跟今天逐字相同**（標題、說明、右側一行時間、class）。
3. **說明（`片名 SxxExx · …`）：**
   - 片名：`mediaTitle`；`mediaType === 'episode'` 且有季、集 → 後面接 ` S01E02`（兩位數補零）；電影、影集不寫集數。`mediaTitle` 是空字串 →「已不在媒體庫的項目」（⚖️ Sally 2026-10-19 已追認，見 Dev Notes）。
   - 成功：` · ` ＋ 路線字：`translate`「翻譯 N 句」、`asr`「聽寫＋翻譯 N 句」、`deliver_direct`「片內中文字幕 N 句」、`convert_then_deliver`「簡體轉繁體 N 句」、`online`「線上字幕 N 句」、`""`／未知「N 句」（後四個 ⚖️ 待 Sally 追認）。沒有 `cueCount` → 只寫片名。
   - 失敗：` · ` ＋ `failureReason` 的人話（AC #6）。
   - 例：「末日光明 S01E02 · 翻譯 568 句」、「末日光明 S01E01 · 影片所在的資料夾寫不進字幕」。
4. **花多久 · 花多少錢（meta）：** 兩段用 ` · ` 接，缺的那段省略，兩段都缺就沒有 meta。
   - 花多久 `durationSeconds`：`< 1` →「不到 1 秒」；`< 60` →「N 秒」；其餘 `formatRuntime(Math.round(s / 60))`（211 →「4 分鐘」、3570 →「1 小時」、3900 →「1 小時 5 分」）。沒有這個欄位 → 不寫。
   - 花多少錢 `spentUsd`：沒有這個欄位 → 不寫（沒記錄 ≠ $0）；等於 0 →「沒花錢」；大於 0 但 `usd()` 會變成 `$0.00` →「不到 US$0.01」；其餘 `US` ＋ `usd(v)`（0.197439 →「US$0.20」）。比較只用 `gtUsd`／`ltUsd`。
5. **版面（裁定 2、5、6）：**
   - 桌機（`sm` 以上）：字幕列右側兩行、靠右——上行 meta（`font-mono text-sm text-[var(--text-secondary)]`），下行相對時間（`text-xs text-[var(--text-muted)]`）。說明單行（可 truncate）。
   - 手機（`sm` 以下）：右側**只留相對時間**；meta 移到說明下面當第三行（`font-mono text-xs text-[var(--text-secondary)]`）；字幕列的說明**可以換行**（不 truncate）。
   - meta 在 DOM 裡**只有一個節點**（用 CSS grid area 或同等做法在兩個位置之間移動），不准桌機、手機各放一份再用 `hidden` 切——不然讀螢幕軟體與測試會讀到兩次。
   - 解析列（與其他非字幕列）維持一行相對時間（裁定 5）。
   - 不出現 `text-[Npx]`（`ActivityHub.spec.tsx:589-592` 照綠）。
6. **失敗原因文字只有一份：** `generationEventCopy.ts` 新增 `FAILURE_COPY_BY_REASON: Record<RunFailureReason, FailureCopy>`，代碼＝A 單 AC #1 那十個；既有 `TABLE` 的每一列改成引用這張表裡的同一個物件（`failureCopy()` 對每個既有輸入的回傳**不變**，既有測試照綠）。新的兩個：`target_not_writable`「影片所在的資料夾寫不進字幕」、`cancelled`「中途停止」（⚖️ 待 Sally 追認）；`other` 與未知代碼「生成失敗」。活動記錄只用 `text`，不看 `stopped`（標題一律「字幕沒做成」，設計只有兩種狀態）。
7. **元件與檔頭：** 活動記錄搬到新檔 `apps/web/src/components/activity/RecentSection.tsx`（export `RecentSection`，props `section`、`onRetry`、`now?: number`），`ActivityHub` 改用它，畫面與 testid 不變。檔頭兩行：`// Design ref: ux-design.pen Screen K1-D-v2 (kMeWS) · K1-M-v2 (QIwY1) — 活動記錄列 (LndN2) (EVrLx) (JLTBV) (akt4i)` 與 `// Clock-injected: RecentSection accepts a \`now\` prop; no fixture-side mock needed`。時間／錢／說明的純函式放 `apps/web/src/components/activity/recentEventCopy.ts`（檔頭 `// Implements: <utility — no .pen counterpart>`，同 `generationEventCopy.ts:1`）。`ActivityRow` 加可選的 `meta?: ReactNode`（與手機說明換行的開關），沒傳時輸出的 DOM 與 class 跟今天一樣。
8. **testid：** 每列仍是 `activity-recent-row`，另加 `data-kind`（`parse`／`subtitle`）；meta 節點 `activity-recent-meta`、相對時間節點 `activity-recent-time`。
9. **單元測試：** `recentEventCopy.spec.ts`——AC #4 的每一個邊界（0.4、4、59、60、89、90、211、3570、3900 秒；0、0.004、0.005、0.197439、1.36365413125、沒有欄位）、AC #3 的每條路線與電影／單集／影集／片名空白、AC #6 每個代碼。`RecentSection.spec.tsx`——混排的解析＋字幕列標題正確；字幕列有一個且只有一個 `activity-recent-meta`；解析列沒有；`now` 固定時相對時間固定。`ActivityHub.spec.tsx` 既有測試全綠。
10. **畫廊夾具＋基準：** `activity-recent/subtitle-runs`（`width: 896`）與 `activity-recent/subtitle-runs-mobile`（`viewport: { width: 390, height: 844 }`），`penNode: 'screen-section'`，固定 `now`。三列（按時間由新到舊，跟後端排法一致）：字幕完成（末日光明 S01E02、translate、568 句、`durationSeconds: 180`、`spentUsd: 0.27`、2 分鐘前）、解析失敗（`The.Movie.2024.mkv`、1 小時前）、字幕沒做成（末日光明 S01E01、`target_not_writable`、`durationSeconds: 4`、`spentUsd: 0`、3 小時前）——字幕兩列的文字與設計稿逐字相同。本機產 `-darwin` 基準；**不產、不 commit `-linux`**（CI 的 Visual Regression 會開 bootstrap PR）。
11. **Rule 24 lane ①：** `tests/support/helpers/generation-workspace-stubs.ts:67` 的 `items: []` 改成 `events: []`。
12. `pnpm nx run web:test`（activity＋subtitle 相關 spec）、`lint:all`（看「✖ N problems (X errors」那行）、typecheck、`format:check`、`python3 scripts/check-design-refs.py` 綠。
13. **UX 驗收（必做）：** 桌機夾具截圖對 `k1-d.png` 活動記錄那兩列（文字、右側兩行的位置與顏色）；手機夾具對 `JLTBV`／`akt4i` 節點（Sally MCP 唯讀讀節點；手機截圖看不到這段）。量一次手機夾具：說明欄寬 ≥ 200px、meta 在說明下面、右側只有時間。

## Tasks / Subtasks

- [ ] **T1 型別（AC #1）**：`activityService.ts` 的 `RecentEvent`。
- [ ] **T2 文字（AC #3、#4、#6）**：`generationEventCopy.ts` 的 `RunFailureReason`＋`FAILURE_COPY_BY_REASON`、`TABLE` 改引用；新檔 `recentEventCopy.ts`（`formatRunDuration`、`formatRunSpend`、`recentSubtitleDetail`、`recentSubtitleMeta`）＋ spec。
- [ ] **T3 列元件（AC #5、#7）**：`ActivityRow` 加 `meta`／說明換行；grid area 讓同一個 meta 節點桌機在右上、手機在說明下。
- [ ] **T4 活動記錄（AC #2、#7、#8）**：新檔 `RecentSection.tsx`（從 `ActivityHub.tsx:301-332` 搬出、加字幕分支、`now` 注入）；`ActivityHub` 改用；spec。
- [ ] **T5 夾具＋基準（AC #10）**：兩個夾具、本機 `-darwin` 基準。
- [ ] **T6 收尾（AC #11、#12、#13）**：stub 鍵名、lint／typecheck／format／design-refs、UX 驗收與量測。

## Dev Notes

### ⚖️ Sally 2026-10-19 追認（四項都照 Bob 的建議）

設計稿只畫了翻譯（成功）與寫不進資料夾（失敗）兩種。下面是 Bob 照既有文字補的建議：
1. 其他路線的說明：asr「聽寫＋翻譯 N 句」（裁定 2 已定）；deliver_direct「片內中文字幕 N 句」；convert_then_deliver「簡體轉繁體 N 句」；online「線上字幕 N 句」；舊紀錄（route 空白；正式機成功紀錄有 18 筆是這種）「N 句」。
2. 取消（使用者按取消或容器重啟）：「中途停止」。
3. 片子已從媒體庫移除：片名寫「已不在媒體庫的項目」。
4. `AI_INVALID_RESPONSE`（正式機第二常見的失敗）落在「生成失敗」。要更具體（例如「AI 回覆的格式不對」）得 A、B 兩邊各加一個代碼——建議先不加。

Sally 追認理由：1–3 都沿用畫面上已經在用的說法，不新造詞；4 先落「生成失敗」是因為多一個代碼要前後端各改一處，等正式機真的常看到再加。

### 不要做的事

- 不要改設計稿裡解析列、掃描列的樣子（裁定 5）；不要動 `formatRelativeTime` 的規則。
- 不要把錢用 `===`／`<` 比（`currency.ts:10-21`）。
- 不要讓 meta 出現兩份（AC #5）。
- 不要依「花錢失敗」改成警告色或「已停止」：活動記錄只有成功／沒做成兩種（設計）。
- 不要產或 commit `-linux.png`（CLAUDE.md）。
- `font-mono` 目前沒有吃到 JetBrains Mono、中文會掉到系統字（`disc-2026-10-font-mono-not-jetbrains`，ready-for-dev）。本單不處理；那張單上線時會重產全部基準，本單的基準跟著重產即可。

### 實作提示

- 手機斷點用 `sm`（640px，同頁其他區塊）。grid 參考：`grid-template-areas: "icon text time" "icon meta time"`（手機）↔ `"icon text meta" "icon text time"`（桌機），meta 節點 `grid-area: meta`。用 Tailwind 的任意屬性或在 `ActivityRow` 內寫兩組 class 都可以，但不准 `text-[Npx]`。
- 說明換行只對字幕列開（手機）；解析列維持 truncate。
- 相對時間在桌機字幕列是 `text-xs`（設計 Label 字級），跟解析列今天的 `text-xs` 一樣。

### 測試標準

- Vitest＋Testing Library；斷言用可見文字與 testid（Rule 16）。visual：Playwright gallery，本機 darwin 基準。
- 不要背景跑測試（會留下孤兒 vitest）。

### Project Structure Notes

- 動到：`apps/web/src/services/activityService.ts`、`apps/web/src/components/activity/{ActivityHub,ActivityRow}.tsx`、新檔 `RecentSection.tsx`／`recentEventCopy.ts`（＋spec）、`apps/web/src/components/subtitle/generationEventCopy.ts`、`apps/web/src/routes/test/-gallery.fixtures.tsx`（在花費卡夾具後面加一段，避免跟 b1／b2 的夾具撞行）、`tests/support/helpers/generation-workspace-stubs.ts`、`tests/visual/…/activity-recent/` 下的 darwin 基準。
- 跟進行中的 `disc-2026-10-library-list-triage-b1`／`-b2` 只共用 `-gallery.fixtures.tsx`（各加各的區塊）。

### Time-dependent visual coverage

- **YES** — `RecentSection.tsx` 讀現在時間（相對時間）。採 Rule 23 (ii) `Clock-injected`：元件收 `now` prop，夾具傳固定值，**一張基準即可**（不需要 `clockTime`／`withFixedClock`）。基準路徑：`tests/visual/components.visual.spec.ts-snapshots/components/activity-recent/subtitle-runs/default-visual-darwin.png`、`…/subtitle-runs-mobile/default-visual-darwin.png`。

### References

- [Source: _bmad-output/implementation-artifacts/disc-2026-10-activity-subtitle-run-time-design.md#Sally 的設計裁定（1–6，6 改判）]
- [Source: _bmad-output/implementation-artifacts/disc-2026-10-activity-subtitle-run-time-a.md AC #1 `[@contract-v1]`]
- [Source: apps/web/src/components/activity/ActivityHub.tsx:1-3, 301-332, 424；ActivityRow.tsx:38-81]
- [Source: apps/web/src/components/subtitle/generationEventCopy.ts:1-71]
- [Source: apps/web/src/lib/currency.ts:10-21, 47-50, 94-110；lib/formatMedia.ts:19-25；utils/relativeTime.ts:6-22]
- [Source: apps/web/src/services/activityService.ts:61-77, 100]
- [Source: apps/web/src/routes/test/-gallery.fixtures.tsx:689-733, 6846-6875]
- [Source: project-context.md Rule 16、Rule 20（ack）、Rule 21、Rule 23、Rule 24]

## Dev Agent Record

### Agent Model Used

### Debug Log References

### Completion Notes List

### Discovery Triage

- **① expand-scope-in-place** — `generation-workspace-stubs.ts:67` 的 stub 鍵名寫成 `items`（應為 `events`）→ AC #11。

### File List
