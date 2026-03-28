import { renderHook, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import {
  useMovieDetails,
  useTVShowDetails,
  useMovieCredits,
  useTVShowCredits,
  useLocalMovieDetails,
  useLocalSeriesDetails,
  detailKeys,
} from './useMediaDetails';
import { tmdbService } from '../services/tmdb';
import { libraryService } from '../services/libraryService';
import type { MovieDetails, TVShowDetails, Credits } from '../types/tmdb';
import type { LibraryMovie, LibrarySeries } from '../types/library';

// Mock the tmdb service
vi.mock('../services/tmdb', () => ({
  tmdbService: {
    getMovieDetails: vi.fn(),
    getTVShowDetails: vi.fn(),
    getMovieCredits: vi.fn(),
    getTVShowCredits: vi.fn(),
  },
}));

// Mock the library service
vi.mock('../services/libraryService', () => ({
  libraryService: {
    getMovieById: vi.fn(),
    getSeriesById: vi.fn(),
    listLibrary: vi.fn(),
    searchLibrary: vi.fn(),
    getStats: vi.fn(),
    getGenres: vi.fn(),
    getRecentlyAdded: vi.fn(),
    deleteMovie: vi.fn(),
    deleteSeries: vi.fn(),
    reparseMovie: vi.fn(),
    reparseSeries: vi.fn(),
    exportMovie: vi.fn(),
    exportSeries: vi.fn(),
    getMovieVideos: vi.fn(),
    getSeriesVideos: vi.fn(),
    batchDelete: vi.fn(),
    batchReparse: vi.fn(),
    batchExport: vi.fn(),
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

  describe('useLocalMovieDetails', () => {
    const mockLocalMovie: LibraryMovie = {
      id: '0ce73c75-a742-4fc0-955a-4d915a7ee465',
      title: '全面啟動',
      originalTitle: 'Inception',
      releaseDate: '2010-07-15',
      genres: ['動作', '科幻'],
      posterPath: '/poster.jpg',
      tmdbId: 27205,
      overview: '偷技高超的神偷唐姆柯比...',
      voteAverage: 8.4,
      parseStatus: 'success',
      metadataSource: 'tmdb',
      createdAt: '2026-03-27T10:00:00Z',
      updatedAt: '2026-03-27T10:00:00Z',
    };

    it('[P0] should fetch movie from local API by UUID', async () => {
      vi.mocked(libraryService.getMovieById).mockResolvedValue(mockLocalMovie);

      const { result } = renderHook(
        () => useLocalMovieDetails('0ce73c75-a742-4fc0-955a-4d915a7ee465'),
        { wrapper: createWrapper() }
      );

      await waitFor(() => expect(result.current.isSuccess).toBe(true));

      expect(libraryService.getMovieById).toHaveBeenCalledWith(
        '0ce73c75-a742-4fc0-955a-4d915a7ee465'
      );
      expect(result.current.data?.id).toBe('0ce73c75-a742-4fc0-955a-4d915a7ee465');
      expect(result.current.data?.title).toBe('全面啟動');
      expect(result.current.data?.tmdbId).toBe(27205);
    });

    it('[P0] should not fetch when id is empty string', () => {
      const { result } = renderHook(() => useLocalMovieDetails(''), {
        wrapper: createWrapper(),
      });

      expect(result.current.fetchStatus).toBe('idle');
      expect(libraryService.getMovieById).not.toHaveBeenCalled();
    });

    it('[P1] should handle movie without TMDB metadata', async () => {
      const unenrichedMovie: LibraryMovie = {
        id: 'unenriched-uuid',
        title: 'DASS-880.mp4',
        releaseDate: '',
        genres: [],
        parseStatus: '',
        createdAt: '2026-03-27T10:00:00Z',
        updatedAt: '2026-03-27T10:00:00Z',
      };
      vi.mocked(libraryService.getMovieById).mockResolvedValue(unenrichedMovie);

      const { result } = renderHook(() => useLocalMovieDetails('unenriched-uuid'), {
        wrapper: createWrapper(),
      });

      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(result.current.data?.tmdbId).toBeUndefined();
      expect(result.current.data?.posterPath).toBeUndefined();
      expect(result.current.data?.title).toBe('DASS-880.mp4');
    });

    it('[P1] should handle API error', async () => {
      vi.mocked(libraryService.getMovieById).mockRejectedValue(new Error('Not found'));

      const { result } = renderHook(() => useLocalMovieDetails('nonexistent'), {
        wrapper: createWrapper(),
      });

      await waitFor(() => expect(result.current.isError).toBe(true));
      expect(result.current.error?.message).toBe('Not found');
    });
  });

  describe('useLocalSeriesDetails', () => {
    const mockLocalSeries: LibrarySeries = {
      id: 'series-uuid-1',
      title: 'Breaking Bad',
      firstAirDate: '2008-01-20',
      genres: ['劇情', '犯罪'],
      posterPath: '/bb.jpg',
      tmdbId: 1396,
      parseStatus: 'success',
      createdAt: '2026-03-27T10:00:00Z',
      updatedAt: '2026-03-27T10:00:00Z',
    };

    it('[P0] should fetch series from local API by UUID', async () => {
      vi.mocked(libraryService.getSeriesById).mockResolvedValue(mockLocalSeries);

      const { result } = renderHook(() => useLocalSeriesDetails('series-uuid-1'), {
        wrapper: createWrapper(),
      });

      await waitFor(() => expect(result.current.isSuccess).toBe(true));

      expect(libraryService.getSeriesById).toHaveBeenCalledWith('series-uuid-1');
      expect(result.current.data?.title).toBe('Breaking Bad');
    });

    it('[P0] should not fetch when id is empty string', () => {
      const { result } = renderHook(() => useLocalSeriesDetails(''), {
        wrapper: createWrapper(),
      });

      expect(result.current.fetchStatus).toBe('idle');
      expect(libraryService.getSeriesById).not.toHaveBeenCalled();
    });
  });

  describe('detailKeys (local)', () => {
    it('should generate correct query keys for local movie', () => {
      expect(detailKeys.localMovie('uuid-123')).toEqual(['details', 'local-movie', 'uuid-123']);
    });

    it('should generate correct query keys for local series', () => {
      expect(detailKeys.localSeries('uuid-456')).toEqual(['details', 'local-series', 'uuid-456']);
    });
  });
});
