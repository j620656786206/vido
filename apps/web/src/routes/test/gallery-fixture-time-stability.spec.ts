import { readFileSync } from 'node:fs';
import { join } from 'node:path';
import { describe, it, expect } from 'vitest';

/**
 * Visual-baseline rot guard.
 *
 * `ServiceStatusCard` renders `檢查於 <formatRelativeTime(lastCheckAt)>`. When a
 * gallery fixture pins `lastCheckAt` to a hardcoded absolute date, the rendered
 * label grows by one every day ("68 天前" → "69 天前" → …) and the committed
 * visual baseline goes red every single day with nobody having changed anything.
 * Regenerating `-linux` baselines needs a CI round-trip (they cannot be made on
 * darwin), so this failure mode is expensive and easy to misread as a flake.
 *
 * Fixtures feeding a relative-time renderer must therefore be expressed as an
 * offset from now (`JUST_NOW()` / `MINUTES_AGO()` / `HOURS_AGO()`), never as a
 * literal timestamp.
 */
const FIXTURES = join(__dirname, '-gallery.fixtures.tsx');
const ABSOLUTE_ISO = /\d{4}-\d{2}-\d{2}T\d{2}:\d{2}/;

describe('gallery fixtures — relative-time fields stay time-stable', () => {
  it('never pins lastCheckAt to a literal timestamp', () => {
    const offenders = readFileSync(FIXTURES, 'utf8')
      .split('\n')
      .map((line, i) => ({ line: line.trim(), no: i + 1 }))
      .filter(({ line }) => line.startsWith('lastCheckAt:') && ABSOLUTE_ISO.test(line));

    expect(
      offenders,
      `-gallery.fixtures.tsx pins lastCheckAt to an absolute date at line(s) ${offenders
        .map((o) => o.no)
        .join(', ')}. Use JUST_NOW() / MINUTES_AGO() / HOURS_AGO() instead — an ` +
        'absolute date renders a day count that changes daily and rots the visual baseline.'
    ).toEqual([]);
  });

  it('keeps the offset helpers mid-bucket so a slow render cannot tip the label', () => {
    const source = readFileSync(FIXTURES, 'utf8');
    // formatRelativeTime buckets: <45s 剛剛 · <60min N 分鐘前 · <24h N 小時前.
    expect(source).toContain('const JUST_NOW = () => agoIso(5);');
    expect(source).toContain('const MINUTES_AGO = () => agoIso(330);');
    expect(source).toContain('const HOURS_AGO = () => agoIso(9000);');
  });
});
