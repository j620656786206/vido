// Design ref: ux-design.pen Screen A4p-D (b1H71g) · A4p-M (EkB8g)
/**
 * The library list view as a 修字幕清單 table (disc-2026-10-library-list-triage-b1).
 *
 * ONE frame around the whole list (1px `--border-subtle`, `--radius-md`, no fill) and
 * a hairline between rows; the rows themselves carry no frame and no fill (⚖️ Sally 1,
 * DESIGN.md「同一層級只准一道框」). The frame clips (`overflow-hidden`) so a hovered
 * row's tint stays inside the rounded corners — which is why each row draws its focus
 * ring inset.
 *
 * `@container` sits on a wrapper OUTSIDE the frame, so the breakpoints in
 * `libraryListCols.ts` are measured against the list's full width (the frame's 1px
 * border would otherwise shift each threshold by 2px).
 *
 * The header row (片名／字幕／評分／規格) exists from `@xl` up and is hidden from
 * assistive tech: every row's link already speaks its own cells by name (AC #9).
 */
import type React from 'react';
import { LibraryListRowV2, type LibraryListRowData } from './LibraryListRowV2';
import { LIBRARY_LIST_COLS, LIBRARY_LIST_GAP } from './libraryListCols';

interface LibraryListV2Props {
  rows: LibraryListRowData[];
  /** Selection mode (ux3-cutover-2): rows toggle instead of navigating. */
  selectable?: boolean;
  selectedIds?: ReadonlySet<string>;
  onSelect?: (id: string, e: React.MouseEvent) => void;
}

const HEAD = 'truncate text-xs font-semibold text-[var(--text-muted)]';

export function LibraryListV2({ rows, selectable, selectedIds, onSelect }: LibraryListV2Props) {
  return (
    <div className="@container">
      <div
        data-testid="library-list-v2"
        className="overflow-hidden rounded-[var(--radius-md)] border border-[var(--border-subtle)]"
      >
        <div
          data-testid="library-list-header"
          aria-hidden="true"
          className="hidden h-9 items-center gap-4 border-b border-[var(--border-subtle)] px-3 @xl:flex"
        >
          {/* The selection indicator's column, left blank on the header. */}
          {selectable && <span className="w-5 shrink-0" />}
          <div className={`grid min-w-0 flex-1 ${LIBRARY_LIST_COLS} ${LIBRARY_LIST_GAP}`}>
            <span className={`${HEAD} col-start-2 row-start-1`}>片名</span>
            <span className={`${HEAD} @xl:col-start-3 @xl:row-start-1`}>字幕</span>
            <span className={`${HEAD} @xl:col-start-4 @xl:row-start-1`}>評分</span>
            <span
              data-testid="library-list-header-spec"
              className={`${HEAD} hidden @2xl:col-start-5 @2xl:row-start-1 @2xl:block`}
            >
              規格
            </span>
          </div>
        </div>
        {/* The frame clips to its rounded corners, so the rows at its corners round their
            own edge to match — the inset focus ring then follows the curve instead of
            being cut. The first row only meets the frame's top where there is no header. */}
        <ul className="divide-y divide-[var(--border-subtle)] [&>li:first-child>a]:rounded-t-[calc(var(--radius-md)-1px)] [&>li:last-child>a]:rounded-b-[calc(var(--radius-md)-1px)] @xl:[&>li:first-child>a]:rounded-t-none">
          {rows.map((row) => (
            <LibraryListRowV2
              key={`${row.type}-${row.id}`}
              {...row}
              selectable={selectable}
              selected={!!selectable && !!selectedIds?.has(row.id)}
              onSelect={selectable && onSelect ? (e) => onSelect(row.id, e) : undefined}
            />
          ))}
        </ul>
      </div>
    </div>
  );
}
