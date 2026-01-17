import { describe, it, expect } from 'vitest';
import { getImageUrl, getImageSrcSet, getImageSizes } from './image';

describe('image utilities', () => {
  describe('getImageUrl', () => {
    it('returns correct URL for valid path with default size', () => {
      const result = getImageUrl('/abc123.jpg');
      expect(result).toBe('https://image.tmdb.org/t/p/w342/abc123.jpg');
    });

    it('returns correct URL for specified size', () => {
      const result = getImageUrl('/poster.jpg', 'w500');
      expect(result).toBe('https://image.tmdb.org/t/p/w500/poster.jpg');
    });

    it('returns null for null path', () => {
      const result = getImageUrl(null);
      expect(result).toBeNull();
    });

    it('returns null for null path with specified size', () => {
      const result = getImageUrl(null, 'w185');
      expect(result).toBeNull();
    });

    it('handles all supported sizes', () => {
      const sizes = ['w92', 'w154', 'w185', 'w342', 'w500', 'w780', 'original'] as const;
      sizes.forEach((size) => {
        const result = getImageUrl('/test.jpg', size);
        expect(result).toBe(`https://image.tmdb.org/t/p/${size}/test.jpg`);
      });
    });
  });

  describe('getImageSrcSet', () => {
    it('returns srcset string for valid path', () => {
      const result = getImageSrcSet('/poster.jpg');
      expect(result).toContain('https://image.tmdb.org/t/p/w185/poster.jpg 185w');
      expect(result).toContain('https://image.tmdb.org/t/p/w342/poster.jpg 342w');
      expect(result).toContain('https://image.tmdb.org/t/p/w500/poster.jpg 500w');
    });

    it('returns null for null path', () => {
      const result = getImageSrcSet(null);
      expect(result).toBeNull();
    });

    it('returns comma-separated values', () => {
      const result = getImageSrcSet('/test.jpg');
      expect(result).toMatch(/,.*,/); // At least 2 commas for 3 entries
    });
  });

  describe('getImageSizes', () => {
    it('returns sizes attribute string', () => {
      const result = getImageSizes();
      expect(result).toBe('(max-width: 640px) 45vw, (max-width: 1024px) 25vw, 200px');
    });

    it('includes mobile breakpoint', () => {
      const result = getImageSizes();
      expect(result).toContain('max-width: 640px');
    });

    it('includes tablet breakpoint', () => {
      const result = getImageSizes();
      expect(result).toContain('max-width: 1024px');
    });

    it('includes desktop fallback', () => {
      const result = getImageSizes();
      expect(result).toContain('200px');
    });
  });
});
