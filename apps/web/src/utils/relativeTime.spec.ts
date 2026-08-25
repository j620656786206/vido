import { describe, it, expect } from 'vitest';
import { formatRelativeTime } from './relativeTime';

const NOW = Date.parse('2026-06-15T12:00:00Z');

describe('formatRelativeTime', () => {
  // Found live: the status API returned Go's zero time and the card rendered
  // 「檢查於 739852 天前」. Ancient sentinels are "never", not "long ago".
  it('treats pre-2000 sentinel dates as missing', () => {
    expect(formatRelativeTime('0001-01-01T00:00:00Z')).toBe('');
    expect(formatRelativeTime('1970-01-01T00:00:00Z')).toBe('');
  });

  it('returns 剛剛 for very recent times', () => {
    expect(formatRelativeTime('2026-06-15T11:59:40Z', NOW)).toBe('剛剛');
  });

  it('formats minutes', () => {
    expect(formatRelativeTime('2026-06-15T11:58:00Z', NOW)).toBe('2 分鐘前');
    expect(formatRelativeTime('2026-06-15T11:42:00Z', NOW)).toBe('18 分鐘前');
  });

  it('formats hours', () => {
    expect(formatRelativeTime('2026-06-15T11:00:00Z', NOW)).toBe('1 小時前');
  });

  it('formats days', () => {
    expect(formatRelativeTime('2026-06-13T12:00:00Z', NOW)).toBe('2 天前');
  });

  it('returns empty string for missing/invalid input (never throws — F3)', () => {
    expect(formatRelativeTime(undefined, NOW)).toBe('');
    expect(formatRelativeTime('not-a-date', NOW)).toBe('');
  });
});
