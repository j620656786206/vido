// Design ref: ux-design.pen Screen AS-1 Advanced Filter Chips Desktop (rsAxf)
// Source: ux-design.pen (Pencil app)
import { useEffect, useRef, useState } from 'react';
import { Check } from 'lucide-react';
import { cn } from '../../lib/utils';
import {
  GENRE_FILTER_OPTIONS,
  REGION_OPTIONS,
  PLATFORM_OPTIONS,
  RATING_OPTIONS,
  SORT_OPTIONS,
  type DiscoverFilters,
  type SortKey,
} from '../../lib/discoverFilters';

interface FilterPanelProps {
  filters: DiscoverFilters;
  /** Controlled change handler. Desktop wires this to instant URL updates (AC #5). */
  onChange: (next: DiscoverFilters) => void;
  className?: string;
  /**
   * ux3-3-2 AC #4: debounce (ms) for the NUMERIC year inputs only — categorical
   * chips (genre/region/rating/platform/sort) always stay instant. `0` (default)
   * commits every keystroke, preserving the legacy/per-keystroke behavior; the v2
   * rail passes a positive value so typing "1995" fires ONE query, not four.
   */
  debounceMs?: number;
}

const chipClass = (active: boolean) =>
  cn(
    'inline-flex items-center gap-1 rounded-full px-3 py-1.5 text-sm transition-colors',
    active
      ? 'border border-[var(--accent-primary)] bg-[var(--accent-primary)]/15 text-blue-300'
      : 'border border-transparent bg-[var(--bg-tertiary)] text-[var(--text-secondary)] hover:text-white'
  );

const sectionLabelClass =
  'mb-2 text-xs font-medium uppercase tracking-wide text-[var(--text-secondary)]';

/**
 * Controlled filter panel — renders genre / region / year-range / rating /
 * platform / sort controls (Task 3). Fully controlled so the desktop sidebar can
 * apply changes instantly to the URL and the mobile bottom sheet can drive a
 * local draft before committing.
 */
export function FilterPanel({ filters, onChange, className, debounceMs = 0 }: FilterPanelProps) {
  // Latest filters in a ref so a debounced year commit composes against fresh
  // categorical state instead of a stale closure. Synced in an effect (never
  // mutate a ref during render).
  const filtersRef = useRef(filters);
  useEffect(() => {
    filtersRef.current = filters;
  }, [filters]);

  const toggleGenre = (id: number) => {
    onChange({
      ...filters,
      genre: filters.genre.includes(id)
        ? filters.genre.filter((g) => g !== id)
        : [...filters.genre, id],
    });
  };

  const selectRegion = (code: string) => {
    onChange({ ...filters, region: filters.region === code ? undefined : code });
  };

  // ux3-3-2 AC #4: the year inputs render from local state (so typing is
  // immediately responsive) but COMMIT to onChange on a debounce. Both year
  // bounds are committed together from the latest input refs so editing one
  // bound never drops a pending edit to the other.
  const [yearGteInput, setYearGteInput] = useState<string>(filters.yearGte?.toString() ?? '');
  const [yearLteInput, setYearLteInput] = useState<string>(filters.yearLte?.toString() ?? '');
  const inputsRef = useRef({ gte: yearGteInput, lte: yearLteInput });
  const yearTimer = useRef<ReturnType<typeof setTimeout> | null>(null);

  // Resync local inputs when the committed year bounds change externally (chip
  // removed, 清除全部, preset applied) — keep state AND ref in lockstep.
  useEffect(() => {
    const s = filters.yearGte?.toString() ?? '';
    setYearGteInput(s);
    inputsRef.current.gte = s;
  }, [filters.yearGte]);
  useEffect(() => {
    const s = filters.yearLte?.toString() ?? '';
    setYearLteInput(s);
    inputsRef.current.lte = s;
  }, [filters.yearLte]);

  // Cancel a pending debounce on unmount.
  useEffect(
    () => () => {
      if (yearTimer.current) clearTimeout(yearTimer.current);
    },
    []
  );

  const commitYears = () => {
    const parse = (raw: string): number | undefined => {
      if (raw.trim() === '') return undefined;
      const n = parseInt(raw, 10);
      return Number.isFinite(n) ? n : undefined;
    };
    onChange({
      ...filtersRef.current,
      yearGte: parse(inputsRef.current.gte),
      yearLte: parse(inputsRef.current.lte),
    });
  };

  const setYear = (key: 'yearGte' | 'yearLte', raw: string) => {
    if (key === 'yearGte') {
      setYearGteInput(raw);
      inputsRef.current.gte = raw;
    } else {
      setYearLteInput(raw);
      inputsRef.current.lte = raw;
    }
    if (debounceMs > 0) {
      if (yearTimer.current) clearTimeout(yearTimer.current);
      yearTimer.current = setTimeout(commitYears, debounceMs);
    } else {
      commitYears();
    }
  };

  const selectRating = (value: number) => {
    onChange({ ...filters, ratingGte: filters.ratingGte === value ? undefined : value });
  };

  const togglePlatform = (id: number) => {
    onChange({
      ...filters,
      platform: filters.platform.includes(id)
        ? filters.platform.filter((p) => p !== id)
        : [...filters.platform, id],
    });
  };

  return (
    <div className={cn('flex flex-col gap-5', className)} data-testid="filter-panel">
      {/* Genre (類型) — Task 3.2 */}
      <section>
        <h4 className={sectionLabelClass}>類型</h4>
        <div className="flex flex-wrap gap-1.5">
          {GENRE_FILTER_OPTIONS.map((genre) => {
            const active = filters.genre.includes(genre.id);
            return (
              <button
                key={genre.id}
                onClick={() => toggleGenre(genre.id)}
                data-testid={`filter-genre-${genre.id}`}
                aria-pressed={active}
                className={chipClass(active)}
              >
                {active && <Check className="h-3.5 w-3.5" />}
                {genre.label}
              </button>
            );
          })}
        </div>
      </section>

      {/* Region (地區) — Task 3.4 */}
      <section>
        <h4 className={sectionLabelClass}>地區</h4>
        <div className="flex flex-wrap gap-1.5">
          {REGION_OPTIONS.map((region) => {
            const active = filters.region === region.code;
            return (
              <button
                key={region.code}
                onClick={() => selectRegion(region.code)}
                data-testid={`filter-region-${region.code}`}
                aria-pressed={active}
                className={chipClass(active)}
              >
                <span aria-hidden="true">{region.flag}</span>
                {region.label}
              </button>
            );
          })}
        </div>
      </section>

      {/* Year range (年份範圍) — Task 3.3 */}
      <section>
        <h4 className={sectionLabelClass}>年份範圍</h4>
        <div className="flex items-center gap-2">
          <input
            type="number"
            inputMode="numeric"
            value={yearGteInput}
            onChange={(e) => setYear('yearGte', e.target.value)}
            placeholder="不限"
            aria-label="最早年份"
            data-testid="filter-year-gte"
            className="w-24 rounded-md border border-[var(--border-subtle)] bg-[var(--bg-tertiary)] px-2 py-1.5 text-sm text-white placeholder:text-[var(--text-muted)] focus:border-[var(--accent-primary)] focus:outline-none"
          />
          <span className="text-[var(--text-secondary)]">—</span>
          <input
            type="number"
            inputMode="numeric"
            value={yearLteInput}
            onChange={(e) => setYear('yearLte', e.target.value)}
            placeholder="不限"
            aria-label="最晚年份"
            data-testid="filter-year-lte"
            className="w-24 rounded-md border border-[var(--border-subtle)] bg-[var(--bg-tertiary)] px-2 py-1.5 text-sm text-white placeholder:text-[var(--text-muted)] focus:border-[var(--accent-primary)] focus:outline-none"
          />
        </div>
      </section>

      {/* Minimum rating (最低評分) — Task 3.5 */}
      <section>
        <h4 className={sectionLabelClass}>最低評分</h4>
        <div className="flex flex-wrap gap-1.5">
          {RATING_OPTIONS.map((value) => {
            const active = filters.ratingGte === value;
            return (
              <button
                key={value}
                onClick={() => selectRating(value)}
                data-testid={`filter-rating-${value}`}
                aria-pressed={active}
                className={chipClass(active)}
              >
                ★{value}+
              </button>
            );
          })}
        </div>
      </section>

      {/* Platform (平台) — Task 3.6 */}
      <section>
        <h4 className={sectionLabelClass}>平台</h4>
        <div className="flex flex-wrap gap-1.5">
          {PLATFORM_OPTIONS.map((platform) => {
            const active = filters.platform.includes(platform.id);
            return (
              <button
                key={platform.id}
                onClick={() => togglePlatform(platform.id)}
                data-testid={`filter-platform-${platform.id}`}
                aria-pressed={active}
                className={chipClass(active)}
              >
                {active && <Check className="h-3.5 w-3.5" />}
                {platform.label}
              </button>
            );
          })}
        </div>
      </section>

      {/* Sort (排序方式) — Task 3.7 */}
      <section>
        <h4 className={sectionLabelClass}>排序方式</h4>
        <select
          value={filters.sortBy}
          onChange={(e) => onChange({ ...filters, sortBy: e.target.value as SortKey })}
          data-testid="filter-sort"
          aria-label="排序方式"
          className="w-full rounded-md border border-[var(--border-subtle)] bg-[var(--bg-tertiary)] px-2 py-1.5 text-sm text-white focus:border-[var(--accent-primary)] focus:outline-none"
        >
          {SORT_OPTIONS.map((option) => (
            <option key={option.value} value={option.value}>
              {option.label}
            </option>
          ))}
        </select>
      </section>
    </div>
  );
}
