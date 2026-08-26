// Design ref: ux-design.pen — no current screen frame; the 繼續觀看 slot was drawn
// only in the v2 homepage frames, and Home v3 removed the slot entirely (⚖️ R3:
// a door to a feature nobody can enable). Those v2 frames were deleted with the
// v3 cutover, so this DORMANT component honestly has no design to point at until
// Epic 17 gives it a real one.
/**
 * Continue-watching reserved slot (UX Redesign Phase 3 — ux3-1-3). UNMOUNTED
 * since the critique-R3 ruling; kept for Epic 17.
 *
 * The 繼續觀看 own-content block was the FIRST element of the D3 own-above-external
 * zone, but Vido has no playback path yet — continue-watching data is blocked on
 * Epic 17 (P4-011). It renders a quiet, never-broken affordance: a bordered panel
 * with 「連接 Plex / Jellyfin 後顯示」. It is purely presentational — NO data fetch,
 * NO query, NO console noise when there is no media server. When Epic 17 lands,
 * this slot becomes a real continue-watching row and needs a v3 design frame of
 * its own before it is mounted again.
 *
 * Token-only colors; Noto Sans TC (CJK). The affordance panel itself is not an
 * interactive control — it is an explanatory placeholder, so no 44px hit target.
 */
export function ContinueWatchingSlot() {
  return (
    <section data-testid="home-continue-watching" aria-labelledby="home-cw-title">
      <h2 id="home-cw-title" className="mb-3 text-lg font-semibold text-[var(--text-primary)]">
        繼續觀看
      </h2>
      <div className="flex h-24 items-center justify-center rounded-[var(--radius-lg)] bg-[var(--bg-secondary)] px-6 text-center">
        <p className="text-sm text-[var(--text-muted)]">連接 Plex / Jellyfin 後顯示</p>
      </div>
    </section>
  );
}
