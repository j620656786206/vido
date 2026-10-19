// Implements: <utility — no .pen counterpart>
/**
 * The ONE column table for the library list (disc-2026-10-library-list-triage-b1). The
 * header row (`LibraryListV2`) and every row (`LibraryListRowV2`) read the same string,
 * so the 字幕 column starts at the same x on every row — per-row flex let it drift
 * with the title's length.
 *
 * The list re-flows on ITS OWN width (`@container` on the list), not the window:
 * the sidebar (240) and the filter rail (264 + 24) take a different bite at every
 * window size, so a 1280 window with the rail open leaves 704 and a 1024 one 448.
 * ⚖️ Sally 2026-10-19 (story ⚖️ SM 提案):
 * - < 576 (base): the A4p-M three-line row — thumb · text · ›.
 * - `@xl` 576–671: the table without 規格.
 * - `@2xl` ≥ 672: A4p-D — thumb 48 · 片名 ≤360 · 字幕 104 · 評分 64 · 規格 ≥120 · › 20.
 *
 * 片名 is `minmax(0,360px)`, not the story's `minmax(200px,360px)`: at 672 in
 * selection mode the 20px indicator column (+16 gap) leaves the title 174 — a 200
 * floor would push the row 26px past the list's clipped edge. Outside selection mode
 * the title never drops under 200 at ≥672 anyway (672 − 2 − 24 − 236 − 80 − 120 = 210).
 */
export const LIBRARY_LIST_COLS =
  'grid-cols-[40px_minmax(0,1fr)_16px] @xl:grid-cols-[48px_minmax(0,1fr)_104px_64px_20px] @2xl:grid-cols-[48px_minmax(0,360px)_104px_64px_minmax(120px,1fr)_20px]';

/** Column gap: 12 on the three-line row (A4p-M), 16 in the table (A4p-D `$Space/lg`). */
export const LIBRARY_LIST_GAP = 'gap-x-3 @xl:gap-x-4';
