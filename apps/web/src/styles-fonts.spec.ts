import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import { join } from 'node:path';

/**
 * Font-family guards (DESIGN.md §Typography). Both faces ship with Vido, so the
 * NAS renders the design even offline. Two of these traps have already sprung
 * once: a `.font-mono` rule in `@layer base` sat dead for months because
 * Tailwind's own `font-mono` utility (utilities layer) always wins, and a
 * `.font-primary` class that no component used meant Noto never drew a glyph.
 *
 * Whether the face actually paints is a real-browser question — see
 * tests/e2e/font-mono.spec.ts and tests/e2e/font-sans.spec.ts. jsdom does not
 * resolve CSS, so this file only guards the source.
 */
const CSS = readFileSync(join(__dirname, 'styles.css'), 'utf8');
const MAIN = readFileSync(join(__dirname, 'main.tsx'), 'utf8');
const INDEX_HTML = readFileSync(join(__dirname, '..', 'index.html'), 'utf8');

/** Brace-matched contents of the first block that starts at `open`. */
function blockAt(open: number): string {
  let depth = 0;
  for (let i = CSS.indexOf('{', open); i < CSS.length; i++) {
    if (CSS[i] === '{') depth++;
    else if (CSS[i] === '}' && --depth === 0) return CSS.slice(open, i + 1);
  }
  throw new Error('unbalanced block');
}

function layerBase(): string {
  const open = CSS.indexOf('@layer base {');
  expect(open, '@layer base not found in styles.css').toBeGreaterThan(-1);
  return blockAt(open);
}

/** The font stack declared for `--<name>` inside an `@theme` block, as a list. */
function themeStack(name: string): string[] {
  const m = CSS.match(new RegExp(`@theme \\{[^}]*?--${name}:\\s*([^;]+);`));
  expect(m, `--${name} not declared in an @theme block`).not.toBeNull();
  return m![1].split(',').map((s) => s.trim().replace(/\s+/g, ' '));
}

describe('--font-sans (body text)', () => {
  const stack = themeStack('font-sans');

  it('leads with the bundled Noto Sans TC variable face', () => {
    expect(stack[0]).toBe("'Noto Sans TC Variable'");
  });

  it('names the Traditional Chinese system faces before the generic sans-serif', () => {
    const generic = stack.indexOf('sans-serif');
    expect(generic).toBeGreaterThan(-1);
    expect(stack.indexOf("'PingFang TC'")).toBeGreaterThan(0);
    expect(stack.indexOf("'PingFang TC'")).toBeLessThan(generic);
    expect(stack.indexOf("'Microsoft JhengHei'")).toBeGreaterThan(0);
    expect(stack.indexOf("'Microsoft JhengHei'")).toBeLessThan(generic);
  });

  it('keeps the emoji faces at the tail (UI strings carry 🎬🔍🔴)', () => {
    expect(stack.slice(-4)).toEqual([
      "'Apple Color Emoji'",
      "'Segoe UI Emoji'",
      "'Segoe UI Symbol'",
      "'Noto Color Emoji'",
    ]);
  });
});

describe('--font-mono (readouts)', () => {
  const stack = themeStack('font-mono');

  it('leads with the bundled JetBrains Mono variable face', () => {
    expect(stack[0]).toBe("'JetBrains Mono Variable'");
  });

  it('falls back to Noto Sans TC for CJK in paths, before the generic monospace', () => {
    const noto = stack.indexOf("'Noto Sans TC Variable'");
    expect(noto).toBeGreaterThan(0);
    expect(stack.indexOf('monospace')).toBe(stack.length - 1);
    expect(noto).toBe(stack.length - 2);
  });

  it('turns ligatures off, so `--` and `...` in file names read as typed', () => {
    expect(CSS).toMatch(/--font-mono--font-feature-settings:\s*'calt' 0,\s*'liga' 0;/);
  });
});

describe('dead font rules stay gone', () => {
  it('@layer base declares no font-family on .font-mono (it would always lose to the utility)', () => {
    expect(layerBase()).not.toMatch(/\.font-mono\s*\{/);
  });

  it('there is no .font-primary class anywhere in styles.css', () => {
    expect(CSS).not.toContain('.font-primary');
  });
});

describe('the faces are bundled, not fetched', () => {
  it('main.tsx imports the upright JetBrains Mono variable CSS', () => {
    expect(MAIN).toContain("import '@fontsource-variable/jetbrains-mono/wght.css';");
    expect(MAIN).not.toContain('wght-italic.css');
  });

  it('main.tsx imports the Noto Sans TC variable CSS', () => {
    expect(MAIN).toContain("import '@fontsource-variable/noto-sans-tc/wght.css';");
  });

  it('no font CDN is referenced from the stylesheet or the page shell', () => {
    for (const src of [CSS, INDEX_HTML, MAIN]) {
      expect(src).not.toMatch(/fonts\.googleapis|fonts\.gstatic|cdn\.jsdelivr|unpkg/);
    }
  });
});

describe('page language', () => {
  it('index.html declares Traditional Chinese (Taiwan), so readers and font fallback treat it as zh-TW', () => {
    expect(INDEX_HTML).toMatch(/<html lang="zh-Hant-TW">/);
  });
});
