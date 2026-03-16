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
      <div className="mb-4 rounded-full bg-slate-800 p-4">
        <Icon className="h-8 w-8 text-slate-500" data-testid="placeholder-icon" />
      </div>
      <h2 className="mb-2 text-xl font-semibold text-slate-200" data-testid="placeholder-title">
        {title}
      </h2>
      <p className="max-w-sm text-sm text-slate-400" data-testid="placeholder-description">
        {description}
      </p>
      <span className="mt-4 rounded-full bg-slate-800 px-3 py-1 text-xs text-slate-500">
        即將推出
      </span>
    </div>
  );
}
