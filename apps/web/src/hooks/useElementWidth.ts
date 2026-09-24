import { useLayoutEffect, useState, type RefObject } from 'react';

/**
 * The rendered width of `ref`'s element, kept current with a ResizeObserver.
 * `null` until measured — and forever where ResizeObserver does not exist
 * (jsdom), so callers must treat `null` as "unknown", not as zero.
 *
 * For layouts that depend on the space a component actually GETS rather than
 * on the viewport: in the settings pages the sidebar takes 240px from 640 up,
 * so "the viewport is wider than 640" says little about the column.
 */
export function useElementWidth(ref: RefObject<HTMLElement | null>): number | null {
  const [width, setWidth] = useState<number | null>(null);
  useLayoutEffect(() => {
    const el = ref.current;
    if (!el || typeof window.ResizeObserver === 'undefined') return;
    setWidth(el.getBoundingClientRect().width);
    const ro = new window.ResizeObserver(([entry]) => setWidth(entry.contentRect.width));
    ro.observe(el);
    return () => ro.disconnect();
  }, [ref]);
  return width;
}
