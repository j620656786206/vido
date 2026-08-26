/**
 * Theme preference — 夜行 (dark) / 日巡 (light), persisted to localStorage so the
 * choice survives navigations and reloads. Follows `useDownloadsView.ts`, the
 * house pattern for a per-user display preference: module-level key, a
 * try/catch'd reader used as a SYNCHRONOUS useState initialiser (no effect, so
 * no flash), a useCallback setter that writes through inside try/catch.
 *
 * Three things about this file are load-bearing and easy to "tidy" into bugs:
 *
 * 1. **Dark writes NO attribute.** `:root` in styles.css IS the dark theme, so
 *    light stamps `data-theme="light"` and dark REMOVES the attribute. Writing
 *    `data-theme="dark"` would create a second source of truth for the default
 *    that could silently disagree with `:root`.
 *
 * 2. **`prefers-color-scheme` is consulted only when the user has made no
 *    choice** (⚖️ Alexyu 2026-08-26). A stored preference always wins — the OS
 *    is the default, not an override — and the listener stops mattering the
 *    moment the user picks a side.
 *
 * 3. **Every `matchMedia` touch is guarded.** jsdom does not implement it; the
 *    repo's own precedent for that is `DownloadsBrowseV2.tsx:59-64`. A stub now
 *    exists in test-setup.ts, but the guards stay — a hook that crashes the
 *    whole suite when a stub is removed is a trap for the next person.
 *
 * The pre-paint stamp lives in `apps/web/index.html`; this hook is what keeps
 * React's view of the theme and the DOM attribute in sync afterwards.
 */
import { useCallback, useEffect, useState } from 'react';

export type Theme = 'dark' | 'light';

const STORAGE_KEY = 'vido:theme';
const LIGHT_QUERY = '(prefers-color-scheme: light)';

/** The stored choice, or null when the user has never picked one. */
function readStored(): Theme | null {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    return raw === 'light' || raw === 'dark' ? raw : null;
  } catch {
    return null;
  }
}

function prefersLight(): boolean {
  try {
    if (typeof window.matchMedia !== 'function') return false;
    return window.matchMedia(LIGHT_QUERY).matches;
  } catch {
    return false;
  }
}

/** Stored choice first, OS preference second, dark last. */
export function resolveTheme(): Theme {
  return readStored() ?? (prefersLight() ? 'light' : 'dark');
}

export function applyTheme(theme: Theme): void {
  const root = document.documentElement;
  if (theme === 'light') {
    root.setAttribute('data-theme', 'light');
  } else {
    root.removeAttribute('data-theme');
  }
}

export function useTheme(): [Theme, (next: Theme) => void] {
  const [theme, setThemeState] = useState<Theme>(resolveTheme);

  const setTheme = useCallback((next: Theme) => {
    setThemeState(next);
    applyTheme(next);
    try {
      localStorage.setItem(STORAGE_KEY, next);
    } catch {
      /* ignore — a broken localStorage just means the choice isn't persisted */
    }
  }, []);

  // Track the OS while — and only while — the user has expressed no choice.
  // Someone who flips their Mac to dark at sunset should see Vido follow,
  // unless they have already told Vido otherwise.
  useEffect(() => {
    if (readStored() !== null) return;
    if (typeof window.matchMedia !== 'function') return;
    let mq: MediaQueryList;
    try {
      mq = window.matchMedia(LIGHT_QUERY);
    } catch {
      return;
    }
    const onChange = (e: MediaQueryListEvent) => {
      if (readStored() !== null) return;
      const next: Theme = e.matches ? 'light' : 'dark';
      setThemeState(next);
      applyTheme(next);
    };
    mq.addEventListener('change', onChange);
    return () => mq.removeEventListener('change', onChange);
  }, []);

  // The pre-paint script in index.html and this hook must not drift: it reads
  // the same key and the same media query, but it runs before React exists.
  useEffect(() => {
    applyTheme(theme);
  }, [theme]);

  return [theme, setTheme];
}
