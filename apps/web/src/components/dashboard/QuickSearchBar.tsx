import { useNavigate } from '@tanstack/react-router';
import { useState, useEffect, useRef, useCallback } from 'react';
import { Clock, X } from 'lucide-react';
import { cn } from '../../lib/utils';

const RECENT_SEARCHES_KEY = 'vido-recent-searches';
const MAX_RECENT_SEARCHES = 10;

interface QuickSearchBarProps {
  className?: string;
}

export function QuickSearchBar({ className }: QuickSearchBarProps) {
  const navigate = useNavigate();
  const [query, setQuery] = useState('');
  const [recentSearches, setRecentSearches] = useState<string[]>([]);
  const [showDropdown, setShowDropdown] = useState(false);
  const [focusedIndex, setFocusedIndex] = useState(-1);
  const inputRef = useRef<HTMLInputElement>(null);
  const dropdownRef = useRef<HTMLDivElement>(null);

  // Load recent searches from sessionStorage
  useEffect(() => {
    try {
      const stored = sessionStorage.getItem(RECENT_SEARCHES_KEY);
      if (stored) {
        setRecentSearches(JSON.parse(stored));
      }
    } catch {
      // Ignore parse errors
    }
  }, []);

  // Save a new search to recent searches
  const saveRecentSearch = useCallback((searchQuery: string) => {
    const trimmed = searchQuery.trim();
    if (!trimmed) return;

    setRecentSearches((prev) => {
      const filtered = prev.filter((s) => s !== trimmed);
      const updated = [trimmed, ...filtered].slice(0, MAX_RECENT_SEARCHES);
      try {
        sessionStorage.setItem(RECENT_SEARCHES_KEY, JSON.stringify(updated));
      } catch {
        // Ignore storage errors
      }
      return updated;
    });
  }, []);

  // Remove a search from recent searches
  const removeRecentSearch = useCallback((searchQuery: string, e: React.MouseEvent) => {
    e.stopPropagation();
    e.preventDefault();
    setRecentSearches((prev) => {
      const updated = prev.filter((s) => s !== searchQuery);
      try {
        sessionStorage.setItem(RECENT_SEARCHES_KEY, JSON.stringify(updated));
      } catch {
        // Ignore storage errors
      }
      return updated;
    });
  }, []);

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (query.trim()) {
      saveRecentSearch(query);
      setShowDropdown(false);
      navigate({ to: '/search', search: { q: query.trim() } });
    }
  };

  const handleSelectRecent = (searchQuery: string) => {
    setQuery(searchQuery);
    setShowDropdown(false);
    saveRecentSearch(searchQuery);
    navigate({ to: '/search', search: { q: searchQuery } });
  };

  const handleInputFocus = () => {
    if (recentSearches.length > 0 && !query) {
      setShowDropdown(true);
    }
  };

  const handleInputChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    setQuery(e.target.value);
    // Show dropdown only when input is empty and we have recent searches
    if (!e.target.value && recentSearches.length > 0) {
      setShowDropdown(true);
    } else {
      setShowDropdown(false);
    }
    setFocusedIndex(-1);
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (!showDropdown || recentSearches.length === 0) return;

    if (e.key === 'ArrowDown') {
      e.preventDefault();
      setFocusedIndex((prev) => (prev < recentSearches.length - 1 ? prev + 1 : 0));
    } else if (e.key === 'ArrowUp') {
      e.preventDefault();
      setFocusedIndex((prev) => (prev > 0 ? prev - 1 : recentSearches.length - 1));
    } else if (e.key === 'Enter' && focusedIndex >= 0) {
      e.preventDefault();
      handleSelectRecent(recentSearches[focusedIndex]);
    } else if (e.key === 'Escape') {
      setShowDropdown(false);
      setFocusedIndex(-1);
    }
  };

  // Close dropdown when clicking outside
  useEffect(() => {
    const handleClickOutside = (e: MouseEvent) => {
      if (
        dropdownRef.current &&
        !dropdownRef.current.contains(e.target as Node) &&
        inputRef.current &&
        !inputRef.current.contains(e.target as Node)
      ) {
        setShowDropdown(false);
      }
    };

    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, []);

  return (
    <div className={cn('relative', className)} data-testid="quick-search-bar">
      <form onSubmit={handleSubmit} className="relative">
        <svg
          className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-[var(--text-secondary)]"
          width="16"
          height="16"
          fill="none"
          viewBox="0 0 24 24"
          stroke="currentColor"
          data-testid="search-icon"
        >
          <path
            strokeLinecap="round"
            strokeLinejoin="round"
            strokeWidth={2}
            d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"
          />
        </svg>
        <input
          ref={inputRef}
          type="text"
          value={query}
          onChange={handleInputChange}
          onFocus={handleInputFocus}
          onKeyDown={handleKeyDown}
          placeholder="搜尋媒體庫..."
          className="w-full rounded-lg border border-[var(--border-subtle)] bg-[var(--bg-secondary)] py-2.5 pl-10 pr-4 text-sm text-[var(--text-primary)] placeholder-[var(--text-muted)] transition-colors focus:border-[var(--accent-primary)] focus:outline-none focus:ring-1 focus:ring-[var(--accent-primary)]"
          autoComplete="off"
        />
      </form>

      {/* Recent Searches Dropdown */}
      {showDropdown && recentSearches.length > 0 && (
        <div
          ref={dropdownRef}
          className="absolute left-0 right-0 top-full z-10 mt-1 overflow-hidden rounded-lg border border-[var(--border-subtle)] bg-[var(--bg-secondary)] shadow-lg"
          data-testid="recent-searches-dropdown"
          role="listbox"
        >
          <div className="border-b border-[var(--border-subtle)] px-3 py-2">
            <span className="text-xs font-medium text-[var(--text-secondary)]">最近搜尋</span>
          </div>
          <ul>
            {recentSearches.map((search, index) => (
              <li key={search}>
                <button
                  type="button"
                  onClick={() => handleSelectRecent(search)}
                  onMouseEnter={() => setFocusedIndex(index)}
                  className={cn(
                    'flex w-full items-center gap-2 px-3 py-2 text-left text-sm text-[var(--text-primary)] transition-colors',
                    focusedIndex === index
                      ? 'bg-[var(--bg-tertiary)]'
                      : 'hover:bg-[var(--bg-tertiary)]/50'
                  )}
                  role="option"
                  aria-selected={focusedIndex === index}
                >
                  <Clock className="h-3.5 w-3.5 shrink-0 text-[var(--text-muted)]" />
                  <span className="flex-1 truncate">{search}</span>
                  <button
                    type="button"
                    onClick={(e) => removeRecentSearch(search, e)}
                    className="shrink-0 rounded p-0.5 text-[var(--text-muted)] transition-colors hover:bg-[var(--bg-tertiary)] hover:text-[var(--text-secondary)]"
                    aria-label={`移除 ${search}`}
                  >
                    <X className="h-3.5 w-3.5" />
                  </button>
                </button>
              </li>
            ))}
          </ul>
        </div>
      )}
    </div>
  );
}
