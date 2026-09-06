# Story bugfix: 手機上候選清單的每一列，副標「路線 · 片長」被右側徽章擠到看不見

Status: review

## Story

As 在手機上打開「產生字幕」bottom sheet 的使用者,
I want 每一列都看得到片名、金額，以及完整的「這列會走哪條路線 · 片長多少」,
so that 我在按「開始產生」之前，知道每一部片為什麼要收錢、收多少，而不是看到一排被切掉一半的句子。

## Evidence

- 立案來源：`backlog-f15-row-mobile-identity-collapse`（sub-6-10b 於 2026-09-06 MCP 複審時填，Rule 24 lane ③）。
- 量測（Chromium 390×844，列寬 = 手機 bottom-sheet 真實內容寬，`generation-consent/list` 與 `grouped` 兩個 fixture）：副標需要 227–303px，實際只拿到 115–131px。「· 片長」那半段**每一列都看不到**；最慘的一列（`未知片長的電影`：`無文字字幕軌 → 語音辨識 + 翻譯 · 片長未知（估 45 分）` 需 303px、給 115px）連路線都被切掉一半。列本身沒有水平溢位（`scrollWidth - clientWidth = 0`），純粹是 flex 擠壓：副標 `truncate`，右側徽章區 `shrink-0`，身分欄 `flex-1 min-w-0`，窄的時候身分欄被徽章吃光。
- 根因：`apps/web/src/components/subtitle/consent/CandidateListPanel.tsx` 的 `CandidateRow` **只有一種列型**（桌面的三欄 flex），F15-M-v2（`fdu4y`）畫的手機列型從來沒有實作。
- 設計稿已經畫出答案（`ux-design.pen` F15-M-v2 `fdu4y`，以 MCP 讀出結構）：`textcol`（vertical, gap 4, fill）→ `line1` = `ident`（片名 + 未匹配）+ `meta`（[資料夾無法寫入] + 金額），`sub` 為 `textGrowth: fixed-width`（**會換行**），身分欄拿到 256px；沒有「抽取／語音辨識」路線徽章。
- 不是 sub-6-10b 造成：這條路線在 sub-6-10b 之前就已被截（單獨的路線句 ≈172px 仍 > 131px）；該 story 把字變長只是讓它更明顯。
- 視覺回歸抓不到：`-gallery.fixtures.tsx` 的 consent fixture 寬度是 900／720，沒有手機寬度；而且視覺 harness 把 viewport 釘在 1280×800（story 19-4 契約），任何 `sm:` 之類的 viewport 斷點在 gallery 裡永遠是桌面態。

## Acceptance Criteria

1. 列寬小於 36rem（576px）時，列改用 F15-M-v2 的排法：金額（以及「資料夾無法寫入」）搬到片名那一行的右側；「抽取／語音辨識」路線徽章不顯示（金額顏色已經說了 綠=抽取／琥珀=語音辨識）；副標「路線 · 片長」佔滿片名下方的整個寬度並**換行**，不再截斷。
2. 列寬大於等於 36rem 時，桌面列型**逐像素不變**：三個既有桌面 fixture（`generation-consent/list`／`grouped`／`over-budget`）對著已提交的 `-darwin` 基準線 0 diff。
3. 排版依**列所在清單的寬度**（CSS container query）而非 viewport 切換，所以 (a) 桌面上把對話框縮窄也會得到同樣的手機列型，(b) 一個 390px 寬的 gallery fixture 就能在 1280 viewport 的視覺 harness 裡釘住它。
4. 同一個 DOM 服務兩種排法：金額、未匹配、資料夾無法寫入都只有一個節點，沒有「桌面一份、手機一份」的重複（sub-6-10b 的 `consent-row-usd-*` 等 testid 契約不變）。
5. 新增 gallery fixture `generation-consent/list-mobile`（寬 390 = 設計稿的手機寬；body `p-6` 之後清單 342px，落在 36rem 之下），內容對齊 F15-M-v2 畫的四種列：一般抽取列、影集群組內的集數列、資料夾無法寫入列、未匹配＋片長未知列。提交 `-darwin` 基準線；`-linux` 由 main 的 incremental bootstrap 自動補（CLAUDE.md 規定不可在本機產）。
6. Gates：`pnpm nx test web` 全綠、`pnpm nx lint web` 0 errors、`pnpm run format:check` 綠。零後端變更。
7. （Alexyu 2026-09-06 追加，「ci typecheck 要修，合併在一起沒關係」）`pnpm nx run web:typecheck` 歸零，並加進 CI 的 lint job；期間發現的真 bug（空片庫 CTA 死路由）一併修掉。

## Tasks / Subtasks

- [x] Task 1 — 列改成 container-query 兩列 grid（AC #1–#4）
  - [x] 1.1 `<ul data-testid="consent-candidate-list">` 加 `@container`
  - [x] 1.2 `CandidateRow`：外層 `<li>` 的 flex（checkbox · 封面 · 身分欄）**原封不動**；身分欄改成 `grid min-w-0 flex-1 grid-cols-[minmax(0,1fr)_auto] items-center gap-x-3`，並把徽章群收進來：片名列 `col-start-1 row-start-1`；副標 `col-start-1 row-start-2 truncate @max-xl:col-end-3 @max-xl:whitespace-normal`；徽章群 `col-start-2 row-start-1 @xl:row-end-3`；路線徽章 `hidden @xl:inline`
  - [x] 1.3 只用 `*-start-*`／`*-end-*` 長寫，不用 `*-span-*`（見 Dev Notes）
  - [x] 1.4 第一版把整個 `<li>` 做成 grid，第一輪 CI 抓到桌面三張基準線各 ~1% diff；改成巢狀 grid 後本機量測 li 80px／封面 y=13／片名 y=22／副標 y=42，與舊版相同（見 Dev Notes）
- [x] Task 2 — 390px fixture（AC #5）
  - [x] 2.1 `-gallery.fixtures.tsx` 新增 `CONSENT_MOBILE_CANDIDATES` 與 `generation-consent/list-mobile`
  - [x] 2.2 產 `-darwin` 基準線並肉眼比對 F15-M-v2
- [x] Task 3 — 測試（AC #1、#3、#4）
  - [x] 3.1 `CandidateListPanel.spec.tsx` 新增 describe「phone-width row」5 個測試
- [x] Task 4 — Gates（AC #6）
- [x] Task 5 — typecheck 歸零 + 接進 CI（AC #7）
  - [x] 5.1 `-gallery.fixtures.tsx`（141／153 個錯誤）：`GalleryFixture.component` 改 `ComponentType<any>`（附理由與 eslint-disable），拿掉 166 個 `as ComponentType<Record<string, unknown>>` 轉型（其中 126 個本身就是 TS2352 錯誤）；5 個 request-row fixture 把「寫在前面卻被後面的 `...{}` spread 蓋掉」的 11 個死鍵改成實際生效的值（渲染完全相同）；TVShowInfo fixture 補齊 `TVShowDetails` 缺的 7 個欄位（該元件不渲染它們）；9R-10c EpisodeList fixture 補 `penNode: 'Z54xAd'`；本 story 自己的 `runtimeSource: 'measured'` 是不存在的值，改 `'ffprobe'`（typecheck 第一天就抓到我）
  - [x] 5.2 React 19 的 `useRef<T>()` 要初始值：5 個計時器 ref 改 `useRef<… | undefined>(undefined)`（EmptyReadyForScan／ScanProgressCard／ScanProgressSheet／ScannerSettings／useScanProgress／useSubtitleBatchProgress）
  - [x] 5.3 真 bug：`EmptyNoFolder`／`EmptyNoQBT` 的「設定媒體資料夾」連到不存在的 `/settings/libraries` → 改 `/setup`（設定精靈的媒體資料夾步驟就是今天設定資料夾的地方），spec 同步；`RecentMediaPanel` 的 `/media/$id` → `/media/$type/$id`；`MetadataExport` 的 `result.message` 在型別上可為空 → 補 `?? '匯出完成'`
  - [x] 5.4 `test.yml` lint job 加 `Typecheck (web)` 步驟（`pnpm nx run web:typecheck --skip-nx-cache`），ESLint 之後、format 之前
  - [x] 5.5 驗證：typecheck 0 錯誤；eslint 0；web 全套 3212 passed；視覺全套用正確字串 grep 0 diff（fixture 改動全是型別／metadata 層，不改渲染）

## Dev Notes

- **為什麼是 container query 而不是 `sm:`**：專案其他地方都用 viewport 斷點（bottom sheet 本身就是 `sm:` 切的）。但這列要解的是「拿到的寬度不夠」，不是「螢幕多大」——桌面上把對話框縮到 640px 一樣會擠；而且視覺 harness 的 viewport 釘死 1280，`sm:` 版的手機列在 gallery 裡永遠拍不到，backlog 要求的「390px fixture 釘住它」在 `sm:` 之下做不到（除非改 19-4 的 harness 契約讓每個 fixture 帶自己的 viewport）。Tailwind v4 內建 container query（`@container` / `@xl:` / `@max-xl:`），這是 repo 第一次用；建置後的 CSS 已確認產出 `@container (min-width:36rem)` 與 `@container not (min-width:36rem)` 兩組規則。
- **36rem 的來源**：桌面列最壞情況（未匹配長片名 + 資料夾無法寫入 + 路線徽章 + 金額）約需 620px 才完全不截；常見情況約 510px。取 Tailwind 的 `xl`（576px）：手機 bottom sheet（viewport − 48）在 viewport < 624 都走手機列；置中對話框（`calc(100vw-4rem)` − 48）在 viewport < 688 也走手機列——小平板直放拿到緊湊列型是合理的。
- **`*-span-*` 陷阱（第一版基準線就踩到）**：Tailwind 的 `col-span-2`／`row-span-2` 是 `grid-column`／`grid-row` **簡寫**，會把旁邊 `col-start-3` 設的起點重設成 `span 2`，格子退回自動排列——第一張手機截圖的副標跑到第 1–2 欄。改用 `col-end-3`／`row-end-3` 長寫後正確。
- **桌面逐像素不變，是第二次才做到的**：第一版把整個 `<li>` 改成四欄 grid、封面跨兩列。本機視覺跑完我用錯字串（`Screenshot comparison failed`）去 grep，誤報 0 diff；**第一輪 CI 的 Linux shard 抓到 `list`／`grouped`／`over-budget` 各 5,469–6,189 px（~1%）差異**。量測後根因：封面 54px 比兩行文字 36px 高，Chromium 把多出來的高度分進文字那兩個 track（實測 track 變成 17/20/16/17，每列高 96 而非 80）；試過 `[1fr auto auto 1fr]` 也一樣。最後改成**巢狀 grid**：外層 flex 一字不動，只有身分欄變 grid、徽章群搬進身分欄裡；grid 裡沒有比兩行字更高的東西，列高回到 80、封面 y=13、片名 y=22、副標 y=42（Playwright 量測），三張桌面 fixture 對 `-darwin` 基準線用**正確的字串**（`pixels (ratio`）grep 確認 0 diff。
- **手機 fixture 裡清單以外的東西**（chips、工具列、footer）仍是 viewport `sm:` 規則，在 1280 viewport 下被擠成桌面樣式（footer「已選 2 部 · 預估」直排就是這個）。那不是這個 fixture 的主題，也不是真機的樣子（真機 viewport < 640 走 `flex-col`）。若之後要讓整個面板都以 container 決定，是另一個 story。
- **群組 header 在 342px** 也很擠（路線徽章 + 已選 n/N · 金額），設計稿的手機 header 沒有路線徽章——屬於 `backlog-f15-group-header-pen-code-drift`，本 story 不動。
- Rule 7／10／20：N/A（無錯誤碼、無路由、無 wire contract）。Rule 23：N/A（無時間相依邏輯）。

### Discovery Triage（Rule 24）

- ③ `backlog-visual-bootstrap-parser-misses-current-playwright-diff-wording` — 就是上一條踩到的坑：`apps/web/src/visual-harness/bootstrap-detection.mjs` 靠 `/Screenshot comparison failed/` 認「真的像素差異」，但 Playwright 1.58 印的是 `expect(locator).toHaveScreenshot(expected) failed` + `N pixels (ratio 0.01 of all image pixels) are different.`，本機與 CI log 裡都 grep 不到舊字串。後果：main push 若**同時**有缺基準線與真差異，parser 算出 pixel_diff=0 → `bootstrap_needed=true` → 走 incremental bootstrap 開 PR、`Fail job on real regression` 被跳過，真差異靜默放行。本 PR 合併後就是「缺 1 張 + 0 差異」的情境，沒踩到，但下一個同時發生的就會。CI gate-integrity，非本 story 範圍，立案。

- ③→**併入本 story（Alexyu 裁定 2026-09-06）** `bugfix-empty-library-cta-dead-link-settings-libraries` — 跑 `pnpm nx run web:typecheck` 時發現 `EmptyNoFolder.tsx:24`／`EmptyNoQBT.tsx:32` 的 `<Link to="/settings/libraries">` 指向**不存在的路由**（`routeTree.gen.ts` 的 settings 子路由沒有 `libraries`；multi-library 的 PRD 完成但路由未建）。兩個元件由 `LibraryBrowseV2.tsx:624-626` 在空片庫狀態真的掛出來，使用者按「前往設定」會到 not-found。非本 story 範圍（列排版），立案。
- ③→**併入本 story（Alexyu 裁定 2026-09-06，Task 5）** `backlog-web-typecheck-red-and-not-in-ci` — `web:typecheck` 當時 **153 個錯誤**，而 CI（test.yml lint job）只跑 ESLint + Prettier，從不跑 typecheck，所以這些錯誤不會讓任何 PR 變紅。上面那條死連結就是這樣漏掉的。另含 `RecentMediaPanel.tsx:102` 的 `/media/$id`（已被 `RecentlyAddedRowV2` 取代的舊元件，未掛載，死碼）與 React 19 的 `useRef()` 無初始值錯誤等。

- ③ `backlog-visual-main-4-workers-flaky-media-detail-focus` — 為了替本 PR 補 `-linux` 基準線，對分支 `workflow_dispatch` 跑 Main job（run 34010222934）：verify-probe 4 桶全過（只缺 list-mobile），但接著的 `update-missing`（同一台 runner、同一份程式碼、幾分鐘後）在 `media-media-detail-panel/focus` 報 24,445 px（7%）差異——diff 圖是頂部 backdrop 帶 + 播放鈕的 focus 態，典型的時序抖動。同一 fixture 同一機器前後兩次不同結果 = flake，觸發條件是 #391 的 `--workers=4`（4 個 Chromium 搶 4 vCPU）。這次後果是 bootstrap step 失敗、沒開 PR（安全的失敗）；但它會讓 main 的 Main job 偶爾紅。#391 註解已寫「截圖不穩定先降 --workers」，這是第一筆觀察，先立案不動；再出現一次就降到 2。
- 本 PR 的 `-linux` 基準線因此改走 **手動 artifact extraction**（README／bootstrap PR body 明文的 PR #11 先例）：拿 CI 在 PR #392 shard 1 對同一份程式碼渲染的 `default-actual.png`（ubuntu-24.04，ImageVersion 20260831.293.1）直接入庫，audit doc 追加一行註明來源。不是本機產的 `-linux`（CLAUDE.md 禁的是那個）。

### References

- [Source: apps/web/src/components/subtitle/consent/CandidateListPanel.tsx — `CandidateRow`、`<ul @container>`]
- [Source: apps/web/src/routes/test/-gallery.fixtures.tsx — `CONSENT_MOBILE_CANDIDATES`、`generation-consent/list-mobile`]
- [Source: tests/visual/components.visual.spec.ts-snapshots/components/generation-consent/list-mobile/default-visual-darwin.png]
- [Design: ux-design.pen F15-M-v2 (`fdu4y`) item-list rows；`_bmad-output/screenshots/flow-f-subtitle-v2/f15-m-v2.png`]
- [Origin: sprint-status `backlog-f15-row-mobile-identity-collapse`；`sub-6-10b-candidate-identity-frontend.md` 複審]

## Dev Agent Record

### Agent Model Used

Claude Fable 5.1（claude-fable-5-1），2026-09-06

### Completion Notes

- 桌面三個 consent fixture 對 `-darwin` 基準線 0 pixel diff——**第二版（巢狀 grid）才成立**；第一版被第一輪 CI 抓到 ~1% diff，經過見 Dev Notes。驗證方式：本機 `VISUAL_BUCKETS=4 --workers=4` 跑全套視覺，以 `pixels (ratio` 為準的 grep 只剩 `glossary-panel-v2/seeded`（其 -darwin 基準線 2026-07-06 起就沒更新，本機字型漂移，CI 的 Linux shard 沒有這個失敗，與本 story 無關）；本來就缺 `-darwin` 的 `ui-tmdb-attribution*` 報 missing，產出的暫存 PNG 未提交。
- 手機 fixture 截圖逐列對照 F15-M-v2：沙丘（金額在片名列右側、副標單行）、全面啟動（資料夾無法寫入 + 金額同列、副標兩行）、星際效應（未匹配貼片名、≈ $0.24、副標第二行「（估 45 分）」）、怪奇物語 S04E07（群組內）——與稿子一致；路線徽章在所有列都不出現。
- 單元測試：consent + routes/test 141 passed；全套 web 3212 passed。ESLint 0 errors、Prettier 綠、建置產出的 CSS 含 container query 規則。
- `-linux` 基準線：本 PR 的 `Visual Regression / PR` 會因 `generation-consent/list-mobile` 缺 `-linux` 而紅（verify-only 的正確語意）；合併後 main 的 incremental bootstrap 會自動開 `chore(visual): bootstrap 1 missing -linux baselines` PR，貼 `requires-manual-review`，請 Alexyu 肉眼確認後合併。

### File List

- apps/web/src/components/subtitle/consent/CandidateListPanel.tsx
- apps/web/src/components/subtitle/consent/CandidateListPanel.spec.tsx
- apps/web/src/routes/test/-gallery.fixtures.tsx
- tests/visual/components.visual.spec.ts-snapshots/components/generation-consent/list-mobile/default-visual-darwin.png
- tests/visual/components.visual.spec.ts-snapshots/components/generation-consent/list-mobile/default-visual-linux.png（CI artifact，見 Discovery Triage）
- _bmad-output/audit/visual-baseline-19-4.md
- （Task 5）.github/workflows/test.yml、apps/web/src/components/library/{EmptyNoFolder,EmptyNoQBT}.tsx + .spec.tsx、EmptyReadyForScan.tsx、scanner/{ScanProgressCard,ScanProgressSheet}.tsx、settings/{ScannerSettings,MetadataExport}.tsx、dashboard/RecentMediaPanel.tsx、hooks/{useScanProgress,useSubtitleBatchProgress}.ts
- _bmad-output/implementation-artifacts/bugfix-f15-row-mobile-identity-collapse.md
- _bmad-output/implementation-artifacts/sprint-status.yaml
