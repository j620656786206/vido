import * as React from 'react';
import { cva, type VariantProps } from 'class-variance-authority';
import { cn } from '@/lib/utils';

const badgeVariants = cva(
  'inline-flex items-center rounded-[var(--radius-sm)] px-2 py-0.5 text-xs font-medium transition-colors',
  {
    variants: {
      variant: {
        default: 'bg-[var(--accent-primary)] text-white',
        secondary: 'bg-[var(--bg-tertiary)] text-[var(--text-primary)]',
        destructive: 'bg-[var(--error)] text-white',
        outline: 'border border-[var(--border-subtle)] text-[var(--text-secondary)]',
        success: 'bg-[var(--success)]/20 text-[var(--success)]',
        warning: 'bg-[var(--warning)]/20 text-[var(--warning)]',
        info: 'bg-[var(--info)]/20 text-[var(--info)]',
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
