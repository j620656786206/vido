/**
 * SSE-powered scan progress hook (Story 7.4)
 * Connects to EventSource at /api/v1/events, filters scan_progress events.
 * Falls back to polling GET /scanner/status on SSE error/timeout.
 */

import { useEffect, useReducer, useCallback, useRef } from 'react';
import { scannerService } from '../services/scannerService';
import type { ScanProgressEvent, ScanStatus } from '../services/scannerService';

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
      const pct = action.payload.percent_done;
      const found = action.payload.files_found;
      // Estimate filesProcessed from percent_done × filesFound when not provided by SSE
      const estimatedProcessed = found > 0 ? Math.round((pct / 100) * found) : 0;
      return {
        ...state,
        isScanning: true,
        percentDone: pct,
        currentFile: action.payload.current_file,
        filesFound: found,
        filesProcessed: (action.payload as Record<string, unknown>).files_processed
          ? Number((action.payload as Record<string, unknown>).files_processed)
          : estimatedProcessed,
        errorCount: action.payload.error_count,
        estimatedTime: action.payload.estimated_time,
        isComplete: false,
        isCancelled: false,
        isDismissed: false,
      };
    }
    case 'STATUS_UPDATE': {
      const p = action.payload;
      if (!p.is_scanning && state.isScanning) {
        // Scan just finished — mark complete
        return {
          ...state,
          isScanning: false,
          percentDone: 100,
          filesFound: p.files_found,
          filesProcessed: p.files_processed,
          errorCount: p.error_count,
          currentFile: '',
          estimatedTime: '',
          isComplete: true,
          isDismissed: false,
        };
      }
      if (p.is_scanning) {
        return {
          ...state,
          isScanning: true,
          percentDone: p.percent_done,
          currentFile: p.current_file,
          filesFound: p.files_found,
          filesProcessed: p.files_processed,
          errorCount: p.error_count,
          estimatedTime: p.estimated_time,
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

const SSE_TIMEOUT_MS = 5000;
const POLL_INTERVAL_MS = 3000;
const SSE_RECONNECT_MS = 10000;

export function useScanProgress() {
  const [state, dispatch] = useReducer(scanProgressReducer, initialState);
  const eventSourceRef = useRef<EventSource | null>(null);
  const pollTimerRef = useRef<ReturnType<typeof setInterval>>();
  const sseTimeoutRef = useRef<ReturnType<typeof setTimeout>>();
  const sseReconnectRef = useRef<ReturnType<typeof setTimeout>>();
  const mountedRef = useRef(true);

  const stopPolling = useCallback(() => {
    if (pollTimerRef.current) {
      clearInterval(pollTimerRef.current);
      pollTimerRef.current = undefined;
    }
  }, []);

  const pollStatus = useCallback(async () => {
    try {
      const status = await scannerService.getScanStatus();
      if (!mountedRef.current) return;
      dispatch({ type: 'STATUS_UPDATE', payload: status });
    } catch {
      // Ignore poll errors — will retry on next interval
    }
  }, []);

  const startPolling = useCallback(() => {
    stopPolling();
    dispatch({ type: 'SET_CONNECTION', payload: 'polling' });
    // Immediate first poll
    pollStatus();
    pollTimerRef.current = setInterval(pollStatus, POLL_INTERVAL_MS);
  }, [stopPolling, pollStatus]);

  const resetSseTimeout = useCallback(() => {
    if (sseTimeoutRef.current) clearTimeout(sseTimeoutRef.current);
    sseTimeoutRef.current = setTimeout(() => {
      // SSE silent for too long — fall back to polling
      startPolling();
    }, SSE_TIMEOUT_MS);
  }, [startPolling]);

  const connectSSE = useCallback(() => {
    // Close existing connection
    if (eventSourceRef.current) {
      eventSourceRef.current.close();
    }
    stopPolling();

    const url = scannerService.getSSEUrl();
    const es = new EventSource(url);
    eventSourceRef.current = es;

    es.addEventListener('connected', () => {
      if (!mountedRef.current) return;
      dispatch({ type: 'SET_CONNECTION', payload: 'sse' });
      resetSseTimeout();
    });

    es.addEventListener('scan_progress', (e: MessageEvent) => {
      if (!mountedRef.current) return;
      resetSseTimeout();
      try {
        const event = JSON.parse(e.data);
        const data: ScanProgressEvent = event.data || event;
        dispatch({ type: 'SSE_UPDATE', payload: data });
      } catch {
        // Ignore parse errors
      }
    });

    es.addEventListener('scan_complete', (e: MessageEvent) => {
      if (!mountedRef.current) return;
      resetSseTimeout();
      try {
        const event = JSON.parse(e.data);
        const data = event.data || event;
        // Update final counts then mark complete
        dispatch({
          type: 'SSE_UPDATE',
          payload: {
            files_found: data.files_found ?? 0,
            current_file: '',
            percent_done: 100,
            error_count: data.error_count ?? 0,
            estimated_time: '',
          },
        });
        dispatch({ type: 'SCAN_COMPLETE' });
      } catch {
        dispatch({ type: 'SCAN_COMPLETE' });
      }
    });

    es.addEventListener('scan_cancelled', () => {
      if (!mountedRef.current) return;
      resetSseTimeout();
      dispatch({ type: 'SCAN_CANCELLED' });
    });

    es.addEventListener('ping', () => {
      if (!mountedRef.current) return;
      resetSseTimeout();
    });

    es.onerror = () => {
      if (!mountedRef.current) return;
      // SSE connection lost — fall back to polling, schedule reconnect
      es.close();
      startPolling();
      if (sseReconnectRef.current) clearTimeout(sseReconnectRef.current);
      sseReconnectRef.current = setTimeout(() => {
        if (mountedRef.current) connectSSE();
      }, SSE_RECONNECT_MS);
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [stopPolling, startPolling, resetSseTimeout]);

  // Keep a stable ref to connectSSE to avoid useEffect re-firing
  const connectSSERef = useRef(connectSSE);
  connectSSERef.current = connectSSE;

  // Connect on mount, fetch initial status
  useEffect(() => {
    mountedRef.current = true;

    // Fetch initial status to decide if scan is running
    (async () => {
      try {
        const status = await scannerService.getScanStatus();
        if (!mountedRef.current) return;
        if (status.is_scanning) {
          dispatch({ type: 'STATUS_UPDATE', payload: status });
        }
      } catch {
        // Ignore initial fetch errors
      }
    })();

    connectSSERef.current();

    return () => {
      mountedRef.current = false;
      if (eventSourceRef.current) {
        eventSourceRef.current.close();
        eventSourceRef.current = null;
      }
      if (pollTimerRef.current) {
        clearInterval(pollTimerRef.current);
        pollTimerRef.current = undefined;
      }
      if (sseTimeoutRef.current) clearTimeout(sseTimeoutRef.current);
      if (sseReconnectRef.current) clearTimeout(sseReconnectRef.current);
    };
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
    toggleMinimize,
    dismiss,
  };
}
