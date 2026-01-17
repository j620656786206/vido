import { describe, it, expect } from 'vitest';
import { GENRE_MAP, getGenreNames } from './genres';

describe('genres utilities', () => {
  describe('GENRE_MAP', () => {
    it('contains movie genres in Traditional Chinese', () => {
      expect(GENRE_MAP[28]).toBe('動作');
      expect(GENRE_MAP[16]).toBe('動畫');
      expect(GENRE_MAP[35]).toBe('喜劇');
      expect(GENRE_MAP[18]).toBe('劇情');
      expect(GENRE_MAP[27]).toBe('恐怖');
      expect(GENRE_MAP[878]).toBe('科幻');
    });

    it('contains TV genres in Traditional Chinese', () => {
      expect(GENRE_MAP[10759]).toBe('動作冒險');
      expect(GENRE_MAP[10765]).toBe('科幻奇幻');
      expect(GENRE_MAP[10767]).toBe('脫口秀');
    });
  });

  describe('getGenreNames', () => {
    it('returns genre names for valid IDs', () => {
      const result = getGenreNames([28, 16, 14]);
      expect(result).toEqual(['動作', '動畫', '奇幻']);
    });

    it('returns empty array for empty input', () => {
      const result = getGenreNames([]);
      expect(result).toEqual([]);
    });

    it('filters out unknown genre IDs', () => {
      const result = getGenreNames([28, 99999, 16]);
      expect(result).toEqual(['動作', '動畫']);
    });

    it('returns empty array when all IDs are unknown', () => {
      const result = getGenreNames([99999, 88888, 77777]);
      expect(result).toEqual([]);
    });

    it('limits results to 3 by default', () => {
      const result = getGenreNames([28, 16, 14, 35, 18]);
      expect(result).toHaveLength(3);
      expect(result).toEqual(['動作', '動畫', '奇幻']);
    });

    it('respects custom limit parameter', () => {
      const result = getGenreNames([28, 16, 14, 35, 18], 2);
      expect(result).toHaveLength(2);
      expect(result).toEqual(['動作', '動畫']);
    });

    it('returns all when fewer genres than limit', () => {
      const result = getGenreNames([28, 16], 5);
      expect(result).toHaveLength(2);
      expect(result).toEqual(['動作', '動畫']);
    });

    it('handles limit of 0', () => {
      const result = getGenreNames([28, 16, 14], 0);
      expect(result).toEqual([]);
    });

    it('handles mixed valid and invalid IDs with limit', () => {
      const result = getGenreNames([99999, 28, 88888, 16, 77777, 14], 3);
      // Slices first 3 from [99999, 28, 88888], then filters
      expect(result).toEqual(['動作']);
    });
  });
});
