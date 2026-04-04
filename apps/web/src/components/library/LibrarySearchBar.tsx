import { useState, useCallback, useRef, useEffect } from 'react';
import { useDebouncedCallback } from 'use-debounce';
import { Search, X } from 'lucide-react';
import { cn } from '../../lib/utils';

interface LibrarySearchBarProps {
  onSearch: (query: string) => void;
  initialQuery?: string;
  resultCount?: number;
  className?: string;
}

export function LibrarySearchBar({
  onSearch,
  initialQuery = '',
  resultCount,
  className,
}: LibrarySearchBarProps) {
  const [value, setValue] = useState(initialQuery);
  const inputRef = useRef<HTMLInputElement>(null);

  const debouncedSearch = useDebouncedCallback((query: string) => {
    if (query.length >= 2 || query.length === 0) {
      onSearch(query);
    }
  }, 500);

  const handleChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      const newValue = e.target.value;
      setValue(newValue);
      debouncedSearch(newValue);
    },
    [debouncedSearch]
  );

  const handleClear = useCallback(() => {
    setValue('');
    onSearch('');
    inputRef.current?.focus();
  }, [onSearch]);

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent<HTMLInputElement>) => {
      if (e.key === 'Escape') {
        handleClear();
      }
    },
    [handleClear]
  );

  // Ctrl+K / Cmd+K global shortcut to focus search
  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
        e.preventDefault();
        inputRef.current?.focus();
      }
    };
    document.addEventListener('keydown', handler);
    return () => document.removeEventListener('keydown', handler);
  }, []);

  const showResultCount = value.length >= 2 && resultCount !== undefined;

  return (
    <div className={cn('w-full', className)}>
      <div className="relative w-full max-w-md">
        <Search
          className="absolute left-3 top-1/2 -translate-y-1/2 h-5 w-5 text-[var(--text-secondary)]"
          aria-hidden="true"
        />
        <input
          ref={inputRef}
          type="text"
          value={value}
          onChange={handleChange}
          onKeyDown={handleKeyDown}
          placeholder="搜尋媒體標題..."
          aria-label="搜尋媒體標題"
          className="w-full pl-10 pr-10 py-2.5 bg-[var(--bg-secondary)] border border-[var(--border-subtle)] rounded-full
                     text-white placeholder-[var(--text-muted)] focus:outline-none focus:ring-2
                     focus:ring-[var(--accent-primary)] focus:border-transparent transition-colors text-sm"
        />
        {value && (
          <button
            onClick={handleClear}
            className="absolute right-3 top-1/2 -translate-y-1/2 text-[var(--text-secondary)] hover:text-white
                       transition-colors focus:outline-none focus:ring-2 focus:ring-[var(--accent-primary)] rounded"
            aria-label="清除搜尋"
            type="button"
          >
            <X className="h-4 w-4" />
          </button>
        )}
      </div>
      {showResultCount && (
        <p className="mt-2 text-sm text-[var(--text-secondary)]" data-testid="search-result-count">
          找到 {resultCount} 個結果
        </p>
      )}
    </div>
  );
}
