import { Link } from '@tanstack/react-router';
import { ArrowUp, ArrowDown, Check } from 'lucide-react';
import { cn } from '../../lib/utils';
import { HighlightText } from '../ui/HighlightText';
import type { LibraryItem, SortField, SortOrder } from '../../types/library';

interface LibraryTableProps {
  items: LibraryItem[];
  isLoading?: boolean;
  sortBy?: string;
  sortOrder?: SortOrder;
  onSort?: (field: SortField) => void;
  highlightQuery?: string;
  selectionMode?: boolean;
  selectedIds?: Set<string>;
  onSelect?: (id: string, e: React.MouseEvent) => void;
}

interface Column {
  key: SortField | 'poster' | 'genre';
  label: string;
  sortable: boolean;
  className?: string;
}

const COLUMNS: Column[] = [
  { key: 'poster', label: '', sortable: false, className: 'w-16' },
  { key: 'title', label: '標題', sortable: true },
  { key: 'release_date', label: '年份', sortable: true, className: 'w-24 text-center' },
  { key: 'genre', label: '類型', sortable: false },
  { key: 'rating', label: '評分', sortable: true, className: 'w-20 text-center' },
  { key: 'created_at', label: '新增日期', sortable: true, className: 'w-28 text-right' },
];

function getItemData(item: LibraryItem) {
  if (item.type === 'movie' && item.movie) {
    const m = item.movie;
    return {
      id: m.tmdbId ?? 0,
      type: 'movie' as const,
      title: m.title,
      originalTitle: m.originalTitle,
      posterPath: m.posterPath,
      releaseDate: m.releaseDate,
      genres: m.genres ?? [],
      rating: m.voteAverage,
      createdAt: m.createdAt,
    };
  }
  if (item.type === 'series' && item.series) {
    const s = item.series;
    return {
      id: s.tmdbId ?? 0,
      type: 'tv' as const,
      title: s.title,
      originalTitle: s.originalTitle,
      posterPath: s.posterPath,
      releaseDate: s.firstAirDate,
      genres: s.genres ?? [],
      rating: s.voteAverage,
      createdAt: s.createdAt,
    };
  }
  return null;
}

function formatDate(dateStr: string | undefined): string {
  if (!dateStr) return '-';
  try {
    return new Date(dateStr).toLocaleDateString('zh-TW', {
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
    });
  } catch {
    return '-';
  }
}

function formatYear(dateStr: string | undefined): string {
  if (!dateStr) return '-';
  return dateStr.slice(0, 4) || '-';
}

function getItemId(item: LibraryItem): string | undefined {
  return item.movie?.id || item.series?.id;
}

export function LibraryTable({
  items,
  isLoading,
  sortBy,
  sortOrder,
  onSort,
  highlightQuery,
  selectionMode,
  selectedIds,
  onSelect,
}: LibraryTableProps) {
  if (isLoading) {
    return (
      <div data-testid="library-table-loading" className="space-y-2">
        {Array.from({ length: 8 }).map((_, i) => (
          <div key={i} className="h-14 animate-pulse rounded bg-slate-800" />
        ))}
      </div>
    );
  }

  if (items.length === 0) {
    return null;
  }

  return (
    <div data-testid="library-table" className="overflow-x-auto">
      <table className="w-full border-collapse">
        <thead>
          <tr className="border-b border-slate-700 bg-slate-800/50 text-left text-sm text-slate-400">
            {selectionMode && <th className="w-10 px-3 py-2" />}
            {COLUMNS.map((col) => (
              <th key={col.key} className={cn('px-3 py-2 font-medium', col.className)}>
                {col.sortable ? (
                  <button
                    onClick={() => onSort?.(col.key as SortField)}
                    className="inline-flex items-center gap-1 transition-colors hover:text-white"
                    data-testid={`sort-${col.key}`}
                  >
                    {col.label}
                    {sortBy === col.key && (
                      <span data-testid={`sort-indicator-${col.key}`}>
                        {sortOrder === 'asc' ? <ArrowUp size={14} /> : <ArrowDown size={14} />}
                      </span>
                    )}
                  </button>
                ) : (
                  col.label
                )}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {items.map((item, index) => {
            const data = getItemData(item);
            if (!data) return null;
            const itemId = getItemId(item);
            const isSelected = selectionMode && itemId && selectedIds?.has(itemId);

            return (
              <tr
                key={`${data.type}-${data.id}-${index}`}
                data-testid="library-table-row"
                className={cn(
                  'border-b border-slate-800 transition-colors hover:bg-slate-800/50',
                  selectionMode && 'cursor-pointer',
                  isSelected && 'bg-blue-600/10'
                )}
                onClick={selectionMode && itemId ? (e) => onSelect?.(itemId, e) : undefined}
              >
                {selectionMode && (
                  <td className="px-3 py-2">
                    <div
                      data-testid="table-selection-checkbox"
                      className={cn(
                        'flex h-5 w-5 items-center justify-center rounded border-2 transition-colors',
                        isSelected
                          ? 'border-blue-500 bg-blue-500 text-white'
                          : 'border-slate-500 bg-transparent'
                      )}
                    >
                      {isSelected && <Check className="h-3 w-3" />}
                    </div>
                  </td>
                )}
                <td className="px-3 py-2">
                  <Link to="/media/$type/$id" params={{ type: data.type, id: String(data.id) }}>
                    {data.posterPath ? (
                      <img
                        src={`https://image.tmdb.org/t/p/w92${data.posterPath}`}
                        alt=""
                        className="h-[72px] w-12 rounded object-cover"
                        loading="lazy"
                      />
                    ) : (
                      <div className="flex h-[72px] w-12 items-center justify-center rounded bg-slate-700 text-xs text-slate-500">
                        N/A
                      </div>
                    )}
                  </Link>
                </td>
                <td className="px-3 py-2">
                  <Link
                    to="/media/$type/$id"
                    params={{ type: data.type, id: String(data.id) }}
                    className="block"
                  >
                    <div className="text-sm font-medium text-white hover:text-blue-400">
                      <HighlightText text={data.title} query={highlightQuery} />
                    </div>
                    {data.originalTitle && data.originalTitle !== data.title && (
                      <div className="text-xs text-slate-500">
                        <HighlightText text={data.originalTitle} query={highlightQuery} />
                      </div>
                    )}
                  </Link>
                </td>
                <td className="px-3 py-2 text-center text-sm text-slate-400">
                  {formatYear(data.releaseDate)}
                </td>
                <td className="px-3 py-2">
                  <div className="flex flex-wrap gap-1">
                    {data.genres.slice(0, 3).map((genre) => (
                      <span
                        key={genre}
                        className="rounded bg-slate-700 px-1.5 py-0.5 text-xs text-slate-300"
                      >
                        {genre}
                      </span>
                    ))}
                  </div>
                </td>
                <td className="px-3 py-2 text-center text-sm">
                  {data.rating != null ? (
                    <span className="text-yellow-400">★ {data.rating.toFixed(1)}</span>
                  ) : (
                    <span className="text-slate-600">-</span>
                  )}
                </td>
                <td className="px-3 py-2 text-right text-sm text-slate-400">
                  {formatDate(data.createdAt)}
                </td>
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
  );
}
