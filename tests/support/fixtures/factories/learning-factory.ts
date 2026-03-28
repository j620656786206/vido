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
  metadata_id: string;
  metadata_type: 'movie' | 'series';
  tmdb_id?: number;
}

export interface LearnedPatternData {
  id: string;
  pattern: string;
  pattern_type: string;
  pattern_regex?: string;
  fansub_group?: string;
  title_pattern?: string;
  metadata_type: string;
  metadata_id: string;
  tmdb_id?: number;
  confidence: number;
  use_count: number;
  created_at: string;
  last_used_at?: string;
}

export interface PatternStatsData {
  total_patterns: number;
  total_applied: number;
  most_used_pattern?: string;
  most_used_count?: number;
}

export interface PatternListResponseData {
  patterns: LearnedPatternData[];
  total_count: number;
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
 * const custom = createLearnPatternRequest({ metadata_type: 'series' });
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
    metadata_id: `series-${faker.string.alphanumeric(8)}`,
    metadata_type: 'series',
    tmdb_id: faker.number.int({ min: 10000, max: 999999 }),
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
    metadata_id: `movie-${faker.string.alphanumeric(8)}`,
    metadata_type: 'movie',
    tmdb_id: faker.number.int({ min: 10000, max: 999999 }),
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
    metadata_id: 'series-kimetsu-001',
    metadata_type: 'series',
    tmdb_id: 85937,
  }),

  fansubSeriesEpisode2: createLearnPatternRequest({
    filename: '[Leopard-Raws] Kimetsu no Yaiba - 27 (BD 1920x1080 x264 FLAC).mkv',
    metadata_id: 'series-kimetsu-001',
    metadata_type: 'series',
    tmdb_id: 85937,
  }),

  standardMovie: createMoviePatternRequest({
    filename: 'Inception.2010.1080p.BluRay.x264-SPARKS.mkv',
    metadata_id: 'movie-inception-001',
    metadata_type: 'movie',
    tmdb_id: 27205,
  }),

  chineseBrackets: createLearnPatternRequest({
    filename: '【字幕組】進擊的巨人 第01話.mp4',
    metadata_id: 'series-aot-001',
    metadata_type: 'series',
    tmdb_id: 1429,
  }),
} as const;
