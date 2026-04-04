/**
 * Connection history side panel (Story 4.6 - AC4)
 */

import { Wifi, WifiOff, AlertTriangle, RefreshCw, Loader2 } from 'lucide-react';
import { useState } from 'react';
import { cn } from '../../lib/utils';
import { formatRelativeTimeZh } from '../../lib/timeFormat';
import { SidePanel } from '../ui/SidePanel';
import { useConnectionHistory, type ConnectionEvent } from '../../hooks/useConnectionHealth';
import type { ConnectionEventType } from '../../services/healthService';

interface ConnectionHistoryPanelProps {
  isOpen: boolean;
  onClose: () => void;
}

const eventTypeConfig: Record<
  ConnectionEventType,
  { label: string; Icon: React.ComponentType<{ className?: string }>; color: string }
> = {
  connected: { label: '已連線', Icon: Wifi, color: 'text-emerald-400' },
  disconnected: { label: '已斷線', Icon: WifiOff, color: 'text-[var(--error)]' },
  error: { label: '錯誤', Icon: AlertTriangle, color: 'text-[var(--warning)]' },
  recovered: { label: '已恢復', Icon: RefreshCw, color: 'text-emerald-400' },
};

const validEventTypes: ConnectionEventType[] = ['connected', 'disconnected', 'error', 'recovered'];

export function ConnectionHistoryPanel({ isOpen, onClose }: ConnectionHistoryPanelProps) {
  const [filterType, setFilterType] = useState<ConnectionEventType | 'all'>('all');
  const { data: history, isLoading } = useConnectionHistory('qbittorrent', isOpen);

  const filteredHistory =
    filterType === 'all' ? history : history?.filter((e) => e.eventType === filterType);

  return (
    <SidePanel isOpen={isOpen} onClose={onClose} title="qBittorrent 連線記錄">
      <div className="flex flex-col gap-4">
        {/* Filter buttons */}
        <div className="flex flex-wrap gap-1.5" role="group" aria-label="篩選事件類型">
          <button
            onClick={() => setFilterType('all')}
            className={cn(
              'rounded-full px-2.5 py-1 text-xs transition-colors',
              filterType === 'all'
                ? 'bg-[var(--accent-primary)] text-white'
                : 'bg-[var(--bg-tertiary)] text-[var(--text-secondary)] hover:bg-[var(--bg-tertiary)]'
            )}
          >
            全部
          </button>
          {validEventTypes.map((type) => {
            const config = eventTypeConfig[type];
            return (
              <button
                key={type}
                onClick={() => setFilterType(type)}
                className={cn(
                  'rounded-full px-2.5 py-1 text-xs transition-colors',
                  filterType === type
                    ? 'bg-[var(--accent-primary)] text-white'
                    : 'bg-[var(--bg-tertiary)] text-[var(--text-secondary)] hover:bg-[var(--bg-tertiary)]'
                )}
              >
                {config.label}
              </button>
            );
          })}
        </div>

        {/* Content */}
        {isLoading ? (
          <div className="flex items-center justify-center py-8">
            <Loader2 className="h-5 w-5 animate-spin text-[var(--text-secondary)]" />
            <span className="ml-2 text-sm text-[var(--text-secondary)]">載入中...</span>
          </div>
        ) : !filteredHistory || filteredHistory.length === 0 ? (
          <div className="py-8 text-center text-sm text-[var(--text-muted)]">沒有連線記錄</div>
        ) : (
          <div className="flex flex-col gap-1" role="list" aria-label="連線記錄列表">
            {filteredHistory.map((event) => (
              <ConnectionEventItem key={event.id} event={event} />
            ))}
          </div>
        )}
      </div>
    </SidePanel>
  );
}

function ConnectionEventItem({ event }: { event: ConnectionEvent }) {
  const config = eventTypeConfig[event.eventType] || eventTypeConfig.error;
  const { Icon } = config;

  return (
    <div
      className="flex items-start gap-3 rounded-lg p-2.5 hover:bg-[var(--bg-secondary)]/50"
      role="listitem"
    >
      <Icon className={cn('mt-0.5 h-4 w-4 shrink-0', config.color)} aria-hidden="true" />
      <div className="min-w-0 flex-1">
        <p className="text-sm font-medium text-[var(--text-primary)]">{config.label}</p>
        {event.message && (
          <p className="truncate text-xs text-[var(--text-muted)]">{event.message}</p>
        )}
      </div>
      <time className="shrink-0 text-xs text-[var(--text-muted)]">
        {formatRelativeTimeZh(event.createdAt)}
      </time>
    </div>
  );
}

export default ConnectionHistoryPanel;
