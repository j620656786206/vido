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
  original_title?: string;
  release_date: string;
  genres: string[];
  rating?: number;
  vote_average?: number;
  overview?: string;
  poster_path?: string;
  backdrop_path?: string;
  runtime?: number;
  original_language?: string;
  status?: string;
  imdb_id?: string;
  tmdb_id?: number;
  file_path?: string;
  file_size?: number;
  parse_status: string;
  metadata_source?: string;
  created_at: string;
  updated_at: string;
}

export interface TMDbVideo {
  id: string;
  key: string;
  name: string;
  site: string;
  type: string;
  official: boolean;
  published_at: string;
}

export interface VideosResponse {
  id: number;
  results: TMDbVideo[];
}

export interface LibrarySeries {
  id: string;
  title: string;
  original_title?: string;
  first_air_date: string;
  last_air_date?: string;
  genres: string[];
  rating?: number;
  vote_average?: number;
  overview?: string;
  poster_path?: string;
  backdrop_path?: string;
  number_of_seasons?: number;
  number_of_episodes?: number;
  status?: string;
  original_language?: string;
  imdb_id?: string;
  tmdb_id?: number;
  in_production?: boolean;
  file_path?: string;
  file_size?: number;
  parse_status: string;
  metadata_source?: string;
  created_at: string;
  updated_at: string;
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
