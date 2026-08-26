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

/**
 * Split styles.css into per-theme blocks BEFORE matching any token.
 *
 * The bug this replaces: `CSS.match(/--x:\s*(#[0-9a-f]{6})/)` has no /g, so it
 * returned the FIRST occurrence in the whole FILE — every assertion measured
 * the `:root` (dark) values no matter which theme it claimed to test. Proven
 * 2026-08-26: with a light block declaring #eae4d6 text on a #faf6ea ground
 * (1.10:1 — literally invisible) this suite reported 30/30 GREEN. That would
 * have been the fourth instance of the defect class the docstring above warns
 * about, pre-installed on the day the second theme shipped.
 */
function themeBlock(label: string, head: RegExp): string {
  const m = CSS.match(head);
  if (!m) throw new Error(`theme block "${label}" not found in styles.css`);
  return m[1];
}

const THEMES = {
  dark: themeBlock('dark (:root)', /^:root\s*\{([\s\S]*?)^\}/m),
  light: themeBlock(
    'light ([data-theme="light"])',
    /^\[data-theme=['"]light['"]\]\s*\{([\s\S]*?)^\}/m
  ),
} as const;
type Theme = keyof typeof THEMES;
const ALL_THEMES = Object.keys(THEMES) as Theme[];

/**
 * Literal 6-digit hex ONLY, per theme. The literal requirement is a FEATURE: a
 * color-mix(), oklch() or var() alias makes this THROW rather than silently
 * measure nothing. Do not "modernise" it.
 */
function token(theme: Theme, name: string): string {
  const m = THEMES[theme].match(new RegExp(`--${name}:\\s*(#[0-9a-fA-F]{6})\\b`));
  if (!m) throw new Error(`token --${name} not found in the ${theme} block of styles.css`);
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

/** Composite an #RRGGBBAA tint over an opaque surface. */
function flatten(tint8: string, surface: string): string {
  const n = parseInt(tint8.slice(1, 7), 16);
  const a = parseInt(tint8.slice(7, 9), 16) / 255;
  const b = parseInt(surface.slice(1), 16);
  const mix = (x: number, y: number) => Math.round(x * a + y * (1 - a));
  const r = mix((n >> 16) & 255, (b >> 16) & 255);
  const g = mix((n >> 8) & 255, (b >> 8) & 255);
  const bl = mix(n & 255, b & 255);
  return `#${((r << 16) | (g << 8) | bl).toString(16).padStart(6, '0')}`;
}

function token8(theme: Theme, name: string): string {
  const m = THEMES[theme].match(new RegExp(`--${name}:\\s*(#[0-9a-fA-F]{8})\\b`));
  if (!m) throw new Error(`8-digit token --${name} not found in the ${theme} block of styles.css`);
  return m[1];
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
  'warning-text',
  'success-text',
] as const;

/**
 * Each *-text token paired with the family tint it usually sits on (an error
 * message inside an error banner).
 */
const TINT_PAIRS = [
  ['error-text', 'error-tint'],
  ['accent-text', 'accent-tint'],
  ['info-text', 'info-tint'],
  ['warning-text', 'warning-tint'],
  ['success-text', 'success-tint'],
] as const;

describe.each(ALL_THEMES)('styles.css [%s] — token contrast gate', (theme) => {
  const tk = (n: string) => token(theme, n);
  const tk8 = (n: string) => token8(theme, n);
  const at = (fg: string, bg: string) => Number(ratio(fg, bg).toFixed(2));

  describe('text tokens clear WCAG AA on every surface', () => {
    const cases = BODY_TEXT_TOKENS.flatMap((t) => SURFACES.map((s) => [t, s] as const));

    it.each(cases)('--%s on --%s is at least 4.5:1', (tokenName, surface) => {
      const r = at(tk(tokenName), tk(surface));
      expect(
        r,
        `[${theme}] --${tokenName} (${tk(tokenName)}) on --${surface} (${tk(surface)}) is ${r}:1`
      ).toBeGreaterThanOrEqual(4.5);
    });
  });

  // The plain-surface cases above have a blind spot this suite was caught by:
  // --warning (#f59e0b) passes every plain surface yet measures 4.31:1 on its
  // OWN tint over --bg-tertiary — and semantic text usually sits on its own
  // tint (an error message inside an error banner).
  //
  // Widened 2026-08-26 from --bg-tertiary alone to ALL THREE grounds: tertiary
  // being the worst case is a coincidence of dark (it is the lightest ground
  // there, the darkest in light), and a gate should not rest on a coincidence.
  describe('text tokens clear AA on their own tint, over every surface', () => {
    const cases = TINT_PAIRS.flatMap(([text, tint]) =>
      SURFACES.map((s) => [text, tint, s] as const)
    );

    it.each(cases)('--%s on its own --%s over --%s is ≥4.5:1', (text, tint, surface) => {
      const composited = flatten(tk8(tint), tk(surface));
      const r = at(tk(text), composited);
      expect(
        r,
        `[${theme}] --${text} (${tk(text)}) on ${composited} (--${tint} over --${surface}) is ${r}:1`
      ).toBeGreaterThanOrEqual(4.5);
    });
  });

  // --accent-subtle is the PR #287 surface (the settings active nav at 3.04:1)
  // and was ungated until now.
  //
  // Scoped to --accent-text and --text-primary ON PURPOSE. --text-muted on this
  // wash measures 3.92:1 in DARK, so gating it would fail a theme this change
  // does not touch — and the pairing is not live: SidebarNavItem.tsx:78 paints
  // --text-muted only in the BASE state, and overrides to --accent-hover under
  // data-[status=active], which is the only state that renders --accent-subtle.
  describe('the active-nav wash (--accent-subtle) carries its foregrounds', () => {
    const FOREGROUNDS = ['accent-text', 'text-primary'] as const;
    const cases = FOREGROUNDS.flatMap((fg) => SURFACES.map((s) => [fg, s] as const));

    it.each(cases)('--%s on --accent-subtle over --%s is ≥4.5:1', (fg, surface) => {
      const composited = flatten(tk8('accent-subtle'), tk(surface));
      const r = at(tk(fg), composited);
      expect(
        r,
        `[${theme}] --${fg} (${tk(fg)}) on ${composited} (--accent-subtle over --${surface}) is ${r}:1`
      ).toBeGreaterThanOrEqual(4.5);
    });

    // SidebarNavItem.tsx:78 and :113 render an ICON in --accent-hover on that
    // same wash. WCAG 1.4.11 gives non-text UI a 3:1 floor, not 4.5:1.
    it.each(SURFACES)(
      '--accent-hover as an ICON on --accent-subtle over --%s is ≥3:1',
      (surface) => {
        const composited = flatten(tk8('accent-subtle'), tk(surface));
        const r = at(tk('accent-hover'), composited);
        expect(
          r,
          `[${theme}] --accent-hover (${tk('accent-hover')}) on ${composited} is ${r}:1`
        ).toBeGreaterThanOrEqual(3);
      }
    );
  });

  // --accent-hover is written as a TEXT colour at six live sites (ActivityHub,
  // SidebarGroupParent, GlossaryRowV2, BatchSubtitleDialog). Nothing gated it
  // in either theme.
  describe('--accent-hover as text', () => {
    it.each(SURFACES)('--accent-hover on --%s is ≥4.5:1', (surface) => {
      const r = at(tk('accent-hover'), tk(surface));
      expect(
        r,
        `[${theme}] --accent-hover (${tk('accent-hover')}) on --${surface} (${tk(surface)}) is ${r}:1`
      ).toBeGreaterThanOrEqual(4.5);
    });
  });

  // --text-on-accent is the label that sits on a SOLID semantic fill. Ungated
  // until now, which is how GlossaryRowV2.tsx:185 shipped at 3.33:1.
  //
  // --error and --error-pressed are not here: 硃砂 does not invert, so its
  // label does not either — every cinnabar fill now carries --text-on-scrim
  // (5.03:1 / 6.36:1 in BOTH themes), asserted in its own block below. That
  // closes GlossaryRowV2.tsx:185, which shipped at 3.33:1 in dark.
  describe('--text-on-accent carries every solid fill it lands on', () => {
    const SOLIDS = [
      'accent-primary',
      'accent-hover',
      'accent-pressed',
      'success',
      'warning',
      'warning-pressed',
      'info',
    ] as const;

    it.each(SOLIDS)('--text-on-accent on --%s is ≥4.5:1', (solid) => {
      const r = at(tk('text-on-accent'), tk(solid));
      expect(
        r,
        `[${theme}] --text-on-accent (${tk('text-on-accent')}) on --${solid} (${tk(solid)}) is ${r}:1`
      ).toBeGreaterThanOrEqual(4.5);
    });
  });

  // The poster scrim is the one surface that does NOT invert — a photograph
  // needs a dark veil in daylight too — so --text-on-scrim is the one text
  // token declared identically in both blocks. Gated over the three extremes
  // of key art, because the scrim is translucent and the art shows through:
  // a white poster is the worst case and it is a real one.
  describe('--text-on-scrim carries the poster scrim over any key art', () => {
    const KEY_ART = [
      ['black art', '#000000'],
      ['mid-grey art', '#808080'],
      ['white art', '#ffffff'],
    ] as const;

    it.each(KEY_ART)('--text-on-scrim over the scrim on %s is ≥4.5:1', (_label, art) => {
      const veiled = flatten(token8(theme, 'overlay-scrim'), art);
      const r = at(tk('text-on-scrim'), veiled);
      expect(
        r,
        `[${theme}] --text-on-scrim (${tk('text-on-scrim')}) on ${veiled} (scrim over ${_label}) is ${r}:1`
      ).toBeGreaterThanOrEqual(4.5);
    });
  });

  // 硃砂 is the one pigment that survives inversion unchanged, so a label on
  // it must not invert either. --text-on-accent measures 3.33 / 2.63 here in
  // dark, which is how GlossaryRowV2.tsx:185 shipped broken.
  describe('--text-on-scrim carries the cinnabar fills', () => {
    it.each(['error', 'error-pressed'] as const)('--text-on-scrim on --%s is ≥4.5:1', (solid) => {
      const r = at(tk('text-on-scrim'), tk(solid));
      expect(
        r,
        `[${theme}] --text-on-scrim (${tk('text-on-scrim')}) on --${solid} (${tk(solid)}) is ${r}:1`
      ).toBeGreaterThanOrEqual(4.5);
    });
  });

  // WCAG 1.4.11: a control's own boundary owes 3:1. --focus-ring is applied
  // app-wide from one rule in styles.css, so a value that cannot be seen on a
  // raised card is a keyboard user's problem on every surface at once.
  describe('non-text UI clears the 3:1 boundary floor', () => {
    const UI_TOKENS = ['accent-primary', 'focus-ring'] as const;
    const cases = UI_TOKENS.flatMap((t) => SURFACES.map((s) => [t, s] as const));

    it.each(cases)('--%s on --%s is ≥3:1', (tokenName, surface) => {
      const r = at(tk(tokenName), tk(surface));
      expect(
        r,
        `[${theme}] --${tokenName} (${tk(tokenName)}) on --${surface} (${tk(surface)}) is ${r}:1`
      ).toBeGreaterThanOrEqual(3);
    });
  });

  // Kept honest: this token IS below AA and that is a recorded decision, not
  // drift. Asserted PER THEME — a naive inversion of the dark value measures
  // 4.69:1 on paper, which would PASS and silently void the exemption.
  it('--text-disabled stays the one documented sub-AA exemption', () => {
    expect(CSS).toMatch(/--text-disabled:.*intentionally sub-AA/);
    expect(at(tk('text-disabled'), tk('bg-primary'))).toBeLessThan(4.5);
  });
});

// A token added to :root and forgotten in the light block would silently
// inherit the dark value — invisible to every assertion above, because each one
// reads its own theme's block and would simply never see the omission.
describe('the two theme blocks declare the same token set', () => {
  const names = (block: string) =>
    new Set([...block.matchAll(/(--[a-z0-9-]+):/g)].map((m) => m[1]));
  // Theme-independent: geometry and spacing live only in :root by design.
  const THEME_INDEPENDENT = /^--(radius|gap)-/;

  it('every theme-dependent token in :root is redeclared for light', () => {
    const dark = [...names(THEMES.dark)].filter((n) => !THEME_INDEPENDENT.test(n));
    const light = names(THEMES.light);
    expect(dark.filter((n) => !light.has(n))).toEqual([]);
  });

  it('light declares nothing that :root does not', () => {
    const dark = names(THEMES.dark);
    expect([...names(THEMES.light)].filter((n) => !dark.has(n))).toEqual([]);
  });
});
