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
import { ThemeToggle } from './ThemeToggle';
import { LogoutButton } from './LogoutButton';
import type { ServiceConnectionStatus } from '../../services/serviceStatusService';
import { Tooltip } from '../ui/Tooltip';
import { cn } from '../../lib/utils';

/**
 * Health is carried by SHAPE as well as colour.
 *
 * These are 8px dots. A screen reader was already served (each carries an
 * aria-label), but a low-vision SIGHTED user got red-versus-green at 8px with
 * nothing else to go on — meaning by colour alone, which is exactly what the
 * accessibility floor forbids. Three distinguishable forms now do the work and
 * colour only adds nuance on top:
 *
 *   filled            → healthy
 *   filled + halo     → needs attention (limited / error / offline)
 *   hollow ring       → never configured, so nothing is wrong
 *
 * The halo also makes the dots that matter physically larger than the ones that
 * do not, which is the right way round.
 */
const DOT_SHAPE: Record<ServiceConnectionStatus, string> = {
  connected: 'bg-[var(--success)]',
  rate_limited: 'bg-[var(--warning)] ring-2 ring-[var(--warning-tint)]',
  error: 'bg-[var(--error)] ring-2 ring-[var(--error-tint)]',
  disconnected: 'bg-[var(--error)] ring-2 ring-[var(--error-tint)]',
  unconfigured: 'border border-[var(--text-disabled)] bg-transparent',
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
  /**
   * False in the mobile 更多 sheet. ThemeToggle's own contract says it is never
   * rendered twice at one breakpoint — desktop puts it here, mobile puts it in
   * the sticky header — but this component is reused by the sheet, which quietly
   * broke that promise: at 390px the switch appeared in the header AND in the
   * sheet. The sheet opts out instead of the rule being abandoned.
   */
  showThemeToggle?: boolean;
}

export function SidebarFooter({ collapsed = false, showThemeToggle = true }: SidebarFooterProps) {
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
            className={cn('inline-block h-2 w-2 rounded-full', DOT_SHAPE[s.status])}
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
            className="inline-block h-2 w-2 rounded-full border border-[var(--text-disabled)]"
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
        {/* ⚖️「放到設定裡面有點太深了」— the theme switch sits with the ambient
            strip because both are shell-level state you glance at, never
            navigate to. Icon-only on the collapsed rail, like the nav items. */}
        {showThemeToggle && <ThemeToggle variant="rail" />}
        {dots}
        {/* Below the readouts and behind a rule: on the rail the logout glyph
            (bracket + arrow) is a near-twin of the collapse glyph at the top of
            the same 44px track, and it used to sit directly above a red health
            dot that read as ITS status. The separator and the order fix both. */}
        <div className="mt-1 w-6 border-t border-[var(--border-subtle)] pt-1.5">
          <LogoutButton variant="rail" />
        </div>
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
      {/* ⚖️「放到設定裡面有點太深了」— see the collapsed branch above. */}
      {showThemeToggle && <ThemeToggle variant="row" className="-mx-0.5" />}
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

      {/* Below the ambient strip and behind its own rule. Everything above this
          line is a readout you glance at; this is the one thing here that fires.
          Same weight as a nav destination (14px / --text-secondary), because it
          has more consequence than one, not less. */}
      <div className="-mx-0.5 border-t border-[var(--border-subtle)] pt-2">
        <LogoutButton variant="row" />
      </div>
    </div>
  );
}
