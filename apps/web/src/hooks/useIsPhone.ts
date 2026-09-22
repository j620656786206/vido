import { useSyncExternalStore } from 'react';

/**
 * The phone breakpoint: below Tailwind's `sm`, which is `40rem` — the same query
 * `max-sm:` compiles to and the one `MobileTabBar`'s `sm:hidden` flips on, so "a
 * phone" here and "the tab bar is showing" never disagree. In rem, not 640px: a
 * rem media query follows the browser's default font size (a 20px default makes
 * `sm` 800px), and a px query here would part ways with the CSS layout.
 *
 * Asked as "narrower than" and read straight from `matches`, never as a negated
 * `min-width`: `test-setup.ts` stubs `matchMedia` to `matches: false` for every
 * query, and a negation would put every jsdom spec on the phone path.
 */
export const PHONE_MQ = '(width < 40rem)';

function subscribe(cb: () => void) {
  if (typeof window.matchMedia !== 'function') return () => {};
  const mql = window.matchMedia(PHONE_MQ);
  mql.addEventListener('change', cb);
  return () => mql.removeEventListener('change', cb);
}

function getSnapshot() {
  return typeof window.matchMedia === 'function' ? window.matchMedia(PHONE_MQ).matches : false;
}

/** True below 640px. `false` without `matchMedia` and on the server. */
export function useIsPhone(): boolean {
  return useSyncExternalStore(subscribe, getSnapshot, () => false);
}
