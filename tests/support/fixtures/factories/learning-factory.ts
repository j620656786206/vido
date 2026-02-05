/**
 * Learning Factory for Test Data Generation
 *
 * Creates filename mapping pattern data for testing the learning system.
 * Auto-cleanup handled by test afterEach hooks.
 *
 * Pattern: Factory with overrides
 * @tags @api @learning @story-3-9
 */

import { faker } from '@faker-js/faker';

// =============================================================================
// Types
// =============================================================================

export interface CreatePatternRequestData {
  filename: string;
  metadataId: string;
  metadataType: 'movie' | 'series';
  tmdbId?: number;
}

export interface LearnedPatternData {
  id: string;
  pattern: string;
  patternType: string;
  patternRegex?: string;
  fansubGroup?: string;
  titlePattern?: string;
  metadataType: string;
  metadataId: string;
  tmdbId?: number;
  confidence: number;
  useCount: number;
  createdAt: string;
  lastUsedAt?: string;
}

export interface PatternStatsData {
  totalPatterns: number;
  totalApplied: number;
  mostUsedPattern?: string;
  mostUsedCount?: number;
}

export interface PatternListResponseData {
  patterns: LearnedPatternData[];
  totalCount: number;
  stats?: PatternStatsData;
}

export type PartialCreatePatternRequest = Partial<CreatePatternRequestData>;

// =============================================================================
// Factory Implementation
// =============================================================================

let patternCounter = 0;

/**
 * Generate a create pattern request with fansub-style filename
 *
 * @example
 * const request = createLearnPatternRequest();
 * const custom = createLearnPatternRequest({ metadataType: 'series' });
 */
export function createLearnPatternRequest(
  overrides: PartialCreatePatternRequest = {}
): CreatePatternRequestData {
  patternCounter++;
  const fansubGroup = faker.helpers.arrayElement([
    'Leopard-Raws',
    'SubsPlease',
    'Erai-raws',
    'HorribleSubs',
    'Commie',
  ]);
  const title = `Test Anime ${patternCounter}`;
  const episode = faker.number.int({ min: 1, max: 99 });

  return {
    filename: `[${fansubGroup}] ${title} - ${String(episode).padStart(2, '0')} (BD 1920x1080 x264 FLAC).mkv`,
    metadataId: `series-${faker.string.alphanumeric(8)}`,
    metadataType: 'series',
    tmdbId: faker.number.int({ min: 10000, max: 999999 }),
    ...overrides,
  };
}

/**
 * Generate a create pattern request with standard movie filename
 *
 * @example
 * const request = createMoviePatternRequest();
 */
export function createMoviePatternRequest(
  overrides: PartialCreatePatternRequest = {}
): CreatePatternRequestData {
  patternCounter++;

  return {
    filename: `Test.Movie.${patternCounter}.2024.1080p.BluRay.x264-GROUP.mkv`,
    metadataId: `movie-${faker.string.alphanumeric(8)}`,
    metadataType: 'movie',
    tmdbId: faker.number.int({ min: 10000, max: 999999 }),
    ...overrides,
  };
}

/**
 * Reset the counter (useful between test files)
 */
export function resetLearningFactory(): void {
  patternCounter = 0;
}

// =============================================================================
// Preset Data for Common Scenarios
// =============================================================================

export const presetPatternRequests = {
  fansubSeries: createLearnPatternRequest({
    filename: '[Leopard-Raws] Kimetsu no Yaiba - 26 (BD 1920x1080 x264 FLAC).mkv',
    metadataId: 'series-kimetsu-001',
    metadataType: 'series',
    tmdbId: 85937,
  }),

  fansubSeriesEpisode2: createLearnPatternRequest({
    filename: '[Leopard-Raws] Kimetsu no Yaiba - 27 (BD 1920x1080 x264 FLAC).mkv',
    metadataId: 'series-kimetsu-001',
    metadataType: 'series',
    tmdbId: 85937,
  }),

  standardMovie: createMoviePatternRequest({
    filename: 'Inception.2010.1080p.BluRay.x264-SPARKS.mkv',
    metadataId: 'movie-inception-001',
    metadataType: 'movie',
    tmdbId: 27205,
  }),

  chineseBrackets: createLearnPatternRequest({
    filename: '【字幕組】進擊的巨人 第01話.mp4',
    metadataId: 'series-aot-001',
    metadataType: 'series',
    tmdbId: 1429,
  }),
} as const;
