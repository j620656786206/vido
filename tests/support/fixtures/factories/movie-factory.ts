/**
 * Movie Factory for Test Data Generation
 *
 * Creates realistic movie data for testing.
 * Auto-cleanup handled by fixture teardown.
 *
 * Pattern: Factory with overrides
 * @see https://playwright.dev/docs/test-fixtures
 */

// =============================================================================
// Types
// =============================================================================

export interface MovieData {
  id: string;
  tmdb_id: number;
  title: string;
  original_title: string;
  overview: string;
  release_date: string;
  poster_path: string;
  backdrop_path: string;
  vote_average: number;
  vote_count: number;
  popularity: number;
  genres: string[];
}

export type PartialMovieData = Partial<MovieData>;

// =============================================================================
// Factory Implementation
// =============================================================================

let movieCounter = 0;

/**
 * Generate unique movie data with optional overrides
 *
 * @example
 * const movie = createMovieData();
 * const customMovie = createMovieData({ title: 'Custom Title' });
 */
export function createMovieData(overrides: PartialMovieData = {}): MovieData {
  movieCounter++;
  const uniqueId = `test-movie-${movieCounter}-${Date.now()}`;

  return {
    id: uniqueId,
    tmdb_id: 100000 + movieCounter,
    title: `測試電影 ${movieCounter}`,
    original_title: `Test Movie ${movieCounter}`,
    overview: `這是一部用於測試的電影，編號 ${movieCounter}。包含各種測試場景所需的描述內容。`,
    release_date: '2024-01-15',
    poster_path: `/posters/${uniqueId}.jpg`,
    backdrop_path: `/backdrops/${uniqueId}.jpg`,
    vote_average: 7.5,
    vote_count: 1000,
    popularity: 50.0,
    genres: ['動作', '科幻'],
    ...overrides,
  };
}

/**
 * Generate multiple movies for list testing
 *
 * @example
 * const movies = createMovieList(5);
 * const customMovies = createMovieList(3, { genres: ['喜劇'] });
 */
export function createMovieList(count: number, overrides: PartialMovieData = {}): MovieData[] {
  return Array.from({ length: count }, () => createMovieData(overrides));
}

/**
 * Reset the counter (useful between test files)
 */
export function resetMovieFactory(): void {
  movieCounter = 0;
}

// =============================================================================
// Preset Movies for Common Scenarios
// =============================================================================

export const presetMovies = {
  inception: createMovieData({
    tmdb_id: 27205,
    title: '全面啟動',
    original_title: 'Inception',
    release_date: '2010-07-16',
    vote_average: 8.4,
    genres: ['動作', '科幻', '冒險'],
  }),

  darkKnight: createMovieData({
    tmdb_id: 155,
    title: '黑暗騎士',
    original_title: 'The Dark Knight',
    release_date: '2008-07-18',
    vote_average: 9.0,
    genres: ['動作', '犯罪', '劇情'],
  }),

  interstellar: createMovieData({
    tmdb_id: 157336,
    title: '星際效應',
    original_title: 'Interstellar',
    release_date: '2014-11-07',
    vote_average: 8.7,
    genres: ['冒險', '劇情', '科幻'],
  }),
} as const;
