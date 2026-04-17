import { useEffect, useState, type RefObject } from 'react';

interface UseInViewportOptions {
  rootMargin?: string;
  threshold?: number | number[];
  // If true, the hook stops observing after the first intersection.
  // Story 10-5 Task 2.3 — once a block becomes visible we never want to tear
  // down the query that was just fired, so "once" is the right default.
  once?: boolean;
}

/**
 * Tracks whether a DOM node is inside (or within rootMargin of) the viewport.
 *
 * Story 10-5 AC #2 / Task 2.3 — used by ExploreBlocksList to gate the
 * per-block content fetch until the block approaches the viewport, so the
 * above-the-fold blocks win the network race for LCP.
 *
 * The hook is SSR-safe (returns false when IntersectionObserver is absent)
 * and unmounts the observer on cleanup. The `once` flag latches visibility
 * permanently so the caller can keep data cached even if the user scrolls
 * back up past the element.
 */
export function useInViewport(
  ref: RefObject<Element | null>,
  { rootMargin = '0px', threshold = 0, once = true }: UseInViewportOptions = {}
): boolean {
  const [isInViewport, setIsInViewport] = useState(false);

  useEffect(() => {
    const node = ref.current;
    if (!node) return;
    if (typeof IntersectionObserver === 'undefined') {
      // No observer (SSR / legacy jsdom) — fall back to visible so we do not
      // strand content forever.
      setIsInViewport(true);
      return;
    }

    const observer = new IntersectionObserver(
      (entries) => {
        for (const entry of entries) {
          if (entry.isIntersecting) {
            setIsInViewport(true);
            if (once) {
              observer.disconnect();
            }
            return;
          }
        }
        if (!once) {
          setIsInViewport(false);
        }
      },
      { rootMargin, threshold }
    );

    observer.observe(node);
    return () => observer.disconnect();
  }, [ref, rootMargin, threshold, once]);

  return isInViewport;
}
