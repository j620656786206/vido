/**
 * bugfix-type-scale-code-migration — the eight-step type scale (DESIGN.md
 * §Hierarchy) has no 10 / 11 / 13 / 15px. Arbitrary `text-[13px]` sets only a
 * size, no line height, so those lines fall back to the browser's ≈1.2; 10 and
 * 11px are below the 12px floor. This guard stops them from coming back.
 */
import { describe, it, expect } from 'vitest';
import { readFileSync, readdirSync, statSync } from 'node:fs';
import { join, relative } from 'node:path';

const SRC = join(__dirname);
const BANNED = /text-\[(10|11|13|15)px\]/g;

function sourceFiles(dir: string): string[] {
  const out: string[] = [];
  for (const e of readdirSync(dir)) {
    const p = join(dir, e);
    if (statSync(p).isDirectory()) out.push(...sourceFiles(p));
    else if (/\.(ts|tsx)$/.test(e) && !/\.spec\.(ts|tsx)$/.test(e)) out.push(p);
  }
  return out;
}

describe('type scale (DESIGN.md: eight even steps, 12px floor)', () => {
  it('no source file uses a 10 / 11 / 13 / 15px arbitrary text size', () => {
    const hits: string[] = [];
    for (const file of sourceFiles(SRC)) {
      readFileSync(file, 'utf8')
        .split('\n')
        .forEach((line, i) => {
          for (const m of line.matchAll(BANNED)) {
            hits.push(`${relative(SRC, file)}:${i + 1} ${m[0]}`);
          }
        });
    }
    expect(hits, 'use text-xs (12) or text-sm (14) instead').toEqual([]);
  });
});

/**
 * bugfix-type-line-height-weight — every rung's line-height comes from
 * DESIGN.md §Hierarchy, not Tailwind's Latin defaults. Reads the REAL table
 * and the REAL styles.css, so editing either one alone turns this red.
 */
describe('type scale line-heights (DESIGN.md §Hierarchy ↔ styles.css @theme)', () => {
  const design = readFileSync(join(SRC, '../../../DESIGN.md'), 'utf8');
  const css = readFileSync(join(SRC, 'styles.css'), 'utf8');
  // | **Body** | **14 / 1.625 (22.75)** | 14 | 400 | `text-sm` | …
  const ROW =
    /^\|\s*\*\*[^|]+\|\s*\**\d+ \/ ([\d.]+) \([\d.]+\)\**\s*\|[^|]+\|\s*(\d+)\s*\|\s*`text-(\w+)`/gm;
  const rows = [...design.matchAll(ROW)].map((m) => ({
    rung: m[3],
    lineHeight: m[1],
    weight: m[2],
  }));
  const themeVar = (name: string) =>
    css.match(new RegExp(`--text-${name}:\\s*([^;]+);`))?.[1].trim();

  it('parses all eight rungs from the DESIGN.md table', () => {
    expect(rows.map((r) => r.rung)).toEqual(['4xl', '3xl', '2xl', 'xl', 'lg', 'base', 'sm', 'xs']);
  });

  it('each rung sets the line-height the table gives it', () => {
    const drift = rows
      .filter((r) => themeVar(`${r.rung}--line-height`) !== r.lineHeight)
      .map(
        (r) => `text-${r.rung}: table ${r.lineHeight}, css ${themeVar(`${r.rung}--line-height`)}`
      );
    expect(drift).toEqual([]);
  });

  it('Label (text-xs) defaults to the table weight 500', () => {
    const label = rows.find((r) => r.rung === 'xs');
    expect(label?.weight).toBe('500');
    expect(themeVar('xs--font-weight')).toBe('500');
  });
});
