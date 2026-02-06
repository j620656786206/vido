/**
 * SearchResultsGrid Component (Story 3.7 - AC2)
 * Grid display for manual search results
 */

import { Search, Loader2 } from 'lucide-react';
import { SearchResultCard } from './SearchResultCard';
import type { ManualSearchResultItem } from '../../services/metadata';

export interface SearchResultsGridProps {
  results: ManualSearchResultItem[];
  selectedId: string | null;
  onSelect: (item: ManualSearchResultItem) => void;
  isLoading: boolean;
  searchedSources?: string[];
}

/**
 * Loading skeleton for search results
 */
function SearchResultsSkeleton() {
  return (
    <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-4">
      {Array.from({ length: 8 }).map((_, i) => (
        <div key={i} className="animate-pulse">
          <div className="aspect-[2/3] bg-slate-700 rounded-lg" />
          <div className="mt-2 h-4 bg-slate-700 rounded w-3/4" />
          <div className="mt-1 h-3 bg-slate-700 rounded w-1/2" />
        </div>
      ))}
    </div>
  );
}

/**
 * Empty state when no results found
 */
function EmptyState({ searchedSources }: { searchedSources?: string[] }) {
  return (
    <div className="flex flex-col items-center justify-center py-12 text-center">
      <Search className="h-12 w-12 text-slate-500 mb-4" />
      <h3 className="text-lg font-medium text-white mb-2">找不到結果</h3>
      <p className="text-slate-400 max-w-md">試試其他關鍵字或選擇不同的來源</p>
      {searchedSources && searchedSources.length > 0 && (
        <p className="text-slate-500 text-sm mt-2">已搜尋：{searchedSources.join(', ')}</p>
      )}
    </div>
  );
}

/**
 * Initial state before search
 */
function InitialState() {
  return (
    <div className="flex flex-col items-center justify-center py-12 text-center">
      <Search className="h-12 w-12 text-slate-500 mb-4" />
      <h3 className="text-lg font-medium text-white mb-2">搜尋 Metadata</h3>
      <p className="text-slate-400">輸入至少 2 個字元開始搜尋</p>
    </div>
  );
}

export function SearchResultsGrid({
  results,
  selectedId,
  onSelect,
  isLoading,
  searchedSources,
}: SearchResultsGridProps) {
  // Loading state
  if (isLoading) {
    return (
      <div className="relative">
        <SearchResultsSkeleton />
        <div className="absolute inset-0 flex items-center justify-center bg-slate-900/50">
          <Loader2 className="h-8 w-8 text-blue-400 animate-spin" />
        </div>
      </div>
    );
  }

  // Initial state (no search performed yet)
  if (!searchedSources || searchedSources.length === 0) {
    return <InitialState />;
  }

  // Empty results
  if (results.length === 0) {
    return <EmptyState searchedSources={searchedSources} />;
  }

  // Results grid
  return (
    <div className="space-y-4">
      {/* Results count and sources */}
      <div className="flex items-center justify-between text-sm text-slate-400">
        <span>找到 {results.length} 個結果</span>
        <span>來源：{searchedSources.join(', ')}</span>
      </div>

      {/* Grid */}
      <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-4">
        {results.map((item) => (
          <SearchResultCard
            key={item.id}
            item={item}
            isSelected={item.id === selectedId}
            onSelect={() => onSelect(item)}
          />
        ))}
      </div>
    </div>
  );
}

export default SearchResultsGrid;
