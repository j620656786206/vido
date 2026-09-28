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
