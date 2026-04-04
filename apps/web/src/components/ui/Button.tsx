import * as React from 'react';
import { Slot } from '@radix-ui/react-slot';
import { cva, type VariantProps } from 'class-variance-authority';
import { cn } from '@/lib/utils';

const buttonVariants = cva(
  'inline-flex items-center justify-center gap-2 whitespace-nowrap rounded-[var(--radius-md)] text-sm font-medium transition-all disabled:pointer-events-none disabled:opacity-50 [&_svg]:pointer-events-none [&_svg:not([class*=size-])]:size-4 [&_svg]:shrink-0',
  {
    variants: {
      variant: {
        default:
          'bg-[var(--accent-primary)] text-white shadow-[var(--shadow-sm)] hover:bg-[var(--accent-hover)] active:bg-[var(--accent-pressed)]',
        destructive:
          'bg-[var(--error)] text-white shadow-[var(--shadow-sm)] hover:bg-[var(--error)]/90',
        outline:
          'border border-[var(--border-subtle)] bg-transparent text-[var(--text-primary)] hover:bg-[var(--bg-tertiary)]',
        secondary:
          'bg-[var(--bg-tertiary)] text-[var(--text-primary)] shadow-[var(--shadow-sm)] hover:bg-[var(--bg-tertiary)]/80',
        ghost: 'text-[var(--text-primary)] hover:bg-[var(--bg-tertiary)]',
        link: 'text-[var(--accent-primary)] underline-offset-4 hover:underline',
      },
      size: {
        default: 'h-9 px-4 py-2',
        sm: 'h-8 rounded-[var(--radius-md)] px-3 text-xs',
        lg: 'h-10 rounded-[var(--radius-md)] px-6',
        icon: 'size-9',
      },
    },
    defaultVariants: {
      variant: 'default',
      size: 'default',
    },
  }
);

function Button({
  className,
  variant,
  size,
  asChild = false,
  ...props
}: React.ComponentProps<'button'> &
  VariantProps<typeof buttonVariants> & {
    asChild?: boolean;
  }) {
  const Comp = asChild ? Slot : 'button';
  return <Comp className={cn(buttonVariants({ variant, size, className }))} {...props} />;
}

export { Button, buttonVariants };
