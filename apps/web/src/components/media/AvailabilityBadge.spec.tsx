import { describe, expect, it } from 'vitest';
import { render, screen } from '@testing-library/react';
import { AvailabilityBadge } from './AvailabilityBadge';

// Critique R3 remake (2026-08-26): 已有 is a CLASSIFICATION → neutral scrim
// pill (green is reserved for happening-now); 已請求 is amber grammar → tint
// pill over an opaque backing; both 11px (detector floor); no aria-live (a
// resolving grid used to fire 12 simultaneous announcements).
describe('AvailabilityBadge (Story 10-4 AC #3 · R3 remake)', () => {
  it('renders 已有 label for owned variant', () => {
    render(<AvailabilityBadge variant="owned" />);
    const el = screen.getByTestId('availability-badge-owned');
    expect(el).toBeInTheDocument();
    expect(el).toHaveTextContent('已有');
  });

  it('renders 已請求 label for requested variant', () => {
    render(<AvailabilityBadge variant="requested" />);
    const el = screen.getByTestId('availability-badge-requested');
    expect(el).toBeInTheDocument();
    expect(el).toHaveTextContent('已請求');
  });

  it('已有 wears the neutral scrim — never running green (固定詞彙)', () => {
    render(<AvailabilityBadge variant="owned" />);
    const el = screen.getByTestId('availability-badge-owned');
    expect(el.className).toContain('bg-[var(--overlay-scrim)]');
    // 日巡: the scrim does not flip, so its label must not either —
    // --text-primary here would be near-black ink on a dark veil (2.34:1).
    expect(el.className).toContain('text-[var(--text-on-scrim)]');
    expect(el.className).not.toContain('--success');
  });

  it('已請求 wears the amber tint over an opaque backing (V2 badge recipe)', () => {
    render(<AvailabilityBadge variant="requested" />);
    const el = screen.getByTestId('availability-badge-requested');
    expect(el.className).toContain('bg-[var(--bg-secondary)]');
    const inner = el.querySelector('span');
    expect(inner?.className).toContain('bg-[var(--warning-tint)]');
    expect(inner?.className).toContain('text-[var(--warning-text)]');
    expect(el.className).not.toContain('--success');
  });

  it('pill shape, 11px text, and NO live-region spam', () => {
    render(<AvailabilityBadge variant="owned" />);
    const el = screen.getByTestId('availability-badge-owned');
    expect(el.className).toContain('rounded-full');
    expect(el.className).toContain('text-[11px]');
    expect(el).not.toHaveAttribute('role');
    expect(el).not.toHaveAttribute('aria-live');
  });

  it('merges caller-supplied className without dropping variant styles', () => {
    render(<AvailabilityBadge variant="owned" className="opacity-50" />);
    const el = screen.getByTestId('availability-badge-owned');
    expect(el.className).toContain('opacity-50');
    expect(el.className).toContain('bg-[var(--overlay-scrim)]');
  });
});
