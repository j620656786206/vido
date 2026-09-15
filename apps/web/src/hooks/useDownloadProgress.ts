/**
 * Lazy SSE download-progress hook (ux3-4-3b AC4) — mirrors useScanProgress.ts.
 *
 * `confirmed against [@contract-v1]` (ux3-4-2b): the `download_progress` event's `Data` is a bare,
 * snake_case `qbittorrent.Torrent[]` snapshot (NO `import_status`, NOT the paginated envelope, the FULL
 * unpaginated list). This hook reconciles those three deltas into the `useDownloads` cache:
 *   1. NO import_status → MERGE (never replace): keep each cached item's `importStatus` (dl-import-1).
 *   2. bare array → mapped into each cached page's `.items`.
 *   3. full list → refresh live fields (progress/speed/eta/status) of items present by hash, and DROP
 *      items no longer in the snapshot (removed torrents). New / re-sorted torrents are reconciled by
 *      useDownloadActions' invalidate + refetchOnWindowFocus — the snapshot is all/added_on/desc and
 *      can't be re-sliced into arbitrary filtered/sorted pages without re-implementing the backend.
 *
 * §8 lazy-SSE: NO connect on mount — the consumer calls startTracking() when the page is active
 * (DownloadsBrowseV2 gates it on page visibility), so eager connects never break Playwright networkidle.
 */
import { useCallback, useEffect, useRef } from 'react';
import { useQueryClient, type QueryClient } from '@tanstack/react-query';
import {
  downloadService,
  type Download,
  type PaginatedDownloads,
} from '../services/downloadService';
import { downloadKeys } from './useDownloads';
import { snakeToCamel } from '../utils/caseTransform';

const SSE_RECONNECT_MS = 10000;
// How often an open page re-reads the list while an import status may be moving (the server refreshes
// its Sonarr/Radarr snapshot about once a minute, so faster would only re-read the same answer).
const IMPORT_STATUS_REFRESH_MS = 60000;

/**
 * Merge a full snapshot into every cached downloads list page (see the three-delta contract above).
 *
 * Returns true when an item's import status may have moved on and only a list re-read can tell
 * (dl-import-2): a torrent that just finished, or a finished one still awaiting import or scan.
 * A torrent that is not fully downloaded (again — e.g. a recheck) drops its import status: this
 * download has not been imported.
 */
export function applyDownloadSnapshot(queryClient: QueryClient, snapshot: Download[]): boolean {
  const byHash = new Map(snapshot.map((d) => [d.hash, d]));
  let importStatusStale = false;
  const queries = queryClient.getQueryCache().findAll({ queryKey: [...downloadKeys.all, 'list'] });

  for (const q of queries) {
    const old = q.state.data as PaginatedDownloads | undefined;
    if (!old) continue;

    const items = old.items
      .filter((it) => byHash.has(it.hash)) // drop removed torrents
      .map((it) => {
        const fresh = byHash.get(it.hash) as Download; // fresh live fields
        const finished = fresh.progress >= 1;
        const state = it.importStatus?.state;
        if (
          finished &&
          (it.progress < 1 || state === 'awaiting_import' || state === 'awaiting_scan')
        ) {
          importStatusStale = true;
        }
        return { ...fresh, importStatus: finished ? it.importStatus : undefined }; // keep import_status
      });

    const removed = old.items.length - items.length;
    queryClient.setQueryData<PaginatedDownloads>(
      q.queryKey,
      removed
        ? { ...old, items, totalItems: Math.max(0, old.totalItems - removed) }
        : { ...old, items }
    );
  }
  return importStatusStale;
}

export function useDownloadProgress() {
  const queryClient = useQueryClient();
  const esRef = useRef<EventSource | null>(null);
  const reconnectRef = useRef<ReturnType<typeof setTimeout> | undefined>(undefined);
  const lastImportRefreshRef = useRef(0);
  const mountedRef = useRef(true);
  // Holds the latest connect() so the reconnect timer can call it without a self-reference (which the
  // linter flags as use-before-declare); assigned just after connect is defined.
  const connectRef = useRef<() => void>(() => {});

  const connect = useCallback(() => {
    if (esRef.current) esRef.current.close();
    const es = new EventSource(downloadService.getSSEUrl());
    esRef.current = es;

    es.addEventListener('download_progress', (e: MessageEvent) => {
      if (!mountedRef.current) return;
      try {
        const event = JSON.parse(e.data);
        // The SSE wire wraps the payload as the whole Event {id,type,data}; the snapshot is event.data.
        const snapshot = snakeToCamel<Download[]>(event.data ?? event);
        if (Array.isArray(snapshot) && applyDownloadSnapshot(queryClient, snapshot)) {
          const now = Date.now();
          if (now - lastImportRefreshRef.current >= IMPORT_STATUS_REFRESH_MS) {
            lastImportRefreshRef.current = now;
            void queryClient.invalidateQueries({ queryKey: [...downloadKeys.all, 'list'] });
          }
        }
      } catch {
        // ignore malformed frames
      }
    });

    es.onerror = () => {
      if (!mountedRef.current) return;
      es.close();
      if (reconnectRef.current) clearTimeout(reconnectRef.current);
      reconnectRef.current = setTimeout(() => {
        if (mountedRef.current) connectRef.current();
      }, SSE_RECONNECT_MS);
    };
  }, [queryClient]);

  // Keep the reconnect timer pointed at the latest connect (connect is stable — queryClient is stable —
  // so this runs once). Assigning in an effect avoids mutating a ref during render.
  useEffect(() => {
    connectRef.current = connect;
  }, [connect]);

  useEffect(() => {
    mountedRef.current = true;
    // NO connect on mount (§8) — consumers call startTracking() when the page is active.
    return () => {
      mountedRef.current = false;
      if (esRef.current) {
        esRef.current.close();
        esRef.current = null;
      }
      if (reconnectRef.current) clearTimeout(reconnectRef.current);
    };
  }, []);

  const startTracking = useCallback(() => {
    // Connect only if not already open (readyState 2 === CLOSED). Idempotent — safe to call repeatedly.
    if (!esRef.current || esRef.current.readyState === 2) connect();
  }, [connect]);

  const stopTracking = useCallback(() => {
    if (esRef.current) {
      esRef.current.close();
      esRef.current = null;
    }
    if (reconnectRef.current) clearTimeout(reconnectRef.current);
  }, []);

  return { startTracking, stopTracking };
}
