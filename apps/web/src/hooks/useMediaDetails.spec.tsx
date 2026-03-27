import { renderHook, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import {
  useMovieDetails,
  useTVShowDetails,
  useMovieCredits,
  useTVShowCredits,
  detailKeys,
} from './useMediaDetails';
import { tmdbService } from '../services/tmdb';
import type { MovieDetails, TVShowDetails, Credits } from '../types/tmdb';

// Mock the tmdb service
vi.mock('../services/tmdb', () => ({
  tmdbService: {
    getMovieDetails: vi.fn(),
    getTVShowDetails: vi.fn(),
    getMovieCredits: vi.fn(),
    getTVShowCredits: vi.fn(),
  },
}));

const mockMovieDetails: MovieDetails = {
  id: 123,
  title: '測試電影',
  originalTitle: 'Test Movie',
  overview: '測試劇情',
  releaseDate: '2024-01-15',
  posterPath: '/poster.jpg',
  backdropPath: '/backdrop.jpg',
  voteAverage: 8.5,
  voteCount: 1000,
  runtime: 120,
  budget: 10000000,
  revenue: 50000000,
  status: 'Released',
  tagline: '',
  genres: [{ id: 1, name: '動作' }],
  productionCountries: [],
  spokenLanguages: [],
  imdbId: 'tt1234567',
  homepage: null,
};

const mockTVShowDetails: TVShowDetails = {
  id: 456,
  name: '測試影集',
  originalName: 'Test TV Show',
  overview: '測試劇情',
  firstAirDate: '2023-06-01',
  lastAirDate: '2024-01-01',
  posterPath: '/tv_poster.jpg',
  backdropPath: '/tv_backdrop.jpg',
  voteAverage: 9.0,
  voteCount: 2000,
  episodeRunTime: [45],
  numberOfSeasons: 3,
  numberOfEpisodes: 30,
  status: 'Returning Series',
  type: 'Scripted',
  tagline: '',
  genres: [{ id: 3, name: '劇情' }],
  createdBy: [],
  networks: [],
  inProduction: true,
  seasons: [],
};

const mockCredits: Credits = {
  id: 123,
  cast: [{ id: 1, name: '演員一', character: '角色一', profilePath: null, order: 0 }],
  crew: [{ id: 2, name: '導演', job: 'Director', department: 'Directing', profilePath: null }],
};

function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,
      },
    },
  });
  return ({ children }: { children: React.ReactNode }) => (
    <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  );
}

describe('useMediaDetails hooks', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  afterEach(() => {
    vi.resetAllMocks();
  });

  describe('detailKeys', () => {
    it('should generate correct query keys for movie', () => {
      expect(detailKeys.movie(123)).toEqual(['details', 'movie', 123]);
    });

    it('should generate correct query keys for movie credits', () => {
      expect(detailKeys.movieCredits(123)).toEqual(['details', 'movie', 123, 'credits']);
    });

    it('should generate correct query keys for tv', () => {
      expect(detailKeys.tv(456)).toEqual(['details', 'tv', 456]);
    });

    it('should generate correct query keys for tv credits', () => {
      expect(detailKeys.tvCredits(456)).toEqual(['details', 'tv', 456, 'credits']);
    });
  });

  describe('useMovieDetails', () => {
    it('should fetch movie details', async () => {
      vi.mocked(tmdbService.getMovieDetails).mockResolvedValue(mockMovieDetails);

      const { result } = renderHook(() => useMovieDetails(123), {
        wrapper: createWrapper(),
      });

      await waitFor(() => expect(result.current.isSuccess).toBe(true));

      expect(tmdbService.getMovieDetails).toHaveBeenCalledWith(123);
      expect(result.current.data).toEqual(mockMovieDetails);
    });

    it('should not fetch when id is 0', () => {
      const { result } = renderHook(() => useMovieDetails(0), {
        wrapper: createWrapper(),
      });

      expect(result.current.fetchStatus).toBe('idle');
      expect(tmdbService.getMovieDetails).not.toHaveBeenCalled();
    });

    it('should handle error state', async () => {
      vi.mocked(tmdbService.getMovieDetails).mockRejectedValue(new Error('API Error'));

      const { result } = renderHook(() => useMovieDetails(123), {
        wrapper: createWrapper(),
      });

      await waitFor(() => expect(result.current.isError).toBe(true));
      expect(result.current.error?.message).toBe('API Error');
    });
  });

  describe('useTVShowDetails', () => {
    it('should fetch TV show details', async () => {
      vi.mocked(tmdbService.getTVShowDetails).mockResolvedValue(mockTVShowDetails);

      const { result } = renderHook(() => useTVShowDetails(456), {
        wrapper: createWrapper(),
      });

      await waitFor(() => expect(result.current.isSuccess).toBe(true));

      expect(tmdbService.getTVShowDetails).toHaveBeenCalledWith(456);
      expect(result.current.data).toEqual(mockTVShowDetails);
    });

    it('should not fetch when id is 0', () => {
      const { result } = renderHook(() => useTVShowDetails(0), {
        wrapper: createWrapper(),
      });

      expect(result.current.fetchStatus).toBe('idle');
      expect(tmdbService.getTVShowDetails).not.toHaveBeenCalled();
    });
  });

  describe('useMovieCredits', () => {
    it('should fetch movie credits', async () => {
      vi.mocked(tmdbService.getMovieCredits).mockResolvedValue(mockCredits);

      const { result } = renderHook(() => useMovieCredits(123), {
        wrapper: createWrapper(),
      });

      await waitFor(() => expect(result.current.isSuccess).toBe(true));

      expect(tmdbService.getMovieCredits).toHaveBeenCalledWith(123);
      expect(result.current.data).toEqual(mockCredits);
    });
  });

  describe('useTVShowCredits', () => {
    it('should fetch TV show credits', async () => {
      vi.mocked(tmdbService.getTVShowCredits).mockResolvedValue(mockCredits);

      const { result } = renderHook(() => useTVShowCredits(456), {
        wrapper: createWrapper(),
      });

      await waitFor(() => expect(result.current.isSuccess).toBe(true));

      expect(tmdbService.getTVShowCredits).toHaveBeenCalledWith(456);
      expect(result.current.data).toEqual(mockCredits);
    });
  });
});
