// Design ref: ux-design.pen Screen A2p-D (EsoIv) · A2p-M (qBWQC) · A4p-D (b1H71g) · A4p-M (EkB8g) · A7p-D (R3FqJc) · A8p-D (dVGIa)
// Skeleton / no-result / error, one frame each. (Was still carrying the 19-8
// `pending` placeholder, which Rule 21 says no components/ file should have.)
/**
 * v2 Browse state components (UX Redesign Phase 2 — UX2-2, AC #6, §7).
 * Loading skeleton (grid-shaped, reduced-motion aware), no-result (distinct from
 * empty — acknowledges the filter), and a per-section fail-soft error (compact
 * inline + error code + retry; the page never hard-fails — F3). The Empty state
 * reuses the existing EmptyLibrary 3-state classifier in the container.
 */
import { SearchX, AlertTriangle } from 'lucide-react';
import type { LibraryMediaType } from '../../types/library';
import { LIBRARY_GRID_COLS } from './libraryGridCols';
import { LIBRARY_LIST_COLS, LIBRARY_LIST_GAP } from './libraryListCols';

/**
 * 沒有「電影」符合… — the subject of the no-result sentence.
 *
 * `all` is 「內容」, NOT 「項目」: the page header four lines above counts the same
 * rows in 部 (⚖️ 2026-09-16), and one screen saying 「媒體庫 0 部」 over
 * 「沒有項目符合…」 is the same two-voices problem that ruling was made to end.
 * (Whether 部 is right for the 影集 and mixed views at all is open —
 * disc-2026-09-count-noun-only-evidenced-on-movies.)
 */
const TYPE_NOUN: Record<LibraryMediaType, string> = {
  all: '內容',
  movie: '電影',
  tv: '影集',
};

/**
 * Skeleton matching the grid shape — one tile per card, the SAME column table as the grid
 * (dsr-1b-c: `LIBRARY_GRID_COLS`, so loading→loaded never reflows), each tile a 2:3 poster
 * block, a full-width title line and a 60px meta line (A2p-M `qBWQC`: 171×294 = the card).
 */
export function LibraryGridSkeletonV2({
  count = 12,
  railCollapsed = false,
}: {
  count?: number;
  /** Mirror of the page's rail state so the skeleton draws the columns the grid will. */
  railCollapsed?: boolean;
}) {
  const cols = railCollapsed ? LIBRARY_GRID_COLS.railCollapsed : LIBRARY_GRID_COLS.railOpen;
  return (
    <div
      data-testid="library-grid-skeleton"
      aria-busy="true"
      aria-label="載入中"
      className={`grid ${LIBRARY_GRID_COLS.gap} ${cols}`}
    >
      {Array.from({ length: count }).map((_, i) => (
        // Same skeleton as PosterCardV2's box model: poster, then ONE text block whose title
        // area is the card's `min-h-[2.75em] text-sm` two-line reserve and whose meta line is
        // a 12px (text-xs) mono line box — so a tile is exactly as tall as the card it stands in for.
        <div key={i} className="flex flex-col gap-2">
          <div className="aspect-[2/3] animate-pulse rounded-[var(--radius-lg)] bg-[var(--bg-tertiary)] motion-reduce:animate-none" />
          <div>
            <div className="flex min-h-[2.75em] items-center text-sm leading-snug">
              <div className="h-3 w-full animate-pulse rounded-[var(--radius-sm)] bg-[var(--bg-tertiary)] motion-reduce:animate-none" />
            </div>
            <div className="mt-0.5 font-mono text-xs">
              <span className="inline-block h-2.5 w-[60px] animate-pulse rounded-[var(--radius-sm)] bg-[var(--bg-secondary)] align-middle motion-reduce:animate-none" />
            </div>
          </div>
        </div>
      ))}
    </div>
  );
}

// No `display` utility: every bar is a flex or grid item (blockified), and a `block`
// here would fight the 規格 bar's `hidden @2xl:block`.
const FILL = 'bg-[var(--bg-tertiary)]';
/** A text-line placeholder (the pill and the thumb set their own radius instead). */
const BAR = `${FILL} rounded-[var(--radius-sm)]`;
/** Title widths, cycled, so eight rows do not read as one stamped shape. */
const TITLE_WIDTHS = ['w-2/5', 'w-1/2', 'w-1/3', 'w-3/5'];

/**
 * Skeleton for the LIST view (disc-2026-10-library-list-triage-b2 AC #6) — the list used
 * to load under the GRID skeleton, so switching to 列表 showed poster tiles first.
 *
 * Same box as `LibraryListV2`: the `@container` wrapper outside one framed list, the
 * header strip from `@xl`, and every row on the SAME column table (`libraryListCols.ts`)
 * with the row's own padding — so the three tiers (three-line row < 576, table without
 * 規格, full table ≥ 672) swap at the same widths and loading → loaded does not jump.
 * Each row: thumb, title, meta, the 字幕 badge's pill, rating, 規格 (from `@2xl`).
 * The bars breathe (`animate-breathe` — a load really is in flight; reduced motion
 * stops it, via the token collapse in styles.css).
 */
export function LibraryListSkeletonV2({ count = 8 }: { count?: number }) {
  return (
    <div className="@container">
      <div
        data-testid="library-list-skeleton"
        aria-busy="true"
        aria-label="載入中"
        className="overflow-hidden rounded-[var(--radius-md)] border border-[var(--border-subtle)]"
      >
        <div
          aria-hidden="true"
          className="hidden h-9 items-center gap-4 border-b border-[var(--border-subtle)] px-3 @xl:flex"
        >
          <div className={`grid min-w-0 flex-1 ${LIBRARY_LIST_COLS} ${LIBRARY_LIST_GAP}`}>
            <span className={`${BAR} col-start-2 row-start-1 h-2.5 w-8`} />
            <span className={`${BAR} h-2.5 w-8 @xl:col-start-3 @xl:row-start-1`} />
            <span className={`${BAR} h-2.5 w-8 @xl:col-start-4 @xl:row-start-1`} />
            <span
              className={`${BAR} hidden h-2.5 w-8 @2xl:col-start-5 @2xl:row-start-1 @2xl:block`}
            />
          </div>
        </div>
        <ul aria-hidden="true" className="divide-y divide-[var(--border-subtle)]">
          {Array.from({ length: count }).map((_, i) => (
            // The row's box model (LibraryListRowV2): px-3 py-2.5, then the column grid.
            // Bars sit in line boxes of the real text's height, so a row is as tall as one.
            <li key={i} className="flex items-center gap-3 px-3 py-2.5 @xl:gap-4">
              <div
                className={`grid min-w-0 flex-1 items-center gap-y-0.5 motion-safe:animate-breathe ${LIBRARY_LIST_COLS} ${LIBRARY_LIST_GAP}`}
              >
                <span
                  className={`${FILL} col-start-1 row-span-2 row-start-1 h-[60px] w-10 rounded-[var(--radius-sm)] @xl:row-span-1 @xl:row-start-1 @xl:h-[72px] @xl:w-12 @xl:rounded-[var(--radius-md)]`}
                />
                <div className="col-start-2 row-start-1 flex min-w-0 flex-col gap-0.5 self-end @xl:gap-1 @xl:self-center">
                  <div className="flex h-5 items-center @xl:h-6">
                    <span className={`${BAR} h-3 ${TITLE_WIDTHS[i % TITLE_WIDTHS.length]}`} />
                  </div>
                  <div className="flex h-4 items-center @xl:h-5">
                    <span className={`${BAR} h-2.5 w-24`} />
                  </div>
                </div>
                <div className="col-start-2 row-start-2 flex min-w-0 items-center gap-2 self-start @xl:contents">
                  <span
                    className={`${FILL} h-6 w-14 shrink-0 rounded-full @xl:col-start-3 @xl:row-start-1`}
                  />
                  <span className={`${BAR} h-2.5 w-9 shrink-0 @xl:col-start-4 @xl:row-start-1`} />
                </div>
                <span
                  className={`${BAR} hidden h-2.5 w-32 @2xl:col-start-5 @2xl:row-start-1 @2xl:block`}
                />
              </div>
            </li>
          ))}
        </ul>
      </div>
    </div>
  );
}

/**
 * No-result — distinct from Empty; names what excluded everything.
 *
 * A7p-D says 「沒有電影符合目前的篩選條件（4K）。」 rather than a subjectless
 * 「試著調整或清除目前的篩選條件。」, and the difference matters: the generic line
 * sends you hunting for which filter did it, when the answer is already known here.
 * With nothing filtering there is no subject to name, so the generic line is the
 * honest one and stays as the fallback.
 */
export function LibraryNoResultV2({
  onClearFilters,
  mediaType = 'all',
  activeFilters = [],
}: {
  onClearFilters: () => void;
  mediaType?: LibraryMediaType;
  /** Human labels of the filters currently narrowing the query, in chip order. */
  activeFilters?: string[];
}) {
  const detail =
    activeFilters.length > 0
      ? `沒有${TYPE_NOUN[mediaType]}符合目前的篩選條件（${activeFilters.join('、')}）。試著調整或清除篩選。`
      : '試著調整或清除目前的篩選條件。';
  return (
    <div
      data-testid="library-no-result"
      className="flex flex-col items-center justify-center rounded-[var(--radius-lg)] bg-[var(--bg-secondary)] px-6 py-16 text-center"
    >
      <SearchX className="h-10 w-10 text-[var(--text-muted)]" aria-hidden="true" />
      <h2 className="mt-3 text-base font-semibold text-[var(--text-primary)]">找不到符合的結果</h2>
      <p className="mt-1 max-w-md text-sm text-[var(--text-secondary)]">{detail}</p>
      <button
        type="button"
        onClick={onClearFilters}
        data-testid="clear-all-filters"
        className="mt-4 min-h-[44px] rounded-[var(--radius-md)] bg-[var(--bg-tertiary)] px-4 text-sm font-medium text-[var(--text-primary)] transition-colors hover:bg-[var(--accent-subtle)]"
      >
        清除全部篩選
      </button>
    </div>
  );
}

/** Per-section fail-soft error — compact, carries a code, offers retry. */
export function LibraryErrorV2({ code, onRetry }: { code?: string; onRetry: () => void }) {
  return (
    <div
      data-testid="library-error"
      role="alert"
      className="flex flex-col items-center justify-center rounded-[var(--radius-lg)] bg-[var(--error-tint)] px-6 py-16 text-center"
    >
      <AlertTriangle className="h-10 w-10 text-[var(--error-text)]" aria-hidden="true" />
      <h2 className="mt-3 text-base font-semibold text-[var(--text-primary)]">無法載入媒體庫</h2>
      {/* A8p-D. The reassurance is the point: when the library will not load, the
          first thought is 「我的檔案還在嗎」, and a bare 「請稍後再試」 leaves it
          hanging. The query failed; the files on disk were never touched. */}
      {/* --text-secondary, NOT --error-text. A8p-D renders this line neutral, and
          for good reason: the sentence exists to REASSURE ("你的檔案沒有受影響"),
          and alarm-red works against the only thing it is there to do. The red
          belongs to the icon and the heading above it. */}
      <p className="mt-1 max-w-md text-sm text-[var(--text-secondary)]">
        媒體庫資料查詢失敗，你的檔案沒有受影響。
      </p>
      {/* A8p-D puts the code on its own line as a mono pill. Inline it produced
          「…沒有受影響。（DB_QUERY_FAILED）」 — a sentence-final 。 with a
          parenthetical hanging off it, which reads as an afterthought rather than
          the thing you paste into a bug report. */}
      {code ? (
        <span
          data-testid="library-error-code"
          className="mt-2 rounded-[var(--radius-sm)] bg-[var(--bg-tertiary)] px-2 py-0.5 font-mono text-xs text-[var(--text-muted)]"
        >
          {code}
        </span>
      ) : null}
      <button
        type="button"
        onClick={onRetry}
        data-testid="library-error-retry"
        className="mt-4 min-h-[44px] rounded-[var(--radius-md)] bg-[var(--bg-tertiary)] px-4 text-sm font-medium text-[var(--text-primary)] transition-colors hover:bg-[var(--bg-secondary)]"
      >
        重試
      </button>
    </div>
  );
}
