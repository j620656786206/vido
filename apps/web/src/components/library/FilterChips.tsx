import { X } from 'lucide-react';
import type { FilterValues } from './FilterPanel';

interface FilterChipsProps {
  filters: FilterValues;
  onRemoveGenre: (genre: string) => void;
  onRemoveYearMin: () => void;
  onRemoveYearMax: () => void;
  onClearAll: () => void;
}

export function FilterChips({
  filters,
  onRemoveGenre,
  onRemoveYearMin,
  onRemoveYearMax,
  onClearAll,
}: FilterChipsProps) {
  const hasFilters =
    filters.genres.length > 0 || filters.yearMin !== undefined || filters.yearMax !== undefined;

  if (!hasFilters) return null;

  return (
    <div className="flex flex-wrap items-center gap-2">
      {filters.genres.map((genre) => (
        <span
          key={genre}
          className="inline-flex items-center gap-1 rounded-full bg-[var(--accent-primary)]/20 px-3 py-1 text-sm text-blue-300"
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

      {filters.yearMin !== undefined && (
        <span className="inline-flex items-center gap-1 rounded-full bg-[var(--accent-primary)]/20 px-3 py-1 text-sm text-blue-300">
          {filters.yearMin} 年起
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
        <span className="inline-flex items-center gap-1 rounded-full bg-[var(--accent-primary)]/20 px-3 py-1 text-sm text-blue-300">
          至 {filters.yearMax} 年
          <button
            onClick={onRemoveYearMax}
            className="ml-0.5 rounded-full p-0.5 hover:bg-[var(--accent-primary)]/30"
            aria-label="移除最晚年份篩選"
          >
            <X className="h-3 w-3" />
          </button>
        </span>
      )}

      <button
        onClick={onClearAll}
        className="text-sm text-[var(--text-secondary)] underline-offset-2 hover:text-white hover:underline"
      >
        清除全部篩選
      </button>
    </div>
  );
}
