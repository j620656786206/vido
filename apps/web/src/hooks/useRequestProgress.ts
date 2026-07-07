/**
 * Lazy SSE request-progress hook (Story 13-3b AC #1) — a line-for-line clone of
 * useDownloadProgress.ts. The house SSE convention is that each hook self-contains
 * its connect/reconnect anatomy (download_progress / scan_* / subtitle_* all do);
 * no shared abstraction is extracted until a third clone earns it (YAGNI).
 *
 * `confirmed against [@contract-v1]` (Story 13-3a AC #4): the `request_progress`
 * event's `Data` is a bare, snake_case snapshot of ALL non-terminal-stale requests
 * — every active row PLUS rows that transitioned THIS tick — each being the 13-1a
 * request resource + an ephemeral `progress` (0–1 float, present only while
 * downloading). This hook reconciles that snapshot into the useRequestedMedia
 * cache; see applyRequestSnapshot for the merge rules and their [@contract-v1]
 * consumer divergence from the downloads template.
 *
 * §8 lazy-SSE: NO connect on mount — DiscoverBrowseV2 calls startTracking() only
 * while the 想要清單 view is active AND the page is visible, so eager connects
 * never break Playwright networkidle.
 */
import { useCallback, useEffect, useRef } from 'react';
import { useQueryClient, type QueryClient } from '@tanstack/react-query';
import { requestService, type MediaRequest, type RequestStatus } from '../services/requestService';
import { requestKeys } from './useRequestedMedia';
import { snakeToCamel } from '../utils/caseTransform';

const SSE_RECONNECT_MS = 10000;

/** A live 想要清單 row carries an ephemeral, SSE-only `progress` (0–1). */
export type LiveRequest = MediaRequest & { progress?: number };

const TERMINAL_STATUSES: readonly RequestStatus[] = ['completed', 'failed'];

/**
 * Merge a full request snapshot into every cached 想要清單 list (Story 13-3b AC #2).
 *
 * [@contract-v1-consumer] divergence from applyDownloadSnapshot, recorded here: the
 * requests cache is a BARE array (requestService.listRequests — no pagination
 * envelope, unlike the template's PaginatedDownloads), and the wire snapshot is a
 * FULL snapshot of live + just-transitioned rows. Merge rules:
 *   - snapshot row present in cache      → REPLACE by id (progress + status ride in);
 *   - snapshot row absent from cache     → APPEND (created/activated in another tab);
 *   - cached row absent from snapshot    →
 *       · terminal (completed/failed)    → KEEP  (history the snapshot no longer carries);
 *       · active  (pending/searching/…)  → DROP.
 *
 * DROP-absent-active diverges from AC #2's original "keep all absent" wording,
 * adjusted per the Dev Notes STALE-MARK (13-7a hard-DELETE cancel). Because the
 * snapshot carries EVERY active row every tick (13-3a AC #4), an absent-yet-active
 * cached row is genuinely gone — cancelled in another tab (13-7a `DELETE
 * /requests/{id}`) or transitioned in a frame this client missed — so keeping it
 * would strand a phantom pending/downloading row. Never `invalidateQueries` on an
 * SSE frame — `setQueryData` only (no refetch storm; downloads convention).
 *
 * Accepted limitation (does NOT auto-recover): if this client MISSES a row's
 * terminal-transition frame (e.g. a completion lands during the 10s reconnect gap),
 * that row is dropped and NOT restored by a background refetch — each frame's
 * `setQueryData` keeps the query fresh (staleTime 30s, no `refetchInterval`), so
 * `refetchOnWindowFocus` never fires while frames stream; it reappears (as
 * `completed`) only on a manual reload or an explicit list invalidation. Chosen
 * over KEEP because the completed media is already in the library (low harm) while
 * KEEP would leave a stuck-`downloading` row that never resolves, and because
 * 13-7a's hard-DELETE makes DROP the only correct rule for cross-tab cancels.
 */
export function applyRequestSnapshot(queryClient: QueryClient, snapshot: LiveRequest[]): void {
  const byId = new Map(snapshot.map((r) => [r.id, r]));
  const queries = queryClient.getQueryCache().findAll({ queryKey: [...requestKeys.all, 'list'] });

  for (const q of queries) {
    const old = q.state.data as LiveRequest[] | undefined;
    if (!old) continue;

    const seen = new Set<string>();
    const merged: LiveRequest[] = [];
    for (const row of old) {
      const snap = byId.get(row.id);
      if (snap) {
        merged.push(snap); // fresh row replaces by id — progress rides into rendering
        seen.add(row.id);
      } else if (TERMINAL_STATUSES.includes(row.status)) {
        merged.push(row); // terminal history the snapshot no longer carries — keep
      }
      // else: absent + active → drop (phantom-row hazard — STALE-MARK / 13-7a)
    }
    // Rows created/activated in another tab the cache has not seen yet. Iterate
    // byId.values() (not the raw `snapshot`) so a duplicated id on the wire is
    // appended at most once — the append path has no `seen.add`, so raw `snapshot`
    // could double a brand-new id into two rows with the same React key.
    for (const snap of byId.values()) {
      if (!seen.has(snap.id)) merged.push(snap);
    }

    queryClient.setQueryData<LiveRequest[]>(q.queryKey, merged);
  }
}

export function useRequestProgress() {
  const queryClient = useQueryClient();
  const esRef = useRef<EventSource | null>(null);
  const reconnectRef = useRef<ReturnType<typeof setTimeout> | undefined>(undefined);
  const mountedRef = useRef(true);
  // Holds the latest connect() so the reconnect timer can call it without a
  // self-reference (which the linter flags as use-before-declare); assigned just
  // after connect is defined.
  const connectRef = useRef<() => void>(() => {});

  const connect = useCallback(() => {
    if (esRef.current) esRef.current.close();
    const es = new EventSource(requestService.getSSEUrl());
    esRef.current = es;

    es.addEventListener('request_progress', (e: MessageEvent) => {
      if (!mountedRef.current) return;
      try {
        const event = JSON.parse(e.data);
        // The SSE wire wraps the payload as the whole Event {id,type,data}; the snapshot is event.data.
        const snapshot = snakeToCamel<LiveRequest[]>(event.data ?? event);
        if (Array.isArray(snapshot)) applyRequestSnapshot(queryClient, snapshot);
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

  // Keep the reconnect timer pointed at the latest connect (connect is stable —
  // queryClient is stable — so this runs once). Assigning in an effect avoids
  // mutating a ref during render.
  useEffect(() => {
    connectRef.current = connect;
  }, [connect]);

  useEffect(() => {
    mountedRef.current = true;
    // NO connect on mount (§8) — consumers call startTracking() when the view is active.
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
