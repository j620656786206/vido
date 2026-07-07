/**
 * Lazy SSE generation-jobs feed hook (Story ux3-ai-2 AC 4/5) — the generation
 * WORKSPACE's live surface. Clones the `useGenerationProgress` SSE anatomy
 * (lazy §8, double-nested `parsed.data` unwrap, snakeToCamel, 10s reconnect,
 * `mountedRef`, latest-ref connect) but listens UNFILTERED — no `media_id`
 * filter — to surface two derived outputs from the one shared stream:
 *
 *   - `feed`: a session-scoped event log (AC 4). Every `transcription_*` event
 *     (per-item granularity) plus each `generation_batch_progress` status change
 *     (batch lifecycle) appends one row, in arrival order, capped to keep memory
 *     bounded on long batches. NO timestamps — SSE payloads carry none, so the
 *     log is order-only (Rule 23-clean: the consuming component reads no clock).
 *     Row keys use a monotonic `seq` (deterministic, not `Date.now()`).
 *   - `singleJobs`: a per-media map of in-flight transcription jobs (AC 5). A job
 *     appears on its NEXT event and is retired on its terminal (complete/failed).
 *     HONEST limitation: a job started elsewhere is invisible until its next SSE
 *     event — `active_jobs` has no transcription kind
 *     (disc-2026-07-transcription-active-jobs, backlog). The workspace renders
 *     these as queue rows only when NO batch is running (a running batch owns the
 *     queue; its items still flow into the feed).
 *
 * §8 lazy-SSE: NO connect on mount — the workspace calls `startTracking()` while
 * the view is active AND visible; `stop()` closes the stream on leave/hide.
 */
import { useCallback, useEffect, useReducer, useRef } from 'react';
import { snakeToCamel } from '../utils/caseTransform';
import type { GenerationPhase } from './useGenerationProgress';

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || '/api/v1';
const SSE_RECONNECT_MS = 10000;
/** Cap the feed so a long-running batch never grows an unbounded array. */
export const FEED_CAP = 200;

/** Visual tone for a feed row / single job — mapped to design tokens in the view. */
export type FeedTone = 'active' | 'done' | 'failed' | 'info';

export interface FeedRow {
  /** Monotonic React key (deterministic, wall-clock-free — Rule 23). */
  seq: number;
  tone: FeedTone;
  /** Stage/label, e.g. 提取音訊 / 轉錄中 / 完成 / 批次完成. */
  stage: string;
  /** UUID-string movie id of the item this row concerns, when known. */
  mediaId?: string;
  /** Server-supplied message (never a client clock read). */
  message?: string;
  /** Mono trailing value, e.g. `45%`. */
  trail?: string;
}

export interface SingleJobState {
  mediaId: string;
  phase: GenerationPhase;
  message: string;
  percentage: number | null;
}

export interface GenerationJobsFeedState {
  feed: FeedRow[];
  singleJobs: Record<string, SingleJobState>;
  seq: number;
  /** Last batch status seen — feed appends a batch row only on a status CHANGE (no per-tick spam). */
  lastBatchStatus: string;
}

const initialState: GenerationJobsFeedState = {
  feed: [],
  singleJobs: {},
  seq: 0,
  lastBatchStatus: '',
};

/** Camelized transcription_* payload (envelope-unwrapped). */
interface TranscriptionPayload {
  mediaId?: string;
  phase?: string;
  percentage?: number;
  message?: string;
  error?: string;
}
/** Camelized generation_batch_progress payload (envelope-unwrapped). */
interface BatchPayload {
  status?: string;
  currentMediaId?: string;
  currentItem?: string;
}

type ActivePhase = 'extracting' | 'transcribing' | 'translating';

const TRANSCRIPTION_EVENTS: ReadonlyArray<{ event: string; phase: ActivePhase }> = [
  { event: 'transcription_extracting', phase: 'extracting' },
  { event: 'transcription_progress', phase: 'transcribing' },
  { event: 'translation_progress', phase: 'translating' },
];

const PHASE_LABEL: Record<GenerationPhase, string> = {
  idle: '排入佇列',
  extracting: '提取音訊',
  transcribing: '轉錄中',
  translating: '翻譯中',
  complete: '完成',
  failed: '失敗',
};

const BATCH_STATUS_LABEL: Record<string, string> = {
  complete: '批次完成',
  cancelled: '批次已取消',
  error: '批次發生錯誤',
  budget_ceiling: '已達本次預算上限',
};

type Action =
  | { type: 'PHASE'; phase: ActivePhase; payload: TranscriptionPayload }
  | { type: 'COMPLETE'; payload: TranscriptionPayload }
  | { type: 'FAILED'; payload: TranscriptionPayload }
  | { type: 'BATCH'; payload: BatchPayload }
  | { type: 'RESET' };

/** Append a row, assigning the next seq and enforcing FEED_CAP (drop-oldest). */
function appendFeed(state: GenerationJobsFeedState, row: Omit<FeedRow, 'seq'>): FeedRow[] {
  const next = [...state.feed, { ...row, seq: state.seq + 1 }];
  return next.length > FEED_CAP ? next.slice(next.length - FEED_CAP) : next;
}

function reducer(state: GenerationJobsFeedState, action: Action): GenerationJobsFeedState {
  switch (action.type) {
    case 'PHASE': {
      const { phase, payload } = action;
      if (!payload.mediaId) return state;
      const genPhase = phase as GenerationPhase;
      const pct =
        phase === 'translating' && typeof payload.percentage === 'number'
          ? Math.round(payload.percentage)
          : null;
      return {
        ...state,
        seq: state.seq + 1,
        feed: appendFeed(state, {
          tone: 'active',
          stage: PHASE_LABEL[genPhase],
          mediaId: payload.mediaId,
          message: payload.message,
          trail: pct !== null ? `${pct}%` : undefined,
        }),
        singleJobs: {
          ...state.singleJobs,
          [payload.mediaId]: {
            mediaId: payload.mediaId,
            phase: genPhase,
            message: payload.message ?? state.singleJobs[payload.mediaId]?.message ?? '',
            percentage: pct ?? state.singleJobs[payload.mediaId]?.percentage ?? null,
          },
        },
      };
    }
    case 'COMPLETE': {
      const { payload } = action;
      if (!payload.mediaId) return state;
      const singleJobs = { ...state.singleJobs };
      delete singleJobs[payload.mediaId]; // terminal — retire the in-flight entry
      return {
        ...state,
        seq: state.seq + 1,
        feed: appendFeed(state, {
          tone: 'done',
          stage: '完成',
          mediaId: payload.mediaId,
          message: payload.message,
        }),
        singleJobs,
      };
    }
    case 'FAILED': {
      const { payload } = action;
      if (!payload.mediaId) return state;
      const singleJobs = { ...state.singleJobs };
      delete singleJobs[payload.mediaId];
      return {
        ...state,
        seq: state.seq + 1,
        feed: appendFeed(state, {
          tone: 'failed',
          stage: '失敗',
          mediaId: payload.mediaId,
          message: payload.error ?? payload.message,
        }),
        singleJobs,
      };
    }
    case 'BATCH': {
      const status = action.payload.status ?? '';
      // Only a status CHANGE contributes a lifecycle row — per-tick frames don't spam.
      if (!status || status === state.lastBatchStatus) {
        return { ...state, lastBatchStatus: status || state.lastBatchStatus };
      }
      const label = BATCH_STATUS_LABEL[status];
      if (!label) return { ...state, lastBatchStatus: status }; // 'running' → no row
      return {
        ...state,
        seq: state.seq + 1,
        lastBatchStatus: status,
        feed: appendFeed(state, {
          tone: status === 'error' ? 'failed' : status === 'complete' ? 'done' : 'info',
          stage: label,
          message: action.payload.currentItem,
        }),
      };
    }
    case 'RESET':
      return initialState;
    default:
      return state;
  }
}

export function useGenerationJobsFeed() {
  const [state, dispatch] = useReducer(reducer, initialState);
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

  const parse = useCallback(<T>(e: MessageEvent): T | null => {
    try {
      const parsed = JSON.parse(e.data);
      return snakeToCamel<T>(parsed.data ?? parsed);
    } catch {
      return null; // ignore malformed frames
    }
  }, []);

  const connect = useCallback(() => {
    if (reconnectRef.current) {
      clearTimeout(reconnectRef.current);
      reconnectRef.current = undefined;
    }
    if (esRef.current) esRef.current.close();
    const es = new EventSource(`${API_BASE_URL}/events`);
    esRef.current = es;

    for (const { event, phase } of TRANSCRIPTION_EVENTS) {
      es.addEventListener(event, (e: MessageEvent) => {
        if (!mountedRef.current) return;
        const payload = parse<TranscriptionPayload>(e);
        if (payload) dispatch({ type: 'PHASE', phase, payload });
      });
    }
    es.addEventListener('transcription_complete', (e: MessageEvent) => {
      if (!mountedRef.current) return;
      const payload = parse<TranscriptionPayload>(e);
      if (payload) dispatch({ type: 'COMPLETE', payload });
    });
    es.addEventListener('transcription_failed', (e: MessageEvent) => {
      if (!mountedRef.current) return;
      const payload = parse<TranscriptionPayload>(e);
      if (payload) dispatch({ type: 'FAILED', payload });
    });
    es.addEventListener('generation_batch_progress', (e: MessageEvent) => {
      if (!mountedRef.current) return;
      const payload = parse<BatchPayload>(e);
      if (payload) dispatch({ type: 'BATCH', payload });
    });

    es.onerror = () => {
      if (!mountedRef.current) return;
      es.close();
      if (reconnectRef.current) clearTimeout(reconnectRef.current);
      reconnectRef.current = setTimeout(() => {
        if (mountedRef.current) connectRef.current();
      }, SSE_RECONNECT_MS);
    };
  }, [parse]);

  useEffect(() => {
    connectRef.current = connect;
  }, [connect]);

  useEffect(() => {
    mountedRef.current = true;
    // NO connect on mount (§8) — the workspace calls startTracking() when active+visible.
    return () => {
      mountedRef.current = false;
      closeSSE();
    };
  }, [closeSSE]);

  const startTracking = useCallback(() => {
    if (!esRef.current || esRef.current.readyState === 2) connect();
  }, [connect]);

  const stop = useCallback(() => {
    closeSSE();
  }, [closeSSE]);

  return {
    feed: state.feed,
    singleJobs: state.singleJobs,
    startTracking,
    stop,
  };
}
