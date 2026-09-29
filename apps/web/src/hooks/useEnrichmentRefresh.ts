// Implements: <utility — no .pen counterpart>
/**
 * useEnrichmentRefresh (disc-2026-09-batch-reparse-never-runs,
 * disc-2026-09-batch-reparse-progress-in-dialog)
 *
 * A batch 重新解析 sets rows back to 整理中 and the backend then runs the
 * matching pass in the background, reporting on the `enrich_progress` (every
 * 5 rows) and `enrich_complete` SSE events. While `enabled`, this hook holds
 * one EventSource on /events, refetches every library query when a pass
 * completes, and returns what the dialog shows: queued (nothing heard yet) →
 * running (progress) → done (the pass result). A pass queued behind a running
 * one reports `done` for the earlier pass and then `running` again — the
 * dialog simply follows. Lazy on purpose: the page opens no SSE connection
 * until a re-parse has actually been requested.
 *
 * Rule 23: every number and title here is the server's; no client clock.
 */
import { useEffect, useState } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { libraryKeys } from './useLibrary';
import { scannerService } from '../services/scannerService';
import { snakeToCamel } from '../utils/caseTransform';

export interface EnrichmentMatching {
  phase: 'queued' | 'running' | 'done';
  /** Rows in THIS pass — every pending row, not only the ones just re-parsed. */
  total: number;
  processed: number;
  succeeded: number;
  failed: number;
  skipped: number;
  /** Row being matched right now ('' once done). */
  currentTitle: string;
}

const QUEUED: EnrichmentMatching = {
  phase: 'queued',
  total: 0,
  processed: 0,
  succeeded: 0,
  failed: 0,
  skipped: 0,
  currentTitle: '',
};

function payloadOf(e: MessageEvent): Record<string, unknown> {
  const event = JSON.parse(e.data);
  return snakeToCamel<Record<string, unknown>>(event.data || event);
}

const num = (v: unknown) => (typeof v === 'number' && Number.isFinite(v) ? v : 0);

/**
 * `runId`: bump it on every new re-parse. Without it a second 重新解析 while the
 * watch is already on would keep showing the previous pass's `done` instead of
 * starting again from 已排入 (React batches `false`→`true` into no change).
 */
export function useEnrichmentRefresh(enabled: boolean, runId = 0): EnrichmentMatching | null {
  const queryClient = useQueryClient();
  const [matching, setMatching] = useState<EnrichmentMatching | null>(null);

  useEffect(() => {
    if (!enabled || typeof EventSource === 'undefined') {
      setMatching(null);
      return;
    }
    setMatching(QUEUED);

    const es = new EventSource(scannerService.getSSEUrl());
    const onProgress = (e: MessageEvent) => {
      try {
        const p = payloadOf(e);
        setMatching({
          phase: 'running',
          total: num(p.total),
          processed: num(p.processed),
          succeeded: num(p.succeeded),
          failed: num(p.failed),
          skipped: num(p.skipped),
          currentTitle: typeof p.currentTitle === 'string' ? p.currentTitle : '',
        });
      } catch {
        // Ignore parse errors
      }
    };
    const onComplete = (e: MessageEvent) => {
      queryClient.invalidateQueries({ queryKey: libraryKeys.all });
      try {
        const r = payloadOf(e);
        setMatching({
          phase: 'done',
          total: num(r.total),
          processed: num(r.total),
          succeeded: num(r.succeeded),
          failed: num(r.failed),
          skipped: num(r.skipped),
          currentTitle: '',
        });
      } catch {
        setMatching((prev) => (prev ? { ...prev, phase: 'done', currentTitle: '' } : prev));
      }
    };
    es.addEventListener('enrich_progress', onProgress);
    es.addEventListener('enrich_complete', onComplete);
    return () => {
      es.removeEventListener('enrich_progress', onProgress);
      es.removeEventListener('enrich_complete', onComplete);
      es.close();
    };
  }, [enabled, runId, queryClient]);

  return matching;
}
