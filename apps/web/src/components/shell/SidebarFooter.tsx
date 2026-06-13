// Implements: Component/SidebarFooterStatus (PrmQG)
/**
 * Ambient status strip in the sidebar footer (UX Redesign Phase 2 — UX2-1,
 * ADR D4-2 / §6.4). Renders disk headroom · active scan · queue count · service
 * health dots.
 *
 * PILOT-DEGRADED (story AC #9): only the service-health dots are wired live for
 * the pilot — sourced from the existing `GET /settings/services` query, which is
 * a plain TanStack Query (no eager SSE in this always-mounted footer, per Rule 8).
 * Disk / active-scan / queue depend on a Phase-3 aggregate `/status/summary`
 * endpoint and render as fail-soft placeholders for now (F3 — empty/stale, never
 * an error). Collapsed rail shows the dots only.
 */
import { useServiceStatuses } from '../../hooks/useServiceStatus';
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

interface SidebarFooterProps {
  collapsed?: boolean;
}

export function SidebarFooter({ collapsed = false }: SidebarFooterProps) {
  const { data, isError } = useServiceStatuses();
  const services = data?.services ?? [];

  const dots =
    services.length > 0 && !isError ? (
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

  return (
    <div
      className="space-y-2 border-t border-[var(--border-subtle)] px-2.5 py-3"
      data-testid="sidebar-footer-status"
    >
      {/* Disk headroom — pilot-degraded (Phase-3 aggregate endpoint) */}
      <div>
        <div className="flex items-center justify-between text-[11px] text-[var(--text-muted)]">
          <span>儲存空間</span>
          <span className="font-mono tabular-nums">—</span>
        </div>
        <div
          className="mt-1 h-1.5 overflow-hidden rounded-full bg-[var(--bg-tertiary)]"
          aria-hidden="true"
        />
      </div>
      {/* Service-health dots — live */}
      <div className="flex items-center gap-1.5" aria-label="服務狀態">
        {dots}
      </div>
    </div>
  );
}
