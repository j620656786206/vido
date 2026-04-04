/**
 * New Media Notifications Container (Story 4.3 - Task 8)
 * Displays toast notifications when new media is added to the library
 * AC2: Notification indicates successful addition
 */

import { useEffect, useState } from 'react';
import { X } from 'lucide-react';
import { cn } from '../../lib/utils';
import type { NewMediaNotification } from '../../hooks/useNewMediaNotifications';

const mediaTypeLabels: Record<string, string> = {
  movie: '電影',
  tv: '影集',
};

interface NewMediaNotificationsProps {
  notifications: NewMediaNotification[];
  onDismiss: (id: string) => void;
  className?: string;
}

export function NewMediaNotifications({
  notifications,
  onDismiss,
  className,
}: NewMediaNotificationsProps) {
  if (notifications.length === 0) {
    return null;
  }

  return (
    <div
      className={cn('fixed bottom-4 right-4 z-50 flex max-w-sm flex-col gap-2', className)}
      data-testid="new-media-notifications"
      role="log"
      aria-live="polite"
    >
      {notifications.map((notification) => (
        <NewMediaToastItem
          key={notification.id}
          notification={notification}
          onDismiss={onDismiss}
        />
      ))}
    </div>
  );
}

interface NewMediaToastItemProps {
  notification: NewMediaNotification;
  onDismiss: (id: string) => void;
}

function NewMediaToastItem({ notification, onDismiss }: NewMediaToastItemProps) {
  const [isVisible, setIsVisible] = useState(false);
  const { media } = notification;

  useEffect(() => {
    // Trigger enter animation
    const showTimer = setTimeout(() => setIsVisible(true), 10);

    // Auto dismiss after 5 seconds
    const dismissTimer = setTimeout(() => {
      setIsVisible(false);
      setTimeout(() => onDismiss(notification.id), 300);
    }, 5000);

    return () => {
      clearTimeout(showTimer);
      clearTimeout(dismissTimer);
    };
  }, [notification.id, onDismiss]);

  const handleDismiss = () => {
    setIsVisible(false);
    setTimeout(() => onDismiss(notification.id), 300);
  };

  return (
    <div
      className={cn(
        'flex items-center gap-3 rounded-lg border border-emerald-500/20 bg-[var(--bg-secondary)] p-3 shadow-lg transition-all duration-300',
        isVisible ? 'translate-x-0 opacity-100' : 'translate-x-4 opacity-0'
      )}
      data-testid={`new-media-toast-${notification.id}`}
      role="status"
    >
      {/* Poster thumbnail */}
      <div className="h-12 w-8 shrink-0 overflow-hidden rounded bg-[var(--bg-tertiary)]">
        {media.posterUrl ? (
          <img src={media.posterUrl} alt={media.title} className="h-full w-full object-cover" />
        ) : (
          <div className="flex h-full items-center justify-center text-[var(--text-muted)]">
            <span className="text-xs">🎬</span>
          </div>
        )}
      </div>

      {/* Content */}
      <div className="min-w-0 flex-1">
        <p className="text-sm font-medium text-emerald-400">已新增至媒體庫</p>
        <p className="truncate text-xs text-[var(--text-secondary)]">{media.title}</p>
        <span className="text-[10px] text-[var(--text-secondary)]">
          {mediaTypeLabels[media.mediaType]}
        </span>
      </div>

      {/* Close button */}
      <button
        onClick={handleDismiss}
        className="shrink-0 text-[var(--text-secondary)] transition-colors hover:text-[var(--text-secondary)]"
        aria-label="關閉通知"
      >
        <X className="h-4 w-4" />
      </button>
    </div>
  );
}

export default NewMediaNotifications;
