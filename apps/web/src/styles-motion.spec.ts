import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
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
