import { renderHook } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { useFilterState } from './useFilterState';
import { useNavigate, useSearch } from '@tanstack/react-router';

vi.mock('@tanstack/react-router', () => ({
  useNavigate: vi.fn(),
  useSearch: vi.fn(),
}));

const mockedUseSearch = vi.mocked(useSearch);
const mockedUseNavigate = vi.mocked(useNavigate);

describe('useFilterState', () => {
  const navigate = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();
    mockedUseNavigate.mockReturnValue(navigate);
  });

  it('parses filters from the URL search params (AC #4)', () => {
    mockedUseSearch.mockReturnValue({ genre: '16', region: 'JP', rating_gte: 8 });
    const { result } = renderHook(() => useFilterState());
    expect(result.current.filters).toMatchObject({
      genre: [16],
      region: 'JP',
      ratingGte: 8,
      sortBy: 'popularity',
    });
  });

  it('setFilters navigates to /discover and resets the page to 1', () => {
    mockedUseSearch.mockReturnValue({});
    const { result } = renderHook(() => useFilterState());

    result.current.setFilters({ genre: [28], platform: [], sortBy: 'rating' });

    expect(navigate).toHaveBeenCalledTimes(1);
    const arg = navigate.mock.calls[0][0];
    expect(arg.to).toBe('/discover');
    // The search updater merges with prev and resets page.
    expect(arg.search({ page: 5, type: 'movie' })).toEqual({
      type: 'movie',
      genre: '28',
      year_gte: undefined,
      year_lte: undefined,
      region: undefined,
      rating_gte: undefined,
      platform: undefined,
      sort_by: 'rating',
      page: 1,
    });
  });

  it('clearAll keeps the current sort but drops every filter', () => {
    mockedUseSearch.mockReturnValue({ genre: '16,28', sort_by: 'date' });
    const { result } = renderHook(() => useFilterState());

    result.current.clearAll();

    const arg = navigate.mock.calls[0][0];
    const nextSearch = arg.search({});
    expect(nextSearch.genre).toBeUndefined();
    expect(nextSearch.sort_by).toBe('date');
    expect(nextSearch.page).toBe(1);
  });
});
