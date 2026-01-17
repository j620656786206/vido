/**
 * Parser Factory for Test Data Generation
 *
 * Creates sample filenames for parser testing.
 * Covers various formats: movies, TV shows, fansubs, edge cases.
 *
 * Pattern: Factory with overrides
 * @see https://playwright.dev/docs/test-fixtures
 */

// =============================================================================
// Types
// =============================================================================

export interface ParseTestCase {
  filename: string;
  expectedType: 'movie' | 'tv' | 'unknown';
  expectedStatus: 'success' | 'needs_ai' | 'failed';
  expectedTitle?: string;
  expectedYear?: number;
  expectedSeason?: number;
  expectedEpisode?: number;
  expectedQuality?: string;
}

// =============================================================================
// Sample Filenames by Category
// =============================================================================

/**
 * Standard movie filenames (scene releases)
 */
export const movieFilenames = {
  standard: 'Inception.2010.1080p.BluRay.x264-SPARKS.mkv',
  withYear: 'The.Dark.Knight.2008.2160p.UHD.BluRay.mkv',
  chinese: '全面啟動.Inception.2010.1080p.BluRay.mkv',
  multiLang: 'Interstellar.2014.1080p.BluRay.DTS.x264.mkv',
  simple: 'Avatar.2009.mkv',
  hdr: 'Dune.2021.2160p.WEB-DL.DDP5.1.Atmos.DV.HDR.H.265-FLUX.mkv',
  remux: 'The.Matrix.1999.2160p.UHD.BluRay.REMUX.HDR.HEVC.Atmos-EPSiLON.mkv',
};

/**
 * Standard TV show filenames
 */
export const tvFilenames = {
  standard: 'Breaking.Bad.S01E01.720p.BluRay.x264.mkv',
  fullSeason: 'Game.of.Thrones.S08E06.The.Iron.Throne.1080p.WEB-DL.mkv',
  multiEpisode: 'Friends.S01E01-E03.DVDRip.mkv',
  chinese: '權力遊戲.Game.of.Thrones.S01E01.mkv',
  anime: 'Attack.on.Titan.S04E28.1080p.WEB-DL.mkv',
  dailyShow: 'The.Daily.Show.2024.01.15.720p.WEB.h264.mkv',
};

/**
 * Fansub filenames (complex, may need AI)
 */
export const fansubFilenames = {
  brackets: '[SubGroup] Anime Name - 01 [1080p].mkv',
  chinese: '[字幕組] 某動漫 第01話.mp4',
  japanese: '[SubTeam] アニメ名 - 第01話 [720p].mkv',
  withTags: '[SubGroup] Show Name - 01 (BDRip 1920x1080 HEVC FLAC).mkv',
  complex: '[Multiple-Subs][Show.Name][01][HEVC-10bit-1080p][AAC].mkv',
};

/**
 * Edge case filenames
 */
export const edgeCaseFilenames = {
  noYear: 'Random.Movie.1080p.mkv',
  noQuality: 'Some.Movie.2023.mkv',
  specialChars: 'Movie: The Sequel? (2023) [1080p].mkv',
  unicode: '電影名稱.2024.1080p.mkv',
  veryLong:
    'This.Is.A.Very.Long.Movie.Title.That.Goes.On.And.On.2023.1080p.BluRay.x264-GROUPNAME.mkv',
  minimalInfo: 'movie.mkv',
  empty: '',
};

// =============================================================================
// Factory Functions
// =============================================================================

/**
 * Create a test case for parser testing
 */
export function createParseTestCase(overrides: Partial<ParseTestCase> = {}): ParseTestCase {
  return {
    filename: movieFilenames.standard,
    expectedType: 'movie',
    expectedStatus: 'success',
    expectedTitle: 'Inception',
    expectedYear: 2010,
    expectedQuality: '1080p',
    ...overrides,
  };
}

/**
 * Create multiple test cases for batch testing
 */
export function createBatchTestCases(count: number): ParseTestCase[] {
  const allFilenames = [
    ...Object.values(movieFilenames),
    ...Object.values(tvFilenames),
    ...Object.values(fansubFilenames),
  ];

  return Array.from({ length: count }, (_, i) => ({
    filename: allFilenames[i % allFilenames.length],
    expectedType: i < Object.keys(movieFilenames).length ? 'movie' : 'tv',
    expectedStatus: 'success' as const,
  }));
}

/**
 * Get all movie filenames as array
 */
export function getAllMovieFilenames(): string[] {
  return Object.values(movieFilenames);
}

/**
 * Get all TV filenames as array
 */
export function getAllTVFilenames(): string[] {
  return Object.values(tvFilenames);
}

/**
 * Get all fansub filenames as array
 */
export function getAllFansubFilenames(): string[] {
  return Object.values(fansubFilenames);
}

// =============================================================================
// Preset Test Cases
// =============================================================================

export const presetTestCases = {
  movieSuccess: createParseTestCase({
    filename: movieFilenames.standard,
    expectedType: 'movie',
    expectedStatus: 'success',
    expectedTitle: 'Inception',
    expectedYear: 2010,
    expectedQuality: '1080p',
  }),

  tvSuccess: createParseTestCase({
    filename: tvFilenames.standard,
    expectedType: 'tv',
    expectedStatus: 'success',
    expectedTitle: 'Breaking Bad',
    expectedSeason: 1,
    expectedEpisode: 1,
    expectedQuality: '720p',
  }),

  fansubNeedsAI: createParseTestCase({
    filename: fansubFilenames.chinese,
    expectedType: 'unknown',
    expectedStatus: 'needs_ai',
  }),

  edgeCaseNoYear: createParseTestCase({
    filename: edgeCaseFilenames.noYear,
    expectedType: 'movie',
    expectedStatus: 'success',
    expectedYear: undefined,
    expectedQuality: '1080p',
  }),
};
