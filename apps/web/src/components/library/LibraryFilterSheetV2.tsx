// Design ref: ux-design.pen Screen A6p-M (Bz0YN)
/**
 * The phone (and tablet, <1024) sort + filter sheet (dsr-1b-b, redrawn from UX2-2).
 *
 * Three fixed bands, like `downloads/DownloadDetailSheet`: a header row (title + 重設),
 * a scrolling middle (sort chips + the same `FilterPanel` the desktop rail uses, in
 * instant mode so every tap lands in this sheet's draft), and a footer that stays put
 * with ONE primary action — 「套用篩選 · N 部」, where N is what the draft would return,
 * fetched before you commit (a 1-row list query; no facet endpoint).
 *
 * The draft lives here, not in the URL: Esc / scrim = discard, 套用 = write sort + filters
 * to the page in one go. Sort is the desktop `SORT_OPTIONS`, imported, never re-typed;
 * tapping the checked chip flips asc/desc (the desktop dropdown does the same on re-pick).
 * 全部/電影/影集 are hidden — on a phone the page title IS the media type.
 */
import { useEffect, useRef, useState, type ComponentProps, type KeyboardEvent } from 'react';
import { ArrowDown, ArrowUp } from 'lucide-react';
import { cn } from '../../lib/utils';
import { useLibraryList } from '../../hooks/useLibrary';
import type {
  LibraryListParams,
  LibraryMediaType,
  SortField,
  SortOrder,
} from '../../types/library';
import { Sheet } from '../ui/Sheet';
import { FilterPanel, type FilterValues } from './FilterPanel';
import { SORT_OPTIONS } from './SortSelector';
import { joinSubtitleStatusCsv } from './subtitleStatusFilter';

const DEFAULT_SORT: { sortBy: SortField; sortOrder: SortOrder } = {
  sortBy: 'created_at',
  sortOrder: 'desc',
};
const EMPTY_FILTERS: FilterValues = { genres: [] };
const COUNT_DEBOUNCE_MS = 200;

/**
 * The exact params behind the footer count. Exported so the gallery fixture can seed
 * `libraryKeys.list(sheetCountParams(...))` byte-for-byte — a key that differs in one
 * field misses the cache, hits the real backend and makes the baseline height flake.
 */
export function sheetCountParams(
  draft: FilterValues,
  mediaType: LibraryMediaType,
  sortBy: SortField,
  sortOrder: SortOrder
): LibraryListParams {
  return {
    page: 1,
    pageSize: 1,
    type: mediaType,
    sortBy,
    sortOrder,
    genres: draft.genres.length ? draft.genres.join(',') : undefined,
    yearMin: draft.yearMin,
    yearMax: draft.yearMax,
    unmatched: draft.unmatched || undefined,
    subtitleStatus: joinSubtitleStatusCsv(draft.subtitleStatus),
  };
}

interface LibraryFilterSheetV2Props {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  sortBy: SortField;
  sortOrder: SortOrder;
  onSortChange: (field: SortField, order: SortOrder) => void;
  filters: FilterValues;
  mediaType: LibraryMediaType;
  unmatchedCount?: number;
  onApply: (filters: FilterValues) => void;
  onClear: () => void;
  /** Kept for the desktop rail's shared prop shape; the phone sheet hides the type chips. */
  onTypeChange?: (type: LibraryMediaType) => void;
  /** Where focus lands once the sheet has closed — the button that opened it. */
  finalFocus?: ComponentProps<typeof Sheet>['finalFocus'];
}

export function LibraryFilterSheetV2({
  open,
  onOpenChange,
  sortBy,
  sortOrder,
  onSortChange,
  filters,
  mediaType,
  unmatchedCount,
  onApply,
  onClear,
  onTypeChange,
  finalFocus,
}: LibraryFilterSheetV2Props) {
  const [draft, setDraft] = useState<FilterValues>(filters);
  const [draftSort, setDraftSort] = useState({ sortBy, sortOrder });
  // Re-seed the draft from the page each time the sheet opens (during render, so the
  // first painted frame is already the committed state, never yesterday's draft).
  const [wasOpen, setWasOpen] = useState(open);
  if (wasOpen !== open) {
    setWasOpen(open);
    if (open) {
      setDraft(filters);
      setDraftSort({ sortBy, sortOrder });
    }
  }

  // The footer count: one 1-row list query per DISTINCT draft, 200ms after the last tap
  // (five quick genre taps = one request, not five), and the previous number stays on the
  // button while the next one loads (keepPreviousData) so the label never flickers.
  const countParams = sheetCountParams(draft, mediaType, draftSort.sortBy, draftSort.sortOrder);
  const [debouncedParams, setDebouncedParams] = useState(countParams);
  const countKey = JSON.stringify(countParams);
  const debouncedKey = JSON.stringify(debouncedParams);
  useEffect(() => {
    if (countKey === debouncedKey) return;
    const t = setTimeout(() => setDebouncedParams(countParams), COUNT_DEBOUNCE_MS);
    return () => clearTimeout(t);
    // countParams is a fresh object each render; countKey is its identity.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [countKey, debouncedKey]);
  const count = useLibraryList(debouncedParams, { enabled: open, keepPrevious: true });
  const total = count.data?.totalItems;
  const countStale = countKey !== debouncedKey || count.isPlaceholderData || count.isFetching;

  // Sort chips: roving tabindex + Left/Right/Up/Down wrap (the DownloadSortSheet pattern),
  // so the group is ONE Tab stop and arrows walk it.
  const sortRefs = useRef<(HTMLButtonElement | null)[]>([]);
  const onSortKeyDown = (e: KeyboardEvent<HTMLButtonElement>, at: number) => {
    const step =
      e.key === 'ArrowRight' || e.key === 'ArrowDown'
        ? 1
        : e.key === 'ArrowLeft' || e.key === 'ArrowUp'
          ? -1
          : 0;
    if (!step) return;
    e.preventDefault();
    const n = SORT_OPTIONS.length;
    sortRefs.current[(at + step + n) % n]?.focus();
  };

  const pickSort = (field: SortField) => {
    const option = SORT_OPTIONS.find((o) => o.field === field);
    if (!option) return;
    setDraftSort((cur) =>
      cur.sortBy === field
        ? { sortBy: field, sortOrder: cur.sortOrder === 'asc' ? 'desc' : 'asc' }
        : { sortBy: field, sortOrder: option.defaultOrder }
    );
  };

  const reset = () => {
    setDraft(EMPTY_FILTERS);
    setDraftSort(DEFAULT_SORT);
  };

  const apply = () => {
    if (draftSort.sortBy !== sortBy || draftSort.sortOrder !== sortOrder) {
      onSortChange(draftSort.sortBy, draftSort.sortOrder);
    }
    onApply(draft);
    onOpenChange(false);
  };

  return (
    <Sheet
      open={open}
      onOpenChange={onOpenChange}
      ariaLabel="排序與篩選"
      testId="library-sort-filter-sheet"
      className="flex flex-col overflow-hidden p-0 pt-2"
      finalFocus={finalFocus}
    >
      {/* Header row: the visible title (the sr-only Dialog.Title carries the name) + 重設 */}
      <div className="flex items-center justify-between pl-4 pr-2">
        <p aria-hidden="true" className="text-lg font-bold text-[var(--text-primary)]">
          排序與篩選
        </p>
        <button
          type="button"
          onClick={reset}
          data-testid="library-filter-reset"
          className="flex size-11 items-center justify-center rounded-[var(--radius-md)] text-sm font-semibold text-[var(--accent-text)]"
        >
          重設
        </button>
      </div>
      <div aria-hidden="true" className="h-px bg-[var(--border-subtle)]" />

      <div className="min-h-0 flex-1 overflow-y-auto px-4 py-4">
        <div className="mb-4">
          <h3
            id="library-sort-heading"
            className="mb-2 text-xs font-medium uppercase tracking-wide text-[var(--text-secondary)]"
          >
            排序
          </h3>
          <div role="radiogroup" aria-label="排序方式" className="flex flex-wrap gap-1.5">
            {SORT_OPTIONS.map((o, i) => {
              const checked = draftSort.sortBy === o.field;
              const Arrow = draftSort.sortOrder === 'asc' ? ArrowUp : ArrowDown;
              return (
                <button
                  key={o.field}
                  ref={(el) => {
                    sortRefs.current[i] = el;
                  }}
                  type="button"
                  role="radio"
                  aria-checked={checked}
                  tabIndex={checked ? 0 : -1}
                  data-testid={`library-sort-${o.field}`}
                  data-order={checked ? draftSort.sortOrder : undefined}
                  onClick={() => pickSort(o.field)}
                  onKeyDown={(e) => onSortKeyDown(e, i)}
                  className={cn(
                    'inline-flex min-h-11 items-center gap-1.5 rounded-[var(--radius-md)] px-3 text-sm transition-colors',
                    checked
                      ? 'bg-[var(--accent-subtle)] font-semibold text-[var(--accent-text)]'
                      : 'bg-[var(--bg-tertiary)] text-[var(--text-secondary)] hover:text-[var(--text-primary)]'
                  )}
                >
                  {checked && <Arrow className="size-3.5" aria-hidden="true" />}
                  {o.label}
                </button>
              );
            })}
          </div>
        </div>

        <FilterPanel
          filters={draft}
          mediaType={mediaType}
          unmatchedCount={unmatchedCount}
          onApply={setDraft}
          onClear={onClear}
          onTypeChange={onTypeChange ?? (() => undefined)}
          instant
          hideTypeChips
        />
      </div>

      <div className="shrink-0 border-t border-[var(--border-subtle)] px-4 pt-3 pb-[max(1.5rem,env(safe-area-inset-bottom))]">
        <button
          type="button"
          onClick={apply}
          data-testid="library-filter-apply"
          className="flex h-12 w-full items-center justify-center rounded-[var(--radius-md)] bg-[var(--accent-primary)] text-base font-semibold text-[var(--text-on-accent)]"
        >
          {typeof total === 'number' ? (
            <>
              套用篩選 ·{' '}
              <span
                data-testid="library-filter-apply-count"
                data-stale={countStale ? 'true' : undefined}
                className={cn('tabular-nums', countStale && 'opacity-70')}
              >
                {total.toLocaleString()}
              </span>{' '}
              部
            </>
          ) : (
            '套用篩選'
          )}
        </button>
      </div>
    </Sheet>
  );
}
