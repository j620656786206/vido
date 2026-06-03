// Story 11-2 Task 1 — filter state synced with the URL search params.
// TanStack Router is the single source of truth (no Zustand/Redux — Rule 5),
// so back/forward navigation preserves filter state for free (AC #4).
import { useNavigate, useSearch } from '@tanstack/react-router';
import {
  parseFiltersFromSearch,
  serializeFilters,
  type DiscoverFilters,
  type DiscoverSearch,
} from '../lib/discoverFilters';

export interface UseFilterStateResult {
  filters: DiscoverFilters;
  /** Replace the whole filter object; resets pagination to page 1. */
  setFilters: (next: DiscoverFilters) => void;
  /** Clear every active filter (keeps the current sort). */
  clearAll: () => void;
}

/**
 * Reads/writes the discover filter state from the `/discover` route's URL search
 * params. Bound to the discover route via the non-strict `useSearch`, so it stays
 * unit-testable by mocking `@tanstack/react-router`.
 */
export function useFilterState(): UseFilterStateResult {
  const search = useSearch({ strict: false }) as DiscoverSearch;
  const navigate = useNavigate();

  const filters = parseFiltersFromSearch(search);

  const setFilters = (next: DiscoverFilters) => {
    navigate({
      to: '/discover',
      search: (prev: Record<string, unknown>) => ({
        ...prev,
        ...serializeFilters(next),
        // Any filter change resets to the first page of results.
        page: 1,
      }),
    });
  };

  const clearAll = () => {
    setFilters({ genre: [], platform: [], sortBy: filters.sortBy });
  };

  return { filters, setFilters, clearAll };
}
