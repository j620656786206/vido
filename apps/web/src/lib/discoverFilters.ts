// Shared filter model for the discover/browse flow (Story 11-2).
// Maps the URL search params (Task 1.2) to a structured filter object and
// to the Story 11-1 TMDb discover backend query params.
//
// URL param  ↔  backend query param (Story 11-1 parseDiscoverParams):
//   genre       → genre            (CSV of TMDb genre IDs)
//   year_gte    → year_gte
//   year_lte    → year_lte
//   region      → region
//   rating_gte  → vote_gte         (TMDb 0-10 rating)
//   platform    → watch_providers  (+ watch_region, defaulted from region/TW)
//   sort_by     → sort             (mapped to a TMDb-native sort key per media type)
import { GENRE_MAP } from './genres';

export type DiscoverMediaType = 'all' | 'movie' | 'tv';

/** Abstract sort keys exposed in the URL/UI; mapped to TMDb-native keys per media type. */
export type SortKey = 'popularity' | 'date' | 'rating';

export interface DiscoverFilters {
  /** Selected TMDb genre IDs (AND-combined upstream). */
  genre: number[];
  /** Minimum release/first-air year. */
  yearGte?: number;
  /** Maximum release/first-air year. */
  yearLte?: number;
  /** ISO 3166-1 region code (TW/JP/KR/US/CN). */
  region?: string;
  /** Minimum TMDb rating, 0-10 → backend vote_gte. */
  ratingGte?: number;
  /** Selected TMDb watch-provider IDs → backend watch_providers. */
  platform: number[];
  sortBy: SortKey;
}

/** Shape of the `/discover` route search params (snake_case to mirror the backend). */
export interface DiscoverSearch {
  genre?: string;
  year_gte?: number;
  year_lte?: number;
  region?: string;
  rating_gte?: number;
  platform?: string;
  sort_by?: SortKey;
}

export const DEFAULT_FILTERS: DiscoverFilters = {
  genre: [],
  platform: [],
  sortBy: 'popularity',
};

/** Genre quick-select options (zh-TW labels from the shared GENRE_MAP). */
export const GENRE_FILTER_OPTIONS: { id: number; label: string }[] = [
  28, 16, 18, 878, 12, 35, 80, 14, 27, 53, 10749, 9648, 99, 10751, 36, 10402, 10752, 37,
].map((id) => ({ id, label: GENRE_MAP[id] }));

/** Region quick-select (Task 3.4). */
export const REGION_OPTIONS: { code: string; label: string; flag: string }[] = [
  { code: 'TW', label: '台灣', flag: '🇹🇼' },
  { code: 'JP', label: '日本', flag: '🇯🇵' },
  { code: 'KR', label: '韓國', flag: '🇰🇷' },
  { code: 'US', label: '美國', flag: '🇺🇸' },
  { code: 'CN', label: '中國', flag: '🇨🇳' },
];

// Platform watch-provider IDs (Task 3.6). TMDb provider IDs are stable for the
// majors; Story 11-1 CR M1 removed the backend TWWatchProviderIDs map, so the
// display→ID mapping now lives here on the frontend. Netflix=8 / Disney+=337 are
// confident; KKTV=425 per TMDb's TW provider catalog.
export const PLATFORM_OPTIONS: { id: number; label: string }[] = [
  { id: 8, label: 'Netflix' },
  { id: 337, label: 'Disney+' },
  { id: 425, label: 'KKTV' },
];

/** Minimum-rating quick chips (Task 3.5) — ★6+ / ★7+ / ★8+ / ★9+. */
export const RATING_OPTIONS: number[] = [6, 7, 8, 9];

export const SORT_OPTIONS: { value: SortKey; label: string }[] = [
  { value: 'popularity', label: '熱門' },
  { value: 'date', label: '上映日期' },
  { value: 'rating', label: '評分' },
];

function parseCsvInts(raw?: string): number[] {
  if (!raw) return [];
  return raw
    .split(',')
    .map((x) => parseInt(x, 10))
    .filter((n) => Number.isFinite(n));
}

/** Parse the URL search object into a structured filter object. */
export function parseFiltersFromSearch(search: DiscoverSearch): DiscoverFilters {
  return {
    genre: parseCsvInts(search.genre),
    yearGte: search.year_gte,
    yearLte: search.year_lte,
    region: search.region || undefined,
    ratingGte: search.rating_gte,
    platform: parseCsvInts(search.platform),
    sortBy: search.sort_by ?? 'popularity',
  };
}

/**
 * Serialize structured filters back to URL search params. Empty/default values
 * become `undefined` so TanStack Router drops them from the URL (keeps it clean
 * and keeps back/forward history meaningful — AC #4).
 */
export function serializeFilters(filters: DiscoverFilters): DiscoverSearch {
  return {
    genre: filters.genre.length ? filters.genre.join(',') : undefined,
    year_gte: filters.yearGte,
    year_lte: filters.yearLte,
    region: filters.region || undefined,
    rating_gte: filters.ratingGte,
    platform: filters.platform.length ? filters.platform.join(',') : undefined,
    sort_by: filters.sortBy === 'popularity' ? undefined : filters.sortBy,
  };
}

export interface FilterChipDescriptor {
  /** Stable React key. */
  key: string;
  /** Display label, e.g. "類型: 動畫". */
  label: string;
  /** The filter object after this chip is removed. */
  next: DiscoverFilters;
}

/** Build the removable-chip descriptors for the currently-active filters. */
export function activeFilterChips(filters: DiscoverFilters): FilterChipDescriptor[] {
  const chips: FilterChipDescriptor[] = [];

  for (const id of filters.genre) {
    chips.push({
      key: `genre-${id}`,
      label: `類型: ${GENRE_MAP[id] ?? id}`,
      next: { ...filters, genre: filters.genre.filter((g) => g !== id) },
    });
  }

  if (filters.yearGte !== undefined || filters.yearLte !== undefined) {
    let label: string;
    if (filters.yearGte !== undefined && filters.yearLte !== undefined) {
      label = `年份: ${filters.yearGte}-${filters.yearLte}`;
    } else if (filters.yearGte !== undefined) {
      label = `年份: ${filters.yearGte} 起`;
    } else {
      label = `年份: 至 ${filters.yearLte}`;
    }
    chips.push({
      key: 'year',
      label,
      next: { ...filters, yearGte: undefined, yearLte: undefined },
    });
  }

  if (filters.region) {
    const region = REGION_OPTIONS.find((o) => o.code === filters.region);
    chips.push({
      key: 'region',
      label: `地區: ${region ? region.label : filters.region}`,
      next: { ...filters, region: undefined },
    });
  }

  if (filters.ratingGte !== undefined) {
    chips.push({
      key: 'rating',
      label: `評分: ${filters.ratingGte}+`,
      next: { ...filters, ratingGte: undefined },
    });
  }

  for (const id of filters.platform) {
    const platform = PLATFORM_OPTIONS.find((o) => o.id === id);
    chips.push({
      key: `platform-${id}`,
      label: `平台: ${platform ? platform.label : id}`,
      next: { ...filters, platform: filters.platform.filter((p) => p !== id) },
    });
  }

  return chips;
}

export function countActiveFilters(filters: DiscoverFilters): number {
  return activeFilterChips(filters).length;
}

export function hasActiveFilters(filters: DiscoverFilters): boolean {
  return countActiveFilters(filters) > 0;
}

/** Map an abstract sort key to a TMDb-native sort string for the given media type. */
export function sortKeyToTmdb(sort: SortKey, mediaType: 'movie' | 'tv'): string {
  switch (sort) {
    case 'rating':
      return 'vote_average.desc';
    case 'date':
      return mediaType === 'movie' ? 'primary_release_date.desc' : 'first_air_date.desc';
    case 'popularity':
    default:
      return 'popularity.desc';
  }
}

/** Build the Story 11-1 TMDb discover backend query params from structured filters. */
export function buildDiscoverParams(
  filters: DiscoverFilters,
  mediaType: 'movie' | 'tv',
  page = 1
): URLSearchParams {
  const params = new URLSearchParams();
  if (filters.genre.length) params.set('genre', filters.genre.join(','));
  if (filters.yearGte !== undefined) params.set('year_gte', String(filters.yearGte));
  if (filters.yearLte !== undefined) params.set('year_lte', String(filters.yearLte));
  if (filters.region) params.set('region', filters.region);
  if (filters.ratingGte !== undefined) params.set('vote_gte', String(filters.ratingGte));
  if (filters.platform.length) {
    params.set('watch_providers', filters.platform.join(','));
    // Watch-provider filtering requires a watch region; default to the selected
    // region, falling back to TW (matches the backend's own region→TW fallback).
    params.set('watch_region', filters.region || 'TW');
  }
  params.set('sort', sortKeyToTmdb(filters.sortBy, mediaType));
  params.set('page', String(page));
  return params;
}
