import { describe, it, expect } from 'vitest';
import { snakeToCamel } from './caseTransform';

describe('snakeToCamel', () => {
  it('converts flat object keys from snake_case to camelCase', () => {
    const input = { poster_path: '/poster.jpg', release_date: '2024-01-01' };
    const result = snakeToCamel<{ posterPath: string; releaseDate: string }>(input);
    expect(result).toEqual({ posterPath: '/poster.jpg', releaseDate: '2024-01-01' });
  });

  it('handles nested objects recursively', () => {
    const input = {
      movie: {
        poster_path: '/poster.jpg',
        vote_average: 8.5,
      },
    };
    const result = snakeToCamel<{ movie: { posterPath: string; voteAverage: number } }>(input);
    expect(result).toEqual({
      movie: {
        posterPath: '/poster.jpg',
        voteAverage: 8.5,
      },
    });
  });

  it('handles arrays of objects', () => {
    const input = [
      { tmdb_id: 1, poster_path: '/a.jpg' },
      { tmdb_id: 2, poster_path: '/b.jpg' },
    ];
    const result = snakeToCamel<{ tmdbId: number; posterPath: string }[]>(input);
    expect(result).toEqual([
      { tmdbId: 1, posterPath: '/a.jpg' },
      { tmdbId: 2, posterPath: '/b.jpg' },
    ]);
  });

  it('handles null values', () => {
    expect(snakeToCamel(null)).toBeNull();
  });

  it('handles undefined values', () => {
    expect(snakeToCamel(undefined)).toBeUndefined();
  });

  it('passes through primitive values unchanged', () => {
    expect(snakeToCamel('hello')).toBe('hello');
    expect(snakeToCamel(42)).toBe(42);
    expect(snakeToCamel(true)).toBe(true);
  });

  it('handles null values within objects', () => {
    const input = { poster_path: null, release_date: '2024-01-01' };
    const result = snakeToCamel<{ posterPath: null; releaseDate: string }>(input);
    expect(result).toEqual({ posterPath: null, releaseDate: '2024-01-01' });
  });

  it('preserves keys that are already camelCase', () => {
    const input = { title: 'Test', id: '123' };
    const result = snakeToCamel<{ title: string; id: string }>(input);
    expect(result).toEqual({ title: 'Test', id: '123' });
  });

  it('handles edge case: tmdb_id → tmdbId', () => {
    const input = { tmdb_id: 12345 };
    expect(snakeToCamel<{ tmdbId: number }>(input)).toEqual({ tmdbId: 12345 });
  });

  it('handles edge case: imdb_id → imdbId', () => {
    const input = { imdb_id: 'tt1234567' };
    expect(snakeToCamel<{ imdbId: string }>(input)).toEqual({ imdbId: 'tt1234567' });
  });

  it('handles edge case: created_at → createdAt', () => {
    const input = { created_at: '2024-01-01T00:00:00Z' };
    expect(snakeToCamel<{ createdAt: string }>(input)).toEqual({
      createdAt: '2024-01-01T00:00:00Z',
    });
  });

  it('handles deeply nested structures', () => {
    const input = {
      success: true,
      data: {
        items: [
          {
            type: 'movie',
            movie: {
              poster_path: '/p.jpg',
              vote_average: 7.5,
              original_language: 'en',
            },
          },
        ],
        total_items: 1,
        total_pages: 1,
      },
    };
    const result = snakeToCamel<Record<string, unknown>>(input);
    expect(result).toEqual({
      success: true,
      data: {
        items: [
          {
            type: 'movie',
            movie: {
              posterPath: '/p.jpg',
              voteAverage: 7.5,
              originalLanguage: 'en',
            },
          },
        ],
        totalItems: 1,
        totalPages: 1,
      },
    });
  });

  it('handles empty object', () => {
    expect(snakeToCamel({})).toEqual({});
  });

  it('handles empty array', () => {
    expect(snakeToCamel([])).toEqual([]);
  });

  it('handles mixed arrays (objects and primitives)', () => {
    const input = [{ first_name: 'A' }, 'string', 42, null];
    const result = snakeToCamel(input);
    expect(result).toEqual([{ firstName: 'A' }, 'string', 42, null]);
  });
});
