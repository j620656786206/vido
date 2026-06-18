// Design ref: ux-design.pen Screen 7 Search + Filter Desktop (rsAxf)
import { useState, useCallback, useEffect } from 'react';
import { Check, RotateCcw, TriangleAlert } from 'lucide-react';
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
  /**
   * ux3-0-7: instant mode (desktop rail) applies every control change immediately
   * (controlled off `filters`) and hides the 套用/重置 actions. Default false keeps
   * the batch behaviour the mobile bottom sheet relies on.
   */
  instant?: boolean;
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
  instant = false,
}: FilterPanelProps) {
  const [localGenres, setLocalGenres] = useState<string[]>(filters.genres);
  const [localDecades, setLocalDecades] = useState<string[]>(() =>
    getSelectedDecades(filters.yearMin, filters.yearMax)
  );
  const [localUnmatched, setLocalUnmatched] = useState(filters.unmatched ?? false);

  const {
    data: availableGenres = [],
    isLoading: genresLoading,
    isError: genresError,
    refetch: refetchGenres,
  } = useLibraryGenres();

  // Sync local state when external filters change. Instant mode (rail) reads `filters`
  // directly and never touches local state, so skip the dead updates there.
  useEffect(() => {
    if (instant) return;
    setLocalGenres(filters.genres);
    setLocalDecades(getSelectedDecades(filters.yearMin, filters.yearMax));
    setLocalUnmatched(filters.unmatched ?? false);
  }, [instant, filters.genres, filters.yearMin, filters.yearMax, filters.unmatched]);

  // Instant mode (desktop rail) is controlled off `filters`; batch mode off local state.
  const selectedGenres = instant ? filters.genres : localGenres;
  const selectedDecades = instant
    ? getSelectedDecades(filters.yearMin, filters.yearMax)
    : localDecades;
  const selectedUnmatched = instant ? (filters.unmatched ?? false) : localUnmatched;

  const emitInstant = useCallback(
    (next: { genres: string[]; decades: string[]; unmatched: boolean }) => {
      const yearRange = decadesToYearRange(next.decades);
      onApply({
        genres: next.genres,
        yearMin: yearRange.yearMin,
        yearMax: yearRange.yearMax,
        unmatched: next.unmatched || undefined,
      });
    },
    [onApply]
  );

  const handleGenreToggle = useCallback(
    (genre: string) => {
      const next = selectedGenres.includes(genre)
        ? selectedGenres.filter((g) => g !== genre)
        : [...selectedGenres, genre];
      if (instant)
        emitInstant({ genres: next, decades: selectedDecades, unmatched: selectedUnmatched });
      else setLocalGenres(next);
    },
    [instant, selectedGenres, selectedDecades, selectedUnmatched, emitInstant]
  );

  const handleDecadeToggle = useCallback(
    (decade: string) => {
      const toggled = selectedDecades.includes(decade)
        ? selectedDecades.filter((d) => d !== decade)
        : [...selectedDecades, decade];
      const next = normalizeDecadeSelection(toggled);
      if (instant)
        emitInstant({ genres: selectedGenres, decades: next, unmatched: selectedUnmatched });
      else setLocalDecades(next);
    },
    [instant, selectedGenres, selectedDecades, selectedUnmatched, emitInstant]
  );

  const handleUnmatchedToggle = useCallback(() => {
    const next = !selectedUnmatched;
    if (instant) emitInstant({ genres: selectedGenres, decades: selectedDecades, unmatched: next });
    else setLocalUnmatched(next);
  }, [instant, selectedGenres, selectedDecades, selectedUnmatched, emitInstant]);

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
      {!instant && <h3 className="mb-4 text-sm font-semibold text-white">篩選條件</h3>}

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
              className={`inline-flex min-h-[44px] items-center gap-1 rounded-full px-3 py-1.5 text-sm transition-colors ${
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

      {/* Genre Section — fail-soft: loading skeleton / error+retry / chips */}
      <div className="mb-4">
        <h4 className="mb-2 text-xs font-medium uppercase tracking-wide text-[var(--text-secondary)]">
          類別
        </h4>
        {genresLoading ? (
          <div
            className="flex flex-wrap gap-1.5"
            data-testid="filter-genre-loading"
            aria-busy="true"
          >
            {['w-16', 'w-12', 'w-20', 'w-14', 'w-16'].map((w, i) => (
              <span
                key={i}
                className={`h-7 animate-pulse rounded-full bg-[var(--bg-tertiary)] opacity-60 motion-reduce:animate-none ${w}`}
              />
            ))}
          </div>
        ) : genresError ? (
          <div
            data-testid="filter-genre-error"
            className="flex flex-col items-start gap-1.5 rounded-[var(--radius-md)] border border-[var(--error)] bg-[var(--error-tint)] p-3"
          >
            <span className="inline-flex items-center gap-1.5 text-sm text-[var(--error-text)]">
              <TriangleAlert className="h-3.5 w-3.5" aria-hidden="true" />
              類別載入失敗
            </span>
            <span className="text-xs text-[var(--text-secondary)]">其他篩選仍可使用</span>
            <button
              type="button"
              onClick={() => refetchGenres()}
              data-testid="filter-genre-retry"
              className="inline-flex min-h-[44px] items-center gap-1 text-sm font-medium text-[var(--accent-text)] hover:underline"
            >
              <RotateCcw className="h-3 w-3" aria-hidden="true" />
              重試
            </button>
          </div>
        ) : (
          <div className="flex flex-wrap gap-1.5">
            {availableGenres.map((genre) => (
              <button
                key={genre}
                onClick={() => handleGenreToggle(genre)}
                data-testid={`filter-genre-${genre}`}
                aria-pressed={selectedGenres.includes(genre)}
                className={`inline-flex min-h-[44px] items-center gap-1 rounded-full px-3 py-1.5 text-sm transition-colors ${
                  selectedGenres.includes(genre)
                    ? 'border border-[var(--accent-primary)] bg-[var(--accent-primary)]/15 text-blue-300'
                    : 'border border-transparent bg-[var(--bg-tertiary)] text-[var(--text-secondary)] hover:bg-[var(--bg-tertiary)]'
                }`}
              >
                {selectedGenres.includes(genre) && <Check className="h-3.5 w-3.5" />}
                {genre}
              </button>
            ))}
            {availableGenres.length === 0 && (
              <span className="text-sm text-[var(--text-muted)]">無可用類別</span>
            )}
          </div>
        )}
      </div>

      {/* Status Section */}
      <div className="mb-4">
        <h4 className="mb-2 text-xs font-medium uppercase tracking-wide text-[var(--text-secondary)]">
          狀態
        </h4>
        <div className="flex flex-wrap gap-1.5">
          <button
            onClick={handleUnmatchedToggle}
            data-testid="filter-unmatched"
            aria-pressed={selectedUnmatched}
            className={`inline-flex min-h-[44px] items-center gap-1 rounded-full px-3 py-1.5 text-sm transition-colors ${
              selectedUnmatched
                ? 'border border-[var(--accent-primary)] bg-[var(--accent-primary)]/15 text-blue-300'
                : 'border border-transparent bg-[var(--bg-tertiary)] text-[var(--text-secondary)] hover:bg-[var(--bg-tertiary)]'
            }`}
          >
            {selectedUnmatched && <Check className="h-3.5 w-3.5" />}
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
              aria-pressed={selectedDecades.includes(decade.label)}
              className={`inline-flex min-h-[44px] items-center gap-1 rounded-full px-3 py-1.5 text-sm transition-colors ${
                selectedDecades.includes(decade.label)
                  ? 'border border-[var(--accent-primary)] bg-[var(--accent-primary)]/15 text-blue-300'
                  : 'border border-transparent bg-[var(--bg-tertiary)] text-[var(--text-secondary)] hover:bg-[var(--bg-tertiary)]'
              }`}
            >
              {selectedDecades.includes(decade.label) && <Check className="h-3.5 w-3.5" />}
              {decade.label}
            </button>
          ))}
        </div>
      </div>

      {/* Actions — batch mode only; instant mode (rail) applies on change */}
      {!instant && (
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
      )}
    </div>
  );
}
