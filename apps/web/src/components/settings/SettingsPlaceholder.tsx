import type { LucideIcon } from 'lucide-react';

interface SettingsPlaceholderProps {
  icon: LucideIcon;
  title: string;
  description: string;
}

export function SettingsPlaceholder({ icon: Icon, title, description }: SettingsPlaceholderProps) {
  return (
    <div
      className="flex flex-col items-center justify-center py-20 text-center"
      data-testid="settings-placeholder"
    >
      <div className="mb-4 rounded-full bg-[var(--bg-secondary)] p-4">
        <Icon className="h-8 w-8 text-[var(--text-muted)]" data-testid="placeholder-icon" />
      </div>
      <h2
        className="mb-2 text-xl font-semibold text-[var(--text-primary)]"
        data-testid="placeholder-title"
      >
        {title}
      </h2>
      <p
        className="max-w-sm text-sm text-[var(--text-secondary)]"
        data-testid="placeholder-description"
      >
        {description}
      </p>
      <span className="mt-4 rounded-full bg-[var(--bg-secondary)] px-3 py-1 text-xs text-[var(--text-muted)]">
        此功能將在後續版本中提供
      </span>
    </div>
  );
}
