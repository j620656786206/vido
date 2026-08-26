import { describe, it, expect, afterEach, vi } from 'vitest';
import { prefersReducedMotion, scrollByMotionSafe } from './motion';

/**
 * This module exists for ONE reason — `scrollBy({ behavior: 'smooth' })`
 * ignores `scroll-behavior: auto`, so the CSS reduced-motion net in styles.css
 * provably cannot reach it. That makes the reduced-motion branch the whole
 * product, and an untested branch here is a promise nobody checked.
 *
 * test-setup.ts installs a `matches: false` matchMedia only when the global is
 * missing, and documents that specs which care override it — so these stub it
 * explicitly rather than relying on the default either way.
 */
function stubMatchMedia(matches: boolean) {
  const seen: string[] = [];
  vi.stubGlobal(
    'matchMedia',
    vi.fn((q: string) => {
      seen.push(q);
      return { matches, media: q, addEventListener() {}, removeEventListener() {} };
    }) as unknown as typeof window.matchMedia
  );
  return seen;
}

afterEach(() => vi.unstubAllGlobals());

describe('prefersReducedMotion', () => {
  it('asks for the reduce preference specifically', () => {
    const seen = stubMatchMedia(true);
    expect(prefersReducedMotion()).toBe(true);
    expect(seen).toEqual(['(prefers-reduced-motion: reduce)']);
  });

  it('is false when the user expressed no preference', () => {
    stubMatchMedia(false);
    expect(prefersReducedMotion()).toBe(false);
  });

  it('survives a DOM with no matchMedia rather than throwing', () => {
    vi.stubGlobal('matchMedia', undefined);
    expect(prefersReducedMotion()).toBe(false);
  });

  // A module-scope boolean would answer with whatever was true at import time.
  // Someone can flip the OS setting with the tab open, so this reads live.
  it('re-reads on every call, so flipping the OS setting mid-session is honoured', () => {
    stubMatchMedia(false);
    expect(prefersReducedMotion()).toBe(false);
    stubMatchMedia(true);
    expect(prefersReducedMotion()).toBe(true);
  });
});

describe('scrollByMotionSafe', () => {
  const spyEl = () =>
    ({ scrollBy: vi.fn() }) as unknown as Element & { scrollBy: ReturnType<typeof vi.fn> };

  it('scrolls smoothly when no preference is set', () => {
    stubMatchMedia(false);
    const el = spyEl();
    scrollByMotionSafe(el, { left: 240 });
    expect(el.scrollBy).toHaveBeenCalledWith({ left: 240, behavior: 'smooth' });
  });

  /**
   * The one that matters. Note it still SCROLLS — the user asked the row to
   * move and the row must move. Reduced motion means it arrives instead of
   * travelling, never that the control stops working.
   */
  it('still scrolls under reduced motion, but arrives instead of travelling', () => {
    stubMatchMedia(true);
    const el = spyEl();
    scrollByMotionSafe(el, { left: -240 });
    expect(el.scrollBy).toHaveBeenCalledWith({ left: -240, behavior: 'auto' });
    expect(el.scrollBy).toHaveBeenCalledTimes(1);
  });

  it('never lets a caller pass its own behavior through', () => {
    stubMatchMedia(true);
    const el = spyEl();
    scrollByMotionSafe(el, { left: 10, behavior: 'smooth' } as { left: number });
    expect(el.scrollBy).toHaveBeenCalledWith({ left: 10, behavior: 'auto' });
  });
});
