/**
 * Metadata Factory for Test Data Generation
 *
 * Creates metadata-related data for testing manual search and metadata editor.
 * Auto-cleanup handled by fixture teardown.
 *
 * Pattern: Factory with overrides
 * @see https://playwright.dev/docs/test-fixtures
 */

import { faker } from '@faker-js/faker';

// =============================================================================
// Types
// =============================================================================

export interface ManualSearchRequestData {
  query: string;
  mediaType?: 'movie' | 'tv';
  year?: number;
  source?: 'all' | 'tmdb' | 'douban' | 'wikipedia';
}

export interface ManualSearchResultItemData {
  id: string;
  source: 'tmdb' | 'douban' | 'wikipedia';
  title: string;
  titleZhTW?: string;
  year: number;
  mediaType: 'movie' | 'tv';
  overview?: string;
  posterUrl?: string;
  rating?: number;
}

export interface UpdateMetadataRequestData {
  mediaType?: 'movie' | 'series';
  title: string;
  titleEnglish?: string;
  year: number;
  genres?: string[];
  director?: string;
  cast?: string[];
  overview?: string;
  posterUrl?: string;
}

export type PartialManualSearchRequest = Partial<ManualSearchRequestData>;
export type PartialManualSearchResultItem = Partial<ManualSearchResultItemData>;
export type PartialUpdateMetadataRequest = Partial<UpdateMetadataRequestData>;

// =============================================================================
// Factory Implementation
// =============================================================================

let searchCounter = 0;
let resultCounter = 0;

/**
 * Generate manual search request data
 *
 * @example
 * const request = createManualSearchRequest();
 * const customRequest = createManualSearchRequest({ query: 'Inception', year: 2010 });
 */
export function createManualSearchRequest(
  overrides: PartialManualSearchRequest = {}
): ManualSearchRequestData {
  searchCounter++;

  return {
    query: `Test Movie ${searchCounter}`,
    mediaType: faker.helpers.arrayElement(['movie', 'tv']),
    year: faker.number.int({ min: 2000, max: 2024 }),
    source: faker.helpers.arrayElement(['all', 'tmdb', 'douban', 'wikipedia']),
    ...overrides,
  };
}

/**
 * Generate manual search result item data
 *
 * @example
 * const result = createManualSearchResultItem();
 * const customResult = createManualSearchResultItem({ source: 'tmdb', title: 'Inception' });
 */
export function createManualSearchResultItem(
  overrides: PartialManualSearchResultItem = {}
): ManualSearchResultItemData {
  resultCounter++;
  const source = overrides.source || faker.helpers.arrayElement(['tmdb', 'douban', 'wikipedia']);

  return {
    id: `${source}-${resultCounter}-${Date.now()}`,
    source,
    title: `Test Result ${resultCounter}`,
    titleZhTW: `測試結果 ${resultCounter}`,
    year: faker.number.int({ min: 2000, max: 2024 }),
    mediaType: faker.helpers.arrayElement(['movie', 'tv']),
    overview: faker.lorem.paragraph(),
    posterUrl: `https://example.com/posters/${resultCounter}.jpg`,
    rating: faker.number.float({ min: 5.0, max: 10.0, fractionDigits: 1 }),
    ...overrides,
  };
}

/**
 * Generate multiple search result items
 *
 * @example
 * const results = createManualSearchResultList(5);
 */
export function createManualSearchResultList(
  count: number,
  overrides: PartialManualSearchResultItem = {}
): ManualSearchResultItemData[] {
  return Array.from({ length: count }, () => createManualSearchResultItem(overrides));
}

/**
 * Generate update metadata request data
 *
 * @example
 * const request = createUpdateMetadataRequest();
 * const customRequest = createUpdateMetadataRequest({ title: '新標題', year: 2024 });
 */
export function createUpdateMetadataRequest(
  overrides: PartialUpdateMetadataRequest = {}
): UpdateMetadataRequestData {
  return {
    mediaType: faker.helpers.arrayElement(['movie', 'series']),
    title: `測試電影 ${faker.number.int({ min: 1, max: 1000 })}`,
    titleEnglish: `Test Movie ${faker.number.int({ min: 1, max: 1000 })}`,
    year: faker.number.int({ min: 2000, max: 2024 }),
    genres: faker.helpers.arrayElements(['動作', '科幻', '劇情', '喜劇', '恐怖'], 2),
    director: faker.person.fullName(),
    cast: [faker.person.fullName(), faker.person.fullName(), faker.person.fullName()],
    overview: faker.lorem.paragraphs(2),
    posterUrl: `https://example.com/posters/${faker.string.alphanumeric(10)}.jpg`,
    ...overrides,
  };
}

/**
 * Reset the counters (useful between test files)
 */
export function resetMetadataFactory(): void {
  searchCounter = 0;
  resultCounter = 0;
}

// =============================================================================
// Preset Data for Common Scenarios
// =============================================================================

export const presetSearchRequests = {
  inceptionMovie: createManualSearchRequest({
    query: 'Inception',
    mediaType: 'movie',
    year: 2010,
    source: 'tmdb',
  }),

  chineseMovie: createManualSearchRequest({
    query: '全面啟動',
    mediaType: 'movie',
    source: 'all',
  }),

  tvShowSearch: createManualSearchRequest({
    query: 'Breaking Bad',
    mediaType: 'tv',
    source: 'tmdb',
  }),

  doubanSearch: createManualSearchRequest({
    query: '霸王别姬',
    mediaType: 'movie',
    source: 'douban',
  }),
} as const;

export const presetSearchResults = {
  inceptionResult: createManualSearchResultItem({
    id: 'tmdb-27205',
    source: 'tmdb',
    title: 'Inception',
    titleZhTW: '全面啟動',
    year: 2010,
    mediaType: 'movie',
    overview:
      'A thief who steals corporate secrets through the use of dream-sharing technology is given the inverse task of planting an idea into the mind of a C.E.O.',
    posterUrl: 'https://image.tmdb.org/t/p/w500/9gk7adHYeDvHkCSEqAvQNLV5Ber.jpg',
    rating: 8.4,
  }),

  darkKnightResult: createManualSearchResultItem({
    id: 'tmdb-155',
    source: 'tmdb',
    title: 'The Dark Knight',
    titleZhTW: '黑暗騎士',
    year: 2008,
    mediaType: 'movie',
    overview:
      'Batman raises the stakes in his war on crime and sets out to dismantle the remaining criminal organizations that plague the streets.',
    posterUrl: 'https://image.tmdb.org/t/p/w500/qJ2tW6WMUDux911r6m7haRef0WH.jpg',
    rating: 9.0,
  }),

  breakingBadResult: createManualSearchResultItem({
    id: 'tmdb-1396',
    source: 'tmdb',
    title: 'Breaking Bad',
    titleZhTW: '絕命毒師',
    year: 2008,
    mediaType: 'tv',
    overview:
      'A high school chemistry teacher diagnosed with inoperable lung cancer turns to manufacturing and selling methamphetamine.',
    posterUrl: 'https://image.tmdb.org/t/p/w500/ggFHVNu6YYI5L9pCfOacjizRGt.jpg',
    rating: 9.5,
  }),
} as const;

export const presetUpdateRequests = {
  movieUpdate: createUpdateMetadataRequest({
    mediaType: 'movie',
    title: '全面啟動',
    titleEnglish: 'Inception',
    year: 2010,
    genres: ['動作', '科幻', '冒險'],
    director: 'Christopher Nolan',
    cast: ['Leonardo DiCaprio', 'Joseph Gordon-Levitt', 'Elliot Page'],
    overview: '多姆科布是一名經驗豐富的盜夢者，他能潛入別人的夢境中，竊取他們最珍貴的秘密。',
  }),

  seriesUpdate: createUpdateMetadataRequest({
    mediaType: 'series',
    title: '絕命毒師',
    titleEnglish: 'Breaking Bad',
    year: 2008,
    genres: ['劇情', '犯罪', '驚悚'],
    director: 'Vince Gilligan',
    cast: ['Bryan Cranston', 'Aaron Paul', 'Anna Gunn'],
    overview: '一名高中化學老師在被診斷出肺癌後，開始製毒販毒，逐漸墜入黑暗世界。',
  }),
} as const;
