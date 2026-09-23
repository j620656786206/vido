// Design ref: ux-design.pen Screen C8-D (wqcqY) · C8-M (qx8Ma)
// 每張卡：col（名稱＋說明）· meta（回應／最後檢查，手機移到第二行）· pill · retest（桌機 36、手機 44）
import { CircleCheck, TriangleAlert, CircleX, WifiOff, CircleSlash, RefreshCw } from 'lucide-react';
import type { ServiceStatus, ServiceConnectionStatus } from '../../services/serviceStatusService';
import { formatRelativeTime } from '../../utils/relativeTime';
import { cn } from '../../lib/utils';
import { STATUS_LABELS, getServiceLabel } from './serviceLabels';

// Pill colours follow the status vocabulary (project_pen_design_token_system):
// 青碧 = has an answer, 赭 = asked but it did not happen, 硃砂 = broken, neutral = not set up.
const pillConfig: Record<ServiceConnectionStatus, { className: string; icon: React.ElementType }> =
  {
    connected: {
      className: 'bg-[var(--success-tint)] text-[var(--success-text)]',
      icon: CircleCheck,
    },
    rate_limited: {
      className: 'bg-[var(--warning-tint)] text-[var(--warning-text)]',
      icon: TriangleAlert,
    },
    error: { className: 'bg-[var(--error-tint)] text-[var(--error-text)]', icon: CircleX },
    disconnected: { className: 'bg-[var(--error-tint)] text-[var(--error-text)]', icon: WifiOff },
    unconfigured: {
      className: 'bg-[var(--bg-tertiary)] text-[var(--text-muted)]',
      icon: CircleSlash,
    },
  };

interface ServiceStatusCardProps {
  service: ServiceStatus;
  onTest: (serviceName: string) => void;
  isTesting: boolean;
}

export function ServiceStatusCard({ service, onTest, isTesting }: ServiceStatusCardProps) {
  const label = getServiceLabel(service);
  const pill = pillConfig[service.status] || pillConfig.error;
  const PillIcon = pill.icon;

  // Freshness is unconditional: a status readout without a time is asking to be
  // taken on faith, and connected/unconfigured are the states people check most.
  const lastCheckLabel = formatRelativeTime(service.lastCheckAt) || '—';
  const responseLabel =
    service.status === 'connected' && service.responseTimeMs > 0
      ? `${service.responseTimeMs} ms`
      : '—';

  return (
    <div
      className="flex flex-wrap items-center gap-x-3 gap-y-2 rounded-[var(--radius-lg)] border border-[var(--border-subtle)] bg-[var(--bg-secondary)] p-4 sm:flex-nowrap sm:gap-4"
      data-testid={`service-card-${service.name}`}
    >
      <div className="min-w-0 flex-1">
        <h3 className="text-sm font-semibold text-[var(--text-primary)]">{label.name}</h3>
        {label.role && <p className="mt-0.5 text-xs text-[var(--text-muted)]">{label.role}</p>}
      </div>

      {/* Phone: its own full-width second line (order-last + basis-full wraps it).
          Desktop: a right-aligned stack between the name and the pill. */}
      <div className="order-last flex basis-full gap-3 font-mono text-xs sm:order-none sm:basis-auto sm:flex-col sm:items-end sm:gap-0.5">
        <span className="text-[var(--text-secondary)]">回應 {responseLabel}</span>
        <span className="text-[var(--text-muted)]" data-testid={`last-check-${service.name}`}>
          最後檢查 {lastCheckLabel}
        </span>
      </div>

      <span
        className={cn(
          'inline-flex shrink-0 items-center gap-1 rounded-full px-2 py-0.5 text-xs font-semibold',
          pill.className
        )}
        data-testid={`status-pill-${service.name}`}
      >
        <PillIcon className="size-3" aria-hidden="true" />
        {STATUS_LABELS[service.status] ?? STATUS_LABELS.error}
      </span>

      <button
        type="button"
        onClick={() => onTest(service.name)}
        disabled={isTesting}
        aria-label={`重新檢查 ${label.name}`}
        className="flex size-11 shrink-0 items-center justify-center rounded-[var(--radius-md)] bg-[var(--bg-tertiary)] text-[var(--text-secondary)] transition-colors hover:text-[var(--text-primary)] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--focus-ring)] disabled:opacity-50 sm:size-9"
        data-testid={`test-btn-${service.name}`}
      >
        <RefreshCw
          className={cn('size-4', isTesting && 'motion-safe:animate-spin')}
          aria-hidden="true"
          data-testid={isTesting ? `test-spinner-${service.name}` : undefined}
        />
      </button>
    </div>
  );
}
