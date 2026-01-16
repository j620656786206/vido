import { useState, useCallback } from 'react';
import { useDebouncedCallback } from 'use-debounce';
import { Search, X } from 'lucide-react';
import { cn } from '../../lib/utils';

interface SearchBarProps {
  onSearch: (query: string) => void;
  initialQuery?: string;
  placeholder?: string;
  className?: string;
}

export function SearchBar({
  onSearch,
  initialQuery = '',
  placeholder = '搜尋電影或影集...',
  className,
}: SearchBarProps) {
  const [value, setValue] = useState(initialQuery);

  const debouncedSearch = useDebouncedCallback((query: string) => {
    if (query.length >= 2 || query.length === 0) {
      onSearch(query);
    }
  }, 300);

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
  }, [onSearch]);

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent<HTMLInputElement>) => {
      if (e.key === 'Escape') {
        handleClear();
      }
    },
    [handleClear]
  );

  return (
    <div className={cn('relative w-full max-w-2xl', className)}>
      <Search
        className="absolute left-3 top-1/2 -translate-y-1/2 h-5 w-5 text-slate-400"
        aria-hidden="true"
      />
      <input
        type="text"
        value={value}
        onChange={handleChange}
        onKeyDown={handleKeyDown}
        placeholder={placeholder}
        aria-label="搜尋"
        className="w-full pl-10 pr-10 py-3 bg-slate-800 border border-slate-700 rounded-lg
                   text-white placeholder-slate-400 focus:outline-none focus:ring-2
                   focus:ring-blue-500 focus:border-transparent transition-colors"
      />
      {value && (
        <button
          onClick={handleClear}
          className="absolute right-3 top-1/2 -translate-y-1/2 text-slate-400 hover:text-white
                     transition-colors focus:outline-none focus:ring-2 focus:ring-blue-500 rounded"
          aria-label="清除搜尋"
          type="button"
        >
          <X className="h-5 w-5" />
        </button>
      )}
    </div>
  );
}
