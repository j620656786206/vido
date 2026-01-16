import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { tmdbService, getImageUrl } from './tmdb';

describe('tmdbService', () => {
  const mockFetch = vi.fn();

  beforeEach(() => {
    vi.stubGlobal('fetch', mockFetch);
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  describe('searchMovies', () => {
    it('should search movies with query and default page', async () => {
      const mockResponse = {
        success: true,
        data: {
          page: 1,
          results: [
            {
              id: 1,
              title: '鬼滅之刃',
              original_title: 'Demon Slayer',
              overview: 'A boy becomes a demon slayer.',
              release_date: '2019-04-06',
              poster_path: '/poster.jpg',
              backdrop_path: '/backdrop.jpg',
              vote_average: 8.5,
              vote_count: 1000,
              genre_ids: [16, 28],
            },
          ],
          total_pages: 1,
          total_results: 1,
        },
      };

      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve(mockResponse),
      });

      const result = await tmdbService.searchMovies('鬼滅之刃');

      expect(mockFetch).toHaveBeenCalledWith(
        expect.stringContaining('/tmdb/search/movies?query=%E9%AC%BC%E6%BB%85%E4%B9%8B%E5%88%83&page=1')
      );
      expect(result.results).toHaveLength(1);
      expect(result.results[0].title).toBe('鬼滅之刃');
    });

    it('should search movies with custom page', async () => {
      const mockResponse = {
        success: true,
        data: {
          page: 2,
          results: [],
          total_pages: 5,
          total_results: 100,
        },
      };

      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve(mockResponse),
      });

      await tmdbService.searchMovies('test', 2);

      expect(mockFetch).toHaveBeenCalledWith(
        expect.stringContaining('page=2')
      );
    });

    it('should throw error on API failure', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: false,
        status: 500,
        json: () =>
          Promise.resolve({
            success: false,
            error: { message: 'Internal server error' },
          }),
      });

      await expect(tmdbService.searchMovies('test')).rejects.toThrow(
        'Internal server error'
      );
    });

    it('should throw error on unsuccessful response', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: () =>
          Promise.resolve({
            success: false,
            error: { message: 'Invalid query' },
          }),
      });

      await expect(tmdbService.searchMovies('test')).rejects.toThrow(
        'Invalid query'
      );
    });
  });

  describe('searchTVShows', () => {
    it('should search TV shows with query', async () => {
      const mockResponse = {
        success: true,
        data: {
          page: 1,
          results: [
            {
              id: 2,
              name: '進擊的巨人',
              original_name: 'Attack on Titan',
              overview: 'Humanity fights against titans.',
              first_air_date: '2013-04-07',
              poster_path: '/poster.jpg',
              backdrop_path: '/backdrop.jpg',
              vote_average: 9.0,
              vote_count: 2000,
              genre_ids: [16, 10759],
            },
          ],
          total_pages: 1,
          total_results: 1,
        },
      };

      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve(mockResponse),
      });

      const result = await tmdbService.searchTVShows('進擊的巨人');

      expect(mockFetch).toHaveBeenCalledWith(
        expect.stringContaining('/tmdb/search/tv?')
      );
      expect(result.results).toHaveLength(1);
      expect(result.results[0].name).toBe('進擊的巨人');
    });
  });
});

describe('getImageUrl', () => {
  it('should return null for null path', () => {
    expect(getImageUrl(null)).toBeNull();
  });

  it('should return full URL with default size', () => {
    const url = getImageUrl('/poster.jpg');
    expect(url).toBe('https://image.tmdb.org/t/p/w342/poster.jpg');
  });

  it('should return full URL with custom size', () => {
    const url = getImageUrl('/poster.jpg', 'w500');
    expect(url).toBe('https://image.tmdb.org/t/p/w500/poster.jpg');
  });

  it('should return full URL with original size', () => {
    const url = getImageUrl('/poster.jpg', 'original');
    expect(url).toBe('https://image.tmdb.org/t/p/original/poster.jpg');
  });
});
