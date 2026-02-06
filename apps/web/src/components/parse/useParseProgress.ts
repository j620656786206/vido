/**
 * useParseProgress Hook (Story 3.10 - Task 9)
 * React hook for subscribing to parse progress SSE events
 */

import { useEffect, useRef, useState, useCallback } from 'react';
import type {
  ParseProgress,
  ParseStatus,
  ParseEventType,
  StepEventData,
  ParseCompletedData,
  ParseFailedData,
  ParseStartedData,
} from './types';
import { STANDARD_PARSE_STEPS } from './types';

// API base URL - should match your backend
const API_BASE_URL = '/api/v1';

export interface UseParseProgressOptions {
  /** Called when connection is established */
  onConnected?: () => void;
  /** Called when parse starts */
  onParseStarted?: (data: ParseStartedData) => void;
  /** Called when a step starts */
  onStepStarted?: (data: StepEventData) => void;
  /** Called when a step completes */
  onStepCompleted?: (data: StepEventData) => void;
  /** Called when a step fails */
  onStepFailed?: (data: StepEventData) => void;
  /** Called when parse completes successfully */
  onParseCompleted?: (data: ParseCompletedData) => void;
  /** Called when parse fails */
  onParseFailed?: (data: ParseFailedData) => void;
  /** Called on any error */
  onError?: (error: Error) => void;
  /** Auto-reconnect on disconnect (default: true) */
  autoReconnect?: boolean;
  /** Reconnection delay in ms (default: 3000) */
  reconnectDelay?: number;
  /** Maximum reconnection attempts (default: 5, 0 for unlimited) */
  maxReconnectAttempts?: number;
}

export interface UseParseProgressResult {
  /** Current progress state */
  progress: ParseProgress | null;
  /** Current overall status */
  status: ParseStatus;
  /** Connection error if any */
  error: Error | null;
  /** Whether currently connected to SSE */
  isConnected: boolean;
  /** Whether currently reconnecting */
  isReconnecting: boolean;
  /** Manually disconnect from SSE */
  disconnect: () => void;
  /** Manually reconnect to SSE */
  reconnect: () => void;
}

/**
 * Hook for subscribing to parse progress events via SSE
 */
export function useParseProgress(
  taskId: string | null,
  options: UseParseProgressOptions = {}
): UseParseProgressResult {
  const {
    onConnected,
    onParseStarted,
    onStepStarted,
    onStepCompleted,
    onStepFailed,
    onParseCompleted,
    onParseFailed,
    onError,
    autoReconnect = true,
    reconnectDelay = 3000,
    maxReconnectAttempts = 5,
  } = options;

  const [progress, setProgress] = useState<ParseProgress | null>(null);
  const [status, setStatus] = useState<ParseStatus>('pending');
  const [error, setError] = useState<Error | null>(null);
  const [isConnected, setIsConnected] = useState(false);
  const [isReconnecting, setIsReconnecting] = useState(false);

  const eventSourceRef = useRef<EventSource | null>(null);
  const reconnectTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const isManuallyDisconnectedRef = useRef(false);
  const connectRef = useRef<(() => void) | null>(null);
  const reconnectAttemptsRef = useRef(0);

  // Initialize default progress
  const initializeProgress = useCallback(
    (filename = 'Unknown') => {
      const initialProgress: ParseProgress = {
        taskId: taskId || '',
        filename,
        status: 'pending',
        steps: [...STANDARD_PARSE_STEPS],
        currentStep: 0,
        percentage: 0,
        startedAt: new Date().toISOString(),
      };
      setProgress(initialProgress);
      setStatus('pending');
    },
    [taskId]
  );

  // Handle incoming SSE events
  const handleEvent = useCallback(
    (eventType: ParseEventType, data: unknown) => {
      switch (eventType) {
        case 'connected':
          setIsConnected(true);
          setIsReconnecting(false);
          onConnected?.();
          break;

        case 'parse_started': {
          const startedData = data as ParseStartedData;
          const newProgress: ParseProgress = {
            taskId: taskId || '',
            filename: startedData.filename,
            status: 'pending',
            steps: startedData.steps,
            currentStep: 0,
            percentage: 0,
            startedAt: new Date().toISOString(),
          };
          setProgress(newProgress);
          setStatus('pending');
          onParseStarted?.(startedData);
          break;
        }

        case 'step_started': {
          const stepData = data as StepEventData;
          if (stepData.progress) {
            setProgress(stepData.progress);
            setStatus(stepData.progress.status);
          }
          onStepStarted?.(stepData);
          break;
        }

        case 'step_completed': {
          const stepData = data as StepEventData;
          if (stepData.progress) {
            setProgress(stepData.progress);
            setStatus(stepData.progress.status);
          }
          onStepCompleted?.(stepData);
          break;
        }

        case 'step_failed': {
          const stepData = data as StepEventData;
          if (stepData.progress) {
            setProgress(stepData.progress);
            setStatus(stepData.progress.status);
          }
          onStepFailed?.(stepData);
          break;
        }

        case 'step_skipped': {
          const stepData = data as StepEventData;
          if (stepData.progress) {
            setProgress(stepData.progress);
            setStatus(stepData.progress.status);
          }
          break;
        }

        case 'parse_completed': {
          const completedData = data as ParseCompletedData;
          setProgress(completedData.progress);
          setStatus('success');
          onParseCompleted?.(completedData);
          break;
        }

        case 'parse_failed': {
          const failedData = data as ParseFailedData;
          setProgress(failedData.progress);
          setStatus('failed');
          onParseFailed?.(failedData);
          break;
        }

        case 'progress_update': {
          const updateData = data as { progress: ParseProgress };
          if (updateData.progress) {
            setProgress(updateData.progress);
            setStatus(updateData.progress.status);
          }
          break;
        }

        case 'ping':
          // Keepalive ping, no action needed
          break;

        default:
          console.warn('Unknown parse event type:', eventType);
      }
    },
    [
      taskId,
      onConnected,
      onParseStarted,
      onStepStarted,
      onStepCompleted,
      onStepFailed,
      onParseCompleted,
      onParseFailed,
    ]
  );

  // Connect to SSE
  const connect = useCallback(() => {
    if (!taskId) return;
    if (eventSourceRef.current) {
      eventSourceRef.current.close();
    }

    isManuallyDisconnectedRef.current = false;
    setError(null);

    const url = `${API_BASE_URL}/parse/progress/${taskId}`;
    const eventSource = new EventSource(url);
    eventSourceRef.current = eventSource;

    // Handle generic message (fallback)
    eventSource.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data);
        handleEvent(data.type, data);
      } catch (err) {
        console.error('Failed to parse SSE message:', err);
      }
    };

    // Handle specific event types
    const eventTypes: ParseEventType[] = [
      'connected',
      'parse_started',
      'step_started',
      'step_completed',
      'step_failed',
      'step_skipped',
      'parse_completed',
      'parse_failed',
      'progress_update',
      'ping',
    ];

    eventTypes.forEach((type) => {
      eventSource.addEventListener(type, (event: MessageEvent) => {
        try {
          const data = JSON.parse(event.data);
          handleEvent(type, data.data || data);
        } catch (err) {
          console.error(`Failed to parse ${type} event:`, err);
        }
      });
    });

    // Handle connection open
    eventSource.onopen = () => {
      setIsConnected(true);
      setIsReconnecting(false);
      setError(null);
      reconnectAttemptsRef.current = 0; // Reset on successful connection
    };

    // Handle errors
    eventSource.onerror = () => {
      setIsConnected(false);
      const err = new Error('SSE connection failed');
      setError(err);
      onError?.(err);

      eventSource.close();
      eventSourceRef.current = null;

      // Attempt reconnection if enabled, not manually disconnected, and within retry limit
      const withinRetryLimit =
        maxReconnectAttempts === 0 || reconnectAttemptsRef.current < maxReconnectAttempts;
      if (autoReconnect && !isManuallyDisconnectedRef.current && withinRetryLimit) {
        reconnectAttemptsRef.current++;
        setIsReconnecting(true);
        reconnectTimeoutRef.current = setTimeout(() => {
          connectRef.current?.();
        }, reconnectDelay);
      } else if (!withinRetryLimit) {
        setIsReconnecting(false);
        const maxRetriesErr = new Error(
          `SSE connection failed after ${maxReconnectAttempts} attempts`
        );
        setError(maxRetriesErr);
        onError?.(maxRetriesErr);
      }
    };
  }, [taskId, handleEvent, onError, autoReconnect, reconnectDelay, maxReconnectAttempts]);

  // Store connect function in ref for recursive reconnection
  useEffect(() => {
    connectRef.current = connect;
  }, [connect]);

  // Disconnect from SSE
  const disconnect = useCallback(() => {
    isManuallyDisconnectedRef.current = true;

    if (reconnectTimeoutRef.current) {
      clearTimeout(reconnectTimeoutRef.current);
      reconnectTimeoutRef.current = null;
    }

    if (eventSourceRef.current) {
      eventSourceRef.current.close();
      eventSourceRef.current = null;
    }

    setIsConnected(false);
    setIsReconnecting(false);
  }, []);

  // Manual reconnect
  const reconnect = useCallback(() => {
    disconnect();
    isManuallyDisconnectedRef.current = false;
    reconnectAttemptsRef.current = 0; // Reset attempts on manual reconnect
    connect();
  }, [connect, disconnect]);

  // Effect to connect when taskId changes
  useEffect(() => {
    if (taskId) {
      initializeProgress();
      connect();
    }

    return () => {
      disconnect();
    };
  }, [taskId, connect, disconnect, initializeProgress]);

  return {
    progress,
    status,
    error,
    isConnected,
    isReconnecting,
    disconnect,
    reconnect,
  };
}

export default useParseProgress;
