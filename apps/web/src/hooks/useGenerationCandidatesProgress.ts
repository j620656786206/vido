/**
 * Lazy SSE candidate-ANALYSIS progress hook (sub-4-3 AC #1) — clone of the
 * useGenerationBatchProgress lazy-SSE skeleton listening for the sub-4-1
 * `generation_candidates_progress` events on the shared GET /api/v1/events
 * stream. Drives the F14 「分析字幕軌 {analyzed} / {total}」 progress view.
 *
 * ⚠️ Double-nested envelope: the SSE `data:` line carries the FULL `Event`
 * struct `{"id","type","data":{…payload…}}` — unwrap via `parsed.data`, then
 * snakeToCamel (Rule 18). Payload carries COUNTS ONLY (`status, analyzed,
 * total, error`) — the candidate result never rides the stream; fetch it via
 * GET /subtitles/generation-candidates on the `ready` transition (sub-4-1
 * CR L2 contract).
 *
 * §8 lazy-SSE rules: NO connect on mount — `startTracking()` opens the stream
 * (also the 409 TRANSCRIPTION_ANALYSIS_RUNNING join path). Terminal statuses
 * `ready | cancelled | error` close the stream; 10s reconnect backoff;
 * `mountedRef` guard; cleanup on unmount.
 *
 * Rule 23: zero wall-clock reads — counts are all SSE-supplied.
 */
import { useCallback, useEffect, useReducer, useRef } from 'react';
import { snakeToCamel } from '../utils/caseTransform';
import type { CandidateAnalysisStatus } from '../services/subtitleService';

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || '/api/v1';
const SSE_RECONNECT_MS = 10000;

/** Wire payload of `generation_candidates_progress` (counts only, no result). */
interface CandidatesProgressPayload {
  status: Exclude<CandidateAnalysisStatus, 'idle'>;
  analyzed: number;
  total: number;
  error?: string;
}

export interface CandidateAnalysisProgressState {
  status: CandidateAnalysisStatus;
  analyzed: number;
  total: number;
  error: string | null;
}

const initialState: CandidateAnalysisProgressState = {
  status: 'idle',
  analyzed: 0,
  total: 0,
  error: null,
};

type Action =
  | { type: 'START'; payload: Partial<CandidateAnalysisProgressState> }
  | { type: 'SSE_UPDATE'; payload: CandidatesProgressPayload }
  | { type: 'RESET' };

function reducer(
  state: CandidateAnalysisProgressState,
  action: Action
): CandidateAnalysisProgressState {
  switch (action.type) {
    case 'START':
      return { ...initialState, status: 'analyzing', ...action.payload };
    case 'SSE_UPDATE': {
      const p = action.payload;
      return {
        status: p.status ?? state.status,
        analyzed: p.analyzed ?? state.analyzed,
        total: p.total ?? state.total,
        error: p.error ?? null,
      };
    }
    case 'RESET':
      return initialState;
    default:
      return state;
  }
}

function isTerminal(status: CandidateAnalysisStatus): boolean {
  return status === 'ready' || status === 'cancelled' || status === 'error';
}

export function useGenerationCandidatesProgress() {
  const [progress, dispatch] = useReducer(reducer, initialState);
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

    es.addEventListener('generation_candidates_progress', (e: MessageEvent) => {
      if (!mountedRef.current) return;
      try {
        const parsed = JSON.parse(e.data);
        // data: line = full Event struct {id,type,data} → payload is parsed.data.
        const payload = snakeToCamel<CandidatesProgressPayload>(parsed.data ?? parsed);
        dispatch({ type: 'SSE_UPDATE', payload });
        if (isTerminal(payload.status)) closeSSE();
      } catch {
        // Ignore malformed frames.
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
  }, [closeSSE]);

  useEffect(() => {
    connectRef.current = connect;
  }, [connect]);

  useEffect(() => {
    mountedRef.current = true;
    // NO connect on mount (§8) — consumers call startTracking() after POST
    // analyze 202s (or joins a 409-already-running analysis).
    return () => {
      mountedRef.current = false;
      closeSSE();
    };
  }, [closeSSE]);

  /**
   * Open the stream and enter the analyzing state. Call AFTER POST analyze
   * succeeds or 409-joins — also seedable from a GET snapshot (`analyzed` /
   * `total` counts) so the bar doesn't jump from 0.
   */
  const startTracking = useCallback(
    (seed?: Partial<CandidateAnalysisProgressState>) => {
      // CR sub-4-3 M3: a caller's in-flight promise can resolve AFTER unmount
      // (Escape/✕ before the GET settles) — opening a stream then would leak a
      // connection nothing ever closes.
      if (!mountedRef.current) return;
      dispatch({ type: 'START', payload: seed ?? {} });
      if (!esRef.current || esRef.current.readyState === 2) connect();
    },
    [connect]
  );

  /** Tear down the stream and return to idle (e.g. when the dialog closes). */
  const reset = useCallback(() => {
    closeSSE();
    dispatch({ type: 'RESET' });
  }, [closeSSE]);

  return { progress, status: progress.status, startTracking, reset };
}
