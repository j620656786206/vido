/**
 * New Media Notifications Hook (Story 4.3 - Task 8)
 * Detects newly added media and triggers toast notifications
 * AC2: Notification when media is added to library
 */

import { useEffect, useRef, useCallback, useState } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { mediaKeys, useRecentMedia } from './useDashboardData';
import type { RecentMedia } from '../services/mediaService';

export interface NewMediaNotification {
  id: string;
  media: RecentMedia;
  timestamp: number;
}

/**
 * Hook for detecting and notifying when new media is added
 */
export function useNewMediaNotifications() {
  const [notifications, setNotifications] = useState<NewMediaNotification[]>([]);
  const previousIdsRef = useRef<Set<string>>(new Set());
  const isInitializedRef = useRef(false);
  const queryClient = useQueryClient();

  const { data: recentMedia } = useRecentMedia(8);

  // Detect new media items
  useEffect(() => {
    if (!recentMedia || recentMedia.length === 0) return;

    const currentIds = new Set(recentMedia.map((m) => m.id));

    // Skip first load - just initialize the set
    if (!isInitializedRef.current) {
      previousIdsRef.current = currentIds;
      isInitializedRef.current = true;
      return;
    }

    // Find newly added media (in current but not in previous)
    const newItems = recentMedia.filter(
      (media) => !previousIdsRef.current.has(media.id) && media.justAdded
    );

    if (newItems.length > 0) {
      const newNotifications = newItems.map((media) => ({
        id: `new-media-${media.id}-${Date.now()}`,
        media,
        timestamp: Date.now(),
      }));
      setNotifications((prev) => [...prev, ...newNotifications]);
    }

    previousIdsRef.current = currentIds;
  }, [recentMedia]);

  const dismissNotification = useCallback((id: string) => {
    setNotifications((prev) => prev.filter((n) => n.id !== id));
  }, []);

  const clearAllNotifications = useCallback(() => {
    setNotifications([]);
  }, []);

  // Manually trigger a refresh to check for new media
  const checkForNewMedia = useCallback(() => {
    queryClient.invalidateQueries({ queryKey: mediaKeys.all });
  }, [queryClient]);

  return {
    notifications,
    dismissNotification,
    clearAllNotifications,
    checkForNewMedia,
  };
}
