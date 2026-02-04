/**
 * Series Factory for Test Data Generation
 *
 * Creates realistic TV series data for testing.
 * Auto-cleanup handled by fixture teardown.
 *
 * Pattern: Factory with overrides
 * @see https://playwright.dev/docs/test-fixtures
 */

import { faker } from '@faker-js/faker';

// =============================================================================
// Types
// =============================================================================

export interface SeriesData {
  id: string;
  tmdbId: number;
  title: string;
  originalTitle: string;
  overview: string;
  firstAirDate: string;
  lastAirDate?: string;
  posterPath: string;
  backdropPath: string;
  genres: string[];
  rating: number;
  numberOfSeasons: number;
  numberOfEpisodes: number;
  status: 'Returning Series' | 'Ended' | 'Canceled' | 'In Production';
  inProduction: boolean;
}

export type PartialSeriesData = Partial<SeriesData>;

// =============================================================================
// Factory Implementation
// =============================================================================

let seriesCounter = 0;

/**
 * Generate unique series data with optional overrides
 *
 * @example
 * const series = createSeriesData();
 * const customSeries = createSeriesData({ title: 'Custom Title', numberOfSeasons: 5 });
 */
export function createSeriesData(overrides: PartialSeriesData = {}): SeriesData {
  seriesCounter++;
  const uniqueId = `test-series-${seriesCounter}-${Date.now()}`;
  const firstYear = faker.number.int({ min: 2010, max: 2023 });
  const isEnded = faker.datatype.boolean();

  return {
    id: uniqueId,
    tmdbId: 200000 + seriesCounter,
    title: `測試影集 ${seriesCounter}`,
    originalTitle: `Test Series ${seriesCounter}`,
    overview: `這是一部用於測試的影集，編號 ${seriesCounter}。包含多季內容和精彩劇情。`,
    firstAirDate: `${firstYear}-01-15`,
    lastAirDate: isEnded ? `${firstYear + faker.number.int({ min: 1, max: 5 })}-12-20` : undefined,
    posterPath: `/posters/${uniqueId}.jpg`,
    backdropPath: `/backdrops/${uniqueId}.jpg`,
    genres: ['劇情', '科幻'],
    rating: faker.number.float({ min: 6.0, max: 9.5, fractionDigits: 1 }),
    numberOfSeasons: faker.number.int({ min: 1, max: 8 }),
    numberOfEpisodes: faker.number.int({ min: 10, max: 100 }),
    status: isEnded ? 'Ended' : 'Returning Series',
    inProduction: !isEnded,
    ...overrides,
  };
}

/**
 * Generate multiple series for list testing
 *
 * @example
 * const seriesList = createSeriesList(5);
 * const customSeriesList = createSeriesList(3, { genres: ['喜劇'] });
 */
export function createSeriesList(count: number, overrides: PartialSeriesData = {}): SeriesData[] {
  return Array.from({ length: count }, () => createSeriesData(overrides));
}

/**
 * Reset the counter (useful between test files)
 */
export function resetSeriesFactory(): void {
  seriesCounter = 0;
}

// =============================================================================
// Preset Series for Common Scenarios
// =============================================================================

export const presetSeries = {
  breakingBad: createSeriesData({
    tmdbId: 1396,
    title: '乖離毒師',
    originalTitle: 'Breaking Bad',
    firstAirDate: '2008-01-20',
    lastAirDate: '2013-09-29',
    rating: 9.5,
    numberOfSeasons: 5,
    numberOfEpisodes: 62,
    status: 'Ended',
    inProduction: false,
    genres: ['劇情', '犯罪', '驚悚'],
  }),

  gameOfThrones: createSeriesData({
    tmdbId: 1399,
    title: '權力遊戲',
    originalTitle: 'Game of Thrones',
    firstAirDate: '2011-04-17',
    lastAirDate: '2019-05-19',
    rating: 9.3,
    numberOfSeasons: 8,
    numberOfEpisodes: 73,
    status: 'Ended',
    inProduction: false,
    genres: ['奇幻', '劇情', '冒險'],
  }),

  strangerThings: createSeriesData({
    tmdbId: 66732,
    title: '怪奇物語',
    originalTitle: 'Stranger Things',
    firstAirDate: '2016-07-15',
    rating: 8.6,
    numberOfSeasons: 4,
    numberOfEpisodes: 34,
    status: 'Returning Series',
    inProduction: true,
    genres: ['科幻', '恐怖', '劇情'],
  }),
} as const;
