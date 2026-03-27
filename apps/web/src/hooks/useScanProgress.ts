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
        const data = snakeToCamel<ScanProgressEvent>(event.data || event);
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
        const raw = snakeToCamel<Record<string, unknown>>(event.data || event);
        // Update final counts then mark complete
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
  }, [stopPolling, startPolling, resetSseTimeout]);

  // Keep a stable ref to connectSSE to avoid useEffect re-firing
  const connectSSERef = useRef(connectSSE);
  connectSSERef.current = connectSSE;

  // Lazy SSE: only connect when scan is active (avoids persistent SSE connection
  // that blocks Playwright networkidle detection in E2E tests).
  // When idle, poll every 10s to detect if a scan started elsewhere.
  const idlePollRef = useRef<ReturnType<typeof setInterval>>();

  const checkAndConnect = useCallback(async () => {
    try {
      const status = await scannerService.getScanStatus();
      if (!mountedRef.current) return;
      if (status.isScanning) {
        dispatch({ type: 'STATUS_UPDATE', payload: status });
        // Stop idle polling, connect SSE for real-time updates
        if (idlePollRef.current) {
          clearInterval(idlePollRef.current);
          idlePollRef.current = undefined;
        }
        if (!eventSourceRef.current || eventSourceRef.current.readyState === 2) {
          connectSSERef.current();
        }
      }
    } catch {
      // Ignore — will retry on next poll
    }
  }, []);

  useEffect(() => {
    mountedRef.current = true;

    // Initial check — connect SSE only if scan is already running
    checkAndConnect();

    // Light idle poll (10s) to detect scan triggered from settings or schedule
    idlePollRef.current = setInterval(checkAndConnect, 10000);

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
      if (idlePollRef.current) {
        clearInterval(idlePollRef.current);
        idlePollRef.current = undefined;
      }
      if (sseTimeoutRef.current) clearTimeout(sseTimeoutRef.current);
      if (sseReconnectRef.current) clearTimeout(sseReconnectRef.current);
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // Connect SSE on demand — called when a scan is triggered externally
  const startTracking = useCallback(() => {
    if (!eventSourceRef.current || eventSourceRef.current.readyState === 2) {
      connectSSERef.current();
    }
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
