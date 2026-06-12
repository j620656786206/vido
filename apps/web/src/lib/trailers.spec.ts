import { describe, it, expect } from 'vitest';
import { pickBestTrailer, pickTmdbVideoFallbackUrl } from './trailers';
import type { Video } from '../types/tmdb';

function video(overrides: Partial<Video> = {}): Video {
  return {
    key: 'abc123',
    name: 'Trailer',
    site: 'YouTube',
    type: 'Trailer',
    official: true,
    publishedAt: '2024-01-01T00:00:00.000Z',
    ...overrides,
  };
}

describe('pickBestTrailer', () => {
  it('returns null for empty/undefined results', () => {
    expect(pickBestTrailer(undefined)).toBeNull();
    expect(pickBestTrailer([])).toBeNull();
  });

  it('filters out non-YouTube trailers', () => {
    const results = [
      video({ key: 'vimeo1', site: 'Vimeo' }),
      video({ key: 'tt', site: 'YouTube' }),
    ];
    expect(pickBestTrailer(results)?.key).toBe('tt');
  });

  it('filters out non-Trailer types (Teaser, Featurette)', () => {
    const results = [
      video({ key: 'teaser1', type: 'Teaser' }),
      video({ key: 'real-trailer', type: 'Trailer' }),
    ];
    expect(pickBestTrailer(results)?.key).toBe('real-trailer');
  });

  it('returns null when videos exist but none is a YouTube Trailer (AC #3 fallback trigger)', () => {
    const results = [
      video({ key: 'vimeo1', site: 'Vimeo', type: 'Trailer' }),
      video({ key: 'teaser1', site: 'YouTube', type: 'Teaser' }),
    ];
    expect(pickBestTrailer(results)).toBeNull();
  });

  it('rejects keys with invalid characters (XSS guard)', () => {
    const results = [
      video({ key: '<script>', type: 'Trailer' }),
      video({ key: 'safe_key-123', type: 'Trailer' }),
    ];
    expect(pickBestTrailer(results)?.key).toBe('safe_key-123');
  });

  it('prefers official over unofficial', () => {
    const results = [
      video({ key: 'fan', official: false, publishedAt: '2025-01-01' }),
      video({ key: 'official', official: true, publishedAt: '2020-01-01' }),
    ];
    expect(pickBestTrailer(results)?.key).toBe('official');
  });

  it('among same officiality, prefers most recent (newest publishedAt)', () => {
    const results = [
      video({ key: 'old', official: true, publishedAt: '2020-01-01' }),
      video({ key: 'new', official: true, publishedAt: '2025-06-01' }),
    ];
    expect(pickBestTrailer(results)?.key).toBe('new');
  });
});

describe('pickTmdbVideoFallbackUrl', () => {
  it('builds the movie videos-tab URL', () => {
    expect(pickTmdbVideoFallbackUrl(550, 'movie')).toBe(
      'https://www.themoviedb.org/movie/550/videos'
    );
  });

  it('builds the tv videos-tab URL', () => {
    expect(pickTmdbVideoFallbackUrl(1396, 'tv')).toBe('https://www.themoviedb.org/tv/1396/videos');
  });
});
