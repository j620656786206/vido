import { useState, useRef, useEffect, useCallback } from 'react';
import { ArrowUpDown, ArrowUp, ArrowDown } from 'lucide-react';
import { cn } from '../../lib/utils';

export type SortField = 'title' | 'release_date' | 'rating' | 'created_at';
export type SortOrder = 'asc' | 'desc';

interface SortOption {
  field: SortField;
  label: string;
  defaultOrder: SortOrder;
}

const SORT_OPTIONS: SortOption[] = [
  { field: 'created_at', label: '新增日期', defaultOrder: 'desc' },
  { field: 'title', label: '標題', defaultOrder: 'asc' },
  { field: 'release_date', label: '年份', defaultOrder: 'desc' },
  { field: 'rating', label: '評分', defaultOrder: 'desc' },
];

interface SortSelectorProps {
  sortBy: SortField;
  sortOrder: SortOrder;
  onSortChange: (field: SortField, order: SortOrder) => void;
}

export function SortSelector({ sortBy, sortOrder, onSortChange }: SortSelectorProps) {
  const [isOpen, setIsOpen] = useState(false);
  const dropdownRef = useRef<HTMLDivElement>(null);

  const currentLabel = SORT_OPTIONS.find((o) => o.field === sortBy)?.label ?? '新增日期';

  const handleClickOutside = useCallback((e: MouseEvent) => {
    if (dropdownRef.current && !dropdownRef.current.contains(e.target as Node)) {
      setIsOpen(false);
    }
  }, []);

  const handleKeyDown = useCallback((e: KeyboardEvent) => {
    if (e.key === 'Escape') {
      setIsOpen(false);
    }
  }, []);

  useEffect(() => {
    if (isOpen) {
      document.addEventListener('mousedown', handleClickOutside);
      document.addEventListener('keydown', handleKeyDown);
      return () => {
        document.removeEventListener('mousedown', handleClickOutside);
        document.removeEventListener('keydown', handleKeyDown);
      };
    }
  }, [isOpen, handleClickOutside, handleKeyDown]);

  const handleOptionClick = (option: SortOption) => {
    if (option.field === sortBy) {
      onSortChange(option.field, sortOrder === 'asc' ? 'desc' : 'asc');
    } else {
      onSortChange(option.field, option.defaultOrder);
    }
    setIsOpen(false);
  };

  return (
    <div ref={dropdownRef} className="relative">
      <button
        onClick={() => setIsOpen(!isOpen)}
        className="inline-flex items-center gap-2 rounded-lg bg-slate-800 px-3 py-2 text-sm text-slate-300 transition-colors hover:bg-slate-700 hover:text-white"
        aria-label="排序方式"
        data-testid="sort-selector-button"
      >
        <ArrowUpDown size={16} />
        <span>{currentLabel}</span>
        <span data-testid="sort-direction-indicator">
          {sortOrder === 'asc' ? <ArrowUp size={14} /> : <ArrowDown size={14} />}
        </span>
      </button>

      {isOpen && (
        <div
          data-testid="sort-dropdown"
          className="absolute left-0 top-full z-30 mt-1 w-48 rounded-lg bg-slate-800 py-1 shadow-xl ring-1 ring-slate-700"
          role="listbox"
          aria-label="排序選項"
        >
          {SORT_OPTIONS.map((option) => (
            <button
              key={option.field}
              role="option"
              aria-selected={sortBy === option.field}
              data-testid={`sort-option-${option.field}`}
              onClick={() => handleOptionClick(option)}
              className={cn(
                'flex w-full items-center justify-between px-4 py-2 text-sm transition-colors',
                sortBy === option.field
                  ? 'bg-blue-600 text-white'
                  : 'text-slate-300 hover:bg-slate-700 hover:text-white'
              )}
            >
              <span>{option.label}</span>
              {sortBy === option.field && (
                <span>{sortOrder === 'asc' ? <ArrowUp size={14} /> : <ArrowDown size={14} />}</span>
              )}
            </button>
          ))}
        </div>
      )}
    </div>
  );
}
