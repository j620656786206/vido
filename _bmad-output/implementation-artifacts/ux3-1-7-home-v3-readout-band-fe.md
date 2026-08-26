# Story ux3-1-7 — Home v3 讀數帶 (readout band FE)

**Epic:** ux3-home-v3 · **Status:** ready-for-dev · **Type:** frontend
**Design:** H1-D-v3 (k2Otv) band, H2-M-v3 (uGCAU) 2×2, H8-SPEC-v3 (iWUSV) 金額規則 —
screenshots `flow-h-homepage-v3/` · **Consumes:** ux3-1-6 `GET /api/v1/home-summary`
`[@contract-v1]` · **Pairs with:** ux3-1-8 (hero+tail; independent, may land either order)

## What

The four-cell Operate readout at the very top of the homepage: 繁中字幕 42/55 ·
今天處理 3 部 · 需要注意（琥珀例外格）· 進行中 2 個任務. A dense BAND, not
dashboard cards. Every cell is a door.

## Acceptance criteria

1. New `homeSummaryService.ts` (fetchApi boundary camelCases per Rule 18) +
   `useHomeSummary` hook mirroring `useActivity`'s visibility-gated polling
   (30s interval, staleTime 25s — band numbers move slower than job counts; no
   SSE per tech-spec non-goals).
2. New `HomeReadoutBand` mounted in `HomeBrowseV2` ABOVE everything (band is
   first element; section order beneath it is NOT touched by this story —
   1-8 owns the hero/tail reorder).
3. Desktop: single-row flex, 4 equal cells, 11px labels + mono digits
   (`font-mono tabular-nums`); mobile: 2×2 grid, digits 16px, the 需要注意
   cell breaks to two lines (第一行 N 部失敗 / 第二行金額) per H8-SPEC-v3.
4. Each cell is one `<Link>` with ≥44px hit area. Doors: coverage →
   `/library`（若媒體庫已有缺字幕篩選參數則帶上；沒有就先進 /library 並以
   Rule-24 立案，不得發明查詢參數）; 今天處理 → `/activity`; 需要注意 →
   `/activity`; 進行中 → `/activity`.
5. Honesty states (brief §2/§5, cell-independent):
   - cell `status: "unavailable"` → that cell renders its label with NO number
     (量不到的格不顯示數字); siblings unaffected.
   - `0` renders as `0` (0 是資訊). Coverage 55/55 renders normally (慶祝).
   - attention `failed_count == 0` → 「一切正常」 in normal text colour —
     the cell never disappears (「沒有壞消息」本身是好消息).
   - attention `failed_count > 0` → text wears `--warning-text` amber
     (固定詞彙: 要求了但沒發生). Amber ONLY in the exception state.
   - coverage `0/0`（首跑）→ cell shows 0/0 with a 「開始掃描」 door to the
     scanner settings route (use the EXISTING route; do not invent).
6. Amount display implements H8-SPEC-v3 folding EXACTLY, as a pure util +
   unit tests: `<$10` one decimal ($1.2) · `$10–999` integer ($123) ·
   `≥$1000` k with one decimal ($1.2k); applies to both spent and budget.
   Spend shown only when the trio is present; copy varies by `spend_source`
   (`live_batch` → 執行中花費, `last_run` → 最近一次執行) — absent trio =
   no amount text at all (absent ≠ $0).
7. Component spec covers: 4-cell OK render, per-cell unavailable, 一切正常,
   amber exception, 0/0 first-run door, folding util cases, mobile grid
   classes, door hrefs. Full web suite + lint + format green.
8. Visual: new component picks up visual snapshots — darwin baselines
   committed; missing `-linux` arrive via the auto bootstrap PR after merge
   (never generated locally).

## Non-goals

- No SSE/live push; no reorder of hero/explore (1-8); no backend change.
- No sparkline/chart (brief anti-goal).
