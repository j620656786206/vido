// Implements: Component/SidebarGroupLabel (v5Io8)
/**
 * Sidebar group header (UX Redesign Phase 2 — UX2-1, §5.3). Uppercase-style
 * 11px/600 +tracking label in `text-muted` separating 內容 from 任務 (N3). Hidden
 * on the collapsed rail (the rail groups by spacing, not labels).
 */
interface SidebarGroupLabelProps {
  children: React.ReactNode;
}

export function SidebarGroupLabel({ children }: SidebarGroupLabelProps) {
  return (
    <div className="px-2.5 pb-1 pt-4 text-[11px] font-semibold uppercase tracking-wider text-[var(--text-muted)]">
      {children}
    </div>
  );
}
