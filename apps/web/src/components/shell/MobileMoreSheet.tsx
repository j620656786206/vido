// Implements: Component/MobileMoreSheet (mfDKV)
/**
 * Mobile "More" sheet (UX Redesign Phase 2 — UX2-1, ADR D1-b / §6.3). Opened from
 * the 5th bottom-tab slot. Holds the ambient status strip at the top, then the
 * destinations that did not earn a bottom-4 slot (設定; 系統/活動 join in Phase 3).
 * Built on the Base UI Dialog-backed Sheet wrapper (focus trap + scrim + Escape).
 */
import { Link } from '@tanstack/react-router';
import { Sheet } from '../ui/Sheet';
import { SidebarFooter } from './SidebarFooter';
import { MORE_DESTS } from './navModel';

interface MobileMoreSheetProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function MobileMoreSheet({ open, onOpenChange }: MobileMoreSheetProps) {
  return (
    <Sheet open={open} onOpenChange={onOpenChange} title="更多">
      {/* Destinations FIRST. The sheet opens because someone asked "where else
          can I go", and it used to answer with a theme switch, a disk bar and
          five health dots before naming a single place. The ambient strip keeps
          its slot, underneath, where a glanceable readout belongs. */}
      <div className="space-y-1">
        {MORE_DESTS.map((d) => {
          const Icon = d.icon;
          return (
            <Link
              key={d.key}
              to={d.to}
              search={d.search}
              onClick={() => onOpenChange(false)}
              data-testid={`nav-${d.key}`}
              aria-label={d.label}
              className="flex min-h-[44px] items-center gap-3 rounded-[var(--radius-md)] px-2.5 py-2 text-sm font-medium text-[var(--text-secondary)] transition-colors hover:bg-[var(--bg-tertiary)] hover:text-[var(--text-primary)] data-[status=active]:bg-[var(--accent-subtle)] data-[status=active]:text-[var(--text-primary)]"
            >
              <Icon className="h-5 w-5 shrink-0 text-[var(--text-muted)]" aria-hidden="true" />
              <span>{d.label}</span>
            </Link>
          );
        })}
      </div>

      {/* Ambient status strip + 登出 (§6.3). No theme toggle here: the mobile
          header already carries it, and ThemeToggle's contract is one instance
          per breakpoint. */}
      <div className="mt-2">
        <SidebarFooter showThemeToggle={false} />
      </div>
    </Sheet>
  );
}
