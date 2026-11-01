import { describe, expect, it } from 'vitest';
import { formatRunDuration } from './formatRunDuration';

describe('formatRunDuration', () => {
  it('seconds, minutes and seconds, hours and minutes', () => {
    expect(formatRunDuration('2026-11-01T10:40:09.5Z', '2026-11-01T10:40:54.4Z')).toBe('45 秒');
    // A cache hit really takes ~70 ms.
    expect(formatRunDuration('2026-11-01T05:47:02.244Z', '2026-11-01T05:47:02.316Z')).toBe(
      '不到 1 秒'
    );
    expect(formatRunDuration('2026-11-01T10:40:09.558Z', '2026-11-01T10:48:08.461Z')).toBe(
      '7 分 59 秒'
    );
    expect(formatRunDuration('2026-11-01T10:00:00Z', '2026-11-01T11:05:30Z')).toBe('1 小時 5 分');
  });

  it('a missing, unreadable or backwards end is undefined — never 「0 秒」', () => {
    expect(formatRunDuration('2026-11-01T10:00:00Z', null)).toBeUndefined();
    expect(formatRunDuration(undefined, '2026-11-01T10:00:00Z')).toBeUndefined();
    expect(formatRunDuration('nope', '2026-11-01T10:00:00Z')).toBeUndefined();
    expect(formatRunDuration('2026-11-01T10:00:05Z', '2026-11-01T10:00:00Z')).toBeUndefined();
  });

  it('nanosecond timestamps from the API still parse', () => {
    expect(
      formatRunDuration('2026-11-01T10:40:09.558765419Z', '2026-11-01T10:48:08.461912524Z')
    ).toBe('7 分 59 秒');
  });
});
