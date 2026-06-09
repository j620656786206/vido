/**
 * SSE-powered batch subtitle progress hook (Story 8.11).
 *
 * Consumes the `subtitle_batch_progress` events broadcast by the Story 8-9
 * backend over the shared GET /api/v1/events stream.
 *
 * CRITICAL — Lazy SSE (project-context.md §8, Epic 7 retro lesson):
 * NEVER open EventSource on mount. It connects only after `startTracking()` is
 * called (i.e. once a batch has actually started), and closes on the terminal
 * `complete` / `cancelled` event and on unmount. Modeled on `useScanProgress`.
 */

import { useEffect, useReducer, useCallback, useRef } from 'react';
import { snakeToCamel } from '../utils/caseTransform';
import type { BatchProgress } from '../services/subtitleService';

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || '/api/v1';
const SSE_RECONNECT_MS = 10000;

export type BatchStatus = 'idle' | 'running' | 'complete' | 'cancelled' | 'error';

export interface BatchProgressState {
  batchId: string;
  totalItems: number;
  currentIndex: number;
  currentItem: string;
  successCount: number;
  failCount: number;
  status: BatchStatus;
}

const initialState: BatchProgressState = {
  batchId: '',
  totalItems: 0,
  currentIndex: 0,
  currentItem: '',
  successCount: 0,
  failCount: 0,
  status: 'idle',
};

type Action =
  | { type: 'START'; payload: Partial<BatchProgressState> }
  | { type: 'SSE_UPDATE'; payload: BatchProgress }
  | { type: 'RESET' };

function reducer(state: BatchProgressState, action: Action): BatchProgressState {
  switch (action.type) {
    case 'START':
      return {
        ...initialState,
        ...action.payload,
        status: 'running',
      };
    case 'SSE_UPDATE': {
      const p = action.payload;
      // Map the backend status string onto our state machine. The backend uses
      // "running" | "complete" | "cancelled" | "error".
      const status: BatchStatus = p.status ?? 'running';
      return {
        ...state,
        batchId: p.batchId || state.batchId,
        totalItems: p.totalItems ?? state.totalItems,
        currentIndex: p.currentIndex ?? state.currentIndex,
        // The terminal "complete" event omits current_item — keep the last one.
        currentItem: p.currentItem ?? state.currentItem,
        successCount: p.successCount ?? state.successCount,
        failCount: p.failCount ?? state.failCount,
        status,
      };
    }
    case 'RESET':
      return initialState;
    default:
      return state;
  }
}

function isTerminal(status: BatchStatus): boolean {
  return status === 'complete' || status === 'cancelled' || status === 'error';
}

export function useSubtitleBatchProgress() {
  const [progress, dispatch] = useReducer(reducer, initialState);
  const eventSourceRef = useRef<EventSource | null>(null);
  const sseReconnectRef = useRef<ReturnType<typeof setTimeout>>();
  const mountedRef = useRef(true);
  // Holds the latest connectSSE so the reconnect timer can call it without a
  // self-referential useCallback (which trips react-hooks/immutability).
  const connectRef = useRef<() => void>(() => {});

  const closeSSE = useCallback(() => {
    if (eventSourceRef.current) {
      eventSourceRef.current.close();
      eventSourceRef.current = null;
    }
    if (sseReconnectRef.current) {
      clearTimeout(sseReconnectRef.current);
      sseReconnectRef.current = undefined;
    }
  }, []);

  const connectSSE = useCallback(() => {
    if (eventSourceRef.current) {
      eventSourceRef.current.close();
    }

    const es = new EventSource(`${API_BASE_URL}/events`);
    eventSourceRef.current = es;

    es.addEventListener('subtitle_batch_progress', (e: MessageEvent) => {
      if (!mountedRef.current) return;
      try {
        const event = JSON.parse(e.data);
        const data = snakeToCamel<BatchProgress>(event.data || event);
        dispatch({ type: 'SSE_UPDATE', payload: data });
        // Terminal event → close the stream (no polling fallback).
        if (isTerminal((data.status as BatchStatus) ?? 'running')) {
          closeSSE();
        }
      } catch {
        // Ignore malformed payloads.
      }
    });

    es.onerror = () => {
      if (!mountedRef.current) return;
      es.close();
      // Reconnect with a fixed backoff while a batch is still in flight.
      if (sseReconnectRef.current) clearTimeout(sseReconnectRef.current);
      sseReconnectRef.current = setTimeout(() => {
        if (mountedRef.current) connectRef.current();
      }, SSE_RECONNECT_MS);
    };
  }, [closeSSE]);

  useEffect(() => {
    mountedRef.current = true;
    // Keep the ref pointed at the latest connectSSE for the reconnect timer.
    connectRef.current = connectSSE;
    // No connect-on-mount. SSE opens only via startTracking().
    return () => {
      mountedRef.current = false;
      closeSSE();
    };
  }, [closeSSE, connectSSE]);

  /** Open the SSE stream and enter the running state. Call AFTER a batch starts. */
  const startTracking = useCallback(
    (seed?: Partial<BatchProgressState>) => {
      dispatch({ type: 'START', payload: seed ?? {} });
      if (!eventSourceRef.current || eventSourceRef.current.readyState === 2) {
        connectSSE();
      }
    },
    [connectSSE]
  );

  /** Tear down the stream and return to idle (e.g. when the dialog closes). */
  const reset = useCallback(() => {
    closeSSE();
    dispatch({ type: 'RESET' });
  }, [closeSSE]);

  return {
    progress,
    status: progress.status,
    startTracking,
    reset,
  };
}
