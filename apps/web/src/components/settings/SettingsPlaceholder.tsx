// Design ref: ux-design.pen Screen C14-D (aJSKl) · C14-M (JUEUD)
// 分頁列上「效能監控」是 aria-disabled，這兩張畫的是路由真的被打開時的樣子
import type { LucideIcon } from 'lucide-react';

interface SettingsPlaceholderProps {
  icon: LucideIcon;
  title: string;
  description: string;
}

export function SettingsPlaceholder({ icon: Icon, title, description }: SettingsPlaceholderProps) {
  return (
    <div
      className="flex flex-col items-center justify-center gap-4 py-20 text-center"
      data-testid="settings-placeholder"
    >
      <div className="flex size-18 items-center justify-center rounded-full bg-[var(--bg-secondary)] max-sm:size-16">
        <Icon
          className="size-8 text-[var(--text-muted)] max-sm:size-7"
          data-testid="placeholder-icon"
        />
      </div>
      <h2
        className="text-lg font-bold text-[var(--text-primary)] sm:text-xl"
        data-testid="placeholder-title"
      >
        {title}
      </h2>
      <p
        className="max-w-[420px] text-sm text-[var(--text-secondary)]"
        data-testid="placeholder-description"
      >
        {description}
      </p>
      <span className="rounded-full bg-[var(--bg-secondary)] px-3 py-1 text-xs text-[var(--text-muted)]">
        此功能將在後續版本中提供
      </span>
    </div>
  );
}
