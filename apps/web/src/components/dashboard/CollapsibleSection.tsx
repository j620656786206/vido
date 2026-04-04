/**
 * CollapsibleSection Component (Story 4.3 - AC4)
 * Provides collapsible behavior for dashboard panels on mobile
 */

import { useState } from 'react';
import { ChevronDown } from 'lucide-react';
import { cn } from '../../lib/utils';

interface CollapsibleSectionProps {
  title: string;
  icon?: React.ReactNode;
  badge?: React.ReactNode;
  rightContent?: React.ReactNode;
  children: React.ReactNode;
  defaultExpanded?: boolean;
  className?: string;
  testId?: string;
}

export function CollapsibleSection({
  title,
  icon,
  badge,
  rightContent,
  children,
  defaultExpanded = true,
  className,
  testId,
}: CollapsibleSectionProps) {
  const [isExpanded, setIsExpanded] = useState(defaultExpanded);

  return (
    <div
      className={cn(
        'rounded-lg border border-[var(--border-subtle)] bg-[var(--bg-secondary)]/50',
        className
      )}
      data-testid={testId}
    >
      {/* Collapsible Header - clickable on mobile */}
      <button
        type="button"
        onClick={() => setIsExpanded(!isExpanded)}
        className="flex w-full items-center justify-between border-b border-[var(--border-subtle)] px-4 py-3 text-left lg:cursor-default"
        aria-expanded={isExpanded}
        aria-controls={`${testId}-content`}
      >
        <div className="flex items-center gap-2">
          {icon}
          <h2 className="text-lg font-semibold text-[var(--text-primary)]">{title}</h2>
          {badge}
        </div>
        <div className="flex items-center gap-2">
          {rightContent}
          {/* Chevron only visible on mobile */}
          <ChevronDown
            className={cn(
              'h-5 w-5 text-[var(--text-secondary)] transition-transform lg:hidden',
              isExpanded && 'rotate-180'
            )}
          />
        </div>
      </button>

      {/* Collapsible Content */}
      <div
        id={`${testId}-content`}
        className={cn(
          'overflow-hidden transition-all duration-300',
          // On mobile: collapse/expand, on desktop: always show
          isExpanded
            ? 'max-h-[2000px] opacity-100'
            : 'max-h-0 opacity-0 lg:max-h-none lg:opacity-100'
        )}
      >
        {children}
      </div>
    </div>
  );
}
