import { describe, it, expect } from 'vitest';
import { formatLocalDateTime } from './formatLocalDateTime';

// ISO strings WITHOUT an offset parse as local time, so these are TZ-independent.
describe('formatLocalDateTime', () => {
  it('pads to YYYY-MM-DD HH:mm by default', () => {
    expect(formatLocalDateTime('2026-09-01T03:05:09')).toBe('2026-09-01 03:05');
  });

  it('adds seconds when asked', () => {
    expect(formatLocalDateTime('2026-09-01T03:05:09', 'second')).toBe('2026-09-01 03:05:09');
  });

  it('uses the local zone for a UTC stamp', () => {
    const utc = '2026-03-20T14:00:00Z';
    const d = new Date(utc);
    expect(formatLocalDateTime(utc)).toBe(
      `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')} ${String(d.getHours()).padStart(2, '0')}:00`
    );
  });

  it.each([[''], [null], [undefined], ['not a date']])('returns empty for %p', (v) => {
    expect(formatLocalDateTime(v as string)).toBe('');
  });
});
