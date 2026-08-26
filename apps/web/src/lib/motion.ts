/**
 * The JS half of the motion contract.
 *
 * styles.css owns everything CSS can express — the duration and distance
 * tokens, and the `*` reduced-motion net. This file exists for the one thing
 * CSS provably cannot reach:
 *
 *   `element.scrollBy({ behavior: 'smooth' })` IGNORES `scroll-behavior: auto`.
 *   An explicit JS behavior argument outranks the CSS property, so a
 *   reduced-motion user still gets a full smooth pan. A vestibular trigger does
 *   not care which layer started the movement — the guard has to live here.
 *
 * Read live rather than cached at module load: someone can flip the OS setting
 * with the tab open, and a module-scope boolean would keep the old answer for
 * the life of the session.
 */

const REDUCED_MOTION_QUERY = '(prefers-reduced-motion: reduce)';

/** True when the OS asks for less motion. Safe in SSR and in test DOMs with no matchMedia. */
export function prefersReducedMotion(): boolean {
  if (typeof window === 'undefined' || typeof window.matchMedia !== 'function') return false;
  return window.matchMedia(REDUCED_MOTION_QUERY).matches;
}

/**
 * scrollBy that respects the preference: it still SCROLLS — the user asked to
 * move the row and the row must move — it just arrives instead of travelling.
 * Reduced motion means gentler, never unresponsive.
 */
export function scrollByMotionSafe(el: Element, options: { left?: number; top?: number }): void {
  el.scrollBy({ ...options, behavior: prefersReducedMotion() ? 'auto' : 'smooth' });
}
