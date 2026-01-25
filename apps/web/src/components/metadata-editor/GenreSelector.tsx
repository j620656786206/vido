/**
 * GenreSelector Component (Story 3.8 - AC1)
 * Multi-select genre picker with toggle buttons
 */

import { cn } from '../../lib/utils';

export interface GenreOption {
  value: string;
  label: string;
}

export interface GenreSelectorProps {
  selectedGenres: string[];
  onToggle: (genre: string) => void;
  options?: GenreOption[];
  label?: string;
}

// Default genre options following project conventions
export const GENRE_OPTIONS: GenreOption[] = [
  { value: 'action', label: '動作' },
  { value: 'adventure', label: '冒險' },
  { value: 'animation', label: '動畫' },
  { value: 'comedy', label: '喜劇' },
  { value: 'crime', label: '犯罪' },
  { value: 'documentary', label: '紀錄片' },
  { value: 'drama', label: '劇情' },
  { value: 'family', label: '家庭' },
  { value: 'fantasy', label: '奇幻' },
  { value: 'history', label: '歷史' },
  { value: 'horror', label: '恐怖' },
  { value: 'music', label: '音樂' },
  { value: 'mystery', label: '懸疑' },
  { value: 'romance', label: '愛情' },
  { value: 'sci-fi', label: '科幻' },
  { value: 'thriller', label: '驚悚' },
  { value: 'war', label: '戰爭' },
  { value: 'western', label: '西部' },
];

export function GenreSelector({
  selectedGenres,
  onToggle,
  options = GENRE_OPTIONS,
  label = '類型',
}: GenreSelectorProps) {
  return (
    <div>
      <label className="block text-sm font-medium text-slate-300 mb-2">{label}</label>
      <div className="flex flex-wrap gap-2" data-testid="genre-selector">
        {options.map((genre) => (
          <button
            key={genre.value}
            type="button"
            onClick={() => onToggle(genre.value)}
            className={cn(
              'px-3 py-1.5 rounded-full text-sm font-medium transition-colors',
              selectedGenres.includes(genre.value)
                ? 'bg-blue-600 text-white'
                : 'bg-slate-800 text-slate-400 hover:bg-slate-700 hover:text-white'
            )}
            aria-pressed={selectedGenres.includes(genre.value)}
          >
            {genre.label}
          </button>
        ))}
      </div>
    </div>
  );
}

export default GenreSelector;
