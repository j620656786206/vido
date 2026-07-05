/**
 * Lazy SSE generation-progress hook (ux3-subtitle-v2 AC 8) — models
 * useDownloadProgress.ts / useSubtitleBatchProgress.ts.
 *
 * Listens for the Route C transcription events on the shared GET /api/v1/events
 * stream: `transcription_extracting` / `transcription_progress` /
 * `translation_progress` / `transcription_complete` / `transcription_failed`
 * (declared in services/transcription_service.go — NOT the sse package; do not
 * confuse with the Epic 8 fetch-era `subtitle_progress`, whose payload has
 * `stage` not `phase`).
 *
 * ⚠️ Double-nested envelope: the SSE `data:` line carries the FULL `Event`
 * struct `{"id","type","data":{…payload…}}` — the payload is `parsed.data`,
 * then snakeToCamel at ingest (Rule 18). Payload `media_id` is the INT64 movie
 * id; events are filtered by it (one shared stream, many producers).
 *
 * §8 lazy-SSE rules: NO connect on mount — `startTracking(mediaId)` opens the
 * stream (also the 409-attach path: attach to a job already running server-side).
 * Merge-not-replace reducer, `mountedRef` guard, 10s reconnect backoff (no
 * polling fallback), terminal close on complete/failed, cleanup on unmount.
 */
import { useCallback, useEffect, useReducer, useRef } from 'react';
import { snakeToCamel } from '../utils/caseTransform';

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || '/api/v1';
const SSE_RECONNECT_MS = 10000;

/** Wire phases (transcription_service.go) + the hook's idle resting state. */
export type GenerationPhase =
  | 'idle'
  | 'extracting'
  | 'transcribing'
  | 'translating'
  | 'complete'
  | 'failed';

/** Camelized SSE payload (after envelope unwrap + snakeToCamel). */
interface GenerationEventPayload {
  jobId?: string;
  mediaId?: number;
  phase?: string;
  percentage?: number;
  message?: string;
  error?: string;
  srtPath?: string;
  zhSrtPath?: string;
  duration?: number;
}

export interface GenerationProgressState {
  phase: GenerationPhase;
  /** The phase that was live when `transcription_failed` arrived (失敗於{stage}). */
  failedPhase: 'extracting' | 'transcribing' | 'translating' | null;
  /** translation_progress only (0–100 float). */
  percentage: number | null;
  /** Server-supplied progress text — ALL timing/ETA display comes from here (Rule 23). */
  message: string;
  jobId: string | null;
  error: string | null;
  srtPath: string | null;
  /** Present on complete only when the pipeline translated (translate=true). */
  zhSrtPath: string | null;
}

const initialState: GenerationProgressState = {
  phase: 'idle',
  failedPhase: null,
  percentage: null,
  message: '',
  jobId: null,
  error: null,
  srtPath: null,
  zhSrtPath: null,
};

type ActivePhase = 'extracting' | 'transcribing' | 'translating';

type Action =
  | { type: 'START' }
  | { type: 'PHASE'; phase: ActivePhase; payload: GenerationEventPayload }
  | { type: 'COMPLETE'; payload: GenerationEventPayload }
  | { type: 'FAILED'; payload: GenerationEventPayload }
  | { type: 'RESET' };

function lastActivePhase(phase: GenerationPhase): ActivePhase {
  return phase === 'transcribing' || phase === 'translating' ? phase : 'extracting';
}

function reducer(state: GenerationProgressState, action: Action): GenerationProgressState {
  switch (action.type) {
    case 'START':
      return { ...initialState, phase: 'extracting' };
    case 'PHASE':
      // Merge-not-replace: keep the last message/jobId when an event omits them.
      return {
        ...state,
        phase: action.phase,
        failedPhase: null,
        error: null,
        percentage:
          action.phase === 'translating' ? (action.payload.percentage ?? state.percentage) : null,
        message: action.payload.message ?? state.message,
        jobId: action.payload.jobId ?? state.jobId,
      };
    case 'COMPLETE':
      return {
        ...state,
        phase: 'complete',
        failedPhase: null,
        error: null,
        percentage: null,
        message: action.payload.message ?? state.message,
        jobId: action.payload.jobId ?? state.jobId,
        srtPath: action.payload.srtPath ?? null,
        zhSrtPath: action.payload.zhSrtPath ?? null,
      };
    case 'FAILED':
      return {
        ...state,
        phase: 'failed',
        // The failed payload's own phase is the literal "failed" — the stage where it
        // failed is the last active phase this hook observed.
        failedPhase: lastActivePhase(state.phase),
        percentage: null,
        message: action.payload.message ?? state.message,
        jobId: action.payload.jobId ?? state.jobId,
        error: action.payload.error ?? action.payload.message ?? '生成失敗',
      };
    case 'RESET':
      return initialState;
    default:
      return state;
  }
}

const PHASE_EVENTS: ReadonlyArray<{ event: string; phase: ActivePhase }> = [
  { event: 'transcription_extracting', phase: 'extracting' },
  { event: 'transcription_progress', phase: 'transcribing' },
  { event: 'translation_progress', phase: 'translating' },
];

export interface UseGenerationProgressOptions {
  /** Fired once per `transcription_complete` for the tracked media (AC 6 invalidation hook). */
  onComplete?: (payload: { srtPath: string | null; zhSrtPath: string | null }) => void;
}

export function useGenerationProgress(options?: UseGenerationProgressOptions) {
  const [progress, dispatch] = useReducer(reducer, initialState);
  const esRef = useRef<EventSource | null>(null);
  const reconnectRef = useRef<ReturnType<typeof setTimeout> | undefined>(undefined);
  const mountedRef = useRef(true);
  const connectRef = useRef<() => void>(() => {});
  /** int64 movie id currently tracked; null = drop everything. */
  const mediaIdRef = useRef<number | null>(null);
  const onCompleteRef = useRef(options?.onComplete);

  useEffect(() => {
    onCompleteRef.current = options?.onComplete;
  }, [options?.onComplete]);

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

  /** Unwrap the double-nested envelope and filter by tracked media_id. */
  const parsePayload = useCallback((e: MessageEvent): GenerationEventPayload | null => {
    try {
      const parsed = JSON.parse(e.data);
      // data: line = full Event struct {id,type,data} → payload is parsed.data.
      const payload = snakeToCamel<GenerationEventPayload>(parsed.data ?? parsed);
      if (mediaIdRef.current === null || payload.mediaId !== mediaIdRef.current) return null;
      return payload;
    } catch {
      return null; // ignore malformed frames
    }
  }, []);

  const connect = useCallback(() => {
    // Cancel any pending backoff reconnect: a stale timer surviving into a fresh
    // connection would bounce the healthy stream 10s later — a terminal event
    // landing in that gap would be lost (no further events ever recover it).
    if (reconnectRef.current) {
      clearTimeout(reconnectRef.current);
      reconnectRef.current = undefined;
    }
    if (esRef.current) esRef.current.close();
    const es = new EventSource(`${API_BASE_URL}/events`);
    esRef.current = es;

    for (const { event, phase } of PHASE_EVENTS) {
      es.addEventListener(event, (e: MessageEvent) => {
        if (!mountedRef.current) return;
        const payload = parsePayload(e);
        if (payload) dispatch({ type: 'PHASE', phase, payload });
      });
    }

    es.addEventListener('transcription_complete', (e: MessageEvent) => {
      if (!mountedRef.current) return;
      const payload = parsePayload(e);
      if (!payload) return;
      dispatch({ type: 'COMPLETE', payload });
      onCompleteRef.current?.({
        srtPath: payload.srtPath ?? null,
        zhSrtPath: payload.zhSrtPath ?? null,
      });
      closeSSE(); // terminal — no more events for this job
    });

    es.addEventListener('transcription_failed', (e: MessageEvent) => {
      if (!mountedRef.current) return;
      const payload = parsePayload(e);
      if (!payload) return;
      dispatch({ type: 'FAILED', payload });
      closeSSE(); // terminal
    });

    es.onerror = () => {
      if (!mountedRef.current) return;
      es.close();
      if (reconnectRef.current) clearTimeout(reconnectRef.current);
      reconnectRef.current = setTimeout(() => {
        if (mountedRef.current) connectRef.current();
      }, SSE_RECONNECT_MS);
    };
  }, [closeSSE, parsePayload]);

  useEffect(() => {
    connectRef.current = connect;
  }, [connect]);

  useEffect(() => {
    mountedRef.current = true;
    // NO connect on mount (§8) — consumers call startTracking() after triggering
    // (or attaching to) a generation job.
    return () => {
      mountedRef.current = false;
      closeSSE();
    };
  }, [closeSSE]);

  /**
   * Open the stream and track one movie's generation job. Also the 409-attach
   * path — safe to call when the job was already running server-side.
   */
  const startTracking = useCallback(
    (mediaId: number) => {
      mediaIdRef.current = mediaId;
      dispatch({ type: 'START' });
      if (!esRef.current || esRef.current.readyState === 2) connect();
    },
    [connect]
  );

  /** Tear down the stream and return to idle (e.g. when the dialog closes). */
  const reset = useCallback(() => {
    mediaIdRef.current = null;
    closeSSE();
    dispatch({ type: 'RESET' });
  }, [closeSSE]);

  return { progress, startTracking, reset };
}
