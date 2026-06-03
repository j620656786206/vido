import { describe, it, expect } from 'vitest';
import {
  parseFiltersFromSearch,
  serializeFilters,
  activeFilterChips,
  countActiveFilters,
  hasActiveFilters,
  sortKeyToTmdb,
  buildDiscoverParams,
  DEFAULT_FILTERS,
  type DiscoverFilters,
} from './discoverFilters';

describe('parseFiltersFromSearch', () => {
  it('parses CSV genre/platform ids and scalar params', () => {
    const filters = parseFiltersFromSearch({
      genre: '28,16',
      year_gte: 2020,
      year_lte: 2024,
      region: 'JP',
      rating_gte: 7,
      platform: '8,337',
      sort_by: 'rating',
    });
    expect(filters).toEqual({
      genre: [28, 16],
      yearGte: 2020,
      yearLte: 2024,
      region: 'JP',
      ratingGte: 7,
      platform: [8, 337],
      sortBy: 'rating',
    });
  });

  it('defaults to popularity sort and empty arrays for an empty search', () => {
    expect(parseFiltersFromSearch({})).toEqual(DEFAULT_FILTERS);
  });

  it('ignores non-numeric csv entries', () => {
    expect(parseFiltersFromSearch({ genre: '28,abc,16' }).genre).toEqual([28, 16]);
  });
});

describe('serializeFilters', () => {
  it('drops empty arrays and the default popularity sort for a clean URL', () => {
    expect(serializeFilters(DEFAULT_FILTERS)).toEqual({
      genre: undefined,
      year_gte: undefined,
      year_lte: undefined,
      region: undefined,
      rating_gte: undefined,
      platform: undefined,
      sort_by: undefined,
    });
  });

  it('round-trips a populated filter set', () => {
    const filters: DiscoverFilters = {
      genre: [28, 16],
      yearGte: 2020,
      yearLte: 2024,
      region: 'JP',
      ratingGte: 8,
      platform: [8],
      sortBy: 'date',
    };
    expect(parseFiltersFromSearch(serializeFilters(filters))).toEqual(filters);
  });
});

describe('activeFilterChips', () => {
  it('produces a zh-TW labelled chip per active filter', () => {
    const filters: DiscoverFilters = {
      genre: [16],
      yearGte: 2022,
      yearLte: 2024,
      region: 'JP',
      ratingGte: 7,
      platform: [8],
      sortBy: 'popularity',
    };
    const chips = activeFilterChips(filters);
    expect(chips.map((c) => c.label)).toEqual([
      '類型: 動畫',
      '年份: 2022-2024',
      '地區: 日本',
      '評分: 7+',
      '平台: Netflix',
    ]);
  });

  it('removing a chip clears only that filter', () => {
    const filters: DiscoverFilters = {
      genre: [16, 28],
      platform: [],
      sortBy: 'popularity',
    };
    const animeChip = activeFilterChips(filters).find((c) => c.key === 'genre-16');
    expect(animeChip?.next.genre).toEqual([28]);
  });

  it('renders an open-ended year chip when only the lower bound is set', () => {
    const chips = activeFilterChips({
      genre: [],
      platform: [],
      sortBy: 'popularity',
      yearGte: 2020,
    });
    expect(chips[0].label).toBe('年份: 2020 起');
  });

  it('does not treat sort as an active filter', () => {
    expect(hasActiveFilters({ genre: [], platform: [], sortBy: 'rating' })).toBe(false);
    expect(countActiveFilters(DEFAULT_FILTERS)).toBe(0);
  });
});

describe('sortKeyToTmdb', () => {
  it('maps abstract keys to TMDb-native keys per media type', () => {
    expect(sortKeyToTmdb('popularity', 'movie')).toBe('popularity.desc');
    expect(sortKeyToTmdb('rating', 'tv')).toBe('vote_average.desc');
    expect(sortKeyToTmdb('date', 'movie')).toBe('primary_release_date.desc');
    expect(sortKeyToTmdb('date', 'tv')).toBe('first_air_date.desc');
  });
});

describe('buildDiscoverParams', () => {
  it('maps frontend filters to the Story 11-1 backend query params', () => {
    const params = buildDiscoverParams(
      {
        genre: [28, 16],
        yearGte: 2020,
        yearLte: 2024,
        region: 'JP',
        ratingGte: 7,
        platform: [8, 337],
        sortBy: 'rating',
      },
      'movie',
      2
    );
    expect(params.get('genre')).toBe('28,16');
    expect(params.get('year_gte')).toBe('2020');
    expect(params.get('year_lte')).toBe('2024');
    expect(params.get('region')).toBe('JP');
    expect(params.get('vote_gte')).toBe('7');
    expect(params.get('watch_providers')).toBe('8,337');
    expect(params.get('watch_region')).toBe('JP');
    expect(params.get('sort')).toBe('vote_average.desc');
    expect(params.get('page')).toBe('2');
  });

  it('defaults watch_region to TW when a platform is set without a region', () => {
    const params = buildDiscoverParams({ genre: [], platform: [8], sortBy: 'popularity' }, 'tv');
    expect(params.get('watch_region')).toBe('TW');
  });

  it('omits empty dimensions', () => {
    const params = buildDiscoverParams(DEFAULT_FILTERS, 'movie');
    expect(params.get('genre')).toBeNull();
    expect(params.get('vote_gte')).toBeNull();
    expect(params.get('watch_providers')).toBeNull();
    expect(params.get('sort')).toBe('popularity.desc');
  });

  it('normalizes an inverted year range so gte <= lte (L1)', () => {
    const params = buildDiscoverParams(
      { genre: [], platform: [], sortBy: 'popularity', yearGte: 2024, yearLte: 2020 },
      'movie'
    );
    expect(params.get('year_gte')).toBe('2020');
    expect(params.get('year_lte')).toBe('2024');
  });
});
