// Design ref: ux-design.pen Screen I11-D (Qaz1x) — 篩選 Rail 收合／展開動態 spec（桌面）
/**
 * disc-2026-10-filter-rail-toggle-no-motion: the ONE way the desktop filter rail
 * is collapsed or expanded, shared by 媒體庫 (LibraryBrowseV2) and 探索
 * (DiscoverBrowseV2) so both pages move the same way (I11-D 適用範圍).
 *
 * The motion itself is the browser's View Transitions API: we flip the state
 * inside `document.startViewTransition`, and the `::view-transition-*` rules in
 * styles.css (scoped to `:root[data-rail-motion]`) slide the rail out/in, morph
 * the 「篩選」 title into the toolbar button and cross-fade the poster grid.
 *
 * Three things this hook guarantees, whichever path runs:
 *   - the DOM is updated SYNCHRONOUSLY (flushSync) and focus is moved right after,
 *     so the collapse button that just unmounted never drops focus onto <body>;
 *   - reduced motion never starts a transition (see the CSS note in styles.css —
 *     the `*` reduced-motion net cannot reach `::view-transition-*` pseudos);
 *   - only a click animates. Mount / reload / route change read the remembered
 *     state straight from localStorage and never come through here.
 */
import { useCallback } from 'react';
import { flushSync } from 'react-dom';
import { prefersReducedMotion } from '../lib/motion';

/** Set on <html> for the life of one transition; styles.css keys every rule off it. */
export const RAIL_MOTION_ATTR = 'data-rail-motion';

/**
 * Which click owns `data-rail-motion` right now. Module scope, not a ref: the
 * attribute is document-global, so the ownership token has to be too. A second
 * click mid-transition makes the browser skip the first one, whose `finished`
 * then settles while the second is still running — only the LATEST transition
 * may take the attribute off, or the second animation loses its names halfway.
 */
let motionOwner = 0;

interface Options {
  /** The page's state setter (it also persists to localStorage). */
  setRailCollapsed: (next: boolean) => void;
  /** Focus target once the rail is gone — the toolbar 篩選 button, or a fallback. */
  getCollapsedFocusTarget: () => HTMLElement | null;
  /** Focus target once the rail is back — its collapse button. */
  getExpandedFocusTarget: () => HTMLElement | null;
}

export function useFilterRailTransition({
  setRailCollapsed,
  getCollapsedFocusTarget,
  getExpandedFocusTarget,
}: Options) {
  const toggle = useCallback(
    (next: boolean) => {
      const commit = () => {
        // flushSync is load-bearing (and the first use in this repo): the browser
        // photographs the NEW picture as soon as this callback returns. A batched
        // React update would still be pending then, so the "after" shot would be
        // the "before" shot and nothing would animate. The no-API path needs it
        // too: tests and callers assert on the DOM right after the click.
        flushSync(() => setRailCollapsed(next));
        // Only now does the target exist (the expand button / the rail just mounted).
        (next ? getCollapsedFocusTarget : getExpandedFocusTarget)()?.focus();
      };

      // lib.dom types startViewTransition as always present, so `if (document.
      // startViewTransition)` would type-check as always-true — test the runtime value.
      if (typeof document.startViewTransition !== 'function' || prefersReducedMotion()) {
        commit();
        return;
      }

      const root = document.documentElement;
      const owner = ++motionOwner;
      root.setAttribute(RAIL_MOTION_ATTR, next ? 'collapse' : 'expand');
      const transition = document.startViewTransition(commit);
      // `finished` settles whether the transition ran, was skipped by a newer one, or
      // its update callback threw — any of those ends this click's claim on the attribute.
      transition.finished
        .catch(() => undefined)
        .finally(() => {
          if (owner === motionOwner) root.removeAttribute(RAIL_MOTION_ATTR);
        });
    },
    [setRailCollapsed, getCollapsedFocusTarget, getExpandedFocusTarget]
  );

  const collapse = useCallback(() => toggle(true), [toggle]);
  const expand = useCallback(() => toggle(false), [toggle]);
  return { collapse, expand };
}
