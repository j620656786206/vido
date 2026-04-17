import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import { useRef } from 'react';
import { useInViewport } from './useInViewport';

type ObserverCb = (entries: IntersectionObserverEntry[]) => void;

describe('useInViewport', () => {
  let capturedCallback: ObserverCb | null;
  let observedElements: Element[];
  let disconnectSpy: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    capturedCallback = null;
    observedElements = [];
    disconnectSpy = vi.fn();

    class Spyable {
      root = null;
      rootMargin = '';
      thresholds: number[] = [];
      constructor(cb: ObserverCb) {
        capturedCallback = cb;
      }
      observe(el: Element) {
        observedElements.push(el);
      }
      unobserve() {
        /* noop */
      }
      disconnect() {
        disconnectSpy();
      }
      takeRecords() {
        return [] as IntersectionObserverEntry[];
      }
    }
    Object.defineProperty(globalThis, 'IntersectionObserver', {
      value: Spyable,
      writable: true,
      configurable: true,
    });
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  function renderWithRef() {
    return renderHook(() => {
      const ref = useRef<HTMLDivElement | null>(null);
      const visible = useInViewport(ref);
      // Attach a real element so the observer can observe it.
      if (!ref.current) {
        ref.current = document.createElement('div');
      }
      return { visible, ref };
    });
  }

  it('[P1] returns false initially', () => {
    const { result } = renderWithRef();
    expect(result.current.visible).toBe(false);
  });

  it('[P1] flips to true when the observer reports intersection', () => {
    const { result, rerender } = renderWithRef();
    expect(result.current.visible).toBe(false);

    act(() => {
      capturedCallback?.([{ isIntersecting: true } as IntersectionObserverEntry]);
    });

    rerender();
    expect(result.current.visible).toBe(true);
  });

  it('[P1] disconnects after first intersection when once=true (default)', () => {
    renderWithRef();
    act(() => {
      capturedCallback?.([{ isIntersecting: true } as IntersectionObserverEntry]);
    });
    expect(disconnectSpy).toHaveBeenCalled();
  });

  it('[P1] falls back to visible=true when IntersectionObserver is missing', () => {
    Object.defineProperty(globalThis, 'IntersectionObserver', {
      value: undefined,
      writable: true,
      configurable: true,
    });

    const { result } = renderHook(() => {
      const ref = useRef<HTMLDivElement | null>(null);
      if (!ref.current) ref.current = document.createElement('div');
      return useInViewport(ref);
    });

    expect(result.current).toBe(true);
  });
});
