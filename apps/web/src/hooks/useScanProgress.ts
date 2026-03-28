/**
 * SSE-powered scan progress hook (Story 7.4)
 * Connects to EventSource at /api/v1/events, filters scan_progress events.
 * Falls back to polling GET /scanner/status on SSE error/timeout.
 */

import { useEffect, useReducer, useCallback, useRef } from 'react';
import { scannerService } from '../services/scannerService';
import type { ScanProgressEvent, ScanStatus } from '../services/scannerService';
import { snakeToCamel } from '../utils/caseTransform';

export interface ScanProgressState {
  isScanning: boolean;
  percentDone: number;
  currentFile: string;
  filesFound: number;
  filesProcessed: number;
  errorCount: number;
  estimatedTime: string;
  isComplete: boolean;
  isCancelled: boolean;
  isMinimized: boolean;
  isDismissed: boolean;
  connectionStatus: 'sse' | 'polling' | 'disconnected';
}

type ScanProgressAction =
  | { type: 'SSE_UPDATE'; payload: ScanProgressEvent }
  | { type: 'STATUS_UPDATE'; payload: ScanStatus }
  | { type: 'SCAN_COMPLETE' }
  | { type: 'SCAN_CANCELLED' }
  | { type: 'SCAN_IDLE' }
  | { type: 'TOGGLE_MINIMIZE' }
  | { type: 'DISMISS' }
  | { type: 'SET_CONNECTION'; payload: ScanProgressState['connectionStatus'] };

const initialState: ScanProgressState = {
  isScanning: false,
  percentDone: 0,
  currentFile: '',
  filesFound: 0,
  filesProcessed: 0,
  errorCount: 0,
  estimatedTime: '',
  isComplete: false,
  isCancelled: false,
  isMinimized: false,
  isDismissed: true,
  connectionStatus: 'disconnected',
};

function scanProgressReducer(
  state: ScanProgressState,
  action: ScanProgressAction
): ScanProgressState {
  switch (action.type) {
    case 'SSE_UPDATE': {
      const pct = action.payload.percentDone;
      const found = action.payload.filesFound;
      // Estimate filesProcessed from percentDone × filesFound when not provided by SSE
      const estimatedProcessed = found > 0 ? Math.round((pct / 100) * found) : 0;
      return {
        ...state,
        isScanning: true,
        percentDone: pct,
        currentFile: action.payload.currentFile,
        filesFound: found,
        filesProcessed: (action.payload as unknown as Record<string, unknown>).filesProcessed
          ? Number((action.payload as unknown as Record<string, unknown>).filesProcessed)
          : estimatedProcessed,
        errorCount: action.payload.errorCount,
        estimatedTime: action.payload.estimatedTime,
        isComplete: false,
        isCancelled: false,
        isDismissed: false,
      };
    }
    case 'STATUS_UPDATE': {
      const p = action.payload;
      if (!p.isScanning && state.isScanning) {
        // Scan just finished — mark complete
        return {
          ...state,
          isScanning: false,
          percentDone: 100,
          filesFound: p.filesFound,
          filesProcessed: p.filesProcessed,
          errorCount: p.errorCount,
          currentFile: '',
          estimatedTime: '',
          isComplete: true,
          isDismissed: false,
        };
      }
      if (p.isScanning) {
        return {
          ...state,
          isScanning: true,
          percentDone: p.percentDone,
          currentFile: p.currentFile,
          filesFound: p.filesFound,
          filesProcessed: p.filesProcessed,
          errorCount: p.errorCount,
          estimatedTime: p.estimatedTime,
          isComplete: false,
          isCancelled: false,
          isDismissed: false,
        };
      }
      // Not scanning and we weren't scanning — stay idle
      return state;
    }
    case 'SCAN_COMPLETE':
      return {
        ...state,
        isScanning: false,
        percentDone: 100,
        isComplete: true,
        isCancelled: false,
        currentFile: '',
        estimatedTime: '',
        isDismissed: false,
      };
    case 'SCAN_CANCELLED':
      return {
        ...state,
        isScanning: false,
        isCancelled: true,
        isComplete: false,
        currentFile: '',
        estimatedTime: '',
        isDismissed: false,
      };
    case 'SCAN_IDLE':
      return initialState;
    case 'TOGGLE_MINIMIZE':
      return { ...state, isMinimized: !state.isMinimized };
    case 'DISMISS':
      return { ...state, isDismissed: true, isComplete: false, isCancelled: false };
    case 'SET_CONNECTION':
      return { ...state, connectionStatus: action.payload };
    default:
      return state;
  }
}

const SSE_RECONNECT_MS = 10000;

export function useScanProgress() {
  const [state, dispatch] = useReducer(scanProgressReducer, initialState);
  const eventSourceRef = useRef<EventSource | null>(null);
  const sseReconnectRef = useRef<ReturnType<typeof setTimeout>>();
  const mountedRef = useRef(true);

  const connectSSE = useCallback(() => {
    // Close existing connection
    if (eventSourceRef.current) {
      eventSourceRef.current.close();
    }

    const url = scannerService.getSSEUrl();
    const es = new EventSource(url);
    eventSourceRef.current = es;

    es.addEventListener('connected', () => {
      if (!mountedRef.current) return;
      dispatch({ type: 'SET_CONNECTION', payload: 'sse' });
    });

    es.addEventListener('scan_progress', (e: MessageEvent) => {
      if (!mountedRef.current) return;
      try {
        const event = JSON.parse(e.data);
        const data = snakeToCamel<ScanProgressEvent>(event.data || event);
        dispatch({ type: 'SSE_UPDATE', payload: data });
      } catch {
        // Ignore parse errors
      }
    });

    es.addEventListener('scan_complete', (e: MessageEvent) => {
      if (!mountedRef.current) return;
      try {
        const event = JSON.parse(e.data);
        const raw = snakeToCamel<Record<string, unknown>>(event.data || event);
        dispatch({
          type: 'SSE_UPDATE',
          payload: {
            filesFound: (raw.filesFound as number) ?? 0,
            currentFile: '',
            percentDone: 100,
            errorCount: (raw.errorCount as number) ?? 0,
            estimatedTime: '',
          },
        });
        dispatch({ type: 'SCAN_COMPLETE' });
      } catch {
        dispatch({ type: 'SCAN_COMPLETE' });
      }
    });

    es.addEventListener('scan_cancelled', () => {
      if (!mountedRef.current) return;
      dispatch({ type: 'SCAN_CANCELLED' });
    });

    es.addEventListener('ping', () => {
      // Keep-alive — no action needed
    });

    es.onerror = () => {
      if (!mountedRef.current) return;
      // SSE connection lost — schedule reconnect (no polling fallback)
      es.close();
      dispatch({ type: 'SET_CONNECTION', payload: 'disconnected' });
      if (sseReconnectRef.current) clearTimeout(sseReconnectRef.current);
      sseReconnectRef.current = setTimeout(() => {
        if (mountedRef.current) connectSSE();
      }, SSE_RECONNECT_MS);
    };
  }, []);

  useEffect(() => {
    mountedRef.current = true;

    // No polling — SSE only. Connect on demand via startTracking().

    return () => {
      mountedRef.current = false;
      if (eventSourceRef.current) {
        eventSourceRef.current.close();
        eventSourceRef.current = null;
      }
      if (sseReconnectRef.current) clearTimeout(sseReconnectRef.current);
    };
  }, []);

  // Connect SSE on demand — called when a scan is triggered externally
  const startTracking = useCallback(() => {
    if (!eventSourceRef.current || eventSourceRef.current.readyState === 2) {
      connectSSE();
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const toggleMinimize = useCallback(() => {
    dispatch({ type: 'TOGGLE_MINIMIZE' });
  }, []);

  const dismiss = useCallback(() => {
    dispatch({ type: 'DISMISS' });
  }, []);

  const isVisible = state.isScanning || state.isComplete || state.isCancelled;
  const showCard = isVisible && !state.isDismissed;

  return {
    ...state,
    isVisible: showCard,
    startTracking,
    toggleMinimize,
    dismiss,
  };
}
