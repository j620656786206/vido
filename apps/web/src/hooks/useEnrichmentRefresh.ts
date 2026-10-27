// Implements: <utility — no .pen counterpart>
/**
 * useEnrichmentRefresh (disc-2026-09-batch-reparse-never-runs,
 * disc-2026-09-batch-reparse-progress-in-dialog)
 *
 * A batch 重新解析 sets rows back to 整理中 and the backend then runs the
 * matching pass in the background, reporting on the `enrich_progress` (every
 * 5 rows) and `enrich_complete` SSE events. `watch()` opens one EventSource on
 * /events and resolves once it is connected; from then on the hook refetches
 * every library query when a pass completes and reports what the dialog
 * shows: queued (nothing heard yet) → running (progress) → done (the pass
 * result). A pass queued behind a running one reports `done` for the earlier
 * pass and then `running` again — the dialog simply follows. Lazy on purpose:
 * the page opens no SSE connection until a re-parse has actually been
 * requested — and it opens it BEFORE the POST, so a pass that ends at once is
 * still heard (disc-2026-10-reparse-dialog-misses-instant-pass).
 *
 * The stream is not trusted to arrive (disc-2026-10-reparse-dialog-stuck-over-tunnel):
 * through the Cloudflare tunnel it never opened, and while it hung the page's
 * other responses — the re-parse POST included — did not come back either, so
 * the dialog sat on 處理中 0 / 1 after a 2-second pass. A stream that has not
 * opened by ENRICHMENT_CONNECT_TIMEOUT_MS is closed, and `track()` — called once
 * the POST has queued rows — polls GET /scanner/enrich/status until the pass
 * ends, whichever of the two reports first.
 *
 * Rule 23: every number and title here is the server's; no client clock.
 */
import { useCallback, useEffect, useRef, useState } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { libraryKeys } from './useLibrary';
import { scannerService, type EnrichStatus } from '../services/scannerService';
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

// How long a re-parse waits for the progress stream before POSTing anyway: a
// broken stream must never stop the re-parse itself (same rule as
// SCAN_TRACKING_CONNECT_TIMEOUT_MS).
export const ENRICHMENT_CONNECT_TIMEOUT_MS = 3000;

// How often track() asks whether the pass is still running.
export const ENRICHMENT_POLL_INTERVAL_MS = 2000;

export interface EnrichmentWatch {
  /** What the dialog shows; null while nothing is being watched. */
  matching: EnrichmentMatching | null;
  /**
   * Open the stream (again from 已排入) and resolve once it is CONNECTED, or
   * after ENRICHMENT_CONNECT_TIMEOUT_MS. Await it BEFORE the re-parse POST:
   * the server registers the stream before it answers, so every event after
   * that reaches the dialog — even from a pass that ends in milliseconds
   * (disc-2026-10-reparse-dialog-misses-instant-pass). A stream that has not
   * opened by then is closed: a hung one held up the page's other responses.
   * Never rejects.
   */
  watch: () => Promise<void>;
  /**
   * The re-parse POST queued rows: poll the pass status until it ends, so the
   * dialog finishes even when the stream never delivers.
   */
  track: () => void;
  /** Close the stream and clear the dialog state (nothing was queued). */
  stop: () => void;
}

export function useEnrichmentRefresh(): EnrichmentWatch {
  const queryClient = useQueryClient();
  const [matching, setMatching] = useState<EnrichmentMatching | null>(null);
  const esRef = useRef<EventSource | null>(null);
  const pollRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  // The dialog has reached 完成 for this watch — later reports are stale.
  const doneRef = useRef(false);

  const stopPolling = useCallback(() => {
    if (pollRef.current) clearTimeout(pollRef.current);
    pollRef.current = null;
  }, []);

  const stop = useCallback(() => {
    stopPolling();
    esRef.current?.close();
    esRef.current = null;
    setMatching(null);
  }, [stopPolling]);

  const watch = useCallback((): Promise<void> => {
    stopPolling();
    doneRef.current = false;
    esRef.current?.close();
    esRef.current = null;
    if (typeof EventSource === 'undefined') {
      setMatching(null);
      return Promise.resolve();
    }
    setMatching(QUEUED);

    const es = new EventSource(scannerService.getSSEUrl());
    esRef.current = es;
    es.addEventListener('enrich_progress', (e: MessageEvent) => {
      if (doneRef.current) return;
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
    });
    es.addEventListener('enrich_complete', (e: MessageEvent) => {
      stopPolling();
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
    });

    return new Promise<void>((resolve) => {
      let opened = false;
      const giveUp = () => {
        // Never opened: close it rather than leave it hanging (see top).
        es.close();
        if (esRef.current === es) esRef.current = null;
        resolve();
      };
      const timer = setTimeout(giveUp, ENRICHMENT_CONNECT_TIMEOUT_MS);
      es.onopen = () => {
        opened = true;
        clearTimeout(timer);
        resolve();
      };
      // A known failure must not hold the dialog for the full timeout. After
      // opening, an error is a drop the browser reconnects on its own.
      es.onerror = () => {
        if (opened) return;
        clearTimeout(timer);
        giveUp();
      };
    });
  }, [queryClient, stopPolling]);

  const track = useCallback(() => {
    stopPolling();
    const poll = async () => {
      pollRef.current = null;
      let status: EnrichStatus | null = null;
      try {
        status = await scannerService.getEnrichStatus();
      } catch {
        // Unreachable for a moment: ask again.
      }
      if (doneRef.current) return;
      if (status && !status.isActive) {
        doneRef.current = true;
        esRef.current?.close();
        esRef.current = null;
        queryClient.invalidateQueries({ queryKey: libraryKeys.all });
        const finished = status;
        setMatching((prev) =>
          // An empty follow-up pass reports zeros; keep what this pass showed.
          finished.total === 0 && prev && prev.total > 0
            ? { ...prev, phase: 'done', currentTitle: '' }
            : {
                phase: 'done',
                total: num(finished.total),
                processed: num(finished.total),
                succeeded: num(finished.succeeded),
                failed: num(finished.failed),
                skipped: num(finished.skipped),
                currentTitle: '',
              }
        );
        return;
      }
      if (status && status.total > 0) {
        const running = status;
        setMatching({
          phase: 'running',
          total: num(running.total),
          processed: num(running.processed),
          succeeded: num(running.succeeded),
          failed: num(running.failed),
          skipped: num(running.skipped),
          currentTitle: running.currentTitle ?? '',
        });
      }
      pollRef.current = setTimeout(poll, ENRICHMENT_POLL_INTERVAL_MS);
    };
    pollRef.current = setTimeout(poll, ENRICHMENT_POLL_INTERVAL_MS);
  }, [queryClient, stopPolling]);

  useEffect(
    () => () => {
      stopPolling();
      esRef.current?.close();
    },
    [stopPolling]
  );

  return { matching, watch, track, stop };
}
