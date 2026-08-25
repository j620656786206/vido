import { readFileSync } from 'node:fs';
import { join } from 'node:path';
import { describe, it, expect } from 'vitest';

/**
 * Token contrast gate.
 *
 * PRODUCT.md:103 makes WCAG AA 4.5:1 a 硬性要求, and this codebase has now been
 * bitten by the same class of defect three times: the settings active nav at
 * 3.04:1 (PR #287), --error-text at 4.05:1 on --bg-tertiary, and --accent-text
 * at 4.40:1 on the same surface. Every one was found by measuring by hand, late.
 *
 * A token whose NAME says it is body text has one job. This asserts it.
 */
const CSS = readFileSync(join(__dirname, 'styles.css'), 'utf8');

function token(name: string): string {
  const m = CSS.match(new RegExp(`--${name}:\\s*(#[0-9a-fA-F]{6})\\b`));
  if (!m) throw new Error(`token --${name} not found in styles.css`);
  return m[1];
}

const channel = (c: number) => {
  const s = c / 255;
  return s <= 0.03928 ? s / 12.92 : Math.pow((s + 0.055) / 1.055, 2.4);
};

function luminance(hex: string): number {
  const n = parseInt(hex.slice(1), 16);
  return (
    0.2126 * channel((n >> 16) & 255) + 0.7152 * channel((n >> 8) & 255) + 0.0722 * channel(n & 255)
  );
}

function ratio(fg: string, bg: string): number {
  const a = luminance(fg);
  const b = luminance(bg);
  return (Math.max(a, b) + 0.05) / (Math.min(a, b) + 0.05);
}

/** Every surface a body-text token can legitimately land on. */
const SURFACES = ['bg-primary', 'bg-secondary', 'bg-tertiary'] as const;

/**
 * Tokens whose name promises readable text. `--text-disabled` is deliberately
 * excluded and documented as intentionally sub-AA (TC-1) — an exemption with a
 * reason, not an oversight.
 */
const BODY_TEXT_TOKENS = [
  'text-primary',
  'text-secondary',
  'text-muted',
  'error-text',
  'accent-text',
  'info-text',
] as const;

describe('styles.css — text tokens clear WCAG AA on every surface', () => {
  const cases = BODY_TEXT_TOKENS.flatMap((t) => SURFACES.map((s) => [t, s] as const));

  it.each(cases)('--%s on --%s is at least 4.5:1', (tokenName, surface) => {
    const r = ratio(token(tokenName), token(surface));
    expect(
      Number(r.toFixed(2)),
      `--${tokenName} (${token(tokenName)}) on --${surface} (${token(surface)}) is ${r.toFixed(2)}:1`
    ).toBeGreaterThanOrEqual(4.5);
  });

  // Kept honest: this token IS below AA and that is a recorded decision, not drift.
  it('--text-disabled stays the one documented sub-AA exemption', () => {
    expect(CSS).toMatch(/--text-disabled:.*intentionally sub-AA/);
    expect(ratio(token('text-disabled'), token('bg-primary'))).toBeLessThan(4.5);
  });
});
