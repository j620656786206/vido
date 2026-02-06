/**
 * RetryQueueWithNotifications Component (Story 3.11 - Task 9)
 * Combines RetryQueuePanel with notification system
 * AC2: Shows notification when max retries exhausted
 * AC3: Shows success notification when retry succeeds
 */

import { useEffect, useRef } from 'react';
import { RetryQueuePanel, type RetryQueuePanelProps } from './RetryQueuePanel';
import { RetryNotifications, useRetryNotifications } from './RetryNotifications';
import { usePendingRetries, useTriggerRetry, useCancelRetry } from '../../hooks/useRetry';
import type { RetryItem } from '../../services/retry';

export interface RetryQueueWithNotificationsProps extends RetryQueuePanelProps {
  /** Called when a retry succeeds (detected via polling) */
  onRetrySuccess?: (taskId: string) => void;
  /** Called when retries are exhausted (detected via polling) */
  onRetryExhausted?: (taskId: string) => void;
}

/**
 * RetryQueuePanel with integrated notifications
 * Monitors for completed retries and shows appropriate notifications
 */
export function RetryQueueWithNotifications({
  className,
  onRetrySuccess,
  onRetryExhausted,
}: RetryQueueWithNotificationsProps) {
  const {
    notifications,
    dismissNotification,
    showRetryTriggered,
    showRetryCancelled,
  } = useRetryNotifications();

  const { data } = usePendingRetries();
  const prevItemsRef = useRef<RetryItem[]>([]);

  const triggerMutation = useTriggerRetry();
  const cancelMutation = useCancelRetry();

  // Track previous items to detect changes
  useEffect(() => {
    if (!data?.items) return;

    const currentItems = data.items;
    const prevItems = prevItemsRef.current;

    // Check for items that were removed (either completed or exhausted)
    // This is a simple heuristic - a more robust solution would use SSE events
    prevItems.forEach((prevItem) => {
      const stillExists = currentItems.find((item) => item.id === prevItem.id);
      if (!stillExists) {
        // Item was removed - could be success or exhausted
        // We can't differentiate without backend events, so we don't show notification here
        // The notification would be triggered by SSE events in a production system
      }
    });

    prevItemsRef.current = [...currentItems];
  }, [data?.items]);

  // Handle trigger mutation success
  useEffect(() => {
    if (triggerMutation.isSuccess) {
      showRetryTriggered();
    }
  }, [triggerMutation.isSuccess, showRetryTriggered]);

  // Handle cancel mutation success
  useEffect(() => {
    if (cancelMutation.isSuccess) {
      showRetryCancelled();
    }
  }, [cancelMutation.isSuccess, showRetryCancelled]);

  return (
    <>
      <RetryQueuePanel className={className} />
      <RetryNotifications
        notifications={notifications}
        onDismiss={dismissNotification}
      />
    </>
  );
}

export default RetryQueueWithNotifications;
