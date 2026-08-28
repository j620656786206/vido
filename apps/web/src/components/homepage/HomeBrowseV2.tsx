// Design ref: ux-design.pen Screen H1-D-v3 (k2Otv)
// Companion frames: H7-D-v3 (EoCQ4) degraded · H2-M-v3 (uGCAU) mobile
/**
 * Home v2 composition (UX Redesign Phase 3 — ux3-1-2 / epic ux3-home-v2).
 *
 * Rendered by the `/` route (sole render since ux3-cutover-3). This is the D3
 * ordering law made structural: the OWN-CONTENT zone (繼續觀看 reserved slot +
 * 最近新增 row) is ALWAYS above the EXTERNAL curation (Hero + Explore). The
 * deterministic DOM-order assertion lives in HomeBrowseV2.spec; the *felt*
 * own-above-external experience is the post-build P10 manual browser gate
 * (390/768/1440), NOT a CI numeric AC.
 *
 * Dashboard remnants (DownloadPanel / QBStatusIndicator / ConnectionHistoryPanel)
 * are intentionally ABSENT here (ux3-1-4, D3 guardrail #3 — home is curation-first).
 * Their data stays reachable: QB/connection via the Epic-0 sidebar status strip
 * (ux3-0-4), in-flight downloads via the /downloads page. (DownloadPanel's eventual
 * home is the Activity hub = Epic 2; the temporary at-a-glance-download gap is a
 * known, acknowledged trade — Rule-24.) The legacy home (flag OFF) keeps them all.
 *
 * Each section is independently fail-soft (F3): Hero, Explore, and the recently-added
 * row each own their loading/empty/error state, so one failing section degrades alone
 * and the page never hard-fails. Full-bleed under the v2 shell — the Hero spans edge
 * to edge while the own-content zone and Explore blocks share the max-w-7xl gutter.
 */
import { HeroBanner } from './HeroBanner';
import { ExploreBlocksList } from './ExploreBlocksList';
import { HomeReadoutBand } from './HomeReadoutBand';
import { RecentlyAddedRowV2 } from './RecentlyAddedRowV2';

export function HomeBrowseV2() {
  return (
    <div data-testid="home-v2-root" className="flex flex-col gap-6 py-6 md:gap-8">
      {/* Keep the page identity visible even when data-backed sections degrade.
          It is intentionally restrained: the readout band remains the first
          operating surface, while「首頁」prevents an error state from looking
          like a standalone 最近新增 page. */}
      <h1 className="mx-auto w-full max-w-7xl px-4 text-lg font-semibold text-[var(--text-primary)] sm:px-6">
        首頁
      </h1>
      {/* Home v3 讀數帶 (ux3-1-7, H1-D-v3): the Operate readout sits above
          EVERYTHING — the returning user reads first, browses second. */}
      <HomeReadoutBand />

      {/* OWN-LIBRARY hero (ux3-1-8, H1-D-v3): the page's largest surface now
          sells the user's OWN shelf — static, manually switched, absent when
          no backdrop exists. D3 own-above-external holds and strengthens: the
          hero itself is own content now, so the ENTIRE first fold is yours. */}
      <HeroBanner />

      {/* OWN-CONTENT zone — still structurally ABOVE external curation (D3). */}
      <section
        data-testid="home-own-content"
        aria-label="我的媒體庫"
        className="mx-auto flex w-full max-w-7xl flex-col gap-6 px-4 sm:px-6"
      >
        {/* ⚖️ R3 ruling (2026-08-26): ContinueWatchingSlot is UNMOUNTED until
            Epic 17 exists. Its「連接 Plex / Jellyfin 後顯示」was a door that
            leads nowhere — 固定詞彙 says不出現＝你沒要求 is the only honest
            state for a feature nobody can enable. The component survives for
            Epic 17. */}
        <RecentlyAddedRowV2 />
      </section>

      {/* EXTERNAL curation retreats to the TAIL (ux3-1-8, H1-D-v3): TMDb rows
          filter out owned items (discovery is their only job) and the whole
          group is absent when TMDb is degraded (H7-D-v3 — the page stays
          complete and isomorphic without it). */}
      <ExploreBlocksList />
    </div>
  );
}
