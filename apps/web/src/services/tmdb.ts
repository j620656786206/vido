import type {
  MovieSearchResponse,
  TVShowSearchResponse,
  ApiResponse,
  MovieDetails,
  TVShowDetails,
  Credits,
  VideosResponse,
  UnifiedSearchResult,
} from '../types/tmdb';
import { snakeToCamel } from '../utils/caseTransform';

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || '/api/v1';

// bugfix-d CR L2: the image base and the ImageSize union used to be duplicated
// here alongside a second getImageUrl. Both now come from the single
// implementation in lib/image.ts so they cannot drift apart again.
export type { ImageSize } from '../lib/image';

/**
 * Contextual per-facet result counts (Story ux3-discover-facet-aggregation-fe,
 * consumes ux3-discover-facet-aggregation-be [@contract-v1]). Outer key =
 * dimension; inner key = the facet value exactly as the FE supplied it
 * (`String(genre.id)` / `region.code` / `String(ratingValue)` /
 * `String(platform.id)`) → the contextual movie+tv total for (base filter + that
 * facet). `partial` is true when the BE could not resolve every requested facet
 * within its time budget; unresolved keys are omitted (they render as the
 * computing "–" placeholder and fill on a re-poll, AC5).
 */
export interface FacetCounts {
  counts: {
    genre?: Record<string, number>;
    region?: Record<string, number>;
    rating?: Record<string, number>;
    platform?: Record<string, number>;
  };
  partial: boolean;
}

/**
 * bugfix-d CR L2: this was a second, silently-diverging copy of `lib/image.ts`'s
 * helper — it lacked the absolute-URL passthrough and would have re-prefixed any
 * stored absolute poster. Re-exported from the one implementation so the two can
 * never drift again.
 */
export { getImageUrl } from '../lib/image';

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

  return snakeToCamel<T>(data.data);
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

  // Story 10-2 — trending feeds for hero banner.
  async getTrendingMovies(
    timeWindow: 'day' | 'week' = 'week',
    page = 1
  ): Promise<MovieSearchResponse> {
    const params = new URLSearchParams({ time_window: timeWindow, page: String(page) });
    return fetchApi<MovieSearchResponse>(`/tmdb/trending/movies?${params}`);
  },

  async getTrendingTVShows(
    timeWindow: 'day' | 'week' = 'week',
    page = 1
  ): Promise<TVShowSearchResponse> {
    const params = new URLSearchParams({ time_window: timeWindow, page: String(page) });
    return fetchApi<TVShowSearchResponse>(`/tmdb/trending/tv?${params}`);
  },

  // Story 10-2 — videos (trailers/teasers) for the trailer modal.
  async getMovieVideos(movieId: number): Promise<VideosResponse> {
    return fetchApi<VideosResponse>(`/tmdb/movies/${movieId}/videos`);
  },

  async getTVShowVideos(tvId: number): Promise<VideosResponse> {
    return fetchApi<VideosResponse>(`/tmdb/tv/${tvId}/videos`);
  },

  // Story 11-2 — multi-dimensional discover (consumes the Story 11-1 filter
  // engine). `params` are the backend-shaped query params from buildDiscoverParams.
  async discoverMovies(params: URLSearchParams): Promise<MovieSearchResponse> {
    return fetchApi<MovieSearchResponse>(`/tmdb/discover/movies?${params.toString()}`);
  },

  async discoverTVShows(params: URLSearchParams): Promise<TVShowSearchResponse> {
    return fetchApi<TVShowSearchResponse>(`/tmdb/discover/tv?${params.toString()}`);
  },

  // Story ux3-discover-facet-aggregation-fe — contextual per-facet result counts
  // for the Discover rail (consumes ux3-discover-facet-aggregation-be
  // [@contract-v1]). `params` carry the base discover filter PLUS the *_values
  // candidate CSVs (built by buildFacetCountParams). The response counts are keyed
  // by dimension → the facet value exactly as the FE supplied it, so they align
  // 1:1 with the chip keys. `fetchApi` snakeToCamel-transforms the response; the
  // `counts`/`partial` keys and the numeric/region-code inner keys survive as-is.
  async discoverFacetCounts(params: URLSearchParams): Promise<FacetCounts> {
    return fetchApi<FacetCounts>(`/tmdb/discover/facet-counts?${params.toString()}`);
  },

  // Story 11-3 — unified instant search: dual-language (zh-TW + en) movies, TV,
  // and people in one call, with zh-TW title matches boosted server-side.
  async unifiedSearch(query: string, page = 1): Promise<UnifiedSearchResult> {
    const params = new URLSearchParams({ q: query, page: String(page) });
    return fetchApi<UnifiedSearchResult>(`/search?${params}`);
  },
};

export default tmdbService;
