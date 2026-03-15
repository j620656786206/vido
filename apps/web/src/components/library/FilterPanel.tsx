import { useState, useCallback } from 'react';
import { useLibraryGenres, useLibraryStats } from '../../hooks/useLibrary';

export interface FilterValues {
  genres: string[];
  yearMin?: number;
  yearMax?: number;
}

interface FilterPanelProps {
  filters: FilterValues;
  onApply: (filters: FilterValues) => void;
  onClear: () => void;
}

export function FilterPanel({ filters, onApply, onClear }: FilterPanelProps) {
  const [isOpen, setIsOpen] = useState(false);
  const [localGenres, setLocalGenres] = useState<string[]>(filters.genres);
  const [localYearMin, setLocalYearMin] = useState<string>(
    filters.yearMin ? String(filters.yearMin) : ''
  );
  const [localYearMax, setLocalYearMax] = useState<string>(
    filters.yearMax ? String(filters.yearMax) : ''
  );

  const { data: availableGenres = [] } = useLibraryGenres();
  const { data: stats } = useLibraryStats();

  const handleGenreToggle = useCallback((genre: string) => {
    setLocalGenres((prev) =>
      prev.includes(genre) ? prev.filter((g) => g !== genre) : [...prev, genre]
    );
  }, []);

  const handleApply = useCallback(() => {
    onApply({
      genres: localGenres,
      yearMin: localYearMin ? parseInt(localYearMin, 10) : undefined,
      yearMax: localYearMax ? parseInt(localYearMax, 10) : undefined,
    });
  }, [localGenres, localYearMin, localYearMax, onApply]);

  const handleClear = useCallback(() => {
    setLocalGenres([]);
    setLocalYearMin('');
    setLocalYearMax('');
    onClear();
  }, [onClear]);

  const hasActiveFilters =
    filters.genres.length > 0 || filters.yearMin !== undefined || filters.yearMax !== undefined;

  return (
    <div>
      <button
        onClick={() => setIsOpen(!isOpen)}
        className={`flex items-center gap-2 rounded-lg px-4 py-2 text-sm font-medium transition-colors ${
          hasActiveFilters
            ? 'bg-blue-600 text-white'
            : 'bg-slate-800 text-slate-400 hover:bg-slate-700 hover:text-white'
        }`}
      >
        <svg className="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path
            strokeLinecap="round"
            strokeLinejoin="round"
            strokeWidth={2}
            d="M3 4a1 1 0 011-1h16a1 1 0 011 1v2.586a1 1 0 01-.293.707l-6.414 6.414a1 1 0 00-.293.707V17l-4 4v-6.586a1 1 0 00-.293-.707L3.293 7.293A1 1 0 013 6.586V4z"
          />
        </svg>
        篩選
        {hasActiveFilters && (
          <span className="rounded-full bg-white/20 px-1.5 text-xs">
            {filters.genres.length + (filters.yearMin ? 1 : 0) + (filters.yearMax ? 1 : 0)}
          </span>
        )}
      </button>

      {isOpen && (
        <div className="mt-2 rounded-lg border border-slate-700 bg-slate-800 p-4">
          {/* Genre Section */}
          <div className="mb-4">
            <h4 className="mb-2 text-sm font-medium text-slate-300">類型</h4>
            <div className="flex flex-wrap gap-2">
              {availableGenres.map((genre) => (
                <label
                  key={genre}
                  className="flex cursor-pointer items-center gap-1.5 rounded-md bg-slate-700 px-2.5 py-1.5 text-sm transition-colors hover:bg-slate-600"
                >
                  <input
                    type="checkbox"
                    checked={localGenres.includes(genre)}
                    onChange={() => handleGenreToggle(genre)}
                    className="rounded border-slate-500 bg-slate-600 text-blue-500 focus:ring-blue-500 focus:ring-offset-0"
                  />
                  <span className="text-slate-200">{genre}</span>
                </label>
              ))}
              {availableGenres.length === 0 && (
                <span className="text-sm text-slate-500">無可用類型</span>
              )}
            </div>
          </div>

          {/* Year Range Section */}
          <div className="mb-4">
            <h4 className="mb-2 text-sm font-medium text-slate-300">年份範圍</h4>
            <div className="flex items-center gap-2">
              <input
                type="number"
                value={localYearMin}
                onChange={(e) => setLocalYearMin(e.target.value)}
                placeholder={stats?.yearMin ? String(stats.yearMin) : '最早'}
                min={stats?.yearMin}
                max={stats?.yearMax}
                className="w-24 rounded-md border border-slate-600 bg-slate-700 px-3 py-1.5 text-sm text-slate-200 placeholder-slate-500 focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
              />
              <span className="text-slate-500">—</span>
              <input
                type="number"
                value={localYearMax}
                onChange={(e) => setLocalYearMax(e.target.value)}
                placeholder={stats?.yearMax ? String(stats.yearMax) : '最新'}
                min={stats?.yearMin}
                max={stats?.yearMax}
                className="w-24 rounded-md border border-slate-600 bg-slate-700 px-3 py-1.5 text-sm text-slate-200 placeholder-slate-500 focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
              />
            </div>
          </div>

          {/* Actions */}
          <div className="flex gap-2">
            <button
              onClick={handleApply}
              className="rounded-md bg-blue-600 px-4 py-1.5 text-sm font-medium text-white transition-colors hover:bg-blue-500"
            >
              套用篩選
            </button>
            <button
              onClick={handleClear}
              className="rounded-md bg-slate-700 px-4 py-1.5 text-sm text-slate-300 transition-colors hover:bg-slate-600"
            >
              清除
            </button>
          </div>
        </div>
      )}
    </div>
  );
}
