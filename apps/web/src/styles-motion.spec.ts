import { describe, it, expect } from 'vitest';
import { readFileSync, readdirSync, statSync } from 'node:fs';
import { join } from 'node:path';

/**
 * The motion vocabulary's structural guards — the sibling of
 * styles-contrast.spec.ts, and for the same reason: these are rules a reviewer
 * cannot see by reading a diff, and two of them are silent when broken.
 *
 * Nothing here measures taste. Each assertion is a trap this stylesheet has
 * already fallen into once, or would fall into on the next tidy-up.
 */
const CSS = readFileSync(join(__dirname, 'styles.css'), 'utf8');

/** The contents of `@layer base { ... }` — brace-matched, since it nests. */
function layerBase(): string {
  const open = CSS.indexOf('@layer base {');
  expect(open, '@layer base not found in styles.css').toBeGreaterThan(-1);
  let depth = 0;
  for (let i = CSS.indexOf('{', open); i < CSS.length; i++) {
    if (CSS[i] === '{') depth++;
    else if (CSS[i] === '}' && --depth === 0) return CSS.slice(open, i + 1);
  }
  throw new Error('@layer base is unbalanced');
}

/** The reduced-motion block that redeclares the tokens (not the `*` net). */
function reducedMotionTokenBlock(): string {
  const m = CSS.match(
    /@media \(prefers-reduced-motion: reduce\) \{\s*:root,\s*\[data-theme='light'\] \{([\s\S]*?)\}/
  );
  expect(m, 'the reduced-motion TOKEN override block was not found').not.toBeNull();
  return m![1];
}

describe('motion tokens', () => {
  const TIME = ['motion-touch', 'motion-state', 'motion-move', 'motion-arrive', 'motion-leave'];
  const DISTANCE = ['motion-lift', 'motion-press', 'motion-rise', 'motion-turn'];

  it.each([...TIME, ...DISTANCE, 'breath'])('--%s is declared in :root', (name) => {
    expect(CSS).toMatch(new RegExp(`^\\s*--${name}:`, 'm'));
  });

  it.each(['ease-settle', 'ease-leave'])('--%s is a cubic-bezier, never a keyword', (name) => {
    expect(CSS).toMatch(new RegExp(`--${name}:\\s*cubic-bezier\\(`));
  });

  // 輕功: quick off the ground, slow to settle, NEVER a rebound. A spring or
  // elastic curve overshoots past 1, which puts weight back into a world built
  // on weightlessness. Catch it by the y2 control point.
  it.each(['ease-settle', 'ease-leave'])('--%s does not overshoot (no bounce)', (name) => {
    const m = CSS.match(new RegExp(`--${name}:\\s*cubic-bezier\\(([^)]*)\\)`));
    const [, y1, , y2] = m![1].split(',').map((n) => Number(n.trim()));
    expect(Math.max(y1, y2), `--${name} overshoots past 1 — that is a spring`).toBeLessThanOrEqual(
      1
    );
  });

  it('exit is faster than entrance', () => {
    const ms = (n: string) => Number(CSS.match(new RegExp(`--${n}:\\s*(\\d+)ms`))![1]);
    expect(ms('motion-leave')).toBeLessThan(ms('motion-arrive'));
  });

  // The breath is a claim that work is happening. Fast enough and it stops
  // reading as 活著 and starts reading as 警告 — and alarm belongs to amber,
  // not to motion. (styles.css motion block, licence ③.)
  it('the in-flight breath is slow enough to read as alive, not as alarm', () => {
    expect(Number(CSS.match(/--breath:\s*(\d+)ms/)![1])).toBeGreaterThanOrEqual(1500);
  });

  /**
   * ⚠️ THE ONE THAT SHIPPED BROKEN ONCE. A press state written as
   * `active:scale-100` reads like feedback and is a NO-OP: 1 is where the
   * element already sits. On desktop the bug hides, because hover has moved
   * the card to 1.02 first and the press appears to settle it. On a phone
   * there is no hover, so the tap does nothing at all — on the surface
   * PRODUCT.md calls 同等重要. The press value must be BELOW rest, or it is
   * not a press.
   */
  it('--motion-press goes BELOW rest — a press that lands on 1 is a no-op on touch', () => {
    const press = Number(CSS.match(/--motion-press:\s*([\d.]+)\s*;/)![1]);
    const lift = Number(CSS.match(/--motion-lift:\s*([\d.]+)\s*;/)![1]);
    expect(press, '--motion-press must be < 1 or touch gets no feedback').toBeLessThan(1);
    expect(lift, '--motion-lift must be > 1 or hover gets no feedback').toBeGreaterThan(1);
  });
});

describe('prefers-reduced-motion', () => {
  const block = reducedMotionTokenBlock();

  it('every TIME token collapses', () => {
    for (const n of [
      'motion-touch',
      'motion-state',
      'motion-move',
      'motion-arrive',
      'motion-leave',
    ])
      expect(block, `--${n} has no reduced-motion override`).toMatch(
        new RegExp(`--${n}:\\s*[01]ms`)
      );
  });

  // The whole reason distance is a separate axis: collapsing durations does not
  // REMOVE a hover lift, it turns it into an instant jump. Only zeroing the
  // spatial amount actually stops the movement.
  it('every DISTANCE token goes to zero, so nothing snaps into a new position', () => {
    expect(block).toMatch(/--motion-lift:\s*1\s*;/);
    expect(block).toMatch(/--motion-press:\s*1\s*;/);
    expect(block).toMatch(/--motion-rise:\s*0px/);
    expect(block).toMatch(/--motion-turn:\s*0deg/);
  });

  /**
   * ⚠️ THE SILENT ONE. Custom properties declared inside a cascade layer LOSE
   * to unlayered ones, and :root is unlayered. Move this block inside
   * @layer base — which is where its sibling `*` net lives, so it looks like
   * the tidy thing to do — and every override above is discarded with no
   * error, no warning, and no visible difference until someone with reduced
   * motion enabled opens the app.
   */
  it('the token override stays OUTSIDE @layer base or it silently loses the cascade', () => {
    expect(layerBase()).not.toContain('--motion-touch:');
  });

  /**
   * ⚠️ THE HAZARDOUS ONE. Without iteration-count, an INFINITE keyframe (the
   * skeleton pulse, the in-flight breath) keeps looping while its duration is
   * clamped to ~0 — i.e. it strobes, which is the precise harm this media
   * query exists to prevent. The canonical snippet this file was copied from
   * carries the line; this project's copy had dropped it.
   */
  it('the global net stops infinite animations rather than speeding them up', () => {
    expect(CSS).toMatch(/animation-iteration-count:\s*1\s*!important/);
  });

  it('smooth scrolling is switched off too', () => {
    expect(CSS).toMatch(/scroll-behavior:\s*auto\s*!important/);
  });
});

/**
 * ⚠️ THE ONE THAT SHIPPED BROKEN TWICE — first as `active:scale-100`, then as
 * `transition-[opacity,transform]`. Both are the same mistake wearing different
 * clothes: a transition that names a property nothing changes, next to a
 * property that changes and is not named.
 *
 * Tailwind v4 compiles rotate-*, scale-* and translate-* to the INDIVIDUAL
 * transform properties (`.rotate-0 { rotate: none }`), which
 * `transition-property: transform` does not cover. v4's own
 * `transition-transform` expands to `transform, translate, scale, rotate`
 * precisely for this reason — so the built-in is safe and a HAND-WRITTEN
 * arbitrary list naming `transform` is the trap. It fails silently: the
 * element still reaches its final state, it just teleports there, which on a
 * cross-fade looks like a design decision rather than a bug.
 */
describe('Tailwind v4 individual transform properties', () => {
  const SRC = join(__dirname);
  const files: string[] = [];
  (function walk(dir: string) {
    for (const e of readdirSync(dir)) {
      const p = join(dir, e);
      if (statSync(p).isDirectory()) walk(p);
      else if (p.endsWith('.tsx') && !p.includes('.spec.')) files.push(p);
    }
  })(SRC);

  /** An arbitrary transition list — `transition-[a,b]` — that names `transform`. */
  const HANDWRITTEN_TRANSFORM = /transition-\[[^\]]*\btransform\b[^\]]*\]/;
  /** rotate-45 / scale-95 / -translate-x-1/2 / scale-[var(--x)] … */
  const INDIVIDUAL = /(?:^|[\s"'`:])-?(?:rotate|scale|translate-[xyz]?)-[[\w./[\]()-]/;

  it('no component hand-writes a transition list naming `transform` beside an individual transform utility', () => {
    const offenders = files.filter((f) => {
      const src = readFileSync(f, 'utf8');
      return HANDWRITTEN_TRANSFORM.test(src) && INDIVIDUAL.test(src);
    });
    expect(
      offenders.map((f) => f.slice(SRC.length + 1)),
      'Use the built-in `transition-transform` (which covers transform, translate, scale and rotate) or name the individual property — `transition-[opacity,rotate]`.'
    ).toEqual([]);
  });

  it('ThemeToggle transitions `rotate`, the property its faces actually change', () => {
    const src = readFileSync(join(SRC, 'components/shell/ThemeToggle.tsx'), 'utf8');
    const face = src.match(/const face =\s*'([^']*)'/)![1];
    expect(face, 'the face transition list must name rotate').toContain('rotate');
    expect(
      face,
      '`transform` is never set on these faces — naming it hides the missing rotate'
    ).not.toMatch(/transition-\[[^\]]*\btransform\b/);
  });
});

/**
 * ⚠️ THE ONE THAT WENT UNNOTICED FOR MONTHS.
 *
 * `animate-shrink` was declared in apps/web/tailwind.config.js — a v3-shaped
 * config that Tailwind v4 never loads, because styles.css carries no @config.
 * It emitted zero CSS, so ScanProgressCard's auto-dismiss countdown sat frozen
 * at full width while a real 10-second setTimeout destroyed the card the user
 * was reading. The tailwindcss-animate family (animate-in, fade-in-0,
 * slide-in-from-*, zoom-in-*, fill-mode-*) was dead the same way: the plugin
 * that defines it was never installed.
 *
 * Nothing warns you. A class that does not exist is indistinguishable from a
 * class that exists and does nothing — the element simply appears. So the
 * guard has to be "every animation named in source is actually registered".
 */
describe('every animation named in source actually exists', () => {
  const SRC = join(__dirname);
  const files: string[] = [];
  (function walk(dir: string) {
    for (const e of readdirSync(dir)) {
      const p = join(dir, e);
      if (statSync(p).isDirectory()) walk(p);
      else if (/\.tsx?$/.test(p) && !p.includes('.spec.')) files.push(p);
    }
  })(SRC);
  /**
   * Comments are stripped before scanning. A class name inside a comment is
   * not a class — and every one of these rules is documented in prose that
   * quotes the very names it bans, so without this the guard reports its own
   * explanations as violations.
   */
  const stripComments = (s: string) =>
    s.replace(/\/\*[\s\S]*?\*\//g, ' ').replace(/(^|[^:])\/\/[^\n]*/g, '$1');

  const sources = files.map(
    (f) => [f.slice(SRC.length + 1), stripComments(readFileSync(f, 'utf8'))] as const
  );

  /** Registered by Tailwind core — these need no @theme entry. */
  const CORE = new Set(['spin', 'ping', 'pulse', 'bounce', 'none']);
  /** Registered by this project, in the @theme block of styles.css. */
  const REGISTERED = new Set([...CSS.matchAll(/--animate-([a-z0-9-]+):/g)].map((m) => m[1]));
  /** Keyframes this stylesheet defines, for the `animate-[name_…]` arbitrary form. */
  const KEYFRAMES = new Set([...CSS.matchAll(/@keyframes\s+([A-Za-z0-9_-]+)/g)].map((m) => m[1]));

  it('styles.css registers the animations this pass added', () => {
    for (const n of ['breathe', 'overlay-enter', 'overlay-exit', 'dialog-enter', 'dialog-exit'])
      expect(REGISTERED, `--animate-${n} is missing from @theme`).toContain(n);
  });

  it('every `animate-<name>` utility in source is registered somewhere', () => {
    const unknown: string[] = [];
    for (const [file, src] of sources) {
      // Skip the arbitrary form here; it gets its own assertion below.
      for (const m of src.matchAll(/(?:^|[\s"'`:])animate-([a-z][a-z0-9-]*)\b/g)) {
        const name = m[1];
        if (CORE.has(name) || REGISTERED.has(name)) continue;
        unknown.push(`${file}: animate-${name}`);
      }
    }
    expect(
      [...new Set(unknown)],
      'Register it in the @theme block of styles.css (`--animate-<name>: <keyframes> …`) with a matching @keyframes, the way --animate-breathe is. A class Tailwind never emits fails silently.'
    ).toEqual([]);
  });

  it('every `animate-[<keyframes>_…]` arbitrary animation names a real @keyframes', () => {
    const unknown: string[] = [];
    for (const [file, src] of sources)
      for (const m of src.matchAll(/animate-\[([A-Za-z0-9_-]+?)[_\]]/g))
        if (!KEYFRAMES.has(m[1]))
          unknown.push(`${file}: @keyframes ${m[1]} not found in styles.css`);
    expect([...new Set(unknown)]).toEqual([]);
  });

  /**
   * These are tailwindcss-animate's vocabulary. The plugin is deliberately NOT
   * a dependency: it redefines the `duration`, `delay` and `ease` utilities to
   * drive `animation-*` as well as `transition-*`, which would reintroduce
   * exactly the literal durations `local/no-hardcoded-duration` was written to
   * close.
   */
  it('the tailwindcss-animate vocabulary stays out — the plugin is not installed', () => {
    const DEAD =
      /(?:^|[\s"'`:])(animate-in|animate-out|fade-in(?:-\d+)?|fade-out(?:-\d+)?|slide-in-from-[a-z]+-\d+|slide-out-to-[a-z]+-\d+|zoom-in(?:-\d+)?|zoom-out(?:-\d+)?|fill-mode-[a-z]+)\b/;
    const offenders = sources.filter(([, src]) => DEAD.test(src)).map(([f]) => f);
    expect(
      offenders,
      'These come from tailwindcss-animate, which this project does not install — they emit nothing. Use a --animate-* token from styles.css instead.'
    ).toEqual([]);
  });
});
