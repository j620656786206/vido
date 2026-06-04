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
              originalTitle: 'Demon Slayer',
              overview: 'A boy becomes a demon slayer.',
              releaseDate: '2019-04-06',
              posterPath: '/poster.jpg',
              backdropPath: '/backdrop.jpg',
              voteAverage: 8.5,
              voteCount: 1000,
              genreIds: [16, 28],
            },
          ],
          totalPages: 1,
          totalResults: 1,
        },
      };

      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve(mockResponse),
      });

      const result = await tmdbService.searchMovies('鬼滅之刃');

      expect(mockFetch).toHaveBeenCalledWith(
        expect.stringContaining(
          '/tmdb/search/movies?query=%E9%AC%BC%E6%BB%85%E4%B9%8B%E5%88%83&page=1'
        )
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
          totalPages: 5,
          totalResults: 100,
        },
      };

      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve(mockResponse),
      });

      await tmdbService.searchMovies('test', 2);

      expect(mockFetch).toHaveBeenCalledWith(expect.stringContaining('page=2'));
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

      await expect(tmdbService.searchMovies('test')).rejects.toThrow('Internal server error');
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

      await expect(tmdbService.searchMovies('test')).rejects.toThrow('Invalid query');
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
              originalName: 'Attack on Titan',
              overview: 'Humanity fights against titans.',
              firstAirDate: '2013-04-07',
              posterPath: '/poster.jpg',
              backdropPath: '/backdrop.jpg',
              voteAverage: 9.0,
              voteCount: 2000,
              genreIds: [16, 10759],
            },
          ],
          totalPages: 1,
          totalResults: 1,
        },
      };

      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve(mockResponse),
      });

      const result = await tmdbService.searchTVShows('進擊的巨人');

      expect(mockFetch).toHaveBeenCalledWith(expect.stringContaining('/tmdb/search/tv?'));
      expect(result.results).toHaveLength(1);
      expect(result.results[0].name).toBe('進擊的巨人');
    });
  });

  describe('unifiedSearch (Story 11-3)', () => {
    it('calls the unified /search endpoint with q + page and camelCases the response', async () => {
      const mockResponse = {
        success: true,
        data: {
          query: '你的名字',
          page: 1,
          movies: [{ id: 1, title: '你的名字', original_title: 'Your Name', vote_average: 8.4 }],
          tv_shows: [{ id: 2, name: '影集', original_name: 'Show' }],
          people: [
            {
              id: 3,
              name: '新海誠',
              original_name: 'Makoto Shinkai',
              known_for_department: 'Directing',
            },
          ],
        },
      };
      mockFetch.mockResolvedValueOnce({ ok: true, json: () => Promise.resolve(mockResponse) });

      const result = await tmdbService.unifiedSearch('你的名字');

      expect(mockFetch).toHaveBeenCalledWith(
        expect.stringContaining('/search?q=%E4%BD%A0%E7%9A%84%E5%90%8D%E5%AD%97&page=1')
      );
      // snake_case → camelCase at the API boundary (Rule 18)
      expect(result.tvShows).toHaveLength(1);
      expect(result.movies[0].originalTitle).toBe('Your Name');
      expect(result.people[0].knownForDepartment).toBe('Directing');
    });

    it('does not touch the legacy per-type search endpoints (backward compatibility — AC #6)', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: () =>
          Promise.resolve({
            success: true,
            data: { query: 'x', page: 1, movies: [], tv_shows: [], people: [] },
          }),
      });

      await tmdbService.unifiedSearch('x');

      const calledUrl = mockFetch.mock.calls[0][0] as string;
      expect(calledUrl).toContain('/search?');
      expect(calledUrl).not.toContain('/tmdb/search/');
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
