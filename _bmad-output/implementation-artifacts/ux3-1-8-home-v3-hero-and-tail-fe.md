# Story ux3-1-8 — Home v3 自家 hero＋TMDb 尾巴 (hero & tail FE)

**Epic:** ux3-home-v3 · **Status:** ready-for-dev · **Type:** frontend
**Design:** H1-D-v3 (k2Otv), H7-D-v3 degraded (EoCQ4), H2-M-v3 (uGCAU) —
screenshots `flow-h-homepage-v3/` · **Backend deps:** NONE (tech-spec D4 confirmed:
hero data = own library newest-with-backdrop, already served) · **Pairs with:** ux3-1-7

## What

The identity half: the hero sells YOUR library (static, manual switching), TMDb
retreats to a filtered tail, and the degraded page is the same page minus one block.

## Acceptance criteria

1. **HeroBanner data source flips to own library**: derive from the SAME
   `useRecentlyAdded(20)` query the row uses (TanStack dedupes — zero new
   requests): newest items WITH `backdropPath`, cap 5. No backdrops → hero
   renders `null` (例外訊號原則: absent, not an empty frame). Exactly 1 item →
   dots hidden, single static slide.
2. **Static**: the 8s autoplay interval, hover/focus pause handlers, and the
   pause button are REMOVED (nothing rotates, so WCAG 2.2.2 has nothing to
   stop — ⚖️ R3 autoplay ruling + shape ruling 靜止＋手動). Manual dots stay
   (≥44px targets on the scrim pill), plus prev/next chevrons per H1-D-v3.
   Manual switch keeps the asymmetric cross-fade (700/300).
3. **Hero content** per design: 最新入庫 eyebrow · title (stretched Link to
   the item's LIBRARY detail route — the id is OUR media id, not a TMDb id) ·
   year/type meta · subtitle-status badge from the EXISTING badge recipe
   (繁中字幕✓已就緒 green tint; 缺字幕 → amber, linking to the item's detail)
   · single gold CTA 查看詳情. The TMDb trailer CTA leaves the hero
   (TrailerModal component file survives; its hero call site goes).
4. **Section order in HomeBrowseV2** becomes: readout band (1-7, if landed) →
   own hero → 最近新增 → TMDb tail. D3 own-above-external is PRESERVED — the
   hero is now own content; update the deterministic ordering spec to assert
   the new truth (hero above 最近新增 above explore).
5. **TMDb tail filters owned**: `ExploreBlock` rows drop items where
   `ownership.isOwned(id)` (predicate already hoisted — frontend-only, zero
   new API). Row caption 「已擁有的作品不會出現這裡」 per design (muted, under
   the block title). While ownership is still loading, render UNFILTERED
   rather than flashing a shrink (no layout jump on settle: filter applies on
   first settled render).
6. **Degraded (H7-D-v3)**: TMDb absent → whole explore block group absent +
   the existing amber `ExploreDegradedNotice` as the page FOOTER line with
   前往連線設定 door; hero and 最近新增 unaffected (they never touch TMDb).
   Page must render complete with zero TMDb — degraded and full states are
   isomorphic.
7. **e2e sweep is MANDATORY** (the #318 lesson): `tests/e2e/hero-banner.spec.ts`
   asserts autoplay/pause behaviours that DIE with this story — rewrite them
   into static-hero + manual-switch + own-content guards; run the e2e suite
   locally (`pnpm exec playwright test --project=chromium`) before pushing.
8. **Visual baselines**: hero/homepage visual snapshots CHANGE — update darwin
   baselines locally; DELETE the stale `-linux` twins in the same commit so
   the incremental bootstrap can regenerate them (a pixel-DIFF blocks the
   bootstrap; only MISSING files get filled — the #321 lesson). CI visual red
   on this PR is expected only for the deleted files' first run.
9. Component specs updated (HeroBanner static behaviour, ExploreBlock filter,
   HomeBrowseV2 order); full web suite + lint + format green.

## Non-goals

- 最近新增 row internals, sidebar/shell, Activity hub: untouched (brief §4).
- No new state colours; no autoplay under any flag.
