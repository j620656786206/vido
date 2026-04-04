import * as React from 'react';
import { cn } from '@/lib/utils';

function Skeleton({ className, ...props }: React.ComponentProps<'div'>) {
  return (
    <div
      className={cn('animate-pulse rounded-[var(--radius-md)] bg-[var(--bg-tertiary)]', className)}
      {...props}
    />
  );
}

export { Skeleton };
