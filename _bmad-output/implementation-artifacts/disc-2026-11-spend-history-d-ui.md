# Story disc-2026-11-spend-history-d-ui：花費歷程頁（前端）

Status: done

**Source:** 傘狀 `disc-2026-11-spend-history`。照 `disc-2026-11-spend-history-c-design.md` 的 Sally 設計決定（**沒有 .pen 稿**，⚖️ Alexyu 2026-11-01：Pencil 要收費，先用現有 design system 做）實作新頁和活動頁入口，讀 -b 的 `GET /api/v1/subtitles/runs`。

## Story

身為想知道錢花在哪裡的人，我希望打開一頁就能看到每天花了多少、每一次跑了哪支片、用了哪些模型、實際和預估差多少。

## Acceptance Criteria

1. [x] `SpendCard` 加「看明細 →」連到 `/activity/spend`。
2. [x] `/activity/spend`：標題列（返回、月份切換、不能切到未來）、摘要列（總額、次數、預估準度）、每日堆疊長條圖（inline SVG，四類顏色＋圖例，桌機 hover／手機點擊顯示當日明細）、逐筆列表（桌機表格、手機卡片、處理方式 chip 篩選、「載入更多」）——全部照 -c 的設計決定。
3. [x] 處理方式的字照 -c 的對照表；「語音辨識」不寫成「聽寫」。
4. [x] 誠實規則：沒記錄的寫「沒記錄」，不顯示 US$0.00；實際超過預估 50% 加 ⚠；失敗／中途停止照列照算。
5. [x] 載入／空／錯誤三種狀態，錯誤可重試。
6. [x] 測試：元件 spec（含沒記錄、超出預估、失敗列、空狀態、手機版面）、gallery fixture（不要自己產 `-linux` 基準，CI 會開 bootstrap PR）、e2e 一條（從活動頁點「看明細」到看到列表）。
7. [ ] 完成前用瀏覽器實際打開頁面（桌機＋手機寬度）截圖，跟 K5 截圖並排看是同一個家族（取代原本對照 .pen 稿的 UX 驗證）。

## Dev Agent Record

### 做了什麼

- **入口**：`SpendCard` 右下角加「看明細 ›」（`Link` 到 `/activity/spend`，箭頭用活動頁其他連結同款的 `ChevronRight`）。月份沒紀錄時卡片不顯示，入口跟著不顯示（沿用原本行為）。
- **路由**：`routes/activity_.spend.tsx` → 網址 `/activity/spend`。用 TanStack 的「`activity_`」寫法不巢狀在 `ActivityHub` 底下（`ActivityHub` 沒有 `<Outlet/>`，也不該為整頁另一個頁面長一個）。`?month=YYYY-MM`（過去月份才留；本月或未來一律丟掉，Rule 26：`2026-10` 不是合法 JSON，不會被轉成數字）＋`?category=`（四類 enum）。`routeTree.gen.ts` 用 repo 的 `@tanstack/router-generator`（跟 vite plugin 同設定）重新產生，只多了這一條。
- **API 層**：`services/subtitleRunHistoryService.ts`（snake → camel，Rule 18）。注意：`snakeToCamel` 會把 `by_category` 的鍵也改成 `chineseTrack`，邊界把它轉回分類值，讓 `byCategory[item.category]` 和 chip 用同一套字。`hooks/useSubtitleRunHistory.ts`：`useInfiniteQuery`，一頁 50 筆、「載入更多」接下一頁；切分類時保留這個月的摘要和圖表（`placeholderData` 只在同月份生效），切月份時改顯示骨架（避免新月份標題下面是舊月份數字）。
- **頁面**：`SpendHistoryPage.tsx`（`SpendHistoryView` 純 props，gallery 直接畫；`SpendHistoryPage` 接 query）。標題列（← 活動、AI 花費明細、‹ 2026 年 10 月 ›，本月時 › 停用）、摘要（`SpendStat`，從 `SpendSection` 抽出來共用，K5 同一家族）、說明「依開始時間歸到這個月，失敗和中途停止的也算進去」、沒記錄金額的次數與準度只算幾次。
- **圖表**：`SpendDailyChart.tsx`，inline SVG，`viewBox` 每天 10 單位、`preserveAspectRatio="none"` 撐滿寬度；長條依四類堆疊（片內中文字幕 `--success`、翻譯英文字幕 `--info`、語音辨識＋翻譯 `--accent-primary`、舊紀錄 `--text-muted`；`--warning` 留給 ⚠）；沒花錢的日子畫 1 單位底線。所有文字（日期軸、最高值、當日明細、圖例）用 HTML，不會被拉伸。每天一個整欄透明點擊區（`role=button`、可 tab）：桌機 hover 預覽、手機點一下固定，明細顯示在圖下方固定高度的區塊（不跳版）。月底 30／31 不會擠在一起。金額比較一律走 `lib/currency`。
- **列表**：`SpendRunList.tsx`，桌機 7 欄表格（md 以上）、手機四行卡片；處理方式 chip（全部＋四類）；字照 -c 對照表（`spendHistoryCopy.ts`），「語音辨識」不寫成「聽寫」。誠實規則：沒記錄 → 「沒記錄」（muted，不顯示 $0.00；真的 $0 照寫 $0.00）；實際 > 預估 × 1.5（精確小數判斷，剛好 1.5 倍不算）→「⚠ 比預估高 N%」；失敗（附原因，沿用 `runFailureCopy`）／中途停止（`cancelled`）照列照算；已刪除影片寫「影片已刪除」；模型列翻譯模型＋「ElevenLabs scribe_v2」，都沒有寫「—」。
- **狀態**：載入（摘要／圖表／列表各一塊骨架，列表沿用 `ActivitySkeleton`）、空（「這個月還沒有生成過字幕」＋回活動頁；標題列和月份切換仍在）、錯誤（`ActivitySectionError` 無法載入＋重試）、篩選後沒資料（摘要和圖表保留，列表寫「這個月沒有這種處理方式的紀錄」）、載入更多失敗（就地重試）。

### 實作時做的判斷

- 金額格式沿用 `usd()` 的 `$4.50`（跟 K5 卡片一樣），設計稿文字寫的「US$0.00」當作示意。
- 時間欄用 `formatLocalDateTime` 去掉年份（`10-28 13:33`，瀏覽器時區）；月份標題在 URL 沒指定時跟伺服器回的 `month`。「不能切到未來」用瀏覽器時鐘算本月（只用在路由，元件不碰時鐘，Rule 23）。
- 視覺基準：`SpendCard` 多了連結，4 張 `activity-spend-card` 的 `-linux` 基準一定會有像素差，所以**刪掉**（不是重產），讓 CI 當成缺基準自動補（memory `infra_visual_regression_genuine_diff_baseline`）。`-darwin`：4 張 SpendCard 用 `--update-snapshots=changed` 更新、3 張新頁（desktop／mobile／empty）新增；其他 fixture 沒有任何變動。**沒有產生任何 `-linux.png`。**

### 檔案

- 新增：`apps/web/src/services/subtitleRunHistoryService.ts`（＋spec）、`apps/web/src/hooks/useSubtitleRunHistory.ts`、`apps/web/src/components/activity/spendHistoryCopy.ts`（＋spec）、`SpendDailyChart.tsx`、`SpendRunList.tsx`、`SpendHistoryPage.tsx`（＋spec）、`apps/web/src/routes/activity_.spend.tsx`（＋spec）
- 修改：`apps/web/src/components/activity/SpendSection.tsx`（看明細＋抽出 `SpendStat`）、`SpendSection.spec.tsx`、`ActivityHub.spec.tsx`、`apps/web/src/routeTree.gen.ts`、`apps/web/src/routes/test/-gallery.fixtures.tsx`（3 個新 fixture）、`tests/e2e/activity-spend.spec.ts`（+1 條）
- 視覺基準：`tests/visual/components.visual.spec.ts-snapshots/components/activity-spend-card/*/default-visual-darwin.png`（更新）、`…/activity-spend-card/*/default-visual-linux.png`（刪除，等 CI 補）、`…/activity-spend-history/{desktop,mobile,empty}/default-visual-darwin.png`（新增）

### 測試

- `pnpm nx test web`：303 files／4990 tests 全過（新：頁面 spec 21、copy spec 18、service spec 3、route spec 3、SpendSection +1、ActivityHub +1）。涵蓋：沒記錄、⚠（含剛好 1.5 倍不算）、失敗／中途停止列、空狀態、錯誤＋重試、手機卡片四行、chip 篩選、月份不能切到未來（按鈕停用＋URL 丟掉未來月份）、跨年切換、看明細連結與導覽、圖表 hover／點擊固定、$0 底線、圖例。
- `pnpm nx lint web`：0 errors（168 warnings 都是既有的，新檔 0 warning）。`pnpm nx run web:typecheck`：通過。`prettier --check`（改到的檔）：通過。
- e2e：`tests/e2e/activity-spend.spec.ts` 在 chromium＋mobile-chrome 跑 6/6 過（新的一條：活動頁點「看明細」→ `/activity/spend` → 看到兩筆，含 ⚠ 84%、中途停止、沒記錄）。
- 視覺（darwin，本機）：`pnpm run test:visual`（4 buckets）verify 通過。

### 沒做的

- **AC 7**（瀏覽器實際開頁、桌機＋手機截圖跟 K5 並排）留給 Alexyu 驗。gallery 的 darwin 基準圖可以先看：`…/activity-spend-history/desktop`、`mobile`。
- 後端 `-b` 還在另一個 PR，這頁照合約用 mock 做；合併後要用真資料點一次。

### AC 7 瀏覽器驗證（2026-11-01，review）

- 本機用正式機資料庫副本（跑過 migration 048）起後端＋`nx serve web`，Playwright 截 `/activity/spend?month=2026-10`：桌機夜行、桌機日巡（滑過 10/31 長條顯示「合計 $8.02 · 38 次・語音辨識＋翻譯 $8.02」）、手機 Pixel 7。10 月 82 次、$23.34，四類顏色與圖例一致，手機列表是卡片，「沒記錄」照規則顯示，跟 K5 卡片同一套邊框／底色／字級。
- 本機 `vitest`（activity、service、route 共 8 檔 182 個測試）、`web:typecheck`、`nx lint web`（0 errors）重跑通過。
- 本機環境備註：`node_modules` 原本是 x86_64 版（nx／esbuild 在 arm64 node 跑不起來），用 `pnpm install --frozen-lockfile` 重裝成 arm64，lockfile 沒變。

## Change Log

- 2026-11-01：建立（Bob）。
- 2026-11-01：-c 改成不出稿，AC 依 Sally 設計決定補齊。
- 2026-11-01：實作完成（Amelia），Status → review；AC 7 瀏覽器截圖驗證留給 Alexyu。
