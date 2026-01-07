/**
 * Core data models for the Vido media management application
 */

/**
 * Movie interface representing a single movie entity
 */
export interface Movie {
  /** Unique identifier for the movie */
  id: string;
  /** Movie title */
  title: string;
  /** Original title in the original language */
  originalTitle?: string;
  /** Release date in ISO 8601 format (YYYY-MM-DD) */
  releaseDate: string;
  /** Array of genre names */
  genres: string[];
  /** Average user rating (0-10 scale) */
  rating?: number;
  /** Overview/plot description */
  overview?: string;
  /** Relative path to poster image */
  posterPath?: string;
  /** Relative path to backdrop image */
  backdropPath?: string;
  /** Runtime in minutes */
  runtime?: number;
  /** Original language code (e.g., 'en', 'ja') */
  originalLanguage?: string;
  /** Current status (e.g., 'Released', 'In Production') */
  status?: string;
  /** IMDb ID */
  imdbId?: string;
  /** TMDb ID */
  tmdbId?: number;
}

/**
 * Series interface representing a TV series entity
 */
export interface Series {
  /** Unique identifier for the series */
  id: string;
  /** Series title */
  title: string;
  /** Original title in the original language */
  originalTitle?: string;
  /** First air date in ISO 8601 format (YYYY-MM-DD) */
  firstAirDate: string;
  /** Last air date in ISO 8601 format (YYYY-MM-DD) */
  lastAirDate?: string;
  /** Array of genre names */
  genres: string[];
  /** Average user rating (0-10 scale) */
  rating?: number;
  /** Overview/plot description */
  overview?: string;
  /** Relative path to poster image */
  posterPath?: string;
  /** Relative path to backdrop image */
  backdropPath?: string;
  /** Total number of seasons */
  numberOfSeasons?: number;
  /** Total number of episodes */
  numberOfEpisodes?: number;
  /** Current status (e.g., 'Returning Series', 'Ended', 'Canceled') */
  status?: string;
  /** Original language code (e.g., 'en', 'ja') */
  originalLanguage?: string;
  /** IMDb ID */
  imdbId?: string;
  /** TMDb ID */
  tmdbId?: number;
  /** Whether the series is still in production */
  inProduction?: boolean;
}

/**
 * Generic API response wrapper
 * @template T The type of data being returned
 */
export interface ApiResponse<T> {
  /** Indicates if the request was successful */
  success: boolean;
  /** The response data (only present on success) */
  data?: T;
  /** Error message (only present on failure) */
  error?: string;
  /** Additional message or context */
  message?: string;
}

/**
 * Paginated API response wrapper
 * @template T The type of items being returned
 */
export interface PaginatedApiResponse<T> extends ApiResponse<T[]> {
  /** Current page number (1-indexed) */
  page?: number;
  /** Total number of pages */
  totalPages?: number;
  /** Total number of items across all pages */
  totalResults?: number;
  /** Number of items per page */
  pageSize?: number;
}
