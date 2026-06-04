// Implements: Component/SearchInput (6MxLT)
// Design ref: ux-design.pen Screen AS-2 - Search Suggestions Dropdown (TMaw5)
// Source: ux-design.pen (Pencil app)
import { useCallback, useEffect, useRef, useState } from 'react';
import { useNavigate } from '@tanstack/react-router';
import { useDebounce } from 'use-debounce';
import { Search, X } from 'lucide-react';
import { useInstantSearch } from '../../hooks/useSearchMedia';
import { cn } from '../../lib/utils';
import {
  SearchSuggestions,
  buildNavigableItems,
  searchOptionId,
  type NavigableItem,
} from './SearchSuggestions';

interface InstantSearchBarProps {
  variant?: 'desktop' | 'mobile';
  autoFocus?: boolean;
  /** Called after a navigation occurs (used to close the mobile overlay). */
  onClose?: () => void;
  className?: string;
}

const MIN_QUERY_LENGTH = 2;

export function InstantSearchBar({
  variant = 'desktop',
  autoFocus = false,
  onClose,
  className,
}: InstantSearchBarProps) {
  const navigate = useNavigate();
  const [value, setValue] = useState('');
  const [focused, setFocused] = useState(false);
  const [activeIndex, setActiveIndex] = useState(-1);
  const blurTimer = useRef<ReturnType<typeof setTimeout> | undefined>(undefined);

  // 300ms client-side debounce — the server is never asked to debounce.
  const [debouncedQuery] = useDebounce(value.trim(), 300);

  const { data, isLoading } = useInstantSearch(debouncedQuery);

  const navigable = buildNavigableItems(data);
  const open = focused && debouncedQuery.length >= MIN_QUERY_LENGTH;

  // Reset the highlighted row whenever the result set changes.
  useEffect(() => {
    setActiveIndex(-1);
  }, [debouncedQuery]);

  useEffect(() => () => clearTimeout(blurTimer.current), []);

  const close = useCallback(() => {
    setFocused(false);
    onClose?.();
  }, [onClose]);

  const goToItem = useCallback(
    (item: NavigableItem) => {
      navigate({
        to: '/media/$type/$id',
        params: { type: item.type, id: String(item.id) },
      });
      close();
    },
    [navigate, close]
  );

  const submitAll = useCallback(() => {
    const q = value.trim();
    if (q.length === 0) return;
    navigate({ to: '/search', search: { q } });
    close();
  }, [navigate, value, close]);

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent<HTMLInputElement>) => {
      switch (e.key) {
        case 'ArrowDown':
          if (open && navigable.length > 0) {
            e.preventDefault();
            setActiveIndex((i) => Math.min(i + 1, navigable.length - 1));
          }
          break;
        case 'ArrowUp':
          if (open && navigable.length > 0) {
            e.preventDefault();
            setActiveIndex((i) => Math.max(i - 1, -1));
          }
          break;
        case 'Enter':
          e.preventDefault();
          if (open && activeIndex >= 0 && navigable[activeIndex]) {
            goToItem(navigable[activeIndex]);
          } else {
            submitAll();
          }
          break;
        case 'Escape':
          if (value) {
            setValue('');
          } else {
            close();
          }
          break;
        default:
          break;
      }
    },
    [open, navigable, activeIndex, goToItem, submitAll, value, close]
  );

  const handleBlur = useCallback(() => {
    // Delay so a click on a suggestion (which blurs the input) still registers.
    blurTimer.current = setTimeout(() => setFocused(false), 150);
  }, []);

  const handleFocus = useCallback(() => {
    clearTimeout(blurTimer.current);
    setFocused(true);
  }, []);

  const isMobile = variant === 'mobile';

  return (
    <div
      className={cn(isMobile ? 'flex h-full flex-col' : 'relative w-full', className)}
      data-testid="instant-search-bar"
    >
      <div className="relative">
        <Search
          className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-[var(--text-muted)]"
          aria-hidden="true"
        />
        <input
          type="text"
          value={value}
          onChange={(e) => setValue(e.target.value)}
          onKeyDown={handleKeyDown}
          onFocus={handleFocus}
          onBlur={handleBlur}
          autoFocus={autoFocus}
          autoComplete="off"
          placeholder="搜尋媒體庫..."
          aria-label="搜尋"
          role="combobox"
          aria-expanded={open}
          aria-controls="search-suggestions"
          aria-activedescendant={open && activeIndex >= 0 ? searchOptionId(activeIndex) : undefined}
          aria-autocomplete="list"
          className={cn(
            'w-full border border-[var(--border-subtle)]/50 bg-[var(--bg-secondary)]/60 text-[var(--text-primary)] placeholder-[var(--text-muted)] transition-colors focus:border-[var(--accent-primary)] focus:outline-none focus:ring-1 focus:ring-[var(--accent-primary)]',
            isMobile
              ? 'rounded-lg py-2.5 pl-10 pr-10 text-base'
              : 'rounded-full py-1.5 pl-9 pr-9 text-sm'
          )}
          data-testid="instant-search-input"
        />
        {value && (
          <button
            type="button"
            onClick={() => setValue('')}
            className="absolute right-3 top-1/2 -translate-y-1/2 text-[var(--text-secondary)] transition-colors hover:text-white focus:outline-none"
            aria-label="清除搜尋"
            data-testid="instant-search-clear"
          >
            <X className="h-4 w-4" />
          </button>
        )}
      </div>

      {open && (
        <SearchSuggestions
          result={data}
          isLoading={isLoading}
          query={debouncedQuery}
          activeIndex={activeIndex}
          onSelect={goToItem}
          onSubmitAll={submitAll}
          onActiveIndexChange={setActiveIndex}
          floating={!isMobile}
        />
      )}
    </div>
  );
}
