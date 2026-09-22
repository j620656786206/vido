// Implements: Component/FilterChip (jD7gF)
// Source: ux-design.pen (Pencil app)
import { X } from 'lucide-react';
import { cn } from '../../lib/utils';
import { yearFilterLabel, type FilterValues } from './FilterPanel';
import { subtitleStatusLabel } from './subtitleStatusFilter';

interface FilterChipsProps {
  filters: FilterValues;
  onRemoveGenre: (genre: string) => void;
  onRemoveYearMin: () => void;
  onRemoveYearMax: () => void;
  /**
   * ux3-0-7: atomic removal of a full decade range (both bounds set). Optional —
   * falls back to calling onRemoveYearMin + onRemoveYearMax. Prefer wiring this so a
   * single navigate clears both bounds (no two-step race).
   */
  onRemoveYears?: () => void;
  onRemoveUnmatched: () => void;
  /** dsr-1b-b: remove ONE subtitle status value (the chip row shows one chip per value). */
  onRemoveSubtitleStatus?: (value: string) => void;
  onClearAll: () => void;
  /** Merged into the row — the page hands in its phone single-row scroller classes (dsr-1b-b). */
  className?: string;
}

export function FilterChips({
  filters,
  onRemoveGenre,
  onRemoveYearMin,
  onRemoveYearMax,
  onRemoveYears,
  onRemoveUnmatched,
  onRemoveSubtitleStatus,
  onClearAll,
  className,
}: FilterChipsProps) {
  const subtitleStatuses = filters.subtitleStatus ?? [];
  // A full decade range (both bounds) is ONE facet — render it as a single chip so the
  // chip row matches the rail's active-count badge (decade-as-one). Half-open ranges
  // (only one bound) keep their individual chip.
  const hasYearRange = filters.yearMin !== undefined && filters.yearMax !== undefined;
  const removeYearRange =
    onRemoveYears ??
    (() => {
      onRemoveYearMin();
      onRemoveYearMax();
    });
  const hasFilters =
    filters.genres.length > 0 ||
    filters.yearMin !== undefined ||
    filters.yearMax !== undefined ||
    filters.unmatched === true ||
    subtitleStatuses.length > 0;

  if (!hasFilters) return null;

  return (
    <div className={cn('flex flex-wrap items-center gap-2', className)}>
      {filters.genres.map((genre) => (
        <span
          key={genre}
          className="inline-flex items-center gap-1 rounded-full bg-[var(--accent-primary)]/20 px-3 py-1 text-sm text-[var(--accent-text)] max-sm:shrink-0"
        >
          {genre}
          <button
            onClick={() => onRemoveGenre(genre)}
            className="ml-0.5 rounded-full p-0.5 hover:bg-[var(--accent-primary)]/30"
            aria-label={`移除 ${genre} 篩選`}
          >
            <X className="h-3 w-3" />
          </button>
        </span>
      ))}

      {hasYearRange ? (
        <span className="inline-flex items-center gap-1 rounded-full bg-[var(--accent-primary)]/20 px-3 py-1 text-sm text-[var(--accent-text)] max-sm:shrink-0">
          {yearFilterLabel(filters)}
          <button
            onClick={removeYearRange}
            className="ml-0.5 rounded-full p-0.5 hover:bg-[var(--accent-primary)]/30"
            aria-label="移除年份篩選"
          >
            <X className="h-3 w-3" />
          </button>
        </span>
      ) : (
        <>
          {filters.yearMin !== undefined && (
            <span className="inline-flex items-center gap-1 rounded-full bg-[var(--accent-primary)]/20 px-3 py-1 text-sm text-[var(--accent-text)] max-sm:shrink-0">
              {yearFilterLabel({ yearMin: filters.yearMin })}
              <button
                onClick={onRemoveYearMin}
                className="ml-0.5 rounded-full p-0.5 hover:bg-[var(--accent-primary)]/30"
                aria-label="移除最早年份篩選"
              >
                <X className="h-3 w-3" />
              </button>
            </span>
          )}

          {filters.yearMax !== undefined && (
            <span className="inline-flex items-center gap-1 rounded-full bg-[var(--accent-primary)]/20 px-3 py-1 text-sm text-[var(--accent-text)] max-sm:shrink-0">
              {yearFilterLabel({ yearMax: filters.yearMax })}
              <button
                onClick={onRemoveYearMax}
                className="ml-0.5 rounded-full p-0.5 hover:bg-[var(--accent-primary)]/30"
                aria-label="移除最晚年份篩選"
              >
                <X className="h-3 w-3" />
              </button>
            </span>
          )}
        </>
      )}

      {filters.unmatched && (
        <span className="inline-flex items-center gap-1 rounded-full bg-[var(--accent-primary)]/20 px-3 py-1 text-sm text-[var(--accent-text)] max-sm:shrink-0">
          未匹配
          <button
            onClick={onRemoveUnmatched}
            className="ml-0.5 rounded-full p-0.5 hover:bg-[var(--accent-primary)]/30"
            aria-label="移除未匹配篩選"
          >
            <X className="h-3 w-3" />
          </button>
        </span>
      )}

      {subtitleStatuses.map((value) => (
        <span
          key={`subtitle-${value}`}
          className="inline-flex items-center gap-1 rounded-full bg-[var(--accent-primary)]/20 px-3 py-1 text-sm text-[var(--accent-text)] max-sm:shrink-0"
        >
          {subtitleStatusLabel(value)}
          <button
            onClick={() => onRemoveSubtitleStatus?.(value)}
            className="ml-0.5 rounded-full p-0.5 hover:bg-[var(--accent-primary)]/30"
            aria-label={`移除${subtitleStatusLabel(value)}篩選`}
          >
            <X className="h-3 w-3" />
          </button>
        </span>
      ))}

      <button
        onClick={onClearAll}
        className="text-sm text-[var(--text-secondary)] underline-offset-2 hover:text-[var(--text-primary)] hover:underline max-sm:shrink-0"
      >
        清除全部篩選
      </button>
    </div>
  );
}
