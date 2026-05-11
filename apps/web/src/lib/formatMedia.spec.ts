import { describe, it, expect } from 'vitest';
import { formatRuntime, formatSeriesCount, formatPosterMeta } from './formatMedia';

describe('formatMedia', () => {
  describe('formatRuntime', () => {
    it('returns empty string for 0 minutes', () => {
      expect(formatRuntime(0)).toBe('');
    });

    it('returns empty string for undefined', () => {
      expect(formatRuntime(undefined)).toBe('');
    });

    it('returns empty string for null', () => {
      expect(formatRuntime(null)).toBe('');
    });

    it('returns empty string for negative minutes', () => {
      expect(formatRuntime(-10)).toBe('');
    });

    it('formats sub-hour runtimes with 分鐘', () => {
      expect(formatRuntime(47)).toBe('47 分鐘');
      expect(formatRuntime(59)).toBe('59 分鐘');
    });

    it('drops the minutes segment when the runtime is a whole number of hours', () => {
      expect(formatRuntime(60)).toBe('1 小時');
      expect(formatRuntime(120)).toBe('2 小時');
    });

    it('formats hour + minute runtimes', () => {
      expect(formatRuntime(125)).toBe('2 小時 5 分');
      expect(formatRuntime(139)).toBe('2 小時 19 分');
    });
  });

  describe('formatSeriesCount', () => {
    it('returns empty string when seasons is 0', () => {
      expect(formatSeriesCount(0, 12)).toBe('');
    });

    it('returns empty string when seasons is undefined', () => {
      expect(formatSeriesCount(undefined, 12)).toBe('');
    });

    it('returns empty string when seasons is null', () => {
      expect(formatSeriesCount(null, 12)).toBe('');
    });

    it('returns seasons-only when episodes is undefined', () => {
      expect(formatSeriesCount(1, undefined)).toBe('1 季');
    });

    it('returns seasons-only when episodes is 0', () => {
      expect(formatSeriesCount(1, 0)).toBe('1 季');
    });

    it('returns seasons + episodes when both are present', () => {
      expect(formatSeriesCount(4, 34)).toBe('4 季 34 集');
    });
  });

  describe('formatPosterMeta', () => {
    it('joins year and extra with a middot', () => {
      expect(formatPosterMeta(2022, '2 小時 19 分')).toBe('2022 · 2 小時 19 分');
    });

    it('returns year-only when extra is empty', () => {
      expect(formatPosterMeta(2022, '')).toBe('2022');
    });

    it('returns extra-only when year is null', () => {
      expect(formatPosterMeta(null, '4 季')).toBe('4 季');
    });

    it('returns empty string when neither is present', () => {
      expect(formatPosterMeta(null, '')).toBe('');
    });
  });
});
