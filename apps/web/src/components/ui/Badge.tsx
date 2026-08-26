// Implements: <utility — no .pen counterpart>
import * as React from 'react';
import { cva, type VariantProps } from 'class-variance-authority';
import { cn } from '@/lib/utils';

const badgeVariants = cva(
  'inline-flex items-center rounded-[var(--radius-sm)] px-2 py-0.5 text-xs font-medium transition-colors',
  {
    variants: {
      variant: {
        /* Labels on a SOLID semantic fill take --text-on-accent, not a literal
           white: the token flips with the theme (ink in 夜行, paper in 日巡),
           so the same fill keeps a legible label in both. */
        default: 'bg-[var(--accent-primary)] text-[var(--text-on-accent)]',
        secondary: 'bg-[var(--bg-tertiary)] text-[var(--text-primary)]',
        destructive: 'bg-[var(--error)] text-[var(--text-on-scrim)]',
        outline: 'border border-[var(--border-subtle)] text-[var(--text-secondary)]',
        success: 'bg-[var(--success)]/20 text-[var(--success-text)]',
        warning: 'bg-[var(--warning)]/20 text-[var(--warning-text)]',
        info: 'bg-[var(--info)]/20 text-[var(--info-text)]',
      },
    },
    defaultVariants: {
      variant: 'default',
    },
  }
);

function Badge({
  className,
  variant,
  ...props
}: React.ComponentProps<'span'> & VariantProps<typeof badgeVariants>) {
  return <span className={cn(badgeVariants({ variant }), className)} {...props} />;
}

export { Badge, badgeVariants };
