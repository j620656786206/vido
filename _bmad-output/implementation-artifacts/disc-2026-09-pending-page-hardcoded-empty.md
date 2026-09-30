# Story: disc-2026-09-pending-page-hardcoded-empty — 刪掉永遠說「沒有東西」的待解析頁

Status: review

<!-- SM Bob create-story 2026-09-30。Alexyu：「把這功能留下來，照上面 1 到 5 的步驟去做」（Party Mode 兩輪結論，步驟 1）。
     原單：TestSprite 九月額度衝刺第 3 輪建案時 curl 核實。行號為 main `3e6e354a`。 -->

## Story

身為打開 `/pending` 的人，
我不該看到一頁寫死的「尚未有待解析的媒體檔案」——明明片庫裡有 2 部整理中的片。
這頁沒有任何地方連過去、也沒有資料來源；「待解析」這件事已經有別的地方在做，所以整頁刪掉。

## 背景（查到的事）

- `apps/web/src/routes/pending.tsx:8-24`：整個元件是靜態 JSX，沒有任何 query、沒有 props——文案永遠是「尚未有待解析的媒體檔案」。
- `git grep "to=\"/pending\"\|to: '/pending'" -- apps/web/src` 零命中：沒有連結帶使用者過去，只有手打網址才看得到（v1 的 `TabNavigation` 待解析分頁早已隨 v2 殼層退場）。
- 同一件事現在由三處承接：
  - `components/activity/ActivityHub.tsx:178-192`「待解析項目 · N 個項目待處理 → 前往處理」，連到 `/library?unmatched=true`。
  - 媒體庫 `FilterChips.tsx:117` / `FilterPanel.tsx:12` 的 `unmatched` 篩選。
  - 首頁 `RecentlyAddedRowV2.tsx:148-158`「整理中 · N」連到 `/activity`。
- 唯一引用者：`routes/pending.spec.tsx`（只測靜態文案）、`routes/test/gallery.tsx:119` 與 `-gallery.fixtures.tsx:644` 的 stub 路徑清單（沒有任何 fixture 用 `routePath: '/pending'`）、`tests/visual/README.md:190` 文件。

## Acceptance Criteria

1. `/pending` 路由不存在：`routes/pending.tsx`、`pending.spec.tsx` 刪除，`routeTree.gen.ts` 不再含 `PendingRoute`。
2. 圖庫 stub 路徑清單（`gallery.tsx` `STUB_TAB_PATHS`、`-gallery.fixtures.tsx` `StubRoutePath`）與 `tests/visual/README.md` 同步拿掉 `/pending`。
3. 沒有任何產品連結或測試再指向 `/pending`（`git grep "/pending'"` 只剩 `services/retry.ts` 的 `/retry/pending` API 路徑）。
4. web 全量測試、lint、typecheck 綠。

## Tasks / Subtasks

- [x] Task 1 — 刪頁面與 spec、更新 stub 路徑與文件
- [x] Task 2 — 全量測試、lint、typecheck

## Dev Notes

- 這頁是 Story 5-0（2026-03）殼層時代的佔位頁（「to be created in future story」），後繼者是活動中心（ux3-2-x）。
- 不新增任何 redirect：從沒有入口，沒有書籤可保護。

### Time-dependent visual coverage

- N/A — 沒有畫面新增；刪掉的頁面沒有視覺基準。

## Dev Agent Record

### Agent Model Used

Claude Fable 5.1（Amelia）

### Completion Notes List

- 刪 `pending.tsx`／`pending.spec.tsx`；`routeTree.gen.ts` 由 TanStack 外掛自動重新產生（−21 行）。AC3 grep：只剩 `services/retry.ts` 的 `/retry/pending`。
- 全量：`pnpm nx test web` 288 檔／4,391 條綠；`web:typecheck` 綠；eslint 0 errors；prettier 全過（與 `bugfix-learned-patterns-ui-orphaned` 同一次跑）。
- 🔗 AC Drift: NONE。📎 Contract Stamps: NONE。🎭 A11y Pre-Flight: PASS（只刪）。🎨 UX Verification: SKIPPED — 沒有新畫面。

### Discovery Triage

- 無。

### File List

- apps/web/src/routes/pending.tsx、pending.spec.tsx（刪除）
- apps/web/src/routeTree.gen.ts（重新產生）
- apps/web/src/routes/test/gallery.tsx、-gallery.fixtures.tsx
- tests/visual/README.md
- _bmad-output/implementation-artifacts/disc-2026-09-pending-page-hardcoded-empty.md、sprint-status.yaml

## Change Log

| Date       | Change                                                            |
| ---------- | ----------------------------------------------------------------- |
| 2026-09-30 | create-story（SM Bob）；dev-story（Amelia）→ review。同 PR 與 `bugfix-learned-patterns-ui-orphaned` 一起出。 |
