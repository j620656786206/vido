// Design ref: ux-design.pen Screen H1-D-v2 (yixu1) — own-content zone, 繼續觀看 slot (wb1QN)
/**
 * Continue-watching reserved slot (UX Redesign Phase 3 — ux3-1-3).
 *
 * The 繼續觀看 own-content block is the FIRST element of the D3 own-above-external
 * zone, but Vido has no playback path yet — continue-watching data is blocked on
 * Epic 17 (P4-011). Per the design (H1/H5/H6-D-v2 reserved slot) this renders a
 * quiet, never-broken affordance: a bordered panel with 「連接 Plex / Jellyfin 後
 * 顯示」. It is purely presentational — NO data fetch, NO query, NO console noise
 * when there is no media server. When Epic 17 lands, this slot becomes a real
 * continue-watching row ABOVE the recently-added row (D3 ordering law).
 *
 * Token-only colors; Noto Sans TC (CJK). The affordance panel itself is not an
 * interactive control — it is an explanatory placeholder, so no 44px hit target.
 */
export function ContinueWatchingSlot() {
  return (
    <section data-testid="home-continue-watching" aria-labelledby="home-cw-title">
      <h2 id="home-cw-title" className="mb-3 text-xl font-semibold text-[var(--text-primary)]">
        繼續觀看
      </h2>
      <div className="flex h-24 items-center justify-center rounded-[var(--radius-lg)] border border-[var(--border-subtle)] bg-[var(--bg-secondary)] px-6 text-center">
        <p className="text-sm text-[var(--text-muted)]">連接 Plex / Jellyfin 後顯示</p>
      </div>
    </section>
  );
}
