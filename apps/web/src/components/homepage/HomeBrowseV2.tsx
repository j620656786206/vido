// Design ref: ux-design.pen Screen H1-D-v2 (yixu1)
/**
 * Home v2 composition (UX Redesign Phase 3 — ux3-1-2 / epic ux3-home-v2).
 *
 * Rendered by the `/` route ONLY under the v2 shell (the route branches on
 * `useShellVersion()`; the flag stays read-once in __root, F4). This is the D3
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
import { ContinueWatchingSlot } from './ContinueWatchingSlot';
import { RecentlyAddedRowV2 } from './RecentlyAddedRowV2';

export function HomeBrowseV2() {
  return (
    <div data-testid="home-v2-root" className="flex flex-col gap-6 py-6 md:gap-8">
      {/* OWN-CONTENT zone — structurally ABOVE external curation (D3 ordering law). */}
      <section
        data-testid="home-own-content"
        aria-label="我的媒體庫"
        className="mx-auto flex w-full max-w-7xl flex-col gap-6 px-4 sm:px-6"
      >
        <ContinueWatchingSlot />
        <RecentlyAddedRowV2 />
      </section>

      {/* EXTERNAL curation — below own-content. Epic 10's Hero + Explore, kept in full. */}
      <HeroBanner />
      <ExploreBlocksList />
    </div>
  );
}
