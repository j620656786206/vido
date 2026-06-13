// Implements: Component/SidebarNavItem (W5KQr)
/**
 * One sidebar destination row (UX Redesign Phase 2 — UX2-1, §5.3 / §6.1).
 *
 * Active state comes from TanStack Router's own matching — the `Link` emits
 * `data-status="active"`, which Tailwind `data-[status=active]:` variants style
 * (NOT a hand-rolled `startsWith`, per the ADR). Two layouts: expanded (icon +
 * label + optional Mono count / accent badge) and collapsed rail (44px icon-only,
 * label+count surfaced through a Base UI Tooltip). 44px min hit area either way.
 */
import { Link } from '@tanstack/react-router';
import type { LucideIcon } from 'lucide-react';
import { cn } from '../../lib/utils';
import { Tooltip } from '../ui/Tooltip';

export interface SidebarNavItemProps {
  to: string;
  label: string;
  icon: LucideIcon;
  /** testid → `nav-{navKey}`. */
  navKey: string;
  search?: Record<string, string>;
  /** Right-aligned Mono count (e.g. library size). */
  count?: number;
  /** Accent pill for in-flight job counts. */
  badge?: number;
  /** Rail mode — icon-only with tooltip. */
  collapsed?: boolean;
  /** Child row under a group parent (電影/影集) — indented, smaller icon. */
  indent?: boolean;
  /** Exact active match (Home). */
  exact?: boolean;
  /** Whether search params participate in active matching (default true). */
  includeSearch?: boolean;
}

export function SidebarNavItem({
  to,
  label,
  icon: Icon,
  navKey,
  search,
  count,
  badge,
  collapsed = false,
  indent = false,
  exact = false,
  includeSearch,
}: SidebarNavItemProps) {
  const activeOptions = { exact, ...(includeSearch === undefined ? {} : { includeSearch }) };
  const hasCount = typeof count === 'number';

  if (collapsed) {
    return (
      <Tooltip content={hasCount ? `${label} · ${count!.toLocaleString()}` : label}>
        <Link
          to={to}
          search={search}
          activeOptions={activeOptions}
          data-testid={`nav-${navKey}`}
          aria-label={hasCount ? `${label}（${count}）` : label}
          className="flex h-11 w-11 items-center justify-center rounded-[var(--radius-md)] text-[var(--text-muted)] transition-colors hover:bg-[var(--bg-tertiary)] hover:text-[var(--text-primary)] data-[status=active]:bg-[var(--accent-subtle)] data-[status=active]:text-[var(--accent-hover)]"
        >
          <Icon className="h-5 w-5" aria-hidden="true" />
        </Link>
      </Tooltip>
    );
  }

  return (
    <Link
      to={to}
      search={search}
      activeOptions={activeOptions}
      data-testid={`nav-${navKey}`}
      aria-label={label}
      className={cn(
        'group/navitem flex min-h-[44px] items-center gap-3 rounded-[var(--radius-md)] px-2.5 py-2 text-sm font-medium transition-colors',
        'text-[var(--text-secondary)] hover:bg-[var(--bg-tertiary)] hover:text-[var(--text-primary)]',
        'data-[status=active]:bg-[var(--accent-subtle)] data-[status=active]:font-semibold data-[status=active]:text-[var(--text-primary)]',
        indent && 'pl-9'
      )}
    >
      <Icon
        className={cn(
          'shrink-0 text-[var(--text-muted)] transition-colors group-hover/navitem:text-[var(--text-secondary)] group-data-[status=active]/navitem:text-[var(--accent-hover)]',
          indent ? 'h-4 w-4' : 'h-[18px] w-[18px]'
        )}
        aria-hidden="true"
      />
      <span className="truncate">{label}</span>
      {hasCount && (
        <span className="ml-auto font-mono text-[11px] tabular-nums text-[var(--text-muted)]">
          {count!.toLocaleString()}
        </span>
      )}
      {typeof badge === 'number' && badge > 0 && (
        <span className="ml-auto rounded-full bg-[var(--accent-primary)] px-1.5 py-0.5 font-mono text-[11px] leading-none text-[var(--text-on-accent)]">
          {badge}
        </span>
      )}
    </Link>
  );
}
