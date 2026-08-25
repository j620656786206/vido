# Fix: third-critique P1 batch — one header, honest colours, real thumbs

Status: ready-for-review

> Closes the three P1s from critique R3 (29/40), plus two live finds from
> Alexyu's own ultra-wide screenshot mid-run.

## P1#1 — one header contract (was three systems)

All 8 settings pages now render a ROUTE-level `h1 text-2xl (mb-2)` + one-line
static description (`mb-6`) — the connection/keys pattern, which itself was
inconsistent (mb-6/mb-8 vs mb-2/mb-6) and is now aligned too. Components keep
only their LIVE data readouts (總計 X · 共 N 筆), demoted from heading anatomy
to icon+text lines. ExploreBlocks keeps its action button, right-aligned.
Live-verified: 8/8 pages h1 = 24px at identical x.

## P1#2 — the vocabulary stops lying at the two highest-stakes moments

BackupManagement feedback now carries its OUTCOME (`{tone, text}`):
ok → neutral (done-ness never wears green/gold), warn → --warning-*,
error → --error-*. Previously「校驗碼不符，可能已損壞」wore 你在這裡 gold and
「還原完成」wore 你要求了但沒發生 amber. Emoji ✅/⚠️ replaced with lucide —
they were the app's only non-lucide glyphs. LogEntry/LogFilters INFO moves
from gold to --info-* : thousands of log rows in gold would dilute the one
colour that must stay rare.

## P1#3 — 44px thumbs on the doing-controls

Mobile-first `min-h-[44px] … sm:min-h-0` (the tab strip's proven pattern,
inverted): 測試連線, per-type 清除/取消, log expand toggle (glyph stays 16px,
hit area 44×44), all four ExploreBlocks row actions incl. delete.
Live-verified at 390px: 44 / 44×44 / 44×44.

## Alexyu's mid-run finds (from his ultra-wide screenshot)

1. **Centering ruling reversed**: spare width no longer piles on the right —
   the strip and content share one `mx-auto max-w-6xl` container. 1152px, not
   1024: the ten-tab strip measures 1072 and the 1440 pane is exactly 1152, so
   the change is INVISIBLE at 1440 and only buys balance on wide screens.
   (The historical dead-gap bug centered the layout root while a second rail
   existed; the rail is a tab strip now, so nothing can detach.)
2. **「檢查於 739852 天前」**: the status API returns Go's zero time and
   formatRelativeTime rendered an absurd day count. Pre-2000 sentinels now
   fall back to the honest 尚未檢查. Guarded + falsified (and re-applied after
   a `git checkout` reverted the uncommitted fix — the same trap as the R2
   falsification, now noted twice).

## Verification

- Suites: settings 324/324, relativeTime 6/6 (guard falsified), full suite
  green except the 16 pre-existing eslint-rules env failures.
- 6 spec files updated where they asserted the removed in-component headers —
  each now asserts what legitimately remains (data readouts / testids).
- Visual: 6 components' darwin baselines regenerated (header removal); -linux
  deleted for CI bootstrap. Centering + zero-time guard added zero further churn.
- Live measurements: headers 8/8 uniform; targets 44px; pane margins equal
  (368/368 at 2000px).
