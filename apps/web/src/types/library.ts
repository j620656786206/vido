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
  videoCodec?: string;
  videoResolution?: string;
  audioCodec?: string;
  audioChannels?: number;
  subtitleTracks?: string;
  hdrFormat?: string;
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
  videoCodec?: string;
  videoResolution?: string;
  audioCodec?: string;
  audioChannels?: number;
  subtitleTracks?: string;
  hdrFormat?: string;
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
