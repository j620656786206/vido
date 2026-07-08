import type { ProductionCountry, Credits, SpokenLanguage } from './tmdb';

export type SortField = 'title' | 'release_date' | 'rating' | 'created_at';
export type SortOrder = 'asc' | 'desc';

export const VALID_SORT_FIELDS: readonly SortField[] = [
  'title',
  'release_date',
  'rating',
  'created_at',
] as const;

export type LibraryMediaType = 'all' | 'movie' | 'tv';

export interface LibraryMovie {
  id: string;
  title: string;
  originalTitle?: string;
  releaseDate: string;
  genres: string[];
  rating?: number;
  voteAverage?: number;
  voteCount?: number;
  // Douban rating fields (Story 12-1) — populated lazily via the douban-rating endpoint.
  doubanId?: string;
  doubanRating?: number;
  doubanVoteCount?: number;
  overview?: string;
  posterPath?: string;
  backdropPath?: string;
  runtime?: number;
  originalLanguage?: string;
  status?: string;
  imdbId?: string;
  tmdbId?: number;
  filePath?: string;
  fileSize?: number;
  parseStatus: string;
  metadataSource?: string;
  // Subtitle engine result — exposed to the library list by ux3-0-1 (N1 badge source).
  subtitleStatus?: string;
  subtitleLanguage?: string;
  videoCodec?: string;
  videoResolution?: string;
  audioCodec?: string;
  audioChannels?: number;
  subtitleTracks?: string;
  hdrFormat?: string;
  // Production countries (§9b CN-subtitle policy source) — exposed by
  // disc-2026-07-production-countries-detail-api. NULL until re-scanned.
  productionCountries?: ProductionCountry[];
  // Persisted credits + spoken languages (disc-2026-07-credits-spoken-languages-persist).
  // `credits` is present only for manually-edited movies (Metadata Editor); the detail
  // page prefers it over live TMDb when metadataSource === 'manual'. `spokenLanguages`
  // is persist-only (no UI consumer yet). Both NULL until re-edited/re-scanned.
  credits?: Credits;
  spokenLanguages?: SpokenLanguage[];
  createdAt: string;
  updatedAt: string;
}

export interface TMDbVideo {
  id: string;
  key: string;
  name: string;
  site: string;
  type: string;
  official: boolean;
  publishedAt: string;
}

export interface VideosResponse {
  id: number;
  results: TMDbVideo[];
}

export interface LibrarySeries {
  id: string;
  title: string;
  originalTitle?: string;
  firstAirDate: string;
  lastAirDate?: string;
  genres: string[];
  rating?: number;
  voteAverage?: number;
  voteCount?: number;
  // Douban rating fields (Story 12-1) — populated lazily via the douban-rating endpoint.
  doubanId?: string;
  doubanRating?: number;
  doubanVoteCount?: number;
  overview?: string;
  posterPath?: string;
  backdropPath?: string;
  numberOfSeasons?: number;
  numberOfEpisodes?: number;
  status?: string;
  originalLanguage?: string;
  imdbId?: string;
  tmdbId?: number;
  inProduction?: boolean;
  filePath?: string;
  fileSize?: number;
  parseStatus: string;
  metadataSource?: string;
  // Subtitle engine result — exposed to the library list by ux3-0-1 (N1 badge source).
  subtitleStatus?: string;
  subtitleLanguage?: string;
  videoCodec?: string;
  videoResolution?: string;
  audioCodec?: string;
  audioChannels?: number;
  subtitleTracks?: string;
  hdrFormat?: string;
  // Persisted credits (disc-2026-07-credits-spoken-languages-persist). Present only for
  // manually-edited series; the detail page prefers it over live TMDb when
  // metadataSource === 'manual'. NULL until re-edited. Series has no spoken_languages.
  credits?: Credits;
  createdAt: string;
  updatedAt: string;
}

// Douban rating enrichment payload (Story 12-1). The endpoint returns
// `data: null` when no Douban rating is available (graceful degradation).
export interface DoubanRating {
  doubanId: string;
  doubanRating: number;
  doubanVoteCount: number;
}

export type DoubanRatingResponse = DoubanRating | null;

// A single Douban short comment (短評) — text is server-converted to Traditional
// Chinese (Story 12-6 AC #3). rating is a 0–5 star score (0 when unrated).
export interface ReviewComment {
  author: string;
  rating: number;
  text: string;
}

// Douban review-summary enrichment payload (Story 12-6). The endpoint returns
// `data: null` when no review summary is available (graceful degradation).
export interface DoubanReviewSummary {
  id: string;
  totalComments: number;
  topComments: ReviewComment[];
}

export type DoubanReviewSummaryResponse = DoubanReviewSummary | null;

// Season/episode accordion types (Story 12-2). Field names are camelCase —
// the API returns snake_case and libraryService transforms via snakeToCamel.
export interface SeasonSummary {
  id: number;
  seasonNumber: number;
  name?: string;
  overview?: string;
  posterPath?: string;
  airDate?: string;
  episodeCount?: number;
}

// MergedEpisode = TMDb episode metadata + local file/subtitle enrichment.
// Subtitle fields are only meaningful when hasLocalFile is true (AC #5/#6).
export interface MergedEpisode {
  episodeNumber: number;
  name: string;
  overview?: string;
  airDate?: string;
  runtime?: number;
  stillPath?: string;
  voteAverage?: number;
  hasLocalFile: boolean;
  subtitleStatus?: string;
  subtitleLanguage?: string;
  filePath?: string;
}

export interface SeasonEpisodesResponse {
  season: SeasonSummary;
  episodes: MergedEpisode[];
}

// Story 12-3 — related-content recommendations. A normalized tile shape shared
// across movie/TV recommendations (backend RecommendationItem; camelCased via
// snakeToCamel at the fetchApi boundary, Rule 18).
export interface RecommendationItem {
  id: number;
  mediaType: 'movie' | 'tv';
  title: string;
  posterPath: string | null;
  releaseDate?: string;
  voteAverage?: number;
  isOwned: boolean;
}

export interface RecommendationsResponse {
  results: RecommendationItem[];
  /** Which TMDB endpoint filled the list: "recommendations" | "similar" | "". */
  source: string;
}

// Story 12-4 — streaming-platform availability (TMDB watch providers, sourced
// from JustWatch). camelCased via snakeToCamel at the fetchApi boundary (Rule 18).
// NOTE: `results` is keyed by ISO 3166-1 region code (e.g. "TW", "US"). Those
// keys are uppercase with no underscores, so snakeToCamel's `_([a-z])` rewrite
// leaves them untouched — `results` stays a faithful Record<string, …>.
export interface WatchProvider {
  providerId: number;
  providerName: string;
  logoPath: string | null;
  displayPriority: number;
}

export interface WatchProviderRegion {
  link: string;
  flatrate?: WatchProvider[];
  rent?: WatchProvider[];
  buy?: WatchProvider[];
}

export interface WatchProvidersResponse {
  id: number;
  results: Record<string, WatchProviderRegion>;
}

export interface LibraryItem {
  type: 'movie' | 'series';
  movie?: LibraryMovie;
  series?: LibrarySeries;
}

export interface LibraryListResponse {
  items: LibraryItem[];
  page: number;
  pageSize: number;
  totalItems: number;
  totalPages: number;
}

export interface LibraryListParams {
  page?: number;
  pageSize?: number;
  type?: LibraryMediaType;
  sortBy?: string;
  sortOrder?: 'asc' | 'desc';
  genres?: string;
  yearMin?: number;
  yearMax?: number;
  unmatched?: boolean;
}

export interface MediaStats {
  total: number;
  unmatchedCount: number;
}

export interface LibraryStats {
  yearMin: number;
  yearMax: number;
  movieCount: number;
  tvCount: number;
  totalCount: number;
}

export interface LibrarySearchResult {
  type: 'movie' | 'series';
  movie?: LibraryMovie;
  series?: LibrarySeries;
}

export interface LibrarySearchResponse {
  results: LibrarySearchResult[];
  totalCount: number;
}

export interface BatchResult {
  successCount: number;
  failedCount: number;
  errors?: BatchError[];
}

export interface BatchError {
  id: string;
  message: string;
}
