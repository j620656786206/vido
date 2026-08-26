import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import { useTheme, resolveTheme, applyTheme } from './useTheme';

/**
 * The theme resolution rules are product decisions (⚖️ Alexyu 2026-08-26), not
 * implementation detail, so they get guards:
 *   stored choice > OS preference > dark.
 * Plus the one that is easy to "tidy" into a bug: dark writes NO attribute,
 * because :root in styles.css IS the dark theme and a second source of truth
 * for the default could silently disagree with it.
 */
type Listener = (e: MediaQueryListEvent) => void;

function mockMatchMedia(prefersLight: boolean) {
  const listeners: Listener[] = [];
  const mql = {
    matches: prefersLight,
    media: '(prefers-color-scheme: light)',
    onchange: null,
    addEventListener: (_: string, fn: Listener) => listeners.push(fn),
    removeEventListener: (_: string, fn: Listener) => {
      const i = listeners.indexOf(fn);
      if (i >= 0) listeners.splice(i, 1);
    },
    addListener: () => {},
    removeListener: () => {},
    dispatchEvent: () => false,
  };
  vi.stubGlobal('matchMedia', vi.fn(() => mql) as unknown as typeof window.matchMedia);
  return {
    fire: (matches: boolean) => listeners.forEach((fn) => fn({ matches } as MediaQueryListEvent)),
  };
}

describe('useTheme', () => {
  beforeEach(() => {
    localStorage.clear();
    document.documentElement.removeAttribute('data-theme');
  });
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  describe('resolution order: stored > OS > dark', () => {
    it('with no stored choice and no OS preference, resolves dark', () => {
      mockMatchMedia(false);
      expect(resolveTheme()).toBe('dark');
    });

    it('with no stored choice and a light-preferring OS, resolves light', () => {
      mockMatchMedia(true);
      expect(resolveTheme()).toBe('light');
    });

    it('a stored choice BEATS the OS — the OS is the default, not an override', () => {
      mockMatchMedia(true);
      localStorage.setItem('vido:theme', 'dark');
      expect(resolveTheme()).toBe('dark');
    });

    it('garbage in storage falls back to the OS rather than throwing', () => {
      mockMatchMedia(true);
      localStorage.setItem('vido:theme', 'chartreuse');
      expect(resolveTheme()).toBe('light');
    });

    it('survives a browser with no matchMedia at all', () => {
      vi.stubGlobal('matchMedia', undefined);
      expect(resolveTheme()).toBe('dark');
    });
  });

  describe('the DOM attribute', () => {
    it('light STAMPS data-theme; dark REMOVES it so :root stays the only default', () => {
      applyTheme('light');
      expect(document.documentElement.getAttribute('data-theme')).toBe('light');
      applyTheme('dark');
      expect(document.documentElement.hasAttribute('data-theme')).toBe(false);
    });
  });

  describe('setting a theme', () => {
    it('persists the choice and stamps the DOM', () => {
      mockMatchMedia(false);
      const { result } = renderHook(() => useTheme());
      act(() => result.current[1]('light'));
      expect(result.current[0]).toBe('light');
      expect(localStorage.getItem('vido:theme')).toBe('light');
      expect(document.documentElement.getAttribute('data-theme')).toBe('light');
    });
  });

  describe('following the OS', () => {
    it('follows an OS change while the user has expressed no choice', () => {
      const mm = mockMatchMedia(false);
      const { result } = renderHook(() => useTheme());
      expect(result.current[0]).toBe('dark');

      act(() => mm.fire(true));
      expect(result.current[0]).toBe('light');
      expect(document.documentElement.getAttribute('data-theme')).toBe('light');
    });

    it('STOPS following once the user picks a side', () => {
      const mm = mockMatchMedia(false);
      const { result } = renderHook(() => useTheme());
      act(() => result.current[1]('dark'));

      // The OS goes light at sunset; the user already said dark.
      act(() => mm.fire(true));
      expect(result.current[0]).toBe('dark');
      expect(document.documentElement.hasAttribute('data-theme')).toBe(false);
    });
  });
});
