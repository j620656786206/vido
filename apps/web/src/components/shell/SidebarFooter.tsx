// Implements: Component/SidebarFooterStatus (PrmQG)
/**
 * Ambient status strip in the sidebar footer (UX Redesign D4-2 / §6.4). Renders disk
 * headroom · active scan · queue count · service-health dots from the fail-soft
 * `GET /api/v1/status/summary` aggregate (ux3-0-3, consumed here in ux3-0-4).
 *
 * Per-section fail-soft (ADR F3, frontend half): a section whose status is not "ok"
 * (or the whole query erroring/loading) renders an empty/placeholder treatment and
 * NEVER throws. Collapsed rail shows the health dots only. The active-scan pulse
 * respects `prefers-reduced-motion`.
 */
import { useStatusSummary } from '../../hooks/useStatusSummary';
import type { ServiceConnectionStatus } from '../../services/serviceStatusService';
import { Tooltip } from '../ui/Tooltip';
import { cn } from '../../lib/utils';

const DOT_COLOR: Record<ServiceConnectionStatus, string> = {
  connected: 'bg-[var(--success)]',
  rate_limited: 'bg-[var(--warning)]',
  error: 'bg-[var(--error)]',
  disconnected: 'bg-[var(--error)]',
  unconfigured: 'bg-[var(--text-disabled)]',
};
const DOT_LABEL: Record<ServiceConnectionStatus, string> = {
  connected: '正常',
  rate_limited: '限流',
  error: '異常',
  disconnected: '離線',
  unconfigured: '未設定',
};

/** Decimal TB, one decimal (NAS vendors use decimal TB). */
function formatTB(bytes: number): string {
  return (bytes / 1e12).toFixed(1);
}

interface SidebarFooterProps {
  collapsed?: boolean;
}

export function SidebarFooter({ collapsed = false }: SidebarFooterProps) {
  const { data } = useStatusSummary();

  const health = data?.serviceHealth;
  const disk = data?.diskHeadroom;
  const scan = data?.activeScan;
  const queue = data?.downloadQueue;

  const services = health?.status === 'ok' ? health.services : [];
  const dots =
    services.length > 0 ? (
      services.map((s) => (
        <Tooltip key={s.name} content={`${s.displayName} · ${DOT_LABEL[s.status]}`} side="top">
          <span
            role="img"
            aria-label={`${s.displayName}：${DOT_LABEL[s.status]}`}
            data-testid={`status-dot-${s.name}`}
            className={cn('inline-block h-2 w-2 rounded-full', DOT_COLOR[s.status])}
          />
        </Tooltip>
      ))
    ) : (
      // Fail-soft: unknown service health → muted placeholder dots, never an error.
      <>
        {[0, 1, 2].map((i) => (
          <span
            key={i}
            aria-hidden="true"
            className="inline-block h-2 w-2 rounded-full bg-[var(--text-disabled)]"
          />
        ))}
      </>
    );

  if (collapsed) {
    return (
      <div
        className="flex flex-col items-center gap-1.5 border-t border-[var(--border-subtle)] py-3"
        data-testid="sidebar-footer-status"
      >
        {dots}
      </div>
    );
  }

  const diskOk = disk?.status === 'ok' && disk.totalBytes > 0;
  const usedRatio = diskOk ? Math.min(1, disk.usedBytes / disk.totalBytes) : 0;
  const diskFill =
    usedRatio >= 0.9
      ? 'var(--error)'
      : usedRatio >= 0.8
        ? 'var(--warning)'
        : 'var(--accent-primary)';

  const scanActive = scan?.status === 'ok' && scan.active;
  const queueCount = queue?.status === 'ok' ? queue.downloading : 0;

  return (
    <div
      className="space-y-2 border-t border-[var(--border-subtle)] px-2.5 py-3"
      data-testid="sidebar-footer-status"
    >
      {/* Disk headroom */}
      <div data-testid="status-disk">
        <div className="flex items-center justify-between text-[11px] text-[var(--text-muted)]">
          <span>儲存空間</span>
          <span className="font-mono tabular-nums">
            {diskOk ? `${formatTB(disk.usedBytes)} / ${formatTB(disk.totalBytes)} TB` : '—'}
          </span>
        </div>
        <div className="mt-1 h-1.5 overflow-hidden rounded-full bg-[var(--bg-tertiary)]">
          {diskOk && (
            <div
              className="h-full rounded-full"
              style={{ width: `${usedRatio * 100}%`, backgroundColor: diskFill }}
              aria-hidden="true"
            />
          )}
        </div>
      </div>

      {/* Scan · queue · service dots */}
      <div className="flex items-center gap-2 text-[11px] text-[var(--text-muted)]">
        {scanActive && (
          <span className="flex items-center gap-1" data-testid="status-scan">
            <span className="inline-block h-2 w-2 rounded-full bg-[var(--accent-primary)] motion-safe:animate-pulse" />
            掃描中
          </span>
        )}
        {queueCount > 0 && (
          <span className="font-mono tabular-nums" data-testid="status-queue">
            佇列 {queueCount}
          </span>
        )}
        <span className="ml-auto flex items-center gap-1.5" aria-label="服務狀態">
          {dots}
        </span>
      </div>
    </div>
  );
}
