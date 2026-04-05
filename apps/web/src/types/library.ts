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
