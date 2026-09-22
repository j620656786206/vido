// Implements: <utility — no .pen counterpart>
/**
 * The ONE column table for the library grid (dsr-1b-c AC #2). `LibraryBrowseV2` (the
 * real grid) and `LibraryGridSkeletonV2` (what shows while it loads) both read from
 * here, so the page cannot reflow between "loading" and "loaded" — it did: at lg the
 * skeleton drew 4 columns over a 3-column grid, at xl 6 over 4 (invisible on a phone,
 * where both are 2, but the same component).
 *
 * `railOpen` / `railCollapsed`: the desktop filter rail takes the left column at lg+
 * (ux3-0-7), so the grid gets one fewer column while it is open. `gap`: phone 16
 * (A3p-M `wW2oF` gap `$Space/lg`), sm 12, md+ 16 — the sm/md values are the pre-existing
 * desktop spacing, unchanged.
 */
export const LIBRARY_GRID_COLS = {
  railOpen: 'grid-cols-2 sm:grid-cols-3 lg:grid-cols-3 xl:grid-cols-4 2xl:grid-cols-5',
  railCollapsed: 'grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 xl:grid-cols-6',
  gap: 'gap-4 sm:gap-3 md:gap-4',
} as const;
