// Implements: <utility — no .pen counterpart>
/**
 * useEnrichmentRefresh (disc-2026-09-batch-reparse-never-runs)
 *
 * A batch 重新解析 sets rows back to 整理中 and the backend then runs the
 * matching pass in the background, reporting on the `enrich_complete` SSE
 * event. Nothing on the library page listened for it, so the rows stayed
 * 整理中 on screen until the user reloaded. While `enabled`, this hook holds
 * one EventSource on /events and refetches every library query when a pass
 * completes. Lazy on purpose: the page opens no SSE connection until a
 * re-parse has actually been requested.
 */
import { useEffect } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { libraryKeys } from './useLibrary';
import { scannerService } from '../services/scannerService';

export function useEnrichmentRefresh(enabled: boolean): void {
  const queryClient = useQueryClient();

  useEffect(() => {
    if (!enabled || typeof EventSource === 'undefined') return;

    const es = new EventSource(scannerService.getSSEUrl());
    const refresh = () => {
      queryClient.invalidateQueries({ queryKey: libraryKeys.all });
    };
    es.addEventListener('enrich_complete', refresh);
    return () => {
      es.removeEventListener('enrich_complete', refresh);
      es.close();
    };
  }, [enabled, queryClient]);
}
