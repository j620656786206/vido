/**
 * Downloads List|Table view preference (ux3-4-4 AC1), persisted to localStorage so the choice
 * survives navigations/reloads. Defaults to 'list'; an unreadable/invalid value falls back to 'list'.
 */
import { useCallback, useState } from 'react';

export type DownloadsView = 'list' | 'table';

const STORAGE_KEY = 'vido:downloads:view';

function readStored(): DownloadsView {
  try {
    return localStorage.getItem(STORAGE_KEY) === 'table' ? 'table' : 'list';
  } catch {
    return 'list';
  }
}

export function useDownloadsView(): [DownloadsView, (view: DownloadsView) => void] {
  const [view, setViewState] = useState<DownloadsView>(readStored);

  const setView = useCallback((next: DownloadsView) => {
    setViewState(next);
    try {
      localStorage.setItem(STORAGE_KEY, next);
    } catch {
      /* ignore — a broken localStorage just means the choice isn't persisted */
    }
  }, []);

  return [view, setView];
}
