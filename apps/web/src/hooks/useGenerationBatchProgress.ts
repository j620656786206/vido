/**
 * Lazy SSE generation-BATCH progress hook (ux3-subtitle-v2-batch AC 3) —
 * sibling of useSubtitleBatchProgress (reducer/terminal-close shape) listening
 * for the Story 9R-16 `generation_batch_progress` events on the shared
 * GET /api/v1/events stream.
 *
 * ⚠️ Double-nested envelope: the SSE `data:` line carries the FULL `Event`
 * struct `{"id","type","data":{…payload…}}` — unwrap via `parsed.data`
 * (slice-1 `useGenerationProgress` form; the fetch hook's `event.data || event`
 * is an EQUIVALENT unwrap — do not "fix" it), then snakeToCamel (Rule 18).
 *
 * §8 lazy-SSE rules: NO connect on mount — `startTracking()` opens the stream
 * (also the 409/recover-attach path). Terminal statuses
 * `complete | cancelled | error | budget_ceiling` close the stream (no polling
 * fallback); 10s reconnect backoff; `mountedRef` guard; cleanup on unmount.
 *
 * Rule 23: zero wall-clock reads — progress/cost are all SSE-supplied.
 */
import { useCallback, useEffect, useReducer, useRef, useState } from 'react';
import { snakeToCamel } from '../utils/caseTransform';
import type {
  GenerationBatchItemState,
  GenerationBatchProgress,
  GenerationBatchStatus,
} from '../services/subtitleService';

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || '/api/v1';
const SSE_RECONNECT_MS = 10000;

/** Hook-side state machine: the wire statuses + the idle resting state. */
export type GenerationBatchHookStatus = 'idle' | GenerationBatchStatus;

/**
 * Running totals of this batch's `subtitle_run_receipt` events (sub-7-6a
 * [@contract-v1]; sub-7-6c AC #2). COMPLETED runs only — a failed run has no
 * cues to count — and `cacheMeasuredRuns` tells how many of them reported a
 * cache split at all, so an unmeasured cache is never read as 0 hits.
 */
export interface GenerationBatchRunTotals {
  completedRuns: number;
  cueCount: number;
  cacheHitCues: number;
  cacheMeasuredRuns: number;
  /** In first-seen order; more than one means the batch changed model midway. */
  modelIds: string[];
}

export const EMPTY_RUN_TOTALS: GenerationBatchRunTotals = {
  completedRuns: 0,
  cueCount: 0,
  cacheHitCues: 0,
  cacheMeasuredRuns: 0,
  modelIds: [],
};

export interface GenerationBatchProgressState {
  batchId: string;
  totalItems: number;
  currentIndex: number;
  /** UUID string movie id of the in-flight item ([@contract-v2]); null before the first event. */
  currentMediaId: string | null;
  currentItem: string;
  successCount: number;
  failCount: number;
  pausedCount: number;
  status: GenerationBatchHookStatus;
  spentUsd: number;
  budgetUsd: number;
  /**
   * The queue with each entry's state (dsr-6d-a AC #1). The SSE event sends it
   * only on the terminal broadcast — while running it carries a single
   * changed_item instead — so this stays null until a consumer seeds it from
   * the 202 / status probe (dsr-6d-b / dsr-6d-c).
   */
  items: GenerationBatchItemState[] | null;
  /** The model the batch was priced and run with (sub-7-6a `model_id`); '' on older servers. */
  modelId?: string;
  /** Live receipt sum for the F8c completion line; reset with every batch. */
  runTotals?: GenerationBatchRunTotals;
}

const initialState: GenerationBatchProgressState = {
  batchId: '',
  totalItems: 0,
  currentIndex: 0,
  currentMediaId: null,
  currentItem: '',
  successCount: 0,
  failCount: 0,
  pausedCount: 0,
  status: 'idle',
  spentUsd: 0,
  budgetUsd: 0,
  items: null,
  modelId: '',
  runTotals: EMPTY_RUN_TOTALS,
};

/**
 * The SSE payload = the snapshot shape PLUS the running-only `changed_item`
 * (dsr-6d-a AC #2): while a batch runs the event sends `items: null` and the
 * ONE entry whose state just changed, so a 2,400-item select-all does not ship
 * the whole queue on every tick.
 */
interface GenerationBatchSsePayload extends GenerationBatchProgress {
  changedItem?: GenerationBatchItemState | null;
}

/** The `subtitle_run_receipt` payload keys this hook reads (camelCased). */
interface SubtitleRunReceiptPayload {
  batchId?: string;
  status?: string;
  modelId?: string;
  cueCount?: number;
  cacheHitCues?: number | null;
}

type Action =
  | { type: 'START'; payload: Partial<GenerationBatchProgressState> }
  | { type: 'ATTACH'; payload: GenerationBatchProgress }
  | { type: 'SSE_UPDATE'; payload: GenerationBatchSsePayload }
  | { type: 'RUN_RECEIPT'; payload: SubtitleRunReceiptPayload }
  | { type: 'RESET' };

/**
 * Queue merge rules (dsr-6d-b AC #5), in priority order:
 *  1. a whole `items` array (the terminal broadcast) replaces what we had;
 *  2. otherwise a `changed_item` REPLACES the matching entry in place — an id
 *     we do not know is dropped, never appended (the queue's membership comes
 *     from the 202/status snapshot, not from the stream);
 *  3. with no queue seeded yet (`null`) a changed_item is ignored entirely —
 *     a one-row queue invented from a single event would be a lie about scope.
 */
function mergeItems(
  current: GenerationBatchItemState[] | null,
  p: GenerationBatchSsePayload
): GenerationBatchItemState[] | null {
  if (p.items) return p.items;
  const changed = p.changedItem;
  if (!changed || current === null) return current;
  let found = false;
  const next = current.map((it) => {
    if (it.mediaId !== changed.mediaId) return it;
    found = true;
    return changed;
  });
  return found ? next : current;
}

function reducer(
  state: GenerationBatchProgressState,
  action: Action
): GenerationBatchProgressState {
  switch (action.type) {
    case 'START':
      return { ...initialState, ...action.payload, status: 'running' };
    case 'ATTACH': {
      // Seed a snapshot we are NOT streaming (the `last` terminal result, or a
      // 409 body): the snapshot's own status stands — unlike START, which
      // forces `running` because it is opening a stream.
      const p = action.payload;
      return {
        ...initialState,
        batchId: p.batchId ?? '',
        totalItems: p.totalItems ?? 0,
        currentIndex: p.currentIndex ?? 0,
        currentMediaId: p.currentMediaId || null,
        currentItem: p.currentItem ?? '',
        successCount: p.successCount ?? 0,
        failCount: p.failCount ?? 0,
        pausedCount: p.pausedCount ?? 0,
        status: p.status ?? 'idle',
        spentUsd: p.spentUsd ?? 0,
        budgetUsd: p.budgetUsd ?? 0,
        items: p.items ?? null,
        modelId: p.modelId ?? '',
        // A snapshot carries no receipts; the ledger (`by_batch`) fills this
        // line in for an attached batch.
        runTotals: EMPTY_RUN_TOTALS,
      };
    }
    case 'SSE_UPDATE': {
      const p = action.payload;
      return {
        ...state,
        batchId: p.batchId || state.batchId,
        totalItems: p.totalItems ?? state.totalItems,
        currentIndex: p.currentIndex ?? state.currentIndex,
        currentMediaId: p.currentMediaId ?? state.currentMediaId,
        currentItem: p.currentItem ?? state.currentItem,
        successCount: p.successCount ?? state.successCount,
        failCount: p.failCount ?? state.failCount,
        pausedCount: p.pausedCount ?? state.pausedCount,
        status: p.status ?? 'running',
        spentUsd: p.spentUsd ?? state.spentUsd,
        budgetUsd: p.budgetUsd ?? state.budgetUsd,
        items: mergeItems(state.items, p),
        modelId: p.modelId ?? state.modelId,
      };
    }
    case 'RUN_RECEIPT': {
      const r = action.payload;
      // Only THIS batch's completed runs: a solo 生成字幕 click elsewhere emits
      // receipts too (sub-7-6a), with no batch_id or another one.
      if (!r.batchId || r.batchId !== state.batchId || r.status !== 'completed') return state;
      const t = state.runTotals ?? EMPTY_RUN_TOTALS;
      const measured = typeof r.cacheHitCues === 'number';
      return {
        ...state,
        runTotals: {
          completedRuns: t.completedRuns + 1,
          cueCount: t.cueCount + (r.cueCount ?? 0),
          cacheHitCues: t.cacheHitCues + (measured ? (r.cacheHitCues as number) : 0),
          cacheMeasuredRuns: t.cacheMeasuredRuns + (measured ? 1 : 0),
          modelIds:
            r.modelId && !t.modelIds.includes(r.modelId) ? [...t.modelIds, r.modelId] : t.modelIds,
        },
      };
    }
    case 'RESET':
      return initialState;
    default:
      return state;
  }
}

function isTerminal(status: GenerationBatchHookStatus): boolean {
  return (
    status === 'complete' ||
    status === 'cancelled' ||
    status === 'error' ||
    status === 'budget_ceiling'
  );
}

export function useGenerationBatchProgress() {
  const [progress, dispatch] = useReducer(reducer, initialState);
  /**
   * Bumped once per BACKOFF reconnect (never for the first connect). A dropped
   * terminal event would otherwise leave the UI stuck on 進行中 forever — the
   * consumer watches this and re-reads GET …/status, which answers with
   * `{running:false, last}` (9R-16 CR L1).
   */
  const [connectionEpoch, setConnectionEpoch] = useState(0);
  const esRef = useRef<EventSource | null>(null);
  const reconnectRef = useRef<ReturnType<typeof setTimeout> | undefined>(undefined);
  const mountedRef = useRef(true);
  const connectRef = useRef<() => void>(() => {});

  const closeSSE = useCallback(() => {
    if (esRef.current) {
      esRef.current.close();
      esRef.current = null;
    }
    if (reconnectRef.current) {
      clearTimeout(reconnectRef.current);
      reconnectRef.current = undefined;
    }
  }, []);

  const connect = useCallback(() => {
    // Cancel any pending backoff reconnect (slice-1 timer-leak lesson): a stale
    // timer surviving into a fresh connection would bounce the healthy stream.
    if (reconnectRef.current) {
      clearTimeout(reconnectRef.current);
      reconnectRef.current = undefined;
    }
    if (esRef.current) esRef.current.close();
    const es = new EventSource(`${API_BASE_URL}/events`);
    esRef.current = es;

    es.addEventListener('generation_batch_progress', (e: MessageEvent) => {
      if (!mountedRef.current) return;
      try {
        const parsed = JSON.parse(e.data);
        // data: line = full Event struct {id,type,data} → payload is parsed.data.
        const payload = snakeToCamel<GenerationBatchSsePayload>(parsed.data ?? parsed);
        dispatch({ type: 'SSE_UPDATE', payload });
        if (isTerminal(payload.status)) closeSSE();
      } catch {
        // Ignore malformed frames.
      }
    });

    // sub-7-6c AC #2: each finished run's receipt (cues, cache split, model)
    // rides its own event; summed here so the F8c line can draw the moment
    // the terminal event lands, before the ledger answers.
    es.addEventListener('subtitle_run_receipt', (e: MessageEvent) => {
      if (!mountedRef.current) return;
      try {
        const parsed = JSON.parse(e.data);
        const payload = snakeToCamel<SubtitleRunReceiptPayload>(parsed.data ?? parsed);
        dispatch({ type: 'RUN_RECEIPT', payload });
      } catch {
        // Ignore malformed frames.
      }
    });

    es.onerror = () => {
      if (!mountedRef.current) return;
      es.close();
      if (reconnectRef.current) clearTimeout(reconnectRef.current);
      reconnectRef.current = setTimeout(() => {
        if (!mountedRef.current) return;
        // A gap in the stream may have swallowed the terminal event — tell the
        // consumer to re-read status. Bumped HERE (the actual reconnect), not
        // in startTracking's first connect.
        setConnectionEpoch((n) => n + 1);
        connectRef.current();
      }, SSE_RECONNECT_MS);
    };
  }, [closeSSE]);

  useEffect(() => {
    connectRef.current = connect;
  }, [connect]);

  useEffect(() => {
    mountedRef.current = true;
    // NO connect on mount (§8) — consumers call startTracking() after a batch
    // starts (or when attaching to one already running server-side).
    return () => {
      mountedRef.current = false;
      closeSSE();
    };
  }, [closeSSE]);

  /**
   * Open the stream and enter the running state. Call AFTER a batch starts —
   * also the 409/status-probe attach path (seed with the recovered snapshot).
   */
  const startTracking = useCallback(
    (seed?: Partial<GenerationBatchProgressState>) => {
      dispatch({ type: 'START', payload: seed ?? {} });
      if (!esRef.current || esRef.current.readyState === 2) connect();
    },
    [connect]
  );

  /**
   * Seed a snapshot WITHOUT opening a stream — for the `last` terminal result
   * (GET …/status) and any other already-finished snapshot. `startTracking`
   * cannot do this job: it forces `status: 'running'` and opens an
   * EventSource, which would paint a finished batch as in-flight and leave a
   * connection nobody closes (dsr-6d-b AC #5).
   */
  const attachSnapshot = useCallback((snapshot: GenerationBatchProgress) => {
    dispatch({ type: 'ATTACH', payload: snapshot });
  }, []);

  /** Tear down the stream and return to idle (e.g. when the dialog closes). */
  const reset = useCallback(() => {
    closeSSE();
    dispatch({ type: 'RESET' });
  }, [closeSSE]);

  return {
    progress,
    status: progress.status,
    startTracking,
    attachSnapshot,
    reset,
    connectionEpoch,
  };
}
