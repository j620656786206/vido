import { describe, it, expect, vi, afterEach } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import { useIsPhone, PHONE_MQ } from './useIsPhone';

const originalMatchMedia = window.matchMedia;

/** A controllable MediaQueryList: flip `matches`, then fire the captured listener. */
function stubMatchMedia(initial: boolean) {
  const state = { matches: initial };
  const listeners = new Set<() => void>();
  const add = vi.fn((_: string, cb: () => void) => listeners.add(cb));
  const remove = vi.fn((_: string, cb: () => void) => listeners.delete(cb));
  const queries: string[] = [];
  window.matchMedia = vi.fn((query: string) => {
    queries.push(query);
    return {
      get matches() {
        return state.matches;
      },
      media: query,
      addEventListener: add,
      removeEventListener: remove,
    };
  }) as unknown as typeof window.matchMedia;
  return {
    queries,
    add,
    remove,
    set(next: boolean) {
      state.matches = next;
      listeners.forEach((cb) => cb());
    },
  };
}

describe('useIsPhone', () => {
  afterEach(() => {
    window.matchMedia = originalMatchMedia;
  });

  it("asks Tailwind's own `sm` query directly (rem, not px) — never a negated min-width one", () => {
    // test-setup.ts stubs matchMedia to `matches:false` for EVERY query; a
    // negated `(min-width: 640px)` would put every jsdom spec on the phone path.
    const mm = stubMatchMedia(false);
    renderHook(() => useIsPhone());
    expect(PHONE_MQ).toBe('(width < 40rem)');
    expect(mm.queries.length).toBeGreaterThan(0);
    expect(new Set(mm.queries)).toEqual(new Set([PHONE_MQ]));
  });

  it('matches:true → true', () => {
    stubMatchMedia(true);
    const { result } = renderHook(() => useIsPhone());
    expect(result.current).toBe(true);
  });

  it('matches:false → false', () => {
    stubMatchMedia(false);
    const { result } = renderHook(() => useIsPhone());
    expect(result.current).toBe(false);
  });

  it('the global test-setup stub reads as "not a phone"', () => {
    const { result } = renderHook(() => useIsPhone());
    expect(result.current).toBe(false);
  });

  it('no matchMedia at all → false', () => {
    // @ts-expect-error — simulating a non-browser environment
    window.matchMedia = undefined;
    const { result } = renderHook(() => useIsPhone());
    expect(result.current).toBe(false);
  });

  it('a change event updates the value', () => {
    const mm = stubMatchMedia(false);
    const { result } = renderHook(() => useIsPhone());
    expect(result.current).toBe(false);
    act(() => mm.set(true));
    expect(result.current).toBe(true);
    act(() => mm.set(false));
    expect(result.current).toBe(false);
  });

  it('unmount removes the same listener it added', () => {
    const mm = stubMatchMedia(false);
    const { unmount } = renderHook(() => useIsPhone());
    expect(mm.add).toHaveBeenCalledWith('change', expect.any(Function));
    const cb = mm.add.mock.calls[0][1];
    unmount();
    expect(mm.remove).toHaveBeenCalledWith('change', cb);
  });
});
