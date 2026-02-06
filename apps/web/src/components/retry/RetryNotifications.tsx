/**
 * RetryNotifications Component (Story 3.11 - Task 9)
 * Shows toast notifications for retry events
 * AC2: Notification when max retries exhausted
 * AC3: Success notification when retry succeeds
 */

import { useEffect, useState } from 'react';
import { CheckCircle2, XCircle, AlertTriangle, X } from 'lucide-react';
import { cn } from '../../lib/utils';

export type NotificationType = 'success' | 'error' | 'warning' | 'info';

export interface Notification {
  id: string;
  type: NotificationType;
  message: string;
  description?: string;
  duration?: number;
}

export interface RetryNotificationsProps {
  notifications: Notification[];
  onDismiss: (id: string) => void;
  className?: string;
}

/**
 * Container component for displaying retry-related notifications
 */
export function RetryNotifications({
  notifications,
  onDismiss,
  className,
}: RetryNotificationsProps) {
  if (notifications.length === 0) {
    return null;
  }

  return (
    <div
      className={cn(
        'fixed bottom-4 right-4 z-50 flex flex-col gap-2 max-w-sm',
        className
      )}
      data-testid="retry-notifications"
      role="log"
      aria-live="polite"
    >
      {notifications.map((notification) => (
        <NotificationToast
          key={notification.id}
          notification={notification}
          onDismiss={onDismiss}
        />
      ))}
    </div>
  );
}

interface NotificationToastProps {
  notification: Notification;
  onDismiss: (id: string) => void;
}

function NotificationToast({ notification, onDismiss }: NotificationToastProps) {
  const [isVisible, setIsVisible] = useState(false);

  useEffect(() => {
    // Trigger animation
    setTimeout(() => setIsVisible(true), 10);

    // Auto dismiss
    const duration = notification.duration || 5000;
    const timer = setTimeout(() => {
      setIsVisible(false);
      setTimeout(() => onDismiss(notification.id), 300);
    }, duration);

    return () => clearTimeout(timer);
  }, [notification.id, notification.duration, onDismiss]);

  const handleDismiss = () => {
    setIsVisible(false);
    setTimeout(() => onDismiss(notification.id), 300);
  };

  const Icon = getIconForType(notification.type);
  const colors = getColorsForType(notification.type);

  return (
    <div
      className={cn(
        'rounded-lg border shadow-lg p-3 transition-all duration-300',
        colors.bg,
        colors.border,
        isVisible
          ? 'opacity-100 translate-x-0'
          : 'opacity-0 translate-x-4'
      )}
      data-testid={`notification-${notification.id}`}
      role="status"
    >
      <div className="flex items-start gap-3">
        <Icon className={cn('h-5 w-5 flex-shrink-0 mt-0.5', colors.icon)} />
        <div className="flex-1 min-w-0">
          <p className={cn('font-medium text-sm', colors.text)}>
            {notification.message}
          </p>
          {notification.description && (
            <p className="text-xs text-slate-400 mt-0.5">
              {notification.description}
            </p>
          )}
        </div>
        <button
          onClick={handleDismiss}
          className="text-slate-400 hover:text-slate-300 transition-colors"
          aria-label="關閉通知"
        >
          <X className="h-4 w-4" />
        </button>
      </div>
    </div>
  );
}

function getIconForType(type: NotificationType) {
  switch (type) {
    case 'success':
      return CheckCircle2;
    case 'error':
      return XCircle;
    case 'warning':
      return AlertTriangle;
    default:
      return CheckCircle2;
  }
}

function getColorsForType(type: NotificationType) {
  switch (type) {
    case 'success':
      return {
        bg: 'bg-green-500/10',
        border: 'border-green-500/20',
        icon: 'text-green-500',
        text: 'text-green-400',
      };
    case 'error':
      return {
        bg: 'bg-red-500/10',
        border: 'border-red-500/20',
        icon: 'text-red-500',
        text: 'text-red-400',
      };
    case 'warning':
      return {
        bg: 'bg-yellow-500/10',
        border: 'border-yellow-500/20',
        icon: 'text-yellow-500',
        text: 'text-yellow-400',
      };
    default:
      return {
        bg: 'bg-blue-500/10',
        border: 'border-blue-500/20',
        icon: 'text-blue-500',
        text: 'text-blue-400',
      };
  }
}

/**
 * Hook for managing retry notifications state
 */
export function useRetryNotifications() {
  const [notifications, setNotifications] = useState<Notification[]>([]);

  const addNotification = (
    type: NotificationType,
    message: string,
    description?: string,
    duration?: number
  ) => {
    const id = `notification-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`;
    setNotifications((prev) => [
      ...prev,
      { id, type, message, description, duration },
    ]);
    return id;
  };

  const dismissNotification = (id: string) => {
    setNotifications((prev) => prev.filter((n) => n.id !== id));
  };

  const showRetrySuccess = (taskId: string) => {
    addNotification('success', '重試成功', `任務 ${taskId} 已完成`);
  };

  const showRetryExhausted = (taskId: string) => {
    addNotification(
      'warning',
      '重試次數已用盡',
      `任務 ${taskId} 需要手動處理`,
      8000
    );
  };

  const showRetryCancelled = () => {
    addNotification('info', '已取消重試');
  };

  const showRetryTriggered = () => {
    addNotification('info', '已觸發立即重試');
  };

  return {
    notifications,
    addNotification,
    dismissNotification,
    showRetrySuccess,
    showRetryExhausted,
    showRetryCancelled,
    showRetryTriggered,
  };
}

export default RetryNotifications;
