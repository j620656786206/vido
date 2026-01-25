/**
 * ManualSearchDialog Component (Story 3.7 - AC1, AC2, AC4)
 * Dialog for manual metadata search and selection
 */

import { useState, useCallback, useEffect } from 'react';
import { X, Search, Loader2 } from 'lucide-react';
import { useDebouncedCallback } from 'use-debounce';
import { cn } from '../../lib/utils';
import { useManualSearch, useApplyMetadata } from '../../hooks/useManualSearch';
import { SearchResultsGrid } from './SearchResultsGrid';
import { FallbackStatusDisplay } from './FallbackStatusDisplay';
import type { ManualSearchResultItem } from '../../services/metadata';

export interface FallbackStatus {
  attempts: Array<{
    source: string;
    success: boolean;
    skipped?: boolean;
    skipReason?: string;
  }>;
  totalDuration?: number;
  cancelled?: boolean;
}

export interface ManualSearchDialogProps {
  isOpen: boolean;
  onClose: () => void;
  initialQuery?: string;
  mediaId: string;
  fallbackStatus?: FallbackStatus;
  onSuccess: (metadata: ManualSearchResultItem) => void;
}

type SourceType = 'all' | 'tmdb' | 'douban' | 'wikipedia';
type MediaType = 'movie' | 'tv';

const SOURCE_OPTIONS: { value: SourceType; label: string }[] = [
  { value: 'all', label: '全部' },
  { value: 'tmdb', label: 'TMDb' },
  { value: 'douban', label: '豆瓣' },
  { value: 'wikipedia', label: 'Wikipedia' },
];

export function ManualSearchDialog({
  isOpen,
  onClose,
  initialQuery = '',
  mediaId,
  fallbackStatus,
  onSuccess,
}: ManualSearchDialogProps) {
  const [query, setQuery] = useState(initialQuery);
  const [debouncedQuery, setDebouncedQuery] = useState(initialQuery);
  const [source, setSource] = useState<SourceType>('all');
  const [mediaType, setMediaType] = useState<MediaType>('movie');
  const [selectedItem, setSelectedItem] = useState<ManualSearchResultItem | null>(null);
  const [showConfirmation, setShowConfirmation] = useState(false);

  // Debounced search (300ms per story requirements)
  const debouncedSearch = useDebouncedCallback((value: string) => {
    setDebouncedQuery(value);
  }, 300);

  // Reset state when dialog opens with new initial query
  useEffect(() => {
    if (isOpen) {
      setQuery(initialQuery);
      setDebouncedQuery(initialQuery);
      setSelectedItem(null);
      setShowConfirmation(false);
    }
  }, [isOpen, initialQuery]);

  // Manual search query
  const { data, isLoading, error } = useManualSearch({
    query: debouncedQuery,
    mediaType,
    source,
  });

  // Apply metadata mutation
  const applyMetadata = useApplyMetadata();

  const handleQueryChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      const newValue = e.target.value;
      setQuery(newValue);
      debouncedSearch(newValue);
    },
    [debouncedSearch]
  );

  const handleSelect = useCallback((item: ManualSearchResultItem) => {
    setSelectedItem(item);
    setShowConfirmation(true);
  }, []);

  const handleConfirm = useCallback(async () => {
    if (!selectedItem) return;

    try {
      await applyMetadata.mutateAsync({
        mediaId,
        mediaType: mediaType === 'tv' ? 'series' : 'movie',
        selectedItem: {
          id: selectedItem.id,
          source: selectedItem.source,
        },
        learnPattern: true,
      });

      onSuccess(selectedItem);
      onClose();
    } catch (err) {
      // Error is handled by the mutation state
      console.error('Failed to apply metadata:', err);
    }
  }, [selectedItem, applyMetadata, mediaId, mediaType, onSuccess, onClose]);

  const handleCancelConfirmation = useCallback(() => {
    setShowConfirmation(false);
  }, []);

  const handleKeyDown = useCallback(
    (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        if (showConfirmation) {
          setShowConfirmation(false);
        } else {
          onClose();
        }
      }
    },
    [showConfirmation, onClose]
  );

  useEffect(() => {
    if (isOpen) {
      document.addEventListener('keydown', handleKeyDown);
      document.body.style.overflow = 'hidden';
    }

    return () => {
      document.removeEventListener('keydown', handleKeyDown);
      document.body.style.overflow = '';
    };
  }, [isOpen, handleKeyDown]);

  const handleBackdropClick = (e: React.MouseEvent) => {
    if (e.target === e.currentTarget) {
      onClose();
    }
  };

  if (!isOpen) return null;

  return (
    <div
      className="fixed inset-0 z-50"
      onClick={handleBackdropClick}
      role="dialog"
      aria-modal="true"
      aria-labelledby="manual-search-title"
    >
      {/* Backdrop */}
      <div className="absolute inset-0 bg-black/60 backdrop-blur-sm" />

      {/* Dialog */}
      <div
        className={cn(
          'fixed left-1/2 top-1/2 z-50 -translate-x-1/2 -translate-y-1/2',
          'w-[90vw] max-w-4xl max-h-[85vh]',
          'bg-slate-900 rounded-xl shadow-2xl',
          'flex flex-col overflow-hidden'
        )}
        data-testid="manual-search-dialog"
      >
        {/* Header */}
        <div className="flex items-center justify-between border-b border-slate-700 px-6 py-4">
          <h2 id="manual-search-title" className="text-xl font-semibold text-white">
            手動搜尋 Metadata
          </h2>
          <button
            onClick={onClose}
            className={cn(
              'rounded-lg p-2 text-gray-400',
              'hover:bg-slate-800 hover:text-white',
              'focus:outline-none focus-visible:ring-2 focus-visible:ring-blue-500',
              'transition-colors'
            )}
            aria-label="關閉"
          >
            <X className="h-5 w-5" />
          </button>
        </div>

        {/* Fallback Status Display (UX-4) */}
        {fallbackStatus && <FallbackStatusDisplay status={fallbackStatus} />}

        {/* Search Controls */}
        <div className="px-6 py-4 space-y-4 border-b border-slate-800">
          {/* Search Input */}
          <div className="relative">
            <Search
              className="absolute left-3 top-1/2 -translate-y-1/2 h-5 w-5 text-slate-400"
              aria-hidden="true"
            />
            <input
              type="text"
              value={query}
              onChange={handleQueryChange}
              placeholder="輸入電影或影集名稱..."
              autoFocus
              className={cn(
                'w-full pl-10 pr-4 py-3',
                'bg-slate-800 border border-slate-700 rounded-lg',
                'text-white placeholder-slate-400',
                'focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent',
                'transition-colors'
              )}
            />
            {isLoading && (
              <Loader2
                className="absolute right-3 top-1/2 -translate-y-1/2 h-5 w-5 text-blue-400 animate-spin"
                aria-hidden="true"
              />
            )}
          </div>

          {/* Filters Row */}
          <div className="flex flex-wrap gap-4">
            {/* Media Type Toggle */}
            <div className="flex items-center gap-2">
              <span className="text-sm text-slate-400">類型：</span>
              <div className="flex rounded-lg bg-slate-800 p-1">
                <button
                  onClick={() => setMediaType('movie')}
                  className={cn(
                    'px-3 py-1.5 rounded-md text-sm font-medium transition-colors',
                    mediaType === 'movie'
                      ? 'bg-blue-600 text-white'
                      : 'text-slate-400 hover:text-white'
                  )}
                >
                  電影
                </button>
                <button
                  onClick={() => setMediaType('tv')}
                  className={cn(
                    'px-3 py-1.5 rounded-md text-sm font-medium transition-colors',
                    mediaType === 'tv'
                      ? 'bg-blue-600 text-white'
                      : 'text-slate-400 hover:text-white'
                  )}
                >
                  影集
                </button>
              </div>
            </div>

            {/* Source Selector */}
            <div className="flex items-center gap-2">
              <span className="text-sm text-slate-400">來源：</span>
              <select
                value={source}
                onChange={(e) => setSource(e.target.value as SourceType)}
                className={cn(
                  'px-3 py-1.5 rounded-lg text-sm',
                  'bg-slate-800 border border-slate-700 text-white',
                  'focus:outline-none focus:ring-2 focus:ring-blue-500'
                )}
              >
                {SOURCE_OPTIONS.map((opt) => (
                  <option key={opt.value} value={opt.value}>
                    {opt.label}
                  </option>
                ))}
              </select>
            </div>
          </div>
        </div>

        {/* Results Area */}
        <div className="flex-1 overflow-y-auto px-6 py-4">
          {error && (
            <div className="text-center py-8 text-red-400">
              <p>搜尋失敗：{error.message}</p>
            </div>
          )}

          <SearchResultsGrid
            results={data?.results ?? []}
            selectedId={selectedItem?.id ?? null}
            onSelect={handleSelect}
            isLoading={isLoading}
            searchedSources={data?.searchedSources ?? []}
          />
        </div>

        {/* Confirmation Dialog */}
        {showConfirmation && selectedItem && (
          <div className="absolute inset-0 bg-black/70 flex items-center justify-center">
            <div
              className="bg-slate-800 rounded-xl p-6 max-w-md w-full mx-4 shadow-2xl"
              data-testid="confirmation-dialog"
            >
              <h3 className="text-lg font-semibold text-white mb-4">確認選擇</h3>
              <div className="flex gap-4 mb-6">
                {selectedItem.posterUrl && (
                  <img
                    src={selectedItem.posterUrl}
                    alt={selectedItem.title}
                    className="w-20 h-30 object-cover rounded"
                  />
                )}
                <div>
                  <p className="text-white font-medium">{selectedItem.title}</p>
                  {selectedItem.titleZhTW && selectedItem.titleZhTW !== selectedItem.title && (
                    <p className="text-slate-400 text-sm">{selectedItem.titleZhTW}</p>
                  )}
                  <p className="text-slate-500 text-sm">
                    {selectedItem.year} · {selectedItem.source.toUpperCase()}
                  </p>
                </div>
              </div>
              <div className="flex gap-3 justify-end">
                <button
                  onClick={handleCancelConfirmation}
                  className="px-4 py-2 rounded-lg text-slate-300 hover:bg-slate-700 transition-colors"
                >
                  取消
                </button>
                <button
                  onClick={handleConfirm}
                  disabled={applyMetadata.isPending}
                  className={cn(
                    'px-4 py-2 rounded-lg bg-blue-600 text-white',
                    'hover:bg-blue-700 transition-colors',
                    'disabled:opacity-50 disabled:cursor-not-allowed',
                    'flex items-center gap-2'
                  )}
                >
                  {applyMetadata.isPending && <Loader2 className="h-4 w-4 animate-spin" />}
                  確認套用
                </button>
              </div>
              {applyMetadata.error && (
                <p className="mt-4 text-sm text-red-400">套用失敗：{applyMetadata.error.message}</p>
              )}
            </div>
          </div>
        )}
      </div>
    </div>
  );
}

export default ManualSearchDialog;
