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

## Tasks / Subtasks

- [x] Task 1 — 列改成 container-query 兩列 grid（AC #1–#4）
  - [x] 1.1 `<ul data-testid="consent-candidate-list">` 加 `@container`
  - [x] 1.2 `CandidateRow`：`flex` → `grid grid-cols-[auto_auto_minmax(0,1fr)_auto]`；checkbox／封面 `row-start-1 row-end-3`；片名列 `col-start-3 row-start-1`；副標 `col-start-3 row-start-2 truncate @max-xl:col-end-5 @max-xl:whitespace-normal`；徽章群 `col-start-4 row-start-1 @xl:row-end-3`；路線徽章 `hidden @xl:inline`
  - [x] 1.3 只用 `*-start-*`／`*-end-*` 長寫，不用 `*-span-*`（見 Dev Notes）
- [x] Task 2 — 390px fixture（AC #5）
  - [x] 2.1 `-gallery.fixtures.tsx` 新增 `CONSENT_MOBILE_CANDIDATES` 與 `generation-consent/list-mobile`
  - [x] 2.2 產 `-darwin` 基準線並肉眼比對 F15-M-v2
- [x] Task 3 — 測試（AC #1、#3、#4）
  - [x] 3.1 `CandidateListPanel.spec.tsx` 新增 describe「phone-width row」5 個測試
- [x] Task 4 — Gates（AC #6）

## Dev Notes

- **為什麼是 container query 而不是 `sm:`**：專案其他地方都用 viewport 斷點（bottom sheet 本身就是 `sm:` 切的）。但這列要解的是「拿到的寬度不夠」，不是「螢幕多大」——桌面上把對話框縮到 640px 一樣會擠；而且視覺 harness 的 viewport 釘死 1280，`sm:` 版的手機列在 gallery 裡永遠拍不到，backlog 要求的「390px fixture 釘住它」在 `sm:` 之下做不到（除非改 19-4 的 harness 契約讓每個 fixture 帶自己的 viewport）。Tailwind v4 內建 container query（`@container` / `@xl:` / `@max-xl:`），這是 repo 第一次用；建置後的 CSS 已確認產出 `@container (min-width:36rem)` 與 `@container not (min-width:36rem)` 兩組規則。
- **36rem 的來源**：桌面列最壞情況（未匹配長片名 + 資料夾無法寫入 + 路線徽章 + 金額）約需 620px 才完全不截；常見情況約 510px。取 Tailwind 的 `xl`（576px）：手機 bottom sheet（viewport − 48）在 viewport < 624 都走手機列；置中對話框（`calc(100vw-4rem)` − 48）在 viewport < 688 也走手機列——小平板直放拿到緊湊列型是合理的。
- **`*-span-*` 陷阱（第一版基準線就踩到）**：Tailwind 的 `col-span-2`／`row-span-2` 是 `grid-column`／`grid-row` **簡寫**，會把旁邊 `col-start-3` 設的起點重設成 `span 2`，格子退回自動排列——第一張手機截圖的副標跑到第 1–2 欄。改用 `col-end-5`／`row-end-3` 長寫後正確。
- **桌面逐像素不變的保證**：AC #2 用三個既有 fixture 對 `-darwin` 基準線驗過（0 diff）。grid 的 `items-center` + 兩列高度（20px 片名 + 16px 副標）與原本 flex 內兩個 block 疊起來相同；徽章群 `row-end-3` 跨兩列置中 = 原本 `items-center`。
- **手機 fixture 裡清單以外的東西**（chips、工具列、footer）仍是 viewport `sm:` 規則，在 1280 viewport 下被擠成桌面樣式（footer「已選 2 部 · 預估」直排就是這個）。那不是這個 fixture 的主題，也不是真機的樣子（真機 viewport < 640 走 `flex-col`）。若之後要讓整個面板都以 container 決定，是另一個 story。
- **群組 header 在 342px** 也很擠（路線徽章 + 已選 n/N · 金額），設計稿的手機 header 沒有路線徽章——屬於 `backlog-f15-group-header-pen-code-drift`，本 story 不動。
- Rule 7／10／20：N/A（無錯誤碼、無路由、無 wire contract）。Rule 23：N/A（無時間相依邏輯）。

### Discovery Triage（Rule 24）

- ③ `bugfix-empty-library-cta-dead-link-settings-libraries` — 跑 `pnpm nx run web:typecheck` 時發現 `EmptyNoFolder.tsx:24`／`EmptyNoQBT.tsx:32` 的 `<Link to="/settings/libraries">` 指向**不存在的路由**（`routeTree.gen.ts` 的 settings 子路由沒有 `libraries`；multi-library 的 PRD 完成但路由未建）。兩個元件由 `LibraryBrowseV2.tsx:624-626` 在空片庫狀態真的掛出來，使用者按「前往設定」會到 not-found。非本 story 範圍（列排版），立案。
- ③ `backlog-web-typecheck-red-and-not-in-ci` — `web:typecheck` 目前 **153 個錯誤**，而 CI（test.yml lint job）只跑 ESLint + Prettier，從不跑 typecheck，所以這些錯誤不會讓任何 PR 變紅。上面那條死連結就是這樣漏掉的。另含 `RecentMediaPanel.tsx:102` 的 `/media/$id`（已被 `RecentlyAddedRowV2` 取代的舊元件，未掛載，死碼）與 React 19 的 `useRef()` 無初始值錯誤等。

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

- 桌面三個 consent fixture 對 `-darwin` 基準線 0 pixel diff（AC #2 驗證方式：本機 `VISUAL_BUCKETS=4 --workers=4` 跑全套視覺，只有本來就缺 `-darwin` 的 `ui-tmdb-attribution*`／`retry-retry-notifications` 報 missing，與本 story 無關，產出的暫存 PNG 未提交）。
- 手機 fixture 截圖逐列對照 F15-M-v2：沙丘（金額在片名列右側、副標單行）、全面啟動（資料夾無法寫入 + 金額同列、副標兩行）、星際效應（未匹配貼片名、≈ $0.24、副標第二行「（估 45 分）」）、怪奇物語 S04E07（群組內）——與稿子一致；路線徽章在所有列都不出現。
- 單元測試：consent + routes/test 141 passed；全套 web 3212 passed。ESLint 0 errors、Prettier 綠、建置產出的 CSS 含 container query 規則。
- `-linux` 基準線：本 PR 的 `Visual Regression / PR` 會因 `generation-consent/list-mobile` 缺 `-linux` 而紅（verify-only 的正確語意）；合併後 main 的 incremental bootstrap 會自動開 `chore(visual): bootstrap 1 missing -linux baselines` PR，貼 `requires-manual-review`，請 Alexyu 肉眼確認後合併。

### File List

- apps/web/src/components/subtitle/consent/CandidateListPanel.tsx
- apps/web/src/components/subtitle/consent/CandidateListPanel.spec.tsx
- apps/web/src/routes/test/-gallery.fixtures.tsx
- tests/visual/components.visual.spec.ts-snapshots/components/generation-consent/list-mobile/default-visual-darwin.png
- _bmad-output/implementation-artifacts/bugfix-f15-row-mobile-identity-collapse.md
- _bmad-output/implementation-artifacts/sprint-status.yaml
