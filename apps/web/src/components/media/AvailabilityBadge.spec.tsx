import { describe, expect, it } from 'vitest';
import { render, screen } from '@testing-library/react';
import { AvailabilityBadge } from './AvailabilityBadge';

describe('AvailabilityBadge (Story 10-4, AC #3)', () => {
  it('renders 已有 label for owned variant', () => {
    render(<AvailabilityBadge variant="owned" />);
    const el = screen.getByTestId('availability-badge-owned');
    expect(el).toBeInTheDocument();
    expect(el).toHaveTextContent('已有');
    expect(el).toHaveAttribute('aria-label', '已有');
  });

  it('renders 已請求 label for requested variant', () => {
    render(<AvailabilityBadge variant="requested" />);
    const el = screen.getByTestId('availability-badge-requested');
    expect(el).toBeInTheDocument();
    expect(el).toHaveTextContent('已請求');
    expect(el).toHaveAttribute('aria-label', '已請求');
  });

  it('applies distinct variant colours so owned and requested are not visually identical', () => {
    const { rerender } = render(<AvailabilityBadge variant="owned" />);
    const ownedClass = screen.getByTestId('availability-badge-owned').className;
    expect(ownedClass).toContain('--success');

    rerender(<AvailabilityBadge variant="requested" />);
    const requestedClass = screen.getByTestId('availability-badge-requested').className;
    expect(requestedClass).toContain('--warning');
    expect(requestedClass).not.toContain('--success');
  });

  it('uses the shared pill-style typography (matches sibling new-badge sizing)', () => {
    render(<AvailabilityBadge variant="owned" />);
    const el = screen.getByTestId('availability-badge-owned');
    // Font-size and padding are enforced in CSS — assert the tokens are
    // applied so design drift fails the test rather than silently passing.
    expect(el.className).toContain('rounded');
    expect(el.className).toContain('text-[10px]');
    expect(el.className).toContain('font-bold');
  });

  it('merges caller-supplied className without dropping variant styles', () => {
    render(<AvailabilityBadge variant="owned" className="opacity-50" />);
    const el = screen.getByTestId('availability-badge-owned');
    expect(el.className).toContain('opacity-50');
    expect(el.className).toContain('--success');
  });
});
