import { useState, useCallback, useEffect } from 'react';
import { Check } from 'lucide-react';
import { useLibraryGenres } from '../../hooks/useLibrary';
import type { LibraryMediaType } from '../../types/library';

export interface FilterValues {
  genres: string[];
  yearMin?: number;
  yearMax?: number;
  unmatched?: boolean;
}

const DECADE_OPTIONS = [
  { label: '2020s', min: 2020, max: 2029 },
  { label: '2010s', min: 2010, max: 2019 },
  { label: '2000s', min: 2000, max: 2009 },
  { label: '1990s', min: 1990, max: 1999 },
  { label: '更早', min: 0, max: 1989 },
];

interface FilterPanelProps {
  filters: FilterValues;
  mediaType: LibraryMediaType;
  unmatchedCount?: number;
  onApply: (filters: FilterValues) => void;
  onClear: () => void;
  onTypeChange: (type: LibraryMediaType) => void;
}

function getSelectedDecades(yearMin?: number, yearMax?: number): string[] {
  if (yearMin === undefined && yearMax === undefined) return [];
  return DECADE_OPTIONS.filter((d) => {
    if (yearMin !== undefined && yearMax !== undefined) {
      return d.max >= yearMin && d.min <= yearMax;
    }
    if (yearMin !== undefined) return d.max >= yearMin;
    if (yearMax !== undefined) return d.min <= yearMax;
    return false;
  }).map((d) => d.label);
}

function decadesToYearRange(selectedDecades: string[]): { yearMin?: number; yearMax?: number } {
  if (selectedDecades.length === 0) return {};
  const selected = DECADE_OPTIONS.filter((d) => selectedDecades.includes(d.label));
  const yearMin = Math.min(...selected.map((d) => d.min));
  const yearMax = Math.max(...selected.map((d) => d.max));
  return {
    yearMin: yearMin === 0 ? undefined : yearMin,
    yearMax,
  };
}

// Auto-fill gaps between non-contiguous decade selections since backend only supports min/max range.
// e.g., selecting 2020s + 2000s auto-selects 2010s to match actual query behavior.
function normalizeDecadeSelection(decades: string[]): string[] {
  if (decades.length <= 1) return decades;
  const indices = DECADE_OPTIONS.map((d, i) => (decades.includes(d.label) ? i : -1)).filter(
    (i) => i >= 0
  );
  const minIdx = Math.min(...indices);
  const maxIdx = Math.max(...indices);
  return DECADE_OPTIONS.slice(minIdx, maxIdx + 1).map((d) => d.label);
}

export function FilterPanel({
  filters,
  mediaType,
  unmatchedCount,
  onApply,
  onClear,
  onTypeChange,
}: FilterPanelProps) {
  const [localGenres, setLocalGenres] = useState<string[]>(filters.genres);
  const [localDecades, setLocalDecades] = useState<string[]>(() =>
    getSelectedDecades(filters.yearMin, filters.yearMax)
  );
  const [localUnmatched, setLocalUnmatched] = useState(filters.unmatched ?? false);

  const { data: availableGenres = [] } = useLibraryGenres();

  // Sync local state when external filters change
  useEffect(() => {
    setLocalGenres(filters.genres);
    setLocalDecades(getSelectedDecades(filters.yearMin, filters.yearMax));
    setLocalUnmatched(filters.unmatched ?? false);
  }, [filters.genres, filters.yearMin, filters.yearMax, filters.unmatched]);

  const handleGenreToggle = useCallback((genre: string) => {
    setLocalGenres((prev) =>
      prev.includes(genre) ? prev.filter((g) => g !== genre) : [...prev, genre]
    );
  }, []);

  const handleDecadeToggle = useCallback((decade: string) => {
    setLocalDecades((prev) => {
      const toggled = prev.includes(decade) ? prev.filter((d) => d !== decade) : [...prev, decade];
      return normalizeDecadeSelection(toggled);
    });
  }, []);

  const handleApply = useCallback(() => {
    const yearRange = decadesToYearRange(localDecades);
    onApply({
      genres: localGenres,
      yearMin: yearRange.yearMin,
      yearMax: yearRange.yearMax,
      unmatched: localUnmatched || undefined,
    });
  }, [localGenres, localDecades, localUnmatched, onApply]);

  const handleClear = useCallback(() => {
    setLocalGenres([]);
    setLocalDecades([]);
    setLocalUnmatched(false);
    onClear();
  }, [onClear]);

  return (
    <div className="flex h-full flex-col" data-testid="filter-panel">
      <h3 className="mb-4 text-sm font-semibold text-white">篩選條件</h3>

      {/* Type Section */}
      <div className="mb-4">
        <h4 className="mb-2 text-xs font-medium uppercase tracking-wide text-[var(--text-secondary)]">
          類型
        </h4>
        <div className="flex flex-wrap gap-1.5">
          {(['all', 'movie', 'tv'] as const).map((t) => (
            <button
              key={t}
              onClick={() => onTypeChange(t)}
              data-testid={`filter-type-${t}`}
              aria-pressed={mediaType === t}
              className={`inline-flex items-center gap-1 rounded-full px-3 py-1.5 text-sm transition-colors ${
                mediaType === t
                  ? 'border border-[var(--accent-primary)] bg-[var(--accent-primary)]/15 text-blue-300'
                  : 'border border-transparent bg-[var(--bg-tertiary)] text-[var(--text-secondary)] hover:bg-[var(--bg-tertiary)]'
              }`}
            >
              {mediaType === t && <Check className="h-3.5 w-3.5" />}
              {t === 'all' ? '全部' : t === 'movie' ? '電影' : '影集'}
            </button>
          ))}
        </div>
      </div>

      {/* Genre Section */}
      <div className="mb-4">
        <h4 className="mb-2 text-xs font-medium uppercase tracking-wide text-[var(--text-secondary)]">
          類別
        </h4>
        <div className="flex flex-wrap gap-1.5">
          {availableGenres.map((genre) => (
            <button
              key={genre}
              onClick={() => handleGenreToggle(genre)}
              data-testid={`filter-genre-${genre}`}
              aria-pressed={localGenres.includes(genre)}
              className={`inline-flex items-center gap-1 rounded-full px-3 py-1.5 text-sm transition-colors ${
                localGenres.includes(genre)
                  ? 'border border-[var(--accent-primary)] bg-[var(--accent-primary)]/15 text-blue-300'
                  : 'border border-transparent bg-[var(--bg-tertiary)] text-[var(--text-secondary)] hover:bg-[var(--bg-tertiary)]'
              }`}
            >
              {localGenres.includes(genre) && <Check className="h-3.5 w-3.5" />}
              {genre}
            </button>
          ))}
          {availableGenres.length === 0 && (
            <span className="text-sm text-[var(--text-muted)]">無可用類別</span>
          )}
        </div>
      </div>

      {/* Status Section */}
      <div className="mb-4">
        <h4 className="mb-2 text-xs font-medium uppercase tracking-wide text-[var(--text-secondary)]">
          狀態
        </h4>
        <div className="flex flex-wrap gap-1.5">
          <button
            onClick={() => setLocalUnmatched(!localUnmatched)}
            data-testid="filter-unmatched"
            aria-pressed={localUnmatched}
            className={`inline-flex items-center gap-1 rounded-full px-3 py-1.5 text-sm transition-colors ${
              localUnmatched
                ? 'border border-[var(--accent-primary)] bg-[var(--accent-primary)]/15 text-blue-300'
                : 'border border-transparent bg-[var(--bg-tertiary)] text-[var(--text-secondary)] hover:bg-[var(--bg-tertiary)]'
            }`}
          >
            {localUnmatched && <Check className="h-3.5 w-3.5" />}
            未匹配{unmatchedCount != null ? ` (${unmatchedCount})` : ''}
          </button>
        </div>
      </div>

      {/* Year Section — Decade Chips */}
      <div className="mb-4">
        <h4 className="mb-2 text-xs font-medium uppercase tracking-wide text-[var(--text-secondary)]">
          年份
        </h4>
        <div className="flex flex-wrap gap-1.5">
          {DECADE_OPTIONS.map((decade) => (
            <button
              key={decade.label}
              onClick={() => handleDecadeToggle(decade.label)}
              data-testid={`filter-decade-${decade.label}`}
              aria-pressed={localDecades.includes(decade.label)}
              className={`inline-flex items-center gap-1 rounded-full px-3 py-1.5 text-sm transition-colors ${
                localDecades.includes(decade.label)
                  ? 'border border-[var(--accent-primary)] bg-[var(--accent-primary)]/15 text-blue-300'
                  : 'border border-transparent bg-[var(--bg-tertiary)] text-[var(--text-secondary)] hover:bg-[var(--bg-tertiary)]'
              }`}
            >
              {localDecades.includes(decade.label) && <Check className="h-3.5 w-3.5" />}
              {decade.label}
            </button>
          ))}
        </div>
      </div>

      {/* Actions */}
      <div className="mt-auto flex gap-2">
        <button
          onClick={handleApply}
          data-testid="filter-apply"
          className="flex-1 rounded-md bg-[var(--accent-primary)] px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-[var(--accent-hover)]"
        >
          套用
        </button>
        <button
          onClick={handleClear}
          data-testid="filter-reset"
          className="flex-1 rounded-md border border-[var(--border-subtle)] px-4 py-2 text-sm text-[var(--text-secondary)] transition-colors hover:bg-[var(--bg-tertiary)]"
        >
          重置
        </button>
      </div>
    </div>
  );
}
