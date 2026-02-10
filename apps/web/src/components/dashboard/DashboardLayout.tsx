import { cn } from '../../lib/utils';

interface DashboardLayoutProps {
  children: React.ReactNode;
  className?: string;
}

export function DashboardLayout({ children, className }: DashboardLayoutProps) {
  return (
    <div className={cn('mx-auto max-w-7xl px-4 py-6', className)} data-testid="dashboard-layout">
      <div className="grid grid-cols-1 gap-6 lg:grid-cols-[400px_1fr]" data-testid="dashboard-grid">
        {children}
      </div>
    </div>
  );
}
