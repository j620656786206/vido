import type {
  MovieSearchResponse,
  TVShowSearchResponse,
  ApiResponse,
  MovieDetails,
  TVShowDetails,
  Credits,
} from '../types/tmdb';

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080/api/v1';

const TMDB_IMAGE_BASE = 'https://image.tmdb.org/t/p';

export type ImageSize = 'w92' | 'w154' | 'w185' | 'w342' | 'w500' | 'w780' | 'original';

export function getImageUrl(path: string | null, size: ImageSize = 'w342'): string | null {
  if (!path) return null;
  return `${TMDB_IMAGE_BASE}/${size}${path}`;
}

async function fetchApi<T>(endpoint: string): Promise<T> {
  const response = await fetch(`${API_BASE_URL}${endpoint}`);

  if (!response.ok) {
    const errorData = await response.json().catch(() => ({}));
    throw new Error(errorData.error?.message || `API request failed: ${response.status}`);
  }

  const data: ApiResponse<T> = await response.json();

  if (!data.success) {
    throw new Error(data.error?.message || 'API request failed');
  }

  return data.data as T;
}

export const tmdbService = {
  async searchMovies(query: string, page = 1): Promise<MovieSearchResponse> {
    const params = new URLSearchParams({
      query,
      page: String(page),
    });
    return fetchApi<MovieSearchResponse>(`/tmdb/search/movies?${params}`);
  },

  async searchTVShows(query: string, page = 1): Promise<TVShowSearchResponse> {
    const params = new URLSearchParams({
      query,
      page: String(page),
    });
    return fetchApi<TVShowSearchResponse>(`/tmdb/search/tv?${params}`);
  },

  async getMovieDetails(movieId: number): Promise<MovieDetails> {
    return fetchApi<MovieDetails>(`/tmdb/movies/${movieId}`);
  },

  async getTVShowDetails(tvId: number): Promise<TVShowDetails> {
    return fetchApi<TVShowDetails>(`/tmdb/tv/${tvId}`);
  },

  async getMovieCredits(movieId: number): Promise<Credits> {
    return fetchApi<Credits>(`/tmdb/movies/${movieId}/credits`);
  },

  async getTVShowCredits(tvId: number): Promise<Credits> {
    return fetchApi<Credits>(`/tmdb/tv/${tvId}/credits`);
  },
};

export default tmdbService;
